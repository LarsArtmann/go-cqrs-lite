# ADR Emission Groundwork Sketch (design-only, blocked:upstream)

- **Created:** 2026-10-15 20:05 CEST
- **Status:** SKETCH — **NO implementation until eventcatalog-hub confirms
  the upstream ADR frontmatter schema** (hub plan task T7.1). Guessing a
  schema = guaranteed rework; this file exists so the design is ready the
  day the confirmation lands.
- **Source asks:** TODO_LIST § "Upstream asks from eventcatalog-hub" row 1
  (hub T7); the hub audit's top content gap (our docs/adr/0xxx corpus is
  exactly what an "ask your architecture" consumer wants; core renders ADRs
  as first-class resources).
- **Living home:** the TODO_LIST row owns the work item; this file is the
  point-in-time design.

## 1. What an EventCatalog ADR is (what we know today)

EventCatalog v5 renders architecture decision records as first-class
resources: frontmatter (id, title, status, date, ...) + a Markdown body, in
an `adrs/` collection. UNVERIFIED for us: the exact zod schema (required
fields, status enum values, whether `authors`/`deciders` are people
pointers or strings, sidebar/owners support, `x-*` extension acceptance).
The hub's T7.1 confirmation must answer all of these before the exporter
grows emission code.

## 2. Catalog API sketch (resource type)

Follows the `catalog.Flow`/`catalog.Entity` precedent: plain typed struct,
registered on the `Registry`/`Builder`, copied immutably at `Build` time,
exported as a directory. NO new dependencies (`DEP_BUDGET[catalog]=5` is
FULL — the type is stdlib-only).

```go
// ADRStatus is the lifecycle state of an architecture decision record.
type ADRStatus string

const (
	ADRProposed   ADRStatus = "proposed"
	ADRAccepted   ADRStatus = "accepted"
	ADRDeprecated ADRStatus = "deprecated"
	ADRSuperceded ADRStatus = "superceded"
)

// ADR represents an architecture decision record. Status vocabulary and
// required frontmatter fields track the upstream core schema — the
// exporter converts, this type stays schema-agnostic.
type ADR struct {
	ID           ADRID        `json:"id"`
	Title        Name         `json:"name"`
	Version      Version      `json:"version"`
	Status       ADRStatus    `json:"status"`
	Date         *time.Time   `json:"date,omitempty"`
	Summary      Summary      `json:"summary,omitempty"`
	Deciders     []string     `json:"deciders,omitempty"`
	Consulted    []string     `json:"consulted,omitempty"`
	Informed     []string     `json:"informed,omitempty"`
	Supersedes   []ADRID      `json:"supersedes,omitempty"`
	SupersededBy []ADRID      `json:"supersededBy,omitempty"`
	Body         string       `json:"body,omitempty"`
	Owners       []string     `json:"owners,omitempty"`
	Badges       []Badge      `json:"badges,omitempty"`
}
```

Wiring additions (all mechanical, following `AddFlow` as the template):
`ADRID` branded string in `catalogid.go`'s file (NOT the frozen `cId`
namespace — that one stays untouched), `Catalog.ADRs` slice + registry
`AddADR`, `Builder.AddADR` forwarder, `copyADR`/`sortedCopy` in
`registry_copy.go`, `Format contract` table row in README.

## 3. Exporter acceptance criteria (definition of done)

1. Emission dir `adrs/<id>/index.mdx`; versioned dirs follow the message
   pattern (`adrs/<id>/<version>/index.mdx`) IF the upstream schema versions
   ADRs — T7.1 decides; otherwise single-dir.
2. Frontmatter carries ONLY schema-known fields plus `x-*` extensions
   (unknown top-level keys are rejected by core — the existing format
   contract rule). Anything we cannot map rides `x-go-*`.
3. `Body` renders as the page Markdown below the frontmatter fence.
4. Cross-references (`supersedes`/`supersededBy`) emit as pointer objects
   `{id, version?}` — never plain strings (pointer-list contract).
5. `WithPlainRefIDs` behavior defined and pinned (lint profile must pass
   `refs/resource-exists` at full severity).
6. Golden coverage: `testdata/golden/adr.snap` in the eventcatalog suite
   (sacred regen procedure applies); ec-fixture gains one ADR exercising
   every field class; `check-eventcatalog` stays green.
7. Gate additions: the render-gate semantic checks gain one ADR assertion
   (page renders + supersedes link resolves).
8. Docs: README export-table row, catalog/AGENTS.md contract bullet,
   FEATURES row, modules.md/skill references if consumer-visible.

## 4. Explicit non-goals (this sketch)

- No `catalog.ADR` code, no `adrs/` emission, no goldens until T7.1 lands.
- No ADR EXTRACTION from docs/adr/ (parsing our 150+ ADR markdown files
  into `catalog.ADR` values is a separate consumer-side decision — likely a
  hub-side merge script or a small cmd in this repo, decided when the
  schema is real).
- No change to the frozen `cId` namespace.
