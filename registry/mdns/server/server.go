// Package server provides an MDNS server.
package server

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-orb/go-orb/log"
	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/go-orb/plugins/registry/mdns/util"
	"github.com/go-orb/plugins/registry/mdns/zone"
)

// mDNS Groups.
//
//nolint:gochecknoglobals
var (
	MDNSGroupIPv4 = net.ParseIP("224.0.0.251")
	MDNSGroupIPv6 = net.ParseIP("ff02::fb")
)

// mDNS wildcard addresses.
//
//nolint:gochecknoglobals
var (
	MDNSWildcardAddrIPv4 = &net.UDPAddr{
		IP:   net.ParseIP("224.0.0.0"),
		Port: 5353,
	}
	MDNSWildcardAddrIPv6 = &net.UDPAddr{
		IP:   net.ParseIP("ff02::"),
		Port: 5353,
	}
)

// mDNS endpoint addresses.
//
//nolint:gochecknoglobals
var (
	IPv4Addr = &net.UDPAddr{
		IP:   MDNSGroupIPv4,
		Port: 5353,
	}
	IPv6Addr = &net.UDPAddr{
		IP:   MDNSGroupIPv6,
		Port: 5353,
	}
)

// GetMachineIP is a func which returns the outbound IP of this machine.
// Used by the server to determine whether to attempt send the response on a local address.
type GetMachineIP func() net.IP

// Config is used to configure the mDNS server.
type Config struct {
	// Zone must be provided to support responding to queries
	Zone zone.Zone

	// Iface if provided binds the multicast listener to the given
	// interface. If not provided, the system default multicase interface
	// is used.
	Iface *net.Interface

	// Port If it is not 0, replace the port 5353 with this port number.
	Port int

	// GetMachineIP is a function to return the IP of the local machine
	GetMachineIP GetMachineIP

	// LocalhostChecking if enabled asks the server to also send responses to
	// 0.0.0.0 if the target IP
	// is this host (as defined by GetMachineIP).
	// Useful in case machine is on a VPN which blocks comms on non standard ports
	LocalhostChecking bool
}

// Server is an mDNS server used to listen for mDNS queries and respond if we
// have a matching local record.
type Server struct {
	config *Config

	ipv4List *net.UDPConn
	ipv6List *net.UDPConn

	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
	wg           sync.WaitGroup

	outboundIP net.IP
}

// NewServer is used to create a new mDNS server from a config.
func NewServer(config *Config) (*Server, error) { //nolint:gocyclo,funlen
	setCustomPort(config.Port)

	// Create the listeners
	// Create wildcard connections (because :5353 can be already taken by other apps)
	ipv4List, _ := net.ListenUDP("udp4", MDNSWildcardAddrIPv4) //nolint:errcheck
	ipv6List, _ := net.ListenUDP("udp6", MDNSWildcardAddrIPv6) //nolint:errcheck

	if ipv4List == nil && ipv6List == nil {
		return nil, errors.New("mdns: failed to bind to any udp port")
	}

	if ipv4List == nil {
		ipv4List = &net.UDPConn{}
	}

	if ipv6List == nil {
		ipv6List = &net.UDPConn{}
	}

	// Join multicast groups to receive announcements
	connIPv4 := ipv4.NewPacketConn(ipv4List)
	connIPv6 := ipv6.NewPacketConn(ipv6List)

	if err := connIPv4.SetMulticastLoopback(true); err != nil {
		return nil, err
	}

	if err := connIPv6.SetMulticastLoopback(true); err != nil {
		return nil, err
	}

	if config.Iface != nil { //nolint:nestif
		if err := connIPv4.JoinGroup(config.Iface, &net.UDPAddr{IP: MDNSGroupIPv4}); err != nil {
			return nil, err
		}

		if err := connIPv6.JoinGroup(config.Iface, &net.UDPAddr{IP: MDNSGroupIPv6}); err != nil {
			return nil, err
		}
	} else {
		ifaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}

		// Check if we succeed on atleast one interface.
		success := false

		for _, iface := range ifaces {
			i := iface
			if err := connIPv4.JoinGroup(&i, &net.UDPAddr{IP: MDNSGroupIPv4}); err != nil {
				continue
			}

			i = iface
			if err := connIPv6.JoinGroup(&i, &net.UDPAddr{IP: MDNSGroupIPv6}); err != nil {
				continue
			}

			success = true
		}

		if !success {
			return nil, errors.New("failed to join multicast group on all interfaces")
		}
	}

	ipFunc := getOutboundIP
	if config.GetMachineIP != nil {
		ipFunc = config.GetMachineIP
	}

	s := &Server{
		config:     config,
		ipv4List:   ipv4List,
		ipv6List:   ipv6List,
		shutdownCh: make(chan struct{}),
		outboundIP: ipFunc(),
	}

	go s.recv(s.ipv4List)
	go s.recv(s.ipv6List)

	s.wg.Add(1)

	go s.probe()

	return s, nil
}

// Shutdown is used to shutdown the listener.
func (s *Server) Shutdown() error {
	s.shutdownLock.Lock()
	defer s.shutdownLock.Unlock()

	if s.shutdown {
		return nil
	}

	s.shutdown = true
	close(s.shutdownCh)

	if err := s.unregister(); err != nil {
		return err
	}

	s.wg.Wait()

	var gerr error

	if s.ipv4List != nil {
		if err := s.ipv4List.Close(); err != nil {
			gerr = fmt.Errorf("close IPv4 UDP connection: %w", err)
		}
	}

	if s.ipv6List != nil {
		if err := s.ipv6List.Close(); err != nil {
			gerr = fmt.Errorf("close IPv4 UDP connection: %w", err)
		}
	}

	return gerr
}

// recv is a long running routine to receive packets from an interface.
func (s *Server) recv(c *net.UDPConn) {
	if c == nil {
		return
	}

	buf := make([]byte, 65536)

	for {
		s.shutdownLock.Lock()
		if s.shutdown {
			s.shutdownLock.Unlock()
			return
		}
		s.shutdownLock.Unlock()

		n, from, err := c.ReadFrom(buf)
		if err != nil {
			continue
		}

		if err := s.parsePacket(buf[:n], from); err != nil {
			log.Error("[ERR] mdns: Failed to handle query", err)
		}
	}
}

// parsePacket is used to parse an incoming packet.
func (s *Server) parsePacket(packet []byte, from net.Addr) error {
	var msg dns.Msg
	if err := msg.Unpack(packet); err != nil {
		log.Error("[ERR] mdns: Failed to unpack packet", err)
		return err
	}
	// TODO: This is a bit of a hack
	// We decided to ignore some mDNS answers for the time being
	// See: https://tools.ietf.org/html/rfc6762#section-7.2
	msg.Truncated = false

	return s.handleQuery(&msg, from)
}

// handleQuery is used to handle an incoming query.
func (s *Server) handleQuery(query *dns.Msg, from net.Addr) error {
	if query.Opcode != dns.OpcodeQuery {
		// "In both multicast query and multicast response messages, the OPCODE MUST
		// be zero on transmission (only standard queries are currently supported
		// over multicast).  Multicast DNS messages received with an OPCODE other
		// than zero MUST be silently ignored."  Note: OpcodeQuery == 0
		return fmt.Errorf("mdns: received query with non-zero Opcode %v: %v", query.Opcode, *query)
	}

	if query.Rcode != 0 {
		// "In both multicast query and multicast response messages, the Response
		// Code MUST be zero on transmission.  Multicast DNS messages received with
		// non-zero Response Codes MUST be silently ignored."
		return fmt.Errorf("mdns: received query with non-zero Rcode %v: %v", query.Rcode, *query)
	}

	// TODO(reddaly): Handle "TC (Truncated) Bit":
	//    In query messages, if the TC bit is set, it means that additional
	//    Known-Answer records may be following shortly.  A responder SHOULD
	//    record this fact, and wait for those additional Known-Answer records,
	//    before deciding whether to respond.  If the TC bit is clear, it means
	//    that the querying host has no additional Known Answers.
	if query.Truncated {
		return fmt.Errorf("[ERR] mdns: support for DNS requests with high truncated bit not implemented: %v", *query)
	}

	var unicastAnswer, multicastAnswer []dns.RR

	// Handle each question
	for _, q := range query.Question {
		mrecs, urecs := s.handleQuestion(q)
		multicastAnswer = append(multicastAnswer, mrecs...)
		unicastAnswer = append(unicastAnswer, urecs...)
	}

	// See section 18 of RFC 6762 for rules about DNS headers.
	resp := func(unicast bool) *dns.Msg {
		// 18.1: ID (Query Identifier)
		// 0 for multicast response, query.Id for unicast response
		id := uint16(0)
		if unicast {
			id = query.Id
		}

		var answer []dns.RR
		if unicast {
			answer = unicastAnswer
		} else {
			answer = multicastAnswer
		}

		if len(answer) == 0 {
			return nil
		}

		return &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Id: id,

				// 18.2: QR (Query/Response) Bit - must be set to 1 in response.
				Response: true,

				// 18.3: OPCODE - must be zero in response (OpcodeQuery == 0)
				Opcode: dns.OpcodeQuery,

				// 18.4: AA (Authoritative Answer) Bit - must be set to 1
				Authoritative: true,

				// The following fields must all be set to 0:
				// 18.5: TC (TRUNCATED) Bit
				// 18.6: RD (Recursion Desired) Bit
				// 18.7: RA (Recursion Available) Bit
				// 18.8: Z (Zero) Bit
				// 18.9: AD (Authentic Data) Bit
				// 18.10: CD (Checking Disabled) Bit
				// 18.11: RCODE (Response Code)
			},
			// 18.12 pertains to questions (handled by handleQuestion)
			// 18.13 pertains to resource records (handled by handleQuestion)

			// 18.14: Name Compression - responses should be compressed (though see
			// caveats in the RFC), so set the Compress bit (part of the dns library
			// API, not part of the DNS packet) to true.
			Compress: true,
			Question: query.Question,
			Answer:   answer,
		}
	}

	if mresp := resp(false); mresp != nil {
		if err := s.sendResponse(mresp, from); err != nil {
			return fmt.Errorf("mdns: error sending multicast response: %w", err)
		}
	}

	if uresp := resp(true); uresp != nil {
		if err := s.sendResponse(uresp, from); err != nil {
			return fmt.Errorf("mdns: error sending unicast response: %w", err)
		}
	}

	return nil
}

// handleQuestion is used to handle an incoming question
//
// The response to a question may be transmitted over multicast, unicast, or
// both.  The return values are DNS records for each transmission type.
func (s *Server) handleQuestion(q dns.Question) (multicastRecs, unicastRecs []dns.RR) {
	records := s.config.Zone.Records(q)
	if len(records) == 0 {
		return nil, nil
	}

	// Handle unicast and multicast responses.
	// TODO(reddaly): The decision about sending over unicast vs. multicast is not
	// yet fully compliant with RFC 6762.  For example, the unicast bit should be
	// ignored if the records in question are close to TTL expiration.  For now,
	// we just use the unicast bit to make the decision, as per the spec:
	//     RFC 6762, section 18.12.  Repurposing of Top Bit of qclass in Question
	//     Section
	//
	//     In the Question Section of a Multicast DNS query, the top bit of the
	//     qclass field is used to indicate that unicast responses are preferred
	//     for this particular question.  (See Section 5.4.)
	if q.Qclass&(1<<15) != 0 {
		return nil, records
	}

	return records, nil
}

func (s *Server) probe() {
	defer s.wg.Done()

	mdnsService, ok := s.config.Zone.(*zone.MDNSService)
	if !ok {
		return
	}

	name := fmt.Sprintf("%s.%s.%s.", mdnsService.Instance,
		util.TrimDot(mdnsService.Service), util.TrimDot(mdnsService.Domain))

	msg := new(dns.Msg)
	msg.SetQuestion(name, dns.TypePTR)
	msg.RecursionDesired = false

	srv := &dns.SRV{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeSRV,
			Class:  dns.ClassINET,
			Ttl:    zone.DefaultTTL,
		},
		Priority: 0,
		Weight:   0,
		Port:     uint16(mdnsService.Port), //nolint:gosec
		Target:   mdnsService.HostName,
	}
	txt := &dns.TXT{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeTXT,
			Class:  dns.ClassINET,
			Ttl:    zone.DefaultTTL,
		},
		Txt: mdnsService.TXT,
	}
	msg.Ns = []dns.RR{srv, txt}

	randomizer := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	for i := 0; i < 3; i++ {
		if err := s.SendMulticast(msg); err != nil {
			log.Error("[ERR] mdns: failed to send probe", err)
		}

		time.Sleep(time.Duration(randomizer.Intn(250)) * time.Millisecond)
	}

	resp := new(dns.Msg)
	resp.Response = true

	// set for query
	msg.SetQuestion(name, dns.TypeANY)

	resp.Answer = append(resp.Answer, s.config.Zone.Records(msg.Question[0])...)

	// reset
	msg.SetQuestion(name, dns.TypePTR)

	// From RFC6762
	//    The Multicast DNS responder MUST send at least two unsolicited
	//    responses, one second apart. To provide increased robustness against
	//    packet loss, a responder MAY send up to eight unsolicited responses,
	//    provided that the interval between unsolicited responses increases by
	//    at least a factor of two with every response sent.
	timeout := 1 * time.Second
	timer := time.NewTimer(timeout)

	for i := 0; i < 3; i++ {
		if err := s.SendMulticast(resp); err != nil {
			log.Error("[ERR] mdns: failed to send announcement", err)
		}
		select {
		case <-timer.C:
			timeout *= 2
			timer.Reset(timeout)
		case <-s.shutdownCh:
			timer.Stop()
			return
		}
	}
}

// SendMulticast us used to send a multicast response packet.
func (s *Server) SendMulticast(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return err
	}

	if s.ipv4List != nil {
		if _, err := s.ipv4List.WriteToUDP(buf, IPv4Addr); err != nil {
			return err
		}
	}

	if s.ipv6List != nil {
		if _, err := s.ipv6List.WriteToUDP(buf, IPv6Addr); err != nil {
			return err
		}
	}

	return nil
}

// sendResponse is used to send a response packet.
func (s *Server) sendResponse(resp *dns.Msg, from net.Addr) error {
	// TODO(reddaly): Respect the unicast argument, and allow sending responses
	// over multicast.
	buf, err := resp.Pack()
	if err != nil {
		return err
	}

	// Determine the socket to send from
	addr := from.(*net.UDPAddr) //nolint:errcheck
	conn := s.ipv4List
	backupTarget := net.IPv4zero

	if addr.IP.To4() == nil {
		conn = s.ipv6List
		backupTarget = net.IPv6zero
	}

	_, err = conn.WriteToUDP(buf, addr)
	// If the address we're responding to is this machine then we can also
	// attempt sending on 0.0.0.0
	// This covers the case where this machine is using a VPN and certain ports
	// are blocked so the response never gets there
	// Sending two responses is OK
	if s.config.LocalhostChecking && addr.IP.Equal(s.outboundIP) {
		// ignore any errors, this is best efforts
		if _, err = conn.WriteToUDP(buf, &net.UDPAddr{IP: backupTarget, Port: addr.Port}); err != nil {
			return err
		}
	}

	return err
}

func (s *Server) unregister() error {
	sd, ok := s.config.Zone.(*zone.MDNSService)
	if !ok {
		return nil
	}

	atomic.StoreUint32(&sd.TTL, 0)
	name := fmt.Sprintf("%s.%s.%s.", sd.Instance, util.TrimDot(sd.Service), util.TrimDot(sd.Domain))

	q := new(dns.Msg)
	q.SetQuestion(name, dns.TypeANY)

	resp := new(dns.Msg)
	resp.Response = true
	resp.Answer = append(resp.Answer, s.config.Zone.Records(q.Question[0])...)

	return s.SendMulticast(resp)
}

func setCustomPort(port int) {
	if port != 0 {
		if MDNSWildcardAddrIPv4.Port != port {
			MDNSWildcardAddrIPv4.Port = port
		}

		if MDNSWildcardAddrIPv6.Port != port {
			MDNSWildcardAddrIPv6.Port = port
		}

		if IPv4Addr.Port != port {
			IPv4Addr.Port = port
		}

		if IPv6Addr.Port != port {
			IPv6Addr.Port = port
		}
	}
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// no net connectivity maybe so fallback
		return nil
	}
	defer conn.Close() //nolint:errcheck

	localAddr := conn.LocalAddr().(*net.UDPAddr) //nolint:errcheck

	return localAddr.IP
}
