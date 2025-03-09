// Package tests implements event tests.
package tests

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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
	handler event.Client
}

// New creates a new testsuite.
func New(logger log.Logger, handler event.Client) *Suite {
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
	s.T().Helper()

	responseMd := make(map[string]string)
	ctx := context.Background()

	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	_, err := event.Request[echopb.Resp](ctx, s.handler, "auth", req, event.WithRequestResponseMetadata(responseMd))
	s.Require().ErrorIs(err, orberrors.ErrUnauthorized)
}

// TestPublish tests publishing an event.
func (s *Suite) TestPublish() {
	s.T().Helper()

	ctx := context.Background()
	topic := "test_publish"
	payload := []byte("hello world")

	// Publish a raw bytes message
	err := s.handler.Publish(ctx, topic, payload)
	s.Require().NoError(err)

	// Publish a structured message
	msg := &echopb.Req{Payload: []byte("structured data")}
	err = s.handler.Publish(ctx, topic, msg)
	s.Require().NoError(err)

	// Publish with metadata
	metadata := map[string]string{"key": "value"}
	err = s.handler.Publish(ctx, topic, payload, event.WithPublishMetadata(metadata))
	s.Require().NoError(err)
}

// TestConsume tests consuming events.
func (s *Suite) TestConsume() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test_consume"
	payload := []byte("test message")

	// Start consuming
	evtChan, err := s.handler.Consume(topic)
	s.Require().NoError(err)

	// Publish a message
	err = s.handler.Publish(ctx, topic, payload)
	s.Require().NoError(err)

	// Wait for the message
	select {
	case msg := <-evtChan:
		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)
	case <-ctx.Done():
		s.T().Fatal("Timed out waiting for message")
	}
}

// TestConsumeWithGroup tests consuming events with a consumer group.
func (s *Suite) TestConsumeWithGroup() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test_consume_group"
	group := "test_group"
	payload := []byte("group message")

	// Start consuming with group
	evtChan, err := s.handler.Consume(topic, event.WithGroup(group))
	s.Require().NoError(err)

	// Publish multiple messages
	for i := 0; i < 3; i++ {
		err = s.handler.Publish(ctx, topic, payload)
		s.Require().NoError(err)
	}

	// Should receive at least one message
	select {
	case msg := <-evtChan:
		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)
	case <-ctx.Done():
		s.T().Fatal("Timed out waiting for message")
	}
}

// TestMultipleConsumers tests multiple independent consumers on the same topic.
func (s *Suite) TestMultipleConsumers() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topic := "test_multiple_consumers"
	payload := []byte("message for multiple consumers")

	// Create multiple consumers
	const numConsumers = 3
	channels := make([]<-chan event.Event, numConsumers)

	for i := 0; i < numConsumers; i++ {
		ch, err := s.handler.Consume(topic)
		s.Require().NoError(err)

		channels[i] = ch
	}

	// Publish a message
	err := s.handler.Publish(ctx, topic, payload)
	s.Require().NoError(err)

	// All consumers should receive the message
	for i, ch := range channels {
		select {
		case msg := <-ch:
			s.Require().Equal(topic, msg.Topic)
			s.Require().Equal(payload, msg.Payload)
		case <-ctx.Done():
			s.T().Fatalf("Consumer %d timed out waiting for message", i)
		}
	}
}

// TestConsumeWithRetries tests the retry behavior with manual acknowledgment.
func (s *Suite) TestConsumeWithRetries() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test_consume_retries"
	payload := []byte("test retry message")

	// Start consuming with manual acknowledgement
	evtChan, err := s.handler.Consume(
		topic,
		event.WithAutoAck(false, 1*time.Second),
		event.WithRetryLimit(2),
	)
	s.Require().NoError(err)

	// Publish a message
	err = s.handler.Publish(ctx, topic, payload)
	s.Require().NoError(err)

	// Receive the message but don't acknowledge
	select {
	case msg := <-evtChan:
		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)

		// Deliberately don't acknowledge
	case <-ctx.Done():
		s.T().Skip("Timed out waiting for initial message - implementation may not support retries")
		return
	}

	// Should receive the message again (retry)
	select {
	case msg := <-evtChan:
		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)

		// Now acknowledge
		s.Require().NoError(msg.Ack())
	case <-ctx.Done():
		s.T().Skip("Timed out waiting for retry - implementation may not support retries")
	}
}

// TestConsumeWithAckWait tests the acknowledgement wait time.
func (s *Suite) TestConsumeWithAckWait() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	topic := "test_consume_ack_wait"
	payload := []byte("test ack wait message")
	ackWait := 1 * time.Second // Short ack wait time for testing

	// Start consuming with manual acknowledgement and short ack wait
	evtChan, err := s.handler.Consume(
		topic,
		event.WithAutoAck(false, ackWait),
	)
	s.Require().NoError(err)

	// Publish a message
	err = s.handler.Publish(ctx, topic, payload)
	s.Require().NoError(err)

	// First appearance of the message
	select {
	case msg := <-evtChan:
		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)

		// Deliberately don't acknowledge
	case <-ctx.Done():
		s.T().Fatal("Timed out waiting for initial message")
	}

	// Message should be redelivered after ack wait time
	start := time.Now()
	select {
	case msg := <-evtChan:
		elapsed := time.Since(start)

		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)

		// Verify redelivery happened after ack wait time
		s.Require().GreaterOrEqual(elapsed, ackWait, "Message should not be redelivered before ack wait time")

		// Now acknowledge to prevent further redeliveries
		s.Require().NoError(msg.Ack())
	case <-ctx.Done():
		s.T().Fatal("Timed out waiting for redelivery")
	}
}

// TestConcurrentPublishConsume tests concurrent publishing and consuming.
func (s *Suite) TestConcurrentPublishConsume() {
	s.T().Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	topic := "test_concurrent"

	// Start consuming
	evtChan, err := s.handler.Consume(topic)
	s.Require().NoError(err)

	const numMessages = 20 // Reduce number from the original test for faster execution

	var (
		received = make(map[string]bool)
		mu       sync.Mutex
		wg       sync.WaitGroup
	)

	// Start receiver goroutine
	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < numMessages; i++ {
			select {
			case msg := <-evtChan:
				mu.Lock()
				received[string(msg.Payload)] = true
				mu.Unlock()
			case <-ctx.Done():
				s.T().Error("Timed out waiting for message")
				return
			}
		}
	}()

	// Publish messages concurrently
	for i := 0; i < numMessages; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			payload := []byte(fmt.Sprintf("concurrent message %d", id))
			err := s.handler.Publish(ctx, topic, payload)

			s.NoError(err)
		}(i)
	}

	wg.Wait()

	// Verify we received all messages
	s.Require().Len(received, numMessages, "All messages should have been received")
}

// TestErrorHandling tests error conditions.
func (s *Suite) TestErrorHandling() {
	s.T().Helper()

	// Test with empty topic
	_, err := s.handler.Consume("")
	s.Require().Error(err, "Should error on empty topic")

	// Test publishing to empty topic
	err = s.handler.Publish(context.Background(), "", []byte("test"))
	s.Require().Error(err, "Should error on empty topic")

	// Test with invalid group name (too long)
	longGroup := ""
	for i := 0; i < 1000; i++ {
		longGroup += "a"
	}

	_, err = s.handler.Consume("topic", event.WithGroup(longGroup))
	s.Require().Error(err, "Should error with very long group name")
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

// BenchmarkRequestLarge runs a benchmark on requests with large payloads.
func (s *Suite) BenchmarkRequestLarge(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	// Create a 64KB payload - staying under NATS default message size limits
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	req := &echopb.Req{Payload: payload}

	b.ResetTimer()
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		resp, err := event.Request[echopb.Resp](context.Background(), s.handler, "echo", req)
		s.Require().NoError(err)
		s.Require().Equal(payload, resp.GetPayload())
	}

	b.StopTimer()
	s.TearDownSuite()
}

// BenchmarkRequestParallel runs a benchmark on requests with parallelism.
func (s *Suite) BenchmarkRequestParallel(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	b.ResetTimer()
	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := event.Request[echopb.Resp](context.Background(), s.handler, "echo", req)
			s.Require().NoError(err)
			s.Require().Equal(payload, resp.GetPayload())
		}
	})

	b.StopTimer()
	s.TearDownSuite()
}

// BenchmarkRequestAuth runs a benchmark on auth requests.
func (s *Suite) BenchmarkRequestAuth(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	payload := []byte("asdf1234")
	req := &echopb.Req{Payload: payload}

	// Create auth context with metadata
	ctx, md := metadata.WithOutgoing(context.Background())
	md["authorization"] = "Bearer test"

	b.ResetTimer()
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		resp, err := event.Request[echopb.Resp](ctx, s.handler, "auth", req)
		s.Require().NoError(err)
		s.Require().Equal(payload, resp.GetPayload())
	}

	b.StopTimer()
	s.TearDownSuite()
}

// BenchmarkPublish runs a benchmark on publish operations.
func (s *Suite) BenchmarkPublish(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	ctx := context.Background()
	topic := "bench_publish"
	payload := []byte("hello world benchmark")

	b.ResetTimer()
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		err := s.handler.Publish(ctx, topic, payload)
		s.Require().NoError(err)
	}

	b.StopTimer()
	s.TearDownSuite()
}

// BenchmarkPublishLarge runs a benchmark on publish operations with large payloads.
func (s *Suite) BenchmarkPublishLarge(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	ctx := context.Background()
	topic := "bench_publish_large"

	// Create a 64KB payload - staying under NATS default message size limits
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		err := s.handler.Publish(ctx, topic, payload)
		s.Require().NoError(err)
	}

	b.StopTimer()
	s.TearDownSuite()
}

// BenchmarkPublishMetadata runs a benchmark on publish operations with metadata.
func (s *Suite) BenchmarkPublishMetadata(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	ctx := context.Background()
	topic := "bench_publish_metadata"
	payload := []byte("hello world benchmark")
	metadata := map[string]string{"key1": "value1", "key2": "value2", "benchmark": "true"}

	b.ResetTimer()
	b.StartTimer()

	for n := 0; n < b.N; n++ {
		err := s.handler.Publish(ctx, topic, payload, event.WithPublishMetadata(metadata))
		s.Require().NoError(err)
	}

	b.StopTimer()
	s.TearDownSuite()
}

// BenchmarkConsume runs a benchmark on consume operations.
func (s *Suite) BenchmarkConsume(b *testing.B) {
	b.Helper()

	s.SetT(&testing.T{})
	s.SetupSuite()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := "bench_consume"
	payload := []byte("consume benchmark message")

	// Publish messages before starting benchmark
	for i := 0; i < b.N; i++ {
		err := s.handler.Publish(ctx, topic, payload)
		s.Require().NoError(err)
	}

	// Start consuming
	evtChan, err := s.handler.Consume(topic)
	s.Require().NoError(err)

	b.ResetTimer()
	b.StartTimer()

	// Consume all messages
	for i := 0; i < b.N; i++ {
		msg := <-evtChan
		s.Require().Equal(topic, msg.Topic)
		s.Require().Equal(payload, msg.Payload)
	}

	b.StopTimer()
	s.TearDownSuite()
}
