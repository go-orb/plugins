package mdns

import (
	"go-micro.dev/v5/config/source/cli"
	"go-micro.dev/v5/log"
	"go-micro.dev/v5/registry"
)

const name = "mdns"

var DefaultDomain = "orb"

func init() {
	//nolint:errcheck
	_ = cli.Flags.Add(cli.NewFlag(
		"registry_domain",
		DefaultDomain,
		cli.ConfigPathSlice([]string{"registry", "domain"}),
		cli.Usage("Registry domain."),
	))

	if err := registry.Plugins.Add(name, Provide); err != nil {
		panic(err)
	}
}

type Config struct {
	*registry.Config `yaml:",inline"`

	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`
	Logger log.Logger
}

func NewConfig() *Config {
	return &Config{
		Config: registry.NewConfig(),
	}
}
