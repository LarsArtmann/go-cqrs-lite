# Status Report — Session 101

**Date:** 2026-05-25 23:43 CEST
**Branch:** `master` (up to date with `origin/master`)
**Last 5 commits:**

| Commit    | Message                                                                    |
| --------- | -------------------------------------------------------------------------- |
| `549f17a` | docs(status): add session 100 comprehensive status report                  |
| `48912a0` | fix(catalog,event/todo): enrich error messages with more context           |
| `5bc35d3` | docs(AGENTS.md): update coverage, catalog structure, session 100           |
| `613f989` | style(catalog): rename svcID params to serviceID in asyncapi and openapi   |
| `3910f4f` | refactor(catalog,memory): use catalog.MessageID in maps, rename short vars |

---

## a) FULLY DONE

### Core Library (10 modules, all green)

| Module                | Test Coverage | Status                                                  |
| --------------------- | ------------- | ------------------------------------------------------- |
| `core/command`        | 92.3%         | ✅ Stable, branded IDs, error taxonomy                  |
| `core/decider`        | 93.6%         | ✅ Pure-function aggregates, time-travel                |
| `core/event`          | 93.8%         | ✅ Store/Bus/SnapshotStore interfaces, immutable events |
| `core/pkg/dispatcher` | 100.0%        | ✅ Generic dispatcher, lifecycle mixin                  |
| `core/pkg/id`         | 100.0%        | ✅ Branded IDs via go-branded-id                        |
| `core/query`          | 98.4%         | ✅ Typed dispatch, pagination                           |
| `memory`              | 99.6%         | ✅ In-memory Store/Bus/SnapshotStore                    |
| `middleware`          | 100.0%        | ✅ Logging, retry, recovery, validation, metrics        |
| `testhelpers`         | 91.3%         | ✅ Shared test utilities                                |
| `projection`          | 94.4%         | ✅ Runner with replay, Builder with On[T]()             |

### Catalog System

| Sub-package                   | Coverage | Status                                      |
| ----------------------------- | -------- | ------------------------------------------- |
| `catalog` (core)              | 96.3%    | ✅ Registry, SchemaFromType[T], branded IDs |
| `catalog/asyncapi`            | 93.7%    | ✅ AsyncAPI 3.0 YAML/JSON export            |
| `catalog/d2`                  | 95.0%    | ✅ D2 diagram text export                   |
| `catalog/eventcatalog`        | 85.7%    | ✅ EventCatalog MDX generator (auto-derive) |
| `catalog/openapi`             | 94.4%    | ✅ OpenAPI 3.0 export                       |
| `catalog/docserver`           | 90.1%    | ✅ HTTP server for generated docs           |
| `catalog/internal/caseutil`   | 100.0%   | ✅ Case conversion utilities                |
| `catalog/internal/schemautil` | 84.2%    | ✅ Schema helpers                           |

### Storage Module

| Sub-package | Coverage | Status                                                |
| ----------- | -------- | ----------------------------------------------------- |
| `storage`   | 88.7%    | ✅ SQLite, Turso, event store, snapshot store, outbox |

### Integration & Examples

| Module                | Status                                             |
| --------------------- | -------------------------------------------------- |
| `integration/command` | ✅ Middleware chain tests                          |
| `integration/event`   | ✅ BDD + benchmark tests                           |
| `integration/query`   | ✅ Middleware chain tests                          |
| `example/user`        | ✅ Full CQRS + Decider + middleware + EventCatalog |
| `example/todo`        | ✅ Typed handlers, Pagination, Turso sync          |

### Infrastructure

- ✅ All Go modules at `go 1.26.3` — consistent across workspace
- ✅ Nix flake with build/test/lint/format/vet/coverage/check apps
- ✅ GitHub Actions CI (Nix-based)
- ✅ Zero TODO/FIXME/HACK comments in production code
- ✅ Zero production files over 250 lines (largest: `catalog/registry.go` at 369, over limit — see improvements)
- ✅ All tests pass, `go vet` clean
- ✅ Only 2 pre-existing lint issues (noinlineerr in command/query dispatcher)

### Session 100 Work (just completed)

- ✅ Error message enrichment in `catalog/eventcatalog/writer.go`, `example/todo/commands/create_todo.go`, `example/todo/commands/update_todo.go`
- ✅ Correctly identified false positives: `decider/load.go:67` (already includes context), `storage/turso_sync.go:27` (authToken is a secret)

---

## b) PARTIALLY DONE

### EventCatalog Coverage (85.7% — down from 91.3%)

- The auto-derive feature (`auto_derive.go`) was added but coverage dropped
- Missing test coverage for error paths in `exporter_resources.go` and `writer.go`
- **Priority:** Medium

### `catalog/registry.go` (369 lines)

- Over the 250-line project convention
- Has deep copy methods and merge logic that could be extracted
- **Priority:** Low-Medium

### Replace Directives (23 across 9 go.mod files)

- All replace directives for intra-workspace modules are redundant with `go.work`
- Not harmful but adds noise; should be cleaned up
- **Priority:** Low

### `storage/sql_base.go` (untracked)

- New file extracted from storage module, not yet committed
- Contains `sqlBase` struct and `newSQLBase` constructor
- **Priority:** Commit it

---

## c) NOT STARTED

### High-Value Features

1. **Saga/Process Manager** — Design doc exists (`docs/planning/SAGA_DESIGN.md`) but no implementation
2. **Watermill integration** — Listed in architecture as "planned" but not started
3. **Outbox Transaction API** — Design doc exists (`docs/planning/OUTBOX_TRANSACTION_API.md`) but not implemented
4. **Event versioning/migration** — No design doc, no implementation
5. **Stream/slice-based event loading** — No `event.Stream` or iterator pattern

### Documentation

6. **GoDoc/Go Reference docs** — No hosted API documentation
7. **Getting Started guide** — `docs/getting-started.md` exists but needs review
8. **README.md** — Needs review for accuracy against current state

### Code Quality

9. **`query.Handler` returns `any`** — Known issue, workaround via `DispatchTyped[T]`. Design doc closed as "acceptable"
10. **Root-level markdown files** — 5 files (`BDD_TESTS_REVIEW.md`, `DOMAIN_GLOSSARY.md`, `FEATURES.md`, `PUBLIC_OR_PRIVATE.md`, `TODO_LIST.md`) sitting in repo root; should be in `docs/`
11. **Empty `report/` directory** — Contains stale `jscpd-report.json`, should be cleaned
12. **122 archived status reports** — `docs/status/archive/` is very large; could be pruned

---

## d) TOTALLY FUCKED UP! 🚨

### Nothing is catastrophically broken.

All tests pass. All modules build. No data loss risk. No security vulnerabilities known. The project is in excellent shape.

**However, there are two items of concern:**

1. **`report/jscpd-report.json` (487KB)** — A code duplication report sitting in the repo root, uncommitted and un-.gitignored. This is a build artifact, not source code. It should be in `.gitignore`.

2. **`catalog/eventcatalog` coverage dropped from 91.3% to 85.7%** — The auto-derive feature added code without proportional test coverage. This needs attention before the next release.

---

## e) WHAT WE SHOULD IMPROVE!

### Code Quality

| Issue                                           | Severity | Detail                                                         |
| ----------------------------------------------- | -------- | -------------------------------------------------------------- |
| `catalog/registry.go` over 250 lines            | Medium   | 369 lines. Extract deep copy methods to `registry_copy.go`     |
| `catalog/eventcatalog/exporter.go` at 302 lines | Medium   | Over 250-line limit. Split export logic                        |
| 23 redundant `replace` directives               | Low      | Remove from all go.mod files; `go.work` handles this           |
| `noinlineerr` lint warnings (2)                 | Low      | `core/command/dispatcher.go:59`, `core/query/dispatcher.go:83` |
| `query.Handler` returns `any`                   | Low      | Architectural; mitigated by `DispatchTyped[T]`                 |
| `stret/testify` as indirect dep in core/catalog | Low      | Should be direct or removed via `go mod tidy`                  |

### Project Hygiene

| Issue                                                    | Severity | Detail                                                                                                             |
| -------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------ |
| `report/` directory with 487KB artifact                  | Medium   | Add to `.gitignore`, delete the file                                                                               |
| Root-level .md files                                     | Low      | Move `BDD_TESTS_REVIEW.md`, `DOMAIN_GLOSSARY.md`, `FEATURES.md`, `PUBLIC_OR_PRIVATE.md`, `TODO_LIST.md` to `docs/` |
| 122 archived status reports                              | Low      | Could prune older ones                                                                                             |
| `docs/architecture-understanding/` has session artifacts | Low      | Move session-specific files to `docs/status/`                                                                      |
| `docs/quality/` has session-specific files               | Low      | Consolidate into status                                                                                            |

### Architecture

| Issue                          | Severity | Detail                                                               |
| ------------------------------ | -------- | -------------------------------------------------------------------- |
| No Saga/Process Manager        | Medium   | Design doc exists, implementation needed for real-world usage        |
| No event versioning            | Medium   | Essential for long-lived event-sourced systems                       |
| No stream-based event loading  | Low      | Current API loads all events into memory; large aggregates could OOM |
| No snapshot strategy interface | Low      | Decider has snapshot support but no pluggable strategy               |

### Testing

| Issue                                 | Severity | Detail                                       |
| ------------------------------------- | -------- | -------------------------------------------- |
| `catalog/eventcatalog` coverage 85.7% | Medium   | Dropped from 91.3%; needs attention          |
| `catalog/internal/schemautil` 84.2%   | Low      | Below project target of 80%+ but close       |
| `storage` at 88.7%                    | Low      | Could be improved with more error path tests |

---

## f) Top #25 Things We Should Get Done Next

### Tier 1: Ship-Blockers (must do before v1.0)

1. **Commit `storage/sql_base.go`** — Untracked file, part of storage refactor
2. **Add `report/` to `.gitignore`** — Prevent build artifacts from being committed
3. **Fix `catalog/eventcatalog` coverage** — Get back above 90%
4. **Remove 23 redundant `replace` directives** — Clean go.mod files across all modules
5. **Fix 2 `noinlineerr` lint issues** — `core/command/dispatcher.go:59`, `core/query/dispatcher.go:83`

### Tier 2: Quality & Hygiene (should do soon)

6. **Split `catalog/registry.go`** (369 lines) — Extract deep copy to `registry_copy.go`
7. **Split `catalog/eventcatalog/exporter.go`** (302 lines) — Extract export functions
8. **Run `go mod tidy` across all modules** — Clean up indirect deps (testify, rogpeppe)
9. **Move root-level .md files to `docs/`** — Better project organization
10. **Clean up `docs/architecture-understanding/`** — Move session artifacts to status
11. **Prune `docs/status/archive/`** — 122 files is excessive
12. **Update `TODO_LIST.md` and `FEATURES.md`** — Last updated 2026-05-25, verify against current code

### Tier 3: Feature Work (important for library adoption)

13. **Implement Saga/Process Manager** — Design doc exists, high value for consumers
14. **Implement Outbox Transaction API** — Design doc exists, needed for production use
15. **Add event versioning/migration** — Essential for long-lived systems
16. **Add stream-based event loading** — Iterator pattern for large event streams
17. **Implement Watermill integration** — Listed as planned in architecture

### Tier 4: Polish & DX (nice to have)

18. **Add GoDoc examples** — Runnable example functions for key types
19. **Review and update `docs/getting-started.md`** — Ensure accuracy
20. **Review and update `README.md`** — Reflect current module structure
21. **Add `storage/sql_base.go` tests** — New untracked file needs test coverage
22. **Improve `catalog/internal/schemautil` coverage** — 84.2% → 90%+
23. **Add snapshot strategy interface** — Pluggable snapshot policies for decider
24. **Review `example/user/main.go`** (340 lines) — Over 250-line convention
25. **Clean up `docs/quality/` directory** — Consolidate session-specific files

---

## g) Top #1 Question I Cannot Figure Out Myself

**What is the target release timeline for v1.0?**

The project has excellent quality metrics (97/100 branching-flow score, 92-100% coverage across most modules, zero critical issues). But I cannot determine:

- Is v1.0 blocked on Saga/Process Manager, or can it ship without it?
- Is Watermill integration a v1.0 requirement or a post-release module?
- Should `storage/turso_sync.go` and the Turso dependency be part of the v1.0 surface, or is it still experimental?

This matters because it determines whether items #13-17 in the top-25 list are blockers or backlog.

---

## Test Coverage Summary (Current)

| Package                       | Coverage | Trend          |
| ----------------------------- | -------- | -------------- |
| `core/pkg/dispatcher`         | 100.0%   | —              |
| `core/pkg/id`                 | 100.0%   | —              |
| `middleware`                  | 100.0%   | —              |
| `catalog/internal/caseutil`   | 100.0%   | —              |
| `memory`                      | 99.6%    | —              |
| `core/query`                  | 98.4%    | —              |
| `catalog`                     | 96.3%    | ↓ (was 96.8%)  |
| `catalog/d2`                  | 95.0%    | —              |
| `catalog/openapi`             | 94.4%    | —              |
| `projection`                  | 94.4%    | —              |
| `core/event`                  | 93.8%    | —              |
| `catalog/asyncapi`            | 93.7%    | —              |
| `core/decider`                | 93.6%    | —              |
| `core/command`                | 92.3%    | —              |
| `testhelpers`                 | 91.3%    | —              |
| `catalog/docserver`           | 90.1%    | —              |
| `storage`                     | 88.7%    | ↓ (was 89.3%)  |
| `catalog/internal/schemautil` | 84.2%    | —              |
| `catalog/eventcatalog`        | 85.7%    | ↓↓ (was 91.3%) |

**Modules that dropped coverage since last report:**

- `catalog` — 96.8% → 96.3% (new auto-derive code)
- `storage` — 89.3% → 88.7% (sql_base extraction)
- `catalog/eventcatalog` — 91.3% → 85.7% (auto-derive feature added without tests)

---

## Lint Status

```
2 issues (pre-existing, not from recent changes):
* core/command/dispatcher.go:59 — noinlineerr
* core/query/dispatcher.go:83 — noinlineerr
```

## Build Status

✅ `go build` — clean
✅ `go vet` — clean
✅ `go test` — all 23 packages pass
✅ `nix fmt` — formatting passes
⚠️ `nix run .#lint` — 2 pre-existing issues

---

_Generated: 2026-05-25 23:43 CEST | Session 101_
