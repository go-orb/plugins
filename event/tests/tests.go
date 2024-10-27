// Package tests implements event tests.
package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-orb/go-orb/event"
	echopb "github.com/go-orb/plugins/event/tests/pb/echo"
	"github.com/stretchr/testify/suite"
)

// Suite is our testsuite.
type Suite struct {
	suite.Suite

	cancelRequestHandler context.CancelFunc

	handler event.Handler
}

// New creates a new testsuite.
func New(handler event.Handler) *Suite {
	return &Suite{
		handler: handler,
	}
}

func (s *Suite) echoHandler(_ context.Context, req *echopb.Req) (*echopb.Resp, error) {
	if req.GetError() {
		return nil, errors.New("here's the error you asked for")
	}

	return &echopb.Resp{Payload: req.GetPayload()}, nil
}

// SetupSuite creates the handler.
func (s *Suite) SetupSuite() {
	ctx, cancel := context.WithCancel(context.Background())
	event.HandleRequest(ctx, s.handler, "echo", s.echoHandler)

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
}

// TestRequest tests a request.
func (s *Suite) TestRequest() {
	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	resp, err := event.Request[echopb.Resp](context.Background(), s.handler, "echo", req)
	s.Require().NoError(err)

	s.Require().Equal(payload, resp.GetPayload())
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
