# cqrs-bench — CLI Benchmark Tool

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench)

Benchmark any go-cqrs-lite backend with named workload profiles. Thin CLI front-end over the [benchkit](../../benchkit/README.md) library.

> **Design decisions:** See [ADR-0060](../../docs/adr/0060-benchkit-design-decisions.md) for rationale on codec-aware padding, warmup isolation, ReadRatio-as-passes, SkipPhases, and the DiskSizer interface.

## Install

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench/v4@latest
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

| Flag                           | Values                                                                                                          | Default  | Description                                           |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------- | -------- | ----------------------------------------------------- |
| `--backend`                    | `memory`, `sqlite`, `sqlite-cgo`, `pebble`, `bbolt`, `turso`, `postgres`, `mysql` (aliases: `mem`, `sq`, `peb`) | `memory` | Backend to benchmark                                  |
| `--dsn`                        | string                                                                                                          | temp     | Database DSN (sqlite/postgres/mysql)                  |
| `--dir`                        | path                                                                                                            | temp     | Data directory (pebble)                               |
| `--profile`                    | `dev`, `small`, `medium`, `large`, `stress`, `write-heavy`, `read-heavy`, `analytical`                          | `dev`    | Workload profile                                      |
| `--codec`                      | `json`, `cbor`                                                                                                  | `json`   | Payload codec                                         |
| `--format`                     | `auto`, `table`, `text`, `json`, `csv`, `tsv`, `markdown`, `benchstat`, `manifest`                              | `auto`   | Output format (`auto`: table in TTY, text when piped) |
| `--output`                     | path                                                                                                            | stdout   | Output file                                           |
| `--payload-size`               | int                                                                                                             | `256`    | Payload size in bytes                                 |
| `--payload-sizes`              | `64,256,4096`                                                                                                   | —        | Mixed per-event payload sizes (uniform random)        |
| `--warmup`                     | int                                                                                                             | `0`      | Warmup iterations before timing                       |
| `--repeat`                     | int                                                                                                             | `0`      | Run N times: median + per-metric cross-run CoV        |
| `--soak`                       | duration (`5m`, `1h`)                                                                                           | `0`      | Soak mode: leak/degradation trends                    |
| `--strict`                     | bool                                                                                                            | `false`  | Fail on skipped phases (CI gate)                      |
| `--progress`                   | duration                                                                                                        | `0`      | Heartbeat per phase                                   |
| `--quiet`                      | bool                                                                                                            | `false`  | Summary-only output                                   |
| `--cpuprofile`, `--memprofile` | path                                                                                                            | —        | pprof output                                          |

## Workload Profiles

| Profile       | Aggregates | Events/Agg | Total Events | Concurrency | ReadRatio | BatchSize | Description                  |
| ------------- | ---------- | ---------- | ------------ | ----------- | --------- | --------- | ---------------------------- |
| `dev`         | 100        | 5          | 500          | 1           | 0.2       | 1         | Quick smoke test             |
| `small`       | 1,000      | 10         | 10K          | 4           | 0.3       | 1         | Small dataset                |
| `medium`      | 10,000     | 50         | 500K         | 16          | 0.4       | 5         | Typical production load      |
| `large`       | 100,000    | 100        | 10M          | 32          | 0.5       | 10        | Large dataset                |
| `stress`      | 10,000     | 500        | 5M           | 64          | 0.2       | 1         | Stress test                  |
| `write-heavy` | 10,000     | 100        | 1M           | 32          | 0.1       | 1         | Write-dominated              |
| `read-heavy`  | 10,000     | 100        | 1M           | 32          | 0.8       | 1         | Read-dominated               |
| `analytical`  | 10,000     | 10         | 100K         | 16          | 0.9       | 1         | 90% reads + 5x journal scans |

## Statistical rigor (repeats and benchstat)

A single run is a point estimate. Before comparing backends, gating a
regression, or claiming an optimization win, run repeats and look at the
dispersion:

```bash
# Median of 5 runs + per-metric cross-run CoV. The 'Variation:' section
# lists every metric whose CoV exceeded 10%.
cqrs-bench run --backend sqlite --profile small --repeat 5

# benchstat-ready output: with --repeat N every metric gets N samples,
# which is what benchstat needs to report a confidence interval.
cqrs-bench run --backend sqlite --profile small --repeat 10 --format benchstat > new.txt
benchstat old.txt new.txt   # old.txt captured from the previous revision
```

Reading the numbers:

- `Repeat: median of N runs | CoV=...` is the write-throughput headline; the
  `Variation:` section extends the same analysis to every measured metric.
- A metric flagged `NOISY` (CoV >= 10%) is not decision-grade at that sample
  count: increase `--repeat`, use a larger profile, or bench on a quieter
  machine.
- `P100`/`Max` is the exact worst observed latency, not a percentile estimate;
  a single scheduler hiccup dominates it.
- The run records `Environment.LoadAvg1` and emits a warning when the machine
  was oversubscribed (load > CPU count), because such latencies include
  scheduler wait.

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
