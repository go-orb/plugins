// Package text provides a the slog text handler.
package text

import (
	"os"

	"github.com/go-orb/go-orb/log"
	"golang.org/x/exp/slog"
)

func init() {
	if err := log.Plugins.Add("textstdout", NewHandlerstdout); err != nil {
		panic(err)
	}

	if err := log.Plugins.Add("textstderr", NewHandlerStderr); err != nil {
		panic(err)
	}
}

// NewHandlerstdout writes text to stdout.
func NewHandlerstdout(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewTextHandler(os.Stdout), nil
}

// NewHandlerStderr writes text to stderr.
func NewHandlerStderr(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewTextHandler(os.Stderr), nil
}
