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

	// Call does the actual call to the service, it's important that any errors returned by Call are orberrors.
	Call(ctx context.Context, req *client.Request[any, any], opts *client.CallOptions) (*client.RawResponse, error)

	// CallNoCodec is the same as call but it's using the codec from the transport.
	CallNoCodec(ctx context.Context, req *client.Request[any, any], result any, opts *client.CallOptions) error
}

// TransportType is the type returned by NewTransportFunc.
type TransportType struct {
	Transport
}

// TransportFactory is used by a transports to register itself with the global "Transports" below.
type TransportFactory = func(log.Logger, *Config) (TransportType, error)

//nolint:gochecknoglobals
var (
	Transports = container.NewMap[string, TransportFactory]()
)

// RegisterTransport registers a transport with the orb client.
func RegisterTransport(name string, transport TransportFactory) {
	Transports.Add(name, transport)
}
