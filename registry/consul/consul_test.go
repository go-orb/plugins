package consul

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
)

func createServer() (*tests.TestSuite, func() error, error) {
	logger, err := log.New()
	if err != nil {
		log.Error("failed to create logger", err)
		os.Exit(1)
	}

	server, err := createServer1(&testing.T{})
	if err != nil {
		logger.Error("failed to create a consul server", err)
		os.Exit(1)
	}

	cfg1, err := NewConfig(types.ServiceName("test1.service"), nil, WithAddress(server.HTTPAddr))
	if err != nil {
		log.Error("failed to create config", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	reg1 := New("", "", cfg1, logger)
	if err := reg1.Start(); err != nil {
		log.Error("failed to connect registry one to Consul server", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	cfg2, err := NewConfig(types.ServiceName("test2.service"), nil, WithAddress(server.HTTPAddr))
	if err != nil {
		log.Error("failed to create config", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	reg2 := New("", "", cfg2, logger)
	if err := reg2.Start(); err != nil {
		log.Error("failed to connect registry two to Consul server", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	cfg3, err := NewConfig(types.ServiceName("test3.service"), nil, WithAddress(server.HTTPAddr))
	if err != nil {
		log.Error("failed to create config", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	reg3 := New("", "", cfg3, logger)
	if err := reg3.Start(); err != nil {
		log.Error("failed to connect registry three to Consul server", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	cleanup := func() error {
		_ = server.Stop() //nolint:errcheck
		return nil
	}

	return tests.CreateSuite(logger, []registry.Registry{reg1, reg2, reg3}, 0, 1), cleanup, nil
}

func createServer1(t testing.TB) (*testutil.TestServer, error) {
	// Compile our consul path.
	myConsulPath, err := filepath.Abs(filepath.Join("./test/bin/", runtime.GOOS+"_"+runtime.GOARCH))
	if err != nil {
		return nil, err
	}

	// Prepend path with our consul path.
	path := os.Getenv("PATH")
	t.Setenv("PATH", myConsulPath+":"+path)

	server, err := testutil.NewTestServerConfigT(t, func(c *testutil.TestServerConfig) {
		c.EnableDebug = true
	})
	if err != nil {
		return nil, err
	}

	// Revert path.
	t.Setenv("PATH", path)

	return server, nil
}

func TestSuite(t *testing.T) {
	s, cleanup, err := createServer()
	require.NoError(t, err, "while creating a server")

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
