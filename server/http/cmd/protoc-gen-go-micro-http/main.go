package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	omitempty := flag.Bool("omitempty", true, "omit if google.api is empty")
	flag.Parse()

	if *showVersion {
		_, err := fmt.Fprintf(os.Stdout, "protoc-gen-go-micro-http %v\n", release)
		if err != nil {
			log.Fatal(err)
		}

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
