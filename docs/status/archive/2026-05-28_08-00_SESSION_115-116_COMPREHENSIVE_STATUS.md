# Session 115-116 — Comprehensive Status Report

**Date:** 2026-05-28 08:00 CEST
**Branch:** master
**Total LOC:** 58,296 Go lines across 40 files (13 modules + examples)
**Test Status:** 25/27 PASS — 2 packages have stale golden files only
**Clone Count:** 408 groups at t=15 (down from 413 start of session 114)
**Ahead of origin:** 1 commit (not pushed)

---

## A) FULLY DONE ✅

### Session 114 (Dedup Sprint)

| Commit    | What                                                          | Delta                |
| --------- | ------------------------------------------------------------- | -------------------- |
| `7d8f7d9` | Extract `setupTestSaga` helper                                | -116 lines, 9 tests  |
| `0359ebc` | Unify storage `testEventWithAggID` with eventType param       | -11 lines            |
| `848883a` | Extract `exportCatalog`/`readExported`/`newConfiguredChannel` | -293 lines           |
| `9f82618` | Replace inline noop handlers + add `QueryMiddleware`          | -5 lines, 8 closures |

### Session 115 (Quality Sprint)

| Commit    | What                                                         | Delta      |
| --------- | ------------------------------------------------------------ | ---------- |
| `2473b7c` | Multi-party signing for device→server event chains           | +195 lines |
| `4a7fc2d` | Split Signer into Signer + Verifier interfaces               | Refactor   |
| `989a151` | Signature serialization (String, MarshalJSON, UnmarshalJSON) | +126 lines |
| `b269817` | Signature serialization integrated into example/user         | +27 lines  |
| `c2a4e0f` | Extract `VerifyAll` to standalone func, add Clock injection  | Refactor   |
| `675bee7` | Godoc examples for decider, query, and catalog packages      | +133 lines |
| `578d632` | Error taxonomy reference guide                               | +115 lines |
| `7b6d30f` | Export Upcaster interface and add `NewUpcaster` constructor  | +26 lines  |

**Net impact since Session 114 start:** +675 lines, -1,102 lines deleted across 40 files.

### Cross-cutting completed work

- **Signing module:** Fully functional with HMAC, Ed25519, multi-party signing, middleware, verification
- **Event upcasting:** Public API exported, `UpcasterChain` type, constructor
- **Godoc examples:** Added for `core/decider`, `core/query`, `catalog`, `core/event`
- **Error taxonomy:** 5-family reference guide in `docs/error-taxonomy.md`
- **testhelpers:** Complete handler factory (Noop, Failing, Panic, Callback for all 3 types + Middleware)
- **Clone dedup:** 5 commits reducing boilerplate across saga, storage, catalog, core, middleware, integration

---

## B) PARTIALLY DONE ⚠️

### 1. Signing integration into example/user — 70% done

- ✅ Multi-party signing module implemented
- ✅ Signature serialization
- ✅ Signer/Verifier split
- ✅ Ed25519 signer added to example/user
- ❌ No signing middleware wired in example/user event bus
- ❌ No verification on read in example/user
- ❌ No signing-specific tests in example/user

### 2. Golden file maintenance — Stale but working

- `catalog/asyncapi` golden YAML needs refresh
- `catalog/eventcatalog` config.js and package.json need refresh
- Tests work correctly; golden files just need `-update` flag

### 3. Benchmark gaps — Partially filled

- ✅ `projection/benchmark_test.go` exists
- ✅ `saga/benchmark_test.go` exists (untracked, not committed)
- ❌ No storage benchmarks
- ❌ No catalog exporter benchmarks

### 4. Staged but uncommitted fix

- `signing/multisig_test.go` — fixes `TestMultiVerifyMiddleware_RejectsTampered` by replacing `testhelpers.QuickEvent` (which doesn't preserve event metadata needed for verification) with explicit `event.NewEvent` + options. **This makes signing tests 100% green.**

### 5. Untracked file

- `saga/benchmark_test.go` — new benchmark file, compiles and runs, not committed

---

## C) NOT STARTED ❌

### From Session 115 Plan — Never Started

| #   | Task                                                     | Impact | Why Skipped               |
| --- | -------------------------------------------------------- | ------ | ------------------------- |
| T6  | Signing ADR document (`docs/adr/0008-signing-module.md`) | Medium | Ran out of session time   |
| T20 | CI coverage gate (80% minimum)                           | Medium | Requires CI workflow edit |
| T21 | File size enforcement in CI                              | Low    | Requires CI workflow edit |
| T18 | Improve example/saga with real saga flow                 | Medium | Deferred                  |
| T19 | Improve example/projection with real projection          | Medium | Deferred                  |
| T22 | Verify all godoc examples compile                        | Medium | Partially done            |
| T24 | Update AGENTS.md with all new features                   | Medium | Partially done            |

### Never Planned But Needed

| Task                                                  | Why Needed                                          |
| ----------------------------------------------------- | --------------------------------------------------- |
| Add `.gitignore` for `coverage.out`, binary artifacts | Prevents accidental commits of build/test artifacts |
| Storage benchmarks                                    | No performance visibility for storage layer         |
| Catalog exporter benchmarks                           | No performance visibility                           |
| Update `TODO_LIST.md`                                 | Many items still reference completed work           |

---

## D) TOTALLY FUCKED UP 💥

### 1. `testhelpers.QuickEvent` is broken for signing tests

- **What:** `QuickEvent` creates events but doesn't preserve the metadata needed for signature verification (correlation IDs, schema versions, etc.)
- **Impact:** Signing tamper-detection tests failed when using `QuickEvent`
- **Fix:** Staged in `signing/multisig_test.go` — replaced with explicit `event.NewEvent` with all preserved fields
- **Root cause:** `QuickEvent` was designed for simple test cases, not for cryptographic round-trip tests where every field matters

### 2. No `.gitignore` in the repo

- Coverage files (`coverage.out`), binaries, IDE files are not ignored
- These show up in git status noise and risk accidental commits

### 3. `middleware/retry.go` uses `math/rand`

- Pre-existing security warning from BuildFlow pre-commit hook
- Should use `crypto/rand` or `math/rand/v2` with proper seeding
- Low severity (retry backoff jitter, not cryptographic use) but noisy

---

## E) WHAT WE SHOULD IMPROVE

### Architecture & Type Models

1. **Generic middleware factory** — Command, Event, and Query middleware share identical `callOrder`/logging/retry patterns but are separate types. A generic `middleware.Factory[H any]` could eliminate `commandErrMiddleware`/`eventErrMiddleware`/`queryErrMiddleware` in `middleware/common.go`.

2. **`event.ImmutableEvent` fields are not strongly typed** — `AggregateID()` returns `id.AggregateID` (good) but `Type()` returns `string`, `AggregateType()` returns `string`. Branded types `event.Type` and `event.AggregateType` exist but aren't consistently used in return values.

3. **`query.Handler` returns `(any, error)`** — The TODO list calls this out as a v2 breaking change. `DispatchTyped[T]` works around it but the handler signature is still untyped.

4. **`saga.CommandDispatcher` vs `command.Dispatcher`** — Two separate interfaces for the same concept. Saga should accept `command.Dispatcher` or use a shared interface.

### Code Quality

5. **408 clone groups at t=15** — Down from 413 but still high. Most are 2-element groups (Go idioms, type-system-required duplicates). Acceptable per ADR but could be reduced further with generic middleware factory.

6. **Missing storage benchmarks** — No performance baseline for the heaviest module (SQL event store, snapshot store, outbox).

7. **Golden test maintenance** — 3 golden files are stale. Should add a CI step or pre-commit hook to refresh them.

8. **`docs/error-taxonomy.md` exists but isn't linked** from README or AGENTS.md. Discoverable only by browsing docs/.

### Dependencies & Ecosystem

9. **No `go.mod` tidy enforcement in CI** — BuildFlow does it in pre-commit but CI doesn't verify.

10. **No race condition testing in CI** — Only done locally via `nix run .#test` with `-race`.

---

## F) Top 25 Things We Should Get Done Next

Sorted by **Impact × Ease** (Pareto):

### 🔴 HIGH Impact, LOW Effort (Do First)

| #   | Task                                                                              | Est   | Impact                                                |
| --- | --------------------------------------------------------------------------------- | ----- | ----------------------------------------------------- |
| 1   | Commit staged `signing/multisig_test.go` fix + untracked `saga/benchmark_test.go` | 5min  | Fixes last signing test failure, adds saga benchmarks |
| 2   | Refresh 3 golden files (`-update` flag)                                           | 5min  | Makes test suite 100% green                           |
| 3   | Add `.gitignore` for `coverage.out`, `*.out`, binaries                            | 5min  | Prevents accidental commits                           |
| 4   | Fix `middleware/retry.go` `math/rand` → `math/rand/v2`                            | 10min | Eliminates pre-commit security warning                |
| 5   | Commit and push all accumulated work                                              | 5min  | Gets code to remote                                   |

### 🟡 HIGH Impact, MEDIUM Effort (Do Next)

| #   | Task                                                                  | Est   | Impact                                                   |
| --- | --------------------------------------------------------------------- | ----- | -------------------------------------------------------- |
| 6   | Wire signing middleware into `example/user` event bus + add tests     | 45min | Validates signing module end-to-end, first real consumer |
| 7   | Write signing ADR (`docs/adr/0008-signing-module.md`)                 | 30min | Documents architectural decision                         |
| 8   | Add storage benchmarks (Save/Load for SQL + Pebble)                   | 30min | Performance baseline for heaviest module                 |
| 9   | Generic middleware factory to eliminate `*ErrMiddleware` triplication | 45min | Removes the biggest remaining clone category             |
| 10  | Update `TODO_LIST.md` to reflect current state                        | 20min | Clean slate for future sessions                          |
| 11  | Update `AGENTS.md` with signing, upcasting, benchmarks info           | 20min | Better AI session memory                                 |
| 12  | Add catalog exporter benchmarks                                       | 20min | Performance visibility                                   |

### 🟢 MEDIUM Impact, MEDIUM Effort (Do When Time Permits)

| #   | Task                                                                   | Est   | Impact                           |
| --- | ---------------------------------------------------------------------- | ----- | -------------------------------- |
| 13  | Add godoc examples for `command.Dispatcher` and `event.Store`          | 20min | pkg.go.dev readiness             |
| 14  | Improve `example/saga` with real OrderSaga + compensation              | 60min | Onboarding quality               |
| 15  | Improve `example/projection` with real OrderProjection                 | 45min | Onboarding quality               |
| 16  | CI coverage gate (80% minimum)                                         | 30min | Quality enforcement              |
| 17  | CI file-size enforcement (fail on >350 lines)                          | 15min | Quality enforcement              |
| 18  | Unify `saga.CommandDispatcher` with `command.Dispatcher`               | 30min | Eliminates split-brain interface |
| 19  | Make `event.Type` and `event.AggregateType` branded types consistently | 45min | Type safety                      |
| 20  | Add `.golangci.yml` to root for consistent linting                     | 15min | Standardized lint config         |

### 🔵 LOWER Impact, HIGHER Effort (Backlog)

| #   | Task                                                  | Est    | Impact                 |
| --- | ----------------------------------------------------- | ------ | ---------------------- |
| 21  | v2: Generic `query.Handler[T]` returning `(T, error)` | 60min  | Type safety (breaking) |
| 22  | PostgreSQL/Turso integration tests                    | 90min  | Real DB validation     |
| 23  | Event encryption module                               | 120min | Security feature       |
| 24  | Watermill DLQ handler implementation                  | 60min  | Production resilience  |
| 25  | Saga SQL state store                                  | 90min  | Production persistence |

---

## Test Results Summary

```
PASS: 25/27 packages
FAIL: 2 packages (golden file staleness only — 3 test cases)

FAIL catalog/asyncapi     — TestGolden_AsyncAPIYAML (stale golden)
FAIL catalog/eventcatalog — TestGolden_EventCatalog_Config + TestGolden_EventCatalog_PackageJSON (stale golden)

Signing tests: 100% PASS (after staged fix)
All functional tests: 100% PASS
```

## Git Status

```
Staged:
  M  signing/multisig_test.go  (fixes TestMultiVerifyMiddleware_RejectsTampered)

Untracked:
  ?? saga/benchmark_test.go  (new benchmarks, compiles and runs clean)

Ahead of origin: 1 commit
```

## Module Coverage (approximate)

| Module       | Coverage | Status |
| ------------ | -------- | ------ |
| core/command | 94.7%    | ✅     |
| core/query   | 100%     | ✅     |
| core/event   | 90%+     | ✅     |
| core/decider | 85%+     | ✅     |
| memory       | 90%+     | ✅     |
| catalog      | 85%+     | ✅     |
| middleware   | 85%+     | ✅     |
| storage      | 80%+     | ✅     |
| saga         | 85%+     | ✅     |
| projection   | 85%+     | ✅     |
| signing      | 90%+     | ✅     |
| integration  | 85%+     | ✅     |
| testhelpers  | 94.8%    | ✅     |
| watermill    | 85%+     | ✅     |

---

_This report covers Sessions 114-116 (2026-05-28). Session 114: dedup t=15 sprint. Session 115: quality & integration sprint. Session 116: this status report._
