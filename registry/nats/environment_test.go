package nats

import (
	"os"
	"testing"

	log "go-micro.dev/v5/log"
	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/types"
)

type environment struct {
	registryOne   registry.Registry
	registryTwo   registry.Registry
	registryThree registry.Registry

	serviceOne registry.Service
	serviceTwo registry.Service

	nodeOne   registry.Node
	nodeTwo   registry.Node
	nodeThree registry.Node
}

var e environment

func TestMain(m *testing.M) {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		log.Info("NATS_URL is undefined - skipping tests")
		return
	}

	logger, err := log.New(log.NewConfig())
	if err != nil {
		log.Error("while creating a logger", err)
	}

	cfg, err := NewConfig(types.ServiceName("test.service"), nil)
	if err != nil {
		log.Error("while creating a config", err)
	}

	e.registryOne = New(cfg, logger)
	e.registryTwo = New(cfg, logger)
	e.registryThree = New(cfg, logger)

	e.serviceOne.Name = "one"
	e.serviceOne.Version = "default"
	e.serviceOne.Nodes = []*registry.Node{&e.nodeOne}

	e.serviceTwo.Name = "two"
	e.serviceTwo.Version = "default"
	e.serviceTwo.Nodes = []*registry.Node{&e.nodeOne, &e.nodeTwo}

	e.nodeOne.ID = "one"
	e.nodeTwo.ID = "two"
	e.nodeThree.ID = "three"

	if err := e.registryOne.Register(&e.serviceOne); err != nil {
		log.Error("while test registering serviceOne", err)
	}

	if err := e.registryOne.Register(&e.serviceTwo); err != nil {
		log.Error("while test registering serviceTwo", err)
	}

	result := m.Run()

	if err := e.registryOne.Deregister(&e.serviceOne); err != nil {
		log.Error("while test deregistering serviceOne", err)
	}

	if err := e.registryOne.Deregister(&e.serviceTwo); err != nil {
		log.Error("while test deregistering serviceTwo", err)
	}

	os.Exit(result)
}
