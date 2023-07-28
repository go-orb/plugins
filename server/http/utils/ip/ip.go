// Package ip provides validation and parsing utilities.
package ip

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ErrNoAddress is returned when no address is provided.
	ErrNoAddress = errors.New("no adddress provided")
	// ErrPortInvalid is returned when the provided port is below 0.
	ErrPortInvalid = errors.New("port must be >= 0")
	// ErrInvalidIP is returned an invalid IP is provided.
	ErrInvalidIP = errors.New("invalid IP provided, must be between 0 and 65535")
)

var ipRe = regexp.MustCompile(`^(?:[0-9]{1,3}\.){3}[0-9]{1,3}`)

// ValidateAddress will do basic validation on an address string.
func ValidateAddress(address string) error {
	if len(address) == 0 {
		return ErrNoAddress
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("split host and port from address: %w", err)
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	if p < 0 || p > 65535 {
		return ErrPortInvalid
	}

	// No host is a valid host, to listen on all interfaces.
	if len(host) == 0 {
		return nil
	}

	if ipRe.MatchString(host) && net.ParseIP(host) == nil {
		return ErrInvalidIP
	}

	return nil
}

// ParsePort will take the port from an address and return it as int.
func ParsePort(address string) (int, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return 0, fmt.Errorf("split host and port from address: %w", err)
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return 0, err
	}

	return p, nil
}

//nolint:gochecknoglobals
var networkTypesHTTP3 = map[string]string{
	"unix": "unixgram",
	"tcp4": "udp4",
	"tcp6": "udp6",
}

// RegisterNetworkHTTP3 registers a mapping from non-HTTP/3 network to HTTP/3
// network. This should be called during init() and will panic if the network
// type is standard, reserved, or already registered.
//
// EXPERIMENTAL: Subject to change.
func RegisterNetworkHTTP3(originalNetwork, h3Network string) {
	if _, ok := networkTypesHTTP3[strings.ToLower(originalNetwork)]; ok {
		panic("network type " + originalNetwork + " is already registered")
	}

	networkTypesHTTP3[originalNetwork] = h3Network
}

// GetHTTP3Network maps tcp -> udp.
func GetHTTP3Network(originalNetwork string) string {
	h3Network, ok := networkTypesHTTP3[strings.ToLower(originalNetwork)]
	if !ok {
		// TODO: Maybe a better default is to not enable HTTP/3 if we do not know the network?
		return "udp"
	}

	return h3Network
}
