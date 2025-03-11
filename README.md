# github.com/go-orb/go-orb -> plugins

This repo contains plugins for [github.com/go-orb/go-orb](https://github.com/go-orb/go-orb). These plugins implement the functionality of the core Go-Orb framework with various implementations for servers, clients, codecs, configuration sources, event handlers, and more.

With a single `_` import you can import a plugin and it's ready to use with the interfaces of [the core](https://github.com/go-orb/go-orb).

## Plugins

### Server

Server plugins provide different protocol implementations for your Go-Orb services.

All servers implement the `server.Entrypoint` interface and are configureable with Handlers and Middlewares.

See [RPC Benchmarks](https://github.com/go-orb/go-orb/wiki/RPC-Benchmarks) for a comparison of the different servers.

#### HTTP

- **HTTP/HTTPS/HTTP2/HTTP3**: Complete HTTP protocol family with support for REST APIs
- Features include middleware support, automatic TLS configuration
- Location: [`/server/http`](https://github.com/go-orb/plugins/tree/main/server/http)

#### DRPC

- High-performance RPC server implementation using the Storj DRPC protocol
- Optimized for efficient connection handling and request processing
- Offers excellent performance characteristics with low overhead
- Location: [`/server/drpc`](https://github.com/go-orb/plugins/tree/main/server/drpc)

#### gRPC

- Google's high-performance RPC framework implementation
- Support for streaming, authentication, and load balancing
- Strong typing via Protocol Buffers
- Location: [`/server/grpc`](https://github.com/go-orb/plugins/tree/main/server/grpc)

### Client

Client plugins provide transport implementations and middleware for communicating with services.

#### Transport Implementations

- **DRPC**: Connection-pooled DRPC client transport
  - Location: [`/client/orb_transport/drpc`](https://github.com/go-orb/plugins/tree/main/client/orb_transport/drpc)
- **gRPC/gRPCs**: Standard and TLS-enabled gRPC client transports, also Connection-pooled.
  - Location: [`/client/orb_transport/grpc`](https://github.com/go-orb/plugins/tree/main/client/orb_transport/grpc)
- **HTTP/HTTPS/HTTP2/HTTP3**: Full HTTP protocol family client transports
  - Location: [`/client/orb_transport/http`](https://github.com/go-orb/plugins/tree/main/client/orb_transport/http)

#### Middleware

- **Logging**: Request/response logging for debugging and observability
  - Location: [`/client/middleware/log`](https://github.com/go-orb/plugins/tree/main/client/middleware/log)

### Codecs

Codec plugins provide serialization and deserialization for messages:

- **Protocol Buffers**: Efficient binary serialization
  - Location: [`/codecs/proto`](https://github.com/go-orb/plugins/tree/main/codecs/proto)
- **JSON**: Standard and high-performance JSON implementations
  - Location: [`/codecs/json`](https://github.com/go-orb/plugins/tree/main/codecs/json), [`/codecs/goccyjson`](https://github.com/go-orb/plugins/tree/main/codecs/goccyjson)
- **YAML/TOML**: Human-readable configuration formats
  - Location: [`/codecs/yaml`](https://github.com/go-orb/plugins/tree/main/codecs/yaml), [`/codecs/toml`](https://github.com/go-orb/plugins/tree/main/codecs/toml)

### Configuration

Configuration plugins provide different sources for application configuration:

- **File**: Load configuration from JSON, YAML, or TOML files
  - Location: [`/config/source/file`](https://github.com/go-orb/plugins/tree/main/config/source/file)
- **Memory**: In-memory configuration store
  - Location: [`/config/source/memory`](https://github.com/go-orb/plugins/tree/main/config/source/memory)

### Event

Event plugins provide pub/sub messaging capabilities:

- **NATS**: NATS JetStream implementations
  - Location: [`/event/natsjs`](https://github.com/go-orb/plugins/tree/main/event/natsjs)

### Registry

Registry plugins provide service discovery and registration:

- **Consul**: HashiCorp Consul integration for service discovery
  - Location: [`/registry/consul`](https://github.com/go-orb/plugins/tree/main/registry/consul)
- **mDNS**: Local network service discovery via multicast DNS
  - Location: [`/registry/mdns`](https://github.com/go-orb/plugins/tree/main/registry/mdns)
- **Memory**: In-memory registry for testing
  - Location: [`/registry/memory`](https://github.com/go-orb/plugins/tree/main/registry/memory)
- **kvstore**: Key-value store registry
  - Location: [`/registry/kvstore`](https://github.com/go-orb/plugins/tree/main/registry/kvstore)

### KV Store

Key-Value Stores, provides easy access to configuration and other data.

- **NATS**: NATS JetStream key-value store, it implements the `kvstore.Watcher` interface for `/registry/kvstore`
  - Location: [`/kvstore/natsjs`](https://github.com/go-orb/plugins/tree/main/kvstore/natsjs)

## Community

- Chat with us on [Discord](https://discord.gg/4n6E4NYjnR) or on [Matrix](https://matrix.to/#/#go-orb:jochum.dev).

## Development

We do not accept commit's with a "replace" line in a go.mod.

### Run the tests

Install [dagger](https://docs.dagger.io/quickstart/cli)

```sh
dagger call test --root=.
```

### Check linting

```sh
dagger call lint --root=.
```

#### Run the tests for a single plugin

```sh
cd server/http
go test ./... -v -race -cover
cd ...
```

or with dagger

```sh
dagger call test --root=./server/http
```

### Quirks

#### It's not allowed to import plugins in github.com/go-orb/go-orb

To prevent import cycles it's not allowed to import plugins in github.com/go-orb/go-orb.

## Authors

- [David Brouwer](https://github.com/Davincible/)
- [René Jochum](https://github.com/jochumdev)

## License

go-orb is Apache 2.0 licensed same as go-micro.
