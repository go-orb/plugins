// Package chi provides a Chi implementation of the router interface.
package chi

import (
	"net/http"

	"github.com/go-chi/chi"

	"github.com/go-orb/plugins/server/http/router"
)

var _ router.Router = (*Router)(nil)

func init() {
	router.Plugins.Add("chi", newRouter)
}

// Router router implements the router interface.
type Router struct {
	*chi.Mux
}

func newRouter() router.Router {
	return NewRouter()
}

// NewRouter creates a new Chi router.
func NewRouter() *Router {
	r := Router{
		Mux: chi.NewRouter(),
	}

	return &r
}

// Middlewares returns the list of currently registered middlewares.
func (r *Router) Middlewares() []func(http.Handler) http.Handler {
	m := r.Mux.Middlewares()

	return []func(http.Handler) http.Handler(m)
}

// Routes returns a tree of registered routes.
func (r *Router) Routes() []router.Route {
	return getRoutes(r.Mux.Routes())
}

func getRoutes(routes []chi.Route) []router.Route {
	out := make([]router.Route, 0, len(routes))

	for _, route := range routes {
		var subroutes []router.Route
		if route.SubRoutes != nil {
			subroutes = getRoutes(route.SubRoutes.Routes())
		}

		out = append(out, router.Route{
			SubRoutes: subroutes,
			Handlers:  route.Handlers,
			Pattern:   route.Pattern,
		})
	}

	return out
}
