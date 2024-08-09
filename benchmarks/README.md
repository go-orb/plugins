# Benchmarks

This repo contains benchmarks for go-orb.

## rps

### `Requests per second` benchmark

The rps benchmark sends X bytes (default `1000`) to server which echoes it to the client.

This means 1000 bytes get encoded by the given content-type (default `application/x-protobuf`).

```txt
GLOBAL OPTIONS:
   --registry value                                           Registry for discovery. etcd, mdns (default: "mdns") [$REGISTRY]
   --log_level value                                          Log level (FATAL, ERROR, NOTICE, WARN, INFO, DEBUG, TRACE) (default: "INFO") [$LOG_LEVEL]
   --transport value                                          Transport to use (grpc, hertzhttp, http, uvm.) [$TRANSPORT]
   --content_type value                                       Content-Type (application/x-protobuf, application/json) (default: "application/x-protobuf") [$CONTENT_TYPE]
   --package_size value                                       Per request package size (default: 1000) [$PACKAGE_SIZE]
   --bypass_registry value                                    Bypasses the registry by caching it, set to 0 to disable (default: 1) [$BYPASS_REGISTRY]
   --connections value                                        Connections to keep open (default: 256) [$CONNECTIONS]
   --duration value                                           Duration in seconds (default: 15) [$DURATION]
   --timeout value                                            Timeout in seconds (default: 8) [$TIMEOUT]
   --threads value                                            Number of threads to use (default: 24) [$THREADS]
   --config value [ --config value ]                          Config file
   --help, -h                                                 show help
```

## Run

### Server

```bash
./registry/consul/test/bin/linux_amd64/consul agent -dev
go run ./server/... -- --registry="consul"
```

### Client

```bash
pushd client
go build .
./client --transport="grpc" -- --registry="consul"
popd
```

## A note about Benchmark "ERROR" messages

Currently there are context canceled ERRORS this is a known issue and a wip.

## jochumdev results

```bash
for transport in "hertzh2c" "http3" "http" "h2c" "grpc" "drpc" "hertzhttp"; do
   ./client --transport="${transport}" --threads=4 --connections=32 -- --registry="consul" 2>&1 | grep "Summary"
done
```

```text
time=2024-08-08T03:33:09.923+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=hertzh2c package_size=1000 content_type=application/x-protobuf reqsOk=557247 reqsError=27
time=2024-08-08T13:30:00.156+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=http3 package_size=1000 content_type=application/x-protobuf reqsOk=686080 reqsError=0
time=2024-08-08T03:34:10.958+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=http package_size=1000 content_type=application/x-protobuf reqsOk=1172285 reqsError=0
time=2024-08-08T03:34:41.473+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=h2c package_size=1000 content_type=application/x-protobuf reqsOk=1168378 reqsError=0
time=2024-08-08T03:35:11.987+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=grpc package_size=1000 content_type=application/x-protobuf reqsOk=1291629 reqsError=8
time=2024-08-08T03:33:40.440+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=drpc package_size=1000 content_type=application/x-protobuf reqsOk=1874516 reqsError=10
time=2024-08-08T03:35:42.503+02:00 level=INFO msg=Summary bypass_registry=1 connections=32 duration=15 timeout=8 threads=4 transport=hertzhttp package_size=1000 content_type=application/x-protobuf reqsOk=2168585 reqsError=2
```
