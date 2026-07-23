# stack/bench — Bundle Overhead Benchmarks

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/stack/bench/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/stack/bench/v4)

Benchmarks proving the `Bundle` composition layer is zero-overhead: accessing stores through `Bundle` fields is identical in `ns/op` and `allocs/op` to accessing them directly.

## Why?

The `stack.Bundle` is a struct of interface fields. Consumers access stores via `bundle.EventStore()`, `bundle.Repository(d)`, etc. These benchmarks verify that the indirection through the Bundle costs nothing at runtime — the compiler inlines field access, so there is zero overhead.

## Usage

```bash
go test -bench=. -benchmem ./...
```

## What's Measured

| Benchmark                              | What it proves                                  |
| -------------------------------------- | ----------------------------------------------- |
| `BenchmarkBundleEventStoreAccess`      | `bundle.EventStore()` == direct `store` access  |
| `BenchmarkBundleRepositoryAccess`      | `bundle.Repository(d)` == direct `repo` access  |
| `BenchmarkBundleEventBusAccess`        | `bundle.EventBus()` == direct `bus` access      |

Each benchmark runs both paths (through Bundle vs direct) and compares the results. If the Bundle adds overhead, the numbers diverge.

## Design

- **Same allocs/op**: Bundle field access should not allocate.
- **Same ns/op**: Bundle field access should not add measurable latency.
- **Contract**: These benchmarks are part of CI. If they diverge, the Bundle composition layer has a performance regression.

## Related Modules

- [**stack**](../README.md) — The `Bundle` type under benchmark
- [**benchkit**](../../benchkit/README.md) — The benchmarking library for workload profiles
