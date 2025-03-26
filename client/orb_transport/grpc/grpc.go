package grpc

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	gmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/client/orb_transport/grpc/pool"
)

// Name is the transports name.
const Name = "grpc"

func init() {
	orb.RegisterTransport(Name, NewTransport(Name, "tcp"))
	orb.RegisterTransport("grpcs", NewTransport("grpcs", "tcp"))
	orb.RegisterTransport("unix+"+Name, NewTransport("unix+"+Name, "unix"))
}

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	config  *orb.Config
	logger  log.Logger
	name    string
	network string

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

	t.poolLock.Lock()
	defer t.poolLock.Unlock()

	if t.pool != nil {
		return nil
	}

	// Create a factory function for connections
	factory := func(ctx context.Context, addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
		return t.createConn(ctx, addr, tlsConfig)
	}

	// Create the pool
	var err error

	t.pool, err = pool.New(factory, t.config.PoolSize, time.Duration(t.config.PoolTTL))
	if err != nil {
		return toOrbError(err)
	}

	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	t.poolLock.Lock()
	defer t.poolLock.Unlock()

	if t.pool != nil {
		t.pool.Close()
		t.pool = nil
	}

	return nil
}

// Name returns the name of this transport.
func (t *Transport) Name() string {
	return t.name
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

// Request does the actual rpc request to the server.
func (t *Transport) Request(ctx context.Context, infos client.RequestInfos, req any, result any, opts *client.CallOptions) error {
	conn, err := t.pool.Get(ctx, infos.Address, opts.TLSConfig)
	if err != nil {
		return toOrbError(err)
	}

	// Append go-orb metadata to grpc.
	kv := []string{}
	for k, v := range opts.Metadata {
		kv = append(kv, k, v)
	}

	ctx = gmetadata.AppendToOutgoingContext(ctx, kv...)

	ctx, cancel := context.WithTimeout(ctx, opts.RequestTimeout)
	defer cancel()

	resMeta := gmetadata.MD{}
	callOpts := []grpc.CallOption{grpc.Header(&resMeta)}

	if opts.ContentType == codecs.MimeJSON {
		callOpts = append(callOpts, grpc.CallContentSubtype("json"))
	}

	err = conn.Invoke(ctx, infos.Endpoint, req, result, callOpts...)
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

// Stream creates a bidirectional gRPC stream to the service endpoint.
func (t *Transport) Stream(ctx context.Context, infos client.RequestInfos, opts *client.CallOptions) (client.StreamIface[any, any], error) {
	var cancel context.CancelFunc

	// Apply timeout to the stream if configured.
	if opts.StreamTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.StreamTimeout)
	} else {
		// The caller will handle cancellation when the stream is closed.
		ctx, cancel = context.WithCancel(ctx)
	}

	// Append go-orb metadata to grpc.
	kv := []string{}
	for k, v := range opts.Metadata {
		kv = append(kv, k, v)
	}

	ctx = gmetadata.AppendToOutgoingContext(ctx, kv...)

	callOpts := []grpc.CallOption{}

	if opts.ContentType == codecs.MimeJSON {
		callOpts = append(callOpts, grpc.CallContentSubtype("json"))
	}

	// Get an existing connection from the pool.
	conn, err := t.pool.Get(ctx, infos.Address, opts.TLSConfig)
	if err != nil {
		cancel()
		return nil, toOrbError(err)
	}

	// Create a new gRPC stream.
	grpcStream, err := conn.ClientConn.NewStream(ctx, &grpc.StreamDesc{
		StreamName:    infos.Endpoint,
		ServerStreams: true,
		ClientStreams: true,
	}, infos.Endpoint, callOpts...)

	if err != nil {
		conn.Unhealthy()
		_ = conn.Close() //nolint:errcheck

		cancel()

		return nil, toOrbError(err)
	}

	// Wrap the gRPC stream in our Stream interface.
	return &grpcClientStream{
		closed:               false,
		sendClosed:           false,
		stream:               grpcStream,
		ctx:                  ctx,
		conn:                 conn,
		cancel:               cancel,
		optsResponseMetadata: opts.ResponseMetadata,
	}, nil
}

// grpcClientStream wraps a gRPC stream to implement the client.Stream interface.
type grpcClientStream struct {
	stream               grpc.ClientStream
	ctx                  context.Context
	conn                 *pool.ClientConn
	cancel               context.CancelFunc
	closed               bool
	sendClosed           bool
	optsResponseMetadata map[string]string
}

// Context returns the context for this stream.
func (g *grpcClientStream) Context() context.Context {
	return g.ctx
}

// Send sends a message to the stream.
func (g *grpcClientStream) Send(msg interface{}) error {
	if g.closed {
		return orberrors.ErrBadRequest.WrapNew("stream is closed")
	}

	if g.sendClosed {
		return orberrors.ErrBadRequest.WrapNew("send direction is closed")
	}

	if err := g.stream.SendMsg(msg); err != nil {
		g.conn.Unhealthy()
		return toOrbError(err)
	}

	return nil
}

// Recv receives a message from the stream.
func (g *grpcClientStream) Recv(msg interface{}) error {
	if g.closed {
		return orberrors.ErrBadRequest.WrapNew("stream is closed")
	}

	if err := g.stream.RecvMsg(msg); err != nil {
		g.conn.Unhealthy()
		return toOrbError(err)
	}

	// Capture response metadata from the gRPC stream
	if g.optsResponseMetadata != nil {
		header, err := g.stream.Header()
		if err == nil {
			for k, v := range header {
				if len(v) > 0 {
					g.optsResponseMetadata[k] = v[0]
				}
			}
		}
	}

	return nil
}

// Close closes the stream.
func (g *grpcClientStream) Close() error {
	if g.closed {
		return nil
	}

	g.closed = true
	g.sendClosed = true

	// Cancel the context
	if g.cancel != nil {
		g.cancel()
	}

	// Close the stream
	err := g.stream.CloseSend()

	// Also return the connection to the pool
	if g.conn != nil {
		_ = g.conn.Close() //nolint:errcheck
	}

	return err
}

// CloseSend closes the send direction of the stream but leaves the receive side open.
// This allows the client to signal it's done sending while still being able to receive responses.
func (g *grpcClientStream) CloseSend() error {
	if g.closed {
		return orberrors.ErrBadRequest.WrapNew("stream is closed")
	}

	if g.sendClosed {
		return nil
	}

	g.sendClosed = true

	return g.stream.CloseSend()
}

// SendMsg is an alias for Send to satisfy the client.Stream interface.
func (g *grpcClientStream) SendMsg(m interface{}) error {
	return g.Send(m)
}

// RecvMsg is an alias for Recv to satisfy the client.Stream interface.
func (g *grpcClientStream) RecvMsg(m interface{}) error {
	return g.Recv(m)
}

// createConn creates a new grpc client with the given config.
func (t *Transport) createConn(_ context.Context, addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error) {
	if t.network == "unix" {
		return grpc.NewClient("unix://"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	var opts []grpc.DialOption

	// Setup authentication.
	switch {
	case tlsConfig != nil:
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	case t.name == "grpcs":
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true, NextProtos: []string{"h2"}}) //nolint:gosec
		opts = append(opts, grpc.WithTransportCredentials(creds))
	default:
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Setup dialer.
	opts = append(opts, grpc.WithContextDialer(func(_ context.Context, addr string) (net.Conn, error) {
		return net.DialTimeout(t.network, addr, time.Duration(t.config.DialTimeout))
	}))

	// Increase the max receive buffer size to 10MB to allow for large gRPC messages.
	// Note: this is probably only has an effect if the underlying protocol is TCP.
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(10*1024*1024),
		grpc.MaxCallSendMsgSize(10*1024*1024),
	))

	// Connect.
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// NewTransport creates a Transport.
func NewTransport(name string, network string) func(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return func(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
		return orb.TransportType{Transport: &Transport{
			logger:  logger,
			config:  cfg,
			network: network,
			name:    name,
		}}, nil
	}
}
