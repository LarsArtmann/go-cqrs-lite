# Review: DiscordSync Read/Write Census and Metaengine Feedback

**Source feedback:** [`new/2026-08-08_DiscordSync_read-write-census-and-metaengine-feedback.md`](../new/2026-08-08_DiscordSync_read-write-census-and-metaengine-feedback.md)
**Date reviewed:** 2026-08-13
**Outcome:** Maintainer decisions (from appendix A3) confirmed and executed. Three docs-only wins completed. Strategic items remain deferred.

---

## Write-Side Sink Gaps — REJECTED (confirmed)

The 5 proposed sink methods (`IncrementClamped`, `MultiIncrement`, `UpdateExpr`, `QueryRow`, `InsertSelect`) are not being added. go-cqrs-lite is not an ORM. The `Tx()` escape hatch already exists and is documented for irreducible cases. The `Increment` non-clamping philosophy remains the correct default.

**Status:** No action needed — decision confirmed.

---

## DX/Documentation Wins — COMPLETED

### 1. `WithoutViewAutoMigrate` — ✅ Documented

The option exists at `storage/view/options.go:50-54` with a clear doc comment. Additionally, a new `storage/view/README.md` has been created documenting `WithoutViewAutoMigrate` and all other view store APIs, with `AutoMapper` as the recommended default path.

### 2. `AutoMapper` as documented default — ✅ Completed

`AutoMapper` was already well-documented in `storage/view/auto.go` (lines 18-53). The new `storage/view/README.md` positions AutoMapper as the recommended default and manual ViewMapper as the escape hatch, with a complete quick-start example.

### 3. `Increment` non-clamping philosophy — ✅ Already documented

The rationale is in `storage/relational/sink.go:86-89`: "a counter going below zero signals inconsistent events (more deletes than creates), and silent clamping would hide that data-loss bug."

---

## ADTStreamLog Inconsistency — ✅ ALREADY FIXED

The feedback (appendix A2.3) reported `ADTStreamLog` was defined but not in `AllADTs()`. This has been fixed — `ADTStreamLog` is now included in `AllADTs()` at `metaengine/enum_validation.go:12`, and `ADTStreamLog.Valid()` returns `true`.

---

## Metaengine Strategic Items — DEFERRED

### DuckDB Real Aggregation Pushdown — APPROVED, NOT YET IMPLEMENTED

Implementing `AggregateReader` on the DuckDB engine so `CounterGet`, GROUP BY, SUM, AVG push down to columnar SQL. This is the highest-leverage metaengine work but requires significant effort. Remains a strategic priority.

### Cross-Projection JOIN — DEFERRED TO ADR

Metaengine stays single-collection. The tension between "metaengine as projection materializer" vs "metaengine as relational query optimizer" needs its own design work.

---

## Summary

| Item | Status | Action |
| --- | --- | --- |
| 5 sink API gaps | Rejected | `Tx()` escape hatch is the intended design |
| WithoutViewAutoMigrate docs | ✅ Done | Doc comment + README |
| AutoMapper as default | ✅ Done | README with quick-start example |
| Increment non-clamping philosophy | ✅ Done | Already documented in sink.go |
| ADTStreamLog inconsistency | ✅ Fixed | Already in AllADTs() |
| DuckDB aggregation pushdown | Strategic | Approved, high effort, future work |
| Cross-projection JOIN | Strategic | Deferred to separate ADR |
| `sink.Tx()` code-smell doc | Rejected | Existing `Tx()` doc is sufficient |
