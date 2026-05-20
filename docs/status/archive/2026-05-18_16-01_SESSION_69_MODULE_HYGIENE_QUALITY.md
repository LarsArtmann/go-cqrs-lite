# Session 69: Comprehensive Status — Module Hygiene, Architecture & Quality

**Date:** 2026-05-18 16:01  
**Branch:** master  
**Commits this session:** 3  
**Prior sessions in sweep:** 65 (type safety plan + changes), 66 (go.work hygiene), 67 (type safety quality sweep), 68 (GitHub issues triage)

---

## Executive Summary

Deep architectural analysis and execution of the go-cqrs-lite multi-module monorepo. This session focused on **DAG violation elimination**, **file size compliance**, **lint warning eradication**, and **comprehensive planning**. Created an 86-task execution plan and executed the highest-impact items from Tier 1 (1%→51% value).

**Result:** 20/20 test packages pass, 0 lint issues, 0 lint warnings, 0 production files >250 lines, 0 DAG violations. 43 benchmarks. 35 sentinel errors classified.

---

## a) FULLY DONE

### S1: Storage→Memory DAG Violation Fix (CRITICAL)

**Problem:** `storage/pebble_config.go` imported `memory` in production code, violating ADR-0003's rule that infrastructure modules depend only on `core`.

**Fix:** Made `PebbleBackendMemory` return `ErrPebbleProviderRequired` instead of calling `memory.NewMemoryStore()`. Removed `memory` from `storage/go.mod` entirely.

**Verification:** `go build ./...`, `go test ./...`, `nix run .#test` all pass. `go mod graph` confirms `storage` no longer depends on `memory`.

### S2: File Size Compliance — All Production Files Under 250 Lines

| File                            | Before | After | Extracted To                  | Lines |
| ------------------------------- | ------ | ----- | ----------------------------- | ----- |
| `storage/helpers.go`            | 433    | 239   | `storage/sql_helpers.go`      | 205   |
| `catalog/asyncapi/exporter.go`  | 258    | 79    | `catalog/asyncapi/builder.go` | 182   |
| `storage/pebble_event_store.go` | 321    | 156   | `storage/pebble_helpers.go`   | 176   |
| `catalog/registry.go`           | 254    | 208   | `catalog/registry_helpers.go` | 47    |

**Result:** `find . -name "*.go" -not -name "*_test.go" -not -path "*/example/*" | xargs wc -l | awk '$1 > 250'` returns **zero files**.

### S3: Lint Warning Elimination

- Replaced deprecated `gomodguard` with `gomodguard_v2` in `.golangci.yml`
- Removed empty `import ()` block from `catalog/asyncapi/exporter.go`
- **Result:** 0 issues, 0 warnings across all 8 linted modules

### S4: Comprehensive 86-Task Execution Plan

Created `docs/planning/2026-05-18_15-10-COMPREHENSIVE-EXECUTION-PLAN.md` with:

- Pareto-sorted tasks (1%→51%, 4%→64%, 20%→80%)
- Each task ≤12 minutes
- Dependency graph (D2)
- Quality gates per phase
- 86 total tasks across 3 tiers

### S5: Documentation Updates

- `CHANGELOG.md`: Added session 68 changes (DAG fix, file splits, gomodguard migration)
- `AGENTS.md`: Added session 68 history entry
- `docs/modularization/PROJECT_GROUPS_IMPROVEMENTS.md`: Architectural analysis with recommendations

### S6: Architecture Discovery — flake.nix Uses GOWORK=off

**Critical discovery:** `flake.nix` sets `GOWORK=off` in its `shellHook`. This means:

- The project intentionally uses replace directives in sub-module go.mod files
- `go.work` is only for IDE tooling (gopls), not for builds
- All `nix run .#*` commands use GOWORK=off + replace directives
- The "remove replace directives" recommendation was WRONG for this project

---

## b) PARTIALLY DONE

None. All tasks started were completed.

---

## c) NOT STARTED (from 86-task plan)

### Tier 2 (4% → 64% of Value) — 33 tasks

- **BDD tests** for `Version`, `SchemaVersion`, `OutboxStatus`, `NodeID`, `SyncMessageType`, `Pagination` (6 tasks)
- **CatalogMeta consolidation** — extract base type, embed in 3 packages (5 tasks)
- **Table name constants** in storage — extract from magic strings (5 tasks, deferred: only 4 occurrences)
- **Production deduplication** — `catalog/schema.go` Ptr-unwrapping, `pkg/id/id.go` validation blocks (2 tasks)
- **Test deduplication** — refactor 11 test files to table-driven (11 tasks)
- **AGENTS.md module guidance** — document `GOWORK=off` pattern, replace directive rationale (3 tasks)

### Tier 3 (20% → 80% of Value) — 29 tasks

- **TransactionalStore implementation** — interface + SQLite + PG + integration tests (9 tasks)
- **Saga/Process Manager** — Core + Step + Coordinator + BDD tests (8 tasks)
- **CI/CD improvements** — GOWORK=off CI job, per-module parallel, path filters (5 tasks)
- **Documentation** — CONTRIBUTING.md, module READMEs, FEATURES.md update (6 tasks)
- **Misc** — example/todo split, `io.Closer` evaluation, storage coverage, `v0.1.0-alpha` tag (5 tasks)

---

## d) TOTALLY FUCKED UP

### Near Miss: Removing Replace Directives

Attempted to remove all `replace` directives from sub-module go.mod files based on the assumption that `go.work` handles local resolution. This **broke the build** because `flake.nix` sets `GOWORK=off` in the dev shell — all builds go through replace directives, not go.work.

**Caught by:** `go build ./...` failure from within `core/` directory. Go tried to resolve `projection@v1.1.0` from the remote (which doesn't exist as a git tag).

**Fixed by:** `git checkout --` all go.mod files. Discovered the `GOWORK=off` setting in `flake.nix` shellHook. Updated the plan to reflect this reality.

**Lesson:** Always check `flake.nix` for `GOWORK`/`GOFLAGS` settings before proposing module-level changes. The workspace file is secondary to the Nix build system in this project.

---

## e) WHAT WE SHOULD IMPROVE

1. **`integration/go.mod` has inconsistent versions** — `middleware v0.0.0-00010101000000-000000000000`, `projection v0.0.0`, `storage v0.0.0` while `core`, `memory`, `testhelpers` are `v1.1.0`. This is cosmetic (replace directives override) but confusing.

2. **`catalog/go.mod` uses `core v0.0.0`** — should be `v1.1.0` for consistency (again cosmetic, replace overrides).

3. **`example/user/go.mod` uses `catalog v0.0.0` and `middleware v0.0.0`** — same inconsistency.

4. **Coverage at 84.0%** — `testhelpers` has 0% coverage (no test files) and drags the total down. Should either exclude it from coverage or add basic tests.

5. **`go.work` vs `GOWORK=off` duality** — The project runs two modes: workspace for IDE, replace directives for builds. This is fragile. Should document clearly in AGENTS.md or consider migrating to workspace-only builds.

6. **No `go.work.sum` consistency check in CI** — If someone adds a dep to one module but forgets to run `go work sync`, the build still passes (because GOWORK=off) but IDE tooling breaks.

7. **86-task plan has items that don't apply** — The "remove replace directives" tasks (1.1.1–1.1.8) were based on a wrong assumption. Plan needs updating.

8. **No per-module CI parallelization** — All modules build/test sequentially in a single CI job.

9. **`MemoryBus.Publish` holds RLock during handler execution** — Documented low-severity concurrency issue. Could cause deadlocks if handlers call back into the bus.

10. **`example/todo/cmd/api/main.go` at 329 lines** — Over file size limit but in example code, not production. Should still be split for demo quality.

---

## f) Top 25 Things to Do Next

### High Impact (1% → 51%)

1. **Standardize `integration/go.mod` versions** — Change `middleware v0.0.0-pseudo`, `projection v0.0.0`, `storage v0.0.0` to `v1.1.0`
2. **Standardize `catalog/go.mod` core version** — `v0.0.0` → `v1.1.0`
3. **Standardize `example/user/go.mod` versions** — `catalog v0.0.0`, `middleware v0.0.0` → `v1.1.0`
4. **Update 86-task plan** — Remove "remove replace directives" tasks (1.1.1–1.1.8), add "standardize versions" tasks
5. **Exclude `testhelpers` from coverage** — Or add basic tests to boost from 84% to ~93%
6. **Add AGENTS.md guidance on GOWORK=off** — Document why replace directives exist, how flake.nix builds work

### Medium Impact (4% → 64%)

7. **BDD tests for `event.Version`** — `Int()`, `String()`, `IsZero()`, increment
8. **BDD tests for `event.SchemaVersion`** — Distinct from Version, parsing, zero value
9. **BDD tests for `storage.OutboxStatus`** — `Pending` value, string representation
10. **BDD tests for `sync.NodeID`** — Parse, MustParse, IsZero
11. **BDD tests for `sync.SyncMessageType`** — Request vs Response
12. **BDD tests for `query.Pagination`** — NewPagination, Offset(), uint safety, edge cases
13. **Consolidate `CatalogMeta`** — Extract `catalogmeta.Base`, embed in 3 packages (5 tasks)
14. **Deduplicate `catalog/schema.go`** — Ptr-unwrapping logic (1 clone group)
15. **Deduplicate `pkg/id/id.go`** — Validation blocks (1 clone group)
16. **Refactor `catalog/registry_test.go`** — Table-driven (4 clone groups)
17. **Refactor `catalog/asyncapi/exporter_test.go`** — Table-driven (5 clone groups)
18. **Refactor `catalog/schema_test.go`** — Table-driven (4 clone groups)

### Lower Impact (20% → 80%)

19. **Add `GOWORK=off` CI verification job** — Ensure replace directives work without workspace
20. **Parallelize CI** — Per-module matrix job (8 parallel jobs instead of 1 sequential)
21. **Create `CONTRIBUTING.md`** — Architecture guidelines, module structure, build commands
22. **Add `storage/README.md`** — Document store implementations and backends
23. **Split `example/todo/cmd/api/main.go`** — Extract routes and handlers (329→<250)
24. **Review `TransactionalStore` design doc** — `docs/planning/OUTBOX_TRANSACTION_API.md`
25. **Tag `v0.1.0-alpha`** — First public release after all version standardization

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should the project migrate from `GOWORK=off + replace directives` to `go.work`-only builds?**

**Current state:** `flake.nix` sets `GOWORK=off`, so all builds use per-module replace directives. `go.work` exists only for IDE tooling. This creates a dual-mode reality where:

- IDE developers get workspace-aware completion and navigation
- CI and Nix builds use replace directives
- go.work.sum can drift without anyone noticing (CI doesn't check it)

**The alternative:** Remove `GOWORK=off` from `flake.nix`, remove all replace directives, and rely purely on `go.work` for builds. This would:

- Eliminate the dual-mode confusion
- Make `go.work.sum` consistency checkable in CI
- Simplify go.mod files (no replace blocks)
- But: require `go work sync` as a CI step
- And: make `GOWORK=off go build` in individual modules impossible without publishing tags

**Why I can't decide:** This is a project infrastructure decision that depends on the publish/release workflow. If modules are published to a Go proxy, `GOWORK=off` builds are essential for verifying versioned imports work. If they're only used locally, workspace-only is cleaner. The current hybrid approach works but is fragile.

---

## Metrics

| Metric                           | Value                              |
| -------------------------------- | ---------------------------------- |
| Test packages                    | 20/20 pass                         |
| Test functions                   | ~878 (across 78 test files)        |
| Benchmarks                       | 43                                 |
| Total coverage                   | 84.0%                              |
| Production LOC                   | 11,954                             |
| Test LOC                         | 25,641                             |
| Total LOC                        | 37,595                             |
| Modules                          | 10 (incl. 2 example modules)       |
| Lint issues                      | 0 (across 8 modules)               |
| Lint warnings                    | 0 (was 8 before gomodguard_v2 fix) |
| Files over 250 lines (prod)      | 0 (was 3)                          |
| DAG violations                   | 0 (was 1: storage→memory)          |
| Sentinel errors                  | 35+                                |
| Error classifications registered | 39                                 |
| Commits since May 1              | ~170                               |
| Commits this session             | 3                                  |

---

## Files Modified This Session

| File                                                             | Change                                                                    |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `storage/pebble_config.go`                                       | Remove memory import, return ErrPebbleProviderRequired for memory backend |
| `storage/pebble_event_store_test.go`                             | Update memory backend test to expect error                                |
| `storage/go.mod`                                                 | Remove memory dependency + replace directive                              |
| `storage/helpers.go`                                             | Split: 433→239 lines (event scanning stays)                               |
| `storage/sql_helpers.go`                                         | NEW — SQL-agnostic shared helpers (205 lines)                             |
| `storage/pebble_event_store.go`                                  | Split: 321→156 lines (Save, Load, LoadFromVersion stay)                   |
| `storage/pebble_helpers.go`                                      | NEW — Delete, AppendBatch, Close, internal helpers (176 lines)            |
| `catalog/asyncapi/exporter.go`                                   | Split: 258→79 lines (struct, options, constructor), remove empty import   |
| `catalog/asyncapi/builder.go`                                    | NEW — Export logic, message/channel/operation building (182 lines)        |
| `catalog/registry.go`                                            | Split: 254→208 lines (Registry struct + methods stay)                     |
| `catalog/registry_helpers.go`                                    | NEW — Copy helpers (47 lines)                                             |
| `core/go.mod`                                                    | Removed memory+testhelpers direct deps, go mod tidy                       |
| `.golangci.yml`                                                  | `gomodguard` → `gomodguard_v2`                                            |
| `CHANGELOG.md`                                                   | Added session 68 changes                                                  |
| `AGENTS.md`                                                      | Added session 68 history                                                  |
| `docs/modularization/PROJECT_GROUPS_IMPROVEMENTS.md`             | NEW — Architectural analysis                                              |
| `docs/planning/2026-05-18_15-10-COMPREHENSIVE-EXECUTION-PLAN.md` | NEW — 86-task plan                                                        |

---

## Module Dependency Graph (After Session)

```
core (zero internal deps)
├── command/  (depends on core)
├── query/    (depends on core)
├── event/    (depends on core)
├── aggregate/ (depends on core)
├── decider/  (depends on core)
└── pkg/
    ├── id/        (depends on core)
    └── dispatcher/ (depends on core)

testhelpers → core
memory → core, testhelpers
storage → core  ← FIXED (was: core, memory)
projection → core, memory, testhelpers
middleware → core, testhelpers
catalog → core
sync → (standalone)
integration → core, memory, middleware, projection, storage, testhelpers
```

All arrows point downward. No cycles. `core` has zero internal dependencies.
