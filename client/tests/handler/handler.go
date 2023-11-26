// Package handler provdes a test handler.
package handler

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/go-orb/plugins/client/tests/proto"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

// EchoHandler is a test handler.
type EchoHandler struct {
	proto.UnsafeOrbStreamsServer
}

// Call implements the call method.
func (c *EchoHandler) Call(_ context.Context, in *proto.CallRequest) (*proto.CallResponse, error) {
	switch in.GetName() {
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

		return &proto.CallResponse{Msg: "Hello " + in.GetName(), Payload: msg}, nil
	default:
		return &proto.CallResponse{Msg: "Hello " + in.GetName()}, nil
	}
}
