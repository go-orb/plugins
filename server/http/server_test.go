package http

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/types"

	"github.com/stretchr/testify/require"

	"github.com/go-micro/plugins/server/http/utils/tests"
	"github.com/go-micro/plugins/server/http/utils/tests/handler"

	_ "github.com/go-micro/plugins/codecs/form"
	_ "github.com/go-micro/plugins/codecs/jsonpb"
	_ "github.com/go-micro/plugins/codecs/proto"
	_ "github.com/go-micro/plugins/log/text"

	_ "github.com/go-micro/plugins/server/http/router/chi"
)

// TODO: test if asking for specific content type back works
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
	_, cleanup := setupServer(t, false, WithInsecure())
	defer cleanup()

	addr := "http://0.0.0.0:42069"
	makeRequests(t, addr, tests.TypeInsecure)
}

func TestServerHTTPS(t *testing.T) {
	_, cleanup := setupServer(t, false, WithDisableHTTP2())
	defer cleanup()

	addr := "https://localhost:42069"
	makeRequests(t, addr, tests.TypeHTTP1)
}

func TestServerHTTP2(t *testing.T) {
	_, cleanup := setupServer(t, false)
	defer cleanup()

	addr := "https://localhost:42069"
	makeRequests(t, addr, tests.TypeHTTP2)
}

func TestServerH2c(t *testing.T) {
	_, cleanup := setupServer(t, false,
		WithInsecure(),
		WithAllowH2C(),
	)
	defer cleanup()

	addr := "http://localhost:42069"
	makeRequests(t, addr, tests.TypeH2C)
}

func TestServerHTTP3(t *testing.T) {
	// To fix warning about buf size run: sysctl -w net.core.rmem_max=2500000
	_, cleanup := setupServer(t, false,
		WithHTTP3(),
	)
	defer cleanup()

	addr := "https://localhost:42069"
	makeRequests(t, addr, tests.TypeHTTP3)
}

func TestServerMultipleEntrypoints(t *testing.T) {
	addrs := []string{"localhost:45451", "localhost:45452", "localhost:45453", "localhost:45454", "localhost:45455"}
	_, cleanup := setupServer(t, false, WithAddress(addrs...))
	defer cleanup()

	for _, addr := range addrs {
		addr = "https://" + addr
		makeRequests(t, addr, tests.TypeHTTP2)
	}
}

func TestServerEntrypointsStarts(t *testing.T) {
	addrs := []string{"localhost:45451", "localhost:45452", "localhost:45453", "localhost:45454", "localhost:45455"}
	server, cleanup := setupServer(t, false, WithAddress(addrs...))

	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	for _, addr := range addrs {
		addr = "https://" + addr
		makeRequests(t, addr, tests.TypeHTTP2)
	}

	cleanup()
	cleanup()
	cleanup()
}

func TestServerGzip(t *testing.T) {
	_, cleanup := setupServer(t, false, WithGzip())
	defer cleanup()

	addr := "https://localhost:42069"
	makeRequests(t, addr, tests.TypeHTTP2)
}

func TestServerInvalidContentType(t *testing.T) {
	_, cleanup := setupServer(t, false, WithGzip())
	defer cleanup()

	addr := "https://localhost:42069"
	require.Error(t, tests.TestPostRequestProto(t, addr, "application/abcdef", tests.TypeHTTP2), "POST Proto")
}

func BenchmarkHTTPInsecureJSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestJSON(tb, addr, tests.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1, WithInsecure())
}

func BenchmarkHTTPInseucreProto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestProto(tb, addr, "application/octet-stream", tests.TypeInsecure)
	}

	benchmark(b, testFunc, 16, 1, WithInsecure())
}

func BenchmarkHTTP1JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestJSON(tb, addr, tests.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, WithDisableHTTP2())
}

func BenchmarkHTTP1Form16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestGetRequest(tb, addr, tests.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, WithDisableHTTP2())
}

func BenchmarkHTTP1Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestProto(tb, addr, "application/octet-stream", tests.TypeHTTP1)
	}

	benchmark(b, testFunc, 16, 1, WithDisableHTTP2())
}

func BenchmarkHTTP2JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestJSON(tb, addr, tests.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1)
}

func BenchmarkHTTP2Proto16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestProto(tb, addr, "application/octet-stream", tests.TypeHTTP2)
	}

	benchmark(b, testFunc, 16, 1)
}

func BenchmarkHTTP3JSON16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestJSON(tb, addr, tests.TypeHTTP3)
	}

	benchmark(b, testFunc, 16, 1, WithHTTP3())
}

func BenchmarkHTTP3PROTO16(b *testing.B) {
	testFunc := func(tb testing.TB, addr string) error {
		return tests.TestPostRequestProto(tb, addr, "application/octet-stream", tests.TypeHTTP3)
	}

	benchmark(b, testFunc, 16, 1, WithHTTP3())
}

func benchmark(b *testing.B, testFunc func(testing.TB, string) error, pN, sN int, opts ...Option) {
	b.StopTimer()
	b.ReportAllocs()

	server, cleanup := setupServer(b, true, opts...)
	defer cleanup()

	addr := "https://localhost:42069"
	if server.Config.EntrypointDefaults.Insecure {
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
			tests.RefreshClients()
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

func setupServer(t testing.TB, nolog bool, opts ...Option) (*Server, func()) {
	name := types.ServiceName("test-server")
	lopts := []log.Option{}
	if nolog {
		lopts = append(lopts, log.WithLevel(log.ErrorLevel))
	}
	logger, err := log.ProvideLogger(name, nil, lopts...)
	if err != nil {
		t.Fatalf("failed to setup logger: %v", err)
	}

	server, err := ProvideServerHTTP(name, nil, logger, opts...)
	if err != nil {
		t.Fatalf("failed to provide http server: %v", err)
	}

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	cleanup := func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatal("failed to stop", err)
		}
	}

	router := server.Router()
	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

	return server, cleanup
}

func makeRequests(t *testing.T, addr string, reqType tests.ReqType) {
	require.NoError(t, tests.TestGetRequest(t, addr, reqType), "GET")
	require.NoError(t, tests.TestPostRequestJSON(t, addr, reqType), "POST JSON")
	require.NoError(t, tests.TestPostRequestProto(t, addr, "application/octet-stream", reqType), "POST Proto")
	require.NoError(t, tests.TestPostRequestProto(t, addr, "application/proto", reqType), "POST Proto")
	require.NoError(t, tests.TestPostRequestProto(t, addr, "application/x-proto", reqType), "POST Proto")
	require.NoError(t, tests.TestPostRequestProto(t, addr, "application/protobuf", reqType), "POST Proto")
	require.NoError(t, tests.TestPostRequestProto(t, addr, "application/x-protobuf", reqType), "POST Proto")
}
