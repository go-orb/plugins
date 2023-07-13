// Package tests contains common tests for go-orb registries.
package tests

import (
	"testing"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

// Suite is a global to use in tests.
var Suite *RegistryTests //nolint:gochecknoglobals

// RegistryTests is the struct we use for tests.
type RegistryTests struct {
	logger log.Logger

	registries []registry.Registry
	nodes      []*registry.Node
	services   []*registry.Service

	updateTime    time.Duration
	serviceOffset int
}

// CreateSuite creates the suite for test usage.
func CreateSuite(logger log.Logger, registries []registry.Registry, updateTime time.Duration, serviceOffset int) {
	r := &RegistryTests{
		logger:        logger,
		registries:    registries,
		updateTime:    updateTime,
		serviceOffset: serviceOffset,
	}

	r.nodes = append(r.nodes, &registry.Node{ID: "node0-http", Address: "10.0.0.10:1234", Scheme: "http"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-grpc", Address: "10.0.0.10:1234", Scheme: "grpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-frpc", Address: "10.0.0.10:1234", Scheme: "frpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-http", Address: "10.0.0.11:1234", Scheme: "http"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-grpc", Address: "10.0.0.11:1234", Scheme: "grpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-frpc", Address: "10.0.0.11:1234", Scheme: "frpc"})

	r.services = append(r.services, &registry.Service{Name: "micro.test.svc.0", Version: "v1", Nodes: []*registry.Node{r.nodes[0]}})
	r.services = append(r.services, &registry.Service{Name: "micro.test.svc.1", Version: "v1", Nodes: []*registry.Node{r.nodes[1]}})
	r.services = append(r.services, &registry.Service{Name: "micro.test.svc.2", Version: "v1", Nodes: []*registry.Node{r.nodes[2]}})

	Suite = r
}

// Setup setups the test suite.
func (r *RegistryTests) Setup() {
	for _, service := range r.services {
		if err := r.registries[0].Register(service); err != nil {
			r.logger.Error("Failed to register service", err, slog.String("service", service.Name))
		}
	}
}

// TearDown runs after all tests.
func (r *RegistryTests) TearDown() {
	for _, service := range r.services {
		if err := r.registries[0].Deregister(service); err != nil {
			r.logger.Error("Failed to deregister service", err, slog.String("service", service.Name))
		}
	}
}

// TestRegister tests registering.
func (r *RegistryTests) TestRegister(t *testing.T) {
	service := registry.Service{Name: "micro.test.svc.3", Version: "v1.0.0", Nodes: []*registry.Node{r.nodes[3]}}
	require.NoError(t, r.registries[0].Register(&service))
	time.Sleep(r.updateTime)

	defer func() {
		err := r.registries[0].Deregister(&service)
		if err != nil {
			panic(err)
		}
	}()

	for _, reg := range r.registries {
		services, err := reg.ListServices()
		require.NoError(t, err)

		require.Equal(t, len(r.services)+1+r.serviceOffset, len(services))

		services, err = reg.GetService(service.Name)
		require.NoError(t, err)
		require.Equal(t, 1, len(services))
		require.Equal(t, service.Version, services[0].Version)
	}
}

// TestDeregister tests deregistering.
func (r *RegistryTests) TestDeregister(t *testing.T) {
	service1 := registry.Service{Name: "micro.test.svc.4", Version: "v1", Nodes: []*registry.Node{r.nodes[4]}}
	service2 := registry.Service{Name: "micro.test.svc.4", Version: "v2", Nodes: []*registry.Node{r.nodes[5]}}

	require.NoError(t, r.registries[0].Register(&service1))
	time.Sleep(r.updateTime)

	services, err := r.registries[0].ListServices()
	require.NoError(t, err)
	require.Equal(t, len(r.services)+1+r.serviceOffset, len(services))

	services, err = r.registries[0].GetService(service1.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))
	require.Equal(t, service1.Version, services[0].Version)

	require.NoError(t, r.registries[0].Register(&service2))
	time.Sleep(r.updateTime)

	services, err = r.registries[0].GetService(service2.Name)
	require.NoError(t, err)
	require.Equal(t, 2, len(services))

	require.NoError(t, r.registries[0].Deregister(&service1))
	time.Sleep(r.updateTime)

	services, err = r.registries[0].GetService(service1.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))

	require.NoError(t, r.registries[0].Deregister(&service2))
	time.Sleep(r.updateTime)

	services, err = r.registries[0].GetService(service1.Name)
	require.NoError(t, err)
	require.Equal(t, 0, len(services))
}

// TestGetServiceAllRegistries tests a service on all registries.
func (r *RegistryTests) TestGetServiceAllRegistries(t *testing.T) {
	for _, reg := range r.registries {
		services, err := reg.GetService(r.services[0].Name)
		require.NoError(t, err)
		require.Equal(t, 1, len(services))
		require.Equal(t, r.services[0].Name, services[0].Name)
		require.Equal(t, len(r.services[0].Nodes), len(services[0].Nodes))
	}
}

// TestGetServiceWithNoNodes tests a non existent service.
func (r *RegistryTests) TestGetServiceWithNoNodes(t *testing.T) {
	services, err := r.registries[0].GetService("missing")
	require.NoError(t, err)
	require.Equal(t, 0, len(services))
}

// BenchmarkGetService benchmarks.
func (r *RegistryTests) BenchmarkGetService(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := r.registries[0].GetService(r.services[0].Name)
		require.NoError(b, err)
		require.Equal(b, 1, len(services))
		require.Equal(b, r.services[0].Name, services[0].Name)
		require.Equal(b, len(r.services[0].Nodes), len(services[0].Nodes))
	}
}

// BenchmarkGetServiceWithNoNodes benchmarks.
func (r *RegistryTests) BenchmarkGetServiceWithNoNodes(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := r.registries[0].GetService("missing")
		require.NoError(b, err)
		require.Equal(b, 0, len(services))
	}
}
