# v5 Consumer API Implementation — Superb Execution Plan

> **Date:** 2026-08-09 09:07
> **Design doc:** [`docs/design/v5-consumer-api.md`](../design/v5-consumer-api.md)
> **Session 1 status:** [`docs/status/2026-08-09_07-56_v5-unification-execution-session-1.md`](../status/2026-08-09_07-56_v5-unification-execution-session-1.md)
> **Decision:** [ADR-0123](../adr/0123-v5-unification-single-composition-root.md)

---

## Context

Session 1 built the auto-projection MVP: `system.View[V,K](name).From(events...)`
works end-to-end (command dispatch → event store → projection host → typed
query). Two tests pass. The feature is real.

But the API has three problems identified during this conversation:

1. **`Projections []any`** — type-erased slice. Strings, nils, typos compile fine.
   This was the original question that started the redesign.
2. **One `View` constructor for everything** — doesn't tell the engine the access
   pattern (point lookup vs filtered scan vs counter). The engine can't optimize.
3. **Fold logic mixed with read configuration** — `v.Status = "done"` is a
   materialization rule sitting on a read-pattern declaration.

The design doc at `docs/design/v5-consumer-api.md` specifies the target:
**three-way split (Commands + Evolutions + Queries)** with sealed interfaces,
typed graph queries, and runtime read functions.

This plan describes the path from the current working code to the target API.
**Every phase is independently shippable. Nothing is rewritten — only evolved.**

---

## Pareto Analysis

### The 1% that delivers 51%

**Seal the type system.** Replace `Projections []any` with
`Queries []ProjectionDeclaration` (sealed interface). This is THE fix the user
asked about. It's the foundation for everything else. Without sealed types,
no subsequent task is type-safe. ~45min of mechanical changes, zero behavior
modification.

### The 4% that delivers 64%

**Query constructors + Runtime API.** Three constructors (`Lookup`, `QuerySet`,
`Count`) make the API honest about access patterns — the engine learns what
storage shape to build. Runtime functions (`system.Get[R]`, `system.Find[R]`)
let the developer actually read data. ~135min.

### The 20% that delivers 80%

**Evolutions + Example migration.** Evolutions separate materialization rules
(how state emerges from events) from access patterns (how you read it). The
example migration proves the API works on real consumer code. ~135min.

### The remaining 20%

**Domain/Deployment restructure, Graph types, Docs.** Structural cleanup
(move infrastructure off Domain), edge-case support (graph traversal), and
documentation. ~180min.

---

## Comprehensive Plan (30-100min tasks)

Sorted by importance/impact/effort/customer-value.

| ID | Task | Customer Value | Impact | Effort | Risk | Deps |
|---|---|---|---|---|---|---|
| **P1** | **Seal `[]any` → sealed `ProjectionDeclaration` interface** | Eliminates runtime panics from typos | Critical | 45min | LOW | — |
| **P2** | **Rename View→Lookup, add QuerySet + Count constructors** | API honest about access patterns; engine can optimize | High | 90min | MEDIUM | P1 |
| **P3** | **Runtime API: `system.Get[R]` + `system.Find[R]`** | Developer can actually read data | High | 45min | LOW | P1 |
| **P4** | **Evolutions: three-way split (fold on Evolution, not Query)** | Clean separation of materialization vs access | Medium | 90min | MEDIUM | P2 |
| **P6** | **Migrate metaengine-quickstart example** | Proves API works on real consumer code | High (validation) | 45min | LOW | P1-P3 |
| **P5** | **Domain/Deployment restructure** | Clean Domain/Deployment boundary | Medium | 60min | MEDIUM | P2,P4 |
| **P7** | **Graph query types (Traversal, Path)** | Graph-native query support | Low (5% cases) | 90min | MEDIUM | P2 |
| **P8** | **ADR + design doc implementation notes** | Documentation for future contributors | Low | 30min | NONE | P1-P6 |
| **P9** | **Full verify gate (build + test + lint + race)** | Confidence the changes don't break anything | Critical (gate) | 30min | NONE | All |

**Total: ~555min (~9.25h)**

### What each task touches

| Task | Files modified | Files created | Tests updated |
|---|---|---|---|
| P1 | `projection_builder.go`, `config_types.go`, `constructor.go` | — | `system_auto_projection_test.go` |
| P2 | `projection_builder.go` | — | `system_auto_projection_test.go` |
| P3 | `runtime.go` (new) | `runtime.go` | `runtime_test.go` (new) |
| P4 | `evolutions.go` (new), `projection_builder.go`, `constructor.go` | `evolutions.go` | `evolutions_test.go` (new) |
| P5 | `config_types.go`, `constructor.go` | — | All system tests |
| P6 | — | `example/metaengine-quickstart/main.go` | — |
| P7 | `projection_builder.go`, `runtime.go` | `graph_queries.go` (new) | `graph_queries_test.go` (new) |
| P8 | — | `docs/adr/0124-*.md` | — |
| P9 | — | — | — |

---

## Fine-Grained Plan (12min tasks)

### P1: Seal types (4 tasks × 12min = 48min)

| Sub-ID | Task | What changes |
|---|---|---|
| P1a | Add `ProjectionDeclaration` sealed interface + marker on `ProjectionSpec` | `projection_builder.go`: add interface with `isProjectionDeclaration()`, add marker method on `ProjectionSpec` |
| P1b | Add `rawQuerySpec` wrapper + `system.Query()` constructor for raw QueryDecl passthrough | `projection_builder.go`: new unexported type + exported generic constructor |
| P1c | Change `DomainConfig.Projections` from `[]any` to `[]ProjectionDeclaration`; replace `hasProjectionSpec()` with always-iterate in `buildProjections` | `config_types.go`, `projection_builder.go`, `constructor.go` |
| P1d | Update `system_auto_projection_test.go` to use `[]ProjectionDeclaration`; build + verify | Test file + `go build ./system/...` |

### P2: Query constructors (8 tasks × 12min = 96min)

| Sub-ID | Task | What changes |
|---|---|---|
| P2a | Rename `View[R,K]` → `Lookup[R]`, drop K param (always string/StreamID); rename `viewBuilder` → `lookupBuilder` | `projection_builder.go` |
| P2b | Change `.From(samples...)` to `.On(eventType, sample)` chain + `.Done()` returns `ProjectionDeclaration` | `projection_builder.go` |
| P2c | Add `querySetBuilder[R]` struct + `QuerySet[R]()` constructor | `projection_builder.go` |
| P2d | Add `.Filterable(fields...)` + `.Sortable(field, order)` + `.Done()` on QuerySet | `projection_builder.go` |
| P2e | Add `countBuilder` + `Count()` constructor + `.On(eventType, delta, key)` + `.Done()` | `projection_builder.go` |
| P2f | Wire Lookup/QuerySet/Count into `buildProjections` via closed type-switch | `projection_builder.go` |
| P2g | Update all existing tests to new API names | `system_auto_projection_test.go` |
| P2h | Add tests for QuerySet (filterable) + Count; build + verify | New test cases |

### P3: Runtime API (4 tasks × 12min = 48min)

| Sub-ID | Task | What changes |
|---|---|---|
| P3a | Add `system.Get[R](ctx, sys, name, key)` — point lookup wrapping `metaengine.ExecuteTyped` | New file `runtime.go` |
| P3b | Add `system.Find[R](ctx, sys, name, opts...)` + `Where()`, `Limit()`, `OrderBy()` option types | `runtime.go` |
| P3c | Add cursor pagination: `After(cursor)` option + `PaginatedResult[R]` return type | `runtime.go` |
| P3d | Tests for Get + Find + pagination; build + verify | New file `runtime_test.go` |

### P4: Evolutions (8 tasks × 12min = 96min)

| Sub-ID | Task | What changes |
|---|---|---|
| P4a | Add `EvolutionSpec` sealed interface + `evolutionBuilder[R]` struct + `Evolve[R]()` constructor | New file `evolutions.go` |
| P4b | Add `.On[E](eventType, sample, fold...)` generic method with type inference | `evolutions.go` |
| P4c | Add `.Done()` → `EvolutionSpec`; wire into Domain as `Evolutions []EvolutionSpec` | `evolutions.go`, `config_types.go` |
| P4d | Add counter fold support: `.On(eventType, +1, "key")` variant | `evolutions.go` |
| P4e | Wire Evolutions into constructor: if present, queries reference Evolution folds by result type | `constructor.go`, `projection_builder.go` |
| P4f | Connect commands to Evolution folds: command handler State type → find matching Evolution → use its fold for state loading | `register.go`, `constructor.go` |
| P4g | Add `system.Internal()` option for non-queryable state-only Evolutions | `evolutions.go` |
| P4h | Tests for Evolutions + command state loading; build + verify | New file `evolutions_test.go` |

### P5: Domain/Deployment restructure (5 tasks × 12min = 60min)

| Sub-ID | Task | What changes |
|---|---|---|
| P5a | Create `Domain` struct: `Commands []CommandSpec`, `Evolutions []EvolutionSpec`, `Queries []QuerySpec`, `Middleware` | New file `domain.go` |
| P5b | Create `Deployment` struct: `Engines`, `Topology`, `Bus`, `Durability`, `ProjectionHost`, `ManifestPath` | New file `deployment.go` |
| P5c | Move infrastructure fields off `DomainConfig` → `Deployment` (ProjectionHostOptions, CheckpointStore, ShutdownDependencies) | `config_types.go` |
| P5d | Update `system.New()` to accept `Domain` + `Deployment`; keep `DomainConfig`+`DeploymentConfig` as deprecated aliases | `constructor.go` |
| P5e | Tests; build + verify | All system tests |

### P6: Migrate metaengine-quickstart (4 tasks × 12min = 48min)

| Sub-ID | Task | What changes |
|---|---|---|
| P6a | Define events + result types in example | `example/metaengine-quickstart/main.go` |
| P6b | Rewrite commands using `system.On()` | `main.go` |
| P6c | Rewrite queries using `system.Lookup` / `system.QuerySet` / `system.Count` | `main.go` |
| P6d | Verify example compiles + runs | Manual run |

### P7: Graph query types (8 tasks × 12min = 96min)

| Sub-ID | Task | What changes |
|---|---|---|
| P7a | Add `system.Edge("From", "To")` option on `Evolve` for edge type detection | `evolutions.go` |
| P7b | Add `traversalSpec[R]` + `Traversal[R]()` constructor | New file `graph_queries.go` |
| P7c | Add `.From(nodeColl).Via(edgeColl).Depth(n).Done()` builder methods | `graph_queries.go` |
| P7d | Add `pathSpec[R]` + `Path[R]()` constructor | `graph_queries.go` |
| P7e | Wire Traversal/Path into `buildProjections` via type-switch | `projection_builder.go` |
| P7f | Add `system.Traverse[R](ctx, sys, name, startNode)` runtime function | `runtime.go` |
| P7g | Add `system.FindPath[R](ctx, sys, name, from, to)` runtime function | `runtime.go` |
| P7h | Tests for graph queries; build + verify | New file `graph_queries_test.go` |

### P8: Docs (3 tasks × 12min = 36min)

| Sub-ID | Task | What changes |
|---|---|---|
| P8a | Write ADR-0124: sealed type system + three-way split | New file `docs/adr/0124-*.md` |
| P8b | Add implementation notes to design doc (what was built, how it connects to metaengine) | `docs/design/v5-consumer-api.md` |
| P8c | Update DOMAIN_LANGUAGE.md with new terms (Evolution, Lookup, QuerySet, Traversal) | `docs/DOMAIN_LANGUAGE.md` |

### P9: Verify (3 tasks × 12min = 36min)

| Sub-ID | Task | What changes |
|---|---|---|
| P9a | Build all affected modules (`go build -tags "goexperiment.jsonv2" ./system/... ./metaengine/...`) | — |
| P9b | Run tests with -race (`go test -tags "goexperiment.jsonv2" -race -count=1 ./system/...`) | — |
| P9c | Run lint (`gofumpt -w`, `goimports -w`); fix any issues | — |

---

## Mermaid Execution Graph

```mermaid
graph TD
    %% ═══ Phase 1: Foundation (1% → 51%) ═══
    P1[P1: Seal []any → sealed interfaces<br/>45min — CRITICAL]
    P1 --> P2
    P1 --> P3

    %% ═══ Phase 2: API Surface (4% → 64%) ═══
    P2[P2: Query constructors<br/>Lookup + QuerySet + Count<br/>90min]
    P3[P3: Runtime API<br/>Get + Find<br/>45min]
    P2 --> P4
    P2 --> P6
    P3 --> P6

    %% ═══ Phase 3: Separation (20% → 80%) ═══
    P4[P4: Evolutions<br/>three-way split<br/>90min]
    P4 --> P5
    P4 --> P6

    P6[P6: Migrate example<br/>45min — VALIDATION]

    %% ═══ Phase 4: Remaining 20% ═══
    P5[P5: Domain/Deployment<br/>restructure<br/>60min]
    P7[P7: Graph query types<br/>Traversal + Path<br/>90min]
    P8[P8: Docs + ADR<br/>30min]

    P2 --> P7

    %% ═══ Verify gate (runs after each phase) ═══
    P9[P9: Full verify gate<br/>build + test + lint + race<br/>30min]

    P1 --> V1[Verify checkpoint]
    P2 --> V2[Verify checkpoint]
    P3 --> V3[Verify checkpoint]
    P4 --> V4[Verify checkpoint]
    P6 --> V6[Verify checkpoint]
    P5 --> V5[Verify checkpoint]
    P7 --> V7[Verify checkpoint]
    P8 --> P9

    %% ═══ Styling ═══
    classDef critical fill:#fadbd8,stroke:#c0392b,stroke-width:3px
    classDef high fill:#fdebd0,stroke:#e67e22,stroke-width:2px
    classDef medium fill:#d5f5e3,stroke:#27ae60,stroke-width:2px
    classDef low fill:#d6eaf8,stroke:#2980b9,stroke-width:1px
    classDef gate fill:#f5b7b1,stroke:#c0392b,stroke-width:3px,stroke-dasharray: 5 5

    class P1 critical
    class P2,P3,P6 high
    class P4,P5 medium
    class P7,P8 low
    class P9,V1,V2,V3,V4,V5,V6,V7 gate
```

---

## Risk Assessment

### What could verschlimmbessern the system

| Risk | Mitigation |
|---|---|
| Renaming `View[R,K]` → `Lookup[R]` breaks the 2 tests from session 1 | Tests are updated in the same task (P2g). Both tests are in `system_auto_projection_test.go` — easy to find. |
| Evolutions add a concept nobody asked for | Evolutions are OPTIONAL. If `Domain.Evolutions` is empty, queries carry their own folds (current behavior). Evolutions are a refactor, not a requirement. |
| Domain/Deployment restructure breaks `system.New()` signature | P5d keeps `DomainConfig`+`DeploymentConfig` as deprecated type aliases. Consumers can migrate gradually. |
| Graph types add complexity for 5% of use cases | P7 is the LOWEST priority task. Can be deferred indefinitely without affecting the core API. |
| Generic methods (`On[E]`) might not compile in all Go versions | Go 1.18+ supports generic methods with additional type params. The project uses Go 1.26.5. This is safe. |

### Safety principles

1. **Every phase is independently shippable.** P1 alone makes the API type-safe.
   P1+P2+P3 makes it honest about access patterns. Each adds value.
2. **No behavior change without a test.** Each task includes a verify step.
3. **Evolve, don't rewrite.** `buildProjections()` stays as the chokepoint — it
   just gets a richer type-switch instead of `hasProjectionSpec()`.
4. **Keep `DomainConfig`/`DeploymentConfig` as aliases.** No consumer breaks
   until they choose to migrate.

---

## What This Plan Explicitly Defers

| Deferred | Why | When |
|---|---|---|
| Engine self-registration (T08-T12 from session 1 plan) | Infrastructure work, not consumer API | After API stabilizes |
| Batch atomicity (T16-T20) | S2 spike proved it's ~6h, not blocking API | After API stabilizes |
| Universal ADT coverage (T21-T25) | Engine internals, not consumer API | Post-v5 |
| v1 tier deletion (stack.Bundle, Materialize, etc.) | Breaking change, needs migration guide | v5 cut |
| cqrs-lint rule updates | Rules detect `stack.*` imports which still exist | After v1 deletion |
| Record consolidation (T13-T15) | Parallel track, not blocking API | Post-v5 |
| Graph `map[string]any` API cleanup inside metaengine/graphadapter | Internal cleanup, consumer API already typed | Post-v5 |

---

## Implementation Notes (for the executing engineer)

### Where fold inference lives

```
Current (session 1):
  View[R,K]("name").From(samples...)
    → buildProjections() → AutoCRUDByNamedEvents[R] → []Fold
    → metaengine.Query[LookupInput[K], R] → Plan()
    → buildEventDecoder() → projectionadapter

Target (this plan):
  Evolve[R]("name").On("event.type", Sample{}, fold...)
    → EvolutionSpec (carries folds + decoder entries)

  Lookup[R]("name").Done()
  QuerySet[R]("name").Filterable(...).Done()
    → QuerySpec (carries access pattern only)

  buildProjections() type-switches on:
    - EvolutionSpec → extract folds + decoder entries
    - lookupSpec → wrap in Query[LookupInput, R] + match Evolution by R
    - querySetSpec → wrap in Query[ScanInput, R] + match Evolution by R
    - countSpec → wrap in Query[VoidInput, map[string]int64]
    - rawQuerySpec → passthrough to Plan()
```

### How command state loading changes

```
Current:
  RegisterDecider[State](sys, "Task", decider)  ← developer writes decider
  RegisterCommand[Cmd, State](sys, name, handler)

Target:
  system.On("task.complete", func(..., state TaskSummary) {...})
    → System finds Evolve[TaskSummary] by type match
    → Auto-builds decider.Decider[TaskSummary] from Evolution fold
    → Auto-registers Repository[TaskSummary] on the event store
    → Command handler receives state loaded via Evolution fold
```

### The type-switch in buildProjections (the ONE function that changes most)

```go
// P1c: sealed interface replaces []any
func buildProjections(decls []ProjectionDeclaration) (
    queryDecls []any,
    eventDecoder eventDecoderFn,
    err error,
) {
    for _, decl := range decls {
        switch d := decl.(type) {
        case lookupSpec:    // P2: point lookup
        case querySetSpec:  // P2: filtered scan
        case countSpec:     // P2: counter
        case traversalSpec: // P7: graph traversal
        case rawQuerySpec:  // P1: passthrough QueryDecl
        default:
            return nil, nil, fmt.Errorf("system: unreachable: unknown ProjectionDeclaration %T", decl)
        }
    }
}
```

This replaces the current `hasProjectionSpec()` probe + `buildProjections()` pair
with a single clean pass over a sealed slice.
