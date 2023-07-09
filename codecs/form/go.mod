module github.com/go-orb/plugins/codecs/form

go 1.20

replace github.com/go-orb/plugins/codecs/proto => ../proto

require (
	github.com/go-orb/go-orb v0.0.0-20230709080055-9c340136e7d1
	github.com/go-playground/form/v4 v4.2.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require github.com/stretchr/testify v1.8.4
