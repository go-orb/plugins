// bench_client contains a client to benchmark `tests_server`.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	// go-orb.
	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"

	// Own imports.
	echoproto "github.com/go-orb/plugins/benchmarks/rps/proto/echo"

	_ "github.com/go-orb/plugins/client/orb"
	_ "github.com/go-orb/plugins/codecs/jsonpb"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/cli/urfave"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/registry/consul"

	_ "github.com/go-orb/plugins/client/middleware/log"

	// All transports.
	_ "github.com/go-orb/plugins/client/orb_transport/drpc"
	_ "github.com/go-orb/plugins/client/orb_transport/grpc"
	_ "github.com/go-orb/plugins/client/orb_transport/h2c"
	_ "github.com/go-orb/plugins/client/orb_transport/hertzh2c"
	_ "github.com/go-orb/plugins/client/orb_transport/hertzhttp"
	_ "github.com/go-orb/plugins/client/orb_transport/http"
	_ "github.com/go-orb/plugins/client/orb_transport/http3"
	_ "github.com/go-orb/plugins/client/orb_transport/https"
)

const serverName = "benchmarks.rps.server"

type stats struct {
	Ok    uint64
	Error uint64
}

func connection(
	ctx context.Context,
	wg *sync.WaitGroup,
	cli client.Type,
	logger log.Logger,
	msg []byte,
	opts []client.CallOption,
	connectionNum int,
	statsChan chan stats,
) {
	var (
		reqsOk    uint64
		reqsError uint64
	)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("Connection results", "connection", connectionNum, "reqsOk", reqsOk, "reqsError", reqsError)
			wg.Done()

			statsChan <- stats{Ok: reqsOk, Error: reqsError}

			return
		default:
		}

		// Create a request.
		req := &echoproto.Req{Payload: msg}

		// Run the query.
		resp, err := client.Call[echoproto.Resp](
			ctx,
			cli,
			serverName,
			"echo.Echo/Echo",
			req,
			opts...,
		)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				continue
			}

			logger.Error("while requesting", "error", err)
			reqsError++

			continue
		}

		// Check if response equals.
		if !bytes.Equal(req.GetPayload(), resp.GetPayload()) {
			logger.Error("request and response are not the same")
			reqsError++

			continue
		}

		reqsOk++
	}
}

// bench.
//
//nolint:funlen
func bench(
	sn types.ServiceName,
	configs types.ConfigData,
	logger log.Logger,
	cli client.Type,
) error {
	cfg := &clientConfig{
		BypassRegistry: defaultBypassRegistry,
		Connections:    defaultConnections,
		Duration:       defaultDuration,
		Timeout:        defaultTimeout,
		Threads:        defaultThreads,
		Transport:      defaultTransport,
		PackageSize:    defaultPackageSize,
		ContentType:    defaultContentType,
	}

	sections := append(types.SplitServiceName(sn), configSection)
	if err := config.Parse(sections, configs, &cfg); err != nil {
		return err
	}

	logger.Info(
		"Config",
		"bypass_registry", cfg.BypassRegistry,
		"connections", cfg.Connections,
		"duration", cfg.Duration,
		"timeout", cfg.Timeout,
		"threads", cfg.Threads,
		"transport", cfg.Transport,
		"package_size", cfg.PackageSize,
		"content_type", cfg.ContentType,
	)

	runtime.GOMAXPROCS(cfg.Threads)

	// Setup client options.
	opts := []client.CallOption{
		client.WithPoolSize(cfg.Connections),
		client.WithPreferredTransports(cfg.Transport),
		client.WithContentType(cfg.ContentType),
	}

	if err := cli.With(client.WithClientPoolSize(cfg.Connections)); err != nil {
		return err
	}

	wCtx, wCancel := context.WithCancel(context.Background())

	// Cache URL
	if cfg.BypassRegistry == 1 {
		logger.Debug("Resolving", "server", serverName)

		nodes, err := cli.ResolveService(wCtx, serverName, cfg.Transport)
		if err != nil {
			logger.Error("Failed to resolve service, did you start the server?", "error", err, "server", serverName)
			wCancel()

			return err
		}

		var preferredTransports []string
		if len(cfg.Transport) != 0 {
			preferredTransports = []string{cfg.Transport}
		} else {
			preferredTransports = cli.Config().PreferredTransports
		}

		node, err := cli.Config().Selector(wCtx, serverName, nodes, preferredTransports, false)
		if err != nil {
			logger.Error("Failed to resolve service, did you start the server?", "error", err, "server", serverName)
			wCancel()

			return err
		}

		opts = append(opts, client.WithURL(fmt.Sprintf("%s://%s", node.Transport, node.Address)))

		logger.Info("Using transport", "transport", node.Transport)
	}

	// Create random bytes to ping-pong on each request.
	msg := make([]byte, cfg.PackageSize)
	if _, err := rand.Reader.Read(msg); err != nil {
		logger.Error("Failed to make a request", "error", err)
		wCancel()

		return err
	}

	var wg sync.WaitGroup

	quit := make(chan os.Signal, 1)
	done := make(chan os.Signal, 1)

	// End requests on SIGINT/SIGTERM.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	//
	// Warmup
	//

	timer := time.AfterFunc(time.Second*time.Duration(cfg.Duration), func() {
		done <- syscall.SIGINT
	})

	logger.Info("Warming up...")

	nullChan := make(chan stats, cfg.Connections)

	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)

		go connection(wCtx, &wg, cli, logger, msg, opts, i, nullChan)
	}

	select {
	case <-done:
		wCancel()
		timer.Stop()
	case <-quit:
		timer.Stop()
		os.Exit(1)
	}

	//
	// Bench
	//
	logger.Info("Now running the benchmark")

	ctx, cancel := context.WithCancel(context.Background())

	// Timer to end requests
	timer = time.AfterFunc(time.Second*time.Duration(cfg.Duration), func() {
		done <- syscall.SIGINT
	})

	// Statistics channel
	statsChan := make(chan stats, cfg.Connections)

	// Run the requests.
	for i := 0; i < cfg.Connections; i++ {
		wg.Add(1)

		go connection(ctx, &wg, cli, logger, msg, opts, i, statsChan)
	}

	// Blocks until timer/signal happened
	select {
	case <-done:
		timer.Stop()
		// stops requesting
		cancel()

		// Wait for all goroutines to exit properly.
		wg.Wait()
	case <-quit:
		timer.Stop()
		os.Exit(0)
	}

	// Calculate stats
	mStats := stats{}

	for i := 0; i < cfg.Connections; i++ {
		cStat := <-statsChan

		mStats.Ok += cStat.Ok
		mStats.Error += cStat.Error
	}

	logger.Info("Summary",
		"bypass_registry", cfg.BypassRegistry,
		"connections", cfg.Connections,
		"duration", cfg.Duration,
		"timeout", cfg.Timeout,
		"threads", cfg.Threads,
		"transport", cfg.Transport,
		"package_size", cfg.PackageSize,
		"content_type", cfg.ContentType,
		"reqsOk", mStats.Ok,
		"reqsError", mStats.Error,
	)

	return nil
}

func main() {
	var (
		serviceName    = types.ServiceName("benchmarks.rps.client")
		serviceVersion = types.ServiceVersion("v0.0.1")
	)

	if _, err := run(serviceName, serviceVersion, bench); err != nil {
		log.Error("While running", err)
		os.Exit(1)
	}
}
