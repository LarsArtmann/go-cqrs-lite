# Session 33 — Comprehensive Status Report

> **Date:** 2026-05-02 20:44
> **Branch:** master (clean, up to date with origin)
> **Test Suite:** 21 packages, ALL PASS ✅
> **Total Coverage:** 83.3% (production code)
> **Production Code:** 9,079 lines across 96 files
> **Test Code:** 18,838 lines across 76 files

---

## A) FULLY DONE ✅

### Session 31 — Error Taxonomy + Offline-First Primitives (5 commits)
| Commit | Description |
|--------|-------------|
| `f899c3f` | `event.Family` enum (5 families), `event.Error` struct, `Classify()`, `IsRetryable()`, constructors |
| `857f753` | Fixed silent error discard in `projection/runner.go`, wired `WithRetry` with exponential backoff |
| `29cdc53` | `id.ClientID` branded type, `WithClientID()`, `WithClientOccurredAt()`, `DefaultRetryConfig` uses `IsRetryable` |
| `4166a40` | **BREAKING**: `IdempotencyKey() string` added to `command.Command` interface |
| `566b08d` | AGENTS.md updated, OFFLINE_FIRST_METADATA.md created |

### Session 32 — Test Coverage + Type Quality + Cleanup (3 commits)
| Commit | Description |
|--------|-------------|
| `cc9d6c3` | Tests: taxonomy nil/duplicate, ClientID, WithClientID, WithClientOccurredAt, projection retry |
| `c9c4c6d` | `event.Error` implements `fmt.Formatter`, `event.Version.String()`, `MustParseClientID` panic test |
| `7bf7525` | Removed stale FakeStore/MemoryStore Known Issue, status report, AGENTS.md session notes |

### Per-Package Coverage (Production)
| Package | Coverage | Status |
|---------|----------|--------|
| `core/command` | 97.6% | ✅ |
| `core/event` | 97.3% | ✅ |
| `core/query` | 100.0% | ✅ |
| `core/pkg/dispatcher` | 100.0% | ✅ |
| `core/pkg/id` | 100.0% | ✅ |
| `core/aggregate` | 92.9% | ✅ |
| `middleware` | 99.4% | ✅ |
| `catalog/d2` | 97.7% | ✅ |
| `catalog/asyncapi` | 95.9% | ✅ |
| `catalog/eventcatalog` | 95.6% | ✅ |
| `catalog/adapters` | 95.5% | ✅ |
| `catalog` | 94.4% | ✅ |
| `storage` | 93.1% | ✅ |
| `memory` | 91.9% | ⚠️ (missing godoc tests not counted) |
| `projection` | 85.8% | ⚠️ (dead options pull down coverage) |

---

## B) PARTIALLY DONE 🔧

| Item | What's Done | What's Missing |
|------|-------------|----------------|
| Error taxonomy | Core system complete (`Family`, `Error`, `Classify`, `IsRetryable`, constructors, `fmt.Formatter`) | Not adopted by downstream packages — `projection/errors.go`, `aggregate/errors.go`, `command/errors.go`, `query/errors.go`, `storage/errors.go` still use bare `errors.New` |
| Compile-time interface checks | 21 checks exist across codebase | Missing for `*projection.Runner` (io.Closer), `*OutboxPublisher` (io.Closer), `command.Dispatcher`/`query.Dispatcher` |
| Godoc | All core/event + core/pkg/id types documented | Missing on: `projection.Runner` (6 symbols), `memory.*` (25+ symbols), `catalog/eventcatalog.Exporter` (3), `asyncapi` types (12), error sentinels in projection/aggregate |

---

## C) NOT STARTED ⏳

### Dead Code Cleanup
- `projection/options.go`: `WithBatchSize`, `WithBatchWindow`, `WithConcurrency` — public API, zero consumers, fields never read by runner
- `projection/errors.go`: `ErrRunnerStopped`, `ErrDuplicateHandler`, `ErrCheckpointLoad`, `ErrStoreLoad`, `ErrNilStore` — zero references outside definition
- `testhelpers/fake_checkpoint.go`: `FakeCheckpointStore` — never imported anywhere in the project

### Type Model Improvements
- `event.Type` has no `String()` method (inconsistent with `event.Version`)
- `event.AggregateType` has no `String()` method (inconsistent with `event.Source`, `event.IPAddress`)
- `command.Type`, `query.Type` have no `String()` method
- `*event.Error` has no `Is(error) bool` method — two Errors with same Code don't match via `errors.Is`

### File Size Violations
- `testhelpers/helpers.go` — 293 lines (43 over 250 limit)
- `core/aggregate/repository.go` — 254 lines (4 over 250 limit)

### Missing Tests
- `core/event/builder.go` — 7 exported methods, zero test coverage (`Builder.WithPayload`, `WithOptions`, `WithCorrelationID`, `WithCausationID`, `WithUserID`, `Build`, `MustBuild`)
- `core/event/catalog.go` — `CatalogCore`, `MustNewCatalogCore` — zero direct test coverage
- `core/command/catalog.go` — `CatalogCore`, `MustNewCatalogCore` — zero test coverage
- `core/query/catalog.go` — `CatalogCore`, `MustNewCatalogCore` — zero test coverage
- `core/event/enricher.go` — `CompositeEnricher`, `EnrichEvent` — zero test coverage
- `command.Core.IdempotencyKey()` — no test verifying default `""` return

### Error Taxonomy Adoption (Downstream Packages)
- `projection/errors.go` — 4 used sentinels + 5 unused sentinels, all bare `errors.New`
- `aggregate/errors.go` — 4 sentinels, all bare `errors.New`
- `command/errors.go` — 2 sentinels, bare `errors.New`
- `query/errors.go` — 2 sentinels, bare `errors.New`
- `storage/errors.go` — 1 sentinel, bare `errors.New`

---

## D) TOTALLY FUCKED UP 💥

**Nothing is catastrophically broken.** The codebase compiles, all 21 test packages pass, no lint failures.

### Things We Got Wrong / Should Have Done Better:

1. **Dead code shipped as public API** — `WithBatchSize`, `WithBatchWindow`, `WithConcurrency` were declared as public options but never wired. This is a broken promise to consumers. Should have been removed before committing or never committed without wiring.

2. **Unused error sentinels accumulated** — 5 of 9 projection sentinels are never referenced. Dead code in a library is worse than dead code in an app — it pollutes the public API surface.

3. **FakeCheckpointStore committed with zero consumers** — 52 lines of dead code in `testhelpers/`.

4. **Taxonomy built but not adopted** — We built a beautiful 5-family error classification system in `core/event/errors.go` but then didn't migrate any of the ~25 sentinels across 5 packages to use it. `Classify()` maps `event` package sentinels but everything else falls through to `default: Transient`. The `projection.ErrNilBus` error (Infrastructure) gets classified as Transient (retryable) — **wrong**.

5. **Godoc debt in `memory` package** — The `memory` module is what every consumer uses for testing, yet it has zero godoc on any of its ~25 exported symbols. For a library, this is embarrassing.

6. **File size limit ignored** — `testhelpers/helpers.go` at 293 lines has been over the 250-line limit for multiple sessions without being addressed.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture
- **`query.Handler` returns `any`** — This is the single biggest type safety gap. Every query middleware propagates `any`. The workaround (`DispatchTyped[T]`) exists but the core API violates the project's own "no any" rule. This needs a breaking change with a generic Handler type.
- **`CatalogMeta` duplicated 3x** — `event.CatalogMeta`, `command.CatalogMeta`, `query.CatalogMeta` are nearly identical. Should be a shared type.
- **`subscribesTo` duplicated** — Same unexported function in `core/event/runner.go:172` and `projection/runner.go:187`.
- **`Root.LoadEvents` vs `Core.LoadFromHistory` mismatch** — Every aggregate must implement both and delegate. Unnecessary boilerplate.

### Process
- **Don't commit dead public API** — Wire first, export second. Options without consumers are broken promises.
- **Adopt taxonomy immediately** — After building `Classify()`, we should have migrated all sentinels in the same session. Leaving it half-done means `Classify()` is less useful.
- **Enforce godoc** — No exported symbol should be committed without godoc. Linter should catch this.
- **Enforce file size** — The 250-line limit exists but isn't checked automatically.

### Type Safety
- **Missing `Is()` on `*event.Error`** — Without it, `errors.Is(NewRejection("not_found", "..."), NewRejection("not_found", "..."))` returns `false` even with identical codes. This undermines the taxonomy's usefulness.
- **String types without `String()`** — `event.Type`, `event.AggregateType`, `command.Type`, `query.Type` are all `type X string` without a `String()` method. Inconsistent with `event.Version` and `id.Of[T]`.

---

## F) TOP 25 THINGS TO DO NEXT

Sorted by (impact × customer_value) / effort:

| # | Task | Impact | Effort | Type |
|---|------|--------|--------|------|
| 1 | Remove dead `WithBatchSize`, `WithBatchWindow`, `WithConcurrency` options + fields from `projection/` | HIGH | LOW | Dead code |
| 2 | Remove 5 unused error sentinels from `projection/errors.go` | HIGH | LOW | Dead code |
| 3 | Remove unused `FakeCheckpointStore` from `testhelpers/` | MED | LOW | Dead code |
| 4 | Add godoc to all 6 exported symbols in `projection/runner.go` | HIGH | LOW | Docs |
| 5 | Add godoc to `memory/bus.go` (7 exported symbols) | HIGH | MED | Docs |
| 6 | Add godoc to `memory/store.go` (8 exported symbols) | HIGH | MED | Docs |
| 7 | Add godoc to `memory/snapshot.go` (all exported symbols) | HIGH | MED | Docs |
| 8 | Add godoc to `memory/checkpoint.go` + `memory/outbox.go` | MED | LOW | Docs |
| 9 | Add godoc to `catalog/eventcatalog/exporter.go` (3 symbols) | MED | LOW | Docs |
| 10 | Add godoc to all 12 exported types in `catalog/asyncapi/types.go` | MED | MED | Docs |
| 11 | Add godoc to `projection/errors.go` (remaining sentinels) | MED | LOW | Docs |
| 12 | Add godoc to `aggregate/errors.go`, `command/errors.go`, `query/errors.go` | MED | LOW | Docs |
| 13 | Add `Is(error) bool` to `*event.Error` for `errors.Is` support | MED | LOW | Type |
| 14 | Add `String()` to `event.Type`, `event.AggregateType` | MED | LOW | Type |
| 15 | Add `String()` to `command.Type`, `query.Type` | MED | LOW | Type |
| 16 | Add compile-time `var _ io.Closer` for `*projection.Runner` | MED | LOW | Type |
| 17 | Add compile-time `var _ io.Closer` for `*OutboxPublisher` | LOW | LOW | Type |
| 18 | Migrate `projection/errors.go` sentinels to taxonomy wrappers | MED | MED | Taxonomy |
| 19 | Migrate `aggregate/errors.go` sentinels to taxonomy wrappers | MED | LOW | Taxonomy |
| 20 | Migrate `command/errors.go` + `query/errors.go` sentinels | MED | LOW | Taxonomy |
| 21 | Migrate `storage/errors.go` sentinel to taxonomy wrapper | MED | LOW | Taxonomy |
| 22 | Add tests for `event.Builder` (7 methods, zero coverage) | MED | MED | Test |
| 23 | Add tests for `CatalogCore` / `MustNewCatalogCore` (3 packages) | MED | LOW | Test |
| 24 | Add tests for `CompositeEnricher` / `EnrichEvent` | MED | MED | Test |
| 25 | Split `testhelpers/helpers.go` (293 lines → under 250) | MED | MED | Refactor |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should the error taxonomy adoption (tasks 18–21) change the sentinel error VALUES or just wrap them?**

Currently, all sentinels are `errors.New("some string")`. Two approaches:

**Option A — Keep sentinels, add `Classify()` mappings:**
```go
// projection/errors.go — no change to definitions
var ErrNilBus = errors.New("projection: nil bus")

// core/event/errors.go — add cross-package sentinel mapping
// PROBLEM: circular import (projection → core/event, core/event → projection)
```
This is **blocked by circular imports**. `core/event` cannot import `projection` or `aggregate` or `storage`.

**Option B — Change sentinels to taxonomy-typed errors:**
```go
// projection/errors.go
var ErrNilBus = event.NewInfrastructure("projection.nil_bus", "projection: nil bus")
```
This **requires `projection` to import `core/event`** — which it already does. No circular dependency.
But it **changes the error type** from `*errors.errorString` to `*event.Error`, which is a **breaking change** for anyone using `errors.As[*errors.errorString]()` (unlikely but possible).

**Option C — Dual approach: keep sentinel, add typed wrapper:**
```go
var ErrNilBus = errors.New("projection: nil bus")

// In Classify(), use string matching or a registration mechanism
```
Over-engineered for a library.

**My recommendation: Option B** — change sentinels to typed errors. The only consumers are within this repo, and `errors.Is` still works (`Unwrap` not needed for direct comparison). But I want your call before breaking public error types.

---

## Session History

| Session | Commits | Focus |
|---------|---------|-------|
| 30 | 1 | Architecture roadmap + planning |
| 31 | 5 | Error taxonomy, projection fix, ClientID, IdempotencyKey breaking change |
| 32 | 3 | Test coverage, fmt.Formatter, Version.String, stale issue cleanup |
| 33 | 0 | Planning + status audit (this report) |

**All on master, all pushed, clean working tree.**
