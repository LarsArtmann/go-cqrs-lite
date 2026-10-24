# ADR-0160: Architecture-reference selections select resources by kind and id, not by custom properties

Date: 2026-10-17 · Status: Accepted · Supersedes: nothing · Relates to: ADR-0159 (self-contained HTML export), ADR-0111 (Record/type model)

> **Status banner (2026-10-24):** SHIPPED IN FULL — selections, references modes,
> hide, `Inspect`/`-check`, frontmatter specs, the docserver route, the example
> and the benchmark all landed 2026-10-17→24. Every ruling below is annotated
> in the [implementation addendum](#implementation-addendum-2026-10-24) with
> file:line evidence; nothing was reversed, two surface details deviated as
> recorded there.

## Context

ADR-0159 shipped the all-or-nothing reference: every resource in the catalog
lands in one HTML file. EventCatalog 5.2 complements that with **custom
exports** — a frontmatter declaration that picks a subset
(`include`/`exclude`), controls how references to unselected resources render
(`references: omit|name-only|include`), suppresses sensitive blocks
(`hide: [owners, specifications, …]`), pins resources that must never appear
(`neverExport`), and proves the result with a `--check` report.

The fleet ask for this is real (share the ordering slice with a partner without
the payments slice), but the selection model has one design fork the catalog
model does not yet resolve: **what can a selector select ON?**

EventCatalog selectors key on resource ids, kinds, and — critically —
**custom properties** declared in frontmatter (`x-audience: partner` and
friends). Our catalog model has no such concept: `BaseConfig` carries id,
name, version, summary, owners, badges and tags, and nothing else open-ended.
The only custom-shaped surfaces are `CustomDoc` pages and flow-node custom
types, neither of which is a queryable resource attribute.

## Decision

**A selection is a set of `(kind, id)` selectors. Custom-property selectors
are rejected until the model grows a custom-property concept.**

- `Selector{Kind, ID}` addresses exactly one resource; convenience set
  builders (a whole domain and its members, a service and its messages) are
  derived, not new selector semantics.
- `include` and `exclude` are selector lists; **exclude always wins**, and an
  include of a container (domain) brings its members — the natural closure —
  but never resurrects an excluded resource.
- Resources outside the kept set render only as their `references` mode
  allows (omit by default); nothing outside the set is reachable by link,
  sidebar, or search index.
- No `x-*` frontmatter matching. Adding `BaseConfig.Properties
  map[string]any` to enable it would (a) grow the published model for an
  unproven need, (b) reintroduce the `any`-typed data the repo's strong-types
  contract forbids in domain logic, and (c) couple export visibility to a
  mechanism EventCatalog owns and we cannot gate. If a real consumer needs
  property-driven selection, the model change gets its own ADR with typed
  properties — not a stringly-typed map.

## Consequences

- The selection API stays small and fully typed: `Selector`, a kept-set
  computation, and the report type. No schema migration, no frontmatter
  grammar, no YAML dependency beyond what the catalog module already carries.
- EventCatalog-shaped `.mdx` export files, when supported, may still declare
  `include`/`exclude` **by id and kind only**; a file that uses custom-property
  selectors fails validation with a message pointing at this ADR instead of
  silently selecting nothing.
- The never-leak guarantee is testable by construction: the renderer filters
  the kept set before any section, nav or search entry is built, so a leak
  test (excluded resource name appears nowhere in the output) pins the whole
  pipeline, not individual renderers.
- Semantics that depend on closure order (exclude-wins, container-brings-
  members) are pinned by tests, so consumer tooling can rely on them.
- Cost of the ruling: consumers who today filter by convention (separate
  catalogs per audience) keep doing that. The export selection solves the
  sharing problem for id-addressable slices; audience metadata remains a
  catalog-per-audience concern until the model says otherwise.

## Implementation addendum (2026-10-24)

Every ruling of the Decision, against what actually shipped:

| Ruling | Status | Evidence |
|---|---|---|
| `Selector{Kind, ID}` addresses exactly one resource | DONE | `catalog/architecture/selection.go:16`; kind spellings are the `adrKind` vocabulary (`domain`, `service`, `command`, `event`, `query`, `channel`, `container`, `flow`, `entity`, `data-product`, `agent`, `decision`, `team`, `person`, `doc`) |
| include brings the natural closure (domain → services → messages); exclude always wins; neverExport beats include | DONE | `catalog/architecture/plan.go:30` (`plan`: ban first, closure second, `neverExport` then `exclude`); pinned by `TestSelectionExcludeWins`, `TestSelectionDomainBringsMembers`, `TestSelectionNeverExportBeatsInclude` |
| Outside-set resources render only as the references mode allows; nothing outside is reachable by link, sidebar or search index | DONE | modes: `catalog/architecture/references.go:8`; kept-set gating of sections/nav/search: `catalog/architecture/collect.go:84` (`addResource`); the leak probe is `TestExcludedResourceNameNeverAppears` plus the composed-path suite `TestComposedRedactionNeverLeaks` (mutation-verified) |
| Convenience set builders are derived, not new selector semantics | DONE (as references-pull) | one-level edge pull under `references: include`: `catalog/architecture/edges.go:29` + `plan.go:118`; a DOMINANT deviation from the ADR text: the "set builders" idea shipped as the references pull and the closure, not as separate builder functions — no separate API exists |
| Report type / proof of the result (`--check`) | DONE | `catalog/architecture/report.go:23` (`ExportReport`: `Included`/`Excluded`/`NotFound`/`Dropped`/`Hidden` + `Failed()` at `report.go:44`), `Inspect` at `report.go:51`; CLI `-check` dry-run exits 1 on any failed dimension: `catalog/cmd/catalog-export/main.go:142` |
| `.mdx` export files supported, id+kind selectors only, custom-property selectors fail validation loudly | DONE | `LoadExport` (`catalog/architecture/frontmatter.go:63`) rejects `x-*` selectors with a message naming this ADR and (since 2026-10-24) unknown `references` spellings; pinned by `TestLoadExportParsesSpec`, `TestLoadExportValidatesReferencesMode` |
| No `x-*` frontmatter matching; no `BaseConfig.Properties map[string]any` | UNCHANGED (rejected) | the model grew no custom-property concept; a property-selection need still requires its own typed-model ADR |
| `hide: [owners, specifications, …]` | DONE, vocabulary narrowed | `catalog/architecture/hide.go:6`: `owners`, `repository`, `examples`, `specifications`, `docs`. Deliberately NO `contacts` — no resource in the model carries contact fields, so a contacts hide would guard nothing (ADR-0159's section inventory) |

Deviations from EventCatalog's surface (deliberate, recorded):

- The CLI loads a spec with `-spec <file>` instead of EventCatalog's
  `-export/--dir` pairing — the flag composes with the same `-include/
  -exclude/-never-export/-references/-hide` flags (spec first, flags extend;
  pinned by `TestSpecAndFlagPrecedence`).
- `hide` is an exporter option (`WithHide`) as well as frontmatter; the CLI
  spells sections comma-separated (`-hide owners,docs`).
- Unknown `references` values in a spec file are a Rejection; on the CLI they
  degrade to omit (files are maintained artifacts, flags are lenient) —
  recorded on `ExportSpec.References`.
