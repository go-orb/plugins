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

	"golang.org/x/exp/slog"

	log "github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	_ "github.com/go-orb/plugins/log/text"

	nserver "github.com/nats-io/nats-server/v2/server"
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
	logger, err := log.New(log.NewConfig())
	if err != nil {
		log.Error("while creating a logger", err)
	}

	clusterName := "gomicro-registry-test-cluster"

	var (
		cleanup func()
		addr    string
		started bool
	)

	// Sometimes the nats server has isssues with starting, so we attempt 5
	// times.
	for i := 0; i < 5; i++ {
		logger.Info("starting NATS server", slog.Int("attempt", i))

		// start the NATS with JetStream server
		addr, cleanup, err = natsServer(clusterName)
		if err != nil {
			log.Error("failed to setup NATS server", err, slog.Int("attempt", i))
			continue
		}

		cfg, err := NewConfig(types.ServiceName("test.service"), nil, WithAddress(addr))
		if err != nil {
			log.Error("failed to create config", err)
		}

		e.registryOne = New(cfg, logger)
		if err := e.registryOne.Start(); err != nil {
			log.Error("failed to connect registry one to NATS server", err, slog.Int("attempt", i))
			continue
		}

		e.registryTwo = New(cfg, logger)
		if err := e.registryOne.Start(); err != nil {
			log.Error("failed to connect registry two to NATS server", err, slog.Int("attempt", i))
			continue
		}

		e.registryThree = New(cfg, logger)
		if err := e.registryOne.Start(); err != nil {
			log.Error("failed to connect registry three to NATS server", err, slog.Int("attempt", i))
			continue
		}

		started = true
		break
	}

	if !started {
		log.Error("failed to start NATS server", nil)
		os.Exit(1)
	}

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

	cleanup()

	os.Exit(result)
}

//nolint:errcheck
func getFreeLocalhostAddress() string {
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	defer l.Close()
	return l.Addr().String()
}

func natsServer(clustername string) (string, func(), error) {
	addr := getFreeLocalhostAddress()
	host := strings.Split(addr, ":")[0]
	port, _ := strconv.Atoi(strings.Split(addr, ":")[1]) //nolint:errcheck

	opts := &nserver.Options{
		Host:       host,
		TLSTimeout: 180,
		Port:       port,
		Cluster: nserver.ClusterOpts{
			Name: clustername,
		},
	}

	server, err := nserver.NewServer(opts)
	if err != nil {
		return "", nil, fmt.Errorf("nats new server: %w", err)
	}

	server.SetLoggerV2(
		NewLogWrapper(),
		false, false, false,
	)

	tmpdir := os.TempDir()
	natsdir := filepath.Join(tmpdir, "nats-js-tests")
	jsConf := &nserver.JetStreamConfig{
		StoreDir: natsdir,
	}

	// first start NATS
	go server.Start()

	time.Sleep(time.Second)

	// second start JetStream
	if err = server.EnableJetStream(jsConf); err != nil {
		server.Shutdown()
		return "", nil, fmt.Errorf("enable jetstream: %w", err)
	}

	time.Sleep(2 * time.Second)

	// This fixes some issues where tests fail because directory cleanup fails
	cleanup := func() {
		server.Shutdown()

		contents, _ := filepath.Glob(natsdir + "/*") //nolint:errcheck
		for _, item := range contents {
			_ = os.RemoveAll(item) //nolint:errcheck
		}
		_ = os.RemoveAll(natsdir) //nolint:errcheck
	}

	slog.Info("NATS server started")

	return addr, cleanup, nil
}

func NewLogWrapper() *LogWrapper {
	return &LogWrapper{}
}

type LogWrapper struct {
}

// Noticef logs a notice statement.
func (l *LogWrapper) Noticef(format string, v ...interface{}) {
}

// Warnf logs a warning statement.
func (l *LogWrapper) Warnf(format string, v ...interface{}) {
	slog.Warn(fmt.Sprintf(format+"\n", v...))
}

// Fatalf logs a fatal statement.
func (l *LogWrapper) Fatalf(format string, v ...interface{}) {
	slog.Error(fmt.Sprintf(format+"\n", v...), nil)
}

// Errorf logs an error statement.
func (l *LogWrapper) Errorf(format string, v ...interface{}) {
	slog.Error(fmt.Sprintf(format+"\n", v...), nil)
}

// Debugf logs a debug statement.
func (l *LogWrapper) Debugf(format string, v ...interface{}) {
}

// Tracef logs a trace statement.
func (l *LogWrapper) Tracef(format string, v ...interface{}) {
}
