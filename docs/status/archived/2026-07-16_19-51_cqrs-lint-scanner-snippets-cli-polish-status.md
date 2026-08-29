# cqrs-lint: Scanner Accuracy, Snippets & CLI Polish — Status Report

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../CHANGELOG.md) and
> [TODO_LIST.md](../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

**Date:** 2026-07-16 19:51
**Commit:** `3ff5d255` — "Add source snippets, fix scanner accuracy, and polish CLI output"
**Session goal:** Execute the full Pareto plan from `docs/planning/2026-07-16_18-20_cqrs-lint-scanner-foundation-and-cli-hardening.md`

---

## Executive Summary

All 5 epics from the planning doc were completed: golden file fix, source snippets, scanner accuracy improvement, CLI polish, docs updates, and analyzer tests. 32 files changed, 625 insertions, 81 deletions. Build passes, lint passes (0 issues), 119 tests pass across 11 packages.

**One critical bug was found and fixed during testing:** `capturePayloadType` was matching the aggregate ID argument (arg index 1) instead of the payload argument (arg index 4) in `event.New()` calls. This means S002 (PII encryption check) was silently broken — it was capturing `"id"` as the payload type instead of the actual struct.

---

## a) FULLY DONE

### Golden File Test (P0)

- Fixed trailing-newline flakiness: `TestGoldenFile_JSONOutput` was failing because the golden file kept losing its trailing newline on regeneration
- Root cause: `os.WriteFile` wrote `jsonOut` without a `\n`, but editors/git added one
- Fix: normalize comparison with `strings.TrimRight(expected, "\n")` instead of exact byte match
- Golden file regenerated and stable

### Source Snippets (Epic C — 34 of 61 detectors)

Added `.WithSnippet(ctx.SourceLine(...))` to 34 detector `Build()` calls across 19 files:

| Package      | Files Modified                                                 | Detectors Snippet'd |
| ------------ | -------------------------------------------------------------- | ------------------- |
| api          | a001-a008, a009_a013, a011_a014_a017, a015_a019                | 15                  |
| architecture | e001_e002_e006, e003_e007, rules.go (E004/E005/E006)           | 5                   |
| boilerplate  | b004_b008, b006_b007, b009_b010_b012_b015, b011_b014, rules.go | 11                  |
| consistency  | d003_d005, rules.go                                            | 3                   |
| security     | s002_s003                                                      | 2                   |

The remaining 27 `Build()` calls intentionally lack snippets — they are project-level findings (e.g., "no stack preset", "missing OTel middleware") that point at `go.mod:1:1` because there is no single source line to show.

### Scanner Accuracy (Epic A/B — already committed in prior session, verified this session)

- `EventTypesEmitted` upgraded from `map[string]string` to `map[string]EventEmission{File, Line}` — E004/E006 now point at real emission sites
- `capturePayloadType` bug **fixed**: now correctly starts scanning at arg index 4 (the payload position in `event.New(type, aggID, aggType, version, payload, opts...)`)
- `scanProjectionSubscription` cleaned up: removed unused `pkgName` parameter (lint fix)

### CLI Polish (Epic D)

- `outputFindings` moved from `main.go` → `output.go` (D1)
- `--verbose` flag now works: activates module-grouped output with `=== module/path ===` headers (D2/D3)
- Verbose mode also prints module count, detector count, pre-filter finding count to stderr
- `SourceLine` caching via `sync.Map` (D4): file contents cached after first read, eliminating redundant disk I/O when multiple findings reference the same file
- Fixed lint issues: unchecked type assertion (`forcetypeassert`), nilerr on WalkDir error handling, unused parameter

### Documentation (Epic E)

- README.md: updated 52→61 rules, corrected per-category breakdown (API 16→19, boilerplate 9→15), added 9 missing rule table entries (A011, A014, A017, B006, B007, B009, B010, B012, B015), documented `--color`/`--exclude`/`--verbose` flags
- AGENTS.md: updated rule count 52→61, added flag mentions

### Analyzer Tests (Epic E)

Created `pkg/analyzer/scanner_test.go` — **13 tests** covering:

- `capturePayloadType` correctly captures payload struct, skips non-payloads
- `EventTypesEmitted` stores file+line for both `event.New` and `event.NewEvent`
- `catalog.Event` registration tracking
- `RegisterTyped` command type tracking
- `NewProjection` registration with event types
- `bus.Subscribe` projection inference
- `scanGenDecl` command detection (BasicCommand embed)
- `filterEventPayloads` removes non-payload structs, handles empty payloads
- `SourceLine` valid file read, caching behavior, edge cases (empty filename, line 0, negative, nonexistent, out-of-range)
- `CommandByName` lookup

**Test totals:** 119 tests across 17 test files, 11 packages — all passing.

### Quality Gates

- `nix fmt` — applied, 0 conflicts
- `nix run .#lint` — 0 issues (3 found and fixed: forcetypeassert, nilerr, unused-param)
- `go test -tags "goexperiment.jsonv2" ./... -count=1` — all pass
- BuildFlow pre-commit — passed (78/79 steps, only pre-existing nix-checker warnings)
- File line limits — all files under 350 lines (largest: `correctness/helpers.go` at 333)

---

## b) PARTIALLY DONE

### Snippet Coverage

- **34 of 61** detectors have snippets (56%)
- The remaining 27 are project-level findings pointing at `go.mod:1:1` — snippets add no value there since there's no single source line to show
- However, some of those `go.mod:1:1` findings **could** be improved to point at a more meaningful location (e.g., A016 "missing idempotency middleware" could point at the dispatcher construction site, B013 "missing correlation enricher" could point at the repository construction)

### `go.mod:1:1` Finding Locations

- **15 findings** still point at `go.mod:1:1` as a fallback position
- E004/E006 now use real emission locations when available (fixed this session)
- E006 has a fallback to `go.mod:1:1` when `emission.File == ""`
- The rest are legitimately project-level (no single source location applies)

---

## c) NOT STARTED

These were **not** in the original plan and were deliberately excluded:

1. **Monorepo fixture test** — the plan mentioned it in E3 but there was no time. The monorepo support works (verified against real projects) but has no dedicated test.
2. **`--exclude` filter tests** — the flag works but has no unit tests
3. **SARIF/Markdown output tests** — only JSON has a golden file test
4. **Integration test against the go-cqrs-lite repo itself** — would be a good smoke test
5. **Performance benchmarking** — SourceLine caching was added but not benchmarked
6. **Rule documentation pages** — each rule should have a dedicated doc page with examples
7. **`.cqrs-lint.json` schema validation** — config file is loaded but not validated

---

## d) TOTALLY FUCKED UP (Honest Assessment)

### The `capturePayloadType` Bug — SILENT FAILURE

**This was the worst issue found.** The function was iterating ALL arguments of `event.New()` looking for the first `*ast.CompositeLit` or `*ast.Ident`. Since `event.New("user.created", id, "User", 1, UserCreated{...})` has `id` (an `*ast.Ident`) at arg index 1, the function captured `"id"` as the payload type and returned immediately — never reaching the actual payload at arg index 4.

**Impact:** S002 (PII encryption check) was completely broken. It was checking a type called `"id"` for PII fields instead of the real payload structs like `UserCreated{Name, Email}`. Every S002 finding was either wrong or missing.

**Root cause:** The original code had no concept of argument position — it treated all args equally.

**Fix:** Start scanning at arg index 4 (the payload position in the `event.New` signature).

**How it slipped through:** There were zero analyzer tests before this session. The bug was only caught when I wrote `TestScanCallExpr_EventPayloadCapture` and it failed.

### What I Should Have Done Better

1. **Should have tested the scanner from day one** — the `capturePayloadType` bug existed since commit `055ba7d1` and was never caught because `pkg/analyzer/` had `[no test files]`
2. **Should have noticed the snippet count discrepancy earlier** — I added snippets to 34 detectors but 27 remain, and I didn't clearly communicate which were intentionally skipped vs. forgotten
3. **Should have run the linter BEFORE committing** — found 3 lint issues post-build that required a follow-up fix cycle
4. **The golden file fix was sloppy** — I initially just regenerated it (removing the trailing newline), then had to fix the comparison logic. Should have fixed the root cause first.

---

## e) WHAT WE SHOULD IMPROVE

### Scanner Accuracy (High Priority)

1. **S002 false-negative risk**: If `event.New()` uses a variable reference instead of a composite literal (e.g., `payload := UserCreated{}; event.New(..., payload)`), `capturePayloadType` captures the variable name `"payload"` not the type `"UserCreated"`. Need type info resolution.
2. **S003 detection is fragile**: Uses string matching on method names (`Save`, `AppendBatch`, `Publish`) with package name heuristics. Will produce false negatives on non-standard naming.
3. **Projection scanning is naive**: `scanProjectionSubscription` captures ANY `Subscribe()` call without verifying the receiver type is an event bus.
4. **No type info in scanner**: The scanner works purely on AST patterns. Adding `types.Info` would eliminate most false positives but requires the full package loading path.
5. **`EventTypesEmitted` only tracks the first emission site** — if the same event type is emitted from multiple files, only the last one wins (map overwrite).

### Architecture (Medium Priority)

6. **`correctness/helpers.go` at 333 lines** — approaching the 350-line CI limit. Should be split.
7. **`b009_b010_b012_b015.go` at 315 lines** — 4 detectors in one file. Should be split into separate files.
8. **Duplicate helper functions**: `extractJSONTag` and `hasLower` appear in both `api/helpers.go` and `correctness/helpers.go`. Should be consolidated.
9. **No `internal/` package boundary** — scanner internals are exported from `pkg/analyzer` and could be depended on by rules in ways that create coupling.

### Testing (Medium Priority)

10. **No test for the `--verbose` module grouping** — the feature was added but never tested end-to-end.
11. **No test for `--color` modes** — auto/always/never logic is untested.
12. **No test for `--exclude` filter** — path exclusion is untested.
13. **No test for SARIF output** — only JSON has a golden file.
14. **No monorepo fixture test** — monorepo support works but has no regression test.
15. **Rule tests don't verify snippets** — tests check finding count/message but not that `.Snippet` is populated.
16. **No benchmark for SourceLine caching** — the cache was added but performance impact is unmeasured.

### CLI/UX (Low Priority)

17. **`--health-score` prints after findings in text mode** — in JSON mode it's not included at all.
18. **No `--rules` filter for specific rule IDs in the `rules` subcommand** — can't search/filter the rules table.
19. **No exit code documentation** — non-zero on error-severity findings, but this isn't documented in `--help`.
20. **Config file not validated** — `.cqrs-lint.json` keys are silently ignored if misspelled.

### Observability (Low Priority)

21. **No timing per detector** — can't tell which rules are slow.
22. **No progress indicator** — large monorepos scan silently for seconds.
23. **No `--debug` flag** — can't inspect scanner registry state.

---

## f) Up to 50 Things to Get Done Next

### P0 — Correctness & Accuracy

1. Fix S002 false-negative: resolve payload type from `*ast.Ident` variable references using `types.Info`
2. Fix S003 false-negative: verify receiver type of `Save()`/`Publish()` calls using type info
3. Fix projection subscription scanning: verify `bus.Subscribe` receiver is an event bus, not a generic subscriber
4. Add `EventTypesEmitted` multi-site tracking (slice of emissions, not single map value)
5. Add type-info-backed command detection (not just `BasicCommand` string matching)

### P1 — Testing

6. Add `--verbose` module grouping integration test
7. Add `--color` mode unit tests (auto/always/never, TTY detection)
8. Add `--exclude` filter unit tests
9. Add SARIF output golden file test
10. Add monorepo fixture test (multi-module testdata)
11. Add snippet presence assertions to existing rule tests
12. Add SourceLine cache benchmark (before/after allocation comparison)
13. Add `filterEventPayloads` edge case tests (duplicate payloads, nil events)
14. Add `scanProjectionRegistration` test for composite lit event type lists
15. Add integration test: run cqrs-lint on `example/taskmanager/`

### P2 — Finding Location Quality

16. Improve A009 (missing stack preset): point at first manual store construction
17. Improve A016 (missing idempotency): point at dispatcher `.Use()` chain
18. Improve A018 (no actual event sourcing): point at first import of event package
19. Improve A019 (vendored cqrs): point at vendor directory
20. Improve B013 (missing correlation): point at repository construction
21. Improve B014 (missing OTel): point at bus/dispatcher construction
22. Improve B015 (missing test utilities): point at test file directory
23. Improve D001 (inconsistent naming): point at first event type string
24. Improve D003 (inconsistent logging): point at first log import
25. Improve D004 (inconsistent JSON casing): point at first struct with JSON tags
26. Improve E001 (layer violation): point at the import statement, not go.mod
27. Improve E002 (circular dependency): point at the import causing the cycle
28. Improve E003 (missing module boundary): point at the first CQRS construct

### P3 — Architecture & Code Quality

29. Split `correctness/helpers.go` (333 lines → 2 files)
30. Split `boilerplate/b009_b010_b012_b015.go` (315 lines → 4 files)
31. Consolidate duplicate `extractJSONTag`/`hasLower` helpers into shared package
32. Extract common finding-builder pattern into a helper (reduce boilerplate in every detector)
33. Add `internal/` package boundary for scanner internals
34. Consider a `findingBuilder` fluent helper that auto-adds snippet from position

### P4 — CLI/UX

35. Add `--debug` flag to dump scanner registry state
36. Add per-detector timing in `--verbose` mode
37. Add progress indicator for large monorepos
38. Add `--rules` search/filter in the `rules` subcommand
39. Document exit codes in `--help` output
40. Validate `.cqrs-lint.json` keys and warn on unknown fields
41. Add `--ci` mode (auto-detect CI environment, set color=never, format=sarif)
42. Add `--baseline` flag to only report new findings since a reference run

### P5 — Documentation

43. Add per-rule documentation pages with fix examples
44. Add a "Getting Started" guide for new users
45. Add a "Rule Suppression" guide (`//cqrs-lint:disable`)
46. Add architecture documentation for the scanner pipeline
47. Add a comparison table vs. other Go linters

### P6 — Advanced Features

48. Add `--fix-dry-run` with diff output (show what would change)
49. Add auto-fix support for more rules (currently only C001 has a fix)
50. Add `cqrs-lint init` wizard that creates `.cqrs-lint.json` interactively

---

## g) Top 2 Questions I Cannot Answer Myself

### 1. Should the 27 remaining `go.mod:1:1` findings be improved to point at real source locations?

Many of these are "absence" findings (e.g., A016 "missing idempotency middleware", B014 "missing OTel middleware"). There is no single source line to point at — the finding fires because something is **missing**. Options:

- **(a)** Keep `go.mod:1:1` — it's honest, there's no specific location
- **(b)** Point at the most relevant construction site (e.g., the dispatcher setup function) — more useful but requires scanning for where the dispatcher is created
- **(c)** Point at `main.go` or the package's init function — a reasonable "you should add it here" location

I can't decide this without knowing what feels most useful to consumers.

### 2. Should the scanner gain `types.Info` support for accurate type resolution?

Currently the scanner works purely on AST patterns (string matching on identifiers, heuristics). This causes false negatives (S002 misses payloads passed as variables, S003 misses non-standard method names). Adding `types.Info` would fix these but:

- Requires the full `packages.Load` path (already used in `BuildContext` but not in `BuildContextFromSource`)
- Would make `BuildContextFromSource` tests harder to write (need real type checking)
- Adds complexity to the scanner

Is the accuracy gain worth the complexity cost, or should we keep the AST-only approach and accept the false-negative risk?
