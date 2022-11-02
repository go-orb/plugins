package handler

import (
	"context"

	"github.com/go-micro/plugins/server/http/utils/tests/proto"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

type EchoHandler struct {
	proto.UnimplementedStreamsServer
}

func (c *EchoHandler) Call(ctx context.Context, in *proto.CallRequest) (*proto.CallResponse, error) {
	// Can be used to test large messages, e.g. to bench gzip compression
	// msg := make([]byte, 1024*1024*10)
	// rand.Reader.Read(msg)
	return &proto.CallResponse{Msg: "Hello " + in.Name}, nil
}
