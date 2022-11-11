package test

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go-micro.dev/v5/config"

	_ "github.com/go-micro/plugins/codecs/json"
	_ "github.com/go-micro/plugins/codecs/yaml"
	_ "github.com/go-micro/plugins/config/source/file"
	_ "github.com/go-micro/plugins/config/source/http"
)

func testSet2URLs(t *testing.T, urls []*url.URL) {
	t.Helper()

	datas, err := config.Read(urls, []string{"app"})
	require.NoError(t, err)

	// Merge all data from the URL's.
	cfg := newRegistryNatsConfig()
	err = config.Parse([]string{"app", "registry"}, datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.Equal(t, true, cfg.Enabled)
	assert.Equal(t, "nats", cfg.Plugin)
	assert.Equal(t, 600, cfg.Timeout)
	assert.Equal(t, false, cfg.Secure)
	assert.EqualValues(t, []string{"nats://localhost:4222"}, cfg.Addresses)
}

func TestSet2FileJsonYaml(t *testing.T) {
	u1, err := url.Parse("./data/set2/registry1.json")
	require.NoError(t, err)

	u2, err := url.Parse("./data/set2/registry2.yaml")
	require.NoError(t, err)

	testSet2URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet2FileYamlJson(t *testing.T) {
	u1, err := url.Parse("./data/set2/registry1.yaml")
	require.NoError(t, err)

	u2, err := url.Parse("./data/set2/registry2.json")
	require.NoError(t, err)

	testSet2URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet2HttpYamlJson(t *testing.T) {
	u1, err := url.Parse(urlRepo + "/set2/registry1.yaml")
	require.NoError(t, err)

	u2, err := url.Parse(urlRepo + "/set2/registry2.json")
	require.NoError(t, err)

	testSet2URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet2FileAndHttp(t *testing.T) {
	u1, err := url.Parse("./data/set2/registry1.json")
	require.NoError(t, err)

	u2, err := url.Parse(urlRepo + "/set2/registry2.json")
	require.NoError(t, err)

	testSet2URLs(t, []*url.URL{
		u1,
		u2,
	})
}
