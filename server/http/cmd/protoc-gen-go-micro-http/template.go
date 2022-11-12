package main

import (
	"bytes"
	"strings"
	"text/template"
)

//nolint:gochecknoglobals
var httpTemplate = `
{{$svrType := .ServiceType}}
{{$svrName := .ServiceName}}
import (
	"google.golang.org/grpc"

	"go-micro.dev/v5/server"

	mhttp "github.com/go-micro/plugins/server/http"
)

func Register{{.ServiceType}}HTTPHandler(srv *mhttp.ServerHTTP, handler {{.ServiceType}}Server ) {
	r := srv.Router()
	{{- range .Methods}}
	r.{{.Method}}("{{.Path}}", mhttp.NewGRPCHandler(srv, handler.{{.Name}}))
	{{- end}}
}

func RegisterFunc{{.ServiceType}}(handler {{.ServiceType}}Server) server.RegistrationFunc {
	return server.RegistrationFunc(func(s any) {
		switch srv := any(s).(type) {
		case *mhttp.ServerHTTP:
			Register{{.ServiceType}}HTTPHandler(srv, handler)
		case grpc.ServiceRegistrar:
			// RegisterStreamsgRPCHandler(srv, handler)
		default:
			// Maybe we should log here with slog global logger
		}
	})
}
`

type serviceDesc struct {
	ServiceType string // Greeter
	ServiceName string // helloworld.Greeter
	Metadata    string // api/helloworld/helloworld.proto
	Methods     []*methodDesc
	MethodSets  map[string]*methodDesc
}

type methodDesc struct {
	// method
	Name         string
	OriginalName string // The parsed original name
	Num          int
	Request      string
	Reply        string
	// http_rule
	Path         string
	Method       string
	HasVars      bool
	HasBody      bool
	Body         string
	ResponseBody string
}

func (s *serviceDesc) execute() string {
	s.MethodSets = make(map[string]*methodDesc)
	for _, m := range s.Methods {
		s.MethodSets[m.Name] = m
	}

	buf := new(bytes.Buffer)

	tmpl, err := template.New("http").Parse(strings.TrimSpace(httpTemplate))
	if err != nil {
		panic(err)
	}

	if err := tmpl.Execute(buf, s); err != nil {
		panic(err)
	}

	return strings.Trim(buf.String(), "\r\n")
}
