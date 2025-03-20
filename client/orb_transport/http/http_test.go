package http

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

	"github.com/go-orb/plugins/server/http"

	echohandler "github.com/go-orb/plugins/client/tests/handler/echo"
	echoproto "github.com/go-orb/plugins/client/tests/proto/echo"

	// Blank imports here are fine.
	_ "github.com/go-orb/plugins-experimental/registry/mdns"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/log/slog"
)

func setupServer(sn string) (*tests.SetupData, error) {
	ctx, cancel := context.WithCancel(context.Background())

	setupData := &tests.SetupData{}

	sv := ""

	logger, err := log.New()
	if err != nil {
		cancel()

		return nil, err
	}

	reg, err := registry.New(sn, sv, nil, &types.Components{}, logger)
	if err != nil {
		cancel()

		return nil, err
	}

	hInstance := new(echohandler.Handler)
	hRegister := echoproto.RegisterStreamsHandler(hInstance)

	ep1, err := http.New(
		http.NewConfig(
			server.WithEntrypointName("http"),
			http.WithHandlers(hRegister),
			http.WithInsecure(),
		),
		logger,
		reg,
	)
	if err != nil {
		cancel()

		return nil, err
	}

	ep2, err := http.New(
		http.NewConfig(
			server.WithEntrypointName("h2c"),
			http.WithHandlers(hRegister),
			http.WithInsecure(),
			http.WithAllowH2C(),
		),
		logger,
		reg,
	)
	if err != nil {
		cancel()

		return nil, err
	}
	ep3, err := http.New(
		http.NewConfig(
			server.WithEntrypointName("https"),
			http.WithHandlers(hRegister),
		),
		logger,
		reg,
	)
	if err != nil {
		cancel()

		return nil, err
	}
	ep4, err := http.New(
		http.NewConfig(
			server.WithEntrypointName("http3"),
			http.WithHandlers(hRegister),
			http.WithHTTP3(),
		),
		logger,
		reg,
	)
	if err != nil {
		cancel()

		return nil, err
	}

	setupData.Logger = logger
	setupData.Registry = reg
	setupData.Entrypoints = []server.Entrypoint{ep1, ep2, ep3, ep4}
	setupData.Ctx = ctx
	setupData.Stop = cancel

	return setupData, nil
}

func newSuite() *tests.TestSuite {
	s := tests.NewSuite(setupServer, []string{"http", "h2c", "https", "http3"})
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
