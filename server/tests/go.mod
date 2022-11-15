module github.com/go-micro/plugins/servers/tests

go 1.19

require (
	github.com/go-micro/plugins/codecs/form v0.0.0-00010101000000-000000000000
	github.com/go-micro/plugins/codecs/jsonpb v0.0.0-00010101000000-000000000000
	github.com/go-micro/plugins/codecs/proto v0.0.0-00010101000000-000000000000
	github.com/go-micro/plugins/log/text v0.0.0-00010101000000-000000000000
	github.com/go-micro/plugins/server/http v0.0.0-00010101000000-000000000000
	github.com/go-micro/plugins/server/tests v0.0.0-00010101000000-000000000000
	github.com/lucas-clemente/quic-go v0.30.1-0.20221107095222-2de4af00d068
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.8.1
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	golang.org/x/net v0.2.0
	google.golang.org/genproto v0.0.0-20221111202108-142d8a6fa32e
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-chi/chi v1.5.4 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/pprof v0.0.0-20221103000818-d260c55eee4c // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.13.0 // indirect
	github.com/marten-seemann/qpack v0.3.0 // indirect
	github.com/marten-seemann/qtls-go1-18 v0.1.3 // indirect
	github.com/marten-seemann/qtls-go1-19 v0.1.1 // indirect
	github.com/onsi/ginkgo/v2 v2.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.1.0 // indirect
	golang.org/x/exp v0.0.0-20221109205753-fc8884afc316 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/tools v0.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-micro/plugins/server/tests => ./

replace github.com/go-micro/plugins/server/http => ../http

replace github.com/go-micro/plugins/codecs/proto => ../../codecs/proto

replace github.com/go-micro/plugins/codecs/jsonpb => ../../codecs/jsonpb

replace github.com/go-micro/plugins/codecs/yaml => ../../codecs/yaml

replace github.com/go-micro/plugins/codecs/form => ../../codecs/form

replace github.com/go-micro/plugins/log/text => ../../log/text

replace github.com/go-micro/plugins/log/json => ../../log/json

replace go-micro.dev/v5 => ../../../go-micro/
