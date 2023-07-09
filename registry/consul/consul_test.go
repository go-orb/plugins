package consul

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	_ "github.com/go-orb/plugins/log/text"
	"github.com/go-orb/plugins/registry/tests"
)

func TestMain(m *testing.M) {
	logger, err := log.New(log.NewConfig())
	if err != nil {
		log.Error("failed to create logger", err)
		os.Exit(1)
	}

	server, err := createServer(&testing.T{})
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

	reg1 := New(cfg1, logger)
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

	reg2 := New(cfg2, logger)
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

	reg3 := New(cfg3, logger)
	if err := reg3.Start(); err != nil {
		log.Error("failed to connect registry three to Consul server", err)
		server.Stop() //nolint:errcheck
		os.Exit(1)
	}

	tests.CreateSuite(logger, []registry.Registry{reg1, reg2, reg3}, 0, 1)
	tests.Suite.Setup()

	exitVal := m.Run()

	tests.Suite.TearDown()

	server.Stop() //nolint:errcheck

	os.Exit(exitVal)
}

func createServer(t testing.TB) (*testutil.TestServer, error) {
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

func TestRegister(t *testing.T) {
	tests.Suite.TestRegister(t)
}

func TestDeregister(t *testing.T) {
	tests.Suite.TestDeregister(t)
}

func TestGetServiceAllRegistries(t *testing.T) {
	tests.Suite.TestGetServiceAllRegistries(t)
}

func TestGetServiceWithNoNodes(t *testing.T) {
	tests.Suite.TestGetServiceWithNoNodes(t)
}

func BenchmarkGetService(b *testing.B) {
	tests.Suite.BenchmarkGetService(b)
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	tests.Suite.BenchmarkGetServiceWithNoNodes(b)
}
