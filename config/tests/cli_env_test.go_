package test

import (
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

func TestCLIEnv(t *testing.T) {
	flags := cli.Flags.Clone()

	err := flags.Add(cli.NewFlag(
		"registry",
		"mdns",
		cli.EnvVars("ORB_REGISTRY"),
		cli.ConfigPath("registry.plugin"),
		cli.Usage("string flag usage"),
	))
	require.NoError(t, err)

	err = flags.Add(cli.NewFlag(
		"registry_ttl",
		300,
		cli.EnvVars("ORB_REGISTRY_TIMEOUT"),
		cli.ConfigPathSlice([]string{"registry", "timeout"}),
		cli.Usage("int flag usage"),
	))
	require.NoError(t, err)

	err = flags.Add(cli.NewFlag(
		"nats-address",
		[]string{},
		cli.EnvVars("ORB_REGISTRY_NATS_ADDRESS"),
		cli.ConfigPathSlice([]string{"registry", "addresses"}),
		cli.Usage("NATS Address"),
	))
	require.NoError(t, err)

	// Set Environ variables.
	t.Setenv("ORB_REGISTRY", "nats")
	t.Setenv("ORB_REGISTRY_TIMEOUT", "301")
	t.Setenv("ORB_REGISTRY_NATS_ADDRESS", "nats://localhost:4222")

	os.Args = []string{
		"testapp",
	}

	// Setup the CLI parser.
	app, err := urfave.ProvideApp("app", "v1.0.0")
	require.NoError(t, err)
	parser, err := urfave.ProvideParserFunc(app, flags.List())
	require.NoError(t, err)
	source.Plugins.Set(cli.New(parser))

	u1, err := url.Parse("cli://")
	require.NoError(t, err)

	datas, err := config.Read([]*url.URL{u1}, []string{})
	require.NoError(t, err)

	require.NoError(t, config.Dump(datas))

	// Merge all data from the URL's.
	cfg := newRegistryNatsConfig()
	err = config.Parse([]string{"registry"}, datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.True(t, cfg.Enabled, "Enabled by default")
	assert.Equal(t, "nats", cfg.Plugin, "Plugin")
	assert.Equal(t, 301, cfg.Timeout, "Timeout")
	assert.True(t, cfg.Secure, "Secure by default")
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses, "Addresses")
}
