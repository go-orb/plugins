package http

import (
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"

	"github.com/go-orb/go-orb/log"
	"github.com/julienschmidt/httprouter"
)

// Router is a router based on httprouter.
type Router struct {
	logger     log.Logger
	routes     map[string]http.HandlerFunc
	httprouter *httprouter.Router
}

// NewRouter creates a new router.
func NewRouter(logger log.Logger) *Router {
	r := &Router{
		logger:     logger,
		routes:     map[string]http.HandlerFunc{},
		httprouter: httprouter.New(),
	}

	r.httprouter.PanicHandler = func(w http.ResponseWriter, _ *http.Request, i interface{}) {
		r.logger.Error("Panic", slog.String("error", fmt.Sprint(i)))
		http.Error(w, fmt.Sprint(i), http.StatusInternalServerError)
	}

	// Performance optimizations for httprouter
	r.httprouter.RedirectTrailingSlash = false
	r.httprouter.RedirectFixedPath = false
	r.httprouter.HandleMethodNotAllowed = false
	r.httprouter.HandleOPTIONS = false

	return r
}

// Routes returns the list of routes.
func (r *Router) Routes() []string {
	return slices.Collect(maps.Keys(r.routes))
}

// Post registers a new route for POST requests.
func (r *Router) Post(path string, handler http.HandlerFunc) {
	r.routes[path] = handler
	r.httprouter.HandlerFunc(http.MethodPost, path, handler)
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.httprouter.ServeHTTP(w, req)
}
