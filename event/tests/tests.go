// Package tests implements event tests.
package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-orb/go-orb/event"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	echopb "github.com/go-orb/plugins/event/tests/pb/echo"
	"github.com/stretchr/testify/suite"

	// Blank imports are fine.
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"
	_ "github.com/go-orb/plugins/log/slog"
)

// Suite is our testsuite.
type Suite struct {
	suite.Suite

	cancelRequestHandler context.CancelFunc

	logger  log.Logger
	handler event.Handler
}

// New creates a new testsuite.
func New(logger log.Logger, handler event.Handler) *Suite {
	return &Suite{
		logger:  logger,
		handler: handler,
	}
}

type requestHandler struct {
	logger log.Logger
}

func (rh *requestHandler) Echo(_ context.Context, req *echopb.Req) (*echopb.Resp, error) {
	if req.GetError() {
		return nil, orberrors.New(http.StatusBadRequest, "here's the error you asked for")
	}

	return &echopb.Resp{Payload: req.GetPayload()}, nil
}

func (rh *requestHandler) Auth(ctx context.Context, req *echopb.Req) (*echopb.Resp, error) {
	mdInc, ok := metadata.Incoming(ctx)
	if !ok {
		return nil, orberrors.ErrUnauthorized
	}

	if mdInc["authorization"] != "Bearer pleaseHackMe" {
		return nil, orberrors.ErrUnauthorized
	}

	md, ok := metadata.Outgoing(ctx)
	if ok {
		md["tracing-id"] = "asfdjhladhsfashf"
	}

	return &echopb.Resp{Payload: req.GetPayload()}, nil
}

// SetupSuite creates the handler.
func (s *Suite) SetupSuite() {
	reqHandler := &requestHandler{
		logger: s.logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	event.HandleRequest(ctx, s.handler, "echo", reqHandler.Echo)
	event.HandleRequest(ctx, s.handler, "auth", reqHandler.Auth)

	s.cancelRequestHandler = cancel

	time.Sleep(time.Second)
}

// TearDownSuite stops the handler.
func (s *Suite) TearDownSuite() {
	s.cancelRequestHandler()
}

// TestBadRequest tests a bad request.
func (s *Suite) TestBadRequest() {
	req := &echopb.Req{Error: true}
	_, err := event.Request[echopb.Resp](context.Background(), s.handler, "echo", req)
	s.Require().Error(err)
	s.Require().ErrorIs(err, orberrors.New(http.StatusBadRequest, "here's the error you asked for"))
}

// TestRequest tests a request.
func (s *Suite) TestRequest() {
	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	resp, err := event.Request[echopb.Resp](context.Background(), s.handler, "echo", req)
	s.Require().NoError(err)

	s.Require().Equal(payload, resp.GetPayload())
}

// TestAuthorizedRequest tests an authorized request.
func (s *Suite) TestAuthorizedRequest() {
	ctx := context.Background()
	ctx, md := metadata.WithOutgoing(ctx)
	md["authorization"] = "Bearer pleaseHackMe"

	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	responseMd := make(map[string]string)
	resp, err := event.Request[echopb.Resp](ctx, s.handler, "auth", req, event.WithRequestResponseMetadata(responseMd))
	s.Require().NoError(err)

	s.Require().Equal(payload, resp.GetPayload())

	rspHandler, ok := responseMd["tracing-id"]
	s.Require().True(ok, "Transport does not transport metadata - tracing-id")
	s.Require().Equal("asfdjhladhsfashf", rspHandler)
}

// TestFailingAuthorization tests an authorization call that must fail.
func (s *Suite) TestFailingAuthorization() {
	responseMd := make(map[string]string)
	ctx := context.Background()

	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	_, err := event.Request[echopb.Resp](ctx, s.handler, "auth", req, event.WithRequestResponseMetadata(responseMd))
	s.Require().ErrorIs(err, orberrors.ErrUnauthorized)
}

// BenchmarkRequest runs a benchmark on requests.
func (s *Suite) BenchmarkRequest(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	b.StartTimer()

	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	for n := 0; n < b.N; n++ {
		resp, err := event.Request[echopb.Resp](context.Background(), s.handler, "echo", req)
		s.Require().NoError(err)
		s.Require().Equal(payload, resp.GetPayload())
	}

	b.StopTimer()
	s.TearDownSuite()
}
