# cqrs-lint: Dead Rule Fixes, Duplicate Removal & Test Coverage — Status Report

**Date:** 2026-07-16 20:47
**Commits:** `28d15288` → `e7629a01` (7 commits)
**Goal:** Audit-driven hardening — find broken rules, dead code, and test gaps; fix them all.

---

## a) FULLY DONE

### Fix Dead C004 Rule (CRITICAL)

**Commit:** `663a2775`

- C004 (checkpoint-before-async-complete) was **completely non-functional** since creation — `HasAsync` was never set on `ProjectionInfo`
- Scanner now inspects handler function literals passed to `NewProjection` for `go` statements (goroutine launches)
- Added 3 tests: no projections (negative), async projection (positive), sync projection (negative)

### Fix isOOAggregate Scanner Bug (CRITICAL)

**Commit:** `e7629a01`

- `isOOAggregate` called `nodeString(fn.Body)` which only handles `ast.Expr` — `*ast.BlockStmt` is NOT an `ast.Expr`, so it always returned `""`
- This meant OO aggregate detection (needed by A007 dual-model rule) was **silently broken since creation**
- Rewrote to use `ast.Inspect` for identifier extraction from the function body

### Merge Duplicate D002/D004 Rules

**Commit:** `f84a818e`

- D002 (per-file) and D004 (project-wide) both detected mixed JSON casing but gave **contradictory suggestions** (camelCase vs snake_case)
- Removed D004 entirely, kept D002 (better per-file locations)
- Fixed D002 suggestion to be neutral: "camelCase for API types, snake_case for event payloads"

### Remove 10 Dead Code Items

**Commit:** `f84a818e`

- `ManualType`, `ManualAggID` fields (never set, checked in `isCommandType`)
- `IsFunctional` field on DeciderInfo (never set)
- `HandlerInfo` type and `Handlers` slice (never populated)
- `IsEventEmitted()` method (never called)
- `AssertFindingsCount()` test helper (buggy: ignored ruleID parameter, counted all findings)
- `PositionOf()` method (never called)
- `LoadPackages()` function (never called)
- `BuildContextFromPackages()` function (never called)

### Split Oversized Files

**Commit:** `f84a818e`, `99ae12fc`

- `correctness/helpers.go` (333→190+148): split into `tx_helpers.go` + `swallow_helpers.go`
- `boilerplate/b009_b010_b012_b015.go` (315→260+65): split into `b009_b010_b012.go` + `b015.go`

### Add Positive Tests for 9 Previously Untested Rules

**Commit:** `e7629a01`

- A001 (manual command interface), A003 (explicit codec), A004 (type assertion), A005 (custom projection), A007 (dual model), C007 (time.Now in decider), D001 (inconsistent event naming)
- All 129 tests pass

### Fix README Inaccuracies

**Commit:** `f84a818e`

- Added C001 to autofix list (was missing)
- Fixed `--fast` description ("Critical/High" → "Critical")
- Removed D004 row
- Updated rule count to 60

### Deduplicate Shared Helpers

**Commit:** `28d15288`

- Consolidated 4 duplicated helper functions into `analyzer` package exports

### Wire Pipeline Metrics + Variable Payload Tracking

**Commit:** `0662a2f7`

- `--verbose` now shows per-detector timing sorted slowest-first
- Variable payload tracking: resolves `payload := UserCreated{}; event.New(..., payload)` to the actual type

### CLI Output Tests

**Commit:** `99ae12fc`

- 10 new tests for `formatFindingsText`, `parseColorMode`, `filterByExcludedPaths`, `groupFindingsByModule`

---

## b) PARTIALLY DONE

### Test Coverage

- **129 tests** total (up from ~114 at session start)
- Still missing positive tests for: B002, B003, B004 (only smoke), B005 (only smoke), A012 (only smoke), A013 (broken `_ = findings`), A016 (only smoke), A019 (only smoke)
- No SARIF golden file test
- No monorepo fixture test

### Finding Location Quality

- E004/E006 now point at real emission sites (done prior session)
- D001, D003, E003, B013, B014, A016, A019 still point at `go.mod:1:1`
- Some are legitimately project-level (no single source location)

---

## c) NOT STARTED

1. **Scanner detects `Type()` and `AggregateID()` methods** — `ManualType`/`ManualAggID` fields were removed (dead), but command detection only finds `ID()` methods. Commands with only `Type()`/`AggregateID()` won't be detected.
2. **Functional decider registration** — `decider.Decider[State]{}` composite literals are detected by A007's own AST scan, but NOT registered in `Registry.Deciders` with `IsFunctional`. (Field was removed as dead.)
3. **SARIF golden file test** — only JSON output is golden-tested
4. **Monorepo fixture test** — monorepo support works but has no dedicated test
5. **Catalog consolidation** — 3 catalog files (catalog.go, catalog_extra.go, catalog_extra2.go) still separate
6. **cobra dependency** — only used for 2 lines in main.go, could be replaced

---

## d) TOTALLY FUCKED UP

### `nodeString` Bug — Silent Failure Since Day One

The `nodeString` function only handled `ast.Expr` nodes. `isOOAggregate` passed `fn.Body` (a `*ast.BlockStmt`), which is NOT an `ast.Expr`, so `nodeString` returned `""` every single time. This means:

- **OO aggregate detection never worked** — `Registry.Deciders` was always empty for OO aggregates
- **A007 (dual-model detection) could never fire** — it depends on `Registry.Deciders` having `IsOO=true` entries
- This bug existed since the scanner was written and was never caught because there were **zero tests for A007** until I wrote one

**Root cause:** Go's type system doesn't prevent passing a `Node` that isn't an `Expr` to a function that only handles `Expr`. The function silently returns empty instead of failing.

**Lesson:** Silent fallbacks (returning `""` instead of panicking) are worse than crashes. A crash would have been caught immediately.

### C004 Dead From Creation

Same pattern: `HasAsync` field defined, read by C004, but never assigned anywhere. The entire rule was dead code masquerading as a working detector. No test caught it.

### My Own Failure: Forgot the Report

User explicitly asked for a `.md` status report. I got distracted by failing tests and committed without writing the report. Had to be reminded.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Every registry field should be populated or removed** — we just removed 10 dead fields, but the scanner still has blind spots (Type(), AggregateID() methods not detected)
2. **Every rule should have at least one positive test** — still ~9 rules with only smoke/negative tests
3. **Catalog should be auto-generated from detector constructors** — 3 manually-maintained catalog files are drift-prone
4. **`nodeString` should be deleted or rewritten** — it only handles Expr, not Stmt/Decl, causing silent failures

### Scanner Accuracy

5. **`detectFoldFunc` matches any type containing "Event"** — would match `EventBus`, `EventCounter`, etc.
6. **`isLikelyDecider` matches any function with "decide" in the name** — `decidedToLeave`, `indecisiveHandler` would match
7. **Event type constants not resolved** — `bus.Subscribe(SomeEvent, handler)` where `SomeEvent` is a const won't be captured
8. **No tracking of Save/Publish/Use/UsePublish calls in registry** — each rule re-scans the AST independently

### Type Model

9. **`CommandInfo` detection is incomplete** — only finds structs with `ID()` methods or `BasicCommand` embedding. Misses structs with `Type()` + `AggregateID()` but no `ID()`.
10. **`DeciderInfo` has no functional decider support** — `IsFunctional` was removed. Need a clean approach: either detect functional deciders in the scanner or remove the `IsOO`/functional distinction.

---

## f) Up to 50 Things to Get Done Next

### P0 — Critical Correctness (1% → 51%)

1. Add positive test for A013 (pointer BasicCommand) — currently has `_ = findings`
2. Add positive test for B002 (manual repository wiring)
3. Add positive test for B003 (subscribeall large switch)
4. Add positive test for B004 (command constructor boilerplate)
5. Add positive test for B005 (fold switch boilerplate)
6. Add positive test for A012 (missing tombstone handling)

### P1 — Scanner Accuracy (4% → 64%)

7. Fix `detectFoldFunc` — check for `event.Event` type, not just any type containing "Event"
8. Fix `isLikelyDecider` — require function name to start with "decide"/"Decide" (prefix match, not contains)
9. Implement `Type()` method scanning — register commands with manual `Type()` methods
10. Implement `AggregateID()` method scanning
11. Track `Save()`/`Publish()` calls in registry (eliminate per-rule AST re-scanning)
12. Track `bus.Use()`/`UsePublish()` calls in registry

### P2 — Finding Location Quality (20% → 80%)

13. D001: point at first mixed-convention event type emission
14. D003: point at first mixed logging library import
15. E003: point at first CQRS construct in the monolith package
16. B013: point at `NewRepository` call site
17. B014: point at `Use`/`UsePublish` call site
18. A016: point at dispatcher construction
19. A019: point at vendored import path

### P3 — Test Coverage

20. SARIF golden file test
21. Monorepo fixture test (multi-module testdata)
22. Snippet presence assertions in existing rule tests
23. A016 positive test (needs registry data for dispatcher)
24. A018 positive test (needs no Save/Publish/Folds)
25. A019 positive test (needs vendor/ directory)
26. E001 positive test (needs tier violation)
27. E002 positive test (needs circular dependency)
28. E005 positive test (needs unregistered command)
29. E006 positive test (needs emitted but unprojected event)

### P4 — Architecture & Cleanup

30. Consolidate 3 catalog files into 1
31. Remove `nodeString` or rewrite to handle all ast.Node types
32. Extract common finding-builder pattern (reduce per-detector boilerplate)
33. Replace `cobra.MaximumNArgs` with cmdguard equivalent (remove cobra dependency)
34. Add `--debug` flag to dump scanner registry state
35. Add exit code documentation to `--help`
36. Add `.cqrs-lint.json` schema validation

### P5 — CLI/UX

37. Per-detector timing in all output modes (not just --verbose)
38. Progress indicator for large monorepos
39. `--ci` mode (auto-detect CI, set color=never, format=sarif)
40. `--baseline` flag (only report new findings since a reference)
41. `cqrs-lint init` interactive wizard
42. `rules` subcommand search/filter
43. More auto-fix rules (currently only C001)

### P6 — Documentation

44. Per-rule documentation pages with fix examples
45. "Getting Started" guide
46. "Rule Suppression" guide
47. Architecture documentation for the scanner pipeline
48. Comparison table vs. other Go linters

### P7 — Future

49. Type info resolution (`go/types.Info`) for accurate payload matching
50. `--fix-dry-run` with diff output

---

## g) Top 3 Questions I Cannot Answer Myself

### 1. Should we add functional decider (`decider.Decider[State]{}`) registration to the scanner?

Currently the scanner only detects OO aggregates (via `uncommittedEvents` in method bodies). Functional deciders are detected by each rule's own AST scan independently. Options:

- **(a)** Register functional deciders in `Registry.Deciders` during scanning — cleaner, centralizes detection, but requires deciding what the `DeciderInfo` type should look like without `IsFunctional` (which we just removed)
- **(b)** Leave it to per-rule AST scans — simpler, works now, but duplicates scanning logic
- **(c)** Add a dedicated `FunctionalDeciders []DeciderInfo` slice — explicit separation

I can't decide the right type model here without understanding whether any other rule besides A007 needs to know about functional deciders.

### 2. Should D002 check all event payload structs (not just structs in general)?

D002 currently checks ALL structs with JSON tags. Most structs with JSON tags are NOT event payloads — they're API types, config structs, database models. Should D002 only check structs that are registered as `EventPayloadTypes`? This would reduce false positives but might miss inconsistencies in related types. The alternative is leaving it broad (it's an Info-severity rule, low confidence).

### 3. Should we remove `cobra` as a dependency?

`cobra` is used for exactly 2 lines in `main.go`:

```go
rootCmd.Args = cobra.MaximumNArgs(1)  // line 77
rootCmd.RunE = func(cmd *cobra.Command, ...) // line 78 (type reference)
```

cmdguard wraps cobra internally, so removing cobra means using cmdguard's API for arg validation. But I don't know if cmdguard exposes `MaximumNArgs` or equivalent — it might require wrapping the root command differently.
