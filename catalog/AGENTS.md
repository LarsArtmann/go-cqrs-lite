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
- `@eventcatalog/core` is pinned via the `eventCatalogCoreVersion` constant
  (exporter.go). Bump only after verifying the new release renders the generated
  MDX; then regen goldens (`UPDATE_SNAPS=true go test ./...` from the module root —
  NOTE: run it on the whole module, the golden dir is shared across packages and a
  single-package update run deletes other packages' snaps).

## Budgets

- `DEP_BUDGET[catalog]=5` and it is FULL: go-faster/yaml, go-error-family,
  go-snaps (via the cattest golden helpers), templ, templ-components. Any new
  production dependency needs an explicit budget review.
- test-only deps (go-snaps, ginkgo, gomega) don't count.
