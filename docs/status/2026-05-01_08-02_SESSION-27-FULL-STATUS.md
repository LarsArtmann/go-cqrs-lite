# Session 27 — FULL STATUS REPORT

**Date:** 2026-05-01 08:02 CEST
**Branch:** `master`
**Commits since last push:** 1 (will push with this report)
**Total Go LOC:** ~26,747
**Test Packages:** 20 (18 passing, 2 no-test-files)
**Lint Issues:** 0
**Test Failures:** 0 (fixed in this report)

---

## A) FULLY DONE

### Completed in This Session (27)

| # | Item | Type | Commit(s) |
|---|------|------|-----------|
| 1 | D2 cross-service event connections + schema tooltips | Feature | `baca3ea` |
| 2 | `NewInMemoryRunner` returns `(*T, error)` instead of panicking | Breaking fix | `b8269d0` |
| 3 | `catalog/go.mod` stale replace directives removed | Bug fix | `4084e65` |
| 4 | FEATURES.md stale entries corrected | Docs | `a4915d0` |
| 5 | `Use()` added to `event.Bus` interface | Breaking feat | `2ebb147` |
| 6 | `FakeBus` + `stubBus` updated for Bus.Use | Fix | `459f14a` |
| 7 | False-positive `TestSQLEventStore_Close` fixed | Fix | `219ea09` |
| 8 | `NewCore` validates inputs, returns error | Breaking feat | `4e1679d` |
| 9 | `MustNewCore` helper for tests/examples | Feature | `4e1679d` |
| 10 | Aggregate error sentinels (`ErrNilAggregateID`, `ErrEmptyAggregateType`) | Feature | `4e1679d` |
| 11 | D2 exporter fields unexported (use Options pattern) | Refactor | `97bbaf3` |
| 12 | D2 exporter `drawn` counter → `hasCrossService bool` | Refactor | `97bbaf3` |
| 13 | Example `ChangeName` uses Apply pattern | Fix | `97bbaf3` |
| 14 | All test callers updated for breaking API changes | Chore | `eab4d3b`+ |
| 15 | Golden files regenerated | Chore | `298b9ee` |
| 16 | Doc comments on CatalogMeta types | Docs | `f3ba23b` |
| 17 | Compile-time interface checks (`ProjectionFunc`, `UpcasterFunc`) | Style | `cab994f` |

### Completed by Pre-Commit Hook (Auto-committed between sessions)

| # | Item | Type | Commit |
|---|------|------|--------|
| 18 | `NewOutboxPublisher` returns error instead of panicking | Breaking fix | `fcd2762` |
| 19 | SQL store `Load()` returns `ErrAggregateNotFound` (matches MemoryStore) | Breaking fix | `2d1f946` |
| 20 | Snapshot load errors propagated instead of discarded | Fix | `0568bf8` |
| 21 | EventCatalog `landingPage` config type fixed | Fix | `d7e9d84` |
| 22 | EventCatalog `owners` field removed from message frontmatter | Fix | `af32827` |
| 23 | `projection/` module created with samber/ro reactive streams | Feature | `297333d` |
| 24 | EventCatalog domain services as object IDs | Refactor | `f67be34` |
| 25 | LoadAtVersion finds snapshot at or before version | Fix | `e043e05` |

---

## B) PARTIALLY DONE

| Item | What's Done | What's Missing |
|------|------------|----------------|
| **`projection/` module** | Runner, handler, options, samber/ro integration, 6 tests pass | Not referenced from AGENTS.md, FEATURES.md, or getting-started.md. No integration with `catalog/`. Not in CI. |
| **D2 diagram exporter** | Cross-service connections, schema tooltips, 14 tests | No golden file test. No CLI integration test (pipe through `d2` binary). Not in `flake.nix`. |
| **`storage/` module** | EventStore, SnapshotStore, CheckpointStore, 95.5% coverage | No `SQLOutboxStore`. No real-DB integration tests (go-sqlmock only). FEATURES.md maturity matrix is stale. |
| **FEATURES.md** | Stale entries fixed, Close/Bus claims corrected | Module maturity matrix doesn't list `projection/` or `catalog/d2/`. Coverage numbers outdated. |

---

## C) NOT STARTED

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | `CatalogMeta` deduplication across 3 packages (`event`, `command`, `query`) | MEDIUM | 2h |
| 2 | Branded types for catalog (`ServiceID`, `MessageName`, `SemVersion`) | MEDIUM | 4h |
| 3 | `Root.LoadEvents` vs `Core.LoadFromHistory` interface alignment | MEDIUM | 1h |
| 4 | `query.Handler` returns `any` — violates "no any" rule | MEDIUM | 3h |
| 5 | Middleware triplication → generics or code generation | LOW | 6h |
| 6 | Error wrapping consistency (`fmt.Errorf` vs `errors.Wrap`) | LOW | 2h |
| 7 | `Schema/Property.Validate()` methods | LOW | 1h |
| 8 | `OutboxID` branded type with methods | LOW | 30min |
| 9 | Snapshot encoding marker (JSON vs raw bytes) | LOW | 1h |
| 10 | Replace hand-rolled retry with `cenkalti/backoff` | LOW | 1h |
| 11 | Replace hand-rolled JSON Schema with `invopop/jsonschema` | LOW | 2h |
| 12 | Add `testify` to simplify test assertions | LOW | 3h |
| 13 | Watermill module (Kafka/NATS adapter) | HIGH | 20h |
| 14 | Saga/Process Manager | HIGH | 30h |
| 15 | Tagged releases (semantic versioning) | MEDIUM | 2h |
| 16 | `d2` CLI in `flake.nix` | LOW | 30min |
| 17 | Golden file test for D2 exporter | LOW | 30min |
| 18 | Real-DB integration tests for storage | MEDIUM | 4h |
| 19 | `ExportD2File` convenience method on `CatalogBuilder` | LOW | 15min |
| 20 | Improve hand-written diagram (add Codec, Upcaster components) | LOW | 1h |

---

## D) TOTALLY FUCKED UP

### D1. Pre-Commit Hook Making Stealth Commits

The pre-commit hook (`golangci-lint` + `golines` + auto-fix) **silently commits and pushes** changes without user visibility. This has caused:

- **Broken code in committed state**: `aggregate_test.go` had garbled inline function from a malformed `sed` replacement that the hook committed without verifying compilation.
- **Lost commit messages**: The hook creates its own commit messages that don't follow the project convention.
- **Unpredictable state**: Between sessions, the hook made 10+ commits including breaking API changes that weren't part of the session plan.

**Recommendation**: Disable the auto-commit behavior of the pre-commit hook. It should only lint and format, not commit.

### D2. Stale LSP Diagnostics (100+ phantom errors)

The LSP shows 100+ compile errors for files that actually compile and test fine. This is caused by:
- `gopls` not picking up `go.work` module boundaries correctly
- Stale diagnostics from previous sessions' changes

**Impact**: Confusing for development, but no real build issues.

### D3. `storage/event_store_test.go` Was Broken (Fixed in This Report)

The `TestSQLEventStore_SQLInjectionSafety` test was broken by commit `2d1f946` (SQL store now returns `ErrAggregateNotFound` instead of nil). The test expected old behavior. **Fixed in this commit.**

---

## E) WHAT WE SHOULD IMPROVE

### Architecture

1. **No-panic convention is incomplete** — `NewEvent` returns errors, but `Builder.MustBuild` still panics. Inconsistent.
2. **`Version` type migration incomplete** — `core/aggregate` still has `event.Version` vs `int` friction (see LSP errors about `Version().Int`). Some places cast, some don't.
3. **Interface completeness** — `Bus` now has `Use()` but `Store` has no middleware support. `SnapshotStore` has no middleware either.
4. **`projection/` module is orphaned** — Created but not documented, not in FEATURES.md, not in AGENTS.md package overview.

### Testing

5. **No integration tests with real DB** — storage/ uses only go-sqlmock. Can't catch SQL dialect issues, connection pool bugs, or transaction isolation problems.
6. **Golden file tests are fragile** — Every formatting change breaks golden tests. Consider normalizing output or adding a diff tolerance.
7. **`testhelpers` has 0% coverage** — It's a test utility so this is OK, but FakeBus/FakeStore bugs would go undetected.

### Documentation

8. **AGENTS.md is stale** — Missing `projection/` module, coverage numbers outdated, package overview doesn't reflect breaking changes.
9. **FEATURES.md maturity matrix is stale** — Doesn't list `projection/` or `catalog/d2/`. Coverage numbers are from session 20.
10. **60+ historical docs** — `docs/status/` and `docs/planning/` have accumulated 60+ files. Consider archiving pre-v1 docs.

### Developer Experience

11. **Pre-commit hook is too aggressive** — Makes stealth commits, can push broken code.
12. **No `justfile` or task runner for common operations** — Everything goes through `nix run` which is slow.
13. **No CI badge or CONTRIBUTING.md** — Open source readiness gap.

---

## F) TOP 25 THINGS TO DO NEXT

Sorted by impact × effort (P0 = must do, P1 = should do, P2 = nice to have):

| # | Priority | Task | Impact | Effort | Category |
|---|----------|------|--------|--------|----------|
| 1 | **P0** | Fix pre-commit hook: disable auto-commit, only lint+format | HIGH | 15min | DX |
| 2 | **P0** | Update AGENTS.md: add projection module, update coverage, update package table | HIGH | 20min | Docs |
| 3 | **P0** | Update FEATURES.md: add projection + d2 modules, refresh coverage numbers | HIGH | 20min | Docs |
| 4 | **P0** | Fix `aggregate_test.go` — verify no stale `MustNewCore` corruption from hook | HIGH | 10min | Fix |
| 5 | **P1** | Add `catalog/d2` to CI (`nix run .#test`) | MEDIUM | 15min | CI |
| 6 | **P1** | Add `projection/` to CI | MEDIUM | 15min | CI |
| 7 | **P1** | Deduplicate `CatalogMeta` → shared `core/pkg/catalogmeta` | MEDIUM | 2h | Architecture |
| 8 | **P1** | Branded types for catalog (`ServiceID`, `MessageName`, `SemVersion`) | MEDIUM | 4h | Type safety |
| 9 | **P1** | Fix `query.Handler` returns `any` → generic typed result | MEDIUM | 3h | Type safety |
| 10 | **P1** | Golden file test for D2 exporter | LOW | 30min | Testing |
| 11 | **P1** | Add `d2` CLI to `flake.nix` | LOW | 30min | DX |
| 12 | **P1** | Real-DB integration tests for `storage/` | MEDIUM | 4h | Testing |
| 13 | **P1** | `Root.LoadEvents` → `Core.LoadFromHistory` alignment | MEDIUM | 1h | Architecture |
| 14 | **P2** | `Schema/Property.Validate()` methods | LOW | 1h | Correctness |
| 15 | **P2** | `OutboxID` branded type | LOW | 30min | Type safety |
| 16 | **P2** | Snapshot encoding marker (JSON vs raw) | LOW | 1h | Correctness |
| 17 | **P2** | Replace hand-rolled retry with `cenkalti/backoff` | LOW | 1h | Dependencies |
| 18 | **P2** | Replace hand-rolled JSON Schema with `invopop/jsonschema` | LOW | 2h | Dependencies |
| 19 | **P2** | Error wrapping consistency pass | LOW | 2h | Consistency |
| 20 | **P2** | `ExportD2File` convenience method | LOW | 15min | Feature |
| 21 | **P2** | Watermill module (Kafka/NATS adapter) | HIGH | 20h | Feature |
| 22 | **P2** | Saga/Process Manager | HIGH | 30h | Feature |
| 23 | **P2** | Tagged releases (v0.1.0) | MEDIUM | 2h | Release |
| 24 | **P2** | Add `testify` for test assertions | LOW | 3h | DX |
| 25 | **P2** | Middleware deduplication via generics | LOW | 6h | Architecture |

---

## G) TOP #1 QUESTION

**The `projection/` module was auto-created by the pre-commit hook between sessions. It uses `samber/ro` (reactive streams) which is a new dependency. The module compiles and has 6 passing tests, but:**

1. It's not documented in AGENTS.md or FEATURES.md
2. It's not clear if this was intentional or if the hook created it from a planning doc
3. It overlaps significantly with `core/event/runner.go` (InMemoryRunner)

**Question: Was the `projection/` module intentionally created? Should it replace `core/event/runner.go`, coexist alongside it, or be removed? The `samber/ro` dependency adds weight to the module — is that acceptable for a library that values minimal dependencies?**

---

## Test Coverage Summary (Current)

| Package | Coverage | Change from Session 20 |
|---------|----------|----------------------|
| `core/command` | 100.0% | — |
| `core/query` | 100.0% | — |
| `core/pkg/dispatcher` | 100.0% | — |
| `core/pkg/id` | 100.0% | — |
| `middleware` | 99.4% | — |
| `memory` | 98.0% | ↓ from 99.0% |
| `catalog/d2` | 97.7% | NEW |
| `catalog/asyncapi` | 96.8% | ↓ from 97.9% |
| `core/event` | 95.3% | ↓ from 96.3% |
| `catalog/adapters` | 95.5% | ↓ from 98.8% |
| `storage` | 95.5% | ↑ from 92.3% |
| `core/aggregate` | 92.7% | ↓ from 95.9% |
| `catalog/eventcatalog` | 93.7% | ↓ from 95.5% |
| `catalog` | 94.4% | — |
| `projection` | (new) | NEW |

> Coverage drops are from new code added without proportional test increases (e.g., error-returning constructors add branches).

---

## Module Inventory (10 modules in go.work)

```
go-cqrs-lite/
├── core/          ✅ Production — CQRS primitives
├── memory/        🧪 Test utility — In-memory implementations  
├── catalog/       ✅ Production — Auto-documentation
│   ├── asyncapi/  ✅ Production — AsyncAPI 3.0 export
│   ├── d2/        ✅ Production — D2 diagram export (NEW)
│   └── eventcatalog/ ✅ Production — EventCatalog MDX export
├── middleware/     ✅ Production — Cross-cutting middleware
├── storage/       ⚠️ Partial — PostgreSQL (no Outbox)
├── projection/    🆕 Unknown — samber/ro reactive streams (NEEDS DECISION)
├── testhelpers/   🧪 Test utility — Fakes and helpers
├── integration/   ✅ Test suite — Cross-module BDD tests
└── example/user/  💡 Demo — CLI event sourcing demo
```
