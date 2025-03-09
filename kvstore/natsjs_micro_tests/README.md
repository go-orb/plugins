# go-orb-natsjs-micro-tests

This repository contains interoperability tests and micro benchmarks for go-orb-natsjs and go-micro-natsjs.

Go-Orb runs in non-compatible mode to go-micro in the benchmarks.

## Benchmark results

```
goos: linux
goarch: amd64
pkg: github.com/go-orb/plugins/kvstore/natsjs_micro_tests
cpu: AMD Ryzen 9 5900X 12-Core Processor            
BenchmarkOrbSet-24                         26001             46239 ns/op            2786 B/op         56 allocs/op
BenchmarkMicroWrite-24                     25856             45857 ns/op            2900 B/op         57 allocs/op
BenchmarkOrbSetGet-24                      12591             93330 ns/op            7084 B/op        130 allocs/op
BenchmarkMicroWriteRead-24                 12084             96749 ns/op            8358 B/op        140 allocs/op
BenchmarkOrbGet-24                         28144             42355 ns/op            3721 B/op         68 allocs/op
BenchmarkMicroRead-24                      26840             44827 ns/op            4310 B/op         76 allocs/op
BenchmarkOrbList-24                          410           2620050 ns/op         2418802 B/op      27796 allocs/op
BenchmarkMicroList-24                          6         194777617 ns/op        231174128 B/op   2910016 allocs/op
BenchmarkOrbListPagination-24                421           2460378 ns/op         2344519 B/op      25997 allocs/op
BenchmarkMicroListPagination-24              482           2471308 ns/op         2371703 B/op      28785 allocs/op
BenchmarkOrbSetLarge-24                     1200            919767 ns/op         2113711 B/op         67 allocs/op
BenchmarkMicroWriteLarge-24                  530           2139809 ns/op         5312348 B/op         76 allocs/op
BenchmarkOrbGetLarge-24                     2284            487448 ns/op         3190474 B/op         81 allocs/op
BenchmarkMicroReadLarge-24                   183           6572089 ns/op         5315176 B/op         96 allocs/op
BenchmarkOrbSetGetLarge-24                   750           1444972 ns/op         6979392 B/op        149 allocs/op
BenchmarkMicroWriteReadLarge-24              134           8717133 ns/op        12316065 B/op        175 allocs/op
BenchmarkOrbDeleteLarge-24                   939           1168799 ns/op         4837791 B/op        132 allocs/op
BenchmarkMicroDeleteLarge-24                 453           2387964 ns/op         8422971 B/op        122 allocs/op
PASS
ok      github.com/go-orb/plugins/kvstore/natsjs_micro_tests    83.509s
```

## Benchmark Analysis

### Small Value Operations (typical key-value pairs)

#### Write Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set     | 46.2 µs | 2.8 KB | 56 allocs |
| go-micro Write | 45.9 µs | 2.9 KB | 57 allocs |

*Nearly identical performance*

#### Read Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Get     | 42.4 µs | 3.7 KB | 68 allocs |
| go-micro Read  | 44.8 µs | 4.3 KB | 76 allocs |

*go-orb slightly faster with less memory usage*

#### Write+Read Combined

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set+Get     | 93.3 µs | 7.1 KB | 130 allocs |
| go-micro Write+Read| 96.7 µs | 8.4 KB | 140 allocs |

*Very similar, go-orb slightly more efficient*

### Large Value Operations (1MB values)

#### Write Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set     | 919.8 µs | 2.1 MB | 67 allocs |
| go-micro Write | 2.1 ms | 5.3 MB | 76 allocs |

*go-orb is 2.3x faster, uses 2.5x less memory*

#### Read Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Get     | 487.4 µs | 3.2 MB | 81 allocs |
| go-micro Read  | 6.6 ms | 5.3 MB | 96 allocs |

*go-orb is 13.5x faster, uses 1.7x less memory*

#### Write+Read Combined

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set+Get      | 1.4 ms | 7.0 MB | 149 allocs |
| go-micro Write+Read | 8.7 ms | 12.3 MB| 175 allocs |

*go-orb is 6.0x faster, uses 1.8x less memory*

### List Operations (100 tables × 1000 items)

#### Full List

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb List     | 2.6 ms | 2.4 MB | 27.8K allocs |
| go-micro List   | 194.8 ms | 231.2 MB | 2.9M allocs |

*go-orb is 74.7x faster and uses 96.3x less memory*

#### Paginated List (100 items, offset 100)

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb List     | 2.5 ms | 2.3 MB | 26.0K allocs |
| go-micro List   | 2.5 ms | 2.4 MB | 28.8K allocs |

*Similar performance for paginated requests, go-orb has fewer allocations*

### Delete Operations (1MB values)

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Delete   | 1.2 ms | 4.8 MB | 132 allocs |
| go-micro Delete | 2.4 ms | 8.4 MB | 122 allocs |

*go-orb is 2.0x faster but uses slightly more allocations*

## Key Findings

1. Both implementations perform similarly for small values
2. go-orb shows significant advantages with large values (1MB)
3. **go-orb is dramatically more efficient for List operations across multiple tables**
4. Paginated listing performance is comparable between implementations
5. Memory usage is consistently lower in go-orb, especially for large operations
6. go-orb implementation shows better optimization for large-scale operations