package grpc

import (
	"context"
	"os"
	"testing"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/plugins/client/tests"
	"github.com/stretchr/testify/suite"

	"github.com/go-orb/plugins/server/grpc"

	echohandler "github.com/go-orb/plugins/client/tests/handler/echo"
	echoproto "github.com/go-orb/plugins/client/tests/proto/echo"

	filehandler "github.com/go-orb/plugins/client/tests/handler/file"
	fileproto "github.com/go-orb/plugins/client/tests/proto/file"

	// Blank imports here are fine.
	_ "github.com/go-orb/plugins-experimental/registry/mdns"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/log/slog"
)

func setupServer(sn types.ServiceName) (*tests.SetupData, error) {
	ctx, cancel := context.WithCancel(context.Background())

	setupData := &tests.SetupData{}

	sv := types.ServiceVersion("")

	logger, err := log.New()
	if err != nil {
		cancel()

		return nil, err
	}

	reg, err := registry.New(sn, sv, &types.Components{}, registry.NewConfig(), nil, []string{}, logger)
	if err != nil {
		cancel()

		return nil, err
	}

	hInstance := new(echohandler.Handler)
	hRegister := echoproto.RegisterStreamsHandler(hInstance)

	fileHInstance := new(filehandler.Handler)
	fileHRegister := fileproto.RegisterFileServiceHandler(fileHInstance)

	ep1, err := grpc.New(grpc.NewConfig(server.WithEntrypointName("grpc"), grpc.WithHandlers(hRegister, fileHRegister), grpc.WithInsecure()), logger, reg)
	if err != nil {
		cancel()

		return nil, err
	}

	ep2, err := grpc.New(grpc.NewConfig(server.WithEntrypointName("grpcs"), grpc.WithHandlers(hRegister, fileHRegister)), logger, reg)
	if err != nil {
		cancel()

		return nil, err
	}

	setupData.Logger = logger
	setupData.Registry = reg
	setupData.Entrypoints = []server.Entrypoint{ep1, ep2}
	setupData.Ctx = ctx
	setupData.Stop = cancel

	return setupData, nil
}

func newSuite() *tests.TestSuite {
	s := tests.NewSuite(setupServer, []string{"grpc", "grpcs"})
	// s.Debug = true
	return s
}

func TestSuite(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	// Run the tests.
	suite.Run(t, newSuite())
}
