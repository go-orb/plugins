//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.
package main

import (
	"fmt"
	"net/url"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	echohandler "github.com/go-orb/plugins/benchmarks/rps/handler/echo"
	echopb "github.com/go-orb/plugins/benchmarks/rps/proto/echo"

	mgrpc "github.com/go-orb/plugins/server/grpc"
	// mhertz "github.com/go-orb/plugins/server/hertz"
	mhttp "github.com/go-orb/plugins/server/http"

	"github.com/google/wire"
	"github.com/hashicorp/consul/sdk/freeport"
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

// provideServerOpts provides options for the go-orb server.
// TODO(jochumdev): We should simplify server opts.
func provideServerOpts() ([]server.Option, error) {
	// Get some free ports
	ports, err := freeport.Take(7)
	if err != nil {
		return nil, err
	}

	// Our lonely handler
	hInstance := new(echohandler.Handler)

	return []server.Option{
		mgrpc.WithEntrypoint(
			mgrpc.WithName("grpc"),
			mgrpc.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[0])),
			mgrpc.WithInsecure(true),
			mgrpc.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		),
		// mhertz.WithEntrypoint(
		// 	mhertz.WithName("hertzhttp"),
		// 	mhertz.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[1])),
		// 	mhertz.WithInsecure(),
		// 	mhertz.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		// ),
		// mhertz.WithEntrypoint(
		// 	mhertz.WithName("hertzh2c"),
		// 	mhertz.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[2])),
		// 	mhertz.WithInsecure(),
		// 	mhertz.WithAllowH2C(),
		// 	mhertz.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		// ),
		mhttp.WithEntrypoint(
			mhttp.WithName("http"),
			mhttp.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[3])),
			mhttp.WithInsecure(),
			mhttp.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		),
		mhttp.WithEntrypoint(
			mhttp.WithName("h2c"),
			mhttp.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[4])),
			mhttp.WithInsecure(),
			mhttp.WithAllowH2C(),
			mhttp.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		),
		mhttp.WithEntrypoint(
			mhttp.WithName("http3"),
			mhttp.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[5])),
			mhttp.WithHTTP3(),
			mhttp.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		),
		mhttp.WithEntrypoint(
			mhttp.WithName("https"),
			mhttp.WithAddress(fmt.Sprintf("127.0.0.1:%d", ports[6])),
			mhttp.WithRegistration("Streams", echopb.OrbRegister(hInstance)),
		),
	}, nil
}

// provideComponents creates a slice of components out of the arguments.
func provideComponents(
	serviceName types.ServiceName,
	serviceVersion types.ServiceVersion,
	cfgData types.ConfigData,
	logger log.Logger,
	reg registry.Type,
	srv server.Server,
) ([]types.Component, error) {
	components := []types.Component{}
	components = append(components, logger)
	components = append(components, reg)
	components = append(components, &srv)

	return components, nil
}

// newComponents combines everything above and returns a slice of components.
func newComponents(
	serviceName types.ServiceName,
	serviceVersion types.ServiceVersion,
) ([]types.Component, error) {
	panic(wire.Build(
		provideConfigData,
		wire.Value([]log.Option{}),
		log.ProvideLogger,
		wire.Value([]registry.Option{}),
		registry.ProvideRegistry,
		provideServerOpts,
		server.ProvideServer,
		provideComponents,
	))
}
