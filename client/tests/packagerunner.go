package tests

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/go-orb/go-orb/log"
	"golang.org/x/exp/slog"
)

// PackageRunnerOptions contains the options.
type PackageRunnerOptions struct {
	OverWrite    bool
	StdOut       io.Writer
	StdErr       io.Writer
	RunEnv       []string
	Args         []string
	NumProcesses int
}

// PackageRunnerOption is a option, see below.
type PackageRunnerOption func(*PackageRunnerOptions)

// WithOverwrite recompiles the binary.
func WithOverwrite() PackageRunnerOption {
	return func(o *PackageRunnerOptions) {
		o.OverWrite = true
	}
}

// WithStdOut sets where to write stdout.
func WithStdOut(n io.Writer) PackageRunnerOption {
	return func(o *PackageRunnerOptions) {
		o.StdOut = n
	}
}

// WithStdErr sets where to write stderr.
func WithStdErr(n io.Writer) PackageRunnerOption {
	return func(o *PackageRunnerOptions) {
		o.StdErr = n
	}
}

// WithRunEnv sets environment variables in the form of KEY=VAR.
func WithRunEnv(n ...string) PackageRunnerOption {
	return func(o *PackageRunnerOptions) {
		o.RunEnv = n
	}
}

// WithArgs sets commandline args.
func WithArgs(n ...string) PackageRunnerOption {
	return func(o *PackageRunnerOptions) {
		o.Args = n
	}
}

// WithNumProcesses set's the number of parallel processes to start.
func WithNumProcesses(n int) PackageRunnerOption {
	return func(o *PackageRunnerOptions) {
		o.NumProcesses = n
	}
}

// PackageRunner builds and runs go packages.
type PackageRunner struct {
	logger log.Logger

	options PackageRunnerOptions

	packagePath string
	binaryPath  string

	tempDir      string
	subprocesses []*exec.Cmd
}

// NewPackageRunner creates a new package runner.
func NewPackageRunner(logger log.Logger, packagePath string, binaryPath string, opts ...PackageRunnerOption) *PackageRunner {
	options := PackageRunnerOptions{
		NumProcesses: 1,
	}
	for _, o := range opts {
		o(&options)
	}

	return &PackageRunner{
		logger:       logger.With(slog.String("component", "packageRunner")),
		options:      options,
		packagePath:  packagePath,
		binaryPath:   binaryPath,
		subprocesses: []*exec.Cmd{},
	}
}

// Build makes a binary from the package.
func (p *PackageRunner) Build() error {
	if p.binaryPath == "" {
		// Extract the binary name from the package path
		binaryName := filepath.Base(p.packagePath)

		// Create a temporary directory to store the built binary
		tmpDir, err := os.MkdirTemp(os.TempDir(), "")
		if err != nil {
			return fmt.Errorf("while creating a temporary directory: %w", err)
		}

		p.tempDir = tmpDir

		// Build the Go package and generate the binary in the temporary directory
		p.binaryPath = filepath.Join(p.tempDir, binaryName)
	} else {
		dir := filepath.Dir(p.binaryPath)

		err := os.MkdirAll(dir, 0o700)
		if err != nil {
			return fmt.Errorf("while creating directories: %w", err)
		}

		path, err := filepath.Abs(filepath.Join(dir, filepath.Base(p.packagePath)))
		if err != nil {
			return fmt.Errorf("while creating directories: %w", err)
		}

		p.binaryPath = path
	}

	// Do not overwrite.
	if !p.options.OverWrite {
		if _, err := os.Stat(p.binaryPath); err == nil {
			return nil
		}
	}

	p.logger.Debug("Compiling", "packagePath", p.packagePath, "binaryPath", p.binaryPath)
	buildCmd := exec.Command("go", "build", "-o", p.binaryPath, p.packagePath) //nolint:gosec

	err := buildCmd.Run()
	if err != nil {
		return fmt.Errorf("whilte building the package: %w", err)
	}

	return nil
}

// Start runs the binary build with Build().
func (p *PackageRunner) Start() error {
	p.logger.Debug("Starting", "binaryPath", p.binaryPath, "numProcs", p.options.NumProcesses, "args", p.options.Args)

	for i := 0; i < p.options.NumProcesses; i++ {
		// Run the compiled binary as a subprocess
		cmd := exec.Command(p.binaryPath, p.options.Args...) //nolint:gosec

		cmd.Env = append(cmd.Env, p.options.RunEnv...)

		// Redirect standard output and error to display the output in the console (optional)
		if p.options.StdOut != nil {
			cmd.Stdout = p.options.StdOut
		}

		if p.options.StdErr != nil {
			cmd.Stderr = p.options.StdErr
		}

		// Start the command
		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("error starting the subprocess: %w", err)
		}

		p.subprocesses = append(p.subprocesses, cmd)
	}

	return nil
}

// Kill kills the started process.
func (p *PackageRunner) Kill() error {
	if len(p.subprocesses) < 1 {
		return nil
	}

	for i := 0; i < p.options.NumProcesses; i++ {
		cmd := p.subprocesses[i]

		if cmd != nil && runtime.GOOS != "windows" {
			// On Unix-like systems, send the SIGINT signal (equivalent to Ctrl+C)
			err := cmd.Process.Signal(os.Interrupt)
			if err != nil {
				return fmt.Errorf("error killing the subprocess: %w", err)
			}
		} else if cmd != nil {
			// On Windows, unfortunately, sending signals is not supported directly.
			// We'll have to forcefully kill the process instead.
			err := cmd.Process.Kill()
			if err != nil {
				return fmt.Errorf("error killing the subprocess: %w", err)
			}
		}

		if cmd != nil {
			// Wait for the command to complete using the `Wait()` method to ensure the subprocess has terminated.
			err := cmd.Wait()
			if err != nil {
				return fmt.Errorf("error waiting for the subprocess: %w", err)
			}
		}
	}

	// Remove the temporary directory
	if p.tempDir != "" {
		if err := os.RemoveAll(p.tempDir); err != nil {
			return fmt.Errorf("error removing temporary directory: %w", err)
		}

		p.tempDir = ""
	}

	return nil
}
