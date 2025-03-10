// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2024 go-orb Authors.
// See LICENSE for copying information.

package memory

import (
	"context"
	"reflect"

	"github.com/zeebo/errs"

	"storj.io/drpc"
)

type streamWrapper struct {
	drpc.Stream
	ctx context.Context
}

func (s *streamWrapper) Context() context.Context { return s.ctx }

// HandleRPC handles the rpc that has been requested by the stream.
func (m *Mux) HandleRPC(stream drpc.Stream, rpc string) (err error) {
	data, ok := m.rpcs[rpc]
	if !ok {
		return drpc.ProtocolError.New("unknown rpc: %q", rpc)
	}

	req := any(stream)

	if data.in1 != streamType {
		msg, ok := reflect.New(data.in1.Elem()).Interface().(drpc.Message)
		if !ok {
			return drpc.InternalError.New("invalid rpc input type")
		}

		if err := stream.MsgRecv(msg, data.enc); err != nil {
			return errs.Wrap(err)
		}

		req = msg
	}

	ctx := stream.Context()

	stream = &streamWrapper{Stream: stream, ctx: ctx}

	// Apply middleware.
	h := func(ctx context.Context, req any) (any, error) {
		// The actual call.
		return data.receiver(data.srv, ctx, req, stream)
	}
	for _, m := range m.orbSrv.middlewares {
		h = m.Call(h)
	}

	// Calls all middlewares until the actual call.
	out, err := h(ctx, req)

	if err != nil {
		return err
	}

	err = stream.MsgSend(out, data.enc)
	if err != nil {
		return err
	}

	return stream.CloseSend()
}
