package text

import (
	"os"

	"go-micro.dev/v5/log"
	"golang.org/x/exp/slog"
)

func init() {
	if err := log.Plugins.Add("textstdout", TextStdoutPlugin); err != nil {
		panic(err)
	}

	if err := log.Plugins.Add("textstderr", TextStderrPlugin); err != nil {
		panic(err)
	}
}

// TextStdoutPlugin writes text to stdout.
func TextStdoutPlugin(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewTextHandler(os.Stdout), nil
}

// TextStderrPlugin writes text to stderr.
func TextStderrPlugin(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewTextHandler(os.Stderr), nil
}
