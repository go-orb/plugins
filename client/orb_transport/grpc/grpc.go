package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	gmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb_transport/grpc/pool"
)

// Name is the transports name.
const Name = "grpc"

func init() {
	orb.RegisterTransport(Name, NewTransport(Name))
	orb.RegisterTransport("grpcs", NewTransport("grpcs"))
}

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	config *orb.Config
	logger log.Logger
	name   string

	poolLock sync.Mutex
	pool     *pool.Pool
}

// Start starts the transport.
func (t *Transport) Start() error {
	if encoding.GetCodec("json") == nil {
		codec, err := codecs.GetMime(codecs.MimeJSON)
		if err != nil {
			return err
		}

		encoding.RegisterCodec(codec)
	}

	factory := func(_ context.Context, addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
		gopts := []grpc.DialOption{}

		//nolint:gocritic
		if tlsConfig != nil {
			gopts = append(gopts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
		} else if t.name == "grpcs" {
			creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2"}}) //nolint:gosec
			gopts = append(gopts, grpc.WithTransportCredentials(creds))
		} else {
			gopts = append(gopts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		}

		// TODO(jochumdev): Bring back opts.DialTimeout
		return grpc.NewClient(addr, gopts...)
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
	t.poolLock.Lock()
	defer t.poolLock.Unlock()

	if t.pool != nil {
		t.pool.Close()
	}

	return nil
}

func (t *Transport) String() string {
	return t.name
}

// NeedsCodec returns false for grpc the transport.
func (t *Transport) NeedsCodec() bool {
	return false
}

// Request is a noop for grpc.
func (t *Transport) Request(_ context.Context, _ *client.Req[any, any], _ *client.CallOptions) (*client.RawResponse, error) {
	return nil, orberrors.ErrInternalServerError
}

// toOrbError converts a grpc error to an orb error.
func toOrbError(err error) error {
	if errors.Is(err, pool.ErrTimeout) {
		return orberrors.HTTP(504).Wrap(context.DeadlineExceeded)
	}

	gErr, ok := status.FromError(err)
	if !ok {
		return orberrors.From(err)
	}

	httpStatusCode := CodeToHTTPStatus(gErr.Code())

	orbE := orberrors.HTTP(httpStatusCode)
	if httpStatusCode == 504 {
		return orbE.Wrap(context.DeadlineExceeded)
	}

	if httpStatusCode == 499 {
		return orbE.Wrap(context.Canceled)
	}

	return orbE.Wrap(gErr.Err())
}

// RequestNoCodec does the actual rpc request to the server.
func (t *Transport) RequestNoCodec(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
	node, err := req.Node(ctx, opts)
	if err != nil {
		return orberrors.From(err)
	}

	conn, err := t.pool.Get(ctx, node.Address, opts.TLSConfig)
	if err != nil {
		return toOrbError(err)
	}

	// Append go-orb metadata to grpc.
	if md, ok := metadata.Outgoing(ctx); ok {
		kv := []string{}
		for k, v := range md {
			kv = append(kv, k, v)
		}

		ctx = gmetadata.AppendToOutgoingContext(ctx, kv...)
	}

	ctx, cancel := context.WithTimeout(ctx, opts.RequestTimeout)
	defer cancel()

	resMeta := gmetadata.MD{}
	callOpts := []grpc.CallOption{grpc.Header(&resMeta)}

	if opts.ContentType == codecs.MimeJSON {
		callOpts = append(callOpts, grpc.CallContentSubtype("json"))
	}

	err = conn.Invoke(ctx, req.Endpoint(), req.Req(), result, callOpts...)
	if err != nil {
		conn.Unhealthy()
		_ = conn.Close() //nolint:errcheck

		return toOrbError(err)
	}

	if opts.ResponseMetadata != nil {
		for k, v := range resMeta {
			opts.ResponseMetadata[k] = v[0]
		}
	}

	err = conn.Close()
	if err != nil {
		return toOrbError(err)
	}

	return nil
}

// NewTransport creates a Transport.
func NewTransport(name string) func(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return func(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
		return orb.TransportType{Transport: &Transport{
			name:   name,
			config: cfg,
			logger: logger,
		}}, nil
	}
}
