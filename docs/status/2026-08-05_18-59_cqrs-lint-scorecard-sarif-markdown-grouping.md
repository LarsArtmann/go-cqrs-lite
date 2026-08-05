# cqrs-lint Session Status Report

**Date:** 2026-08-05 18:59
**Session scope:** cqrs-lint scorecard SARIF output + markdown grouping + backlog triage verification
**Pre-existing breakage:** `feature_detect.go:204` — `isHTTPFrameworkImport` (gopls reports undefined; `go build` succeeds — stale LSP cache, not a real build error)

---

## What Was Done This Session

### New Code Shipped

1. **Scorecard SARIF output** (`scorecard_render.go:196-312`)
   - `cqrs-lint scorecard --format sarif` now emits SARIF 2.1.0
   - Missing modules → info-level results with `ruleId: scorecard/missing-module`
   - Adoption summary (coverage %, grade, used/missing counts) in `run.properties`
   - Rule metadata (driver rules, informationUri) included for SARIF compliance
   - 5 tests: `TestRenderSARIF_Valid`, `TestRenderSARIF_HasSummary`, `TestRenderSARIF_MissingModulesAsResults`, `TestRenderSARIF_NoMissingEmptyResults`, `TestRenderScorecard_SARIFDispatch`
   - Help text updated in `scorecard_command.go` and `README.md`

2. **Aggregate/module grouping in markdown output** (`output.go:417-435`)
   - `--group-by aggregate --format markdown` renders groups as `## GroupName (N)` sections
   - `--group-by module --format markdown` same with module directory names
   - Uses existing `groupFindingsByAggregate` / `groupFindingsByModule` helpers
   - 3 tests: `TestPrintMarkdownGrouped_AggregateMode`, `TestPrintMarkdownGrouped_ModuleMode`, `TestPrintMarkdownGrouped_DefaultMode`
   - Closes IMPROVEMENT_IDEAS.md item 195 (markdown half; SARIF `logicalLocations` half still pending)

### Verification Done

3. **Per-module migration status verified** — the paste text claimed A015, B014, E008/E011, A009/A013 still needed migration. This was **stale** — all are already migrated to `ctx.ProfileForFile(path)`. Confirmed via grep: the only remaining `ctx.FeatureProfile` direct usages are in F-series adoption rules (5 files, 15 usages), which are intentionally project-level per the design ("F-series rules are intentionally project-level — they coach the whole project").

4. **L1.5 domain severity calibration verified done** — `DomainKind` enum, `DetectFeatures` auto-detection, `applyDomainBias` wired into `run.go:341`, 7 tests all passing in `domain_bias_test.go`.

### Docs Updated

5. `IMPROVEMENT_IDEAS.md` item 195 marked done (markdown), SARIF half noted as pending.
6. `CHANGELOG.md` — two new entries added under [Unreleased] → Added.
7. `README.md` — scorecard SARIF example added to quick-start block.

### Test Results

- **Build:** `go build -tags "goexperiment.jsonv2" ./...` — PASS
- **Vet:** `go vet -tags "goexperiment.jsonv2" ./...` — PASS
- **Tests (race):** all 16 packages pass with `-race`
- **Tests (fast):** all 16 packages pass

---

## a) FULLY DONE (this session)

| Item | Status | Evidence |
| ---- | ------ | -------- |
| Scorecard SARIF output | ✅ | `scorecard_render.go`, 5 tests, help text, README |
| Markdown aggregate/module grouping | ✅ | `output.go:417-435`, 3 tests |
| Scorecard format help text (text, json, markdown, sarif) | ✅ | `scorecard_command.go:20` |
| CHANGELOG entries | ✅ | `CHANGELOG.md` [Unreleased] |
| IMPROVEMENT_IDEAS.md item 195 updated | ✅ | markdown marked done |
| Full suite build+vet+test+race | ✅ | all 16 packages green |

## b) PARTIALLY DONE

| Item | What's done | What's missing |
| ---- | ----------- | -------------- |
| IMPROVEMENT_IDEAS.md item 195 | Markdown grouping shipped | SARIF `logicalLocations` grouping not done — the findings-level SARIF output (via `go-finding`) doesn't support grouping. Would require extending the `finding.Finding` SARIF renderer in `go-finding` to accept logicalLocations, or post-processing the SARIF JSON. |
| Scorecard SARIF for CI pipelines | Format + data emitted | No GitHub Actions workflow example showing `cqrs-lint scorecard --format sarif` + `upload-sarif` action. The README has the lint-level SARIF example but not the scorecard one. |

## c) NOT STARTED (from paste backlog — confirmed not done)

| Item | Notes |
| ---- | ----- |
| Publish cqrs-lint v4.4.0 | Blocked on user approval. Post-v4.3.0 work is unreleased. |
| Run cqrs-lint against real consumer projects | Blocked on access to consumer repos. |
| L1.15: CI self-lint job | `--self-lint` auto-detection works, but no GitHub Actions step gates it. |
| L1.19: Feature adoption scorecard | `--scorecard` and `scorecard` subcommand exist. But L1.19 in the Pareto plan is listed as Open — needs verification if this is a different scope. |
| L1.20: Grouped output by aggregate/domain | Text + markdown done. The Pareto item may have predated the text implementation. |
| L1.23: Parallel rule safety + benchmark suite | Open. |
| L1.30: Orphaned event types (extend E006 for adapters) | Open. |
| L1.31: Orphaned commands detection | Open. |
| L1.45: Shared mutable state in event handler (extend A015) | Open. |
| L1.47-L1.51: New rule categories (DOC/OBS/RES/DI) | Ambitious, all open. |
| F-series migration to per-module | Intentionally not done — F-series coaches the whole project. |

## d) TOTALLY FUCKED UP

| Item | Impact | Fix |
| ---- | ------ | --- |
| **Pre-existing build error in `feature_detect.go:204`** | gopls reports `isHTTPFrameworkImport` undefined. The function exists at line 496. `go build` succeeds — this is a stale gopls LSP cache issue, not a real error. | `lsp_restart gopls` or ignore — the build is authoritative. **Not caused by this session.** |
| **Nothing broken by this session** | All changes build, vet, and pass tests with `-race`. | N/A |

## e) WHAT WE SHOULD IMPROVE

### Process Issues

1. **The paste text was stale** — it listed A015, B014, E008/E011, A009/A013 as needing per-module migration, but all were already migrated. Time was spent verifying this instead of building. The lesson: **status reports from prior sessions are point-in-time, not living documents** (AGENTS.md already says this, but the paste reinforced the trap).

2. **No SARIF golden test** — the scorecard SARIF output has structural tests (valid JSON, has summary, has results) but no golden file like `testdata/sarif_output.json` for the findings-level SARIF. A golden would catch structural drift.

3. **Markdown grouping test doesn't verify markdown table content** — `TestPrintMarkdownGrouped_AggregateMode` checks headers exist and ordering, but doesn't verify that the findings within each group are actually rendered as markdown tables. The test relies on `finding.FormatMarkdown` which is tested upstream.

### Code Quality Observations

4. **SARIF types are inline** — the `sarifReport`, `sarifRun`, `sarifResult` etc. types are defined in `scorecard_render.go` but the findings-level SARIF output uses `go-finding`'s `report.WriteSARIF`. Two parallel SARIF type hierarchies. If `go-finding` ever exposes its SARIF types, the scorecard should reuse them.

5. **`renderScorecard` error return is inconsistent** — markdown returns `(string, nil)`, SARIF returns `(string, error)`, JSON returns `(string, error)`, text returns `(string, nil)`. The text and markdown paths can't error but the signature forces nil. Not a bug, just noise.

6. **Scorecard SARIF doesn't include Used modules** — only Missing modules are emitted as results. Used modules are implicitly represented by the coverage percentage in `run.properties`. This is a design choice (SARIF `results` are for things that need attention), but a CI dashboard might want the full list.

### Architecture Observations

7. **`feature_detect.go` is 503 lines** — exceeds the 350-line CI limit if enforced on the analyzer package. The file contains import scanning (180 lines) + AST scanning (170 lines) + HTTP framework detection (20 lines) + helper functions (130 lines). Should be split into `feature_detect_imports.go` and `feature_detect_ast.go`.

8. **`output.go` is 437 lines** — approaching the 350-line limit. The grouping functions (`groupFindingsByModule`, `groupFindingsByAggregate`, `printMarkdownGrouped`, `printFindingsGrouped`, `printFindingsByAggregate`) could be extracted into `output_grouping.go`.

## f) Up to 50 Things We Should Get Done Next

### High Impact (Pareto top 20%)

1. **Publish cqrs-lint v4.4.0** — tag and release all post-v4.3.0 work
2. **Run cqrs-lint against real consumer projects** — validate false-positive rates
3. **Add GitHub Actions workflow for scorecard SARIF** — `cqrs-lint scorecard --format sarif` + `upload-sarif`
4. **Add SARIF golden test for scorecard** — `testdata/scorecard_sarif.json` with structural comparison
5. **L1.5 already done** — update Pareto plan status column to mark it ✅
6. **Update Pareto plan** — mark L1.19 (scorecard) and L1.20 (group-by) as done if applicable
7. **Split `feature_detect.go` (503→2 files <350)** — extract AST detection
8. **Split `output.go` (437→2 files <350)** — extract grouping functions
9. **CI self-lint job (L1.15)** — gate regressions on self-lint baseline
10. **Verify `isHTTPFrameworkImport` build** — confirm it's a gopls cache issue only

### Medium Impact

11. **SARIF `logicalLocations` for aggregate grouping** — extend finding-level SARIF output
12. **Scorecard Used modules in SARIF** — optionally emit as `pass` results
13. **Color-code markdown aggregate headers by severity** — item 196
14. **Severity sub-totals in group headers** — `User (5: 2 errors, 3 warnings)` — item 196
15. **`--aggregate-filter` flag** — show only one aggregate's findings — item 196
16. **L1.30: Orphaned event types (extend E006)** — detect events no adapter consumes
17. **L1.31: Orphaned commands** — detect commands no HTTP handler exposes
18. **L1.45: Shared mutable state in handlers** — extend A015
19. **L1.23: Parallel rule safety benchmark** — verify no data races in detectors
20. **Detector-level aggregate stamping** — item 193, top 5 detectors
21. **`--group-by` in config file** — item 194
22. **F-series per-module evaluation** — if false-positive rates warrant it
23. **L1.47: DOC-series rules** — missing docs, stale catalog, undocumented events
24. **L1.48: OBS-series rules** — tracing spans, metrics, structured logging
25. **L1.49: RES-series rules** — retry, circuit breaker, DLQ, graceful shutdown
26. **L1.50: DI-series rules** — optimistic concurrency, idempotency, tx consistency

### Quality / DX

27. **Scorecard `--format sarif` with `--scorecard-threshold`** — verify threshold gate works with SARIF output (currently only text)
28. **`explain` subcommand** — add scorecard SARIF format to the documented formats
29. **Benchmark scorecard rendering** — verify SARIF doesn't regress on large catalogs
30. **Consolidate SARIF type definitions** — if go-finding exposes them
31. **Test scorecard with empty catalog** — edge case
32. **Test scorecard with all-missing** — edge case (0% coverage)
33. **Test scorecard with all-used** — edge case (100% coverage, no results in SARIF)
34. **Integration test: full `cqrs-lint scorecard --format sarif` CLI run** — E2E
35. **Integration test: `cqrs-lint --format markdown --group-by aggregate`** — E2E
36. **Add scorecard SARIF to `.github/workflows/cqrs-lint.yml`** — CI example
37. **Update cqrs-lint VALIDATION_REPORT.md** — if it tracks output formats
38. **Verify `nix run .#build` compiles cqrs-lint** — workspace-level check
39. **Verify `nix run .#lint` passes on changed files** — golines/gofumpt
40. **Run `cmd/doc-check`** — verify README import paths still valid
41. **Update api-stability golden** — if scorecard SARIF types are exported (they're not — unexported)
42. **Test markdown grouping with empty findings** — edge case
43. **Test markdown grouping with single finding** — edge case
44. **Test markdown grouping with uncategorized findings** — no aggregate tag
45. **Consolidate `renderScorecard` error signatures** — text/markdown should return `(string, error)` or make them all `string`

### Documentation

46. **Document scorecard SARIF schema in README** — what fields, where to find coverage %
47. **Add scorecard SARIF example output** — sample JSON in README or docs/
48. **Update IMPROVEMENT_IDEAS.md summary** — rule count, open count, status table
49. **Update Pareto plan** — mark completed items, recalculate remaining count
50. **Cross-reference CHANGELOG entries** — ensure [Unreleased] accumulates correctly

---

## g) Questions I Cannot Answer Myself

1. **Should scorecard SARIF emit Used modules as well?** Currently only Missing modules appear as results (info level). An alternative design emits Used as `pass`-level results (SARIF supports `level: "none"` for informational pass-through). This would let a CI dashboard show the full adoption picture from one SARIF file, but it doubles the result count. I can't decide whether the noise tradeoff is worth it without knowing how you consume scorecard output in CI.

2. **Should I publish cqrs-lint v4.4.0 now?** The paste says it's blocked on user approval. The post-v4.3.0 work (init SHOWSTOPPER fix, C038-C040, scorecard + markdown, group-by aggregate, per-module detection, JSONC config loader, explain command, doctor overhaul, E006 fold-aware) plus this session's scorecard SARIF + markdown grouping represent substantial shipped but unreleased value. Should I proceed with tagging, or do you want to bundle more work first?

3. **Is the `feature_detect.go:204` gopls error real or cache?** The function `isHTTPFrameworkImport` exists at line 496 and `go build` succeeds, but gopls consistently reports it as undefined. I believe it's a stale LSP snapshot (AGENTS.md documents this pattern), but I can't restart gopls from here. Should I `lsp_restart gopls` or is there something else going on?
