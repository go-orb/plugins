package retry

import (
	"context"
	"errors"
	"time"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/util/orberrors"
)

// Always always retry on error.
// WARNING: This will retry on all errors, including business logic errors.
func Always(_ context.Context, _ error, _ *client.CallOptions) (bool, error) {
	return true, nil
}

// OnTimeoutError retries a request on a 408 timeout error, as well as on 503/504 connection errors.
func OnTimeoutError(ctx context.Context, err error, options *client.CallOptions) (bool, error) {
	if err == nil {
		return false, nil
	}

	var orbe *orberrors.Error

	err = orberrors.From(err)
	if errors.As(err, &orbe) {
		switch orbe.Code {
		// Retry on timeout, not on 500 internal server error, as that is a business
		// logic error that should be handled by the user.
		case 408:
			return true, nil
		case 504:
			fallthrough
		// Retry on connection error: Service Unavailable
		case 503:
			timeout := time.After(options.DialTimeout)
			select {
			case <-ctx.Done():
				return false, nil
			case <-timeout:
				return true, nil
			}
		default:
			return false, nil
		}
	}

	return false, nil
}

// OnConnectionError retries a request on a 503/504 connection error.
// This is the default.
func OnConnectionError(ctx context.Context, err error, options *client.CallOptions) (bool, error) {
	if err == nil {
		return false, nil
	}

	var orbe *orberrors.Error

	err = orberrors.From(err)
	if errors.As(err, &orbe) {
		switch orbe.Code {
		case 504:
			fallthrough
		// Retry on connection error: Service Unavailable
		case 503:
			timeout := time.After(options.DialTimeout)
			select {
			case <-ctx.Done():
				return false, nil
			case <-timeout:
				return true, nil
			}
		default:
			return false, nil
		}
	}

	return false, nil
}
