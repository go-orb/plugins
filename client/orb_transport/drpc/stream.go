package drpc

import (
	"context"
	"errors"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/server/drpc/message"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
)

// drpcClientStream wraps a DRPC stream to implement the client.Stream interface.
type drpcClientStream[TReq any, TResp any] struct {
	opts             *client.CallOptions
	stream           drpc.Stream
	ctx              context.Context
	conn             *drpcconn.Conn
	cancel           context.CancelFunc
	contentTypeCodec codecs.Marshaler
	encoder          *encoder
	closed           bool
	sendClosed       bool
}

// Context returns the context for this stream.
func (d *drpcClientStream[TReq, TResp]) Context() context.Context {
	return d.ctx
}

// Send sends a message to the stream.
func (d *drpcClientStream[TReq, TResp]) Send(msg TReq) error {
	if d.closed {
		return orberrors.ErrBadRequest.Wrap(errors.New("stream is closed"))
	}

	if d.sendClosed {
		return orberrors.ErrBadRequest.Wrap(errors.New("send direction is closed"))
	}

	if err := d.stream.MsgSend(msg, d.encoder); err != nil {
		d.conn.Close()
		return orberrors.HTTP(int(drpcerr.Code(err))).Wrap(err) //nolint:gosec
	}

	return nil
}

// Recv receives a message from the stream.
func (d *drpcClientStream[TReq, TResp]) Recv(msg TResp) error {
	if d.closed {
		return orberrors.ErrBadRequest.Wrap(errors.New("stream is closed"))
	}

	// if err := d.stream.MsgRecv(msg, d.encoder); err != nil {
	// 	d.conn.Close()
	// 	return orberrors.HTTP(int(drpcerr.Code(err))).Wrap(err) //nolint:gosec
	// }

	// return nil

	mdResult := &message.Response{}

	if err := d.stream.MsgRecv(mdResult, d.encoder); err != nil {
		d.conn.Close()
		return orberrors.HTTP(int(drpcerr.Code(err))).Wrap(err) //nolint:gosec
	}

	// Retrieve metadata from drpc.
	if d.opts.ResponseMetadata != nil {
		for k, v := range mdResult.GetMetadata() {
			d.opts.ResponseMetadata[k] = v
		}
	}

	// Unmarshal the result.
	if d.opts.ContentType == codecs.MimeProto {
		protoMsg, ok := any(msg).(proto.Message)
		if ok {
			if err := mdResult.GetData().UnmarshalTo(protoMsg); err != nil {
				return orberrors.From(err)
			}
		} else {
			return orberrors.From(errors.New("message is not a proto.Message"))
		}
	} else {
		// Convert the Any proto message to JSON first
		jsonBytes, err := protojson.Marshal(mdResult.GetData())
		if err != nil {
			return orberrors.From(err)
		}

		// Use contentTypeCodec to unmarshal the JSON into the result
		if err := d.contentTypeCodec.Unmarshal(jsonBytes, msg); err != nil {
			return orberrors.From(err)
		}
	}

	return nil
}

// Close closes the stream.
func (d *drpcClientStream[TReq, TResp]) Close() error {
	if d.closed {
		return orberrors.ErrBadRequest.Wrap(errors.New("stream is closed"))
	}

	d.closed = true
	d.sendClosed = true

	// Cancel the context
	if d.cancel != nil {
		d.cancel()
	}

	// Close the stream
	err := d.stream.Close()

	// Also return the connection to the pool
	if d.conn != nil {
		_ = d.conn.Close() //nolint:errcheck
	}

	if err != nil {
		return orberrors.HTTP(int(drpcerr.Code(err))).Wrap(err) //nolint:gosec
	}

	return nil
}

// CloseSend closes the send direction of the stream but leaves the receive side open.
// This follows the gRPC and DRPC pattern for client streams that need to signal
// completion of sending but still need to receive responses.
func (d *drpcClientStream[TReq, TResp]) CloseSend() error {
	if d.closed {
		return orberrors.ErrBadRequest.Wrap(errors.New("stream is closed"))
	}

	if d.sendClosed {
		return nil
	}

	d.sendClosed = true

	// Inform the DRPC stream that we're done sending
	return d.stream.CloseSend()
}

// SendMsg is an alias for Send to satisfy the client.Stream interface.
func (d *drpcClientStream[TReq, TResp]) SendMsg(m TReq) error {
	return d.Send(m)
}

// RecvMsg is an alias for Recv to satisfy the client.Stream interface.
func (d *drpcClientStream[TReq, TResp]) RecvMsg(m TResp) error {
	return d.Recv(m)
}
