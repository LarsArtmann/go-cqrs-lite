# ADR-0071: DuckDB CGo Introduction

|             |                                                                                          |
| ----------- | ---------------------------------------------------------------------------------------- |
| **Status**  | Accepted                                                                                 |
| **Date**    | 2026-07-28                                                                               |
| **Context** | go-cqrs-lite was pure-Go; DuckDB requires CGo (C++ static link)                          |

## Context

go-cqrs-lite has been a pure-Go project since inception. Every storage backend
(SQLite via modernc.org/sqlite, Pebble, Turso) compiles with `CGO_ENABLED=0`.
This is a deliberate design constraint: pure-Go means trivial cross-compilation,
static binaries, reproducible builds, and no system C toolchain dependency.

DuckDB is an embedded analytical (OLAP) SQL engine. It excels at read-heavy
analytical workloads, complex aggregations, window functions, and columnar
scans — workloads where SQLite's row-oriented engine struggles. The
[DuckDB Go driver](https://github.com/duckdb/duckdb-go) statically links the
DuckDB C++ engine (~30-50MB binary increase), which fundamentally requires CGo.

### Why DuckDB?

The library targets event-sourced CQRS systems. These systems generate large
volumes of events that are increasingly consumed for analytics (dashboards,
reporting, aggregations). DuckDB's columnar engine makes SQL view models
(`SQLViewModel`) dramatically faster for GROUP BY / window / analytical scans
than row-oriented SQLite, without requiring a separate analytical database
service.

### Alternatives Considered

1. **Pure-Go DuckDB client** — DuckDB does not have a pure-Go implementation.
   The engine itself is C++. No Go-native alternative exists.

2. **Arrow/Parquet for analytics** — A pure-Go Arrow + Parquet pipeline could
   serve analytical read models without CGo. This is tracked as a separate
   initiative (ROADMAP Phase 1: `storage/parquet`). However, it provides a
   file format, not a SQL engine — no ad-hoc GROUP BY, JOINs, or window
   functions.

3. **External analytical DB** — Require consumers to run ClickHouse, Postgres
   with columnar extensions, or a cloud warehouse. This violates the library
   principle: consumers import what they need; no external services required.

4. **Do nothing** — Leave DuckDB unsupported. Consumers who need analytical
   queries must use external tools or accept SQLite's row-store performance.

## Decision

**Introduce CGo — but isolate it in a single optional Go module.**

Create `stack/duckdb/` as an independent `go.mod` with `//go:build cgo` tags
on every file that touches the DuckDB driver. The DuckDB dialect lives in the
shared `storage/sql/` package (zero CGo — it's pure SQL strings), so all
existing SQL store code (SQLEventStore, SQLCommandStore, etc.) is reused.

### Isolation Strategy

| Layer                        | CGo? | Why                                                            |
| ---------------------------- | ---- | ------------------------------------------------------------- |
| `storage/sql/DuckDBDialect`  | No   | Pure SQL string generation — no driver import                 |
| `storage/DuckDBInitSchema()` | No   | Runs DDL via `database/sql` interface — no driver import      |
| `storage/NewDuckDBBackend()` | No   | Creates a `SQLBackend` with `DuckDBDialect{}` — no driver     |
| `stack/duckdb/drivers.go`    | Yes  | Blank-imports `github.com/duckdb/duckdb-go/v2` to register    |
| `stack/duckdb/preset.go`     | No   | Opens DB by driver name string `"duckdb"` — no direct import  |
| `stack/duckdb/*_cgo_test.go` | Yes  | Tests that exercise the actual DuckDB engine                  |

Consumers who never import `stack/duckdb` never pull in CGo. The rest of
go-cqrs-lite remains pure-Go and compiles with `CGO_ENABLED=0`.

### Build Pipeline

The Nix flake test/verify apps now set `CGO_ENABLED=1` so DuckDB tests run in
CI. The `build` app keeps `CGO_ENABLED=0` (default) to verify the pure-Go
build path. `stack/duckdb` is excluded from pure-Go builds via `//go:build cgo`
tags — its files are simply not compiled.

## Consequences

- **+** Analytical SQL backend without external services
- **+** CGo is opt-in: only consumers who import `stack/duckdb` pay the cost
- **+** All existing SQL store code reused via the Dialect interface
- **+** `CGO_ENABLED=0 go build ./...` still works for all other modules
- **-** Cross-compilation for `stack/duckdb` consumers requires a C cross-toolchain
- **-** Binary size increases ~30-50MB when DuckDB is linked
- **-** Nix build for `stack/duckdb` requires `pkgs.gcc` in the environment
- **-** CI must run two configurations: `CGO_ENABLED=0` (pure-Go path) and
       `CGO_ENABLED=1` (full DuckDB test path)

## References

- DuckDB Go driver: `github.com/duckdb/duckdb-go/v2` (DuckDB v1.5.5)
- Implementation: `stack/duckdb/` module, `storage/sql.DuckDBDialect`
- ROADMAP: Phase 2 (`storage/duckdb`) and Phase 3 (`stack/duckdb`) — shipped
