package test

import (
	"net/url"
	"os"
	"testing"

	_ "github.com/go-micro/plugins/codecs/json"
	_ "github.com/go-micro/plugins/config/source/cli/urfave"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v5/config"
	"go-micro.dev/v5/config/source/cli"
)

func TestCliConfig(t *testing.T) {
	// Clear flags from previous tests.
	cli.Flags.Clear()

	// Setup os.Args
	os.Args = []string{
		"testapp",
		"--config",
		"./data/set2/registry2",
	}

	// Setup the urls.
	u1, err := url.Parse("./data/set2/registry1")
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
	assert.Equal(t, true, cfg.Enabled, "Enabled")
	assert.Equal(t, "nats", cfg.Plugin, "Plugin")
	assert.Equal(t, 600, cfg.Timeout, "Timeout")
	assert.Equal(t, false, cfg.Secure, "Secure")
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses, "Addresses")
}
