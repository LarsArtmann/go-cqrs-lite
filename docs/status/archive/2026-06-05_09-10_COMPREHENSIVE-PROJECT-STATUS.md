# Comprehensive Status Report

> **Date:** 2026-06-05 09:10 CEST
> **Branch:** master
> **Head:** `46347e9a` — 49 commits since v2.1.0
> **Scope:** Full project audit — all 30 modules

---

## Executive Summary

go-cqrs-lite is a **healthy, production-quality library** with 30 Go modules, 24,788 production LoC, 45,033 test LoC (1.8:1 test-to-prod ratio), zero lint issues, zero race conditions, and comprehensive pkg.go.dev documentation. The codebase is in the strongest state in its history.

**Releases:** v2.0.0 (2026-06-01) → v2.1.0 (2026-06-03) → HEAD (unreleased, 49 post-v2.1.0 commits)

**Build/Test/Lint:**

| Check                    | Result                                  |
| ------------------------ | --------------------------------------- |
| `go build ./...`         | ✅ PASS (all 30 modules)                |
| `go test ./... -count=1` | ✅ PASS (all 30 modules)                |
| `go vet ./...`           | ✅ ZERO issues                          |
| `go test -race`          | ✅ PASS (core modules, zero data races) |
| `nix run .#lint`         | ✅ ZERO issues (all 21 library modules) |
| `nix run .#build`        | ✅ PASS                                 |

---

## a) FULLY DONE

### Core CQRS Infrastructure

| Feature              | Module        | Status | Detail                                                                                                                                                                                                                                    |
| -------------------- | ------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Event System         | `event/`      | ✅     | NewEvent, ImmutableEvent, Store (Sink+Source), Journal, SeekableJournal, BackwardsSource, 19 options, metadata, context enricher, defensive copies, Clone, tombstones, time-travel, Projection/BatchProjection interfaces, error taxonomy |
| Command Dispatch     | `command/`    | ✅     | Dispatcher, Handler, TypedHandler[T], Middleware, CommandStore (Sink+Source), metadata, catalog introspection                                                                                                                             |
| Query Dispatch       | `query/`      | ✅     | Dispatcher, TypedHandler[T], DispatchTyped[T], Pagination, PaginatedResult[T], catalog introspection                                                                                                                                      |
| Decider (Aggregates) | `decider/`    | ✅     | Decider[State] pure-function pattern, Repository[State], Execute/Load/LoadAtVersion/LoadAtTime, crash recovery                                                                                                                            |
| Branded IDs          | `id/`         | ✅     | id.Of[T] = cbid.ID[T, ulid.ULID], 8 built-in types, JSON/SQL/binary/text, compile-time type safety                                                                                                                                        |
| Generic Dispatcher   | `dispatcher/` | ✅     | Generic Dispatcher[H, M], LifecycleMixin, ErrDispatcherClosed sentinel                                                                                                                                                                    |
| Codec                | `codec/`      | ✅     | JSONCodec, RawCodec, Codec interface                                                                                                                                                                                                      |

### Storage Backends

| Backend            | Module       | Status | Detail                                                                                                                 |
| ------------------ | ------------ | ------ | ---------------------------------------------------------------------------------------------------------------------- |
| In-Memory          | `memory/`    | ✅     | MemoryStore, MemoryBus, MemorySnapshotStore, MemoryCheckpointStore, MemoryCommandStore — thread-safe, defensive copies |
| SQL (PG/SQLite)    | `storage/`   | ✅     | SQLEventStore, SQLSnapshotStore, SQLCheckpointStore, SQLCommandStore — dialect abstraction, optimistic concurrency     |
| Embedded KV        | `pebble/`    | ✅     | PebbleEventStore — scoped to aggregate, time-travel, version bounds, early termination                                 |
| Turso (LibSQL)     | `turso/`     | ✅     | Open/OpenInMemory/OpenSync, event store, snapshot store, checkpoint store                                              |
| Watermill Protocol | `watermill/` | ✅     | PublisherAdapter, SubscriberAdapter — bidirectional event↔message translation, 15+ metadata keys                       |

### Cross-Cutting Concerns

| Concern              | Module                  | Status | Detail                                                                                                                               |
| -------------------- | ----------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------ |
| Projection Runner    | `projection/`           | ✅     | Replay+live, checkpoint tracking, Builder pattern, On[T] typed handlers, dead letter, retry with backoff                             |
| Schema Evolution     | `schema/`               | ✅     | Upcaster, UpcasterRegistry, VersionedStore, cycle detection                                                                          |
| Snapshot Persistence | `snapshot/`             | ✅     | SnapshotSink/Source/Store ISP split, EveryNEvents strategy                                                                           |
| Event Signing        | `signing/`              | ✅     | HMAC-SHA256, Ed25519, multisig, sign/verify middleware                                                                               |
| Middleware           | `middleware/`           | ✅     | 24 factories: logging, retry, recovery, validation, metrics, OTel tracing+metrics (command+event+query)                              |
| OTel Helpers         | `otel/`                 | ✅     | Tracer, Meter, Spans, Attributes (opt-in, no-op when unconfigured)                                                                   |
| Aggregate Listing    | `listing/`              | ✅     | AggregateReader, ListBuilder, InMemoryAggregateReader, SQLAggregateReader, AggregateProjection, StatusMiddleware, tombstone policies |
| OpenTelemetry        | `middleware/` + `otel/` | ✅     | Tracing and metrics middleware for all three domains                                                                                 |

### Auto-Documentation

| Feature      | Module                 | Status | Detail                                       |
| ------------ | ---------------------- | ------ | -------------------------------------------- |
| Registry     | `catalog/`             | ✅     | Registry, SchemaFromType[T], Build, Validate |
| AsyncAPI 3.0 | `catalog/asyncapi`     | ✅     | Full spec generation                         |
| D2 Diagrams  | `catalog/d2`           | ✅     | Architecture diagram export                  |
| EventCatalog | `catalog/eventcatalog` | ✅     | EventCatalog.com format                      |
| OpenAPI      | `catalog/openapi`      | ✅     | REST API documentation                       |
| Doc Server   | `catalog/docserver`    | ✅     | In-process HTTP server                       |
| JSON Schema  | `catalog/schema`       | ✅     | Reflection engine                            |

### Code Generation & Tooling

| Tool          | Module              | Status | Detail                                         |
| ------------- | ------------------- | ------ | ---------------------------------------------- |
| cqrs-gen      | `cmd/cqrs-gen`      | ✅     | AST-based typed handler generation             |
| API Stability | `cmd/api-stability` | ✅     | Exported symbol comparison against golden file |

### Package Documentation (pkg.go.dev)

| Deliverable                 | Count                                  | Status      |
| --------------------------- | -------------------------------------- | ----------- |
| `doc.go` files              | 11 of 11 needed                        | ✅ ALL DONE |
| `README.md` files (library) | 10 of 10 needed                        | ✅ ALL DONE |
| `README.md` files (example) | 6 of 6 needed                          | ✅ ALL DONE |
| `example_test.go`           | 9 modules                              | ✅ ALL DONE |
| `errors.go` sentinel files  | 4 new (schema, watermill, id, command) | ✅ DONE     |

### Architecture Decision Records

| ADR                                    | Status                             |
| -------------------------------------- | ---------------------------------- |
| 0001: Decider over Aggregate           | ✅ Adopted                         |
| 0002: Error Taxonomy                   | ✅ Adopted                         |
| 0003: Multi-Module Monorepo            | ✅ Adopted                         |
| 0004: Saga Process Manager             | ✅ Superseded (projection+command) |
| 0006: Sink/Source Split                | ✅ Adopted                         |
| 0007: gopls Workspace Workaround       | ✅ Active                          |
| 0008: Typed Handler Signature          | ✅ Adopted                         |
| 0009: Pebble Scope Event Store Only    | ✅ Adopted                         |
| 0010: Remove io.Closer from Interfaces | 📐 Proposed (v3)                   |
| 0011: Unify ErrDispatcherClosed        | 📐 Proposed (v3)                   |
| 0012: Split Catalog Modules            | 📐 Proposed (v3)                   |

### Examples

| Example                | Status | Uses                                                       |
| ---------------------- | ------ | ---------------------------------------------------------- |
| `example/user`         | ✅     | Event sourcing lifecycle with projection.Runner            |
| `example/todo`         | ✅     | Full HTTP API with Pebble + MemoryStore, projection.Runner |
| `example/storage`      | ✅     | SQL event store with embedded SQLite                       |
| `example/projection`   | ✅     | Replay+live projection with checkpoints                    |
| `example/saga-pattern` | ✅     | Multi-step saga via projection + command dispatch          |
| `example/listing`      | ✅     | Aggregate listing with tombstone detection                 |

### Performance Benchmarks (Session 142)

| Module         | Operation     |  ns/op |   B/op | allocs/op |
| -------------- | ------------- | -----: | -----: | --------: |
| event          | NewEvent      |    201 |    384 |         3 |
| event          | DecodePayload |    419 |    560 |        10 |
| id             | New           |    100 |     16 |         1 |
| id             | Parse         |     17 |      0 |         0 |
| command        | New           |     50 |    208 |         2 |
| memory         | Store Save    |    583 |    736 |         9 |
| memory         | Bus Publish   |     66 |     48 |         3 |
| signing        | HMAC Sign     |    662 |    864 |        12 |
| signing        | Ed25519 Sign  | 13,486 |    416 |         7 |
| storage/SQLite | Save          | 41,042 |  4,080 |        92 |
| storage/SQLite | Load          | 48,505 | 20,233 |       554 |

---

## b) PARTIALLY DONE

### turso/ Module — 28.6% Coverage (Target: 80%+)

Only basic happy-path tests exist. Missing:

- Error paths: nil DB, closed store, invalid args
- SnapshotStore: version-aware load, empty result, concurrent access
- CheckpointStore: overwrite, concurrent reads
- SyncDB: real sync scenarios
- `t.Parallel()` not used

### command/ Module — Module Boundary Violations

- `command/aggregate_ref.go` re-exports `event.AggregateType`, `event.AggregateRef`, `event.ParseAggregateType`, `event.NewAggregateRef`
- `command.Metadata` mirrors `event.Metadata` fields (split brain)
- Documented in ADR-0010/0011 context but not yet resolved

### Middleware — 3x Duplication (~500 lines)

Every middleware factory is triplicated for command/event/query. Bug fixes must be applied 3 times. O(N×M) growth where N=concerns, M=domains. Noted in multiple planning docs but no refactor done.

### Catalog — 9,319 LoC Monolith

The largest module bundles core registry + 5 independent exporters. ADR-0012 proposes splitting into 5 modules but no code changes made. All exporters share no code beyond registry types.

### storage/sql — Test Coverage Gaps

- `storage/sql/helpers.go`: shared SQL helpers used by all backends, only tested via integration
- `storage/sql_aggregate_reader.go:ListWithStatus`: 115-line function, longest in codebase, needs decomposition

### Reactive Extensions (samber/ro)

- `event.EventBus`, `command.CommandBus`, `query.QueryBus` exist as reactive subjects
- NOT wired into dispatchers — consumers must use bus.Subscribe directly
- Examples (user, todo) use projection.Runner instead of raw reactive API

---

## c) NOT STARTED

### Query Store

No `QueryStore` / `QuerySink` / `QuerySource` interfaces exist. Query dispatcher only dispatches — no persistence. Command got its Store in this cycle; Query has none.

### Outbox Pattern

Documented in multiple planning docs (CONTEXT.md, SAGA_DESIGN.md) but no implementation. The outbox table was removed in Session 152-153. Currently no reliable at-least-once event publishing guarantee.

### Schema Registry

JSON Schema middleware for runtime event validation mentioned in docs/planning but no code exists.

### Catalog Diff / Breaking-Change Detection

Mentioned in TODO_LIST.md as `[FUTURE]`. No implementation.

### PostgreSQL Integration Tests

`[BLOCKED]` in TODO_LIST — requires testcontainers/Docker setup. Only SQLite is tested in CI.

### v3 Breaking Changes (ADRs 0010-0012)

All three ADRs are "Proposed" status — no implementation:

- Remove `io.Closer` from 9 core interfaces
- Unify `ErrDispatcherClosed` into single sentinel
- Split catalog into 5 modules

### TransactionID Branded Type

Mentioned in TODO_LIST as `[v2]` (now would be v3). Global branded type for cross-aggregate consistency.

### High-Level Test Utilities

AggregateTester, ProjectionTester, BusTester fluent APIs — `[FUTURE]` in TODO_LIST.

---

## d) TOTALLY FUCKED UP

### Untracked Binaries in Working Tree

Two binary files exist in the working tree that should never be committed:

- `saga-pattern` (34KB binary)
- `example/todo/cmd/api/api` (106KB binary)

These were likely compiled locally and forgotten. They are correctly untracked (not staged) but should be added to `.gitignore`.

### go.work.sum Drift

`go.work.sum` has unstaged changes — workspace checksum file not in sync with current module state. Minor but indicates `go mod tidy` hasn't been run workspace-wide since last dependency change.

### 3 Untracked doc.go Files

```
codec/doc.go    (untracked)
event/doc.go    (untracked)
middleware/doc.go (untracked)
```

These were created in a previous session but never committed. They exist on disk but are invisible to consumers cloning the repo.

### 5 errors.go Files Still Missing

Modules that create errors inline without centralized `errors.go`:

- `snapshot/errors.go` — 4 `fmt.Errorf` calls scattered
- `catalog/errors.go` — 31 `fmt.Errorf` calls, largest module
- `storage/errors.go` — errors in `sql/` subpackage but not storage root
- `otel/errors.go` — no errors.go at all
- `listing/errors.go` — 4 `fmt.Errorf` calls scattered

---

## e) WHAT WE SHOULD IMPROVE

### Architecture & Design

1. **Middleware triplication** — The biggest code health issue. 500 lines of duplication, O(N×M) growth. Generic middleware interface could cut this to ~170 lines.

2. **command↔event coupling** — command re-exports 4 event types and mirrors Metadata. This is a module boundary violation that makes independent versioning harder.

3. **io.Closer in 9 interfaces** — Forces every consumer to implement `Close() error` even for stateless adapters. ADR-0010 proposed but deferred to v3.

4. **ErrHandlerNotFound × 3** — Three separate sentinels in dispatcher/command/query with identical purpose. ADR-0011 proposed unification.

5. **Catalog monolith** — 9,319 LoC in a single module. Four exporters share zero code. ADR-0012 proposes splitting.

### Coverage & Testing

6. **turso 28.6%** — Most urgent coverage gap. Production module with barely tested error paths.

7. **storage/sql helpers 0%** — Shared SQL helpers untested in isolation.

8. **No PostgreSQL tests** — Only SQLite tested. PG dialect code path has zero integration coverage.

9. **Benchmark regression pipeline** — Benchmarks exist but no automated comparison in CI.

### Consumer Experience

10. **No quickstart guide** — Individual modules have doc.go but no single "Getting Started" document showing the full CQRS stack composition.

11. **Event README stale references** — Still mentions old `go-cqrs-lite/core` in some places.

12. **Reactive API not wired** — EventBus/CommandBus/QueryBus exist but aren't the default dispatch path. Consumers must discover and wire them manually.

### Operational

13. **Binary artifacts in working tree** — `saga-pattern` and `example/todo/cmd/api/api` compiled binaries should be gitignored.

14. **Untracked files from previous sessions** — 3 doc.go files, 3 example_test.go files committed in earlier sessions but some still showing as modified or have siblings that are untracked.

15. **go.work.sum drift** — Workspace checksum file not synced.

---

## f) Top 25 Things We Should Get Done Next

### Priority 1: Fix What's Broken (Do Immediately)

| #   | Task                                                                                  | Module    | Est | Impact                                  |
| --- | ------------------------------------------------------------------------------------- | --------- | --- | --------------------------------------- |
| 1   | **Delete or gitignore binary artifacts** (`saga-pattern`, `example/todo/cmd/api/api`) | root      | 2m  | Prevents accidental binary commits      |
| 2   | **Commit untracked doc.go files** (codec, event, middleware)                          | 3 modules | 2m  | pkg.go.dev docs incomplete without them |
| 3   | **Run `go mod tidy` workspace-wide** + commit go.work.sum                             | all       | 5m  | Dependency hygiene                      |
| 4   | **Fix event/README.md stale references** to old `core` package                        | event     | 5m  | Consumer-facing docs correctness        |

### Priority 2: Close Coverage Gaps (High Impact)

| #   | Task                                                                                   | Module  | Est | Impact                          |
| --- | -------------------------------------------------------------------------------------- | ------- | --- | ------------------------------- |
| 5   | **Turso tests: Save+Load edge cases, AppendBatch, LoadFromVersion, concurrent access** | turso   | 12m | 28.6% → 80%                     |
| 6   | **Turso tests: error paths (nil DB, closed store, invalid args)**                      | turso   | 10m | Error paths completely untested |
| 7   | **Turso tests: SnapshotStore + CheckpointStore full coverage**                         | turso   | 10m | Only SaveAndLoad tested         |
| 8   | **Add storage/sql/helpers_test.go** — shared SQL helper coverage                       | storage | 10m | 0% → meaningful                 |
| 9   | **Add pebble edge-case tests** — concurrent Save, Close-then-access                    | pebble  | 10m | 86.6% → 90%+                    |

### Priority 3: Error Hygiene (Consistency)

| #   | Task                                                             | Module   | Est | Impact                    |
| --- | ---------------------------------------------------------------- | -------- | --- | ------------------------- |
| 10  | **Create `snapshot/errors.go`** — consolidate 4 fmt.Errorf calls | snapshot | 5m  | Consistent error matching |
| 11  | **Create `catalog/errors.go`** — consolidate 31 fmt.Errorf calls | catalog  | 12m | Largest gap               |
| 12  | **Create `listing/errors.go`** — consolidate 4 fmt.Errorf calls  | listing  | 5m  | Quick win                 |
| 13  | **Create `storage/errors.go`** — re-export sql/errors.go         | storage  | 5m  | Centralized API surface   |

### Priority 4: Code Health

| #   | Task                                                                                     | Module    | Est | Impact                       |
| --- | ---------------------------------------------------------------------------------------- | --------- | --- | ---------------------------- |
| 14  | **Decompose `storage/sql_aggregate_reader.go:ListWithStatus`** (115L → 3 funcs)          | storage   | 12m | Longest function in codebase |
| 15  | **Decompose `watermill/protocol.go:messageToEvent`** (86L → 4 funcs)                     | watermill | 12m | Second longest function      |
| 16  | **Decompose `storage/event_store.go:Save`** (55L → 2 funcs)                              | storage   | 10m | Core write path              |
| 17  | **Decompose `signing/multisig/middleware.go:RequireMultiSigMiddleware`** (55L → 2 funcs) | signing   | 8m  | Complex verification         |

### Priority 5: Consumer Experience

| #   | Task                                                                                                    | Module  | Est | Impact                                  |
| --- | ------------------------------------------------------------------------------------------------------- | ------- | --- | --------------------------------------- |
| 18  | **Write a unified Getting Started guide** (docs/getting-started.md) showing full CQRS stack composition | docs    | 20m | No single entry point for new consumers |
| 19  | **Add command/example_test.go** — New + Register + Dispatch roundtrip                                   | command | 5m  | Most-used pattern has no example        |
| 20  | **Add query/example_test.go** — New + Register + DispatchTyped                                          | query   | 5m  | Same for queries                        |
| 21  | **Add decider/example_test.go** — Decider + Repository + Execute                                        | decider | 8m  | Core aggregate pattern                  |
| 22  | **Update storage/README.md** — add Turso section, v2 import paths                                       | storage | 5m  | Missing backend documentation           |

### Priority 6: Architecture (v3 Prep)

| #   | Task                                                                                         | Module         | Est | Impact                               |
| --- | -------------------------------------------------------------------------------------------- | -------------- | --- | ------------------------------------ |
| 23  | **Design generic middleware interface** to eliminate 3x duplication                          | middleware     | 30m | Biggest code smell, design doc only  |
| 24  | **Decouple command from event types** — extract shared types to `id/` or new `types/` module | command, event | 20m | Module boundary fix, design doc only |
| 25  | **Set up PostgreSQL integration tests with testcontainers**                                  | integration    | 20m | PG code path has zero test coverage  |

---

## g) My Top 1 Question I Cannot Figure Out Myself

**What is the v3 release timeline and breaking-change budget?**

ADR-0010 (io.Closer removal), ADR-0011 (ErrDispatcherClosed unification), and ADR-0012 (catalog split) are all "Proposed" for v3. The middleware 3x duplication and command↔event coupling are also v3-scale changes. Without knowing when v3 is planned:

- Should we invest in v3 design documents now (risk: specs rot if v3 is far away)?
- Should we continue polishing v2.x within current API constraints (safe but the debt grows)?
- Is there a v2.2.0 release planned before v3, and if so, what's the scope?

This determines whether the next 25 tasks should focus on v2.x polish or v3 preparation.

---

## Module Coverage Summary

| Module         | Coverage | Grade |
| -------------- | -------- | ----- |
| dispatcher     | 100.0%   | A+    |
| decider        | 100.0%   | A+    |
| middleware     | 98.5%    | A+    |
| memory         | 98.2%    | A+    |
| otel           | 96.4%    | A     |
| id             | 94.5%    | A     |
| query          | 94.3%    | A     |
| listing        | 94.9%    | A     |
| signing        | 85.5%\*  | B+    |
| command        | 80.5%    | B     |
| event          | 89.4%    | B+    |
| projection     | ~95%     | A     |
| schema         | 89.7%    | B+    |
| snapshot       | 92.3%    | A-    |
| storage (root) | 86.9%    | B+    |
| storage/sql    | 34.7%    | D     |
| pebble         | 86.7%    | B+    |
| watermill      | 92.6%    | A-    |
| codec          | 93.3%    | A     |
| catalog        | 87.4%    | B+    |
| turso          | 28.6%    | D     |

\*signing coverage pulled down by internal testutil package

## LoC Summary

| Category        |        LoC | Files |
| --------------- | ---------: | ----: |
| Production code |     24,788 |     — |
| Test code       |     45,033 |     — |
| Example code    |      5,942 |     — |
| **Total**       | **75,763** |     — |

## Git Activity

| Metric                    | Value                      |
| ------------------------- | -------------------------- |
| Commits since v2.1.0      | 49                         |
| Commits in v2.1.0 release | 63                         |
| Total v2.x commits        | 112+                       |
| ADRs                      | 11 (8 adopted, 3 proposed) |
| Tags                      | v2.0.0, v2.1.0             |
