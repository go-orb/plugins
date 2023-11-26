module github.com/go-orb/plugins/client/orb/transport/basehttp

go 1.20

require (
	github.com/go-orb/go-orb v0.0.0-20231126093803-b366a8714a50
	github.com/go-orb/plugins/client/orb v0.0.0-20231126093807-8047bcf7d3d6
)

require golang.org/x/exp v0.0.0-20231110203233-9a3e6036ecaa // indirect

replace github.com/go-orb/plugins/client/orb => ../..
