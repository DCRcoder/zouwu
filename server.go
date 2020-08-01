package zouwu

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/ngaut/log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

var (
	_              IRouter = &Engine{}
	default405Body         = []byte("405 method not allowed")
	default404Body         = []byte("404 page not found")
)

// HandlerFunc http request handler function.
type HandlerFunc func(ctx *Context) error

var defaultErrorHandler = func(ctx *Context, err error) {
	switch e := err.(type) {
	case *Error:
		ctx.Set(HeaderContentType, MIMETextPlainCharsetUTF8)
		ctx.Ctx.Response.SetBodyString(e.Error())
		ctx.Status(e.Code)
	default:
		ctx.Set(HeaderContentType, MIMETextPlainCharsetUTF8)
		ctx.Ctx.Response.SetBodyString(e.Error())
		ctx.Status(http.StatusInternalServerError)
	}
}

// ServerConfig is the bm server config model
type ServerConfig struct {
	Network      string
	Addr         string
	Timeout      time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Engine service engine
type Engine struct {
	RouterGroup

	lock sync.RWMutex
	conf *ServerConfig

	pcLock        sync.RWMutex
	methodConfigs map[string]*MethodConfig

	trees  methodTrees
	server *fasthttp.Server

	// If enabled, the url.RawPath will be used to find parameters.
	UseRawPath bool

	// If true, the path value will be unescaped.
	// If UseRawPath is false (by default), the UnescapePathValues effectively is true,
	// as url.Path gonna be used, which is already unescaped.
	UnescapePathValues bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	allNoRoute  []HandlerFunc
	allNoMethod []HandlerFunc
	noRoute     []HandlerFunc
	noMethod    []HandlerFunc
	DebugMode   bool

	pool sync.Pool

	errorHandler func(ctx *Context, err error)
}

// NewServer returns a new blank Engine instance without any middleware attached.
func NewServer() *Engine {
	conf := defaultServerConfig()
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		conf:                   conf,
		trees:                  make(methodTrees, 0, 9),
		methodConfigs:          make(map[string]*MethodConfig),
		HandleMethodNotAllowed: true,
		DebugMode:              false,
	}
	if err := engine.SetConfig(conf); err != nil {
		panic(err)
	}
	engine.pool.New = func() interface{} {
		return engine.newContext()
	}
	engine.RouterGroup.engine = engine
	engine.NoRoute(func(c *Context) error {
		c.Bytes(404, MIMETextHTML, default404Body)
		c.Abort()
		return nil
	})
	engine.NoMethod(func(c *Context) error {
		c.Bytes(405, MIMETextHTML, default405Body)
		c.Abort()
		return nil
	})
	return engine
}

// defaultServerConfig return default server config
func defaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		Timeout:      1 * time.Second,
		Addr:         "127.0.0.1:8888",
	}
}

// SetMethodConfig is used to set config on specified path
func (engine *Engine) SetMethodConfig(path string, mc *MethodConfig) {
	engine.pcLock.Lock()
	engine.methodConfigs[path] = mc
	engine.pcLock.Unlock()
}

// SetDebugMode  set debug mode will log engine info
func (engine *Engine) SetDebugMode() {
	engine.DebugMode = true
}

func (engine *Engine) addRoute(method, path string, handlers ...HandlerFunc) {
	if path[0] != '/' {
		panic("[zouwu Engine]: path must begin with '/'")
	}
	if method == "" {
		panic("[zouwu Engine]: HTTP method can not be empty")
	}
	if len(handlers) == 0 {
		panic("[zouwu Engine]: there must be at least one handler")
	}
	root := engine.trees.get(method)
	if root == nil {
		root = new(node)
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}

	prelude := func(c *Context) error {
		c.method = method
		c.RoutePath = path
		return nil
	}
	if engine.DebugMode {
		log.Infof("[zouwu engine]add method %s path: %s\n", method, path)
	}
	handlers = append([]HandlerFunc{prelude}, handlers...)
	root.addRoute(path, handlers)
}

// MethodConfig is
type MethodConfig struct {
	Timeout time.Duration
}

// Start listen and serve bm engine by given DSN.
func (engine *Engine) Start() error {
	conf := engine.conf
	l, err := net.Listen(conf.Network, conf.Addr)
	if err != nil {
		panic(errors.Wrapf(err, "[zouwu Engine]: listen tcp: %s", conf.Addr))
	}

	log.Infof("[zouwu Engine]: start http listen addr: %s", l.Addr().String())
	server := &fasthttp.Server{
		ReadTimeout:  time.Duration(conf.ReadTimeout),
		WriteTimeout: time.Duration(conf.WriteTimeout),
	}
	if err := engine.RunServer(server, l); err != nil {
		if errors.Cause(err) == http.ErrServerClosed {
			log.Info("[zouwu Engine]: server closed")
			return nil
		}
		panic(errors.Wrapf(err, "[zouwu Engine]: engine.ListenServer(%+v, %+v)", server, l))
	}
	return nil
}

// AcquireCtx get context from pool and transform fasthttp.RequestCtx to zouwu.Context
func (engine *Engine) AcquireCtx(rctx *fasthttp.RequestCtx) *Context {
	ctx := engine.pool.Get().(*Context)
	ctx.reset()
	ctx.engine = engine
	ctx.Ctx = rctx
	return ctx
}

// ReleaseCtx reset context and put back pool
func (engine *Engine) ReleaseCtx(ctx *Context) {
	ctx.reset()
	engine.pool.Put(ctx)
}

func (engine *Engine) handler(rctx *fasthttp.RequestCtx) {
	ctx := engine.AcquireCtx(rctx)
	engine.prepareHanlder(ctx)
	ctx.Next()
	engine.ReleaseCtx(ctx)
}

func (engine *Engine) prepareHanlder(ctx *Context) {
	method := string(ctx.Ctx.Method())
	rPath := string(ctx.Ctx.Request.URI().Path())
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != method {
			continue
		}
		root := t[i].root
		// Find route in tree
		handlers, params, _ := root.getValue(rPath, ctx.Params, false)
		if handlers != nil {
			ctx.handlers = handlers
			ctx.Params = params
			return
		}
		break
	}

	if engine.HandleMethodNotAllowed {
		for _, tree := range engine.trees {
			if tree.method == method {
				continue
			}
			if handlers, _, _ := tree.root.getValue(rPath, nil, false); handlers != nil {
				ctx.handlers = engine.allNoMethod
				return
			}
		}
	}
	ctx.handlers = engine.allNoRoute
}

// RunServer will serve and start listening HTTP requests by given server and listener.
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunServer(server *fasthttp.Server, l net.Listener) (err error) {
	engine.server = server
	engine.server.Handler = engine.handler
	if err = server.Serve(l); err != nil {
		err = errors.Wrapf(err, "listen server: %+v/%+v", server, l)
		return
	}
	return
}

// Run will run server with address
func (engine *Engine) Run(address string) error {
	engine.conf.Addr = address
	return engine.Start()
}

//newContext for sync.pool
func (engine *Engine) newContext() *Context {
	return &Context{engine: engine}
}

// SetConfig is used to set the engine configuration.
// Only the valid config will be loaded.
func (engine *Engine) SetConfig(conf *ServerConfig) (err error) {
	if conf.Timeout <= 0 {
		return errors.New("[zouwu Engine]: config timeout must greater than 0")
	}
	if conf.Network == "" {
		conf.Network = "tcp"
	}
	engine.lock.Lock()
	engine.conf = conf
	engine.lock.Unlock()
	return
}

// NoRoute adds handlers for NoRoute. It return a 404 code by default.
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// NoMethod sets the handlers called when... TODO.
func (engine *Engine) NoMethod(handlers ...HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

// Use attaches a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (engine *Engine) Use(middleware ...HandlerFunc) IRoutes {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
	return engine
}
