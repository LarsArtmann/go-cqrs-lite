# ADR 0012: Split catalog into Core + Per-Exporter Modules

## Status

Proposed

## Context

The `catalog` module is the largest in the workspace at 9,319 lines of code (81 files). It bundles:

| Subsystem                               | Files |    LoC | Purpose                           |
| --------------------------------------- | ----: | -----: | --------------------------------- |
| Root (registry, types, validate, build) |    34 | ~3,033 | Core registry + SchemaFromType[T] |
| `eventcatalog/`                         |    14 | ~2,273 | EventCatalog.com exporter         |
| `schema/`                               |     7 |   ~968 | JSON Schema reflection engine     |
| `asyncapi/`                             |     6 |   ~787 | AsyncAPI 3.0 emitter              |
| `d2/`                                   |     5 |   ~674 | D2 diagram emitter                |
| `openapi/`                              |     5 |   ~495 | OpenAPI emitter                   |
| `docserver/`                            |     5 |   ~544 | In-process HTTP doc server        |
| `internal/`                             |     5 |   ~545 | Test helpers                      |

The module is larger than 7 of the 21 other library modules combined. Four independent exporters share no code beyond the registry types.

## Decision

Split into 5 modules under `catalog/`:

1. **`catalog`** — `Registry`, `SchemaFromType[T]`, `Build`, `Validate`, type definitions (~3k LoC)
2. **`catalog/schema`** — JSON Schema reflection engine (~968 LoC, already in its own subdirectory)
3. **`catalog/asyncapi`** — AsyncAPI 3.0 exporter (~787 LoC)
4. **`catalog/openapi`** — OpenAPI exporter (~495 LoC)
5. **`catalog/d2`** — D2 diagram exporter (~674 LoC)

`eventcatalog/` and `docserver/` remain as sub-packages of `catalog/` since they depend heavily on the registry.

Each exporter gets its own `go.mod`, allowing consumers to import only the formats they need. The shared dependency is `catalog` (the registry).

## Consequences

- **Breaking change** — import paths change from `catalog/v2/asyncapi` to separate module paths
- Reduces dependency footprint: consumers who only need AsyncAPI don't pull in OpenAPI/D2 code
- Each exporter can be versioned independently
- Test isolation: a bug in D2 exporter doesn't block AsyncAPI release
- Module count increases from 30 to 34
