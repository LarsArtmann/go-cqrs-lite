# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [4.1.0] - 2026-07-25

### Added
- **Recovery phase** (`Config.Recovery`): crash-recovery benchmark — closes the bundle mid-run, reopens via factory, measures replay time and recovered event count. For persistent backends (SQLite, Pebble), all events survive. For memory backends, documents zero recovery.
- **Replay-only mode** (`Config.ReplayOnly`): benchmarks existing event stores without writing. Discovers streams via batched `SeekableJournal.ReadFrom` (1000-event pages) to avoid OOM on large stores.
- **`benchtest.RunSuite`**: preset integration for `testing.B` — one-call benchmark suite with `b.ReportMetric` for custom metrics (throughput, latency p99, recovery time).
- **`ProfileAnalytical`**: OLAP-style workload profile (wide/shallow streams, 5 journal scans, 0.9 read ratio) for comparing cross-stream scan performance.
- **`Profile.JournalScans`** field: controls how many times `runJournalScans` iterates the journal (defaults to 1).
- **Real kv.Store projection handler**: `newKVCountingProjection` exercises real projection I/O (Get + Set per event) instead of a no-op counter, with graceful `kv.ErrNotFound` handling for first-seen keys.
- **Postgres benchmark backend**: `cmd/cqrs-bench --backend=postgres` with `POSTGRES_TEST_DSN` support (skips when unset).
- **CLI flags**: `--recovery` and `--replay` for `cmd/cqrs-bench`.

### Fixed
- **`readModelPhase` race condition**: `ctx.Err()` check between Set and Get phases prevents `kv.ErrNotFound` failures when Duration timeout interrupts mid-phase.
- **`recoveryPhase` context cancellation**: uses `context.WithoutCancel(parent)` so Load calls succeed even after the benchmark context expires.
- **Phase ctx.Err guards**: all phases (read, readModel, projection, write) now check `ctx.Err()` for graceful early-exit on Duration timeout, preventing wasted work and hung tests.
- **`discoverStreams` OOM risk**: replaced `Journal.ReadAll()` (loads all events) with batched `SeekableJournal.ReadFrom` (1000-event pages).

### Changed
- `newCountingProjection` and `newKVCountingProjection` no longer take a dead `aggIDs` parameter.
- `discoverStreams` refactored into `discoverFromSeekable` + `discoverFromJournal` + `collectStreamIDs` for clarity and reduced nesting.

## [4.0.0] - 2026-07-01

### Added
- Initial stable release: factory-driven benchmarking suite with latency percentiles, throughput, memory tracking, CPU monitoring, and event-size scaling analysis.
