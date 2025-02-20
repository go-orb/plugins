// A generated module for GoOrb functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/go-orb/internal/dagger"
	"runtime"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
)

type WorkerResult struct {
	Module string
	Logs   string
	Err    error
	Source *dagger.Directory
}

type AllResult struct {
	Logs   []string
	Source *dagger.Directory
}

type GoOrb struct{}

func (m *GoOrb) runAll(ctx context.Context, root *dagger.Directory, worker func(ctx context.Context, wg *sync.WaitGroup, inpupt <-chan string, res chan<- *WorkerResult, root *dagger.Directory)) (*AllResult, error) {
	mods, err := m.Modules(ctx, root)
	if err != nil {
		return nil, err
	}

	input := make(chan string, len(mods))
	res := make(chan *WorkerResult, len(mods))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go worker(ctx, &wg, input, res, root)
	}

	// Add jobs
	for _, mod := range mods {
		if len(mod) > 0 {
			input <- mod
		}
	}
	close(input)

	// Wait for all workers to complete
	wg.Wait()
	close(res)

	// Aggregate results
	result := &AllResult{Logs: []string{}, Source: dag.Directory()}
	for r := range res {
		if r.Err != nil {
			err = multierror.Append(err, r.Err)

			if len(r.Logs) > 0 {
				result.Logs = append(result.Logs, "## "+r.Module+"\n"+r.Logs+"\n\n")
			}
		}

		if r.Source != nil {
			result.Source = result.Source.WithDirectory(r.Module, r.Source)
		}
	}

	return result, err
}

// Returns all modules in `root`
func (m *GoOrb) Modules(ctx context.Context, root *dagger.Directory) ([]string, error) {
	mods, err := root.Glob(ctx, "**/go.mod")
	if err != nil {
		return nil, err
	}

	for i, mod := range mods {
		// Ignore dagger
		if strings.HasPrefix(mod, ".github") {
			mods[i] = ""
			continue
		}

		if len(mod) > 7 {
			mods[i] = mod[0 : len(mod)-7]
		} else {
			mods[i] = "."
		}
	}

	return mods, err
}

// Lints all modules starting from `root` with golangci-lint
func (m *GoOrb) Lint(
	ctx context.Context,
	root *dagger.Directory,
	// +defaultPath="/.golangci.yaml"
	golangciConfig *dagger.File,
) (*AllResult, error) {
	lintWorker := func(ctx context.Context, wg *sync.WaitGroup, input <-chan string, res chan<- *WorkerResult, root *dagger.Directory) {
		defer wg.Done()
		for dir := range input {
			select {
			case <-ctx.Done():
				return
			default:
			}

			out, err := dag.Container().From("golangci/golangci-lint:v1.64.5").
				WithMountedCache("/go/pkg/mod",
					dag.CacheVolume("go-mod"),
					dagger.ContainerWithMountedCacheOpts{
						Source:  dag.Directory().WithNewDirectory("~/go/pkg/mod"),
						Sharing: dagger.CacheSharingMode("SHARED"),
					},
				).
				WithMountedCache("/root/.cache/go-build",
					dag.CacheVolume("go-build"),
					dagger.ContainerWithMountedCacheOpts{
						Source:  dag.Directory().WithNewDirectory("~/.cache/go-build"),
						Sharing: dagger.CacheSharingMode("SHARED"),
					},
				).
				WithMountedCache("/root/.cache/golangci-lint", dag.CacheVolume("golangci-lint")).
				WithDirectory("/work/src", root.Directory(dir)).
				WithWorkdir("/work/src").
				WithMountedFile("/work/config", golangciConfig).
				WithExec([]string{"golangci-lint", "run", "--config", "/work/config", "--timeout=10m"}).
				Stdout(ctx)

			res <- &WorkerResult{Module: dir, Logs: out, Err: err}
		}
	}

	return m.runAll(ctx, root, lintWorker)
}

func (m *GoOrb) goContainer(root *dagger.Directory, dir string) *dagger.Container {
	return dag.Container().From("golang:1.23").
		WithMountedCache("/go/pkg/mod",
			dag.CacheVolume("go-mod"),
			dagger.ContainerWithMountedCacheOpts{
				Source:  dag.Directory().WithNewDirectory("~/go/pkg/mod"),
				Sharing: dagger.CacheSharingMode("SHARED"),
			},
		).
		WithMountedCache("/root/.cache/go-build",
			dag.CacheVolume("go-build"),
			dagger.ContainerWithMountedCacheOpts{
				Source:  dag.Directory().WithNewDirectory("~/.cache/go-build"),
				Sharing: dagger.CacheSharingMode("SHARED"),
			},
		).
		WithDirectory("/work/src", root.Directory(dir)).
		WithWorkdir("/work/src")
}

// Tests all modules starting from `root` with `go test ./... -v -race -cover`
func (m *GoOrb) Test(ctx context.Context, root *dagger.Directory) (*AllResult, error) {
	testWorker := func(ctx context.Context, wg *sync.WaitGroup, input <-chan string, res chan<- *WorkerResult, root *dagger.Directory) {
		defer wg.Done()
		for dir := range input {
			select {
			case <-ctx.Done():
				return
			default:
			}

			c := m.goContainer(root, dir).
				WithExec([]string{"apt-get", "-qy", "update"}).
				WithExec([]string{"apt-get", "-qy", "install", "unzip"}).
				WithExec([]string{"bash", "-c", "test -f ./scripts/pre_test.sh && ./scripts/pre_test.sh || exit 0"}).
				WithExec([]string{"go", "mod", "download"}).
				WithExec([]string{"go", "test", "./...", "-v", "-race", "-cover"})

			stdout, err := c.Stdout(ctx)
			if err != nil {
				res <- &WorkerResult{Module: dir, Err: err}
				continue
			}

			stderr, err := c.Stderr(ctx)
			if err != nil {
				res <- &WorkerResult{Module: dir, Err: err}
				continue
			}

			res <- &WorkerResult{Module: dir, Logs: stdout + stderr, Err: err}
		}
	}

	return m.runAll(ctx, root, testWorker)
}

// Runs `go mod tidy -go=1.23.0` in all modules starting with `root`
func (m *GoOrb) Tidy(ctx context.Context, root *dagger.Directory) (*AllResult, error) {
	tidyWorker := func(ctx context.Context, wg *sync.WaitGroup, input <-chan string, res chan<- *WorkerResult, root *dagger.Directory) {
		defer wg.Done()
		for dir := range input {
			select {
			case <-ctx.Done():
				return
			default:
			}

			c := m.goContainer(root, dir).
				WithExec([]string{"go", "mod", "tidy", "-go=1.23.0"})
			stdout, err := c.Stdout(ctx)
			if err != nil {
				res <- &WorkerResult{Module: dir, Source: c.Directory("/work/src"), Err: err}
				continue
			}

			stderr, err := c.Stderr(ctx)
			if err != nil {
				res <- &WorkerResult{Module: dir, Source: c.Directory("/work/src"), Err: err}
				continue
			}

			res <- &WorkerResult{Module: dir, Logs: stdout + stderr, Source: c.Directory("/work/src"), Err: err}
		}
	}

	return m.runAll(ctx, root, tidyWorker)
}

// Runs `go get -u -t ./...` in all modules starting with `root`
func (m *GoOrb) Update(ctx context.Context, root *dagger.Directory) (*AllResult, error) {
	updateWorker := func(ctx context.Context, wg *sync.WaitGroup, input <-chan string, res chan<- *WorkerResult, root *dagger.Directory) {
		defer wg.Done()
		for dir := range input {
			select {
			case <-ctx.Done():
				return
			default:
			}

			c := m.goContainer(root, dir).
				WithEnvVariable("GOPROXY", "direct").
				WithEnvVariable("GOSUMDB", "off").
				WithExec([]string{"go", "get", "-u", "-t", "./..."}).
				WithExec([]string{"go", "get", "-u", "github.com/go-orb/go-orb@main"}).
				WithExec([]string{"bash", "-c", "for m in $(grep github.com/go-orb/plugins go.mod | grep -E -v \"^module\" | awk '{ print $1 }'); do go get -u \"${m}@main\"; done"}).
				WithExec([]string{"go", "mod", "tidy", "-go=1.23.0"})

			stdout, err := c.Stdout(ctx)
			if err != nil {
				res <- &WorkerResult{Module: dir, Source: c.Directory("/work/src"), Err: err}
				continue
			}

			stderr, err := c.Stderr(ctx)
			if err != nil {
				res <- &WorkerResult{Module: dir, Source: c.Directory("/work/src"), Err: err}
				continue
			}

			res <- &WorkerResult{Module: dir, Logs: stdout + stderr, Source: c.Directory("/work/src"), Err: err}
		}
	}

	return m.runAll(ctx, root, updateWorker)
}
