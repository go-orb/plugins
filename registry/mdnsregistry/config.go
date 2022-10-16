package mdnsregistry

import (
	"jochum.dev/orb/orb/registry"
)

type Config interface {
	registry.Config

	Domain() string
	SetDomain(n string)
}

type ConfigImpl struct {
	*registry.ConfigImpl

	Domain string
}

func NewConfig() *ConfigImpl {
	return &ConfigImpl{
		ConfigImpl: registry.NewComponentConfig(),
	}
}

func (c *ConfigImpl) GetDomain() string { return c.Domain }
