// Package tests contains common tests for go-orb registries.
package tests

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/stretchr/testify/suite"
)

// TestSuite is the struct we use for tests.
type TestSuite struct {
	suite.Suite

	logger log.Logger

	registries []registry.Registry
	nodes      []*registry.Node
	services   []*registry.Service

	updateTime    time.Duration
	serviceOffset int
}

// CreateSuite creates the suite for test usage.
func CreateSuite(logger log.Logger, registries []registry.Registry, updateTime time.Duration, serviceOffset int) *TestSuite {
	r := &TestSuite{
		logger:        logger,
		registries:    registries,
		updateTime:    updateTime,
		serviceOffset: serviceOffset,
	}

	r.nodes = append(r.nodes, &registry.Node{ID: "node0-http", Address: "10.0.0.10:1234", Transport: "http"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-grpc", Address: "10.0.0.10:1234", Transport: "grpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-frpc", Address: "10.0.0.10:1234", Transport: "frpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-http", Address: "10.0.0.11:1234", Transport: "http"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-grpc", Address: "10.0.0.11:1234", Transport: "grpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-frpc", Address: "10.0.0.11:1234", Transport: "frpc"})

	r.services = append(r.services, &registry.Service{Name: "orb.test.svc.0", Version: "v1", Nodes: []*registry.Node{r.nodes[0]}})
	r.services = append(r.services, &registry.Service{Name: "orb.test.svc.1", Version: "v1", Nodes: []*registry.Node{r.nodes[1]}})
	r.services = append(r.services, &registry.Service{Name: "orb.test.svc.2", Version: "v1", Nodes: []*registry.Node{r.nodes[2]}})

	return r
}

// SetupSuite setups the test suite.
func (r *TestSuite) SetupSuite() {
	for _, service := range r.services {
		if err := r.registries[0].Register(service); err != nil {
			r.logger.Error("Failed to register service", "error", err, "service", service.Name)
		}
	}
}

// TearDownSuite runs after all tests.
func (r *TestSuite) TearDownSuite() {
	for _, service := range r.services {
		if err := r.registries[0].Deregister(service); err != nil {
			r.logger.Error("Failed to deregister service", "error", err, "service", service.Name)
		}
	}
}

// TestRegister tests registering.
func (r *TestSuite) TestRegister() {
	service := registry.Service{Name: "orb.test.svc.3", Version: "v1.0.0", Nodes: []*registry.Node{r.nodes[3]}}
	r.Require().NoError(r.registries[0].Register(&service))
	time.Sleep(r.updateTime)

	defer func() {
		err := r.registries[0].Deregister(&service)
		if err != nil {
			panic(err)
		}
	}()

	for idx, reg := range r.registries {
		r.Run(fmt.Sprintf("reg-%d", idx), func() {
			services, err := reg.ListServices()
			r.Require().NoError(err)

			r.Require().Len(services, len(r.services)+1+r.serviceOffset)

			services, err = reg.GetService(service.Name)
			r.Require().NoError(err)
			r.Len(services, 1)
			r.Equal(service.Version, services[0].Version)
			r.Equal(service.Nodes[0].Transport, services[0].Nodes[0].Transport)
		})
	}
}

// TestDeregister tests deregistering.
func (r *TestSuite) TestDeregister() {
	service1 := registry.Service{Name: "orb.test.svc.4", Version: "v1", Nodes: []*registry.Node{r.nodes[4]}}
	service2 := registry.Service{Name: "orb.test.svc.4", Version: "v2", Nodes: []*registry.Node{r.nodes[5]}}

	r.Require().NoError(r.registries[0].Register(&service1))
	time.Sleep(r.updateTime)

	services, err := r.registries[0].ListServices()
	r.Require().NoError(err)
	r.Require().Len(services, len(r.services)+1+r.serviceOffset)

	services, err = r.registries[0].GetService(service1.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)
	r.Require().Equal(service1.Version, services[0].Version)

	r.Require().NoError(r.registries[0].Register(&service2))
	time.Sleep(r.updateTime)

	services, err = r.registries[0].GetService(service2.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 2)

	r.Require().NoError(r.registries[0].Deregister(&service1))
	time.Sleep(r.updateTime)

	services, err = r.registries[0].GetService(service1.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)

	r.Require().NoError(r.registries[0].Deregister(&service2))
	time.Sleep(r.updateTime)

	services, err = r.registries[0].GetService(service1.Name)
	r.Require().ErrorIs(registry.ErrNotFound, err)
	r.Require().Empty(services)
}

// TestGetServiceAllRegistries tests a service on all registries.
func (r *TestSuite) TestGetServiceAllRegistries() {
	for idx, reg := range r.registries {
		r.Run(fmt.Sprintf("reg-%d", idx), func() {
			services, err := reg.GetService(r.services[0].Name)
			r.Require().NoError(err)
			r.Require().Len(services, 1)
			r.Require().Equal(r.services[0].Name, services[0].Name)
			r.Require().Equal(len(r.services[0].Nodes), len(services[0].Nodes))
		})
	}
}

// TestGetServiceWithNoNodes tests a non existent service.
func (r *TestSuite) TestGetServiceWithNoNodes() {
	services, err := r.registries[rand.Intn(len(r.registries))].GetService("missing") //nolint:gosec
	r.Require().ErrorIs(registry.ErrNotFound, err)
	r.Require().Empty(services)
}

// BenchmarkGetService benchmarks.
func (r *TestSuite) BenchmarkGetService(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		id := rand.Intn(len(r.services)) //nolint:gosec

		services, err := r.registries[rand.Intn(len(r.registries))].GetService(r.services[id].Name) //nolint:gosec
		r.Require().NoError(err)
		r.Require().Len(services, 1)
		r.Require().Equal(r.services[id].Name, services[0].Name)
		r.Require().Equal(len(r.services[id].Nodes), len(services[0].Nodes))
	}

	b.StopTimer()
	r.TearDownSuite()
}

// BenchmarkGetServiceWithNoNodes is a 404 benchmark.
func (r *TestSuite) BenchmarkGetServiceWithNoNodes(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		services, err := r.registries[0].GetService("missing")
		r.Require().ErrorIs(registry.ErrNotFound, err)
		r.Require().Empty(services)
	}

	b.StopTimer()
	r.TearDownSuite()
}

// BenchmarkParallelGetService benchmarks.
func (r *TestSuite) BenchmarkParallelGetService(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := rand.Intn(len(r.services)) //nolint:gosec

			services, err := r.registries[rand.Intn(len(r.registries))].GetService(r.services[id].Name) //nolint:gosec
			r.Require().NoError(err)
			r.Require().Len(services, 1)
			r.Require().Equal(r.services[id].Name, services[0].Name)
			r.Require().Equal(len(r.services[id].Nodes), len(services[0].Nodes))
		}
	})

	b.StopTimer()
	r.TearDownSuite()

}
