# Status: Per-Module Feature Profiles for cqrs-lint Coaching Rules

> **✅ FULLY RESOLVED 2026-08-11 — archived.** Every actionable item in this report shipped. See CHANGELOG `[Unreleased]` for where the work landed and TODO_LIST.md for any remaining follow-ups (none specific to this report).

> **Date:** 2026-08-11 19:20
> **Session scope:** Consumer feedback item "Per-module feature profiles" from cqrs-htmx feedback (2026-08-04)
> **Feedback file:** `docs/feedback/new/2026-08-04_cqrs-htmx_cqrs-lint-feedback-round2.md` Issue 2

---

## What Was the Problem

cqrs-htmx is a Go workspace with 19 independent modules: a library + example apps.
cqrs-lint detected features across the ENTIRE workspace and applied a single merged
`FeatureProfile` to all modules. This meant an example app's `ListenAndServe` set
`server=true` for the library module, enabling server-only rules (E009, E014, F011,
F012, F013, P012, P013) on library code where they're false positives. Estimated
~10 false positives per workspace.

The infrastructure (`DetectFeaturesPerModule`, `FeatureProfiles`, `ProfileForFile`)
was already wired in the loader (`loader.go:167-172`). The gap was that **18 rules**
still read the global `ctx.FeatureProfile` instead of evaluating per-module.

---

## a) FULLY DONE

### Infrastructure created

| File | Role | Lines |
|---|---|---|
| `module_scope.go` | `moduleScope` struct, `coachingScopes()` iterator, `attributeModule()`, `nonTestFiles()` | 97 |
| `scan_in.go` | File-slice-scoped scan helpers (`importsPathIn`, `hasCallIn`, `hasSelectorIn`, `firstCallPosIn`, `firstFilePosIn`, `firstFuncDeclPosIn`, `firstCallByNameIn`, `countCallsIn`) | 284 |

### Rules migrated to per-module evaluation (15 adoption + 3 resilience = 18 total)

| Rule | File | Migration approach |
|---|---|---|
| F003 | `f003_f004.go` | `coachingScopes` — HasServer per module |
| F004 | `f003_f004.go` | `coachingScopes` — HasServer + ServerLocal per module |
| F007 | `f007_f008.go` | `coachingScopes` — CommandFlow per module |
| F009 | `f009.go` | `coachingScopes` — HasServer + CommandFlow per module |
| F012 | `f012_f013.go` | `coachingScopes` — CommandFlow per module |
| F013 | `f012_f013.go` | `coachingScopes` — HasServer + ServerLocal + HasTransport per module |
| F017 | `f015_f016_f017.go` | `coachingScopes` — HasAsyncBus per module |
| F022 | `f022.go` | `coachingScopes` — Store.IsSQL() per module |
| F023 | `f023_f024_f025.go` | `coachingScopes` — Store.IsSQL() per module |
| F024 | `f023_f024_f025.go` | `coachingScopes` — Store.IsSQL() per module |
| F025 | `f023_f024_f025.go` | `coachingScopes` — Store.IsSQL() per module |
| F026 | `f026.go` | `coachingScopes` — HasMetaengine per module |
| F027 | `f027_f028_f029.go` | `coachingScopes` — HasServer per module |
| F028 | `f027_f028_f029.go` | `coachingScopes` — HasServer per module |
| F029 | `f027_f028_f029.go` | `coachingScopes` — HasServer per module |
| B029 | `b029.go` | `ProfileForFile(pos.Filename).HasServer` per finding |
| B030 | `b030.go` | `ProfileForFile(pos.Filename).HasServer` per finding |
| B031 | `b031.go` | `ProfileForFile(pos.Filename).HasServer` per finding |

### Existing scan helpers refactored (delegation, zero duplication)

All ctx-based scan helpers now delegate to the `In` variants:
- `importsPath` -> `importsPathIn`
- `projectHasCallAny` -> `hasCallIn`
- `projectHasSelector` -> `hasSelectorIn`
- `firstCallPos` -> `firstCallPosIn`
- `firstFuncDeclPos` -> `firstFuncDeclPosIn`
- `firstCallByName` -> `firstCallByNameIn`
- `firstManualSortPos` -> `firstManualSortPosIn`
- `firstManualFilterPos` -> `firstManualFilterPosIn`
- `firstManualPaginationPos` -> `firstManualPaginationPosIn`
- `firstManualAggregationPos` -> `firstManualAggregationPosIn`
- `firstNewReaderPos` -> `firstNewReaderPosIn`
- `hasTimeBasedPatterns` -> `hasTimeBasedPatternsIn`
- `hasWebFrameworkHandlers` -> `hasWebFrameworkHandlersIn`
- `countCalls` -> `countCallsIn`

Dead code removed: `hasSQLStore` (was only used by F022-F025 which now use
`sc.profile.Store.IsSQL()` inline).

### Tests added

| Test file | Tests | Verifies |
|---|---|---|
| `coaching_permodule_test.go` | 5 tests | F003 library suppression, F013 library suppression, F022 store isolation, F003 single-module fallback, `coachingScopes` single-module file count |
| `b029_b031_permodule_test.go` | 4 tests | B029 library suppression, B029 server-module fires, B031 library suppression, B029 single-module fallback |

### Verification

- `go build` — clean
- `go vet` — clean
- `go test ./...` — all 18 packages pass
- `go test -race -count=1 ./...` — all 18 packages pass
- Zero stale `ctx.FeatureProfile` references in rule production code
- All files under 350-line CI limit (module_scope.go: 97, scan_in.go: 284)

---

## b) PARTIALLY DONE

### F015 and F016 — NOT migrated (workspace-global count rules)

**F015** (`f015_f016_f017.go`) counts query registrations across the ENTIRE workspace
via `countCalls(ctx, "query", "Register*")`. In a multi-module workspace, if the library
has 2 queries and an example has 1, the workspace total is 3 — triggering the rule when
neither module alone would qualify. This is a per-module sensitivity gap.

**F016** counts distinct aggregate types via `distinctAggregateCount(ctx)` which reads
`ctx.Registry.EventTypesEmitted` — a workspace-global set. Same cross-module counting risk.

**Why not migrated:** Neither uses `ctx.FeatureProfile` directly. They were not caught
by the `grep ctx.FeatureProfile` audit. They use workspace-global registry/count helpers.

**Impact:** LOW. False positives require queries/events spread across modules such that
no single module crosses the threshold but the workspace total does.

### F018, F019, F020, F021 — NOT migrated (metaengine coaching rules)

These use `usesMetaengine(ctx)` (workspace-global import scan) as their gate. The actual
API call detection (`firstCallPos`, AST inspection) also scans workspace-global.

**Why not migrated:** They don't reference `ctx.FeatureProfile`. The leakage risk is LOW
because metaengine imports are unlikely to be in one module while the API calls (FilterOn,
etc.) are in another.

**Impact:** LOW. But for consistency, they should use `coachingScopes` too.

### Test coverage — only 5 of 18 migrated rules have per-module tests

Only F003, F013, F022, B029, B031 have per-module regression tests. The remaining 13
migrated rules (F004, F007, F009, F012, F017, F023, F024, F025, F026, F027, F028, F029, B030)
have NO per-module tests verifying that cross-module leakage is suppressed.

---

## c) NOT STARTED

1. **`nix fmt` not run** — AGENTS.md mandates formatting before finishing. Treefmt may
   reformat the new files.
2. **CHANGELOG.md not updated** — cqrs-lint has a CHANGELOG.md that should record this fix.
3. **Doctor command not updated** — `doctor` shows only the primary profile. In a multi-module
   workspace, consumers can't see per-module profiles to debug why a rule fired or didn't.
4. **API stability golden not regenerated** — no exported symbols changed, but the check should
   be run to confirm.
5. **Library self-lint test not run** — `library_self_lint_test.go` verifies cqrs-lint works
   on the go-cqrs-lite repo itself. Not run this session.
6. **Integration test on real multi-module workspace** — the unit tests use synthetic source.
   No test runs cqrs-lint against a real multi-module `go.work` project.
7. **F008 workspace-global `eventCount(ctx)`** — reads `ctx.Registry.EventTypesEmitted` which
   is workspace-global. In a multi-module workspace, event types from all modules are merged.
   This could cause F008 (CBOR coaching) to fire when no single module has 5+ events.

---

## d) TOTALLY FUCKED UP

### Auto-git daemon committed 374-line module_scope.go (CI limit: 350)

The auto-git daemon committed `module_scope.go` at commit `f3c29ec76` when it was 374 lines
(exceeding the 350-line CI limit). The working tree now has the fix (split into `module_scope.go`
97 lines + `scan_in.go` 284 lines), but the committed version is non-compliant.

**Fix:** The working tree is correct. The daemon will re-commit the split. If CI runs before
that, the 374-line check will fail at `f3c29ec76`.

### Stale LSP diagnostics showing phantom errors

The LSP keeps reporting `unusedfunc` warnings for `coachingScopes`, `firstFilePosIn`, and
`countCallsIn` at line numbers from the OLD 374-line version of `module_scope.go`. These are
phantom errors from a stale gopls snapshot — the functions exist and are used in `scan_in.go`.
A `gopls` restart would clear them.

---

## e) WHAT WE SHOULD IMPROVE

1. **The `grep ctx.FeatureProfile` audit was insufficient.** The real test is "does this rule
   scan workspace-global files?" — not just "does it read the profile?" Rules like F015/F016/F018-
   F021 use workspace-global helpers (`usesMetaengine`, `countCalls`, `eventCount`) that have the
   same cross-module leakage risk but don't reference `ctx.FeatureProfile` directly.

2. **No integration test on a real multi-module workspace.** Unit tests with synthetic source
   verify the logic, but a test that creates a real `go.work` with two `go.mod` files and runs
   the full linter would catch wiring issues the unit tests miss.

3. **The doctor command is blind to per-module profiles.** A consumer debugging "why did/didn't
   this rule fire?" can only see the primary profile. Per-module profile visibility in doctor
   output would make the feature discoverable and debuggable.

4. **Single-module fallback path is tested but not the multi-module real-loader path.** The
   `BuildContextFromSource` test helper sets `FeatureProfiles` manually. The actual loader path
   (`loader.go:167-172`) that calls `DetectFeaturesPerModule` is only tested indirectly.

5. **No regression test for the specific cqrs-htmx scenario.** The feedback describes a library
   with 19 modules where examples flip server=true. The test should replicate that exact scenario
   (library + example with ListenAndServe) end-to-end.

---

## f) NEXT 50 THINGS TO GET DONE

### High priority (false positive vectors)

1. Migrate F015 to `coachingScopes` — per-module query registration count
2. Migrate F016 to `coachingScopes` — per-module aggregate count
3. Migrate F018 to `coachingScopes` — per-module FilterOn detection
4. Migrate F019 to `coachingScopes` — per-module SortOn detection
5. Migrate F020 to `coachingScopes` — per-module CounterIncrement detection
6. Migrate F021 to `coachingScopes` — per-module pagination detection
7. Audit F008 `eventCount(ctx)` — should it count per-module?
8. Audit F010 `hasTraversalPatterns(ctx)` — workspace-global AST scan
9. Audit F011 `firstManualFilterPos(ctx)` — workspace-global AST scan
10. Audit F006 `hasPIIInEventPayloads(ctx)` — workspace-global AST scan

### Test coverage

11. Add per-module test for F004 (Prometheus coaching, library suppression)
12. Add per-module test for F007 (idempotency coaching, library suppression)
13. Add per-module test for F009 (scheduling coaching, library suppression)
14. Add per-module test for F012 (deriver coaching, library suppression)
15. Add per-module test for F017 (dedup coaching, library suppression)
16. Add per-module test for F023 (filter pushdown, store isolation)
17. Add per-module test for F024 (pagination pushdown, store isolation)
18. Add per-module test for F025 (counter ADT, store isolation)
19. Add per-module test for F026 (prefetch, metaengine module isolation)
20. Add per-module test for F027 (OTel init, library suppression)
21. Add per-module test for F028 (slog init, library suppression)
22. Add per-module test for F029 (tracing middleware, library suppression)
23. Add per-module test for B030 (circuit breaker, library suppression)
24. Add integration test: real `go.work` with 2+ `go.mod` files
25. Add regression test replicating the cqrs-htmx 19-module scenario

### Doctor and diagnostics

26. Update `doctor` command to show per-module feature profiles
27. Add `--modules` flag to doctor showing all detected modules + their profiles
28. Show which module a finding belongs to in the output grouping
29. Add per-module profile to `explain` command output

### Cleanup and compliance

30. Run `nix fmt` on all new/modified files
31. Run `nix run .#verify` (full verification gate)
32. Update `cmd/cqrs-lint/CHANGELOG.md` with this fix
33. Run API stability golden regen (`cmd/api-stability`)
34. Run library self-lint test (`library_self_lint_test.go`)
35. Run doc-check on any skill reference updates
36. Verify the committed 374-line file is replaced by the split version

### Architecture improvements

37. Consider a `moduleAwareRule` helper that wraps `coachingScopes` boilerplate
38. Extract a `coachingRule(name, eval)` function to reduce per-rule loop boilerplate
39. Consider per-module registry (not just per-module profile) — events/commands/queries
    are currently workspace-global in `ctx.Registry`
40. Add `ctx.FilesForModule(dir)` helper for rules that need direct file access per module
41. Consider `ctx.GoFilesByModule` map pre-computed in BuildContext

### Other feedback items in the same batch

42. Review Issue 3 from cqrs-htmx feedback: `library` preset doesn't go far enough
43. Consider `library-framework` preset disabling F002/F006/F009/F010/F011/S002/S003/S007
44. Review browser-history feedback (2026-08-11) for related suppression drift issues
45. Review file-renamer feedback (2026-08-09) for circuit breaker / DLQ extraction patterns

### Documentation

46. Document per-module feature profile behavior in the skill references
47. Add consumer-facing docs on multi-module workspace linting
48. Add troubleshooting: "why is this rule firing/not firing in my sub-module?"
49. Update IMPROVEMENT_IDEAS.md with the per-module coaching scope pattern
50. Move the feedback file from `docs/feedback/new/` to `docs/feedback/reviewed/`

---

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

### 1. Should F015/F016 (count-based coaching) be per-module or workspace-global?

F015 fires when a project has 3+ query registrations but no metaengine. In cqrs-htmx,
the library module defines queries and the example modules use them. Should F015 count
per-module (library alone has enough queries to trigger coaching) or workspace-wide
(total across all modules)? The "right" answer depends on whether metaengine adoption
is a per-module or project-wide decision. **I cannot determine the intended consumer
behavior from the code alone.**

### 2. Should the `library` preset auto-apply per-module?

The feedback (Issue 3) suggests a `library-framework` preset that disables F002, F006,
F009, F010, F011, S002, S003, S007 for library modules. Should this be automatic (detected
per-module: no `main()` package = library = auto-disable these rules) or explicit (consumer
creates per-module `.cqrs-lint.json`)? **This is a product/design decision, not a technical one.**

### 3. Should per-module profiles support `.cqrs-lint.json` overrides per directory?

The feedback (Issue 2, workaround section) mentions creating per-module `.cqrs-lint.json`
files. Currently, config inheritance (`loadParentRulesConfig`) merges rule disables from
parent directories. Should feature profile overrides (preset, features) also inherit per-
directory? This would let a root `.cqrs-lint.json` set `preset: library` and an
`examples/.cqrs-lint.json` set `preset: production`. **The config loading architecture would
need significant changes, and I don't know if this is the desired direction vs. auto-detection.**
