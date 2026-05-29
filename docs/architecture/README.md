# go-cqrs-lite Module Dependency Graph

## Overview

This directory contains the architecture visualization of the go-cqrs-lite multi-module workspace.

| File | Description |
|------|-------------|
| `module-graph.d2` | Source D2 diagram (editable) |
| `module-graph.svg` | Compiled SVG output (open in browser or markdown embed) |

## Layers

The modules are organized into 8 layers, from low-level primitives to high-level composition:

| Layer | Modules | Role |
|-------|---------|------|
| **0** | `codec`, `otel` | External infrastructure primitives. No internal deps. |
| **1** | `core` | The heart of the library: command, query, event, decider, id, dispatcher. |
| **2** | `testhelpers` | Test utilities (Noop/Failing/Panic handlers, FakeMetrics). Depends on `core`. |
| **3** | `memory`, `catalog`, `signing` | Thin near-core wrappers. `catalog` has zero internal deps. |
| **4** | `middleware`, `stream`, `watermill` | Mid-tier modules bridging infrastructure to core. |
| **5** | `storage`, `projection`, `pebble`, `turso` | Higher-level composition modules (SQL stores, projection runner). |
| **6** | `example/*`, `integration` | Integration tests and usage demos (compose everything). |
| **7** | `cmd/cqrs-gen` | Code generation tool. No internal deps. |

## Key Dependency Rules

1. **No cycles** — The graph is strictly acyclic (DAG)
2. **Downward only** — Higher layers only depend on lower layers
3. **`core` is the center** — Every production module transitively depends on `core`
4. **`otel` and `codec` are infra** — Only imported by modules that need them; `core` is the primary consumer of both
5. **`testhelpers` is test-only** — Imported only by `_test.go` files in most cases
6. **Examples compose freely** — They demonstrate end-to-end usage patterns

## Edge Legend

| Style | Meaning |
|-------|---------|
| Solid arrow | Production dependency (appears in source code) |
| Dashed arrow | Test-only dependency (only in `_test.go` / test modules) |
| Color | Layer identity: green = infra, blue = core, yellow = module, purple = test, pink = example |

## Generating

```bash
# Rebuild the SVG from D2 source
d2 docs/architecture/module-graph.d2 docs/architecture/module-graph.svg
```

Requires [`d2`](https://d2lang.com/) to be installed (available via `nix develop` in this project).
