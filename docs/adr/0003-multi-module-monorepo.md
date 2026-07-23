# ADR-0003: Multi-Module Monorepo Structure

**Status:** Accepted  
**Date:** 2026-05-03

## Context

The library has 9 modules (`core`, `memory`, `catalog`, `middleware`, `testhelpers`, `integration`, `storage`, `projection`, plus `example/user`). Each module has its own `go.mod` with independent dependency management.

> **Historical note (2026-07-23):** This was the original module count. The
> project has since grown to 55 modules. See `AGENTS.md` for the current
> structure. This ADR documents the foundational decision; the module count
> is historical.

The alternative is a single module with all packages. This was considered and rejected.

## Decision

Use Go workspace (`go.work`) with independent `go.mod` per module.

**Dependency rules:**

- `core` has zero internal dependencies — independently publishable
- `memory`, `catalog`, `middleware`, `testhelpers`, `storage`, `projection` depend on `core`
- `integration` depends on `core` + `memory` + `testhelpers`
- `example/user` depends on `core` + `memory` + `catalog` + `middleware`
- No circular dependencies between modules

**Each module can be published independently** — consumers import only what they need (e.g., `go-cqrs-lite/core` without pulling in `storage`'s SQL dependencies).

## Consequences

**Positive:**

- Minimal dependency footprint per consumer — don't pull `go-sqlmock` if you only need `core`
- Independent versioning possible — `core` can reach v1.0 before `storage`
- Clean dependency graph enforced by `go.work`
- `core` is independently publishable — the most stable module

**Negative:**

- More `go.mod` files to maintain (55 modules as of 2026-07)
- Dependency bumps must be applied per-module
- CI must test each module independently AND via `go.work`
- Replace directives in `go.mod` files for local development

**Neutral:**

- `go.work` ties modules together for development but is not used by consumers
- Test-only dependencies (ginkgo, gomega, go-sqlmock) are isolated to their modules
