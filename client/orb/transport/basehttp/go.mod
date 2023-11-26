module github.com/go-orb/plugins/client/orb/transport/basehttp

go 1.21.4

require (
	github.com/go-orb/go-orb v0.0.0-20231126231708-592c8d8d05c6
	github.com/go-orb/plugins/client/orb v0.0.0-20231126210304-ce92168466b6
)

require golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect

replace github.com/go-orb/plugins/client/orb => ../..
