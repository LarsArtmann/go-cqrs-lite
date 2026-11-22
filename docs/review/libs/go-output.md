# go-output — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-output` @ 90fb2a6 (latest local tag v0.38.7) · Consumer go-cqrs-lite @ feec9bc74 · Group G4

**Verdict:** UNDERUSED-SEVERE
**Score:** 4.5/11 capabilities fully leveraged = 41% (low)

The two CLI tools that pin go-output directly (cqrs-bench, cqrs-lint) use its
table/delimited/color core exactly as intended — and are version-current at
v0.38.7. But that is where adoption stops. The diagram half of the library
(d2, plantuml, graph, escape) is pinned by nobody, while `catalog/v4`
hand-rolls a **633-line D2 DSL exporter** (`catalog/v4/d2/`) that re-creates
what `go-output/d2` + `go-output/escape` ship; cqrs-bench hand-rolls a
160-line NOM-style progress reporter that `go-output/nom` exists to replace;
and every tool hand-rolls string-constant format vocabularies that the root
module's validated `Format` enum was built for. The fleet's shared
write-once-render-anywhere layer is used for exactly one shape (tables) in
exactly two of seven CLI-bearing modules.

## 1. What the library is

"Write your data once. Render it anywhere" — one library, sixteen formats,
three data shapes (`Table`, `TreeNode`, `GraphNode`), zero heavy dependencies
in the root (only `golang.org/x/term`) (go-output/README.md:136-142). The
root module owns the data model, the CQRS build→freeze→render flow
(`NewTableBuilder().Build()`, `NewGraphBuilder()`), cross-shape projections,
the `Format` enum with `ParseFormat`/`Format.Supports`/`FormatsForShape` for
runtime format selection, and `ColorMode` TTY/`NO_COLOR`/`CI` detection.
Format renderers live in isolated sub-modules: `delimited` (CSV/TSV),
`serialization` (JSON/YAML/TOML/JSONL of table/tree/graph shapes), `markup`
(XML/HTML/AsciiDoc), `table` (lipgloss terminal tables), `markdown`,
`tree` (ASCII box-drawing), `d2` (SQL tables, 20 node shapes, grid layouts),
`graph` (DOT/Mermaid), `plantuml`, `daghtml` (interactive HTML DAG),
`escape` (per-format string escaping incl. `escape.D2`), plus `nom`
(NOM-style real-time progress) and `tui` (go-output/README.md:232-243,
go-output/FEATURES.md:219-223). Requires Go 1.26+; API locked by ADR 006
(go-output/README.md:142). Multi-module repo (go.work), latest tag
**v0.38.7** (2026-10-19, go-output/CHANGELOG.md:11).

## 2. Usage census in go-cqrs-lite (evidence)

Direct pins (all at **v0.38.7 = latest**):

- `cmd/cqrs-bench/v4/go.mod:20-23` — root + `delimited` + `markdown` +
  `table`, direct.
- `cmd/cqrs-lint/v4/go.mod:10-12` — root + `delimited` + `table`, direct
  (no `markdown` pin).

Indirect-only via cmdguard (its output-format machinery,
`cmdguard/go.mod:33-43,83`), never imported by consumer code:

- `cmd/api-stability/v4/go.mod:39-50`, `cmd/cqrs-gen/v4/go.mod:39-50`,
  `cmd/doc-check/v4/go.mod:39-50`, and the sub-module blocks at
  `cmd/cqrs-bench/v4/go.mod:124-131`, `cmd/cqrs-lint/v4/go.mod:48-56` — all
  `// indirect`.

Zero pins: `catalog/v4/go.mod` (carries templ, yaml, humanize — no
go-output) and every library module, consistent with the per-module
production dep budgets (`nix run .#check-arch`).

Import sites and what they call:

- cqrs-bench: `cmd/cqrs-bench/v4/render.go:12-15` (root, delimited, markdown,
  table), `cmd/cqrs-bench/v4/run_render.go:8-10`,
  `cmd/cqrs-bench/v4/layout.go:13-14`,
  `cmd/cqrs-bench/v4/render_variation.go:6`. Surface: `output.NewTable` (render.go:109, 309,
  334, 423…), `gotable.Write(..., WithColorMode(output.ColorModeAuto))`
  (render.go:72, 284, 394; cmd/cqrs-bench/v4/run_render.go:28;
  cmd/cqrs-bench/v4/layout.go:212),
  `delimited.WriteCSV`/`WriteTSV` (render.go:88-94, 293-299, 402-408;
  cmd/cqrs-bench/v4/run_render.go:65-71), `markdown.Render(..., ColorModeNever)`
  (render.go:52), `output.ColorModeAuto.ShouldColor()` (render.go:38).
- cqrs-lint: `cmd/cqrs-lint/v4/output.go:24-25` (root, table),
  `output_grouping.go:13-14`, `scorecard_render.go:8-9`, `commands.go:12`.
  Surface: `output.NewTableBuilder` (output.go:97, 122, 136;
  scorecard_render.go:123), `table.Render(data, WithColorMode(cm))`
  (output.go:113, 127, 149; scorecard_render.go:141),
  `delimited.RenderCSV`/`RenderTSV` (output_grouping.go:91, 99),
  `output.ParseColorMode` (output.go:36).

Notable non-usage (documented by the consumer itself):
`cmd/cqrs-lint/v4/output.go:1-13` records the render layering and the reason
findings text bypasses go-output ("go-output has no styled-text API, only
format renderers, so raw ANSI codes are used here", output.go:6-8).

## 3. Capability inventory

| # | Capability (module) | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Core data model: `Table`/builders, build-freeze-render (root) | **Full** | `output.NewTable` across `cmd/cqrs-bench/v4/render.go:102-423`; `NewTableBuilder` across `cmd/cqrs-lint/v4/output.go:97-149` |
| 2 | ColorMode/TTY/NO_COLOR/CI detection (root) | **Full** | `output.ParseColorMode` (`cmd/cqrs-lint/v4/output.go:35-42`), `ColorModeAuto.ShouldColor()` (`cmd/cqrs-bench/v4/render.go:38`) |
| 3 | Terminal tables (`table`) | **Full** | `gotable.Write` ×5 sites in bench; `table.Render` ×4 sites in lint |
| 4 | CSV/TSV (`delimited`) | **Full** | `WriteCSV`/`WriteTSV` (bench render.go:88-94), `RenderCSV`/`RenderTSV` (lint output_grouping.go:91-99) |
| 5 | Validated `Format` enum for CLI `--format` flags (root) | Missed | library ships `ParseFormat` + `Supports` exactly for this (go-output/README.md:73-79); bench instead hand-rolls 9 string constants + `resolveFormat` (`cmd/cqrs-bench/v4/render.go:22-47`), lint hand-rolls per-command vocabularies + `validateFormatFlag` (`cmd/cqrs-lint/v4/output.go:47-66`, cmd/cqrs-lint/v4/formats.go:13-16) |
| 6 | Markdown tables (`markdown`) | Partial | bench renders via `markdown.Render` but post-processes away the leading border pipe (`cmd/cqrs-bench/v4/render.go:48-61`); lint pins no markdown and hand-rolls two emitters instead: scorecard (`cmd/cqrs-lint/v4/scorecard_render.go:209-286`) and grouped findings (`cmd/cqrs-lint/v4/output_grouping.go:75-88,278-290`) |
| 7 | ASCII trees (`tree`) | Missed | no import anywhere; lint's `--group module`/`aggregate` findings and bench's layout/sweep hierarchies are flat tables |
| 8 | D2 diagrams (`d2`) | Missed | `catalog/v4/d2/` hand-rolls 633 lines of D2 DSL via `strings.Builder` (`catalog/v4/d2/connections.go:23-46`, `services.go:12-40`); catalog/v4 pins no go-output at all |
| 9 | Per-format escaping (`escape`) | Missed | `escape.D2()` exists (go-output/FEATURES.md:221); catalog's exporter hand-rolls `sanitizeID` + `%q` quoting (`catalog/v4/d2/connections.go:221`, `services.go:84-327`) |
| 10 | DOT/Mermaid (`graph`) | Missed | zero imports; docserver's D2 page degrades to raw source when no server-side D2SVG hook is configured (`catalog/v4/docserver/d2view.templ:10-14`) — Mermaid renders client-side and would close that gap |
| 11 | PlantUML (`plantuml`) | N/A | no PlantUML artifact anywhere in the consumer; D2 is the single blessed diagram language — flip trigger: doc consumers request PlantUML alongside `d2.txt` |
| 12 | XML/HTML/AsciiDoc (`markup`) | N/A | docserver HTML is templ + templ-components doctrine; no XML/AsciiDoc consumer exists |
| 13 | JSON/YAML/TOML/JSONL of table/tree/graph shapes (`serialization`) | N/A | the tools' JSON is typed domain structs via `encoding/json/v2` (`benchkit/v4/report_json.go:44`, `cmd/doc-check/v4/json_report.go:4`) — correct tool; flip trigger: a tabular JSON/YAML export of findings or bench tables |
| 14 | Interactive HTML DAG (`daghtml`) | N/A | docserver renders diagrams via the D2SVG hook + templ; no demand for the JS/CSS asset bundle |
| 15 | NOM real-time progress (`nom`) | Missed | bench hand-rolls a heartbeat/phase progress reporter (`benchkit/v4/progress.go:53-151`, wired via `cmd/cqrs-bench/v4/flags.go:95-104`) — dependency trees, activity counts, and timing estimates are exactly nom's product (go-output/README.md:139) |
| 16 | Cross-shape projections (Table↔Tree↔Graph) | N/A | no conversion need today; every producer has exactly one output shape |

Score derivation: rows N=16, N/A rows 11–14 and 16 → **A=11**. Full=4
(4.0), Partial=1 (0.5), Missed=6 (0). Score = round(100 × 4.5 / 11) =
**41% → low band**.

## 4. Gap analysis

1. **The repo's flagship diagram is a private re-implementation of a
   submodule it doesn't pin.** `catalog/v4/d2` (244+331+48+10 production
   lines) writes D2 by hand — header/subtitle/classes/style blocks through
   `fmt.Fprintf` (`catalog/v4/d2/services.go:12-40`), node/edge emission and
   its own ID sanitizer (`catalog/v4/d2/connections.go:23-221`) — and the
   docserver serves it (`catalog/v4/docserver/d2.go:7,35-54`,
   d2view.templ). `go-output/d2` ships the DSL (SQL tables, 20 shapes,
   grids) and `go-output/escape` ships `escape.D2()`. Every D2 DSL bug fixed
   upstream (quoting, nesting, layout) must be re-fixed in catalog/v4. The
   honest counterweight: catalog/v4 has a strict production dep budget
   (10 direct requires in `catalog/v4/go.mod:10-21`), and go-output/d2 pulls
   the root module — but the tools already accept that graph via cmdguard,
   and the 633-line maintenance liability is larger than the budget line.
2. **cqrs-lint re-implements markdown twice beside the module it doesn't
   pin.** Scorecard markdown (`cmd/cqrs-lint/v4/scorecard_render.go:209-286`)
   and grouped-findings markdown (`cmd/cqrs-lint/v4/output_grouping.go:278-290`)
   are `strings.Builder`/`fmt.Fprintf` tables; cqrs-bench demonstrates the
   library path (`cmd/cqrs-bench/v4/render.go:51-61`). One of the two
   approaches is redundant; today it is the library's.
3. **Every tool hand-rolls the `Format` enum.** Bench's nine format
   constants + `resolveFormat` (`cmd/cqrs-bench/v4/render.go:22-47`) and
   lint's four per-command vocabularies + `validateFormatFlag`
   (`cmd/cqrs-lint/v4/formats.go:13-16`, output.go:47-66) re-create
   `output.ParseFormat`/`Format.Supports` with strings, so vocabulary drift
   between tools is possible and the auto/table/text TTY fallback logic is
   duplicated (render.go:36-47).
4. **NOM progress re-invented at 160 lines.** `benchkit/v4/progress.go`
   implements heartbeat, phase begin/end, and debounced printing — nom's
   dependency-tree + O(1) summary-bar product (go-output/README.md:139)
   would also give time estimates and the activity model for free across
   every bench subcommand.

## 5. Version currency

- All consumer pins are **v0.38.7 = the latest local tag** (2026-10-19) —
  fully current, no drift, including the indirect blocks.
- HEAD 90fb2a6 (2026-11-17) is five days past the tag with only
  `[Unreleased]` "Nothing yet" ahead of v0.38.7
  (go-output/CHANGELOG.md:7-14), so pins match the tip of released work.
- v0.38.x recency worth knowing: v0.38.6 added `daghtml.Edge.Label` tooltips
  and CI coverage for daghtml; v0.38.7 is release-engineering and website
  fixes only (go-output/CHANGELOG.md:11-58) — nothing the consumer uses
  changed behavior, so currency is currently free.
- Fleet note: cmdguard itself carries all 11 submodules at v0.38.7
  (`cmdguard/go.mod:33-43,83`), so the consumer's indirect graph will
  track go-output releases as long as cmdguard bumps — the drift risk is
  cmdguard's cadence, not go-cqrs-lite's.

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only fix
bugs all day long." go-output exists so that format logic is fixed once for
the whole fleet. go-cqrs-lite is the fleet's template repo, and its template
currently teaches: tables via the library (good), diagrams by hand
(catalog/v4/d2), progress by hand (benchkit/v4/progress.go), markdown by
hand in the very tool whose sibling uses the library. When go-output fixes a
D2 quoting bug or adds a markdown footer convention, every fleet app that
copied the recipe stays broken until each one ports the fix. The doctrine
constraint is not the blocker people might assume: dep budgets bind the
library modules, but `catalog/v4` already carries templ, yaml, and
humanize in production, and the docserver is where diagrams live. The
conflict is not doctrine vs adoption; it is an unowned 633-line DSL
duplicating an owned one.

## 7. Ranked opportunities

1. **Port `catalog/v4/d2` to `go-output/d2` + `go-output/escape`**
   (high, largest leverage): delete the hand-rolled DSL, keep the golden
   tests (`catalog/v4/d2/golden_test.go`) as the acceptance gate, and add
   the two production deps to `catalog/v4/go.mod` with a budget note. One
   place for D2 bugs fleet-wide.
2. **Pin `go-output/markdown` in cqrs-lint and delete the two hand-rolled
   emitters** (high): `scorecard_render.go:209-286` and
   `output_grouping.go:278-290` collapse to `markdown.Render` calls; the
   bench TrimPrefix quirk (render.go:57-59) should be reported upstream so
   both tools share one markdown shape.
3. **Adopt the `Format` enum for `--format` flags** (medium): replace
   bench's constants and lint's vocabularies with `output.ParseFormat` +
   `Supports`, keeping the tools' aliases (benchstat, manifest, sarif) as
   pre-parsed pass-throughs.
4. **Evaluate `go-output/nom` for bench progress** (medium): swap
   `benchkit/v4/progress.go` for nom's dependency-tree rendering during
   multi-phase runs and soaks; keep the benchkit `ProgressWriter` seam.
5. **Mermaid fallback in docserver** (low): `graph.RenderMermaid` from the
   same `Graph` data would let `d2view.templ` render client-side when the
   D2SVG hook is absent instead of degrading to source text.

## 8. Reconsider when

- catalog/v4's dep budget is renegotiated, or go-output/d2 offers a
  zero-extra-dep path (then opportunity 1 becomes trivially right).
- The docserver grows a second diagram export (PlantUML/Mermaid/DOT rows
  flip from N/A/Missed to required).
- A fleet app wants browser-native diagrams without a server-side render
  hook (Mermaid row).
- bench soaks become interactive operator workflows (tui/nom rows).
- go-output ships a styled-text API — the consumer has pre-documented the
  demand at `cmd/cqrs-lint/v4/output.go:6-8`.

## 9. How to verify

```
rg -n 'larsartmann/go-output' -g go.mod /home/lars/projects/go-cqrs-lite
rg -ln 'larsartmann/go-output' --type go /home/lars/projects/go-cqrs-lite
git -C /home/lars/projects/go-output log -1 --format='%H' && git -C /home/lars/projects/go-output tag --list | sort -V | tail -3
sed -n '22,61p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-bench/v4/render.go
sed -n '1,13p;47,66p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/output.go
sed -n '205,290p' /home/lars/projects/go-cqrs-lite/cmd/cqrs-lint/v4/scorecard_render.go
sed -n '12,60p' /home/lars/projects/go-cqrs-lite/catalog/v4/d2/services.go
sed -n '23,60p;221,244p' /home/lars/projects/go-cqrs-lite/catalog/v4/d2/connections.go
sed -n '53,151p' /home/lars/projects/go-cqrs-lite/benchkit/v4/progress.go
sed -n '136,142p;232,243p' /home/lars/projects/go-output/README.md
```

Related reports in this directory: `cmdguard.md` (carries all go-output
submodules indirectly), `go-codec.md` (the other format layer, for wire
codecs rather than presentation).
