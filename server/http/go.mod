module github.com/go-micro/plugins/server/http

go 1.19

require (
	github.com/go-chi/chi v1.5.4
	github.com/google/uuid v1.3.0
	github.com/lucas-clemente/quic-go v0.30.1-0.20221107095222-2de4af00d068
	github.com/stretchr/testify v1.8.1
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	golang.org/x/exp v0.0.0-20221111204811-129d8d6c17ab
	golang.org/x/net v0.2.0
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/google/pprof v0.0.0-20221112000123-84eb7ad69597 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/marten-seemann/qpack v0.3.0 // indirect
	github.com/marten-seemann/qtls-go1-18 v0.1.3 // indirect
	github.com/marten-seemann/qtls-go1-19 v0.1.1 // indirect
	github.com/onsi/ginkgo/v2 v2.5.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/crypto v0.2.0 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/text v0.4.0 // indirect
	golang.org/x/tools v0.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-micro/plugins/server/http => ./

replace go-micro.dev/v5 => ../../../go-micro/
