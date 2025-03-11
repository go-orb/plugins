// Package log provides a logging middleware for client.
package log

import (
	"context"
	"fmt"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
)

func init() {
	client.Middlewares.Add(Name, Provide)
}

// Name is the middlewares name.
const Name = "log"

var _ client.Middleware = (*Middleware)(nil)

// Middleware is the log Middleware for client.
type Middleware struct {
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

// Request wraps the original RequestNoCodec method or other middlewares.
func (m *Middleware) Request(
	next client.MiddlewareRequestHandler,
) client.MiddlewareRequestHandler {
	return func(ctx context.Context, req *client.Req[any, any], result any, opts *client.CallOptions) error {
		node, err := req.Node(ctx, opts)
		if err != nil {
			// Call the client/next middleware on resolve errors, this might trigger the retry logic.
			return next(ctx, req, result, opts)
		}

		m.logger.TraceContext(
			ctx,
			"Making a request",
			"url", fmt.Sprintf("%s://%s%s", node.Transport, node.Address, req.Endpoint()),
			"content-type", opts.ContentType,
		)

		err = next(ctx, req, result, opts)

		if err != nil {
			m.logger.ErrorContext(
				ctx,
				"Got an error",
				"error", err,
				"url", fmt.Sprintf("%s://%s%s", node.Transport, node.Address, req.Endpoint()),
				"content-type", opts.ContentType,
			)
		} else {
			m.logger.TraceContext(
				ctx,
				"Got a result",
				"url", fmt.Sprintf("%s://%s%s", node.Transport, node.Address, req.Endpoint()),
				"content-type", opts.ContentType,
			)
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

	return &Middleware{
		logger: logger,
	}, nil
}
