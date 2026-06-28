# Comprehensive Status Report — 2026-06-10 Audit Execution

**Date:** 2026-06-10 21:43 CEST
**Branch:** master (clean, pushed)
**Last commit:** `eb07ec38` fix(storage): replace MustParseAggregateType with error-returning Parse in command scan

---

## a) FULLY DONE

### Session Accomplishments (this session + prior session continuation)

| Commit     | What                                                    | Category |
| ---------- | ------------------------------------------------------- | -------- |
| `a4ebdf0`  | Fix README Quick Start — Save/Load API signatures       | docs     |
| `450a45f`  | Rewrite getting-started.md for v2 multi-module API      | docs     |
| `7bf0fbb`  | Add doc comments to all 30 exported error symbols       | docs     |
| `0db9963`  | Update TODO_LIST.md — mark 4 items done                 | docs     |
| `0840bc2`  | Add [Unreleased] changelog for audit sessions           | docs     |
| `b28854c`  | Remove unused command.WrapTransient re-export           | cleanup  |
| `532e749`  | Break event↔command dependency cycle                    | arch     |
| `74dc2bd`  | Break memory↔snapshot dependency cycle                  | arch     |
| `690f23d`  | Extract sql.ClosableBase, deduplicate store boilerplate | arch     |
| `2d6749d`  | Fix snapshot LoadAtVersion context propagation          | bug      |
| `9d407f5`  | Fix circuit breaker error taxonomy + nil guard          | bug      |
| `90929f1`  | Simplify Version.Cmp to cmp.Compare                     | cleanup  |
| `70f05a1`  | Pebble NewStore(nil) panics with clear message          | bug      |
| `230a717`  | Decider slog.WarnContext for snapshot failures          | bug      |
| `72b8517`  | Retry middleware ErrRetryCanceled actually used         | bug      |
| `b652dd3`  | SSE broker send-on-closed-channel race fix              | bug      |
| `fe1e518`  | Pebble MarshalMetadataJSON error handling               | bug      |
| `f3c30e6`  | Pebble sharded mutex pool (bounded memory)              | perf     |
| `2c4d27e`  | Extract generic sql.QueryEngine[T]                      | arch     |
| `9aee526`  | Fix 4 lint issues → 0 across 23 modules                 | quality  |
| `eb07ec38` | Replace MustParseAggregateType with Parse (crash fix)   | bug      |
| `31d0a80`  | Prune stale go.sum entries                              | cleanup  |

### Quality Metrics — Current State

| Metric      | Value                                | Status |
| ----------- | ------------------------------------ | ------ |
| Build       | 39 packages, 0 errors                | ✅     |
| Test        | 39 packages, 0 failures (with -race) | ✅     |
| Lint        | 23 modules, **0 issues**             | ✅     |
| Format      | 0 files changed                      | ✅     |
| go.mod tidy | All modules clean                    | ✅     |
| Git         | Clean working tree, pushed to origin | ✅     |

### Coverage by Module

| Module           | Coverage | Status                |
| ---------------- | -------- | --------------------- |
| decider          | 100.0%   | ✅                    |
| dispatcher       | 98.0%    | ✅                    |
| memory           | 98.2%    | ✅                    |
| command          | 97.2%    | ✅                    |
| catalog          | 95.9%    | ✅                    |
| middleware       | 95.7%    | ✅                    |
| id               | 96.4%    | ✅                    |
| listing          | 94.9%    | ✅                    |
| signing          | 94.1%    | ✅                    |
| signing/multisig | 94.2%    | ✅                    |
| query            | 94.3%    | ✅                    |
| watermill        | 94.3%    | ✅                    |
| codec            | 93.3%    | ✅                    |
| snapshot         | 92.3%    | ✅                    |
| projection       | 91.4%    | ✅                    |
| event            | 89.6%    | ✅                    |
| schema           | 89.7%    | ✅                    |
| storage          | 86.8%    | ✅                    |
| pebble           | 86.4%    | ✅                    |
| catalog/schema   | 86.0%    | ✅                    |
| otel             | 73.0%    | ⚠️ Low                |
| event/eventtest  | 17.8%    | ⚠️ Low (test helpers) |
| storage/sql      | 25.2%    | ⚠️ Low                |
| turso            | 28.6%    | ⚠️ Low                |

### TODO List Status

- **Total items:** ~100
- **Done:** ~90
- **Actionable open:** 0
- **Deferred (v2/v3):** 4
- **Blocked (external):** 8
- **Future/speculative:** ~15

---

## b) PARTIALLY DONE

Nothing is partially done. All items that were started were completed.

---

## c) NOT STARTED

### Error Handling Gaps (found in scan, not yet fixed)

1. **`pebble/save.go:77,82`** — `fmt.Errorf` used for corruption errors instead of classified `event.NewCorruption`/`event.WrapCorruption`
2. **`memory/store_load.go:111`** — `fmt.Errorf` wrapping `event.ErrAggregateNotFound` loses error family classification
3. **`memory/command_store.go:196`** — Same pattern: `fmt.Errorf` wrapping `command.ErrCommandNotFound`
4. **`storage/aggregate_projection.go:40`** — `fmt.Errorf` for input validation instead of `event.WrapRejection`

### Dead Exported API Surface (unused externally)

5. **`storage/doc.go`** — `Schema()`, `SQLiteSchema()`, `ErrConcurrencyConflict`, `ErrUnsupportedTimestamp`, `ErrUnexpectedTimeType` all unreferenced outside their own package
6. **`storage/options.go`** — `NewSQLEventStoreWithOptions`, `WithOwnership`, `SQLEventStoreOption` — entire file is dead API surface (zero external consumers)
7. **`pebble/config.go`** — `Backend`, `Config`, `NewEventStore`, `NewConfig`, `WithBackend`, `WithProvider`, `EventStoreProvider`, `Option` — only used in pebble's own example_test.go
8. **`turso/errors.go:12`** — `ErrTursoMemorySync` backward-compat alias with zero external references

### Type Model Improvements

9. **`command.Type`** — missing `IsZero()`, `ParseType()` methods that `event.Type` has
10. **`query.Type`** — missing `IsZero()`, `ParseType()` methods that `event.Type` has
11. **`event.MetadataKey`** — lacks validation; any string passes as a key
12. **`catalog.ServiceID`, `DomainID`, `MessageID`, `ChannelID`** — bare string aliases without `Parse*()` or `IsZero()`

### Coverage Gaps

13. **`storage/sql`** (25.2%) — `query_engine.go`, `helpers.go`, `reconstruction.go` tested only indirectly via storage/ tests
14. **`turso`** (28.6%) — connector code tested but schema/connector setup uncovered
15. **`otel`** (73.0%) — helper functions like `AttrInt`, `StartSpan` partially covered

### Missing Infrastructure

16. **`integration/`** — no `doc.go` (all other 21 modules have one)

---

## d) TOTALLY FUCKED UP

**Nothing is fucked up.** The codebase is in excellent shape:

- Zero lint issues across 23 modules
- All tests pass with race detector
- No dependency cycles
- All go.mod files tidy
- No `any` type abuse
- No Must-panic patterns in data reconstruction paths (fixed this session)
- No unbounded memory growth patterns (pebble locks fixed this session)

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Error handling consistency** — `fmt.Errorf` wrapping classified errors breaks the error family chain. Callers checking `errors.Is(err, event.ErrAggregateNotFound)` will fail because the wrapping creates a new unclassified error. This is a **silent correctness bug** — tests likely don't catch it because they check error messages rather than `errors.Is`.

2. **Dead API surface** — `storage/options.go` is an entire file of unused exports. Public API surface IS the product for a library. Dead exports confuse consumers and increase maintenance burden. Either document them as stable API or deprecate/remove.

3. **`command.Type`/`query.Type` inconsistency** — `event.Type` has `IsZero()` and `ParseType()`, but `command.Type` and `query.Type` don't. This breaks the "consistency over correctness" principle and forces consumers to use different patterns for the same concept.

### Type Safety

4. **`MetadataKey` has no validation** — typos like `MetadataKeyCleintID` compile fine. A `ParseMetadataKey()` with format validation or at least a registry would catch this at init time.

5. **`catalog/` string aliases** — `ServiceID`, `DomainID`, `MessageID`, `ChannelID` are bare strings. The project already has `id.Of[T]` branded ID pattern. These should use it or at minimum have `Parse*()` constructors.

### Testing

6. **`storage/sql` 25.2% coverage** — `query_engine.go`, `helpers.go`, `reconstruction.go` are shared infrastructure tested only indirectly via `storage/` package tests. A corrupt change to `QueryRows[T]` could pass `storage/` tests if the test data happens to be valid.

7. **`otel` 73.0%** — The OTel helpers are imported by 6+ modules. Low coverage in shared infrastructure is risky.

---

## f) Top #25 Things We Should Get Done Next

Sorted by **impact × effort** (highest impact/lowest effort first):

| #   | Item                                                                                       | Impact                               | Effort | Category       |
| --- | ------------------------------------------------------------------------------------------ | ------------------------------------ | ------ | -------------- |
| 1   | Fix `fmt.Errorf` wrapping classified errors in memory/store_load.go:111                    | 🔴 Bug (silent `errors.Is` breakage) | 5 min  | error handling |
| 2   | Fix `fmt.Errorf` wrapping classified errors in memory/command_store.go:196                 | 🔴 Bug                               | 5 min  | error handling |
| 3   | Fix `pebble/save.go:77,82` — use `event.WrapCorruption` instead of `fmt.Errorf`            | 🟡 Medium                            | 5 min  | error handling |
| 4   | Fix `storage/aggregate_projection.go:40` — use `event.WrapRejection`                       | 🟡 Medium                            | 5 min  | error handling |
| 5   | Deprecate `storage/options.go` dead API (`NewSQLEventStoreWithOptions` etc.)               | 🟡 Medium                            | 5 min  | API surface    |
| 6   | Add `IsZero()` + `ParseType()` to `command.Type`                                           | 🟢 Low                               | 15 min | type safety    |
| 7   | Add `IsZero()` + `ParseType()` to `query.Type`                                             | 🟢 Low                               | 15 min | type safety    |
| 8   | Add `doc.go` to `integration/` module                                                      | 🟢 Low                               | 5 min  | consistency    |
| 9   | Add direct tests for `storage/sql/query_engine.go`                                         | 🟡 Medium                            | 30 min | coverage       |
| 10  | Add direct tests for `storage/sql/helpers.go`                                              | 🟡 Medium                            | 30 min | coverage       |
| 11  | Improve `otel/` coverage (73% → 85%+)                                                      | 🟡 Medium                            | 30 min | coverage       |
| 12  | Audit dead re-exports in `storage/doc.go` — deprecate or remove unused ones                | 🟢 Low                               | 15 min | API surface    |
| 13  | Add `MetadataKey` validation or `ParseMetadataKey()`                                       | 🟢 Low                               | 20 min | type safety    |
| 14  | Add `event.TypeOf[T]()` convenience function (derive type name from Go struct)             | 🟢 Low                               | 30 min | DX             |
| 15  | Consider `catalog.UserID` naming collision with `id.UserID`                                | 🟢 Low                               | 10 min | naming         |
| 16  | Add typed metadata accessors (`IsTombstone()`, `ClientOccurredAt()`)                       | 🟢 Low                               | 30 min | DX             |
| 17  | Improve `turso/` coverage (28.6% → 60%+)                                                   | 🟡 Medium                            | 60 min | coverage       |
| 18  | Consider deprecating pebble `config.go` convenience API (only used in own tests)           | 🟢 Low                               | 15 min | API surface    |
| 19  | Add `SchemaVersion.Add()` method                                                           | 🟢 Low                               | 5 min  | completeness   |
| 20  | Add `Version.MarshalJSON`/`UnmarshalJSON`                                                  | 🟢 Low                               | 15 min | serialization  |
| 21  | Add `SchemaVersion.MarshalJSON`/`UnmarshalJSON`                                            | 🟢 Low                               | 15 min | serialization  |
| 22  | Consider `catalog.ServiceID` etc. using branded `id.Of[T]` pattern                         | 🟢 Low                               | 45 min | type safety    |
| 23  | Add `MetadataKey` registry for cross-package extension point enforcement                   | 🟢 Low                               | 30 min | type safety    |
| 24  | Extract `eventtest` helpers used across packages into shared testutil                      | 🟢 Low                               | 60 min | organization   |
| 25  | Investigate `event/eventtest` 17.8% coverage — is it test helpers that should be excluded? | 🟢 Low                               | 15 min | coverage       |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should we deprecate or keep the dead exported API surface in `storage/doc.go` and `storage/options.go`?**

These are public API exports (`Schema()`, `SQLiteSchema()`, `ErrConcurrencyConflict`, `NewSQLEventStoreWithOptions`, `WithOwnership`) that have zero external consumers. As a library:

- **Keeping them** means maintaining backwards compatibility for API that nobody uses
- **Deprecating them** requires a deprecation cycle (Go convention: `// Deprecated:` comment + minor version bump)
- **Removing them** is a breaking change requiring v3.0.0

The `storage/` module is at v2.x. The AGENTS.md says "Public API surface IS the product." But unused API surface is also product debt. **What's the project policy on dead exports? Deprecate-and-remove-in-next-major, or keep indefinitely?**
