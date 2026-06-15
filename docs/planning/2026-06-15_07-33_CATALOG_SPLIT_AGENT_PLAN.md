# AI Agent Plan: Catalog Exporter Module Split

> **For:** AI coding agent (Crush, Claude, GPT, etc.) · **Estimated time:** 2-3 hours
>
> **Goal:** Split 5 catalog exporter sub-packages into independent Go modules so consumers
> can import only the formats they need without transitively depending on all exporters.

## Background

The `catalog/` module currently contains 6 sub-packages: `asyncapi/`, `openapi/`, `d2/`,
`eventcatalog/`, `docserver/`, and `schema/`. Each exporter depends only on the catalog
core types (`Registry`, `Builder`, `Catalog`, `Message`, etc.) and 1-2 external libs.

A consumer who only needs AsyncAPI export currently gets D2, EventCatalog, and docserver
code transitively. Splitting into separate modules lets consumers import exactly what they need.

**CRITICAL CONSTRAINT:** The import paths must NOT change. `catalog/v2/asyncapi` stays
`catalog/v2/asyncapi`. Only the `go.mod` structure changes — each sub-package gets its own
`go.mod` that depends on `catalog/v2` core. This is transparent to consumers.

## Current State

```
catalog/
├── go.mod                    ← module github.com/larsartmann/go-cqrs-lite/catalog/v2
├── registry.go               ← core: Registry
├── build.go                  ← core: Builder
├── types.go                  ← core: Catalog, Service, Message, etc.
├── validate.go               ← core: validation
├── message_config.go         ← core: message configuration
├── auto_name.go              ← core: auto-naming from Go types
├── exporter.go               ← core: Exporter[T] interface
├── asyncapi/
│   └── *.go                  ← depends on catalog core + go-faster/yaml
├── openapi/
│   └── *.go                  ← depends on catalog core
├── d2/
│   └── *.go                  ← depends on catalog core
├── eventcatalog/
│   └── *.go                  ← depends on catalog core
├── docserver/
│   └── *.go                  ← depends on catalog core + embed.FS
└── schema/
    └── *.go                  ← depends on nothing external (JSON Schema types)
```

## What NOT to Split

- **`catalog/schema/`** — shared JSON Schema types used by all exporters. Stays in catalog core.
- **`catalog/internal/`** — internal helpers. Stays in catalog core.

## Step-by-Step Plan

### Step 1: Split catalog/asyncapi → own module (30min)

**1.1** Create `catalog/asyncapi/go.mod`:
```
module github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi

go 1.26.3

require (
    github.com/larsartmann/go-cqrs-lite/catalog/v2 v2.3.0
    github.com/go-faster/yaml v0.x.x
)

replace github.com/larsartmann/go-cqrs-lite/catalog/v2 => ../
```

**1.2** Add `./catalog/asyncapi` to `go.work` in the root.

**1.3** Remove asyncapi deps from catalog core `go.mod` (go-faster/yaml if only asyncapi uses it).

**1.4** Run `cd catalog/asyncapi && GOWORK=off go mod tidy`.

**1.5** Verify: `cd catalog/asyncapi && GOWORK=off go build ./...` and `GOWORK=off go test ./...`.

**1.6** Verify root build: `nix run .#build` and `nix run .#test`.

**1.7** Commit: `feat(catalog): split asyncapi exporter into independent module`

### Step 2: Split catalog/openapi → own module (30min)

Same pattern as Step 1, but for `catalog/openapi/`.

**2.1** Create `catalog/openapi/go.mod` (depends on `catalog/v2`, replace `=> ../`).
**2.2** Add `./catalog/openapi` to `go.work`.
**2.3** Run `go mod tidy` in the new module.
**2.4** Verify build + test.
**2.5** Commit: `feat(catalog): split openapi exporter into independent module`

### Step 3: Split catalog/d2 → own module (30min)

Same pattern.

**3.1** Create `catalog/d2/go.mod` (depends on `catalog/v2`, replace `=> ../`).
**3.2** Add `./catalog/d2` to `go.work`.
**3.3** Run `go mod tidy`.
**3.4** Verify build + test.
**3.5** Commit: `feat(catalog): split d2 exporter into independent module`

### Step 4: Split catalog/eventcatalog → own module (30min)

Same pattern.

**4.1** Create `catalog/eventcatalog/go.mod` (depends on `catalog/v2`, replace `=> ../`).
**4.2** Add `./catalog/eventcatalog` to `go.work`.
**4.3** Run `go mod tidy`.
**4.4** Verify build + test.
**4.5** Commit: `feat(catalog): split eventcatalog exporter into independent module`

### Step 5: Split catalog/docserver → own module (30min)

Same pattern. Note: docserver has `embed.FS` assets — make sure those stay with the module.

**5.1** Create `catalog/docserver/go.mod` (depends on `catalog/v2`, replace `=> ../`).
**5.2** Add `./catalog/docserver` to `go.work`.
**5.3** Run `go mod tidy`.
**5.4** Verify build + test.
**5.5** Commit: `feat(catalog): split docserver exporter into independent module`

### Step 6: Final verification + cleanup (30min)

**6.1** Run full suite: `nix run .#build && nix run .#test && nix run .#lint`.
**6.2** Verify `GOWORK=off` builds work for ALL split modules:
```bash
for mod in catalog/asyncapi catalog/openapi catalog/d2 catalog/eventcatalog catalog/docserver; do
    echo "=== $mod ==="
    (cd $mod && GOWORK=off go build ./... && GOWORK=off go test ./...)
done
```

**6.3** Clean up catalog core `go.mod` — remove deps only needed by exporters:
```bash
cd catalog && GOWORK=off go mod tidy
```

**6.4** Update `.go-arch-lint.yml` if it references catalog sub-packages.
**6.5** Update `scripts/check-module-layers.sh` if needed.
**6.6** Update `flake.nix` `testModules` list to include the new modules.
**6.7** Commit: `chore(catalog): final cleanup after exporter split`
**6.8** Push.

## Safety Rules

1. **Build must pass after every step** — `nix run .#build`.
2. **Tests must pass after every step** — `nix run .#test`.
3. **Lint must pass after every step** — `nix run .#lint`.
4. **`GOWORK=off` builds must work** for each new module — this is what consumers experience.
5. **Import paths must NOT change** — `catalog/v2/asyncapi` stays the same.
6. **One module per commit** — if something breaks, it's revertable.
7. **Commit message format:** `feat(catalog): split <name> exporter into independent module`.

## Checklist (print and tick off)

```
[ ] catalog/asyncapi/go.mod created
[ ] catalog/asyncapi in go.work
[ ] catalog/asyncapi builds with GOWORK=off
[ ] catalog/asyncapi tests pass with GOWORK=off

[ ] catalog/openapi/go.mod created
[ ] catalog/openapi in go.work
[ ] catalog/openapi builds with GOWORK=off
[ ] catalog/openapi tests pass with GOWORK=off

[ ] catalog/d2/go.mod created
[ ] catalog/d2 in go.work
[ ] catalog/d2 builds with GOWORK=off
[ ] catalog/d2 tests pass with GOWORK=off

[ ] catalog/eventcatalog/go.mod created
[ ] catalog/eventcatalog in go.work
[ ] catalog/eventcatalog builds with GOWORK=off
[ ] catalog/eventcatalog tests pass with GOWORK=off

[ ] catalog/docserver/go.mod created
[ ] catalog/docserver in go.work
[ ] catalog/docserver builds with GOWORK=off
[ ] catalog/docserver tests pass with GOWORK=off

[ ] Full suite: nix run .#build && nix run .#test && nix run .#lint — ALL PASS
[ ] catalog core go.mod cleaned (go mod tidy)
[ ] flake.nix testModules updated
[ ] All changes committed and pushed
```
