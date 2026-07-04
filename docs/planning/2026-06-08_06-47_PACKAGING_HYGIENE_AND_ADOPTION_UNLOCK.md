# Packaging Hygiene & Adoption Unlock Plan

> **Date**: 2026-06-08
> **Source**: Feedback from project-discovery-sdk — "Why I Can't Use You"
> **Goal**: Make go-cqrs-lite genuinely "lite" by cleaning dependency graph, no architecture changes.

## Pareto Breakdown

### 1% → 51% impact

**go.mod audit**: Move test-only deps out of `event/go.mod`. This single change kills the "event depends on 7 siblings" perception immediately.

### 4% → 64% impact

**go.mod audit + eventtest as separate module**: Full separation of test helpers into their own `go.mod` so event/ becomes a true leaf (2 production siblings + 2 external deps).

### 20% → 80% impact

**go.mod audit + eventtest + reactive split + OTel shim**: The complete packaging hygiene pass that makes every module genuinely lightweight.

## Execution Status

### Done

- ✅ **go mod tidy** all modules — no stale deps found
- ✅ **Dep budget CI** — `scripts/check-module-layers.sh` now enforces per-module direct dep limits
- ✅ **Nix flake wiring** — `nix run .#check-layers` runs layer + budget checks
- ✅ **AGENTS.md** updated with dep budget principle and check-layers command

### Blocked — Go module system limitations

- ✅ **eventtest as separate module** — RESOLVED (2026-07-05). The original claim "Go doesn't support nested modules" was incorrect. Nested modules ARE supported, but the directory must match the module path per the Go spec. Moved from `event/eventtest/` to `event/v3/eventtest/` so VCS resolution finds `go.mod` at the expected path. See `docs/adr/0044-eventtest-module-path-fix.md`.
- ⚠️ **Reactive split** — Same nested module problem. Would need a top-level module (e.g. `eventbus/`) which changes import paths. The `samber/ro` dep is lightweight and acceptable for an event sourcing library.

### Deferred — multi-day effort

- ⏳ **OTel shim** — Modules use OTel types too deeply (`trace.Tracer`, `trace.Span`, `attribute.KeyValue`, `trace.SpanKind`, etc.) for a thin re-export. A proper interface-based shim requires ~10 new interfaces, migrating 15+ files across 4 modules. Estimated 2-3 days. The current `otel/` module already centralizes helper functions (`NewTracer`, `StartSpan`, `RecordError`, attribute helpers), but the 4 modules still import raw `go.opentelemetry.io/otel/*` types for their implementations.

---

| #   | Task                                                                          | Module      | Impact | Effort | Depends |
| --- | ----------------------------------------------------------------------------- | ----------- | ------ | ------ | ------- |
| 1   | Create `event/eventtest/go.mod` with own deps                                 | event       | HIGH   | 10min  | —       |
| 2   | Remove `command/v2` from `event/go.mod` requires                              | event       | HIGH   | 5min   | #1      |
| 3   | Remove `query/v2` from `event/go.mod` requires                                | event       | HIGH   | 5min   | #1      |
| 4   | Remove `memory/v2` from `event/go.mod` requires                               | event       | HIGH   | 5min   | #1      |
| 5   | Remove `schema/v2` from `event/go.mod` requires                               | event       | HIGH   | 5min   | #1      |
| 6   | Remove `snapshot/v2` from `event/go.mod` requires                             | event       | HIGH   | 5min   | #1      |
| 7   | Remove `ginkgo/v2` from `event/go.mod` requires                               | event       | MED    | 5min   | #1      |
| 8   | Remove `gomega` from `event/go.mod` requires                                  | event       | MED    | 5min   | #1      |
| 9   | Remove `rapid` from `event/go.mod` requires                                   | event       | MED    | 5min   | #1      |
| 10  | Add `eventtest` replace directive to `event/go.mod`                           | event       | HIGH   | 3min   | #1      |
| 11  | Update `go.work` to include `event/eventtest`                                 | workspace   | HIGH   | 3min   | #1      |
| 12  | Run `go mod tidy` in `event/` and `event/eventtest/`                          | event       | HIGH   | 5min   | #1-11   |
| 13  | Fix import paths in `event/eventtest/*.go` (add module prefix)                | eventtest   | HIGH   | 10min  | #1      |
| 14  | Fix import paths in `event/*_test.go` for eventtest types                     | event       | HIGH   | 8min   | #1      |
| 15  | Verify `GOWORK=off go build ./...` in `event/`                                | event       | HIGH   | 3min   | #12     |
| 16  | Verify `GOWORK=off go test ./...` in `event/`                                 | event       | HIGH   | 5min   | #15     |
| 17  | Verify `GOWORK=off go test ./...` in `event/eventtest/`                       | eventtest   | HIGH   | 5min   | #15     |
| 18  | Check other modules that import `eventtest` — fix breakage                    | cross-mod   | HIGH   | 10min  | #13     |
| 19  | Run full workspace `go build ./...`                                           | workspace   | HIGH   | 5min   | #18     |
| 20  | Run full workspace `go test ./...`                                            | workspace   | HIGH   | 10min  | #19     |
| 21  | Update `check-module-layers.sh` if eventtest is new module                    | scripts     | MED    | 5min   | #11     |
| 22  | Run `nix run .#build` to verify Nix build                                     | infra       | MED    | 5min   | #20     |
| 23  | Run `nix run .#test` to verify Nix tests                                      | infra       | MED    | 10min  | #22     |
| 24  | Run `nix run .#lint` to verify linting passes                                 | infra       | MED    | 5min   | #23     |
| 25  | Create `event/reactive/` sub-directory                                        | event       | MED    | 2min   | —       |
| 26  | Create `event/reactive/go.mod` (deps: event + samber/ro)                      | event       | MED    | 5min   | #25     |
| 27  | Create `event/reactive/go.sum`                                                | event       | MED    | 2min   | #26     |
| 28  | Move `event/reactive.go` → `event/reactive/reactive.go`                       | event       | MED    | 5min   | #25     |
| 29  | Move `event/bus.go` → `event/reactive/bus.go`                                 | event       | MED    | 5min   | #28     |
| 30  | Change package decl in reactive files from `event` to `reactive`              | event       | MED    | 5min   | #28-29  |
| 31  | Fix internal references in reactive files (Event → event.Event, etc.)         | event       | MED    | 10min  | #30     |
| 32  | Create re-export types in reactive: `type EventBus = ro.Subject[event.Event]` | event       | MED    | 8min   | #31     |
| 33  | Move `event/reactive_test.go` → `event/reactive/` and fix imports             | event       | MED    | 8min   | #31     |
| 34  | Update `go.work` to include `event/reactive`                                  | workspace   | MED    | 2min   | #26     |
| 35  | Add `event/reactive` replace directive to consuming modules' go.mod           | cross-mod   | MED    | 5min   | #26     |
| 36  | Update imports in `example/user/server.go` (uses EventBus)                    | example     | MED    | 5min   | #31     |
| 37  | Update imports in `integration/otel_integration_test.go`                      | integration | MED    | 3min   | #31     |
| 38  | Update imports in `projection/runner_replay_test.go`                          | projection  | MED    | 3min   | #31     |
| 39  | Remove `samber/ro` from `event/go.mod` requires                               | event       | HIGH   | 3min   | #28     |
| 40  | Run `go mod tidy` in `event/reactive/`                                        | event       | MED    | 3min   | #39     |
| 41  | Verify `GOWORK=off go build ./...` in `event/reactive/`                       | event       | MED    | 3min   | #40     |
| 42  | Verify `GOWORK=off go test ./...` in `event/reactive/`                        | event       | MED    | 5min   | #41     |
| 43  | Verify `GOWORK=off go build ./...` in `event/` (without ro)                   | event       | HIGH   | 3min   | #39     |
| 44  | Define OTel interface shims in `otel/` (Tracer, Span, Meter)                  | otel        | MED    | 10min  | —       |
| 45  | Replace direct OTel imports in `decider/` with otel/ shims                    | decider     | MED    | 8min   | #44     |
| 46  | Replace direct OTel imports in `middleware/` with otel/ shims                 | middleware  | MED    | 8min   | #44     |
| 47  | Replace direct OTel imports in `projection/` with otel/ shims                 | projection  | MED    | 8min   | #44     |
| 48  | Replace direct OTel imports in `storage/` with otel/ shims                    | storage     | MED    | 10min  | #44     |
| 49  | Add no-op implementations in `otel/` for zero-config usage                    | otel        | MED    | 8min   | #44     |
| 50  | Remove `go.opentelemetry.io/otel/*` from decider/go.mod                       | decider     | MED    | 3min   | #45     |
| 51  | Remove `go.opentelemetry.io/otel/*` from middleware/go.mod                    | middleware  | MED    | 3min   | #46     |
| 52  | Remove `go.opentelemetry.io/otel/*` from projection/go.mod                    | projection  | MED    | 3min   | #47     |
| 53  | Remove `go.opentelemetry.io/otel/*` from storage/go.mod                       | storage     | MED    | 3min   | #48     |
| 54  | Verify full workspace build after OTel shim migration                         | workspace   | HIGH   | 5min   | #50-53  |
| 55  | Verify full workspace tests after OTel shim migration                         | workspace   | HIGH   | 10min  | #54     |
| 56  | Verify full workspace lint after OTel shim migration                          | workspace   | MED    | 5min   | #55     |
| 57  | Add dep-count check to `check-module-layers.sh`                               | scripts     | MED    | 10min  | —       |
| 58  | Define per-module dep budgets (leaf: ≤3, mid: ≤5, outer: ≤8)                  | scripts     | MED    | 5min   | #57     |
| 59  | Add `nix run .#check-layers` to flake.nix                                     | infra       | LOW    | 8min   | #57     |
| 60  | Wire layer check into CI pipeline (if CI exists)                              | infra       | LOW    | 5min   | #59     |
| 61  | Update `AGENTS.md` with new module structure                                  | docs        | MED    | 5min   | #19     |
| 62  | Update module README for `event/` (new import paths)                          | docs        | MED    | 5min   | #19     |
| 63  | Create module README for `event/reactive/`                                    | docs        | LOW    | 5min   | #31     |
| 64  | Create module README for `event/eventtest/` (if now separate)                 | docs        | LOW    | 5min   | #13     |
| 65  | Update `docs/planning/CATALOG_ARCHITECTURE.md` if referenced                  | docs        | LOW    | 5min   | —       |
| 66  | Add ID backing type example to `id/` README                                   | docs        | LOW    | 8min   | —       |
| 67  | Final full build + test + lint verification                                   | all         | HIGH   | 10min  | ALL     |
| 68  | Run `nix fmt` on all changed files                                            | all         | MED    | 3min   | #67     |

---

## Execution Phases

### Phase 1: go.mod Audit (tasks 1–24) — ~2.5h total

The single highest-impact change. Makes `event/` a true leaf with only 2 internal deps + 2 external deps.

### Phase 2: Reactive Split (tasks 25–43) — ~1.5h total

Removes `samber/ro` from core `event/`. Only consumers who use EventBus pay for reactive streams.

### Phase 3: OTel Shim (tasks 44–56) — ~2h total

Makes OTel fully optional. Modules depend on `otel/` interfaces, not the raw SDK.

### Phase 4: Dep Budget CI + Docs (tasks 57–68) — ~1.5h total

Prevents regression and documents the new structure.

---

## D2 Execution Graph

```d2
direction: right

title: Packaging Hygiene — Execution Flow

phase1: {
  label: Phase 1: go.mod Audit
  shape: rectangle
  style.fill: "#0e4429"
  style.stroke: "#238636"

  create_eventtest_mod: "Create eventtest/go.mod"
  remove_siblings: "Remove 5 test-only siblings from event/go.mod"
  remove_test_frameworks: "Remove ginkgo/gomega/rapid from event/go.mod"
  fix_imports: "Fix import paths in eventtest/ + test files"
  verify_event: "GOWORK=off build+test event/"
  verify_workspace_1: "Full workspace build+test"
  verify_nix_1: "Nix build+test+lint"

  create_eventtest_mod -> remove_siblings -> remove_test_frameworks
  remove_test_frameworks -> fix_imports -> verify_event -> verify_workspace_1 -> verify_nix_1
}

phase2: {
  label: Phase 2: Reactive Split
  shape: rectangle
  style.fill: "#0e3a5e"
  style.stroke: "#1f6feb"

  create_reactive_mod: "Create event/reactive/go.mod"
  move_files: "Move reactive.go + bus.go → reactive/"
  fix_package: "Fix package decl + references"
  update_consumers: "Update 3 consumers (example/integration/projection)"
  remove_ro: "Remove samber/ro from event/go.mod"
  verify_reactive: "GOWORK=off build+test reactive/"
  verify_event_no_ro: "Verify event/ builds without ro"

  create_reactive_mod -> move_files -> fix_package -> update_consumers
  update_consumers -> remove_ro -> verify_reactive -> verify_event_no_ro
}

phase3: {
  label: Phase 3: OTel Shim
  shape: rectangle
  style.fill: "#4a2800"
  style.stroke: "#d29922"

  define_shims: "Define Tracer/Span/Meter interfaces in otel/"
  add_noop: "Add no-op implementations"
  migrate_decider: "Replace OTel imports in decider/"
  migrate_middleware: "Replace OTel imports in middleware/"
  migrate_projection: "Replace OTel imports in projection/"
  migrate_storage: "Replace OTel imports in storage/"
  remove_otel_deps: "Remove go.opentelemetry.io from 4 modules"
  verify_otel: "Full workspace build+test+lint"

  define_shims -> add_noop -> {migrate_decider; migrate_middleware; migrate_projection; migrate_storage}
  migrate_decider -> remove_otel_deps
  migrate_middleware -> remove_otel_deps
  migrate_projection -> remove_otel_deps
  migrate_storage -> remove_otel_deps
  remove_otel_deps -> verify_otel
}

phase4: {
  label: Phase 4: Dep Budget CI + Docs
  shape: rectangle
  style.fill: "#3a0845"
  style.stroke: "#8b5cf6"

  add_dep_budget: "Add dep-count check to layer script"
  wire_ci: "Wire into Nix flake"
  update_agents: "Update AGENTS.md + READMEs"
  final_verify: "Final full build+test+lint+fmt"

  add_dep_budget -> wire_ci -> update_agents -> final_verify
}

phase1 -> phase2 -> phase3 -> phase4
```

---

## Expected Outcome

### Before (current)

```
event/go.mod: 7 internal siblings + go-error-family + samber/ro + ginkgo + gomega + rapid
              = consumer sees "event depends on everything"
```

### After

```
event/go.mod:           id/v2 + codec/v2 + go-error-family          (3 deps — true leaf)
event/reactive/go.mod:  event/v2 + samber/ro                        (2 deps — opt-in streams)
event/eventtest/go.mod: event/v2 + id/v2 + snapshot/v2 + test libs  (isolated test helpers)
decider/go.mod:         event/v2 + snapshot/v2 + otel/v2            (no direct OTel SDK)
middleware/go.mod:       event/v2 + command/v2 + query/v2 + otel/v2  (no direct OTel SDK)
projection/go.mod:      event/v2 + otel/v2                          (no direct OTel SDK)
storage/go.mod:         event/v2 + otel/v2                          (no direct OTel SDK)
```

### Impact on consumer perception

- **"event depends on 7 siblings"** → event depends on **2** production siblings
- **"not lite"** → event is now genuinely a leaf module
- **OTel mandatory** → OTel is now opt-in through interfaces
- **Test deps pollute graph** → test deps isolated in eventtest/
