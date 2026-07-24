# cqrs-bench — CLI Benchmark Tool

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench)

Benchmark any go-cqrs-lite backend with named workload profiles. Thin CLI front-end over the [benchkit](../../benchkit/README.md) library.

> **Design decisions:** See [ADR-0060](../../docs/adr/0060-benchkit-design-decisions.md) for rationale on codec-aware padding, warmup isolation, ReadRatio-as-passes, SkipPhases, and the DiskSizer interface.

## Install

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench@latest
```

## Usage

### Run a single backend

```bash
cqrs-bench run --backend sqlite --profile medium --codec json
cqrs-bench run --backend pebble --dir /tmp/pebble-bench --profile stress --codec cbor
cqrs-bench run --backend memory --profile dev --format json --output results.json
```

### Compare backends

```bash
cqrs-bench compare --profile medium --backends mem,sq,peb --format markdown
```

### Flags

| Flag             | Values                                                                   | Default  | Description                                      |
| ---------------- | ------------------------------------------------------------------------ | -------- | ------------------------------------------------ |
| `--backend`      | `memory`, `sqlite`, `pebble` (aliases: `mem`, `sq`, `peb`)               | `memory` | Backend to benchmark                             |
| `--dsn`          | string                                                                   | temp     | Database DSN (sqlite); ignored for memory/pebble |
| `--dir`          | path                                                                     | temp     | Data directory (pebble)                          |
| `--profile`      | `dev`, `small`, `medium`, `large`, `stress`, `write-heavy`, `read-heavy` | `dev`    | Workload profile                                 |
| `--codec`        | `json`, `cbor`                                                           | `json`   | Payload codec                                    |
| `--format`       | `text`, `json` (`markdown` in compare only)                              | `text`   | Output format                                    |
| `--output`       | path                                                                     | stdout   | Output file                                      |
| `--payload-size` | int                                                                      | `256`    | Payload size in bytes                            |
| `--warmup`       | int                                                                      | `0`      | Warmup iterations before timing                  |

## Workload Profiles

| Profile       | Aggregates | Events/Agg | Total Events | Concurrency | ReadRatio | BatchSize | Description             |
| ------------- | ---------- | ---------- | ------------ | ----------- | --------- | --------- | ----------------------- |
| `dev`         | 100        | 5          | 500          | 1           | 0.2       | 1         | Quick smoke test        |
| `small`       | 1,000      | 10         | 10K          | 4           | 0.3       | 1         | Small dataset           |
| `medium`      | 10,000     | 50         | 500K         | 16          | 0.4       | 5         | Typical production load |
| `large`       | 100,000    | 100        | 10M          | 32          | 0.5       | 10        | Large dataset           |
| `stress`      | 10,000     | 500        | 5M           | 64          | 0.2       | 1         | Stress test             |
| `write-heavy` | 10,000     | 100        | 1M           | 32          | 0.1       | 1         | Write-dominated         |
| `read-heavy`  | 10,000     | 100        | 1M           | 32          | 0.8       | 1         | Read-dominated          |

## Output

```
Benchmark: sqlite | profile=medium | codec=json
============================================================
Workload: 10,000 aggregates x 50 events = 500,000 events
Payload:  256 bytes/event
Duration: 4.2s

Write Performance:
  Latency: P50=455µs P95=2.1ms P99=4.8ms Max=12ms
  Throughput: 119,047 events/sec

Read Performance:
  Latency: P50=125µs P95=891µs P99=1.8ms Max=5ms

Read Model:
  Set: P50=98µs P95=412µs P99=780µs Max=2ms
  Get: P50=52µs P95=201µs P99=390µs Max=1ms

Projection: 500,000 events, lag=2.1s

Resources:
  Heap:  42 MB peak
  Delta: 18 MB
  CPU:   3.2s

Storage:
  Database: 12 MB
  Events:   8 MB
  Overhead: 50.0%
```

## Design

- **Factory-based backends**: Each backend is a `Factory` function (`func() (*stack.Bundle, error)`) selected by name. Adding a backend = one case in the switch.
- **Temp dir management**: SQLite and Pebble auto-create and clean up temp dirs when `--dir` is not specified.
- **30-minute context timeout cap** prevents runaway benchmarks.

## Related Modules

- [**benchkit**](../../benchkit/README.md) — The benchmarking library powering this CLI
- [**stack**](../../stack/README.md) — Bundle presets for each backend
