// Package drpc provides a drpc transport for client/orb.
package drpc

import (
	"context"
	"crypto/tls"
	"net"

	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcerr"
	"storj.io/drpc/drpcmetadata"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb_transport/drpc/pool"
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
	config *orb.Config
	logger log.Logger
	pool   *pool.Pool
}

// Start starts the transport.
func (t *Transport) Start() error {
	factory := func(dialCtx context.Context, addr string, _ *tls.Config) (*drpcconn.Conn, error) {
		// Use the dial timeout from options
		timeoutCtx, cancel := context.WithTimeout(dialCtx, t.config.DialTimeout)
		defer cancel()

		dialer := net.Dialer{}
		rawconn, err := dialer.DialContext(timeoutCtx, "tcp", addr)

		if err != nil {
			t.logger.Error("Failed to dial DRPC server", "address", addr, "error", err)
			return nil, err
		}

		// Create a new DRPC connection
		return drpcconn.New(rawconn), nil
	}

	t.logger.Debug(
		"Creating a transport pool",
		"pool_hosts", t.config.PoolHosts,
		"pool_size", t.config.PoolSize,
		"conn_timeout", t.config.ConnectionTimeout,
		"pool_ttl", t.config.PoolTTL,
	)

	pool, err := pool.New(factory, t.config.PoolHosts*t.config.PoolSize, t.config.PoolTTL)
	if err != nil {
		return orberrors.From(err)
	}

	t.pool = pool

	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	t.pool.Close()
	return nil
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

	conn, err := t.pool.Get(ctx, node.Address, nil)
	if err != nil {
		return orberrors.From(err)
	}

	// Add metadata to drpc request
	md, ok := metadata.Outgoing(ctx)
	if ok {
		md["Content-Type"] = opts.ContentType
		ctx = drpcmetadata.AddPairs(ctx, md)
	}

	// Get codecs for content type conversion
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

	// Create context with timeout for the request
	reqCtx, cancel := context.WithTimeout(ctx, opts.RequestTimeout)
	defer cancel()

	// Prepare response container
	mdResult := &message.Response{}

	// Invoke the RPC method
	if err := conn.Invoke(
		reqCtx,
		req.Endpoint(),
		&encoder{marshaler: contentTypeCodec, unmarshaler: protoCodec},
		req.Req(),
		mdResult,
	); err != nil {
		conn.Unhealthy()

		if closeErr := conn.Close(); closeErr != nil {
			t.logger.Error("Failed to close failed connection", "error", closeErr)
		}

		orbError := orberrors.HTTP(int(drpcerr.Code(err))) //nolint:gosec
		orbError = orbError.Wrap(err)

		return orbError
	}

	err = conn.Close()
	if err != nil {
		return orberrors.From(err)
	}

	// Retrieve metadata from drpc.
	if opts.ResponseMetadata != nil {
		for k, v := range mdResult.GetMetadata() {
			opts.ResponseMetadata[k] = v
		}
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

	return nil
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	logger.Debug("Creating transport", "pool_hosts", cfg.PoolHosts, "pool_size", cfg.PoolSize, "conn_timeout", cfg.ConnectionTimeout)

	return orb.TransportType{Transport: &Transport{
		config: cfg,
		logger: logger,
	}}, nil
}
