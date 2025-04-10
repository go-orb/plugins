package natsjs

import (
	"time"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/event"
	"github.com/nats-io/nats.go"
)

// Name provides the name of this event client.
const Name = "natsjs"

// Defaults.
//
//nolint:gochecknoglobals
var (
	DefaultMaxConcurrent = 256

	// DefaultSyncPublish allows using a synchronous publishing instead of the default asynchronous.
	DefaultSyncPublish = false

	// DefaultDisableDurableStreams configures whether to disable durable streams.
	DefaultDisableDurableStreams = false
)

func init() {
	event.Register(Name, Provide)
}

// NatsOptions can be used to create a customized connection.
type NatsOptions struct {
	// URL represents a single NATS server url to which the client
	// will be connecting. If the Servers option is also set, it
	// then becomes the first server in the Servers array.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// InProcessServer represents a NATS server running within the
	// same process. If this is set then we will attempt to connect
	// to the server directly rather than using external TCP conns.
	InProcessServer nats.InProcessConnProvider `json:"-" yaml:"-"`

	// Servers is a configured set of servers which this client
	// will use when attempting to connect.
	Servers []string `json:"servers,omitempty" yaml:"servers,omitempty"`

	// NoRandomize configures whether we will randomize the
	// server pool.
	NoRandomize bool `json:"noRandomize,omitempty" yaml:"noRandomize,omitempty"`

	// AllowReconnect enables reconnection logic to be used when we
	// encounter a disconnect from the current server.
	AllowReconnect bool `json:"allowReconnect,omitempty" yaml:"allowReconnect,omitempty"`

	// MaxReconnect sets the number of reconnect attempts that will be
	// tried before giving up. If negative, then it will never give up
	// trying to reconnect.
	// Defaults to 60.
	MaxReconnect int `json:"maxReconnect,omitempty" yaml:"maxReconnect,omitempty"`

	// ReconnectWait sets the time to backoff after attempting a reconnect
	// to a server that we were already connected to previously.
	// Defaults to 2s.
	ReconnectWait config.Duration `json:"reconnectWait,omitempty" yaml:"reconnectWait,omitempty"`

	// CustomReconnectDelayCB is invoked after the library tried every
	// URL in the server list and failed to reconnect. It passes to the
	// user the current number of attempts. This function returns the
	// amount of time the library will sleep before attempting to reconnect
	// again. It is strongly recommended that this value contains some
	// jitter to prevent all connections to attempt reconnecting at the same time.
	CustomReconnectDelayCB nats.ReconnectDelayHandler `json:"-" yaml:"-"`

	// ReconnectJitter sets the upper bound for a random delay added to
	// ReconnectWait during a reconnect when no TLS is used.
	// Defaults to 100ms.
	ReconnectJitter config.Duration `json:"reconnectJitter,omitempty" yaml:"reconnectJitter,omitempty"`

	// ReconnectJitterTLS sets the upper bound for a random delay added to
	// ReconnectWait during a reconnect when TLS is used.
	// Defaults to 1s.
	ReconnectJitterTLS config.Duration `json:"reconnectJitterTls,omitempty" yaml:"reconnectJitterTls,omitempty"`

	// Timeout sets the timeout for a Dial operation on a connection.
	// Defaults to 2s.
	Timeout config.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`

	// DrainTimeout sets the timeout for a Drain Operation to complete.
	// Defaults to 30s.
	DrainTimeout config.Duration `json:"drainTimeout,omitempty" yaml:"drainTimeout,omitempty"`

	// FlusherTimeout is the maximum time to wait for write operations
	// to the underlying connection to complete (including the flusher loop).
	// Defaults to 1m.
	FlusherTimeout config.Duration `json:"flusherTimeout,omitempty" yaml:"flusherTimeout,omitempty"`

	// PingInterval is the period at which the client will be sending ping
	// commands to the server, disabled if 0 or negative.
	// Defaults to 2m.
	PingInterval config.Duration `json:"pingInterval,omitempty" yaml:"pingInterval,omitempty"`

	// MaxPingsOut is the maximum number of pending ping commands that can
	// be awaiting a response before raising an ErrStaleConnection error.
	// Defaults to 2.
	MaxPingsOut int `json:"maxPingsOut,omitempty" yaml:"maxPingsOut,omitempty"`

	// ClosedCB sets the closed handler that is called when a client will
	// no longer be connected.
	ClosedCB nats.ConnHandler `json:"-" yaml:"-"`

	// DisconnectedErrCB sets the disconnected error handler that is called
	// whenever the connection is disconnected.
	// Disconnected error could be nil, for instance when user explicitly closes the connection.
	// DisconnectedCB will not be called if DisconnectedErrCB is set
	DisconnectedErrCB nats.ConnErrHandler `json:"-" yaml:"-"`

	// ConnectedCB sets the connected handler called when the initial connection
	// is established. It is not invoked on successful reconnects - for reconnections,
	// use ReconnectedCB. ConnectedCB can be used in conjunction with RetryOnFailedConnect
	// to detect whether the initial connect was successful.
	ConnectedCB nats.ConnHandler `json:"-" yaml:"-"`

	// ReconnectedCB sets the reconnected handler called whenever
	// the connection is successfully reconnected.
	ReconnectedCB nats.ConnHandler `json:"-" yaml:"-"`

	// DiscoveredServersCB sets the callback that is invoked whenever a new
	// server has joined the cluster.
	DiscoveredServersCB nats.ConnHandler `json:"-" yaml:"-"`

	// AsyncErrorCB sets the async error handler (e.g. slow consumer errors)
	AsyncErrorCB nats.ErrHandler `json:"-" yaml:"-"`

	// ReconnectBufSize is the size of the backing bufio during reconnect.
	// Once this has been exhausted publish operations will return an error.
	// Defaults to 8388608 bytes (8MB).
	ReconnectBufSize int `json:"reconnectBufSize,omitempty" yaml:"reconnectBufSize,omitempty"`

	// SubChanLen is the size of the buffered channel used between the socket
	// Go routine and the message delivery for SyncSubscriptions.
	// NOTE: This does not affect AsyncSubscriptions which are
	// dictated by PendingLimits()
	// Defaults to 65536.
	SubChanLen int `json:"subChanLen,omitempty" yaml:"subChanLen,omitempty"`

	// UserJWT sets the callback handler that will fetch a user's JWT.
	UserJWT nats.UserJWTHandler `json:"-" yaml:"-"`

	// Nkey sets the public nkey that will be used to authenticate
	// when connecting to the server. UserJWT and Nkey are mutually exclusive
	// and if defined, UserJWT will take precedence.
	Nkey string `json:"nkey,omitempty" yaml:"nkey,omitempty"`

	// SignatureCB designates the function used to sign the nonce
	// presented from the server.
	SignatureCB nats.SignatureHandler `json:"-" yaml:"-"`

	// User sets the username to be used when connecting to the server.
	User string `json:"user,omitempty" yaml:"user,omitempty"`

	// Password sets the password to be used when connecting to a server.
	Password string `json:"password,omitempty" yaml:"password,omitempty"`

	// Token sets the token to be used when connecting to a server.
	Token string `json:"token,omitempty" yaml:"token,omitempty"`

	// TokenHandler designates the function used to generate the token to be used when connecting to a server.
	TokenHandler nats.AuthTokenHandler `json:"-" yaml:"-"`

	// CustomDialer allows to specify a custom dialer (not necessarily
	// a *net.Dialer).
	CustomDialer nats.CustomDialer `json:"-" yaml:"-"`

	// UseOldRequestStyle forces the old method of Requests that utilize
	// a new Inbox and a new Subscription for each request.
	UseOldRequestStyle bool `json:"useOldRequestStyle,omitempty" yaml:"useOldRequestStyle,omitempty"`

	// NoCallbacksAfterClientClose allows preventing the invocation of
	// callbacks after Close() is called. Client won't receive notifications
	// when Close is invoked by user code. Default is to invoke the callbacks.
	NoCallbacksAfterClientClose bool `json:"noCallbacksAfterClientClose,omitempty" yaml:"noCallbacksAfterClientClose,omitempty"`

	// LameDuckModeHandler sets the callback to invoke when the server notifies
	// the connection that it entered lame duck mode, that is, going to
	// gradually disconnect all its connections before shutting down. This is
	// often used in deployments when upgrading NATS Servers.
	LameDuckModeHandler nats.ConnHandler `json:"-" yaml:"-"`

	// RetryOnFailedConnect sets the connection in reconnecting state right
	// away if it can't connect to a server in the initial set. The
	// MaxReconnect and ReconnectWait options are used for this process,
	// similarly to when an established connection is disconnected.
	// If a ReconnectHandler is set, it will be invoked on the first
	// successful reconnect attempt (if the initial connect fails),
	// and if a ClosedHandler is set, it will be invoked if
	// it fails to connect (after exhausting the MaxReconnect attempts).
	RetryOnFailedConnect bool `json:"retryOnFailedConnect,omitempty" yaml:"retryOnFailedConnect,omitempty"`

	// For websocket connections, indicates to the server that the connection
	// supports compression. If the server does too, then data will be compressed.
	Compression bool `json:"compression,omitempty" yaml:"compression,omitempty"`

	// For websocket connections, adds a path to connections url.
	// This is useful when connecting to NATS behind a proxy.
	ProxyPath string `json:"proxyPath,omitempty" yaml:"proxyPath,omitempty"`

	// InboxPrefix allows the default _INBOX prefix to be customized
	InboxPrefix string `json:"inboxPrefix,omitempty" yaml:"inboxPrefix,omitempty"`

	// IgnoreAuthErrorAbort - if set to true, client opts out of the default connect behavior of aborting
	// subsequent reconnect attempts if server returns the same auth error twice (regardless of reconnect policy).
	IgnoreAuthErrorAbort bool `json:"ignoreAuthErrorAbort,omitempty" yaml:"ignoreAuthErrorAbort,omitempty"`

	// SkipHostLookup skips the DNS lookup for the server hostname.
	SkipHostLookup bool `json:"skipHostLookup,omitempty" yaml:"skipHostLookup,omitempty"`
}

// ToOptions converts the NatsOptions to nats.Options.
func (o NatsOptions) ToOptions() nats.Options {
	options := nats.Options{
		Url:                         o.URL,
		InProcessServer:             o.InProcessServer,
		Servers:                     o.Servers,
		NoRandomize:                 o.NoRandomize,
		AllowReconnect:              o.AllowReconnect,
		MaxReconnect:                o.MaxReconnect,
		ReconnectWait:               time.Duration(o.ReconnectWait),
		CustomReconnectDelayCB:      o.CustomReconnectDelayCB,
		ReconnectJitter:             time.Duration(o.ReconnectJitter),
		ReconnectJitterTLS:          time.Duration(o.ReconnectJitterTLS),
		Timeout:                     time.Duration(o.Timeout),
		DrainTimeout:                time.Duration(o.DrainTimeout),
		FlusherTimeout:              time.Duration(o.FlusherTimeout),
		PingInterval:                time.Duration(o.PingInterval),
		MaxPingsOut:                 o.MaxPingsOut,
		ClosedCB:                    o.ClosedCB,
		DisconnectedErrCB:           o.DisconnectedErrCB,
		ConnectedCB:                 o.ConnectedCB,
		ReconnectedCB:               o.ReconnectedCB,
		DiscoveredServersCB:         o.DiscoveredServersCB,
		AsyncErrorCB:                o.AsyncErrorCB,
		ReconnectBufSize:            o.ReconnectBufSize,
		SubChanLen:                  o.SubChanLen,
		UserJWT:                     o.UserJWT,
		Nkey:                        o.Nkey,
		SignatureCB:                 o.SignatureCB,
		User:                        o.User,
		Password:                    o.Password,
		Token:                       o.Token,
		TokenHandler:                o.TokenHandler,
		CustomDialer:                o.CustomDialer,
		UseOldRequestStyle:          o.UseOldRequestStyle,
		NoCallbacksAfterClientClose: o.NoCallbacksAfterClientClose,
		LameDuckModeHandler:         o.LameDuckModeHandler,
		RetryOnFailedConnect:        o.RetryOnFailedConnect,
		Compression:                 o.Compression,
		ProxyPath:                   o.ProxyPath,
		InboxPrefix:                 o.InboxPrefix,
		IgnoreAuthErrorAbort:        o.IgnoreAuthErrorAbort,
		SkipHostLookup:              o.SkipHostLookup,
	}

	return options
}

// Config provides configuration for the NATS event client.
type Config struct {
	event.Config `yaml:",inline"`

	NatsOptions `yaml:",inline"`

	// RequestCodec configures the codec to use for encoding and decoding.
	// Default: application/x-protobuf
	RequestCodec string `json:"requestCodec,omitempty" yaml:"requestCodec,omitempty"`

	// PublishCodec configures the codec to use for publishing.
	// Default: application/json
	PublishCodec string `json:"publishCodec,omitempty" yaml:"publishCodec,omitempty"`

	// MaxConcurrent configures the maximum number of concurrent workers.
	// Default: 10
	MaxConcurrent int `json:"maxConcurrent,omitempty" yaml:"maxConcurrent,omitempty"`

	// SyncPublish configures whether to publish synchronously.
	// Default: false
	SyncPublish bool `json:"syncPublish,omitempty" yaml:"syncPublish,omitempty"`

	// DisableDurableStreams configures whether to disable durable streams.
	// Default: false
	DisableDurableStreams bool `json:"disableDurableStreams,omitempty" yaml:"disableDurableStreams,omitempty"`
}

// ApplyOptions applies a set of options to the config.
func (c *Config) ApplyOptions(opts ...event.Option) {
	for _, o := range opts {
		o(c)
	}
}

// WithURL sets the URL of the NATS server.
func WithURL(url string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.URL = url
		}
	}
}

// WithServers sets the list of NATS servers to connect to.
func WithServers(servers []string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.Servers = servers
		}
	}
}

// WithNoRandomize configures whether to randomize the server pool.
func WithNoRandomize(noRandomize bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.NoRandomize = noRandomize
		}
	}
}

// WithAllowReconnect enables reconnection logic when disconnected from the server.
func WithAllowReconnect(allowReconnect bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.AllowReconnect = allowReconnect
		}
	}
}

// WithMaxReconnect sets the number of reconnect attempts before giving up.
func WithMaxReconnect(maxReconnect int) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.MaxReconnect = maxReconnect
		}
	}
}

// WithReconnectWait sets the time to backoff after attempting a reconnect.
func WithReconnectWait(reconnectWait time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.ReconnectWait = config.Duration(reconnectWait)
		}
	}
}

// WithReconnectJitter sets the upper bound for random delay added to ReconnectWait.
func WithReconnectJitter(reconnectJitter time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.ReconnectJitter = config.Duration(reconnectJitter)
		}
	}
}

// WithReconnectJitterTLS sets the upper bound for random delay added to ReconnectWait when TLS is used.
func WithReconnectJitterTLS(reconnectJitterTLS time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.ReconnectJitterTLS = config.Duration(reconnectJitterTLS)
		}
	}
}

// WithTimeout sets the timeout for a Dial operation on a connection.
func WithTimeout(timeout time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.Timeout = config.Duration(timeout)
		}
	}
}

// WithDrainTimeout sets the timeout for a Drain Operation to complete.
func WithDrainTimeout(drainTimeout time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.DrainTimeout = config.Duration(drainTimeout)
		}
	}
}

// WithFlusherTimeout sets the maximum time to wait for write operations to complete.
func WithFlusherTimeout(flusherTimeout time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.FlusherTimeout = config.Duration(flusherTimeout)
		}
	}
}

// WithPingInterval sets the period for sending ping commands to the server.
func WithPingInterval(pingInterval time.Duration) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.PingInterval = config.Duration(pingInterval)
		}
	}
}

// WithMaxPingsOut sets the maximum number of pending ping commands before raising an error.
func WithMaxPingsOut(maxPingsOut int) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.MaxPingsOut = maxPingsOut
		}
	}
}

// WithReconnectBufSize sets the size of the backing bufio during reconnect.
func WithReconnectBufSize(reconnectBufSize int) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.ReconnectBufSize = reconnectBufSize
		}
	}
}

// WithSubChanLen sets the size of the buffered channel used for SyncSubscriptions.
func WithSubChanLen(subChanLen int) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.SubChanLen = subChanLen
		}
	}
}

// WithNkey sets the public nkey for authentication.
func WithNkey(nkey string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.Nkey = nkey
		}
	}
}

// WithUser sets the username for authentication.
func WithUser(user string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.User = user
		}
	}
}

// WithPassword sets the password for authentication.
func WithPassword(password string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.Password = password
		}
	}
}

// WithToken sets the token for authentication.
func WithToken(token string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.Token = token
		}
	}
}

// WithUseOldRequestStyle forces the old method of Requests.
func WithUseOldRequestStyle(useOldRequestStyle bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.UseOldRequestStyle = useOldRequestStyle
		}
	}
}

// WithNoCallbacksAfterClientClose prevents callbacks after Close() is called.
func WithNoCallbacksAfterClientClose(noCallbacksAfterClientClose bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.NoCallbacksAfterClientClose = noCallbacksAfterClientClose
		}
	}
}

// WithRetryOnFailedConnect sets the connection in reconnecting state if it can't connect initially.
func WithRetryOnFailedConnect(retryOnFailedConnect bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.RetryOnFailedConnect = retryOnFailedConnect
		}
	}
}

// WithCompression enables compression for websocket connections.
func WithCompression(compression bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.Compression = compression
		}
	}
}

// WithProxyPath adds a path to connections URL for websocket connections.
func WithProxyPath(proxyPath string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.ProxyPath = proxyPath
		}
	}
}

// WithInboxPrefix allows customizing the default _INBOX prefix.
func WithInboxPrefix(inboxPrefix string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.InboxPrefix = inboxPrefix
		}
	}
}

// WithIgnoreAuthErrorAbort opts out of aborting reconnect attempts on repeated auth errors.
func WithIgnoreAuthErrorAbort(ignoreAuthErrorAbort bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.IgnoreAuthErrorAbort = ignoreAuthErrorAbort
		}
	}
}

// WithSkipHostLookup skips the DNS lookup for the server hostname.
func WithSkipHostLookup(skipHostLookup bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.NatsOptions.SkipHostLookup = skipHostLookup
		}
	}
}

// WithRequestCodec sets the internal codec.
func WithRequestCodec(n string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.RequestCodec = n
		}
	}
}

// WithPublishCodec sets the internal codec.
func WithPublishCodec(n string) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.PublishCodec = n
		}
	}
}

// WithMaxConcurrent sets the number of concurrent workers.
func WithMaxConcurrent(n int) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.MaxConcurrent = n
		}
	}
}

// WithSyncPublish configures whether to publish synchronously.
func WithSyncPublish(syncPublish bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.SyncPublish = syncPublish
		}
	}
}

// WithDisableDurableStreams configures whether to disable durable streams.
func WithDisableDurableStreams(disableDurableStreams bool) event.Option {
	return func(c event.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.DisableDurableStreams = disableDurableStreams
		}
	}
}

// NewConfig creates a new config object.
func NewConfig(opts ...event.Option) (Config, error) {
	cfg := Config{
		Config: event.NewConfig(),

		NatsOptions: NatsOptions{
			AllowReconnect:     true,
			MaxReconnect:       nats.DefaultMaxReconnect,
			ReconnectWait:      config.Duration(nats.DefaultReconnectWait),
			ReconnectJitter:    config.Duration(nats.DefaultReconnectJitter),
			ReconnectJitterTLS: config.Duration(nats.DefaultReconnectJitterTLS),
			Timeout:            config.Duration(nats.DefaultTimeout),
			PingInterval:       config.Duration(nats.DefaultPingInterval),
			MaxPingsOut:        nats.DefaultMaxPingOut,
			SubChanLen:         nats.DefaultMaxChanLen,
			ReconnectBufSize:   nats.DefaultReconnectBufSize,
			DrainTimeout:       config.Duration(nats.DefaultDrainTimeout),
			FlusherTimeout:     config.Duration(nats.DefaultFlusherTimeout),
		},

		RequestCodec:  event.DefaultRequestContentType,
		PublishCodec:  event.DefaultPublishContentType,
		MaxConcurrent: DefaultMaxConcurrent,
		SyncPublish:   DefaultSyncPublish,
	}

	cfg.ApplyOptions(opts...)

	return cfg, nil
}
