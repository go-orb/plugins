// Package main contains a server for running tests on.
package main

import (
	"fmt"
	"os"

	"github.com/go-orb/go-orb/cli"
	_ "github.com/go-orb/plugins-experimental/registry/mdns"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/server/drpc"
	_ "github.com/go-orb/plugins/server/grpc"
	_ "github.com/go-orb/plugins/server/http"
)

func main() {
	app := cli.App{
		Name:     "service",
		Version:  "unset",
		Usage:    "A foobar example app",
		NoAction: false,
		Flags: []*cli.Flag{
			{
				Name:        "log_level",
				Default:     "INFO",
				EnvVars:     []string{"LOG_LEVEL"},
				ConfigPaths: []cli.FlagConfigPath{{Path: []string{"logger", "level"}}},
				Usage:       "Set the log level, one of TRACE, DEBUG, INFO, WARN, ERROR",
			},
		},
		Commands: []*cli.Command{},
	}

	appContext := cli.NewAppContext(&app)

	_, err := run(appContext, os.Args)
	if err != nil {
		//nolint:forbidigo
		fmt.Printf("run error: %s\n", err)
		os.Exit(1)
	}
}
