package test

import (
	"net/url"
	"testing"

	"github.com/go-orb/go-orb/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/config/source/http"
)

const urlRepo = "https://raw.githubusercontent.com/go-orb/plugins/main/config/tests/data"

func testSet1URLs(t *testing.T, urls []*url.URL) {
	t.Helper()

	datas := map[string]any{}

	for _, url := range urls {
		d, err := config.Read(url)
		require.NoError(t, err)

		require.NoError(t, config.Merge(&datas, d))
	}

	// Merge all data from the URL's.
	cfg := newRegistryMdnsConfig()
	err := config.Parse([]string{"app"}, "registry", datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.True(t, cfg.Enabled)
	assert.Equal(t, "mdns", cfg.Plugin)
	assert.Equal(t, "app", cfg.Domain)
	assert.Equal(t, 600, cfg.Timeout)
}

func TestSet1FileJsonYaml(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1.json")
	require.NoError(t, err)

	u2, err := url.Parse("./data/set1/registry2.yaml")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet1FileYamlJson(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1.yaml")
	require.NoError(t, err)

	u2, err := url.Parse("./data/set1/registry2.json")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet1HttpYamlJson(t *testing.T) {
	u1, err := url.Parse(urlRepo + "/set1/registry1.yaml")
	require.NoError(t, err)

	u2, err := url.Parse(urlRepo + "/set1/registry2.json")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet1FileAndHttp(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1.json")
	require.NoError(t, err)

	u2, err := url.Parse(urlRepo + "/set1/registry2.json")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet1FailUnknown(t *testing.T) {
	u3, err := url.Parse("./data/set1/unknown.yaml")
	require.NoError(t, err)

	_, err = config.Read(u3)
	require.ErrorIs(t, err, config.ErrFileNotFound)
}
