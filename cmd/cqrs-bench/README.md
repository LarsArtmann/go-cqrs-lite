# cqrs-bench — CLI Benchmark Tool

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/cmd/cqrs-bench)

Benchmark any go-cqrs-lite backend with named workload profiles. Thin CLI front-end over the [benchkit](../../benchkit/README.md) library.

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

| Flag             | Values                              | Default | Description                          |
| ---------------- | ----------------------------------- | ------- | ------------------------------------ |
| `--backend`      | `memory`, `sqlite`, `pebble`        | —       | Backend to benchmark (aliases: `mem`, `sq`, `peb`) |
| `--dsn`          | string                              | —       | Database DSN (sqlite/postgres)       |
| `--dir`          | path                                | temp    | Data directory (pebble)              |
| `--profile`      | `dev`, `small`, `medium`, `large`, `stress`, `write-heavy`, `read-heavy` | — | Workload profile |
| `--codec`        | `json`, `cbor`                      | `json`  | Payload codec                        |
| `--format`       | `text`, `json`, `markdown`          | `text`  | Output format (compare only: `markdown`) |
| `--output`       | path                                | stdout  | Output file                          |
| `--payload-size` | int                                 | profile | Override payload size in bytes       |
| `--warmup`       | int                                 | profile | Warmup iterations before timing      |

## Workload Profiles

| Profile       | Events | Reads | Payload | Description                        |
| ------------- | ------ | ----- | ------- | ---------------------------------- |
| `dev`         | 100    | 100   | 256 B   | Quick smoke test                   |
| `small`       | 1K     | 1K    | 512 B   | Small dataset                      |
| `medium`      | 10K    | 10K   | 1 KB    | Typical production load            |
| `large`       | 100K   | 50K   | 2 KB    | Large dataset                      |
| `stress`      | 1M     | 100K  | 4 KB    | Stress test                        |
| `write-heavy` | 100K   | 1K    | 1 KB    | Write-dominated                    |
| `read-heavy`  | 10K    | 1M    | 1 KB    | Read-dominated                     |

## Output

```
Backend:     sqlite
Profile:     medium
Codec:       json

Write Throughput:  45,231 ops/s  (avg latency: 22.1 µs)
Read  Throughput:  89,102 ops/s  (avg latency: 11.2 µs)

Memory: 14.2 MB
```

## Design

- **Factory-based backends**: Each backend is a `Factory` function (`func() (*stack.Bundle, error)`) selected by name. Adding a backend = one case in the switch.
- **Temp dir management**: SQLite and Pebble auto-create and clean up temp dirs when `--dir` is not specified.
- **30-minute context timeout cap** prevents runaway benchmarks.

## Related Modules

- [**benchkit**](../../benchkit/README.md) — The benchmarking library powering this CLI
- [**stack**](../../stack/README.md) — Bundle presets for each backend
