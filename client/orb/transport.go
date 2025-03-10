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
	String() string

	// NoCodec indicates whatever the transport does the encoding on its own.
	NeedsCodec() bool

	// Request does the actual call to the service, it's important that any errors returned by Request are orberrors.
	Request(ctx context.Context, req *client.Req[any, any], opts *client.CallOptions) (*client.RawResponse, error)

	// RequestNoCodec is the same as Request but it's using the codec from the transport.
	RequestNoCodec(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error
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
