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
	drpc.Stream
	ctx context.Context
}

func (s *streamWrapper) Context() context.Context { return s.ctx }

// HandleRPC handles the rpc that has been requested by the stream.
//
//nolint:funlen
func (m *Mux) HandleRPC(stream drpc.Stream, rpc string) (err error) {
	data, rpcOk := m.rpcs[rpc]
	if !rpcOk {
		return drpc.ProtocolError.New("unknown rpc: %q", rpc)
	}

	req := interface{}(stream)

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
	}

	ctx := stream.Context()

	ctx, reqMd := metadata.WithIncoming(ctx)
	ctx, outMd := metadata.WithOutgoing(ctx)

	dMeta, rpcOk := drpcmetadata.Get(ctx)
	if !rpcOk {
		dMeta = make(map[string]string)
	}

	for k, v := range dMeta {
		reqMd[k] = v
	}

	fmSplit := strings.Split(rpc, "/")

	if len(fmSplit) >= 3 {
		reqMd[metadata.Service] = fmSplit[1]
		reqMd[metadata.Method] = fmSplit[2]
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

	if out != nil && err == nil {
		outData, err := anypb.New(out.(proto.Message)) //nolint:errcheck
		if err != nil {
			return errs.Wrap(err)
		}

		return stream.MsgSend(&message.Response{Metadata: outMd, Data: outData, Error: &message.Error{}}, data.enc)
	} else if err != nil {
		orbErr := orberrors.From(err)

		wrapped := ""
		if orbErr.Unwrap() != nil {
			wrapped = orbErr.Unwrap().Error()
		}

		return stream.MsgSend(&message.Response{Metadata: outMd, Data: &anypb.Any{}, Error: &message.Error{
			Code:    int64(orbErr.Code),
			Message: orbErr.Message,
			Wrapped: wrapped,
		}}, data.enc)
	}

	return stream.CloseSend()
}
