# Status Report: DuckDB Columnar-Native Hardening & Cleanup

**Date:** 2026-08-02 22:17
**Session Scope:** Fix correctness bugs in the columnar-native storage feature (REAL→DOUBLE, decorative generic, reserved-name collision), add regression tests, write ADR, update docs, run full verify gate.
**Verify Gate:** ✅ GREEN (build + vet + test + race + lint + API stability + doc check — all 80+ modules, 0 issues)

---

## What This Session Did

Resumed from a prior session that implemented the DuckDB columnar-native storage feature (`WithColumnarLayout`, `LayoutPlanApplier`, `BuildColumnarLayoutPlan`). That session left 3 known bugs and several documentation gaps. This session fixed all three bugs, added a 4th test, wrote the ADR, updated docs, and ran the full verify gate to GREEN.

### Changes Made (11 tasks, all completed)

| #   | Category    | File                                                 | Change                                                                                                                   |
| --- | ----------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| 1   | Correctness | `metaengine/layout.go:296`                           | `sqlTypeOf`: split `float32→REAL`, `float64→DOUBLE`. Both previously mapped to REAL (32-bit), truncating Go's `float64`. |
| 2   | Correctness | `metaengine/duckdbengine/layout_planner.go:392`      | `coerceForColumn`: added `DOUBLE`/`FLOAT`/`FLOAT4`/`FLOAT8` cases.                                                       |
| 3   | Correctness | `metaengine/layout.go:249`                           | `BuildColumnarLayoutPlan`: skip fields named `key`/`value` (case-insensitive) to prevent DDL column collision.           |
| 4   | API cleanup | `metaengine/query.go:42`                             | `WithColumnarLayout`: removed decorative `[R any]` generic. `R` was unused.                                              |
| 5   | Test        | `metaengine/duckdbengine/layout_planner_cgo_test.go` | Updated 3 tests to use realistic decimal values (1.99, 0.75, 0.99, 2.49).                                                |
| 6   | Test        | `metaengine/duckdbengine/layout_planner_cgo_test.go` | Added `TestDuckDBEngine_ColumnarDoublePrecision` — Pi roundtrip proves DOUBLE precision.                                 |
| 7   | Docs        | `docs/adr/0092-duckdb-columnar-native-storage.md`    | Full ADR (context, decision, consequences, alternatives).                                                                |
| 8   | Docs        | `AGENTS.md`                                          | Added columnar-native feature comment + updated duckdbengine description.                                                |
| 9   | Docs        | `TODO_LIST.md`                                       | Marked `coerceForColumn` sub-item as resolved.                                                                           |
| 10  | Docs        | `docs/README.md`                                     | Indexed ADR-0092.                                                                                                        |
| 11  | Verify      | Full repo                                            | `nix run .#verify` GREEN across all 80+ modules.                                                                         |

---

## A) FULLY DONE

1. **REAL→DOUBLE fix** — `sqlTypeOf` now correctly maps `float64→DOUBLE` and `float32→REAL`. Verified with `TestDuckDBEngine_ColumnarDoublePrecision` (Pi to 15 digits roundtrips exactly).
2. **Decorative generic removal** — `WithColumnarLayout()` no longer takes a type parameter. The result type is already known from `Query[Q, R]`.
3. **Reserved-name collision fix** — Fields named `key`/`value` are silently skipped in `BuildColumnarLayoutPlan`. Discovered by the precision test crashing with "Column with name Value already exists!".
4. **`coerceForColumn` DOUBLE case** — Handles all float SQL type aliases.
5. **ADR-0092** — Written and indexed.
6. **AGENTS.md** — Columnar feature documented in two places (metaengine comment block + duckdbengine module description).
7. **TODO_LIST.md** — `coerceForColumn` item marked resolved with evidence.
8. **Verify gate** — Full GREEN, not stale. Ran in THIS session after all code changes.

---

## B) PARTIALLY DONE

1. **DuckDB LayoutPlanner follow-ups (TODO_LIST.md:52)** — The `coerceForColumn` sub-item is resolved, but 5 other sub-items remain: `explainScan`, helper centralization, `ApplyLayout` backfill docs, DuckDB layout benchmark, `adttest` matrix coverage.
2. **Status report from prior session** (`docs/status/2026-08-02_21-18_DuckDB-Columnar-Native-Storage.md`) — Still references `[R]()` generic, `REAL` type, and "TOTALLY FUCKED UP" items that are now fixed. Not updated.

---

## C) NOT STARTED

1. **SQLite `LayoutPlanApplier`** — Only DuckDB implements `LayoutPlanApplier`. SQLite/Postgres still fall back to `LayoutPlanner.ApplyLayout` with name heuristics, losing reflection-derived types. A `WithColumnarLayout` query on SQLite would silently mis-type float columns.
2. **Schema evolution** — No `ALTER TABLE ADD COLUMN` when result type changes. New fields are silently dropped from planned columns.
3. **DuckDB layout benchmark** — No benchmark proving the columnar advantage (vectorized GROUP BY vs JSON scan).
4. **`adttest.RunMatrix` coverage** — The `LayoutPlanner`/`LayoutPlanApplier` capability is not tested in the cross-engine matrix.
5. **COOKBOOK.md example** — No `WithColumnarLayout` recipe.
6. **README.md feature list** — `metaengine/README.md` doesn't mention columnar-native.
7. **CHANGELOG.md** — The `[Unreleased]` section doesn't mention the REAL→DOUBLE fix or the generic removal.
8. **SKILL.md** — No `WithColumnarLayout` recipe in the consumer-facing skill.

---

## D) TOTALLY FUCKED UP

1. **The REAL bug shipped and nobody noticed for multiple sessions.** `sqlTypeOf` mapped `float64→REAL` (FLOAT4, 32-bit). The prior session noticed precision issues and worked around them by using power-of-2-safe float values in tests instead of fixing the root cause. This is the definition of hiding a bug behind a test workaround. The tests PASSED but the feature was broken for any consumer with decimal data (prices, percentages, coordinates). Fixed this session.
2. **The reserved-name collision was undiscovered.** The prior session's 3 tests all used result types with fields named `Name`, `Category`, `Price`, `Quantity` — none collided with `key`/`value`. A consumer using a result type with a field named `Value` (an extremely common name) would get a DDL error. Fixed this session with a reserved-name guard.
3. **The decorative `[R]` generic was known and shipped anyway.** The prior session's status report explicitly documented it as a problem ("the wrong type would be silently ignored") but shipped it. Fixed this session by removing the generic entirely.

---

## E) WHAT WE SHOULD IMPROVE

### Process

1. **Stop shipping known bugs.** The prior session documented 3 bugs in its own status report and shipped them anyway. "Functionally complete" should mean "correct," not "compiles and tests pass with workarounds."
2. **Test with realistic data from day one.** Power-of-2-safe floats are a red flag. If tests need special values to pass, the implementation is wrong.
3. **Test field names that real users would use.** `Value`, `Data`, `Type`, `ID` — these are extremely common Go struct field names. None were tested.
4. **The prior session's status report had a section literally called "TOTALLY FUCKED UP" listing the REAL precision issue** — and it still took a new session to fix it. When you identify a correctness bug, fix it immediately, don't document it for later.

### Architecture

5. **The two-dispatch-path design (`LayoutPlanApplier` vs `LayoutPlanner`) is defensible but creates a silent degradation path.** On SQLite, `WithColumnarLayout` silently falls back to name heuristics. This should either be documented loudly or SQLite should implement `LayoutPlanApplier`.
6. **`sqlTypeOf` is shared between all engines.** The REAL→DOUBLE fix benefits all engines, but only DuckDB currently uses the reflection-derived types via `LayoutPlanApplier`. SQLite/Postgres get the old name heuristic.
7. **No integration test through `projectionadapter` or `stack/duckdb`.** The columnar feature is tested only at the engine level, not through the full projection lifecycle.

---

## F) Up to 50 Things We Should Get Done Next

### High Priority (correctness & completeness)

1. Implement `LayoutPlanApplier` on `sqliteEngine` so `WithColumnarLayout` produces accurate column types on SQLite too.
2. Implement `LayoutPlanApplier` on `pgEngine` for the same reason.
3. Add schema evolution support (`ALTER TABLE ADD COLUMN`) when result type changes between `Plan()` calls.
4. Add a DuckDB layout benchmark proving vectorized GROUP BY advantage over JSON scan.
5. Add `LayoutPlanner`/`LayoutPlanApplier` coverage to `adttest.RunMatrix`.
6. Write `WithColumnarLayout` integration test through `projectionadapter`.
7. Write `WithColumnarLayout` integration test through `stack/duckdb`.
8. Test `WithColumnarLayout` with nested structs (should flatten or store as JSON).
9. Test `WithColumnarLayout` with slices/maps in result type (should store as JSON or skip).
10. Test `WithColumnarLayout` with `json:"-"` tagged fields (should be skipped).
11. Test `WithColumnarLayout` dispatch on Memory engine (graceful no-op).
12. Test `WithColumnarLayout` dispatch on Pebble engine (graceful no-op).

### Medium Priority (documentation & DX)

13. Add `WithColumnarLayout` recipe to `metaengine/COOKBOOK.md`.
14. Add `WithColumnarLayout` to `metaengine/README.md` feature list.
15. Update `CHANGELOG.md [Unreleased]` with REAL→DOUBLE fix + generic removal.
16. Add `WithColumnarLayout` recipe to `SKILL.md` references.
17. Update the prior session's status report (`2026-08-02_21-18`) to reflect the fixes.
18. Document the no-backfill semantics of `ApplyLayout` (existing rows in `meta_map` remain invisible to planned-table queries).
19. Add `explainScan` for planned DuckDB paths (currently returns placeholder).
20. Centralize planned-table helpers (`extractFields`, `jsonFieldName`, `quoteIdent`, `plansColumnCompatible`) duplicated between `planned_sqlite.go` and `duckdbengine/layout_planner.go`.

### Lower Priority (polish & exploration)

21. Add `WithColumnarLayout` auto-detection for Counter/Aggregate read patterns on DuckDB.
22. Explore columnar compression (DuckDB native) for large planned tables.
23. Add `LayoutPlan` JSON serialization for plan persistence across restarts.
24. Add a diagnostic when `WithColumnarLayout` is used on a non-`LayoutPlanApplier` engine (warn about name-heuristic fallback).
25. Add `ApplyLayoutPlan` idempotency test with type changes (same collection, different result type).
26. Add a test verifying `key`/`value` field skipping with various casing (`KEY`, `Value`, `VALUE`).
27. Add a test for `coerceInteger` with `bool` input.
28. Add a test for `coerceReal` with string input ("3.14").
29. Add a test for `coerceForColumn` with `TEXT` type and non-string values.
30. Document the type coercion table in the ADR.
31. Add `WithColumnarLayout` to `FEATURES.md` status table (currently lists it as 🧪).
32. Update `FEATURES.md` to reflect columnar as DONE (not experimental) once SQLite `LayoutPlanApplier` lands.
33. Add a `PlanResult.Explain()` output showing columnar-native diagnostics.
34. Explore `WithColumnarLayout` + `FilterOnField`/`SortOnField` combination (should work: all columns extracted + specific ones indexed).
35. Test concurrent `ApplyLayoutPlan` calls (thread safety of `layoutMu`).
36. Add a test for `BuildColumnarLayoutPlan` with an empty struct (0 exported fields).
37. Add a test for `BuildColumnarLayoutPlan` with a pointer-to-struct type.
38. Add a test for `BuildColumnarLayoutPlan` with a non-struct type (map, slice).
39. Add a test for `BuildColumnarLayoutPlan` with embedded structs (anonymous fields).
40. Add a test for `BuildColumnarLayoutPlan` with `json:"name,omitempty"` tags.
41. Explore DuckDB zone map effectiveness on columnar planned tables.
42. Add a migration path for existing planned tables when `sqlTypeOf` changes (e.g., REAL→DOUBLE).
43. Document `LayoutPlanApplier` in `metaengine/doc.go`.
44. Add a `metaengine` godoc example for `WithColumnarLayout`.
45. Explore materializing aggregation results into a separate planned table (rollup).
46. Add OpenTelemetry tracing for `ApplyLayoutPlan` calls.
47. Add a health check for planned table existence on engine startup.
48. Explore `WithColumnarLayout` + `metaengine.MaterializeCost` interaction.
49. Add a test verifying `SerializablePlan` includes columnar layout info.
50. Explore automatic columnar detection based on DuckDB `EngineProfile.Layouts[ADTMap] == LayoutColumnar`.

---

## G) Questions I Cannot Answer Myself

### Q1: Should the prior session's status report (`docs/status/2026-08-02_21-18_DuckDB-Columnar-Native-Storage.md`) be updated or annotated?

It still references `[R]()` generic, `REAL` type, and a "TOTALLY FUCKED UP" section describing bugs that are now fixed. Per the `update-old-docs` skill convention, point-in-time reports should be annotated, not rewritten. But this report is from the same day — should I annotate it with a resolution note, or leave it as-is since it's only hours old?

### Q2: Should I implement SQLite `LayoutPlanApplier` now, or is DuckDB-only acceptable?

Currently `WithColumnarLayout` on SQLite silently degrades to name heuristics. This is a correctness gap (float columns get INTEGER type if the name contains "price" etc.). I can implement it (mechanical: copy the DuckDB pattern), but it expands the scope. Is DuckDB-only the right boundary for now, or should SQLite get it before this feature is considered shippable?

### Q3: Should `WithColumnarLayout` emit a warning diagnostic when the engine doesn't implement `LayoutPlanApplier`?

When a consumer writes `WithColumnarLayout()` but the assigned engine is SQLite/Memory/Pebble, the columnar flag is silently ignored (falls back to `LayoutPlanner` with name heuristics or no-op). A `Diagnostic{Level: Warn}` at plan time would make this visible. Should I add it?

---

## Resolution (2026-08-03)

REAL→DOUBLE fix shipped (`sqlTypeOf`: float64→DOUBLE, float32→REAL). Decorative `[R any]` generic removed. Reserved-name collision fix (fields named `key`/`value` skipped). `TestDuckDBEngine_ColumnarDoublePrecision` added. ADR-0092 written + indexed. AGENTS.md + TODO_LIST.md updated. Verify GREEN (80+ modules).

**Still open:** SQLite/Postgres `LayoutPlanApplier`; schema evolution (`ALTER TABLE`); DuckDB layout benchmark; `adttest.RunMatrix` coverage. Captured in TODO_LIST.md.
