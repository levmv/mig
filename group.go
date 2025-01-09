package mig

import (
	"net/http"
)

type RouteGroup struct {
	Mig          *Mig
	middlewares  []MiddlewareFunc
	ParentRouter *RouteGroup
}

// Create new routing group. Note that group's pattern must end with slash.
func (rg *RouteGroup) Group(middlewares ...MiddlewareFunc) *RouteGroup {
	g := &RouteGroup{
		Mig:          rg.Mig,
		ParentRouter: rg,
	}
	g.middlewares = append(g.middlewares, middlewares...)

	return g
}

func (rg *RouteGroup) Use(middlewares ...MiddlewareFunc) {
	rg.middlewares = append(rg.middlewares, middlewares...)
}

// Handle is used to register new route handler.
func (rg *RouteGroup) Handle(pattern string, handler Handler) {
	var fhandler Handler
	rg.Mig.Mux.HandleFunc(pattern, func(rw http.ResponseWriter, r *http.Request) {
		if fhandler == nil {
			fhandler = handler
			prt := rg
			for {
				if len(prt.middlewares) > 0 {
					for i := len(prt.middlewares) - 1; i >= 0; i-- {
						fhandler = prt.middlewares[i](fhandler)
					}
				}
				if prt.ParentRouter == nil {
					break
				}
				prt = prt.ParentRouter
			}
		}

		rg.Mig.Execute(fhandler, rw, r)
	})
}
