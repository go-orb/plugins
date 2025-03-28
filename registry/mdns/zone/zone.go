// Package zone provides mDNS zone utilities.
package zone

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"

	"github.com/go-orb/plugins/registry/mdns/util"
	"github.com/miekg/dns"
)

const (
	// DefaultTTL is the default TTL value in returned DNS records in seconds.
	DefaultTTL = 120
)

// Zone is the interface used to integrate with the server and
// to serve records dynamically.
type Zone interface {
	// Records returns DNS records in response to a DNS question.
	Records(q dns.Question) []dns.RR
}

// MDNSService is used to export a named service by implementing a Zone.
type MDNSService struct {
	Instance     string   // Instance name (e.g. "hostService name")
	Service      string   // Service name (e.g. "_http._tcp.")
	Domain       string   // If blank, assumes "local"
	HostName     string   // Host machine DNS name (e.g. "mymachine.net.")
	Port         int      // Service Port
	IPs          []net.IP // IP addresses for the service's host
	TXT          []string // Service TXT records
	TTL          uint32
	serviceAddr  string // Fully qualified service address
	instanceAddr string // Fully qualified instance address
	enumAddr     string // _services._dns-sd._udp.<domain>
}

// validateFQDN returns an error if the passed string is not a fully qualified
// hdomain name (more specifically, a hostname).
func validateFQDN(s string) error {
	if len(s) == 0 {
		return errors.New("FQDN must not be blank")
	}

	if s[len(s)-1] != '.' {
		return fmt.Errorf("FQDN must end in period: %s", s)
	}
	// TODO(reddaly): Perform full validation.

	return nil
}

// NewMDNSService returns a new instance of MDNSService.
//
// If domain, hostName, or ips is set to the zero value, then a default value
// will be inferred from the operating system.
//
// TODO(reddaly): This interface may need to change to account for "unique
// record" conflict rules of the mDNS protocol.  Upon startup, the server should
// check to ensure that the instance name does not conflict with other instance
// names, and, if required, select a new name.  There may also be conflicting
// hostName A/AAAA records.
func NewMDNSService(
	instance, service, domain, hostName string, port int, ips []net.IP, txt []string,
) (*MDNSService, error) {
	// Sanity check inputs
	if instance == "" {
		return nil, errors.New("missing service instance name")
	}

	if service == "" {
		return nil, errors.New("missing service name")
	}

	if port == 0 {
		return nil, errors.New("missing service port")
	}

	// Set default domain
	if domain == "" {
		domain = "local."
	}

	if err := validateFQDN(domain); err != nil {
		return nil, fmt.Errorf("domain %q is not a fully-qualified domain name: %w", domain, err)
	}

	// Get host information if no host is specified.
	if hostName == "" {
		var err error

		hostName, err = os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("could not determine host: %w", err)
		}

		hostName += "."
	}

	if err := validateFQDN(hostName); err != nil {
		return nil, fmt.Errorf("hostName %q is not a fully-qualified domain name: %w", hostName, err)
	}

	if len(ips) == 0 {
		var err error

		ips, err = net.LookupIP(util.TrimDot(hostName))
		if err != nil {
			// Try appending the host domain suffix and lookup again
			// (required for Linux-based hosts)
			tmpHostName := fmt.Sprintf("%s%s", hostName, domain)

			ips, err = net.LookupIP(util.TrimDot(tmpHostName))

			if err != nil {
				return nil, fmt.Errorf("could not determine host IP addresses for %s", hostName)
			}
		}
	}

	for _, ip := range ips {
		if ip.To4() == nil && ip.To16() == nil {
			return nil, fmt.Errorf("invalid IP address in IPs list: %v", ip)
		}
	}

	return &MDNSService{
		Instance:     instance,
		Service:      service,
		Domain:       domain,
		HostName:     hostName,
		Port:         port,
		IPs:          ips,
		TXT:          txt,
		TTL:          DefaultTTL,
		serviceAddr:  fmt.Sprintf("%s.%s.", util.TrimDot(service), util.TrimDot(domain)),
		instanceAddr: fmt.Sprintf("%s.%s.%s.", instance, util.TrimDot(service), util.TrimDot(domain)),
		enumAddr:     fmt.Sprintf("_services._dns-sd._udp.%s.", util.TrimDot(domain)),
	}, nil
}

// Records returns DNS records in response to a DNS question.
func (m *MDNSService) Records(q dns.Question) []dns.RR {
	switch q.Name {
	case m.enumAddr:
		return m.serviceEnum(q)
	case m.serviceAddr:
		return m.serviceRecords(q)
	case m.instanceAddr:
		return m.instanceRecords(q)
	case m.HostName:
		if q.Qtype == dns.TypeA || q.Qtype == dns.TypeAAAA {
			return m.instanceRecords(q)
		}

		fallthrough
	default:
		return nil
	}
}

func (m *MDNSService) serviceEnum(q dns.Question) []dns.RR {
	switch q.Qtype {
	case dns.TypeANY:
		fallthrough
	case dns.TypePTR:
		rr := &dns.PTR{
			Hdr: dns.RR_Header{ //nolint:nosnakecase
				Name:   q.Name,
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    atomic.LoadUint32(&m.TTL),
			},
			Ptr: m.serviceAddr,
		}

		return []dns.RR{rr}
	default:
		return nil
	}
}

// serviceRecords is called when the query matches the service name.
func (m *MDNSService) serviceRecords(q dns.Question) []dns.RR {
	switch q.Qtype {
	case dns.TypeANY:
		fallthrough
	case dns.TypePTR:
		// Build a PTR response for the service
		rr := &dns.PTR{
			Hdr: dns.RR_Header{ //nolint:nosnakecase
				Name:   q.Name,
				Rrtype: dns.TypePTR,
				Class:  dns.ClassINET,
				Ttl:    atomic.LoadUint32(&m.TTL),
			},
			Ptr: m.instanceAddr,
		}
		servRec := []dns.RR{rr}

		// Get the instance records
		instRecs := m.instanceRecords(dns.Question{
			Name:  m.instanceAddr,
			Qtype: dns.TypeANY,
		})

		// Return the service record with the instance records
		return append(servRec, instRecs...)
	default:
		return nil
	}
}

// serviceRecords is called when the query matches the instance name.
func (m *MDNSService) instanceRecords(question dns.Question) []dns.RR { //nolint:funlen
	switch question.Qtype {
	case dns.TypeANY:
		// Get the SRV, which includes A and AAAA
		recs := m.instanceRecords(dns.Question{
			Name:  m.instanceAddr,
			Qtype: dns.TypeSRV,
		})

		// Add the TXT record
		recs = append(recs, m.instanceRecords(dns.Question{
			Name:  m.instanceAddr,
			Qtype: dns.TypeTXT,
		})...)

		return recs

	case dns.TypeA:
		var rr []dns.RR

		for _, ip := range m.IPs {
			if ip4 := ip.To4(); ip4 != nil {
				rr = append(rr, &dns.A{
					Hdr: dns.RR_Header{ //nolint:nosnakecase
						Name:   m.HostName,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    atomic.LoadUint32(&m.TTL),
					},
					A: ip4,
				})
			}
		}

		return rr

	case dns.TypeAAAA:
		var rr []dns.RR

		for _, ip := range m.IPs {
			if ip.To4() != nil {
				// TODO(reddaly): IPv4 addresses could be encoded in IPv6 format and
				// putinto AAAA records, but the current logic puts ipv4-encodable
				// addresses into the A records exclusively.  Perhaps this should be
				// configurable?
				continue
			}

			if ip16 := ip.To16(); ip16 != nil {
				rr = append(rr, &dns.AAAA{
					Hdr: dns.RR_Header{ //nolint:nosnakecase
						Name:   m.HostName,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    atomic.LoadUint32(&m.TTL),
					},
					AAAA: ip16,
				})
			}
		}

		return rr

	case dns.TypeSRV:
		// Create the SRV Record
		srv := &dns.SRV{
			Hdr: dns.RR_Header{ //nolint:nosnakecase
				Name:   question.Name,
				Rrtype: dns.TypeSRV,
				Class:  dns.ClassINET,
				Ttl:    atomic.LoadUint32(&m.TTL),
			},
			Priority: 10,
			Weight:   1,
			Port:     uint16(m.Port), //nolint:gosec
			Target:   m.HostName,
		}
		recs := []dns.RR{srv}

		// Add the A record
		recs = append(recs, m.instanceRecords(dns.Question{
			Name:  m.instanceAddr,
			Qtype: dns.TypeA,
		})...)

		// Add the AAAA record
		recs = append(recs, m.instanceRecords(dns.Question{
			Name:  m.instanceAddr,
			Qtype: dns.TypeAAAA,
		})...)

		return recs

	case dns.TypeTXT:
		txt := &dns.TXT{
			Hdr: dns.RR_Header{ //nolint:nosnakecase
				Name:   question.Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    atomic.LoadUint32(&m.TTL),
			},
			Txt: m.TXT,
		}

		return []dns.RR{txt}
	}

	return nil
}

// GetServiceAddr returns the service address.
func (m *MDNSService) GetServiceAddr() string {
	return m.serviceAddr
}

// TestMDNSService is used for tests. Don't use.
var TestMDNSService = MDNSService{ //nolint:gochecknoglobals
	serviceAddr: "_foobar._tcp.local.",
	Domain:      "local",
}
