# Module Boundary Analysis — go-cqrs-lite

> **Date:** 2026-06-10 · **Current State:** Workspace mode with 31 modules · **Verdict:** Already modularized, needs cycle-breaking

## Phase 1: Current State

| State          | Indicators                      | Assessment       |
| -------------- | ------------------------------- | ---------------- |
| Workspace mode | go.work with 31 entries         | Active           |
| Partial split  | 4 bidirectional cycles          | Needs refinement |
| Already split  | Clean layer structure (0→1→2→3) | Good foundation  |

### Module Count: 31

- 22 library modules
- 6 example modules
- 1 integration test module
- 2 cmd tools

### Phase 1.5: Re-modularization Assessment

| Module      | Cohesion | Coupling          | Independence | Action                                              |
| ----------- | -------- | ----------------- | ------------ | --------------------------------------------------- |
| codec       | 5/5      | 0                 | Yes          | **Keep**                                            |
| id          | 5/5      | 0                 | Yes          | **Keep**                                            |
| dispatcher  | 5/5      | 0                 | Yes          | **Keep**                                            |
| catalog     | 5/5      | 0                 | Yes          | **Keep**                                            |
| otel        | 4/5      | 0                 | Yes          | **Keep**                                            |
| event       | 3/5      | 7 deps + 2 cycles | No           | **Reorganize** — remove command, query, memory deps |
| command     | 4/5      | 1 cycle           | No           | **Reorganize** — remove error re-exports            |
| query       | 5/5      | 1 dep             | Yes          | **Keep**                                            |
| decider     | 5/5      | 6 deps            | Yes          | **Keep**                                            |
| schema      | 5/5      | 3 deps            | Yes          | **Keep**                                            |
| snapshot    | 4/5      | 1 cycle           | No           | **Reorganize** — break memory cycle                 |
| memory      | 4/5      | 2 cycles          | No           | **Reorganize** — break cycles                       |
| listing     | 5/5      | 3 deps            | Yes          | **Keep**                                            |
| middleware  | 4/5      | 6 deps            | Yes          | **Reorganize** — extract HTTP                       |
| signing     | 5/5      | 2 deps            | Yes          | **Keep**                                            |
| projection  | 5/5      | 5 deps            | Yes          | **Keep**                                            |
| storage     | 3/5      | 7 deps            | Yes          | **Reorganize** — extract query engine               |
| watermill   | 5/5      | 3 deps            | Yes          | **Keep**                                            |
| pebble      | 5/5      | 3 deps            | Yes          | **Keep**                                            |
| turso       | 5/5      | 4 deps            | Yes          | **Keep**                                            |
| integration | 5/5      | 14 deps           | N/A (tests)  | **Keep**                                            |

### Cycles to Break

| Cycle                                     | Root Cause                       | Fix                                |
| ----------------------------------------- | -------------------------------- | ---------------------------------- |
| event ↔ command                           | CatalogDispatcher lives in event | Move to command                    |
| event ↔ memory                            | eventtest imports memory         | Extract eventtest as module        |
| memory ↔ snapshot                         | Both import each other           | Make snapshot depend only on event |
| 3-node: memory → command → event → memory | Transitive                       | Fix above 3 cycles                 |

### Replace Directive Strategy

| Strategy           | Current              | Assessment                 |
| ------------------ | -------------------- | -------------------------- |
| go.work            | ✅ In use            | Correct for development    |
| replace directives | Also in go.mod files | Retained for GOWORK=off CI |

**Recommendation:** Keep both. go.work for development, replace directives for per-module CI isolation.

### Versioning Strategy

| Strategy                     | Current               |
| ---------------------------- | --------------------- |
| Independent semver with /v2  | All modules at v2.x.x |
| Replace directives in go.mod | Retained for CI       |
| go.work for local dev        | Active                |

**Recommendation:** Continue with independent semver. Already correct.

## Phase 7: Reflection

1. **Structure** — Clean and intuitive. 31 modules with clear naming.
2. **DAG** — 4 cycles need breaking. After fixes, clean DAG.
3. **Independence** — Most modules can be versioned independently. Cycles prevent 4 from doing so.
4. **Robustness** — Replace directives in go.mod ensure per-module CI works even without go.work.
5. **Granularity** — No modules should be merged or split. Current granularity is correct.
6. **Documentation** — AGENTS.md and FEATURES.md reflect current state accurately.

**Net assessment:** The modularization is 90% correct. The 4 cycles are the remaining 10% — breaking them makes the module structure textbook-perfect.
