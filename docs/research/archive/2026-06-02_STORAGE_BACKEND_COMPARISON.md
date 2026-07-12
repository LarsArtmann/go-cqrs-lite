# Storage Backend Comparison

\n> **Status:** RESOLVED — informed multi-backend storage design

**Date:** 2026-06-02
**Environment:** AMD RYZEN AI MAX+ 395, 96GB RAM, Go 1.26.3, linux/amd64

## Methodology

SQLite benchmarks use `modernc.org/sqlite` (pure Go, no CGo) with in-memory databases.
PostgreSQL benchmarks use `go-sqlmock` (mock — measures framework overhead, not real I/O).
Pebble benchmarks use real embedded key-value store.

## Save (single event, new aggregate)

| Backend           |     ns/op |   B/op | allocs/op | Notes                                                    |
| ----------------- | --------: | -----: | --------: | -------------------------------------------------------- |
| SQLite (real)     |    41,042 |  4,080 |        92 | Full transaction: BEGIN + checkVersion + INSERT + COMMIT |
| PostgreSQL (mock) | 1,040,194 | 12,897 |       177 | Mock overhead dominates — not real I/O                   |
| MemoryStore       |       582 |    736 |         9 | In-memory baseline                                       |
| Pebble            |       N/A |    N/A |       N/A | Uses AppendBatch, no Save with version check             |

## Load (single aggregate, 1 event)

| Backend           |   ns/op |   B/op | allocs/op | Notes                     |
| ----------------- | ------: | -----: | --------: | ------------------------- |
| SQLite (real)     |  48,505 | 20,233 |       554 | SQL engine + row scanning |
| PostgreSQL (mock) | 118,443 | 12,958 |       130 | Mock overhead             |
| MemoryStore       |     216 |  1,792 |         1 | In-memory baseline        |

## ReadAll (100 events cross-aggregate)

| Backend       |   ns/op |    B/op | allocs/op |
| ------------- | ------: | ------: | --------: |
| SQLite (real) | 388,482 | 180,010 |     5,133 |

## Recommendations

1. **For development/testing:** MemoryStore — zero I/O, 582ns Save
2. **For single-process production:** SQLite — real ACID, 41µs Save, no external dependency
3. **For distributed production:** PostgreSQL (real, not mock) — needs testcontainers for honest numbers
4. **For embedded/high-throughput:** Pebble — embedded KV, no SQL overhead, but lacks version checks

> **Note:** PostgreSQL mock benchmarks are unreliable for performance comparison.
> They measure `go-sqlmock` overhead, not actual database performance.
> A real PostgreSQL benchmark requires a running instance (testcontainers or CI service).
