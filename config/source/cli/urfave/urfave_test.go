package urfave

import (
	"testing"

	"github.com/go-orb/go-orb/config"
	oCli "github.com/go-orb/go-orb/config/source/cli"
	"github.com/go-orb/go-orb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/urfave/cli/v2"

	_ "github.com/go-orb/plugins/codecs/json"
)

const (
	FlagString = "string"
	FlagInt    = "int"
)

type testConfig struct {
	String string `json:"string"`
	Int    int    `json:"int"`
}

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
		&cli.App{
			Name:   "testapp",
			Usage:  "A testapp",
			Action: func(ctx *cli.Context) error { return nil },
		},
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

	var cfg testConfig
	require.NoError(t, config.Parse([]string{"orb", "registry"}, types.ConfigData{result}, &cfg))
	assert.Equal(t, "demo", cfg.String)
	assert.Equal(t, 42, cfg.Int)
}
