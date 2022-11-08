package http

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-micro.dev/v5/log"
	"go-micro.dev/v5/types"

	"github.com/go-micro/plugins/server/http/router/router"
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
	server, router := setupServer(t, false, WithInsecure())

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	defer func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

	addr := "http://0.0.0.0:42069"

	if err := tests.TestGetRequest(t, addr, tests.TypeInsecure); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestJSON(t, addr, tests.TypeInsecure); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/octet-stream", tests.TypeInsecure); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/proto", tests.TypeInsecure); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/x-proto", tests.TypeInsecure); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/protobuf", tests.TypeInsecure); err != nil {
		t.Fatal(err)
	}
}

func TestServerHTTPS(t *testing.T) {
	server, router := setupServer(t, false, WithDisableHTTP2())

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	defer func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

	addr := "https://localhost:42069"

	if err := tests.TestGetRequest(t, addr, tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestJSON(t, addr, tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/octet-stream", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/proto", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/x-proto", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/protobuf", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}
}

func TestServerHTTP2(t *testing.T) {
	server, router := setupServer(t, false)

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	defer func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

	addr := "https://localhost:42069"

	if err := tests.TestGetRequest(t, addr, tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestJSON(t, addr, tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/octet-stream", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/proto", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/x-proto", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/protobuf", tests.TypeHTTP2); err != nil {
		t.Fatal(err)
	}
}

func TestServerH2c(t *testing.T) {
	server, router := setupServer(t, false,
		WithInsecure(),
		WithAllowH2C(),
	)

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	defer func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

	addr := "http://localhost:42069"

	if err := tests.TestGetRequest(t, addr, tests.TypeH2C); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestJSON(t, addr, tests.TypeH2C); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/octet-stream", tests.TypeH2C); err != nil {
		t.Fatal(err)
	}
}

func TestServerHTTP3(t *testing.T) {
	// To fix warning about buf size run: sysctl -w net.core.rmem_max=2500000
	server, router := setupServer(t, false,
		WithHTTP3(),
	)

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		t.Fatal("failed to start", err)
	}

	defer func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*5))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			t.Fatal("failed to stop", err)
		}
	}()

	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

	addr := "https://localhost:42069"

	if err := tests.TestGetRequest(t, addr, tests.TypeHTTP3); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestJSON(t, addr, tests.TypeHTTP3); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/octet-stream", tests.TypeHTTP3); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/proto", tests.TypeHTTP3); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/x-proto", tests.TypeHTTP3); err != nil {
		t.Fatal(err)
	}

	if err := tests.TestPostRequestProto(t, addr, "application/protobuf", tests.TypeHTTP3); err != nil {
		t.Fatal(err)
	}
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

	server, router := setupServer(b, true, opts...)

	h := new(handler.EchoHandler)
	if err := server.Start(); err != nil {
		b.Fatal("failed to start", err)
	}

	defer func() {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			b.Fatal("failed to stop", err)
		}
	}()

	router.Get("/echo", NewGRPCHandler(server, h.Call))
	router.Post("/echo", NewGRPCHandler(server, h.Call))

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

func setupServer(t testing.TB, nolog bool, opts ...Option) (*Server, router.Router) {
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

	return server, server.Router()
}
