package test

import (
	"net/url"
	"os"
	"testing"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source/cli"
	_ "github.com/go-orb/plugins/config/source/cli/urfave"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliConfig(t *testing.T) {
	// Clear flags from previous tests.
	cli.Flags.Clear()

	// Setup os.Args
	os.Args = []string{
		"testapp",
		"--config",
		"./data/set2/registry2.yaml",
	}

	// Setup the urls.
	u1, err := url.Parse("./data/set2/registry1.yaml")
	require.NoError(t, err)

	u2, err := url.Parse("cli://urfave")
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
