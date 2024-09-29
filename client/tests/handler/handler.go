// Package handler provdes a test handler.
package handler

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/tests/proto"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

// EchoHandler is a test handler.
type EchoHandler struct {
	proto.UnsafeStreamsServer
}

// Call implements the call method.
func (c *EchoHandler) Call(_ context.Context, request *proto.CallRequest) (*proto.CallResponse, error) {
	switch request.GetName() {
	case "error":
		return nil, errors.New("you asked for an error, here you go")
	case "32byte":
		msg := make([]byte, 32)
		if _, err := rand.Reader.Read(msg); err != nil {
			return nil, err
		}

		return &proto.CallResponse{Msg: "", Payload: msg}, nil
	case "big":
		// Can be used to test large messages, e.g. to bench gzip compression
		msg := make([]byte, 1024*1024*10)
		if _, err := rand.Reader.Read(msg); err != nil {
			return nil, err
		}

		return &proto.CallResponse{Msg: "Hello " + request.GetName(), Payload: msg}, nil
	default:
		return &proto.CallResponse{Msg: "Hello " + request.GetName()}, nil
	}
}

// AuthorizedCall requires Authorization by metadata.
func (c *EchoHandler) AuthorizedCall(ctx context.Context, _ *proto.CallRequest) (*proto.CallResponse, error) {
	mdout, _ := metadata.OutgoingFrom(ctx)
	mdout["tracing-id"] = "asfdjhladhsfashf"

	mdin, _ := metadata.IncomingFrom(ctx)
	if mdin["authorization"] != "bearer pleaseHackMe" {
		return nil, orberrors.ErrUnauthorized
	}

	return &proto.CallResponse{Msg: "Hello World"}, nil
}
