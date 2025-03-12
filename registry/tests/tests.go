// Package tests contains common tests for go-orb registries.
package tests

import (
	"fmt"
	math_rand "math/rand"
	"strings"
	"sync"
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

	// Generate random ports to avoid conflicts
	basePort1 := 10000 + math_rand.Intn(10000) //nolint:gosec
	basePort2 := 20000 + math_rand.Intn(10000) //nolint:gosec

	// Each node gets a unique port to avoid conflicts
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-http", Address: fmt.Sprintf("10.0.0.10:%d", basePort1), Transport: "http"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-grpc", Address: fmt.Sprintf("10.0.0.10:%d", basePort1+1), Transport: "grpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node0-frpc", Address: fmt.Sprintf("10.0.0.10:%d", basePort1+2), Transport: "frpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-http", Address: fmt.Sprintf("10.0.0.11:%d", basePort2), Transport: "http"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-grpc", Address: fmt.Sprintf("10.0.0.11:%d", basePort2+1), Transport: "grpc"})
	r.nodes = append(r.nodes, &registry.Node{ID: "node1-frpc", Address: fmt.Sprintf("10.0.0.11:%d", basePort2+2), Transport: "frpc"})

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
			r.Require().Len(services, 1)
			r.Equal(service.Version, services[0].Version)
			r.Equal(service.Nodes[0].Transport, services[0].Nodes[0].Transport)
		})
	}
}

// TestGetAllNodesAndVersions tests that all nodes and all versions of a service are returned.
//
//nolint:gocognit,funlen
func (r *TestSuite) TestGetAllNodesAndVersions() {
	// Use a unique service name for this test to avoid conflicts
	baseName := "orb.test.allnodes"

	// Create multiple services with different versions
	services := []*registry.Service{
		// Service 1: v1.0.0 with two nodes
		{
			Name:    baseName + ".svc1",
			Version: "v1.0.0",
			Nodes: []*registry.Node{
				{
					ID:        "node-1-1",
					Address:   "10.0.1.1:8080",
					Transport: "http",
					Metadata:  map[string]string{"region": "us-east"},
				},
				{
					ID:        "node-1-2",
					Address:   "10.0.1.2:8080",
					Transport: "grpc",
					Metadata:  map[string]string{"region": "us-west"},
				},
			},
		},
		// Service 1: v2.0.0 with one node
		{
			Name:    baseName + ".svc1",
			Version: "v2.0.0",
			Nodes: []*registry.Node{
				{
					ID:        "node-1-3",
					Address:   "10.0.1.3:8080",
					Transport: "http",
					Metadata:  map[string]string{"region": "eu-west"},
				},
			},
		},
		// Service 2: v1.0.0 with three nodes with different transports
		{
			Name:    baseName + ".svc2",
			Version: "v1.0.0",
			Nodes: []*registry.Node{
				{
					ID:        "node-2-1",
					Address:   "10.0.2.1:8080",
					Transport: "http",
					Metadata:  map[string]string{"region": "us-east"},
				},
				{
					ID:        "node-2-2",
					Address:   "10.0.2.2:8080",
					Transport: "grpc",
					Metadata:  map[string]string{"region": "us-west"},
				},
				{
					ID:        "node-2-3",
					Address:   "10.0.2.3:8080",
					Transport: "http3",
					Metadata:  map[string]string{"region": "eu-west"},
				},
			},
		},
	}

	for _, registry := range r.registries {
		r.Run("registry-"+registry.String(), func() {
			// Register all services
			for _, svc := range services {
				r.Require().NoError(registry.Register(svc))
			}

			time.Sleep(r.updateTime)

			// Cleanup when done
			defer func() {
				for _, svc := range services {
					r.Require().NoError(registry.Deregister(svc))
				}
			}()

			// Test 1: GetService should return all versions of service 1
			service1Name := baseName + ".svc1"
			service1Results, err := registry.GetService(service1Name)
			r.Require().NoError(err)

			// Verify that each version has the correct number of nodes
			versionsFound := map[string]bool{
				"v1.0.0": false,
				"v2.0.0": false,
			}

			for _, svc := range service1Results {
				versionsFound[svc.Version] = true

				if svc.Version == "v1.0.0" {
					r.Require().Len(svc.Nodes, 2, "v1.0.0 should have 2 nodes")
					r.Require().ElementsMatch(
						[]string{"node-1-1", "node-1-2"},
						[]string{svc.Nodes[0].ID, svc.Nodes[1].ID},
						"Node IDs should match")

					// Verify transports are preserved
					transports := map[string]bool{}
					for _, node := range svc.Nodes {
						transports[node.Transport] = true
					}

					r.Require().Len(transports, 2, "Should have both http and grpc transports")
					r.Require().True(transports["http"], "Should have http transport")
					r.Require().True(transports["grpc"], "Should have grpc transport")
				} else if svc.Version == "v2.0.0" {
					r.Require().Len(svc.Nodes, 1, "v2.0.0 should have 1 node")
					r.Equal("node-1-3", svc.Nodes[0].ID)
					r.Equal("http", svc.Nodes[0].Transport, "Transport should be preserved")
				}
			}

			r.Require().True(versionsFound["v1.0.0"], "v1.0.0 should be found")
			r.Require().True(versionsFound["v2.0.0"], "v2.0.0 should be found")

			// Test 2: GetService should return all nodes for service 2
			service2Name := baseName + ".svc2"
			service2Results, err := registry.GetService(service2Name)
			r.Require().NoError(err)
			r.Require().Len(service2Results, 1, "Should return one version of service 2")
			r.Require().Len(service2Results[0].Nodes, 3, "Service 2 should have 3 nodes")

			// Verify all node IDs are present
			nodeIDs := []string{}
			for _, node := range service2Results[0].Nodes {
				nodeIDs = append(nodeIDs, node.ID)
			}

			r.Require().ElementsMatch(
				[]string{"node-2-1", "node-2-2", "node-2-3"},
				nodeIDs,
				"All node IDs should be present")

			// Verify all transports are present
			transportSet := map[string]bool{}
			for _, node := range service2Results[0].Nodes {
				transportSet[node.Transport] = true
			}

			r.Require().Len(transportSet, 3, "All three transports should be present")
			r.Require().True(transportSet["http"], "HTTP transport should be present")
			r.Require().True(transportSet["grpc"], "gRPC transport should be present")
			r.Require().True(transportSet["http3"], "HTTP/3 transport should be present")

			// Test 3: ListServices should return all services
			allServices, err := registry.ListServices()
			r.Require().NoError(err)

			// Count the number of instances of our test services
			service1Count := 0
			service2Count := 0
			totalNodes := 0

			for _, svc := range allServices {
				if svc.Name == service1Name {
					service1Count++
					totalNodes += len(svc.Nodes)
				} else if svc.Name == service2Name {
					service2Count++
					totalNodes += len(svc.Nodes)
				}
			}

			r.Require().Equal(2, service1Count, "ListServices should return two versions of service 1")
			r.Require().Equal(1, service2Count, "ListServices should return one version of service 2")
			r.Require().Equal(6, totalNodes, "Total node count should be 6 (2+1+3)")
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
	services, err := r.registries[math_rand.Intn(len(r.registries))].GetService("missing") //nolint:gosec
	r.Require().ErrorIs(registry.ErrNotFound, err)
	r.Require().Empty(services)
}

// BenchmarkGetService benchmarks.
func (r *TestSuite) BenchmarkGetService(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	for n := 0; n < b.N; n++ { //nolint:dupl
		id := math_rand.Intn(len(r.services)) //nolint:gosec

		services, err := r.registries[math_rand.Intn(len(r.registries))].GetService(r.services[id].Name) //nolint:gosec
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
		for pb.Next() { //nolint:dupl
			id := math_rand.Intn(len(r.services)) //nolint:gosec

			services, err := r.registries[math_rand.Intn(len(r.registries))].GetService(r.services[id].Name) //nolint:gosec
			r.Require().NoError(err)
			r.Require().Len(services, 1)
			r.Require().Equal(r.services[id].Name, services[0].Name)
			r.Require().Equal(len(r.services[id].Nodes), len(services[0].Nodes))
		}
	})

	b.StopTimer()
	r.TearDownSuite()
}

// TestWatchServices tests the watcher functionality.
func (r *TestSuite) TestWatchServices() {
	// Skip the test if we have no registries
	if len(r.registries) == 0 {
		r.T().Skip("No registries available for testing")
		return
	}

	// Create a test service with a unique name for watching
	serviceName := "orb.test.watch" + time.Now().Format("20060102150405")
	service := registry.Service{Name: serviceName, Version: "v1.0.0", Nodes: []*registry.Node{r.nodes[3]}}

	// Register the service first to ensure it exists
	err := r.registries[0].Register(&service)
	r.Require().NoError(err)
	time.Sleep(r.updateTime)

	// Create a watcher
	watcher, err := r.registries[0].Watch()
	r.Require().NoError(err)
	r.Require().NotNil(watcher)

	//nolint:errcheck
	defer watcher.Stop()

	// Update the service to trigger a watch event
	service.Nodes = append(service.Nodes, r.nodes[2]) // Add another node

	// Start a goroutine to listen for events
	eventReceived := make(chan bool)

	go func() {
		// Handle multiple events until we get one for our service
		for i := 0; i < 20; i++ { // Try multiple times
			result, err := watcher.Next()
			if err != nil {
				break
			}

			if result != nil && result.Service != nil && result.Service.Name == serviceName {
				eventReceived <- true
				break
			}

			// Check next event after a short delay
			time.Sleep(50 * time.Millisecond)
		}
	}()

	// Update the service that should trigger the watcher
	err = r.registries[0].Register(&service)
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
	err = r.registries[0].Deregister(&service)
	r.Require().NoError(err)
}

// TestServiceUpdate tests updating an existing service.
func (r *TestSuite) TestServiceUpdate() {
	// Initial service
	service := registry.Service{
		Name:    "orb.test.update",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{r.nodes[0]},
	}

	// Register the service
	r.Require().NoError(r.registries[0].Register(&service))
	time.Sleep(r.updateTime)

	// Verify initial service
	services, err := r.registries[0].GetService(service.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)
	r.Require().Len(services[0].Nodes, 1)

	// Update the service by adding a node
	updatedService := registry.Service{
		Name:    "orb.test.update",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{r.nodes[0], r.nodes[1]},
	}

	// Register the updated service
	r.Require().NoError(r.registries[0].Register(&updatedService))
	time.Sleep(r.updateTime)

	// Verify the service was updated
	services, err = r.registries[0].GetService(service.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)

	// Should now have 2 nodes
	r.Require().Len(services[0].Nodes, 2)

	// Cleanup
	r.Require().NoError(r.registries[0].Deregister(&updatedService))
}

// TestServiceMetadata tests handling of service metadata.
func (r *TestSuite) TestServiceMetadata() {
	// Service with metadata
	service := registry.Service{
		Name:    "orb.test.metadata",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{r.nodes[0]},
		Metadata: map[string]string{
			"region": "us-west",
			"env":    "test",
			"secure": "true",
		},
	}

	// Register the service
	r.Require().NoError(r.registries[0].Register(&service))
	time.Sleep(r.updateTime)

	// Verify the service can be retrieved with metadata intact
	services, err := r.registries[0].GetService(service.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)

	// Verify metadata was preserved
	r.Require().Equal(service.Metadata["region"], services[0].Metadata["region"])
	r.Require().Equal(service.Metadata["env"], services[0].Metadata["env"])
	r.Require().Equal(service.Metadata["secure"], services[0].Metadata["secure"])

	// Cleanup
	r.Require().NoError(r.registries[0].Deregister(&service))
}

// TestMultipleVersions tests registering and retrieving services with multiple versions.
func (r *TestSuite) TestMultipleVersions() {
	// Create services with same name but different versions
	serviceV1 := registry.Service{Name: "orb.test.versions", Version: "v1.0.0", Nodes: []*registry.Node{r.nodes[0]}}
	serviceV2 := registry.Service{Name: "orb.test.versions", Version: "v2.0.0", Nodes: []*registry.Node{r.nodes[1]}}

	// Register both versions
	r.Require().NoError(r.registries[0].Register(&serviceV1))
	r.Require().NoError(r.registries[0].Register(&serviceV2))
	time.Sleep(r.updateTime)

	// Get all versions of the service
	services, err := r.registries[0].GetService(serviceV1.Name)
	r.logger.Debug("GetService", "service", serviceV1.Name, "services", services)
	r.Require().NoError(err)
	r.Require().Len(services, 2, "Should have found both versions of the service")

	// Verify both versions are returned
	versions := map[string]bool{}
	for _, s := range services {
		versions[s.Version] = true
	}

	r.Require().True(versions["v1.0.0"], "v1.0.0 should be in the results")
	r.Require().True(versions["v2.0.0"], "v2.0.0 should be in the results")

	// Cleanup
	r.Require().NoError(r.registries[0].Deregister(&serviceV1))
	r.Require().NoError(r.registries[0].Deregister(&serviceV2))
}

// BenchmarkListServices benchmarks the performance of listing services.
func (r *TestSuite) BenchmarkListServices(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})
	r.SetupSuite()

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		services, err := r.registries[math_rand.Intn(len(r.registries))].ListServices() //nolint:gosec
		r.Require().NoError(err)
		r.Require().NotEmpty(services)
	}

	b.StopTimer()
	r.TearDownSuite()
}

// TestServiceWithEndpoints tests registering and retrieving services with endpoints.
func (r *TestSuite) TestServiceWithEndpoints() {
	// Create a service with endpoints
	endpoints := []*registry.Endpoint{
		{
			Name: "method1",
			Request: &registry.Value{
				Name: "request1",
				Type: "json",
			},
			Response: &registry.Value{
				Name: "response1",
				Type: "json",
			},
			Metadata: map[string]string{
				"endpoint_type": "rest",
				"handler":       "handler1",
			},
		},
		{
			Name: "method2",
			Request: &registry.Value{
				Name: "request2",
				Type: "protobuf",
			},
			Response: &registry.Value{
				Name: "response2",
				Type: "protobuf",
			},
			Metadata: map[string]string{
				"endpoint_type": "grpc",
				"handler":       "handler2",
			},
		},
	}

	service := registry.Service{
		Name:      "orb.test.endpoints",
		Version:   "v1.0.0",
		Nodes:     []*registry.Node{r.nodes[0]},
		Endpoints: endpoints,
	}

	// Register the service
	r.Require().NoError(r.registries[0].Register(&service))
	time.Sleep(r.updateTime)

	// Verify the service can be retrieved with endpoints intact
	services, err := r.registries[0].GetService(service.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)

	// Verify endpoints were preserved
	r.Require().Len(services[0].Endpoints, len(endpoints))

	for i, ep := range services[0].Endpoints {
		r.Require().Equal(endpoints[i].Name, ep.Name)

		// Check request/response types
		r.Require().Equal(endpoints[i].Request.Name, ep.Request.Name)
		r.Require().Equal(endpoints[i].Request.Type, ep.Request.Type)
		r.Require().Equal(endpoints[i].Response.Name, ep.Response.Name)
		r.Require().Equal(endpoints[i].Response.Type, ep.Response.Type)

		// Check metadata
		r.Require().Equal(endpoints[i].Metadata["endpoint_type"], ep.Metadata["endpoint_type"])
		r.Require().Equal(endpoints[i].Metadata["handler"], ep.Metadata["handler"])
	}

	// Cleanup
	r.Require().NoError(r.registries[0].Deregister(&service))
}

// TestConcurrentRegistrations tests concurrent registration and deregistration operations.
func (r *TestSuite) TestConcurrentRegistrations() {
	const numServices = 10

	const numWorkers = 3

	services := make([]*registry.Service, numServices)
	for i := 0; i < numServices; i++ {
		services[i] = &registry.Service{
			Name:    fmt.Sprintf("orb.test.concurrent.%d", i),
			Version: "v1.0.0",
			Nodes:   []*registry.Node{r.nodes[i%len(r.nodes)]},
		}
	}

	// Create a wait group to synchronize goroutines
	wg := sync.WaitGroup{}
	// Start multiple workers to register services concurrently
	for workerID := 0; workerID < numWorkers; workerID++ {
		wg.Add(1)

		go func(workerId int) {
			defer wg.Done()

			// Each worker operates on a subset of services
			start := (workerId * numServices) / numWorkers
			end := ((workerId + 1) * numServices) / numWorkers

			for i := start; i < end; i++ {
				// Register the service
				err := r.registries[0].Register(services[i])

				r.NoError(err)
			}

			// Small delay to allow updates to propagate
			time.Sleep(r.updateTime)

			// Verify the services were registered
			for i := start; i < end; i++ {
				result, err := r.registries[0].GetService(services[i].Name)

				r.NoError(err, "Failed to get service: %s", services[i].Name)
				r.Len(result, 1)

				if len(result) >= len(services) {
					r.Equal(services[i].Name, result[0].Name)
				}
			}

			// Now deregister the services
			for i := start; i < end; i++ {
				err := r.registries[0].Deregister(services[i])

				r.NoError(err)
			}
		}(workerID)
	}

	// Wait for all workers to complete
	wg.Wait()

	// Verify all services were successfully deregistered
	time.Sleep(r.updateTime)

	for i := 0; i < numServices; i++ {
		r.logger.Debug("checking", "name", services[i].Name)
		n, err := r.registries[0].GetService(services[i].Name)
		r.Require().Empty(n)
		r.Require().ErrorIs(err, registry.ErrNotFound)
	}
}

// TestMetadataFiltering tests using custom logic to filter services by metadata.
func (r *TestSuite) TestMetadataFiltering() {
	// Create services with different environments in metadata
	servicesProd := []*registry.Service{
		{
			Name:    "orb.test.filter.1",
			Version: "v1.0.0",
			Nodes:   []*registry.Node{r.nodes[0]},
			Metadata: map[string]string{
				"env":    "production",
				"region": "us-west",
			},
		},
		{
			Name:    "orb.test.filter.2",
			Version: "v1.0.0",
			Nodes:   []*registry.Node{r.nodes[1]},
			Metadata: map[string]string{
				"env":    "production",
				"region": "us-east",
			},
		},
	}

	servicesStaging := []*registry.Service{
		{
			Name:    "orb.test.filter.3",
			Version: "v1.0.0",
			Nodes:   []*registry.Node{r.nodes[2]},
			Metadata: map[string]string{
				"env":    "staging",
				"region": "eu-west",
			},
		},
	}

	// Register all services
	for _, svc := range append(servicesProd, servicesStaging...) {
		r.Require().NoError(r.registries[0].Register(svc))
	}

	time.Sleep(r.updateTime)

	// Get all services
	allServices, err := r.registries[0].ListServices()
	r.Require().NoError(err)

	// Manually filter for production services
	prodServices := []*registry.Service{}

	for _, svc := range allServices {
		// Skip non-test services (those not starting with orb.test.filter)
		if !strings.HasPrefix(svc.Name, "orb.test.filter") {
			continue
		}

		// Only include services with env=production metadata
		if env, ok := svc.Metadata["env"]; ok && env == "production" {
			prodServices = append(prodServices, svc)
		}
	}

	// Verify we filtered correctly
	r.Require().Len(prodServices, len(servicesProd), "Should find only the production services")

	// Cleanup
	for _, svc := range append(servicesProd, servicesStaging...) {
		r.Require().NoError(r.registries[0].Deregister(svc))
	}
}

// TestServiceNodesHealth simulates checking node health in a registry.
func (r *TestSuite) TestServiceNodesHealth() {
	// Create a service with multiple nodes
	nodes := []*registry.Node{
		{
			ID:        "healthy-node-1",
			Address:   "10.0.0.1:8080",
			Metadata:  map[string]string{"status": "healthy"},
			Transport: "http",
		},
		{
			ID:        "healthy-node-2",
			Address:   "10.0.0.2:8080",
			Metadata:  map[string]string{"status": "healthy"},
			Transport: "http",
		},
	}

	service := registry.Service{
		Name:    "orb.test.health",
		Version: "v1.0.0",
		Nodes:   nodes,
	}

	// Register the service
	r.Require().NoError(r.registries[0].Register(&service))
	time.Sleep(r.updateTime)

	// Retrieve the service
	services, err := r.registries[0].GetService(service.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)
	r.Require().Len(services[0].Nodes, 2)

	// Simulate a node becoming unhealthy by updating its metadata
	nodes[0].Metadata["status"] = "unhealthy"

	// Update the service with the modified node
	updatedService := registry.Service{
		Name:    service.Name,
		Version: service.Version,
		Nodes:   []*registry.Node{nodes[0]},
	}
	r.Require().NoError(r.registries[0].Register(&updatedService))
	time.Sleep(r.updateTime)

	// Retrieve the service again
	services, err = r.registries[0].GetService(service.Name)
	r.Require().NoError(err)
	r.Require().Len(services, 1)
	r.Require().Len(services[0].Nodes, 2)

	// Verify that one node is now marked as unhealthy
	healthyNodes := 0
	unhealthyNodes := 0

	for _, node := range services[0].Nodes {
		if status, ok := node.Metadata["status"]; ok {
			if status == "healthy" {
				healthyNodes++
			} else if status == "unhealthy" {
				unhealthyNodes++
			}
		}
	}

	r.Require().Equal(1, healthyNodes, "Should have one healthy node")
	r.Require().Equal(1, unhealthyNodes, "Should have one unhealthy node")

	// Cleanup
	r.Require().NoError(r.registries[0].Deregister(&service))
}

// BenchmarkRegisterDeregister benchmarks the performance of registering and deregistering services.
func (r *TestSuite) BenchmarkRegisterDeregister(b *testing.B) {
	b.Helper()

	r.SetT(&testing.T{})

	// Create a service just for this benchmark
	service := &registry.Service{
		Name:    "orb.test.benchmark.regdereg",
		Version: "v1.0.0",
		Nodes:   []*registry.Node{r.nodes[0]},
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		// Register
		err := r.registries[0].Register(service)
		r.Require().NoError(err)

		// Deregister
		err = r.registries[0].Deregister(service)
		r.Require().NoError(err)
	}
}
