# templ-components — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/templ-components` @ bc369053 (latest local tag v1.27.2) · Consumer go-cqrs-lite @ feec9bc74 · Group G5

**Verdict:** WELL-USED
**Score:** 12.5/17 applicable capabilities leveraged = 74% (high)

templ-components is pinned by exactly one consumer module — `catalog/v4`,
for the docserver UI — and that one module uses it the way a chassis library
wants to be used: the entire static chrome (page base, containers, nav,
breadcrumbs, badges, stat cards, tables, definition lists, alerts, copy
buttons, collapsible sections, empty states, theme toggle) is the library's,
not hand-rolled HTML. The gaps are a small docs-natural tail — code blocks,
prose, tabs, branded error pages — and a two-patch version lag that includes
a JSON-LD break-out fix on a surface the docserver renders.

## 1. What the library is

A GOTH-stack UI component library on `a-h/templ`: 49 display components
(accordion, tables, charts, modal, command palette, tooltip, …), 22 form
controls, 6 feedback primitives, 9 layout primitives, 8 navigation primitives
(`templ-components/display/*.templ` and siblings), plus separately-versioned
submodules — `icons` (~105 named icons, `templ-components/icons/icon_names.go:10`),
`htmx`, `datastar`, `errorpage`, `charts/echarts`, and `utils` — all released
in lockstep (v1.27.2 across the family in `templ-components/go.mod:6-12`).
Dependencies are minimal (templ, tailwind-merge-go, go-datastar/static,
go-error-family, yaml). The repo pairs hard with a templ generator version:
the v1.27.x line requires templ ≥ v0.3.1100, and the v1.27.1 release is the
postmortem of a templ-pairing incident where CI regenerated with the wrong
generator (`templ-components/CHANGELOG.md:30-56`).

## 2. Usage census in go-cqrs-lite (evidence)

Exactly one module pins the family — the docserver's own module:

- `catalog/v4/go.mod:17` — `templ-components v1.27.0` (direct).
- `catalog/v4/go.mod:18-19` — `icons` + `utils` v1.27.0 (direct).
- `catalog/v4/go.mod:45` — `htmx` v1.27.0 (indirect only; pulled by the root
  module, never imported by consumer code).
- `catalog/v4/go.mod:11` — `a-h/templ v0.3.1100`, the coherent generator
  pairing for the v1.27.x line.

Six packages are imported across the five docserver `.templ` sources:

- `catalog/v4/docserver/layout.templ:4-6` — layout, navigation, utils.
- `catalog/v4/docserver/index.templ:6-8` — display, icons, layout.
- `catalog/v4/docserver/eventcatalogview.templ:6-11` — display, feedback,
  icons, layout, navigation, utils.
- `catalog/v4/docserver/specview.templ:4` — layout.
- `catalog/v4/docserver/d2view.templ:4-7` — display, feedback, layout, utils.

Symbol census (from the `.templ` sources, the compile truth): ~60 distinct
exported symbols. Highlights with sites: `layout.Base`
(`catalog/v4/docserver/specview.templ:42`), `layout.Container`
(`catalog/v4/docserver/layout.templ:59`), `layout.ThemeToggle`
(`catalog/v4/docserver/layout.templ:43`), `layout.Script` with nonce
(`catalog/v4/docserver/specview.templ:50`), `layout.HTMXNone`
(`catalog/v4/docserver/layout.templ:21`), `navigation.SimpleNav`
(`catalog/v4/docserver/layout.templ:51`), `navigation.Breadcrumbs`
(`catalog/v4/docserver/eventcatalogview.templ:215`), the stat-card row
(`catalog/v4/docserver/index.templ:47-53`), `display.Table`
(`catalog/v4/docserver/eventcatalogview.templ:41`),
`display.DefinitionList` (`catalog/v4/docserver/eventcatalogview.templ:226`),
`display.Badge` tones (`catalog/v4/docserver/index.templ:93-99`),
`display.CopyButton` (`catalog/v4/docserver/eventcatalogview.templ:269`),
`display.CollapsibleSection` (`catalog/v4/docserver/d2view.templ:43`),
`display.EmptyState` (`catalog/v4/docserver/eventcatalogview.templ:200`),
`feedback.Alert` (`catalog/v4/docserver/eventcatalogview.templ:100`),
ten named icons (`catalog/v4/docserver/eventcatalogview.templ:327-329`),
`utils.BaseProps` (`catalog/v4/docserver/layout.templ:33`), `utils.Lookup`
(`catalog/v4/docserver/eventcatalogview.templ:238`).

No other module in the 104-module repo requires the family — consistent with
the library-tier dep budgets (`nix run .#check-arch`); UI belongs to the one
docs surface. The consumer's own CSS pipeline stays deliberately local:
`catalog/v4/docserver/static/docs-ui.src.css` is the source, built via
`nix run .#build-docserver-css` and drift-gated (`catalog/v4/AGENTS.md:41-48`),
with a headless-Chromium CSP gate over the real docserver
(`flake.nix:1548-1552`).

## 3. Capability inventory

| # | Capability | Grade | Evidence / reason |
|---|---|---|---|
| 1 | Page chrome (`layout.Base`/`PageProps`/`Container`/`Script`/`Stylesheet`) | **Full** | every docserver page composes them (`catalog/v4/docserver/specview.templ:42-85`); Script threads the CSP nonce for the Scalar/AsyncAPI bundles |
| 2 | Navigation (`SimpleNav`, `Breadcrumbs`, `NavLink`) | **Full** | `catalog/v4/docserver/layout.templ:36-51`, `catalog/v4/docserver/eventcatalogview.templ:215-216` |
| 3 | Dark mode (`layout.ThemeToggle`) | **Full** | `catalog/v4/docserver/layout.templ:43` |
| 4 | Badges + tones (`display.Badge*`) | **Full** | 15 call sites (`catalog/v4/docserver/index.templ:93-99`) |
| 5 | Stat cards (`display.StatCard`) | **Full** | 13 call sites across index + eventcatalog overview (`catalog/v4/docserver/index.templ:47-53`) |
| 6 | Cards + grids (`display.Card`, `Grid`) | **Full** | `catalog/v4/docserver/index.templ:28-47` |
| 7 | Tables (`display.Table`) | **Full** | 8 call sites (`catalog/v4/docserver/eventcatalogview.templ:41`) |
| 8 | Definition lists | **Full** | `catalog/v4/docserver/eventcatalogview.templ:226` |
| 9 | Feedback alerts (`Alert`, `FeedbackInfo/Warning/Error`) | **Full** | `catalog/v4/docserver/eventcatalogview.templ:100-320`, `catalog/v4/docserver/d2view.templ:55-56` |
| 10 | Copy button (CSP-safe) | **Full** | `catalog/v4/docserver/eventcatalogview.templ:269`, `catalog/v4/docserver/d2view.templ:27-61` |
| 11 | Collapsible section | **Full** | `catalog/v4/docserver/d2view.templ:43` |
| 12 | Empty states | **Full** | `catalog/v4/docserver/eventcatalogview.templ:147-200` |
| 13 | Icon set | Partial | 10 of ~105 named icons used (`templ-components/icons/icon_names.go:10`); a docs UI legitimately needs few |
| 14 | Code blocks (`display.CodeBlock`, integrated copy button) | Missed | docserver hand-rolls `<pre><code>` (`catalog/v4/docserver/d2view.templ:49-52`, `catalog/v4/docserver/eventcatalogview.templ:274-277`) while using `CopyButton` separately — it re-assembles the two halves `templ-components/display/code_block.templ:9` ships as one component |
| 15 | Prose container (`display/prose`) | Missed | unused; long-form doc pages (ADR text, guides) render unstyled |
| 16 | Tabs (`display/tabs`) | Missed | spec pages alternate JSON/schema views stacked instead of tabbed |
| 17 | Error pages (`errorpage` submodule, incl. `notfound404`) | Missed | docserver 404s bare via mux with no branded page (`templ-components/errorpage/notfound404.templ:1` unused; no catch-all by design, `catalog/v4/AGENTS.md` Routes section) |
| 18 | Forms suite (22 controls) | N/A | docserver has zero input surfaces; static docs posture |
| 19 | Toast / loading / progress | N/A | no dynamic UI actions; static `Alert`s carry the job |
| 20 | `htmx` submodule | N/A | deliberately suppressed: `layout.HTMXNone` sentinel (`catalog/v4/docserver/layout.templ:21`); interactivity is the vendored Scalar/AsyncAPI SPAs, not htmx |
| 21 | `datastar` submodule | N/A | no datastar surfaces in the consumer's docs posture |
| 22 | `charts/echarts` | N/A | d2 diagrams render server-side SVG (`catalog/v4/docserver/d2view.templ`); no client charts |

Score derivation: rows N=22, N/A rows 18–22 → **A=17**. Full=12 (12.0),
Partial=1 (0.5), Missed rows 14–17 (4 × 0). Score = round(100 × 12.5 / 17) =
**74% → high band**.

## 4. Gap analysis

1. **The docserver re-assembles `CodeBlock` from parts.** It uses the
   library's CSP-safe `CopyButton` next to hand-rolled `<pre><code>`
   (`catalog/v4/docserver/d2view.templ:49-52`), which is exactly the split
   `templ-components/display/code_block.templ:9` exists to close. Three call
   sites; one-swap fix; inherits upstream nonce/copy hardening forever after.
2. **Version lag lands on a security-hardened pair of patches.** Pinned
   v1.27.0; v1.27.1 and v1.27.2 are both hardening releases on surfaces the
   docserver renders: the breadcrumb JSON-LD `</` break-out fix
   (`templ-components/CHANGELOG.md:13`) touches `navigation.Breadcrumbs`,
   which the docserver calls at `catalog/v4/docserver/eventcatalogview.templ:215`
   with item names derived from consuming apps' message names; the
   `utils.ScriptComponent` literal-`</script>` refusal
   (`templ-components/CHANGELOG.md:19`) is the same class the docserver's
   nonce'd inline scripts (`catalog/v4/docserver/specview.templ:55`) guard
   against. Catalog names are developer-controlled today, so this is
   hardening-class rather than an open hole — but the fix costs a pin bump
   plus a CSS rebuild (`catalog/v4/AGENTS.md:46`).
3. **The pairing matrix is documented in the wrong place — and one spot is
   stale.** The "v1.27.x pairs with templ ≥ v0.3.1100" rule that cost two
   sessions (`TODO_LIST.md:121-123`) lives in the consumer's TODO, while
   `catalog/v4/AGENTS.md:14` still instructs regenerating with
   `templ@v0.3.1020` and `catalog/v4/AGENTS.md:17` asserts the go.mod pin is
   v0.3.1020 — but `catalog/v4/go.mod:11` pins v0.3.1100. The next
   contributor following AGENTS.md regenerates with a churn-producing CLI,
   recreating the exact incident v1.27.1's postmortem describes.

## 5. Version currency

- Pinned **v1.27.0** (`catalog/v4/go.mod:17-19`); latest local tag
  **v1.27.2** (2026-11-20, `templ-components/CHANGELOG.md:9`). Two patches
  behind.
- Both missed patches are low-risk to adopt: v1.27.2 is the JSON-LD escape
  + ScriptComponent refusal + release-gate hardening
  (`templ-components/CHANGELOG.md:13-23`); v1.27.1 fixed CI generator pin,
  Toggle `aria-checked`, a stray duplicate `</script>` in ToastContainer
  (unused by the docserver), and pagination aria-current
  (`templ-components/CHANGELOG.md:30-56`). Nothing the docserver uses
  changed shape; the templ pairing (v0.3.1100) is unchanged.
- Bumping ripples the built docserver CSS — the documented ritual is bump,
  then `nix run .#build-docserver-css`, then the drift gate
  (`catalog/v4/AGENTS.md:46-47`; also `CONTRIBUTING.md:320`).

## 6. The fleet argument

The owner's lens: "if every project re-integrates everything I will only fix
bugs all day long." templ-components is where fleet UI fixes land once. The
docserver is the consumer's only browser surface, and it already delegates
its whole chrome there — so a future XSS-class fix in `Badge` or `Table`
reaches every embedded docs site via one pin bump. The two real exposures to
the lens are the hand-rolled `<pre><code>` (a future CodeBlock fix will not
reach it) and the two-patch lag (the JSON-LD fix has not reached it yet).
Both are single-digit-minute fixes; neither is structural. This is the
healthiest "one consumer, deep use" shape in the audit so far.

## 7. Ranked opportunities

1. **Bump v1.27.0 → v1.27.2** (high, trivial): two hardening patches on
   rendered surfaces; rebuild CSS per `catalog/v4/AGENTS.md:46` and run the
   CSP gate (`flake.nix:1548-1552`).
2. **Swap hand-rolled code blocks for `display.CodeBlock`** (medium): three
   sites (`catalog/v4/docserver/d2view.templ:49`, `catalog/v4/docserver/d2view.templ:62`,
   `catalog/v4/docserver/eventcatalogview.templ:274`); drops the local
   pre/code CSS and unifies copy behavior.
3. **Fix the stale regenerate instruction** (high, trivial): update
   `catalog/v4/AGENTS.md:14` and `catalog/v4/AGENTS.md:17` from v0.3.1020 to
   v0.3.1100 so the next regeneration does not churn every committed
   `*_templ.go`.
4. **Branded 404 via `errorpage`** (low): one component, matches the
   docserver's visual identity, removes a bare-mux 404.
5. **Tabs for spec views** (low): JSON/schema/example alternation on
   spec pages is the natural `display/tabs` fit.

## 8. Reconsider when

- The docserver grows user input (search boxes flip Forms from N/A to
  live — `filter_bar`/`combobox` are the natural fit).
- Any fleet app beyond catalog adopts templ UI — the family-wide pairing
  rule (templ CLI = go.mod pin) then deserves a fleet-level check script
  rather than per-repo AGENTS lines.
- templ-components ships a d2/code-diagram primitive — `charts/echarts`
  stays N/A only while diagrams are server-rendered SVG.
- A templ security fix lands (then finding 2 stops being hygiene and
  becomes a patch gate, same ruling as httputil's GO-2026-6859 class).

## 9. How to verify

```
rg -n 'templ-components' -g go.mod /home/lars/projects/go-cqrs-lite
rg -on '(display|layout|navigation|icons|feedback|utils)\.[A-Z][A-Za-z0-9]*' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver -g '*.templ' | sort -u | wc -l
sed -n '17,19p;45p' /home/lars/projects/go-cqrs-lite/catalog/v4/go.mod
sed -n '49,52p' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/d2view.templ
sed -n '274,277p' /home/lars/projects/go-cqrs-lite/catalog/v4/docserver/eventcatalogview.templ
sed -n '9,23p;30,56p' /home/lars/projects/templ-components/CHANGELOG.md
sed -n '14p;17p;41,48p' /home/lars/projects/go-cqrs-lite/catalog/v4/AGENTS.md
git -C /home/lars/projects/templ-components log -1 --format='%H' && git -C /home/lars/projects/templ-components tag --list | sort -V | tail -3
```

Related sibling context: the templ-era pairing history lives in
`TODO_LIST.md:526` (the BuildFlow stale-CLI whipsaw) and the corrected era
matrix in `TODO_LIST.md:123`.
