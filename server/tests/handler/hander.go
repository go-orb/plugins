// Package handler provdes a test handler.
package handler

import (
	"context"
	"crypto/rand"
	"errors"

	"github.com/go-micro/plugins/server/tests/proto"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

// EchoHandler is a test handler.
type EchoHandler struct {
	proto.UnimplementedStreamsServer
}

// Call implements the call method.
func (c *EchoHandler) Call(ctx context.Context, in *proto.CallRequest) (*proto.CallResponse, error) {
	switch in.Name {
	case "error":
		return nil, errors.New("you asked for an error, here you go")
	case "big":
		// Can be used to test large messages, e.g. to bench gzip compression
		msg := make([]byte, 1024*1024*10)
		if _, err := rand.Reader.Read(msg); err != nil {
			return nil, err
		}

		return &proto.CallResponse{Msg: "Hello " + in.Name, Payload: msg}, nil
	default:
		return &proto.CallResponse{Msg: "Hello " + in.Name}, nil
	}
}
