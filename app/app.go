// Package app owns Blue's HTTP server, router, and application middleware.
package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	bluehttp "github.com/yasserelgammal/blue-go/http"
	"github.com/yasserelgammal/blue-go/router"
)

// App is an HTTP handler containing routes, middleware, and an error handler.
type App struct {
	router *router.Router

	mu           sync.RWMutex
	middleware   []Middleware
	errorHandler ErrorHandler
	formatter    bluehttp.ResponseFormatter
	serverConfig ServerConfig
	jsonConfig   bluehttp.JSONConfig

	serverMu sync.Mutex
	server   *http.Server
}

// New creates an empty application.
func New() *App {
	return &App{
		router:       router.New(),
		errorHandler: bluehttp.DefaultErrorHandler,
		formatter:    bluehttp.DefaultResponseFormatter,
		serverConfig: DefaultServerConfig(),
		jsonConfig:   bluehttp.DefaultJSONConfig(),
	}
}

// SetServerConfig configures the HTTP server created by Start and Run. Changes
// take effect the next time the server starts.
func (a *App) SetServerConfig(config ServerConfig) {
	validateServerConfig(config)
	a.mu.Lock()
	a.serverConfig = config
	a.mu.Unlock()
}

// SetJSONConfig configures JSON request binding for every request. Middleware
// may override the configuration for one request with Context.SetJSONConfig.
func (a *App) SetJSONConfig(config bluehttp.JSONConfig) {
	a.mu.Lock()
	a.jsonConfig = config
	a.mu.Unlock()
}

// SetResponseFormatter customizes the JSON shape produced by Respond,
// RespondWithMessage, Paginated, and the default error handler.
func (a *App) SetResponseFormatter(formatter bluehttp.ResponseFormatter) {
	if formatter == nil {
		panic("blue: nil response formatter")
	}
	a.mu.Lock()
	a.formatter = formatter
	a.mu.Unlock()
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
	formatter := a.formatter
	jsonConfig := a.jsonConfig
	a.mu.RUnlock()

	matched, allowed := a.router.MatchRequest(request.Method, request.URL.Path)
	var params map[string]string
	if matched != nil {
		params = matched.Params
	}
	context := bluehttp.NewContext(w, request, params)
	context.SetResponseFormatter(formatter)
	context.SetJSONConfig(jsonConfig)

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

// Run starts an HTTP server at addr and blocks until it stops. It is an alias
// for Start and returns nil after a graceful shutdown.
func (a *App) Run(addr string) error { return a.Start(addr) }

// Start starts an HTTP server at addr and blocks until it stops. Call Shutdown
// from another goroutine to stop it gracefully.
func (a *App) Start(addr string) error {
	server, listener, err := a.prepareServer(addr)
	if err != nil {
		return err
	}
	return a.serve(server, listener)
}

func (a *App) prepareServer(addr string) (*http.Server, net.Listener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	a.mu.RLock()
	config := a.serverConfig
	a.mu.RUnlock()
	server := &http.Server{
		Addr:              addr,
		Handler:           a,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
	}
	a.serverMu.Lock()
	if a.server != nil {
		a.serverMu.Unlock()
		_ = listener.Close()
		return nil, nil, fmt.Errorf("blue: server is already running")
	}
	a.server = server
	a.serverMu.Unlock()
	return server, listener, nil
}

func (a *App) serve(server *http.Server, listener net.Listener) error {
	err := server.Serve(listener)
	a.serverMu.Lock()
	if a.server == server {
		a.server = nil
	}
	a.serverMu.Unlock()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops the active server without interrupting active
// connections. It is safe to call when no server is running.
func (a *App) Shutdown(ctx context.Context) error {
	a.serverMu.Lock()
	server := a.server
	a.serverMu.Unlock()
	if server == nil {
		return nil
	}
	return server.Shutdown(ctx)
}

// RunWithGracefulShutdown starts the server and handles Ctrl+C and SIGTERM.
// Active requests receive up to ten seconds to finish.
func (a *App) RunWithGracefulShutdown(addr string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		// Restore the default signal behavior so a second signal can force an
		// immediate exit while graceful shutdown is in progress.
		stop()
	}()
	return a.runUntil(ctx, addr, 10*time.Second)
}

func (a *App) runUntil(ctx context.Context, addr string, timeout time.Duration) error {
	server, listener, err := a.prepareServer(addr)
	if err != nil {
		return err
	}
	serverResult := make(chan error, 1)
	go func() { serverResult <- a.serve(server, listener) }()

	select {
	case err := <-serverResult:
		return err
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := a.Shutdown(shutdownContext); err != nil {
			return err
		}
		return <-serverResult
	}
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
