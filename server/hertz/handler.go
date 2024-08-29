package hertz

import (
	"context"
	"errors"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
)

// orbHeader is the prefix for every orb HTTP header.
const orbHeader = "__orb-"

// Errors.
var (
	ErrNotHTTPServer = errors.New("server provider is not of type *http.Server")
)

// NewGRPCHandler wraps a gRPC function with a Hertz handler.
func NewGRPCHandler[Tin any, Tout any](
	srv *Server,
	f func(context.Context, *Tin) (*Tout, error),
) func(c context.Context, ctx *app.RequestContext) {
	return func(ctx context.Context, apCtx *app.RequestContext) {
		in := new(Tin)

		if _, err := srv.decodeBody(apCtx, in); err != nil {
			srv.Logger.Error("failed to decode body", "error", err)
			WriteError(apCtx, err)

			return
		}

		// Copy metadata from req Headers into the req.Context.
		reqMd := make(metadata.Metadata)

		apCtx.VisitAllHeaders(func(k, v []byte) {
			sk := string(k)
			if !strings.HasPrefix(strings.ToLower(sk), orbHeader) {
				return
			}

			sk = sk[len(orbHeader):]
			reqMd[sk] = string(v)
		})

		out, err := f(reqMd.To(ctx), in)
		if err != nil {
			srv.Logger.Error("RPC request failed", "error", err)
			WriteError(apCtx, err)

			return
		}

		// Write back metadata to headers.
		for k, v := range reqMd {
			apCtx.Header(orbHeader+k, v)
		}

		if err := srv.encodeBody(apCtx, out); err != nil {
			srv.Logger.Error("failed to encode body", "error", err)
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
