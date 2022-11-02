package http

import (
	"context"
	"fmt"
	"net/http"
)

// NewGRPCHandler will wrap a gRPC function with a HTTP handler.
func NewGRPCHandler[Tin any, Tout any](s ServerHTTP, f func(context.Context, *Tin) (*Tout, error)) http.HandlerFunc {
	srv := s.(*Server)

	return func(w http.ResponseWriter, r *http.Request) {
		in := new(Tin)

		if _, err := srv.decodeBody(w, r, in); err != nil {
			WriteError(w, err)
			return
		}

		out, err := f(r.Context(), in)
		if err != nil {
			WriteError(w, err)
			return
		}

		if err := srv.encodeBody(w, r, out); err != nil {
			WriteError(w, err)
			return
		}
	}
}

func WriteError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, err.Error())
}
