// Package muxconn has been copied from https://gitea.elara.ws/Elara6331/drpc/src/branch/master/muxconn/muxconn.go
// MIT - Copyright (c) 2023 Elara Musayelyan
package muxconn

import (
	"context"
	"errors"
	"io"

	"github.com/hashicorp/yamux"
	"storj.io/drpc"
	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcmanager"
)

// ErrClosed happens when connection has been closed.
var ErrClosed = errors.New("connection closed")

var _ drpc.Conn = (*Conn)(nil)

// Options controls configuration settings for a conn.
type Options struct {
	// Manager controls the options we pass to the manager of this conn.
	Manager drpcmanager.Options
}

// Conn implements drpc.Conn using the yamux
// multiplexer to allow concurrent RPCs.
type Conn struct {
	conn          io.ReadWriteCloser
	sess          *yamux.Session
	isClosed      bool
	closed        chan struct{}
	unblockedChan chan struct{}
}

// New returns a new multiplexed DRPC connection.
func New(conn io.ReadWriteCloser) (*Conn, error) {
	return NewWithOptions(conn, Options{})
}

func NewWithOptions(conn io.ReadWriteCloser, opts Options) (*Conn, error) {
	sess, err := yamux.Client(conn, nil)
	if err != nil {
		return nil, err
	}

	uc := make(chan struct{}, 0)
	close(uc)

	return &Conn{
		conn:          conn,
		sess:          sess,
		closed:        make(chan struct{}),
		unblockedChan: uc,
	}, nil
}

// Close closes the multiplexer session
// and the underlying connection.
func (m *Conn) Close() error {
	if m.isClosed {
		return nil
	}

	m.isClosed = true
	defer close(m.closed)

	err := m.sess.Close()
	if err != nil {
		m.conn.Close()
		return err
	}

	if err = m.conn.Close(); err != nil {
		return err
	}

	return nil
}

// Closed returns a channel that will be closed
// when the connection is closed
func (m *Conn) Closed() <-chan struct{} {
	return m.closed
}

// Invoke issues the rpc on the transport serializing in, waits for a response, and deserializes it into out.
func (m *Conn) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) error {
	if m.isClosed {
		return ErrClosed
	}

	conn, err := m.sess.Open()
	if err != nil {
		return err
	}

	dconn := drpcconn.New(conn)
	invokeErr := dconn.Invoke(ctx, rpc, enc, in, out)

	dconn.Close()
	conn.Close()

	return invokeErr
}

// NewStream begins a streaming rpc on the connection.
func (m *Conn) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (drpc.Stream, error) {
	if m.isClosed {
		return nil, ErrClosed
	}

	conn, err := m.sess.Open()
	if err != nil {
		return nil, err
	}
	dconn := drpcconn.New(conn)

	s, err := dconn.NewStream(ctx, rpc, enc)
	if err != nil {
		return nil, err
	}

	// TODO(jochumdev): don't like to spawn a goroutine here.
	go func() {
		select {
		case <-dconn.Closed():
			conn.Close()
		case <-s.Context().Done():
			dconn.Close()
			conn.Close()
		}
	}()

	return s, nil
}

func (m *Conn) Unblocked() <-chan struct{} {
	return m.unblockedChan
}
