// Package tests contains tests for go-orb/plugins/client/*.
package tests

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/tests/proto/echo"
	"github.com/go-orb/plugins/client/tests/proto/file"
	"github.com/stretchr/testify/suite"
)

//nolint:gochecknoglobals
var (
	// ServiceName is the name of the testing service.
	ServiceName = types.ServiceName("service")

	// DefaultRequests is the list of default requests.
	DefaultRequests = []TestRequest{
		{
			Name:     "32byte",
			Service:  string(ServiceName),
			Endpoint: echo.EndpointStreamsCall,
			Request: &echo.CallRequest{
				Name: "32byte",
			},
			Response: &echo.CallResponse{
				Msg: "",
			},
		},
		{
			Name:        "raw-json",
			Service:     string(ServiceName),
			Endpoint:    echo.EndpointStreamsCall,
			ContentType: "application/json",
			Request:     `{"name": "Alex"}`,
			Response: map[string]any{
				"msg": "Hello Alex",
			},
		},
		{
			Name:     "default codec with URL",
			Service:  string(ServiceName),
			Endpoint: echo.EndpointStreamsCall,
			Request: &echo.CallRequest{
				Name: "Alex",
			},
			Response: &echo.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:     "default codec",
			Service:  string(ServiceName),
			Endpoint: echo.EndpointStreamsCall,
			Request: &echo.CallRequest{
				Name: "Alex",
			},
			Response: &echo.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:        "proto",
			Service:     string(ServiceName),
			Endpoint:    echo.EndpointStreamsCall,
			ContentType: "application/x-protobuf",
			Request: &echo.CallRequest{
				Name: "Alex",
			},
			Response: &echo.CallResponse{
				Msg: "Hello Alex",
			},
		},
		{
			Name:        "json",
			Service:     string(ServiceName),
			Endpoint:    echo.EndpointStreamsCall,
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
			Service:  string(ServiceName),
			Endpoint: echo.EndpointStreamsCall,
			Error:    true,
			Request: &echo.CallRequest{
				Name: "error",
			},
			Response: &echo.CallResponse{
				Msg: "Hello Alex",
			},
		},
	}
)

// SetupData contains the setup data for a test.
type SetupData struct {
	Logger      log.Logger
	Registry    registry.Type
	Entrypoints []server.Entrypoint
	Ctx         context.Context
	Stop        context.CancelFunc
}

// TestSuite runs a bunch of tests.
type TestSuite struct {
	suite.Suite

	// Transports is the list of preferred transports for all requests
	Transports []string

	// Requests is the requests to make.
	Requests []TestRequest

	logger   log.Logger
	registry registry.Type
	client   client.Type

	entrypoints []server.Entrypoint
	ctx         context.Context
	setupServer func(service types.ServiceName) (*SetupData, error)
	stopServer  context.CancelFunc

	// To create more clients in Benchmarks.
	clientName types.ServiceName
}

// NewSuite creates a new test suite.
func NewSuite(setupServer func(service types.ServiceName) (*SetupData, error), transports []string, requests ...TestRequest) *TestSuite {
	s := new(TestSuite)

	s.Transports = transports
	s.setupServer = setupServer

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
	// Expect an error?
	Error bool

	Request  any
	Response any
}

// SetupSuite setups the test suite.
func (s *TestSuite) SetupSuite() {
	var err error

	setupData, err := s.setupServer(ServiceName)
	if err != nil {
		s.Require().NoError(err, "while setting up the server")
	}

	s.logger = setupData.Logger
	s.registry = setupData.Registry
	s.entrypoints = setupData.Entrypoints
	s.ctx = setupData.Ctx
	s.stopServer = setupData.Stop

	s.clientName = types.ServiceName("client")

	s.client, err = client.ProvideNoOpts(s.clientName, nil, &types.Components{}, s.logger, s.registry)
	s.Require().NoError(err, "while setting up the client")

	s.Require().NoError(s.logger.Start(s.ctx))
	s.Require().NoError(s.registry.Start(s.ctx))

	for _, ep := range s.entrypoints {
		s.Require().NoError(ep.Start(s.ctx))
	}

	s.Require().NoError(s.client.Start(s.ctx))
}

// TearDownSuite runs after all tests.
func (s *TestSuite) TearDownSuite() {
	s.stopServer()

	ctx := context.Background()

	s.Require().NoError(s.client.Stop(ctx), "while stopping the client")

	s.Require().NoError(s.registry.Stop(ctx))

	for _, ep := range s.entrypoints {
		s.Require().NoError(ep.Stop(ctx))
	}

	s.Require().NoError(s.logger.Stop(ctx), "while stopping the logger")
}

func (s *TestSuite) doRequest(ctx context.Context, req *TestRequest, clientWire client.Type, transport string) {
	opts := []client.CallOption{}
	if req.ContentType != "" {
		opts = append(opts, client.WithContentType(req.ContentType))
	}

	if len(s.Transports) != 0 {
		opts = append(opts, client.WithPreferredTransports(transport))
	}

	if req.ContentType == "" || req.ContentType == codecs.MimeProto {
		rsp, err := client.Request[echo.CallResponse](
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
			s.Equal(req.Response.(*echo.CallResponse).GetMsg(), rsp.GetMsg(), "unexpected response") //nolint:errcheck
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

// TestRunRequests makes the configured requests.
func (s *TestSuite) TestRunRequests() {
	for _, t := range s.Transports {
		for _, oReq := range s.Requests {
			ctx := context.Background()
			req := oReq

			s.Run(fmt.Sprintf("%s/%s", t, req.Name), func() {
				s.doRequest(ctx, &req, s.client, t)
			})
		}
	}
}

// TestFailingAuthorization tests an authorization call that must fail.
func (s *TestSuite) TestFailingAuthorization() {
	responseMd := make(map[string]string)
	ctx := context.Background()
	streamsClient := echo.NewStreamsClient(s.client)

	_, err := streamsClient.AuthorizedCall(
		ctx,
		string(ServiceName),
		&echo.CallRequest{Name: "empty"},
		client.WithResponseMetadata(responseMd),
	)
	s.Require().ErrorIs(err, orberrors.ErrUnauthorized)
}

// TestMetadata checks if metadata gets transported over the wire.
func (s *TestSuite) TestMetadata() {
	md := make(map[string]string)
	md["authorization"] = "Bearer pleaseHackMe"

	for _, t := range s.Transports {
		s.Run(t, func() {
			responseMd := make(map[string]string)
			streamsClient := echo.NewStreamsClient(s.client)
			_, err := streamsClient.AuthorizedCall(
				context.Background(),
				string(ServiceName),
				&echo.CallRequest{Name: "empty"},
				client.WithMetadata(md),
				client.WithResponseMetadata(responseMd),
			)
			s.Require().NoError(err)

			rspHandler, ok := responseMd["tracing-id"]
			s.Require().True(ok, "Transport does not transport metadata - tracing-id")
			s.Require().Equal("asfdjhladhsfashf", rspHandler)
		})
	}
}

// TestFileUpload tests the client streaming functionality for file uploads.
func (s *TestSuite) TestFileUpload() {
	// Create a file service client
	fileClient := file.NewFileServiceClient(s.client)

	for _, t := range s.Transports {
		s.Run(t, func() {
			// Create a context for the stream
			ctx := context.Background()

			// Open a stream to the service
			stream, err := fileClient.UploadFile(ctx, string(ServiceName), client.WithPreferredTransports(t))
			if errors.Is(err, orberrors.ErrNotImplemented) {
				// Transport does not support streaming.
				return
			}

			s.Require().NoError(err, "Failed to open stream")

			// Send multiple chunks of data
			chunkCount := 5
			for i := 0; i < chunkCount; i++ {
				// Create test data
				data := make([]byte, 1024) // 1KB chunks
				_, err := rand.Read(data)
				s.Require().NoError(err, "Failed to generate random data")

				// Send the chunk
				chunk := &file.FileChunk{
					Filename:    fmt.Sprintf("test-file-%d.bin", i),
					ContentType: "application/octet-stream",
					Data:        data,
				}

				err = stream.Send(chunk)
				s.Require().NoError(err, "Failed to send chunk")
			}

			// Close the stream to tell the server we're done sending data
			// This will signal EOF to the server
			err = stream.CloseSend()
			s.Require().NoError(err, "Failed to close send stream")

			// Get the response
			response := file.UploadResponse{}
			err = stream.Recv(&response)
			s.Require().NoError(err, "Failed to receive response")
			s.Require().NotEmpty(response.GetId(), "Response ID should not be empty")
			s.Require().True(response.GetSuccess(), "Upload should be successful")

			err = stream.Close()
			s.Require().NoError(err, "Failed to close stream")
		})
	}
}

// TestAuthorizedFileUpload tests the authorized client streaming functionality for file uploads.
func (s *TestSuite) TestAuthorizedFileUpload() {
	// Create a file service client
	fileClient := file.NewFileServiceClient(s.client)

	for _, t := range s.Transports {
		s.Run(t+"/Unauthorized", func() {
			// Create a context for the stream
			ctx := context.Background()

			// Open a stream to the service
			stream, err := fileClient.AuthorizedUploadFile(ctx, string(ServiceName), client.WithPreferredTransports(t))
			if errors.Is(err, orberrors.ErrNotImplemented) {
				// Transport does not support streaming.
				return
			}

			s.Require().NoError(err, "Failed to open stream")

			// Create test data
			data := make([]byte, 1024) // 1KB chunk
			_, err = rand.Read(data)
			s.Require().NoError(err, "Failed to generate random data")

			// Send the chunk
			chunk := &file.FileChunk{
				Filename:    "test-file.bin",
				ContentType: "application/octet-stream",
				Data:        data,
			}

			err = stream.Send(chunk)
			s.Require().NoError(err, "Failed to send initial chunk")

			// Close the stream - this will trigger the server to process the request
			err = stream.CloseSend()
			s.Require().NoError(err, "Failed to close stream")

			// Try to receive response, which should fail with unauthorized error
			var response file.UploadResponse
			err = stream.Recv(&response)
			s.Require().Error(err, "Should fail with unauthorized error")
			s.Require().ErrorIs(err, orberrors.ErrUnauthorized, "Should be an unauthorized error")
		})
	}

	for _, t := range s.Transports {
		s.Run(t+"/Authorized", func() {
			// Track response metadata
			md := make(map[string]string)
			md["authorization"] = "Bearer pleaseHackMe"
			responseMd := make(map[string]string)

			// Open a stream to the service
			stream, err := fileClient.AuthorizedUploadFile(context.Background(), string(ServiceName),
				client.WithPreferredTransports(t),
				client.WithMetadata(md),
				client.WithResponseMetadata(responseMd),
			)
			if errors.Is(err, orberrors.ErrNotImplemented) {
				// Transport does not support streaming.
				return
			}

			s.Require().NoError(err, "Failed to open stream")

			// Send multiple chunks of data
			chunkCount := 3
			for i := 0; i < chunkCount; i++ {
				// Create test data
				data := make([]byte, 1024) // 1KB chunks
				_, err := rand.Read(data)
				s.Require().NoError(err, "Failed to generate random data")

				// Send the chunk
				chunk := &file.FileChunk{
					Filename:    fmt.Sprintf("test-file-%d.bin", i),
					ContentType: "application/octet-stream",
					Data:        data,
				}

				err = stream.Send(chunk)
				s.Require().NoError(err, "Failed to send chunk")
			}

			// Close the stream
			err = stream.CloseSend()
			s.Require().NoError(err, "Failed to close stream")

			// Get the response
			var response file.UploadResponse
			err = stream.Recv(&response)
			s.Require().NoError(err, "Failed to receive response")
			s.Require().NotEmpty(response.GetId(), "Response ID should not be empty")
			s.Require().True(response.GetSuccess(), "Upload should be successful")

			// Verify response metadata
			s.Require().Equal("true", responseMd["bytes-received"], "Expected bytes-received metadata")
			s.Require().Equal("completed", responseMd["total-size"], "Expected total-size metadata")
		})
	}
}
