package nats

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/go-micro/plugins/log/text"
	"github.com/go-micro/plugins/registry/tests"

	nserver "github.com/nats-io/nats-server/v2/server"

	log "github.com/go-micro/go-micro/log"
	"github.com/stretchr/testify/require"
	"github.com/go-micro/go-micro/registry"
	"github.com/go-micro/go-micro/types"

	"golang.org/x/exp/slog"
	"github.com/go-orb/go-orb/registry"
)

func TestRegister(t *testing.T) {
	service := registry.Service{Name: "test"}
	require.NoError(t, e.registryOne.Register(&service))
	defer func() {
		if err := e.registryOne.Deregister(&service); err != nil {
			panic(err)
		}

	services, err := e.registryOne.ListServices()
	require.NoError(t, err)
	require.Equal(t, 3, len(services))

	services, err = e.registryTwo.ListServices()
	require.NoError(t, err)
	require.Equal(t, 3, len(services))
}

func TestDeregister(t *testing.T) {
	service1 := registry.Service{Name: "test-deregister", Version: "v1"}
	service2 := registry.Service{Name: "test-deregister", Version: "v2"}

	require.NoError(t, e.registryOne.Register(&service1))
	services, err := e.registryOne.GetService(service1.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))

	require.NoError(t, e.registryOne.Register(&service2))
	services, err = e.registryOne.GetService(service2.Name)
	require.NoError(t, err)
	require.Equal(t, 2, len(services))

	require.NoError(t, e.registryOne.Deregister(&service1))
	services, err = e.registryOne.GetService(service1.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))

	require.NoError(t, e.registryOne.Deregister(&service2))
	services, err = e.registryOne.GetService(service1.Name)
	require.NoError(t, err)
	require.Equal(t, 0, len(services))
}

func TestGetService(t *testing.T) {
	services, err := e.registryTwo.GetService(e.serviceOne.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))
	require.Equal(t, "one", services[0].Name)
	require.Equal(t, 1, len(services[0].Nodes))

	require.Equal(t, e.nodeOne.Scheme, services[0].Nodes[0].Scheme)
}

func TestGetServiceWithNoNodes(t *testing.T) {
	services, err := e.registryOne.GetService("missing")
	require.NoError(t, err)
	require.Equal(t, 0, len(services))
}

func TestGetServiceFromMultipleNodes(t *testing.T) {
	services, err := e.registryOne.GetService(e.serviceTwo.Name)
	require.NoError(t, err)
	require.Equal(t, 1, len(services))
	require.Equal(t, "two", services[0].Name)
	require.Equal(t, 2, len(services[0].Nodes))

	require.Equal(t, e.nodeOne.Scheme, services[0].Nodes[0].Scheme)
	require.Equal(t, e.nodeTwo.Scheme, services[0].Nodes[1].Scheme)
}

func BenchmarkGetService(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := e.registryTwo.GetService(e.serviceOne.Name)
		require.NoError(b, err)
		require.Equal(b, 1, len(services))
		require.Equal(b, "one", services[0].Name)

		require.Equal(b, e.nodeOne.Scheme, services[0].Nodes[0].Scheme)
	}
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := e.registryOne.GetService("missing")
		require.NoError(b, err)
		require.Equal(b, 0, len(services))
	}
}

func BenchmarkGetServiceFromMultipleNodes(b *testing.B) {
	for n := 0; n < b.N; n++ {
		services, err := e.registryTwo.GetService(e.serviceTwo.Name)
		require.NoError(b, err)
		require.Equal(b, 1, len(services))
		require.Equal(b, "two", services[0].Name)
		require.Equal(b, 2, len(services[0].Nodes))

		require.Equal(b, e.nodeOne.Scheme, services[0].Nodes[0].Scheme)
		require.Equal(b, e.nodeTwo.Scheme, services[0].Nodes[1].Scheme)
	}
}
