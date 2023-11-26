// Package orblog is an internal wrapper of hertz/pkg/common/hlog for go-orb/go-orb/log.
package orblog

import (
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/go-orb/go-orb/log"
)

// NewLogger creates a new hertz/hlog->go-orb/log wrapper.
func NewLogger(l log.Logger) *Logger {
	return &Logger{l: l}
}

// Logger is the wrapper for hertz/hlog->go-orb/log.
type Logger struct {
	l log.Logger
}

func (l *Logger) log(level hlog.Level, v ...any) {
	lvl := hLevelToSLevel(level)
	l.l.Log(context.TODO(), lvl, fmt.Sprint(v...))
}

func (l *Logger) logf(level hlog.Level, format string, kvs ...any) {
	lvl := hLevelToSLevel(level)
	l.l.Log(context.TODO(), lvl, fmt.Sprintf(format, kvs...))
}

func (l *Logger) ctxLogf(ctx context.Context, level hlog.Level, format string, v ...any) {
	lvl := hLevelToSLevel(level)
	l.l.Log(ctx, lvl, fmt.Sprintf(format, v...))
}

// Trace logs.
func (l *Logger) Trace(v ...any) {
	l.log(hlog.LevelTrace, v...)
}

// Debug logs.
func (l *Logger) Debug(v ...any) {
	l.log(hlog.LevelDebug, v...)
}

// Info logs.
func (l *Logger) Info(v ...any) {
	l.log(hlog.LevelInfo, v...)
}

// Notice logs.
func (l *Logger) Notice(v ...any) {
	l.log(hlog.LevelNotice, v...)
}

// Warn logs.
func (l *Logger) Warn(v ...any) {
	l.log(hlog.LevelWarn, v...)
}

// Error logs.
func (l *Logger) Error(v ...any) {
	l.log(hlog.LevelError, v...)
}

// Fatal logs.
func (l *Logger) Fatal(v ...any) {
	l.log(hlog.LevelFatal, v...)
}

// Tracef logs.
func (l *Logger) Tracef(format string, v ...any) {
	l.logf(hlog.LevelTrace, format, v...)
}

// Debugf logs.
func (l *Logger) Debugf(format string, v ...any) {
	l.logf(hlog.LevelDebug, format, v...)
}

// Infof logs.
func (l *Logger) Infof(format string, v ...any) {
	l.logf(hlog.LevelInfo, format, v...)
}

// Noticef logs.
func (l *Logger) Noticef(format string, v ...any) {
	l.logf(hlog.LevelNotice, format, v...)
}

// Warnf logs.
func (l *Logger) Warnf(format string, v ...any) {
	l.logf(hlog.LevelWarn, format, v...)
}

// Errorf logs.
func (l *Logger) Errorf(format string, v ...any) {
	l.logf(hlog.LevelError, format, v...)
}

// Fatalf logs.
func (l *Logger) Fatalf(format string, v ...any) {
	l.logf(hlog.LevelFatal, format, v...)
}

// CtxTracef logs.
func (l *Logger) CtxTracef(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelDebug, format, v...)
}

// CtxDebugf logs.
func (l *Logger) CtxDebugf(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelDebug, format, v...)
}

// CtxInfof logs.
func (l *Logger) CtxInfof(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelInfo, format, v...)
}

// CtxNoticef logs.
func (l *Logger) CtxNoticef(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelNotice, format, v...)
}

// CtxWarnf logs.
func (l *Logger) CtxWarnf(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelWarn, format, v...)
}

// CtxErrorf logs.
func (l *Logger) CtxErrorf(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelError, format, v...)
}

// CtxFatalf logs.
func (l *Logger) CtxFatalf(ctx context.Context, format string, v ...any) {
	l.ctxLogf(ctx, hlog.LevelFatal, format, v...)
}

// SetLevel a noop.
func (l *Logger) SetLevel(_ hlog.Level) {
	// lvl := hLevelToSLevel(level)
	// l.cfg.level.Set(lvl)
}

// SetOutput a noop.
func (l *Logger) SetOutput(_ io.Writer) {
	// l.cfg.output = writer
	// l.l = slog.New(slog.NewJSONHandler(writer, l.cfg.handlerOptions))
}
