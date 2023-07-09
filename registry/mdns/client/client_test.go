package client

import (
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	// 	"github.com/go-orb/plugins/registry/mdns/client"
	"github.com/go-orb/plugins/registry/mdns/server"
	"github.com/go-orb/plugins/registry/mdns/zone"
)

func makeService(t *testing.T) *zone.MDNSService {
	t.Helper()
	return makeServiceWithServiceName(t, "_http._tcp")
}

func makeServiceWithServiceName(t *testing.T, service string) *zone.MDNSService {
	t.Helper()

	m, err := zone.NewMDNSService(
		"hostname",
		service,
		"local.",
		"testhost.",
		80, // port
		[]net.IP{net.IP([]byte{192, 168, 0, 42}), net.ParseIP("2620:0:1000:1900:b0c2:d0b2:c411:18bc")},
		[]string{"Local web server"},
	) // TXT
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	return m
}

func TestServer_StartStop(t *testing.T) {
	s := makeService(t)
	serv, err := server.NewServer(&server.Config{Zone: s, LocalhostChecking: true})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer func() {
		if err := serv.Shutdown(); err != nil {
			t.Error(err)
		}
	}()
}

func TestServer_Lookup(t *testing.T) {
	serv, err := server.NewServer(
		&server.Config{
			Zone:              makeServiceWithServiceName(t, "_foobar._tcp"),
			LocalhostChecking: true,
		})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer func() {
		if err = serv.Shutdown(); err != nil {
			t.Error(err)
		}
	}()

	entries := make(chan *ServiceEntry, 1)
	found := false
	doneCh := make(chan struct{})
	errChan := make(chan error)

	go func() {
		select {
		case e := <-entries:
			if e.Name != "hostname._foobar._tcp.local." {
				errChan <- fmt.Errorf("bad: %v", e)
			}
			if e.Port != 80 {
				errChan <- fmt.Errorf("bad: %v", e)
			}
			if e.Info != "Local web server" {
				errChan <- fmt.Errorf("bad: %v", e)
			}
			found = true

		case <-time.After(80 * time.Millisecond):
			errChan <- errors.New("timeout")
		}
		close(doneCh)
	}()

	params := &QueryParam{
		Service: "_foobar._tcp",
		Domain:  "local",
		Timeout: 50 * time.Millisecond,
		Entries: entries,
	}
	err = Query(params)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	select {
	case err := <-errChan:
		t.Fatalf("test failed: %v", err)
	case <-doneCh:
		<-doneCh
	}
	if !found {
		t.Fatalf("record not found")
	}
}
