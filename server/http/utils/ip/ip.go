// Package ip provides validation and parsing utilities.
package ip

import (
	"net"
	"strconv"

	"github.com/pkg/errors"
)

var (
	// ErrNoAddress is returned when no address is provided.
	ErrNoAddress = errors.New("no adddress provided")
	// ErrPortInvalid is returned when the provided port is below 0.
	ErrPortInvalid = errors.New("port must be >= 0")
	// ErrInvalidIP is returned an invalid IP is provided.
	ErrInvalidIP = errors.New("invalid IP provided")
)

// ValidateAddress will do basic validation on an address string.
func ValidateAddress(address string) error {
	if len(address) == 0 {
		return ErrNoAddress
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return errors.Wrap(err, "failed to split host and port from address")
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	if p < 0 {
		return ErrPortInvalid
	}

	// No host is a valid host, to listen on all interfaces.
	if len(host) == 0 {
		return nil
	}

	if net.ParseIP(host) == nil {
		return ErrInvalidIP
	}

	return nil
}

// ParsePort will take the port from an address and return it as int.
func ParsePort(address string) (int, error) {
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return 0, errors.Wrap(err, "failed to split host and port from address")
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return 0, err
	}

	return p, nil
}
