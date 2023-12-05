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
