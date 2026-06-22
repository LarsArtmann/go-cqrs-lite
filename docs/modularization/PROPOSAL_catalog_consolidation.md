# Catalog Sub-Module Consolidation Proposal

> **Date:** 2026-06-16 | **Status:** Active
> **Scope:** Merge 5 catalog sub-modules back into the single `catalog` module as packages.

## Executive Summary

The `catalog/` directory is split into **6 `go.mod` files** (`catalog` + 5 sub-modules:
`asyncapi`, `eventcatalog`, `d2`, `openapi`, `docserver`). This split delivers **zero**
of the benefits that justify multi-module layout, while imposing real ongoing costs.
The fix is mechanical and **non-breaking for consumers**: collapse the 5 sub-modules
into packages within the single `catalog` module. Import paths are preserved because a
package at `catalog/d2/` inside module `catalog/v2` has the identical import path
`github.com/larsartmann/go-cqrs-lite/catalog/v3/d2` as the current standalone module.

**Why now:** The sub-modules have **zero release tags** (only `catalog/v2.3.0` exists;
no `catalog/d2/v*` was ever tagged) and are absent from `release.yml`. They are
**unpublishable** as-is. Merging them is the only way consumers can ever import them.

---

## Current State Analysis

### The 6 modules

| Module path               | Source files                   | Unique deps beyond `catalog/v2`                          | Tags                   | Replace directives                        |
| ------------------------- | ------------------------------ | -------------------------------------------------------- | ---------------------- | ----------------------------------------- |
| `catalog/v2` (parent)     | 15 + 8 (schema) + 7 (internal) | —                                                        | 15 (`v0.1.0`–`v2.3.0`) | none                                      |
| `catalog/v2/d2`           | 4                              | **none**                                                 | **0**                  | 1 (self → parent)                         |
| `catalog/v2/asyncapi`     | 5                              | `go-faster/yaml` (parent already has it)                 | **0**                  | 1 (self → parent)                         |
| `catalog/v2/openapi`      | 5                              | `go-faster/yaml` (parent already has it)                 | **0**                  | 1 (self → parent)                         |
| `catalog/v2/eventcatalog` | 9                              | **none**                                                 | **0**                  | 1 (self → parent)                         |
| `catalog/v2/docserver`    | 4                              | siblings `asyncapi` + `openapi` (both subsets of parent) | **0**                  | **3** (parent + 2 siblings, all `v0.0.0`) |

### Proof there is no isolation

Every sub-module's direct dependency set is a **strict subset** of the parent
`catalog/v2` module's dependency closure. A consumer importing `catalog/v2/d2` today
gets the **exact same transitive dependency tree** as importing `catalog/v2`. The
module boundaries buy nothing.

### Proof they are not really separate modules

- All 5 sub-modules import `catalog/v2/internal/cattest` and/or
  `catalog/v2/internal/caseutil`. The `internal/` rule permits this only because the
  import path prefix `catalog/v2/` is shared — an **accident of naming**, not a real
  boundary. They depend on the parent's internals.
- `docserver` imports siblings `catalog/v2/asyncapi` and `catalog/v2/openapi` at
  placeholder version `v0.0.0` (because no real versions exist) via local `replace`
  directives. This is a 3-way replace tangle for one logical concern.

### Costs being paid

| Cost                           | Detail                                                                                                                                    |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------- |
| **Unpublishable**              | 0 tags on any sub-module; absent from `release.yml`. External consumers literally cannot `go get` them.                                   |
| **5 redundant CI invocations** | `flake.nix` `testModules` lists all 5 separately; each runs `go test` with the same dep closure as the parent.                            |
| **6 go.mod to maintain**       | Dependency bumps must be applied 6× instead of 1×.                                                                                        |
| **3-way replace tangle**       | `docserver/go.mod` has 3 replace directives with `v0.0.0` placeholders.                                                                   |
| **Self-referential replace**   | Each child has `replace github.com/larsartmann/go-cqrs-lite/catalog/v3 => ../` — pointing at the thing it's pretending not to be part of. |
| **5 import paths to memorize** | Consumers must learn `catalog/v2/d2`, `catalog/v2/asyncapi`, … when they're all one product.                                              |

### External consumers

Only **one** file outside `catalog/` imports the sub-modules:

```
./example/user/catalog.go:
  catalog/v2/asyncapi
  catalog/v2/d2
  catalog/v2/eventcatalog
```

Import paths are unchanged by the merge, so this file needs **zero edits**.

---

## Proposed Structure

### Single module, packages preserved

```
catalog/                          # ONE module: github.com/larsartmann/go-cqrs-lite/catalog/v3
├── go.mod                        # only go.mod in the tree
├── *.go                          # package catalog (Registry, SchemaFromType, etc.)
├── schema/                       # package catalog/schema   (unchanged)
├── internal/                     # package catalog/internal/* (unchanged)
├── d2/                           # package catalog/d2       (was module, now package)
├── asyncapi/                     # package catalog/asyncapi (was module, now package)
├── openapi/                      # package catalog/openapi  (was module, now package)
├── eventcatalog/                 # package catalog/eventcatalog (was module, now package)
└── docserver/                    # package catalog/docserver (was module, now package)
```

**Import path preservation** — the critical property:

| Before (separate module)                                  | After (package in catalog module)                         |
| --------------------------------------------------------- | --------------------------------------------------------- |
| `github.com/larsartmann/go-cqrs-lite/catalog/v2/d2`       | `github.com/larsartmann/go-cqrs-lite/catalog/v3/d2`       |
| `github.com/larsartmann/go-cqrs-lite/catalog/v2/asyncapi` | `github.com/larsartmann/go-cqrs-lite/catalog/v3/asyncapi` |
| ...                                                       | (identical for all 5)                                     |

A package at path `catalog/d2/` inside module `catalog/v2` resolves to import path
`catalog/v2/d2`. **Consumers see no change.** This is a non-breaking refactor.

### DAG

Unchanged — `catalog` already has zero internal module dependencies. The sub-modules
depend _upward_ on the parent (via internal/ and schema), which is only legal because
of the shared path prefix. After merge, these become intra-module package imports —
legal and natural.

---

## Breaking Change Analysis

**External consumers:** Non-breaking. Import paths are identical. The only behavioral
change is that the 5 sub-module paths become **publishable for the first time** once
tagged under `catalog/v2.x.x`.

**Internal consumers:** `example/user/catalog.go` needs zero edits (same import paths).

**Build system:** `flake.nix` and CI become simpler — remove 5 entries from
`testModules`; the single `catalog` entry already covers all sub-packages via
`./catalog/...`.

**Versioning:** The 5 sub-modules had no tags and no consumers, so no version migration
is needed. The next `catalog/v2.x.x` tag implicitly includes the packages.

---

## Migration Strategy

5 mechanical steps, each independently revertable. See
[`EXECUTION_PLAN_catalog_consolidation.md`](./EXECUTION_PLAN_catalog_consolidation.md).

1. Remove `catalog/d2/go.mod` + `go.sum`; delete its self-referential `replace`.
2. Remove `catalog/asyncapi/go.mod` + `go.sum`.
3. Remove `catalog/openapi/go.mod` + `go.sum`.
4. Remove `catalog/eventcatalog/go.mod` + `go.sum`.
5. Remove `catalog/docserver/go.mod` + `go.sum` (the 3-way replace tangle).
6. `go work sync` + `go mod tidy` in `catalog/` + update `flake.nix` (drop 5 entries).
7. Build + test + verify.

No source files change. Only `go.mod`/`go.sum` files are deleted and `go.work` /
`flake.nix` are updated.

---

## Risk Assessment

| Risk                                            | Likelihood                                                                                    | Mitigation                                                   |
| ----------------------------------------------- | --------------------------------------------------------------------------------------------- | ------------------------------------------------------------ |
| Import paths break                              | **Impossible** — package path = module path + dir, preserved by construction                  | Verify with `go build ./catalog/...` after each step         |
| `internal/` access breaks                       | None — access was already intra-path-prefix; now it's intra-module (strictly more permissive) | Covered by existing tests                                    |
| Hidden external consumer on a tagged sub-module | None — zero tags exist for any sub-module                                                     | Verified via `git tag -l 'catalog/*/v*'` → 0 results         |
| `go.work` drift                                 | Low                                                                                           | `go work sync` + commit `go.work` and `go.work.sum` together |

---

## Why This Is the Right Fix (and What We Rejected)

| Alternative                                 | Why rejected                                                                                         |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Keep sub-modules, add tags + release wiring | Preserves 6× maintenance cost for zero isolation benefit. The split is the problem, not the tagging. |
| Merge into fewer sub-modules (e.g. 2)       | Still pays multi-module overhead for no gain. One module is correct.                                 |
| Move exporters into `catalog/internal/`     | Would hide them from consumers — they're public API.                                                 |

**Litmus test (Unix philosophy):** Does `catalog` do one thing well? Yes — it exports
catalog documentation in multiple formats. The formats are facets of one concern, not
independent modules. They compose like pipes (each exporter takes a `Registry` and
emits bytes), which is exactly right for packages within one module.
