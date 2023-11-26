// Package handler provdes a test handler.
package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"time"

	"log/slog"

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
		time.Sleep(time.Second * time.Duration(in.GetSleep()))
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

		if err := in.Send(&proto.CallResponse{Msg: "hello " + msg.GetName()}); err != nil {
			slog.Error("failed to send message", err)
			return err
		}
	}
}
