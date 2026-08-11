# Status Report: Layout Planning Quality — Sort Fix, ExplainPlan Annotations, Test Coverage

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions or harvested into TODO_LIST.md. Original content retained below for historical context.**

**Date:** 2026-08-11 08:44
**Session scope:** Follow-up quality items from the ADR-0124 layout planning rollout (Phase 6b)
**Previous session:** `2026-08-11_08-20_layout-planning-followups-safe-backfill-real-rebuilds.md`

---

## a) FULLY DONE

### 1. Replaced O(n²) bubble sort with `slices.Sorted` ✅

**File:** `metaengine/relayout.go`

The `sortedQueryNames` helper had a hand-rolled insertion sort with a comment "avoid adding sort import just for this." This was a quality regression from the `dispatchFolds` extraction — the original `applyWithRecord` used `slices.Sorted(maps.Keys(...))`. Replaced with:

```go
func sortedQueryNames(queries map[string]queryMeta) []string {
    return slices.Sorted(maps.Keys(queries))
}
```

-11 lines, O(n²) → O(n log n). Added `slices` and `maps` imports (both already used elsewhere in the package, so no new dependencies).

### 2. Wired layout/priority annotations into `ExplainPlan()` ✅

**Files:** `metaengine/explain.go`, `metaengine/layout_observability.go`

`ExplainPlan()` is the primary operator diagnostic tool ("WHY did the planner assign this query to this engine?"). It showed engine name, ADT, complexity, cost estimate — but NOT the layout decision (Embed vs Normalize) or the resolved priority. `Doctor()` had a `--- Layout ---` section, but `ExplainPlan()` did not.

Added `layoutExplainAnnotation(pc, profile, queryName)` pure function that returns `" layout=Embed(Balanced)"`-style tags. Called once per query in the ExplainPlan loop. Now operators see:

```
  find_task: map via memory (O(1)) layout=Embed(Balanced)
```

**Design decision:** Made `layoutExplainAnnotation` a pure function (not a Store method) because ExplainPlan already holds `s.mu.RLock()`. A method would need to re-acquire the lock → deadlock. The pure function takes `*PriorityConfig` and `EngineProfile` directly and is safe to call under a held read lock.

**Bonus optimization:** While editing the loop, noticed `eng.Profile()` was called 4× per query iteration. `Profile()` constructs a new `EngineProfile` struct + copies the `Supports` map + runs `ApplyCalibration` — non-trivial allocation. Extracted to a single `profile := q.QueryEngine().Profile()` call. Net result: explain.go went from 387 → 385 lines (actually got smaller).

### 3. Direct test coverage for `dispatchFolds` behavior ✅

**File:** `metaengine/layout_followup_test.go`

`dispatchFolds` (the shared fold-dispatch path extracted last session) was only tested indirectly through Apply and Backfill. Added two behavioral tests:

1. **Multi-query dispatch** — One event (TaskCreated) dispatched to two queries (find_task: Map/Insert + count_by_status: Counter/Delta) on the same engine. Verifies both projections receive and process the event correctly.
2. **Multi-event dispatch** — Three sequential TaskCreated events to one query. Verifies all three entries land in the projection.

`dispatchFolds` is unexported, so tests exercise it through the public `Apply` API. This is intentional — testing through the public API catches integration issues that white-box tests miss.

### 4. End-to-end layout migration test ✅

**File:** `metaengine/layout_followup_test.go`

Full operator workflow test: Plan → Apply events → GetLayoutInfo (Embed) → ReplanLayout with WriteSpeed → verify diff (Embed→Normalize) → ConfirmRebuild → verify data integrity → verify EventLog not double-logged.

This is the test that should have existed when ConfirmRebuild was first written. It exercises the complete priority-driven layout migration path end-to-end.

### 5. ExplainPlan annotation tests ✅

**File:** `metaengine/layout_followup_test.go`

Two tests for the new annotation:
1. Default plan includes `layout=` and `Balanced` in output
2. After SetPriority(WriteSpeed), annotation reflects `WriteSpeed` and `Normalize`

### 6. All verification gates green ✅

| Gate | Result |
|------|--------|
| `go build -tags "goexperiment.jsonv2" ./metaengine/...` | PASS |
| `go vet -tags "goexperiment.jsonv2" ./metaengine/...` | PASS |
| `go test` (205 specs) | PASS (0.032s) |
| `go test -race` (full suite) | PASS (85s) |
| API stability (4085 exports) | PASS |
| Doc-check (724 refs, 44 packages) | PASS |
| Duplication check | 5 pre-existing clones (none from this session) |
| `gofumpt` + `goimports` | Applied to all 4 changed files |

### 7. Documentation updated ✅

- **CHANGELOG.md** — New `### Changed` entry at top of `[Unreleased]` documenting all 3 production changes + 5 new tests.
- **TODO_LIST.md** — Updated 2 items: "Wire layout warnings into Doctor() + EXPLAIN" expanded to include ExplainPlan, new "[x] Replace sortedQueryNames bubble sort" and "[x] dispatchFolds test coverage" items added.

---

## b) PARTIALLY DONE

Nothing is partially done this session. All 5 planned items were completed fully.

---

## c) NOT STARTED (deferred with reasoning)

### Intentionally Deferred

1. **Unify `priorityFactor` and `SelectLayout`** — These answer *different* questions. `priorityFactor` (in `planner.go`) adjusts read-cost estimates for engine routing — it penalizes high complexity under ReadSpeed. `SelectLayout` (in `layout_scoring.go`) scores embed-vs-normalize tradeoffs based on write/storage costs. Both use `Priority.Weights()` as the common input, but they operate on different cost dimensions. Forcing them into one function would be a false unification — the routing decision and the layout decision are legitimately separate concerns. Deferred, not abandoned — if the cost model evolves to share a common cost structure, they could converge naturally.

2. **`Store.ClearQuery(ctx, queryName)` API** — Would enable safe `ConfirmRebuild` for non-idempotent projections (clear first, then replay). Requires a new optional Engine capability interface (`CollectionDropper` or similar) — no engine currently exposes `DropCollection`/`Clear`/`Truncate`. This is a substantial feature (new interface + 9 engine implementations + tests), not a quick fix. The current behavior (refuse non-idempotent folds in ConfirmRebuild) is safe.

3. **Calibrate cost model multipliers** — The placeholder constants in `scoreEmbed`/`scoreNormalize` (0.5, 1.0, 1.3, 2.0) need measured values from real engine benchmarks. Requires benchmarking infrastructure and real workloads. Out of scope for this quality session.

4. **Pre-existing CI failures** — cqrs-lint catalog count (33→34), 4 modules missing from arch maps. Pre-existing, not caused by layout planning work. Noted but not addressed.

---

## d) TOTALLY FUCKED UP

### Nothing this session.

The session was clean — no mistakes, no broken builds, no reverts. This is notable because the previous session (documented in the handoff) had 6 specific mistakes. The key difference: I verified ground truth before acting (git status was clean, all previous work committed), and I caught the `eng.Profile()` 4× allocation issue during self-review before finishing.

### One near-miss worth documenting

**ExplainPlan loop structure** — My first edit of the ExplainPlan query loop accidentally dropped the `if s.plan != nil` guard block. I was editing the loop header (adding the layout annotation) and the old_string match consumed too much context, removing the plan-info conditional. Caught immediately on the next view, fixed in one follow-up edit. Lesson: when editing inside loops with nested conditionals, include MORE context in the old_string, not less, to ensure the edit is unambiguous.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (this codebase)

1. **`explain.go` is at 385 lines — over the 350-line CI limit.** It was 387 before this session (pre-existing). My change actually reduced it by 2 lines, but it's still over. The file has three concerns: TypedReader.Explain, TypedReader.ExplainAggregate, and Store-level observability (ExplainPlan + Doctor). These should be split into `explain.go` (TypedReader methods) and `explain_store.go` (Store methods: ExplainPlan + Doctor). This would bring both files well under 350.

2. **`layout_observability.go` is growing** — 177 lines now. With `layoutExplainAnnotation`, `LayoutDoctorSection`, `GetLayoutInfo`, and `LayoutWarnings`, it's approaching a split point. If more observability methods are added, consider splitting into `layout_info.go` (queries) and `layout_diagnostics.go` (warnings + doctor).

3. **`Profile()` allocation in hot paths** — The `Profile()` method on `memoryEngine` allocates a new struct + map copy + runs `ApplyCalibration` on every call. This is called in ExplainPlan, GetLayoutInfo, LayoutWarnings, ReplanLayout, and the planner. For a profiling/diagnostic path this is fine, but if these methods are ever called in hot loops (e.g., per-query per-event), the allocation matters. Consider caching the profile or providing a `ProfileReadOnly()` that returns a pointer without copying.

4. **`layoutExplainAnnotation` duplicates resolution logic** — It calls `pc.Resolve()` and `SelectLayout()` independently. `GetLayoutInfo` and `LayoutWarnings` do the same resolution. There's a shared "resolve priority + select layout for one query" pattern that could be extracted into a helper. But at 3 call sites with slightly different needs, YAGNI applies.

5. **Test coverage for `ExplainPlan` is thin** — The only ExplainPlan test (`TestStore_ExplainPlan` in `coverage_test.go`) is `t.Skip`'d because it requires SQLite. The two new tests I added verify the annotation exists, but don't test the full output structure. A comprehensive test using the Memory engine (not SQLite) would be valuable.

### Process

6. **The handoff document was excellent** — The "Exact Next Steps" section with 10 numbered items made resumption trivial. The honest self-assessment from the previous session ("what did you forget?") surfaced the exact items I addressed. This pattern should be maintained.

7. **Status reports are point-in-time** — I re-verified git status, build, and tests before acting, despite the handoff claiming "all done." This was the right call — the working tree was clean (auto-commit daemon had committed everything), but if I had assumed the handoff was current, I might have missed changes.

---

## f) Next Tasks (up to 50)

### High Priority (would fix real quality issues)

1. **Split `explain.go` into two files** — `explain.go` (TypedReader methods, ~75 lines) + `explain_store.go` (ExplainPlan + Doctor, ~310 lines). Gets both under 350-line CI limit.
2. **Split `layout_observability.go`** if more methods are added — preemptive, not urgent.
3. **Cache `memoryEngine.Profile()`** — or provide `ProfileRef()` returning `*EngineProfile` without map copy. Allocations in diagnostic paths.
4. **Add comprehensive `ExplainPlan` test** using Memory engine — test full output structure (engines section, queries section, diagnostics section), not just substring checks.
5. **Wire layout info into `Explain()` (TypedReader)** — Per-query `Explain()` shows SQL but not layout. Add layout annotation to the per-query explain path too.
6. **Add `Store.ReplanLayoutFromCurrentPriority()`** — Convenience method that calls `ReplanLayout` with the store's current `priorityConfig`, avoiding the need to pass it explicitly.
7. **Track layout changes in plan version history** — When `SetPriority` → `Replan` changes the layout, record the layout transition in the plan's diagnostic info.

### Medium Priority (feature gaps)

8. **`Store.ClearQuery(ctx, queryName)` API** — Requires new `CollectionDropper` engine capability. Enables safe ConfirmRebuild for non-idempotent projections.
9. **Unify `priorityFactor` and `SelectLayout` cost inputs** — Not merging the functions, but extracting a shared `LayoutCostEstimate` that both can consume.
10. **Calibrate `scoreEmbed`/`scoreNormalize` constants** — Replace placeholder multipliers with measured values from Pebble/SQLite/PG benchmarks.
11. **`cqrs-bench layout` CLI subcommand** — Pre-deployment "what if" exploration tool for operators.
12. **Real workload trace format** — JSON-lines spec for benchmark calibration of the cost model.
13. **Multi-engine integration test with two real backends** — Current tests use MemoryEngine only. Need Pebble+Memory or SQLite+Memory to verify cross-engine backfill.
14. **Wire `Priority` into deployment YAML** — `EngineConfig`/`DriverConfig` + config validation.
15. **Layout audit trail** — Plan version history showing who changed what priority when.
16. **Update SKILL.md + skill references** — Layout planning concepts, priority system, consumer docs.

### Lower Priority (nice to have)

17. **Fold-pipeline sync for Active+DualUse roles** — Event → all Active+DualUse projections in one transaction (strong consistency).
18. **Async replication for Backup+Migration roles** — Eventual consistency, failure-isolated.
19. **Role transition API** — Backup→Active promote, Migration→Active cutover.
20. **Aggregate boundary config** — `WithSharedCollection("Attachment")` opt-in.
21. **Per-fold mutex instead of global foldMu** — Allow parallel fold execution across different queries.
22. **`Hybrid` layout scoring** — `LayoutHybrid` exists but is not scored. Add a third scoring branch for mixed embed+normalize.
23. **Layout-aware schema generation** — When layout is Normalize, auto-generate child table DDL.
24. **Layout migration dry-run** — `ReplanLayout` returns diffs; add a `--dry-run` mode that shows estimated time and resource cost.
25. **Per-query priority override in `QueryDecl`** — Currently priority is operator-only (via `PriorityConfig`). Allow per-query defaults in the declaration.
26. **Layout planning metrics** — OTel counters for layout changes, rebuild triggers, warning emissions.
27. **`Doctor()` layout section improvements** — Show cost scores, not just selected option. Show "why" (cost breakdown).
28. **Integration test: priority change under load** — Apply events concurrently with SetPriority → verify no races or data corruption.
29. **Layout planning in `system.Deployer`** — Wire priority config into the composition root.
30. **Snapshot support for layout migration** — Take a snapshot before ConfirmRebuild, enable rollback if rebuild fails.

### Pre-existing Issues (not from this session, not layout-related)

31. **Fix cqrs-lint catalog count** — Expected 33, got 34. Likely a new module added without updating the expected count.
32. **Fix 4 modules missing from arch maps** — `scripts/check-module-layers.sh` needs updating.
33. **Fix 5 pre-existing duplication clones** — All pre-date layout planning work.
34. **Tag `benchkit/v4.4.0`** — `Truncate`/`TitleCase` added after v4.3.0 tag.
35. **Document commandlifecycle in skill references** — modules.md, recipes.md, advanced.md don't mention it.
36. **Test pgtestcontainer external DSN isolation** — M18 change is untested.
37. **Investigate untracked files from other sessions** — `graph_fallback_e2e_test.go`, `commandlifecycle/projections/projections.go`, bbolt test files, skill reference changes.

### Documentation

38. **ADR-0124 follow-up ADR** — Document the decision that layout is advisory-only (engines don't physically enforce Embed vs Normalize yet).
39. **Update `METAENGINE-LAYOUT-PLANNING-MODEL.md`** — Add the idempotency classification table and the `dispatchFolds` architecture.
40. **Add layout planning section to `recipes.md`** — Consumer-facing recipe for SetPriority → ReplanLayout → ConfirmRebuild workflow.
41. **Add `layoutExplainAnnotation` to module docs** — Explain the `layout=X(Priority)` format in the skill reference.
42. **Document the `Profile()` allocation pattern** — In AGENTS.md gotchas, note that `Profile()` allocates and should be cached in loops.

### Testing

43. **Property-based test for `SelectLayout`** — Use rapid to verify the layout selection is deterministic and consistent across all priority/storage-layout combinations.
44. **Fuzz test for `dispatchFolds`** — Verify it handles arbitrary event payloads without panicking.
45. **Benchmark `layoutExplainAnnotation`** — Ensure it doesn't allocate in the ExplainPlan hot path.
46. **Test `LayoutWarnings` with all 4 priorities × all 4 storage layouts** — Full matrix coverage (currently tests 3 of 16 combinations).
47. **Test `ConfirmRebuild` with mixed idempotent + non-idempotent queries** — Verify the idempotency check filters correctly when some queries are safe and others aren't.
48. **Test `ReplanLayout` with per-engine and per-query priority overrides** — Currently only tests global priority.
49. **Test `ExplainPlan` with `s.plan == nil`** — Verify graceful degradation when plan hasn't been computed yet.
50. **Test `sortedQueryNames` with empty map** — Edge case: zero queries should return empty slice, not panic.

---

## g) Questions

### 1. Should `explain.go` be split now, or wait until it naturally grows more?

It's at 385 lines (pre-existing over the 350 limit). My change reduced it by 2 lines. The `check-file-size` gate is NOT part of `#verify` (it's a separate `nix run .#check-file-size`), so it doesn't block CI. But the AGENTS.md says "Max 350 lines/file (CI-enforced)" — which is aspirational since the gate exists but isn't in the main verify pipeline. Should I split it now (clean 2-file separation: TypedReader methods vs Store methods), or leave it since it's been over 350 for multiple sessions without consequence?

### 2. Is the `layoutExplainAnnotation` format (`layout=Embed(Balanced)`) the right UX?

I chose `layout=Embed(Balanced)` for compactness — it reads as "layout is Embed, selected under Balanced priority." Alternatives: `layout=Embed priority=Balanced` (more verbose), `[Embed/Balanced]` (bracketed), or `Embed(Balanced)` without the `layout=` prefix. This is a diagnostic string operators will read frequently. Is the current format clear, or should it be changed before it's baked into muscle memory?

### 3. Should `Profile()` caching be a priority, or is the allocation acceptable for diagnostic paths?

`memoryEngine.Profile()` allocates a struct + copies the `Supports` map + runs `ApplyCalibration` on every call. It's called in ExplainPlan, GetLayoutInfo, LayoutWarnings, ReplanLayout, and the planner's query loop. For diagnostic methods called occasionally, this is fine. But the planner's `planQuery` calls it in a loop over all engines × all queries — that's the hot path during `Plan()`. Should I add profile caching now, or defer until profiling shows it's a bottleneck?
