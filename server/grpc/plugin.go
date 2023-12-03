package grpc

import (
	"errors"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"google.golang.org/grpc"
)

// Plugin name.
const Plugin = "grpc"

func init() {
	server.Plugins.Add(Plugin, pluginProvider)
	server.NewDefaults.Add(Plugin, newDefaultConfig)
}

//nolint:gochecknoglobals
var (
	// UnaryInterceptors is a plugin container for unary interceptors middleware.
	UnaryInterceptors = container.NewMap[string, grpc.UnaryServerInterceptor]()

	// StreamInterceptors is a plugin container for streaming interceptors middleware.
	StreamInterceptors = container.NewMap[string, grpc.StreamServerInterceptor]()
)

// Errors.
var (
	ErrInvalidConfigType = errors.New("http server: invalid config type provided, not of type http.Config")
)

func pluginProvider(
	service types.ServiceName,
	logger log.Logger,
	reg registry.Type,
	c any,
) (server.Entrypoint, error) {
	cfg, ok := c.(*Config)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	return ProvideServerGRPC(service, logger, reg, *cfg)
}

func newDefaultConfig() server.EntrypointConfig {
	return NewConfig()
}
