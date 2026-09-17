package router

import (
	"net/http"
	"sync"
)

// Group registers routes beneath a shared path prefix and middleware chain.
type Group struct {
	router *Router
	parent *Group
	prefix string

	mu         sync.RWMutex
	middleware []Middleware
}

// Group creates a route group on the router.
func (r *Router) Group(prefix string) *Group {
	return &Group{router: r, prefix: groupPrefix(prefix)}
}

func groupPrefix(prefix string) string {
	prefix = cleanPath(prefix)
	if prefix == "/" {
		return ""
	}
	return prefix
}

func joinPaths(prefix, path string) string {
	path = cleanPath(path)
	if path == "/" {
		if prefix == "" {
			return "/"
		}
		return prefix
	}
	return prefix + path
}

// Use adds middleware to every route in the group.
func (g *Group) Use(middleware ...Middleware) {
	for _, item := range middleware {
		if item == nil {
			panic("blue: nil middleware")
		}
	}
	g.mu.Lock()
	g.middleware = append(g.middleware, middleware...)
	g.mu.Unlock()
}

// Handle registers a handler in the group.
func (g *Group) Handle(method, path string, handler HandlerFunc) {
	g.router.register(method, joinPaths(g.prefix, path), handler, g)
}

func (g *Group) GET(path string, handler HandlerFunc)    { g.Handle(http.MethodGet, path, handler) }
func (g *Group) POST(path string, handler HandlerFunc)   { g.Handle(http.MethodPost, path, handler) }
func (g *Group) PUT(path string, handler HandlerFunc)    { g.Handle(http.MethodPut, path, handler) }
func (g *Group) PATCH(path string, handler HandlerFunc)  { g.Handle(http.MethodPatch, path, handler) }
func (g *Group) DELETE(path string, handler HandlerFunc) { g.Handle(http.MethodDelete, path, handler) }

// Group creates a nested route group.
func (g *Group) Group(prefix string) *Group {
	return &Group{router: g.router, parent: g, prefix: joinPaths(g.prefix, prefix)}
}

func (r *route) groupMiddleware() []Middleware {
	if r.group == nil {
		return nil
	}
	var chain []*Group
	for group := r.group; group != nil; group = group.parent {
		chain = append(chain, group)
	}
	var middleware []Middleware
	for i := len(chain) - 1; i >= 0; i-- {
		chain[i].mu.RLock()
		middleware = append(middleware, chain[i].middleware...)
		chain[i].mu.RUnlock()
	}
	return middleware
}
