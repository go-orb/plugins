package main

import (
	"errors"
	"runtime"

	"github.com/go-orb/go-orb/config/source/cli"
)

const (
	configSection = "bench_client"

	defaultBypassRegistry = 1
	defaultConnections    = 256
	defaultDuration       = 15
	defaultTimeout        = 8
	defaultTransport      = "grpc"
	defaultPackageSize    = 1000
	defaultContentType    = "application/x-protobuf"
)

//nolint:gochecknoglobals
var (
	defaultThreads = runtime.NumCPU()
)

func init() {
	err := cli.Flags.Add(cli.NewFlag(
		"bypass_registry",
		defaultBypassRegistry,
		cli.ConfigPathSlice([]string{configSection, "bypassRegistry"}),
		cli.Usage("Bypasses the registry by caching it, set to 0 to disable"),
		cli.EnvVars("BYPASS_REGISTRY"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = cli.Flags.Add(cli.NewFlag(
		"connections",
		defaultConnections,
		cli.ConfigPathSlice([]string{configSection, "connections"}),
		cli.Usage("Connections to keep open"),
		cli.EnvVars("CONNECTIONS"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = cli.Flags.Add(cli.NewFlag(
		"duration",
		defaultDuration,
		cli.ConfigPathSlice([]string{configSection, "duration"}),
		cli.Usage("Duration in seconds"),
		cli.EnvVars("DURATION"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = cli.Flags.Add(cli.NewFlag(
		"timeout",
		defaultTimeout,
		cli.ConfigPathSlice([]string{configSection, "timeout"}),
		cli.Usage("Timeout in seconds"),
		cli.EnvVars("TIMEOUT"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	// function init is to long.....
	init2()
}

func init2() {
	err := cli.Flags.Add(cli.NewFlag(
		"threads",
		defaultThreads,
		cli.ConfigPathSlice([]string{configSection, "threads"}),
		cli.Usage("Number of threads to use"),
		cli.EnvVars("THREADS"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = cli.Flags.Add(cli.NewFlag(
		"transport",
		defaultTransport,
		cli.ConfigPathSlice([]string{configSection, "transport"}),
		cli.Usage("Transport to use (grpc, hertzhttp, http, uvm.)"),
		cli.EnvVars("TRANSPORT"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = cli.Flags.Add(cli.NewFlag(
		"package_size",
		defaultPackageSize,
		cli.ConfigPathSlice([]string{configSection, "packageSize"}),
		cli.Usage("Per request package size"),
		cli.EnvVars("PACKAGE_SIZE"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}

	err = cli.Flags.Add(cli.NewFlag(
		"content_type",
		defaultContentType,
		cli.ConfigPathSlice([]string{configSection, "contentType"}),
		cli.Usage("Content-Type (application/x-protobuf, application/json)"),
		cli.EnvVars("CONTENT_TYPE"),
	))
	if err != nil && !errors.Is(err, cli.ErrFlagExists) {
		panic(err)
	}
}

type clientConfig struct {
	BypassRegistry int    `json:"bypassRegistry" yaml:"bypassRegistry"`
	Connections    int    `json:"connections" yaml:"connections"`
	Duration       int    `json:"duration" yaml:"duration"`
	Timeout        int    `json:"timeout" yaml:"timeout"`
	Threads        int    `json:"threads" yaml:"threads"`
	Transport      string `json:"transport" yaml:"transport"`
	PackageSize    int    `json:"packageSize" yaml:"packageSize"`
	ContentType    string `json:"contentType" yaml:"contentType"`
}
