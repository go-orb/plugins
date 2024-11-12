package hertz

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
)

var stdHeaders = []string{"Accept", "Accept-Encoding", "Content-Length", "Content-Type", "User-Agent"} //nolint:gochecknoglobals

// Errors.
var (
	ErrNotHTTPServer = errors.New("server provider is not of type *http.Server")
)

// NewGRPCHandler wraps a gRPC function with a Hertz handler.
func NewGRPCHandler[Tin any, Tout any](
	srv *Server,
	fHandler func(context.Context, *Tin) (*Tout, error),
	service string,
	method string,
) func(c context.Context, ctx *app.RequestContext) {
	return func(ctx context.Context, apCtx *app.RequestContext) {
		request := new(Tin)

		if _, err := srv.decodeBody(apCtx, request); err != nil {
			srv.logger.Error("failed to decode body", "error", err)
			WriteError(apCtx, err)

			return
		}

		// Copy metadata from req Headers into the req.Context.
		ctx, reqMd := metadata.WithIncoming(ctx)
		ctx, outMd := metadata.WithOutgoing(ctx)

		apCtx.VisitAllHeaders(func(k, v []byte) {
			sk := string(k)
			if slices.Contains(stdHeaders, sk) {
				return
			}

			reqMd[strings.ToLower(sk)] = string(v)
		})

		reqMd[metadata.Service] = service
		reqMd[metadata.Method] = method

		// Apply middleware.
		h := func(ctx context.Context, req any) (any, error) {
			return fHandler(ctx, req.(*Tin)) //nolint:errcheck
		}
		for _, m := range srv.config.OptMiddlewares {
			h = m.Call(h)
		}

		out, err := h(ctx, request)
		if err != nil {
			srv.logger.Error("RPC request failed", "error", err)
			WriteError(apCtx, err)

			return
		}

		// Write outgoing metadata.
		for k, v := range outMd {
			apCtx.Header(k, v)
		}

		if err := srv.encodeBody(apCtx, out); err != nil {
			srv.logger.Error("failed to encode body", "error", err)
			WriteError(apCtx, err)

			return
		}
	}
}

// WriteError returns an error response to the HTTP request.
func WriteError(ctx *app.RequestContext, err error) {
	if err == nil {
		return
	}

	if orbe, ok := orberrors.As(err); ok {
		ctx.AbortWithError(orbe.Code, err) //nolint:errcheck
	} else {
		ctx.AbortWithError(500, err) //nolint:errcheck
	}
}
