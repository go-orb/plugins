// Package drpc provides a drpc transport for client/orb.
package drpc

import (
	"context"
	"errors"
	"net"
	"time"

	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcmetadata"
	"storj.io/drpc/drpcpool"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/server/drpc/message"
)

var _ drpc.Encoding = (*encoder)(nil)

type encoder struct {
	marshaler   codecs.Marshaler
	unmarshaler codecs.Marshaler
}

func (e *encoder) Marshal(msg drpc.Message) ([]byte, error) {
	return e.marshaler.Marshal(msg)
}

func (e *encoder) Unmarshal(data []byte, msg drpc.Message) error {
	return e.unmarshaler.Unmarshal(data, msg)
}

// Name is the transports name.
const Name = "drpc"

func init() {
	orb.RegisterTransport(Name, NewTransport)
}

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	logger log.Logger
	pool   *drpcpool.Pool[string, drpcpool.Conn]
}

// Start starts the transport.
func (t *Transport) Start() error {
	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	return t.pool.Close()
}

func (t *Transport) String() string {
	return Name
}

// NeedsCodec returns false for grpc the transport.
func (t *Transport) NeedsCodec() bool {
	return false
}

// Request is a noop for grpc.
func (t *Transport) Request(_ context.Context, _ *client.Req[any, any], _ *client.CallOptions) (*client.RawResponse, error) {
	return nil, orberrors.ErrInternalServerError
}

// RequestNoCodec does the actual rpc request to the server.
//
//nolint:gocyclo,funlen
func (t *Transport) RequestNoCodec(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
	node, err := req.Node(ctx, opts)
	if err != nil {
		return orberrors.From(err)
	}

	dial := func(_ context.Context, address string) (drpcpool.Conn, error) {
		// dial the drpc server
		rawconn, err := net.Dial("tcp", address)
		if err != nil {
			return nil, err
		}

		return drpcconn.New(rawconn), nil
	}

	conn := t.pool.Get(ctx, node.Address, dial)

	// Add metadata to drpc.
	md, ok := metadata.Outgoing(ctx)
	if ok {
		md["Content-Type"] = opts.ContentType
		ctx = drpcmetadata.AddPairs(ctx, md)
	}

	contentTypeCodec, err := codecs.GetMime(opts.ContentType)
	if err != nil {
		return orberrors.From(err)
	}

	protoCodec := contentTypeCodec
	if opts.ContentType != codecs.MimeProto {
		protoCodec, err = codecs.GetMime(codecs.MimeProto)
		if err != nil {
			return orberrors.From(err)
		}
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(opts.RequestTimeout))
	defer cancel()

	mdResult := &message.Response{}
	if err := conn.Invoke(
		ctx,
		req.Endpoint(),
		&encoder{marshaler: contentTypeCodec, unmarshaler: protoCodec},
		req.Req(),
		mdResult,
	); err != nil {
		orbError := orberrors.HTTP(int(drpcerr.Code(err))) //nolint:gosec
		orbError = orbError.Wrap(err)

		return orbError
	}

	// Retrieve metadata from drpc.
	if opts.ResponseMetadata != nil {
		for k, v := range mdResult.GetMetadata() {
			opts.ResponseMetadata[k] = v
		}
	}

	if mdResult.GetError().GetCode() != 0 {
		orbE := orberrors.New(int(mdResult.GetError().GetCode()), mdResult.GetError().GetMessage())
		if mdResult.GetError().GetWrapped() != "" {
			orbE = orbE.Wrap(errors.New(mdResult.GetError().GetWrapped()))
		}

		return orbE
	}

	// Unmarshal the result.
	if opts.ContentType == codecs.MimeProto {
		if err := mdResult.GetData().UnmarshalTo(result.(proto.Message)); err != nil { //nolint:errcheck
			return orberrors.From(err)
		}
	} else {
		// Convert the Any proto message to JSON first
		jsonBytes, err := protojson.Marshal(mdResult.GetData())
		if err != nil {
			return orberrors.From(err)
		}

		// Use contentTypeCodec to unmarshal the JSON into the result
		if err := contentTypeCodec.Unmarshal(jsonBytes, result); err != nil {
			return orberrors.From(err)
		}
	}

	err = conn.Close()
	if err != nil {
		return orberrors.From(err)
	}

	return nil
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		logger: logger,
		pool: drpcpool.New[string, drpcpool.Conn](
			drpcpool.Options{
				Capacity:    cfg.PoolSize,
				KeyCapacity: cfg.PoolHosts,
			},
		),
	}}, nil
}
