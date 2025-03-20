package orb

import (
	"github.com/go-orb/go-orb/client"
)

// Name contains the plugins name.
const Name = "orb"

func init() {
	client.Register(Name, Provide)
}

// Config is the config for the orb client.
type Config struct {
	client.Config `yaml:",inline"`
}

// NewConfig creates a new config object.
func NewConfig(
	opts ...client.Option,
) Config {
	cfg := Config{
		Config: client.NewConfig(),
	}

	// Apply options.
	for _, o := range opts {
		o(&cfg)
	}

	return cfg
}
