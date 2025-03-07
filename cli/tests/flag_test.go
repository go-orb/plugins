package tests

import (
	"testing"

	"github.com/go-orb/go-orb/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringFlag(t *testing.T) {
	flag := cli.NewFlag(
		"string",
		"",
		cli.FlagConfigPaths(cli.FlagConfigPath{Path: []string{"registry", "string"}}),
		cli.FlagUsage("demo String flag"),
	)

	// Initial value of flag must be nil
	require.Nil(t, flag.Value)

	flag.Value = int64(0)
	_, err := cli.FlagValue[string](flag)
	require.Error(t, err)

	flag.Value = "somevalue"
	v, err := cli.FlagValue[string](flag)
	require.NoError(t, err)
	assert.Equal(t, "somevalue", v)
}

func TestIntFlag(t *testing.T) {
	flag := cli.NewFlag(
		"int",
		300,
		cli.FlagConfigPaths(cli.FlagConfigPath{Path: []string{"registry", "int"}}),
		cli.FlagUsage("demo Int flag"),
	)

	// Initial value of flag must be nil
	require.Nil(t, flag.Value)

	flag.Value = ""
	_, err := cli.FlagValue[int](flag)
	require.Error(t, err)

	flag.Value = 10
	v, err := cli.FlagValue[int](flag)
	require.NoError(t, err)
	assert.Equal(t, 10, v)
}

func TestStringSliceFlag(t *testing.T) {
	flag := cli.NewFlag(
		"stringslice",
		[]string{"1", "2"},
		cli.FlagConfigPaths(cli.FlagConfigPath{Path: []string{"registry", "stringslice"}}),
		cli.FlagUsage("demo StringSlice flag"),
	)

	// Initial value of flag must be nil
	require.Nil(t, flag.Value)

	flag.Value = ""
	_, err := cli.FlagValue[[]string](flag)
	require.Error(t, err)

	flag.Value = []string{"a", "b"}
	v, err := cli.FlagValue[[]string](flag)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, v)
}
