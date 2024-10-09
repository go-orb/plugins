// Package handler provdes a test handler.
package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/go-orb/plugins/server/grpc/tests/proto"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

// EchoHandler is a test handler.
type EchoHandler struct {
	proto.UnimplementedStreamsServer
}

// Call implements the call method.
func (c *EchoHandler) Call(_ context.Context, in *proto.CallRequest) (*proto.CallResponse, error) {
	if in.GetSleep() != 0 {
		time.Sleep(time.Second * time.Duration(in.GetSleep())) //nolint:gosec
	}

	switch in.GetName() {
	case "error":
		return nil, errors.New("you asked for an error, here you go")
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
