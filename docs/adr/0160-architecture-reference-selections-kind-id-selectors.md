# ADR-0160: Architecture-reference selections select resources by kind and id, not by custom properties

Date: 2026-10-17 · Status: Accepted · Supersedes: nothing · Relates to: ADR-0159 (self-contained HTML export), ADR-0111 (Record/type model)

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
