// Package handler provdes a test handler.
package handler

import (
	"context"

	"github.com/go-micro/plugins/server/tests/proto"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

// EchoHandler is a test handler.
type EchoHandler struct {
	proto.UnimplementedStreamsServer
}

// Call implements the call method.
func (c *EchoHandler) Call(ctx context.Context, in *proto.CallRequest) (*proto.CallResponse, error) {
	// Can be used to test large messages, e.g. to bench gzip compression
	// msg := make([]byte, 1024*1024*10)
	// rand.Reader.Read(msg)
	return &proto.CallResponse{Msg: "Hello " + in.Name}, nil
}
