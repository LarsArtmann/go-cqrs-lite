# Session 114 — Comprehensive Status Report

**Date:** 2026-05-28 07:11 CEST
**Branch:** master
**Since checkpoint:** 19 commits since Session 112c (4b46371)
**Test status:** 27/27 packages OK (including golden files)
**Build:** Passes
**Lint:** 0 issues across all 12 modules (golangci-lint)
**Working tree:** Clean

---

## Project Overview

| Metric | Value |
|--------|-------|
| Language | Go 1.26.3 |
| Workspace modules | 18 (13 library + 5 examples) |
| Production `.go` files | 224 |
| Test `.go` files | 162 |
| Total `.go` files | 386 |
| Files > 350 lines | 0 production (1 test: `signing/multisig_test.go` at 491 lines) |
| TODO/FIXME comments | 0 |
| `any` type usages (non-generic) | 22 files — all justified (query dispatch, SQL args, JSON schema, logging) |
| Core dependencies | `oklog/ulid/v2`, `go-branded-id`, `go-error-family@v0.2.0`, `go-faster/yaml` (catalog) |
| Test frameworks | Ginkgo v2 + Gomega |

---

## Module Health

| Module | Production Files | Test Files | Lint Issues | Status |
|--------|-----------------:|-----------:|------------:|--------|
| core (6 sub-packages) | 52 | 38 | 0 | ✅ Healthy |
| storage | 29 | 22 | 0 | ✅ Healthy |
| catalog (7 sub-packages) | 49 | 29 | 0 | ✅ Healthy |
| middleware | 11 | 14 | 0 | ✅ Healthy |
| memory | 9 | 11 | 0 | ✅ Healthy |
| saga | 9 | 7 | 0 | ✅ Healthy |
| signing | 9 | 2 | 0 | ⚠️ Isolated (no consumers) |
| projection | 6 | 8 | 0 | ✅ Healthy |
| watermill | 3 | 3 | 0 | ✅ Healthy |
| testhelpers | 9 | 6 | 0 | ✅ Healthy |
| integration | 0 | 13 | 0 | ✅ Healthy |
| cmd/cqrs-gen | 1 | 1 | 0 | ✅ Healthy |
| example/user | 11 | 1 | — | ✅ Demo app |
| example/todo | 23 | 7 | — | ✅ Demo app |
| example/saga | 1 | 0 | — | ✅ Demo app |
| example/storage | 1 | 0 | — | ✅ Demo app |
| example/projection | 1 | 0 | — | ✅ Demo app |

---

## A) FULLY DONE

### Dependency Management
- ✅ **go-error-family v0.1.1 → v0.2.0** upgraded across all 16 modules (core, memory, catalog, middleware, testhelpers, integration, projection, signing, storage, saga, watermill, + 5 examples)
- ✅ **No banned dependencies directly imported** — `testify` and `pkg/errors` only appear as transitive deps via Ginkgo v2 and Watermill

### Signing Module
- ✅ **Integrated into workspace** — added to `go.work`, `flake.nix`, CI workflow
- ✅ **Multi-signature (multi-sig) support** — Ed25519 signer/verifier decoupled, multi-sig chains
- ✅ **All tests pass** — including golden file tests

### Code Quality
- ✅ **Zero TODO/FIXME comments** in production code
- ✅ **Zero files > 350 lines** in production code
- ✅ **Error middleware deduplication** — extracted `commandErrMiddleware`, `eventErrMiddleware`, `queryErrMiddleware` into `middleware/common.go`, eliminating triplicated 6-line blocks
- ✅ **Magic string elimination** — `cqrs-gen` now uses `genTypeCommand`/`genTypeQuery` constants
- ✅ **Golden file protection** — `catalog/testdata/golden/**` excluded from `treefmt` in `flake.nix`
- ✅ **Formatting consistency** — gofumpt + oxfmt applied across all modules
- ✅ **golangci-lint clean** — 0 issues across all 12 modules

### Error Classification (5-family taxonomy)
- ✅ **Rejection** — business rule violations (`event.NewRejection`)
- ✅ **Conflict** — concurrency/version conflicts (`event.NewConflict`)
- ✅ **Transient** — retryable infrastructure errors (`event.WrapTransient`)
- ✅ **Infrastructure** — non-retryable system errors (`event.WrapInfrastructure`)
- ✅ **Corruption** — data integrity violations (`event.WrapCorruption`)

### Core Features (from prior sessions)
- ✅ **Branded IDs** — `id.Of[T]` with ULID backing, type-safe across all modules
- ✅ **Event Store** — SQL (PostgreSQL/SQLite/Turso) + Pebble implementations
- ✅ **Event Bus** — Memory + Watermill adapters
- ✅ **Command Dispatcher** — generic `Dispatcher[H, M]` with middleware chain
- ✅ **Query Dispatcher** — typed `DispatchTyped[T]` with `PaginatedResult[T]`
- ✅ **Decider Pattern** — pure-function `Decider[State]` with `Fold`/`Decide`
- ✅ **Projection Runner** — replay + live, DLQ handler, batch projection
- ✅ **Saga Runner** — state machine, compensation, step definition
- ✅ **Circuit Breaker** — middleware for command/event/query
- ✅ **Catalog System** — Registry, AsyncAPI/D2/EventCatalog/OpenAPI exporters
- ✅ **Snapshot Store** — SQL + Memory implementations
- ✅ **Outbox** — SQL + poller with branded OutboxID and CreatedAt timestamp
- ✅ **Test Helpers** — Noop/Failing/Panic handlers, FakeMetrics, AppendEventsHandler

---

## B) PARTIALLY DONE

### Signing Module Integration
- ✅ Module exists, tests pass, workspace integrated
- ⚠️ **Zero consumers** — no module imports `go-cqrs-lite/signing`
- ⚠️ **Not used in any example** — example apps don't demonstrate event signing
- ⚠️ **Not documented in AGENTS.md** — the module list mentions it but design patterns are missing

### Middleware Deduplication
- ✅ Error-return middleware pattern extracted to `common.go`
- ⚠️ **7 middleware files still have triplicated command/event/query variants** — `retry.go`, `validation.go`, `recovery.go`, `logging.go`, `metrics.go`, `tracing.go`, `circuit_breaker.go` each define 3 nearly-identical functions
- ⚠️ Generic middleware function could eliminate this pattern entirely

### Documentation
- ✅ AGENTS.md updated with signing module
- ⚠️ Signing module README exists but API migration guide doesn't cover signing
- ⚠️ No ADR for the signing module design decisions

### Linting / Pre-commit Hook
- ✅ `golangci-lint` clean (0 issues per module)
- ⚠️ `go-structure-linter` has 4 false positives for the workspace root `go.mod`/`go.sum`
- ⚠️ `library-policy` flags `math/rand/v2` in `middleware/retry.go` — this is a **false positive** (used for jitter/backoff, not crypto)
- ⚠️ `golangci-lint` fails with "directory prefix does not contain modules" when run at workspace root (expected for Go workspaces)

---

## C) NOT STARTED

### Architecture & Type Improvements
1. **Generic middleware factory** — Eliminate the 7×3 middleware variant duplication
2. **Event upcasting** — `event.Upcaster` interface exists but no implementation
3. **Event codec** — `event.Codec` interface exists but no JSON/protobuf codec implementation
4. **Projection checkpoint store** — SQL implementation exists but not tested with real PostgreSQL
5. **Saga state store** — interface exists but only memory implementation
6. **Fuzz testing** — Zero fuzz tests in the project
7. **Benchmark tests** — Only in `go-error-family` dependency, none in this project
8. **BDD/Ginkgo tests** — Some modules use Ginkgo but not consistently across all modules
9. **CI race condition testing** — CI runs `-race` but no dedicated race condition tests
10. **Coverage enforcement** — No minimum coverage gate in CI

### Missing Features
11. **Event versioning strategy** — No migration path between event schema versions
12. **Event encryption** — No at-rest encryption for sensitive payloads
13. **Projection rebuild** — No admin API to trigger full projection rebuild
14. **Saga compensation verification** — No dry-run or audit trail for compensating actions
15. **Watermill dead-letter** — No DLQ handler for Watermill adapter
16. **OpenTelemetry integration** — `go.opentelemetry.io/otel` in middleware deps but no actual tracing spans emitted

### Developer Experience
17. **API documentation** — No generated API docs (godoc/pkg.go.dev ready but no examples)
18. **Migration guides** — Only exists for OutboxID branding, not for other breaking changes
19. **Example app quality** — example/todo has 23 production files but example/saga is just 1 file
20. **Error documentation** — No public docs for the 5-family error taxonomy

---

## D) TOTALLY FUCKED UP

### 🚨 Golden File Flakiness (FIXED, but fragility remains)
- **Problem:** `nix fmt` reformats `catalog/testdata/golden/*.yaml` and `*.js` files via `golines`, changing indentation, which breaks golden file snapshot tests
- **Root cause:** YAML/JS golden files were not excluded from the treefmt formatter chain
- **Fix applied:** Added `settings.excludes = [ "catalog/testdata/golden/**" ]` to `flake.nix`
- **Remaining risk:** The pre-commit hook (`buildflow`) runs formatters that may still touch these files. The exclusion in treefmt helps but `oxfmt` (run by buildflow) is separate and may not respect the same exclude patterns.
- **Impact:** Every commit that triggers `nix fmt` or buildflow can break golden tests

### 🟡 Pre-commit Hook Leaves Dirty Tree
- **Problem:** The `buildflow` pre-commit hook runs `goimports`, `oxfmt`, and other formatters that may create changes beyond what was staged. After the hook runs, the working tree can be dirty even if the commit succeeded.
- **Impact:** Developer must check `git status` after every commit and potentially re-stage/re-commit
- **Severity:** Medium — annoying but not breaking

### 🟡 go-structure-linter False Positives
- **Problem:** `go-structure-linter` reports 4 MEDIUM issues about the workspace root `go.mod`/`go.sum` being empty/missing deps. These are **false positives** — in a Go workspace, the root `go.mod` is intentionally empty; deps live in each module's `go.mod`.
- **Impact:** Pre-commit hook shows scary ERROR messages for non-issues
- **Fix needed:** Configure go-structure-linter to skip workspace root or add `.go-structure-linter.toml` with ignores

### 🟡 math/rand/v2 False Positive from library-policy
- **Problem:** `library-policy` flags `middleware/retry.go` for using `math/rand/v2` instead of `crypto/rand`. This is a **false positive** — `math/rand/v2` is used for exponential backoff jitter, which is intentionally non-cryptographic.
- **Impact:** Pre-commit hook shows MEDIUM security warning for correct code
- **Fix needed:** Add inline ignore comment or configure library-policy exclusion

---

## E) WHAT WE SHOULD IMPROVE

### High Priority (Architectural)
1. **Generic middleware factory** — The 7 middleware files × 3 variants = 21 nearly-identical functions. A single generic `MiddlewareFactory[H any, M any]` could eliminate ~70% of this code.
2. **Signing module integration** — It's a ghost module with zero consumers. Either integrate it (add to example, add docs) or remove it to avoid dead code perception.
3. **Event Codec implementation** — The `event.Codec` interface exists but has no concrete JSON implementation. Consumers must marshal/unmarshal manually.
4. **Pre-commit hook reliability** — Golden file flakiness and dirty working tree need permanent fixes.

### Medium Priority (Quality)
5. **Coverage enforcement** — Add minimum coverage gate (80%+) to CI
6. **Benchmark suite** — Add `Benchmark*` tests for hot paths (event store Save/Load, decider Execute, middleware chain)
7. **Integration test matrix** — Test storage against real PostgreSQL/SQLite/Turso, not just Pebble
8. **Error documentation** — Public-facing docs for the 5-family error taxonomy and how consumers should handle each family
9. **Consistent BDD testing** — Some modules use Ginkgo, others use table-driven. Pick one and standardize.
10. **CI parallelization** — Tests run sequentially; parallelize across modules

### Low Priority (Polish)
11. **File size enforcement in CI** — BuildFlow checks but doesn't block on > 350 line files
12. **go-structure-linter config** — Add `.go-structure-linter.toml` to suppress false positives
13. **library-policy config** — Suppress `math_rand_crypto` false positive for `middleware/retry.go`
14. **coverage.out in .gitignore** — These files are not tracked but clutter `git status`
15. **Example app consistency** — example/todo has 23 files, example/saga has 1. Should all be comparable quality.

### Code Smells to Watch
16. **`any` in query dispatcher** — `query.Handler` returns `(any, error)`. The typed variants (`DispatchTyped`, `RegisterTyped`) mitigate this, but the base interface is inherently untyped.
17. **`dialect.go` uses `any` for SQL interop** — This is justified (database/sql API requires `any`) but could benefit from type-safe wrappers.
18. **Replace directives** — All `go.mod` files have `replace` directives pointing to local modules. These are needed until v1.0.0 tags are pushed but create confusion for external consumers.

---

## F) Top #25 Things We Should Get Done Next

Sorted by impact × effort (highest impact, lowest effort first):

### P0 — Must Do (blocks consumers or causes ongoing pain)
| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 1 | **Fix pre-commit hook golden file flakiness** — Ensure buildflow respects treefmt excludes | High | Low | CI |
| 2 | **Integrate or remove signing module** — Add signing to an example app, or document why it's standalone | High | Medium | signing |
| 3 | **Add JSON event codec implementation** — `event.Codec` needs at least one concrete impl | High | Medium | core/event |
| 4 | **Push v1.0.0 tags to remote** — Eliminate `replace` directives requirement for consumers | High | Low | all |
| 5 | **Add coverage.out to .gitignore** — Prevents accidental commits | Low | Trivial | root |

### P1 — Should Do (significant quality improvement)
| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 6 | **Generic middleware factory** — Eliminate 7×3 duplication | High | Medium | middleware |
| 7 | **Add benchmark suite** — Event store Save/Load, decider Execute, middleware chain | Medium | Medium | all |
| 8 | **Add fuzz tests** — Event marshaling, ID parsing, decider fold | Medium | Medium | core |
| 9 | **PostgreSQL integration tests** — Test storage against real PG via testcontainers | High | Medium | storage |
| 10 | **Configure go-structure-linter** — Suppress 4 false positives for workspace root | Low | Trivial | CI |
| 11 | **Suppress math/rand/v2 false positive** — Add library-policy ignore for retry.go | Low | Trivial | middleware |
| 12 | **Add CI coverage gate** — Fail CI if any module drops below 80% | Medium | Low | CI |
| 13 | **Error taxonomy docs** — Public-facing guide for 5-family error handling | Medium | Low | docs |
| 14 | **OpenTelemetry tracing spans** — Actually emit spans in middleware, not just import the package | Medium | Medium | middleware |

### P2 — Nice to Have (polish, not blocking)
| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 15 | **Consistent BDD testing** — Standardize on Ginkgo v2 across all modules | Low | High | all |
| 16 | **Event versioning strategy** — Document and implement upcasting | Medium | High | core/event |
| 17 | **Projection rebuild API** — Admin endpoint to trigger full rebuild | Medium | Medium | projection |
| 18 | **Saga state store (SQL)** — Persist saga state to SQL, not just memory | Medium | Medium | saga |
| 19 | **Watermill DLQ handler** — Dead-letter queue for failed messages | Medium | Medium | watermill |
| 20 | **Example app consistency** — Make all 5 examples comparable quality | Low | Medium | example/* |
| 21 | **API docs with examples** — godoc-ready examples for all public APIs | Low | Medium | all |
| 22 | **Event encryption** — At-rest encryption for sensitive payloads | Low | High | core/event |
| 23 | **CI parallelization** — Run module tests in parallel via matrix strategy | Medium | Low | CI |
| 24 | **File size enforcement in CI** — Block merges on > 350 line files | Low | Trivial | CI |
| 25 | **Turso integration tests** — Test storage against real Turso cloud | Medium | Medium | storage |

---

## G) Top #1 Question I Cannot Figure Out Myself

### **What is the intended relationship between the `signing` module and the rest of the library?**

The signing module is fully implemented (HMAC-SHA256, Ed25519, multi-sig, middleware) but has **zero consumers** in the workspace. No example uses it. No module imports it. It's an island.

**Options I see:**
1. **It's a standalone utility module** — Consumers import it independently, like `catalog`. It doesn't need to be wired into the CQRS pipeline to be valuable.
2. **It should be integrated into examples** — Add signing middleware to the event bus in example/user or example/todo to demonstrate tamper-proof event streams.
3. **It should be removed** — If it's not part of the core library vision, keeping it adds maintenance burden and confuses consumers about the library's scope.

**Why I can't decide:** This is a product/positioning decision, not a technical one. The signing module is high quality (clean API, good tests, nice docs) but its relationship to the CQRS core is unclear. It could be a first-class feature or an optional plugin.

---

## Pre-commit Hook Health Summary

| Step | Status | Notes |
|------|--------|-------|
| gofumpt | ✅ Pass | |
| goimports | ✅ Pass | |
| oxfmt | ✅ Pass | |
| golines | ✅ Pass | |
| d2-fmt | ✅ Pass | |
| go-mod-tidy | ✅ Pass | |
| golangci-lint | ✅ 0 issues | Per-module |
| go-structure-linter | ❌ 4 false positives | Workspace root `go.mod`/`go.sum` — by design |
| library-policy | ⚠️ 1 false positive | `math/rand/v2` in retry jitter |
| TODO scanner | ✅ 0 found | |
| binary scanner | ✅ Clean | |
| file size checker | ⚠️ 1 file over 350 | `signing/multisig_test.go` (491 lines) |
| doc freshness | ✅ All up-to-date | |

---

## Commit History (19 commits since Session 112c)

```
71d9e41 fix: update golden files and fix signing test key length
a1ec72d feat(signing): decouple Ed25519 signer from verifier in multi-sig module
a9bb6a7 chore: normalize golden test fixtures + fix pre-existing test bug in signing
28c6b53 feat(signing): add multi-signature (multi-sig) support for event signing chains
dfef1e7 docs(status): add Session 113 golangci-lint full monorepo sweep report
9847619 refactor: deduplicate error middleware + protect golden files from formatting
ccc47ea style(catalog): normalize golden test fixtures to canonical formatting
a63b798 style(example/user): apply gofumpt formatting and improve code quality
5bd480e style(catalog): apply gofumpt/oxfmt formatting to golden test fixtures
697d353 refactor(memory/bus): extract common error-checking patterns into reusable helper methods
926f487 refactor(example/user): rename shadowing variables and improve error handling
f00f851 style: apply gofumpt/oxfmt formatting across catalog, examples
1c4c72c refactor(cqrs-gen): extract command/query string literals into named constants
3c8ddd5 docs(status): add Session 112e comprehensive deduplication status report
98c24a9 feat: upgrade go-error-family to v0.2.0 across all modules
4456c83 feat: integrate signing module into workspace and add go.sum
0200e01 docs(status): add Session 112d comprehensive status report
a8b1759 docs: add API migration guide, mark verified items in TODO list
c0026ce docs: update TODO_LIST.md with completed items and tag deferred
```

---

*Report generated at 2026-05-28 07:11 CEST*
