package http

import (
	"go-micro.dev/v5/log"
	"go-micro.dev/v5/server"
	"go-micro.dev/v5/types"
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
	c any,
) (server.Entrypoint, error) {
	cfg, ok := c.(*Config)
	if !ok {
		return nil, ErrInvalidConfigType
	}

	return ProvideServerHTTP(service, logger, *cfg)
}

func newDefaultConfig() server.EntrypointConfig {
	return NewConfig()
}
