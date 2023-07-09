module github.com/go-orb/plugins/codecs/form

go 1.20

replace github.com/go-orb/plugins/codecs/proto => ../proto

require (
	github.com/go-orb/go-orb v0.0.0-20230709054753-fbffd5a3e495
	github.com/go-playground/form/v4 v4.2.0
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/subcommands v1.2.0 // indirect
	github.com/google/wire v0.5.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/tools v0.11.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require github.com/stretchr/testify v1.8.4
