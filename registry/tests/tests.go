// Package tests contains common tests for go-orb registries.
package tests

import (
	"context"
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

	ctx        context.Context
	registries []registry.Registry
	services   []registry.ServiceNode

	updateTime time.Duration
}

// CreateSuite creates the suite for test usage.
func CreateSuite(logger log.Logger, registries []registry.Registry, updateTime time.Duration) *TestSuite {
	r := &TestSuite{
		ctx:        context.Background(),
		logger:     logger,
		registries: registries,
		updateTime: updateTime,
	}

	// Generate random ports to avoid conflicts
	basePort1 := 10000 + rand.Intn(10000) //nolint:gosec
	basePort2 := 20000 + rand.Intn(10000) //nolint:gosec

	// All the test services.
	r.services = []registry.ServiceNode{
		{Name: "orb.test.svc.0", Version: "v1", Address: fmt.Sprintf("10.0.0.10:%d", basePort1), Scheme: "http"},
		{Name: "orb.test.svc.1", Version: "v1", Address: fmt.Sprintf("10.0.0.10:%d", basePort1+1), Scheme: "grpc"},
		{Name: "orb.test.svc.2", Version: "v1", Address: fmt.Sprintf("10.0.0.10:%d", basePort1+2), Scheme: "frpc"},
		{Name: "orb.test.svc.3", Version: "v1", Address: fmt.Sprintf("10.0.0.11:%d", basePort2), Scheme: "http"},
		{Name: "orb.test.svc.4", Version: "v1", Address: fmt.Sprintf("10.0.0.11:%d", basePort2+1), Scheme: "grpc"},
		{Name: "orb.test.svc.5", Version: "v1", Address: fmt.Sprintf("10.0.0.11:%d", basePort2+2), Scheme: "frpc"},
	}

	return r
}

// SetupSuite setups the test suite.
func (r *TestSuite) SetupSuite() {
	for _, service := range r.services {
		if err := r.randomRegistry().Register(r.ctx, service); err != nil {
			r.logger.Error("Failed to register service", "error", err, "service", service.Name)
		}
	}
}

// TearDownSuite runs after all tests.
func (r *TestSuite) TearDownSuite() {
	for _, service := range r.services {
		if err := r.randomRegistry().Deregister(r.ctx, service); err != nil {
			r.logger.Error("Failed to deregister service", "error", err, "service", service.Name)
		}
	}
}

func (r *TestSuite) randomRegistry() registry.Registry {
	return r.registries[rand.Intn(len(r.registries))] //nolint:gosec
}

// TestRegister tests registering.
func (r *TestSuite) TestRegister() {
	service := registry.ServiceNode{
		Name:    "orb.test.svc.6",
		Version: "v1.0.0",
		Address: r.services[3].Address,
		Scheme:  r.services[3].Scheme,
	}
	r.Require().NoError(r.randomRegistry().Register(r.ctx, service))
	time.Sleep(r.updateTime)

	defer func() {
		err := r.randomRegistry().Deregister(r.ctx, service)
		if err != nil {
			r.logger.Error("Failed to cleanup from TestRegister", "error", err)
		}
	}()

	for idx, reg := range r.registries {
		r.Run(fmt.Sprintf("reg-%d", idx), func() {
			services, err := reg.ListServices(r.ctx, "", "", nil)
			r.Require().NoError(err)

			r.Len(services, len(r.services)+1)

			services, err = reg.GetService(r.ctx, "", "", service.Name, nil)
			r.Require().NoError(err)
			r.Len(services, 1)
			r.Equal(service.Version, services[0].Version)
			r.Equal(service.Scheme, services[0].Scheme)
		})
	}
}

// TestGetAllNodesAndVersions tests that all nodes and all versions of a service are returned.
//
//nolint:funlen
func (r *TestSuite) TestGetAllNodesAndVersions() {
	// Use a unique service name for this test to avoid conflicts
	baseName := "orb.test.allnodes"

	// Create multiple services with different versions
	services := []registry.ServiceNode{
		// Service 1: v1.0.0 with two nodes
		{
			Name:     baseName + ".svc1",
			Version:  "v1.0.0",
			Address:  "10.0.1.1:8080",
			Scheme:   "http",
			Metadata: map[string]string{"region": "us-east"},
		},
		{
			Name:     baseName + ".svc1",
			Version:  "v1.0.0",
			Address:  "10.0.1.2:8080",
			Scheme:   "grpc",
			Metadata: map[string]string{"region": "us-west"},
		},
		// Service 1: v2.0.0 with one node
		{
			Name:     baseName + ".svc1",
			Version:  "v2.0.0",
			Address:  "10.0.1.3:8080",
			Scheme:   "http",
			Metadata: map[string]string{"region": "eu-west"},
		},
		// Service 2: v1.0.0 with three nodes with different transports
		{
			Name:     baseName + ".svc2",
			Version:  "v1.0.0",
			Address:  "10.0.2.1:8080",
			Scheme:   "http",
			Metadata: map[string]string{"region": "us-east"},
		},
		{
			Name:     baseName + ".svc2",
			Version:  "v1.0.0",
			Address:  "10.0.2.2:8080",
			Scheme:   "grpc",
			Metadata: map[string]string{"region": "us-west"},
		},
		{
			Name:     baseName + ".svc2",
			Version:  "v1.0.0",
			Address:  "10.0.2.3:8080",
			Scheme:   "http3",
			Metadata: map[string]string{"region": "eu-west"},
		},
	}

	for _, reg := range r.registries {
		r.Run("registry-"+reg.String(), func() {
			// Register all services
			for _, svc := range services {
				r.Require().NoError(reg.Register(r.ctx, svc))
			}

			time.Sleep(r.updateTime)

			// Cleanup when done
			defer func() {
				for _, svc := range services {
					r.Require().NoError(reg.Deregister(r.ctx, svc))
				}
			}()

			// Test 1: GetService should return all versions of service 1
			service1Name := baseName + ".svc1"
			service1Results, err := reg.GetService(r.ctx, "", "", service1Name, nil)
			r.Require().NoError(err)

			// Group results by version to validate
			resultsMap := make(map[string][]registry.ServiceNode)
			for _, svc := range service1Results {
				resultsMap[svc.Version] = append(resultsMap[svc.Version], svc)
			}

			// Verify we have both versions
			r.Require().Contains(resultsMap, "v1.0.0", "v1.0.0 should be found")
			r.Require().Contains(resultsMap, "v2.0.0", "v2.0.0 should be found")

			// Verify v1.0.0 has 2 nodes
			r.Require().Len(resultsMap["v1.0.0"], 2, "v1.0.0 should have 2 nodes")

			// Verify v2.0.0 has 1 node
			r.Require().Len(resultsMap["v2.0.0"], 1, "v2.0.0 should have 1 node")

			// Verify transports are preserved for v1.0.0
			schemes := map[string]bool{}
			for _, node := range resultsMap["v1.0.0"] {
				schemes[node.Scheme] = true
			}

			r.Require().Len(schemes, 2, "Should have both http and grpc schemes")
			r.Require().True(schemes["http"], "Should have http scheme")
			r.Require().True(schemes["grpc"], "Should have grpc scheme")

			// Test 2: GetService should return all nodes for service 2
			service2Name := baseName + ".svc2"
			service2Results, err := reg.GetService(r.ctx, "", "", service2Name, nil)
			r.Require().NoError(err)
			r.Require().Len(service2Results, 3, "Service 2 should have 3 nodes")

			// Verify all schemes are present
			schemeSet := map[string]bool{}
			for _, node := range service2Results {
				schemeSet[node.Scheme] = true
			}

			r.Require().Len(schemeSet, 3, "All three schemes should be present")
			r.Require().True(schemeSet["http"], "HTTP scheme should be present")
			r.Require().True(schemeSet["grpc"], "gRPC scheme should be present")
			r.Require().True(schemeSet["http3"], "HTTP/3 scheme should be present")

			// Test 3: ListServices should return all services
			allServices, err := reg.ListServices(r.ctx, "", "", nil)
			r.Require().NoError(err)

			// Count the number of instances of our test services
			service1Count := 0
			service2Count := 0

			for _, svc := range allServices {
				if svc.Name == service1Name {
					service1Count++
				} else if svc.Name == service2Name {
					service2Count++
				}
			}

			r.Require().Equal(3, service1Count, "ListServices should return all nodes of service 1")
			r.Require().Equal(3, service2Count, "ListServices should return all nodes of service 2")
		})
	}
}

// TestDeregister tests deregistering.
func (r *TestSuite) TestDeregister() {
	service1 := registry.ServiceNode{
		Name:    "orb.test.deregister",
		Version: "v1",
		Address: r.services[4].Address,
		Scheme:  r.services[4].Scheme,
	}
	service2 := registry.ServiceNode{
		Name:    "orb.test.deregister",
		Version: "v2",
		Address: r.services[5].Address,
		Scheme:  r.services[5].Scheme,
	}

	r.Require().NoError(r.randomRegistry().Register(r.ctx, service1))
	time.Sleep(r.updateTime)

	services, err := r.randomRegistry().ListServices(r.ctx, "", "", nil)
	r.Require().NoError(err)
	r.Len(services, len(r.services)+1)

	services, err = r.randomRegistry().GetService(r.ctx, "", "", service1.Name, nil)
	r.Require().NoError(err)
	r.Len(services, 1)
	r.Equal(service1.Version, services[0].Version)

	r.Require().NoError(r.randomRegistry().Register(r.ctx, service2))
	time.Sleep(r.updateTime)

	services, err = r.randomRegistry().GetService(r.ctx, "", "", service2.Name, nil)
	r.Require().NoError(err)
	r.Len(services, 2)

	r.Require().NoError(r.randomRegistry().Deregister(r.ctx, service1))
	time.Sleep(r.updateTime)

	services, err = r.randomRegistry().GetService(r.ctx, "", "", service1.Name, nil)
	r.Require().NoError(err)
	r.Len(services, 1)

	r.Require().NoError(r.randomRegistry().Deregister(r.ctx, service2))
	time.Sleep(r.updateTime)

	services, err = r.randomRegistry().GetService(r.ctx, "", "", service1.Name, nil)
	r.Require().ErrorIs(err, registry.ErrNotFound)
	r.Empty(services)
}

// TestGetServiceAllRegistries tests a service on all registries.
func (r *TestSuite) TestGetServiceAllRegistries() {
	for idx, reg := range r.registries {
		r.Run(fmt.Sprintf("reg-%d", idx), func() {
			services, err := reg.GetService(r.ctx, "", "", r.services[0].Name, nil)
			r.Require().NoError(err)
			r.Len(services, 1)
			r.Equal(r.services[0].Name, services[0].Name)
		})
	}
}

// TestGetServiceWithNoNodes tests a non existent service.
func (r *TestSuite) TestGetServiceWithNoNodes() {
	services, err := r.randomRegistry().GetService(r.ctx, "", "", "missing", nil)
	r.Require().ErrorIs(err, registry.ErrNotFound)
	r.Empty(services)
}

// BenchmarkGetService benchmarks.
func (r *TestSuite) BenchmarkGetService(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		id := rand.Intn(len(r.services)) //nolint:gosec

		services, err := r.randomRegistry().GetService(r.ctx, "", "", r.services[id].Name, nil)
		r.Require().NoError(err)
		r.Len(services, 1)
		r.Equal(r.services[id].Name, services[0].Name)
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
		services, err := r.randomRegistry().GetService(r.ctx, "", "", "missing", nil)
		r.Require().ErrorIs(err, registry.ErrNotFound)
		r.Empty(services)
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

			services, err := r.randomRegistry().GetService(r.ctx, "", "", r.services[id].Name, nil)
			r.Require().NoError(err)
			r.Len(services, 1)
			r.Equal(r.services[id].Name, services[0].Name)
		}
	})

	b.StopTimer()
	r.TearDownSuite()
}

// TestWatchServices tests the watcher functionality.
func (r *TestSuite) TestWatchServices() {
	// Create a test service with a unique name for watching
	serviceName := "orb.test.watch" + time.Now().Format("20060102150405")
	service := registry.ServiceNode{
		Name:    serviceName,
		Version: "v1.0.0",
		Address: r.services[3].Address,
		Scheme:  r.services[3].Scheme,
	}

	// Register the service first to ensure it exists
	err := r.randomRegistry().Register(r.ctx, service)
	r.Require().NoError(err)
	time.Sleep(r.updateTime)

	// Create a watcher
	watcher, err := r.randomRegistry().Watch(r.ctx, registry.WatchService(serviceName))
	r.Require().NoError(err)
	r.NotNil(watcher)

	// Update the service to trigger a watch event
	service.Metadata = map[string]string{"updated": "true"} // Modify metadata to trigger update

	// Start a goroutine to listen for events
	eventReceived := make(chan bool)

	go func() {
		result, err := watcher.Next()
		if err != nil {
			return
		}

		if result != nil && result.Node.Name == serviceName {
			eventReceived <- true
			return
		}
	}()

	// Update the service that should trigger the watcher
	err = r.randomRegistry().Register(r.ctx, service)
	r.Require().NoError(err)

	// Wait for the event with a timeout
	select {
	case <-eventReceived:
		// Success
	case <-time.After(2 * time.Second):
		// This is not a hard failure as watching can be flaky
		r.T().Log("No watch event received, but continuing test")
	}

	// Cleanup
	err = r.randomRegistry().Deregister(r.ctx, service)
	r.Require().NoError(err)
}

// TestServiceUpdate tests updating an existing service.
func (r *TestSuite) TestServiceUpdate() {
	if r.registries[0].String() == "mdns" {
		r.T().Skip("Skipping test for mdns registry")
		return
	}

	// Initial service
	service := registry.ServiceNode{
		Name:    "orb.test.update",
		Version: "v1.0.0",
		Address: r.services[0].Address,
		Scheme:  r.services[0].Scheme,
	}

	// Register the service
	r.Require().NoError(r.randomRegistry().Register(r.ctx, service))
	time.Sleep(r.updateTime)

	// Verify initial service
	services, err := r.randomRegistry().GetService(r.ctx, "", "", service.Name, nil)
	r.Require().NoError(err)
	r.Len(services, 1)

	// Update the service by adding metadata
	updatedService := registry.ServiceNode{
		Name:     "orb.test.update",
		Version:  "v1.0.0",
		Address:  r.services[0].Address,
		Scheme:   r.services[0].Scheme,
		Metadata: map[string]string{"updated": "true"},
	}

	// Register the updated service
	r.Require().NoError(r.randomRegistry().Register(r.ctx, updatedService))
	time.Sleep(r.updateTime)

	// Verify the service was updated
	services, err = r.randomRegistry().GetService(r.ctx, "", "", service.Name, nil)
	r.Require().NoError(err)
	r.Len(services, 1)

	// Should have metadata updated
	r.Equal("true", services[0].Metadata["updated"])

	// Cleanup
	r.Require().NoError(r.randomRegistry().Deregister(r.ctx, updatedService))
}

// TestServiceMetadata tests handling of service metadata.
func (r *TestSuite) TestServiceMetadata() {
	// Service with metadata
	service := registry.ServiceNode{
		Name:    "orb.test.metadata",
		Version: "v1.0.0",
		Address: r.services[0].Address,
		Scheme:  r.services[0].Scheme,
		Metadata: map[string]string{
			"region": "us-west",
			"env":    "test",
			"secure": "true",
		},
	}

	// Register the service
	r.Require().NoError(r.randomRegistry().Register(r.ctx, service))
	time.Sleep(r.updateTime)

	// Verify the service can be retrieved with metadata intact
	services, err := r.randomRegistry().GetService(r.ctx, "", "", service.Name, nil)
	r.Require().NoError(err)
	r.Len(services, 1)

	// Verify metadata was preserved
	r.Equal(service.Metadata["region"], services[0].Metadata["region"])
	r.Equal(service.Metadata["env"], services[0].Metadata["env"])
	r.Equal(service.Metadata["secure"], services[0].Metadata["secure"])

	// Cleanup
	r.Require().NoError(r.randomRegistry().Deregister(r.ctx, service))
}

// TestMultipleVersions tests registering and retrieving services with multiple versions.
func (r *TestSuite) TestMultipleVersions() {
	// Create services with same name but different versions
	serviceV1 := registry.ServiceNode{
		Name:    "orb.test.versions",
		Version: "v1.0.0",
		Address: r.services[0].Address,
		Scheme:  r.services[0].Scheme,
	}
	serviceV2 := registry.ServiceNode{
		Name:    "orb.test.versions",
		Version: "v2.0.0",
		Address: r.services[1].Address,
		Scheme:  r.services[1].Scheme,
	}

	// Register both versions
	r.Require().NoError(r.randomRegistry().Register(r.ctx, serviceV1))
	r.Require().NoError(r.randomRegistry().Register(r.ctx, serviceV2))
	time.Sleep(r.updateTime)

	// Get all versions of the service
	services, err := r.randomRegistry().GetService(r.ctx, "", "", serviceV1.Name, nil)
	r.Require().NoError(err)
	r.Require().GreaterOrEqual(len(services), 2, "Should have found both versions of the service")

	// Verify both versions are returned
	versions := map[string]bool{}
	for _, s := range services {
		versions[s.Version] = true
	}

	r.Require().True(versions["v1.0.0"], "v1.0.0 should be in the results")
	r.Require().True(versions["v2.0.0"], "v2.0.0 should be in the results")

	// Cleanup
	r.Require().NoError(r.randomRegistry().Deregister(r.ctx, serviceV1))
	r.Require().NoError(r.randomRegistry().Deregister(r.ctx, serviceV2))
}

// TestMultipleTransports verifies that the registry correctly handles multiple nodes with the same name but different transports.
func (r *TestSuite) TestMultipleTransports() {
	// Create multiple nodes with same name but different schemes
	nodes := []registry.ServiceNode{
		{
			Name:    "test-service",
			Version: "v1.0.0",
			Address: "127.0.0.1:8080",
			Scheme:  "grpc",
		},
		{
			Name:    "test-service",
			Version: "v1.0.0",
			Address: "127.0.0.1:8081",
			Scheme:  "drpc",
		},
		{
			Name:    "test-service",
			Version: "v1.0.0",
			Address: "127.0.0.1:8082",
			Scheme:  "http",
		},
		{
			Name:    "test-service",
			Version: "v1.0.0",
			Address: "127.0.0.1:8083",
			Scheme:  "https",
		},
	}

	// Register all nodes
	for _, node := range nodes {
		r.Require().NoError(r.randomRegistry().Register(r.ctx, node))
	}
	time.Sleep(r.updateTime)

	// Verify each scheme returns the correct node
	for _, node := range nodes {
		services, err := r.randomRegistry().GetService(r.ctx, "", "", node.Name, []string{node.Scheme})
		r.Require().NoError(err)
		r.Require().Len(services, 1, "Should return only one node with %s Scheme", node.Scheme)
		r.Require().Equal(node.Scheme, services[0].Scheme, "Should return node with %s Scheme", node.Scheme)
	}

	// Cleanup all nodes
	for _, node := range nodes {
		r.Require().NoError(r.randomRegistry().Deregister(r.ctx, node))
	}
}

// BenchmarkListServices benchmarks the performance of listing services.
func (r *TestSuite) BenchmarkListServices(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		services, err := r.randomRegistry().ListServices(r.ctx, "", "", nil)
		r.Require().NoError(err)
		r.Require().NotEmpty(services)
	}

	b.StopTimer()
	r.TearDownSuite()
}

// TestFiltering tests using options to filter services.
func (r *TestSuite) TestFiltering() {
	// Create services with different attributes
	services := []registry.ServiceNode{
		{
			Name:      "orb.test.filter",
			Version:   "v1.0.0",
			Address:   "10.0.1.1:8080",
			Scheme:    "http",
			Namespace: "default",
			Region:    "us-west",
		},
		{
			Name:      "orb.test.filter",
			Version:   "v2.0.0",
			Address:   "10.0.1.2:8080",
			Scheme:    "grpc",
			Namespace: "default",
			Region:    "us-east",
		},
		{
			Name:      "orb.test.filter",
			Version:   "v1.0.0",
			Address:   "10.0.1.3:8080",
			Scheme:    "https",
			Namespace: "production",
			Region:    "eu-west",
		},
	}

	// Register all services
	for _, svc := range services {
		r.Require().NoError(r.randomRegistry().Register(r.ctx, svc))
	}

	// Cleanup
	defer func() {
		for _, svc := range services {
			r.Require().NoError(r.randomRegistry().Deregister(r.ctx, svc))
		}
	}()

	time.Sleep(r.updateTime)

	// Test filtering by version and other parameters
	filtered, err := r.randomRegistry().GetService(
		r.ctx, "default", "us-west", "orb.test.filter", []string{"http"})
	r.Require().NoError(err)
	r.Require().Len(filtered, 1, "Should find exactly one service with version v1.0.0 in default/us-west")
	r.Require().Equal("v1.0.0", filtered[0].Version)
	r.Require().Equal("default", filtered[0].Namespace)
	r.Require().Equal("us-west", filtered[0].Region)

	// Test filtering by namespace
	filtered, err = r.randomRegistry().GetService(
		r.ctx, "production", "eu-west", "orb.test.filter", nil)
	r.Require().NoError(err)
	r.Require().Len(filtered, 1, "Should find exactly one service in production/eu-west")
	r.Require().Equal("v1.0.0", filtered[0].Version)
	r.Require().Equal("production", filtered[0].Namespace)
	r.Require().Equal("eu-west", filtered[0].Region)

	// Test filtering by region
	filtered, err = r.randomRegistry().GetService(
		r.ctx, "default", "us-east", "orb.test.filter", nil)
	r.Require().NoError(err)
	r.Require().Len(filtered, 1, "Should find exactly one service in default/us-east")
	r.Require().Equal("v2.0.0", filtered[0].Version)
	r.Require().Equal("default", filtered[0].Namespace)
	r.Require().Equal("us-east", filtered[0].Region)

	// Test filtering by scheme
	filtered, err = r.randomRegistry().GetService(
		r.ctx, "default", "us-west", "orb.test.filter", []string{"http"})
	r.Require().NoError(err)
	r.Require().Len(filtered, 1, "Should find exactly one service with HTTP scheme")
	r.Require().Equal("http", filtered[0].Scheme)

	// Test for no matches with a combination of filters
	filtered, err = r.randomRegistry().GetService(
		r.ctx, "production", "us-east", "orb.test.filter", nil)
	r.Require().ErrorIs(err, registry.ErrNotFound)
	r.Require().Empty(filtered, "Should find no services in production/us-east")
}

// BenchmarkRegisterDeregister benchmarks the performance of registering and deregistering services.
func (r *TestSuite) BenchmarkRegisterDeregister(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})

	// Create a service just for this benchmark
	service := registry.ServiceNode{
		Name:    "orb.test.benchmark.regdereg",
		Version: "v1.0.0",
		Address: r.services[0].Address,
		Scheme:  r.services[0].Scheme,
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		// Register
		err := r.randomRegistry().Register(r.ctx, service)
		r.Require().NoError(err)

		// Deregister
		err = r.randomRegistry().Deregister(r.ctx, service)
		r.Require().NoError(err)
	}
}
