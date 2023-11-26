package hertz

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-orb/go-orb/util/orberrors"
)

// Errors.
var (
	ErrNotHTTPServer = errors.New("server provider is not of type *http.Server")
)

// NewGRPCHandler will wrap a gRPC function with a Hertz handler.
func NewGRPCHandler[Tin any, Tout any](
	srv *ServerHertz,
	f func(context.Context, *Tin) (*Tout, error),
) func(c context.Context, ctx *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		in := new(Tin)

		if _, err := srv.decodeBody(c, in); err != nil {
			srv.Logger.Error("failed to decode body", "error", err)
			WriteError(c, err)

			return
		}

		out, err := f(ctx, in)
		if err != nil {
			srv.Logger.Error("RPC request failed", "error", err)
			WriteError(c, err)

			return
		}

		if err := srv.encodeBody(c, out); err != nil {
			srv.Logger.Error("failed to encode body", "error", err)
			WriteError(c, err)

			return
		}
	}
}

// WriteError returns an error response to the HTTP request.
func WriteError(ctx *app.RequestContext, err error) {
	if err == nil {
		return
	}

	orbe := orberrors.From(err)
	ctx.AbortWithError(orbe.Code, err) //nolint:errcheck
}
