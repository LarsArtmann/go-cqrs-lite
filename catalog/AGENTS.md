# catalog — Module Contracts

Consumer-facing docs: [`README.md`](README.md). This file records maintainer-only
contracts and gotchas. See also root [`AGENTS.md`](../AGENTS.md).

## docserver UI (templ-components)

HTML pages are written in `docserver/*.templ` and code-generated to `*_templ.go`,
which are **committed**. Consumers of `catalog` need no templ CLI.

### Regenerate after editing a .templ file

```bash
cd catalog/docserver && GOWORK=off go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate
```

- The CLI version MUST match the `a-h/templ` pin in `go.mod` (v0.3.1020). A newer
  CLI (e.g. system 0.3.1036) churns every generated file.
- `*_templ.go` are excluded from treefmt (`nix fmt`) and golangci formatters —
  never hand-format them; regenerate instead. Hand-written files still get gofumpt.
- templ syntax rules learned the hard way:
  1. Component calls must be single-line: `@display.Card(cardProps)`. Multi-line
     calls produce broken generated code.
  2. Build complex props in a `{{ }}` Go block before the call.
  3. `if` / `for` inside element bodies take NO `@` prefix.
  4. Script bodies are raw text: `{ expr }` does NOT interpolate. Pass values via
     `data-attr={ value }` and read `el.dataset.attr` in JS.
- templ-components gotchas:
  - `layout.ContainerProps{...}` literals default `Pad` to Go's zero (false) —
    the documented "Pad=true default" only applies via `DefaultContainerProps()`.
    A literal without `Pad: true` silently drops the page gutter. All docserver
    usages pass `Pad: true` explicitly (2026-10-01 audit).
  - Disable htmx with `props.HTMXSrc = layout.HTMXNone` (v1.19.3+) — never clear
    `HTMXVersion`/`HTMXResponseTargets` by hand; the sentinel suppresses script,
    preconnect, and the response-targets extension in one switch.
  - Regenerate from `catalog/docserver` as the cwd (command above): templ bakes
    the invocation cwd into the FileName metadata and `check-templ`'s tripwire
    fails on path-carrying names (`./docserver/...` from `catalog/` was the old
    broken form).

### Stylesheet

- `static/docs-ui.src.css` is the SOURCE. `static/docs-ui.css` is a BUILT artifact
  (tailwind v4, minified) — never edit it directly.
- Rebuild after changing the source or bumping templ-components:
  `nix run .#build-docserver-css`
- Drift is gated: `nix run .#check-docserver-css` runs inside `#verify`/`#verify-fast`.
- The build script resolves templ-components sources from GOMODCACHE and uses
  nixpkgs `tailwindcss_4` (4.3.3 — same version templ-components itself builds with).

### Routes

No subtree catch-all: unknown `/docs/*` paths 404. Exact `/docs/` redirects to
`/docs`. SPA pages use absolute DocsPath-anchored asset URLs plus `<noscript>`
fallbacks — keep both when adding pages.

## EventCatalog exporter

- Output layout: messages deduplicated into top-level `commands|events|queries/`;
  data stores go to `containers/` (EventCatalog 4.x); config carries a stable `cId`.
- The `cId` is a UUIDv5 of the catalog title under the fixed namespace literal in
  `catalogid.go`. Changing that namespace (or the derivation) changes EVERY
  catalog's identity — treat it as frozen. (The original hex-parsing helper
  silently zeroed the dash characters; the byte literal replaced it 2026-08-16.)
- `@eventcatalog/core` is pinned via the `eventCatalogCoreVersion` constant
  (exporter.go). Bump only after verifying the new release renders the generated
  MDX; then regen goldens.
- **Upstream v5 licence shift (announced 2026-09-24):** EventCatalog v5 is BSL
  1.1 — source-available, free for individuals/OSS/orgs under $10M revenue
  AND under $10M raised (self-certified; Free Use for LarsArtmann CONFIRMED
  in the hub's 2026-10-15 audit); larger orgs pay per team. v4 stays MIT but
  got critical-security fixes only until ~2027-03-24 (6 months post-v5),
  then EOL. Offsetting: v5 is one product — enterprise features fold in,
  including routing standalone custom docs pages (verified: v5 community
  builds route them at `/docs/custom/<slug>/`).
  **BUMPED 2026-10-15: the pin is `^5.1.0`** (spike-verified end-to-end;
  procedure + evidence in the pin-bump bullet below).
- **First-party v5 consumer exists: eventcatalog-hub (2026-10-15).** The hub
  runs core `^5.0.1` and builds our v4-format exported trees under its
  link-validation-as-CI-gate — empirical evidence that v5 parses our emitted
  frontmatter. Any pin bump here is a COORDINATED cross-repo change (hub
  guardrail: exporter + pin + lockfile + full rebuild, together — never a
  casual `npm update` on either side). Agent surface is NOT bump-gated: the
  built-in v5 MCP server requires SSR (NO-GO for static serving), while the
  standalone `@eventcatalog/mcp-server` consumes the `llms.txt` this
  exporter already emits.
- **Verified format contract (against core 4.6.3 zod schemas 2026-09-23;
  re-verified against 5.1.1 by the 2026-10-15 render-gate spike — every
  contract point below builds clean under v5).**
  The exporter must keep honoring these or the downstream `eventcatalog build`
  hard-fails (`InvalidContentEntryDataError`) or silently strips data:
  - Pointer lists (`entities`, `flows`, `domains`, `data-products`, `sends`,
    `receives`, `writesTo`, `readsFrom`) are `{id, version?}` OBJECTS — plain
    strings fail schema validation.
  - Channel `messages` pointers need `{collection, name, id, version}` —
    resolved from the catalog via `channelMessageIndex` (exporter.go).
  - Unknown top-level frontmatter keys are REJECTED unless `x-` prefixed
    (`withExtensionProperties.superRefine`): message labels/responses and team
    role/avatarUrl ride `x-*` custom properties.
  - Changelogs are `changelog.mdx` sidecar FILES (not frontmatter); domain
    ubiquitous language is a `ubiquitous-language.mdx` dictionary file;
    examples are individual `examples/example-N.json` files.
  - Badges REQUIRE `backgroundColor` + `textColor` (defaults: blue/white);
    custom docs REQUIRE `title` + `summary` and have no `id` field.
  - Flow step node keys: `container` (data stores), `flow` (sub-flows) — there
    is no channel step kind; flow actors have no `url` (externalSystem does).
  - ADRs (`adrs/<id>/index.mdx`): `status` enum and `appliesTo.type` enum
    and `date` (YYYY-MM-DD) are validated at EXPORT (Rejection errors) —
    schema verified against core 5.1.1's generated adrs collection schema
    (2026-10-15). ADR→ADR pointer lists (`supersedes`, `supersededBy`,
    `amends`, `amendedBy`, `related`) are `{id, version?}` objects like
    every other pointer list; `WithPlainRefIDs` drops their version pins.
    The current ADR version lives ONLY in the base index.mdx frontmatter —
    EventCatalog versions ADR history at `adrs/<id>/versioned/<v>/`, not
    `adrs/<id>/<v>/` (the message pattern); we emit single-version records.
  - `styles` node color/label nest under `styles.node.{color,label}`.
- **Deliberately not exported:** `BaseConfig.ResourceGroups` (export fails with
  a Rejection — EventCatalog needs typed `{id, version, type}` items the
  string-based `catalog.ResourceGroup.Items` cannot express),
  `BaseConfig.DetailsPanel` (no equivalent; EventCatalog's is a per-section
  `{visible}` map), and `FlowStep.Channel` (no channel step kind exists).
- **Known upstream bug — agent-changelog crash: FIXED in the 5.x line**
  (verified 2026-10-15 on core 5.1.1). Core 4.6.3 crashed the build on ANY
  agent changelog page (`getBadgeHref` on an undefined badge — with or
  without a changelog file); `shouldEnableChangelog` (writer.go) kept the
  flag off for agent catalogs until the 2026-10-15 v5 bump dropped the
  guard. If a future pin regresses this: symptom = build crash on
  `/docs/agents/<id>/<version>/changelog`; fix = re-add the agent exclusion
  in `shouldEnableChangelog` plus an agents-free fixture profile.
- **Known upstream WARN — 3-collection sidebar changelog links: OPEN in 5.1.1
  (cosmetic).** With `changelog` enabled, link-validation logs broken sidebar
  links `/docs/{channels,data-products,entities}/<id>/<version>/changelog`.
  The changelog page route (`src/pages/docs/[type]/[id]/[version]/changelog/_index.data.ts`
  `itemTypes`) covers only agents/events/commands/queries/services/domains/
  systems/flows/containers — but the channel/entity/data-product sidebar
  builders emit Changelog links for those 3 collections anyway. Unfixable
  from exporter output (even a `changelog.mdx` cannot make the page
  generate); the render gate must NOT fail on this WARN (the fixture logs 4
  such links across the 3 collections).
- **Pin-bump procedure (executed 2026-10-15, `^4.6.3` → `^5.1.0`; reuse for
  the next bump):** (1) scratch worktree + bump `eventCatalogCoreVersion` +
  `nix run .#check-eventcatalog` — one run answers build/zod acceptance,
  the changelog-crash question, the WARN signature, and custom-docs
  routing; (2) force `changelog: { enabled: true }` into the generated
  `eventcatalog.config.js` and rebuild to exercise the agent-changelog path
  BEFORE touching the guard; (3) check custom-docs routing in dist
  (`/docs/custom/<slug>/`); (4) land pin + goldens + guard/profile changes
  TOGETHER as one decision commit (never pin without goldens); (5) re-run
  the gate from master; (6) receipt note to eventcatalog-hub (they build
  our trees under their own core pin — coordinated cross-repo change).
- Custom docs export to `docs/<slug>/index.mdx`: under core 5.x they route
  standalone at `/docs/custom/<slug>/` in community builds too (verified
  2026-10-15 on 5.1.1; under 4.6.3 only enterprise routed them).
- `llms.txt` (`writer_llms.go`) is the standalone `@eventcatalog/mcp-server`'s
  ONLY catalog input (the built-in v5 MCP server needs SSR — a no-go for
  static serving); its structure is a CONTRACT for agent consumers. Treat
  format changes as breaking (the writer's godoc says the same).
- The render gate (`nix run .#check-eventcatalog`) builds ONE fixture
  profile (default: every resource kind, agents included, changelog on)
  and asserts schema-clean logs plus semantic artifacts (channel message
  links, rendered changelog + agent pages). Extend `cmd/ec-fixture`
  when adding new exported frontmatter.
- **Embedded serving** — `StaticServer` (`server.go`) serves a BUILT
  `dist/` tree from any `fs.FS` (consumers `//go:embed` it), so a site needs
  no Node at deploy time. It routes Astro's directory-index output, redirects
  slash-less directory paths, resolves extensionless `.html`, serves a custom
  `404.html`, and marks `_astro/` immutable; it never lists directories. It is
  stdlib-only (no new deps) and lives in `eventcatalog`, not `docserver`, so
  importers don't pull templ. It complements, not replaces, the native
  `docserver` EventCatalog view (which renders from the live catalog object).
  Its behavior is pinned by the package's Ginkgo BDD suite
  (`eventcatalog_bdd_suite_test.go` bootstrap + `server_bdd_test.go` specs,
  black-box `package eventcatalog_test`) — extend the specs there rather than
  adding plain table tests.

## architecture — self-contained HTML reference

`catalog/architecture` renders the whole catalog as ONE `.html` file (the
EventCatalog 5.1 "architecture reference"). It is deliberately separate from
both `eventcatalog` (MDX for the Node CLI) and `docserver` (multi-page
server-rendered HTML).

- **No new dependencies, no templ.** The package uses only the standard
  library plus `go-error-family`. Keep it that way: the catalog module's
  dependency budget is FULL (`DEP_BUDGET[catalog]=5`), and the point of this
  package is that a service which never serves HTML can still publish a
  reference without pulling in a UI stack. Assets live in
  `architecture/assets/` (CSS/JS) and are inlined via `//go:embed`.
- **Anchors are derived, never stored.** A section carries `kind` + resource
  `id`; `section.anchor()` computes `kindInfo.anchor(id)` (`slug` lowercases
  and replaces anything outside `[a-z0-9_-]`). Sidebar, search index and
  cross-links all call the same function, so a link cannot drift from its
  target. Adding a resource kind means adding a row to `kindTable` (group,
  label, tag, CSS class, anchor prefix) and a section builder — nothing else.
- **Escaping is single-path.** All catalog text goes through
  `htmlWriter.text`/`htmlAttr`; `markup` values are only produced by a
  writer. Prose is rendered by the mini markdown in `markdown.go` +
  `markdown_inline.go`, which deliberately supports no raw HTML, and
  `safeHref` allow-lists `http(s)`, `mailto:`, `data:image/`, `#` and `/`.
  Badge colors pass `safeColor`. When adding a renderer, never interpolate a
  string with a literal `+` into HTML; use the writer.
- **Search index is a published shape.** The `searchEntry` JSON keys
  (`g/k/c/t/a/s/d/h`) are read by `assets/reference.js` and inlined into
  every generated file. Change them only together.
  `TestSearchIndexKeyContractPinned` (surfaces_test.go) pins the key set;
  the anchor↔section tests in anchor_test.go pin navigation integrity.
- **`WithLogo` renders, it does not decorate.** Decision (2026-10-17): the
  option renders an `<img class="brand-logo">` in the top bar when the URL
  passes `safeImageSrc` (http/https/data:image only — a hostile logo is
  dropped entirely, not rewritten). Tables share one `writeTable` helper
  (`table.go`); new table-shaped blocks extend it instead of copying the
  thead/tbody ladder.
- **The docs example is gated.** The recipe (§2.48 in
  `.agents/skills/go-cqrs-lite/references/recipes.md`) is compile-verified by
  `cmd/doc-check`; its catalog entry lives in
  `cmd/doc-check/recipes_catalog_meta4.go` (`recipeCatalogB4`), wired into
  `recipes_compile_test.go`. Reword the fence and the gate fails until the
  entry is updated.
- **`catalog-export -architecture <path>`** writes the reference next to the
  EventCatalog MDX tree — the copy-paste template consumers start from. The
  CLI also carries the selection flags (`-include/-exclude/-never-export
  kind:id`, `-references`, `-hide`, `-spec <file>`) and `-check`, which
  prints `Exporter.Inspect`'s report and exits 1 when a selector matched
  nothing — a redaction that matches nothing is a silent leak risk, so it
  fails loudly.
- **Selections (ADR-0160) are a kept-set with three layers.** `Selection.plan`
  computes `kept` + `banned` + per-resource reasons + not-found selectors.
  Excludes win over includes inside an included container and never cascade;
  a domain selector brings its services and those services' messages (the
  natural closure); `neverExport` beats everything. References modes decide
  how out-of-scope references render: `ReferencesOmit` (default,
  byte-identical to no selection), `ReferencesNameOnly` (degrade to plain
  text names), `ReferencesInclude` (pull one level of pointed-at resources
  in). `WithHide` suppresses whole reference parts (`owners`, `repository`,
  `examples`, `specifications`, `docs`) at the resolver choke points —
  `idx.owners()` returns nil, `collectDocs` returns early — so all render
  sites disappear together. There is deliberately NO `contacts` constant:
  no model surface exists for it (the ADR-0160 `x-*` precedent).
- **Redaction is layered, and the leak test pins the composition.** A banned
  resource's name never enters `refIndex.names`/`namesByID` (the recording
  guard), the degrade paths re-check `banned`/`bannedIDs` (defense in
  depth), and `writeRef` skips zero references entirely.
  `TestExcludedResourceNameNeverAppears` (frontmatter_test.go) fails only
  when ALL layers are disabled — mutation-verified. When you touch the name
  tables or degrade paths, re-run that test.
- **Selector kind spellings live in `adrKind` (section_meta.go).** The
  selector/doc/ADR-target spellings are one vocabulary: `domain`, `service`,
  `command`, `event`, `query`, `channel`, `container` (data store), `flow`,
  `entity`, `data-product`, `agent`, `decision` (ADR), `team`, `person`
  (user), `doc`. Internal kinds spell differently (`data-store`, `adr`) —
  `selectorSpelling` maps between them; `edges()` emits selectors, so a new
  edge kind MUST round-trip through `adrKind` or it is silently dropped.
  `exhaustive` forces every new `resourceKind` through `exists`/`closure`/
  `edges`/`selectorSpelling` — that is registration safety, not noise.
- **`exhaustruct_v5` demands every field, even zero values, on own-package
  literals.** A bare `visibility{}` or `resourceRef{}` fails lint; write the
  full literal (see `visibilityFrom`, `droppedRef`) or a constructor. When a
  struct grows a field, every explicit literal breaks — that is the point
  (registration safety), do not add the struct to ignore-patterns.
- **Decisions recorded (2026-10-18):** (a) NO committed demo artifact — the
  file is regenerable in ~1 s via `go run ./example/architecture`, a 50 KB
  committed blob would rot against the code, and the fixture gate
  (`check-architecture-reference`) already validates the render; (b) NO
  `simple.ExportArchitecture` facade — `NewExporter(...).Export(cat)` IS the
  one-liner, a wrapper would mirror every option for no ceremony win, and
  `simple` is catalog construction, not export plumbing; (c) recipe-gate
  cold start (~70–100 s: proxy fetch + CGO-flag cache split) stays a
  DOCUMENTED DEFERRAL — warm runs are 5–15 s and the gate is offline-only.
- `catalog/.go-arch-lint.yml` registers the package as its own component that
  may depend on `core` only. A new intra-module import needs a matching
  `mayDependOn` entry.

## Golden tests

`testdata/golden/` is shared by FOUR package suites (asyncapi, d2,
eventcatalog, openapi), but each package runs as its own test binary and
go-snaps `Clean` DELETES every `.snap` in the shared dir that its binary did
not touch. So ANY scoped `UPDATE_SNAPS=true go test ./<pkg>/...` deletes the
other three packages' snaps — and module-wide `go test ./...` does NOT save
you (four binaries each clean the others; survivors are nondeterministic).
Safe procedure (regen scoped, then restore the cross-deletions):

    UPDATE_SNAPS=true go test ./eventcatalog/...
    git ls-files -d -- testdata/golden | xargs -r git restore --

If you intentionally removed a golden test, delete its `.snap` explicitly
afterwards (the restore line would otherwise bring it back).

## Budgets

- `DEP_BUDGET[catalog]=5` and it is FULL: go-faster/yaml, go-error-family,
  go-snaps (via the cattest golden helpers), templ, templ-components. Any new
  production dependency needs an explicit budget review.
- test-only deps (go-snaps, ginkgo, gomega) don't count.
