// Package regutil provides utility functions for the registries.
package regutil

import (
	"fmt"

	"github.com/go-orb/go-orb/registry"
	"github.com/go-orb/go-orb/util/orberrors"
)

// isValidChar checks if a character is valid for a service name.
//
// lowercase ascii characters, numbers, hyphens, and periods are allowed.
func isValidChar(c byte) bool {
	if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '-') || (c == '.') {
		return true
	}

	return false
}

func isValidNameText(s string) bool {
	for _, c := range s {
		if !isValidChar(byte(c)) {
			return false
		}
	}

	return true
}

// IsValid checks if a serviceNode has a valid namespace, region, and name.
//
// lowercase ascii characters, numbers, hyphens, and periods are allowed.
func IsValid(serviceNode registry.ServiceNode) error {
	if serviceNode.Name == "" {
		return orberrors.ErrBadRequest.WrapNew("service name must not be empty")
	}

	if !isValidNameText(serviceNode.Namespace) {
		return orberrors.ErrBadRequest.Wrap(fmt.Errorf("namespace must be alphanumeric, got %s", serviceNode.Namespace))
	}

	if !isValidNameText(serviceNode.Region) {
		return orberrors.ErrBadRequest.Wrap(fmt.Errorf("region must be alphanumeric, got %s", serviceNode.Region))
	}

	if !isValidNameText(serviceNode.Name) {
		return orberrors.ErrBadRequest.Wrap(fmt.Errorf("service name must be alphanumeric, got %s", serviceNode.Name))
	}

	return nil
}
