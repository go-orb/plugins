// Package tcp offers tcp utilities.
package tcp

import (
	"crypto/tls"
	"net"
)

// BuildListenerTCP creates a net listener with or without TLS.
func BuildListenerTCP(addr string, tlsConf *tls.Config) (net.Listener, error) {
	// TODO: do we need tcp keep alive listener? To set timeout on keep alive
	if tlsConf != nil {
		return tls.Listen("tcp", addr, tlsConf)
	}

	return net.Listen("tcp", addr)
}
