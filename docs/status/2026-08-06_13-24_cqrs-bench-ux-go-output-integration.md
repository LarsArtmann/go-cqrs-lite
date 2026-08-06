# Session: cqrs-bench UX/UI Overhaul + go-output Integration

**Date:** 2026-08-06 13:24
**Session goal:** Improve cqrs-bench usage design/UX, review go-output integration possibilities

---

## What Was Done

### a) FULLY DONE

1. **Styled terminal tables via go-output** — `cmd/cqrs-bench/render.go` (new file) converts benchkit `Result` structs into `output.Table` and renders via `go-output/table` (lipgloss rounded-border tables with full column names like "GC Max Pause" instead of "GCMaxPau"). Works for `compare`, `sweep`, and `run` subcommands.

2. **Winner summary line** — Compare output now ends with "Best — writes: X (V) | reads: X (V) | allocs: X (V) | heap: X (V) | GC pause: X (V)" — auto-detects the best backend per key metric.

3. **`--format auto` (new default)** — `resolveFormat()` picks `table` when stdout is a TTY, `text` when piped. Existing scripts and CI that capture stdout automatically get plain text. No breaking change.

4. **New formats: `csv`, `tsv`, `table`** — All subcommands (run, compare, sweep, soak) now support CSV and TSV export. Previously only `text`, `json`, `markdown`, `benchstat`, `manifest` existed; sweep had no markdown/csv. Now all formats work across all subcommands.

5. **Halved progress noise** — `benchkit/progress.go` `beginPhase()` no longer prints a "started" line. Each phase shows a single "done (Xms)" line instead of two lines (started + done). Cuts a 10-phase compare run from 20 lines to 10.

6. **Run summary table** — `run --format table` shows a clean 2-column Metric/Value table (Backend, Profile, Codec, Events, Write P50/P99, Throughput, Load, GC, Allocs, Heap) instead of the 100-line detailed report (still available via `--format text`).

7. **Help text updated** — `main.go` longDesc now lists all 9 formats with descriptions. Flags updated to show `auto, table, text, json, csv, tsv, markdown, benchstat, manifest`.

8. **go-output bug fix** — Fixed `markdown/markdown.go` `writeHeader()` double-pipe `||` bug (wrote `|` twice at line start). The fix removes the duplicate `b.WriteString("|")`. Worked around in cqrs-bench with `strings.TrimPrefix(rendered, "|")` until a new go-output version is tagged.

9. **Test fixes** — `main_test.go` assertions updated to be spacing-tolerant (go-output pads cells for alignment; old tests checked `| memory |` with exact spacing).

10. **Dependency promotion** — `go-output`, `go-output/delimited`, `go-output/markdown`, `go-output/table`, `go-humanize` promoted from `// indirect` to direct require in `cmd/cqrs-bench/go.mod`.

### b) PARTIALLY DONE

1. **go-output markdown `||` workaround** — Fixed at the source in `/home/lars/projects/go-output/markdown/markdown.go`, but cqrs-bench consumes the published `v0.35.0` tag from the module cache. Added a `strings.TrimPrefix` workaround in `render.go`. The workaround becomes a no-op once a version > v0.35.0 is tagged and required. **Not committed/tagged in go-output yet.**

2. **Format consistency across subcommands** — Markdown format added to sweep (was missing). TSV added to run and soak. But the `run` subcommand's `table` format shows a summary table, not the full detailed report — some users may want the full report in table form. This is a design choice, not a gap.

### c) NOT STARTED

1. **go-output NOM TUI integration** — go-output has a full NOM-style real-time progress visualization (`nom/` package with dependency trees, activity counts, timing estimates). Could replace the current stderr progress lines with a rich TUI. Requires significant integration work (subscriber pattern, event mapping).

2. **go-output tree format for soak reports** — Soak reports have a hierarchical structure (drift summary → per-iteration samples → phase breakdown) that maps naturally to `output.TreeBuilder`. Could render as an ASCII tree.

3. **go-output graph/D2 format for sweep results** — Sweep results across parameter values could be visualized as a D2/Mermaid graph showing scaling curves.

4. **Color-coded comparison highlighting** — Could highlight the best value in each column green and worst red using `go-output/table` `WithFooterStyle` or custom StyleFunc.

5. **`--format html`** — go-output has an HTML table renderer. Could generate self-contained HTML reports for browser viewing.

6. **`--format yaml`** — go-output serialization module supports YAML. Could add as a machine-readable alternative to JSON.

### d) TOTALLY FUCKED UP

Nothing. All changes build, vet, and pass tests.

---

## What We Should Improve

### Immediate (this session's work)

1. **No test for the new `render.go` functions** — `renderComparison`, `renderSweep`, `renderRunResult`, `renderSoakResult` have no unit tests. Only CLI integration tests cover them indirectly. Should add table-driven tests for the table builders (`buildComparisonTable`, `buildSweepTable`, `buildRunSummaryTable`).

2. **Duplicated formatting helpers** — `render.go` has `fmtDur`, `fmtBytes`, `fmtFloat`, `fmtInt`, `roundDur` that duplicate `benchkit/report_format.go`'s `roundDuration`, `formatBytes`, `formatFloat`, `formatInt`. These should be exported from benchkit or extracted to a shared formatting package. Currently the render.go copies exist because benchkit's versions are unexported.

3. **`comparisonWinnerSummary` uses float64 conversion of durations** — `float64(r.WriteLatency.P50)` works but is semantically odd. A cleaner approach would compare `time.Duration` directly.

4. **The markdown `||` workaround comment says "Remove when a version > v0.35.0 is required"** — But there's no tracking issue or TODO to actually do this. Will be forgotten.

5. **`resolveFormat` relies on `output.ColorModeAuto.ShouldColor()`** — This checks `os.Stdout.Fd()`, which is correct for the common case but could be wrong if the user redirects stdout to a file but still wants table format. An explicit `--format table` always works though.

### Broader UX improvements (not started)

6. **No `--quiet` flag** — For CI/scripting, a `--quiet` that suppresses all progress output (stderr) and only emits the result would be useful.

7. **No `--watch` mode** — For iterative development, a live-refresh mode that re-runs benchmarks on file changes.

8. **No baseline comparison** — `--baseline <file>` that loads a previous JSON result and shows delta/regression alongside current results.

9. **Sweep only supports 1 parameter** — Can't sweep two parameters (e.g., workers × batchSize matrix).

10. **No `list` command** — Can't list available profiles, backends, or codecs without reading help text.

---

## Up to 50 Things We Should Get Done Next

### Rendering & Format (1-10)

1. Export formatting helpers from benchkit (`roundDuration`, `formatBytes`, etc.) to eliminate duplication in render.go
2. Add unit tests for `buildComparisonTable`, `buildSweepTable`, `buildRunSummaryTable`
3. Add color-coded best/worst highlighting in comparison tables (green best, red worst)
4. Tag go-output v0.36.0 with the markdown `||` fix and remove the TrimPrefix workaround
5. Add `--format html` for browser-viewable reports (go-output/markup)
6. Add `--format yaml` as alternative to JSON (go-output/serialization)
7. Add tree-format visualization for soak drift reports (go-output/tree)
8. Add D2/Mermaid graph output for sweep scaling curves
9. Add a `--width` flag to control table column width truncation
10. Add ANSI color legend footer when using `--format table` with color

### CLI UX (11-20)

11. Add `--quiet` flag (suppress progress, result only)
12. Add `--baseline <file>` for regression comparison against saved JSON
13. Add `cqrs-bench list profiles` / `list backends` subcommand
14. Add `--watch` mode for iterative re-runs on file changes
15. Add multi-parameter sweep (workers × batchSize matrix)
16. Add `--filter` flag to show only specific metrics in table/text output
17. Add `--sort-by` flag for comparison tables (sort by WriteP50, Heap, etc.)
18. Add `--threshold` flag that exits non-zero if metrics exceed a threshold
19. Add shell completion for format/profile/backend flag values
20. Add `cqrs-bench version --verbose` with go-output, benchkit, go versions

### NOM TUI Integration (21-28)

21. Prototype go-output NOM TUI for `run` command (real-time progress visualization)
22. Map benchkit phases to NOM activities (write phase, read phase, etc.)
23. Show dependency tree (write → projection → query as a DAG)
24. Add timing estimates based on profile event count
25. Add inline renderer fallback for non-TTY (current text progress)
26. Add NOM-style activity counts (events written, reads completed)
27. Show GC pressure as a NOM activity bar
28. Integrate with go-output `tui/` Bubble Tea program for full TUI mode

### Progress & Reporting (29-35)

29. Add `--progress-format` flag (text/json/nom) to control progress output independently of result format
30. Add progress percentage to each phase (not just "started/done")
31. Add ETA based on profile event count and observed throughput
32. Add a summary "Total time: Xs (write Ys, read Zs, projection Ws)" at the end
33. Add `--output-multiple` flag to write result in multiple formats simultaneously
34. Add markdown report with embedded comparison table for README insertion
35. Add a `--report` flag that generates a standalone HTML report with charts

### Testing & Quality (36-42)

36. Add golden-file tests for each format (text, table, csv, tsv, markdown)
37. Add test for `resolveFormat` auto-detect logic (TTY vs pipe)
38. Add test for `comparisonWinnerSummary` with edge cases (all failed, single backend, ties)
39. Add test for the markdown `||` workaround removal condition
40. Run `nix run .#verify` to confirm the full gate passes with all changes
41. Add `cqrs-bench` to the api-stability golden (if exported symbols changed)
42. Add benchkit test for the progress noise reduction (verify "started" is gone)

### Documentation (43-46)

43. Update `cmd/cqrs-bench/README.md` with new format options and examples
44. Update `AGENTS.md` Quick Reference test command if needed
45. Add a "Format Guide" section to cqrs-bench README showing when to use each format
46. Document the go-output integration architecture in render.go header comment

### go-output Ecosystem (47-50)

47. Add go-output to the `go.work` workspace for local development across both repos
48. Consider a shared `formatting` module between benchkit and go-output for duration/bytes formatting
49. Explore go-output `StreamingHTMLRenderer` for large benchmark results
50. Evaluate go-output `daghtml` for interactive HTML DAG visualization of benchmark phases

---

## Questions

1. **Should we add go-output to the `go.work` workspace?** Currently cqrs-bench consumes the published v0.35.0 tag. Adding `../go-output` to `go.work` would let us develop both repos simultaneously and pick up the markdown fix immediately, but adds workspace complexity. The alternative is tagging go-output v0.36.0 first.

2. **Should the `run --format table` show the full detailed report (like `text`) in table form, or keep the current summary table?** The current summary table is clean but omits detailed metrics (ReadAll time, ReadFrom time, MetaEngine breakdown, Recovery, Storage breakdown). A full table version would need multiple sub-tables or a very long single table.

3. **Should we invest in the NOM TUI integration now, or focus on the simpler CLI improvements first?** The NOM TUI is the most visually impressive improvement but requires significant integration work (mapping phases to activities, subscriber pattern). The simpler improvements (color highlighting, baseline comparison, `--quiet`, list command) deliver incremental value faster.
