# Architecture understanding

Everything that tries to answer "what is this system, really?" — scored reviews, cross-domain
mappings, tier models, and the diagrams that go with them. Reports are **timestamped snapshots**:
they record what was true at their basis commit and are never retro-edited (the version in git is
the record; later corrections land in a new report or in `TODO_LIST.md`).

## Scored reviews (the series)

| Date | Report | Focus | Verdict |
| ---- | ------ | ----- | ------- |
| 2026-10-24 | [`2026-10-24_16-54_architecture-review.html`](2026-10-24_16-54_architecture-review.html) | Modularity, service orientation, composability, scalability | **4.0 / 5** (7 dimensions; Cohesion + Scalability 3, Dependency direction 5) |
| 2026-07-09 | [`archive/2026-07-09_05-18_ARCHITECTURE-REVIEW.html`](archive/2026-07-09_05-18_ARCHITECTURE-REVIEW.html) | First full review (48 modules) | **7.0 / 10** (its 6 roadmap items are all resolved) |

The 2026-10-24 review ships with a diagram pair:

- [`2026-10-24_16-54_architecture-review-current.d2`](2026-10-24_16-54_architecture-review-current.d2)
  / [`.svg`](2026-10-24_16-54_architecture-review-current.svg) — the module constellation as built:
  Tier 0 primitives at the bottom, tooling and the release gates on top, the `core/v5` fork beside
  the v4 tree.
- [`2026-10-24_16-54_architecture-review-improved.d2`](2026-10-24_16-54_architecture-review-improved.d2)
  / [`.svg`](2026-10-24_16-54_architecture-review-improved.svg) — the target state: one `core/v5`
  module with package boundaries, `metaengine` split into five packages, drivers unchanged, tooling
  as a product line of its own.

Preview the same pair as PNG (single-file, no SVG viewer needed):
[current](2026-10-24_16-54_architecture-review-current.png) ·
[improved](2026-10-24_16-54_architecture-review-improved.png). SVG stays canonical — the PNGs are
renders of the `.d2` sources and must be regenerated with them, never edited.

Rendering: `d2 file.d2 file.svg && d2 file.d2 file.png` (d2 ≥ 0.9). Deliberately **flat** node
graphs — nested containers plus cross-container edges make the layout engine produce 15k-pixel-wide
canvases.

## Mappings and models

| Document | What it maps |
| -------- | ------------ |
| [`SEVEN-TIER-MODEL.md`](SEVEN-TIER-MODEL.md) | Every module → one of seven tiers (the layering authority behind `check-module-layers.sh`) |
| [`2026-09-10_cordis-spatiotemporal-composability-mapping.md`](2026-09-10_cordis-spatiotemporal-composability-mapping.md) | Repo structure and the go-modularize skill → the Cordis context paradigm (coeffects vs config) |
| [`2026-09-15_zenoh-spatiotemporal-fabric-mapping.md`](2026-09-15_zenoh-spatiotemporal-fabric-mapping.md) | Repo → Zenoh's spatiotemporal fabric abstractions |
| [`2026-09-24_data-mesh-conformance-mapping.md`](2026-09-24_data-mesh-conformance-mapping.md) | Repo → data-mesh principles (as a conformance argument, not an aspiration) |
| [`ANALYTICAL-VS-TRANSACTIONAL.md`](ANALYTICAL-VS-TRANSACTIONAL.md) | Which module owns which workload shape (OLTP vs OLAP boundaries) |
| [`AGGREGATE-CONCEPT-ANALYSIS.md`](AGGREGATE-CONCEPT-ANALYSIS.md) | DDD aggregate concept vs what the code actually models |
| [`2026-08-04_metaengine-goal-gap.html`](2026-08-04_metaengine-goal-gap.html) | Operator-ergonomics gap analysis for metaengine (historical) |
| [`2026-08-09_self-integration-review.md`](2026-08-09_self-integration-review.md) | How the modules do (and do not) compose in practice |
| [`2026-07-23_book-insights-vs-codebase.html`](2026-07-23_book-insights-vs-codebase.html) / [`…-detailed-answers.md`](2026-07-23_book-insights-detailed-answers.md) | Book-derived principles vs this codebase |
| [`cordis-arxiv-2608.25512.pdf`](cordis-arxiv-2608.25512.pdf) | Source paper for the September mappings |

[`archive/`](archive/) holds 45 older artifacts (13 reports + their diagram pairs) from the
2026-05/07 episodes. They stay readable and unedited; their conclusions are superseded by the
series above.

## Adding a report

1. Name it `YYYY-MM-DD_HH-MM_<slug>.html` (HTML is the canonical report format; `.md` only when the
   owner explicitly asks, and then say so in the header).
2. Read the previous report in the series first, and open with a short reconciliation section —
   "what changed since, and which earlier claims were wrong".
3. Cite a basis commit. Measured numbers must be reproducible: prefer
   `bash scripts/arch-metrics --json` (module/package counts, import edges, fan-in/out,
   instability, LOC split, API concentration) over ad-hoc commands.
4. Every recommendation becomes a `TODO_LIST.md` row **in the same session** — a report without a
   harvest is a document nobody acts on. See the 2026-10-24 follow-up ledger for the pattern
   (52 findings, each with a routing owner).
5. Add the report to the table above; link it from [`docs/README.md`](../README.md) if it changes
   the top-level story.
