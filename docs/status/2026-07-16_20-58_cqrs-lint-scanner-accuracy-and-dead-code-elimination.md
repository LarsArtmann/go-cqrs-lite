# cqrs-lint: Scanner Accuracy, Test Coverage & Dead Code Elimination — Final Status

**Date:** 2026-07-16 20:58
**Commits this session:** `28d15288` → `e3900ec3` (14 commits across 3 rounds)
**Metrics:** 60 rules, 133 tests, 34/60 detectors with snippets, 14 `go.mod:1:1` locations (all legitimate project-level), 0 lint issues, 0 build errors

---

## a) FULLY DONE

### Scanner Accuracy Fixes (3 Critical Bugs Fixed)
| Bug | Fix | Commit |
|-----|-----|--------|
| `capturePayloadType` matched `id` (arg 1) instead of payload (arg 4) in `event.New()` | Start scanning at arg index 4 | `3ff5d255` |
| `detectFoldFunc` matched any type containing "Event" (`EventBus`, `EventCounter`) | New `looksLikeEventType` requires both "event" and "Event" | `b2ff7e65` |
| `isLikelyDecider` matched any function with "decide" anywhere (`indecisiveHandler`) | Changed `Contains` → `HasPrefix` | `b2ff7e65` |
| `isOOAggregate` always returned false (`nodeString` only handled `ast.Expr`, not `ast.BlockStmt`) | Rewrote with `ast.Inspect` | `e7629a01` |
| `C004` (checkpoint-before-async) was completely dead — `HasAsync` never set | Scanner now detects `go` statements in projection handler closures | `663a2775` |
| `Type()` and `AggregateID()` methods not scanned — commands missed | New `scanTypedMethod` alongside existing `scanIDMethod` | `b2ff7e65` |
| Variable payloads (`payload := UserCreated{}; event.New(..., payload)`) not resolved | Track `:=` assignments, resolve variable → type name | `0662a2f7` |

### Dead Code Removed (13 items)
| Item | Location | Commit |
|------|----------|--------|
| `TypeResolver` struct | `types.go` | `28d15288` |
| `ManualType` field | `CommandInfo` | `f84a818e` |
| `ManualAggID` field | `CommandInfo` | `f84a818e` |
| `IsFunctional` field | `DeciderInfo` | `f84a818e` |
| `HandlerInfo` type | `types.go` | `f84a818e` |
| `Handlers` slice | `registry.go` | `f84a818e` |
| `IsEventEmitted()` method | `registry.go` | `f84a818e` |
| `AssertFindingsCount()` helper | `test_helpers.go` (also buggy) | `f84a818e` |
| `PositionOf()` method | `test_helpers.go` | `f84a818e` |
| `LoadPackages()` function | `loader.go` | `f84a818e` |
| `BuildContextFromPackages()` | `loader.go` | `f84a818e` |
| `nodeString()` function | `ast_helpers.go` | `e3900ec3` |
| `internal/ast/` empty directory | filesystem | `28d15288` |

### Helper Deduplication (4 duplicates consolidated)
| Function | Was in | Now in | Commit |
|----------|--------|--------|--------|
| `selectorPackage` (3 copies) | api, boilerplate, analyzer | `analyzer.SelectorPackage` | `28d15288` |
| `baseTypeName` (2 copies) | api, analyzer | `analyzer.BaseTypeName` | `28d15288` |
| `extractJSONTag` (2 copies) | api, consistency | `analyzer.ExtractJSONTag` | `28d15288` |
| `isCQRSModule` (2 copies) | architecture, analyzer | `analyzer.IsCQRSModulePath` | `28d15288` |

### Rule Quality
- **D004 removed** — duplicate of D002 with contradictory suggestion (camelCase vs snake_case)
- **34/60 detectors** now have source snippets (was 0 before this session started)
- **E004/E006** point at real emission sites (track file+line) instead of `go.mod:1:1`
- `EventTypesEmitted` upgraded from `map[string]string` to `map[string]EventEmission{File, Line}`

### File Splits (CI 350-line limit)
| File | Before | After | Commit |
|------|--------|-------|--------|
| `correctness/helpers.go` | 333 lines | `tx_helpers.go` (190) + `swallow_helpers.go` (148) | `99ae12fc` |
| `boilerplate/b009_b010_b012_b015.go` | 315 lines | `b009_b010_b012.go` (260) + `b015.go` (65) | `f84a818e` |

### Test Coverage (133 tests total, up from ~114)
| Tests Added | Coverage Area | Commit |
|-------------|--------------|--------|
| 14 analyzer scanner tests | Event payload capture, emission tracking, catalog, RegisterTyped, NewProjection, Subscribe, command detection, filterEventPayloads, SourceLine caching/edge cases, variable payload resolution | `3ff5d255`, `0662a2f7` |
| 10 CLI output tests | formatFindingsText (empty, basic, snippet), parseColorMode, filterByExcludedPaths (substring, glob, empty), groupFindingsByModule | `99ae12fc` |
| 7 positive rule tests | A001, A003, A004, A005, A007, C007, D001 | `e7629a01` |
| 5 positive rule tests + 1 fix | A013 (broken → fixed), B004, B005, E005, E006 | `20303329` |
| 3 C004 tests | No projections, async projection (positive), sync projection (negative) | `663a2775` |

### CLI Polish
- `--verbose` shows per-detector timing sorted slowest-first (pipeline metrics wired)
- `--verbose` shows module count, detector count, pre-filter finding count
- Module-grouped output (`=== module/path ===` headers)
- `--exclude` now supports `filepath.Match` glob patterns alongside substring matching
- `SourceLine` caches file contents via `sync.Map` (eliminates redundant disk reads)
- `outputFindings` moved from main.go → output.go

### Documentation
- README updated: 60 rules, correct per-category counts, all 15 boilerplate rules documented, A011/A014/A017/B006/B007/B009/B010/B012/B015 added, `--color`/`--exclude`/`--verbose` documented
- AGENTS.md rule count updated (61→60 after D004 removal)
- Autofix list corrected (C001 was missing)
- `--fast` description corrected ("Critical/High" → "Critical")
- Planning doc with mermaid graph at `docs/planning/2026-07-16_20-47_cqrs-lint-comprehensive-quality-plan.md`

---

## b) PARTIALLY DONE

### Test Coverage Gaps
- **B002, B003** — only smoke tests, no positive tests
- **A012, A016, A018, A019** — only smoke/negative tests, no positive tests
- **E001, E002** — only smoke tests (hard to construct positive test data for tier violations)
- No SARIF golden file test (only JSON)
- No monorepo fixture test
- No snippet presence assertions in existing rule tests

### Finding Location Quality
- 14 findings still point at `go.mod:1:1` — all are legitimately project-level (absence checks like "missing OTel", "missing idempotency", "no stack preset")
- E004/E006 improved to point at real emission sites
- D001, D003, B013, B014, A016, A019 could theoretically point at more specific locations but would require construction-site scanning

---

## c) NOT STARTED

1. **Consolidate 3 catalog files** into 1 (catalog.go, catalog_extra.go, catalog_extra2.go)
2. **Replace `cobra` dependency** (used for 2 lines in main.go)
3. **`--debug` flag** to dump scanner registry
4. **SARIF golden file test**
5. **Monorepo fixture test** (multi-module testdata)
6. **Functional decider registration** in scanner (`decider.Decider[State]{}` not in `Registry.Deciders`)
7. **Per-rule documentation pages**
8. **Config file validation** (`.cqrs-lint.json` keys not validated)

---

## d) TOTALLY FUCKED UP

### `nodeString` — Silent Failure Pattern (FIXED)
`nodeString` only handled `ast.Expr`. `isOOAggregate` passed `fn.Body` (`*ast.BlockStmt`, NOT an `ast.Expr`), so it always returned `""`. OO aggregate detection never worked. A007 could never fire. This existed from day one and was only caught when I wrote A007's first positive test and it failed.

**Lesson:** Functions that silently return zero values instead of failing are worse than crashes. The crash would have been caught immediately.

### C004 — Dead From Creation (FIXED)
`HasAsync` was defined, read by C004, but never assigned. The entire rule was dead code. No test caught it until this session.

### `capturePayloadType` — Matched Wrong Argument (FIXED)
Iterated ALL arguments of `event.New()` looking for the first `*ast.CompositeLit` or `*ast.Ident`. Since `id` (arg index 1) is an `*ast.Ident`, it captured `"id"` as the payload type and returned immediately — never reaching the actual payload at arg 4. S002 was completely broken.

### oxfmt Pre-Commit Corruption
The pre-commit oxfmt reformatter corrupted `looksLikeEventPayload(name)` into `slices.Contains()` during a commit, deleting the helper function entirely. Required manual restoration from the previous commit. This is a tooling reliability issue, not a code issue.

### My Own Process Failures
1. **Forgot the report twice** — user had to remind me both times
2. **Committed without writing the status `.md`** — got sidetracked by failing tests
3. **Wrote tests that didn't compile** — A001 test didn't include `ID()` method, A003 used generic syntax the detector didn't handle

---

## e) WHAT WE SHOULD IMPROVE

### Architecture
1. **Catalog is 3 manually-maintained files** — decoupled from detector constructors, drift-prone. Should consolidate.
2. **Every registry field should be populated or removed** — we removed 13 dead items, but `cobra` dependency is still 2 lines
3. **Scanner re-scans AST in every rule** — Save/Publish/Use/UsePublish calls not tracked in registry, each rule inspects independently

### Type Model
4. **`CommandInfo` detection now finds ID(), Type(), AggregateID(), BasicCommand** — but the fields for the latter two were removed. Commands are registered but we lost the ability to distinguish HOW they were detected.
5. **Functional deciders not in registry** — `decider.Decider[State]{}` is detected by A007's own AST scan, but not centralized in `Registry.Deciders`
6. **`looksLikeEventType` is still heuristic** — requires both "event" and "Event" in the type string. Would be better with `types.Info` but that's a bigger change.

### Testing
7. **8 rules still lack positive tests** (B002, B003, A012, A016, A018, A019, E001, E002)
8. **No integration test** running cqrs-lint against a real project
9. **No SARIF output test**
10. **Snippet presence not verified in tests** — tests check finding count but not `.Snippet != ""`

---

## f) Up to 50 Things to Get Done Next

### P0 — Correctness & Accuracy
1. Add positive test for B002 (manual repository wiring)
2. Add positive test for B003 (subscribeall large switch)
3. Add positive test for A012 (fold without tombstone check)
4. Add positive test for A016 (missing idempotency middleware)
5. Add positive test for A018 (no actual event sourcing)
6. Add positive test for A019 (vendored cqrs)
7. Add positive test for E001 (tier violation)
8. Add positive test for E002 (circular dependency)
9. Track `Save()`/`Publish()` calls in registry (eliminate per-rule re-scanning)
10. Track `bus.Use()`/`UsePublish()` calls in registry

### P1 — Architecture & Cleanup
11. Consolidate 3 catalog files into 1 `catalog.go`
12. Replace `cobra.MaximumNArgs` with cmdguard equivalent (remove cobra dependency)
13. Register functional deciders in `Registry.Deciders` during scanning
14. Extract common finding-builder pattern (reduce per-detector boilerplate)
15. Add `--debug` flag to dump scanner registry state
16. Add exit code documentation to `--help`

### P2 — Finding Location Quality
17. D001: point at first mixed-convention event emission
18. D003: point at first mixed logging import
19. B013: point at `NewRepository` call site
20. B014: point at `Use`/`UsePublish` call site
21. A016: point at dispatcher construction
22. A019: point at vendored import path

### P3 — Testing
23. SARIF golden file test
24. Monorepo fixture test (multi-module testdata dir)
25. Snippet presence assertions in existing rule tests
26. Integration test: run cqrs-lint on `example/taskmanager/`
27. Benchmark SourceLine caching (before/after allocation comparison)
28. Config file validation test (`.cqrs-lint.json`)
29. `--verbose` module grouping integration test
30. `--color` mode unit tests (TTY detection mock)

### P4 — CLI/UX
31. Progress indicator for large monorepos
32. `--ci` mode (auto-detect CI, set color=never, format=sarif)
33. `--baseline` flag (only report new findings since a reference run)
34. `cqrs-lint init` interactive wizard
35. `rules` subcommand search/filter
36. More auto-fix rules (currently only C001)
37. `--fix-dry-run` with diff output

### P5 — Scanner Accuracy
38. Resolve event type constants (`bus.Subscribe(SomeEvent, handler)` where `SomeEvent` is a const)
39. `detectFoldFunc` exact type matching using `types.Info` (future)
40. Projection `Subscribe` receiver type verification
41. Multi-site `EventTypesEmitted` tracking (currently last-wins)
42. Detect `event.Single` calls as event emissions

### P6 — Documentation
43. Per-rule documentation pages with fix examples
44. "Getting Started" guide for new users
45. "Rule Suppression" guide (`//cqrs-lint:disable`)
46. Architecture documentation for the scanner pipeline
47. Comparison table vs. other Go linters

### P7 — Advanced
48. `types.Info` resolution for accurate payload matching
49. `--fix-dry-run` with unified diff output
50. Plugin system for custom rules

---

## g) Top 3 Questions I Cannot Answer Myself

### 1. Should we register functional deciders (`decider.Decider[State]{}`) in the scanner?

Currently only OO aggregates (detected via `uncommittedEvents` in method bodies) are in `Registry.Deciders`. Functional deciders are detected by each rule's own AST scan independently. `IsFunctional` was removed as dead code. Options:
- **(a)** Add functional decider detection to the scanner, use a new `Source string` field on `DeciderInfo` to distinguish "oo" vs "functional"
- **(b)** Leave it to per-rule AST scans (works now, but duplicates logic)
- **(c)** Add `FunctionalDeciders []DeciderInfo` separate slice

I can't decide the right type model without knowing if rules other than A007 need functional decider data.

### 2. Should D002 check only event payload structs instead of ALL structs with JSON tags?

D002 currently checks every struct with JSON tags. Most are NOT event payloads (API types, config, DB models). Should D002 only check structs registered as `EventPayloadTypes`? This reduces false positives but might miss inconsistencies. Alternatively, keep it broad since it's Info-severity with low confidence.

### 3. Should we remove `cobra` as a dependency?

`cobra` is used for exactly 2 lines: `cobra.MaximumNArgs(1)` and the `*cobra.Command` type. cmdguard wraps cobra internally, so removing it means finding cmdguard's API for arg validation. I don't know if cmdguard exposes `MaximumNArgs` or equivalent without studying its API surface more deeply.
