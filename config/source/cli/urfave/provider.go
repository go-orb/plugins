package urfave

import (
	"os"

	"github.com/go-orb/go-orb/types"
	"github.com/urfave/cli/v2"

	"github.com/go-orb/go-orb/config/source"
	oCli "github.com/go-orb/go-orb/config/source/cli"
)

// ProvideApp provides a new app with the given service name and version for ProvideParseFunc.
func ProvideApp(
	serviceName types.ServiceName,
	serviceVersion types.ServiceVersion,
) (*cli.App, error) {
	app := cli.NewApp()
	app.Name = string(serviceName)
	app.Version = string(serviceVersion)
	app.Usage = "A go-orb app"
	app.Action = func(c *cli.Context) error {
		return nil
	}

	return app, nil
}

// ProvideFlags return the flags for ProvideParseFunc.
func ProvideFlags() ([]*oCli.Flag, error) {
	return oCli.Flags.List(), nil
}

// ProvideParserFunc returns a parser for go-orb/config/source/cli.
func ProvideParserFunc(app *cli.App, flags []*oCli.Flag) (oCli.ParseFunc, error) {
	parse := func() source.Data {
		return Parse(app, flags, os.Args)
	}

	return parse, nil
}

// ProvideConfigData provides configData from cli by calling go-orb/config/source/cli:ProvideConfigData.
func ProvideConfigData(serviceName types.ServiceName, serviceVersion types.ServiceVersion) (types.ConfigData, error) {
	app, err := ProvideApp(serviceName, serviceVersion)
	if err != nil {
		return nil, err
	}

	flags, err := ProvideFlags()
	if err != nil {
		return nil, err
	}

	parser, err := ProvideParserFunc(app, flags)
	if err != nil {
		return nil, err
	}

	return oCli.ProvideConfigData(serviceName, parser)
}
