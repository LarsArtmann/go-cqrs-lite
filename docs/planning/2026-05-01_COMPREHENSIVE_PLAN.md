# Comprehensive Task Plan

**Created:** 2026-05-01
**Format:** Max 12 min per task, sorted by importance → impact → effort → customer-value

---

## Triage Summary

| Category | Count | Est. Total |
|----------|-------|------------|
| P0 — Bugs & Data Loss | 1 | ~12 min |
| P1 — Stale Docs Cleanup | 1 | ~10 min |
| P2 — Code Quality & Size | 4 | ~45 min |
| P3 — Storage Module Hardening | 5 | ~55 min |
| P4 — Projection & Upcaster Gaps | 4 | ~40 min |
| P5 — Documentation & DX | 5 | ~50 min |
| P6 — New Features | 6 | ~70 min |
| **Total** | **26** | **~4.5 hrs** |

---

## P0 — Bugs & Data Loss

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 1 | Delete orphaned `/user` binary at repo root + add `/user` to `.gitignore` | `.gitignore`, `user` (delete) | 5 min | 9.7 MB committed binary, not gitignored |

## P1 — Stale Docs Cleanup

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 2 | Prune TODO_LIST.md: remove 6 stale items (metadata discarded, codec unused, nowFunc unused, 0% coverage, asyncapi key collision, upcaster >=) | `TODO_LIST.md` | 10 min | Stale TODOs erode trust and waste time |

## P2 — Code Quality & Size

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 3 | Refactor `event.NewEvent` (66→30 lines): extract `validateEventParams` helper | `core/event/event.go` | 12 min | 2× function size limit |
| 4 | Refactor `asyncapi.addMessage` (55→30 lines): extract channel/operation builders | `catalog/asyncapi/exporter.go` | 12 min | Readability |
| 5 | Fix `toDotAddress` number handling: "Get3DView" → "get.3d.view" not "get.3.d.view" | `catalog/asyncapi/exporter.go` | 8 min | Incorrect output |
| 6 | Remove `storage/event_store.go` `Close()` closing caller-owned `*sql.DB` — document ownership contract or accept interface | `storage/event_store.go` | 12 min | Ownership surprise: caller doesn't expect Close() to close their DB |

## P3 — Storage Module Hardening

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 7 | Add storage error path tests: BeginTx failure, version query error, insert error, commit error | `storage/event_store_test.go` | 12 min | 79.8% → ~90% |
| 8 | Add `Scan()` error tests: parse aggregate ID failure, parse event ID failure, row scan failure, rows.Err() | `storage/event_store_test.go` | 10 min | Error path coverage |
| 9 | Add `Schema()` DDL test: verify returned SQL contains expected table/index definitions | `storage/event_store_test.go` | 5 min | 0% → covered |
| 10 | Add `Delete()` error path test | `storage/event_store_test.go` | 5 min | Error path coverage |
| 11 | Add SQL injection security test: verify aggregateID/aggregateType values are parameterized (not interpolated) | `storage/event_store_test.go` | 10 min | Security baseline |

## P4 — Projection & Upcaster Gaps

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 12 | Add duplicate projection detection in `InMemoryRunner.Register` | `core/event/runner.go` | 8 min | Silent duplicate registration is a bug source |
| 13 | Add `UpcasterRegistry` cycle detection: detect A→B→A upcast chains | `core/event/upcaster_registry.go` | 12 min | Prevent infinite loops |
| 14 | Document `InMemoryRunner` fail-fast behavior in godoc | `core/event/runner.go` | 5 min | Users must know error handling semantics |
| 15 | Document `MemoryBus.Publish` RLock behavior in godoc | `memory/bus.go` | 5 min | Documented design constraint |

## P5 — Documentation & DX

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 16 | Update `FEATURES.md`: remove stale "BROKEN" storage status (now 79.8% coverage, metadata works), update upcaster section | `FEATURES.md` | 10 min | Accurate module maturity |
| 17 | Update `AGENTS.md`: update storage coverage, remove stale known issues | `AGENTS.md` | 10 min | Accurate project reference |
| 18 | Add Go doc `Example*` test functions for `command`, `event`, `query`, `id` packages | `core/*/example_test.go` | 12 min | pkg.go.dev discoverability |
| 19 | Write getting-started guide | `docs/getting-started.md` | 12 min | Onboarding |
| 20 | Add E2E throughput benchmarks: commands/sec, events/sec | `integration/event/bench_test.go` | 10 min | Performance baseline |

## P6 — New Features

| # | Task | Files | Est. | Why |
|---|------|-------|------|-----|
| 21 | Add SQL SnapshotStore implementation (PostgreSQL-backed) | `storage/snapshot.go` | 12 min | Production snapshot persistence |
| 22 | Add SQL CheckpointStore implementation (PostgreSQL-backed) | `storage/checkpoint.go` | 10 min | Production projection checkpointing |
| 23 | Add outbox background publisher: goroutine that polls outbox and publishes to bus | `core/event/outbox_publisher.go` | 12 min | Transactional outbox pattern completion |
| 24 | Add saga/process manager design doc | `docs/planning/SAGA_DESIGN.md` | 10 min | Architecture planning |
| 25 | Add fuzz tests for event creation, ID parsing, schema reflection, `DecodePayload`, upcaster chain | `core/*/fuzz_test.go` | 12 min | Robustness |
| 26 | Tag `v0.1.0-alpha` releases for core, memory, catalog, middleware modules | git tags | 5 min | Discoverability & versioning |
