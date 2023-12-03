package hertz

import (
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
)

func init() {
	server.Plugins.Add(Name, pluginProvider)
	server.NewDefaults.Add(Name, newDefaultConfig)
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
