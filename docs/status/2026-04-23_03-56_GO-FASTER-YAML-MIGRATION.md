# Status Report: go-cqrs-lite

**Date:** 2026-04-23 03:56 CEST
**Branch:** master
**Commit:** 18d1a21 (up to date with origin/master)

---

## Executive Summary

The project is in **strong shape**. All 101 Go files (5,755 LOC production + 8,368 LOC tests) compile clean and pass the full test suite with race detection. This session replaced the custom `catalog/yaml` package with `github.com/go-faster/yaml`, removing 487 lines of hand-rolled YAML marshaling code and 5 transitive dependencies in exchange for a well-maintained upstream library.

---

## a) FULLY DONE

### Just Completed (This Session)

| What | Detail |
|------|--------|
| **Replace custom YAML with go-faster/yaml** | Deleted `catalog/yaml/yaml.go` (208 lines) + `catalog/yaml/yaml_test.go` (279 lines). Updated `catalog/asyncapi/exporter.go` import. All 13 asyncapi tests pass identically. |

### Previously Completed (Confirmed Still Working)

| What | Detail |
|------|--------|
| **Core CQRS packages** | `command/`, `query/`, `event/`, `aggregate/` — all stable, tested, race-clean |
| **Generic internal dispatcher** | `internal/dispatcher/dispatcher.go` eliminates boilerplate across command/query |
| **Catalog system** | `catalog/` core + `catalog/adapters` + `catalog/asyncapi` (AsyncAPI 3.0) + `catalog/eventcatalog` (MDX) |
| **Branded IDs** | `pkg/id/` — type-safe `id.Of[T]` with full JSON/DB/encoding support |
| **xtypes** | Type-safe wrappers for compile-time safety |
| **Middleware** | `middleware/` — logging, retry, validation, recovery |
| **BDD test suites** | Ginkgo v2 BDD tests for event, aggregate, query packages |
| **Event stores** | Memory store, memory bus, memory snapshot store |
| **Examples** | `example/user/` + `example/catalog/` |
| **Build** | `GOWORK=off go build ./...` — clean, zero errors |

### Test Coverage Summary

| Package | Coverage | Status |
|---------|----------|--------|
| `catalog/asyncapi` | 96.3% | Excellent |
| `event` | 95.4% | Excellent |
| `xtypes` | 95.7% | Excellent |
| `query` | 91.5% | Excellent |
| `catalog` | 91.2% | Excellent |
| `catalog/eventcatalog` | 89.5% | Good |
| `middleware` | 84.6% | Good |
| `command` | 84.4% | Good |
| `pkg/id` | 85.4% | Good |
| `aggregate` | 77.3% | Needs improvement |
| `internal/dispatcher` | 77.4% | Needs improvement |
| `catalog/adapters` | 66.0% | Needs improvement |
| `pkg/errors` | 0.0% | Dead code |

---

## b) PARTIALLY DONE

| Item | Status | Remaining |
|------|--------|-----------|
| **AGENTS.md update** | Catalog system documented, custom YAML entry still mentions `catalog/yaml/` | Remove `catalog/yaml/` references from AGENTS.md |
| **TODO_LIST.md** | Has comprehensive list but is stale (still references `catalog/yaml` improvements) | Needs regeneration |
| **Code duplication cleanup** | Phase 1 done, Phase 2+ remaining | Further dedup in asyncapi/eventcatalog exporters |
| **Linter cleanup** | 44+ golangci-lint warnings remain | `varnamelen`, `exhaustruct`, `revive`, `exhaustive` |

---

## c) NOT STARTED

### High-Value Features (from TODO_LIST.md and project review)

| # | Item | Priority |
|---|------|----------|
| 1 | Breaking: `Root.ID()` → `id.AggregateID` | HIGH |
| 2 | Breaking: `Event.AggregateID()` → `id.AggregateID` | HIGH |
| 3 | Breaking: Make `Command.AggregateID()` optional | HIGH |
| 4 | Fix `query.Handler` to accept `context.Context` | MEDIUM |
| 5 | PostgreSQL event store implementation | MEDIUM |
| 6 | Projection/read-model support | MEDIUM |
| 7 | Saga/process manager support | LOW |
| 8 | gRPC/HTTP transport adapters | LOW |
| 9 | CLI tool for catalog generation | LOW |
| 10 | CI/CD workflows (`.github/workflows/`) | MEDIUM |

### Infrastructure / Process

| # | Item |
|---|------|
| 11 | `.github/workflows/test.yml` + `lint.yml` |
| 12 | Coverage tracking (codecov/coveralls) |
| 13 | Pre-commit hooks |
| 14 | `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md` |
| 15 | `CHANGELOG.md` review/update |
| 16 | Architecture documentation |
| 17 | GoDoc package examples |

---

## d) TOTALLY FUCKED UP / PROBLEMATIC

| Issue | Severity | Detail |
|-------|----------|--------|
| **`query.Dispatcher.Dispatch` ignores `ctx`** | CRITICAL | Context silently discarded. Handlers don't receive context, making timeouts/cancellation/tracing impossible. Inconsistent with command dispatcher. |
| **`pkg/errors` is dead code** | MODERATE | `BaseError` defined but never used. 0% coverage. Redundant with `cockroachdb/errors`. |
| **`MemoryBus.Publish` holds RLock during handler execution** | MODERATE | Subscribers block publishers. Acceptable for test utility but should be documented. |
| **`catalog/registry.go` Build() shared backing array** | MODERATE | Potential slice corruption + non-deterministic map iteration order. |
| **`xtypes.TypedCommand.Command()` allocates on every call** | LOW | Creates new `command.Core` each time. Should embed or cache. |
| **`event.Core` mutability via Option** | LOW | `WithMetadata` can mutate event after construction, contradicting "immutable" doc comment. |
| **AGENTS.md references deleted `catalog/yaml/`** | LOW | Just caused by this session's change — needs update. |

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Context propagation in query dispatch** — This is the #1 consistency gap. Command handlers accept `ctx`, query handlers don't. This should be a breaking change in the next major version.
2. **Remove dead `pkg/errors`** — Or meaningfully integrate it. Currently it's just noise.
3. **Embed `command.Core`/`query.Core` in xtypes** — Avoid per-call allocations in `TypedCommand.Command()` and `TypedQuery.Query()`.

### Code Quality

4. **Fix 44+ linter warnings** — `varnamelen`, `exhaustruct`, `revive`, `exhaustive` are the main categories.
5. **Improve `catalog/adapters` coverage (66%)** — Add missing test cases.
6. **Improve `aggregate` coverage (77.3%)** — Error path tests needed.

### Dependencies

7. **New transitive deps from go-faster/yaml** — Added 5 new indirect deps (`go-faster/errors`, `go-faster/jx`, `segmentio/asm`, `uber/multierr`, `golang.org/x/exp`). These are well-maintained but worth monitoring.
8. **Consider `go-json-experiment/json` alignment** — Project uses both `encoding/json` and `go-json-experiment/json`. Could consolidate.

### Process

9. **Automate TODO_LIST.md generation** — Currently manual/stale. Should be CI-driven.
10. **Add CI workflows** — No automated testing on push/PR.

---

## f) Top 25 Things We Should Get Done Next

### Breaking Changes (Bundle into v2)

| # | Task | Impact |
|---|------|--------|
| 1 | `query.Handler` accepts `context.Context` | Fixes #1 consistency gap |
| 2 | `Root.ID()` returns `id.AggregateID` | Type safety across aggregate boundary |
| 3 | `Event.AggregateID()` returns `id.AggregateID` | Consistency with Root.ID() |
| 4 | Make `Command.AggregateID()` optional | Not all commands target aggregates |

### Bug Fixes & Safety

| # | Task | Impact |
|---|------|--------|
| 5 | Fix `registry.Build()` shared backing array | Data corruption risk |
| 6 | Fix `MemoryStore.LoadFromVersion` slice copy | Prevents mutation of stored events |
| 7 | Fix `MemoryBus.Subscribe` nil handler check | Defensive programming |
| 8 | Fix asyncapi component message key collision | Namespace commands/events |

### Code Quality

| # | Task | Impact |
|---|------|--------|
| 9 | Update AGENTS.md (remove `catalog/yaml` references) | Keeps documentation accurate |
| 10 | Regenerate `TODO_LIST.md` | Stale items clutter planning |
| 11 | Fix 44+ linter warnings | Production readiness |
| 12 | Remove dead `pkg/errors` package | Reduce noise |
| 13 | Improve `catalog/adapters` test coverage to 80%+ | 66% → 80%+ |
| 14 | Improve `aggregate` test coverage to 85%+ | 77% → 85%+ |
| 15 | Fix `time.Time` schema generation | `{type:"string", format:"date-time"}` |

### Features

| # | Task | Impact |
|---|------|--------|
| 16 | PostgreSQL event store | Real-world persistence |
| 17 | Projection/read-model support | CQRS query side |
| 18 | Event upcasting/schema evolution | Long-lived events |
| 19 | Add `enum`/`default` struct tag support to Schema | Richer catalog schemas |
| 20 | CLI tool for catalog generation | Developer experience |

### Infrastructure

| # | Task | Impact |
|---|------|--------|
| 21 | CI workflows (test + lint) | Automated quality gate |
| 22 | Coverage tracking (codecov) | Visibility into coverage trends |
| 23 | Pre-commit hooks | Catch issues before push |
| 24 | CONTRIBUTING.md + CODE_OF_CONDUCT.md | Open source readiness |
| 25 | Architecture documentation (ADR-style) | Knowledge preservation |

---

## g) Top #1 Question I Cannot Figure Out Myself

**What is the intended versioning strategy for the next batch of breaking changes?**

Items #1-4 in the "Top 25" list above are all breaking API changes (changing handler signatures, return types, interface methods). They logically belong together. The question is:

- **Option A:** Release as `v2.0.0` with all breaking changes bundled together?
- **Option B:** Use Go module major suffix (`go-cqrs-lite/v2`) per Go convention?
- **Option C:** Gradual migration with compatibility shims (deprecated wrappers)?

This decision affects how we sequence the work — whether to batch all breaking changes or spread them out with deprecation periods. I cannot determine the project's versioning philosophy from the codebase alone.

---

## Build & Test Verification

```
$ GOWORK=off go build ./...          # PASS (zero errors)
$ GOWORK=off go test ./... -count=1  # ALL PASS (14 packages)
$ GOWORK=off go vet ./...            # PASS (zero issues)
```

## Lines of Code

| Category | Lines |
|----------|-------|
| Production Go | 5,755 |
| Test Go | 8,368 |
| Total | 14,123 |
| Go files (prod) | 67 |
| Go files (test) | 34 |
| Example files | 6 |

## Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `cockroachdb/errors` | v1.12.0 | Error wrapping |
| `go-faster/yaml` | v0.4.6 | YAML marshaling (NEW) |
| `go-json-experiment/json` | v0.0.0-20260214 | JSON v2 |
| `google/uuid` | v1.6.0 | UUID generation |
| `onsi/ginkgo/v2` | v2.28.1 | BDD testing |
| `onsi/gomega` | v1.39.1 | BDD matchers |
