// Package urfave is a cli wrapper for urfave.
package urfave

import (
	"fmt"
	"slices"

	"iter"

	oCli "github.com/go-orb/go-orb/cli"
	"github.com/hashicorp/go-multierror"
	uCli "github.com/urfave/cli/v2"
)

// ProvideParser provides a new parser.
func ProvideParser() (oCli.ParserFunc, error) {
	return Parse, nil
}

// Parse parses all the CLI flags.
func Parse(appContext *oCli.AppContext, args []string) ([]*oCli.Flag, error) {
	parser, err := newParser(appContext)
	if err != nil {
		return nil, err
	}

	return parser.run(args)
}

type parserCommand struct {
	service          string
	name             string
	previousCommands []string

	uCommand *uCli.Command

	flags            map[string]*oCli.Flag
	stringFlags      map[string]*uCli.StringFlag
	intFlags         map[string]*uCli.IntFlag
	stringSliceFlags map[string]*uCli.StringSliceFlag

	subCommands []*parserCommand
}

func (pc *parserCommand) add(flag *oCli.Flag) error {
	switch d := flag.Default.(type) {
	case int:
		f := &uCli.IntFlag{
			Name:    flag.Name,
			Usage:   flag.Usage,
			Value:   d,
			EnvVars: flag.EnvVars,
		}
		pc.intFlags[flag.Name] = f
	case string:
		f := &uCli.StringFlag{
			Name:    flag.Name,
			Usage:   flag.Usage,
			Value:   d,
			EnvVars: flag.EnvVars,
		}
		pc.stringFlags[flag.Name] = f
	case []string:
		f := &uCli.StringSliceFlag{
			Name:    flag.Name,
			Usage:   flag.Usage,
			Value:   uCli.NewStringSlice(d...),
			EnvVars: flag.EnvVars,
		}
		pc.stringSliceFlags[flag.Name] = f
	default:
		return fmt.Errorf("found a unknown flag: %s", flag.Name)
	}

	pc.flags[flag.Name] = flag

	return nil
}

func (pc *parserCommand) urfaveFlags() []uCli.Flag {
	i := 0
	flags := make([]uCli.Flag, len(pc.stringFlags)+len(pc.intFlags)+len(pc.stringSliceFlags))

	for _, f := range pc.stringFlags {
		flags[i] = f
		i++
	}

	for _, f := range pc.intFlags {
		flags[i] = f
		i++
	}

	for _, f := range pc.stringSliceFlags {
		flags[i] = f
		i++
	}

	return flags
}

func (pc *parserCommand) IsCommand(currentCommand []string) bool {
	return slices.Equal(append(pc.previousCommands, pc.name), currentCommand)
}

// flatten returns a sequence iterator that yields the command and all its subcommands.
func (pc *parserCommand) flatten() iter.Seq[*parserCommand] {
	return iter.Seq[*parserCommand](func(yield func(*parserCommand) bool) {
		// First yield the current command
		if !yield(pc) {
			return
		}

		// Then recursively yield all subcommands
		for _, subCmd := range pc.subCommands {
			for subCmd2 := range subCmd.flatten() {
				// Pass each subcommand to the yield function
				// If yield returns false, we stop iteration
				if !yield(subCmd2) {
					return
				}
			}
		}
	})
}

func newParserCommand(service string, name string, previousCommands []string) *parserCommand {
	return &parserCommand{
		service:          service,
		name:             name,
		previousCommands: previousCommands,

		flags:            make(map[string]*oCli.Flag),
		stringFlags:      make(map[string]*uCli.StringFlag),
		intFlags:         make(map[string]*uCli.IntFlag),
		stringSliceFlags: make(map[string]*uCli.StringSliceFlag),
	}
}

func newParserCommandFromOcli(oCommand *oCli.Command, previousCommands []string) (*parserCommand, error) {
	uCmd := &uCli.Command{
		Name:        oCommand.Name,
		Usage:       oCommand.Usage,
		Category:    oCommand.Category,
		Subcommands: make([]*uCli.Command, 0),
	}
	if !oCommand.NoAction {
		uCmd.Action = func(_ *uCli.Context) error {
			return oCommand.InternalAction()
		}
	}

	pCmd := newParserCommand(oCommand.Service, oCommand.Name, previousCommands)
	pCmd.uCommand = uCmd

	var mErr *multierror.Error

	for _, f := range oCommand.Flags {
		if err := pCmd.add(f); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}

	uCmd.Flags = pCmd.urfaveFlags()

	if mErr.ErrorOrNil() != nil {
		return nil, mErr
	}

	for _, sub := range oCommand.Subcommands {
		pSub, err := newParserCommandFromOcli(sub, append(previousCommands, oCommand.Name))
		if err != nil {
			mErr = multierror.Append(mErr, err)
		}

		pCmd.subCommands = append(pCmd.subCommands, pSub)
		uCmd.Subcommands = append(uCmd.Subcommands, pSub.uCommand)
	}

	if mErr.ErrorOrNil() != nil {
		return nil, mErr
	}

	return pCmd, nil
}

type parser struct {
	appContext *oCli.AppContext
	uApp       *uCli.App

	globalFlags *parserCommand

	commands []*parserCommand
}

//nolint:gocognit,gocyclo,cyclop
func (p *parser) run(args []string) ([]*oCli.Flag, error) {
	var ctx *uCli.Context

	runAction := func(oldAction func(*uCli.Context) error) func(*uCli.Context) error {
		if oldAction == nil {
			return nil
		}

		return func(fCtx *uCli.Context) error {
			// Extract the ctx from the urfave app
			ctx = fCtx

			if oldAction != nil {
				return oldAction(fCtx)
			}

			return nil
		}
	}

	// Set the action for the app, its commands and all subcommands.
	p.uApp.Action = runAction(p.uApp.Action)

	for _, c := range p.commands {
		for c2 := range c.flatten() {
			c2.uCommand.Action = runAction(c2.uCommand.Action)
		}
	}

	if err := p.uApp.Run(args); err != nil {
		return nil, err
	}

	// When help get's called ctx is nil so we exit out.
	if ctx == nil {
		p.appContext.ExitGracefully(0)
	}

	flags := []*oCli.Flag{}

	for n, tf := range p.globalFlags.intFlags {
		if p.globalFlags.flags[n].Default != tf.Get(ctx) {
			p.globalFlags.flags[n].Value = tf.Get(ctx)
			flags = append(flags, p.globalFlags.flags[n])
		}
	}

	for n, tf := range p.globalFlags.stringFlags {
		if p.globalFlags.flags[n].Default != tf.Get(ctx) {
			p.globalFlags.flags[n].Value = tf.Get(ctx)
			flags = append(flags, p.globalFlags.flags[n])
		}
	}

	for n, tf := range p.globalFlags.stringSliceFlags {
		if !slices.Equal(p.globalFlags.flags[n].Default.([]string), tf.Get(ctx)) { //nolint:errcheck
			p.globalFlags.flags[n].Value = tf.Get(ctx)
			flags = append(flags, p.globalFlags.flags[n])
		}
	}

	for _, c := range p.commands {
		for c2 := range c.flatten() {
			if c2.IsCommand(p.appContext.SelectedCommand) {
				// Update the command flag values before collecting them
				for n, tf := range c2.intFlags {
					if c2.flags[n].Default != tf.Get(ctx) {
						c2.flags[n].Value = tf.Get(ctx)
						flags = append(flags, c2.flags[n])
					}
				}

				for n, tf := range c2.stringFlags {
					if c2.flags[n].Default != tf.Get(ctx) {
						c2.flags[n].Value = tf.Get(ctx)
						flags = append(flags, c2.flags[n])
					}
				}

				for n, tf := range c2.stringSliceFlags {
					if !slices.Equal(c2.flags[n].Default.([]string), tf.Get(ctx)) { //nolint:errcheck
						c2.flags[n].Value = tf.Get(ctx)
						flags = append(flags, c2.flags[n])
					}
				}
			}
		}
	}

	return flags, nil
}

func newParser(appContext *oCli.AppContext) (*parser, error) {
	oApp := appContext.App()
	uApp := &uCli.App{
		Name:    oApp.Name,
		Version: oApp.Version,
		Usage:   oApp.Usage,
	}

	if !oApp.NoAction {
		uApp.Action = func(_ *uCli.Context) error {
			return oApp.InternalAction()
		}
	}

	globalFlags := newParserCommand(oApp.Name, oCli.MainActionName, []string{})

	var mErr *multierror.Error

	for _, f := range oApp.Flags {
		if err := globalFlags.add(f); err != nil {
			mErr = multierror.Append(mErr, err)
		}
	}

	uApp.Flags = globalFlags.urfaveFlags()

	if mErr.ErrorOrNil() != nil {
		return nil, mErr
	}

	commands := make([]*parserCommand, 0, len(oApp.Commands))

	for _, c := range oApp.Commands {
		pCmd, err := newParserCommandFromOcli(c, []string{})
		if err != nil {
			mErr = multierror.Append(mErr, err)
			continue
		}

		commands = append(commands, pCmd)
		uApp.Commands = append(uApp.Commands, pCmd.uCommand)
	}

	if mErr.ErrorOrNil() != nil {
		return nil, mErr
	}

	return &parser{
		appContext:  appContext,
		uApp:        uApp,
		globalFlags: globalFlags,
		commands:    commands,
	}, nil
}
