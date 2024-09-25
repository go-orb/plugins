// Package drpc provides a drpc transport for client/orb.
package drpc

import (
	"context"
	"errors"
	"net"
	"time"

	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcpool"

	"google.golang.org/protobuf/proto"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
)

var _ drpc.Encoding = (*encoder)(nil)

type encoder struct{}

func (e *encoder) Marshal(msg drpc.Message) ([]byte, error) {
	message, ok := msg.(proto.Message)
	if !ok {
		return nil, errors.New("unable to marshal non proto field")
	}

	return proto.Marshal(message)
}

func (e *encoder) Unmarshal(data []byte, msg drpc.Message) error {
	message, ok := msg.(proto.Message)
	if !ok {
		return errors.New("unable to unmarshal non proto field")
	}

	return proto.Unmarshal(data, message)
}

// Name is the transports name.
const Name = "drpc"

func init() {
	orb.Transports.Register(Name, NewTransport)
}

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	logger  log.Logger
	pool    *drpcpool.Pool[string, drpcpool.Conn]
	encoder encoder
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

// Call is a noop for grpc.
func (t *Transport) Call(_ context.Context, _ *client.Request[any, any], _ *client.CallOptions) (*client.RawResponse, error) {
	return nil, orberrors.ErrInternalServerError
}

// CallNoCodec does the actual rpc call to the server.
func (t *Transport) CallNoCodec(ctx context.Context, req *client.Request[any, any], result any, opts *client.CallOptions) error {
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

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(opts.RequestTimeout))
	defer cancel()

	err = conn.Invoke(ctx, "/"+req.Endpoint(), &t.encoder, req.Request(), result)
	if err != nil {
		return orberrors.From(err)
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
