package grpc

import (
	"context"
	"crypto/tls"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb/transport/grpc/pool"
)

// Name is the transports name.
const Name = "grpc"

func init() {
	orb.Transports.Register(Name, NewTransport)
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
		t.pool.Close() // nolint:errcheck
	}

	return nil
}

func (t *Transport) String() string {
	return Name
}

func (t *Transport) NeedsCodec() bool {
	return false
}

// Call does the actual rpc call to the server.
func (t *Transport) Call(ctx context.Context, req *client.Request[any, any], opts *client.CallOptions) (*client.RawResponse, error) {
	return nil, orberrors.ErrInternalServerError
}

// Call does the actual rpc call to the server using the transports codecs.
func (t *Transport) CallNoCodec(ctx context.Context, req *client.Request[any, any], result any, opts *client.CallOptions) error {
	node, err := req.Node(ctx, opts)
	if err != nil {
		return orberrors.From(err)
	}

	if t.pool == nil {
		factory := func(ctx context.Context, addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
			gopts := []grpc.DialOption{}

			if tlsConfig != nil {
				gopts = append(gopts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
			} else {
				gopts = append(gopts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			}

			gopts = append(gopts, grpc.WithReturnConnectionError(), grpc.WithConnectParams(grpc.ConnectParams{
				MinConnectTimeout: opts.ConnectionTimeout,
			}))

			return grpc.DialContext(ctx, addr, gopts...)
		}

		pool, err := pool.New(factory, opts.PoolSize, opts.PoolTTL)
		if err != nil {
			return orberrors.From(err)
		}

		t.pool = pool
	}

	conn, err := t.pool.Get(ctx, node.Address, opts.TlsConfig)
	if err != nil {
		return orberrors.From(err)
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(opts.RequestTimeout))
	defer cancel()

	err = conn.Invoke(ctx, req.Endpoint(), req.Request(), result)
	defer conn.Close()
	if err != nil {
		gErr, ok := status.FromError(err)
		if !ok {
			return orberrors.From(err)
		}

		httpStatusCode := CodeToHTTPStatus(gErr.Code())
		return orberrors.New(httpStatusCode, gErr.Message())
	}

	return nil
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		logger: logger,
	}}, nil
}
