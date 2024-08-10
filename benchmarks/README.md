# Benchmarks

This repo contains benchmarks for go-orb.

## rps

### `Requests per second` benchmark

The rps benchmark sends X bytes (default `1000`) to server which echoes it to the client.

This means 1000 bytes get encoded by the given content-type (default `application/x-protobuf`).

```txt
NAME:
   orb-rps-client - A new cli application

USAGE:
   orb-rps-client [global options] command [command options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --logger value                                             Default logger to use (e.g. jsonstderr, jsonstdout, textstderr, textsdout). (default: "slog")
   --registry value                                           Registry for discovery. etcd, mdns (default: "mdns") [$REGISTRY]
   --log_level value                                          Log level (FATAL, ERROR, NOTICE, WARN, INFO, DEBUG, TRACE) (default: "INFO") [$LOG_LEVEL]
   --registry_domain value                                    Registry domain. (default: "micro")
   --transport value                                          Transport to use (grpc, hertzhttp, http, uvm.) (default: "grpc") [$TRANSPORT]
   --content_type value                                       Content-Type (application/x-protobuf, application/json) (default: "application/x-protobuf") [$CONTENT_TYPE]
   --duration value                                           Duration in seconds (default: 15) [$DURATION]
   --timeout value                                            Timeout in seconds (default: 8) [$TIMEOUT]
   --threads value                                            Number of threads to use = runtime.GOMAXPROCS() (default: 24) [$THREADS]
   --package_size value                                       Per request package size (default: 1000) [$PACKAGE_SIZE]
   --registry_timeout value                                   Registry timeout in milliseconds. (default: 100) [$REGISTRY_TIMEOUT]
   --bypass_registry value                                    Bypasses the registry by caching it, set to 0 to disable (default: 1) [$BYPASS_REGISTRY]
   --connections value                                        Connections to keep open (default: 256) [$CONNECTIONS]
   --registry_addresses value [ --registry_addresses value ]  Registry addresses. (default: "localhost:8500")
   --config value [ --config value ]                          Config file
   --help, -h                                                 show help
```

## Run

### Build

```bash
go install github.com/go-orb/plugins/benchmarks/rps/cmd/orb-rps-server
go install github.com/go-orb/plugins/benchmarks/rps/cmd/orb-rps-client
```

### Server

```bash
GOMAXPROCS=4 ./orb-rps-server
```

### Client

Runs all built in transports one after another:

```bash
for i in "hertzhttp" "drpc" "grpc" "h2c" "http" "https" "http3" "hertzh2c"; do
  orb-rps-client --threads=12 --transport=$i 2>&1 | grep "Summary"
  sleep 5
done
```

## A note about Benchmark "ERROR" messages

Currently there are context canceled ERRORS this is a known issue and a wip.

## A note about using `wrk` with `orb-rps-server`

This is how we configure `wrk`.

The port is from the output of `orb-rps-server` you should be able to benchmark any of the HTTP servers with it.

```bash
wrk -H 'Host: 127.0.0.1' -H 'Connection: keep-alive' --latency -d 15 -c 256 --timeout 8 -t 4 http://127.0.0.1:31002/echo.Echo/Echo -s wrk_1000bytes_post.lua -- 16
```

## Results

See our [wiki](https://github.com/go-orb/plugins/wiki/RPC-Benchmarks)
