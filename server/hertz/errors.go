package hertz

import "errors"

// Errors.
var (
	// ErrContentTypeNotSupported is returned when there is no matching codec.
	ErrContentTypeNotSupported = errors.New("content type not supported")
	ErrInvalidConfigType       = errors.New("http server: invalid config type provided, not of type http.Config")
)
