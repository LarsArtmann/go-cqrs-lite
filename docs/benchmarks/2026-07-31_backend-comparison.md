# Backend Comparison Benchmark — 2026-07-31

Full 5-backend comparison across multiple profiles and dimensions. All numbers measured with `cqrs-bench` unless otherwise noted.

## Environment

| Item       | Value                                                            |
| ---------- | ---------------------------------------------------------------- |
| CPU        | AMD Ryzen AI MAX+ 395 (32 cores)                                 |
| RAM        | 96 GB                                                            |
| OS         | Linux (NixOS)                                                    |
| Go         | 1.26.5                                                           |
| GOMAXPROCS | 32                                                               |
| Postgres   | 16.14 (Docker `postgres:16-alpine`, separate container)          |
| DuckDB     | Embedded (CGo, statically linked C++ engine)                     |
| Pebble     | Embedded (cockroachdb/pebble)                                    |
| SQLite     | Embedded (pure-Go `modernc.org/sqlite`, WAL mode + busy_timeout) |
| Codec      | JSON (256-byte payloads unless noted)                            |

## 1. All-Backend Comparison (small profile)

Profile: 1,000 streams x 10 events = 10,000 events, 4 goroutines, BatchSize=1, ReadRatio=0.3.

| Backend  | Write P50 | Write P99 | Write ops/s | Load P50 | Load P99 | Heap  | Disk  |
| -------- | --------- | --------- | ----------- | -------- | -------- | ----- | ----- |
| memory   | 350ns     | 2.9us     | 189.8K/s    | 170ns    | 1.2us    | 17 MB | 0     |
| pebble   | 11.6us    | 129us     | 100.0K/s    | 55.6us   | 240us    | 41 MB | 13 MB |
| sqlite   | 212us     | 990us     | 14.4K/s     | 225us    | 1.1ms    | 37 MB | 12 MB |
| duckdb   | 1.71ms    | 4.97ms    | 2.1K/s      | 1.01ms   | 1.76ms   | 16 MB | --    |
| postgres | 27.6ms    | 779ms     | 47/s        | 137us    | 1.17ms   | 16 MB | --    |

> Turso excluded: `cqrs-bench` has no factory entry for it. Turso is libSQL (SQLite fork), so numbers would be SQLite-class.

### Root Cause: Why Postgres and DuckDB Look Slow

Both use the identical `SQLEventStore` write path. Each `Save()` call executes:

```
BEGIN -> SELECT MAX(version) -> INSERT (1 row) -> COMMIT
```

With `BatchSize=1` (the `small` profile), every event is a separate transaction with a separate COMMIT (fsync).

**Postgres (47/s)** is bottlenecked by `synchronous_commit=on` (default). Each COMMIT fsyncs WAL to disk before returning. At one event per transaction, that's one fsync per event (~20ms each on this container). The `stack/postgres` preset has zero pool tuning: bare `sql.Open("pgx", dsn)`, no `SetMaxOpenConns`, no `synchronous_commit` override.

**DuckDB (2.1K/s)** is a columnar OLAP engine. Single-row INSERTs are its worst case: each INSERT rewrites column chunks instead of appending to a row log. DuckDB's strengths are analytical scans and bulk loads, not OLTP-style row-by-row appends.

## 2. Postgres Durability Experiment

Same Postgres 16 container, same `small` profile, same hardware. Only difference: `synchronous_commit`.

| Config                            | Write P50 | Write P99 | Write ops/s | Load P50 |
| --------------------------------- | --------- | --------- | ----------- | -------- |
| `synchronous_commit=on` (default) | 27.6ms    | 779ms     | 47/s        | 137us    |
| `synchronous_commit=off`          | 130us     | 3.79ms    | 18,200/s    | 101us    |

387x write speedup. Postgres reads are fast in both configs (best Load P50 of all SQL backends).

## 3. Medium Profile (500K events, 16 goroutines)

| Backend | Write P50 | Write P99 | Write ops/s | Load P50 | Heap   | Disk   |
| ------- | --------- | --------- | ----------- | -------- | ------ | ------ |
| memory  | 1.7us     | 36.9us    | 184.8K/s    | 400ns    | 926 MB | 0      |
| pebble  | 40.1us    | 944us     | 109.6K/s    | 624us    | 1.8 GB | 185 MB |
| sqlite  | 2.65ms    | 20.3ms    | 16.0K/s     | 2.81ms   | 1.8 GB | 388 MB |

## 4. Codec Comparison (SQLite, small profile, 3-repeat median)

| Codec | Write P50 | Write ops/s |
| ----- | --------- | ----------- |
| JSON  | 201us     | 14.7K/s     |
| CBOR  | 216us     | 13.9K/s     |

At 256-byte payloads, SQLite's fsync dominates. Codec choice is irrelevant on SQLite at small payload sizes. CBOR's payoff only appears on the memory backend at larger payloads.

## 5. Concurrency Sweep (memory backend, small profile)

| Workers | Write ops/s | RawSink ops/s | Load P50 | Heap   |
| ------- | ----------- | ------------- | -------- | ------ |
| 1       | 235.0K/s    | 2.1M/s        | 110ns    | 21 MB  |
| 2       | 224.8K/s    | 1.7M/s        | 170ns    | 28 MB  |
| 4       | 223.2K/s    | 1.3M/s        | 140ns    | 54 MB  |
| 8       | 226.9K/s    | 1.4M/s        | 250ns    | 65 MB  |

Memory scales flat on concurrency: already CPU-saturated at 1 worker for map operations.

## 6. Read-Heavy vs Write-Heavy (Pebble, 1M events, 32 goroutines)

| Profile     | Write ops/s | Load P50 | Projection Lag |
| ----------- | ----------- | -------- | -------------- |
| write-heavy | 67.4K/s     | 3.38ms   | 95ms           |
| read-heavy  | 69.0K/s     | 3.45ms   | 89ms           |

Little difference: Pebble is not write- or read-bound at this scale. Both profiles produce near-identical generated-write throughput.

## Reproduction

### Build

```bash
# Pure-Go (memory, sqlite, pebble, postgres)
cd cmd/cqrs-bench && GOWORK=off go build -tags "goexperiment.jsonv2" -o /tmp/cqrs-bench .

# With DuckDB support (requires CGo + gcc)
cd cmd/cqrs-bench && CGO_ENABLED=1 GOWORK=off go build -tags "goexperiment.jsonv2" -o /tmp/cqrs-bench-cgo .
```

### Run

```bash
# All-backend comparison (markdown table)
/tmp/cqrs-bench compare --profile small --backends mem,sq,peb --format markdown

# Postgres (requires running instance)
docker run -d --name bench-pg -e POSTGRES_PASSWORD=bench -e POSTGRES_USER=bench \
  -e POSTGRES_DB=bench -p 5433:5432 postgres:16-alpine
/tmp/cqrs-bench run --backend postgres \
  --dsn "postgres://bench:bench@127.0.0.1:5433/bench?sslmode=disable" --profile small

# Postgres with synchronous_commit=off
docker run -d --name bench-pg-fast -e POSTGRES_PASSWORD=bench -e POSTGRES_USER=bench \
  -e POSTGRES_DB=bench -p 5433:5432 postgres:16-alpine -c synchronous_commit=off
/tmp/cqrs-bench run --backend postgres \
  --dsn "postgres://bench:bench@127.0.0.1:5433/bench?sslmode=disable" --profile small

# DuckDB (in-memory)
/tmp/cqrs-bench-cgo run --backend duckdb --dsn "" --profile small

# Concurrency sweep
/tmp/cqrs-bench sweep --param workers --values 1,2,4,8 --backend memory --profile small

# Medium profile (500K events)
/tmp/cqrs-bench compare --profile medium --backends mem,peb,sq --format markdown
```

## Takeaways

1. **Pebble is the persistent write champion** -- 100K/s writes, 7x SQLite, 50x DuckDB point writes.
2. **Postgres reads are excellent** (137us P50) but writes are crippled by fsync-per-event at BatchSize=1. Setting `synchronous_commit=off` yields 387x improvement.
3. **DuckDB is OLAP, not OLTP** -- single-row inserts are its worst case. Use the `analytical` profile to see its scan/aggregation strengths.
4. **Codec choice doesn't matter on SQLite** at small payloads -- fsync dominates, not serialization.
5. **Memory backend scales flat on concurrency** -- CPU-saturated at 1 worker for map operations.
