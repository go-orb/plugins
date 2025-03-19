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
func (t *MemoryTransport) Request(ctx context.Context, infos client.RequestInfos, req any, result any, opts *client.CallOptions) error {
	server, err := client.ResolveMemoryServer(infos.Service)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(err)
	}

	return server.Request(ctx, infos, req, result, opts)
}

// Stream opens a bidirectional stream to the memory server.
func (t *MemoryTransport) Stream(
	_ context.Context, infos client.RequestInfos,
	_ *client.CallOptions,
) (client.StreamIface[any, any], error) {
	t.logger.Debug("creating stream to memory server", "service", infos.Service)

	// server, err := client.ResolveMemoryServer(infos.Service)
	// if err != nil {
	// 	return nil, orberrors.ErrInternalServerError.Wrap(err)
	// }

	return nil, orberrors.ErrNotImplemented
}

// NewTransport creates a Transport.
func NewTransport(logger log.Logger, _ *Config) (TransportType, error) {
	return TransportType{Transport: &MemoryTransport{
		logger: logger,
	}}, nil
}
