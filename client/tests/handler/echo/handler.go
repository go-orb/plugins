// Package echo provides the echo test handler.
package echo

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/tests/proto/echo"
)

var _ echo.StreamsServer = (*Handler)(nil)

// Handler is a test handler.
type Handler struct {
}

// Call implements the call method.
func (c *Handler) Call(_ context.Context, request *echo.CallRequest) (*echo.CallResponse, error) {
	switch request.GetName() {
	case "error":
		return nil, errors.New("you asked for an error, here you go")
	case "32byte":
		msg := make([]byte, 32)
		if _, err := rand.Reader.Read(msg); err != nil {
			return nil, err
		}

		return &echo.CallResponse{Msg: "", Payload: msg}, nil
	case "big":
		// Can be used to test large messages, e.g. to bench gzip compression
		msg := make([]byte, 1024*1024*10)
		if _, err := rand.Reader.Read(msg); err != nil {
			return nil, err
		}

		return &echo.CallResponse{Msg: "Hello " + request.GetName(), Payload: msg}, nil
	default:
		return &echo.CallResponse{Msg: "Hello " + request.GetName()}, nil
	}
}

// AuthorizedCall requires Authorization by metadata.
func (c *Handler) AuthorizedCall(ctx context.Context, _ *echo.CallRequest) (*echo.CallResponse, error) {
	_, mdin := metadata.WithIncoming(ctx)
	if mdin["authorization"] != "Bearer pleaseHackMe" {
		return nil, orberrors.ErrUnauthorized
	}

	_, mdout := metadata.WithOutgoing(ctx)
	mdout["tracing-id"] = "asfdjhladhsfashf"

	return &echo.CallResponse{Msg: "Hello World"}, nil
}
