// Package natjs provides the nats jetstream event client for go-orb.
package natjs

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/event"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
	"github.com/go-orb/go-orb/util/container"
	"github.com/go-orb/go-orb/util/metadata"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// This is here to make sure it implements event.Events.
var _ event.Events = (*NatsJS)(nil)

type replyMessage struct {
	Metadata metadata.Metadata `json:"metadata"`
	Data     []byte            `json:"data"`
	Err      error             `json:"err"`
}

// NatsJS is the nats jetstream event client for go-orb.
type NatsJS struct {
	serviceName string

	config Config
	logger log.Logger

	nc    *nats.Conn
	js    jetstream.JetStream
	codec codecs.Marshaler

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

	codec, err := codecs.GetMime("application/json")
	if err != nil {
		panic(err)
	}

	return &NatsJS{
		serviceName: serviceName,
		config:      cfg,
		logger:      log,
		codec:       codec,
		consumers:   container.NewSafeMap[string, jetstream.ConsumeContext](),
	}
}

// Request runs a REST like call on the given topic.
func (n *NatsJS) Request(_ context.Context, topic string, req *event.Call[[]byte, any], opts ...event.RequestOption) ([]byte, error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, event.ErrMissingTopic
	}

	if req.Err != nil {
		return nil, req.Err
	}

	options := event.NewCallOptions(opts...)

	data, err := n.codec.Encode(req)
	if err != nil {
		return nil, err
	}

	// Send the request and wait for a reply.
	msg, err := n.nc.Request(topic, data, options.RequestTimeout)
	if err != nil {
		n.logger.Error("while publishing a call", "topic", topic, "err", err)
		return nil, err
	}

	reply := &replyMessage{}

	err = n.codec.Decode(msg.Data, reply)
	if err != nil {
		n.logger.Error("while decoding the reply", "topic", topic, "err", err)
		return nil, err
	}

	return reply.Data, reply.Err
}

// HandleRequest subscribes to the given topic and handles the requests.
func (n *NatsJS) HandleRequest(
	_ context.Context,
	topic string,
) (<-chan event.Call[[]byte, []byte], error) {
	// validate the topic
	if len(topic) == 0 {
		return nil, event.ErrMissingTopic
	}

	outChan := make(chan event.Call[[]byte, []byte])

	_, err := n.nc.Subscribe(topic, func(msg *nats.Msg) {
		req := event.Call[[]byte, []byte]{}

		err := n.codec.Decode(msg.Data, &req)
		if err != nil {
			req.Err = orberrors.From(err)
			return
		}

		req.SetReplyFunc(func(result []byte, inErr *orberrors.Error) error {
			reply := &replyMessage{
				Metadata: metadata.Metadata{},
				Data:     result,
				Err:      inErr,
			}

			b, err := n.codec.Encode(reply)
			if err != nil {
				return err
			}

			if err = msg.Respond(b); err != nil {
				return err
			}

			return nil
		})

		outChan <- req
	})
	if err != nil {
		return nil, err
	}
	// sub.Unsubscribe() //nolint:errcheck

	return outChan, nil
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
) (event.Type, error) {
	cfg, err := NewConfig(name, datas, opts...)
	if err != nil {
		return event.Type{}, fmt.Errorf("create nats registry config: %w", err)
	}

	me := New(string(name), cfg, logger)

	return event.Type{Events: me}, nil
}
