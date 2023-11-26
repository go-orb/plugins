// Package lumberjack provides the slog handler.
package lumberjack

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Name is this providers name.
const Name = "lumberjack"

const (
	// DefaultFormat is the default format for lumberjack.
	DefaultFormat = "json"
)

// The register.
func init() {
	log.Register(Name, Factory)
}

// Config is the config struct for slog.
type Config struct {
	log.Config

	// Format is the log format, either json or text.
	Format string `json:"format" yaml:"format"`
	// File is the file to write logs to.  Backup log files will be retained
	// in the same directory.  It uses <processname>-lumberjack.log in
	// os.TempDir() if empty.
	File string `json:"file" yaml:"file"`

	// MaxSize is the maximum size in megabytes of the log file before it gets
	// rotated. It defaults to 100 megabytes.
	MaxSize int `json:"maxSize" yaml:"maxSize"`

	// MaxBackups is the maximum number of old log files to retain.  The default
	// is to retain all old log files (though MaxAge may still cause them to get
	// deleted.)
	MaxBackups int `json:"maxBackups" yaml:"maxBackups"`

	// MaxAge is the maximum number of days to retain old log files based on the
	// timestamp encoded in their filename.  Note that a day is defined as 24
	// hours and may not exactly correspond to calendar days due to daylight
	// savings, leap seconds, etc. The default is not to remove old log files
	// based on age.
	MaxAge int `json:"maxAge" yaml:"maxAge"`

	// LocalTime determines if the time used for formatting the timestamps in
	// backup files is the computer's local time.  The default is to use UTC
	// time.
	LocalTime bool `json:"localTime" yaml:"localTime"`

	// Compress determines if the rotated log files should be compressed
	// using gzip. The default is not to perform compression.
	Compress bool `json:"compress" yaml:"compress"`
}

// NewConfig creates a new config.
func NewConfig(section []string, configs types.ConfigData, opts ...log.Option) (Config, error) {
	cfg := Config{
		Config: log.NewConfig(),
		Format: DefaultFormat,
	}

	for _, o := range opts {
		o(&cfg)
	}

	if err := config.Parse(append(section, "logger"), configs, &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// WithFormat sets the format for the logger.
func WithFormat(n string) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Format = n
		}
	}
}

// WithFile sets the target for the logger,
// available options: os.Stdout, os.Stderr, /somedir/somefile.
func WithFile(n string) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.File = n
		}
	}
}

// WithMaxSize is the maximum size in megabytes of the log file before it gets
// rotated. It defaults to 100 megabytes.
func WithMaxSize(n int) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.MaxSize = n
		}
	}
}

// WithMaxBackups is the maximum number of old log files to retain.  The default
// is to retain all old log files (though MaxAge may still cause them to get
// deleted.)
func WithMaxBackups(n int) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.MaxBackups = n
		}
	}
}

// WithMaxAge is the maximum number of days to retain old log files based on the
// timestamp encoded in their filename.  Note that a day is defined as 24
// hours and may not exactly correspond to calendar days due to daylight
// savings, leap seconds, etc. The default is not to remove old log files
// based on age.
func WithMaxAge(n int) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.MaxAge = n
		}
	}
}

// WithLocalTime determines if the time used for formatting the timestamps in
// backup files is the computer's local time.  The default is to use UTC
// time.
func WithLocalTime(n bool) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.LocalTime = n
		}
	}
}

// WithCompress determines if the rotated log files should be compressed
// using gzip. The default is not to perform compression.
func WithCompress(n bool) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.LocalTime = n
		}
	}
}

var _ (log.Provider) = (*Provider)(nil)

// Provider is the provider for slog.
type Provider struct {
	config Config

	lumberjack *lumberjack.Logger

	handler slog.Handler
}

// Start configures the slog Handler.
func (p *Provider) Start() error {
	p.lumberjack = &lumberjack.Logger{
		Filename:   p.config.File,
		MaxSize:    p.config.MaxSize,
		MaxBackups: p.config.MaxBackups,
		MaxAge:     p.config.MaxAge,
		LocalTime:  p.config.LocalTime,
		Compress:   p.config.Compress,
	}

	switch strings.ToLower(p.config.Format) {
	case "text":
		p.handler = slog.NewTextHandler(p.lumberjack, nil)
	case "json":
		p.handler = slog.NewJSONHandler(p.lumberjack, nil)
	default:
		return errors.New("unknown format given")
	}

	return nil
}

// Stop closes if required a open log file.
func (p *Provider) Stop(_ context.Context) error {
	if p.lumberjack != nil {
		return p.lumberjack.Close()
	}

	return nil
}

// Handler returns the configure handler.
func (p *Provider) Handler() (slog.Handler, error) {
	return p.handler, nil
}

// Key returns an identifier for this handler provider with its config.
func (p *Provider) Key() string {
	return fmt.Sprintf("__%s__-%s-%s", Name, p.config.Format, p.config.File)
}

// Factory is the factory for a slog provider.
func Factory(sections []string, configs types.ConfigData, opts ...log.Option) (log.ProviderType, error) {
	cfg, err := NewConfig(sections, configs, opts...)
	if err != nil {
		return log.ProviderType{}, err
	}

	return log.ProviderType{
		Provider: &Provider{
			config: cfg,
		},
	}, nil
}
