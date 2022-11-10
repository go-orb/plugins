module github.com/go-micro/plugins/registry/mdns

go 1.19

require (
	github.com/go-micro/plugins/log/text v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.3.0
	github.com/miekg/dns v1.1.50
	go-micro.dev/v5 v5.0.0-00010101000000-000000000000
	golang.org/x/net v0.2.0
)

require (
	golang.org/x/exp v0.0.0-20221109205753-fc8884afc316 // indirect
	golang.org/x/mod v0.7.0 // indirect
	golang.org/x/sys v0.2.0 // indirect
	golang.org/x/tools v0.3.0 // indirect
)

replace github.com/go-micro/plugins/log/text => ../../log/text

replace go-micro.dev/v5 => ../../../orb
