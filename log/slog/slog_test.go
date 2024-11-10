package slog

import (
	"context"
	"testing"

	"log/slog"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/config/source"
	"github.com/go-orb/go-orb/log"
	"github.com/stretchr/testify/require"

	_ "github.com/go-orb/plugins/codecs/json"
)

func TestChangeLevel(t *testing.T) {
	l, err := log.New(log.WithSetDefault())
	require.NoError(t, err)

	lDebug := l.WithLevel(log.LevelDebug.String())
	require.NoError(t, err)

	l.Info("Default logger Test")
	l.Debug("Not shown")
	lDebug.Debug("Debug: logger test")
	lDebug.Log(context.TODO(), log.LevelTrace, "Debug: Trace test")
}

func TestComponentLogger(t *testing.T) {
	l, err := log.New()
	require.NoError(t, err)

	l.Info("Message One")

	cfg := log.NewConfig(log.WithLevel(log.LevelTrace.String()), log.WithPlugin("slog"))

	sections := []string{"com", "example", "test", "logger"}
	data, err := config.ParseStruct(sections, &cfg)
	require.NoError(t, err)

	l2, err := l.WithConfig(sections, []source.Data{data})
	l2 = l2.With(slog.String("component", "logger"), slog.String("plugin", "slog"))
	require.NoError(t, err)

	l2.Info("Message Two")
	l2.Debug("Debug Two")
}

func TestCreateCustomLogger(t *testing.T) {
	l, err := log.New(log.WithPlugin("slog"), WithFormat("json"), WithFile("os.Stdout"))
	require.NoError(t, err)

	l.Info("json stdout test")
}
