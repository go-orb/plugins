package grpc

import (
	"errors"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"google.golang.org/grpc"
)

// Plugin name.
const Plugin = "grpc"

func init() {
	if err := server.Plugins.Add(Plugin, pluginProvider); err != nil {
		panic(err)
	}

	if err := server.NewDefaults.Add(Plugin, newDefaultConfig); err != nil {
		panic(err)
	}
}

//nolint:gochecknoglobals
var (
	// UnaryInterceptors is a plugin container for unary interceptors middleware.
	UnaryInterceptors = container.NewPlugins[grpc.UnaryServerInterceptor]()

	// StreamInterceptors is a plugin container for streaming interceptors middleware.
	StreamInterceptors = container.NewPlugins[grpc.StreamServerInterceptor]()
)

// Errors.
var (
	ErrInvalidConfigType = errors.New("http server: invalid config type provided, not of type http.Config")
)

func pluginProvider(
	service types.ServiceName,
	logger log.Logger,
	c any,
) (server.Entrypoint, error) {
	cfg, ok := c.(*Config)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	return ProvideServerGRPC(service, logger, *cfg)
}

func newDefaultConfig() server.EntrypointConfig {
	return NewConfig()
}
