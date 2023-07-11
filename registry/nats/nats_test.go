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

	_ "github.com/go-orb/plugins/log/text"
	"github.com/go-orb/plugins/registry/tests"

	nserver "github.com/nats-io/nats-server/v2/server"

	log "github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	"golang.org/x/exp/slog"
)

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

		regOne   registry.Registry
		regTwo   registry.Registry
		regThree registry.Registry
	)

	// Sometimes the nats server has isssues with starting, so we attempt 5
	// times.
	for i := 0; i < 5; i++ {
		logger.Info("starting NATS server", slog.Int("attempt", i))

		// start the NATS with JetStream server
		addr, cleanup, err = natsServer(clusterName, logger)
		if err != nil {
			log.Error("failed to setup NATS server", err, slog.Int("attempt", i))
			continue
		}

		cfg, err := NewConfig(types.ServiceName("test.service"), nil, WithAddress(addr))
		if err != nil {
			log.Error("failed to create config", err)
		}

		regOne = New(cfg, logger)
		if err := regOne.Start(); err != nil {
			log.Error("failed to connect registry one to NATS server", err, slog.Int("attempt", i))
			continue
		}

		regTwo = New(cfg, logger)
		if err := regTwo.Start(); err != nil {
			log.Error("failed to connect registry two to NATS server", err, slog.Int("attempt", i))
			continue
		}

		regThree = New(cfg, logger)
		if err := regThree.Start(); err != nil {
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

	tests.CreateSuite(logger, []registry.Registry{regOne, regTwo, regThree}, 0, 0)
	tests.Suite.Setup()

	result := m.Run()

	tests.Suite.TearDown()

	cleanup()

	os.Exit(result)
}

func getFreeLocalhostAddress() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	return l.Addr().String(), l.Close()
}

func natsServer(clustername string, logger log.Logger) (string, func(), error) {
	addr, err := getFreeLocalhostAddress()
	if err != nil {
		return "", nil, err
	}
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
		NewLogWrapper(logger),
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

	logger.Info("NATS server started")

	return addr, cleanup, nil
}

func NewLogWrapper(logger log.Logger) *LogWrapper {
	return &LogWrapper{
		logger: logger,
	}
}

type LogWrapper struct {
	logger log.Logger
}

// Noticef logs a notice statement.
func (l *LogWrapper) Noticef(_ string, _ ...interface{}) {
}

// Warnf logs a warning statement.
func (l *LogWrapper) Warnf(format string, v ...interface{}) {
	l.logger.Warn(format, v...)
}

// Fatalf logs a fatal statement.
func (l *LogWrapper) Fatalf(format string, v ...interface{}) {
	l.logger.Error(format, v...)
}

// Errorf logs an error statement.
func (l *LogWrapper) Errorf(format string, v ...interface{}) {
	l.logger.Error(format, v...)
}

// Debugf logs a debug statement.
func (l *LogWrapper) Debugf(format string, v ...interface{}) {
	l.logger.Debug(format, v...)
}

// Tracef logs a trace statement.
func (l *LogWrapper) Tracef(format string, v ...interface{}) {
	l.logger.Trace(format, v...)
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
