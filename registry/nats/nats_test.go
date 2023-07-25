package nats

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/go-orb/plugins/log/slog"
	"github.com/go-orb/plugins/registry/tests"
	"github.com/pkg/errors"

	log "github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"

	"golang.org/x/exp/slog"
)

func TestMain(m *testing.M) {
	logger, err := log.New()
	if err != nil {
		log.Error("while creating a logger", err)
	}

	var (
		started bool

		regOne   registry.Registry
		regTwo   registry.Registry
		regThree registry.Registry
	)

	logger.Info("starting NATS server")

	// start the NATS with JetStream server
	addr, cleanup, err := natsServer()
	if err != nil {
		log.Error("failed to setup NATS server", err)
	}

	// Sometimes the nats server has isssues with starting, so we attempt 5
	// times.
	for i := 0; i < 5; i++ {
		cfg, err := NewConfig(types.ServiceName("test.service"), nil, WithAddress(addr))
		if err != nil {
			log.Error("failed to create config", err)
		}

		regOne = New("", "", cfg, logger)
		if err := regOne.Start(); err != nil {
			log.Error("failed to connect registry one to NATS server", err, slog.Int("attempt", i))

			time.Sleep(time.Second)
			continue
		}

		regTwo = New("", "", cfg, logger)
		if err := regTwo.Start(); err != nil {
			log.Error("failed to connect registry two to NATS server", err, slog.Int("attempt", i))
		}

		regThree = New("", "", cfg, logger)
		if err := regThree.Start(); err != nil {
			log.Error("failed to connect registry three to NATS server", err, slog.Int("attempt", i))
		}

		started = true
	}

	if !started {
		log.Error("failed to start NATS server", nil)
		os.Exit(1)
	}

	tests.CreateSuite(logger, []registry.Registry{regOne, regTwo, regThree}, 0, 0)
	tests.Suite.Setup()

	result := m.Run()

	tests.Suite.TearDown()

	if err := cleanup(); err != nil {
		log.Error("Stopping nats failed", err)
	}

	os.Exit(result)
}

func getFreeLocalhostAddress() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	return l.Addr().String(), l.Close()
}

func natsServer() (string, func() error, error) {
	addr, err := getFreeLocalhostAddress()
	if err != nil {
		return "", nil, err
	}
	host := strings.Split(addr, ":")[0]
	port, _ := strconv.Atoi(strings.Split(addr, ":")[1]) //nolint:errcheck

	natsCmd, err := filepath.Abs(filepath.Join("./test/bin/", runtime.GOOS+"_"+runtime.GOARCH, "nats-server"))
	if err != nil {
		return addr, nil, err
	}

	args := []string{"--addr", host, "--port", fmt.Sprint(port), "-js"}
	cmd := exec.Command(natsCmd, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return addr, nil, errors.Wrap(err, "failed starting command")
	}

	cleanup := func() error {
		if cmd.Process == nil {
			return nil
		}

		if runtime.GOOS == "windows" {
			if err := cmd.Process.Kill(); err != nil {
				return errors.Wrap(err, "failed to kill nats server")
			}
		} else { // interrupt is not supported in windows
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				return errors.Wrap(err, "failed to kill nats server")
			}
		}

		return nil
	}

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
