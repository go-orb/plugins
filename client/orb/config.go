package orb

import (
	"fmt"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/types"
)

// Name contains the plugins name.
const Name = "orb"

func init() {
	client.Register(Name, ProvideClientOrb)
}

// Config is the config for the orb client.
type Config struct {
	client.Config `yaml:",inline"`
}

// NewConfig creates a new config object.
func NewConfig(
	serviceName types.ServiceName,
	datas types.ConfigData,
	opts ...client.Option,
) (Config, error) {
	cfg := Config{
		Config: client.NewConfig(),
	}

	// Apply options.
	for _, o := range opts {
		o(&cfg)
	}

	sections := types.SplitServiceName(serviceName)
	if err := config.Parse(append(sections, client.ComponentType), datas, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
