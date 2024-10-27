// Package natsjs provides the nats jetstream event client for go-orb.
package natsjs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/event"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/ultrapool"
	"github.com/go-orb/plugins/event/natsjs/pb"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// This is here to make sure it implements event.Events.
var _ event.Handler = (*NatsJS)(nil)

// NatsJS is the nats jetstream event client for go-orb.
type NatsJS struct {
	serviceName string

	config Config
	logger log.Logger

	nc    *nats.Conn
	js    jetstream.JetStream
	codec codecs.Marshaler

	evReqPool container.Pool[*event.Req[[]byte, []byte]]
	reqPool   container.Pool[*pb.Request]
	replyPool container.Pool[*pb.Reply]

	consumers *container.SafeMap[string, jetstream.ConsumeContext]
}

func setAddrs(addrs []string) []string {
	var cAddrs []string //nolint:prealloc

	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}

		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}

		cAddrs = append(cAddrs, addr)
	}

	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}

	return cAddrs
}

// New creates a new NATS registry. This functions should rarely be called manually.
// To create a new registry use ProvideRegistryNATS.
func New(serviceName string, cfg Config, log log.Logger) *NatsJS {
	cfg.Addresses = setAddrs(cfg.Addresses)

	codec, err := codecs.GetMime("application/protobuf")
	if err != nil {
		panic(err)
	}

	return &NatsJS{
		serviceName: serviceName,
		config:      cfg,
		logger:      log,
		codec:       codec,

		evReqPool: container.NewPool(func() *event.Req[[]byte, []byte] { return &event.Req[[]byte, []byte]{} }),
		reqPool:   container.NewPool(func() *pb.Request { return &pb.Request{} }),
		replyPool: container.NewPool(func() *pb.Reply { return &pb.Reply{} }),

		consumers: container.NewSafeMap[string, jetstream.ConsumeContext](),
	}
}

// Request runs a REST like call on the given topic.
func (n *NatsJS) Request(
	_ context.Context,
	req *event.Req[[]byte, any],
	opts ...event.RequestOption,
) ([]byte, error) {
	// validate the topic
	if len(req.Topic) == 0 {
		return nil, event.ErrMissingTopic
	}

	if req.Err != nil {
		return nil, req.Err
	}

	options := event.NewCallOptions(opts...)

	pbReq := &pb.Request{}
	defer n.reqPool.Put(pbReq)
	pbReq.Reset()

	pbReq.Data = req.Data
	pbReq.ContentType = req.ContentType
	pbReq.Metadata = req.Metadata

	data, err := n.codec.Encode(pbReq)
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

	err = n.codec.Decode(msg.Data, reply)
	if err != nil {
		n.logger.Error("while decoding the reply", "topic", req.Topic, "err", err, "data", msg.Data)
		return nil, err
	}

	if len(reply.GetError()) != 0 {
		return nil, errors.New(reply.GetError())
	}

	return reply.GetData(), nil
}

// HandleRequest subscribes to the given topic and handles the requests.
//
//nolint:funlen
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
			md, ok := metadata.Outgoing(ctx)
			if !ok {
				md = make(map[string]string)
			}

			reply := n.replyPool.Get()
			defer n.replyPool.Put(reply)
			reply.Reset()

			reply.Data = result
			if inErr != nil {
				reply.Error = inErr.Error()
			}

			reply.Metadata = md

			b, err := n.codec.Encode(reply)
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

		err := n.codec.Decode(msg.Data, req)
		if err != nil {
			n.logger.Error("while decoding the request", "error", err)
			replyFunc(ctx, nil, fmt.Errorf("while decoding the request: %w", err))

			return
		}

		evReq := n.evReqPool.Get()
		defer n.evReqPool.Put(evReq)
		evReq.ContentType = req.GetContentType()
		evReq.Data = req.GetData()
		evReq.SetReplyFunc(replyFunc)

		callbackHandler(context.Background(), evReq)
	})

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
func (n *NatsJS) Start() error {
	nopts := nats.GetDefaultOptions()

	if n.config.TLSConfig != nil {
		nopts.Secure = true
		nopts.TLSConfig = n.config.TLSConfig
	}

	if len(n.config.Addresses) > 0 {
		nopts.Servers = n.config.Addresses
	}

	var err error

	n.nc, err = nopts.Connect()
	if err != nil {
		return err
	}

	// Create a JetStream management interface
	n.js, err = jetstream.New(n.nc)
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
	name types.ServiceName,
	datas types.ConfigData,
	logger log.Logger,
	opts ...event.Option,
) (event.Handler, error) {
	cfg, err := NewConfig(name, datas, opts...)
	if err != nil {
		return nil, fmt.Errorf("create nats registry config: %w", err)
	}

	me := New(string(name), cfg, logger)

	return me, nil
}
