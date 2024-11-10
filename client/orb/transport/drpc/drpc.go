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

	"google.golang.org/protobuf/proto"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/go-orb/plugins/server/drpc/message"
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
	orb.RegisterTransport(Name, NewTransport)
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

	// Add metadata to drpc.
	md, ok := metadata.Outgoing(ctx)
	if ok {
		ctx = drpcmetadata.AddPairs(ctx, md)
	}

	ctx, cancel := context.WithDeadline(ctx, time.Now().Add(opts.RequestTimeout))
	defer cancel()

	mdResult := &message.Response{}
	if err := conn.Invoke(ctx, "/"+req.Endpoint(), &t.encoder, req.Request(), mdResult); err != nil {
		orbError := orberrors.HTTP(int(drpcerr.Code(err))) //nolint:gosec
		orbError = orbError.Wrap(err)

		return orbError
	}

	// Unmarshal the result.
	if err := mdResult.GetData().UnmarshalTo(result.(proto.Message)); err != nil {
		return orberrors.From(err)
	}

	// Retrieve metadata from drpc.
	if opts.ResponseMetadata != nil {
		for k, v := range mdResult.GetMetadata() {
			opts.ResponseMetadata[k] = v
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
