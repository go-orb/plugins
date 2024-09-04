module github.com/go-orb/plugins/server/http

go 1.23

toolchain go1.23.0

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-orb/go-orb v0.0.0-20240902034447-508ce5c54a46
	github.com/go-orb/plugins/codecs/form v0.0.0-20240902051655-0791c4c590b6
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20240902051655-0791c4c590b6
	github.com/go-orb/plugins/codecs/proto v0.0.0-20240902051655-0791c4c590b6
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20240902051655-0791c4c590b6
	github.com/go-orb/plugins/config/source/file v0.0.0-20240902051655-0791c4c590b6
	github.com/go-orb/plugins/log/slog v0.0.0-20240902051655-0791c4c590b6
	github.com/go-orb/plugins/registry/mdns v0.0.0-20240902051655-0791c4c590b6
	github.com/google/uuid v1.6.0
	github.com/pkg/errors v0.9.1
	github.com/quic-go/quic-go v0.46.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/net v0.28.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240903143218-8af14fe29dc1
	google.golang.org/grpc v1.66.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/pprof v0.0.0-20240903155634-a8630aee4ab9 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/miekg/dns v1.1.62 // indirect
	github.com/onsi/ginkgo/v2 v2.20.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.5.0 // indirect
	go.uber.org/mock v0.4.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948 // indirect
	golang.org/x/mod v0.20.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.24.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
)

// Fixing ambiguous import: found package google.golang.org/genproto/googleapis/api/annotations in multiple modules.
exclude google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
