// Package test contains tests for config-plugins.
package test

type logConfig struct {
	Plugin          string         `json:"plugin" yaml:"plugin"`
	Enabled         bool           `json:"enabled" yaml:"enabled"`
	Level           string         `json:"level" yaml:"level"`
	Fields          map[string]any `json:"fields" yaml:"fields"`
	CallerSkipFrame int            `json:"caller_skip_frame" yaml:"callerSkipFrame"`
}

func newLogConfig() *logConfig {
	return &logConfig{
		Plugin:          "jsonstderr",
		Enabled:         false,
		Level:           "info",
		CallerSkipFrame: 2,
	}
}

type registryConfig struct {
	Plugin  string     `json:"plugin" yaml:"plugin"`
	Enabled bool       `json:"enabled" yaml:"enabled"`
	Timeout int        `json:"timeout" yaml:"timeout"`
	Log     *logConfig `json:"log" yaml:"log"`
}

// newRegistryConfig creates a new config with defaults.
func newRegistryConfig() *registryConfig {
	return &registryConfig{
		Plugin:  "mdns",
		Enabled: true,
		Timeout: 600,
		Log:     newLogConfig(),
	}
}

type registryMdnsConfig struct {
	*registryConfig `yaml:",inline"`

	Domain string `json:"domain" yaml:"domain"`
}

// newRegistryMdnsConfig creates a new config with defaults.
func newRegistryMdnsConfig() *registryMdnsConfig {
	return &registryMdnsConfig{
		registryConfig: newRegistryConfig(),
		Domain:         "_orb",
	}
}

type registryNatsConfig struct {
	*registryConfig `yaml:",inline"`

	Addresses []string `json:"addresses" yaml:"addresses"`
	Secure    bool     `json:"secure" yaml:"secure"`

	QueryTopic string `json:"query_topic" yaml:"queryTopic"`
	WatchTopic string `json:"watch_topic" yaml:"watchTopic"`
}

// newRegistryNatsConfig creates a new config with defaults.
func newRegistryNatsConfig() *registryNatsConfig {
	return &registryNatsConfig{
		registryConfig: newRegistryConfig(),
		Secure:         true,
		QueryTopic:     "orb.registry.nats.query",
		WatchTopic:     "orb.registry.nats.watch",
	}
}
