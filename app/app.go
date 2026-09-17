// Package app owns Blue's HTTP server, router, and application middleware.
package app

import (
	"net/http"
	"strings"
	"sync"

	bluehttp "github.com/yasserelgammal/blue-go/http"
	"github.com/yasserelgammal/blue-go/router"
)

// App is an HTTP handler containing routes, middleware, and an error handler.
type App struct {
	router *router.Router

	mu           sync.RWMutex
	middleware   []Middleware
	errorHandler ErrorHandler
}

// New creates an empty application.
func New() *App {
	return &App{router: router.New(), errorHandler: bluehttp.DefaultErrorHandler}
}

// Use adds application middleware in execution order.
func (a *App) Use(middleware ...Middleware) {
	for _, item := range middleware {
		if item == nil {
			panic("blue: nil middleware")
		}
	}
	a.mu.Lock()
	a.middleware = append(a.middleware, middleware...)
	a.mu.Unlock()
}

// SetErrorHandler replaces the centralized error handler.
func (a *App) SetErrorHandler(handler ErrorHandler) {
	if handler == nil {
		panic("blue: nil error handler")
	}
	a.mu.Lock()
	a.errorHandler = handler
	a.mu.Unlock()
}

// Handle registers a handler for an HTTP method and route pattern.
func (a *App) Handle(method, path string, handler HandlerFunc) {
	a.router.Register(method, path, handler)
}

func (a *App) GET(path string, handler HandlerFunc)    { a.Handle(http.MethodGet, path, handler) }
func (a *App) POST(path string, handler HandlerFunc)   { a.Handle(http.MethodPost, path, handler) }
func (a *App) PUT(path string, handler HandlerFunc)    { a.Handle(http.MethodPut, path, handler) }
func (a *App) PATCH(path string, handler HandlerFunc)  { a.Handle(http.MethodPatch, path, handler) }
func (a *App) DELETE(path string, handler HandlerFunc) { a.Handle(http.MethodDelete, path, handler) }

// Group creates a route group with a common path prefix.
func (a *App) Group(prefix string) *router.Group { return a.router.Group(prefix) }

// ServeHTTP makes App compatible with net/http and httptest.
func (a *App) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	a.mu.RLock()
	applicationMiddleware := append([]Middleware(nil), a.middleware...)
	errorHandler := a.errorHandler
	a.mu.RUnlock()

	matched, allowed := a.router.MatchRequest(request.Method, request.URL.Path)
	var params map[string]string
	if matched != nil {
		params = matched.Params
	}
	context := bluehttp.NewContext(w, request, params)

	var handler HandlerFunc
	middleware := applicationMiddleware
	switch {
	case matched != nil:
		handler = matched.Handler
		middleware = append(middleware, matched.Middleware...)
	case len(allowed) > 0:
		context.Response.Header().Set("Allow", strings.Join(allowed, ", "))
		handler = func(*bluehttp.Context) error {
			return bluehttp.MethodNotAllowed("Method Not Allowed")
		}
	default:
		handler = func(*bluehttp.Context) error { return bluehttp.NotFound("Not Found") }
	}
	if err := applyMiddleware(handler, middleware...)(context); err != nil {
		errorHandler(context, err)
	}
}

// Run starts an HTTP server at addr and blocks until it stops.
func (a *App) Run(addr string) error {
	server := &http.Server{Addr: addr, Handler: a}
	return server.ListenAndServe()
}

func applyMiddleware(handler HandlerFunc, middleware ...Middleware) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		if middleware[i] == nil {
			panic("blue: nil middleware")
		}
		handler = middleware[i](handler)
	}
	return handler
}
