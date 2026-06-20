# Execution Plan: Catalog Sub-Module Consolidation

> **Prerequisite:** Read [`PROPOSAL_catalog_consolidation.md`](./PROPOSAL_catalog_consolidation.md).
> **Branch:** `consolidate-catalog` (create before starting).
> **No source files change.** Only `go.mod`/`go.sum` deletions + `go.work`/`flake.nix` edits.

## Tasks (Pareto-ordered)

All tasks are Tier 1 (foundational) — they're all required and all low-risk mechanical
steps. Ordered to isolate each sub-module removal so a failure points to exactly one
module.

### Task 1 — Remove `catalog/d2` as a module

- [ ] `rm catalog/d2/go.mod catalog/d2/go.sum`
- [ ] `cd catalog && go mod tidy`
- [ ] `go work sync` (from repo root)
- [ ] `go build ./catalog/...` — verify `catalog/v2/d2` still resolves
- [ ] `cd catalog && GOWORK=off go test ./... -count=1` — verify in isolation
- [ ] Commit: `refactor(catalog): merge d2 sub-module into catalog module`
- [ ] **Rollback:** `git revert HEAD`

### Task 2 — Remove `catalog/asyncapi` as a module

- [ ] `rm catalog/asyncapi/go.mod catalog/asyncapi/go.sum`
- [ ] `cd catalog && go mod tidy`
- [ ] `go work sync`
- [ ] `go build ./catalog/...`
- [ ] `cd catalog && GOWORK=off go test ./... -count=1`
- [ ] Commit: `refactor(catalog): merge asyncapi sub-module into catalog module`

### Task 3 — Remove `catalog/openapi` as a module

- [ ] `rm catalog/openapi/go.mod catalog/openapi/go.sum`
- [ ] `cd catalog && go mod tidy`
- [ ] `go work sync`
- [ ] `go build ./catalog/...`
- [ ] `cd catalog && GOWORK=off go test ./... -count=1`
- [ ] Commit: `refactor(catalog): merge openapi sub-module into catalog module`

### Task 4 — Remove `catalog/eventcatalog` as a module

- [ ] `rm catalog/eventcatalog/go.mod catalog/eventcatalog/go.sum`
- [ ] `cd catalog && go mod tidy`
- [ ] `go work sync`
- [ ] `go build ./catalog/...`
- [ ] `cd catalog && GOWORK=off go test ./... -count=1`
- [ ] Commit: `refactor(catalog): merge eventcatalog sub-module into catalog module`

### Task 5 — Remove `catalog/docserver` as a module (the 3-way replace tangle)

- [ ] `rm catalog/docserver/go.mod catalog/docserver/go.sum`
- [ ] `cd catalog && go mod tidy` (absorbs asyncapi + openapi as intra-module deps)
- [ ] `go work sync`
- [ ] `go build ./catalog/...`
- [ ] `cd catalog && GOWORK=off go test ./... -count=1`
- [ ] Commit: `refactor(catalog): merge docserver sub-module into catalog module`

### Task 6 — Update workspace and build system

- [ ] Edit `go.work`: remove `./catalog/asyncapi`, `./catalog/openapi`, `./catalog/d2`,
      `./catalog/eventcatalog`, `./catalog/docserver` entries
- [ ] `go work sync` + `go work edit -fmt`
- [ ] Edit `flake.nix` `testModules`: remove the same 5 entries
- [ ] `go build ./...` (full workspace)
- [ ] `go test ./catalog/... ./example/... -count=1` — verify catalog + its only consumer
- [ ] `go vet ./catalog/...`
- [ ] Commit: `refactor(catalog): update go.work and flake.nix after consolidation`

### Task 7 — Final verification

- [ ] `find catalog -name go.mod` returns exactly **1** result (`catalog/go.mod`)
- [ ] `grep -r "replace" catalog/*/go.mod` returns nothing (no more replace directives)
- [ ] `go test ./catalog/... -count=1` passes
- [ ] `cd catalog && GOWORK=off go test ./... -count=1` passes (publishable in isolation)
- [ ] `example/user` builds and tests pass (unchanged import paths)
- [ ] `nix run .#test` passes (if available)
- [ ] Commit `go.work.sum` if it changed

## What we are NOT doing

- **Not changing any `.go` source files.** Import paths are preserved by construction.
- **Not adding deprecation notices.** The sub-modules had zero tags and zero external
  consumers — there is nothing to deprecate.
- **Not touching `catalog/schema/` or `catalog/internal/`.** They were always packages
  in the parent module and remain unchanged.
