# Comprehensive Remaining Work Plan — All Open Items

> **Created:** 2026-06-16 · **Pareto-sorted** · **All micro-tasks ≤12 min**

## Methodology

Every open item was compiled from: TODO_LIST.md, FEATURES.md, ROADMAP.md, latest status
report (Top 25), all planning docs, and codebase verification. Items already done were
removed. Blocked items (external repos/owner decisions) are excluded. Deferred v2/v3
breaking changes are listed separately at the end.

---

## Summary

| Tier | Description                        | Tasks   | Est. Time  |
| ---- | ---------------------------------- | ------- | ---------- |
| 1    | Stale docs + quick wins (1% → 51%) | 14      | ~2.5 hr    |
| 2    | High-impact code (4% → 64%)        | 18      | ~3.5 hr    |
| 3    | Quality lift (20% → 80%)           | 22      | ~6 hr      |
| 4    | Infrastructure & CI                | 12      | ~6 hr      |
| 5    | Major features (long-term)         | 42      | ~40+ hr    |
| 6    | Deferred v2/v3 breaking changes    | 8       | Next major |
| —    | **Total actionable micro-tasks**   | **108** | **~58 hr** |

---

## Tier 1: Quick Wins (1% → 51%)

> Low effort, immediate value. Fixes stale documentation and adds small consumer-facing improvements.

| ID   | Micro-Task                                                                 | Module   | Impact             | Est |
| ---- | -------------------------------------------------------------------------- | -------- | ------------------ | --- |
| 1.1  | Update ROADMAP.md Sprint 7: mark all 5 items as ✅ done                    | docs     | Prevents confusion | 5m  |
| 1.2  | Update FEATURES.md known issues: mark 3 resolved items done                | docs     | Accuracy           | 5m  |
| 1.3  | Update FEATURES.md PLANNED section: mark query sentinel errors done        | docs     | Accuracy           | 5m  |
| 1.4  | Update ROADMAP.md Sprint 6: mark go-snaps as descoped (using AssertGolden) | docs     | Accuracy           | 5m  |
| 1.5  | Add `WithLogger(nil)` option to pebble stores (no-op logger)               | pebble   | DX                 | 10m |
| 1.6  | Document key prefix collision behavior in pebble/doc.go                    | pebble   | Safety docs        | 10m |
| 1.7  | Add `SnapshotStore()` method to SQLBackend facade                          | storage  | Completeness       | 10m |
| 1.8  | Add `CheckpointStore()` method to SQLBackend facade                        | storage  | Completeness       | 10m |
| 1.9  | Add `Close()` method to SQLBackend that closes all stores                  | storage  | Lifecycle safety   | 10m |
| 1.10 | Write ADR for CBOR envelope format (pebble on-disk format)                 | docs/adr | Consumer trust     | 12m |
| 1.11 | Write ADR for EventStore.Close() vs SnapshotStore.Close() asymmetry        | docs/adr | Clarity            | 10m |
| 1.12 | Document pebble `Open()` options in doc.go (shared DB pattern)             | pebble   | DX                 | 10m |
| 1.13 | Update ROADMAP.md Sprint 4: verify/mark Docker CI status                   | docs     | Accuracy           | 5m  |
| 1.14 | Update ROADMAP.md Sprint 5: verify/mark Playwright status                  | docs     | Accuracy           | 5m  |

---

## Tier 2: High-Impact Code (4% → 64%)

> Missing OTel tracing, module boundary fixes, and the PebbleBackend facade.

| ID   | Micro-Task                                                                                   | Module  | Impact                    | Est |
| ---- | -------------------------------------------------------------------------------------------- | ------- | ------------------------- | --- |
| 2.1  | Add OTel span to pebble `EventStore.Save()`                                                  | pebble  | Observability             | 12m |
| 2.2  | Add OTel span to pebble `EventStore.Load()` + `LoadFromVersion()`                            | pebble  | Observability             | 12m |
| 2.3  | Add OTel span to pebble `EventStore.ReadAll()` + `ReadFrom()`                                | pebble  | Observability             | 12m |
| 2.4  | Add OTel span to pebble `EventStore.LoadToVersion()` + `LoadToTimestamp()`                   | pebble  | Observability             | 10m |
| 2.5  | Create `PebbleBackend` struct with `Open()` constructor                                      | pebble  | Consumer DX               | 12m |
| 2.6  | Add `EventStore()` method to PebbleBackend                                                   | pebble  | Consumer DX               | 5m  |
| 2.7  | Add `SnapshotStore()` + `CheckpointStore()` to PebbleBackend                                 | pebble  | Consumer DX               | 10m |
| 2.8  | Add `Close()` to PebbleBackend that closes shared DB                                         | pebble  | Lifecycle                 | 8m  |
| 2.9  | Write `PebbleBackend` integration test (full stack)                                          | pebble  | Safety                    | 12m |
| 2.10 | Audit command/ type aliases — document as intentional in doc.go                              | command | Clarity                   | 10m |
| 2.11 | Audit query/ type aliases — document as intentional in doc.go                                | query   | Clarity                   | 10m |
| 2.12 | Add `PebbleBackend` usage example to pebble/doc.go                                           | pebble  | DX                        | 10m |
| 2.13 | Add `PebbleBackend` pattern to AGENTS.md Key Patterns                                        | docs    | DX                        | 10m |
| 2.14 | Add pebble EventStore OTel to CHANGELOG.md                                                   | docs    | Release notes             | 5m  |
| 2.15 | Benchmark pebble Save before/after OTel overhead                                             | pebble  | Verify no perf regression | 12m |
| 2.16 | Add `Replace directive CI check` script (verify all go.mod replace directives match go.work) | ci      | Safety                    | 12m |
| 2.17 | Wire replace CI check into GitHub Actions workflow                                           | ci      | Safety                    | 10m |
| 2.18 | Document the replace directive CI check in AGENTS.md                                         | docs    | Knowledge                 | 5m  |

---

## Tier 3: Quality Lift (20% → 80%)

> Test coverage, golden tests, integration tests, and fuzz tests.

| ID   | Micro-Task                                                              | Module      | Impact            | Est |
| ---- | ----------------------------------------------------------------------- | ----------- | ----------------- | --- |
| 3.1  | Identify uncovered pebble code paths (go test -coverprofile)            | pebble      | Targeting         | 10m |
| 3.2  | Add tests for pebble serialization error branches                       | pebble      | Coverage          | 12m |
| 3.3  | Add tests for pebble nil-db / closed-db error paths                     | pebble      | Coverage          | 12m |
| 3.4  | Add tests for pebble iteration edge cases (empty, single, end)          | pebble      | Coverage          | 12m |
| 3.5  | Add golden test for pebble CBOR envelope encoding                       | pebble      | Regression safety | 12m |
| 3.6  | Add golden test for pebble snapshot serialization                       | pebble      | Regression safety | 10m |
| 3.7  | Add golden test for pebble checkpoint serialization                     | pebble      | Regression safety | 8m  |
| 3.8  | Add fuzz test for pebble snapshot encode/decode roundtrip               | pebble      | Robustness        | 12m |
| 3.9  | Add fuzz test for pebble checkpoint encode/decode roundtrip             | pebble      | Robustness        | 10m |
| 3.10 | Write integration test: pebble EventStore + projection Runner (replay)  | integration | E2E safety        | 12m |
| 3.11 | Write integration test: pebble EventStore + projection Runner (live)    | integration | E2E safety        | 12m |
| 3.12 | Write integration test: pebble SnapshotStore + decider Repository       | integration | E2E safety        | 12m |
| 3.13 | Add MemorySnapshotStore golden test (baseline for pebble comparison)    | memory      | Regression safety | 10m |
| 3.14 | Benchmark pebble SnapshotStore vs SQLSnapshotStore                      | pebble      | Data-driven       | 12m |
| 3.15 | Benchmark pebble EventStore vs SQLEventStore (Save 100 events)          | pebble      | Data-driven       | 12m |
| 3.16 | Add reactive CommandBus example to command/doc.go                       | command     | DX                | 10m |
| 3.17 | Verify golden test drift in codec/middleware (run tests, fix if needed) | codec       | Correctness       | 10m |
| 3.18 | Add `ErrBackupNotImplemented` documentation to example/todo             | example     | Consumer clarity  | 5m  |
| 3.19 | Add `streaming` build tag stub with documentation                       | event       | Future-ready      | 12m |
| 3.20 | Add `zstd` build tag stub with documentation                            | codec       | Future-ready      | 12m |
| 3.21 | Verify all ADR README index entries match actual ADR files              | docs        | Accuracy          | 5m  |
| 3.22 | Add ADR-0019 placeholder for missing gap (0019 skipped in sequence)     | docs        | Completeness      | 5m  |

---

## Tier 4: Infrastructure & CI

> Docker, Playwright, PostgreSQL integration testing.

| ID   | Micro-Task                                                  | Module  | Impact              | Est |
| ---- | ----------------------------------------------------------- | ------- | ------------------- | --- |
| 4.1  | Write multi-arch Dockerfile (linux amd64 + arm64)           | docker  | Packaging           | 12m |
| 4.2  | Add Docker build CI step to ci.yml                          | ci      | Release safety      | 12m |
| 4.3  | Test Docker build locally for example/user                  | docker  | Verify              | 10m |
| 4.4  | Set up Playwright in example/user/                          | example | E2E testing         | 12m |
| 4.5  | Write Playwright E2E test: health endpoint                  | example | E2E coverage        | 12m |
| 4.6  | Write Playwright E2E test: command→event→query flow         | example | E2E coverage        | 12m |
| 4.7  | Add Playwright CI step to ci.yml                            | ci      | E2E in CI           | 10m |
| 4.8  | Add testcontainers-go dependency to storage/go.mod          | storage | Integration testing | 10m |
| 4.9  | Write PostgreSQL integration test: SQLEventStore CRUD       | storage | Real DB testing     | 12m |
| 4.10 | Write PostgreSQL integration test: SQLCommandStore journal  | storage | Real DB testing     | 12m |
| 4.11 | Write PostgreSQL integration test: SQLSnapshotStore         | storage | Real DB testing     | 12m |
| 4.12 | Add PostgreSQL integration test CI step (service container) | ci      | CI completeness     | 12m |

---

## Tier 5: Major Features (Long-Term)

> Each item is a significant feature requiring its own design + implementation cycle.

| ID   | Feature                                                                     | Module       | Impact              | Est  |
| ---- | --------------------------------------------------------------------------- | ------------ | ------------------- | ---- |
| 5.1  | **Outbox pattern** — design + implement reliable at-least-once publishing   | new module   | HIGH (reliability)  | 8hr  |
| 5.2  | **Schema registry** — JSON Schema validation middleware for events          | new module   | HIGH (correctness)  | 6hr  |
| 5.3  | **Distributed checkpointing** — multi-instance projection coordination      | projection   | MED (scale)         | 6hr  |
| 5.4  | **Reactive CommandBus** — `ro.Subject[Command]` reactive streams            | command      | MED (reactive)      | 4hr  |
| 5.5  | **Reactive QueryBus** — `ro.Subject[Query]` reactive streams                | query        | MED (reactive)      | 4hr  |
| 5.6  | **cqrs-gen v2** — code generator with struct tag scanning                   | cmd/cqrs-gen | MED (DX)            | 8hr  |
| 5.7  | **gRPC transport adapter** — gRPC wrapper for command/query dispatch        | new module   | MED (interop)       | 6hr  |
| 5.8  | **NATS/Redis Stream adapter** — message broker integration                  | new module   | MED (interop)       | 6hr  |
| 5.9  | **Prometheus metrics exporter** — replace custom MetricsRecorder            | middleware   | MED (observability) | 4hr  |
| 5.10 | **Structured logging middleware** — configurable slog levels                | middleware   | MED (observability) | 4hr  |
| 5.11 | **Distributed tracing propagation** — span context across module boundaries | otel         | MED (observability) | 4hr  |
| 5.12 | **Built-in pprof endpoints** — profiling HTTP handler                       | middleware   | LOW (debugging)     | 2hr  |
| 5.13 | **Documentation site** — Docusaurus/MkDocs/Hugo                             | docs         | MED (DX)            | 8hr  |
| 5.14 | **jsonv2 codec experiment** — behind build tag                              | codec        | LOW (experimental)  | 4hr  |
| 5.15 | **Arena allocation experiment** — event module behind build tag             | event        | LOW (experimental)  | 4hr  |
| 5.16 | **SIMD-accelerated serialization** — Go experiment                          | event        | LOW (experimental)  | 8hr  |
| 5.17 | **Streaming event reads** — `StreamLoader` without materializing full slice | event        | MED (scale)         | 6hr  |
| 5.18 | **Saga module** — orchestrated multi-step transactions                      | new module   | HIGH (reliability)  | 12hr |
| 5.19 | **WASM compilation target** — decider module for browser/edge               | decider      | LOW (experimental)  | 8hr  |

---

## Tier 6: Deferred Breaking Changes

> These require a major version bump. Listed for completeness — not actionable now.

### v2 (Next Major)

| ID  | Change                                                   | ADR      | Impact                  |
| --- | -------------------------------------------------------- | -------- | ----------------------- |
| 6.1 | Remove `io.Closer` from core interfaces                  | ADR-0010 | Cleaner interfaces      |
| 6.2 | Split `event.Store` into Writer/Reader/Deleter           | ADR-0010 | ISP compliance          |
| 6.3 | Add global `TransactionID` branded type                  | —        | Cross-aggregate tracing |
| 6.4 | Make event Core truly immutable (deep copy opts pointer) | —        | Correctness             |
| 6.5 | Move HTTP code out of middleware → transport/ module     | —        | Separation of concerns  |
| 6.6 | Fix `query.Handler` returns `any` → `TypedHandler[T]`    | —        | Type safety             |

### v3 (Future Major)

| ID  | Change                                                         | Impact         |
| --- | -------------------------------------------------------------- | -------------- |
| 6.7 | Split `catalog.Message` into Message + MessageMeta (17 fields) | Structured API |
| 6.8 | Split `catalog.Service` into Service + ServiceMeta (16 fields) | Structured API |

---

## Descoped / Declined (For Reference)

| Item                                                  | Reason                                                                  |
| ----------------------------------------------------- | ----------------------------------------------------------------------- |
| `kv/` interface module                                | No second KV backend exists; thin wrapper adds complexity without value |
| Field-level encryption (`encryption/fieldlevel/`)     | No consumer demand; would add API surface                               |
| Turso indexing: comparison report generator           | Indexing advisor covers core need                                       |
| Turso indexing: hooks API (`turso.WithIndexingHooks`) | No consumer demand                                                      |
| Turso indexing: health check integration              | Would couple turso to listing/ unnecessarily                            |
| go-snaps across all modules                           | Using `eventtest.AssertGolden` instead — already integrated             |
| GraphQL query adapter                                 | Framework-level concern, not library scope                              |
| `WithNoCopy()` optimization                           | Risky for a library, deferred indefinitely                              |

---

## Blocked (External Dependencies)

| Item                                                        | Blocker              |
| ----------------------------------------------------------- | -------------------- |
| Move example/todo to own repository                         | Manual repo creation |
| Remove cockroachdb/errors from go-localsync                 | Different repo       |
| Create go-branded-id v0.2.0                                 | Different repo       |
| Extract shared golangci.yml into larsartmann/library-policy | Different repo       |
| Change LICENSE from proprietary to MIT/Apache-2.0           | Owner decision       |
| CI billing (GitHub Actions)                                 | Payment issue        |

---

## Safety Rules

1. **Library, not framework** — every change must preserve consumer import paths
2. **Additive only** — no breaking changes until v2/v3
3. **Build must pass** — `nix run .#build` after each task group
4. **Tests must pass** — `nix run .#test` after each task group
5. **Lint must pass** — `nix run .#lint` after each task group
6. **Format before lint** — `nix fmt` before placing `//nolint` directives
