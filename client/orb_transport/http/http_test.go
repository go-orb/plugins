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
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/registry/mdns"
)

func setupServer(sn string) (*tests.SetupData, error) {
	ctx, cancel := context.WithCancel(context.Background())

	setupData := &tests.SetupData{}

	logger, err := log.New()
	if err != nil {
		cancel()

		return nil, err
	}

	reg, err := registry.New(nil, &types.Components{}, logger)
	if err != nil {
		cancel()

		return nil, err
	}

	hInstance := new(echohandler.Handler)
	hRegister := echoproto.RegisterStreamsHandler(hInstance)

	ep1, err := http.New(
		sn,
		"",
		"http",
		http.NewConfig(
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
		sn,
		"",
		"https",
		http.NewConfig(
			http.WithHandlers(hRegister),
		),
		logger,
		reg,
	)
	if err != nil {
		cancel()

		return nil, err
	}

	ep3, err := http.New(
		sn,
		"",
		"http3",
		http.NewConfig(
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
	setupData.Entrypoints = []server.Entrypoint{ep1, ep2, ep3}
	setupData.Ctx = ctx
	setupData.Stop = cancel

	return setupData, nil
}

func newSuite() *tests.TestSuite {
	s := tests.NewSuite(setupServer, []string{"http", "https", "http3"})
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
