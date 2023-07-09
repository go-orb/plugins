// Package urfave is a cli wrapper for urfave.
package urfave

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
	oCli "github.com/go-orb/go-orb/config/source/cli"
)

func init() {
	if err := oCli.Plugins.Add(
		"urfave",
		Parse,
	); err != nil {
		panic(err)
	}
}

type flagCLI struct {
	flags            map[string]*oCli.Flag
	stringFlags      map[string]*cli.StringFlag
	intFlags         map[string]*cli.IntFlag
	stringSliceFlags map[string]*cli.StringSliceFlag
	config           *oCli.Config
}

// Parse parses all the CLI flags.
func Parse(config *oCli.Config, flags []*oCli.Flag, args []string) error {
	parser := &flagCLI{
		config:           config,
		flags:            make(map[string]*oCli.Flag),
		stringFlags:      make(map[string]*cli.StringFlag),
		intFlags:         make(map[string]*cli.IntFlag),
		stringSliceFlags: make(map[string]*cli.StringSliceFlag),
	}

	var mErr *multierror.Error

	for _, f := range flags {
		if err := parser.add(f); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}

	if mErr.ErrorOrNil() != nil {
		return mErr
	}

	return parser.parse(args)
}

func (c *flagCLI) add(flag *oCli.Flag) error {
	switch d := flag.Default.(type) {
	case int:
		f := &cli.IntFlag{
			Name:    flag.Name,
			Usage:   flag.Usage,
			Value:   d,
			EnvVars: flag.EnvVars,
		}
		c.intFlags[flag.Name] = f
	case string:
		f := &cli.StringFlag{
			Name:    flag.Name,
			Usage:   flag.Usage,
			Value:   d,
			EnvVars: flag.EnvVars,
		}
		c.stringFlags[flag.Name] = f
	case []string:
		f := &cli.StringSliceFlag{
			Name:    flag.Name,
			Usage:   flag.Usage,
			Value:   cli.NewStringSlice(d...),
			EnvVars: flag.EnvVars,
		}
		c.stringSliceFlags[flag.Name] = f
	default:
		return fmt.Errorf("found a unknown flag: %s", flag.Name)
	}

	c.flags[flag.Name] = flag

	return nil
}

func (c *flagCLI) parse(args []string) error {
	i := 0
	flags := make([]cli.Flag, len(c.stringFlags)+len(c.intFlags)+len(c.stringSliceFlags))

	for _, f := range c.stringFlags {
		flags[i] = f
		i++
	}

	for _, f := range c.intFlags {
		flags[i] = f
		i++
	}

	for _, f := range c.stringSliceFlags {
		flags[i] = f
		i++
	}

	var ctx *cli.Context

	app := &cli.App{
		Name:    c.config.Name,
		Version: c.config.Version,
		Flags:   flags,
		Action: func(fCtx *cli.Context) error {
			// Extract the ctx from the urfave app
			ctx = fCtx

			return nil
		},
	}

	if len(c.config.Version) < 1 {
		app.HideVersion = true
	}

	if err := app.Run(args); err != nil {
		return err
	}

	// When help get's called ctx is nil so we exit out.
	if ctx == nil {
		os.Exit(0)
	}

	for n, tf := range c.intFlags {
		c.flags[n].Value = tf.Get(ctx)
	}

	for n, tf := range c.stringFlags {
		c.flags[n].Value = tf.Get(ctx)
	}

	for n, tf := range c.stringSliceFlags {
		c.flags[n].Value = tf.Get(ctx)
	}

	return nil
}
