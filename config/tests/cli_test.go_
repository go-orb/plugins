package test

import (
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source"
	"github.com/go-orb/go-orb/config/source/cli"

	_ "github.com/go-orb/plugins/codecs/yaml"
	"github.com/go-orb/plugins/config/source/cli/urfave"
)

func testSections(t *testing.T, sections []string) {
	t.Helper()

	flags := cli.Flags.Clone()

	err := flags.Add(cli.NewFlag(
		"registry",
		"mdns",
		cli.ConfigPath("registry.plugin"),
		cli.Usage("string flag usage"),
	))
	require.NoError(t, err)

	err = flags.Add(cli.NewFlag(
		"registry_timeout",
		300,
		cli.ConfigPathSlice([]string{"registry", "timeout"}),
		cli.Usage("int flag usage"),
	))
	require.NoError(t, err)

	err = flags.Add(cli.NewFlag(
		"nats-address",
		[]string{},
		cli.ConfigPathSlice([]string{"registry", "addresses"}),
		cli.Usage("NATS Address"),
	))
	require.NoError(t, err)

	os.Args = []string{
		"testapp",
		"--registry",
		"nats",
		"--registry_timeout",
		"600",
		"--nats-address",
		"nats://localhost:4222",
	}

	// Setup the CLI parser.
	app, err := urfave.ProvideApp("app", "v1.0.0")
	require.NoError(t, err)
	parser, err := urfave.ProvideParserFunc(app, flags.List())
	require.NoError(t, err)
	source.Plugins.Set(cli.New(parser))

	u1, err := url.Parse("cli:///?add_section=true")
	require.NoError(t, err)

	datas, err := config.Read([]*url.URL{u1}, sections)
	require.NoError(t, err)

	require.NoError(t, config.Dump(datas))

	// Merge all data from the URL's.
	cfg := newRegistryNatsConfig()
	err = config.Parse(append(sections, "registry"), datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.True(t, cfg.Enabled, "Enabled by default")
	assert.Equal(t, "nats", cfg.Plugin, "Plugin")
	assert.Equal(t, 600, cfg.Timeout, "Timeout")
	assert.True(t, cfg.Secure, "Secure by default")
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses, "Addresses")
}

func TestCliSingleSection(t *testing.T) {
	testSections(t, []string{"app"})
}

func TestCliNoSection(t *testing.T) {
	testSections(t, []string{})
}

func TestCliMultiSection(t *testing.T) {
	testSections(t, []string{"com", "example", "abc"})
}
