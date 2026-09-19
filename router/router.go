// Package router registers and matches Blue routes.
package router

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	bluehttp "github.com/yasserelgammal/blue-go/v2/http"
)

// HandlerFunc and Middleware mirror the framework HTTP primitives.
type HandlerFunc = bluehttp.HandlerFunc
type Middleware = bluehttp.Middleware

// Match is the result of matching a request path and method.
type Match struct {
	Handler    HandlerFunc
	Params     map[string]string
	Middleware []Middleware
}

// Router stores registered routes and safely matches concurrent requests.
type Router struct {
	mu     sync.RWMutex
	routes []*route
}

// New creates an empty router.
func New() *Router { return &Router{} }

// Register adds a route and panics for invalid or ambiguous declarations.
func (r *Router) Register(method, path string, handler HandlerFunc) {
	r.register(method, path, handler, nil)
}

func (r *Router) register(method, path string, handler HandlerFunc, group *Group) {
	if handler == nil {
		panic("blue: nil route handler")
	}
	method = validateMethod(method)
	path, segments := parseRoute(path)
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.routes {
		if existing.method == method && sameShape(existing.segments, segments) {
			panic("blue: duplicate or ambiguous route " + method + " " + path)
		}
	}
	r.routes = append(r.routes, &route{
		method: method, segments: segments, handler: handler,
		group: group, order: len(r.routes),
	})
}

// MatchRequest returns the best route match and all methods registered for the
// path. The methods are used to distinguish 404 from 405 responses.
func (r *Router) MatchRequest(method, path string) (*Match, []string) {
	r.mu.RLock()
	routes := append([]*route(nil), r.routes...)
	r.mu.RUnlock()

	var selected *route
	var selectedParams map[string]string
	bestStatic := -1
	allowed := make(map[string]struct{})
	for _, candidate := range routes {
		params, matches := candidate.match(path)
		if !matches {
			continue
		}
		allowed[candidate.method] = struct{}{}
		if candidate.method != method {
			continue
		}
		score := staticCount(candidate.segments)
		if selected == nil || score > bestStatic || (score == bestStatic && candidate.order < selected.order) {
			selected, selectedParams, bestStatic = candidate, params, score
		}
	}
	methods := make([]string, 0, len(allowed))
	for allowedMethod := range allowed {
		methods = append(methods, allowedMethod)
	}
	sort.Strings(methods)
	if selected == nil {
		return nil, methods
	}
	return &Match{
		Handler: selected.handler, Params: selectedParams,
		Middleware: selected.groupMiddleware(),
	}, methods
}

func validateMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" || strings.ContainsAny(method, " \t\r\n") {
		panic(fmt.Sprintf("blue: invalid HTTP method %q", method))
	}
	return method
}
