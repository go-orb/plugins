package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/types"

	mhttp "github.com/go-micro/plugins/server/http"
	thttp "github.com/go-micro/plugins/server/tests/utils/http"

	"github.com/go-micro/plugins/server/tests/handler"
	"github.com/go-micro/plugins/server/tests/proto"

	_ "github.com/go-micro/plugins/codecs/form"
	_ "github.com/go-micro/plugins/codecs/jsonpb"
	_ "github.com/go-micro/plugins/codecs/proto"
	_ "github.com/go-micro/plugins/log/text"
	_ "github.com/go-micro/plugins/server/http/router/chi"
)

// TODO: test get path params

// TODO: for client, provide info on this error: 		t.Error("As:", errors.As(err, &x509.HostnameError{}))
//       >> change URL to proper hostname
// TODO: Provide context on unknown authority error for client x509.UnknownAuthorityError
//       >> Self signed cert was used

/*
Notes for HTTP/client << micro client
If scheme is HTTPS://
 > use tls dial & check proto for which transport to use
Else
 > use http1 transport without upgrade ()
*/

func TestServerSimple(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithInsecure())
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "http://0.0.0.0:42069"
	makeRequests(t, addr, thttp.TypeInsecure)
}

func TestServerHTTPS(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithDisableHTTP2())
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "https://localhost:42069"
	makeRequests(t, addr, thttp.TypeHTTP1)
}

func TestServerHTTP2(t *testing.T) {
	_, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "https://localhost:42069"
	makeRequests(t, addr, thttp.TypeHTTP2)
}

func TestServerH2c(t *testing.T) {
	_, cleanup, err := setupServer(t, false,
		mhttp.WithInsecure(),
		mhttp.WithAllowH2C(),
	)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "http://localhost:42069"
	makeRequests(t, addr, thttp.TypeH2C)
}

func TestServerHTTP3(t *testing.T) {
	// To fix warning about buf size run: sysctl -w net.core.rmem_max=2500000
	_, cleanup, err := setupServer(t, false,
		mhttp.WithHTTP3(),
	)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "https://localhost:42069"
	makeRequests(t, addr, thttp.TypeHTTP3)
}

func TestServerEntrypointsStarts(t *testing.T) {
	addr := "localhost:45451"
	server, cleanup, err := setupServer(t, false, mhttp.WithAddress(addr))
	if err != nil {
		t.Fatal(err)
	}

	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	addr = "https://" + addr
	makeRequests(t, addr, thttp.TypeHTTP2)

	cleanup()
	cleanup()
	cleanup()
}

func TestServerGzip(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithEnableGzip())
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "https://localhost:42069"
	makeRequests(t, addr, thttp.TypeHTTP2)
}

func TestServerInvalidContentType(t *testing.T) {
	_, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	addr := "https://localhost:42069"
	require.Error(t, thttp.TestPostRequestProto(t, addr, "application/abcdef", thttp.TypeHTTP2), "POST Proto")
	require.Error(t, thttp.TestPostRequestProto(t, addr, "yadayadayada", thttp.TypeHTTP2), "POST Proto")
}

func TestServerNoRouter(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithRouter(""))
	defer cleanup()
	t.Logf("expected error: %v", err)
	require.Error(t, err, "setting an empty router should return an error")

	_, cleanup, err = setupServer(t, false, mhttp.WithRouter("invalid-router"))
	defer cleanup()
	t.Logf("expected error: %v", err)
	require.Error(t, err, "setting an empty router should return an error")
}

func TestServerNoCodecs(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithCodecWhitelist([]string{}))
	defer cleanup()
	t.Logf("expected error: %v", err)
	require.Error(t, err, "setting an empty codec whitelist should return an error")

	_, cleanup, err = setupServer(t, false, mhttp.WithCodecWhitelist([]string{"abc", "def"}))
	defer cleanup()
	t.Logf("expected error: %v", err)
	require.Error(t, err, "setting an empty codec whitelist should return an error")
}

func TestServerNoTLS(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithTLSConfig(&tls.Config{}))
	defer cleanup()
	t.Logf("expected error: %v", err)
	require.Error(t, err, "setting an empty TLS config should return an error")
}

func TestServerNoAddr(t *testing.T) {
	_, cleanup, err := setupServer(t, false, mhttp.WithAddress(""))
	defer cleanup()
	t.Logf("expected error: %v", err)
	require.Error(t, err, "setting an empty address should return an error")
}

func TestServerInvalidMessage(t *testing.T) {
	_, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	thttp.RefreshClients()

	addr := "https://localhost:42069/echo"

	// Broken json.
	msg := `{"name": "Alex}`

	req, err := http.NewRequest(http.MethodPost, addr, bytes.NewReader([]byte(msg))) //nolint:noctx
	if err != nil {
		t.Fatalf("create POST request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := thttp.HTTP2Client.Do(req)
	assert.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	t.Logf("expected error: %v", string(body))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, string(body))
	assert.NoError(t, err)

	// Close connection
	_, err = io.Copy(io.Discard, resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
}

func TestServerErrorRPC(t *testing.T) {
	_, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	thttp.RefreshClients()

	addr := "https://localhost:42069/echo"

	msg := `{"name": "error"}`

	req, err := http.NewRequest(http.MethodPost, addr, bytes.NewReader([]byte(msg))) //nolint:noctx
	if err != nil {
		t.Fatalf("create POST request failed: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := thttp.HTTP2Client.Do(req)
	assert.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	t.Logf("expected error: %v", string(body))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, string(body))
	assert.NoError(t, err)

	// Close connection
	_, err = io.Copy(io.Discard, resp.Body)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
}

func TestServerRequestSpecificContentType(t *testing.T) {
	_, cleanup, err := setupServer(t, false)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}

	thttp.RefreshClients()

	addr := "https://localhost:42069/echo"
	msg := `{"name": "Alex"}`

	testCt := func(expectedCt string) {
		req, err := http.NewRequest(http.MethodPost, addr, bytes.NewReader([]byte(msg))) //nolint:noctx
		if err != nil {
			t.Fatalf("create POST request failed: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", expectedCt)

		resp, err := thttp.HTTP2Client.Do(req)
		assert.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// Validate conent type received
		assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))
		ct, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		assert.NoError(t, err)
		assert.Equal(t, expectedCt, ct, string(body))

		// Close connection
		_, err = io.Copy(io.Discard, resp.Body)
		assert.NoError(t, err)
		assert.NoError(t, resp.Body.Close())
	}

	testCt("application/proto")
	testCt("application/protobuf")
	testCt("application/x-proto")
	testCt("application/json")
	testCt("application/x-www-form-urlencoded")
}

func BenchmarkHTTPInsecureJSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeInsecure)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithInsecure())
}

func BenchmarkHTTPInseucreProto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeInsecure)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithInsecure())
}

func BenchmarkHTTP1JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithDisableHTTP2())
}

func BenchmarkHTTP1Form16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestGetRequest(tb, addr, thttp.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithDisableHTTP2())
}

func BenchmarkHTTP1Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithDisableHTTP2())
}

func BenchmarkHTTP2JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1)
}

func BenchmarkHTTP2Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1)
}

func BenchmarkHTTP3JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestJSON(tb, addr, thttp.TypeHTTP3)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithHTTP3())
}

func BenchmarkHTTP3PROTO16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return thttp.TestPostRequestProto(tb, addr, "application/octet-stream", thttp.TypeHTTP3)
	}

	benchmark(b, testFunc, 16, 1, mhttp.WithHTTP3())
}

func benchmark(b *testing.B, testFunc func(testing.TB, string) error, pN, sN int, opts ...mhttp.Option) {
	b.StopTimer()
	b.ReportAllocs()

	server, cleanup, err := setupServer(b, true, opts...)
	defer cleanup()
	if err != nil {
		b.Fatal(err)
	}

	addr := "https://localhost:42069"
	if server.Config.Insecure {
		addr = "http://localhost:42069"
	}

	runBenchmark(b, addr, testFunc, pN, sN)
}

func runBenchmark(b *testing.B, addr string, testFunc func(testing.TB, string) error, pN, sN int) {
	done := make(chan struct{})
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	b.ResetTimer()
	b.StartTimer()

	// Start requests
	go func() {
		for i := 0; i < b.N; i++ {
			thttp.RefreshClients()
			for p := 0; p < pN; p++ {
				wg.Add(1)
				go func() {
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

func setupServer(t testing.TB, nolog bool, opts ...mhttp.Option) (*mhttp.ServerHTTP, func(), error) {
	name := types.ServiceName("test-server")
	lopts := []log.Option{}
	if nolog {
		lopts = append(lopts, log.WithLevel(log.ErrorLevel))
	} else {
		lopts = append(lopts, log.WithLevel(log.DebugLevel))
	}

	cancel := func() {}

	logger, err := log.ProvideLogger(name, nil, lopts...)
	if err != nil {
		return nil, cancel, fmt.Errorf("failed to setup logger: %w", err)
	}

	h := new(handler.EchoHandler)
	opts = append(opts,
		mhttp.WithRegistrations(
			proto.RegisterFuncStreams(h),
		),
	)

	cfg, err := mhttp.NewDefaultConfig(name, nil, opts...)
	if err != nil {
		return nil, cancel, fmt.Errorf("failed to create config: %w", err)
	}

	server, err := mhttp.ProvideServerHTTP("http-test", name, nil, logger, cfg)
	if err != nil {
		return nil, cancel, fmt.Errorf("failed to provide http server: %w", err)
	}

	cleanup := func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatalf("failed to stop: %v", err)
		}
	}

	if err := server.Start(); err != nil {
		return nil, cancel, fmt.Errorf("failed to start: %w", err)
	}

	return server, cleanup, nil
}

func makeRequests(t *testing.T, addr string, reqType thttp.ReqType) {
	require.NoError(t, thttp.TestGetRequest(t, addr, reqType), "GET")
	require.NoError(t, thttp.TestPostRequestJSON(t, addr, reqType), "POST JSON")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/octet-stream", reqType), "POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/proto", reqType), "POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/x-proto", reqType), "POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/protobuf", reqType), "POST Proto")
	require.NoError(t, thttp.TestPostRequestProto(t, addr, "application/x-protobuf", reqType), "POST Proto")
}
