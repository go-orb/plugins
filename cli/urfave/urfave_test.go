package urfave

import (
	"testing"

	"github.com/go-orb/go-orb/cli"

	"github.com/stretchr/testify/require"
)

// TestEmptyParse tests parsing with an empty command list.
func TestEmptyParse(t *testing.T) {
	appContext := cli.NewAppContext(&cli.App{
		Name:           "testapp",
		Usage:          "A testapp",
		Commands:       []*cli.Command{},
		Flags:          []*cli.Flag{},
		NoGlobalConfig: true,
	})

	flags, err := Parse(appContext, []string{})
	require.NoError(t, err)
	require.Empty(t, flags, "expected 0 flags")
}

// TestGlobalFlags tests parsing with global flags.
func TestGlobalFlags(t *testing.T) {
	strFlag := cli.NewFlag("str-flag", "default-value", cli.FlagUsage("A string flag"))
	intFlag := cli.NewFlag("int-flag", 42, cli.FlagUsage("An integer flag"))
	sliceFlag := cli.NewFlag("slice-flag", []string{"val1", "val2"}, cli.FlagUsage("A string slice flag"))

	appContext := cli.NewAppContext(&cli.App{
		Name:           "testapp",
		Usage:          "A testapp with global flags",
		Flags:          []*cli.Flag{strFlag, intFlag, sliceFlag},
		NoGlobalConfig: true,
	})

	// Test with no flags set (using defaults)
	flags, err := Parse(appContext, []string{"testapp"})
	require.NoError(t, err)
	require.Len(t, flags, 3, "expected 3 flags")

	// Test with flags set via command line
	flags, err = Parse(appContext, []string{
		"testapp",
		"--str-flag", "custom-value",
		"--int-flag", "100",
		"--slice-flag", "val3",
		"--slice-flag", "val4",
	})

	require.NoError(t, err)
	require.Len(t, flags, 3, "expected 3 flags")

	// Verify flag values
	for _, flag := range flags {
		switch flag.Name {
		case "str-flag":
			require.Equal(t, "custom-value", flag.Value)
		case "int-flag":
			require.Equal(t, 100, flag.Value)
		case "slice-flag":
			require.Equal(t, []string{"val3", "val4"}, flag.Value)
		default:
			t.Fatalf("unexpected flag: %s", flag.Name)
		}
	}
}

// TestCommands tests parsing with commands.
func TestCommands(t *testing.T) {

	// Create command with flags
	cmdFlag1 := cli.NewFlag("cmd-flag1", "cmd-default", cli.FlagUsage("A command flag"))
	cmdFlag2 := cli.NewFlag("cmd-flag2", 123, cli.FlagUsage("Another command flag"))

	cmd := &cli.Command{
		Name:    "cmd",
		Service: "service1",
		Usage:   "A test command",
		Flags:   []*cli.Flag{cmdFlag1, cmdFlag2},
	}

	appContext := cli.NewAppContext(&cli.App{
		Name:           "testapp",
		Usage:          "A testapp with commands",
		Commands:       []*cli.Command{cmd},
		NoGlobalConfig: true,
	})

	// Test with command and flags
	flags, err := Parse(appContext, []string{
		"testapp", "cmd",
		"--cmd-flag1", "custom-cmd-value",
		"--cmd-flag2", "456",
	})

	require.NoError(t, err)
	require.Len(t, flags, 2, "expected 2 flags")

	// Verify selected command
	require.Equal(t, []string{"cmd"}, appContext.SelectedCommand)
	require.Equal(t, "service1", appContext.SelectedService)

	// Verify flag values
	for _, flag := range flags {
		switch flag.Name {
		case "cmd-flag1":
			require.Equal(t, "custom-cmd-value", flag.Value)
		case "cmd-flag2":
			require.Equal(t, 456, flag.Value)
		default:
			t.Fatalf("unexpected flag: %s", flag.Name)
		}
	}
}

// TestNestedCommands tests parsing with nested commands.
func TestNestedCommands(t *testing.T) {

	// Create nested subcommands
	subSubCmdFlag := cli.NewFlag("subsub-flag", "subsub-default", cli.FlagUsage("A sub-sub command flag"))
	subSubCmd := &cli.Command{
		Name:    "subsub",
		Service: "service3",
		Usage:   "A sub-sub command",
		Flags:   []*cli.Flag{subSubCmdFlag},
	}

	subCmdFlag := cli.NewFlag("sub-flag", []string{"sub-val1"}, cli.FlagUsage("A sub command flag"))
	subCmd := &cli.Command{
		Name:        "sub",
		Service:     "service2",
		Usage:       "A sub command",
		Flags:       []*cli.Flag{subCmdFlag},
		Subcommands: []*cli.Command{subSubCmd},
	}

	cmdFlag := cli.NewFlag("cmd-flag", 99, cli.FlagUsage("A command flag"))
	cmd := &cli.Command{
		Name:        "cmd",
		Service:     "service1",
		Usage:       "A main command",
		Flags:       []*cli.Flag{cmdFlag},
		Subcommands: []*cli.Command{subCmd},
	}

	appContext := cli.NewAppContext(&cli.App{
		Name:           "testapp",
		Usage:          "A testapp with nested commands",
		Commands:       []*cli.Command{cmd},
		NoGlobalConfig: true,
	})

	// Test with deeply nested command
	flags, err := Parse(appContext, []string{
		"testapp", "cmd", "sub", "subsub",
		"--subsub-flag", "custom-subsub-value",
	})

	require.NoError(t, err)
	require.Len(t, flags, 1, "expected 1 flag")

	// Verify selected command
	require.Equal(t, []string{"cmd", "sub", "subsub"}, appContext.SelectedCommand)
	require.Equal(t, "service3", appContext.SelectedService)

	// Verify flag values
	require.Equal(t, "custom-subsub-value", flags[0].Value)
}

// TestMixedFlagsAndCommands tests parsing with both global and command flags.
func TestMixedFlagsAndCommands(t *testing.T) {

	// Create global flags
	globalFlag1 := cli.NewFlag("global-flag1", "global-default", cli.FlagUsage("A global flag"))
	globalFlag2 := cli.NewFlag("global-flag2", 789, cli.FlagUsage("Another global flag"))

	// Create command with flags
	cmdFlag := cli.NewFlag("cmd-flag", "cmd-default", cli.FlagUsage("A command flag"))
	cmd := &cli.Command{
		Name:    "cmd",
		Service: "service1",
		Usage:   "A test command",
		Flags:   []*cli.Flag{cmdFlag},
	}

	appContext := cli.NewAppContext(&cli.App{
		Name:           "testapp",
		Usage:          "A testapp with global flags and commands",
		Commands:       []*cli.Command{cmd},
		Flags:          []*cli.Flag{globalFlag1, globalFlag2},
		NoGlobalConfig: true,
	})

	// Test with global flags and command flags
	flags, err := Parse(appContext, []string{
		"testapp",
		"--global-flag1", "custom-global-value",
		"--global-flag2", "999",
		"cmd",
		"--cmd-flag", "custom-cmd-value",
	})

	require.NoError(t, err)
	require.Len(t, flags, 3, "expected 3 flags (2 global + 1 command)")

	// Verify selected command
	require.Equal(t, []string{"cmd"}, appContext.SelectedCommand)

	// Check flag values
	globalFlag1Found := false
	globalFlag2Found := false
	cmdFlagFound := false

	for _, flag := range flags {
		switch flag.Name {
		case "global-flag1":
			require.Equal(t, "custom-global-value", flag.Value)
			globalFlag1Found = true
		case "global-flag2":
			require.Equal(t, 999, flag.Value)
			globalFlag2Found = true
		case "cmd-flag":
			require.Equal(t, "custom-cmd-value", flag.Value)
			cmdFlagFound = true
		}
	}

	require.True(t, globalFlag1Found, "global-flag1 not found in results")
	require.True(t, globalFlag2Found, "global-flag2 not found in results")
	require.True(t, cmdFlagFound, "cmd-flag not found in results")
}

// TestMultipleCommands tests parsing with multiple commands at the same level.
func TestMultipleCommands(t *testing.T) {

	// Create multiple commands
	cmd1Flag := cli.NewFlag("cmd1-flag", "cmd1-default", cli.FlagUsage("Command 1 flag"))
	cmd1 := &cli.Command{
		Name:    "cmd1",
		Service: "service1",
		Usage:   "First command",
		Flags:   []*cli.Flag{cmd1Flag},
	}

	cmd2Flag := cli.NewFlag("cmd2-flag", "cmd2-default", cli.FlagUsage("Command 2 flag"))
	cmd2 := &cli.Command{
		Name:    "cmd2",
		Service: "service2",
		Usage:   "Second command",
		Flags:   []*cli.Flag{cmd2Flag},
	}

	appContext := cli.NewAppContext(&cli.App{
		Name:           "testapp",
		Usage:          "A testapp with multiple commands",
		Commands:       []*cli.Command{cmd1, cmd2},
		NoGlobalConfig: true,
	})

	// Test with second command
	flags, err := Parse(appContext, []string{
		"testapp", "cmd2",
		"--cmd2-flag", "custom-cmd2-value",
	})

	require.NoError(t, err)
	require.Len(t, flags, 1, "expected 1 flag")

	// Verify selected command
	require.Equal(t, []string{"cmd2"}, appContext.SelectedCommand)
	require.Equal(t, "service2", appContext.SelectedService)

	// Verify flag value
	require.Equal(t, "custom-cmd2-value", flags[0].Value)
}
