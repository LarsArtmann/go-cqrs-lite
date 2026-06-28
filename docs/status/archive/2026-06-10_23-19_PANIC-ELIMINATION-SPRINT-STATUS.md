# Status Report — go-cqrs-lite Panic Elimination Sprint

**Generated:** 2026-06-10 23:19
**Branch:** master
**Commits in this session:** 5
**Total Go LOC:** 75,252

---

## a) FULLY DONE ✅

### Must\* Panic Wrappers — ALL DELETED (17 functions)

Every exported `Must*` function that swallowed errors with `panic()` has been removed from the public API:

| Function                                                                            | Location                 | Callers at time of removal |
| ----------------------------------------------------------------------------------- | ------------------------ | -------------------------- |
| `id.MustParse[T]`                                                                   | id/id.go                 | ~177 test calls            |
| `id.MustParseAggregateID`                                                           | id/aggregate_id.go       | ~60+ test calls            |
| `id.MustParseUserID/EventID/CommandID/CorrelationID/CausationID/RequestID/ClientID` | id/\*.go                 | ~50 test calls             |
| `command.MustNew`                                                                   | command/command.go       | ~8 test/bench calls        |
| `query.MustNew`                                                                     | query/query.go           | ~6 test/bench calls        |
| `command.MustParseAggregateType`                                                    | command/aggregate_ref.go | ~4 test calls              |
| `event.MustParseAggregateType`                                                      | event/event.go           | 0 callers                  |
| `event.MustNewEvents`                                                               | event/batch.go           | 2 test calls               |
| `event.builder.MustBuild`                                                           | event/builder.go         | 2 test calls               |
| `snapshot.MustEveryNEvents`                                                         | snapshot/strategy.go     | ~7 test calls              |
| `integration/simulation.MustSerialize`                                              | simulation/generator.go  | 0 callers                  |
| `command.MustParseType`                                                             | command/command.go       | removed earlier            |
| `event.MustParseType`                                                               | event/event.go           | removed earlier            |
| `query.MustParseType`                                                               | query/query.go           | removed earlier            |

All test callers replaced with local `Parse` + error-handling helpers.

### Build & Test

- **All 35 test packages pass**
- **`go vet` clean**
- **`go build ./...` clean**

### Storage Build Fix

- Fixed broken `storage/command_store_save.go` — `trace.Span` → `cqrsotel.Span` in `withTx` method

---

## b) PARTIALLY DONE ⚠️

### Production Panics: 10 Remaining (down from 19)

| #   | Location                                 | Function                    | Category          | Prod Callers | Convertible? |
| --- | ---------------------------------------- | --------------------------- | ----------------- | ------------ | ------------ |
| 1   | event/types.go:121                       | `Version.Decrement()`       | Arithmetic guard  | **0**        | ✅ Trivial   |
| 2   | event/types.go:141                       | `Version.Sub(n)`            | Arithmetic guard  | **0**        | ✅ Trivial   |
| 3   | event/types.go:191                       | `SchemaVersion.Decrement()` | Arithmetic guard  | **0**        | ✅ Trivial   |
| 4   | signing/multisig/extract.go:89           | `VerifierMap()`             | Nil guard         | **0**        | ✅ Easy      |
| 5   | pebble/store.go:46                       | `NewStore()`                | Nil guard         | **0**        | ⚠️ Moderate  |
| 6   | middleware/circuit_breaker.go:96         | `circuitBreaker.allow()`    | Exhaustive switch | N/A          | Leave        |
| 7   | listing/in_memory.go:182                 | `applyTombstonePolicy()`    | Exhaustive switch | N/A          | Leave        |
| 8   | catalog/docserver/docserver.go:128       | `mustStaticFS()`            | Embed-time init   | N/A          | Leave        |
| 9   | catalog/internal/cattest/builders.go:220 | `StringSchema()`            | Test helper       | N/A          | Leave        |
| 10  | event/eventtest/handlers.go:44           | `PanicEventHandler()`       | Test helper       | N/A          | Leave        |

**Critical finding:** Items #1–5 have **zero production callers**. They are exclusively called from tests. Converting them to error returns is zero-risk.

### Error Taxonomy Gaps

- **Transient family:** 0 production uses — dead family in this codebase
- **Corruption family:** 1 use — `query.NewCorruption` for type assertion failure
- **samber/oops:** Not used anywhere
- **errorfamily/bridge:** Not used anywhere

### Error Re-export Facades

- `event/errors.go` (125 lines) and `command/errors.go` (122 lines) re-export nearly identical errorfamily APIs
- Internal code (dispatchers, stores) imports `errorfamily` directly, bypassing the facades
- Worst of both worlds: facade exists but isn't enforced

---

## c) NOT STARTED 📐

### Version Type API Inconsistency

- `Version.Add(n)` has **no underflow guard** — negative `n` silently produces negative Version
- `Version.Sub(n)` panics on underflow
- `Version.Decrement()` panics on zero
- Inconsistent: some arithmetic guards, some don't, and the guards use panic not error

### Remaining Panic → Error Conversions

- `Version.Decrement()` → `(Version, error)` — trivial, zero prod callers
- `Version.Sub()` → `(Version, error)` — trivial, zero prod callers
- `SchemaVersion.Decrement()` → `(SchemaVersion, error)` — trivial, zero prod callers
- `VerifierMap()` → `(map[Actor]signing.Verifier, error)` — easy, zero prod callers
- `pebble.NewStore()` → `(*EventStore, error)` — moderate, 9 test callers

### FEATURES.md & AGENTS.md Updates

- FEATURES.md still lists `MustNew panic helper` as a feature
- AGENTS.md still mentions Must\* functions
- docs/api_surface.txt references deleted functions

### Code Quality

- `storage/sql` at 37.4% coverage
- 2 unrelated golden test mismatches fixed (middleware health-check, codec json_encode)
- Unrelated test file changes in working tree from prior sessions (catalog, decider, signing)

---

## d) TOTALLY FUCKED UP 💀

### Script-Driven Refactor Was Painful

The automated Python scripts for replacing Must\* callers had multiple bugs:

1. **Wrong return variable names** — `return s` instead of `return cmd/strategy`
2. **Duplicate helpers** — script added helpers to every file instead of once per package
3. **Generic type inference** — `mustParse[T]` couldn't infer T from unqualified calls
4. **Type mismatch** — `AggregateID = cbid.ID[AggregateMarker, string]` ≠ `Of[T] = cbid.ID[T, ulid.ULID]`
5. **Orphaned function bodies** — some removals stripped the `func` line but left the body

Each bug required manual diagnosis and fix. The session went through ~15 build-fix-verify cycles.

**What I should have done differently:**

1. Written the replacement script with package awareness (one helper per package, not per file)
2. Tested the script on one package first, verified it compiled, then applied to all
3. Recognized that `AggregateID` is a different type than `Of[AggregateMarker]` earlier
4. Used `go build ./...` as the verification gate instead of relying on LSP diagnostics

### No Pre-Flight Analysis

I didn't fully analyze the caller graph before starting. If I had known that all 3 Version methods and VerifierMap had zero production callers, I would have converted them to error returns in the same session instead of leaving them as panics.

---

## e) WHAT WE SHOULD IMPROVE 📈

### 1. Error Return Conversions for Remaining Panics

The 5 convertible panics (#1–5 above) all have zero production callers. Convert them to return `(T, error)` using the existing errorfamily taxonomy. This is the single highest-impact change available.

### 2. Consistent Arithmetic Guards on Version

Either all arithmetic methods guard (Add, Sub, Decrement, Increment) or none do. Currently Add() has no guard while Sub() panics. Pick one strategy:

- **Option A:** All methods return `(Version, error)` — explicit but verbose
- **Option B:** Unsafe methods (current behavior) + `Try*` variants — backward-compatible
- **Option C:** Make Version unsigned — prevent negative at the type level

### 3. Use go-error-family Properly

- Drop the re-export facades (event/errors.go, command/errors.go) and import errorfamily directly everywhere, OR enforce facade usage consistently
- Consider errorfamily/bridge + samber/oops for rich context (stack traces, structured metadata)
- Actually use the Transient family for retryable errors

### 4. Type Model Improvements

- `AggregateID` uses `string` while all other IDs use `ulid.ULID` — this is a split brain in the ID type system
- `Version` is `int` — could be `uint` or a range-checked type to eliminate negative-version bugs at compile time
- `SchemaVersion` minimum is 1 but the type allows 0 — could use a constructor that enforces the invariant

### 5. Developer Experience

- Add `doc.go` with pkg.go.dev examples for remaining ~10 modules
- Add module READMEs for remaining ~10 modules
- The test helper functions (`parseAggID`, `mustNewCmd`, etc.) added to ~30 test files could be consolidated into a shared `eventtest` or new `testutil` module

---

## f) Top 25 Things We Should Get Done Next

Sorted by **Impact × Feasibility / Work Required** (highest first):

| #   | Task                                                                         | Impact | Work  | Risk                  |
| --- | ---------------------------------------------------------------------------- | ------ | ----- | --------------------- |
| 1   | Convert `Version.Decrement()` → `(Version, error)` with errorfamily taxonomy | High   | 30min | Zero — 0 prod callers |
| 2   | Convert `Version.Sub()` → `(Version, error)` with errorfamily taxonomy       | High   | 30min | Zero — 0 prod callers |
| 3   | Convert `SchemaVersion.Decrement()` → `(SchemaVersion, error)`               | High   | 30min | Zero — 0 prod callers |
| 4   | Convert `VerifierMap()` → `(map, error)` with errorfamily taxonomy           | Medium | 30min | Zero — 0 prod callers |
| 5   | Convert `pebble.NewStore()` → `(*EventStore, error)`                         | Medium | 1h    | Low — 9 test callers  |
| 6   | Update FEATURES.md — remove MustNew feature entry                            | Low    | 10min | None                  |
| 7   | Update AGENTS.md — reflect Must\* removal, add note about remaining panics   | Low    | 15min | None                  |
| 8   | Regenerate `docs/api_surface.txt`                                            | Low    | 5min  | None                  |
| 9   | Update CHANGELOG.md with Must\* removal entry                                | Low    | 10min | None                  |
| 10  | Fix `Version.Add()` underflow gap — add guard matching Sub() behavior        | Medium | 30min | Low                   |
| 11  | Consolidate error re-exports: pick facade OR direct import, not both         | Medium | 2h    | Medium — many files   |
| 12  | Add `storage/sql` tests to improve 37.4% → 80%+ coverage                     | High   | 2h    | None                  |
| 13  | Consider `errorfamily/bridge` for structured error context                   | Medium | 2h    | Low — additive        |
| 14  | Fix `AggregateID` type split brain (string vs ulid.ULID base)                | High   | 4h    | High — breaking       |
| 15  | Consider making `Version` unsigned to prevent negatives at compile time      | Medium | 3h    | High — breaking       |
| 16  | Remove Pebble dead `Backend` type/constants                                  | Low    | 5min  | None                  |
| 17  | Update TODO_LIST.md — all items are done/blocked                             | Low    | 30min | None                  |
| 18  | Add module READMEs for remaining ~10 modules                                 | Medium | 2h    | None                  |
| 19  | Add `doc.go` with pkg.go.dev examples for remaining ~10 modules              | Medium | 2h    | None                  |
| 20  | Consolidate test helpers from ~30 files into shared testutil                 | Medium | 3h    | Low — test-only       |
| 21  | Add PostgreSQL integration tests for storage/                                | High   | 4h    | None                  |
| 22  | Add Pebble Journal/SeekableJournal implementation                            | Medium | 4h    | None                  |
| 23  | Add `go test -race` to CI verification                                       | Medium | 30min | None                  |
| 24  | Review and possibly adopt `samber/oops` for error enrichment                 | Medium | 2h    | Low                   |
| 25  | Add catalog diff/breaking-change detection tool                              | Medium | 4h    | None                  |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**Should `AggregateID` use `ulid.ULID` as its base type (like every other ID) or keep `string`?**

Currently:

```go
type AggregateID = cbid.ID[AggregateMarker, string]     // string-based
type EventID    = cbid.ID[eventMarker]                    // = cbid.ID[eventMarker, ulid.ULID] — ULID-based
type UserID     = cbid.Of[userMarker]                     // = cbid.ID[userMarker, ulid.ULID] — ULID-based
```

`AggregateID` is the ONLY ID type that uses `string` instead of `ulid.ULID`. This causes:

- `mustParse[AggregateMarker]` returns `Of[AggregateMarker]` = `cbid.ID[AggregateMarker, ulid.ULID]` — different type than `AggregateID`
- The `DeriveAggregateID()` function returns a string-based ID from SHA-256 hex — this works with `string` base but would need design thought with `ulid.ULID` base
- Every other ID in the system is ULID-based

**Should we unify `AggregateID` to use `ulid.ULID` base?** This would be a breaking change for consumers who derive aggregate IDs from non-ULID sources (hashes, UUIDs, etc.). But it would eliminate the type split brain and make all ID types consistent.

Alternatively: keep `AggregateID` as string-based (it's more flexible for derived IDs) but document this as intentional.

---

## Session Metrics

| Metric                       | Value                            |
| ---------------------------- | -------------------------------- |
| Production panics eliminated | 9 → 0 (Must\* wrappers)          |
| Remaining production panics  | 10 (5 convertible, 5 acceptable) |
| Files modified               | ~60                              |
| Test packages passing        | 35/35                            |
| Coverage                     | 92.8% avg across library modules |
| Build                        | Clean                            |
| Vet                          | Clean                            |
| Race                         | Not tested yet                   |
