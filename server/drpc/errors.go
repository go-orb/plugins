package drpc

import "errors"

// Errors.
var (
	// ErrInvalidConfigType is returned when you provided an invalid config type.
	ErrInvalidConfigType = errors.New("drpc server: invalid config type provided, not of type drpc.Config")
)
