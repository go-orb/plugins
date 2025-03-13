package orb

import (
	"context"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/container"
)

// Transport is the interface for each transport.
type Transport interface {
	Start() error
	Stop(ctx context.Context) error
	Name() string

	// Request does the actual call to the service, it's important that any errors returned by Request are orberrors.
	Request(ctx context.Context, infos client.RequestInfos, req any, result any, opts *client.CallOptions) error

	// Stream opens a bidirectional stream to the service endpoint.
	// It handles streaming communication with the service.
	Stream(ctx context.Context, infos client.RequestInfos, opts *client.CallOptions) (client.StreamIface[any, any], error)
}

// TransportType is the type returned by NewTransportFunc.
type TransportType struct {
	Transport
}

// TransportFactory is used by a transports to register itself with the global "Transports" below.
type TransportFactory = func(log.Logger, *Config) (TransportType, error)

//nolint:gochecknoglobals
var (
	// Transports is a map of registered transports.
	Transports = container.NewMap[string, TransportFactory]()
)

// RegisterTransport registers a transport with the orb client.
func RegisterTransport(name string, transport TransportFactory) {
	Transports.Add(name, transport)
}
