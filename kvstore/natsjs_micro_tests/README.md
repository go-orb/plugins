# go-orb-natsjs-micro-tests

This repository contains interoperability tests and micro benchmarks for go-orb-natsjs and go-micro-natsjs.

## Benchmark results

```
go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/go-orb/plugins/kvstore/natsjs_micro_tests
cpu: AMD Ryzen 9 5900X 12-Core Processor            
BenchmarkOrbSet-24                         26161             46123 ns/op            2786 B/op         56 allocs/op
BenchmarkMicroWrite-24                     26779             46073 ns/op            2901 B/op         57 allocs/op
BenchmarkOrbSetGet-24                      12884             92183 ns/op            7077 B/op        130 allocs/op
BenchmarkMicroWriteRead-24                 12454             98480 ns/op            8330 B/op        140 allocs/op
BenchmarkOrbGet-24                         28482             41512 ns/op            3719 B/op         68 allocs/op
BenchmarkMicroRead-24                      27529             43242 ns/op            4310 B/op         76 allocs/op
BenchmarkOrbList-24                          512           2313099 ns/op         2241771 B/op      27544 allocs/op
BenchmarkMicroList-24                        506           2341094 ns/op         2294626 B/op      28507 allocs/op
BenchmarkOrbListPagination-24                547           2202895 ns/op         2186384 B/op      25714 allocs/op
BenchmarkMicroListPagination-24              514           2323418 ns/op         2292259 B/op      28516 allocs/op
BenchmarkOrbSetLarge-24                     1258            891673 ns/op         2113585 B/op         67 allocs/op
BenchmarkMicroWriteLarge-24                  555           2128019 ns/op         5166856 B/op         77 allocs/op
BenchmarkOrbGetLarge-24                     2283            475933 ns/op         3188447 B/op         81 allocs/op
BenchmarkMicroReadLarge-24                   181           6426774 ns/op         5311065 B/op         95 allocs/op
BenchmarkOrbSetGetLarge-24                   758           1425734 ns/op         6980106 B/op        148 allocs/op
BenchmarkMicroWriteReadLarge-24              136           8662620 ns/op        12417163 B/op        172 allocs/op
BenchmarkOrbDeleteLarge-24                   957           1138639 ns/op         4842394 B/op        132 allocs/op
BenchmarkMicroDeleteLarge-24                 469           2322944 ns/op         8428178 B/op        123 allocs/op
```

## Benchmark Analysis

### Small Value Operations (typical key-value pairs)

#### Write Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set     | 46.6 µs | 2.8 KB | 56 allocs |
| go-micro Write | 46.1 µs | 2.9 KB | 57 allocs |

*Nearly identical performance*

#### Read Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Get     | 43.4 µs | 3.7 KB | 68 allocs |
| go-micro Read  | 44.6 µs | 4.3 KB | 76 allocs |

*go-orb slightly faster with less memory usage*

#### Write+Read Combined

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set+Get     | 95.2 µs | 7.1 KB | 130 allocs |
| go-micro Write+Read| 97.4 µs | 8.4 KB | 140 allocs |

*Very similar, go-orb slightly more efficient*

### Large Value Operations (1MB values)

#### Write Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set     | 894 µs | 2.1 MB | 66 allocs |
| go-micro Write | 2.1 ms | 5.2 MB | 77 allocs |

*go-orb is 2.4x faster, uses 2.5x less memory*

#### Read Performance

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Get     | 484 µs | 3.2 MB | 81 allocs |
| go-micro Read  | 6.5 ms | 5.3 MB | 95 allocs |

*go-orb is 13.4x faster, uses 1.7x less memory*

#### Write+Read Combined

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Set+Get      | 1.4 ms | 7.0 MB | 148 allocs |
| go-micro Write+Read | 8.9 ms | 12.5 MB| 172 allocs |

*go-orb is 6.3x faster, uses 1.8x less memory*

### List Operations (1000 items)

#### Full List

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb List     | 2.35 ms | 2.2 MB | 27.5K allocs |
| go-micro List   | 2.37 ms | 2.3 MB | 28.5K allocs |

*Nearly identical performance*

#### Paginated List (100 items, offset 100)

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb List     | 2.25 ms | 2.2 MB | 25.7K allocs |
| go-micro List   | 2.39 ms | 2.3 MB | 28.5K allocs |

*go-orb slightly faster with fewer allocations*

### Delete Operations (1MB values)

| Implementation | Time | Memory | Allocations |
|----------------|------|---------|-------------|
| go-orb Delete   | 1.1 ms | 4.5 MB | 131 allocs |
| go-micro Delete | 2.4 ms | 8.4 MB | 121 allocs |

*go-orb is 2.2x faster, uses 1.9x less memory*

## Key Findings

1. Both implementations perform similarly for small values
2. go-orb shows significant advantages with large values (1MB)
3. List operations are efficient in both implementations
4. go-orb's pagination shows slightly better performance
5. Memory usage is consistently lower in go-orb, especially for large values