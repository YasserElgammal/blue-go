// Package http contains Blue's request context and HTTP response primitives.
package http

import stdhttp "net/http"

// Context contains the request, response writer, and path parameters captured
// while routing. A Context is valid only for the lifetime of its handler.
type Context struct {
	Request  *stdhttp.Request
	Response stdhttp.ResponseWriter

	params            map[string]string
	requestID         string
	state             *responseState
	responseFormatter ResponseFormatter
	jsonConfig        JSONConfig
}

// NewContext constructs a context for an incoming request.
func NewContext(w stdhttp.ResponseWriter, r *stdhttp.Request, params map[string]string) *Context {
	state, ok := w.(*responseState)
	if !ok {
		state = &responseState{ResponseWriter: w}
	}
	return &Context{
		Request:           r,
		Response:          state,
		params:            params,
		state:             state,
		responseFormatter: DefaultResponseFormatter,
		jsonConfig:        DefaultJSONConfig(),
	}
}

// Param returns a named path parameter, or an empty string when it is absent.
func (c *Context) Param(name string) string { return c.params[name] }

// Query returns the first value for a query parameter.
func (c *Context) Query(name string) string { return c.Request.URL.Query().Get(name) }

// RequestID returns the identifier assigned to the current request, or an
// empty string when request ID middleware is not installed.
func (c *Context) RequestID() string { return c.requestID }

// SetRequestID assigns an identifier to the current request. Applications
// normally use RequestID middleware instead of calling this method directly.
func (c *Context) SetRequestID(id string) { c.requestID = id }

// Status returns the response status, or zero before the response is committed.
func (c *Context) Status() int { return c.state.status }

// Committed reports whether response headers have been written.
func (c *Context) Committed() bool { return c.state.committed }

// SetResponseFormatter changes how unified responses are represented for this
// request. Applications normally configure this once with App.SetResponseFormatter.
func (c *Context) SetResponseFormatter(formatter ResponseFormatter) {
	if formatter == nil {
		panic("blue: nil response formatter")
	}
	c.responseFormatter = formatter
}

// SetJSONConfig changes JSON binding behavior for this request.
func (c *Context) SetJSONConfig(config JSONConfig) { c.jsonConfig = config }
