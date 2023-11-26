package hertz

import (
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
)

func init() {
	if err := server.Plugins.Add(Name, pluginProvider); err != nil {
		panic(err)
	}

	if err := server.NewDefaults.Add(Name, newDefaultConfig); err != nil {
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

	return ProvideServerHertz(service, logger, reg, *cfg)
}

func newDefaultConfig() server.EntrypointConfig {
	return NewConfig()
}
