package main

import (
	"bytes"
	"strings"
	"text/template"
)

//nolint:gochecknoglobals
var httpTemplate = `
import (
	"google.golang.org/grpc"

	"go-micro.dev/v5/server"

	mhttp "github.com/go-micro/plugins/server/http"
)
{{range .Services}}

// Register{{.Type}}HTTPHandler registers the service to an HTTP server.
func Register{{.Type}}HTTPHandler(srv *mhttp.ServerHTTP, handler {{.Type}}Server ) {
	r := srv.Router()
	{{- range .Methods}}
	r.{{.Method}}("{{.Path}}", mhttp.NewGRPCHandler(srv, handler.{{.Name}}))
	{{- end}}
}

// Register{{.Type}}Handler will return a registration function that can be 
// provided to entrypoints as a handler registration.
func Register{{.Type}}Handler(handler {{.Type}}Server) server.RegistrationFunc {
	return server.RegistrationFunc(func(s any) {
		switch srv := any(s).(type) {
		case *mhttp.ServerHTTP:
			Register{{.Type}}HTTPHandler(srv, handler)
		case grpc.ServiceRegistrar:
			// RegisterStreamsgRPCHandler(srv, handler)
		default:
			// Maybe we should log here with slog global logger
		}
	})
}
{{- end}}
`

// protoFile is the object passed to the template.
type protoFile struct {
	Services []serviceDesc
}

// serviceDesc describes a service.
type serviceDesc struct {
	Type     string // Greeter
	Name     string // helloworld.Greeter
	Metadata string // api/helloworld/helloworld.proto
	Methods  []methodDesc
}

// methodDesc describes a service method.
type methodDesc struct {
	Name         string
	OriginalName string // The parsed original name
	Num          int
	Request      string
	Reply        string

	// http_rule annotation properties.
	Path         string
	Method       string
	HasVars      bool
	HasBody      bool
	Body         string
	ResponseBody string
}

func (s *serviceDesc) AddMethod(method methodDesc) {
	s.Methods = append(s.Methods, method)
}

func (p protoFile) Render() string {
	tmpl, err := template.New("http").Parse(strings.TrimSpace(httpTemplate))
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, p); err != nil {
		panic(err)
	}

	return strings.Trim(buf.String(), "\r\n")
}
