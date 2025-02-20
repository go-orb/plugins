// protoc-gen-go-orb is a plugin for the Google protocol buffer compiler to
// generate Go code. Install it by building this program and making it
// accessible within your PATH with the name:
//
//	protoc-gen-go-orb
package main

import (
	"flag"
	"fmt"
	"slices"
	"strings"

	gengo "google.golang.org/protobuf/cmd/protoc-gen-go/internal_gengo"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/go-orb/plugins/server/cmd/protoc-gen-go-orb/orb"
	"github.com/go-orb/plugins/server/cmd/protoc-gen-go-orb/orbdrpc"
	"github.com/go-orb/plugins/server/cmd/protoc-gen-go-orb/orbgrpc"
)

// Blabala...
//
//nolint:gochecknoglobals
var (
	// That version will be shown in each _orb.pb.go.
	Version              = "0.0.1"
	requireUnimplemented *bool
	useGenericStreams    *bool
	servers              *string
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("protoc-gen-go-orb %v\n", Version) //nolint:forbidigo
		return
	}

	var drpcConf orbdrpc.Config

	var flags flag.FlagSet
	requireUnimplemented = flags.Bool("require_unimplemented_servers", false, "set to false to match legacy behavior (gRPC)")
	//nolint:lll
	useGenericStreams = flags.Bool(
		"use_generic_streams_experimental",
		true,
		"set to true to use generic types for streaming client and server objects; this flag is EXPERIMENTAL and may be changed or removed in a future release (gRPC)",
	)
	servers = flags.String("supported_servers", "drpc;grpc;http", "semicolon separated list of servers to generate for")

	flags.StringVar(&drpcConf.Protolib, "protolib", "google.golang.org/protobuf", "which protobuf library to use for encoding")
	flags.BoolVar(&drpcConf.JSON, "json", true, "generate encoders with json support")

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(
			pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL) |
			uint64(pluginpb.CodeGeneratorResponse_FEATURE_SUPPORTS_EDITIONS)
		gen.SupportedEditionsMinimum = descriptorpb.Edition_EDITION_PROTO2
		gen.SupportedEditionsMaximum = descriptorpb.Edition_EDITION_2023

		sSplit := strings.Split(*servers, ";")
		for i := range sSplit {
			sSplit[i] = strings.Trim(strings.ToLower(sSplit[i]), " ")
		}

		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}

			// protoc-gen-go
			gengo.GenerateFile(gen, f)

			// dRPC
			if slices.Contains(sSplit, "drpc") {
				orbdrpc.Version = Version
				orbdrpc.GenerateFile(gen, f, drpcConf)
			}

			// gRPC
			if slices.Contains(sSplit, "grpc") {
				orbgrpc.Version = Version
				orbgrpc.RequireUnimplemented = requireUnimplemented
				orbgrpc.UseGenericStreams = useGenericStreams
				orbgrpc.GenerateFile(gen, f)
			}

			// ORB
			orb.Version = Version
			orb.Servers = sSplit
			orb.GenerateFile(gen, f)
		}

		return nil
	})
}
