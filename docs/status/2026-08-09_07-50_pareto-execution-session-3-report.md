# Status Report: Pareto Execution Plan Session 3 (07:30 – 07:50)

> **Date:** 2026-08-09 07:50
> **Session type:** Continuation of M1–M27 Pareto Execution Plan
> **Verify gate:** GREEN — all 78 modules pass, 3833 API exports verified

---

## Summary

Fixed 2 build-breaking bugs from session 2, implemented C034 context-derivation
tracing (consumer-requested), added Badger system integration test, and wrote
go-arch-lint meta-tests. Cleaned TODO_LIST from ~50+ stale items to ~25 genuinely
open ones.

---

## Completed This Session

| Task                                 | What was done                                                                                                                                                                                                                                         | Key files                         |
| ------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- |
| **Build fix: missing slices import** | E003 file used `slices.Sort()` but didn't import `slices` — build-breaking bug from session 2.                                                                                                                                                        | `e003_e007.go`                    |
| **DQL injection fix**                | CounterIncrement used `fmt.Sprintf` with `cqrs.` — triggered injection test. Rewrote to use `QueryWithVars` with `$keyN` variables.                                                                                                                   | `dgraphengine/counter.go`         |
| **C034 context tracing**             | C034 now recognizes variables derived from `context.WithCancel`/`WithTimeout`/`WithDeadline`/`WithValue` as valid context propagation in goroutines. Fixes DiscordSync FP.                                                                            | `c034.go`, `c034_test.go`         |
| **Badger system test**               | New `TestIntegration_BadgerSource_HealthCheck` — dispatch, load, healthcheck, close against Badger engine.                                                                                                                                            | `integration_badger_test.go`      |
| **Arch-lint meta-tests**             | `TestGoArchLintConfigsAreValid` (all configs valid YAML + component paths exist) + `TestMultiPackageModulesHaveArchLintConfig` (every 3+ package module has a config).                                                                                | `main_test.go`                    |
| **signing/.go-arch-lint.yml**        | New config for signing module (3 packages: core, multisig, testutil).                                                                                                                                                                                 | `signing/.go-arch-lint.yml`       |
| **TODO_LIST cleanup**                | Removed 28+ stale items (M10 regression tests already exist, M14 already done, M23 already shared, M27 taskmanager already updated, all Dgraph items done, all M26 layer items done, all M27 docs items done, etc.). Added status notes for sections. | `TODO_LIST.md`                    |
| **Api-stability golden**             | Updated from 3827→3833 exports (auto-commit daemon added system/ Event, View, LookupInput, ProjectionSpec).                                                                                                                                           | `docs/api_surface.txt`            |
| **Taskmanager golden**               | Updated with sorted E003 output (deterministic after `slices.Sort` fix).                                                                                                                                                                              | `testdata/taskmanager_golden.txt` |

---

## Verified Already Done (stale TODOs)

| Task                                           | Status                                                                                                   |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| M14: Replace `PackagesWithRegistration`        | **Already done** — per-type tracing via `CommandTypesRegistered` map, 6 scanner population sites.        |
| M27: Taskmanager metaengine DX update          | **Already done** — uses `Register` + `NewTypeDecoder`, zero old-pattern references.                      |
| M23: DistinctValues + deferClose consolidation | **Already done** — `metaengine.ScanDistinctValues()` (4 sites), `metaengine.DeferClose()` (47+17 sites). |

---

## What Remains

### Deferred (L-effort or needs user action)

| Task                                       | Why deferred                                                                   |
| ------------------------------------------ | ------------------------------------------------------------------------------ |
| **M2: Cut CHANGELOG v4.7.0 + tag modules** | Needs coordinated `tag-release.sh` execution + user approval for pushing tags. |
| **M13: Per-module feature profiles**       | L-effort — multi-go.mod workspace analysis redesign.                           |
| **M21: ADR-0117 command lifecycle**        | L-effort — full DLQ-as-event-streams implementation.                           |
| **M24: .golangci.yml exclusion audit**     | Risky narrowing — could break the build. Low value.                            |

### Not worth doing

| Task                        | Why                                                                                |
| --------------------------- | ---------------------------------------------------------------------------------- |
| M25: PG per-test isolation  | Only one PG test exists; isolation adds complexity with no immediate benefit.      |
| M25: TestMain consolidation | No driver conflicts exist (unique names); CGo build tag complicates consolidation. |
