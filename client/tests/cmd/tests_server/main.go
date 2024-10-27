// Package main contains a server for running tests on.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"

	_ "github.com/go-orb/plugins-experimental/registry/mdns"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/jsonpb"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/cli/urfave"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/lumberjack"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/server/drpc"
	_ "github.com/go-orb/plugins/server/grpc"
	_ "github.com/go-orb/plugins/server/hertz"
	_ "github.com/go-orb/plugins/server/http"
	_ "github.com/go-orb/plugins/server/http/router/chi"
)

func main() {
	var (
		serviceName    = types.ServiceName("service")
		serviceVersion = types.ServiceVersion("v0.0.1")
	)

	components, err := newComponents(serviceName, serviceVersion)
	if err != nil {
		log.Error("while creating components", "err", err)
		os.Exit(1)
	}

	for _, c := range components {
		err := c.Start()
		if err != nil {
			log.Error("Failed to start", err, "component", c.Type())
			os.Exit(1)
		}
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// Blocks until we get a sigint/sigterm
	<-done

	// Shutdown.
	ctx := context.Background()

	for k := range components {
		c := components[len(components)-1-k]

		err := c.Stop(ctx)
		if err != nil {
			log.Error("Failed to stop", err, "component", c.Type())
		}
	}
}
