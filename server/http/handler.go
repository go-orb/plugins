package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
)

var stdHeaders = []string{"Accept", "Accept-Encoding", "Content-Length", "Content-Type", "User-Agent"} //nolint:gochecknoglobals

// Errors.
var (
	ErrNotHTTPServer = errors.New("server provider is not of type *http.Server")
)

// NewGRPCHandler will wrap a gRPC function with a HTTP handler.
func NewGRPCHandler[Tin any, Tout any](
	srv *Server,
	fHandler func(context.Context, *Tin) (*Tout, error),
	service string,
	method string,
) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		inBody := new(Tin)

		if _, err := srv.decodeBody(resp, req, inBody); err != nil {
			srv.logger.Error("failed to decode request body", "error", err)
			WriteError(resp, orberrors.ErrBadRequest.Wrap(err))

			return
		}

		// Copy metadata from req Headers into the req.Context.
		ctx, reqMd := metadata.WithIncoming(req.Context())
		ctx, outMd := metadata.WithOutgoing(ctx)

		for k, v := range req.Header {
			if slices.Contains(stdHeaders, k) {
				continue
			}

			if len(v) == 1 {
				reqMd[strings.ToLower(k)] = v[0]
			} else {
				reqMd[strings.ToLower(k)] = v[0]
				for i := 1; i < len(v); i++ {
					reqMd[strings.ToLower(k)+"-"+strconv.Itoa(i)] = v[i]
				}
			}
		}

		reqMd[metadata.Service] = service
		reqMd[metadata.Method] = method

		// Apply middleware.
		h := func(ctx context.Context, req any) (any, error) {
			return fHandler(ctx, req.(*Tin)) //nolint:errcheck
		}
		for _, m := range srv.config.OptMiddlewares {
			h = m.Call(h)
		}

		// The actual call.
		out, err := h(ctx, inBody)
		if err != nil {
			srv.logger.Error("RPC request failed", "error", err)
			WriteError(resp, err)

			return
		}

		// Write outgoing metadata.
		for k, v := range outMd {
			resp.Header().Set(k, v)
		}

		if err := srv.encodeBody(resp, req, out); err != nil {
			srv.logger.Error("failed to encode response body", "error", err)
			WriteError(resp, err)

			return
		}
	}
}

// WriteError returns an error response to the HTTP request.
func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	if orbe, ok := orberrors.As(err); ok {
		w.WriteHeader(orbe.Code)
		fmt.Fprint(w, orbe.Error()) //nolint:errcheck
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error()) //nolint:errcheck
	}
}
