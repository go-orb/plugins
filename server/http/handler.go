package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
)

// orbHeader is the prefix for every orb HTTP header.
const orbHeader = "__orb-"

// Errors.
var (
	ErrNotHTTPServer = errors.New("server provider is not of type *http.Server")
)

// NewGRPCHandler will wrap a gRPC function with a HTTP handler.
func NewGRPCHandler[Tin any, Tout any](srv *ServerHTTP, fHandler func(context.Context, *Tin) (*Tout, error)) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		inBody := new(Tin)

		if _, err := srv.decodeBody(resp, req, inBody); err != nil {
			srv.Logger.Error("failed to decode body", "error", err)
			WriteError(resp, err)

			return
		}

		// Copy metadata from req Headers into the req.Context.
		reqMd := make(metadata.Metadata)

		for k, v := range req.Header {
			if !strings.HasPrefix(strings.ToLower(k), orbHeader) {
				continue
			}

			k = k[len(orbHeader):]

			if len(v) == 1 {
				reqMd[k] = v[0]
			} else {
				reqMd[k] = v[0]
				for i := 1; i < len(v); i++ {
					reqMd[k+"-"+strconv.Itoa(i)] = v[i]
				}
			}
		}

		out, err := fHandler(reqMd.To(req.Context()), inBody)
		if err != nil {
			srv.Logger.Error("RPC request failed", "error", err)
			WriteError(resp, err)

			return
		}

		// Write back metadata to headers.
		for k, v := range reqMd {
			resp.Header().Set(orbHeader+k, v)
		}

		if err := srv.encodeBody(resp, req, out); err != nil {
			srv.Logger.Error("failed to encode body", "error", err)
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
		fmt.Fprint(w, err.Error()) //nolint:errcheck
	}
}
