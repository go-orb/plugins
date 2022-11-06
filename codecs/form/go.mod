module github.com/go-micro/plugins/codecs/form

go 1.19

replace github.com/go-micro/plugins/codecs/proto => ../proto

replace go-micro.dev/v5 => ../../../orb

require (
	github.com/go-playground/form/v4 v4.2.0
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/stretchr/testify v1.8.1
)
