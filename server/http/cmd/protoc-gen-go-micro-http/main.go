package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	omitempty := flag.Bool("omitempty", true, "omit if google.api is empty")
	flag.Parse()

	if *showVersion {
		fmt.Fprintf(os.Stdout, "protoc-gen-go-http %v\n", release)
		return
	}

	opts := protogen.Options{ParamFunc: flag.CommandLine.Set}
	opts.Run(func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			generateFile(gen, f, *omitempty)
		}

		return nil
	})
}
