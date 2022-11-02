package chi

import (
	"net/http"

	"github.com/go-chi/chi"
)

type ChiRouter struct {
	*chi.Mux
}

func ProvideChiRouter() *ChiRouter {
	r := ChiRouter{
		Mux: chi.NewRouter(),
	}

	return &r
}

func (r *ChiRouter) Middlewares() []func(http.Handler) http.Handler {
	m := r.Mux.Middlewares()

	return []func(http.Handler) http.Handler(m)
}
