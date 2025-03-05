// Package pool provides a pool of grpc clients
// This is a modified version of: https://github.com/processout/grpc-go-pool/blob/master/pool.go
package pool

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var (
	// ErrClosed is the error when the client pool is closed.
	ErrClosed = errors.New("grpc pool: client pool is closed")
	// ErrTimeout is the error when the client pool timed out.
	ErrTimeout = errors.New("grpc pool: client pool timed out")
	// ErrAlreadyClosed is the error when the client conn was already closed.
	ErrAlreadyClosed = errors.New("grpc pool: the connection was already closed")
	// ErrFullPool is the error when the pool is already full.
	ErrFullPool = errors.New("grpc pool: closing a ClientConn into a full pool")
)

// FactoryWithContext is a function type creating a grpc client
// that accepts the context parameter that could be passed from
// Get or NewWithContext method.
type FactoryWithContext func(ctx context.Context, addr string, tlsConfig *tls.Config) (*grpc.ClientConn, error)

// Pool is the grpc client pool.
type Pool struct {
	capacity        int
	clients         map[string]chan ClientConn
	factory         FactoryWithContext
	idleTimeout     time.Duration
	maxLifeDuration time.Duration
	mu              sync.RWMutex
}

// ClientConn is the wrapper for a grpc client conn.
type ClientConn struct {
	*grpc.ClientConn
	addr          string
	pool          *Pool
	timeUsed      time.Time
	timeInitiated time.Time
	unhealthy     bool
}

// New creates a new clients pool with the given initial and maximum
// capacity, and the timeout for the idle clients. The context parameter would
// be passed to the factory method during initialization. Returns an error if the
// initial clients could not be created.
func New(factory FactoryWithContext, capacity int, idleTimeout time.Duration, maxLifeDuration ...time.Duration) (*Pool, error) {
	if capacity < 1 {
		capacity = 1
	}

	p := &Pool{
		capacity:    capacity,
		clients:     make(map[string]chan ClientConn),
		factory:     factory,
		idleTimeout: idleTimeout,
	}

	if len(maxLifeDuration) > 0 {
		p.maxLifeDuration = maxLifeDuration[0]
	}

	return p, nil
}

// GetClients returns the chan of clients for the given addr.
func (p *Pool) GetClients(addr string) chan ClientConn {
	p.mu.RLock()
	if clients, ok := p.clients[addr]; ok {
		p.mu.RUnlock()
		return clients
	}
	p.mu.RUnlock()

	p.mu.Lock()
	p.clients[addr] = make(chan ClientConn, p.capacity)
	clients := p.clients[addr]
	p.mu.Unlock()

	// Fill the rest of the pool with empty clients
	for i := 0; i < p.capacity; i++ {
		clients <- ClientConn{
			addr: addr,
			pool: p,
		}
	}

	return clients
}

// Close empties the pool calling Close on all its clients.
// You can call Close while there are outstanding clients.
// The pool channel is then closed, and Get will not be allowed anymore.
func (p *Pool) Close() {
	p.mu.Lock()
	for _, clients := range p.clients {
		close(clients)

		for client := range clients {
			if client.ClientConn == nil {
				continue
			}

			client.ClientConn.Close() //nolint:errcheck,gosec
		}
	}

	p.clients = nil
	p.mu.Unlock()
}

// IsClosed returns true if the client pool is closed.
func (p *Pool) IsClosed() bool {
	return p == nil || p.clients == nil
}

// Get will return the next available client. If capacity
// has not been reached, it will create a new one using the factory. Otherwise,
// it will wait till the next client becomes available or a timeout.
// A timeout of 0 is an indefinite wait.
func (p *Pool) Get(ctx context.Context, addr string, tlsConfig *tls.Config) (*ClientConn, error) {
	clients := p.GetClients(addr)

	if clients == nil {
		return nil, ErrClosed
	}

	wrapper := ClientConn{
		addr: addr,
		pool: p,
	}
	select {
	case wrapper = <-clients:
		// All good
	case <-ctx.Done():
		return nil, ErrTimeout // it would better returns ctx.Err()
	}

	// If the wrapper was idle too long, close the connection and create a new
	// one. It's safe to assume that there isn't any newer client as the client
	// we fetched is the first in the channel
	idleTimeout := p.idleTimeout
	if wrapper.ClientConn != nil && idleTimeout > 0 && wrapper.timeUsed.Add(idleTimeout).Before(time.Now()) {
		wrapper.ClientConn.Close() //nolint:errcheck,gosec
		wrapper.ClientConn = nil
	}

	var err error
	if wrapper.ClientConn == nil {
		wrapper.ClientConn, err = p.factory(ctx, addr, tlsConfig)
		if err != nil {
			// If there was an error, we want to put back a placeholder
			// client in the channel
			clients <- ClientConn{
				addr: addr,
				pool: p,
			}
		}
		// This is a new connection, reset its initiated time
		wrapper.timeInitiated = time.Now()
	}

	return &wrapper, err
}

// Unhealthy marks the client conn as unhealthy, so that the connection
// gets reset when closed.
func (c *ClientConn) Unhealthy() {
	c.unhealthy = true
}

// Close returns a ClientConn to the pool. It is safe to call multiple time,
// but will return an error after first time.
func (c *ClientConn) Close() error {
	if c == nil {
		return nil
	}

	if c.ClientConn == nil {
		return ErrAlreadyClosed
	}

	if c.pool.IsClosed() {
		return ErrClosed
	}

	// If the wrapper connection has become too old, we want to recycle it. To
	// clarify the logic: if the sum of the initialization time and the max
	// duration is before Now(), it means the initialization is so old adding
	// the maximum duration couldn't put in the future. This sum therefore
	// corresponds to the cut-off point: if it's in the future we still have
	// time, if it's in the past it's too old
	maxDuration := c.pool.maxLifeDuration
	if maxDuration > 0 && c.timeInitiated.Add(maxDuration).Before(time.Now()) {
		c.Unhealthy()
	}

	// We're cloning the wrapper so we can set ClientConn to nil in the one
	// used by the user
	wrapper := ClientConn{
		addr:       c.addr,
		pool:       c.pool,
		ClientConn: c.ClientConn,
		timeUsed:   time.Now(),
	}
	if c.unhealthy {
		wrapper.ClientConn.Close() //nolint:errcheck,gosec
		wrapper.ClientConn = nil
	} else {
		wrapper.timeInitiated = c.timeInitiated
	}
	select {
	case c.pool.GetClients(c.addr) <- wrapper:
		// All good
	default:
		return ErrFullPool
	}

	c.ClientConn = nil // Mark as closed

	return nil
}

// Capacity returns the capacity.
func (p *Pool) Capacity() int {
	if p.IsClosed() {
		return 0
	}

	return p.capacity
}

// Available returns the number of currently unused clients.
func (p *Pool) Available() int {
	if p.IsClosed() {
		return 0
	}

	return len(p.clients)
}
