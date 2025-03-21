package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	_ "github.com/go-orb/plugins/codecs/json"
)

var testJSON = `{
	"string": "value",
	"stringslice": [
		"value1",
		"value2",
		0,
		true
	],
	"mixedslice": [
		"value1",
		0,
		1,
		2
	],
	"stringmap": {
		"key1": "value1",
		"key2": "value2",
		"key3": 1,
		"key4": true
	},
	"slicemap": [
		{"name": "0", "key": "value0"},
		{"name": "1", "key": "value1"}
	]
}`

func testData(t *testing.T) map[string]any {
	t.Helper()

	codec, err := codecs.GetMime(codecs.MimeJSON)
	if err != nil {
		t.Fatalf("%v", err)
	}

	data := make(map[string]any)
	if err := codec.Unmarshal([]byte(testJSON), &data); err != nil {
		t.Fatalf("error while reading testJSON: %v", err)
	}

	return data
}

func TestReadString(t *testing.T) {
	data := testData(t)

	// Must return the correct value.
	str, err := config.SingleGet(data, "string", "x")
	require.NoError(t, err)
	assert.Equal(t, "value", str)

	// Must return default if type don't match
	i, err := config.SingleGet(data, "string", 10)
	require.ErrorIs(t, err, config.ErrTypesDontMatch)
	assert.Equal(t, 10, i)

	// Must return default
	str, err = config.SingleGet(data, "string2", "x")
	require.ErrorIs(t, err, config.ErrNoSuchKey)
	assert.Equal(t, "x", str)
}

func TestReadStringSlice(t *testing.T) {
	data := testData(t)

	// Must return the correct value.
	strs, err := config.SingleGet(data, "stringslice", []string{})
	require.NoError(t, err)
	assert.Equal(t, []string{"value1", "value2", "0", "true"}, strs)

	// Must return error if not a slice
	_, err = config.SingleGet(data, "string", []string{})
	require.ErrorIs(t, err, config.ErrTypesDontMatch)

	// Must return default
	strs, err = config.SingleGet(data, "stringslice2", []string{"a", "b"})
	require.ErrorIs(t, err, config.ErrNoSuchKey)
	assert.Equal(t, []string{"a", "b"}, strs)
}

func TestReadMixedSlice(t *testing.T) {
	data := testData(t)

	// Must return the correct value.
	anys, err := config.SingleGet(data, "mixedslice", []any{})
	require.NoError(t, err)
	assert.Equal(t, []any{"value1", float64(0), float64(1), float64(2)}, anys)

	// Must return error if not a slice
	_, err = config.SingleGet(data, "string", []any{})
	require.ErrorIs(t, err, config.ErrTypesDontMatch)
}

func TestReadMixedMap(t *testing.T) {
	data := testData(t)

	// Must return the correct value.
	mapa, err := config.SingleGet(data, "stringmap", map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"key1": "value1", "key2": "value2", "key3": float64(1), "key4": true}, mapa)

	// Must return error if not a slice
	_, err = config.SingleGet(data, "string", map[string]any{})
	require.ErrorIs(t, err, config.ErrTypesDontMatch)
}

func TestReadSliceMap(t *testing.T) {
	data := testData(t)

	// Must return the correct value.
	mapa, err := config.SingleGet(data, "slicemap", []any{})
	require.NoError(t, err)
	assert.Equal(t, []any{map[string]any{"name": "0", "key": "value0"}, map[string]any{"name": "1", "key": "value1"}}, mapa)

	// Must return error if not a slice
	_, err = config.SingleGet(data, "string", []any{})
	require.ErrorIs(t, err, config.ErrTypesDontMatch)
}
