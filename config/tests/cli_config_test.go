package test

import (
	"errors"
	"net/url"
	"os"
	"testing"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source"
	"github.com/go-orb/go-orb/config/source/cli"
	"github.com/go-orb/plugins/config/source/cli/urfave"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliConfig(t *testing.T) {
	flags := cli.Flags.Clone()

	// Setup os.Args
	os.Args = []string{
		"testapp",
		"--config",
		"./data/set2/registry2.yaml",
	}

	// Setup the urls.
	u1, err := url.Parse("./data/set2/registry1.yaml")
	require.NoError(t, err)

	// Setup the CLI parser.
	app, err := urfave.ProvideApp("app", "v1.0.0")
	require.NoError(t, err)
	parser, err := urfave.ProvideParserFunc(app, flags.List())
	require.NoError(t, err)
	source.Plugins.Set(cli.New(parser))

	u2, err := url.Parse("cli://")
	require.NoError(t, err)

	// Read the urls.
	datas, err := config.Read([]*url.URL{u1, u2}, []string{"app"})
	require.NoError(t, err)

	cfg := newRegistryNatsConfig()
	err = config.Parse([]string{"app", "registry"}, datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.True(t, cfg.Enabled, "Enabled")
	assert.Equal(t, "nats", cfg.Plugin, "Plugin")
	assert.Equal(t, 600, cfg.Timeout, "Timeout")
	assert.False(t, cfg.Secure, "Secure")
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses, "Addresses")
}

func TestCliConfigWithFlags(t *testing.T) {
	flags := cli.Flags.Clone()

	// Test with some common flags.
	err := flags.Add(cli.NewFlag(
		"registry",
		"mdns",
		cli.ConfigPathSlice([]string{"registry", "plugin"}),
		cli.Usage("Registry for discovery. etcd, mdns"),
		cli.EnvVars("REGISTRY"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = flags.Add(cli.NewFlag(
		"registry_timeout",
		100,
		cli.ConfigPathSlice([]string{"registry", "timeout"}),
		cli.Usage("Registry timeout in milliseconds."),
		cli.EnvVars("REGISTRY_TIMEOUT"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	// Setup os.Args
	os.Args = []string{
		"testapp",
		"--config",
		"./data/set2/registry2.yaml",
	}

	// Setup the CLI parser.
	app, err := urfave.ProvideApp("app", "v1.0.0")
	require.NoError(t, err)
	parser, err := urfave.ProvideParserFunc(app, flags.List())
	require.NoError(t, err)
	source.Plugins.Set(cli.New(parser))

	u1, err := url.Parse("cli://")
	require.NoError(t, err)

	// Read the urls.
	datas, err := config.Read([]*url.URL{u1}, []string{"app"})
	require.NoError(t, err)

	require.NoError(t, config.Dump(datas))

	cfg := newRegistryNatsConfig()
	err = config.Parse([]string{"app", "registry"}, datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.True(t, cfg.Enabled, "Enabled")
	assert.Equal(t, "nats", cfg.Plugin, "Plugin")
	assert.Equal(t, 600, cfg.Timeout, "Timeout")
	assert.False(t, cfg.Secure, "Secure")
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses, "Addresses")
}
