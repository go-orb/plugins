package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-orb/go-orb/util/orberrors"
)

// Errors.
var (
	ErrNotHTTPServer = errors.New("server provider is not of type *http.Server")
)

// NewGRPCHandler will wrap a gRPC function with a HTTP handler.
func NewGRPCHandler[Tin any, Tout any](srv *ServerHTTP, f func(context.Context, *Tin) (*Tout, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := new(Tin)

		if _, err := srv.decodeBody(w, r, in); err != nil {
			srv.Logger.Error("failed to decode body", "error", err)
			WriteError(w, err)

			return
		}

		out, err := f(r.Context(), in)
		if err != nil {
			srv.Logger.Error("RPC request failed", "error", err)
			WriteError(w, err)

			return
		}

		if err := srv.encodeBody(w, r, out); err != nil {
			srv.Logger.Error("failed to encode body", "error", err)
			WriteError(w, err)

			return
		}
	}
}

// WriteError returns an error response to the HTTP request.
func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	orbe := orberrors.From(err)
	w.WriteHeader(orbe.Code)
	fmt.Fprintf(w, orbe.Error())
}
