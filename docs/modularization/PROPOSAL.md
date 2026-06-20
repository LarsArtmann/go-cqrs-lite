# Module Reorganization Proposal

> **Date:** 2026-05-29 | **Status:** ⚠️ SUPERSEDED — `core/` was dissolved in v2.0.0. All sub-packages promoted to workspace root. This document is retained for historical context only.

## Executive Summary

The project has **24 modules** (23 sub-modules + root) with two systemic problems:

1. **Transitive dependency pollution** — `saga` leaks into every module through `testhelpers`, and `memory` is a direct dependency of `core` despite being test-only.
2. **God-package in `core/event`** — 90+ exported symbols across 12 distinct concerns conflated into one package.

Both violate the project's own principles: "small and composable" modules, max 250 lines/file, and ISP.

**Goal:** Fix the coupling first (high leverage, low risk), then split `core/event` into sub-packages (high value, medium effort). Leave `storage/` and `catalog/` as-is — they're borderline but defensible.

---

## Current State Analysis

### Module Inventory

| Module         | Lines     | Internal Deps (direct)                              | State                                           |
| -------------- | --------- | --------------------------------------------------- | ----------------------------------------------- |
| `otel`         | 590       | none                                                | **Clean** — leaf module                         |
| `codec`        | 271       | none                                                | **Clean** — leaf module                         |
| `catalog`      | 11,957    | none                                                | **Clean** — zero internal deps                  |
| `core`         | 14,877    | codec, otel (prod); memory, testhelpers (test-only) | **Go-inherent leak** — test deps in prod go.mod |
| `saga`         | 2,255     | core, otel, testhelpers                             | **Leaky** — pulls saga via testhelpers          |
| `testhelpers`  | 2,512     | core, saga                                          | **Leaky** — saga in production code             |
| `memory`       | 3,386     | core, testhelpers                                   | **Leaky** — pulls saga transitively             |
| `middleware`   | 3,932     | core, otel, testhelpers                             | **Leaky** — pulls saga transitively             |
| `signing`      | 4,156     | core, testhelpers                                   | **Leaky** — pulls saga transitively             |
| `projection`   | 3,120     | core, memory, otel, testhelpers                     | **Leaky** — pulls saga transitively             |
| `storage`      | 8,466     | core, otel, saga, testhelpers                       | **Borderline** — legitimate saga dep            |
| `stream`       | 2,513     | core, memory                                        | **Leaky** — memory is test-only                 |
| `watermill`    | 741       | core, memory, testhelpers                           | **Leaky** — pulls saga transitively             |
| `pebble`       | 1,535     | core, codec, otel, testhelpers                      | **Leaky** — pulls saga transitively             |
| `integration`  | 2,299     | 8 modules                                           | Acceptable — cross-module tests                 |
| `turso`        | 206       | core, storage                                       | **Clean** — thin adapter                        |
| `cmd/cqrs-gen` | 747       | none                                                | **Clean**                                       |
| `example/*`    | 100–3,472 | varies                                              | Acceptable — demos                              |
| ROOT           | 0         | none                                                | Empty shell                                     |

### Replace Directive Inventory

**19 modules have `replace` directives** (93 total). CI runs `GOWORK=off go test` per module, so replace directives are **required and correct**. They are NOT redundant with go.work — they serve a different purpose (isolated module builds).

**However:** 7 modules have **self-referencing** replace directives (`memory => ./`, `saga => ./`, etc.) which is unusual and worth investigating if they're truly needed.

### God-Package: `core/event`

90+ exported symbols across 12 logical concern clusters:

| Cluster               | Files | Key Types                                                         |
| --------------------- | ----- | ----------------------------------------------------------------- |
| Core Event Model      | 7     | `Event`, `ImmutableEvent`, `Type`, `Metadata`, `Option`, `New`    |
| Store Interfaces      | 3     | `EventSink`, `EventSource`, `Store`, `Journal`, `SeekableJournal` |
| Bus / Pub-Sub         | 1     | `Bus`, `Publisher`, `Subscriber`, `Middleware`                    |
| Outbox Pattern        | 3     | `Outbox`, `OutboxPublisher`, `OutboxEntry`                        |
| Snapshotting          | 3     | `Snapshot`, `SnapshotStore`, `SnapshotStrategy`                   |
| Projection/Checkpoint | 2     | `Projection`, `Checkpoint`, `CheckpointStore`                     |
| Upcasting             | 3     | `Upcaster`, `VersionedStore`                                      |
| Tombstone             | 1     | `TombstoneStatus`, `MarkTombstone`                                |
| Error Taxonomy        | 1     | ~30 error family re-exports                                       |
| Context/Enrichment    | 2     | `ContextEnricher`, `WithReplay`                                   |
| Slice Utilities       | 1     | `SliceFromVersion`, `FilterByTimestamp`                           |
| Codec/Batch           | 2     | `DecodePayload`, `NewEvents`                                      |

---

## Root Cause: The `saga` Transitive Leak

```
testhelpers/saga_helpers.go (production file)
  → imports saga
  → every module that depends on testhelpers gets saga transitively

core → testhelpers → saga     (core never uses saga)
memory → testhelpers → saga   (memory never uses saga)
middleware → testhelpers → saga (middleware never uses saga)
signing → testhelpers → saga   (signing never uses saga)
projection → testhelpers → saga (projection never uses saga)
watermill → testhelpers → saga (watermill never uses saga)
pebble → testhelpers → saga    (pebble never uses saga)
```

`saga_helpers.go` uses `testing.T` and `saga.Store` — it's clearly test infrastructure masquerading as production code. Only `saga/` and `storage/` tests actually use `NewSagaState`/`SaveSagaState`.

---

## Proposed Changes

### Change 1: Eliminate the `saga` Transitive Leak

**Move `testhelpers/saga_helpers.go` → `saga/testhelpers/saga_helpers.go`**

- Create `saga/testhelpers/saga_helpers.go` (new file in saga's package)
- Delete `testhelpers/saga_helpers.go`
- Remove `saga` from `testhelpers/go.mod`
- Result: `testhelpers` no longer depends on `saga`, eliminating transitive pollution for 7 modules

**Impact:** `saga` and `storage` tests update imports from `testhelpers.NewSagaState()` to `saga_test.NewSagaState()` (or similar). Minimal change, max leverage.

### Change 2: Remove Test-Only Dependencies from `core/go.mod`

**Move `memory` and `testhelpers` out of `core`'s direct requires.**

`core` is the **foundation module** — every other module depends on it. Having `memory` and `testhelpers` as direct deps is a violation: the core library module should be zero-internal-deps.

Options:

- **A) Inline test doubles** — Create `core/event/testutil` package with minimal fakes needed for tests (no external deps)
- **B) Accept test deps** — Go doesn't support test-only requires, so this is a known limitation

**Recommendation: Option A** — extract the 6 test files that import `memory` into a `core/event/eventtest` sub-package, similar to `httptest`. Remove `memory` from `core`'s direct deps.

### Change 3: Keep Replace + go.work (Corrected After Self-Review)

**Both `replace` directives AND `go.work` are needed and correct.** They serve different purposes:

- `go.work` — developer convenience, workspace builds
- `replace` — per-module isolation (`GOWORK=off go test` in CI)

**Action:** Clean up version inconsistencies (some modules reference `v1.0.0`, others `v1.6.0`). Standardize all internal refs to the same sentinel version.

### Change 4: Split `core/event` into Sub-Packages

Split the god-package while maintaining backward compatibility via type aliases:

| New Package             | Contents                                                                                                                               | Rationale                     |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------- |
| `core/event` (keeps)    | Core model, metadata, options, types, builder, errors, tombstone, enricher, replay, slice, codec, batch                                | The essential event types     |
| `core/event/store`      | `EventSink`, `EventSource`, `Store`, `Journal`, `SeekableJournal`, `BackwardsSource`, `TransactionalSink`, `AggregateRef`, `StreamKey` | ISP — store interfaces        |
| `core/event/bus`        | `Bus`, `Publisher`, `Subscriber`, `Handler`, `Middleware`, `PublishMiddleware`                                                         | Pub-sub is a distinct concern |
| `core/event/outbox`     | `Outbox`, `OutboxEntry`, `OutboxPublisher`, `OutboxPublisherOption`, `PublishNow`                                                      | Outbox pattern is optional    |
| `core/event/snapshot`   | `Snapshot`, `SnapshotStore`, `SnapshotStrategy`, `EveryNEvents`, `ShouldSnapshot`, `SaveSnapshot`                                      | Snapshotting is optional      |
| `core/event/projection` | `Projection`, `BatchProjection`, `NewProjection`, `Checkpoint`, `CheckpointStore`                                                      | Projection support types      |
| `core/event/upcaster`   | `Upcaster`, `NewUpcaster`, `VersionedStore`, `NewVersionedStore`                                                                       | Schema evolution is optional  |

**Backward compatibility:** `core/event` re-exports all types via type aliases and wrapper functions so existing consumers don't break:

```go
// core/event/store_alias.go
package event

// Store interface aliases for backward compatibility.
type Store = store.Store
type EventSink = store.EventSink
// ...
```

### Change 5: Version Inconsistency Fix

`testhelpers` references `saga v1.0.0` while everything else uses `saga v1.6.0`. After removing saga from testhelpers (Change 1), this resolves itself.

---

## Proposed Dependency DAG

```
                    otel (leaf)
                      ↑
                      |
          core ──→ codec (leaf)
            ↑
     ┌──────┼──────┬────────┬──────────┐
     |      |      |        |          |
   saga  memory  signing  middleware  catalog
     ↑      ↑                ↑
     |      |                |
  storage  stream      projection
     ↑
   turso
```

**After changes:**

- `core` → zero internal deps (only codec if needed)
- `testhelpers` → core only (saga removed)
- `saga` → core, otel (testhelpers removed from prod deps)
- No module transitively pulls saga unless it explicitly depends on saga

---

## Replace / Workspace Strategy

| Strategy         | Decision                                                                |
| ---------------- | ----------------------------------------------------------------------- |
| Replace strategy | **Keep both** — go.work for dev, replace for per-module CI (GOWORK=off) |

## Versioning Strategy

**Shared versioning** — single git tag `v1.x.x`, all modules bump together. The library is pre-v1.0.0, single-team, and modules are tightly coupled. Independent semver adds overhead with zero benefit at this stage.

## Risk Assessment

| Risk                                           | Severity | Mitigation                                                |
| ---------------------------------------------- | -------- | --------------------------------------------------------- |
| Splitting `core/event` breaks consumers        | High     | Type aliases in `core/event` provide backward compat      |
| Removing replace breaks builds without go.work | Medium   | Verify `go work sync` works; CI uses go.work              |
| Moving saga_helpers breaks saga/storage tests  | Low      | Simple import path change                                 |
| Circular imports after split                   | Low      | DAG verified; core/event has no imports from sub-packages |

## What We're NOT Changing

| Item                                 | Reason                                                                                 |
| ------------------------------------ | -------------------------------------------------------------------------------------- |
| `storage/` package structure         | All stores share `sqlBase`+`Dialect`; splitting adds import complexity without benefit |
| `catalog/` package structure         | Cohesive domain model; exporters already properly split                                |
| Module boundaries (go.mod locations) | The 24-module split is correct; the problem is coupling, not boundary placement        |
| `example/` modules                   | Demos are fine as-is                                                                   |

---

## Self-Review Corrections (Phase 4)

### Correction 1: Core production deps are clean

Initial analysis said `core` has `memory`/`testhelpers` as production deps. **Wrong.** Core's production code only imports `codec` and `otel`. The `memory`/`testhelpers` deps are test-only (`_test.go` files). Go's module system puts them in the `require` block regardless. This is a Go limitation, not a design flaw.

### Correction 2: Replace directives are necessary

Initial proposal called for removing all `replace` directives. **Wrong.** CI runs `GOWORK=off go test` per module, which requires `replace` directives to resolve local modules. Both strategies serve different purposes and coexist correctly.

### Correction 3: Self-referencing replaces are a smell but functional

7 modules replace themselves (`saga => ./`). This is unusual but works — it tells `go mod tidy` to use the local copy instead of fetching from a proxy. Harmless but could be cleaner.

### Remaining Valid Issues

1. **saga transitive leak through testhelpers** — still a real problem
2. **core/event god-package** — still 90+ exports across 12 concerns
3. **Version inconsistencies** — `testhelpers` references `saga v1.0.0` while others use `v1.6.0`

---

## Execution Results

### T1: Move saga_helpers out of testhelpers - DONE

- Created saga/sagatest/saga_helpers.go
- Deleted testhelpers/saga_helpers.go
- Removed saga from testhelpers/go.mod
- Updated imports in saga/ (3 test files) and storage/ (3 test files)
- Result: testhelpers no longer depends on saga. Seven modules no longer transitively pull saga.

### T2: Version normalization - DONE

- Ran go mod tidy in clean modules, go work sync at root
- Pre-existing stream-to-listing rename was detected and preserved

### T3: core/event split - DEFERRED

- Reason: 242 files import core/event. While type aliases provide backward compat, the internal code update is a large mechanical refactor better suited for a dedicated PR.
- The proposal and execution plan are ready for when the team wants to proceed.

### Pre-existing issue found

- stream module is being renamed to listing - this is in-progress and should be completed separately
- Some example/stream files are staged as deleted; go.work was updated to reflect new listing module
