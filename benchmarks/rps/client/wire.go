//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.
package main

import (
	"context"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	"github.com/google/wire"
)

// provideConfigData reads the config from cli and returns it.
func provideConfigData(
	serviceName types.ServiceName,
	serviceVersion types.ServiceVersion,
) (types.ConfigData, error) {
	u, err := url.Parse("cli://urfave")
	if err != nil {
		return nil, err
	}

	cfgSections := types.SplitServiceName(serviceName)

	data, err := config.Read([]*url.URL{u}, cfgSections)

	return data, err
}

// provideComponents creates a slice of components out of the arguments.
func provideComponents(
	serviceName types.ServiceName,
	serviceVersion types.ServiceVersion,
	cfgData types.ConfigData,
	logger log.Logger,
	reg registry.Type,
	client client.Type,
) ([]types.Component, error) {
	components := []types.Component{}
	components = append(components, logger)
	components = append(components, reg)

	return components, nil
}

type wireRunResult string

type wireRunCallback func(
	serviceName types.ServiceName,
	configs types.ConfigData,
	logger log.Logger,
	cli client.Type,
) error

func wireRun(
	serviceName types.ServiceName,
	components []types.Component,
	configs types.ConfigData,
	logger log.Logger,
	cli client.Type,
	cb wireRunCallback,
) (wireRunResult, error) {
	//
	// Orb start
	for _, c := range components {
		err := c.Start()
		if err != nil {
			log.Error("Failed to start", err, "component", c.Type())
			os.Exit(1)
		}
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	//
	// Actual code
	runErr := cb(serviceName, configs, logger, cli)

	//
	// Orb shutdown.
	ctx := context.Background()

	for k := range components {
		c := components[len(components)-1-k]

		err := c.Stop(ctx)
		if err != nil {
			log.Error("Failed to stop", err, "component", c.Type())
		}
	}

	return "", runErr
}

// newComponents combines everything above and returns a slice of components.
func run(
	serviceName types.ServiceName,
	serviceVersion types.ServiceVersion,
	cb wireRunCallback,
) (wireRunResult, error) {
	panic(wire.Build(
		provideConfigData,
		wire.Value([]log.Option{}),
		log.ProvideLogger,
		wire.Value([]registry.Option{}),
		registry.ProvideRegistry,
		wire.Value([]client.Option{}),
		client.ProvideClient,
		provideComponents,
		wireRun,
	))
}
