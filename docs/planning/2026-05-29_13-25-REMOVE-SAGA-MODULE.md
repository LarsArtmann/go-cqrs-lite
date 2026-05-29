# Plan: Remove `saga/` Module — Teach the Pattern, Don't Own the Abstraction

**Date:** 2026-05-29
**Status:** DRAFT
**Impact:** Eliminates a module that doesn't match the library's design philosophy (boundary-less deciders + event-driven projections), removes `storage→saga` coupling, simplifies the module graph.

---

## Why

The saga module is an **imperative command-dispatching state machine** in a library built on **event-driven projections + pure-function deciders**. The right pattern is for consumers to compose saga-style orchestration from existing primitives (projection + command dispatch). The library should teach the pattern via an example, not own the abstraction.

## What We Get

- **7 fewer go.mod files** with saga replace directives (saga, storage, example/saga, example/todo, example/storage, turso, integration)
- **storage/ loses its only non-core/non-otel production dependency** — becomes a true leaf module
- **Module graph simplifies** — no more `storage→saga` edge
- **~1,600 lines removed** (saga/ module + saga store in storage + tests)
- **Cleaner API surface** — no saga exports to maintain

---

## Pareto Breakdown

### 1% → 51% of result

| # | Task | Est |
|---|------|-----|
| 1 | Delete `saga/` module entirely | 5 min |
| 2 | Remove saga from `storage/` (saga_store.go, sql_backend saga methods, dialect saga schema, tests) | 30 min |
| 3 | Remove `example/saga/` (will be replaced) | 5 min |
| 4 | Clean up go.work, go.mod files, flake.nix, cmd/api-stability | 15 min |
| 5 | Build + test passes | 10 min |

### 4% → 64% of result

| # | Task | Est |
|---|------|-----|
| 6 | Write `example/saga-pattern/` showing saga-style orchestration with projection + command dispatch | 45 min |
| 7 | Update AGENTS.md, FEATURES.md, README.md | 15 min |

### 20% → 80% of result

| # | Task | Est |
|---|------|-----|
| 8 | Update docs/status/ references, docs/architecture/README.md | 10 min |
| 9 | Update docs/modularization/* to reflect saga removal | 10 min |
| 10 | Clean up docs/planning/SAGA_DESIGN.md (mark as archived) | 5 min |
| 11 | Update docs/api_surface.txt (regenerate) | 5 min |
| 12 | Remove Turso saga constructors (`turso/`) | 10 min |
| 13 | Update docs/MIGRATION_v1.md | 5 min |

---

## Detailed Task List

### Phase 1: Delete (high impact, low risk)

| # | Task | Files | Est |
|---|------|-------|-----|
| 1 | Delete entire `saga/` directory | `saga/**` | 5 min |
| 2 | Delete `example/saga/` directory | `example/saga/**` | 2 min |
| 3 | Remove `saga` from `go.work` | `go.work` | 1 min |
| 4 | Remove saga from `flake.nix` test targets | `flake.nix` | 2 min |
| 5 | Remove `"saga"` from `cmd/api-stability/main.go` | `cmd/api-stability/main.go` | 1 min |

### Phase 2: Clean `storage/` (medium impact, medium effort)

| # | Task | Files | Est |
|---|------|-------|-----|
| 6 | Delete `storage/saga_store.go` | `storage/saga_store.go` | 1 min |
| 7 | Delete `storage/saga_store_test.go` | `storage/saga_store_test.go` | 1 min |
| 8 | Delete `storage/sqlite_integration_outbox_saga_test.go` | `storage/sqlite_integration_outbox_saga_test.go` | 1 min |
| 9 | Remove saga store from `storage/sql_backend.go` — remove `sagaStore` field, `SagaStore()` method, Turso saga constructors | `storage/sql_backend.go` | 10 min |
| 10 | Remove saga schema from dialect (SagaSchema methods) | `storage/dialect.go` or wherever schema methods live | 10 min |
| 11 | Remove saga-related tests from `storage/sql_backend_test.go` | `storage/sql_backend_test.go` | 5 min |
| 12 | Remove saga import + replace from `storage/go.mod`, run `go mod tidy` | `storage/go.mod` | 2 min |

### Phase 3: Clean `turso/` (low impact, quick)

| # | Task | Files | Est |
|---|------|-------|-----|
| 13 | Remove Turso saga constructors (`NewTursoSagaStore`, etc.) | `turso/*.go` | 5 min |
| 14 | Remove saga from `turso/go.mod`, run tidy | `turso/go.mod` | 2 min |

### Phase 4: Clean other go.mod files

| # | Task | Files | Est |
|---|------|-------|-----|
| 15 | Remove saga replace from `integration/go.mod` | `integration/go.mod` | 2 min |
| 16 | Remove saga replace from `example/todo/go.mod` | `example/todo/go.mod` | 2 min |
| 17 | Remove saga replace from `example/storage/go.mod` | `example/storage/go.mod` | 2 min |
| 18 | Run `go work sync` + verify build | - | 2 min |

### Phase 5: Write `example/saga-pattern/` (the teaching replacement)

| # | Task | Files | Est |
|---|------|-------|-----|
| 19 | Create `example/saga-pattern/` with go.mod | `example/saga-pattern/go.mod` | 5 min |
| 20 | Write `main.go` — order processing saga using projection + command dispatch | `example/saga-pattern/main.go` | 30 min |
| 21 | Add `example/saga-pattern/` to `go.work` | `go.work` | 1 min |
| 22 | Add to `flake.nix` test targets | `flake.nix` | 1 min |

The example should demonstrate:
- A `projection.HandlerRegistry` that tracks saga state (current step, status)
- On each relevant event, decide next step and dispatch a command
- Compensation: on failure events, dispatch undo commands in reverse
- State persisted as a projection (read model)
- No saga module needed — just projection + command dispatch + decider

### Phase 6: Update documentation

| # | Task | Files | Est |
|---|------|-------|-----|
| 23 | Update `AGENTS.md` — remove saga from modules list, test command, monorepo structure, module graph | `AGENTS.md` | 5 min |
| 24 | Update `FEATURES.md` — remove saga section, update module maturity matrix | `FEATURES.md` | 5 min |
| 25 | Update `README.md` — remove saga import examples | `README.md` | 3 min |
| 26 | Update `storage/README.md` — remove SQLSagaStore section, saga dependency | `storage/README.md` | 3 min |
| 27 | Update `docs/architecture/README.md` — remove saga from dependency tiers | `docs/architecture/README.md` | 2 min |
| 28 | Archive `docs/planning/SAGA_DESIGN.md` | `docs/planning/SAGA_DESIGN.md` | 1 min |
| 29 | Update `docs/modularization/PROPOSAL.md` and `EXECUTION_PLAN.md` — note saga removal | `docs/modularization/*` | 3 min |
| 30 | Update `docs/MIGRATION_v1.md` — remove saga import | `docs/MIGRATION_v1.md` | 2 min |
| 31 | Regenerate `docs/api_surface.txt` | `docs/api_surface.txt` | 2 min |

### Phase 7: Verify

| # | Task | Est |
|---|------|-----|
| 32 | Full build: `nix run .#build` | 2 min |
| 33 | Full test: `nix run .#test` | 5 min |
| 34 | Lint: `nix run .#lint` | 2 min |
| 35 | Verify go.work sync, no dangling references | 2 min |

---

## D2 Execution Graph

```
direction: right

delete_saga: "Phase 1: Delete saga/ module" {
  shape: rectangle
  style.fill: "#FFE0E0"
}

clean_storage: "Phase 2: Clean storage/" {
  shape: rectangle
  style.fill: "#FFF3E0"
}

clean_turso: "Phase 3: Clean turso/" {
  shape: rectangle
  style.fill: "#FFF3E0"
}

clean_gomod: "Phase 4: Clean go.mod files" {
  shape: rectangle
  style.fill: "#FFF3E0"
}

write_example: "Phase 5: Write example/saga-pattern/" {
  shape: rectangle
  style.fill: "#E0FFE0"
}

update_docs: "Phase 6: Update documentation" {
  shape: rectangle
  style.fill: "#E0E0FF"
}

verify: "Phase 7: Build + Test + Lint" {
  shape: rectangle
  style.fill: "#E0FFE0"
}

delete_saga -> clean_storage -> clean_turso -> clean_gomod -> write_example -> update_docs -> verify
```

---

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Consumers depend on `saga.Store` | Low — library has no external consumers yet (pre-v1.0) | Breaking change is acceptable |
| `storage.SQLBackend.SagaStore()` has consumers | Low — same reason | Remove from API surface |
| Example doesn't adequately replace the module | Medium | Keep it focused: show the pattern, not a framework |
| Dangling references in docs/status/ files | Low | These are historical snapshots; leave as-is |

---

## What the Example Should Teach

The `example/saga-pattern/` should show that a saga is just:

1. **A projection** that tracks `saga_instances` as a read model (step, status, saga_type)
2. **Event handlers** that, on each step-completion event, decide whether to advance or compensate
3. **Command dispatch** to trigger the next step's action or undo
4. **No special abstraction** — just composition of projection + command + decider

This teaches the right mental model: sagas emerge from the primitives, they aren't a separate thing.
