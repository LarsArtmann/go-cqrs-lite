# Comprehensive Status Report — go-cqrs-lite

> **Date:** 2026-06-03 12:31
> **Session:** 145 (TypedHandler Signature Fix + Housekeeping)
> **Branch:** master
> **Commits Ahead of Origin:** 0 (all pushed, uncommitted changes pending)

---

## Executive Summary

| Metric | Value |
|--------|-------|
| **Total Modules** | 30 (22 library + 6 examples + 1 cmd + 1 integration) |
| **Source Files** | 259 `.go` files |
| **Test Files** | 237 `_test.go` files |
| **Benchmark Files** | 24 `benchmark_test.go` / `bench_test.go` |
| **Overall Coverage** | **83.5%** |
| **Packages with 100%** | decider, dispatcher, catalog/caseutil |
| **Lowest Coverage** | turso (28.6%), eventtest (18.4%) |
| **Race Detector** | ✅ Zero races across all 32 packages |
| **Build Status** | ✅ Clean (all 38 packages compile + pass) |
| **Lint Status** | ✅ Clean (1 pre-existing issue in schema/) |
| **LSP Diagnostics** | ✅ 0 errors, 0 warnings |

---

## a) FULLY DONE

### This Session (Session 145) — `query.TypedHandler` Signature Fix

**Problem:** `query.TypedHandler[T any]` accepted `query.Query` (untyped), forcing every handler to manually type-assert `q.(*ConcreteQuery)`. Compare with `command.TypedHandler[T Command]` which receives the typed command directly. The query handler had half the type safety.

**Fix:** `TypedHandler[T any]` → `TypedHandler[Q Query, R any]` — two type parameters. `RegisterTyped` now performs the `q.(Q)` assertion internally, mirroring the command pattern.

| # | Change | Files | Impact |
|---|--------|-------|--------|
| 1 | `TypedHandler[Q Query, R any]` — typed query input + typed result output | `query/query.go:53` | Type-safe query handlers — no more manual assertions |
| 2 | `RegisterTyped[Q, R]` with built-in type assertion | `query/dispatcher.go:76` | Framework handles assertion, returns `ErrTypeAssertion` on mismatch |
| 3 | `ErrTypeAssertion` sentinel error | `query/errors.go:23` | Consistent with `command.ErrTypeAssertion` |
| 4 | Removed manual type assertions from all handlers | `example/user/handlers.go`, `integration/full_flow_test.go` | Cleaner consumer code |
| 5 | Updated `example/todo/queries/*.go` — typed `Handle` methods | `get_todo.go`, `list_todos.go`, `count_todos.go` | Handlers receive `*ConcreteQuery` directly |
| 6 | Removed `requireQueryType` helper (no longer needed) | `example/todo/queries/types.go` | -16 lines of dead helper code |
| 7 | Removed dead `errUnexpectedQueryType` | `example/user/handlers.go` | -1 sentinel, removed `errors` import |
| 8 | Updated code generator for `[Q, R]` signature | `cmd/cqrs-gen/main.go:238` | Generates `query.RegisterTyped[*StructName, R]` |
| 9 | Updated all tests | 4 test files | All passing |

**Net impact:** 14 files changed, +60 / -100 lines. Every query handler is now cleaner — zero manual type assertions.

### Previous Session (Session 144) — Performance + Housekeeping

| # | Change | Impact |
|---|--------|--------|
| 1 | MemoryStore global log | ReadAll: 98μs→3.3μs (-96%) |
| 2 | Listing projection cache | List: 840ms→33ms (-96%) |
| 3 | ImmutableEvent slim-down | 336B→304B per event (-10%) |
| 4 | 10 new benchmarks across 4 modules | codec, watermill, turso, memory |
| 5 | 6 housekeeping items | ADR numbering, go.mod tidy, deprecated API removal, CONTRIBUTING rewrite |
| 6 | 14 FakeStore tests | eventtest coverage: 0%→18.4% |

### V2.0.0 Release (Session ~100-143) — Complete

All 23 modules tagged at v2.0.0 with `/v2` semantic import paths. All P0–P5 release blockers resolved.

---

## b) PARTIALLY DONE

| Item | What's Done | What's Missing | Why Blocked |
|------|-------------|----------------|-------------|
| **Pebble Journal** | `Store` interface implemented | `Journal`/`SeekableJournal` not implemented | Needs design decision on iteration order |
| **cqrs-gen tests** | Core logic tested (89.9%) | CLI entry point untested | Requires mock filesystem |
| **Turso coverage** | Basic connector tests (28.6%) | SyncDB Push/Pull/Checkpoint untested | Requires remote Turso server |
| **api-stability tool** | Tool exists, works | Zero tests | Low priority — tool is self-validating |
| **eventtest coverage** | FakeStore tested (18.4%) | VersionQueryFn, other helpers untested | Time constraint |
| **ADR 0008 update** | TypedHandler ADR exists | Needs update to reflect new `[Q, R]` signature | Documentation debt |

---

## c) NOT STARTED

### From TODO_LIST.md (Still Open, Checked)

| # | Item | Source | Priority |
|---|------|--------|----------|
| 1 | `pebble/config.go:59-69` — 20 lines of backward-compat aliases | Session 140 review | LOW |
| 2 | ~~`query.TypedHandler[T]` takes `Query` not `T`~~ — **NOW DONE (this session)** | Session 140 review | ~~LOW~~ ✅ |
| 3 | `ROADMAP.md` creation | Documentation gap | LOW |
| 4 | `CHANGELOG.md` update | 2+ days behind | LOW |
| 5 | Outbox Pattern design doc | FEATURES.md planned | MEDIUM |
| 6 | Schema Registry design doc | FEATURES.md planned | MEDIUM |

### From Planning Docs (Phase C + D — Performance)

| # | Task | Est | Impact |
|---|------|-----|--------|
| C1 | MemoryStore deduplication: store events ONLY in globalLog, index for per-stream | 20min | **2× memory reduction** |
| C2 | Listing cache auto-invalidation via atomic counter | 10min | Correctness + performance |
| C3 | `findCodecOption` elimination — cache default codec | 10min | 1 alloc per `New()` |
| C4 | `sync.Pool` for `ImmutableEvent` construction | 15min | Reduce GC pressure |
| C5 | api-stability tests | 10min | Untested guard tool |
| C6 | `Option` as interface (v3) | 20min | Eliminates func closure alloc |
| D1 | Reactive `AggregateReader` (subscribe to bus) | 15min | Real-time listing |
| D2 | Compact `Metadata` representation | 20min | Reduce 152B per event |
| D3 | Evaluate faster JSON codec (`goccy/go-json`) | 15min | Potential 2-3× speedup |

### From TODO_LIST.md (v2 / BLOCKED / FUTURE)

| # | Item | Status |
|---|------|--------|
| `[v2]` | Add global TransactionID branded type | Deferred to next major |
| `[v2]` | io.Closer removal from core interfaces | Deferred to next major |
| `[v2]` | Split event.Store into Writer/Reader/Deleter | Deferred to next major |
| `[v2]` | Make event Core truly immutable | Deferred to next major |
| `[BLOCKED]` | Add PostgreSQL integration tests with testcontainers | Requires Docker |
| `[BLOCKED]` | Move example/todo to own repository | Requires manual repo creation |
| `[BLOCKED]` | Push signing v1.0.0 tag | Requires manual tag |
| `[BLOCKED]` | Change LICENSE from proprietary to MIT/Apache-2.0 | Requires owner decision |
| `[FUTURE]` | Outbox pattern, schema registry, bi-temporal, HLC, consensus | Far future |

---

## d) TOTALLY FUCKED UP

| # | Issue | Severity | Root Cause | Fix |
|---|-------|----------|------------|-----|
| 1 | **BuildFlow pre-commit hook broken** | HIGH | `build_mode: full` runs heavy checks, missing excludes, no actual git hook plumbing | Deferred per user request |
| 2 | **ADR 0005 missing** | LOW | Gap in sequence — skipped during creation | Cosmetic |
| 3 | **turso coverage at 28.6%** | MEDIUM | SyncDB Push/Pull/Checkpoint require remote server | Needs testcontainers or mock |
| 4 | **Pre-existing lint failure** | LOW | `schema/benchmark_test.go:61` variable `vs` too short | Trivial fix, not touched this session |

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (This Week)

1. **Update TODO_LIST.md** — Mark `query.TypedHandler[T]` as DONE. It was item #13 on the top-25 list and is now resolved.

2. **Update ADR 0008** — The TypedHandler ADR at `docs/adr/0008-typed-handler-signature.md` describes the old single-type-parameter design. It needs to reflect the new `[Q Query, R any]` two-type-parameter design.

3. **Update CHANGELOG.md** — Multiple sessions of work unrecorded. Sessions 144 (performance) and 145 (TypedHandler fix) are missing.

4. **MemoryStore deduplication (C1)** — Events stored in both `events` map AND `globalLog` = 2× memory. A `(type, id) → []int` index would eliminate the `events` map entirely.

5. **Listing cache auto-invalidation (C2)** — Current cache never invalidates automatically. An atomic counter on `Save()`/`AppendBatch()` would detect changes.

### Short-Term (This Month)

6. **Pebble Journal implementation** — Only `Store` implemented. `Journal`/`SeekableJournal` would unlock projection replay on Pebble.

7. **`Option` as interface (v3)** — Current `Option func(*ImmutableEvent)` allocates a closure per option. Interface-based design = zero alloc for common options.

8. **Reactive `AggregateReader`** — Subscribe to event bus, auto-update cache. Eliminates `InvalidateCache()`.

9. **Compact `Metadata`** — 152 bytes/event. Pointer-based fields for CorrelationID/CausationID/UserID/RequestID (most events don't set them).

10. **Faster JSON codec evaluation** — Benchmark `goccy/go-json` vs stdlib `encoding/json`.

### Medium-Term (Next Quarter)

11. **ROADMAP.md** — Document doesn't exist. Long-term direction is scattered across 91 status files and 38 planning docs.

12. **PostgreSQL integration tests** — Only SQLite in-memory today. Testcontainers for real PG validation.

13. **Documentation site** — Docusaurus/MkDocs for consumer-facing docs.

14. **Outbox Pattern implementation** — Documented in context but no code exists.

15. **Schema Registry** — JSON Schema middleware for event validation.

---

## f) Top #25 Things To Get Done Next

Sorted by **impact ÷ effort × urgency**:

| Rank | Task | Est | Impact | Status |
|------|------|-----|--------|--------|
| 1 | **MemoryStore deduplication (2× memory reduction)** | 20min | CRITICAL | Not started |
| 2 | **Listing cache auto-invalidation** | 10min | HIGH | Not started |
| 3 | **`findCodecOption` elimination** | 10min | MEDIUM | Not started |
| 4 | **Update CHANGELOG.md** | 10min | LOW | Not started |
| 5 | **Update TODO_LIST.md** (mark TypedHandler as done) | 5min | LOW | Not started |
| 6 | **Update ADR 0008 for [Q, R] signature** | 10min | MEDIUM | Not started |
| 7 | **Pebble Journal implementation** | 30min | HIGH | Not started |
| 8 | **`sync.Pool` for ImmutableEvent** | 15min | MEDIUM | Not started |
| 9 | **api-stability tests** | 10min | LOW | Not started |
| 10 | **`Option` as interface (v3 design)** | 20min | MEDIUM | Not started |
| 11 | **Compact Metadata representation** | 20min | MEDIUM | Not started |
| 12 | **Faster JSON codec evaluation** | 15min | MEDIUM | Not started |
| 13 | **Reactive AggregateReader** | 15min | MEDIUM | Not started |
| 14 | **pebble/config.go alias cleanup** | 5min | LOW | Not started |
| 15 | **ROADMAP.md creation** | 15min | LOW | Not started |
| 16 | **Outbox Pattern design doc** | 20min | HIGH | Not started |
| 17 | **Schema Registry design doc** | 20min | MEDIUM | Not started |
| 18 | **PostgreSQL integration tests** | 60min | HIGH | Blocked (Docker) |
| 19 | **BuildFlow pre-commit hook fix** | 15min | LOW | Deferred |
| 20 | **cqrs-gen CLI test coverage** | 20min | LOW | Not started |
| 21 | **eventtest remaining helpers** | 15min | LOW | Not started |
| 22 | **Fix schema/ lint failure** (`vs` var name) | 2min | TRIVIAL | Not started |
| 23 | **ADR 0005 gap fill** | 5min | TRIVIAL | Not started |
| 24 | **benchstat baseline establishment** | 10min | LOW | Not started |
| 25 | **GC pressure analysis / pprof harness** | 30min | LOW | Not started |

---

## g) Top #1 Question I Cannot Figure Out Myself

### Should we eliminate the `events` map in MemoryStore entirely?

**Context:** After Session 144's T1 optimization, MemoryStore stores every event in TWO places:
1. `events map[string][]event.Event` — per-stream, used by `Load()`, `LoadFromVersion()`, etc.
2. `globalLog []event.Event` — append-only, used by `ReadAll()`, `ReadFrom()`

This is **2× memory** for every event. For 100K events, that's significant.

**Option A: Keep both** (current)
- Pros: Simple, fast `Load()` (O(1) map lookup + slice clone)
- Cons: 2× memory

**Option B: Derive per-stream from globalLog**
- Add `streamIndex map[string][]int` (stream key → indices in globalLog)
- `Load()` looks up indices, clones events from globalLog
- Pros: 1× memory, single source of truth
- Cons: More complex, `Load()` has indirection

**Option C: Store only in globalLog, no per-stream index**
- `Load()` scans globalLog for matching (type, id)
- Pros: Absolute minimum memory
- Cons: `Load()` becomes O(N) in total events — terrible for large stores

**My recommendation: Option B** — the memory savings justify the complexity. But I want confirmation because it changes the internal architecture of MemoryStore significantly.

---

## Appendix: Coverage by Module

| Module | Coverage | Notes |
|--------|----------|-------|
| decider | 100.0% | Perfect |
| dispatcher | 100.0% | Perfect |
| catalog/caseutil | 100.0% | Perfect |
| middleware | 98.5% | Excellent |
| memory | 99.0% | Excellent |
| command | 93.8% | Excellent |
| query | 95.5% | Excellent |
| signing | 94.1% | Excellent |
| catalog | 95.9% | Excellent |
| id | 94.5% | Excellent |
| codec | 93.3% | Excellent |
| catalog/openapi | 96.4% | Excellent |
| event | 89.4% | Good |
| schema | 89.7% | Good |
| storage | 89.3% | Good |
| listing | 91.5% | Good |
| projection | 90.5% | Good |
| watermill | 92.6% | Good |
| pebble | 88.1% | Good |
| snapshot | 92.3% | Good |
| cmd/cqrs-gen | 89.9% | Good |
| turso | 28.6% | **Poor** — needs SyncDB tests |
| eventtest | 18.4% | Acceptable — test helpers |

---

## Appendix: Uncommitted Changes (This Session)

```
 cmd/cqrs-gen/main.go                 |  5 +++--
 cmd/cqrs-gen/main_test.go            |  8 ++++----
 example/todo/queries/count_todos.go  | 11 +++--------
 example/todo/queries/get_todo.go     | 11 +++--------
 example/todo/queries/list_todos.go   | 17 ++++++-----------
 example/todo/queries/queries_test.go | 11 -----------
 example/todo/queries/types.go        | 16 ----------------
 example/user/handlers.go             | 20 ++++----------------
 integration/full_flow_test.go        | 11 +++--------
 query/dispatcher.go                  | 14 ++++++++++----
 query/dispatcher_test.go             | 20 ++++++++++++--------
 query/errors.go                      |  6 ++++++
 query/example_test.go                |  2 +-
 query/query.go                       |  8 +++++---
 14 files changed, 60 insertions(+), 100 deletions(-)
```

---

*Report generated by Crush on 2026-06-03 at 12:31.*
