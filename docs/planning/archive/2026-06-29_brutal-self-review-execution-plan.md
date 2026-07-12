# Brutal Self-Review Execution Plan — 2026-06-29 (Session 2)

**Trigger:** User asked for honest self-critique + comprehensive ≤12-min/task plan + execute + push.
**Prior session shipped:** 5/6 framework gaps, 3 new modules, ONE silent data-loss bug (now fixed).
**This session's goal:** Pay down the honesty/hygiene/type-safety debt the prior session left behind.

## Tier S — Honesty & Hygiene (do first; all <12 min; no architectural decisions)

| #   | Task                                                                  | Files                                     | Impact | Effort |
| --- | --------------------------------------------------------------------- | ----------------------------------------- | ------ | ------ |
| S1  | Fix 21 lint warnings in example/projectionhost + `nix fmt`            | example/projectionhost/main.go            | High   | 10m    |
| S2  | Modernize `host_test.go` min/max (2 gopls hints)                      | projectionhost/host_test.go               | Low    | 3m     |
| S3  | Clear stale `unusedwrite` diagnostic in integration_test              | projectionhost/integration_test.go        | Low    | 3m     |
| S4  | Truth-fix scenario "import cycle" lie; evaluate real `decider` import | scenario/dsl.go                           | Med    | 8m     |
| S5  | Write ADR-0042 for the pure-replay DLQ correctness decision           | docs/adr/0042-pure-replay-dead-letters.md | Med    | 10m    |
| S6  | Extract duplicated `capturingSlogHandler` → `testutil/slogtest.go`    | testutil/, projectionhost/, scheduling/   | Med    | 10m    |

## Tier A — Type Safety & API Completeness

| #   | Task                                                                                  | Files                            | Impact | Effort |
| --- | ------------------------------------------------------------------------------------- | -------------------------------- | ------ | ------ |
| A1  | Make `Timer` generic: `Timer[P any]` (cascade TimerStore/Dispatch/Scheduler)          | scheduling/\*                    | High   | 12m    |
| A2  | Add entry-scoped `Delete(name, eventID)` to projectionhost DLQ                        | projectionhost/dlq.go, host.go   | High   | 10m    |
| A3  | Replace projectionhost worker hand-rolled backoff with `cenkalti/backoff/v5` (jitter) | projectionhost/worker.go, go.mod | High   | 10m    |
| A4  | Replace scheduling retry with `cenkalti/backoff/v5` (jitter)                          | scheduling/scheduler.go, go.mod  | Med    | 8m     |

## Tier B — Production Readiness

| #   | Task                                                           | Files                         | Impact | Effort |
| --- | -------------------------------------------------------------- | ----------------------------- | ------ | ------ |
| B1  | SQL `CheckpointStore` (SQLite + Postgres) for projectionhost   | storage/, projectionhost/     | High   | 12m    |
| B2  | Pebble `CheckpointStore` for projectionhost                    | storage/pebble/               | Med    | 8m     |
| B3  | Prometheus metrics for projectionhost (lag, DLQ depth, errors) | projectionhost/, prometheus/  | High   | 12m    |
| B4  | Stress test projectionhost with 10K events                     | projectionhost/stress_test.go | Med    | 10m    |

## Tier C — Library Trust & DX

| #   | Task                                                                      | Files                                    | Impact | Effort |
| --- | ------------------------------------------------------------------------- | ---------------------------------------- | ------ | ------ |
| C1  | `example/deriver` runnable demo (deriver has zero consumers/examples)     | example/deriver/                         | Med    | 10m    |
| C2  | SKILL.md reliability recipe (host + idempotency + DLQ trio)               | SKILL.md                                 | Med    | 8m     |
| C3  | DLQ unification ADR — present options, **NO execution** (BLOCKED on user) | docs/adr/0043-dlq-unification-options.md | High   | 10m    |

## Tier D — Polish (lower priority; pick up if time remains)

| #   | Task                                                                  | Files                                    | Impact | Effort |
| --- | --------------------------------------------------------------------- | ---------------------------------------- | ------ | ------ |
| D1  | Pebble `SetIfAbsent` two-adapter test (document shared-adapter limit) | storage/pebble/adapter_test.go           | Low    | 8m     |
| D2  | SSE zero-alloc benchmark (validate the claim)                         | transport/http/                          | Low    | 8m     |
| D3  | Pebble `TimerStore` for scheduling                                    | storage/pebble/                          | Low    | 10m    |
| D4  | SQL DLQ store for projectionhost                                      | storage/, projectionhost/                | Med    | 12m    |
| D5  | Nix flake app for multi-module tag release                            | flake.nix                                | Med    | 12m    |
| D6  | go.work integrity CI check                                            | .github/workflows/ci.yml                 | Low    | 8m     |
| D7  | ADR for eventtest nested-module `-e` workaround (decide & document)   | docs/adr/0044-eventtest-nested-module.md | Med    | 8m     |
| D8  | go.sum lockfile strategy doc                                          | docs/                                    | Low    | 8m     |
| D9  | Audit all `any` at library boundaries (new modules)                   | cross-module                             | Low    | 12m    |
| D10 | Split projectionhost example into per-type files                      | example/projectionhost/                  | Low    | 8m     |

## Sorting Rationale

- **Customer value** (for a library) = consumer trust. Correctness > type safety > production readiness > polish.
- **Impact ÷ effort** within each tier.
- Tier S first because every item is <12m and removes either a lie, a stale claim, or shipped debt.
- Tier A next because type safety + jitter + entry-scoped delete are real reliability wins.
- Tier C3 (DLQ ADR) is **written but not executed** — the unification needs your call (A/B/C/D from prior session).

## Execution Contract

- One commit per task (smallest self-contained change).
- `nix fmt` + `go test ./...` before each commit.
- BuildFlow pre-commit hook must stay green.
- `git push` at the end.
- Status report at the end summarizing what landed.
