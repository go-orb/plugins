package urfave

import (
	"testing"

	oCli "github.com/go-orb/go-orb/config/source/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/urfave/cli/v2"
)

const (
	FlagString = "string"
	FlagInt    = "int"
)

func TestParse(t *testing.T) {
	flagString := oCli.NewFlag(
		FlagString,
		"orb!1!1",
		oCli.ConfigPathSlice([]string{"orb", "registry", FlagString}),
		oCli.Usage("string flag usage"),
	)

	flagInt := oCli.NewFlag(
		FlagInt,
		0,
		oCli.ConfigPathSlice([]string{"orb", "registry", FlagInt}),
		oCli.Usage("int flag usage"),
	)

	result := Parse(
		cli.NewApp(),
		[]*oCli.Flag{flagString, flagInt},
		[]string{
			"testapp",
			"--string",
			"demo",
			"--int",
			"42",
		},
	)
	require.NoError(t, result.Error)

	vString, err := oCli.FlagValue[string](flagString)
	require.NoError(t, err)
	assert.Equal(t, "demo", vString)

	vInt, err := oCli.FlagValue[int](flagInt)
	require.NoError(t, err)
	assert.Equal(t, 42, vInt)
}
