package orb

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

//nolint:gochecknoglobals,lll
const templateStr = `
import (
	"context"
	{{ if or .ServerDRPC}}"fmt"{{ end }}

	"github.com/go-orb/go-orb/client"
	"github.com/go-orb/go-orb/log"
	"github.com/go-orb/go-orb/server"
	{{ if or .ServerDRPC}}
	 "google.golang.org/protobuf/proto"
	 "storj.io/drpc"
	 {{ end }}

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

// orbEncoding_{{.Type}}_proto is a protobuf encoder for the {{.Name}} service.
type orbEncoding_{{.Type}}_proto struct{}

// Marshal implements the drpc.Encoding interface.
func (orbEncoding_{{.Type}}_proto) Marshal(msg drpc.Message) ([]byte, error) {
	m, ok := msg.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("message is not a proto.Message: %T", msg)
	}
	return proto.Marshal(m)
}

// Unmarshal implements the drpc.Encoding interface.
func (orbEncoding_{{.Type}}_proto) Unmarshal(data []byte, msg drpc.Message) error {
	m, ok := msg.(proto.Message)
	if !ok {
		return fmt.Errorf("message is not a proto.Message: %T", msg)
	}
	return proto.Unmarshal(data, m)
}

// Name implements the drpc.Encoding interface.
func (orbEncoding_{{.Type}}_proto) Name() string {
	return "proto"
}

// {{.Type}}Client is the client for {{.Name}}
type {{.Type}}Client struct {
	client client.Client
}

// New{{.Type}}Client creates a new client for {{.Name}}
func New{{.Type}}Client(client client.Client) *{{.Type}}Client {
	return &{{.Type}}Client{client: client}
}

{{- $service := .}}{{ range .Methods }}
{{- if not .ClientStreaming }}
// {{.Name}} requests {{.Name}}.
func (c *{{$service.Type}}Client) {{.Name}}(ctx context.Context, service string, req *{{.Request}}, opts ...client.CallOption) (*{{.Reply}}, error) {
	return client.Request[{{.Reply}}](ctx, c.client, service, Endpoint{{$service.Type}}{{.Name}}, req, opts...)
}
{{- else if and .ClientStreaming (not .ServerStreaming) }}
// {{.Name}} creates a client-streaming connection to {{.Name}}.
func (c *{{$service.Type}}Client) {{.Name}}(ctx context.Context, service string, opts ...client.CallOption) (client.StreamIface[*{{.Request}}, *{{.Reply}}], error) {
	return client.Stream[*{{.Request}}, *{{.Reply}}](ctx, c.client, service, Endpoint{{$service.Type}}{{.Name}}, opts...)
}
{{- else if and (not .ClientStreaming) .ServerStreaming }}
// {{.Name}} creates a server-streaming connection to {{.Name}}.
func (c *{{$service.Type}}Client) {{.Name}}(ctx context.Context, service string, req *{{.Request}}, opts ...client.CallOption) (client.StreamIface[*{{.Request}}, *{{.Reply}}], error) {
	streamReq := &client.StreamReq[*{{.Request}}, *{{.Reply}}]{
		ReqBody: req,
	}
	return c.client.Stream(ctx, streamReq, append(opts, client.WithEndpoint(Endpoint{{$service.Type}}{{.Name}}), client.WithService(service))...)
}
{{- else }}
// {{.Name}} creates a bidirectional-streaming connection to {{.Name}}.
func (c *{{$service.Type}}Client) {{.Name}}(ctx context.Context, service string, opts ...client.CallOption) (client.StreamIface[*{{.Request}}, *{{.Reply}}], error) {
	return client.Stream[*{{.Request}}, *{{.Reply}}](ctx, c.client, service, Endpoint{{$service.Type}}{{.Name}}, opts...)
}
{{- end }}
{{- end }}

// {{.Type}}Handler is the Handler for {{.Name}}
type {{.Type}}Handler interface {
	{{- range .Methods }}
	{{- if not .ClientStreaming }}
	{{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error)
	{{- else if and .ClientStreaming (not .ServerStreaming) }}
	{{.Name}}(stream {{$service.Type}}{{.Name}}Stream) error
	{{- else if and (not .ClientStreaming) .ServerStreaming }}
	{{.Name}}(req *{{.Request}}, stream {{$service.Type}}{{.Name}}Stream) error
	{{- else }}
	{{.Name}}(stream {{$service.Type}}{{.Name}}Stream) error
	{{- end }}
	{{ end -}}
}

{{ $service := . }}
{{- range .Methods }}
{{- if or .ClientStreaming .ServerStreaming }}
// {{$service.Type}}{{.Name}}Stream defines the streaming interface for {{.Name}}
type {{$service.Type}}{{.Name}}Stream interface {
	{{- if and .ClientStreaming (not .ServerStreaming) }}
	Send(*{{.Reply}}) error
	Recv() (*{{.Request}}, error)
	{{- else if and (not .ClientStreaming) .ServerStreaming }}
	Send(*{{.Reply}}) error
	{{- else if and .ClientStreaming .ServerStreaming }}
	Send(*{{.Reply}}) error
	Recv() (*{{.Request}}, error)
	{{- end }}
	Context() context.Context
	Close() error
	CloseSend(*{{.Reply}}) error
}
{{- end }}
{{- end }}

{{- if $.ServerGRPC }}
// orbGRPC{{.Type}} provides the adapter to convert a {{.Type}}Handler to a gRPC {{.Type}}Server.
type orbGRPC{{.Type}} struct {
	handler {{.Type}}Handler
}

{{- $service := .}}{{ range .Methods }}
{{- if not .ClientStreaming }}
// {{.Name}} implements the {{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (s *orbGRPC{{$service.Type}}) {{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error) {
	return s.handler.{{.Name}}(ctx, req)
}
{{- else if and .ClientStreaming (not .ServerStreaming) }}
// {{.Name}} implements the {{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (s *orbGRPC{{$service.Type}}) {{.Name}}(stream grpc.ClientStreamingServer[{{.Request}}, {{.Reply}}]) error {
	// Adapt the gRPC stream to the ORB stream
	adapter := &orbGRPC{{.Name}}StreamAdapter{
		stream: stream,
	}
	return s.handler.{{.Name}}(adapter)
}
{{- else if and (not .ClientStreaming) .ServerStreaming }}
// {{.Name}} implements the {{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (s *orbGRPC{{$service.Type}}) {{.Name}}(req *{{.Request}}, stream grpc.ServerStreamingServer[{{.Reply}}]) error {
	// Adapt the gRPC stream to the ORB stream
	adapter := &orbGRPC{{.Name}}StreamAdapter{
		stream: stream,
	}
	return s.handler.{{.Name}}(req, adapter)
}
{{- else }}
// {{.Name}} implements the {{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (s *orbGRPC{{$service.Type}}) {{.Name}}(stream grpc.BidirectionalStreamingServer[{{.Request}}, {{.Reply}}]) error {
	// Adapt the gRPC stream to the ORB stream
	adapter := &orbGRPC{{.Name}}StreamAdapter{
		stream: stream,
	}
	return s.handler.{{.Name}}(adapter)
}
{{- end }}
{{- end }}

// Stream adapters to convert gRPC streams to ORB streams.
{{- $service := .}}{{ range .Methods }}
{{- if or .ClientStreaming .ServerStreaming }}

// orbGRPC{{.Name}}StreamAdapter adapts a gRPC stream to the ORB {{$service.Type}}{{.Name}}Stream interface.
type orbGRPC{{.Name}}StreamAdapter struct {
	{{- if and .ClientStreaming (not .ServerStreaming) }}
	stream grpc.ClientStreamingServer[{{.Request}}, {{.Reply}}]
	{{- else if and (not .ClientStreaming) .ServerStreaming }}
	stream grpc.ServerStreamingServer[{{.Reply}}]
	{{- else }}
	stream grpc.BidirectionalStreamingServer[{{.Request}}, {{.Reply}}]
	{{- end }}
}

{{- if .ClientStreaming }}
func (a *orbGRPC{{.Name}}StreamAdapter) Recv() (*{{.Request}}, error) {
	return a.stream.Recv()
}
{{- end }}

func (a *orbGRPC{{.Name}}StreamAdapter) Context() context.Context {
	return a.stream.Context()
}

func (a *orbGRPC{{.Name}}StreamAdapter) Close() error {
	// gRPC streams don't have a direct Close method, so we'll return nil.
	return nil
}

func (a *orbGRPC{{.Name}}StreamAdapter) Send(resp *{{.Reply}}) error {
	{{- if and .ClientStreaming (not .ServerStreaming) }}
	return a.stream.SendAndClose(resp)
	{{- else }}
	return a.stream.Send(resp)
	{{- end }}
}

func (a *orbGRPC{{.Name}}StreamAdapter) CloseSend(resp *{{.Reply}}) error {
	{{- if and .ClientStreaming (not .ServerStreaming) }}
	return a.stream.SendAndClose(resp)
	{{- else }}
	return a.stream.Send(resp)
	{{- end }}
}
{{- end }}
{{- end }}

// Verification that our adapters implement the required interfaces.
{{- $service := .}}{{ range .Methods }}
{{- if or .ClientStreaming .ServerStreaming }}
var _ {{$service.Type}}{{.Name}}Stream = (*orbGRPC{{.Name}}StreamAdapter)(nil)
{{- end }}
{{- end }}
var _ {{.Type}}Server = (*orbGRPC{{.Type}})(nil)

// register{{.Type}}GRPCServerHandler registers the service to a gRPC server.
func register{{.Type}}GRPCServerHandler(srv grpc.ServiceRegistrar, handler {{.Type}}Handler) {
	// Create the adapter to convert from {{.Type}}Handler to {{.Type}}Server
	grpcHandler := &orbGRPC{{.Type}}{handler: handler}
	
	srv.RegisterService(&{{.Type}}_ServiceDesc, grpcHandler)
}
{{ end }}

{{- if $.ServerDRPC }}
{{- $service := .}}{{ range .Methods }}
{{- if or .ClientStreaming .ServerStreaming }}

// orbDRPC{{.Name}}Client is the client API for the {{.Name}} method.
type orbDRPC{{.Name}}Client interface {
	{{- if .ClientStreaming }}
	// Send sends a message through the stream.
	Send(msg *{{.Request}}) error
	// CloseSend signals to the server that we're done sending messages.
	CloseSend() error
	{{- end }}
	{{- if or .ServerStreaming .ClientStreaming }}
	// Recv receives a message from the stream.
	Recv(m *{{.Reply}}) error
	{{- end }}
	// Context returns the stream's context.
	Context() context.Context
}

// orbDRPC{{.Name}}ClientStream implements the {{.Name}}Client interface.
type orbDRPC{{.Name}}ClientStream struct {
	stream DRPC{{$service.Type}}_{{.Name}}Stream
}

{{- if .ClientStreaming }}
func (s *orbDRPC{{.Name}}ClientStream) Send(msg *{{.Request}}) error {
	return s.stream.MsgSend(msg, orbEncoding_{{$service.Type}}_proto{})
}
{{- end }}

{{- if or .ServerStreaming .ClientStreaming }}
func (s *orbDRPC{{.Name}}ClientStream) Recv(m *{{.Reply}}) error {
	return s.stream.MsgRecv(m, orbEncoding_{{$service.Type}}_proto{})
}
{{- end }}

func (s *orbDRPC{{.Name}}ClientStream) Context() context.Context {
	return s.stream.Context()
}
{{- end }}
{{- end }}

// orbDRPC{{.Type}}Handler wraps a {{.Type}}Handler to implement DRPC{{.Type}}Server.
type orbDRPC{{.Type}}Handler struct {
	handler {{.Type}}Handler
}

{{- $service := .}}{{ range .Methods }}
{{- if not .ClientStreaming }}
// {{.Name}} implements the DRPC{{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (w *orbDRPC{{$service.Type}}Handler) {{.Name}}(ctx context.Context, req *{{.Request}}) (*{{.Reply}}, error) {
	return w.handler.{{.Name}}(ctx, req)
}
{{- else if and .ClientStreaming (not .ServerStreaming) }}
// {{.Name}} implements the DRPC{{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (w *orbDRPC{{$service.Type}}Handler) {{.Name}}(stream DRPC{{$service.Type}}_{{.Name}}Stream) error {
	// Adapt the DRPC stream to the ORB stream.
	adapter := &orbDRPC{{.Name}}StreamAdapter{
		stream: stream,
	}
	return w.handler.{{.Name}}(adapter)
}
{{- else if and (not .ClientStreaming) .ServerStreaming }}
// {{.Name}} implements the DRPC{{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (w *orbDRPC{{$service.Type}}Handler) {{.Name}}(req *{{.Request}}, stream DRPC{{$service.Type}}_{{.Name}}Stream) error {
	// Adapt the DRPC stream to the ORB stream.
	adapter := &orbDRPC{{.Name}}StreamAdapter{
		stream: stream,
	}
	return w.handler.{{.Name}}(req, adapter)
}
{{- else }}
// {{.Name}} implements the DRPC{{$service.Type}}Server interface by adapting to the {{$service.Type}}Handler.
func (w *orbDRPC{{$service.Type}}Handler) {{.Name}}(stream DRPC{{$service.Type}}_{{.Name}}Stream) error {
	// Adapt the DRPC stream to the ORB stream.
	adapter := &orbDRPC{{.Name}}StreamAdapter{
		stream: stream,
	}
	return w.handler.{{.Name}}(adapter)
}
{{- end }}
{{- end }}

// Stream adapters to convert DRPC streams to ORB streams.
{{- $service := .}}{{ range .Methods }}
{{- if or .ClientStreaming .ServerStreaming }}

// orbDRPC{{.Name}}StreamAdapter adapts a DRPC stream to the ORB {{$service.Type}}{{.Name}}Stream interface.
type orbDRPC{{.Name}}StreamAdapter struct {
	stream DRPC{{$service.Type}}_{{.Name}}Stream
}

{{- if .ClientStreaming }}
func (a *orbDRPC{{.Name}}StreamAdapter) Recv() (*{{.Request}}, error) {
	return a.stream.Recv()
}
{{- end }}

func (a *orbDRPC{{.Name}}StreamAdapter) Context() context.Context {
	return a.stream.Context()
}

func (a *orbDRPC{{.Name}}StreamAdapter) Close() error {
	return a.stream.CloseSend()
}

func (a *orbDRPC{{.Name}}StreamAdapter) Send(resp *{{.Reply}}) error {
	return a.stream.SendAndClose(resp)
}

func (a *orbDRPC{{.Name}}StreamAdapter) CloseSend(resp *{{.Reply}}) error {
	return a.stream.SendAndClose(resp)
}
{{- end }}
{{- end }}

// Verification that our adapters implement the required interfaces.
{{- $service := .}}{{ range .Methods }}
{{- if or .ClientStreaming .ServerStreaming }}
var _ {{$service.Type}}{{.Name}}Stream = (*orbDRPC{{.Name}}StreamAdapter)(nil)
{{- end }}
{{- end }}
var _ DRPC{{.Type}}Server = (*orbDRPC{{.Type}}Handler)(nil)

// register{{.Type}}DRPCHandler registers the service to an dRPC server.
func register{{.Type}}DRPCHandler(srv *mdrpc.Server, handler {{.Type}}Handler) error {
	desc := DRPC{{.Type}}Description{}

	// Wrap the ORB handler with our adapter to make it compatible with DRPC.
	drpcHandler := &orbDRPC{{.Type}}Handler{handler: handler}

	// Register with the server/drpc(.Mux).
	err := srv.Router().Register(drpcHandler, desc)
	if err != nil {
		return err
	}

	// Add each endpoint name of this handler to the orb drpc server.
	{{- range .Methods}}
	srv.AddEndpoint("{{.Path}}")
	{{- end }}

	return nil
}

// register{{.Type}}MemoryHandler registers the service to a memory server.
func register{{.Type}}MemoryHandler(srv *memory.Server, handler {{.Type}}Handler) error {
	desc := DRPC{{.Type}}Description{}

	// Wrap the ORB handler with our adapter to make it compatible with DRPC.
	drpcHandler := &orbDRPC{{.Type}}Handler{handler: handler}

	// Register with the server/drpc(.Mux).
	err := srv.Router().Register(drpcHandler, desc)
	if err != nil {
		return err
	}

	// Add each endpoint name of this handler to the orb drpc server.
	{{- range .Methods}}
	srv.AddEndpoint("{{.Path}}")
	{{- end }}

	return nil
}

{{ end -}}

{{- if $.ServerHTTP }}

// register{{.Type}}HTTPHandler registers the service to an HTTP server.
func register{{.Type}}HTTPHandler(srv *mhttp.Server, handler {{.Type}}Handler) {
	{{- range .Methods}}
	{{- if not (or .ClientStreaming .ServerStreaming) }}
	srv.Router().{{.Method}}("{{.Path}}", mhttp.NewGRPCHandler(srv, handler.{{.Name}}, Handler{{$service.Type}}, "{{.Name}}"))
	{{- else }}
	// HTTP transport does not support streaming for {{.Name}}
	log.Warn("Streaming endpoint not registered with HTTP transport", "endpoint", "{{.Path}}")
	{{- end }}
	{{- end }}
}

{{ end -}}

{{- if $.ServerHertz }}

// register{{.Type}}HertzHandler registers the service to an Hertz server.
func register{{.Type}}HertzHandler(srv *mhertz.Server, handler {{.Type}}Handler) {
	r := srv.Router()
	{{- range .Methods}}
	{{- if not (or .ClientStreaming .ServerStreaming) }}
	r.{{.MethodUpper}}("{{.Path}}", mhertz.NewGRPCHandler(srv, handler.{{.Name}}, Handler{{$service.Type}}, "{{.Name}}"))
	{{- else }}
	// Hertz transport does not support streaming for {{.Name}}
	log.Warn("Streaming endpoint not registered with Hertz transport", "endpoint", "{{.Path}}")
	{{- end }}
	{{- end }}
}
{{- end }}

// Register{{.Type}}Handler will return a registration function that can be
// provided to entrypoints as a handler registration.
func Register{{.Type}}Handler(handler any) server.RegistrationFunc {
	return func(s any) {
		switch srv := s.(type) {
		{{ if $.ServerGRPC }}
		case grpc.ServiceRegistrar:
			register{{.Type}}GRPCServerHandler(srv, handler.({{.Type}}Handler))
		{{- end -}}
		{{ if $.ServerDRPC }}
		case *mdrpc.Server:
			register{{.Type}}DRPCHandler(srv, handler.({{.Type}}Handler))
		case *memory.Server:
			register{{.Type}}MemoryHandler(srv, handler.({{.Type}}Handler))
		{{- end -}}
		{{ if $.ServerHertz }}
		case *mhertz.Server:
			register{{.Type}}HertzHandler(srv, handler.({{.Type}}Handler))
		{{- end -}}
		{{ if $.ServerHTTP }}
		case *mhttp.Server:
			register{{.Type}}HTTPHandler(srv, handler.({{.Type}}Handler))
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
	Services    []serviceDesc
	PackageName string
	SourceFile  string
}

func (f protoFile) Render() string {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"Title":   strings.Title,
	}

	tmpl, err := template.New("orb").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		panic(err)
	}

	buf := bytes.Buffer{}
	if err := tmpl.Execute(&buf, f); err != nil {
		panic(err)
	}

	return buf.String()
}

// serviceDesc describes a service.
type serviceDesc struct {
	Type      string
	TypeLower string
	Name      string
	Metadata  string
	Methods   []methodDesc
}

// methodDesc describes a service method.
type methodDesc struct {
	Name            string
	OriginalName    string
	Num             int
	Request         string
	Reply           string
	ClientStreaming bool
	ServerStreaming bool
	Path            string
	Method          string
	MethodUpper     string
}

func (m *methodDesc) String() string {
	return fmt.Sprintf("%+v", *m)
}

func (s *serviceDesc) AddMethod(method methodDesc) {
	s.Methods = append(s.Methods, method)
}
