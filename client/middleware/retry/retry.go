// Package retry provides a retry middleware for client.
package retry

import (
	"context"
	"math/rand"
	"time"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
)

func init() {
	client.Middlewares.Add(Name, Provide)
}

// Name is the middlewares name.
const Name = "retry"

var _ client.Middleware = (*Middleware)(nil)

// Middleware is the retry Middleware for client.
type Middleware struct {
	config Config
	logger log.Logger
}

// Start the component. E.g. connect to the broker.
func (m *Middleware) Start(_ context.Context) error { return nil }

// Stop the component. E.g. disconnect from the broker.
// The context will contain a timeout, and cancelation should be respected.
func (m *Middleware) Stop(_ context.Context) error { return nil }

// Type returns the component type, e.g. broker.
func (m *Middleware) Type() string {
	return client.MiddlewareComponentType
}

// String returns the name of this middleware.
func (m *Middleware) String() string {
	return Name
}

// Request wraps the original Request method or other middlewares.
func (m *Middleware) Request(
	next client.MiddlewareRequestHandler,
) client.MiddlewareRequestHandler {
	return func(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
		var err error

		// Get config.
		retryFunc := opts.RetryFunc
		if retryFunc == nil {
			retryFunc = m.config.RetryFunc
		}

		retries := opts.Retries
		if retries == 0 {
			retries = m.config.Retries
		}

		// If retries is set to 0 or no retry function is provided, just execute the request once
		if retries <= 0 || retryFunc == nil {
			return next(ctx, req, result, opts)
		}

		// First attempt
		err = next(ctx, req, result, opts)
		if err == nil {
			return nil
		}

		// Use exponential backoff with jitter for retries
		backoff := time.Millisecond * 100 // Start with 100ms
		maxBackoff := time.Second * 30    // Cap at 30 seconds

		// Retry logic
		for retryCount := 1; retryCount <= retries; retryCount++ {
			// Check if context is already done
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Call the retry function with current count
			shouldRetry, retryErr := retryFunc(ctx, err, opts)
			if retryErr != nil {
				return retryErr
			}

			if !shouldRetry {
				return err
			}

			// Apply exponential backoff with jitter before retrying
			jitter := time.Duration(rand.Int63n(int64(backoff) / 2)) //nolint:gosec
			sleepTime := backoff + jitter

			// Wait with context awareness
			timer := time.NewTimer(sleepTime)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			// Attempt the request again
			err = next(ctx, req, result, opts)
			if err == nil {
				return nil
			}

			// Increase backoff for next attempt (exponential)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		return err
	}
}

// Provide will be registered to client.Middlewares, it's a factory for this.
func Provide(sections []string, configs types.ConfigData, _ client.Type, logger log.Logger) (client.Middleware, error) {
	// Configure the logger.
	logger, err := logger.WithConfig(sections, configs)
	if err != nil {
		return nil, err
	}

	cfg := NewConfig()

	err = config.Parse(sections, configs, cfg)
	if err != nil {
		return nil, err
	}

	return &Middleware{
		config: cfg,
		logger: logger,
	}, nil
}
