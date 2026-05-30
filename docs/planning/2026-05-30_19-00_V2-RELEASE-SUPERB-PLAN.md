# V2.0.0 Comprehensive Plan — Superb Release

**Date:** 2026-05-30
**Scope:** Code quality scan + full code review + architecture analysis + features audit + TODO list

---

## Quality Baseline

| Metric | Value |
|--------|-------|
| Build | ✅ PASS |
| Lint | ✅ 0 issues (all 29 modules) |
| Tests | ✅ 32/32 packages pass |
| Clone Groups (dupl -t 25) | 146 (most are benign boilerplate in catalog/id examples) |
| Source Files | 465 .go (253 prod, 212 test) |
| Total Lines (prod) | 23,191 |

---

## Critical Issues (Must Fix for v2.0.0)

### P0 — Correctness & Safety

| # | Module | File:Line | Issue | Effort |
|---|--------|-----------|-------|--------|
| 1 | event | `event_new.go:66,68` | `New()` doesn't clone `[]byte`/`json.RawMessage` — caller can mutate event internals | 15min |
| 2 | dispatcher | `dispatcher.go:240-241` | Data race: `CatalogDispatcher.RegisterHandlerMeta` has no mutex | 15min |
| 3 | catalog/docserver | `html.go:5,28` | XSS vulnerability — unescaped title in HTML | 15min |
| 4 | watermill | `subscriber.go:63` | Double-close panic — no `sync.Once` | 10min |
| 5 | watermill | `protocol.go:166-176` | `MustParse*` panics on untrusted metadata | 20min |
| 6 | memory | `checkpoint.go:29,40` | Missing closed-check on `Load` and `Save` | 10min |
| 7 | signing | `middleware.go:39,82,113` | Panics on nil signer/verifier — should return error | 15min |
| 8 | signing/multisig | `middleware.go:47,78,159` | Same panic-on-nil issue | 10min |
| 9 | cmd/api-stability | `main.go:14-19` | Wrong module paths (`core/` prefix) — tool is non-functional | 15min |
| 10 | cmd/cqrs-gen | `main.go:237` | Generated query handler signature won't compile | 20min |
| 11 | example/storage | `go.mod:20` | Missing `replace` directive for `listing` module | 5min |
| 12 | example/projection | `main.go:137` | Uses `ItemAdded` instead of `ItemRemoved` for removal event | 5min |

### P1 — Type Safety & Design

| # | Module | File:Line | Issue | Effort |
|---|--------|-----------|-------|--------|
| 13 | event | `event.go:275` | Exceeds 250-line rule — split constructor into `event_construct.go` | 15min |
| 14 | projection | `runner.go:280` | Exceeds 250-line rule — split replay into `runner_replay.go` | 15min |
| 15 | pebble | `store.go:265` | Exceeds 250-line rule — extract iteration to `iteration.go` | 15min |
| 16 | middleware | `metrics_otel.go:60` | `Observe()` uses `context.Background()` — loses trace correlation | 10min |
| 17 | middleware | `validation.go:28-33` | Swallows original validator error — loses failure reason | 10min |
| 18 | middleware | `circuit_breaker.go:222,238` | `fmt.Errorf` overwraps structured errors — destroys taxonomy | 10min |
| 19 | pebble | `helpers.go:65` | `logEventOperation` nil-logger panic | 10min |
| 20 | memory | `bus.go:119-122` | Double error wrapping in `Publish` | 10min |
| 21 | decider | `decider.go:176,184` | Snapshot errors silently discarded with `_ = opError(...)` | 10min |
| 22 | decider | `decider.go:48-77` | No validation: snapshot store without codec silently skips | 10min |
| 23 | schema | `versioned_source.go:17` | `NewVersionedStore(nil)` causes nil-pointer panic | 5min |
| 24 | schema | `upcaster.go:13` | `NewUpcaster(nil)` causes nil-pointer panic | 5min |
| 25 | otel | `spans.go:48-56` | `SpanFromContext` and `ComponentTracer` are dead code | 5min |
| 26 | middleware | `metrics_otel.go:16-20` | Unused `metricName*` constants — dead code | 5min |

### P2 — Duplication & Naming

| # | Module | File | Issue | Effort |
|---|--------|------|-------|--------|
| 27 | event | `tombstone.go` | `MarkTombstone`/`MarkRebirth` nearly identical — extract helper | 10min |
| 28 | signing | `middleware.go` vs `multisig/middleware.go` | `extractOrPassThrough` duplicated identically | 10min |
| 29 | middleware | `recovery.go` | Three near-identical recovery functions — extract generic helper | 15min |
| 30 | command/query | `dispatcher.go` | Identical closed-check+wrap boilerplate — extract helper | 15min |
| 31 | catalog | `registry_build.go` | Same sorted-build pattern 7× — extract generic helper | 15min |
| 32 | catalog | `registry_copy.go` | Same copyPtr pattern 7× — extract generic helper | 15min |
| 33 | id | `id.go:70` | `ULID()` uses misleading `struct{}` phantom type | 10min |
| 34 | id | `command_id.go` | Missing doc comments | 5min |
| 35 | query | `errors.go` | `ErrQueryNotSupported` vs command's `ErrHandlerNotFound` — inconsistent | 5min |
| 36 | command | `errors.go:30` | `ErrTypeAssertion` as `Corruption` should be `Rejection` | 5min |
| 37 | pebble | config types | Stuttering `Pebble` prefix on types in `pebble` package | 15min |
| 38 | turso | function names | Stuttering `Turso` prefix in `turso` package | 10min |
| 39 | event | `types.go:80` | `ParseUserAgent` doesn't actually parse — misleading name | 5min |

### P3 — Missing Tests

| # | Module | What's Missing | Effort |
|---|--------|----------------|--------|
| 40 | event/slice.go | Zero tests for `SliceFromVersion`, `SliceToVersion`, `FilterByTimestamp` | 30min |
| 41 | event/context.go | `deadlineCtx` untested | 15min |
| 42 | dispatcher | No test for `DispatcherWithCatalog`, no concurrent dispatch test | 30min |
| 43 | watermill | Zero subscriber tests | 30min |
| 44 | turso | Zero test coverage (entire module) | 60min |
| 45 | schema | No test for nil store/upcaster, no `LoadToTimestamp` test | 30min |
| 46 | listing | No test for `TombstonePolicy.String()`, `AggregateStatus.MarshalJSON()` | 15min |
| 47 | memory | No test for closed-store behavior on checkpoint/snapshot | 15min |
| 48 | integration/event | `event_sourcing_bdd_test.go` is 477 lines — split by topic | 15min |

### P4 — Example & Tool Issues

| # | Module | Issue | Effort |
|---|--------|-------|--------|
| 49 | example/user | Missing handlers for `UserDeleted`/`UserReborn` in projection | 15min |
| 50 | example/user | `catalog.go:20` uses event payload type for command | 10min |
| 51 | example/todo | Dead `CommandTypeError` in `mixin.go:40` | 5min |
| 52 | example/todo | Stale README references to `core/` and `cqrs-htmx` | 10min |
| 53 | example/saga | No test file | 30min |
| 54 | example/listing | No test file | 30min |
| 55 | example/user | `main.go:235` writes eventcatalog to working directory | 10min |

### P5 — Architecture & Deepening Opportunities

| # | Module | Issue | Effort |
|---|--------|-------|--------|
| 56 | catalog | Generic `buildSortedList`/`copyPtr` helpers to eliminate 14× duplication | 30min |
| 57 | memory | Extract `withRLock`/`withLock` helper to reduce repetitive lock patterns | 20min |
| 58 | projection | `runner_live.go:110` — `time.After` leaks timers | 10min |
| 59 | projection | `health.go:47` — `IsRunning` blocks with `context.Background()` | 10min |
| 60 | projection | `runner_live.go:107` — Uncapped exponential backoff can overflow | 10min |
| 61 | storage | SQL SELECT column list duplicated across 5+ queries — extract constant | 15min |
| 62 | pebble | `save.go:18` — `checkVersion` is O(n) for every Save | 30min |
| 63 | listing | `in_memory.go:97` — Stores ALL events per aggregate for tombstone | 20min |
| 64 | pebble | Redundant `Backend` switch (both cases identical) | 10min |
| 65 | turso | `doc.go:10` — `func _()` import hack | 5min |

---

## Execution Order (Pareto — 1% → 51%)

### Batch 1: Critical Safety (P0) — ~2h
Fix items 1-12. These are correctness bugs, data races, and security issues.

### Batch 2: Type Safety & Dead Code (P1) — ~2.5h
Fix items 13-26. File splits, error handling consistency, dead code removal.

### Batch 3: Duplication & Naming (P2) — ~2h
Fix items 27-39. Extract helpers, fix naming, reduce clone count.

### Batch 4: Test Coverage (P3) — ~4h
Fix items 40-48. Fill test gaps for critical modules.

### Batch 5: Examples & Tools (P4) — ~2h
Fix items 49-55. Ensure examples are correct and compile.

### Batch 6: Architecture Polish (P5) — ~3h
Fix items 56-65. Deepening opportunities, performance, cleanup.

---

## D2 Execution Graph

```d2
direction: down

title: V2.0.0 Release Plan

batch_1: P0 — Critical Safety {
  shape: rectangle
  style.fill: "#ff4444"
  fixes: "Data race, XSS, panics, immutability bug"
  hours: 2h
}

batch_2: P1 — Type Safety {
  shape: rectangle
  style.fill: "#ff8800"
  fixes: "File splits, error wrapping, dead code"
  hours: 2.5h
}

batch_3: P2 — Duplication & Naming {
  shape: rectangle
  style.fill: "#ffcc00"
  fixes: "Extract helpers, fix naming, 14×→1× patterns"
  hours: 2h
}

batch_4: P3 — Test Coverage {
  shape: rectangle
  style.fill: "#44cc44"
  fixes: "Missing tests in event, schema, watermill, turso"
  hours: 4h
}

batch_5: P4 — Examples & Tools {
  shape: rectangle
  style.fill: "#4488ff"
  fixes: "Broken examples, stale READMEs, missing tests"
  hours: 2h
}

batch_6: P5 — Architecture Polish {
  shape: rectangle
  style.fill: "#8844ff"
  fixes: "Deepening, performance, cleanup"
  hours: 3h
}

batch_1 -> batch_2 -> batch_3 -> batch_4 -> batch_5 -> batch_6

release: V2.0.0 Release {
  shape: diamond
  style.fill: "#00ff88"
}

batch_6 -> release
```
