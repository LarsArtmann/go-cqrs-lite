# ADR-0083: Metaengine Planner Rule Pipeline

**Date:** 2026-08-01
**Status:** ACCEPTED

## Context

The metaengine planner (`planner.go`) was a 279-line monolith with 4 inline
planning decisions: schema enforcement, auto-layout, write-amplification
detection, and scale-threshold checking. Each future planner capability
(statistics, materialize-vs-replay, cost matrix, temporal routing) would
require adding more inline logic to an already complex function.

DataFusion's core architectural lesson (documented in
`docs/planning/2026-07-31_datafusion-lessons-for-metaengine.md`) is that a
query planner should decompose its decisions into composable,
independently-testable rules. This makes the planner extensible without
modifying the core planning function.

## Decision

Extract the 4 inline decisions into a `PlanRule` interface with a
`RulePipeline` that applies them sequentially:

```go
type PlanRule interface {
    Name() string
    Apply(result *PlanResult, ctx PlanContext) error
}
```

Rules run AFTER engine assignment — they enrich the PlanResult (diagnostics,
layout plans) but do NOT override engine selection.

### Extracted Rules

| Rule | File | Decision |
|------|------|----------|
| `schemaRule` | `rule_schema.go` | Fold value type ≠ result type → WARN |
| `layoutRule` | `rule_layout.go` | LayoutPlanner + FilterOnField/SortOnField → auto DDL |
| `writeAmpRule` | `rule_writeamp.go` | Event updating >3 projections → WARN |
| `materializeRule` | `materialize.go` | Workload stats → materialize-vs-replay INFO/WARN |

### Bug Found During Extraction

During L1.1 (test coverage verification), we discovered that
`Plan()` line 151 **overwrote** all diagnostics with write-amplification
results via `plan.Diagnostics = checkWriteAmplification(...)`, silently
destroying schema enforcement diagnostics. This was a latent bug masked by
the broken `TestSchemaEnforcement` test (substring mismatch: checked for
`"fold returns"` but the actual message has `"fold for <Event> returns"`).

Fixed to `plan.Diagnostics = append(plan.Diagnostics, checkWriteAmplification(...)...)`.

## Consequences

- **planner.go** reduced from 279 → 226 lines
- New rules are one file implementing one interface — no monolith modification
- Rules can record trace entries (`RuleTraceEntry`) for enriched EXPLAIN
- The 48-file test suite is the safety net; each extraction verified independently
- Future capabilities (cost matrix, temporal routing, new ADTs) are additive rules
