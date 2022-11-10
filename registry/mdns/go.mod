module github.com/go-micro/plugins/registry/mdns

go 1.19

require (
	github.com/go-micro/plugins/registry/mdns/mdnsutil v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.3.0
	github.com/miekg/dns v1.1.50
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	golang.org/x/net v0.1.0
)

require (
	golang.org/x/exp v0.0.0-20221108223516-5d533826c662 // indirect
	golang.org/x/mod v0.6.0 // indirect
	golang.org/x/sys v0.1.0 // indirect
	golang.org/x/tools v0.2.0 // indirect
)

replace github.com/go-micro/plugins/registry/mdns/mdnsutil => ./

// replace go-micro.dev/v5 => ../../../orb
