package orb

import (
	"bytes"
	"strings"
	"text/template"
)

//nolint:gochecknoglobals,lll
var orbTemplate = `
import (
	"context"

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"

	{{ if .ServerGRPC }}grpc "google.golang.org/grpc"{{ end }}

	{{ if .ServerDRPC }}mdrpc "github.com/go-orb/plugins/server/drpc"{{ end }}
	{{ if .ServerDRPC }}memory "github.com/go-orb/plugins/server/memory"{{ end }}
	{{ if .ServerHertz }}mhertz "github.com/go-orb/plugins/server/hertz"{{ end }}
	{{ if .ServerHTTP }}mhttp "github.com/go-orb/plugins/server/http"{{ end }}
)

{{- range .Services }}
// Handler{{.Type}} is the name of a service, it's here to static type/reference.
const Handler{{.Type}} = "{{.Name}}"
{{- $service := .}}{{ range .Methods }}
const Endpoint{{$service.Type}}{{.Name}} = "/{{$service.Name}}/{{.Name}}"
{{- end }}
{{- end }}

{{- range .Services }}
// {{.Type}}Client is the client for {{.Name}}
type {{.Type}}Client struct {
	client client.Client
}

// New{{.Type}}Client creates a new client for {{.Name}}
func New{{.Type}}Client(client client.Client) *{{.Type}}Client {
	return &{{.Type}}Client{client: client}
}

{{- $service := .}}{{ range .Methods }}
// {{.Name}} requests {{.Name}}.
func (c *{{$service.Type}}Client) {{.Name}}(ctx context.Context, service string, req *{{.Request}}, opts ...client.CallOption) (*{{.Reply}}, error) {
	return client.Request[{{.Reply}}](ctx, c.client, service, Endpoint{{$service.Type}}{{.Name}}, req, opts...)
}
{{- end }}

// {{.Type}}Handler is the Handler for {{.Name}}
type {{.Type}}Handler interface {
	{{- range .Methods }}
	{{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error)
	{{ end -}}
}

{{- if $.ServerDRPC }}

// register{{.Type}}DRPCHandler registers the service to an dRPC server.
func register{{.Type}}DRPCHandler(srv *mdrpc.Server, handler {{.Type}}Handler) error {
	desc := DRPC{{.Type}}Description{}

	// Register with the server/drpc(.Mux).
	err := srv.Router().Register(handler, desc)
	if err != nil {
		return err
	}

	// Add each endpoint name of this handler to the orb drpc server.
	{{- $service := .}}{{ range .Methods}}
	srv.AddEndpoint("{{.Path}}")
	{{- end }}

	return nil
}

// register{{.Type}}MemoryHandler registers the service to an dRPC server.
func register{{.Type}}MemoryHandler(srv *memory.Server, handler {{.Type}}Handler) error {
	desc := DRPC{{.Type}}Description{}

	// Register with the server/drpc(.Mux).
	err := srv.Router().Register(handler, desc)
	if err != nil {
		return err
	}

	// Add each endpoint name of this handler to the orb drpc server.
	{{- $service := .}}{{ range .Methods}}
	srv.AddEndpoint("{{.Path}}")
	{{- end }}

	return nil
}

{{ end -}}

{{- if $.ServerHTTP }}

// register{{.Type}}HTTPHandler registers the service to an HTTP server.
func register{{.Type}}HTTPHandler(srv *mhttp.Server, handler {{.Type}}Handler) {
	r := srv.Router()
	{{- $service := .}}{{range .Methods}}
	r.{{.Method}}("{{.Path}}", mhttp.NewGRPCHandler(srv, handler.{{.Name}}, Handler{{$service.Type}}, "{{.Name}}"))
	{{- end}}
}

{{ end -}}

{{- if $.ServerHertz }}

// register{{.Type}}HertzHandler registers the service to an Hertz server.
func register{{.Type}}HertzHandler(srv *mhertz.Server, handler {{.Type}}Handler) {
	r := srv.Router()
	{{- $service := .}}{{ range .Methods}}
	r.{{.MethodUpper}}("{{.Path}}", mhertz.NewGRPCHandler(srv, handler.{{.Name}}, Handler{{$service.Type}}, "{{.Name}}"))
	{{- end}}
}
{{- end }}

// Register{{.Type}}Handler will return a registration function that can be
// provided to entrypoints as a handler registration.
func Register{{.Type}}Handler(handler {{.Type}}Handler) server.RegistrationFunc {
	return func(s any) {
		switch srv := s.(type) {
		{{ if $.ServerGRPC }}
		case grpc.ServiceRegistrar:
			register{{.Type}}GRPCHandler(srv, handler)
		{{- end -}}
		{{ if $.ServerDRPC }}
		case *mdrpc.Server:
			register{{.Type}}DRPCHandler(srv, handler)
		case *memory.Server:
			register{{.Type}}MemoryHandler(srv, handler)
		{{- end -}}
		{{ if $.ServerHertz }}
		case *mhertz.Server:
			register{{.Type}}HertzHandler(srv, handler)
		{{- end -}}
		{{ if $.ServerHTTP }}
		case *mhttp.Server:
			register{{.Type}}HTTPHandler(srv, handler)
		{{- end }}
		default:
			log.Warn("No provider for this server found", "proto", "{{.Metadata}}", "handler", "{{.Type}}", "server", s)
		}
	}
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
	tmpl, err := template.New("orb").Parse(strings.TrimSpace(orbTemplate))
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
