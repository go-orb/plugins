// Package json provides a the slog json handler.
package json

import (
	"os"

	"go-micro.dev/v5/log"
	"golang.org/x/exp/slog"
)

func init() {
	if err := log.Plugins.Add("jsonstdout", NewHandlerStdout); err != nil {
		panic(err)
	}

	if err := log.Plugins.Add("jsonstderr", NewHandlerStderr); err != nil {
		panic(err)
	}
}

// NewHandlerStdout writes json to stdout.
func NewHandlerStdout(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewJSONHandler(os.Stdout), nil
}

// NewHandlerStderr writes json to stderr.
func NewHandlerStderr(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewJSONHandler(os.Stderr), nil
}
