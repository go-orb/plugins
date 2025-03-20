package test

import (
	"testing"

	"github.com/go-orb/go-orb/config"
	"github.com/stretchr/testify/require"
)

func TestHasKey(t *testing.T) {
	datas := []map[string]any{
		{
			"com": map[string]any{
				"test": map[string]any{
					"registry": map[string]any{
						"plugin": "mdns",
					},
				},
			},
		},
		{
			"com": map[string]any{
				"test": map[string]any{
					"client": map[string]any{
						"plugin": "orb",
						"middlewares": []map[string]any{
							{
								"name":   "m1",
								"plugin": "log",
							},
							{
								"name":   "m2",
								"plugin": "trace",
							},
						},
					},
				},
			},
		},
	}

	data := map[string]any{}
	for _, d := range datas {
		require.NoError(t, config.Merge(&data, d))
	}

	require.True(t, config.HasKey[[]map[string]any]([]string{"com", "test", "client"}, "middlewares", data),
		"Should have key com.test.client.middlewares",
	)
	require.False(t, config.HasKey[[]map[string]any](
		[]string{"com", "test", "client"}, "middlewares2", data),
		"Should not have com.test.client.middlewares2",
	)
}
