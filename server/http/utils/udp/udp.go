// Package udp offers UDP utilities.
package udp

import "net"

// BuildListenerUDP creates a UDP listener.
func BuildListenerUDP(address string) (net.PacketConn, error) {
	return net.ListenPacket("udp", address)
}
