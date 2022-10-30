package mdnsregistry

import (
	"github.com/go-orb/config/source/cli"
	"github.com/go-orb/orb/registry"
)

const name = "mdns"

var DefaultDomain = "orb"

func init() {
	_ = cli.Flags.Add(cli.NewFlag(
		"registry_domain",
		DefaultDomain,
		cli.CPSlice([]string{"registry", "domain"}),
		cli.Usage("Registry domain."),
	))

	if err := registry.Plugins.Add(name, Provide); err != nil {
		panic(err)
	}
}

type Config struct {
	*registry.Config `yaml:",inline"`

	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`
}

func NewConfig() *Config {
	return &Config{
		Config: registry.NewConfig(),
	}
}
