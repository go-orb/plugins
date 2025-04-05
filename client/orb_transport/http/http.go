// Package http contains the http transport for the orb client,
package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/util/orberrors"
	"github.com/go-orb/plugins/client/orb"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

const networkUnix = "unix"

func init() {
	orb.RegisterTransport("http", NewHTTPTransport("tcp"))
	orb.RegisterTransport("http3", NewHTTP3Transport)
	orb.RegisterTransport("https", NewHTTPSTransport)
	orb.RegisterTransport("unix+http", NewHTTPTransport(networkUnix))
}

//nolint:gochecknoglobals
var stdHeaders = []string{"Content-Length", "Content-Type", "Date", "Server"}

var _ (orb.Transport) = (*Transport)(nil)

// Transport is a go-orb/plugins/client/orb compatible transport.
type Transport struct {
	name    string
	config  *orb.Config
	logger  log.Logger
	hclient *http.Client
	network string
	scheme  string

	bufPool *sync.Pool
}

// Start starts the transport.
func (t *Transport) Start() error {
	t.bufPool = &sync.Pool{
		New: func() any {
			return bytes.NewBuffer(nil)
		},
	}

	return nil
}

// Stop stop the transport.
func (t *Transport) Stop(_ context.Context) error {
	if t.hclient != nil {
		t.hclient.CloseIdleConnections()
	}

	return nil
}

// Name returns the name of this transport.
func (t *Transport) Name() string {
	return t.name
}

// Request does the actual rpc call to the server.
//
//nolint:gocyclo,funlen
func (t *Transport) Request(ctx context.Context, infos client.RequestInfos, req any, result any, opts *client.CallOptions) error {
	codec, err := codecs.GetMime(opts.ContentType)
	if err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	// Encode the request into a *bytes.Buffer{}.
	buff := t.bufPool.Get().(*bytes.Buffer)
	if err := codec.NewEncoder(buff).Encode(req); err != nil { //nolint:errcheck
		return orberrors.ErrBadRequest.Wrap(err)
	}

	// Set the connection timeout
	ctx, cancel := context.WithTimeout(ctx, opts.ConnectionTimeout)
	defer cancel()

	var (
		hReq *http.Request
	)

	// Create a net/http request.
	if t.network == networkUnix {
		hReq, err = http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			fmt.Sprintf("%s://%s%s", t.scheme, networkUnix, infos.Endpoint),
			buff,
		)

		t.hclient.Transport.(*http.Transport).DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) { //nolint:errcheck
			dialer := net.Dialer{
				Timeout: time.Duration(t.config.DialTimeout),
			}

			return dialer.DialContext(ctx, t.network, infos.Address)
		}
	} else {
		hReq, err = http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			fmt.Sprintf("%s://%s%s", t.scheme, infos.Address, infos.Endpoint),
			buff,
		)
	}

	if err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	// Set headers.
	hReq.Header.Set("Content-Type", opts.ContentType)
	hReq.Header.Set("Accept", opts.ContentType)

	// Set metadata key=value to request headers.
	for name, value := range opts.Metadata {
		hReq.Header.Set(name, value)
	}

	// Run the request.
	resp, err := t.hclient.Do(hReq)
	if err != nil {
		return orberrors.From(err)
	}

	// Reset the buffer.
	buff.Reset()

	_, err = buff.ReadFrom(resp.Body)
	if err != nil && !errors.Is(err, io.EOF) {
		return orberrors.From(err)
	}

	// Close the request body.
	if err := resp.Body.Close(); err != nil {
		return orberrors.From(err)
	}

	if opts.ResponseMetadata != nil {
		md := opts.ResponseMetadata

		// Copy headers to opts.Header
		for k, v := range resp.Header {
			// Skip std headers.
			if slices.Contains(stdHeaders, k) {
				continue
			}

			if len(v) == 1 {
				md[strings.ToLower(k)] = v[0]
			} else {
				md[strings.ToLower(k)] = v[0]

				for i := 1; i < len(v); i++ {
					md[strings.ToLower(k)+"-"+strconv.Itoa(i)] = v[i]
				}
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		return orberrors.HTTP(resp.StatusCode)
	}

	// Decode the response into `result`.
	err = codec.NewDecoder(buff).Decode(result)
	if err != nil {
		return orberrors.ErrBadRequest.Wrap(err)
	}

	// Reset the buffer.
	buff.Reset()

	t.bufPool.Put(buff)

	return nil
}

// Stream creates a bidirectional stream to the service endpoint.
// HTTP transport does not support streaming operations by default.
func (t *Transport) Stream(_ context.Context, _ client.RequestInfos, _ *client.CallOptions) (client.StreamIface[any, any], error) {
	return nil, orberrors.ErrNotImplemented.Wrap(client.ErrStreamNotSupported)
}

// NewTransport creates a Transport with a custom http.Client.
func NewTransport(
	name string,
	logger log.Logger,
	scheme string,
	network string,
	cfg *orb.Config,
	hclient *http.Client,
) (orb.TransportType, error) {
	return orb.TransportType{Transport: &Transport{
		config:  cfg,
		name:    name,
		logger:  logger,
		scheme:  scheme,
		network: network,
		hclient: hclient,
	},
	}, nil
}

// NewHTTPTransport creates a new https transport for the orb client.
func NewHTTPTransport(network string) func(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	return func(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
		return NewTransport(
			"http",
			logger,
			"http",
			network,
			cfg,
			&http.Client{
				Timeout: time.Duration(cfg.ConnectionTimeout),
				Transport: &http.Transport{
					MaxIdleConns:          cfg.PoolHosts * cfg.PoolSize,
					MaxIdleConnsPerHost:   cfg.PoolSize,
					MaxConnsPerHost:       cfg.PoolSize + 1,
					IdleConnTimeout:       time.Duration(cfg.PoolTTL),
					ExpectContinueTimeout: 1 * time.Second,
					ForceAttemptHTTP2:     true,
					DisableKeepAlives:     false,
					DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
						dialer := net.Dialer{
							Timeout: time.Duration(cfg.DialTimeout),
						}
						return dialer.DialContext(ctx, network, addr)
					},
				},
			},
		)
	}
}

// NewHTTP3Transport creates a new https transport for the orb client.
// This transport is used for HTTP/3.
func NewHTTP3Transport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	tlsConfig := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
	}

	if cfg.TLSConfig != nil {
		tlsConfig = cfg.TLSConfig
	}

	return NewTransport(
		"http3",
		logger,
		"https",
		"tcp",
		cfg,
		&http.Client{
			Timeout: time.Duration(cfg.ConnectionTimeout),
			Transport: &http3.Transport{
				QUICConfig: &quic.Config{
					MaxIncomingStreams:         int64(cfg.PoolSize * cfg.PoolHosts),
					MaxIncomingUniStreams:      int64(cfg.PoolSize * cfg.PoolHosts),
					MaxStreamReceiveWindow:     3 * (1 << 20),   // 3 MB
					MaxConnectionReceiveWindow: 4.5 * (1 << 20), // 4.5 MB
				},
				TLSClientConfig: tlsConfig,
			},
		},
	)
}

// NewHTTPSTransport creates a new https transport for the orb client.
// This transport is used for HTTPS/1.1.
func NewHTTPSTransport(logger log.Logger, cfg *orb.Config) (orb.TransportType, error) {
	tlsConfig := &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
	}

	if cfg.TLSConfig != nil {
		tlsConfig = cfg.TLSConfig
	}

	return NewTransport(
		"https",
		logger,
		"https",
		"tcp",
		cfg,
		&http.Client{
			Timeout: time.Duration(cfg.ConnectionTimeout),
			Transport: &http.Transport{
				MaxIdleConns:          cfg.PoolHosts * cfg.PoolSize,
				MaxIdleConnsPerHost:   cfg.PoolSize,
				MaxConnsPerHost:       cfg.PoolSize + 1,
				IdleConnTimeout:       time.Duration(cfg.PoolTTL),
				ExpectContinueTimeout: 1 * time.Second,
				ForceAttemptHTTP2:     true,
				DisableKeepAlives:     false,
				Dial: (&net.Dialer{
					Timeout:   time.Duration(cfg.DialTimeout),
					KeepAlive: 15 * time.Second,
					DualStack: false,
				}).Dial,
				TLSHandshakeTimeout: time.Duration(cfg.DialTimeout),
				TLSClientConfig:     tlsConfig,
			},
		},
	)
}
