//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.
package main

import (
	"net/url"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/plugins/client/tests/handler"
	"github.com/go-orb/plugins/client/tests/proto"

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

// provideServerOpts provides options for the go-orb server.
func provideServerOpts() ([]server.ConfigOption, error) {

	hInstance := new(handler.EchoHandler)
	hRegister := proto.RegisterStreamsHandler(hInstance)

	server.Handlers.Add(proto.HandlerStreams, hRegister)

	opts := []server.ConfigOption{}
	// opts = append(opts, server.WithEntrypointConfig(mgrpc.NewConfig(
	// 	mgrpc.WithName("grpc"),
	// 	mgrpc.WithInsecure(),
	// 	mgrpc.WithHandlers(hRegister),
	// )))
	// opts = append(opts, server.WithEntrypointConfig(mhttp.NewConfig(
	// 	mhttp.WithName("http"),
	// 	mhttp.WithInsecure(),
	// 	mhttp.WithHandlers(hRegister),
	// )))
	// opts = append(opts, server.WithEntrypointConfig(mhttp.NewConfig(
	// 	mhttp.WithName("https"),
	// 	mhttp.WithDisableHTTP2(),
	// 	mhttp.WithHandlers(hRegister),
	// )))
	// opts = append(opts, server.WithEntrypointConfig(mhttp.NewConfig(
	// 	mhttp.WithName("h2c"),
	// 	mhttp.WithInsecure(),
	// 	mhttp.WithAllowH2C(),
	// 	mhttp.WithHandlers(hRegister),
	// )))
	// opts = append(opts, server.WithEntrypointConfig(mhttp.NewConfig(
	// 	mhttp.WithName("http2"),
	// 	mhttp.WithHandlers(hRegister),
	// )))
	// opts = append(opts, server.WithEntrypointConfig(mhttp.NewConfig(
	// 	mhttp.WithName("http3"),
	// 	mhttp.WithHTTP3(),
	// 	mhttp.WithHandlers(hRegister),
	// )))
	// opts = append(opts, server.WithEntrypointConfig(drpc.NewConfig(
	// 	drpc.WithName("drpc"),
	// 	drpc.WithHandlers(hRegister),
	// )))

	return opts, nil
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
		log.Provide,
		wire.Value([]registry.Option{}),
		registry.Provide,
		provideServerOpts,
		server.Provide,
		provideComponents,
	))
}
