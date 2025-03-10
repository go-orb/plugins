// Package tests contains tests for go-orb/plugins/client/*.
package tests

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"time"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/tests/proto"
	"github.com/stretchr/testify/suite"

	// Blank imports here are fine.
	_ "github.com/go-orb/plugins-experimental/registry/mdns"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/server/http/router/chi"
)

//nolint:gochecknoglobals
var (
	// ServiceName is the name of the testing service.
	ServiceName = types.ServiceName("service")

	// DefaultRequests is the list of default requests.
	DefaultRequests = []TestRequest{
		{
			Name:     "32byte",
			Endpoint: proto.EndpointStreamsCall,
			Request: &proto.CallRequest{
				Name: "32byte",
			},
			Response: &proto.CallResponse{
				Msg: "",
			},
			URL: "t",
		},
		{
			Name:        "raw-json",
			Endpoint:    proto.EndpointStreamsCall,
			ContentType: "application/json",
			Request:     `{"name": "Alex"}`,
			Response: map[string]any{
				"msg": "Hello Alex",
			},
		},
		{
			Name:     "default codec with URL",
			Endpoint: proto.EndpointStreamsCall,
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
			Endpoint: proto.EndpointStreamsCall,
			Request: &proto.CallRequest{
				Name: "Alex",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:        "proto",
			Endpoint:    proto.EndpointStreamsCall,
			ContentType: "application/x-protobuf",
			Request: &proto.CallRequest{
				Name: "Alex",
			},
			Response: &proto.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:        "json",
			Endpoint:    proto.EndpointStreamsCall,
			ContentType: "application/json",
			Request: map[string]any{
				"name": "Alex",
			},
			Response: map[string]any{
				"msg": "Hello Alex",
			},
		},
		{
			Name:     "error request",
			Endpoint: proto.EndpointStreamsCall,
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

// TestSuite runs a bunch of tests.
type TestSuite struct {
	suite.Suite

	// Transports is the list of preferred transports for all requests
	Transports []string

	// Requests is the requests to make.
	Requests []TestRequest

	logger   log.Logger
	registry registry.Type

	serverRunner *PackageRunner
	client       client.Type

	// To create more clients in Benchmarks.
	clientName types.ServiceName
	configData types.ConfigData
}

// NewSuite creates a new test suite.
func NewSuite(_ string, transports []string, requests ...TestRequest) *TestSuite {
	s := new(TestSuite)

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

	Request  any
	Response any
}

// SetupSuite setups the test suite.
func (s *TestSuite) SetupSuite() {
	version := types.ServiceVersion("v1.0.0")

	cURLs := []*url.URL{}

	cfgData, err := config.Read(cURLs)
	if err != nil {
		s.Require().NoError(err, "while parsing a config")
	}

	s.configData = cfgData
	s.clientName = types.ServiceName("client")

	ctx := context.Background()
	components := types.NewComponents()

	// Logger
	logger, err := log.New()
	s.Require().NoError(err, "while setting up logger")
	s.Require().NoError(logger.Start(ctx))
	s.logger = logger

	// Registry
	reg, err := registry.Provide(s.clientName, version, cfgData, components, logger)
	if err != nil {
		s.Require().NoError(err, "while creating a registry")
	}

	s.Require().NoError(reg.Start(ctx))
	s.registry = reg

	// Client
	c, err := client.Provide(s.clientName, cfgData, components, logger, reg)
	if err != nil {
		s.Require().NoError(err, "while creating a client")
	}

	s.Require().NoError(c.Start(ctx))
	s.client = c

	if len(s.Transports) == 0 {
		s.Transports = s.client.Config().PreferredTransports
	}

	// Start a server
	pro := []PackageRunnerOption{
		// WithNumProcesses(5),
		// WithRunEnv("GOMAXPROCS=1"),
		WithRunEnv("GOMAXPROCS=" + os.Getenv("GOMAXPROCS")),
		WithArgs("--config", "../../cmd/tests_server/config.yaml"),
	}
	// if logger.Level() <= slog.LevelDebug {
	pro = append(pro, WithStdOut(os.Stdout), WithStdErr(os.Stderr))
	// }

	s.serverRunner = NewPackageRunner(
		logger,
		"github.com/go-orb/plugins/client/tests/cmd/tests_server",
		"",
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

func (s *TestSuite) doRequest(ctx context.Context, req *TestRequest, clientWire client.Type) {
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

	if req.ContentType == "" || req.ContentType == codecs.MimeProto {
		rsp, err := client.Request[proto.CallResponse](
			ctx,
			clientWire,
			req.Service,
			req.Endpoint,
			req.Request,
			opts...,
		)

		if req.Error {
			s.Require().Error(err)
		} else {
			s.Require().NoError(err)
			s.Equal(req.Response.(*proto.CallResponse).GetMsg(), rsp.GetMsg(), "unexpected response") //nolint:errcheck
		}

		return
	}

	rsp, err := client.Request[map[string]any](
		ctx,
		clientWire,
		req.Service,
		req.Endpoint,
		req.Request,
		opts...,
	)

	if req.Error {
		s.Require().Error(err)
	} else {
		s.Require().NoError(err)
		s.Equal(req.Response.(map[string]any)["msg"], (*rsp)["msg"], "unexpected response") //nolint:errcheck
	}
}

// TestResolveServiceTransport checks if the right transport has been selected.
func (s *TestSuite) TestResolveServiceTransport() {
	ctx := context.Background()

	nodes, err := s.client.ResolveService(ctx, string(ServiceName), s.Transports...)
	s.Require().NoError(err)

	node, err := s.client.Config().Selector(ctx, string(ServiceName), nodes, s.Transports, false)
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

				node, err := s.client.Config().Selector(ctx, req.Service, nodes, s.Transports, false)
				s.Require().NoError(err)

				req.URL = fmt.Sprintf("%s://%s", node.Transport, node.Address)
			}

			s.doRequest(ctx, &req, s.client)
		})
	}
}

// TestFailingAuthorization tests an authorization call that must fail.
func (s *TestSuite) TestFailingAuthorization() {
	responseMd := make(map[string]string)
	ctx := context.Background()
	streamsClient := proto.NewStreamsClient(s.client)

	_, err := streamsClient.AuthorizedCall(
		ctx,
		string(ServiceName),
		&proto.CallRequest{Name: "empty"},
		client.WithResponseMetadata(responseMd),
	)
	s.Require().ErrorIs(err, orberrors.ErrUnauthorized)
}

// TestMetadata checks if metadata gets transported over the wire.
func (s *TestSuite) TestMetadata() {
	ctx := context.Background()
	ctx, md := metadata.WithOutgoing(ctx)
	md["authorization"] = "Bearer pleaseHackMe"

	responseMd := make(map[string]string)
	streamsClient := proto.NewStreamsClient(s.client)
	_, err := streamsClient.AuthorizedCall(
		ctx,
		string(ServiceName),
		&proto.CallRequest{Name: "empty"},
		client.WithResponseMetadata(responseMd),
	)
	s.Require().NoError(err)

	rspHandler, ok := responseMd["tracing-id"]
	s.Require().True(ok, "Transport does not transport metadata - tracing-id")
	s.Require().Equal("asfdjhladhsfashf", rspHandler)
}
