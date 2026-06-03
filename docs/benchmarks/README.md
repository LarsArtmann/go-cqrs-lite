# Benchmark Results

Hardware: AMD RYZEN AI MAX+ 395 w/ Radeon 8060S (32 cores, 96GB RAM), Go 1.26.3, linux/amd64.

Date: 2026-06-03.

## Micro-Benchmarks (per-operation)

Run with: `go test ./... -bench=. -benchmem -run=^$ -count=1 -timeout=10m`

### Event Construction

| Benchmark              | ns/op | B/op | allocs/op |
| ---------------------- | ----: | ---: | --------: |
| `NewEvent`             |   182 |  336 |         2 |
| `NewEvent_WithOptions` |   184 |  336 |         2 |
| `New()` typed payload  |   424 |  769 |         8 |
| `DecodePayload`        |   409 |  544 |         9 |
| `BusPublish`           |   293 |  867 |         0 |

### Signing

| Benchmark          |  ns/op | B/op | allocs/op |
| ------------------ | -----: | ---: | --------: |
| `CanonicalPayload` |    207 |  336 |         5 |
| `HMAC Sign`        |    637 |  848 |        11 |
| `HMAC Verify`      |    655 |  848 |        11 |
| `Ed25519 Sign`     | 13,341 |  400 |         6 |
| `Ed25519 Verify`   | 30,706 |  336 |         5 |

### IDs

| Benchmark   | ns/op | B/op | allocs/op |
| ----------- | ----: | ---: | --------: |
| `id.New`    |   100 |   16 |         1 |
| `id.Parse`  |    17 |    0 |         0 |
| `id.String` |    45 |   80 |         3 |

### Command / Query / Dispatcher

| Benchmark             | ns/op | B/op | allocs/op |
| --------------------- | ----: | ---: | --------: |
| `command.New`         |    35 |  160 |         1 |
| `command.Dispatch`    |    54 |   48 |         1 |
| `query.New`           |   0.6 |    0 |         0 |
| `dispatcher.Dispatch` |    24 |    0 |         0 |

### Memory Store

| Benchmark             |  ns/op |   B/op | allocs/op |
| --------------------- | -----: | -----: | --------: |
| `Save`                |    523 |    688 |         8 |
| `Load` (100 events)   |    272 |  1,792 |         1 |
| `ReadAll` (1K events) |  3,300 | 16,384 |         1 |
| `ReadFrom` (1K events)|    250 |    800 |         1 |
| `Bus.Publish`         |     66 |     48 |         3 |

#### ReadAll Scaling (post-optimization)

| Events | ns/op | B/op | allocs/op |
| ------ | -----: | -----: | --------: |
| 100    |   222 |  1,792 |         1 |
| 1K     | 2,573 | 16,384 |         1 |
| 10K    |21,322 |163,840 |         1 |
| 100K   |137,938|1,605,634|        1 |

Confirmed **O(n) linear scaling** — no sort overhead.

### Catalog

| Benchmark                 | ns/op |  B/op | allocs/op |
| ------------------------- | ----: | ----: | --------: |
| `SchemaFromType` (cached) |     8 |     0 |         0 |
| `Registry.Build`          | 1,084 | 2,688 |        37 |

### Storage (SQLite, in-memory)

| Benchmark              |   ns/op |    B/op | allocs/op |
| ---------------------- | ------: | ------: | --------: |
| `SQLite.Save`          |  40,392 |   3,950 |        89 |
| `SQLite.Load`          |  47,973 |  19,748 |       544 |
| `SQLite.ReadAll` (100) | 380,068 | 175,212 |     5,033 |
| `SQLite.LoadToVersion` | 107,671 |  45,587 |     1,300 |

### Snapshot

| Benchmark            | ns/op | B/op | allocs/op |
| -------------------- | ----: | ---: | --------: |
| `EveryNEvents`       |     7 |    8 |         1 |
| `SaveSnapshot`       |   138 |  144 |         3 |
| `SnapshotStore.Load` |    64 |  112 |         2 |

### Schema

| Benchmark                          |  ns/op |   B/op | allocs/op |
| ---------------------------------- | -----: | -----: | --------: |
| `VersionedStore.Load` (100 events) | 23,877 | 38,978 |       305 |

---

## Realistic Scale Benchmarks

Run with: `go test ./integration/... -tags=scale -bench=BenchmarkRealistic -benchmem -run=^$ -benchtime=1x -timeout=10m`

Gated by `//go:build scale` — excluded from normal builds and CI.

These use realistic JSON payloads (e-commerce order model) and exercise the full CQRS stack.

### Full Pipeline

10K orders × 2 commands each = 20K events through: command dispatch → decider execute → JSON encode → MemoryStore save → MemoryBus publish → projection Runner (typed JSON decode).

| Metric     | Value               |
| ---------- | ------------------- |
| Total time | 88ms                |
| Throughput | **227K events/sec** |
| Memory     | 68MB, 1.25M allocs  |

### Projection Replay

100K pre-stored events replayed from journal through typed projection handlers (JSON decode per event).

| Metric     | Value               |
| ---------- | ------------------- |
| Total time | 164ms               |
| Throughput | **608K events/sec** |
| Memory     | 101MB, 1.85M allocs |

### Concurrent Decider

32 goroutines (one per CPU core), each executing 100 new aggregates with JSON payloads and snapshot every 50 events.

| Metric     | Value                 |
| ---------- | --------------------- |
| Total time | 10ms (3,200 executes) |
| Throughput | **330K executes/sec** |
| Memory     | 8MB, 151K allocs      |

### HMAC Signing

10K events with realistic `OrderCreated` payloads (~130 bytes each).

| Benchmark | Time | Throughput             | Memory             |
| --------- | ---- | ---------------------- | ------------------ |
| Sign      | 7ms  | **1.39M signs/sec**    | 8.5MB, 110K allocs |
| Verify    | 7ms  | **1.34M verifies/sec** | 8.5MB, 110K allocs |

### Aggregate Listing

10K aggregates (3 events each), cursor-paginated through 100 pages.

| Metric     | Before | After | Change |
| ---------- | ------ | ----- | ------ |
| Total time | 840ms  | 33ms  | **-96% (25x faster)** |
| Throughput | 12K/s  | 303K/s| **+2,425%** |
| Memory     | 809MB  | 165MB | **-80%** |
| Allocs     | 1M     | 10,582| **-99%** |

Fix: `InMemoryAggregateReader` now caches a sorted aggregate index. First `List()` builds the cache; subsequent calls only filter/paginate.

### Snapshot vs Replay Load

100 aggregates × 500 events each. Compares loading by replaying all events vs loading from a snapshot taken every 100 events.

| Method               | Time   | Throughput          | Memory            |
| -------------------- | ------ | ------------------- | ----------------- |
| Replay (500 events)  | 44ms   | **2,292 loads/sec** | 18MB, 401K allocs |
| Snapshot (every 100) | 0.16ms | **616K loads/sec**  | 99KB, 1.8K allocs |
| **Speedup**          |        | **269x**            |                   |

### Codec

| Benchmark      | ns/op | B/op | allocs/op |
| -------------- | ----: | ---: | --------: |
| JSON Encode    |   249 |  192 |         6 |
| JSON Decode    |   519 |  592 |        12 |
| Raw Encode     |    14 |   24 |         1 |
| Raw Decode     |    23 |   48 |         2 |

### Watermill Adapter

| Benchmark          | ns/op | B/op | allocs/op |
| ------------------ | ----: | ---: | --------: |
| eventToMessage     |   442 |  944 |        15 |
| messageToEvent     |   528 |  568 |         9 |
| Publisher Publish  |   593 |  616 |        12 |
| buildMetadata      |   122 |    0 |         0 |

### Turso (Embedded LibSQL)

| Benchmark |  ns/op | B/op | allocs/op |
| --------- | -----: | ---: | --------: |
| Save      | 83,244 |23,133|       365 |
| Load (100)|1,079,546|996,264|    16,450 |

### Query Dispatch

1K queries returning paginated results (50 of 1,000 items).

| Metric     | Value                 |
| ---------- | --------------------- |
| Total time | 0.08ms                |
| Throughput | **12.9M queries/sec** |
| Memory     | 128KB, 3K allocs      |

---

## Optimization History

### Session 142–143 (2026-06-02 to 2026-06-03)

| Change                            | Before                           | After                      | Improvement                      |
| --------------------------------- | -------------------------------- | -------------------------- | -------------------------------- |
| Lazy metadata map                 | 3 allocs, 384B, 204ns (NewEvent) | 2 allocs, 336B, 182ns      | -1 alloc, -11% time              |
| SchemaFromType cache              | 15 allocs, 1,200B, 553ns         | 0 allocs, 0B, 8ns          | -15 allocs, **-99% time**        |
| Remove Payload() clone            | 10 allocs, 560B (DecodePayload)  | 9 allocs, 544B             | -1 alloc (cascades everywhere)   |
| buildEvent() skip copy            | New() had double copy            | New() skips redundant copy | -1 alloc per New()               |
| Stamp encoding directly           | 9 allocs, 793B (New typed)       | 8 allocs, 769B             | -1 alloc, -24B                   |
| canonicalPayload pre-sized buffer | 10 allocs → 6 allocs             | 6→5 allocs, 336B           | -1 alloc (from Payload() change) |

### Session 143–144 (2026-06-03)

|| Change                               | Before                     | After                        | Improvement                    |
| ------------------------------------ | -------------------------- | ---------------------------- | ------------------------------ |
|| MemoryStore global log               | 98μs, 52KB, 12 allocs (1K) | 3.3μs, 16KB, 1 alloc (1K)   | **-96% time, -92% allocs**     |
|| Listing projection cache             | 840ms, 809MB, 1M allocs    | 33ms, 165MB, 10K allocs      | **-96% time, -99% allocs**     |
|| ImmutableEvent opts pointer          | 336B per event             | 304B per event               | -10% memory per event          |

### What can't be optimized (accepted costs)

| Constraint                                  | Reason                                                         |
| ------------------------------------------- | -------------------------------------------------------------- |
| `&ImmutableEvent{...}` heap alloc           | Unavoidable: struct escapes to interface                       |
| `make([]byte, len(payload))` defensive copy | Library safety: prevents caller from mutating event            |
| `ID.String()` 3 allocs per call             | External library (`go-branded-id`)                             |
| `Metadata()` clone on every access          | Map aliasing risk: `Custom` map shares reference if not cloned |
| `findCodecOption` probe alloc               | Requires breaking `Option` type from `func` to interface (v3)  |
| SQLite allocs (554 per load)                | Pure Go SQL engine overhead                                    |
| MemoryStore event duplication               | Events stored in both `events` map and `globalLog` (2× mem)    |
