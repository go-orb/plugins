package orb

import (
	"bytes"
	"strings"
	"text/template"
)

//nolint:gochecknoglobals
var orbTemplate = `
import (
	"context"

	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"

	{{ if .ServerGRPC }}grpc "google.golang.org/grpc"{{ end }}

	{{ if .ServerDRPC }}mdrpc "github.com/go-orb/plugins/server/drpc"{{ end }}
	{{ if .ServerHertz }}mhertz "github.com/go-orb/plugins/server/hertz"{{ end }}
	{{ if .ServerHTTP }}mhttp "github.com/go-orb/plugins/server/http"{{ end }}
)

{{- range .Services }}
type {{.Type}}Handler interface {
	{{.Type}}(ctx context.Context, req *Req) (*Resp, error)
}

{{- if $.ServerDRPC }}

func register{{.Type}}DRPCHandler(srv *mdrpc.Server, handler {{.Type}}Handler) error {
	desc := DRPCEchoDescription{}

	// Register with DRPC.
	r := srv.Router()

	// Register with the drpcmux.
	err := r.Register(handler, desc)
	if err != nil {
		return err
	}

	// Add each endpoint name of this handler to the orb drpc server.
	for i := 0; i < desc.NumMethods(); i++ {
		name, _, _, _, _ := desc.Method(i)
		srv.AddEndpoint(name)
	}

	return nil
}

{{ end -}}

{{- if $.ServerHTTP }}

// register{{.Type}}HTTPHandler registers the service to an HTTP server.
func register{{.Type}}HTTPHandler(srv *mhttp.ServerHTTP, handler {{.Type}}Handler) {
	r := srv.Router()
	{{- range .Methods}}
	r.{{.Method}}("{{.Path}}", mhttp.NewGRPCHandler(srv, handler.{{.Name}}))
	{{- end}}
}

{{ end -}}

{{- if $.ServerHertz }}

// register{{.Type}}HertzHandler registers the service to an Hertz server.
func register{{.Type}}HertzHandler(srv *mhertz.Server, handler {{.Type}}Handler) {
	r := srv.Router()
	{{- range .Methods}}
	r.{{.MethodUpper}}("{{.Path}}", mhertz.NewGRPCHandler(srv, handler.{{.Name}}))
	{{- end}}
}

{{ end -}}

// Register{{.Type}}Service will return a registration function that can be
// provided to entrypoints as a handler registration.
func Register{{.Type}}Service(handler {{.Type}}Handler) server.RegistrationFunc {
	return server.RegistrationFunc(func(s any) {
		switch srv := s.(type) {
		{{ if $.ServerGRPC }}
		case grpc.ServiceRegistrar:
			register{{.Type}}GRPCHandler(srv, handler)
		{{- end -}}
		{{ if $.ServerDRPC }}
		case *mdrpc.Server:
			register{{.Type}}DRPCHandler(srv, handler)
		{{- end -}}
		{{ if $.ServerHertz }}
		case *mhertz.Server:
			register{{.Type}}HertzHandler(srv, handler)
		{{- end -}}
		{{ if $.ServerHTTP }}
		case *mhttp.ServerHTTP:
			register{{.Type}}HTTPHandler(srv, handler)
		{{- end }}
		default:
			log.Warn("No provider for this server found", "proto", "{{.Metadata}}", "handler", "{{.Type}}", "server", s)
		}
	})
}

{{- end }}
`

// protoFile is the object passed to the template.
type protoFile struct {
	ServerGRPC  bool
	ServerDRPC  bool
	ServerHTTP  bool
	ServerHertz bool

	Services []serviceDesc
}

func (p protoFile) Render() string {
	tmpl, err := template.New("http").Parse(strings.TrimSpace(orbTemplate))
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, p); err != nil {
		panic(err)
	}

	return strings.Trim(buf.String(), "\r\n")
}

// serviceDesc describes a service.
type serviceDesc struct {
	Type      string // Greeter
	TypeLower string // greeter
	Name      string // helloworld.Greeter
	Metadata  string // api/helloworld/helloworld.proto
	Methods   []methodDesc
}

// methodDesc describes a service method.
type methodDesc struct {
	Name         string
	OriginalName string // The parsed original name
	Num          int
	Request      string
	Reply        string

	// http_rule annotation properties.
	Path        string
	Method      string
	MethodUpper string
}

func (s *serviceDesc) AddMethod(method methodDesc) {
	s.Methods = append(s.Methods, method)
}
