# Session 66 — Comprehensive Status Report

**Date:** 2026-05-17 06:12
**Trigger:** go.work health audit + module hygiene review
**Total commits:** 744

---

## Executive Summary

The library is in **good shape overall** — all 22 test packages pass, zero TODOs, 43 benchmarks, 39 sentinel errors. However, recent uncommitted refactoring work left **lint failures in middleware** (6 issues from the new `RetryConfig.Validate()` feature) and there are significant **go.work/module hygiene issues** that should be cleaned up.

| Metric               | Value               |
| -------------------- | ------------------- |
| Production LOC       | 14,177              |
| Test LOC             | 27,071              |
| Production files     | 152                 |
| Test files           | 108                 |
| Test packages        | 22 (all pass)       |
| Total coverage       | 84.5%               |
| Benchmarks           | 43                  |
| Sentinel errors      | 39                  |
| Lint issues          | 6 (middleware only) |
| Files over 250 lines | 4 production files  |
| TODO/FIXME/HACK      | 0                   |

---

## A) FULLY DONE ✅

| Item                    | Detail                                                                           | Commit     |
| ----------------------- | -------------------------------------------------------------------------------- | ---------- |
| Core module             | All 7 packages pass, 92.9–100% coverage                                          | —          |
| Memory module           | 99.5% coverage                                                                   | —          |
| Catalog module          | 4 sub-packages, 93.9–100% coverage                                               | —          |
| Middleware module       | 100% coverage, but 6 lint issues (see section D)                                 | —          |
| Testhelpers module      | All helpers functional                                                           | —          |
| Integration module      | All 4 packages pass                                                              | —          |
| Projection module       | 98.3% coverage                                                                   | —          |
| Sync module             | 92.2% coverage, independent of core                                              | —          |
| Storage module          | 85.1% coverage, all tests pass                                                   | —          |
| Error taxonomy          | 38 sentinel errors across 7 modules, all classified via `RegisterClassification` | Session 51 |
| Branded types           | `Version`, `SchemaVersion`, `AggregateID`, `EventID`, `ClientID`, etc.           | Session 65 |
| Type safety sweep       | `uint` pagination, branded sync types, outbox constant extraction                | Session 65 |
| Dead dependency removal | `cockroachdb/errors`, `go-json-experiment/json` removed                          | Session 54 |
| ISP interfaces          | `event.Publisher`, `event.Subscriber` sub-interfaces                             | Session 48 |
| Shared helpers          | `event.PublishChanges()`, `event.SaveSnapshot()`, `event.SnapshotStrategy`       | Session 48 |
| Decider package         | Functional aggregate pattern, 92.7% coverage                                     | Session 37 |
| Example/user            | Full CQRS demo with decider, middleware, catalog                                 | Session 37 |
| Example/todo            | Migrated go-localfirst Todo app                                                  | Latest     |
| CI/CD                   | GitHub Actions (Nix-based): fmt, build, vet, test                                | —          |
| Nix flake               | Build, test, lint, vet, coverage, format apps                                    | —          |

## B) PARTIALLY DONE 🔧

| Item                           | Status                                                                                                      | What's Left                                        |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| OutboxPublisher `closed` field | Uncommitted fix for exhaustruct lint — removed `mu` and `cancel` zero-value init, added `//nolint` comment  | Commit the change (or fix properly)                |
| `RetryConfig.Validate()`       | Feature is complete and tested, but introduced 6 lint issues (3 `noinlineerr`, 3 `revive:unused-parameter`) | Fix lint issues                                    |
| `gomodguard` linter            | Deprecated, needs migration to `gomodguard_v2`                                                              | Update `.golangci.yml`                             |
| go.work hygiene                | File exists and works, but all modules also have redundant `replace` directives                             | Remove replace directives                          |
| Pebble store refactoring       | Serialization extracted to `pebble_serialization.go`, old code removed from `pebble_event_store.go`         | Committed but still has 321 lines (over 250 limit) |
| Aggregate helper extraction    | `load_helpers.go` extracted from `repository.go`                                                            | Committed, clean                                   |

## C) NOT STARTED ⏳

| Item                                                | Impact | Effort | Notes                                                                                                     |
| --------------------------------------------------- | ------ | ------ | --------------------------------------------------------------------------------------------------------- |
| Remove all `replace` directives from `go.mod` files | HIGH   | 8min   | 6 modules have redundant `replace` directives — `go.work` handles this                                    |
| Normalize module version references                 | HIGH   | 10min  | `catalog` → `core v0.0.0`, `integration` → `middleware v0.0.0-000101...`, `storage v0.0.0` — inconsistent |
| Remove or repurpose root `go.mod`                   | MEDIUM | 3min   | Has no `.go` files, module path `github.com/larsartmann/go-cqrs-lite` is unused                           |
| Migrate `gomodguard` → `gomodguard_v2`              | MEDIUM | 3min   | Deprecation warning on every lint run                                                                     |
| Add `tool` directive for lint tools (Go 1.24+)      | MEDIUM | 8min   | Pin `golangci-lint` version in go.mod                                                                     |
| Run `go work sync` to prune `go.work.sum`           | LOW    | 2min   | 352 lines, many stale transitive deps                                                                     |
| Evaluate `example/` modules in `go.work`            | LOW    | 5min   | `example/todo` pulls pebble + turso + cqrs-htmx into workspace resolution                                 |
| Evaluate whether `sync` should depend on `core`     | LOW    | 5min   | Currently independent — intentional or oversight?                                                         |
| Split `storage/helpers.go` (423 lines)              | MEDIUM | 10min  | Largest file, well over 250-line limit                                                                    |
| Split `storage/pebble_event_store.go` (321 lines)   | MEDIUM | 10min  | Over 250-line limit                                                                                       |
| Split `catalog/asyncapi/exporter.go` (258 lines)    | LOW    | 10min  | Slightly over 250-line limit                                                                              |
| Split `example/todo/cmd/api/main.go` (329 lines)    | LOW    | 10min  | Over 250-line limit, but example code                                                                     |
| Storage coverage improvement (85.1% → 90%+)         | MEDIUM | 30min  | Lowest coverage module                                                                                    |

## D) TOTALLY FUCKED UP 💥

| Item                                     | Detail                                                                                                  | Severity                    |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------- | --------------------------- |
| Middleware lint failures (6 issues)      | `RetryConfig.Validate()` introduced `noinlineerr` (3×) and `revive:unused-parameter` (3×) in `retry.go` | HIGH — lint is broken       |
| `gomodguard` deprecated warning          | Appears on every module lint run                                                                        | LOW — cosmetic but noisy    |
| Uncommitted `outbox_publisher.go` change | `//nolint:exhaustruct` hack — should either commit proper fix or revert                                 | MEDIUM — dirty working tree |

## E) WHAT WE SHOULD IMPROVE 🏗️

### Module Hygiene

1. **Remove all `replace` directives** — They're 100% redundant with `go.work`. Every module has both `replace` AND `go.work` resolving the same paths. This creates confusion about which mechanism is actually in use.
2. **Normalize version references** — Some modules use `v1.1.0`, others `v0.0.0`, others `v0.0.0-000101...`. With `go.work` active the specific version doesn't matter for local dev, but it's messy and confusing.
3. **Root `go.mod` has zero purpose** — No `.go` files at root. Either delete it or give it a purpose (e.g., a `tool` directive for project-wide dev tools).

### Go 1.24+ Features

4. **`tool` directive** — Pin `golangci-lint`, `ginkgo`, and other dev tools in a go.mod `tool` directive instead of relying on nix/shell. Makes the project more portable.
5. **`gomodguard_v2`** — Stop the deprecation noise.

### Code Quality

6. **4 files over 250-line limit** — `storage/helpers.go` (423), `storage/pebble_event_store.go` (321), `example/todo/cmd/api/main.go` (329), `catalog/asyncapi/exporter.go` (258).
7. **Storage coverage at 85.1%** — Lowest module, should target 90%+.
8. **`example/todo` heavy dependencies** — pebble, turso, cqrs-htmx bloat the workspace. Consider whether examples should be in `go.work` at all.

### Architecture

9. **`sync` module independence** — Has zero dependency on `core`. Is this intentional (CRDT primitives are standalone) or should it use `core` types (e.g., `event.Event`, `id.AggregateID`)?
10. **`example/todo` replace directive** — Only has `replace` for `storage`, not for `core`/`memory`. This works because `go.work` covers it, but the asymmetry is confusing.

---

## F) Top 25 Things We Should Do Next

Sorted by impact × effort (highest ROI first):

| #   | Task                                                                                 | Impact | Effort | Category              |
| --- | ------------------------------------------------------------------------------------ | ------ | ------ | --------------------- |
| 1   | Fix 6 middleware lint failures (`noinlineerr` + `revive:unused-param` in `retry.go`) | HIGH   | 5min   | Lint Fix              |
| 2   | Commit or revert uncommitted `outbox_publisher.go` change                            | HIGH   | 2min   | Clean Working Tree    |
| 3   | Remove all `replace` directives from 6 module `go.mod` files                         | HIGH   | 8min   | Module Hygiene        |
| 4   | Normalize inter-module version refs (`v0.0.0` → consistent pseudo-version)           | HIGH   | 10min  | Module Hygiene        |
| 5   | Migrate `gomodguard` → `gomodguard_v2` in `.golangci.yml`                            | MEDIUM | 3min   | Lint Config           |
| 6   | Run `go work sync` to prune `go.work.sum` (352→~80 lines)                            | MEDIUM | 2min   | Module Hygiene        |
| 7   | Decide root `go.mod` fate: delete or repurpose with `tool` directive                 | MEDIUM | 5min   | Module Hygiene        |
| 8   | Split `storage/helpers.go` (423→2 files under 250)                                   | MEDIUM | 10min  | Code Quality          |
| 9   | Split `storage/pebble_event_store.go` (321→2 files under 250)                        | MEDIUM | 10min  | Code Quality          |
| 10  | Add `tool` directive for `golangci-lint` in go.mod (Go 1.24+ feature)                | MEDIUM | 8min   | Modernization         |
| 11  | Improve storage test coverage (85.1% → 90%+)                                         | MEDIUM | 30min  | Test Quality          |
| 12  | Split `example/todo/cmd/api/main.go` (329 lines)                                     | LOW    | 10min  | Code Quality          |
| 13  | Split `catalog/asyncapi/exporter.go` (258→2 files)                                   | LOW    | 10min  | Code Quality          |
| 14  | Decide: should `example/` modules stay in `go.work`?                                 | LOW    | 5min   | Architecture          |
| 15  | Decide: should `sync` depend on `core`?                                              | LOW    | 5min   | Architecture          |
| 16  | Add `go 1.26.2` version to `go.work` for consistency                                 | LOW    | 1min   | Consistency           |
| 17  | Document `sync` module purpose in AGENTS.md                                          | LOW    | 5min   | Documentation         |
| 18  | Add README.md to `sync/` module                                                      | LOW    | 10min  | Documentation         |
| 19  | Add integration tests between `sync` and `core` (if applicable)                      | LOW    | 20min  | Test Quality          |
| 20  | Update FEATURES.md with `sync` module and latest coverage numbers                    | LOW    | 10min  | Documentation         |
| 21  | Update TODO_LIST.md with current state                                               | LOW    | 10min  | Documentation         |
| 22  | Consider `io.Closer` removal from store interfaces (breaking change)                 | LOW    | 5min   | Architecture Planning |
| 23  | Evaluate `IdempotencyKey` auto-generation for `BaseCommand`                          | LOW    | 5min   | API Design            |
| 24  | Add benchmark for `sync` module operations                                           | LOW    | 10min  | Performance           |
| 25  | Clean up `go.work.sum` stale entries after `replace` removal                         | LOW    | 2min   | Module Hygiene        |

---

## G) Top #1 Question I Cannot Answer Myself 🤔

**Should `example/todo` and `example/user` remain in `go.work`?**

`example/todo` pulls in `cockroachdb/pebble`, `cqrs-htmx`, `turso.tech/database/tursogo` — heavy deps that bloat `go.work.sum` (352 lines) and slow workspace operations. `example/user` is lighter but still pulls transitive deps.

Options:

1. **Keep in `go.work`** — Examples stay buildable from root, but workspace gets heavy
2. **Remove from `go.work`** — Examples become standalone, use published versions. `go.work` stays lean. But `example/todo` has a `replace` for `storage` suggesting it needs local resolution
3. **Remove from `go.work`, keep `replace` directives only in examples** — Best of both worlds? Examples resolve locally via `replace`, but don't pollute workspace

This is a **product/design decision** about whether examples are first-class workspace citizens or downstream consumers. I cannot decide this autonomously.

---

## Test Coverage by Module

| Package                | Coverage |
| ---------------------- | -------- |
| `core/command`         | 100.0%   |
| `core/query`           | 100.0%   |
| `core/pkg/dispatcher`  | 100.0%   |
| `catalog/adapters`     | 100.0%   |
| `middleware`           | 100.0%   |
| `memory`               | 99.5%    |
| `projection`           | 98.3%    |
| `catalog/d2`           | 97.6%    |
| `core/pkg/id`          | 97.8%    |
| `core/aggregate`       | 96.9%    |
| `catalog/eventcatalog` | 95.7%    |
| `catalog`              | 94.5%    |
| `core/event`           | 92.9%    |
| `catalog/asyncapi`     | 93.9%    |
| `sync`                 | 92.2%    |
| `core/decider`         | 92.7%    |
| `storage`              | 85.1%    |

## go.work Module Inventory

| Module         | In go.work | Has replace                     | Version Refs                     | Notes                   |
| -------------- | ---------- | ------------------------------- | -------------------------------- | ----------------------- |
| `core`         | ✅         | — (producer)                    | —                                | Leaf dependency         |
| `testhelpers`  | ✅         | —                               | `core v1.1.0`                    | Clean                   |
| `memory`       | ✅         | `core`, `testhelpers`           | `core v1.1.0`                    | replace redundant       |
| `middleware`   | ✅         | `core`, `testhelpers`           | `core v1.1.0`                    | replace redundant       |
| `catalog`      | ✅         | `core`                          | `core v0.0.0` ⚠️                 | Version mismatch        |
| `projection`   | ✅         | `core`, `memory`, `testhelpers` | `core v1.1.0`                    | replace redundant       |
| `storage`      | ✅         | `core`, `memory`                | `core v1.1.0`                    | replace redundant       |
| `integration`  | ✅         | 6 replaces                      | `middleware v0.0.0-000101...` ⚠️ | Most replace directives |
| `sync`         | ✅         | —                               | Independent                      | No core dep             |
| `example/user` | ✅         | 4 replaces                      | `catalog v0.0.0` ⚠️              | replace redundant       |
| `example/todo` | ✅         | `storage`                       | `storage v0.0.0` ⚠️              | Heavy deps              |

---

## Uncommitted Changes

| File                             | Change                                                               |
| -------------------------------- | -------------------------------------------------------------------- |
| `core/event/outbox_publisher.go` | Removed `mu`/`cancel` zero-value inits, added `//nolint:exhaustruct` |

---

_Generated by Session 66 — go.work & Module Hygiene Audit_
