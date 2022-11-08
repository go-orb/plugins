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

func testSet1URLs(t *testing.T, urls []*url.URL) {
	t.Helper()

	// Read the urls.
	datas, err := config.Read(urls, []string{"app"})
	require.NoError(t, err)

	// Merge all data from the URL's.
	cfg := newRegistryMdnsConfig()
	err = config.Parse([]string{"app", "registry"}, datas, cfg)
	require.NoError(t, err)

	// Check if it merges right.
	assert.Equal(t, true, cfg.Enabled)
	assert.Equal(t, "mdns", cfg.Plugin)
	assert.Equal(t, "app", cfg.Domain)
	assert.Equal(t, 600, cfg.Timeout)
}

func TestSet1FileNoExt(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1")
	require.NoError(t, err)

	u2, err := url.Parse("./data/set1/registry2")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
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
	u1, err := url.Parse("https://raw.githubusercontent.com/go-orb/config-plugins/main/test/data/set1/registry1.yaml")
	require.NoError(t, err)

	u2, err := url.Parse("https://raw.githubusercontent.com/go-orb/config-plugins/main/test/data/set1/registry2.json")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet1FileNoExtAndHttp(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1")
	require.NoError(t, err)

	u2, err := url.Parse("https://raw.githubusercontent.com/go-orb/config-plugins/main/test/data/set1/registry2.json")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
	})
}

func TestSet1IgnoreUnknown(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1")
	require.NoError(t, err)

	u2, err := url.Parse("https://raw.githubusercontent.com/go-orb/config-plugins/main/test/data/set1/registry2.json")
	require.NoError(t, err)

	u3, err := url.Parse("./data/set1/unknown.yaml?ignore_error=true")
	require.NoError(t, err)

	testSet1URLs(t, []*url.URL{
		u1,
		u2,
		u3,
	})
}

func TestSet1FailUnknown(t *testing.T) {
	u1, err := url.Parse("./data/set1/registry1")
	require.NoError(t, err)

	u2, err := url.Parse("https://raw.githubusercontent.com/go-orb/config-plugins/main/test/data/set1/registry2.json")
	require.NoError(t, err)

	u3, err := url.Parse("./data/set1/unknown.yaml")
	require.NoError(t, err)

	_, err = config.Read([]*url.URL{u1, u2, u3}, []string{"app"})
	require.Error(t, err, config.ErrNoSuchFile)
}
