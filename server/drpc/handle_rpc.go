// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2024 go-orb Authors.
// See LICENSE for copying information.

package drpc

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-orb/go-orb/codecs"
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

var _ drpc.Encoding = (*encoder)(nil)

type encoder struct {
	codec codecs.Marshaler
}

func (e *encoder) Marshal(msg drpc.Message) ([]byte, error) {
	return e.codec.Marshal(msg)
}

func (e *encoder) Unmarshal(data []byte, msg drpc.Message) error {
	return e.codec.Unmarshal(data, msg)
}

type streamWrapper struct {
	stream drpc.Stream
	ctx    context.Context
}

func (s *streamWrapper) Context() context.Context {
	return s.ctx
}

// MsgSend sends the Message to the remote with our custom wrapper implementation.
func (s *streamWrapper) MsgSend(msg drpc.Message, enc drpc.Encoding) error {
	// If the message is already a Response, send it directly
	if _, ok := msg.(*message.Response); ok {
		return s.stream.MsgSend(msg, enc)
	}

	// Otherwise, wrap it in a Response
	_, outMd := metadata.WithOutgoing(s.ctx)

	outData, err := anypb.New(msg.(proto.Message)) //nolint:errcheck
	if err != nil {
		return errs.Wrap(err)
	}

	return s.stream.MsgSend(&message.Response{Metadata: outMd, Data: outData}, enc)
}

// MsgRecv receives a Message from the remote.
func (s *streamWrapper) MsgRecv(msg drpc.Message, enc drpc.Encoding) error {
	return s.stream.MsgRecv(msg, enc)
}

// CloseSend signals to the remote that we will no longer send any messages.
func (s *streamWrapper) CloseSend() error {
	return s.stream.CloseSend()
}

// Close closes the stream.
func (s *streamWrapper) Close() error {
	return s.stream.Close()
}

// HandleRPC handles the rpc that has been requested by the stream.
func (m *Mux) HandleRPC(stream drpc.Stream, rpc string) (err error) {
	data, rpcOK := m.rpcs[rpc]
	if !rpcOK {
		return drpc.ProtocolError.New("unknown rpc: %q", rpc)
	}

	ctx := stream.Context()

	ctx, reqMd := metadata.WithIncoming(ctx)
	ctx, outMd := metadata.WithOutgoing(ctx)

	dMeta, ok := drpcmetadata.Get(ctx)
	if ok {
		for k, v := range dMeta {
			reqMd[k] = v
		}
	}

	fmSplit := strings.Split(rpc, "/")

	if len(fmSplit) >= 3 {
		reqMd[metadata.Service] = fmSplit[1]
		reqMd[metadata.Method] = fmSplit[2]
	}

	var req interface{}

	if data.in1 != streamType {
		// Check for content type in metadata
		ctx := stream.Context()
		dMeta, hasMeta := drpcmetadata.Get(ctx)
		contentType := codecs.MimeProto

		if hasMeta {
			contentType = dMeta["Content-Type"]
		}

		codec, err := codecs.GetMime(contentType)
		if err != nil {
			return drpcerr.WithCode(fmt.Errorf("invalid content type: %q", contentType), http.StatusInternalServerError)
		}

		msg := reflect.New(data.in1.Elem()).Interface()

		if err := stream.MsgRecv(msg, &encoder{codec: codec}); err != nil {
			return errs.Wrap(err)
		}

		req = msg
	} else {
		req = &streamWrapper{stream: stream, ctx: ctx}
	}

	// Apply middleware.
	h := func(ctx context.Context, req any) (any, error) {
		// The actual RPC.
		return data.receiver(data.srv, ctx, req, data.in2)
	}
	for _, m := range m.orbSrv.middlewares {
		h = m.Call(h)
	}

	// Calls all middlewares until the actual RPC.
	out, err := h(ctx, req)

	switch {
	case err != nil:
		orbE := orberrors.From(err)
		drpcE := drpcerr.WithCode(orbE, uint64(orbE.Code)) //nolint:gosec

		return errs.Wrap(drpcE)
	case out != nil:
		outData, err := anypb.New(out.(proto.Message)) //nolint:errcheck
		if err != nil {
			return errs.Wrap(err)
		}

		return stream.MsgSend(&message.Response{Metadata: outMd, Data: outData}, data.enc)
	default:
		return stream.CloseSend()
	}
}
