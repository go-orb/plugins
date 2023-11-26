// Package tests contains tests for go-orb/plugins/client/*.
package tests

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/plugins/client/tests/handler"
	"github.com/go-orb/plugins/client/tests/proto"
	"github.com/stretchr/testify/suite"
	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
)

//nolint:gochecknoglobals
var (
	ServiceName     = types.ServiceName("org.orb.svc.service")
	DefaultRequests = []TestRequest{
		{
			Name:     "32byte",
			Endpoint: "/echo.Streams/Call",
			Request: &proto.CallRequest{
				Name: "32byte",
			},
			Response: &proto.CallResponse{
				Msg: "",
			},
		},
		{
			Name:     "default codec with URL",
			Endpoint: "/echo.Streams/Call",
			Request: &proto.CallRequest{
				Name: "Alex",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
			URL: "t",
		},
		{
			Name:     "default codec",
			Endpoint: "/echo.Streams/Call",
			Request: &proto.CallRequest{
				Name: "Alex",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:        "application/proto",
			Endpoint:    "/echo.Streams/Call",
			ContentType: "application/proto",
			Request: &proto.CallRequest{
				Name: "Alex",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:        "application/json",
			Endpoint:    "/echo.Streams/Call",
			ContentType: "application/json",
			Request: &proto.CallRequest{
				Name: "Alex",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:     "error request",
			Endpoint: "/echo.Streams/Call",
			Error:    true,
			Request: &proto.CallRequest{
				Name: "error",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
		},
	}
)

// TestSuite runs a bunch of tests / benchmarks.
type TestSuite struct {
	suite.Suite

	// The path of plugins/
	PluginsRoot string

	// Transports is the list of preferred transports for all requests
	Transports []string

	// Requests is the requests to make.
	Requests []TestRequest

	logger   log.Logger
	registry registry.Type

	serverRunner *PackageRunner
	client       client.Type
}

// NewSuite creates a new test suite.
func NewSuite(pluginsRoot string, transports []string, requests ...TestRequest) *TestSuite {
	s := new(TestSuite)

	s.PluginsRoot = pluginsRoot
	s.Transports = transports

	if len(requests) == 0 {
		s.Requests = DefaultRequests
	}

	return s
}

// TestRequest contains all informations to run a test request.
type TestRequest struct {
	Name     string
	Service  string
	Endpoint string
	// PreferredTransports overwrites the list of preferred transports.
	PreferredTransports []string
	// ContentType overwrites the client's content-type.
	ContentType string
	// URL when set bypasses the registry.
	URL string
	// Expect an error?
	Error bool

	Request  *proto.CallRequest
	Response *proto.CallResponse
}

func init() {
	server.Handlers.Set("Streams",
		server.NewRegistrationFunc[grpc.ServiceRegistrar, proto.StreamsServer](
			proto.RegisterStreamsServer,
			new(handler.EchoHandler),
		))
}

// SetupSuite setups the test suite.
func (s *TestSuite) SetupSuite() {
	version := types.ServiceVersion("v1.0.0")

	cURLs := []*url.URL{}

	curl, err := url.Parse(fmt.Sprintf("file://%s", filepath.Join(s.PluginsRoot, "client/tests/cmd/tests_server/config.yaml")))
	s.Require().NoError(err, "while parsing a url")

	cURLs = append(cURLs, curl)

	cfgData, err := config.Read(cURLs, nil)
	if err != nil {
		s.Require().NoError(err, "while parsing a config")
	}

	clientName := types.ServiceName("org.orb.svc.client")

	// Logger
	logger, err := log.ProvideLogger(clientName, cfgData)
	s.Require().NoError(err, "while setting up logger")
	s.Require().NoError(logger.Start())
	s.logger = logger

	// Registry
	reg, err := registry.ProvideRegistry(clientName, version, cfgData, logger)
	if err != nil {
		s.Require().NoError(err, "while creating a registry")
	}

	s.Require().NoError(reg.Start())
	s.registry = reg

	// Client
	c, err := client.ProvideClient(clientName, cfgData, logger, reg)
	if err != nil {
		s.Require().NoError(err, "while creating a client")
	}

	s.Require().NoError(c.Start())
	s.client = c

	if len(s.Transports) == 0 {
		s.Transports = s.client.Config().PreferredTransports
	}

	// Start a server
	pro := []PackageRunnerOption{
		WithOverwrite(),
		WithRunEnv("GOMAXPROCS=" + os.Getenv("GOMAXPROCS")),
		WithArgs("--config", filepath.Join(s.PluginsRoot, "client/tests/cmd/tests_server/config.yaml")),
	}
	if logger.Level() <= slog.LevelDebug {
		pro = append(pro, WithStdOut(os.Stdout), WithStdErr(os.Stderr))
	}

	s.serverRunner = NewPackageRunner(
		logger,
		filepath.Join(s.PluginsRoot, "client/tests/cmd/tests_server"),
		filepath.Join(s.PluginsRoot, "client/tests/tmp/tests_server"),
		pro...,
	)
	s.Require().NoError(s.serverRunner.Build())
	s.Require().NoError(s.serverRunner.Start())

	// Wait for the server to be registered (up to 5 seconds)
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)

		if _, err = s.client.ResolveService(context.Background(), string(ServiceName), s.Transports...); err == nil {
			break
		}
	}

	if err != nil {
		s.Require().NoError(err, "failed to wait for the server")
		s.Require().NoError(s.serverRunner.Kill(), "while stopping the server sub process")
	}
}

// TearDownSuite runs after all tests.
func (s *TestSuite) TearDownSuite() {
	ctx := context.Background()

	err := s.client.Stop(ctx)
	s.Require().NoError(err, "while stopping the client")

	err = s.registry.Stop(ctx)
	s.Require().NoError(err, "while stopping the registry")

	err = s.logger.Stop(ctx)
	s.Require().NoError(err, "while stopping the logger")

	if s.serverRunner != nil {
		s.Require().NoError(s.serverRunner.Kill(), "while stopping the server sub process")
	}
}

func (s *TestSuite) doRequest(ctx context.Context, req *TestRequest) {
	opts := []client.CallOption{}
	if req.ContentType != "" {
		opts = append(opts, client.WithContentType(req.ContentType))
	}

	if req.URL != "" {
		opts = append(opts, client.WithURL(req.URL))
	}

	if len(s.Transports) != 0 {
		opts = append(opts, client.WithPreferredTransports(s.Transports...))
	}

	rsp, err := client.Call[proto.CallResponse](
		ctx,
		s.client,
		req.Service,
		req.Endpoint,
		req.Request,
		opts...,
	)

	if req.Error {
		s.Require().Error(err)
	} else {
		s.Require().NoError(err)
		s.Equal(req.Response.GetMsg(), rsp.GetMsg(), "unexpected response")
	}
}

// TestResolveServiceTransport checks if the right transport has been selected.
func (s *TestSuite) TestResolveServiceTransport() {
	ctx := context.Background()

	nodes, err := s.client.ResolveService(ctx, string(ServiceName), s.Transports...)
	s.Require().NoError(err)

	node, err := s.client.Config().Selector(ctx, string(ServiceName), nodes, s.Transports, s.client.Config().AnyTransport)
	s.Require().NoError(err)

	s.Require().True(slices.Contains(s.Transports, node.Transport))
}

// TestRunRequests makes the configured requests.
func (s *TestSuite) TestRunRequests() {
	for _, oReq := range s.Requests {
		ctx := context.Background()
		req := oReq

		s.Run(req.Name, func() {
			req.Service = string(ServiceName)
			if req.URL == "t" {
				nodes, err := s.client.ResolveService(ctx, req.Service, s.Transports...)
				s.Require().NoError(err)

				node, err := s.client.Config().Selector(ctx, req.Service, nodes, s.Transports, s.client.Config().AnyTransport)
				s.Require().NoError(err)
				req.URL = fmt.Sprintf("%s://%s", node.Transport, node.Address)
			}

			s.doRequest(ctx, &req)
		})
	}
}

func (s *TestSuite) runRequestBenchmark(b *testing.B, req TestRequest, pN int) {
	var (
		nodes *container.Map[[]*registry.Node]
		err   error
	)

	if req.URL == "t" {
		nodes, err = s.client.ResolveService(context.Background(), req.Service, req.PreferredTransports...)
		s.Require().NoError(err)
	}

	done := make(chan struct{})

	var wg sync.WaitGroup

	b.StartTimer()

	// Start requests
	go func() {
		for i := 0; i < b.N; i++ {
			// Run parallel requests.
			for p := 0; p < pN; p++ {
				wg.Add(1)

				requestor := func() {
					myReq := req
					myReq.PreferredTransports = s.Transports

					if myReq.URL == "t" {
						node, err := s.client.Config().Selector(
							context.Background(),
							myReq.Service,
							nodes,
							req.PreferredTransports,
							s.client.Config().AnyTransport,
						)
						if err != nil {
							s.logger.Error("While requesting", "err", err)

							wg.Done()

							return
						}

						myReq.URL = fmt.Sprintf("%s://%s", node.Transport, node.Address)
					}

					s.doRequest(context.Background(), &myReq)
					wg.Done()
				}
				go requestor()
			}
			wg.Wait()
		}
		done <- struct{}{}
	}()

	// Wait for all jobs to finish
	<-done

	b.StopTimer()
}

// Benchmark runs b.N times, each with pN requests in parallel.
func (s *TestSuite) Benchmark(b *testing.B, contentType string, pN int) {
	b.StopTimer()

	req := s.Requests[0]
	req.Service = string(ServiceName)
	req.URL = "t" // Set to "t" to bypass the registry in benchmarks.
	req.PreferredTransports = s.Transports
	req.ContentType = contentType

	s.SetT(&testing.T{})
	s.SetupSuite()

	s.runRequestBenchmark(b, req, pN)

	s.TearDownSuite()
}
