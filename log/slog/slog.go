// Package slog provides the slog handler.
package slog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"log/slog"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
)

// Name is this providers name.
const Name = "slog"

const (
	// DefaultFormat is the default format for slog.
	DefaultFormat = "text"
	// DefaultFile is the default target for slog.
	DefaultFile = "os.Stderr"
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
	File   string `json:"file"   yaml:"file"`
}

// NewConfig creates a new config.
func NewConfig(opts ...log.Option) Config {
	cfg := Config{
		Config: log.NewConfig(),
		Format: DefaultFormat,
		File:   DefaultFile,
	}

	for _, o := range opts {
		o(&cfg)
	}

	return cfg
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

var _ (log.Provider) = (*Provider)(nil)

// Provider is the provider for slog.
type Provider struct {
	config Config

	file    *os.File
	handler slog.Handler
}

// Start configures the slog Handler.
func (p *Provider) Start() error {
	var w io.Writer

	switch p.config.File {
	case "os.Stdout":
		w = os.Stdout
	case "os.Stderr":
		w = os.Stderr
	default:
		f, err := os.OpenFile(p.config.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("while opening '%s': %w", p.config.File, err)
		}

		p.file = f
		w = f
	}

	switch strings.ToLower(p.config.Format) {
	case "text":
		p.handler = slog.NewTextHandler(w, nil)
	case "json":
		p.handler = slog.NewJSONHandler(w, nil)
	default:
		return errors.New("unknown format given")
	}

	return nil
}

// Stop closes if required a open log file.
func (p *Provider) Stop(_ context.Context) error {
	if p.file != nil {
		return p.file.Close()
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
func Factory(sections []string, configs map[string]any, opts ...log.Option) (log.ProviderType, error) {
	cfg := NewConfig(opts...)

	if err := config.Parse(sections, "logger", configs, &cfg); err != nil && !errors.Is(err, config.ErrNoSuchKey) {
		return log.ProviderType{}, err
	}

	return log.ProviderType{
		Provider: &Provider{
			config: cfg,
		},
	}, nil
}
