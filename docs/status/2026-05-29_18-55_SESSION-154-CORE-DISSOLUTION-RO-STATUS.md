# Session 154 — Core Dissolution, Schema/Snapshot Extraction, samber/ro Integration

**Date:** 2026-05-29 18:55  
**Branch:** master  
**Commits:** 11d9dc6 → e4e6112 → a40b71f → cc4ceb1

---

## Executive Summary

Dissolved the monolithic `core/` module into 8 flat peer-level modules, extracted `schema/` and `snapshot/` as separate domain modules, and added samber/ro as a reactive stream foundation. **All 32 test packages pass green.** However, the ro integration is additive (not a replacement), go.mod files need cleanup, and several architectural improvements remain.

---

## A) FULLY DONE ✅

### 1. core/ Dissolution (Phase 1+2) — `11d9dc6`
- Moved `core/event` → `event/`, `core/command` → `command/`, `core/query` → `query/`, `core/decider` → `decider/`, `core/pkg/id` → `id/`, `core/pkg/dispatcher` → `dispatcher/`
- Updated ALL import paths across 200+ files (`go-cqrs-lite/core/{pkg}` → `go-cqrs-lite/{pkg}`)
- Updated all 27 consumer `go.mod` files with new require/replace directives
- Updated `go.work`, `flake.nix`, `cmd/cqrs-gen`, `scripts/go-mod-graph-local`
- **All 31 test packages green**

### 2. schema/ Extraction (Phase 3) — `e4e6112`
- Moved `upcaster.go`, `upcaster_registry.go` → `registry.go`, `versioned_store.go` → `versioned_source.go` into `schema/`
- Qualified all types with `event.` prefix (cross-package access)
- Used `event.WithSchemaVersion()` for schema version mutation (field was unexported)
- Moved projection helper test back to `event/projection_test.go`
- Added `schema` dep to `event/go.mod`, `go.work`, `flake.nix`

### 3. snapshot/ Extraction (Phase 4) — `a40b71f`
- Created `snapshot/store.go` (Snapshot, SnapshotSink/Source/Store interfaces)
- Created `snapshot/helper.go` (ShouldSnapshot, SaveSnapshot)
- Created `snapshot/strategy.go` (SnapshotStrategy, EveryNEvents, MustEveryNEvents)
- Updated consumers: decider, memory, storage, testhelpers (17 files)
- Replaced `event.Snapshot*` → `snapshot.*` across all consumer imports
- Replaced `defaultClock()` with `time.Now()` (unexported → eliminated dependency)

### 4. samber/ro Foundation (Phase 5) — `cc4ceb1`
- Added `github.com/samber/ro v0.3.0` to `event/` and `command/`
- Created `event/reactive.go`: `EventBus (= ro.Subject[Event])`, `FilterEventType`, `FilterEventTypes`, `HandlerToObserver`
- Created `command/reactive.go`: `CommandBus (= ro.Subject[Command])`, `FilterCommandType`
- Tests for EventBus: publish/subscribe, multi-type filtering, multi-subscriber

---

## B) PARTIALLY DONE ⚠️

### 1. samber/ro Integration — Additive, NOT Replacement
- `event/reactive.go` sits alongside `event/bus.go` — both exist
- `MemoryBus` (218 lines) is still the hand-rolled implementation
- All middleware (10+ functions) still uses `event.Middleware func(Handler) Handler`
- All examples still use `event.Bus`, `event.Publisher`, `event.Subscriber`
- `CommandBus` has **zero external consumers**
- **What's missing**: Full Option A requires deleting `MemoryBus`, converting middleware to ro operators, updating all consumers

### 2. AGENTS.md Update
- Module list and test command updated
- Monorepo structure diagram update was **started but failed** (edit mismatch)
- Dependencies table, Key Patterns, and Module Graph sections still reference old `core/` structure

### 3. go.mod Cleanup
- **7 modules** have duplicate require entries (need `go mod tidy`)
- **6 modules** have self-referencing replace directives (harmless but noisy)
- **5 modules** missing from `flake.nix testModules`: `codec`, `listing`, `otel`, `pebble`, `turso`

---

## C) NOT STARTED ❌

1. **Full samber/ro migration** — Replace MemoryBus, convert middleware to operators, update all examples
2. **memory/ split** — Decided to keep as single module; no sub-module extraction done
3. **CHANGELOG.md update** — Not touched
4. **Dead code cleanup** — `command/reactive.go` has zero consumers
5. **Snapshot error sentinels** — `ErrSnapshotNotFound`, `ErrSnapshotStoreClosed` still in `event/errors.go` but should move to `snapshot/`
6. **go.work.sum cleanup** — Stale `core` entries still cached
7. **Root go.mod** — Still has placeholder `module github.com/larsartmann/go-cqrs-lite` (no actual code)
8. **Stale gopls cache** — Shows 72 errors from phantom `../core` reference
9. **docs/ research updates** — Proposals not marked as complete

---

## D) TOTALLY FUCKED UP 💥

### 1. Half-Implemented samber/ro
The worst outcome: we added a dependency without committing to it. Now we have:
- `event.Bus` interface with `Publish(ctx, ...Event) error` — context-aware, returns errors
- `event.EventBus` type alias to `ro.Subject[Event]` — no context, no error return
- `event.Handler func(ctx, Event) error` — context-aware
- `event.EventHandler func(ctx, Event) error` — **DUPLICATE** of Handler
- `event.HandlerToObserver()` — **throws away context and errors**

The impedance mismatch between context-aware handlers and ro's `Observer[T]` is fundamental. ro's `Next(T)` doesn't return errors. Our `Publish` does. This needs resolution.

### 2. Handler vs EventHandler Duplication
`event/bus.go` defines `Handler func(ctx, Event) error`.  
`event/reactive.go` defines `EventHandler func(ctx, Event) error`.  
**Two identical function types** in the same package. This is confusing.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Architecture Issues

1. **Impedance mismatch: context-aware vs ro** — Our handlers use `context.Context` and return `error`. ro's `Observer[T]` has `func(value T)` with no context or error. We need a bridge pattern or ro's `ObserverWithContext`.

2. **Middleware as operators** — 10+ middleware functions (`EventLogging`, `EventRetry`, `EventRecovery`, `EventMetrics`, `EventTracing`, etc.) are `func(Handler) Handler`. Under ro, these become `ro.Pipe` operators. Massive rewrite.

3. **Publisher interface vs ro.Subject.Next** — `Publisher.Publish(ctx, ...Event) error` (batch, context, error) vs `Subject.Next(Event)` (single, no context, void). Fundamentally different contracts.

4. **Snapshot error sentinels in wrong package** — `ErrSnapshotNotFound` and `ErrSnapshotStoreClosed` live in `event/errors.go` but belong in `snapshot/`.

5. **EventBus type alias is misleading** — `type EventBus = ro.Subject[Event]` suggests full replacement but nothing uses it. Should either commit or remove.

### Type Model Improvements

6. **CommandBus pattern** — We added `CommandBus = ro.Subject[Command]` but commands in CQRS are dispatched synchronously (not broadcast). `Subject` is multicast. Commands should use `ro.Observable[Command]` with single-consumer semantics, not a Subject.

7. **Event vs Command streaming semantics** — Events are broadcast (1→many). Commands are dispatched (1→1). Using `ro.Subject` for both ignores this distinction.

8. **Branded IDs could use generics more** — `id.Of[T]` is good. But `AggregateRef`, `AggregateType`, `Version`, `Type` are all unbranded strings/ints. Could benefit from stronger typing.

### Dependency Hygiene

9. **go.mod bloat** — 7 files with duplicate requires, 6 with self-referencing replaces. All need `go mod tidy`.

10. **samber/ro pulls in samber/lo + golang.org/x/exp** — Added 3 transitive deps to event/ and command/. If we commit to ro, this is fine. If not, we should remove it.

---

## F) TOP 25 THINGS TO DO NEXT (Pareto-sorted: impact × effort)

### Tier 1: High Impact, Low Effort (Do Now)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | **Fix AGENTS.md monorepo diagram** (edit failed) | 🔴 Missing context for all sessions | 5 min |
| 2 | **Move `ErrSnapshotNotFound`/`ErrSnapshotStoreClosed` to `snapshot/`** | 🔴 Types in wrong package | 15 min |
| 3 | **Remove duplicate `EventHandler` type** (identical to `Handler`) | 🟡 Confusion | 2 min |
| 4 | **Run `go mod tidy` on all 7 bloated go.mod files** | 🟡 Hygiene | 10 min |
| 5 | **Remove self-referencing replace directives** (6 files) | 🟡 Hygiene | 5 min |
| 6 | **Add missing modules to `flake.nix testModules`** (5 modules) | 🔴 CI doesn't test codec/listing/otel/pebble/turso | 5 min |
| 7 | **Clean up `go.work.sum`** (stale core references) | 🟡 Stale cache | 2 min |
| 8 | **Delete root `go.mod`** (3-line placeholder, no code) | 🟡 Clarity | 2 min |

### Tier 2: High Impact, Medium Effort (Do Next)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 9 | **Resolve ro impedance mismatch** — Decide: full adoption or remove | 🔴 Architectural clarity | Research |
| 10 | **If keeping ro**: Replace `MemoryBus` with `ro.PublishSubject[Event]` wrapper | 🔴 218 lines deleted | 1-2 hrs |
| 11 | **If keeping ro**: Convert `PublishMiddleware` chain to ro operators | 🔴 Middleware rewrite | 2-3 hrs |
| 12 | **If removing ro**: Delete `reactive.go` from event/ and command/ | 🟡 Clarity | 5 min |
| 13 | **Update CHANGELOG.md** with all session 154 changes | 🟡 Documentation | 20 min |
| 14 | **Update `docs/planning/` proposals** to mark as COMPLETE | 🟡 Documentation | 10 min |
| 15 | **Remove stale `scripts/go-mod-graph-local/go.mod`** if unused | 🟡 Cleanup | 5 min |

### Tier 3: Medium Impact, Medium Effort (Do Soon)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 16 | **Stronger type branding** — `AggregateType`, `Version`, `Type` as branded types | 🟡 Safety | 1-2 hrs |
| 17 | **Decide CommandBus semantics** — Subject (multicast) is wrong for commands | 🟡 Correctness | Research |
| 18 | **Write migration guide** for consumers upgrading from `core/` imports | 🟢 DX | 30 min |
| 19 | **Add example using reactive EventBus** (if keeping ro) | 🟢 DX | 30 min |
| 20 | **Audit `decider/go.mod`** — missing `codec` and `memory` deps | 🟡 Build hygiene | 10 min |

### Tier 4: Lower Impact, Higher Effort (Do Later)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 21 | **Full memory/ split into sub-modules** | 🟢 Architecture | 3-4 hrs |
| 22 | **Derived Event / CEP research** — docs/research/derived-event.md exists | 🟢 Features | Research |
| 23 | **Scheduled Event research** — docs/research/scheduled-event.md exists | 🟢 Features | Research |
| 24 | **Version tagging** — Push v1.0.0 tags to remove replace directives | 🔴 DX | 1 hr |
| 25 | **CI pipeline update** — Ensure GOWORK=off per-module CI still works | 🟡 CI | 30 min |

---

## G) TOP #1 QUESTION ❓

**The samber/ro impedance mismatch is the critical decision:**

Our `event.Handler` is `func(ctx context.Context, event Event) error` — it takes context and returns errors. samber/ro's `Observer[T]` is `func(value T)` with no context or error return. Even `ObserverWithContext` only gets context, still no error return.

This means:
- **Publish path**: `bus.Publish(ctx, ...events) error` → synchronous, batch, error-returning vs `subject.Next(event)` → fire-and-forget
- **Subscribe path**: `handler(ctx, event) error` → can fail, retry, trace vs `observer.OnNext(event)` → void return
- **Middleware chain**: `func(Handler) Handler` → wraps context+error vs `ro.Pipe(operators...)` → wraps void→void

**Question**: Should we:
1. **Commit to ro fully** — accept the impedance mismatch, use wrappers (`HandlerToObserver`), and accept that context propagation and error handling become lossy?
2. **Use ro internally only** — keep `Bus`/`Publisher`/`Subscriber` interfaces, implement `MemoryBus` using `ro.PublishSubject[Event]` as internal plumbing, preserve context+error semantics in the public API?
3. **Remove ro** — our hand-rolled `MemoryBus` is fine for a library. ro adds value for complex stream processing (projection catch-up via `ReplaySubject`, operator chains) but we don't have those use cases yet?

My recommendation: **Option 2** — ro as internal MemoryBus implementation. Public API stays context-aware and error-returning.

---

## Test Results

**32 packages tested, 0 failures:**
```
ok  event           ok  command         ok  query           ok  decider
ok  id              ok  dispatcher      ok  schema          ok  snapshot
ok  memory          ok  catalog         ok  catalog/*       ok  middleware
ok  integration/*   ok  projection      ok  signing/*       ok  storage
ok  testhelpers     ok  watermill       ok  cmd/cqrs-gen    ok  listing
ok  otel            ok  pebble          ok  codec           ok  turso
```

## Commit History This Session

```
cc4ceb1 feat: integrate samber/ro for reactive event/command streams
a40b71f refactor: extract snapshot/ module for snapshot persistence
e4e6112 refactor: extract schema/ module for upcasting + schema versioning
11d9dc6 refactor: dissolve core/ into peer-level modules (event, command, query, decider, id, dispatcher)
```

## Module Dependency Graph (Current State)

```
Layer 0: id/, dispatcher/, codec/         (leaf modules, no internal deps)
Layer 1: event/ (→id, codec), command/ (→id, dispatcher), query/ (→dispatcher)
Layer 2: schema/ (→event), snapshot/ (→event)   [NEW]
Layer 3: decider/ (→event, snapshot)            [UPDATED: was →event only]
Layer 4: memory/, testhelpers/, signing/, otel/
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
Layer 6: integration/, examples/, cmd/cqrs-gen, catalog/
```
