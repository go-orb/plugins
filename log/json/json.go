package json

import (
	"os"

	"go-micro.dev/v5/log"
	"golang.org/x/exp/slog"
)

func init() {
	if err := log.Plugins.Add("jsonstdout", JSONStdoutPlugin); err != nil {
		panic(err)
	}

	if err := log.Plugins.Add("jsonstderr", JSONStderrPlugin); err != nil {
		panic(err)
	}
}

// JSONStdoutPlugin writes json to stdout.
func JSONStdoutPlugin(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewJSONHandler(os.Stdout), nil
}

// JSONStderrPlugin writes json to stderr.
func JSONStderrPlugin(level slog.Leveler) (slog.Handler, error) {
	return slog.HandlerOptions{Level: level}.NewJSONHandler(os.Stderr), nil
}
