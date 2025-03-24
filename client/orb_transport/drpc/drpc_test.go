package drpc

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

	"github.com/go-orb/plugins/server/drpc"

	echohandler "github.com/go-orb/plugins/client/tests/handler/echo"
	echoproto "github.com/go-orb/plugins/client/tests/proto/echo"

	filehandler "github.com/go-orb/plugins/client/tests/handler/file"
	fileproto "github.com/go-orb/plugins/client/tests/proto/file"

	// Blank imports here are fine.
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/registry/mdns"
)

func setupServer(sn string) (*tests.SetupData, error) {
	ctx, cancel := context.WithCancel(context.Background())

	setupData := &tests.SetupData{}

	logger, err := log.New(log.WithLevel(log.LevelTrace))
	if err != nil {
		cancel()

		return nil, err
	}

	reg, err := registry.New(nil, &types.Components{}, logger)
	if err != nil {
		cancel()

		return nil, err
	}

	echoHInstance := new(echohandler.Handler)
	echoHRegister := echoproto.RegisterStreamsHandler(echoHInstance)

	fileHInstance := new(filehandler.Handler)
	fileHRegister := fileproto.RegisterFileServiceHandler(fileHInstance)

	options := []server.Option{
		drpc.WithHandlers(echoHRegister, fileHRegister),
	}

	ep, err := drpc.New(sn, "", drpc.NewConfig(options...), logger, reg)
	if err != nil {
		cancel()

		return nil, err
	}

	setupData.Logger = logger
	setupData.Registry = reg
	setupData.Entrypoints = []server.Entrypoint{ep}
	setupData.Ctx = ctx
	setupData.Stop = cancel

	return setupData, nil
}

func newSuite() *tests.TestSuite {
	s := tests.NewSuite(setupServer, []string{Name})
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
