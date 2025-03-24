package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/server"
	"github.com/go-orb/go-orb/types"

	mhttp "github.com/go-orb/plugins/server/http"

	"github.com/go-orb/plugins/server/http/tests/handler"
	"github.com/go-orb/plugins/server/http/tests/proto"
	thttp "github.com/go-orb/plugins/server/http/tests/util/http"

	_ "github.com/go-orb/plugins/codecs/form"
	_ "github.com/go-orb/plugins/codecs/json"
	_ "github.com/go-orb/plugins/codecs/proto"

	_ "github.com/go-orb/plugins/codecs/yaml"
	_ "github.com/go-orb/plugins/config/source/file"
	_ "github.com/go-orb/plugins/log/slog"
	_ "github.com/go-orb/plugins/registry/mdns"
)

// TODO(davincible): test get path params

// TODO(davincible): for client, provide info on this error: 		t.Error("As:", errors.As(err, &x509.HostnameError{}))
//       >> change URL to proper hostname

// TODO(davincible): Provide context on unknown authority error for client x509.UnknownAuthorityError
//       >> Self signed cert was used

/*
Notes for HTTP/client << micro client
If scheme is HTTPS://
 > use tls dial & check proto for which transport to use
Else
 > use http1 transport without upgrade ()
*/

func init() {
	log.DefaultLevel = log.LevelDebug.String()
}

func TestServerSimple(t *testing.T) {
	srv, cleanup, err := setupServer(t, false, mhttp.WithInsecure())
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	addr := "http://" + srv.Address()

	makeRequests(t, addr, thttp.TypeInsecure)
}

func TestServerHTTPS(t *testing.T) {
	srv, cleanup, err := setupServer(t, false, mhttp.WithDisableHTTP2())
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	addr := "https://" + srv.Address()

	makeRequests(t, addr, thttp.TypeHTTP1)
}

func TestServerHTTP2(t *testing.T) {
	srv, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	addr := "https://" + srv.Address()

	makeRequests(t, addr, thttp.TypeHTTP2)
}

func TestServerH2c(t *testing.T) {
	srv, cleanup, err := setupServer(t, false,
		mhttp.WithInsecure(),
		mhttp.WithAllowH2C(),
	)
	defer cleanup()
	require.NoError(t, err)

	addr := "http://" + srv.Address()

	makeRequests(t, addr, thttp.TypeH2C)
}

func TestServerHTTP3Twice(t *testing.T) {
	// To fix warning about buf size run:
	// - sysctl -w net.core.rmem_max=2500000
	// - sysctl -w net.core.wmem_max=2500000
	srv, cleanup, err := setupServer(t, false,
		mhttp.WithHTTP3(),
	)
	require.NoError(t, err)

	addr := "https://" + srv.Address()
	makeRequests(t, addr, thttp.TypeHTTP3)
	cleanup()

	// Sleep a bit to let cleanup release the port
	time.Sleep(time.Second)

	// Second run to check if setup/cleanup works
	srv, cleanup, err = setupServer(t, false,
		mhttp.WithHTTP3(),
	)
	if err != nil {
		require.NoError(t, err)
	}

	addr = "https://" + srv.Address()
	makeRequests(t, addr, thttp.TypeHTTP3)
	cleanup()
}

func TestServerEntrypointsStarts(t *testing.T) {
	addr := "localhost:45451"
	server, cleanup, err := setupServer(t, false, mhttp.WithAddress(addr))
	if err != nil {
		require.NoError(t, err)
	}
	defer cleanup()

	require.NoError(t, server.Start(context.Background()), "start server 1")
	require.NoError(t, server.Start(context.Background()), "start server 2")
	require.NoError(t, server.Start(context.Background()), "start server 3")

	addr = "https://" + addr

	makeRequests(t, addr, thttp.TypeHTTP2)
}

func TestServerGzip(t *testing.T) {
	srv, cleanup, err := setupServer(t, false, mhttp.WithGzip())
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	addr := "https://" + srv.Address()

	makeRequests(t, addr, thttp.TypeHTTP2)
}

func TestServerInvalidContentType(t *testing.T) {
	srv, cleanup, err := setupServer(t, false)
	if err != nil {
		require.NoError(t, err)
	}
	defer cleanup()

	addr := "https://" + srv.Address()

	require.ErrorContains(
		t,
		thttp.TestPostRequestProto(t, addr, "application/abcdef", thttp.TypeHTTP2),
		"Post request failed",
		"POST Proto",
	)
	require.ErrorContains(
		t,
		thttp.TestPostRequestProto(t, addr, "yadayadayada", thttp.TypeHTTP2),
		"Post request failed",
		"POST Proto",
	)
}

func TestServerNoTLS(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithTLS(&tls.Config{}))
	defer cleanup()
	require.ErrorContains(
		t,
		err,
		"tls: neither Certificates, GetCertificate, nor GetConfigForClient set in Config",
		"setting an empty TLS config should return an error",
	)
}

func TestServerInvalidMessage(t *testing.T) {
	srv, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	thttp.RefreshClients()

	addr := fmt.Sprintf("https://%s/echo.Streams/Call", srv.Address())

	// Broken json.
	msg := `{"name": "Alex}`

	req, err := http.NewRequest(http.MethodPost, addr, bytes.NewReader([]byte(msg)))
	if err != nil {
		t.Fatalf("create POST request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := thttp.HTTP2Client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	t.Logf("expected error: %v", string(body))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, string(body))
	require.NoError(t, err)

	// Close connection
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

func TestServerErrorRPC(t *testing.T) {
	srv, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	thttp.RefreshClients()

	addr := fmt.Sprintf("https://%s/echo.Streams/Call", srv.Address())

	msg := `{"name": "error"}`

	req, err := http.NewRequest(http.MethodPost, addr, bytes.NewReader([]byte(msg))) //nolint:noctx
	if err != nil {
		t.Fatalf("create POST request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := thttp.HTTP2Client.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// Close connection
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode, string(body))
	require.NoError(t, err)
}

func TestServerRequestSpecificContentType(t *testing.T) {
	srv, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		require.NoError(t, err)
	}

	thttp.RefreshClients()

	addr := fmt.Sprintf("https://%s/echo.Streams/Call", srv.Address())

	msg := `{"name": "Alex"}`

	testCt := func(expectedCt string) {
		req, err := http.NewRequest(http.MethodPost, addr, bytes.NewReader([]byte(msg))) //nolint:noctx
		if err != nil {
			t.Fatalf("create POST request failed: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", expectedCt)

		resp, err := thttp.HTTP2Client.Do(req)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		// Validate conent type received
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))
		ct, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		require.NoError(t, err)
		assert.Equal(t, expectedCt, ct, string(body))

		// Close connection
		_, err = io.Copy(io.Discard, resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}

	testCt("application/proto")
	testCt("application/x-protobuf")
	testCt("application/x-proto")
	testCt("application/x-protobuf")
	testCt("application/json")
	testCt("application/x-www-form-urlencoded")
}

func TestServerIntegration(t *testing.T) {
	name := "com.example.test"
	version := ""

	components := types.NewComponents()

	logger, err := log.New()
	require.NoError(t, err, "failed to setup the logger")

	reg, err := registry.New(nil, components, logger)
	require.NoError(t, err, "failed to setup the registry")

	h := new(handler.EchoHandler)

	srv, err := server.New(name, version, nil, logger, reg,
		server.WithEntrypointConfig(mhttp.NewConfig(
			mhttp.WithName("test-ep-1"),
			mhttp.WithAddress(":48081"),
			mhttp.WithHTTP3(),
			mhttp.WithGzip(),
			mhttp.WithHandlers(proto.RegisterStreamsHandler(h)),
		)),
		server.WithEntrypointConfig(mhttp.NewConfig(
			mhttp.WithName("test-ep-2"),
			mhttp.WithAddress(":48082"),
			mhttp.WithHTTP3(),
			mhttp.WithHandlers(proto.RegisterStreamsHandler(h)),
		)),
		server.WithEntrypointConfig(mhttp.NewConfig(
			mhttp.WithName("test-ep-3"),
			mhttp.WithAddress(":48083"),
			mhttp.WithInsecure(),
			mhttp.WithAllowH2C(),
			mhttp.WithHandlers(proto.RegisterStreamsHandler(h)),
		)),
	)
	require.NoError(t, err, "failed to setup server")
	require.NoError(t, srv.Start(context.Background()), "failed to start the server")

	e, err := srv.GetEntrypoint("test-ep-1")
	require.NoError(t, err, "failed to fetch entrypoint 1")
	require.Len(t, e.(*mhttp.Server).Router().Routes(), 1, "number of routes not equal to 1") //nolint:errcheck
	makeRequests(t, "https://"+e.Address(), thttp.TypeHTTP3)

	e, err = srv.GetEntrypoint("test-ep-2")
	require.NoError(t, err, "failed to fetch entrypoint 2")
	makeRequests(t, "https://"+e.Address(), thttp.TypeHTTP2)

	e, err = srv.GetEntrypoint("test-ep-3")
	require.NoError(t, err, "failed to fetch entrypoint 2")
	makeRequests(t, "https://"+e.Address(), thttp.TypeH2C)

	_, err = srv.GetEntrypoint("fake")
	require.Error(t, err, "fetching invalid entrypoint should fail")

	require.NoError(t, srv.Stop(context.Background()), "failed to stop the server")
}

func TestServerFileConfig(t *testing.T) {
	thttp.RefreshClients()

	server.Handlers.Set("Streams", proto.RegisterStreamsHandler(new(handler.EchoHandler)))
	server.Handlers.Set("handler-1", func(_ any) {})
	server.Handlers.Set("handler-2", func(_ any) {})

	name := "com.example.test"
	version := ""

	fURL, err := url.Parse("file://config/config.yaml")
	t.Logf("%+v", fURL.RawPath)
	require.NoError(t, err, "failed to parse file config url")

	components := types.NewComponents()

	configData, err := config.Read(fURL)
	require.NoError(t, err, "failed to read file config")

	configData, err = config.WalkMap(types.SplitServiceName(name), configData)
	require.NoError(t, err, "failed to walk file config")

	logger, err := log.New()
	require.NoError(t, err, "failed to setup the logger")

	reg, err := registry.New(nil, components, logger)
	require.NoError(t, err, "failed to setup the registry")

	h := new(handler.EchoHandler)
	srv, err := server.New(name, version, configData, logger, reg,
		server.WithEntrypointConfig(mhttp.NewConfig(
			mhttp.WithName("static-ep-1"),
			mhttp.WithAddress(":48081"),
			mhttp.WithHTTP3(),
			mhttp.WithGzip(),
			mhttp.WithHandlers(proto.RegisterStreamsHandler(h)),
		)),
		server.WithEntrypointConfig(mhttp.NewConfig(
			mhttp.WithName("test-ep-5"),
		)),
	)
	require.NoError(t, err, "failed to setup server")
	require.NoError(t, srv.Start(context.Background()), "failed to start server")

	e, err := srv.GetEntrypoint("static-ep-1")
	require.NoError(t, err, "failed to fetch entrypoint 1")
	ep := e.(*mhttp.Server) //nolint:errcheck
	require.True(t, strings.HasSuffix(ep.Address(), ":48081"))
	require.True(t, ep.Config().HTTP3, "HTTP3 static ep 1")
	require.True(t, ep.Config().Gzip, "Gzip static ep 1")
	makeRequests(t, "https://"+e.Address(), thttp.TypeHTTP3)

	e, err = srv.GetEntrypoint("test-ep-1")
	require.NoError(t, err, "failed to fetch entrypoint 1")
	ep = e.(*mhttp.Server) //nolint:errcheck
	require.True(t, strings.HasSuffix(ep.Address(), ":4512"))
	require.True(t, ep.Config().HTTP3, "HTTP3")
	makeRequests(t, "https://"+e.Address(), thttp.TypeHTTP3)

	e, err = srv.GetEntrypoint("test-ep-2")
	require.NoError(t, err, "failed to fetch entrypoint 2")
	ep = e.(*mhttp.Server) //nolint:errcheck
	require.True(t, strings.HasSuffix(ep.Address(), ":4513"))
	require.True(t, ep.Config().Insecure, "Insecure")
	require.True(t, ep.Config().H2C, "H2C")
	makeRequests(t, "https://"+e.Address(), thttp.TypeH2C)

	e, err = srv.GetEntrypoint("test-ep-3")
	require.NoError(t, err, "failed to fetch entrypoint 3")
	ep = e.(*mhttp.Server) //nolint:errcheck
	require.True(t, strings.HasSuffix(ep.Address(), ":4514"))
	require.True(t, ep.Config().HTTP3, "HTTP3")
	require.True(t, ep.Config().H2C, "H2C")
	require.True(t, ep.Config().Gzip, "Gzip")
	makeRequests(t, "https://"+e.Address(), thttp.TypeHTTP3)

	_, err = srv.GetEntrypoint("test-ep-4")
	require.Error(t, err, "should fail to fetch entrypoint 4")

	e, err = srv.GetEntrypoint("test-ep-5")
	require.NoError(t, err, "failed to fetch entrypoint 5")
	ep = e.(*mhttp.Server) //nolint:errcheck
	require.True(t, strings.HasSuffix(ep.Address(), ":4516"))
	require.Len(t, ep.Config().OptHandlers, 3, "Registration len")
	makeRequests(t, "https://"+e.Address(), thttp.TypeHTTP2)

	require.NoError(t, srv.Stop(context.Background()), "failed to start server")
}

func BenchmarkHTTPInsecureJSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeInsecure)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithInsecure())
}

func BenchmarkHTTPInseucreProto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeInsecure)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithInsecure())
}

func BenchmarkHTTP1JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithDisableHTTP2())
}

func BenchmarkHTTP1Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithDisableHTTP2())
}

func BenchmarkHTTP2JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1)
}

func BenchmarkHTTP2Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1)
}

func BenchmarkH2CJSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithAllowH2C(), mhttp.WithInsecure())
}

func BenchmarkH2CProto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithAllowH2C(), mhttp.WithInsecure())
}

func BenchmarkHTTP3JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP3)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithHTTP3())
}

func BenchmarkHTTP3Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		tb.Helper()

		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP3)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithHTTP3())
}

// benchmark.
//
//nolint:unparam
func benchmark(b *testing.B, testFunc func(testing.TB, string) error, pN, sN int, opts ...server.Option) {
	b.Helper()

	b.StopTimer()
	b.ReportAllocs()

	server, cleanup, err := setupServer(b, true, opts...)
	defer cleanup()
	if err != nil {
		b.Fatal(err)
	}

	addr := "https://" + server.Address()
	if server.Config().Insecure {
		addr = "http://" + server.Address()
	}

	runBenchmark(b, addr, testFunc, pN, sN)
}

func runBenchmark(b *testing.B, addr string, testFunc func(testing.TB, string) error, pN, sN int) {
	b.Helper()

	done := make(chan struct{})
	errChan := make(chan error, 1)

	var wg sync.WaitGroup

	b.ResetTimer()
	b.StartTimer()

	// Start requests
	go func() {
		for i := 0; i < b.N; i++ {
			thttp.RefreshClients()
			// Run parallel requests.
			for p := 0; p < pN; p++ {
				wg.Add(1)
				go func() {
					// Run sequential requests.
					for s := 0; s < sN; s++ {
						if err := testFunc(b, addr); err != nil {
							errChan <- err
						}
					}
					wg.Done()
				}()
			}
			wg.Wait()
		}
		done <- struct{}{}
	}()

	select {
	case err := <-errChan:
		b.Fatalf("Benchmark failed: %v", err)
	case <-done:
		b.StopTimer()
	}
}

func setupServer(tb testing.TB, nolog bool, opts ...server.Option) (*mhttp.Server, func(), error) {
	tb.Helper()

	name := "test-server"
	version := "v1.0.0"
	lopts := []log.Option{}
	if nolog {
		lopts = append(lopts, log.WithLevel(log.LevelError.String()))
	} else {
		lopts = append(lopts, log.WithLevel(log.LevelDebug.String()))
	}

	cancel := func() {}

	components := types.NewComponents()

	logger, err := log.New(lopts...)
	if err != nil {
		return nil, cancel, fmt.Errorf("failed to setup logger: %w", err)
	}

	reg, err := registry.New(nil, components, logger)
	if err != nil {
		return nil, nil, fmt.Errorf("setup registry: %w", err)
	}

	h := new(handler.EchoHandler)
	opts = append(opts,
		mhttp.WithHandlers(proto.RegisterStreamsHandler(h)),
	)

	cfg := mhttp.NewConfig(opts...)

	server, err := mhttp.New(name, version, cfg, logger, reg)
	if err != nil {
		return nil, cancel, fmt.Errorf("failed to provide http server: %w", err)
	}

	cleanup := func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			require.NoError(tb, err, "failed to stop")
		}
	}

	if err := server.Start(context.Background()); err != nil {
		return nil, cancel, err
	}

	return server.(*mhttp.Server), cleanup, nil //nolint:errcheck
}

func makeRequests(t *testing.T, addr string, reqType thttp.ReqType) {
	t.Helper()

	require.NoError(t, thttp.TestPostRequestJSON(t, addr, reqType), addr+": POST JSON")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/octet-stream", reqType), addr+": POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/proto", reqType), addr+": POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/x-proto", reqType), addr+": POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/x-protobuf", reqType), addr+": POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/x-protobuf", reqType), addr+": POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/x-protobuf", reqType), addr+": POST Proto")
}
