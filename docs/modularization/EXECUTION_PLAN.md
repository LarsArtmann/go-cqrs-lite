# Modularization Execution Plan

> **Date:** 2026-05-29 | **Based on:** PROPOSAL.md

## Tier Assignment

| Tier | Focus | Tasks |
|---|---|---|
| 1 — Core | High leverage, breaks the most coupling | T1, T2 |
| 2 — Untangle | Structural improvements | T3, T4 |
| 3 — Polish | Long-term health | T5, T6 |

---

## T1: Move `saga_helpers.go` out of `testhelpers` (Tier 1)

**Problem:** `testhelpers/saga_helpers.go` is production code that imports `saga`, causing transitive pollution for 7 modules.

**Steps:**
1. Create `saga/saga_test_helpers.go` (in saga's own test helpers or as a test-internal file)
2. Move `NewSagaState` and `SaveSagaState` there
3. Delete `testhelpers/saga_helpers.go`
4. Remove `saga` from `testhelpers/go.mod` direct requires
5. Update `saga/` and `storage/` tests to use new import path
6. Run `go mod tidy` in `testhelpers`, `saga`, `storage`
7. Verify: `go work sync` + full test suite

**Verification:**
```bash
cd testhelpers && GOWORK=off go mod tidy && GOWORK=off go build ./...
cd saga && GOWORK=off go test ./... -count=1
cd storage && GOWORK=off go test ./... -count=1
go test ./... -count=1  # full suite
```

**Rollback:** `git revert HEAD` — single commit

---

## T2: Normalize internal module versions (Tier 1)

**Problem:** `testhelpers` references `saga v1.0.0` while others use `v1.6.0`. After T1, this may resolve partially.

**Steps:**
1. Run `go mod tidy` in every module
2. Verify all internal module references use consistent versions
3. Check `go work sync` output

**Verification:**
```bash
for mod in $(find . -name go.mod -not -path './vendor/*' | sort); do
  echo "=== $(dirname $mod) ==="
  grep 'larsartmann/go-cqrs-lite/' "$mod" | grep -v replace | grep -v '// indirect'
done
```

**Rollback:** `git revert HEAD`

---

## T3: Split `core/event` into sub-packages (Tier 2)

**Problem:** 90+ exported symbols across 12 concern clusters in one package.

**Approach:** Gradual extraction with backward compatibility via type aliases.

### T3.1: Extract `core/event/store` sub-package

**Steps:**
1. Create `core/event/store/` directory
2. Move `store.go`, `aggregate_ref.go`, `stream.go` into `core/event/store/`
3. Package becomes `store`, types keep their names
4. Add `core/event/store_alias.go` with type aliases: `type Store = store.Store` etc.
5. Update all internal consumers
6. Run tests

### T3.2: Extract `core/event/bus` sub-package

**Steps:**
1. Create `core/event/bus/` directory
2. Move `bus.go` into `core/event/bus/`
3. Add aliases in `core/event/bus_alias.go`
4. Update consumers
5. Run tests

### T3.3: Extract `core/event/snapshot` sub-package

**Steps:**
1. Create `core/event/snapshot/` directory
2. Move `snapshot.go`, `snapshot_strategy.go`, `snapshot_helper.go`
3. Add aliases
4. Update consumers
5. Run tests

### T3.4: Extract `core/event/outbox` sub-package

**Steps:**
1. Create `core/event/outbox/` directory
2. Move `outbox.go`, `outbox_publisher.go`, `publish_helper.go`
3. `outbox_publisher.go` imports `Publisher` from `bus` — add dep on `core/event/bus`
4. Add aliases
5. Update consumers
6. Run tests

### T3.5: Extract `core/event/projection` sub-package

**Steps:**
1. Create `core/event/projection/` directory
2. Move `projection.go`, `checkpoint.go`
3. Add aliases
4. Update consumers
5. Run tests

### T3.6: Extract `core/event/upcaster` sub-package

**Steps:**
1. Create `core/event/upcaster/` directory
2. Move `upcaster.go`, `upcaster_registry.go`, `versioned_store.go`
3. `versioned_store.go` imports `Store` from `store` — add dep
4. Add aliases
5. Update consumers
6. Run tests

**Verification for each sub-step:**
```bash
cd core && GOWORK=off go build ./...
cd core && GOWORK=off go test ./... -count=1
go test ./... -count=1  # full suite
```

**Rollback:** Each sub-step is one commit. `git revert` undoes one extraction.

---

## T4: Extract `core/event/errors` sub-package (Tier 2)

**Problem:** `errors.go` has ~30 error family re-exports that inflate the event package API surface.

**Steps:**
1. Create `core/event/errors/` directory
2. Move error taxonomy to `core/event/errors/`
3. Re-export from `core/event` via aliases for backward compat
4. Update consumers
5. Run tests

**Note:** This is optional. The error taxonomy is heavily used across the codebase and re-exports may be more confusing than helpful. Only do if the type aliases are clean.

---

## T5: Update documentation (Tier 3)

**Steps:**
1. Update `AGENTS.md` — reflect new package structure
2. Update `docs/modularization/MODULE_ASSESSMENT.md` — correct the outdated assessment
3. Update any code examples in README or docs

---

## T6: Update `flake.nix` if needed (Tier 3)

**Steps:**
1. Verify `nix run .#build` still works after all changes
2. Verify `nix run .#test` still works
3. Update any hardcoded paths if package structure changed

---

## Execution Order

```
T1 (saga leak fix)
 └→ T2 (version normalize)
     └→ T3.1 (store extract)
         └→ T3.2 (bus extract)
             └→ T3.3 (snapshot extract)
                 └→ T3.4 (outbox extract)
                     └→ T3.5 (projection extract)
                         └→ T3.6 (upcaster extract)
                             └→ T4 (errors extract — optional)
                                 └→ T5 (docs update)
                                     └→ T6 (flake verify)
```

Each step produces a commit. Each commit leaves the project buildable and tested.
