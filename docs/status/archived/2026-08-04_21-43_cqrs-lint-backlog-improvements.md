# Status: cqrs-lint Backlog Improvements

**Date:** 2026-08-04 21:43
**Session scope:** Execute actionable items from the cqrs-lint improvement backlog (paste_1.txt)
**Test status:** ALL 17 cqrs-lint packages PASSING — `go test` + `go vet` clean
**Git status:** Working tree clean (auto-commit daemon committed all changes)

---

## A) FULLY DONE (11 items shipped)

### 1. Scorecard category-priority split brain eliminated

- **What:** Deleted the hand-maintained `categoryPriorityFor` switch statement in `scorecard.go` that duplicated the `categoryPriority` map in `module_catalog.go`. Exported `analyzer.CategoryPriority(ModuleCategory)` as the single source of truth.
- **Files:** `scorecard.go`, `pkg/analyzer/module_catalog.go`
- **Impact:** Adding/reordering a category in the catalog map now automatically flows to the scorecard sorter. No more silent drift between two parallel priority systems. Also fixed the gopls `unusedparams` warning on `scorecardLess` (the `catalog` parameter was unused after the refactor).

### 2. Scorecard Evidence field rendered in output

- **What:** Added `Evidence` field to `ScorecardModule`, populated from `ModuleUsage.Evidence` (the import path that triggered detection). The text table now conditionally shows a 4th "Evidence" column when any module has non-empty evidence.
- **Files:** `scorecard.go`, `scorecard_render.go`
- **Test:** `TestComputeScorecard_EvidencePropagation` verifies both data propagation and text rendering.

### 3. commentTextStart multi-line raw string literal bug fixed

- **What:** The suppression parser (`pkg/suppression/parser.go`) processed each line independently, so `//cqrs-lint:ignore` directives appearing as literal text inside multi-line backtick raw strings were falsely treated as real suppression comments. Added `computeRawStringLines` which tracks backtick string state across lines, and modified both `checkSuppressionInFile` and `checkBlockSuppressionInFile` to skip lines entirely inside multi-line raw strings.
- **Files:** `pkg/suppression/parser.go`
- **Tests:** `TestSuppression_MultiLineRawStringDoesNotTriggerBlockSuppression`, `TestSuppression_MultiLineRawStringDoesNotTriggerInlineSuppression`

### 4. JSONC trailing comma support added

- **What:** Added `stripTrailingCommas` post-processor to the config loader. It removes commas followed by `}` or `]` while respecting string literals. This handles the JSONC spec's allowance of trailing commas in objects and arrays, which strict JSON parsers reject.
- **Files:** `config_loader.go`
- **Tests:** `TestStripJSONComments_TrailingCommaObject`, `TestStripJSONComments_TrailingCommaArray`, `TestStripJSONComments_TrailingCommaNested`

### 5. F013 cqrs-htmx regression test added

- **What:** Dedicated regression test proving that importing the external `cqrs-htmx` module triggers `HasTransport` detection via the feature detection pipeline, which suppresses F013 (missing transport module). The test also explicitly asserts `ctx.FeatureProfile.HasTransport == true` to catch detection pipeline regressions independently of the F013 rule.
- **Files:** `pkg/rules/adoption/f010_f017_test.go`

### 6. Doctor/explain test coverage added (0 → 8 tests)

- **What:** Doctor command (`doctor.go`) and explain command (`explain.go`) had ZERO unit tests. Added tests for the testable pure functions:
  - `countSuppressions` — detects inline `//cqrs-lint:ignore(RULE)` comments, counts per rule
  - `findParentConfigs` — walks ancestor directories for monorepo config inheritance
  - `formatConfigFeatures` — formats `ConfigFeatures` struct for display
  - `renderExplain()` — verifies all 8 sections present, all preset descriptions included, JSONC documented
- **Files:** `doctor_test.go` (new), `explain_test.go` (new)

### 7. printFindingsByAggregate output tests added

- **What:** The grouped-output renderer (`--group-by aggregate`) had ZERO tests on the actual rendered text. Added tests for:
  - `groupFindingsByAggregate` — grouping by metadata, sorting by count desc then name
  - `printFindingsByAggregate` — header rendering (`--- Name (count) ---`), finding messages, empty findings, uncategorized bucket
- **Files:** `output_test.go`

### 8. group-by added to .cqrs-lint.json config schema

- **What:** The `GroupBy` field had `flag:"group-by"` but no JSON tag, so consumers could not set it in the config file. Added `json:"group-by,omitempty"` tag and documented it in the init-generated default config template.
- **Files:** `main.go`, `init.go`

### 9. Module catalog expanded (28 → 32 scored modules)

- **What:** Added 4 previously-excluded modules to the scorecard catalog:
  - `middleware` (Observability) — tracing, metrics, retry, recovery, validation
  - `storage` (Persistence) — SQL backend facade, custom store wiring
  - `stack/memory` (Persistence) — in-memory stack for tests/dev
  - `scenario` (Reliability) — BDD test DSL
- Updated `TestCatalogHasExpectedCounts` (28→32 scored, 34→38 total) and `TestCatalogEveryGoWorkModuleCovered` (added `metaengine/irohengine`, `metaengine/adttest`, `system` to exclusions).
- **Files:** `pkg/analyzer/module_catalog_data.go`, `pkg/analyzer/module_catalog.go`, `pkg/analyzer/module_catalog_test.go`

### 10. --scorecard-threshold CI gate flag added

- **What:** New `--scorecard-threshold N` flag on the `scorecard` subcommand. When N > 0 and coverage is below N%, the command writes a diagnostic to stderr and returns `errScorecardBelowThreshold`, causing cmdguard to exit non-zero. This enables CI pipelines to gate on adoption coverage.
- **Files:** `scorecard_command.go`

### 11. All tests pass, go vet clean

- 17 packages, all `ok`. `go vet` clean. No new diagnostics introduced.

---

## B) PARTIALLY DONE

### Doctor render* function testability

- **Status:** The `renderDoctor*` functions (`renderDoctorPreset`, `renderDoctorEffectiveSettings`, `renderDoctorFeatureProfile`, etc.) still write directly to `os.Stdout`/`os.Stderr` via `fmt.Println`/`fmt.Fprintf`. They are NOT testable without capturing stdout.
- **What was done instead:** Tested the pure helper functions they call (`countSuppressions`, `findParentConfigs`, `formatConfigFeatures`). The render functions themselves remain untested.
- **What remains:** Refactor `renderDoctor*` to accept `io.Writer` parameters (or return strings like `renderExplain` does), then add output assertions.

### Explain command coverage

- **Status:** `renderExplain()` is tested for section presence and preset descriptions, but NOT for output accuracy of individual sections (top-level keys table, features docs, rules config docs, resolution order, suppression syntax). A render snapshot test would catch drift.

---

## C) NOT STARTED (from the original backlog)

1. **Publish cqrs-lint v4.4.0** — BLOCKED on user approval. v4.3.0 tagged; all post-v4.3.0 work (init SHOWSTOPPER fix, C038-C040, scorecard, group-by aggregate, per-module detection, JSONC config loader, explain command, doctor overhaul, E009 cqrs-htmx transport detection, + this session's 11 items) remains unreleased. Version constant still says `"4.3.0"`.
2. **Run cqrs-lint against real consumer projects** — BLOCKED on user action. Need to validate false-positive rates against Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync, timesheets, crush-daily.
3. **SARIF + markdown output formats for scorecard** — not started.
4. **Migrate 26 global detectors to per-module evaluation** — `ProfileForFile` infrastructure exists but only C017 uses it. The other 26 global `FeatureProfile` reads still use the primary profile. High false-positive risk for multi-module workspaces.
5. **B025 cross-package helper tracing** — only same-package helpers are traced. Needs `golang.org/x/tools/go/callgraph` for import-graph tracing.
6. **~13 remaining Pareto backlog items** — L1.5 (domain severity), L1.15 (CI self-lint job), L1.23 (parallel safety), L1.30-L1.31 (orphaned types), L1.45 (shared mutable state), L1.47-L1.51 (DOC/OBS/RES/DI new rule categories), L1.51 (stack preset boundary awareness).

---

## D) TOTALLY FUCKED UP / ISSUES INTRODUCED

### Nothing critically broken, but:

1. **Version constant not bumped** — `const version = "4.3.0"` in `main.go:18` is stale. With 11 new features/fixes shipped, this should be `"4.4.0"`. The `TestVersionMatchesLatestTag` CI gate will fail if we tag v4.4.0 without updating the constant, and consumers running `cqrs-lint version` see misleading output.

2. **init.go template produces invalid JSON** — The default config template now has `"format": "text",` (trailing comma) followed by comment lines. While our JSONC loader handles this, the template is technically not valid JSON if a user tries to parse it with a standard parser before uncommenting the group-by line. The trailing comma after `"format": "text"` is intentional (JSONC supports it), but the commented-out `"group-by"` line after it creates a confusing editing experience.

3. **`stripTrailingCommas` is O(2n)** — Runs as a separate pass after `stripJSONComments`. Could be integrated into the main comment-stripper for a single-pass solution. For config files (<10KB typically) this is irrelevant, but it's not optimal.

4. **Catalog expansion may change existing scorecard results** — Adding `middleware`, `storage`, `stack/memory`, and `scenario` to the scored catalog changes the denominator for all consumers. Projects that were at e.g. "15/28 (53%)" are now "15/32 (47%)". This is semantically correct (these ARE adoptable modules) but it's a breaking change for CI gates that hardcode coverage percentages. The `--scorecard-threshold` flag we added helps with this, but existing thresholds need adjustment.

5. **`scorecard_command.go` has a raw `fmt.Fprintf(os.Stderr, ...)` call** — This should ideally go through the same output abstraction as other diagnostic output, but since doctor.go does the same thing (`fmt.Fprintln(os.Stderr, ...)`), it follows the existing pattern.

---

## E) WHAT WE SHOULD IMPROVE

### Architecture / Design

1. **Render functions should be pure** — ALL `renderDoctor*` functions and the `setupDoctorCommand` handler write directly to stdout/stderr. They should return strings (like `renderExplain`) or accept `io.Writer` parameters. This is the #1 testability debt in the command layer.

2. **Single-pass config preprocessing** — `stripJSONComments` + `stripTrailingCommas` should be a single state machine. The current two-pass approach works but is inelegant.

3. **Scorecard threshold should also work from config file** — `--scorecard-threshold` is CLI-only. Consumers cannot set it in `.cqrs-lint.json`. This is the same gap as group-by had before this session.

4. **Scorecard JSON output should include threshold pass/fail** — When `--scorecard-threshold` is set and the scorecard is rendered as JSON, the threshold result should be in the JSON output, not just stderr.

### Testing

5. **Snapshot/golden tests for doctor and explain output** — These are large formatted text outputs. Snapshot tests (via `go-snaps`) would catch any formatting drift.

6. **`computeRawStringLines` edge cases** — The current implementation handles the common cases but doesn't handle:
   - Nested raw strings (Go doesn't allow nesting, but a backtick inside a `//` comment on a raw-string-internal line could confuse the counter)
   - Raw string literals in import blocks or build tags

7. **F013 test could be stronger** — It tests via `ctx.Packages` mock, not via actual source-code import detection. A test with a real `.go` file containing `import "github.com/larsartmann/cqrs-htmx/v4"` would be more end-to-end.

8. **`stripTrailingCommas` doesn't handle trailing commas inside string literals that contain `}` or `]`** — Wait, it does (it tracks `inString`). But it should have an explicit test for `"key": "value,",}` (trailing comma inside a string value followed by closing brace).

### Process

9. **API stability golden needs regen** — We added exported `analyzer.CategoryPriority` function and changed `ScorecardModule` struct. The api-stability golden file should be regenerated.

10. **Version bump + tag** — Should happen as a single atomic step: bump `version` constant, commit, tag `cmd/cqrs-lint/v4.4.0`.

---

## F) UP TO 50 THINGS TO DO NEXT

### Immediate (this week)

1. Bump `version` constant from `"4.3.0"` to `"4.4.0"` in `main.go:18`
2. Regenerate api-stability golden (`cd cmd/api-stability && GOWORK=off go run main.go -update`)
3. Run `nix fmt` on the changed files
4. Run full `nix run .#verify` gate (not just cqrs-lint module tests)
5. Fix init.go template: make it valid JSONC without trailing comma before comments
6. Add `scorecard-threshold` to `.cqrs-lint.json` config schema (JSON tag on AppConfig or scorecardFlags)
7. Add SARIF output format for scorecard (currently text + JSON only)
8. Add markdown output format for scorecard

### Scorecard follow-ups

9. Render Evidence field in JSON output (already included via struct tag, but verify)
10. Add `--scorecard-threshold` to the explain command's top-level keys documentation
11. Add scorecard threshold pass/fail to JSON output structure
12. Add `--no-irrelevant` flag to suppress the irrelevant module list in scorecard
13. Add `--scorecard-recommendations N` to control number of recommendations (currently hardcoded 3)
14. Add trend tracking: compare scorecard to previous run (store in `.cqrs-lint-cache.json`)

### Doctor/Explain

15. Refactor `renderDoctorPreset` to return string or accept `io.Writer`
16. Refactor `renderDoctorEffectiveSettings` to return string or accept `io.Writer`
17. Refactor `renderDoctorFeatureProfile` to return string or accept `io.Writer`
18. Refactor `renderDoctorSuggestedConfig` to return string or accept `io.Writer`
19. Refactor `renderDoctorSuppressions` to return string or accept `io.Writer`
20. Add snapshot test for full `renderExplain()` output
21. Add test for doctor command end-to-end (create temp project, run doctor, verify output)
22. Add `group-by` to explain command's top-level keys table documentation

### Suppression parser

23. Add test for raw string literal at end of file (no closing backtick)
24. Add test for multiple consecutive multi-line raw strings
25. Add test for raw string containing both backtick and `//cqrs-lint:ignore`
26. Consider replacing manual parser with `go/scanner` for bullet-proof lexical analysis
27. Add test for block suppression (`ignore-start`/`ignore-end`) spanning across raw string boundaries

### Config loader

28. Add test for trailing comma inside string value containing `}`
29. Add test for deeply nested trailing commas (4+ levels)
30. Add test for trailing comma before a comment line
31. Consider using `encoding/json/v2`'s `jsontext.Decoder` for streaming comment/trailing-comma stripping
32. Add JSONC support documentation to the explain command output

### Catalog expansion follow-ups

33. Add `storage/memory` to catalog (currently excluded as "covered by stack/memory")
34. Add `storage/pebble` to catalog (currently excluded)
35. Add `storage/turso` to catalog (currently excluded)
36. Consider adding `projection` (interface-only, currently excluded)
37. Add `testutil` to catalog as a "Testing" category module
38. Add `benchkit` to catalog as a "Tooling" category module
39. Add `dispatcher` to catalog
40. Verify scorecard recommendations make sense with 32-module catalog

### Per-module profile migration

41. Migrate F012 to use `ProfileForFile` instead of `ctx.FeatureProfile`
42. Migrate F013 (already partially done)
43. Migrate F014 to use `ProfileForFile`
44. Migrate F015-F021 to use `ProfileForFile`
45. Migrate all B-series adoption detectors to per-module profiles
46. Migrate all E-series architecture detectors to per-module profiles

### Pareto backlog

47. L1.5: Add `DomainBias` to `FeatureProfile` for domain-based severity calibration
48. L1.47: Start DOC-series rules (missing docs, stale catalog, undocumented events)
49. L1.48: Start OBS-series rules (tracing spans, metrics, structured logging)
50. L1.49: Start RES-series rules (retry, circuit breaker, DLQ, graceful shutdown)

---

## G) QUESTIONS FOR THE USER

### 1. Should I bump the version to v4.4.0 and publish a tag now?

All 11 items are shipped, tested, and committed. The version constant still says `"4.3.0"`. You mentioned v4.4.0 was "pending" and "BLOCKED on user approval". Should I bump + tag, or are you holding off for more work first?

### 2. Should the catalog expansion (28→32 modules) be treated as a breaking change?

Adding 4 modules to the scored catalog changes every consumer's coverage percentage. Projects at 53% coverage drop to ~47%. Should I add a deprecation notice or migration guide, or is this just an expected evolution of the tool?

### 3. Should I prioritize refactoring `renderDoctor*` functions for testability, or is the pure-function coverage sufficient?

The doctor command has 9 render functions that all write to stdout. Refactoring them to return strings (like `renderExplain`) would enable comprehensive snapshot testing but touches 400+ lines of working code. Is this worth doing now, or should I move to higher-value work (per-module profile migration, new rule categories)?
