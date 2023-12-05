package muxserver

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/hashicorp/yamux"
	"storj.io/drpc"
	"storj.io/drpc/drpcserver"
)

// Server is a DRPC server that handles
// multiplexed streams
type Server struct {
	srv *drpcserver.Server
}

// New returns a new multiplexed Server that serves handler
func New(handler drpc.Handler) *Server {
	return &Server{srv: drpcserver.New(handler)}
}

// NewWithOptions is the same as New but passes options to DRPC
func NewWithOptions(handler drpc.Handler, opts drpcserver.Options) *Server {
	return &Server{srv: drpcserver.NewWithOptions(handler, opts)}
}

// Serve listens on the given listener and handles all multiplexed
// streams.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	connCh := make(chan net.Conn, 2)
	errCh := make(chan error, 2)
	go connChannel(ln, connCh, errCh)

	for {
		select {
		case conn := <-connCh:
			sess, err := yamux.Server(conn, nil)
			if err != nil {
				return err
			}

			go s.handleSession(ctx, sess)
		case err := <-errCh:
			return err
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *Server) handleSession(ctx context.Context, sess *yamux.Session) {
	for {
		conn, err := sess.Accept()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			continue
		}

		go s.srv.ServeOne(ctx, conn)
	}
}

// ServeOne serves a single set of rpcs on the provided transport.
func (s *Server) ServeOne(ctx context.Context, conn io.ReadWriteCloser) error {
	return s.srv.ServeOne(ctx, conn)
}

func connChannel(ln net.Listener, connCh chan net.Conn, errCh chan error) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			errCh <- err
			continue
		}
		connCh <- conn
	}
}
