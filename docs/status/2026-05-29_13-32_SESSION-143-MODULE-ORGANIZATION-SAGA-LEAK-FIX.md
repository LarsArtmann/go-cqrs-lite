# Session 143 — Module Organization Improvement & Saga Leak Fix

**Date:** 2026-05-29 13:32
**Branch:** master
**Status:** Tests green (30/30 packages pass)

---

## A) FULLY DONE

### T1: Eliminated the `saga` transitive dependency leak

**Problem:** `testhelpers/saga_helpers.go` was production code that imported `saga`. Since 7+ modules depend on `testhelpers`, every one of them transitively pulled `saga` — even though none of them use it.

**Fix:**
- Created `saga/sagatest/saga_helpers.go` — a sub-package within the saga module
- Deleted `testhelpers/saga_helpers.go`
- Removed `saga` from `testhelpers/go.mod` (both direct require and replace)
- Updated 6 test files across `saga/` (3 files) and `storage/` (3 files) to import `saga/sagatest`

**Impact:** `testhelpers` now depends only on `core`. Seven modules (`core`, `memory`, `middleware`, `signing`, `projection`, `watermill`, `pebble`) no longer transitively pull `saga`.

**Before:**
```
testhelpers → saga → core, otel
core → testhelpers → saga (transitive leak!)
memory → testhelpers → saga (transitive leak!)
middleware → testhelpers → saga (transitive leak!)
signing → testhelpers → saga (transitive leak!)
projection → testhelpers → saga (transitive leak!)
watermill → testhelpers → saga (transitive leak!)
pebble → testhelpers → saga (transitive leak!)
```

**After:**
```
testhelpers → core only (clean!)
saga/sagatest → saga, core/pkg/id (only imported by tests that need it)
```

### T2: Module version normalization

- Ran `go mod tidy` across all modules
- Ran `go work sync` to align workspace
- Verified all 30 packages pass

### Comprehensive module analysis (Phases 1-5 of go-modularize skill)

- Mapped all 24 modules with their dependencies, file counts, and line counts
- Identified god-packages: `core/event` (90+ exports, 12 concern clusters)
- Identified coupling hotspots: `saga` leak, test-only deps in production go.mod
- Verified `replace` + `go.work` dual strategy is correct and needed
- Verified `core` production deps are clean (only `codec` + `otel`)
- Audited `internal/` usage (only `catalog/internal` — properly contained)

### Documentation

- Created `docs/modularization/PROPOSAL.md` — full analysis with DAG, coupling, proposed changes
- Created `docs/modularization/EXECUTION_PLAN.md` — step-by-step plan with Pareto tiers
- Updated `AGENTS.md` — module graph, `sagatest` sub-package, replace+go.work rationale

---

## B) PARTIALLY DONE

### `stream → listing` module rename (pre-existing, in-progress)

The `stream/` module is being renamed to `listing/`. This was already in-progress before this session:

- `listing/` exists with updated files (module path changed, types renamed)
- `stream/` files are staged for deletion
- `example/stream/` is staged for deletion, `example/listing/` exists
- `go.work` updated to use `listing` instead of `stream`
- `stream/projection.go` moved to `storage/aggregate_projection.go`

**Not yet committed.** 52 files changed (+142, -3222 lines). This is a large rename that needs careful review.

### `core/store/backend.go` (pre-existing, experimental)

An untracked file at `core/store/backend.go` defines a `Backend` interface (universal key-value storage primitive). This is experimental and not yet integrated.

---

## C) NOT STARTED

### T3: Split `core/event` into sub-packages

The biggest remaining structural improvement. `core/event` has 90+ exported symbols across 12 concern clusters. Proposed split:

| New Sub-package | Contents |
|---|---|
| `core/event` (keeps) | Core model, metadata, options, types, builder, errors, tombstone, enricher, replay, slice, codec, batch |
| `core/event/store` | `EventSink`, `EventSource`, `Store`, `Journal`, `SeekableJournal`, `BackwardsSource`, `TransactionalSink` |
| `core/event/bus` | `Bus`, `Publisher`, `Subscriber`, `Handler`, `Middleware`, `PublishMiddleware` |
| `core/event/snapshot` | `Snapshot`, `SnapshotStore`, `SnapshotStrategy`, `EveryNEvents` |
| `core/event/outbox` | `Outbox`, `OutboxEntry`, `OutboxPublisher`, `PublishNow` |
| `core/event/projection` | `Projection`, `BatchProjection`, `Checkpoint`, `CheckpointStore` |
| `core/event/upcaster` | `Upcaster`, `VersionedStore` |

**Impact:** 242 files import `core/event`. Backward-compat type aliases would prevent external breakage, but internal code would need updating.

**Status:** Proposal and execution plan are complete. Ready for a dedicated PR.

### Error taxonomy extraction from `core/event`

`errors.go` has ~30 error family re-exports. Could move to `core/event/errors` or `core/errors`. Optional — the re-exports are heavily used.

### 116 `.go` files exceeding 250-line limit

- 9 production files exceed 250 lines
- 66+ test files exceed 250 lines
- Worst offender: `memory/store_test.go` at 659 lines

---

## D) TOTALLY FUCKED UP

### Nothing is broken.

All 30 test packages pass. No compilation errors. No circular dependencies. The workspace builds clean.

### Close call: `go mod tidy` deleted `stream/` module

During T2 (version normalization), running `go mod tidy` per-module caused `stream/go.mod` to be deleted because `go work sync` had already removed it from `go.work` (part of the `stream → listing` rename). Recovered via `git restore`.

**Lesson:** Don't run `go mod tidy` per-module when a rename is in-progress. The workspace-level `go work sync` should be the authority.

---

## E) WHAT WE SHOULD IMPROVE

### 1. The `core/event` god-package (highest structural priority)

90+ exported symbols. 12 distinct concerns. Every consumer of the library sees all of them. This violates ISP and makes the package hard to navigate.

### 2. 116 files over 250-line limit

The project's own convention is 250 lines max. Most violations are in test files, but 9 production files also exceed it. The worst offenders need splitting regardless of the sub-package effort.

### 3. Test-only deps in production `go.mod`

`core/go.mod` lists `memory` and `testhelpers` as direct dependencies, but they're only used in `_test.go` files. Go has no test-only require block — this is a known Go limitation. Workaround: inline minimal test doubles or accept the status quo.

### 4. 7 self-referencing replace directives

`memory`, `middleware`, `projection`, `saga`, `signing`, `stream`/`listing`, `watermill` all have `replace <self> => ./`. This works but is unusual. It's needed for `GOWORK=off go mod tidy` to work without fetching from a proxy.

### 5. The `stream → listing` rename needs completion

26 files staged for deletion, 17 files modified. This is a large in-progress rename that should be committed and finalized.

### 6. `cmd/api-stability` tool

`cmd/api-stability/main.go` is modified but uncommitted. This should be reviewed and committed.

---

## F) Top 25 Things to Get Done Next

### Tier 1 — High Impact (structural)

| # | Task | Effort | Impact |
|---|---|---|---|
| 1 | **Split `core/event` into sub-packages** (store, bus, snapshot, outbox, projection, upcaster) with backward-compat aliases | Large | Fixes the god-package, improves ISP for all consumers |
| 2 | **Complete `stream → listing` rename** — commit the staged changes | Medium | Removes dead `stream/` module, cleans up go.work |
| 3 | **Split top-5 longest test files** — `memory/store_test.go` (659), `catalog/eventcatalog/exporter_test.go` (527), `testhelpers/fake_store_test.go` (502), `cmd/cqrs-gen/main_test.go` (500), `catalog/eventcatalog/exporter_new_test.go` (483) | Medium | Gets largest files under 250-line limit |
| 4 | **Split top-9 longest production files** under 250 lines | Medium | Enforces project convention |
| 5 | **Commit `core/store/backend.go`** or remove it — don't leave experimental code untracked | Small | Clean git state |

### Tier 2 — Broad Value (quality)

| # | Task | Effort | Impact |
|---|---|---|---|
| 6 | **Add tests for `saga/sagatest`** — currently `[no test files]` | Small | Coverage gap |
| 7 | **Add tests for `core/store`** — currently `[no test files]` (if kept) | Small | Coverage gap |
| 8 | **Remove 7 self-referencing replace directives** — investigate if they're truly needed with go.work | Small | Cleaner go.mod files |
| 9 | **Standardize all internal module versions** to a single version (currently mix of `v0.0.0-000…`, `v1.0.0`, `v1.6.0`) | Small | Version consistency |
| 10 | **Extract inline test doubles in `core/`** to remove `memory` from `core/go.mod` | Medium | Core has zero internal deps |
| 11 | **Add `listing/` tests to CI per-module job** — CI config only lists old module names | Small | CI coverage |
| 12 | **Update `docs/modularization/MODULE_ASSESSMENT.md`** — it incorrectly says "core has zero internal deps in production code" but core depends on `codec` and `otel` | Small | Accurate docs |
| 13 | **Review and commit `cmd/api-stability/main.go`** changes | Small | Don't lose work |
| 14 | **Fix `nix run .#lint` warnings** — unused functions in storage tests | Small | Clean lint |
| 15 | **Add `CODEOWNERS`** or module ownership docs | Small | Governance |

### Tier 3 — Polish (long-term health)

| # | Task | Effort | Impact |
|---|---|---|---|
| 16 | **Create `docs/adr/` for module structure decisions** — document why 24 modules, why replace+go.work | Small | Knowledge preservation |
| 17 | **Generate a visual D2 dependency graph** from actual go.mod data | Small | Architecture visibility |
| 18 | **Add per-module `README.md`** for the 5 biggest modules (core, storage, catalog, signing, middleware) | Medium | Discoverability |
| 19 | **Extract `catalog/schema.go` (352 lines) reflection logic** into `catalog/schema/` sub-package | Medium | Reduces catalog root package size |
| 20 | **Split `storage/saga_store.go` (269 lines)** — exceeds 250-line limit | Small | Convention compliance |
| 21 | **Split `storage/outbox.go` (261 lines)** — exceeds 250-line limit | Small | Convention compliance |
| 22 | **Add `go work vendor`** support for offline builds | Small | Reproducibility |
| 23 | **Investigate removing `scripts/go-mod-graph-local`** — is it still needed? | Small | Dead code removal |
| 24 | **Add integration test for `listing` module** — renamed from stream, needs fresh validation | Medium | Confidence in rename |
| 25 | **Plan v1.0.0 release** — define the stability contract for all 24 modules | Large | Production readiness |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Is `core/store/backend.go` a direction you want to pursue, or should it be removed?**

It defines a universal `Backend` interface (key-value `Get/Put/Delete/Scan`) that would unify all storage implementations (SQL, Pebble, Memory) under one abstraction. This is a significant architectural decision:

- **Pro:** "Add a new backend once → ALL domain stores work." Type-safe adapters over a universal primitive.
- **Con:** It's a leaky abstraction — SQL stores need transactions, Pebble needs iterators, Memory needs concurrency control. A universal `Backend` may not capture what makes each backend special.

This file is untracked and not referenced by any other code. I need your direction before touching it.

---

## Module Inventory

| Module | Prod Files | Test Files | Lines | Internal Deps | State |
|---|---|---|---|---|---|
| `catalog` | 48 | 32 | 11,957 | none | Clean |
| `cmd/cqrs-gen` | 1 | 1 | 747 | none | Clean |
| `codec` | 3 | 1 | 271 | none | Clean |
| `core` | 55 | 58 | 14,967 | codec, otel (prod); memory, testhelpers (test-only) | Clean (prod deps minimal) |
| `integration` | 0 | 15 | 2,299 | 8 modules | Acceptable (cross-module tests) |
| `listing` | 8 | 9 | 2,497 | core, memory | Clean (renamed from stream) |
| `memory` | 9 | 11 | 3,400 | core, testhelpers | Clean |
| `middleware` | 14 | 18 | 3,932 | core, otel, testhelpers | Clean |
| `otel` | 7 | 2 | 590 | none | Clean (leaf) |
| `pebble` | 8 | 4 | 1,613 | core, codec, otel, testhelpers | Clean |
| `projection` | 8 | 14 | 3,300 | core, memory, otel, testhelpers | Clean |
| `saga` | 12 | 11 | 2,288 | core, otel, testhelpers | Clean (+ sagatest sub-pkg) |
| `signing` | 15 | 15 | 4,156 | core, testhelpers | Clean |
| `storage` | 25 | 25 | 8,605 | core, otel, saga, testhelpers | Clean |
| `testhelpers` | 9 | 6 | 2,480 | core | **Clean** (saga removed!) |
| `turso` | 4 | 0 | 206 | core, storage | Clean |
| `watermill` | 3 | 3 | 741 | core, memory, testhelpers | Clean |

**Totals:** 506 Go files, 70,388 lines, 25 go.mod files, 30 test packages (all green).

---

## Files Changed This Session

### New files:
- `saga/sagatest/saga_helpers.go` (extracted from testhelpers)
- `docs/modularization/PROPOSAL.md`
- `docs/modularization/EXECUTION_PLAN.md`

### Modified files:
- `testhelpers/go.mod` — removed saga dependency
- `testhelpers/saga_helpers.go` — deleted
- `saga/saga_bdd_test.go`, `saga/runner_edge_test.go`, `saga/store_test.go` — updated imports
- `storage/saga_store_test.go`, `storage/sql_backend_test.go`, `storage/sqlite_integration_outbox_saga_test.go` — updated imports
- `AGENTS.md` — updated module graph and structure
- `go.work` — synced
- `go.work.sum` — updated

### Pre-existing staged changes (not mine, NOT committed):
- `stream → listing` rename: 52 files changed (+142, -3222 lines)
- `core/store/backend.go` — untracked experimental file
