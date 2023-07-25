module github.com/go-orb/plugins/registry/tests

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20230725002816-6b0ffbf94b15
	github.com/stretchr/testify v1.8.4
	golang.org/x/exp v0.0.0-20230724220655-d98519c11495
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-orb/go-orb => ../../../go-orb
