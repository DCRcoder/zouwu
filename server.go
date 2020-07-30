package zouwu

import (
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/ngaut/log"
	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
)

var (
	default405Body = []byte("405 method not allowed")
	default404Body = []byte("404 page not found")
)

// HandlerFunc http request handler function.
type HandlerFunc func(ctx *Context)

// Handler responds to an HTTP request.
type Handler interface {
	ServeHTTP(c *Context)
}

// ServeHTTP calls f(ctx).
func (f HandlerFunc) ServeHTTP(c *Context) {
	f(c)
}

type injection struct {
	pattern  *regexp.Regexp
	handlers []HandlerFunc
}

// ServerConfig is the bm server config model
type ServerConfig struct {
	Network      string        `dsn:"network"`
	Addr         string        `dsn:"address"`
	Timeout      time.Duration `dsn:"query.timeout"`
	ReadTimeout  time.Duration `dsn:"query.readTimeout"`
	WriteTimeout time.Duration `dsn:"query.writeTimeout"`
}

// Engine service engine
type Engine struct {
	RouterGroup

	lock sync.RWMutex
	conf *ServerConfig

	address       string
	pcLock        sync.RWMutex
	methodConfigs map[string]*MethodConfig

	injections []injection

	trees     methodTrees
	server    *fasthttp.Server                  // store *http.Server
	metastore map[string]map[string]interface{} // metastore is the path as key and the metadata of this path as value, it export via /metadata

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

	pool sync.Pool
}

// SetMethodConfig is used to set config on specified path
func (engine *Engine) SetMethodConfig(path string, mc *MethodConfig) {
	engine.pcLock.Lock()
	engine.methodConfigs[path] = mc
	engine.pcLock.Unlock()
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
	if _, ok := engine.metastore[path]; !ok {
		engine.metastore[path] = make(map[string]interface{})
	}
	engine.metastore[path]["method"] = method
	root := engine.trees.get(method)
	if root == nil {
		root = new(node)
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}

	prelude := func(c *Context) {
		c.method = method
		c.RoutePath = path
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
		return errors.Wrapf(err, "[zouwu Engine]: listen tcp: %s", conf.Addr)
	}

	log.Info("[zouwu Engine]: start http listen addr: %s", l.Addr().String())
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

// RunServer will serve and start listening HTTP requests by given server and listener.
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunServer(server *fasthttp.Server, l net.Listener) (err error) {
	engine.server = server
	if err = server.Serve(l); err != nil {
		err = errors.Wrapf(err, "listen server: %+v/%+v", server, l)
		return
	}
	return
}

// Run will run server with address
func (engine *Engine) Run(address string) error {
	engine.SetConfig(
		&ServerConfig{
			Addr:         address,
			Timeout:      5 * time.Second,
			WriteTimeout: 5 * time.Second,
			ReadTimeout:  5 * time.Second,
		},
	)
	return engine.Start()
}

// NewServer returns a new blank Engine instance without any middleware attached.
func NewServer(conf *ServerConfig) *Engine {
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		address:                "127.0.0.1:8080",
		trees:                  make(methodTrees, 0, 9),
		metastore:              make(map[string]map[string]interface{}),
		methodConfigs:          make(map[string]*MethodConfig),
		HandleMethodNotAllowed: true,
		injections:             make([]injection, 0),
	}
	if err := engine.SetConfig(conf); err != nil {
		panic(err)
	}
	engine.pool.New = func() interface{} {
		return engine.newContext()
	}
	engine.RouterGroup.engine = engine
	engine.NoRoute(func(c *Context) {
		c.Bytes(404, "text/plain", default404Body)
		c.Abort()
	})
	engine.NoMethod(func(c *Context) {
		c.Bytes(405, "text/plain", default405Body)
		c.Abort()
	})
	return engine
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
