package grpc

import (
	"context"
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	gmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb/transport/grpc/pool"
)

// Name is the transports name.
const Name = "grpc"

func init() {
	orb.RegisterTransport(Name, NewTransport)
}

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	logger log.Logger
	pool   *pool.Pool
}

// Start starts the transport.
func (t *Transport) Start() error {
	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	if t.pool != nil {
		t.pool.Close()
	}

	return nil
}

func (t *Transport) String() string {
	return Name
}

// NeedsCodec returns false for grpc the transport.
func (t *Transport) NeedsCodec() bool {
	return false
}

// Call is a noop for grpc.
func (t *Transport) Call(_ context.Context, _ *client.Request[any, any], _ *client.CallOptions) (*client.RawResponse, error) {
	return nil, orberrors.ErrInternalServerError
}

// CallNoCodec does the actual rpc call to the server.
//
//nolint:funlen
func (t *Transport) CallNoCodec(ctx context.Context, req *client.Request[any, any], result any, opts *client.CallOptions) error {
	node, err := req.Node(ctx, opts)
	if err != nil {
		return orberrors.From(err)
	}

	if t.pool == nil {
		factory := func(_ context.Context, addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
			gopts := []grpc.DialOption{}

			if tlsConfig != nil {
				gopts = append(gopts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
			} else {
				gopts = append(gopts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			}

			// TODO(jochumdev): Bring back opts.DialTimeout
			return grpc.NewClient(addr, gopts...)
		}

		pool, err := pool.New(factory, opts.PoolSize, opts.PoolTTL)
		if err != nil {
			return orberrors.From(err)
		}

		t.pool = pool
	}

	conn, err := t.pool.Get(ctx, node.Address, opts.TLSConfig)
	if err != nil {
		return orberrors.From(err)
	}

	// Append go-orb metadata to grpc.
	if md, ok := metadata.Outgoing(ctx); ok {
		kv := []string{}
		for k, v := range md {
			kv = append(kv, k, v)
		}

		ctx = gmetadata.AppendToOutgoingContext(ctx, kv...)
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(opts.RequestTimeout))
	defer cancel()

	resMeta := gmetadata.MD{}

	err = conn.Invoke(ctx, "/"+req.Endpoint(), req.Request(), result, grpc.Header(&resMeta))
	if err != nil {
		gErr, ok := status.FromError(err)
		if !ok {
			_ = conn.Close() //nolint:errcheck

			return orberrors.From(err)
		}

		httpStatusCode := CodeToHTTPStatus(gErr.Code())

		_ = conn.Close() //nolint:errcheck

		orbE := orberrors.HTTP(httpStatusCode)
		return orbE.Wrap(gErr.Err())
	}

	if opts.ResponseMetadata != nil {
		for k, v := range resMeta {
			opts.ResponseMetadata[k] = v[0]
		}
	}

	err = conn.Close()
	if err != nil {
		gErr, ok := status.FromError(err)
		if !ok {
			return orberrors.From(err)
		}

		httpStatusCode := CodeToHTTPStatus(gErr.Code())

		orbE := orberrors.HTTP(httpStatusCode)

		return orbE.Wrap(gErr.Err())
	}

	return nil
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger, _ *orb.Config) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		logger: logger,
	}}, nil
}
