# Session 37 — Comprehensive Status Report

**Date:** 2026-05-02 23:02  
**Branch:** master (clean, up to date with origin)  
**Sessions elapsed:** 37 (Sessions 33–37 in this conversation series)

---

## a) FULLY DONE

### Core Library (production-quality)

| Module | Coverage | Status |
|--------|----------|--------|
| `core/command` | 100.0% | ✅ Complete — dispatcher, middleware, lifecycle, catalog metadata, IdempotencyKey |
| `core/query` | 100.0% | ✅ Complete — dispatcher, typed dispatch, pagination, catalog metadata |
| `core/pkg/dispatcher` | 100.0% | ✅ Complete — generic dispatcher engine, LifecycleMixin |
| `core/pkg/id` | 100.0% | ✅ Complete — branded IDs (AggregateID, EventID, UserID, etc.), ClientID |
| `core/event` | 97.4% | ✅ Complete — Store/Bus/SnapshotStore interfaces, Core, Error taxonomy, Version, outbox publisher, in-memory runner |
| `core/aggregate` | 92.7% | ✅ Complete — Root, Repository, EventSourcedRepository, snapshot strategy, outbox support |
| `middleware` | 99.4% | ✅ Complete — logging, metrics, retry (with IsRetryable), validation, recovery |
| `memory` | 91.9% | ✅ Complete — MemoryStore, MemoryBus, MemorySnapshotStore |
| `storage` | 93.1% | ✅ Complete — PostgreSQL SQLEventStore, SQLSnapshotStore, SQLCheckpointStore, SQLOutboxStore |
| `projection` | 90.4% | ✅ Complete — Runner, HandlerRegistry, retry on transient errors |
| `catalog` | 94.4% | ✅ Complete — Registry, SchemaFromType[T], MessageID |
| `catalog/adapters` | 95.5% | ✅ Complete — CatalogBuilder, FromDispatcher adapters |
| `catalog/asyncapi` | 95.9% | ✅ Complete — AsyncAPI 3.0 YAML/JSON exporter |
| `catalog/d2` | 97.7% | ✅ Complete — D2 diagram exporter |
| `catalog/eventcatalog` | 95.6% | ✅ Complete — EventCatalog MDX generator |

### Quality Infrastructure

- **21 test packages**, ALL PASS
- **97 production files** (9,141 lines), **76 test files** (18,894 lines)
- **All files ≤250 lines** (max: `storage/event_store.go` at 250)
- **Zero TODO/FIXME** in production code
- **Zero `any` type violations** (all uses are structural/stdlib-required)
- **Nix flake** — build, test, lint, vet, coverage, format, dev shell
- **CI** — GitHub Actions (`ci.yml`)

### Sessions 33–37 Cleanup Work

| Work | Session | Commit(s) |
|------|---------|-----------|
| Comprehensive status audit (S33) | 33 | `f5f2854` |
| Dead API removal — 3 options, 5 sentinels, 1 type | 34 | `3b66029` |
| Godoc for 57 exported symbols | 34 | `9cb5467` |
| `String()` on 4 type aliases, `Is()` on Error, `io.Closer` checks | 34 | `5e987f2` |
| README fix — compilable code example, CI badges, design principle | 35 | `8e222af` |
| `IdempotencyKey()` test coverage | 36 | `c200f12` |
| Split `testhelpers/helpers.go` (293→3 files) | 36 | `af97f8e` |
| Trim `repository.go` (254→244), extract `publishChanges` | 36 | `ed3fb83` |
| `var _` interface checks for Core types, Error method godoc | 36 | `4629bae` |

### Error Taxonomy (Session 31)

- `event.Family` enum — Rejection, Conflict, Transient, Corruption, Infrastructure
- `event.Error` struct with Code, Message, Family, cause
- `event.Classify(err) Family` — maps sentinels to families
- `event.IsRetryable(err) bool` — returns true for Transient
- Constructors: `NewRejection`, `NewConflict`, `NewTransient`, `NewCorruption`, `NewInfrastructure`
- `fmt.Formatter` — `%+v` shows `family:code: message` with cause chain
- `*Error.Is(error) bool` — matches by Code+Family

### Type Model Quality

- Branded IDs via `id.Of[T]` — prevents mixing AggregateID/EventID/UserID at compile time
- `event.Version` branded type with `.Int()` and `.String()`
- `event.Type`, `event.AggregateType`, `command.Type`, `query.Type` — all have `String()`
- `IdempotencyKey()` on `Command` interface — empty default, consumers override
- `ClientID` branded type + `WithClientID`/`WithClientOccurredAt` metadata options
- Compile-time `var _` interface checks for 20+ type→interface pairs

---

## b) PARTIALLY DONE

### Lint: 43 Issues Remaining

The linter (`nix run .#lint`) reports 43 issues in `core/`:

| Category | Count | Severity |
|----------|-------|----------|
| `wsl_v5` (whitespace above if) | 23 | LOW — formatting style |
| `noinlineerr` (inline error handling) | 8 | LOW — style preference |
| `exhaustruct` (missing struct fields) | 5 | MEDIUM — Error constructors skip `cause` field |
| `golines` (line length) | 2 | LOW — formatting |
| `errorlint` (comparing with !=) | 2 | LOW — test code comparing unwrapped errors |
| `staticcheck` (QF1008) | 1 | LOW — test code removable selector |
| `unused` | 1 | LOW — test code unused type |
| `nlreturn` (blank line before return) | 1 | LOW — `WithCause` one-liner |

**Note:** Only `core/` is linted by the flake. Other modules (memory, catalog, middleware, projection, storage, testhelpers) are NOT linted. Unknown state.

### Godoc Coverage — Partial

**Missing package-level doc comments:**
- `memory/`, `projection/`, `testhelpers/`, `catalog/`, `catalog/adapters/`, `catalog/d2/`, `catalog/eventcatalog/`

**Missing exported symbol godoc:**
- `catalog/d2/exporter.go` — 5 exported symbols (Exporter, Option, WithDescription, WithDirection, NewExporter)
- `catalog/adapters/` — 8 exported functions across event.go, command.go, query.go
- `core/event/errors.go` — `Family.String()` (1 method)
- `core/pkg/id/id_encoding.go` — 8 encoding methods (MarshalJSON, UnmarshalJSON, etc.)

### README — CI Badges Still Wrong

The README points to `test.yml` and `lint.yml` but only `ci.yml` exists. Session 35 attempted to fix this but the current README still shows the wrong URLs. **This is a regression or the fix didn't land properly.**

### Functions Exceeding 30 Lines — 10 Remaining

| File | Function | ~Lines |
|------|----------|--------|
| `storage/helpers.go` | `scanEvents` | 76 |
| `storage/event_store.go` | `Save` | 71 |
| `core/event/runner.go` | `HandleParallel` | 66 |
| `storage/event_store.go` | `AppendBatch` | 49 |
| `core/event/event.go` | `validateEventParams` | 51 |
| `core/event/event.go` | `NewEvent` | 40 |
| `core/aggregate/repository.go` | `loadEvents` | 39 |
| `core/event/outbox_publisher.go` | `PublishNow` | 39 |
| `core/event/outbox_publisher.go` | `NewOutboxPublisher` | 37 |
| `middleware/retry.go` | `retry` | 31 |

---

## c) NOT STARTED

### From Architecture Roadmap (Session 30)

1. **Error taxonomy propagation to downstream packages** — `aggregate`, `projection`, `storage` sentinels should use `event.NewInfrastructure()` or `event.NewConflict()` instead of `errors.New`. Blocked by circular dependency concern — needs design decision.

2. **Offline-first primitives** — `WithClientTimezone`, `WithCausationID` metadata options documented but not implemented. `docs/OFFLINE_FIRST_METADATA.md` defines convention-based keys but no code uses them.

3. **`query.Handler` generic refactor** — Currently returns `any`. `DispatchTyped[T]` is the workaround. A proper generic handler would be a breaking change.

4. **`CatalogMeta` consolidation** — Duplicated across `event`, `command`, `query` packages. Nearly identical structs. Should live in a shared location or be parameterized.

5. **`Root.LoadEvents` vs `Core.LoadFromHistory` mismatch** — Every aggregate must implement `LoadEvents` and manually delegate to `LoadFromHistory`. This boilerplate is a design smell.

6. **Saga/Process manager** — `docs/planning/SAGA_DESIGN.md` exists with design, zero implementation.

7. **Watermill integration** — `docs/planning/2026-04-23_WATERMILL_PRO_CONTRA.md` evaluated but no code.

8. **Example app overhaul** — `docs/planning/2026-05-02_2307-SUPERB_EXAMPLE.md` exists but not implemented. Current `example/user/main.go` is a basic demo.

9. **Lint for all modules** — Only `core/` is linted. `memory`, `catalog`, `middleware`, `projection`, `storage`, `testhelpers` have zero lint enforcement.

10. **D2 diagram rendering in CI** — `docs/web-client-communication.d2` exists but not auto-rendered.

### Testing Gaps

11. **`testhelpers/` has zero test files** — 668 lines of production code with zero tests. Consumers depend on this being correct.

12. **`catalog/internal/cattest/` has zero test files** — Test helper package untested.

13. **Benchmark coverage** — Only `core/event` has benchmarks. Other modules have none.

---

## d) TOTALLY FUCKED UP

### README CI Badge URLs (REGRESSION)

The README still shows:
```
[![Tests](https://github.com/.../workflows/test.yml/badge.svg)]
[![Lint](https://github.com/.../workflows/lint.yml/badge.svg)]
```

But `.github/workflows/` only contains `ci.yml`. Session 35 commit `8e222af` claimed to fix this. Either the fix was overwritten or the commit didn't contain the change. **This makes the README look broken to visitors — both badges show "build failing" because the workflow files don't exist.**

### Lint Only Runs on `core/`

The `nix run .#lint` command only lints the `core/` module. The other 8 production modules have zero lint enforcement. The 43 existing lint issues in `core/` are unaddressed since Session 20. This means:
- Unknown number of lint issues in `storage/`, `memory/`, `catalog/`, `middleware/`, `projection/`
- CI (`ci.yml`) may or may not run lint — need to verify

### `exhaustruct` False Positives

The 5 `exhaustruct` warnings on `Error` constructors are technically correct — `cause` field is zero-valued. But `cause` is intentionally unexported and set via `WithCause()`. The `//nolint:exhaustruct` comments are missing.

---

## e) WHAT WE SHOULD IMPROVE

### High Impact

1. **Fix README badges** — Point to `ci.yml` or remove separate badges. 5-minute fix, massive first-impression impact.

2. **Lint all modules** — Extend `nix run .#lint` to cover all 9 production modules. Then fix all issues.

3. **Fix 43 existing lint issues** — Especially `exhaustruct` (5), `errorlint` (2), `unused` (1), `staticcheck` (1). The 23 `wsl_v5` and 8 `noinlineerr` are style preferences that could be configured away.

4. **Package-level godoc** — 7 packages missing `// Package ...` doc comments. These show on pkg.go.dev. Low effort, high visibility.

5. **Catalog adapter godoc** — 13 exported functions across `catalog/d2` and `catalog/adapters` without doc comments.

### Medium Impact

6. **Refactor long functions** — `scanEvents` (76 lines) and `SQLEventStore.Save` (71 lines) are the worst offenders. Extract helpers.

7. **Test testhelpers** — 668 lines of code with zero tests. At minimum, compile-time interface checks and a few integration tests.

8. **Consolidate CatalogMeta** — Extract to shared package or parameterize. Reduces 3 near-identical structs to 1.

9. **Fix `Root.LoadEvents` vs `Core.LoadFromHistory`** — Provide default implementation or eliminate the split.

10. **Storage module — move from test-only to production-ready** — Document which operations are safe for production, add connection pool docs, add migration guide.

### Low Impact (but correct)

11. **Add `//nolint:exhaustruct` to Error constructors** — Suppress false positives with documented reason.

12. **Add missing `var _ SnapshotStrategy = (*everyN)(nil)`** — Only interface implementation without compile-time check.

13. **Add godoc to `id.Of[T]` encoding methods** — 8 exported methods without comments.

14. **Render D2 diagram in CI** — Auto-generate SVG from `.d2` source.

15. **Add benchmarks beyond core/event** — Establish performance baselines.

---

## f) Top 25 Things to Do Next

Sorted by impact × effort (highest first):

| # | Task | Effort | Impact | Module |
|---|------|--------|--------|--------|
| 1 | Fix README CI badges (point to ci.yml) | 5 min | HIGH | docs |
| 2 | Add package-level godoc to 7 packages | 30 min | HIGH | memory, projection, testhelpers, catalog, adapters, d2, eventcatalog |
| 3 | Add godoc to 13 catalog adapter/d2 exports | 20 min | MEDIUM | catalog |
| 4 | Fix 43 lint issues in core (or configure away) | 1 hr | MEDIUM | core |
| 5 | Extend lint to all 9 production modules | 1 hr | HIGH | all |
| 6 | Refactor `scanEvents` (76→<30 lines) | 30 min | MEDIUM | storage |
| 7 | Refactor `SQLEventStore.Save` (71→<30 lines) | 30 min | MEDIUM | storage |
| 8 | Refactor `HandleParallel` (66→<30 lines) | 30 min | MEDIUM | core/event |
| 9 | Add `//nolint:exhaustruct` to Error constructors | 5 min | LOW | core/event |
| 10 | Add `var _ SnapshotStrategy` compile-time check | 2 min | LOW | core/aggregate |
| 11 | Add tests for testhelpers package | 1 hr | MEDIUM | testhelpers |
| 12 | Consolidate CatalogMeta (3→1 struct) | 2 hr | MEDIUM | catalog + core |
| 13 | Fix `Root.LoadEvents` vs `Core.LoadFromHistory` boilerplate | 2 hr | MEDIUM | core/aggregate |
| 14 | Add godoc to 8 `id.Of[T]` encoding methods | 15 min | LOW | core/pkg/id |
| 15 | Error taxonomy for downstream packages (design decision) | 3 hr | HIGH | cross-cutting |
| 16 | Implement offline-first metadata options (WithClientTimezone, WithCausationID) | 2 hr | MEDIUM | core/event |
| 17 | Write superb example app (from planning doc) | 3 hr | HIGH | example |
| 18 | Add benchmarks for command, query, aggregate, middleware | 2 hr | LOW | core, middleware |
| 19 | Storage module production readiness docs | 1 hr | MEDIUM | storage |
| 20 | Auto-render D2 diagrams in CI | 1 hr | LOW | CI |
| 21 | Investigate and fix `query.Handler` any return type | 3 hr | MEDIUM | core/query |
| 22 | Saga/Process manager design implementation | 1 week | HIGH | new module |
| 23 | Watermill integration module | 1 week | MEDIUM | new module |
| 24 | README overhaul — add architecture diagram, real-world examples | 2 hr | HIGH | docs |
| 25 | Add Go doc examples (`Example*` test functions) for key APIs | 3 hr | MEDIUM | core |

---

## f) Top #1 Question I Cannot Figure Out Myself

**Should downstream package sentinels (`aggregate.ErrNilStore`, `projection.ErrHandlerNotFound`, `storage.ErrSnapshotNotFound`, etc.) change from `errors.New()` to classified `event.Error` types?**

Current state: `Classify()` in `core/event` can only map `event`-package sentinels because importing `aggregate`/`projection`/`storage` would create circular dependencies. This means:
- `aggregate.ErrNilStore` → classified as `Transient` (default) instead of `Infrastructure`
- `storage.ErrSnapshotNotFound` → classified as `Transient` (default) instead of `Conflict`
- `projection.ErrHandlerNotFound` → classified as `Transient` (default) instead of `Rejection`

**Options I see but can't decide:**
1. **Accept the limitation** — Document that `Classify()` only handles `event` package errors. Consumers classify their own.
2. **Move sentinels to `event` package** — But this couples `event` to domain concepts it shouldn't know about.
3. **Create `core/pkg/errors` package** — Neither imports the other. All sentinels live here. But this moves ALL error definitions away from their domain context.
4. **Registration pattern** — `Classify()` accepts a `map[error]Family` that consumers populate. More flexible but more setup.

This is an architectural decision that affects the entire error handling story. Owner must decide.

---

## Repository Stats

| Metric | Value |
|--------|-------|
| Production files | 97 (9,141 lines) |
| Test files | 76 (18,894 lines) |
| Test:Production ratio | 2.07:1 |
| Modules | 10 (9 production + example) |
| Test packages | 21 (all pass) |
| Total coverage | ~94% weighted average |
| Max file size | 250 lines (`storage/event_store.go`) |
| Functions > 30 lines | 10 |
| Lint issues (core only) | 43 |
| Zero-coverage packages | 2 (`testhelpers`, `cattest` — test helpers) |
| Known issues | 5 (all LOW/MEDIUM, documented) |
| Breaking changes (Sessions 27-36) | 8 |
