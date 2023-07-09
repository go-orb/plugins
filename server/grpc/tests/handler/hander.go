// Package handler provdes a test handler.
package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"github.com/go-orb/plugins/server/grpc/tests/proto"
	"golang.org/x/exp/slog"
)

var _ proto.StreamsServer = (*EchoHandler)(nil)

// EchoHandler is a test handler.
type EchoHandler struct {
	proto.UnimplementedStreamsServer
}

// Call implements the call method.
func (c *EchoHandler) Call(ctx context.Context, in *proto.CallRequest) (*proto.CallResponse, error) {
	if in.Sleep != 0 {
		time.Sleep(time.Second * time.Duration(in.Sleep))
	}

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

// Stream will stream echo messages to a client.
func (c *EchoHandler) Stream(in proto.Streams_StreamServer) error {
	for {
		msg, err := in.Recv()
		if err != nil && !errors.Is(err, io.EOF) {
			slog.Error("failed to receive message", err)
			return err
		} else if err != nil && errors.Is(err, io.EOF) {
			return nil
		}

		if err := in.Send(&proto.CallResponse{Msg: "hello " + msg.Name}); err != nil {
			slog.Error("failed to send message", err)
			return err
		}
	}
}
