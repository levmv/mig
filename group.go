package mig

import "net/http"

// RouteGroup represents a group of routes with shared middleware.
type RouteGroup struct {
	Mig          *Mig
	Prefix       string
	middlewares  []MiddlewareFunc
	ParentRouter *RouteGroup
}

// Group creates a new routing group with a common prefix and shared middleware.
func (rg *RouteGroup) Group(prefix string, middlewares ...MiddlewareFunc) *RouteGroup {
	g := &RouteGroup{
		Mig:          rg.Mig,
		Prefix:       rg.Prefix + prefix,
		ParentRouter: rg,
		middlewares:  append([]MiddlewareFunc{}, rg.middlewares...),
	}
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

// Use adds middleware to the route group.
func (rg *RouteGroup) Use(middlewares ...MiddlewareFunc) {
	rg.middlewares = append(rg.middlewares, middlewares...)
}

// HandleRaw registers a handler for a raw, unprocessed http.ServeMux pattern.
// The group's path prefix is NOT applied. All group middleware is applied.
// This is the advanced method for use cases like host-based routing.
func (rg *RouteGroup) HandleRaw(pattern string, handler Handler) {
	fhandler := handler
	for i := len(rg.middlewares) - 1; i >= 0; i-- {
		fhandler = rg.middlewares[i](fhandler)
	}
	rg.Mig.Mux.HandleFunc(pattern, func(rw http.ResponseWriter, r *http.Request) {
		rg.Mig.Execute(fhandler, rw, r)
	})
}

// Handle registers a handler for a given HTTP method and path.
// It correctly applies the group's path prefix.
func (rg *RouteGroup) Handle(method, path string, handler Handler) {
	fullPath := rg.Prefix + path
	var pattern string

	if method != "" {
		pattern = method + " " + fullPath
	} else {
		pattern = fullPath
	}

	rg.HandleRaw(pattern, handler)
}

// GET registers a GET and HEAD handler for a path.
func (rg *RouteGroup) GET(path string, handler Handler) {
	rg.Handle(http.MethodGet, path, handler)
}

// POST registers a POST handler for a path.
func (rg *RouteGroup) POST(path string, handler Handler) {
	rg.Handle(http.MethodPost, path, handler)
}

// PUT registers a PUT handler for a path.
func (rg *RouteGroup) PUT(path string, handler Handler) {
	rg.Handle(http.MethodPut, path, handler)
}

// DELETE registers a DELETE handler for a path.
func (rg *RouteGroup) DELETE(path string, handler Handler) {
	rg.Handle(http.MethodDelete, path, handler)
}

// Any registers a handler that matches any HTTP method for a path.
func (rg *RouteGroup) Any(path string, handler Handler) {
	rg.Handle("", path, handler)
}
