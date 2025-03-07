//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.
package main

import (
	"context"
	"fmt"

	"github.com/go-orb/go-orb/cli"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/plugins/cli/urfave"
	"github.com/go-orb/plugins/client/tests/handler"
	"github.com/go-orb/plugins/client/tests/proto"

	"github.com/go-orb/wire"
)

// provideServerOpts provides options for the go-orb server.
func provideServerOpts() ([]server.ConfigOption, error) {

	hInstance := new(handler.EchoHandler)
	hRegister := proto.RegisterStreamsHandler(hInstance)

	server.Handlers.Add(proto.HandlerStreams, hRegister)

	opts := []server.ConfigOption{}
	return opts, nil
}

type wireRunResult struct{}

func wireRun(
	serviceContext *cli.ServiceContext,
	components *types.Components,
	logger log.Logger,
	_ server.Server,
) (wireRunResult, error) {
	// Orb start
	for _, c := range components.Iterate(false) {
		logger.Debug("Starting", "component", fmt.Sprintf("%s/%s", c.Type(), c.String()))

		err := c.Start(serviceContext.Context())
		if err != nil {
			logger.Error("Failed to start", "error", err, "component", fmt.Sprintf("%s/%s", c.Type(), c.String()))
			return wireRunResult{}, fmt.Errorf("failed to start component %s/%s: %w", c.Type(), c.String(), err)
		}
	}

	// Let the service work.
	<-serviceContext.Context().Done()

	// Orb shutdown.
	ctx := context.Background()

	for _, c := range components.Iterate(true) {
		logger.Debug("Stopping", "component", fmt.Sprintf("%s/%s", c.Type(), c.String()))

		err := c.Stop(ctx)
		if err != nil {
			logger.Error("Failed to stop", "error", err, "component", fmt.Sprintf("%s/%s", c.Type(), c.String()))
		}
	}

	return wireRunResult{}, nil
}

func run(
	appContext *cli.AppContext,
	args []string,
) (wireRunResult, error) {
	panic(wire.Build(
		urfave.ProvideParser,
		cli.ProvideParsedFlagsFromArgs,

		cli.ProvideSingleServiceContext,
		types.ProvideComponents,

		cli.ProvideConfigData,
		cli.ProvideServiceName,
		cli.ProvideServiceVersion,

		log.ProvideNoOpts,
		registry.ProvideNoOpts,
		provideServerOpts,
		server.Provide,

		wireRun,
	))
}
