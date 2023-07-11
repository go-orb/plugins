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
	// TODO: As long as https://github.com/lucas-clemente/quic-go/issues/765
	//       exists, the client cannot be re-used.
	//nolint:unused
	http3Client   *http.Client
	httpH2CClient *http.Client
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
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}
}

// TestGetRequest makes a GET request to the echo endpoint.
func TestGetRequest(t testing.TB, addr string, reqT ReqType) error {
	url := fmt.Sprintf("%s/echo?name=%s", addr, testName)

	body, err := switchRequest(t, url, "", nil, makeGetReq, reqT)
	if err != nil {
		return err
	}

	return checkJSONResponse(body, testName)
}

// TestPostRequestJSON makes a POST request to the echo endpoint.
func TestPostRequestJSON(t testing.TB, addr string, reqT ReqType) error {
	msg, err := json.Marshal(map[string]string{"name": testName})
	if err != nil {
		t.Fatal("failed to marshall json", err)
	}

	addr += "/echo"
	ct := headers.JSONContentType

	body, err := switchRequest(t, addr, ct, msg, makePostReq, reqT)
	if err != nil {
		return err
	}

	return checkJSONResponse(body, testName)
}

// TestPostRequestProto makes a POST request to the echo endpoint.
func TestPostRequestProto(t testing.TB, addr, ct string, reqT ReqType) error {
	name := "Alex"

	msg, err := proto.Marshal(&pb.CallRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	addr += "/echo"

	body, err := switchRequest(t, addr, ct, msg, makePostReq, reqT)
	if err != nil {
		return err
	}

	return checkProtoResponse(body, name)
}

// TestTLSProto temporary test stuff.
func TestTLSProto(t testing.TB, addr string) error {
	t.Log("Testing TLS")

	conn, err := tls.Dial("tcp", addr, &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec
		NextProtos:         []string{"HTTP/3.0", "My custom proto", "ClientGarbage"},
	})
	if err != nil {
		return fmt.Errorf("failed to dial TLS tcp connection: %w", err)
	}

	state := conn.ConnectionState()
	t.Log(state.NegotiatedProtocol)

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

	if data.Msg != "Hello "+name {
		return fmt.Errorf("request failed; expected different response than: %v", data.Msg)
	}

	return nil
}

func makeGetReq(t testing.TB, addr, _ string, _ []byte, client *http.Client) ([]byte, error) {
	// NOTE: this would be nice to use, but gices TCP errors
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	// defer cancel()
	//
	// req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	// if err != nil {
	// 	return nil, fmt.Errorf("create GET request failed: %w", err)
	// }
	//
	// resp, err := client.Do(req)
	resp, err := client.Get(addr) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to make GET request: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			t.Errorf("failed to close body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logResponse(t, resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET request failed, status not OK: %+v", resp)
	}

	return body, nil
}

func makePostReq(t testing.TB, addr, ct string, data []byte, client *http.Client) ([]byte, error) {
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
	resp, err := client.Post(addr, ct, bytes.NewReader(data)) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("failed to make POST request: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			t.Errorf("failed to close body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	logResponse(t, resp, body)

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Post request failed")
	}

	return body, nil
}

func switchRequest(tb testing.TB, url, ct string, msg []byte, reqFunc ReqFunc, reqT ReqType) ([]byte, error) {
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
		// Required because of issue quic-go#765
		client := http.Client{
			Transport: &http3.RoundTripper{
				TLSClientConfig: &tls.Config{
					//nolint:gosec
					InsecureSkipVerify: true,
				},
			},
		}
		body, err = reqFunc(tb, url, ct, msg, &client)
	case TypeH2C:
		body, err = reqFunc(tb, url, ct, msg, httpH2CClient)
	}

	return body, err
}

func logResponse(tb testing.TB, resp *http.Response, body []byte) {
	// only log if not benchmark
	if t, ok := tb.(*testing.T); ok && len(os.Getenv("MICRO_DEBUG")) > 0 {
		t.Logf(
			"[%+v] Status: %v, \n\tProto: %v, ConentType: %v, Length: %v, \n\tTransferEncoding: %v, Uncompressed: %v, \n\tBody: %v",
			resp.Request.Method, resp.Status, resp.Proto, resp.Header.Get("Content-Type"),
			resp.ContentLength, resp.TransferEncoding, resp.Uncompressed, string(body),
		)
	}
}
