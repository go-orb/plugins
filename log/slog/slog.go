// Package slog provides the slog handler.
package slog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-orb/go-orb/config"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/types"
	"golang.org/x/exp/slog"
)

// Name is this providers name.
const Name = "slog"

const (
	// DefaultFormat is the default format for slog.
	DefaultFormat = "text"
	// DefaultTarget is the default target for slog.
	DefaultTarget = "os.Stderr"
)

// The register.
func init() {
	log.Register(Name, Factory)
}

// Config is the config struct for slog.
type Config struct {
	log.Config

	Format string `json:"format" yaml:"format"`
	Target string `json:"target" yaml:"target"`
}

// NewConfig creates a new config.
func NewConfig(section []string, configs types.ConfigData, opts ...log.Option) (Config, error) {
	cfg := Config{
		Config: log.NewConfig(),
		Format: DefaultFormat,
		Target: DefaultTarget,
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

// WithTarget sets the target for the logger,
// available options: os.Stdout, os.Stderr, /somedir/somefile.
func WithTarget(n string) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Target = n
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

	switch p.config.Target {
	case "os.Stdout":
		w = os.Stdout
	case "os.Stderr":
		w = os.Stderr
	default:
		f, err := os.OpenFile(p.config.Target, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return fmt.Errorf("while opening '%s': %w", p.config.Target, err)
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
	return fmt.Sprintf("__%s__-%s-%s", Name, p.config.Format, p.config.Target)
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
