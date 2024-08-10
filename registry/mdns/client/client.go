// Package client provides utility functions for the mdns registry.
package client

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/plugins/registry/mdns/server"
	"github.com/go-orb/plugins/registry/mdns/util"
	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// ServiceEntry is returned after we query for a service.
type ServiceEntry struct {
	Name       string
	Host       string
	AddrV4     net.IP
	AddrV6     net.IP
	Port       int
	Info       string
	InfoFields []string
	TTL        int
	Type       uint16

	hasTXT bool
	sent   bool
}

// complete is used to check if we have all the info we need.
func (s *ServiceEntry) complete() bool {
	return (len(s.AddrV4) > 0 || len(s.AddrV6) > 0) && s.Port != 0 && s.hasTXT
}

// QueryParam is used to customize how a Lookup is performed.
type QueryParam struct {
	Service             string               // Service to lookup
	Domain              string               // Lookup domain, default "local"
	Type                uint16               // Lookup type, defaults to dns.TypePTR
	Context             context.Context      // Context
	Timeout             time.Duration        // Lookup timeout, default 1 second. Ignored if Context is provided
	Interface           *net.Interface       // Multicast interface to use
	Entries             chan<- *ServiceEntry // Entries Channel
	WantUnicastResponse bool                 // Unicast response desired, as per 5.4 in RFC
}

// DefaultParams is used to return a default set of QueryParam's.
func DefaultParams(service string) *QueryParam {
	return &QueryParam{
		Service:             service,
		Domain:              "local",
		Timeout:             time.Second,
		Entries:             make(chan *ServiceEntry),
		WantUnicastResponse: false, // TODO(reddaly): Change this default.
	}
}

// Query looks up a given service, in a domain, waiting at most
// for a timeout before finishing the query. The results are streamed
// to a channel. Sends will not block, so clients should make sure to
// either read or buffer.
func Query(params *QueryParam) error {
	// Create a new client
	client, err := newClient()
	if err != nil {
		return err
	}
	defer client.Close() //nolint:errcheck

	// Set the multicast interface
	if params.Interface != nil {
		if err = client.setInterface(params.Interface, false); err != nil {
			return err
		}
	}

	// Ensure defaults are set
	if params.Domain == "" {
		params.Domain = "local"
	}

	if params.Context == nil {
		if params.Timeout == 0 {
			params.Timeout = time.Second
		}

		var cancel context.CancelFunc

		params.Context, cancel = context.WithTimeout(context.Background(), params.Timeout)
		defer cancel()

		if err != nil {
			return err
		}
	}

	// Run the query
	return client.query(params)
}

// Listen listens indefinitely for multicast updates.
func Listen(entries chan<- *ServiceEntry, exit chan struct{}) error {
	// Create a new client
	client, err := newClient()
	if err != nil {
		return err
	}
	defer client.Close() //nolint:errcheck

	// client.setInterface(nil, true)

	// Start listening for response packets
	msgCh := make(chan *dns.Msg, 32)

	go client.recv(client.ipv4UnicastConn, msgCh)
	go client.recv(client.ipv6UnicastConn, msgCh)
	go client.recv(client.ipv4MulticastConn, msgCh)
	go client.recv(client.ipv6MulticastConn, msgCh)

	ip := make(map[string]*ServiceEntry)

	for {
		select {
		case <-exit:
			return nil
		case <-client.closedCh:
			return nil
		case m := <-msgCh:
			e := messageToEntry(m, ip)
			if e == nil {
				continue
			}

			// Check if this entry is complete
			if e.complete() {
				if e.sent {
					continue
				}

				e.sent = true
				entries <- e

				ip = make(map[string]*ServiceEntry)
			} else {
				// Fire off a node specific query
				m := new(dns.Msg)
				m.SetQuestion(e.Name, dns.TypePTR)
				m.RecursionDesired = false

				if err := client.sendQuery(m); err != nil {
					log.Error("[mdns] failed to query instance %s: %v", err, e.Name)
				}
			}
		}
	}
}

// Lookup is the same as Query, however it uses all the default parameters.
func Lookup(service string, entries chan<- *ServiceEntry) error {
	params := DefaultParams(service)
	params.Entries = entries

	return Query(params)
}

// Client provides a query interface that can be used to
// search for service providers using mDNS.
type client struct {
	ipv4UnicastConn *net.UDPConn
	ipv6UnicastConn *net.UDPConn

	ipv4MulticastConn *net.UDPConn
	ipv6MulticastConn *net.UDPConn

	closed    bool
	closedCh  chan struct{} // TODO(reddaly): This doesn't appear to be used.
	closeLock sync.Mutex
}

// NewClient creates a new mdns Client that can be used to query
// for records.
func newClient() (*client, error) { //nolint:funlen,gocyclo
	// TODO(reddaly): At least attempt to bind to the port required in the spec.
	// Create a IPv4 listener
	uconn4, err4 := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	uconn6, err6 := net.ListenUDP("udp6", &net.UDPAddr{IP: net.IPv6zero, Port: 0})

	if err4 != nil && err6 != nil {
		log.Error("[mdns] failed to bind to udp port: %v %v", err4, err6)
	}

	if uconn4 == nil && uconn6 == nil {
		return nil, errors.New("failed to bind to any unicast udp port")
	}

	if uconn4 == nil {
		uconn4 = &net.UDPConn{}
	}

	if uconn6 == nil {
		uconn6 = &net.UDPConn{}
	}

	mconn4, err4 := net.ListenUDP("udp4", server.MDNSWildcardAddrIPv4)
	mconn6, err6 := net.ListenUDP("udp6", server.MDNSWildcardAddrIPv6)

	if err4 != nil && err6 != nil {
		log.Error("[mdns] failed to bind to udp port: %v %v", err4, err6)
	}

	if mconn4 == nil && mconn6 == nil {
		return nil, errors.New("failed to bind to any multicast udp port")
	}

	if mconn4 == nil {
		mconn4 = &net.UDPConn{}
	}

	if mconn6 == nil {
		mconn6 = &net.UDPConn{}
	}

	p1 := ipv4.NewPacketConn(mconn4)
	p2 := ipv6.NewPacketConn(mconn6)

	if err := p1.SetMulticastLoopback(true); err != nil {
		return nil, err
	}

	if err := p2.SetMulticastLoopback(true); err != nil {
		return nil, err
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Check if we manage to succeed on atleast one interface.
	success := false

	for _, iface := range ifaces {
		i := iface
		if err := p1.JoinGroup(&i, &net.UDPAddr{IP: server.MDNSGroupIPv4}); err != nil {
			continue
		}

		i = iface
		if err := p2.JoinGroup(&i, &net.UDPAddr{IP: server.MDNSGroupIPv6}); err != nil {
			continue
		}

		success = true
	}

	if !success {
		return nil, errors.New("failed to join multicast group on all interfaces")
	}

	c := &client{
		ipv4MulticastConn: mconn4,
		ipv6MulticastConn: mconn6,
		ipv4UnicastConn:   uconn4,
		ipv6UnicastConn:   uconn6,
		closedCh:          make(chan struct{}),
	}

	return c, nil
}

// Close is used to cleanup the client.
func (c *client) Close() error {
	c.closeLock.Lock()
	defer c.closeLock.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true

	close(c.closedCh)

	var gerr error

	if c.ipv4UnicastConn != nil {
		if err := c.ipv4UnicastConn.Close(); err != nil {
			gerr = fmt.Errorf("close ipv4 unicast connection: %w", err)
		}
	}

	if c.ipv6UnicastConn != nil {
		if err := c.ipv6UnicastConn.Close(); err != nil {
			gerr = fmt.Errorf("close ipv6 unicast connection: %w", err)
		}
	}

	if c.ipv4MulticastConn != nil {
		if err := c.ipv4MulticastConn.Close(); err != nil {
			gerr = fmt.Errorf("close ipv4 multicast connection: %w", err)
		}
	}

	if c.ipv6MulticastConn != nil {
		if err := c.ipv6MulticastConn.Close(); err != nil {
			gerr = fmt.Errorf("close ipv6 multicast connection: %w", err)
		}
	}

	return gerr
}

// setInterface is used to set the query interface, uses system
// default if not provided.
func (c *client) setInterface(iface *net.Interface, loopback bool) error {
	p := ipv4.NewPacketConn(c.ipv4UnicastConn)
	if err := p.JoinGroup(iface, &net.UDPAddr{IP: server.MDNSGroupIPv4}); err != nil {
		return err
	}

	p2 := ipv6.NewPacketConn(c.ipv6UnicastConn)
	if err := p2.JoinGroup(iface, &net.UDPAddr{IP: server.MDNSGroupIPv6}); err != nil {
		return err
	}

	p = ipv4.NewPacketConn(c.ipv4MulticastConn)
	if err := p.JoinGroup(iface, &net.UDPAddr{IP: server.MDNSGroupIPv4}); err != nil {
		return err
	}

	p2 = ipv6.NewPacketConn(c.ipv6MulticastConn)
	if err := p2.JoinGroup(iface, &net.UDPAddr{IP: server.MDNSGroupIPv6}); err != nil {
		return err
	}

	if loopback {
		if err := p.SetMulticastLoopback(true); err != nil {
			return err
		}

		if err := p2.SetMulticastLoopback(true); err != nil {
			return err
		}
	}

	return nil
}

// query is used to perform a lookup and stream results.
func (c *client) query(params *QueryParam) error {
	// Create the service name
	serviceAddr := fmt.Sprintf("%s.%s.", util.TrimDot(params.Service), util.TrimDot(params.Domain))

	// Start listening for response packets
	msgCh := make(chan *dns.Msg, 32)
	go c.recv(c.ipv4UnicastConn, msgCh)
	go c.recv(c.ipv6UnicastConn, msgCh)
	go c.recv(c.ipv4MulticastConn, msgCh)
	go c.recv(c.ipv6MulticastConn, msgCh)

	// Send the query
	msg := new(dns.Msg)
	if params.Type == dns.TypeNone {
		msg.SetQuestion(serviceAddr, dns.TypePTR)
	} else {
		msg.SetQuestion(serviceAddr, params.Type)
	}
	// RFC 6762, section 18.12.  Repurposing of Top Bit of qclass in Question
	// Section
	//
	// In the Question Section of a Multicast DNS query, the top bit of the qclass
	// field is used to indicate that unicast responses are preferred for this
	// particular question.  (See Section 5.4.)
	if params.WantUnicastResponse {
		msg.Question[0].Qclass |= 1 << 15
	}

	msg.RecursionDesired = false

	if err := c.sendQuery(msg); err != nil {
		return err
	}

	// Map the in-progress responses
	inprogress := make(map[string]*ServiceEntry)

	for {
		select {
		case resp := <-msgCh:
			inp := messageToEntry(resp, inprogress)

			if inp == nil {
				continue
			}

			if len(resp.Question) == 0 || resp.Question[0].Name != msg.Question[0].Name {
				// discard anything which we've not asked for
				continue
			}

			// Check if this entry is complete
			if inp.complete() {
				if inp.sent {
					continue
				}

				inp.sent = true
				select {
				case params.Entries <- inp:
				case <-params.Context.Done():
					return nil
				}
			} else {
				// Fire off a node specific query
				m := new(dns.Msg)
				m.SetQuestion(inp.Name, inp.Type)
				m.RecursionDesired = false

				if err := c.sendQuery(m); err != nil {
					log.Error("[mdns] failed to query instance %s", err, inp.Name)
				}
			}
		case <-params.Context.Done():
			return nil
		}
	}
}

// sendQuery is used to multicast a query out.
func (c *client) sendQuery(q *dns.Msg) error {
	buf, err := q.Pack()
	if err != nil {
		return err
	}

	if c.ipv4UnicastConn != nil {
		if _, err := c.ipv4UnicastConn.WriteToUDP(buf, server.IPv4Addr); err != nil {
			return err
		}
	}

	if c.ipv6UnicastConn != nil {
		if _, err := c.ipv6UnicastConn.WriteToUDP(buf, server.IPv6Addr); err != nil {
			return err
		}
	}

	return nil
}

// recv is used to receive until we get a shutdown.
func (c *client) recv(l *net.UDPConn, msgCh chan *dns.Msg) {
	if l == nil {
		return
	}

	buf := make([]byte, 65536)

	for {
		c.closeLock.Lock()
		if c.closed {
			c.closeLock.Unlock()
			return
		}

		c.closeLock.Unlock()

		n, err := l.Read(buf)
		if err != nil {
			continue
		}

		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			continue
		}
		select {
		case msgCh <- msg:
		case <-c.closedCh:
			return
		}
	}
}

// entryFromMap is used to get or create a entry from/into the map inprogress.
func entryFromMap(inprogress map[string]*ServiceEntry, name string, typ uint16) *ServiceEntry {
	if inp, ok := inprogress[name]; ok {
		return inp
	}

	inp := &ServiceEntry{
		Name: name,
		Type: typ,
	}

	inprogress[name] = inp

	return inp
}

func messageToEntry(m *dns.Msg, inprogress map[string]*ServiceEntry) *ServiceEntry {
	var inp *ServiceEntry

	for _, answer := range append(m.Answer, m.Extra...) {
		// TODO(reddaly): Check that response corresponds to serviceAddr?
		switch dnsAnswer := answer.(type) {
		case *dns.PTR:
			// Create new entry for this
			inp = entryFromMap(inprogress, dnsAnswer.Ptr, dnsAnswer.Hdr.Rrtype)
			if inp.complete() {
				continue
			}
		case *dns.SRV:
			// Check for a target mismatch
			if dnsAnswer.Target != dnsAnswer.Hdr.Name {
				srcEntry := entryFromMap(inprogress, dnsAnswer.Hdr.Name, dnsAnswer.Hdr.Rrtype)
				inprogress[dnsAnswer.Target] = srcEntry
			}

			// Get the port
			inp = entryFromMap(inprogress, dnsAnswer.Hdr.Name, dnsAnswer.Hdr.Rrtype)
			if inp.complete() {
				continue
			}

			inp.Host = dnsAnswer.Target
			inp.Port = int(dnsAnswer.Port)
		case *dns.TXT:
			// Pull out the txt
			inp = entryFromMap(inprogress, dnsAnswer.Hdr.Name, dnsAnswer.Hdr.Rrtype)
			if inp.complete() {
				continue
			}

			inp.Info = strings.Join(dnsAnswer.Txt, "|")
			inp.InfoFields = dnsAnswer.Txt
			inp.hasTXT = true
		case *dns.A:
			// Pull out the IP
			inp = entryFromMap(inprogress, dnsAnswer.Hdr.Name, dnsAnswer.Hdr.Rrtype)
			if inp.complete() {
				continue
			}

			inp.AddrV4 = dnsAnswer.A
		case *dns.AAAA:
			// Pull out the IP
			inp = entryFromMap(inprogress, dnsAnswer.Hdr.Name, dnsAnswer.Hdr.Rrtype)
			if inp.complete() {
				continue
			}

			inp.AddrV6 = dnsAnswer.AAAA
		}

		if inp != nil {
			inp.TTL = int(answer.Header().Ttl)
		}
	}

	return inp
}
