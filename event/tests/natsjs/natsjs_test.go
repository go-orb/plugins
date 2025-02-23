package natsjs

import (
	"context"
	"errors"
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

	"github.com/go-orb/go-orb/event"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/plugins/event/tests"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/go-orb/plugins/event/natsjs"
)

func createServer() (log.Logger, event.Handler, context.CancelFunc, error) {
	logger, err := log.New(log.WithLevel("DEBUG"))
	if err != nil {
		log.Error("while creating a logger", err)
		return log.Logger{}, nil, func() {}, errors.New("while creating a logger")
	}

	var (
		started bool

		handler event.Handler
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
		cfg, err := natsjs.NewConfig(types.ServiceName("org.orb.testservice"), nil, natsjs.WithAddresses(addr))
		if err != nil {
			log.Error("failed to create config", err)
		}

		handler = natsjs.New("org.orb.testservice", cfg, logger)
		err = handler.Start()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		started = true
	}

	if !started {
		log.Error("failed to start NATS server", err)
		return log.Logger{}, nil, func() {}, errors.New("failed to start nats server")
	}

	return logger, handler, cleanup, nil
}

func getFreeLocalhostAddress() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	return l.Addr().String(), l.Close()
}

func natsServer() (string, context.CancelFunc, error) {
	addr, err := getFreeLocalhostAddress()
	if err != nil {
		return "", nil, err
	}
	host := strings.Split(addr, ":")[0]
	port, _ := strconv.Atoi(strings.Split(addr, ":")[1]) //nolint:errcheck

	natsCmd, err := filepath.Abs(filepath.Join("../test/bin/", runtime.GOOS+"_"+runtime.GOARCH, "nats-server"))
	if err != nil {
		return addr, nil, err
	}

	args := []string{"--addr", host, "--port", strconv.Itoa(port), "-js"}
	cmd := exec.Command(natsCmd, args...)
	if err := cmd.Start(); err != nil {
		return addr, nil, fmt.Errorf("failed starting command: %w", err)
	}

	cleanup := func() {
		if cmd.Process == nil {
			return
		}

		if runtime.GOOS == "windows" {
			if err := cmd.Process.Kill(); err != nil {
				return
			}
		} else { // interrupt is not supported in windows
			if err := cmd.Process.Signal(os.Interrupt); err != nil {
				return
			}
		}
	}

	return addr, cleanup, nil
}

func TestSuite(t *testing.T) {
	logger, handler, cleanup, err := createServer()
	require.NoError(t, err)

	suite.Run(t, tests.New(logger, handler))

	cleanup()
}

func BenchmarkRequest(b *testing.B) {
	b.StopTimer()

	logger, handler, cleanup, err := createServer()
	require.NoError(b, err, "while creating a server")

	s := tests.New(logger, handler)
	s.BenchmarkRequest(b)

	cleanup()
}
