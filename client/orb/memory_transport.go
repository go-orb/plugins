package orb

import (
	"context"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/orberrors"
)

func init() {
	RegisterTransport("memory", NewTransport)
}

// MemoryTransport is a go-orb/plugins/client/orb compatible transport.
type MemoryTransport struct {
	logger log.Logger
}

// Start starts the transport.
func (t *MemoryTransport) Start() error {
	return nil
}

// Stop stop the transport.
func (t *MemoryTransport) Stop(_ context.Context) error {
	return nil
}

// Name returns the name of this transport.
func (t *MemoryTransport) Name() string {
	return "memory"
}

// Request does the actual rpc request to the server.
func (t *MemoryTransport) Request(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
	t.logger.Debug("requesting memory server", "service", req.Service())

	server, err := client.ResolveMemoryServer(req.Service())
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return server.Request(ctx, req, result, opts)
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger, _ *Config) (TransportType, error) {
	return TransportType{Transport: &MemoryTransport{
		logger: logger,
	}}, nil
}
