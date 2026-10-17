# ADR-0159: Publish the architecture reference as self-contained HTML, rendered in Go

Date: 2026-10-17 · Status: Accepted · Supersedes: nothing · Relates to: ADR-0152 (fleet-first topology, cqrs-htmx/go-appkit companions), ADR-0128 (core extraction), ADR-0123 (system composition root)

## Context

`catalog/v4/eventcatalog` emits an MDX tree that only the Node EventCatalog CLI
can turn into a site, and `catalog/v4/docserver` serves a multi-page,
templ-rendered view from a live `*catalog.Catalog`. Both assume the reader can
reach the catalog: one needs `npm install && npx eventcatalog build`, the other
needs the service running.

EventCatalog 5.1 shipped "export as one HTML file" for exactly the readers who
cannot reach it — a partner integrating, an auditor, an acquired team, a
contractor waiting on VPN access. The capability is real and the need is
durable; the delivery mechanism is not ours, because it lives in the Node
toolchain this repo deliberately removed from the deploy path when
`eventcatalog.StaticServer` was built to serve a pre-built `dist/` without Node.

Meanwhile the catalog model already carries everything a reference needs:
summaries, badges, owners, ubiquitous language, JSON Schemas, examples,
per-direction message relations, channels, stores, flows, ADR bodies and custom
docs. Nothing about the output requires Node — only the renderer was there.

## Decision

**`catalog/architecture` renders a `*catalog.Catalog` as ONE self-contained HTML
document, in Go, with no new dependencies.**

- **Shape**: a filterable, kind-tagged sidebar; a section per domain, service,
  command, event, query, channel, data store, flow, entity, data product, agent,
  decision, team, person and documentation page; every relationship as an
  in-file link; message attribute tables from the schema with a real or
  synthesised example payload, a generated TypeScript interface and the raw
  JSON Schema; flow steps in order; Ctrl/Cmd-K search over an inlined index;
  light/dark theming that follows the OS.
- **Self-containment is the contract**: CSS and JS are `//go:embed`-ed and
  inlined, no fonts or scripts are fetched, and the output is verified
  self-contained by test. A reader can open it from an email attachment years
  from now.
- **Standard library only.** No templ, no markdown engine, no new module
  dependency: the catalog module's budget (`DEP_BUDGET[catalog]=5`) is full, and
  the package's value is that a service which never serves HTML can publish a
  reference without linking a UI stack. Prose is rendered by a deliberate
  markdown subset (headings, lists, quotes, fenced code, inline emphasis/code,
  links, `[[kind|id]]` references) — the escape hatches a full engine adds are
  the ones that make a shared document unsafe to hand to a third party.
- **Safe by default, because it leaves the team**: catalog strings are escaped
  through a single writer, badge colors are sanitised, external URLs are
  scheme-allowlisted, and raw HTML in prose is rejected rather than trusted.
- **Anchors are derived, not stored**: `kindInfo.anchor(id)` is computed from
  the section's kind and resource id, so sidebar, search index and cross-links
  cannot disagree.
- **Two delivery surfaces**: `Exporter.Export/ExportTo/ExportFile` for
  in-process use (`//go:embed` and serve it from an existing route), and
  `catalog-export -architecture <path>` writes one next to the EventCatalog MDX
  tree the hub already consumes.

**Explicitly out of scope for this ADR**: custom exports (frontmatter-selected
subsets with `include`/`exclude`/`references`/`hide`/`neverExport`). The
selection model is a separate decision — it needs a typed selector contract, a
`--check` report, and a never-leak guarantee — and the renderer is a
prerequisite for it, not the other way round. Until then, the reference is
all-or-nothing, which is the honest description of the current behaviour.

## Consequences

- A consumer can publish an architecture reference with zero new dependencies
  and zero toolchain: one call, one file. This is what makes the capability
  reachable for the fleet's plain Go services, not only for EventCatalog users.
- The `eventcatalog` MDX export stays the path for a HOSTED catalog, and
  `docserver` stays the path for a LIVE one. Three surfaces, three audiences: a
  served site, a Node-built site, a file anyone can open. The ADR does not merge
  them, because their lifecycles differ (per-request, per-build-with-Node,
  per-build-without-any-toolchain).
- Two renderers now exist for the same data (templ in `docserver`, writer in
  `architecture`). That duplication is accepted: they share no templates, target
  different outputs (multi-page app vs single file), and coupling them would
  force every consumer of the file export to link templ — the cost this ADR
  exists to avoid. The catalog AGENTS.md records the contracts that keep the
  pair honest (derived anchors, single escaping path, search-index shape,
  dependency budget).
- The reference is a NEW published artifact and a new published shape (the
  inlined search index). Both are gated: export-level tests pin section
  coverage, cross-links, escaping, self-containment and HTML tag balance, unit
  tests pin the markdown subset, TypeScript generation and URL allow-listing,
  and the consumer-facing recipe is compile-verified by `cmd/doc-check`.
- The recipe (~100 s cold) becomes part of the `#verify` doc gate's cost. That
  is the price of shipping a copy-paste example that actually compiles, which
  this repo pays for every other recipe too.
