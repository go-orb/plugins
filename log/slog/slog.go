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

const Name = "slog"

var (
	DefaultFormat = "text"
	DefaultTarget = "os.Stderr"
)

// The register, it's the same as the old func init() {}
var _ = log.Register(Name, Provide)

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

// WithTarget sets the format for the logger.
func WithFormat(n string) log.Option {
	return func(c log.ConfigType) {
		cfg, ok := c.(*Config)
		if ok {
			cfg.Format = n
		}
	}
}

// WithTarget sets the target for the logger,
// available options: os.Stdout, os.Stderr, /somedir/somefile
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

func (p *Provider) Start() error {
	var w io.Writer

	if p.config.Target == "os.Stdout" {
		w = os.Stdout
	} else if p.config.Target == "os.Stderr" {
		w = os.Stderr
	} else if p.config.Target == "" {
		w = os.Stderr
	} else {
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

func (p *Provider) Stop(ctx context.Context) error {
	if p.file != nil {
		return p.file.Close()
	}

	return nil
}

func (p *Provider) Handler() (slog.Handler, error) {
	return p.handler, nil
}

// String returns an identifier for this handler provider with its config.
func (p *Provider) String() string {
	return fmt.Sprintf("__%s__-%s-%s", Name, p.config.Format, p.config.Target)
}

func Provide(sections []string, configs types.ConfigData, opts ...log.Option) (log.ProviderType, error) {
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
