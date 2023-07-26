module github.com/go-orb/plugins/server/http

go 1.20

require (
	github.com/go-chi/chi v1.5.4
	github.com/go-orb/go-orb v0.0.0-20230725195416-0c865f140784
	github.com/go-orb/plugins/codecs/form v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/codecs/proto v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/config/source/file v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/log/slog v0.0.0-20230726151712-9ff06382f83b
	github.com/go-orb/plugins/registry/mdns v0.0.0-20230726151712-9ff06382f83b
	github.com/google/uuid v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/quic-go/quic-go v0.37.0
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230725093048-515e97ebf090
	golang.org/x/net v0.12.0
	google.golang.org/genproto/googleapis/api v0.0.0-20230724170836-66ad5b6ff146
	google.golang.org/grpc v1.56.2
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-playground/form/v4 v4.2.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20230705174524-200ffdc848b8 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/onsi/ginkgo/v2 v2.11.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.3.0 // indirect
	golang.org/x/crypto v0.11.0 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	google.golang.org/genproto v0.0.0-20230724170836-66ad5b6ff146 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230724170836-66ad5b6ff146 // indirect
)

replace github.com/go-orb/plugins/codecs/form => ../../codecs/form

replace github.com/go-orb/plugins/codecs/jsonpb => ../../codecs/jsonpb

replace github.com/go-orb/plugins/codecs/proto => ../../codecs/proto

replace github.com/go-orb/plugins/codecs/yaml => ../../codecs/yaml

replace github.com/go-orb/plugins/config/source/file => ../../config/source/file
