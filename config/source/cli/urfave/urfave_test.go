package urfave

import (
	"testing"

	"github.com/go-orb/go-orb/config/source/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	FlagString = "string"
	FlagInt    = "int"
)

func TestParse(t *testing.T) {
	myConfig := cli.NewConfig()
	myConfig.Name = "test"
	myConfig.Version = "v0.0.1"

	flagString := cli.NewFlag(
		FlagString,
		"orb!1!1",
		cli.ConfigPathSlice([]string{"orb", "registry", FlagString}),
		cli.Usage("string flag usage"),
	)

	flagInt := cli.NewFlag(
		FlagInt,
		0,
		cli.ConfigPathSlice([]string{"orb", "registry", FlagInt}),
		cli.Usage("int flag usage"),
	)

	err := Parse(
		&myConfig,
		[]*cli.Flag{flagString, flagInt},
		[]string{
			"testapp",
			"--string",
			"demo",
			"--int",
			"42",
		},
	)
	require.NoError(t, err)

	vString, err := cli.FlagValue[string](flagString)
	require.NoError(t, err)
	assert.Equal(t, "demo", vString)

	vInt, err := cli.FlagValue[int](flagInt)
	require.NoError(t, err)
	assert.Equal(t, 42, vInt)
}
