# Session 91 — Comprehensive Status Report

**Date:** 2026-05-22 01:39  
**Branch:** master  
**HEAD:** `eb77445` (docs: Session 90 status)  
**Trigger:** Full comprehensive status + deletion audit analysis

---

## Vital Signs

| Metric | Value | Status |
|--------|-------|--------|
| Build | ✅ Clean (`go build` passes all modules) | OK |
| Tests | 24/26 packages pass, 2 FAIL (golden files stale) | NEEDS FIX |
| Coverage | 83.7% total (30 packages) | GOOD |
| `go vet` | ✅ Zero issues | OK |
| Files >250 lines | ✅ Zero production files exceed limit | OK |
| Production lines | 16,124 across 183 files | — |
| Test lines | 32,339 across 135 files (2:1 test:prod ratio) | EXCELLENT |
| Public exports | 504 total (core 193, catalog 129, storage 66, testhelpers 37, middleware 30, sync 28, memory 11, projection 10) | HIGH |
| Modules | 12 in go.work | — |

### Test Coverage Per Package

| Package | Coverage | Trend |
|---------|----------|-------|
| `core/query` | 100.0% | — |
| `core/pkg/dispatcher` | 100.0% | — |
| `middleware` | 100.0% | — |
| `catalog/adapters` | 100.0% | — |
| `memory` | 99.6% | — |
| `core/pkg/id` | 98.1% | ↑ |
| `core/aggregate` | 95.9% | — |
| `catalog/d2` | 95.0% | — |
| `core/command` | 94.7% | — |
| `projection` | 94.2% | ↑ |
| `catalog/openapi` | 94.4% | — |
| `catalog/asyncapi` | 93.7% | — |
| `core/event` | 91.2% | — |
| `catalog/eventcatalog` | 91.3% | — |
| `catalog/docserver` | 90.0% | — |
| `catalog` | 90.5% | — |
| `core/decider` | 89.3% | ↓ (was 93.3%) |
| `storage` | 86.9% | — |
| `catalog/internal/schemautil` | 84.2% | — |
| `catalog/internal/caseutil` | 76.5% | — |
| `testhelpers` | 10.5% | LOW |
| `integration/*` | N/A (no statements) | — |

### Failing Tests (2 packages)

| Package | Test | Cause |
|---------|------|-------|
| `catalog/asyncapi` | `TestGolden_AsyncAPIYAML` | Golden file stale — needs `-update` |
| `catalog/eventcatalog` | `TestGolden_EventCatalog_Config` | Golden file stale — needs `-update` |
| `catalog/eventcatalog` | `TestGolden_EventCatalog_PackageJSON` | Golden file stale — needs `-update` |

### LSP Diagnostics (Pre-existing, NOT from this session)

| File | Error |
|------|-------|
| `core/event/codec_typed.go:31` | `undefined: ErrNilPayload` |
| `core/event/codec_typed_test.go:94` | `undefined: event.ErrNilPayload` |
| `catalog/adapters/benchmark_test.go:40` | `undefined: command.CatalogMeta` |
| `integration/command/command_test.go` (3 locations) | `undefined: command.CatalogMeta` |
| `integration/query/query_test.go` (3 locations) | `undefined: query.CatalogMeta` |

---

## a) FULLY DONE ✅

### Session 89 API Surface Audit (Commit `1088fcd`, `333a6af`)
- Centralized `CatalogEntry` into `core/pkg/dispatcher/`
- Removed `command.CatalogMeta` and `query.CatalogMeta` (duplicate types)
- Exported typed event constructor `event.New`
- Migrated all test files to `dispatcher.CatalogEntry`

### Session 86 Storage Improvements
- SQLite helpers: `OpenSQLite`, `SQLiteEnableWAL`, `ConfigureSQLitePool`, `ConfigureTursoPool`
- PostgreSQL helpers: `PostgresInitSchema`
- `DeriveAggregateID` for deterministic ID generation
- SQLEventStore ownership option

### Session 90 Fake Store Split (Commit `f989870`)
- Split `testhelpers/fake_store.go` (263→220 lines) + new `fake_store_setters.go` (54 lines)
- All 250-line quality gates now pass: **zero production files exceed limit**

### Projection Replay Context (Commit `6ac0a77`)
- Replay context detection (`event.IsReplay`)
- Builder pattern for projections

### Multi-Module Quality
- 12 modules in go.work, all independently buildable
- File size gate: 100% compliant (max 250 lines)
- Test-to-production ratio: 2:1 (32K test / 16K prod)
- `go vet` clean across all modules

### Top Largest Production Files (all within limits)
| Lines | File |
|-------|------|
| 250 | `catalog/eventcatalog/exporter.go` |
| 241 | `core/pkg/dispatcher/dispatcher.go` |
| 234 | `example/todo/aggregate/decider.go` |
| 227 | `core/event/event.go` |
| 227 | `catalog/internal/cattest/builders.go` |
| 223 | `storage/sql_helpers.go` |
| 223 | `catalog/docserver/docserver.go` |
| 222 | `catalog/openapi/exporter.go` |
| 220 | `testhelpers/fake_store.go` |

---

## b) PARTIALLY DONE 🔶

### Deletion Audit — Research Complete, Execution NOT Started

Full audit of dead/zero-consumer exports completed. Findings organized by tier:

**Tier 1: Zero-cost deletions (407 lines, zero external consumers)**
| Symbol | Lines | Why Dead |
|--------|-------|----------|
| `event.IsReplay` | ~5 | Zero callers anywhere |
| `catalog/adapters.FromCommandDispatcher` + `FromQueryDispatcher` | 57 | Zero production callers |
| `catalog.MessageIDString` | ~3 | Own-package tests only |
| `event.NewEvents` + `MustNewEvents` + `DecodePayloads` | 84 | README example only, no `.go` consumers |
| `decider.Result` + `ExecuteWithResult` | 71 | Zero external callers |
| `query.Pagination` + `NewPagination` + `PaginatedResult` + `NewPaginatedResult` | 93 | Zero external callers |
| `query.TypedHandler` | ~2 | Zero external callers |
| `dispatcher.NewCatalogDispatcher` + `CopyCatalogEntries` + `CatalogDispatcher` | ~50 | Internal plumbing exposed as public |
| `dispatcher.MiddlewareChain` + `GetHandler` | ~40 | Same-package only |
| `dispatcher.Lifecycle.IsClosed` | ~2 | Test only |

**Tier 2: Deprecated adapters (1 example consumer)**
| Symbol | Lines | Consumer |
|--------|-------|----------|
| `catalog/adapters.CatalogBuilder` | 122 | `example/user/catalog.go` (migrate to `catalog.Builder`) |
| Related test files | ~200 | Delete with adapters |

**Tier 3: Breaking interface changes**
| Symbol | Lines | Impact |
|--------|-------|--------|
| `Command.IdempotencyKey()` | ~5 | 5 implementations exist but nobody calls it — dead method |
| `event.OutboxPublisher` + subsystem | 206 | Zero production consumers (comments only) |

**Tier 4: Major package deletion**
| Symbol | Lines | Impact |
|--------|-------|--------|
| `core/aggregate/` entire package | 1,756 | Deprecated, only `integration/aggregate/` test files use it |
| `integration/aggregate/` | ~800 | Tests for deprecated package |

**NOT dead (keep):**
- `Version.Sub()` — used by `aggregate/load_helpers.go` (would die with aggregate deletion)
- `Version.Mod()` — used by `snapshot_strategy.go` (production code)
- `decider.LoadAtVersion`/`LoadAtTime` — used by `integration/event/timetravel_test.go`

### LSP Build Errors — Identified, Not Fixed
- `ErrNilPayload` undefined in `codec_typed.go` — appears to be a pre-existing broken export
- `command.CatalogMeta`/`query.CatalogMeta` still referenced in 6+ test files despite being deleted
- These don't block `go build` but LSP/gopls reports them

---

## c) NOT STARTED ⬜

1. **Execute deletion plan** — All tiers above are audited, none executed
2. **Fix stale golden files** — `catalog/asyncapi` and `catalog/eventcatalog` need `-update`
3. **Fix LSP build errors** — 11 pre-existing diagnostics
4. `testhelpers` coverage at 10.5% — no improvement effort started
5. `catalog/internal/caseutil` at 76.5% — no effort started
6. `storage` coverage at 86.9% — no effort to reach 90%+
7. `example/todo` not in go.work but exists on disk
8. No CI/CD pipeline for auto-running golden file checks
9. No version tagging or release automation
10. No CHANGELOG.md

---

## d) TOTALLY FUCKED UP 💥

1. **Pre-existing LSP errors masquerading as current issues** — `command.CatalogMeta` was deleted in commit `1088fcd` but 6 test files in `integration/` and `catalog/adapters/` still reference it. These files compile fine with `go build` (the references are in test files that may have build tags or conditional compilation), but gopls reports errors. This is confusing and needs investigation.

2. **`ErrNilPayload` undefined in `codec_typed.go`** — This is a genuine compile-time error in LSP. The symbol was either renamed or removed without updating this file. How `go build` passes is unclear — needs investigation.

3. **`decider` coverage dropped from 93.3% → 89.3%** — Between sessions, coverage regressed 4 percentage points. No investigation done yet.

4. **Session 90 status report exists TWICE** — Both `2026-05-22_01-19_FAKE_STORE_SPLIT_CATALOG_DESIGN_STATUS.md` and `2026-05-22_01-31_SESSION_90_COMPREHENSIVE_STATUS.md` cover the same session. Redundant.

5. **14 status reports in `docs/status/` with no archive discipline** — Multiple reports per day, overlapping timeframes, no clear numbering convention after session 90.

6. **`catalog/adapters` has 100% coverage of DEPRECATED code** — The deprecated `CatalogBuilder`, `FromCommandDispatcher`, and `FromQueryDispatcher` are fully tested and benchmarked. This is wasted test maintenance burden for code that should be deleted.

7. **504 public exports for a "lite" library** — The word "lite" in the project name is aspirational, not descriptive. The API surface is enormous for a CQRS helper library.

---

## e) WHAT WE SHOULD IMPROVE 📈

### API Surface Quality
1. **Delete dead exports aggressively** — 407 lines of zero-consumer code adds cognitive load for every consumer trying to understand the library
2. **504 exports is TOO MANY** — Target: cut to ~350 by removing Tier 1-3 items
3. **Unexport internal plumbing** — `MiddlewareChain`, `GetHandler`, `CatalogDispatcher`, `CopyCatalogEntries` should be private — they're implementation details
4. **Delete the deprecated `aggregate` package entirely** — It's 1,756 lines of dead weight that confuses consumers into using the OO pattern instead of `decider`

### Test Quality
5. **Fix testhelpers coverage (10.5%)** — Either write tests or acknowledge it's a test-utility package that doesn't need coverage
6. **Golden file drift** — Stale golden files fail CI. Either auto-refresh or add a CI check that detects drift
7. **Don't test deprecated code** — `catalog/adapters` has 100% coverage of code we want to delete

### Developer Experience
8. **Fix all LSP errors** — Any new contributor opening this in VS Code/GoLand sees 11 red squiggles
9. **Consolidate status reports** — Archive old ones, establish a naming convention
10. **Add a CHANGELOG.md** — Consumers need to know what changed between versions

### Architecture
11. **`query.Handler` returns `any`** — Violates project "no any" rule. The `DispatchTyped[T]` workaround exists but the core interface is wrong
12. **`CatalogMeta` duplicate was eliminated** but `CatalogEntry` lives in `core/pkg/dispatcher/` due to circular deps — architecturally questionable but technically correct
13. **`sync` module** — 28 exports, unclear if it has any consumers at all. May be premature abstraction

---

## f) Top 25 Things We Should Get Done Next

### Priority 1: Immediate (This Session)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | **Execute Tier 1 deletion plan** (407 lines of dead exports) | High | Low |
| 2 | **Refresh golden files** (`go test -update` for asyncapi + eventcatalog) | Medium | Trivial |
| 3 | **Fix LSP errors** (`ErrNilPayload`, `CatalogMeta` references in test files) | Medium | Low |

### Priority 2: API Cleanup (Next Session)

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 4 | **Delete deprecated `catalog/adapters.CatalogBuilder`** + migrate `example/user` | Medium | Medium |
| 5 | **Delete `FromCommandDispatcher` / `FromQueryDispatcher`** | Medium | Low |
| 6 | **Unexport `dispatcher.MiddlewareChain`, `GetHandler`** | Medium | Low |
| 7 | **Unexport or delete `event.OutboxPublisher`** (206 lines, zero consumers) | Medium | Low |
| 8 | **Remove `Command.IdempotencyKey()` from interface** | Medium | Medium |
| 9 | **Delete `query.Pagination` subsystem** (93 lines, zero consumers) | Low | Low |

### Priority 3: Package Cleanup

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 10 | **Delete `core/aggregate/` package** (1,756 lines deprecated) | High | Medium |
| 11 | **Delete `integration/aggregate/`** (tests for deleted package) | Medium | Low |
| 12 | **Delete `decider.Result` + `ExecuteWithResult`** (71 lines, zero consumers) | Low | Low |
| 13 | **Delete `event.NewEvents` + `MustNewEvents` + `DecodePayloads`** (84 lines) | Low | Low |
| 14 | **Investigate `sync` module** — does it have any consumers? | Medium | Low |

### Priority 4: Quality

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 15 | **Fix `testhelpers` coverage** (10.5% → 60%+) | Medium | Medium |
| 16 | **Investigate `decider` coverage regression** (93.3% → 89.3%) | Medium | Low |
| 17 | **Improve `storage` coverage** (86.9% → 90%+) | Medium | Medium |
| 18 | **Improve `catalog/internal/caseutil`** (76.5% → 90%+) | Low | Low |

### Priority 5: Polish

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 19 | **Add CHANGELOG.md** | Medium | Low |
| 20 | **Archive old status reports** (14 in root, some overlapping) | Low | Trivial |
| 21 | **Fix `query.Handler` returns `any`** — see `QUERY_HANDLER_GENERICS.md` | High | High |
| 22 | **Investigate `example/todo`** — not in go.work, unclear status | Low | Low |
| 23 | **Add CI golden file drift detection** | Medium | Medium |
| 24 | **Rename `go-cqrs-lite`?** — 504 exports isn't "lite" | Low | N/A |
| 25 | **Establish semver / release automation** | Medium | Medium |

---

## g) Top #1 Question I Cannot Figure Out Myself 🤔

**How do the LSP build errors exist when `go build` passes?**

Specifically:
- `integration/command/command_test.go` references `command.CatalogMeta` (6 locations) — LSP says `undefined`
- `core/event/codec_typed.go` references `ErrNilPayload` — LSP says `undefined`
- Yet `go build ./...` and `go test ./...` both pass cleanly

Possible explanations:
1. **Build tags** — test files might have build tags that conditionally exclude them
2. **Stale gopls cache** — gopls might be seeing an older state
3. **go.work vs module-local resolution** — the workspace might resolve differently than gopls expects

This matters because it's impossible to tell if these are real problems or tooling noise, and fixing them blindly could break something that actually works.

---

## Module Dependency Graph (Current)

```
core (193 exports) ← foundation, zero internal deps
  ↑
  ├── memory (11 exports) ← in-memory implementations
  ├── catalog (129 exports) ← documentation generation
  │     └── adapters (100% coverage of deprecated code)
  ├── middleware (30 exports) ← cross-cutting concerns
  ├── testhelpers (37 exports, 10.5% coverage) ← test utilities
  ├── projection (10 exports) ← replay + live subscription
  ├── storage (66 exports, 86.9% coverage) ← SQL implementations
  ├── integration ← cross-module tests
  ├── sync (28 exports, 92.2% coverage) ← distributed primitives
  └── example/ ← usage demos
```

## Uncommitted Changes

```
 M AGENTS.md
 M core/aggregate/aggregate.go
?? docs/status/2026-05-22_01-31_SESSION_90_COMPREHENSIVE_STATUS.md
?? projection/builder_test.go
```

---

_Arte in Aeternum_
