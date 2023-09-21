module github.com/go-orb/plugins/client/orb/transport/http3

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230805173903-ba3da7c24b9d
	github.com/go-orb/plugins/client/orb v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/client/orb/transport/basehttp v0.0.0-20230725003158-93bc3eff9bfa
	github.com/go-orb/plugins/client/tests v0.0.0-20230713091520-67e7b5a34489
	github.com/go-orb/plugins/codecs/jsonpb v0.0.0-20230805175939-5c62af9f797c
	github.com/go-orb/plugins/codecs/proto v0.0.0
	github.com/go-orb/plugins/codecs/yaml v0.0.0-20230805175939-5c62af9f797c
	github.com/go-orb/plugins/config/source/file v0.0.0-20230805175939-5c62af9f797c
	github.com/go-orb/plugins/log/slog v0.0.0-20230805175939-5c62af9f797c
	github.com/go-orb/plugins/registry/mdns v0.0.0-20230805175939-5c62af9f797c
	github.com/go-orb/plugins/server/http v0.0.0-20230805175939-5c62af9f797c
	github.com/quic-go/quic-go v0.38.1
	github.com/stretchr/testify v1.8.4
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/go-chi/chi v1.5.5 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/pprof v0.0.0-20230907193218-d3ddc7976beb // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.18.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/onsi/ginkgo/v2 v2.12.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/quic-go/qpack v0.4.0 // indirect
	github.com/quic-go/qtls-go1-20 v0.3.4 // indirect
	golang.org/x/crypto v0.13.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/net v0.15.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	google.golang.org/genproto v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/grpc v1.58.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/plugins/client/orb => ../..

replace github.com/go-orb/plugins/client/orb/transports/basehttp => ../basehttp

replace github.com/go-orb/plugins/client/tests => ../../../tests

replace github.com/go-orb/plugins/codecs/jsonpb => ../../../../codecs/jsonpb

replace github.com/go-orb/plugins/codecs/yaml => ../../../../codecs/yaml

replace github.com/go-orb/plugins/codecs/proto => ../../../../codecs/proto

replace github.com/go-orb/plugins/config/source/file => ../../../../config/source/file

replace github.com/go-orb/plugins/log/slog => ../../../../log/slog

replace github.com/go-orb/plugins/registry/mdns => ../../../../registry/mdns

replace github.com/go-orb/plugins/server/http => ../../../../server/http

replace github.com/go-orb/plugins/client/orb/transport/http => ../http

replace github.com/go-orb/go-orb => ../../../../../go-orb

replace github.com/go-orb/plugins/client/orb/transport/basehttp => ../basehttp
