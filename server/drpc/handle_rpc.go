// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2024 go-orb Authors.
// See LICENSE for copying information.

package drpc

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/server/drpc/message"
	"github.com/zeebo/errs"
	proto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"storj.io/drpc"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcmetadata"
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

	req := interface{}(stream)

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

	ctx = metadata.EnsureIncoming(ctx)
	ctx = metadata.EnsureOutgoing(ctx)

	dMeta, ok := drpcmetadata.Get(ctx)
	if !ok {
		dMeta = make(map[string]string)
	}

	incMd, _ := metadata.IncomingFrom(ctx)
	for k, v := range dMeta {
		incMd[k] = v
	}

	fmSplit := strings.Split(rpc, "/")

	if len(fmSplit) >= 3 {
		incMd[metadata.Service] = fmSplit[1]
		incMd[metadata.Method] = fmSplit[2]
	}

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

	switch {
	case err != nil:
		oErr := orberrors.From(err)

		if oErr.Wrapped != nil {
			return drpcerr.WithCode(fmt.Errorf("%s: %s", oErr.Message, oErr.Wrapped.Error()), uint64(oErr.Code))
		}

		return drpcerr.WithCode(errors.New(oErr.Message), uint64(oErr.Code))
	case out != nil && !reflect.ValueOf(out).IsNil():
		outMd, _ := metadata.OutgoingFrom(ctx)

		outData, err := anypb.New(out.(proto.Message))
		if err != nil {
			return errs.Wrap(err)
		}

		return stream.MsgSend(&message.Response{Metadata: outMd, Data: outData}, data.enc)
	default:
		return stream.CloseSend()
	}
}
