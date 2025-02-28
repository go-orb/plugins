// Package urfave is a cli wrapper for urfave.
package urfave

import (
	"fmt"
	"os"
	"slices"

	"github.com/go-orb/go-orb/codecs"
	"github.com/go-orb/go-orb/config/source"
	oCli "github.com/go-orb/go-orb/config/source/cli"
	"github.com/hashicorp/go-multierror"
	"github.com/urfave/cli/v2"
)

func init() {

}

type flagCLI struct {
	app              *cli.App
	flags            map[string]*oCli.Flag
	stringFlags      map[string]*cli.StringFlag
	intFlags         map[string]*cli.IntFlag
	stringSliceFlags map[string]*cli.StringSliceFlag
}

// Parse parses all the CLI flags.
func Parse(app *cli.App, flags []*oCli.Flag, args []string) source.Data {
	result := source.Data{
		Data: make(map[string]any),
	}

	flagParser := &flagCLI{
		app:              app,
		flags:            make(map[string]*oCli.Flag),
		stringFlags:      make(map[string]*cli.StringFlag),
		intFlags:         make(map[string]*cli.IntFlag),
		stringSliceFlags: make(map[string]*cli.StringSliceFlag),
	}

	var mErr *multierror.Error

	for _, f := range flags {
		if err := flagParser.add(f); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}

	if mErr.ErrorOrNil() != nil {
		result.Error = mErr
		return result
	}

	if err := flagParser.parse(args); err != nil {
		result.Error = err
		return result
	}

	oCli.ParseFlags(&result, flags)

	mJSON, err := codecs.GetMime("application/json")
	if err != nil {
		result.Error = err
	} else {
		result.Marshaler = mJSON
	}

	// Clear all flags for future runs.
	for _, f := range flags {
		f.Clear()
	}

	return result
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

	app := c.app
	app.Flags = append(app.Flags, flags...)
	action := app.Action

	app.Action = func(fCtx *cli.Context) error {
		// Extract the ctx from the urfave app
		ctx = fCtx

		if action != nil {
			return action(ctx)
		}

		return nil
	}

	if len(app.Version) < 1 {
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
		if c.flags[n].Default != tf.Get(ctx) {
			c.flags[n].Value = tf.Get(ctx)
		}
	}

	for n, tf := range c.stringFlags {
		if c.flags[n].Default != tf.Get(ctx) {
			c.flags[n].Value = tf.Get(ctx)
		}
	}

	for n, tf := range c.stringSliceFlags {
		if !slices.Equal(c.flags[n].Default.([]string), tf.Get(ctx)) { //nolint:errcheck
			c.flags[n].Value = tf.Get(ctx)
		}
	}

	return nil
}
