package http

import (
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
)

func init() {
	if err := server.Plugins.Add(Plugin, pluginProvider); err != nil {
		panic(err)
	}

	if err := server.NewDefaults.Add(Plugin, newDefaultConfig); err != nil {
		panic(err)
	}
}

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

	return ProvideServerHTTP(service, logger, reg, *cfg)
}

func newDefaultConfig() server.EntrypointConfig {
	return NewConfig()
}
