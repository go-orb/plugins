module github.com/go-orb/plugins/client/orb/transport/basehttp

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231126205116-9614b6032b2c
	github.com/go-orb/plugins/client/orb v0.0.0-20231126111023-0a8c6d6cb2ee
)

require golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect

replace github.com/go-orb/plugins/client/orb => ../..
