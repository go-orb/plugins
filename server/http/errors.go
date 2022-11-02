package http

import "errors"

var (
	// ErrContentTypeNotSupported is returned when there is no matching codec.
	ErrContentTypeNotSupported = errors.New("content type not supported")
)
