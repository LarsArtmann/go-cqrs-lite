# catalog — Module Contracts

Consumer-facing docs: [`README.md`](README.md). This file records maintainer-only
contracts and gotchas. See also root [`AGENTS.md`](../AGENTS.md).

## docserver UI (templ-components)

HTML pages are written in `docserver/*.templ` and code-generated to `*_templ.go`,
which are **committed**. Consumers of `catalog` need no templ CLI.

### Regenerate after editing a .templ file

```bash
cd catalog && GOWORK=off go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate ./docserver/...
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
- **Verified format contract (against core 4.6.3 zod schemas, 2026-09-23).**
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
  - `styles` node color/label nest under `styles.node.{color,label}`.
- **Deliberately not exported:** `BaseConfig.ResourceGroups` (export fails with
  a Rejection — EventCatalog needs typed `{id, version, type}` items the
  string-based `catalog.ResourceGroup.Items` cannot express),
  `BaseConfig.DetailsPanel` (no equivalent; EventCatalog's is a per-section
  `{visible}` map), and `FlowStep.Channel` (no channel step kind exists).
- **Known upstream bug (core 4.6.3):** enabling `changelog` in
  `eventcatalog.config.js` crashes the build on ANY agent changelog page
  (`getBadgeHref` on an undefined badge — with or without a changelog file).
  `shouldEnableChangelog` (writer.go) keeps the flag off when the catalog has
  agents; drop that guard (and the ec-fixture changelog profile) when bumping
  past a fixed core release.
- Custom docs export to `docs/<slug>/index.mdx`: collected by the community
  build (and included for enterprise rendering), but standalone custom pages
  are an EventCatalog enterprise feature — community builds do not route them.
- The render gate (`nix run .#check-eventcatalog`) builds TWO fixture profiles
  (default + changelog) and asserts schema-clean logs plus semantic artifacts
  (channel message links, rendered changelog page). Extend `cmd/ec-fixture`
  when adding new exported frontmatter.

## Golden tests

`testdata/golden/` is shared by every package's suite. `UPDATE_SNAPS=true go
test ./<pkg>/...` treats snaps owned by OTHER packages as obsolete and DELETES
them. Always regen module-wide: `UPDATE_SNAPS=true go test ./...`.

## Budgets

- `DEP_BUDGET[catalog]=5` and it is FULL: go-faster/yaml, go-error-family,
  go-snaps (via the cattest golden helpers), templ, templ-components. Any new
  production dependency needs an explicit budget review.
- test-only deps (go-snaps, ginkgo, gomega) don't count.
