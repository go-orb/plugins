// Package echo provdes a echo handler.
package echo

import (
	"context"

	"github.com/go-orb/plugins/benchmarks/rps/proto/echo"
)

var _ echo.EchoServer = (*Handler)(nil)

// Handler is a test handler.
type Handler struct {
	echo.UnsafeEchoServer
}

// Echo implements the echo method.
func (c *Handler) Echo(_ context.Context, req *echo.Req) (*echo.Resp, error) {
	resp := &echo.Resp{
		Payload: req.GetPayload(),
	}

	return resp, nil
}
