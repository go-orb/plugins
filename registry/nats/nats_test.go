package nats

import (
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	log "github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
)

func createServer() (*tests.TestSuite, func() error, error) {
	logger, err := log.New(log.WithLevel("DEBUG"))
	if err != nil {
		log.Error("while creating a logger", err)
		return nil, func() error { return nil }, errors.New("while creating a logger")
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
		err = regOne.Start()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		regTwo = New("", "", cfg, logger)
		regTwo.Start() //nolint: errcheck

		regThree = New("", "", cfg, logger)
		regThree.Start() //nolint: errcheck

		started = true
	}

	if !started {
		log.Error("failed to start NATS server", err)
		return nil, func() error { return nil }, errors.New("failed to start nats server")
	}

	s := tests.CreateSuite(logger, []registry.Registry{regOne, regTwo, regThree}, 0, 0)
	return s, cleanup, nil
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

	args := []string{"--addr", host, "--port", strconv.Itoa(port), "-js"}
	cmd := exec.Command(natsCmd, args...)
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

func TestSuite(t *testing.T) {
	s, cleanup, err := createServer()
	require.NoError(t, err, "while creating a server")

	// Run the tests.
	suite.Run(t, s)

	require.NoError(t, cleanup(), "while cleaning up")
}

func BenchmarkGetService(b *testing.B) {
	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetService(b)

	require.NoError(b, cleanup(), "while cleaning up")
}

func BenchmarkGetServiceWithNoNodes(b *testing.B) {
	b.StopTimer()

	s, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s.BenchmarkGetServiceWithNoNodes(b)

	require.NoError(b, cleanup(), "while cleaning up")
}
