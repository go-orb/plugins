// Package natsjs provides the nats jetstream event client for go-orb.
package natsjs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-orb/go-orb/cli"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/event"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/go-orb/util/ultrapool"
	"github.com/go-orb/plugins/event/natsjs/pb"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// This is here to make sure it implements event.Events.
var _ event.Client = (*NatsJS)(nil)

// NatsJS is the nats jetstream event client for go-orb.
type NatsJS struct {
	serviceName string

	config Config
	logger log.Logger

	nc           *nats.Conn
	js           jetstream.JetStream
	ncjs         nats.JetStreamContext
	requestCodec codecs.Marshaler
	publishCodec codecs.Marshaler

	evReqPool container.Pool[*event.Req[[]byte, []byte]]
	reqPool   container.Pool[*pb.Request]
	replyPool container.Pool[*pb.Reply]

	consumers *container.SafeMap[string, jetstream.ConsumeContext]
}

// New creates a new NATS registry. This functions should rarely be called manually.
// To create a new registry use Provide.
func New(serviceName string, cfg Config, log log.Logger) (*NatsJS, error) {
	requestCodec, err := codecs.GetMime(cfg.RequestCodec)
	if err != nil {
		return nil, err
	}

	publishCodec, err := codecs.GetMime(cfg.PublishCodec)
	if err != nil {
		return nil, err
	}

	return &NatsJS{
		serviceName:  serviceName,
		config:       cfg,
		logger:       log,
		requestCodec: requestCodec,
		publishCodec: publishCodec,

		evReqPool: container.NewPool(func() *event.Req[[]byte, []byte] { return &event.Req[[]byte, []byte]{} }),
		reqPool:   container.NewPool(func() *pb.Request { return &pb.Request{} }),
		replyPool: container.NewPool(func() *pb.Reply { return &pb.Reply{} }),

		consumers: container.NewSafeMap[string, jetstream.ConsumeContext](),
	}, nil
}

// Clone creates a clone of the handler, this is useful for parallel requests.
func (n *NatsJS) Clone() event.Type {
	return event.Type{Client: &NatsJS{
		serviceName:  n.serviceName,
		config:       n.config,
		logger:       n.logger,
		requestCodec: n.requestCodec,
		publishCodec: n.publishCodec,

		evReqPool: container.NewPool(func() *event.Req[[]byte, []byte] { return &event.Req[[]byte, []byte]{} }),
		reqPool:   container.NewPool(func() *pb.Request { return &pb.Request{} }),
		replyPool: container.NewPool(func() *pb.Reply { return &pb.Reply{} }),

		consumers: container.NewSafeMap[string, jetstream.ConsumeContext](),
	}}
}

// GetPublishCodec returns the codec used by the handler for publish.
func (n *NatsJS) GetPublishCodec() codecs.Marshaler {
	return n.publishCodec
}

// Publish a message to a topic.
func (n *NatsJS) Publish(ctx context.Context, topic string, msg any, opts ...event.PublishOption) error {
	// validate the topic
	if len(topic) == 0 {
		return event.ErrMissingTopic
	}

	// parse the options
	options := event.NewPublishOptions(opts...)

	// encode the message if it's not already encoded
	var payload []byte
	if p, ok := msg.([]byte); ok {
		payload = p
	} else {
		p, err := json.Marshal(msg)
		if err != nil {
			return orberrors.ErrInternalServerError.Wrap(event.ErrEncodingMessage)
		}

		payload = p
	}

	// construct the event
	event := &event.Event{
		ID:        uuid.New().String(),
		Topic:     topic,
		Timestamp: options.Timestamp,
		Metadata:  options.Metadata,
		Payload:   payload,
	}

	// serialize the event to bytes
	bytes, err := json.Marshal(event)
	if err != nil {
		return orberrors.ErrInternalServerError.Wrap(fmt.Errorf("encoding event: %w", err))
	}

	// publish the event to the topic's channel
	// publish synchronously if configured
	if n.config.SyncPublish {
		_, err := n.js.Publish(ctx, event.Topic, bytes)
		if err != nil {
			err = orberrors.ErrInternalServerError.Wrap(fmt.Errorf("publishing message to topic: %w", err))
		}

		return err
	}

	// publish asynchronously by default
	if _, err := n.js.PublishAsync(event.Topic, bytes); err != nil {
		return orberrors.ErrInternalServerError.Wrap(fmt.Errorf("publishing message to topic: %w", err))
	}

	return nil
}

// Consume from a topic.
//
//nolint:funlen,gocyclo,gocognit,cyclop
func (n *NatsJS) Consume(topic string, opts ...event.ConsumeOption) (<-chan event.Event, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, orberrors.ErrInternalServerError.Wrap(event.ErrMissingTopic)
	}

	// parse the options
	options := event.NewConsumeOptions(opts...)

	// Create the channel for events that will be returned
	channel := make(chan event.Event)

	// Setup the message handler
	handleMsg := func(msg *nats.Msg) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		// decode the message
		var evt event.Event
		if err := n.publishCodec.Unmarshal(msg.Data, &evt); err != nil {
			n.logger.Error("decoding message", "error", err)
			// not acknowledging the message is the way to indicate an error occurred
			return
		}

		// Set the handler for unmarshaling the event.
		evt.Handler = n

		if !options.AutoAck {
			// set up the ack funcs
			evt.SetAckFunc(func() error {
				return msg.Ack()
			})
			evt.SetNackFunc(func() error {
				return msg.Nak()
			})
		}

		// push onto the channel and wait for the consumer to take the event off before we acknowledge it.
		channel <- evt

		if !options.AutoAck {
			return
		}

		if err := msg.Ack(nats.Context(ctx)); err != nil {
			n.logger.Error("acknowledging message", "error", err)
		}
	}

	// ensure that a stream exists for that topic
	_, err := n.ncjs.StreamInfo(topic)
	if err != nil {
		// Create a stream with the topic name
		cfg := &nats.StreamConfig{
			Name:     topic,
			Subjects: []string{topic},
		}

		_, err = n.ncjs.AddStream(cfg)
		if err != nil {
			return nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("stream did not exist and adding a stream failed: %w", err))
		}
	}

	// Create a unique consumer identifier for tracking
	consumerID := uuid.New().String()

	// If using a consumer group, create a durable consumer first
	//nolint:nestif
	if options.Group != "" {
		// Create a push-based durable consumer for the group
		consumerConfig := &nats.ConsumerConfig{
			Durable:        options.Group,
			Name:           options.Group,
			DeliverGroup:   options.Group,
			FilterSubject:  topic,
			DeliverSubject: "_INBOX." + consumerID,
			AckPolicy:      nats.AckExplicitPolicy,
			DeliverPolicy:  nats.DeliverNewPolicy,
		}

		// Add additional config options based on our ConsumeOptions
		if options.CustomRetries {
			consumerConfig.MaxDeliver = options.GetRetryLimit()
		}

		if options.AckWait > 0 {
			consumerConfig.AckWait = options.AckWait
		}

		if !options.Offset.IsZero() {
			consumerConfig.OptStartTime = &options.Offset
		}

		// Check if consumer already exists
		consumerInfo, err := n.ncjs.ConsumerInfo(topic, options.Group)
		if err != nil {
			// Consumer doesn't exist, create it
			_, err = n.ncjs.AddConsumer(topic, consumerConfig)
			if err != nil {
				return nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("failed to create consumer: %w", err))
			}
		} else {
			// Use the existing consumer's deliver subject
			consumerConfig.DeliverSubject = consumerInfo.Config.DeliverSubject
		}

		// Now subscribe to the delivery subject with the queue group
		_, err = n.nc.QueueSubscribe(consumerConfig.DeliverSubject, options.Group, handleMsg)
		if err != nil {
			return nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("subscribing to delivery subject with group: %w", err))
		}
	} else {
		// For regular consumers (non-grouped), use a simpler ephemeral consumer
		subOpts := []nats.SubOpt{
			nats.ConsumerName(consumerID),
			nats.DeliverNew(),
		}

		// Configure ack policy
		if options.AutoAck {
			subOpts = append(subOpts, nats.AckNone())
		} else {
			subOpts = append(subOpts, nats.AckExplicit())
		}

		// Configure other options
		if options.CustomRetries {
			subOpts = append(subOpts, nats.MaxDeliver(options.GetRetryLimit()))
		}

		if !options.Offset.IsZero() {
			subOpts = append(subOpts, nats.StartTime(options.Offset))
		}

		if options.AckWait > 0 {
			subOpts = append(subOpts, nats.AckWait(options.AckWait))
		}

		// Create a regular subscription to the topic
		sub, err := n.ncjs.Subscribe(topic, handleMsg, subOpts...)
		if err != nil {
			return nil, orberrors.ErrInternalServerError.Wrap(fmt.Errorf("subscribing to topic: %w", err))
		}

		// Store the subscription for cleanup
		_, err = sub.ConsumerInfo()
		if err != nil {
			n.logger.Debug("Could not get consumer info for subscription", "error", err)
		}

		// Store this subscription in a map entry
		n.consumers.Set(topic+"."+consumerID, nil)
	}

	return channel, nil
}

// Request runs a REST like call on the given topic.
func (n *NatsJS) Request(
	ctx context.Context,
	req *event.Req[[]byte, any],
	opts ...event.RequestOption,
) ([]byte, error) {
	// validate the topic
	if len(req.Topic) == 0 {
		return nil, orberrors.ErrInternalServerError.Wrap(event.ErrMissingTopic)
	}

	if req.Err != nil {
		return nil, orberrors.ErrInternalServerError.Wrap(req.Err)
	}

	options := event.NewRequestOptions(opts...)

	pbReq := &pb.Request{}
	defer n.reqPool.Put(pbReq)
	pbReq.Reset()

	pbReq.Data = req.Data
	pbReq.ContentType = req.ContentType

	// Handle metadata from the client to the server/handler.
	if md, ok := metadata.Outgoing(ctx); ok {
		pbReq.Metadata = md
	} else {
		pbReq.Metadata = make(map[string]string)
	}

	data, err := n.requestCodec.Marshal(pbReq)
	if err != nil {
		n.logger.Error("while encoding the message", "topic", req.Topic, "err", err, "data", data)
		return nil, fmt.Errorf("while encoding the message: %w", err)
	}

	// Send the request and wait for a reply.
	msg, err := n.nc.Request(req.Topic, data, options.RequestTimeout)
	if err != nil {
		n.logger.Error("while publishing a request", "topic", req.Topic, "err", err)
		return nil, err
	}

	reply := n.replyPool.Get()
	defer n.replyPool.Put(reply)
	reply.Reset()

	err = n.requestCodec.Unmarshal(msg.Data, reply)
	if err != nil {
		n.logger.Error("while decoding the reply", "topic", req.Topic, "err", err, "data", msg.Data)
		return nil, err
	}

	// Handle metadata from the server to the client.
	for k, v := range reply.GetMetadata() {
		options.Metadata[k] = v
	}

	if reply.GetCode() != http.StatusOK {
		return nil, orberrors.New(int(reply.GetCode()), reply.GetMessage())
	}

	return reply.GetData(), nil
}

// HandleRequest subscribes to the given topic and handles the requests.
//
//nolint:funlen,gocognit
func (n *NatsJS) HandleRequest(
	ctx context.Context,
	topic string,
	callbackHandler func(context.Context, *event.Req[[]byte, []byte]),
) {
	// validate the topic
	if len(topic) == 0 {
		n.logger.Error("can't handle", "error", event.ErrMissingTopic)
		return
	}

	wPool := ultrapool.NewWorkerPool(func(task ultrapool.Task) {
		msg, ok := task.(*nats.Msg)
		if !ok {
			return
		}

		replyFunc := func(ctx context.Context, result []byte, inErr error) {
			reply := n.replyPool.Get()
			defer n.replyPool.Put(reply)
			reply.Reset()

			// Handle metadata coming from the handler/server to the client.
			if md, ok := metadata.Outgoing(ctx); ok {
				reply.Metadata = md
			} else {
				reply.Metadata = make(map[string]string)
			}

			reply.Data = result

			if inErr != nil {
				orbE := orberrors.From(inErr)
				reply.Code = int32(orbE.Code) //nolint:gosec
				reply.Message = orbE.Error()
			} else {
				reply.Code = http.StatusOK
			}

			b, err := n.requestCodec.Marshal(reply)
			if err != nil {
				n.logger.Error("failed to encode reply, error was", "err", err)
				return
			}

			if err = msg.Respond(b); err != nil {
				n.logger.Error("failed to send reply, error was", "err", err)
				return
			}
		}

		req := n.reqPool.Get()
		defer n.reqPool.Put(req)

		err := n.requestCodec.Unmarshal(msg.Data, req)
		if err != nil {
			n.logger.Error("while decoding the request", "error", err)
			replyFunc(ctx, nil, fmt.Errorf("while decoding the request: %w", err))

			return
		}

		// Handle metadata coming from the client to the server/handler.
		ctx, md := metadata.WithIncoming(ctx)
		for k, v := range req.GetMetadata() {
			md[k] = v
		}

		// Prepare the context for outgoing (to the client) metadata.
		ctx, _ = metadata.WithOutgoing(ctx)

		evReq := n.evReqPool.Get()
		defer n.evReqPool.Put(evReq)
		evReq.ContentType = req.GetContentType()
		evReq.Data = req.GetData()
		evReq.SetReplyFunc(replyFunc)

		callbackHandler(ctx, evReq)
	})
	wPool.SetNumShards(n.config.MaxConcurrent)

	sub, err := n.nc.SubscribeSync(topic)
	if err != nil {
		n.logger.Error("can't handle", "error", err)
		return
	}

	wPool.Start()

	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				break
			}

			n.logger.Error("while getting a message", "error", err)
		}

		if err := wPool.AddTask(msg); err != nil {
			n.logger.Error("while adding a worker task", "error", err)
		}
	}

	// Unsubscribe after the loop has been canceled.
	if err := sub.Unsubscribe(); err != nil {
		n.logger.Error("while unsubscribing", "error", err)
	}

	wPool.Stop()
}

// Start events.
func (n *NatsJS) Start(_ context.Context) error {
	var err error

	n.nc, err = n.config.ToOptions().Connect()
	if err != nil {
		return err
	}

	// Create a JetStream management interface
	n.js, err = jetstream.New(n.nc)
	if err != nil {
		return err
	}

	// Create a NATS JetStream context for operations that require it
	n.ncjs, err = n.nc.JetStream()
	if err != nil {
		return err
	}

	return nil
}

// Stop events.
func (n *NatsJS) Stop(_ context.Context) error {
	// Close all consumers
	n.consumers.Range(func(_ string, v jetstream.ConsumeContext) bool {
		v.Stop()
		return true
	})

	n.consumers = container.NewSafeMap[string, jetstream.ConsumeContext]()

	// Close the connection to nats jetstream.
	n.js = nil
	n.ncjs = nil

	n.nc.Close()
	n.nc = nil

	return nil
}

// String returns the plugin name.
func (n *NatsJS) String() string {
	return Name
}

// Type returns the component type.
func (n *NatsJS) Type() string {
	return event.ComponentType
}

// Provide creates a new NatsJS event client.
func Provide(
	svcCtx *cli.ServiceContext,
	logger log.Logger,
	opts ...event.Option,
) (event.Type, error) {
	cfg, err := NewConfig(opts...)
	if err != nil {
		return event.Type{}, fmt.Errorf("create nats registry config: %w", err)
	}

	if err := config.Parse(nil, event.DefaultConfigSection, svcCtx.Config, &cfg); err != nil {
		return event.Type{}, fmt.Errorf("parse config: %w", err)
	}

	instance, err := New(svcCtx.Name(), cfg, logger)

	return event.Type{Client: instance}, err
}
