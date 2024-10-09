// Package http provides testing utilities.
package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"

	"github.com/go-orb/plugins/server/http/headers"
	pb "github.com/go-orb/plugins/server/http/tests/proto"
)

// ReqType set the HTTP request type to make.
type ReqType int

// ReqFunc sets thd request function to test.
type ReqFunc func(testing.TB, string, string, []byte, *http.Client) ([]byte, error)

// Request types.
const (
	TypeInsecure ReqType = iota + 1
	TypeHTTP1
	TypeHTTP2
	TypeHTTP3
	TypeH2C
)

const (
	testName = "Alex"
)

// HTTP Clients for reuse, to pool connections during tests.
//
//nolint:gochecknoglobals
var (
	httpInsecureClient *http.Client
	http1Client        *http.Client
	HTTP2Client        *http.Client
	http3Client        *http.Client
	httpH2CClient      *http.Client
)

func init() {
	RefreshClients()
}

// RefreshClients creates new clients for the HTTP requests to tetst with.
func RefreshClients() {
	httpInsecureClient = &http.Client{}

	http1Client = &http.Client{
		Timeout: time.Second * 5,
		Transport: &http.Transport{
			// MaxIdleConns:        64,
			// MaxIdleConnsPerHost: 64,
			// TLSHandshakeTimeout: 700 * time.Millisecond,
			ForceAttemptHTTP2: false,
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		},
	}

	HTTP2Client = &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		},
	}

	http3Client = &http.Client{
		Transport: &http3.RoundTripper{
			TLSClientConfig: &tls.Config{
				//nolint:gosec
				InsecureSkipVerify: true,
			},
		},
	}

	httpH2CClient = &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
}

// TestPostRequestJSON makes a POST request to the echo endpoint.
func TestPostRequestJSON(tb testing.TB, addr string, reqT ReqType) error {
	tb.Helper()

	msg, err := json.Marshal(map[string]string{"name": testName})
	if err != nil {
		tb.Fatal("failed to marshall json", err)
	}

	addr += "/echo.Streams/Call"
	ct := headers.JSONContentType

	body, err := switchRequest(tb, addr, ct, msg, makePostReq, reqT)
	if err != nil {
		return err
	}

	return checkJSONResponse(body, testName)
}

// TestPostRequestProto makes a POST request to the echo endpoint.
func TestPostRequestProto(tb testing.TB, addr, ct string, reqT ReqType) error {
	tb.Helper()

	name := "Alex"

	msg, err := proto.Marshal(&pb.CallRequest{Name: name})
	if err != nil {
		return err
	}

	addr += "/echo.Streams/Call"

	body, err := switchRequest(tb, addr, ct, msg, makePostReq, reqT)
	if err != nil {
		return err
	}

	return checkProtoResponse(body, name)
}

// TestTLSProto temporary test stuff.
func TestTLSProto(tb testing.TB, addr string) error {
	tb.Helper()
	tb.Log("Testing TLS")

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec
		NextProtos:         []string{"HTTP/3.0", "My custom proto", "ClientGarbage"},
	})
	if err != nil {
		return fmt.Errorf("failed to dial TLS tcp connection: %w", err)
	}

	state := conn.ConnectionState()
	tb.Log(state.NegotiatedProtocol)

	return nil
}

func checkJSONResponse(body []byte, name string) error {
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil {
		return errors.Wrap(err, "Failed to unmarhsal data")
	}

	if data["msg"] != "Hello "+name {
		return fmt.Errorf("request failed; expected different response than: %v", data)
	}

	return nil
}

func checkProtoResponse(body []byte, name string) error {
	var data pb.CallResponse
	if err := proto.Unmarshal(body, &data); err != nil {
		return errors.Wrap(err, "Failed to unmarhsal data")
	}

	if data.GetMsg() != "Hello "+name {
		return fmt.Errorf("request failed; expected different response than: %v", data.GetMsg())
	}

	return nil
}

func makePostReq(tb testing.TB, addr, ct string, data []byte, client *http.Client) ([]byte, error) {
	// NOTE: this would be nice to use, but gices TCP errors when a context with timeout is passed in.
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// // ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
	// // ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	//
	// req, err := http.NewRequestWithContext(ctx, http.MethodPost, addr, bytes.NewReader(data))
	// if err != nil {
	// 	return nil, fmt.Errorf("create POST request failed: %w", err)
	// }
	//
	// req.Header.Set("Content-Type", ct)
	// // req.Close = true
	//
	// resp, err := client.Do(req)
	tb.Helper()

	resp, err := client.Post(addr, ct, bytes.NewReader(data)) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to make POST request: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			tb.Errorf("failed to close body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logResponse(tb, resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Post request failed")
	}

	return body, nil
}

func switchRequest(tb testing.TB, url, ct string, msg []byte, reqFunc ReqFunc, reqT ReqType) ([]byte, error) {
	tb.Helper()

	var (
		body []byte
		err  error
	)

	switch reqT {
	case TypeInsecure:
		body, err = reqFunc(tb, url, ct, msg, httpInsecureClient)
	case TypeHTTP1:
		body, err = reqFunc(tb, url, ct, msg, http1Client)
	case TypeHTTP2:
		body, err = reqFunc(tb, url, ct, msg, HTTP2Client)
	case TypeHTTP3:
		body, err = reqFunc(tb, url, ct, msg, http3Client)
	case TypeH2C:
		body, err = reqFunc(tb, url, ct, msg, httpH2CClient)
	}

	return body, err
}

func logResponse(tb testing.TB, resp *http.Response, body []byte) {
	tb.Helper()

	// only log if not benchmark
	if t, ok := tb.(*testing.T); ok && len(os.Getenv("MICRO_DEBUG")) > 0 {
		t.Logf(
			"[%+v] Status: %v, \n\tProto: %v, ConentType: %v, Length: %v, \n\tTransferEncoding: %v, Uncompressed: %v, \n\tBody: %v",
			resp.Request.Method, resp.Status, resp.Proto, resp.Header.Get("Content-Type"),
			resp.ContentLength, resp.TransferEncoding, resp.Uncompressed, string(body),
		)
	}
}
