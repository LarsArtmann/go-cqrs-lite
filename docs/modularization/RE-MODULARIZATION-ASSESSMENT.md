# Re-Modularization Assessment — go-cqrs-lite

**Date:** 2026-06-01
**State:** Already fully modularized — 22 library modules, 6 example modules, 2 cmd modules
**go.work:** Active, 30 modules
**Assessment:** Module boundaries are fundamentally sound. No merges or splits required. Boundary violations identified below are fixable within the current structure.

---

## Module Quality Scores

| Module | Cohesion (1-5) | Coupling (1-5) | Independent? | Action |
|--------|:-:|:-:|:-:|--------|
| id | 5 | 5 | ✅ | **Keep** |
| dispatcher | 5 | 5 | ✅ | **Keep** |
| codec | 5 | 5 | ✅ | **Keep** |
| otel | 5 | 5 | ✅ | **Keep** |
| event | 4 | 3 | ✅ | **Keep** — investigate test deps |
| command | 3 | 3 | ✅ | **Reorganize** — remove event re-exports |
| query | 4 | 4 | ✅ | **Keep** |
| schema | 4 | 3 | ✅ | **Keep** |
| snapshot | 5 | 4 | ✅ | **Keep** |
| decider | 5 | 3 | ✅ | **Keep** |
| memory | 5 | 3 | ✅ | **Keep** |
| signing | 4 | 4 | ✅ | **Keep** |
| middleware | 3 | 2 | ⚠️ | **Reorganize** — reduce to generic[H] |
| storage | 4 | 3 | ✅ | **Keep** — unify errors |
| pebble | 4 | 4 | ✅ | **Keep** — unify errors |
| turso | 4 | 3 | ✅ | **Keep** |
| projection | 5 | 3 | ✅ | **Keep** |
| listing | 5 | 4 | ✅ | **Keep** |
| watermill | 5 | 4 | ✅ | **Keep** |
| catalog | 5 | 5 | ✅ | **Keep** |
| integration | 5 | 1 | ✅ | **Keep** (integration tests, high coupling expected) |
| cmd/cqrs-gen | 5 | 5 | ✅ | **Keep** |
| cmd/api-stability | 5 | 5 | ✅ | **Keep** |

---

## Coupling Hotspots

### 1. `command/` → `event/` (Severity: HIGH)

**Evidence:** command/go.mod depends on event, id, codec, dispatcher, snapshot.

- `command/aggregate_ref.go` re-exports `event.AggregateType`, `event.AggregateRef`, `event.ParseAggregateType`, `event.NewAggregateRef`
- `command.Metadata` mirrors `event.Metadata` fields (CorrelationID, CausationID, UserID, RequestID)

**Problem:** command/ depends on event/ not just for domain types but for re-exported symbols. This creates:
- Module boundary violation: command is not a standalone module
- Maintenance drift: adding a metadata field requires updating both packages

**Recommendation:** Remove re-exports. Consumers import `event.AggregateRef` directly. Consider extracting `Metadata` to a shared location or accepting the import.

### 2. `event/` test dependencies (Severity: MEDIUM)

**Evidence:** event/go.mod depends on command, query, memory, schema, snapshot.

These are test-only dependencies (eventtest/ uses types from command, query for testing cross-module flows). But they appear in the production `require` block.

**Problem:** Go doesn't separate test-only requires. `go mod why` confirms these are used only in test files, but the dependency is still listed.

**Recommendation:** Accept this as a Go tooling limitation. No action needed — just document it.

### 3. `middleware/` → `command` + `event` + `query` (Severity: HIGH)

**Evidence:** middleware depends on 7 internal modules: command, event, id, otel, query, codec, snapshot.

**Problem:** Every middleware concern is written 3 times (command/event/query variants). Adding a new domain requires updating every middleware file.

**Recommendation:** Introduce generic `Middleware[H]` per concern. Thin type-alias adapters preserve backward compat. This doesn't change module structure — just internal organization.

### 4. `pebble/` ↔ `storage/sql/` (Severity: MEDIUM)

**Evidence:** Both define semantically identical sentinel errors (ErrAggregateTypeMismatch, ErrVersionMismatch) with different error codes.

**Problem:** Cross-backend error checking is fragile. Consumers checking `errors.Is(err, ...)` must know which backend produced the error.

**Recommendation:** Consolidate errors. pebble/ and storage/sql/ should wrap errors from event/ (e.g., `event.ErrVersionConflict`), not define their own.

---

## DAG Verification

The dependency graph is a clean DAG:

```
L0: id, dispatcher, codec          (0 internal deps)
L1: event, command, query           (depend on L0)
L2: schema, snapshot                (depend on L0-L1)
L3: decider                         (depends on L0-L2 + L4)
L4: memory, signing, otel           (depend on L0-L2)
L5: storage, pebble, projection,    (depend on L0-L4)
    listing, watermill, middleware
L6: integration, catalog,           (depend on everything)
    turso, cmd/*
```

**No circular dependencies detected.** ✅

---

## Replace Directive Audit

Only `query/` has a single replace directive (for event/eventtest test dependency). All other modules use `go.work` for local resolution.

**Status:** Clean. ✅

---

## `internal/` Package Audit

| Module | internal/ packages | Cross-module access? |
|--------|-------------------|---------------------|
| catalog | `internal/cattest`, `internal/caseutil` | ❌ No cross-module imports |
| signing | `internal/testutil` | ❌ No cross-module imports |

**Status:** Clean. ✅

---

## Error Type Placement Analysis

| Error | Location | Used by | Accessible? | Issue |
|-------|----------|---------|:------------|-------|
| ErrVersionConflict | event/ | storage, pebble, decider | ✅ | Correct placement |
| ErrHandlerNotFound | dispatcher/, command/, query/ | cross-module | ⚠️ | Three separate sentinels |
| ErrDispatcherClosed | dispatcher/, command/, query/ | cross-module | ⚠️ | Three separate sentinels |
| ErrAggregateTypeMismatch | pebble/, storage/sql/ | backend-specific | ⚠️ | Duplicate across backends |
| ErrCircuitBreakerOpen | middleware/ | consumers | ⚠️ | Bypasses error taxonomy |
| ErrAggregateNotFound | event/ | all backends | ✅ | Correct placement |

**Key issues:** Sentinel error fragmentation (3× ErrHandlerNotFound, 2× ErrVersionMismatch). These should be consolidated.

---

## Granularity Assessment

| Signal | Value | Assessment |
|--------|-------|------------|
| Total library modules | 22 | Appropriate for the scope |
| Largest module (pkgs) | catalog (9) | Contains 5 sub-packages (asyncapi, d2, eventcatalog, openapi, schema) — reasonable |
| Smallest modules | id, codec, otel, dispatcher (1 pkg each) | Worth keeping — they provide fundamental abstractions |
| Modules always changed together | command + event (in practice) | Not due to coupling, but due to feature development patterns |
| Over-modularized? | No | Each module has a clear, distinct purpose |

---

## Recommendations

### Keep (20 modules)
All existing module boundaries are correct. No merges needed.

### Reorganize (3 modules)

| Module | Issue | Fix |
|--------|-------|-----|
| command | Re-exports event types | Remove `aggregate_ref.go` re-exports; consumers import event/ directly |
| middleware | 3× duplication | Generic `Middleware[H]` per concern (internal change, no module split) |
| pebble + storage/sql | Duplicate error sentinels | Consolidate to event/ errors, backend adds context via wrapping |

### No Splits Needed
No module is large enough or unfocused enough to warrant splitting.

---

## Replace/Workspace Strategy

**Current:** `go.work` for local development, `replace` directives in go.mod for GOWORK=off CI per-module isolation.

**Assessment:** This is the correct strategy per ADR-0003. Both are needed:
- `go.work` for developer convenience
- `replace` for per-module CI (GOWORK=off builds)

**No changes needed.** ✅

---

## Versioning Strategy

**Current:** Per-module semver tags (e.g., `event/v1.7.1`, `dispatcher/v1.7.1`).

**Assessment:** Correct for a library with external consumers. Each module can evolve independently.

---

## Summary

The module structure is **fundamentally sound** — a well-executed multi-module monorepo with clean layering, no circular deps, and appropriate granularity. The issues are **within modules** (duplication, re-exports, error fragmentation), not between them.

**Overall grade: A-**

| What's great | What needs work |
|---|---|
| Clean 7-layer DAG | command/ re-exports event/ types |
| 3 leaf modules with zero deps | middleware/ 3× duplication |
| go.work + replace strategy works | Error sentinel fragmentation |
| Each module independently buildable | FakeStore duplicates MemoryStore |
| No circular dependencies | decider/ uses unclassified errors |
