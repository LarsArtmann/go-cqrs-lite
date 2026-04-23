# In-Depth Project Review: go-cqrs-lite

**Date:** 2026-04-23

## Overview

A well-structured, zero-dependency CQRS library for Go (~14,600 LOC across 70+ files). All tests pass with race detection, no `go vet` issues, and benchmark numbers look healthy. The project is clearly production-minded with strong typing, proper concurrency control, and good documentation generation tooling.

## Architecture Assessment: Excellent

The layered design is clean and idiomatic:

| Layer | Packages | Verdict |
|---|---|---|
| **Core** | `command`, `query`, `event`, `aggregate` | Clean separation, interface-first |
| **Infrastructure** | `internal/dispatcher`, `middleware`, `pkg/id` | Properly internalized, generic-based |
| **Catalog** | `catalog/*` | Impressive auto-doc generation (AsyncAPI + EventCatalog) |
| **Extensions** | `xtypes` | Type-safe wrappers for compile-time safety |

The generic `internal/dispatcher.Dispatcher[H, M]` eliminates boilerplate across command/query dispatchers — a smart refactoring. The `LifecycleMixin` / `CatalogDispatcher` composition pattern is elegant.

## Strengths

### 1. Strong Type Safety via Branded IDs (`pkg/id/`)

`id.Of[T]` prevents mixing AggregateID/UserID/etc. at compile time. Full JSON/DB/encoding support. The `GoString()` and `Format()` methods are a nice touch for debugging.

### 2. Clean Generic Dispatcher

`internal/dispatcher/dispatcher.go` — well-designed generic with `MiddlewareChain[H, M]`, thread-safe handler map, and lifecycle management. No code duplication between command and query dispatchers.

### 3. Comprehensive Catalog System

Three-layer design (core types → registry → exporters) is well-factored. Custom YAML marshaler (`catalog/yaml/`) to avoid pulling in a YAML dependency is pragmatic. AsyncAPI 3.0 + EventCatalog MDX generation is impressive.

### 4. Proper Concurrency

All stores, buses, registries use `sync.RWMutex` correctly. Race detector passes clean.

### 5. Error Handling

Sentinel errors + contextual wrapping with `cockroachdb/errors` is the right pattern.

### 6. Performance

Dispatch benchmarks at ~60ns/0 allocs, event creation at ~240ns, memory store load at ~47ns. All excellent.

## Test Coverage

| Package | Coverage |
|---|---|
| `event` | 95.4% |
| `xtypes` | 95.7% |
| `catalog/asyncapi` | 96.3% |
| `query` | 91.5% |
| `catalog` | 91.2% |
| `catalog/eventcatalog` | 89.5% |
| `pkg/id` | 85.4% |
| `catalog/yaml` | 84.4% |
| `command` | 84.4% |
| `middleware` | 84.6% |
| `internal/dispatcher` | 77.4% |
| `aggregate` | 77.3% |
| `catalog/adapters` | 66.0% |
| `pkg/errors` | 0.0% (no tests) |

## Issues Found

### Critical

**1. `query.Dispatcher.Dispatch` ignores `ctx` parameter** (`query/dispatcher.go:58`)

```go
_ = ctx // Context available for tracing/logging but not required for basic dispatch
```

Context is silently discarded. Middleware and handlers receive no context, making timeouts/cancellation/tracing impossible. The handler signature `func(Query) (any, error)` doesn't even accept context — this is inconsistent with the command dispatcher where handlers are `func(context.Context, Command) error`.

**2. `Middleware` type mismatch between command and query** — command handlers take `ctx`, query handlers don't. This breaks the pattern of shared middleware across CQRS sides.

### Moderate

**3. `pkg/errors` package is dead code** — `BaseError` is defined but never used anywhere in the codebase. It's also redundant alongside `cockroachdb/errors`. Should be removed or purposefully integrated.

**4. `pkg/errors` has no tests** — 0% coverage.

**5. `xtypes.TypedCommand.Command()` creates a new `command.Core` on every call** (`xtypes/command.go:31`) — allocates a new struct each time just to satisfy the interface. Should embed or cache.

**6. `MemoryBus.Publish` holds `RLock` for the entire publish** (`event/memory_bus.go:42-47`) — including all handler execution. This means subscribers block publishers and vice-versa. For a test utility this is acceptable, but worth documenting.

**7. No `context.Context` in `MemoryStore` methods** — all store methods accept but ignore context (`_ context.Context`). Makes it impossible to cancel long-running store operations even in the interface.

**8. `event.Core` is mutable via `Option` after creation** — `WithMetadata` options mutate the event after it's constructed in `NewEvent`. The doc comment says "immutable event data" but the pattern allows post-creation mutation during the options loop.

### Minor / Style

**9. Cyclomatic complexity violations** — 7 functions exceed the project's own max-10 rule:

- `catalog/yaml/yaml.go:marshalValue` (14) — the core YAML marshal switch
- `aggregate/integration_test.go:TestCQRSRoundtrip` (16)
- `catalog/asyncapi/exporter_test.go:TestExporter_Export_BasicCommand` (15)
- Several test functions

**10. `err113` linter warnings** — 9 instances of dynamic error creation via `fmt.Errorf`/`errors.New` that should be sentinel errors per the project's own convention. The linter is correctly flagging these.

**11. `catalog/adapters` coverage at 66%** — lowest of tested packages. The `from_dispatcher.go` and `from_query_dispatcher.go` adapters may be undertested.

**12. `example/user` module has a separate `go.mod`** — this is fine for examples, but it's not in the CI workflow, so it could silently break.

## Recommendations

| Priority | Item | Rationale |
|---|---|---|
| **High** | Fix query handler signature to include `context.Context` | Blocks tracing, cancellation, timeout propagation |
| **High** | Remove `pkg/errors` or integrate it | Dead code confuses users |
| **Medium** | Create sentinel errors for `err113` violations | Consistency with project conventions |
| **Medium** | Reduce `marshalValue` complexity | Split into type-specific marshalers |
| **Medium** | Add tests for `catalog/adapters` to >80% | Critical for doc generation pipeline |
| **Low** | Make `MemoryBus.Publish` drop the lock during handler execution | Better test isolation |
| **Low** | Add CI step for `example/` modules | Prevent silent breakage |
| **Low** | Consider making event `Core` truly immutable | Build it completely in `NewEvent`, return an interface with no mutators |

## Verdict

**Solid, well-engineered library.** The architecture is clean, the generic dispatcher foundation is excellent, the branded ID system is a standout feature, and the catalog system (especially zero-dep YAML + AsyncAPI 3.0) is impressive. The main gap is the query side's missing `context.Context` propagation, which is an architectural inconsistency that should be addressed before widespread use. Everything else is polish.
