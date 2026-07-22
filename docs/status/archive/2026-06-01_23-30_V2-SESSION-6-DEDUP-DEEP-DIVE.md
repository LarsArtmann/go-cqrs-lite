# Comprehensive Status Report — 2026-06-01 23:30

> **Session 6** · V2.0.0 Release Cycle · Code Quality & Deduplication Deep-Dive

## Executive Summary

| Metric           | Value                                                |
| ---------------- | ---------------------------------------------------- |
| Branch           | master (pushed)                                      |
| Go Version       | 1.26.3                                               |
| Total Modules    | 31 (22 library + 6 examples + 1 integration + 2 cmd) |
| Total Go Files   | 484                                                  |
| Total Go Lines   | 63,184                                               |
| Test Packages    | 37 (all pass ✅)                                     |
| Lint Issues      | 0 across 22 modules ✅                               |
| Avg Coverage     | ~92.7%                                               |
| Clones (t=30)    | 12 groups (all accepted, 0 actionable)               |
| Clones (t=50)    | 3 groups (all accepted)                              |
| Open TODOs       | 14 / 283 total (95.1% complete)                      |
| Sessions to Date | 6 in V2.0.0 cycle (160+ historical)                  |

---

## A) FULLY DONE ✅

### Core Architecture

- **Event Sourcing**: Event, Store, Sink/Source ISP split, Journal, SeekableJournal, BackwardsSource
- **CQRS**: Command dispatcher, Query dispatcher with typed bookend pattern, Decider (pure-function aggregate)
- **Branded IDs**: `id.Of[T]` via go-branded-id + ULID, 8 built-in types (EventID, AggregateID, CommandID, etc.)
- **Reactive Streams**: `samber/ro` Subject[Event/Command/Query], operators (Filter, Map, Scan, Tap, Replay)
- **Schema Evolution**: Upcaster, VersionedStore, upcasterRegistry
- **Snapshot**: SnapshotStore, EveryNEvents strategy, auto-snapshot after Execute
- **Catalog/Documentation**: 5 exporters (AsyncAPI, OpenAPI, D2, EventCatalog, DocServer)
- **Middleware**: 8 concerns × 3 message types = 24 middleware factories (logging, retry, recovery, validation, metrics, OTel tracing, OTel metrics, circuit breaker)
- **Event Signing**: HMAC-SHA256, Ed25519, multisig, middleware
- **Storage Backends**: SQL (PostgreSQL/SQLite/Turso), Pebble KV, In-Memory
- **Projection**: Runner (replay+live), HandlerRegistry, Builder with On[T](<>)
- **Listing**: Aggregate listing, tombstone detection, InMemoryAggregateReader
- **Codec**: JSON + Raw passthrough, pluggable encoding
- **Code Generation**: `cqrs-gen` typed handler registration from Go structs

### Quality Gates (All Green)

- ✅ Build: all 32 workspace modules compile
- ✅ Tests: 37/37 packages pass
- ✅ Lint: 0 issues across 22 modules (80+ linters via golangci-lint)
- ✅ Max file length: 250 lines enforced
- ✅ Max function length: 30 lines enforced
- ✅ Cognitive complexity: 35 max per function
- ✅ No `any` in production code (except dialect.go for database/sql)
- ✅ Strong types everywhere (branded IDs, typed errors)

### V2.0.0 Milestones Completed

- ✅ `core/` dissolution — flat module layout (event/, command/, query/)
- ✅ `stream/` → `listing/` rename
- ✅ `Event.Context()` removal from interface
- ✅ `Deadline()` promoted to Event interface
- ✅ Outbox removal (simplified to publish-only)
- ✅ Metadata value migration (string-only values)
- ✅ Codec extraction (separate module)
- ✅ Error classification hardened (5-family taxonomy)
- ✅ OTel telemetry naming standardized
- ✅ `io.Closer` kept on Store/Bus interfaces (decision: correct ISP)
- ✅ All import paths updated (core/ → flat layout)
- ✅ CHANGELOG.md updated for v2.0.0

### This Session's Work

- ✅ Deduplication analysis at threshold 30: 14 → 12 clone groups
- ✅ Extracted `saveTestEvents` helper in schema/versioned_source_test.go
- ✅ Net reduction: -48 lines, +25 lines (23 lines saved)
- ✅ Deep reflection on all 12 remaining clones — all accepted with documented rationale
- ✅ Verified: checkClosed, error sentinels, test helpers are intentional domain-specific duplication

---

## B) PARTIALLY DONE ⚠️

| Module             | Coverage | Status             | What's Missing                                        |
| ------------------ | -------- | ------------------ | ----------------------------------------------------- |
| **storage**        | 72.7%    | Below target (80%) | SQLSnapshotStore edge cases, error path coverage      |
| **turso**          | 28.6%    | Far below target   | Connector lifecycle, sync, embedded LibSQL edge cases |
| **catalog/schema** | 86.1%    | Near target        | JSON Schema reflection edge cases                     |
| **pebble**         | 88.0%    | Near target        | Error paths in iteration, corruption handling         |

---

## C) NOT STARTED 🔲

| Item                                                   | Priority | Notes                                      |
| ------------------------------------------------------ | -------- | ------------------------------------------ |
| Push v2.0.0 tags to remote                             | BLOCKER  | 22 `replace` directives need removal first |
| Remove `replace` directives from go.mod                | BLOCKER  | Requires published tags                    |
| PostgreSQL integration tests (testcontainers)          | BLOCKED  | Needs CI infrastructure                    |
| Performance regression CI                              | Medium   | Benchmark comparison on each PR            |
| Fuzz tests (event creation, ID parsing, schema)        | Medium   | Robustness                                 |
| BDD tests for Version, SchemaVersion, Pagination types | Medium   | Coverage                                   |
| Rewrite example/user/ to demonstrate full CQRS stack   | Medium   | Documentation value                        |
| Benchmark storage backends (PG vs SQLite vs Pebble)    | Low      | Performance                                |
| Pre-commit hooks (gofumpt, goimports, 350-line limit)  | Low      | Developer experience                       |
| ROADMAP.md                                             | Low      | Long-term vision documentation             |

---

## D) TOTALLY FUCKED UP 💥

> Nothing is broken right now. The codebase is clean, green, and healthy.

**Historical issues that were fixed:**

- The `core/` module dissolution was messy (multiple sessions, many import path fixes)
- The `stream/` → `listing/` rename required touching 50+ files
- 160+ status reports accumulated in `docs/status/` — most are noise from rapid iteration

---

## E) WHAT WE SHOULD IMPROVE 🔧

### Architecture

1. **`query.Handler` returns `any`** — This is a Go limitation but the typed bookend pattern could be better documented. Consider if a `Query[A]` / `Result[A]` generic type could reduce boilerplate.
2. **Test helper explosion** — `noopCommandHandler` is copied 4× across test packages. A `commandtest`/`querytest` sub-package was rejected as over-engineering, but the 5× copies still smell.
3. **Status report bloat** — 88 status files in `docs/status/`. Should archive or consolidate.
4. **Error sentinel namespacing** — `ErrDispatcherClosed` is defined 3× (dispatcher, command, query) with only prefix differences. This is intentional for public API but creates a maintenance surface.

### Test Coverage

5. **turso at 28.6%** — This is the weakest module by far. Needs focused test investment.
6. **storage at 72.7%** — Below the 80% target. Error paths and snapshot edge cases need coverage.
7. **No integration test infrastructure** — PostgreSQL tests require testcontainers, which isn't set up.

### Developer Experience

8. **No ROADMAP.md** — Long-term vision isn't documented.
9. **Replace directives** — 22 `replace` directives in go.mod files block v2.0.0 release.
10. **No pre-commit hooks** — Formatting and line-length checks are manual.

### Type Model

11. **AggregateRef could be a branded type** — Currently a struct with Type + ID. Could be `id.Of[aggregateRefMarker]` for stronger type safety.
12. **Event.Version is `int`** — Could be a named type `event.Version` with methods (already done ✅).
13. **Event.Type is a branded string** — Good, but `event.AggregateType` could share a generic `branded.String[T]` foundation.

---

## F) Top #25 Things to Get Done Next

Sorted by **Impact × Ease** (highest first):

| #   | Item                                                          | Impact      | Effort | Category      |
| --- | ------------------------------------------------------------- | ----------- | ------ | ------------- |
| 1   | Push v2.0.0 tags to remote                                    | 🔴 Critical | Small  | Release       |
| 2   | Remove `replace` directives from go.mod                       | 🔴 Critical | Small  | Release       |
| 3   | Increase turso test coverage to 70%+                          | 🟠 High     | Medium | Quality       |
| 4   | Increase storage test coverage to 85%+                        | 🟠 High     | Medium | Quality       |
| 5   | Add ROADMAP.md                                                | 🟠 High     | Small  | Documentation |
| 6   | Rewrite example/user/ as comprehensive demo                   | 🟡 Medium   | Medium | Documentation |
| 7   | Add fuzz tests for event creation + ID parsing                | 🟡 Medium   | Medium | Robustness    |
| 8   | Add PostgreSQL integration tests (testcontainers)             | 🟡 Medium   | Large  | Quality       |
| 9   | Performance regression CI (benchmark comparison)              | 🟡 Medium   | Medium | CI/CD         |
| 10  | Pre-commit hooks (gofumpt, goimports, line limits)            | 🟡 Medium   | Small  | DX            |
| 11  | Parallelize CI matrix (one job per module)                    | 🟡 Medium   | Small  | CI/CD         |
| 12  | Add BDD tests for Version, SchemaVersion, Pagination          | 🟡 Medium   | Medium | Quality       |
| 13  | Archive/consolidate docs/status/ (88 → ~10 files)             | 🟢 Low      | Small  | Cleanup       |
| 14  | Increase projection coverage to 95%+                          | 🟢 Low      | Small  | Quality       |
| 15  | Add listing SQL reader tests                                  | 🟢 Low      | Small  | Quality       |
| 16  | Benchmark storage backends (PG vs SQLite vs Pebble)           | 🟢 Low      | Medium | Performance   |
| 17  | Add E2E throughput benchmarks                                 | 🟢 Low      | Medium | Performance   |
| 18  | Document query.Handler `any` → TypedHandler pattern better    | 🟢 Low      | Small  | Documentation |
| 19  | Consider AggregateRef branded type                            | 🟢 Low      | Small  | Architecture  |
| 20  | Add ServerReceivedAt/ServerStoredAt timestamps                | 🟢 Low      | Medium | Feature       |
| 21  | Add Filter/Predicate types for context queries                | 🟢 Low      | Medium | Feature       |
| 22  | Catalog diff/breaking-change detection tool                   | 🔵 Future   | Large  | Tool          |
| 23  | High-level test utilities (AggregateTester, ProjectionTester) | 🔵 Future   | Large  | DX            |
| 24  | Make transactional projection contract explicit               | 🔵 Future   | Large  | Architecture  |
| 25  | Move example/todo to own repository                           | 🔵 Future   | Small  | Cleanup       |

---

## G) Top #1 Question I Cannot Figure Out Myself

**What is the release strategy for v2.0.0?**

The codebase has 22 `replace` directives in go.mod files pointing to local paths. These exist because no v2.0.0 tags have been pushed to the remote. The AGENTS.md says: _"replace directives required until v1.0.0 tags pushed to remote"_.

**I cannot resolve this without your direction:**

1. Should I push v2.0.0 tags for all 22 library modules simultaneously?
2. Is there a specific tag naming convention (e.g., `event/v2.0.0`, `command/v2.0.0`)?
3. Should the `replace` directives be removed in the same commit as the tag push, or as a follow-up?
4. Is there a CI/CD pipeline that validates the tagged versions work without `replace` directives?

This is the **single biggest blocker** for the project right now. Everything else is incremental improvement.

---

## Clone Analysis (Threshold 30)

12 groups remain, all accepted:

| Group                                    | Files                                       | Reason Accepted                                          |
| ---------------------------------------- | ------------------------------------------- | -------------------------------------------------------- |
| event BDD assertions (3)                 | event/event_bdd_test.go                     | Standard Ginkgo patterns                                 |
| command test helpers (3)                 | command/, integration/command/, middleware/ | Different test packages, different handler types         |
| command/query checkClosed (2)            | command/dispatcher.go, query/dispatcher.go  | Domain-specific public API error codes                   |
| signing test helpers (2)                 | signing/, signing/multisig/                 | Already share internal/testutil                          |
| example compile tests (2)                | example/listing/, example/saga-pattern/     | Separate Go modules                                      |
| decider test helpers (2)                 | decider/decider_helpers_test.go             | Different functions (Save vs SetSnapshot)                |
| catalog builders (2)                     | catalog/internal/cattest/builders.go        | Structural similarity only, one calls the other          |
| integration/query helpers (2)            | integration/query/, middleware/             | Different test packages                                  |
| event reactive subscribe (2)             | event/reactive_test.go                      | Standard test setup                                      |
| catalog AddServiceWith\* (2)             | catalog/internal/cattest/builders.go        | Thin wrappers, already deduped via addServiceWithMessage |
| pebble/storage wrapper (2)               | pebble/, storage/                           | Different modules, no shared test package                |
| middleware/query failingQueryHandler (2) | middleware/, query/                         | Different test packages                                  |

**At threshold 50 (industry standard): 3 groups remain.** Zero actionable production code duplication.

---

## Coverage Breakdown

| Module                    | Coverage  | Status              |
| ------------------------- | --------- | ------------------- |
| codec                     | 100.0%    | ✅                  |
| decider                   | 100.0%    | ✅                  |
| catalog/internal/caseutil | 100.0%    | ✅                  |
| query                     | 97.1%     | ✅                  |
| dispatcher                | 97.0%     | ✅                  |
| otel                      | 96.4%     | ✅                  |
| catalog/openapi           | 96.2%     | ✅                  |
| watermill                 | 96.0%     | ✅                  |
| catalog                   | 95.9%     | ✅                  |
| command                   | 94.9%     | ✅                  |
| catalog/d2                | 95.0%     | ✅                  |
| middleware                | 94.5%     | ✅                  |
| id                        | 94.5%     | ✅                  |
| signing/multisig          | 94.1%     | ✅                  |
| signing                   | 93.9%     | ✅                  |
| listing                   | 93.8%     | ✅                  |
| catalog/asyncapi          | 93.7%     | ✅                  |
| catalog/eventcatalog      | 92.8%     | ✅                  |
| snapshot                  | 92.3%     | ✅                  |
| schema                    | 91.4%     | ✅                  |
| projection                | 91.3%     | ✅                  |
| catalog/docserver         | 90.1%     | ✅                  |
| cmd/cqrs-gen              | 89.9%     | ✅                  |
| event                     | 89.0%     | ✅                  |
| pebble                    | 88.0%     | ✅                  |
| catalog/schema            | 86.1%     | ✅                  |
| memory                    | 99.1%     | ✅                  |
| **storage**               | **72.7%** | ⚠️ Below 80%        |
| **turso**                 | **28.6%** | 🔴 Far below target |

---

## Verification

| Check                    | Result                        |
| ------------------------ | ----------------------------- |
| `go test ./... -count=1` | ✅ 37/37 packages pass        |
| `nix run .#lint`         | ✅ 0 issues across 22 modules |
| `go vet ./...`           | ✅ Clean                      |
| `art-dupl -t 30`         | ✅ 12 groups, 0 actionable    |
| `art-dupl -t 50`         | ✅ 3 groups, 0 actionable     |
| Git working tree         | ✅ Clean                      |
