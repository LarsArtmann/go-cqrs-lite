# Session 143 — Full Comprehensive Status Report

**Generated:** 2026-05-29 13:15 CEST
**Branch:** master (1 commit ahead of origin/master)
**Go:** 1.26.3 | **Total Go files:** 504 (237 test files) | **Total lines:** 70,207

---

## Executive Summary

The project is in **good health** with 29/29 test packages passing (non-race), 83-100% coverage across all modules, and zero `go vet` issues. However, there are **critical partially-done changes** in the staging area that need resolution before the next commit:

1. **`stream` → `listing` rename is INCOMPLETE** — package declarations still say `stream`/`stream_test`, doc.go says "Package stream"
2. **New `core/store.Backend` abstraction** is staged but incomplete (memory + pebble backends untracked, no tests)
3. **`storage` module BROKEN with race detector** due to untracked `sql_aggregate_reader.go` importing `listing` (which still uses `package stream` internally)

The **CheckpointStore ISP split** completed in this session is clean and fully tested.

---

## A) FULLY DONE ✓

### Core Infrastructure

| Feature            | Module                | Coverage | Notes                                                                                                   |
| ------------------ | --------------------- | -------- | ------------------------------------------------------------------------------------------------------- |
| Command dispatcher | `core/command`        | 100%     | Typed handlers, middleware, lifecycle                                                                   |
| Query dispatcher   | `core/query`          | 96.8%    | Typed dispatch, pagination                                                                              |
| Event system       | `core/event`          | 90.7%    | 15 creation options, bus/store, journal, snapshots, outbox, upcasters, tombstones, ISP-split interfaces |
| Decider pattern    | `core/decider`        | ~95%     | Pure-function aggregates                                                                                |
| Branded IDs        | `core/pkg/id`         | 100%     | 7 built-in types + custom                                                                               |
| Generic dispatcher | `core/pkg/dispatcher` | 92.2%    | `Dispatcher[H, M]` with lifecycle                                                                       |
| JSON/Raw codec     | `codec`               | 100%     | Standalone encoding module                                                                              |

### CheckpointStore ISP Split (THIS SESSION)

| Change                                      | File                             | Status                 |
| ------------------------------------------- | -------------------------------- | ---------------------- |
| `CheckpointSink` (Save + Close)             | `core/event/checkpoint.go:27-34` | ✅                     |
| `CheckpointSource` (Load + Close)           | `core/event/checkpoint.go:36-43` | ✅                     |
| `CheckpointStore` (Sink + Source composite) | `core/event/checkpoint.go:45-50` | ✅                     |
| Memory impl assertions                      | `memory/checkpoint.go:55-59`     | ✅                     |
| SQL impl assertions                         | `storage/checkpoint.go:104-108`  | ✅                     |
| All 14 test packages pass                   | —                                | ✅                     |
| Zero breaking changes                       | —                                | ✅ Backward-compatible |

### Storage Layer

| Feature                     | Module    | Coverage |
| --------------------------- | --------- | -------- |
| SQL event store (PG/SQLite) | `storage` | 93.7%    |
| SQL snapshot store          | `storage` | —        |
| SQL checkpoint store        | `storage` | —        |
| SQL outbox                  | `storage` | —        |
| SQL saga store              | `storage` | —        |
| Pebble KV store             | `pebble`  | 87.8%    |
| Turso connector             | `turso`   | —        |

### Middleware Stack (24 factories)

| Concern                                                                                         | Coverage |
| ----------------------------------------------------------------------------------------------- | -------- |
| Logging, Metrics, Recovery, Retry, Tracing, Validation, Circuit Breaker, Timeout, Rate Limiting | 94.0%    |

### Cross-Cutting

| Feature                                                            | Status                                   |
| ------------------------------------------------------------------ | ---------------------------------------- |
| OpenTelemetry integration                                          | ✅ `otel` module, no-op when no provider |
| Event signing (HMAC + Ed25519 + multisig)                          | ✅ 93.7-94.2%                            |
| Watermill adapter                                                  | ✅ 94.4%                                 |
| Catalog exporters (AsyncAPI, D2, OpenAPI, EventCatalog, DocServer) | ✅ 89-96%                                |
| Code generator (`cqrs-gen`)                                        | ✅ 70.8%                                 |
| Memory test implementations                                        | ✅ 99.1%                                 |
| Test helpers                                                       | ✅ 83.7%                                 |

### Modularization Phase 1 (DONE)

- ✅ `saga/sagatest` extraction — testhelpers no longer depends on saga
- ✅ Version normalization across go.mod files
- ✅ `core/event.CheckpointStore` ISP split (this session)
- ✅ `event.Store` → `EventSink` + `EventSource` (done previously)
- ✅ Replace directive strategy documented

---

## B) PARTIALLY DONE ⚠️

### 1. `stream` → `listing` Module Rename — CRITICAL

**Status:** 70% done. Files moved, go.mod updated, go.work updated. But:

| Issue                                       | Files Affected             | Severity                         |
| ------------------------------------------- | -------------------------- | -------------------------------- |
| Production files still `package stream`     | 10 files in `listing/`     | 🔴 Build broken for `GOWORK=off` |
| Test files still `package stream_test`      | 9 test files in `listing/` | 🔴 gopls errors                  |
| `doc.go` says "Package stream"              | `listing/doc.go`           | 🟡 Misleading                    |
| Ghost `stream/` directory (empty go.mod)    | `stream/`                  | 🟡 Cleanup needed                |
| Ghost `example/stream/` directory           | `example/stream/`          | 🟡 Cleanup needed                |
| `example/listing/main.go` imports old paths | `example/listing/main.go`  | 🔴 May reference `stream`        |

Works in workspace mode because Go resolves by directory, not package name. **Will break per-module CI (`GOWORK=off`).**

### 2. `core/store.Backend` Abstraction — NEW, INCOMPLETE

**Status:** 30% done. Interface designed, memory backend written. But:

| Issue                                                     | Status                                                                                        |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `core/store/backend.go` (interface)                       | ✅ Staged                                                                                     |
| `memory/backend.go` (in-memory impl)                      | ⚠️ Untracked, NOT staged                                                                      |
| `pebble/backend.go` (pebble impl)                         | ⚠️ Untracked, NOT staged                                                                      |
| `storage/sql_aggregate_reader.go`                         | ⚠️ Untracked, NOT staged, **BROKEN** (references `listing` package which is `package stream`) |
| No tests for `core/store`                                 | ❌ Missing                                                                                    |
| No go.mod for `core/store` (it's a sub-package of `core`) | ✅ Correct                                                                                    |
| No documentation in AGENTS.md                             | ❌ Missing                                                                                    |

### 3. Projection Parallel Processing — PARTIAL

- `projection/options.go` has `WithParallelism()` option (staged)
- `projection/runner_live.go` has `dispatchParallel()` (staged)
- `projection/runner_parallel_test.go` is staged (new file)
- But untracked changes and unclear test coverage

### 4. AGENTS.md Update — PARTIAL

- Unstaged changes to `AGENTS.md` updating module graph and listing references
- Contains duplicate line (`sagatest` listed twice)

---

## C) NOT STARTED ❌

### From TODO_LIST.md (High-Impact)

1. **`query.Handler` generic typed return** — `any` → `TypedHandler[T]` returning `(T, error)` [v2]
2. **`io.Closer` removal from core interfaces** [v2]
3. **Global `TransactionID` branded type** [v2]
4. **`core/event` god-package split** — 90+ exports across 12 concerns, deferred
5. **PostgreSQL integration tests** — blocked on testcontainers/Docker
6. **Benchmark storage backends** (PG vs SQLite vs Pebble)
7. **Catalog diff/breaking-change detection tool**
8. **High-level test utilities** (AggregateTester, ProjectionTester, BusTester)
9. **Schema registry** — JSON Schema middleware for event validation
10. **Documentation site** (Docusaurus/MkDocs/Hugo)

### Untracked but Not Started

- `core/store.Backend` tests
- `storage/sql_aggregate_reader.go` (broken, references incomplete rename)
- Ghost directory cleanup (`stream/`, `example/stream/`)

---

## D) TOTALLY FUCKED UP 💥

### 1. Storage Module Race-Detector Build Failure

```
storage/sql_aggregate_reader.go:25:7: undefined: listing
```

The untracked file `storage/sql_aggregate_reader.go` imports `github.com/larsartmann/go-cqrs-lite/listing`, but all `.go` files in `listing/` still declare `package stream`. This works in workspace mode (Go resolves by file path) but **breaks `GOWORK=off` CI and race detection**.

### 2. Incomplete Staging Area — Dangerous State

48 files staged mixing **three unrelated concerns**:

| Concern                       | Files     | Risk                         |
| ----------------------------- | --------- | ---------------------------- |
| stream→listing rename         | ~35 files | Incomplete, will break CI    |
| core/store.Backend + adapters | ~5 files  | Untracked, unstaged          |
| Modularization proposal/docs  | ~5 files  | Safe                         |
| AGENTS.md update              | 1 file    | Unstaged, has duplicate line |

**A single monolithic commit of all staged changes would be a mess.**

### 3. gopls Stale Cache for listing/

5 compiler errors from gopls seeing `package stream_test` in renamed files. Requires gopls restart after package rename is completed.

---

## E) WHAT WE SHOULD IMPROVE 📈

### Critical (Fix Now)

1. **Complete the `stream` → `listing` rename** — Change all `package stream` → `package listing`, `package stream_test` → `package listing_test`, update doc.go
2. **Delete ghost directories** — `stream/` and `example/stream/` are empty shells
3. **Split staged changes into focused commits** — Don't mix rename + Backend + docs

### High Impact

4. **`core/store.Backend` needs tests** — Zero coverage on new abstraction
5. **Fix `storage/sql_aggregate_reader.go`** — Either complete it or remove it; currently broken
6. **Stage untracked Backend files** — `memory/backend.go`, `pebble/backend.go` are written but not staged
7. **Fix AGENTS.md duplicate line** — `sagatest` listed twice

### Process

8. **Smaller, focused commits** — Current staging area mixes 3+ concerns
9. **Per-module CI verification** — Run `GOWORK=off go build ./...` before committing renames
10. **Race detector in CI** — Currently `storage` fails race tests

### Architecture

11. **Complete `core/event` god-package split** — 242 files import it; plan exists, execution deferred
12. **Add `listing` → `storage` integration tests** — Currently missing
13. **Standardize Backend key encoding** — Document key schemas in `core/store/backend.go`

---

## F) Top #25 Things to Do Next

| #   | Priority | Task                                                               | Impact                | Effort |
| --- | -------- | ------------------------------------------------------------------ | --------------------- | ------ |
| 1   | 🔴 P0    | Complete `stream` → `listing` package rename (all declarations)    | Fixes CI              | 30min  |
| 2   | 🔴 P0    | Delete ghost `stream/` and `example/stream/` directories           | Cleanup               | 5min   |
| 3   | 🔴 P0    | Fix `storage/sql_aggregate_reader.go` broken build                 | Fixes race tests      | 1hr    |
| 4   | 🔴 P0    | Split staging area into focused commits                            | Git hygiene           | 15min  |
| 5   | 🟡 P1    | Write tests for `core/store.Backend` interface                     | Quality               | 2hr    |
| 6   | 🟡 P1    | Stage and test `memory/backend.go` + `pebble/backend.go`           | Feature complete      | 2hr    |
| 7   | 🟡 P1    | Fix AGENTS.md duplicate sagatest line + update module tree         | Docs accuracy         | 10min  |
| 8   | 🟡 P1    | Restart gopls to clear stale `stream_test` diagnostics             | DX                    | 1min   |
| 9   | 🟡 P1    | Run `go work sync` to fix `example/listing/go.mod` missing require | Module health         | 2min   |
| 10  | 🟡 P1    | Update `listing/doc.go` to say "Package listing"                   | Naming                | 2min   |
| 11  | 🟢 P2    | Add `listing` integration tests with `storage` module              | Coverage              | 4hr    |
| 12  | 🟢 P2    | Implement `query.Handler` generic typed return (`any` → `T`)       | Type safety           | 3hr    |
| 13  | 🟢 P2    | Remove `io.Closer` from core interfaces                            | ISP purity            | 4hr    |
| 14  | 🟢 P2    | Add PostgreSQL integration tests (testcontainers)                  | Coverage              | 6hr    |
| 15  | 🟢 P2    | Benchmark storage backends (PG vs SQLite vs Pebble)                | Performance           | 4hr    |
| 16  | 🟢 P2    | Split `core/event` god-package into sub-packages                   | Maintainability       | 8hr    |
| 17  | 🟢 P2    | Add catalog diff/breaking-change detection                         | Developer tooling     | 8hr    |
| 18  | 🟢 P2    | High-level test utilities (AggregateTester, ProjectionTester)      | Developer experience  | 6hr    |
| 19  | 🟢 P2    | Schema registry — JSON Schema middleware for events                | Validation            | 6hr    |
| 20  | ⚪ P3    | Global `TransactionID` branded type                                | Cross-aggregate       | 3hr    |
| 21  | ⚪ P3    | Performance regression CI — benchmark comparison per PR            | CI quality            | 4hr    |
| 22  | ⚪ P3    | Add fuzz tests for event creation, ID parsing, upcaster chain      | Robustness            | 4hr    |
| 23  | ⚪ P3    | Documentation site (Docusaurus/MkDocs/Hugo)                        | Discoverability       | 8hr    |
| 24  | ⚪ P3    | Thin PostgreSQL store adapter (no Watermill)                       | Independence          | 8hr    |
| 25  | ⚪ P3    | Thin NATS bus adapter (no Watermill)                               | Transport flexibility | 8hr    |

---

## G) My Top #1 Question I Cannot Figure Out Myself

**Is `core/store.Backend` a committed direction, or an experiment?**

The `core/store/backend.go` interface is staged but has no tests, no documentation in AGENTS.md, and untracked adapter implementations (`memory/backend.go`, `pebble/backend.go`). The `storage/sql_aggregate_reader.go` file imports `listing` (which isn't even fully renamed). I need to know:

1. Should I **complete and commit** the Backend abstraction (interface + all 3 adapters + tests)?
2. Or should I **unstage `core/store/backend.go`** and park the whole Backend concept for later?

This determines whether items #5-6 in the top 25 are "next session" or "next quarter".

---

## Test Coverage Summary

```
MODULE                                  COVERAGE
core/command                            100.0%
core/decider                             ~95%
core/event                               90.7%
core/pkg/dispatcher                      92.2%
core/pkg/id                             100.0%
core/query                               96.8%
core/store                                 — (no tests)
codec                                   100.0%
memory                                   99.1%
catalog                                  96.3%
catalog/asyncapi                         93.7%
catalog/d2                               95.0%
catalog/docserver                        89.9%
catalog/eventcatalog                     92.8%
catalog/openapi                          96.2%
middleware                               94.0%
testhelpers                              83.7%
projection                               90.4%
signing                                  93.7%
signing/multisig                         94.2%
storage                                  93.7%
saga                                     94.6%
watermill                                94.4%
pebble                                   87.8%
listing                                   — (tests pass, no coverage data collected)
```

---

## Staging Area Breakdown (48 files)

| Category              | Files                      | Status                                      |
| --------------------- | -------------------------- | ------------------------------------------- |
| stream→listing rename | 35                         | Incomplete (package declarations unchanged) |
| core/store.Backend    | 1 (staged) + 3 (untracked) | No tests                                    |
| Modularization docs   | 2                          | Ready                                       |
| Turso go.sum          | 1                          | Auto-generated                              |
| AGENTS.md             | 1                          | Unstaged, has duplicate line                |

## Unstaged Changes

| File             | Change                                   |
| ---------------- | ---------------------------------------- |
| `AGENTS.md`      | Module graph update (has duplicate line) |
| `storage/go.mod` | Added listing dependency                 |
| `go.work`        | Updated use directives                   |

## Untracked Files

| File                              | Status                                     |
| --------------------------------- | ------------------------------------------ |
| `memory/backend.go`               | Written, not staged                        |
| `pebble/backend.go`               | Written, not staged                        |
| `storage/sql_aggregate_reader.go` | Written, BROKEN (listing package mismatch) |
| `docs/status/`                    | This report                                |
