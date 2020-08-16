package zouwu

import (
	"math"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/valyala/fasthttp"
)

const (
	_abortIndex int8 = math.MaxInt8 / 2
)

var json = jsoniter.Config{
	EscapeHTML:             true,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
	UseNumber:              true, // 避免 float 转换时精度丢失
}.Froze()

// Context 定义
type Context struct {
	Ctx *fasthttp.RequestCtx

	index    int8
	handlers []HandlerFunc

	Keys map[string]interface{}
	mu   sync.RWMutex

	Error error

	method string
	engine *Engine

	RoutePath string

	Params Params
	err    error
}

/************************************/
/********** CONTEXT CREATION ********/
/************************************/
func (c *Context) reset() {
	c.Ctx = nil
	c.index = -1
	c.handlers = nil
	c.Keys = nil
	c.Error = nil
	c.method = ""
	c.RoutePath = ""
	c.err = nil
	c.Params = c.Params[0:0]
}

/************************************/
/*********** FLOW CONTROL ***********/
/************************************/

// Next should be used only inside middleware.
// It executes the pending handlers in the chain inside the calling handler.
// See example in godoc.
func (c *Context) Next() error {
	c.index++
	for c.index < int8(len(c.handlers)) {
		err := c.handlers[c.index](c)
		if err != nil {
			return c.Errors(err)
		}
		c.index++
	}
	return nil
}

// Abort prevents pending handlers from being called. Note that this will not stop the current handler.
// Let's say you have an authorization middleware that validates that the current request is authorized.
// If the authorization fails (ex: the password does not match), call Abort to ensure the remaining handlers
// for this request are not called.
func (c *Context) Abort() {
	c.index = _abortIndex
}

/************************************/
/******** METADATA MANAGEMENT********/
/************************************/

// Set is used to store a new key/value pair exclusively for this context.
// It also lazy initializes  c.Keys if it was not used previously.
func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}

	c.Keys[key] = value
	c.mu.Unlock()
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (c *Context) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.Keys[key]
	c.mu.RUnlock()
	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (c *Context) MustGet(key string) interface{} {
	if value, exists := c.Get(key); exists {
		return value
	}
	panic("Key \"" + key + "\" does not exist")
}

// GetString returns the value associated with the key as a string.
func (c *Context) GetString(key string) (s string) {
	if val, ok := c.Get(key); ok && val != nil {
		s, _ = val.(string)
	}
	return
}

// GetBool returns the value associated with the key as a boolean.
func (c *Context) GetBool(key string) (b bool) {
	if val, ok := c.Get(key); ok && val != nil {
		b, _ = val.(bool)
	}
	return
}

// GetInt returns the value associated with the key as an integer.
func (c *Context) GetInt(key string) (i int) {
	if val, ok := c.Get(key); ok && val != nil {
		i, _ = val.(int)
	}
	return
}

// GetInt64 returns the value associated with the key as an integer.
func (c *Context) GetInt64(key string) (i64 int64) {
	if val, ok := c.Get(key); ok && val != nil {
		i64, _ = val.(int64)
	}
	return
}

// GetFloat64 returns the value associated with the key as a float64.
func (c *Context) GetFloat64(key string) (f64 float64) {
	if val, ok := c.Get(key); ok && val != nil {
		f64, _ = val.(float64)
	}
	return
}

// GetTime returns the value associated with the key as time.
func (c *Context) GetTime(key string) (t time.Time) {
	if val, ok := c.Get(key); ok && val != nil {
		t, _ = val.(time.Time)
	}
	return
}

// GetDuration returns the value associated with the key as a duration.
func (c *Context) GetDuration(key string) (d time.Duration) {
	if val, ok := c.Get(key); ok && val != nil {
		d, _ = val.(time.Duration)
	}
	return
}

// GetStringSlice returns the value associated with the key as a slice of strings.
func (c *Context) GetStringSlice(key string) (ss []string) {
	if val, ok := c.Get(key); ok && val != nil {
		ss, _ = val.([]string)
	}
	return
}

// GetStringMap returns the value associated with the key as a map of interfaces.
func (c *Context) GetStringMap(key string) (sm map[string]interface{}) {
	if val, ok := c.Get(key); ok && val != nil {
		sm, _ = val.(map[string]interface{})
	}
	return
}

// GetStringMapString returns the value associated with the key as a map of strings.
func (c *Context) GetStringMapString(key string) (sms map[string]string) {
	if val, ok := c.Get(key); ok && val != nil {
		sms, _ = val.(map[string]string)
	}
	return
}

// GetStringMapStringSlice returns the value associated with the key as a map to a slice of strings.
func (c *Context) GetStringMapStringSlice(key string) (smss map[string][]string) {
	if val, ok := c.Get(key); ok && val != nil {
		smss, _ = val.(map[string][]string)
	}
	return
}

/************************************/
/************ INPUT DATA ************/
/************************************/

// URLParam returns the value of the URL param.
// It is a shortcut for c.Params.ByName(key)
//     router.GET("/user/:id", func(c *gin.Context) {
//         // a GET request to /user/john
//         id := c.Param("id") // id == "john"
//     })
func (c *Context) URLParam(key string) string {
	return c.Params.ByName(key)
}

// URLParamInt64 return param as int64
func (c *Context) URLParamInt64(key string) int64 {
	return cast.ToInt64(c.Params.ByName(key))
}

// URLParamUint64 return param as uint64
func (c *Context) URLParamUint64(key string) uint64 {
	return cast.ToUint64(c.Params.ByName(key))
}

// URLParamInt32 return param as int32
func (c *Context) URLParamInt32(key string) int32 {
	return cast.ToInt32(c.Params.ByName(key))
}

// URLParamUint32 return param as uint32
func (c *Context) URLParamUint32(key string) uint32 {
	return cast.ToUint32(c.Params.ByName(key))
}

// URLParamInt16 return param as int16
func (c *Context) URLParamInt16(key string) int16 {
	return cast.ToInt16(c.Params.ByName(key))
}

// URLParamUint16 return param as uint16
func (c *Context) URLParamUint16(key string) uint16 {
	return cast.ToUint16(c.Params.ByName(key))
}

// URLParamInt8 return param as int8
func (c *Context) URLParamInt8(key string) int8 {
	return cast.ToInt8(c.Params.ByName(key))
}

// URLParamUint8 return param as uint8
func (c *Context) URLParamUint8(key string) uint8 {
	return cast.ToUint8(c.Params.ByName(key))
}

// URLParamInt return param as int
func (c *Context) URLParamInt(key string) int {
	return cast.ToInt(c.Params.ByName(key))
}

// URLParamUint return param as int
func (c *Context) URLParamUint(key string) uint {
	return cast.ToUint(c.Params.ByName(key))
}

// Query returns the keyed url query value if it exists,
// otherwise it returns an empty string `("")`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//     GET /path?id=1234&name=Manu&value=
// 	   c.Query("id") == "1234"
// 	   c.Query("name") == "Manu"
// 	   c.Query("value") == ""
// 	   c.Query("wtf") == ""
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

// DefaultQuery returns the keyed url query value if it exists,
// otherwise it returns the specified defaultValue string.
// See: Query() and GetQuery() for further information.
//     GET /?name=Manu&lastname=
//     c.DefaultQuery("name", "unknown") == "Manu"
//     c.DefaultQuery("id", "none") == "none"
//     c.DefaultQuery("lastname", "none") == ""
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// DefaultQueryInt64 returns keyed url query value as int64 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryInt64(key string, defaultValue int64) int64 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToInt64(value)
	}
	return defaultValue
}

// DefaultQueryUint64 returns keyed url query value as uint64 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryUint64(key string, defaultValue uint64) uint64 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToUint64(value)
	}
	return defaultValue
}

// DefaultQueryInt32 returns keyed url query value as int32 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryInt32(key string, defaultValue int32) int32 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToInt32(value)
	}
	return defaultValue
}

// DefaultQueryUint32 returns keyed url query value as uint32 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryUint32(key string, defaultValue uint32) uint32 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToUint32(value)
	}
	return defaultValue
}

// DefaultQueryInt16 returns keyed url query value as int16 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryInt16(key string, defaultValue int16) int16 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToInt16(value)
	}
	return defaultValue
}

// DefaultQueryUint16 returns keyed url query value as uint16 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryUint16(key string, defaultValue uint16) uint16 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToUint16(value)
	}
	return defaultValue
}

// DefaultQueryInt8 returns keyed url query value as int8 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryInt8(key string, defaultValue int8) int8 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToInt8(value)
	}
	return defaultValue
}

// DefaultQueryUint8 returns keyed url query value as uint8 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryUint8(key string, defaultValue uint8) uint8 {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToUint8(value)
	}
	return defaultValue
}

// DefaultQueryInt returns keyed url query value as int8 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryInt(key string, defaultValue int) int {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToInt(value)
	}
	return defaultValue
}

// DefaultQueryUint returns keyed url query value as int8 it it existed,
// otherwise it returns the specified defaultValue
func (c *Context) DefaultQueryUint(key string, defaultValue uint) uint {
	if value, ok := c.GetQuery(key); ok {
		return cast.ToUint(value)
	}
	return defaultValue
}

// GetQuery is like Query(), it returns the keyed url query value
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//     GET /?name=Manu&lastname=
//     ("Manu", true) == c.GetQuery("name")
//     ("", false) == c.GetQuery("id")
//     ("", true) == c.GetQuery("lastname")
func (c *Context) GetQuery(key string) (string, bool) {
	values := c.Ctx.QueryArgs().PeekMulti(key)
	if len(values) != 0 {
		return string(values[0]), true
	}
	return "", false
}

// GetRequestBody return request body
func (c *Context) GetRequestBody() []byte {
	return c.Ctx.PostBody()
}

// Status sets the HTTP response code.
func (c *Context) Status(code int) {
	c.Ctx.SetStatusCode(code)
}

// Errors render error
func (c *Context) Errors(err error) error {
	if c.engine.errorHandler != nil {
		c.engine.errorHandler(c, err)
	} else {
		defaultErrorHandler(c, err)
	}
	c.Abort()
	return nil
}

// JSON render json
func (c *Context) JSON(data interface{}) error {
	raw, err := json.Marshal(data)
	// Check for errors
	if err != nil {
		return err
	}
	// Set http headers
	c.Ctx.Response.Header.SetContentType(MIMEApplicationJSON)
	c.Ctx.Response.SetBodyRaw(raw)
	return nil
}

// JSON render string
func (c *Context) String(data string) error {
	c.Ctx.Response.Header.SetContentType(MIMETextPlainCharsetUTF8)
	c.Ctx.Response.SetBodyString(data)
	return nil
}

// Bytes writes some data into the body stream and updates the HTTP code.
func (c *Context) Bytes(code int, contentType string, data []byte) {
	c.Ctx.Response.Header.SetContentType(MIMETextPlain)
	c.Ctx.Response.SetBodyRaw(data)
	c.Status(code)
}
