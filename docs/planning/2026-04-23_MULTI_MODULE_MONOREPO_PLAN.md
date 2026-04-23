# Multi-Module Monorepo Migration Plan

**Date:** 2026-04-23
**Status:** Draft

## Goal

Restructure go-cqrs-lite as a multi-module monorepo so users only pay for what they import.

## Core Principle

> Each module has its own `go.mod` with **only the dependencies it needs**.
> A root `go.work` ties them together for development.

## Module Layout

```
go-cqrs-lite/
├── go.work
│
├── core/                           # github.com/larsartmann/go-cqrs-lite/core
│   └── go.mod                      # deps: ONLY cockroachdb/errors, google/uuid
│   ├── command/
│   ├── query/
│   ├── event/                      # interfaces only (Store, Bus — no MemoryStore)
│   ├── aggregate/
│   └── pkg/
│       ├── id/                     # branded IDs
│       └── errors/                 # delete — unused, dead code
│
├── memory/                         # github.com/larsartmann/go-cqrs-lite/memory
│   └── go.mod                      # deps: core
│   ├── store/                      # MemoryStore, MemorySnapshotStore
│   └── bus/                        # MemoryBus
│
├── catalog/                        # github.com/larsartmann/go-cqrs-lite/catalog
│   └── go.mod                      # deps: core, go-faster/yaml (delete custom marshaler)
│   ├── types.go
│   ├── registry.go
│   ├── schema.go
│   ├── asyncapi/                   # uses real YAML lib
│   ├── eventcatalog/
│   └── adapters/
│
├── middleware/                      # github.com/larsartmann/go-cqrs-lite/middleware
│   └── go.mod                      # deps: core
│   ├── logging.go
│   ├── metrics.go
│   ├── recovery.go
│   ├── retry.go
│   └── validation.go
│
├── xtypes/                         # github.com/larsartmann/go-cqrs-lite/xtypes
│   └── go.mod                      # deps: core
│
├── integrations/                   # NEW — one go.mod per backend
│   ├── postgres/                   # github.com/larsartmann/go-cqrs-lite/integrations/postgres
│   │   └── go.mod                  # deps: core, pgx
│   │   └── eventstore.go
│   ├── nats/                       # github.com/larsartmann/go-cqrs-lite/integrations/nats
│   │   └── go.mod                  # deps: core, nats.go
│   │   └── eventbus.go
│   └── redis/                      # github.com/larsartmann/go-cqrs-lite/integrations/redis
│       └── go.mod                  # deps: core, go-redis
│       └── eventstore.go
│
└── examples/                       # separate modules, CI-tested
    ├── user/
    │   └── go.mod
    └── catalog/
        └── go.mod
```

## go.work

```go
go 1.26

use (
    ./core
    ./memory
    ./catalog
    ./middleware
    ./xtypes
    ./integrations/postgres
    ./integrations/nats
    ./integrations/redis
    ./examples/user
    ./examples/catalog
)
```

## Dependency Graph

```
                    core (zero deps*)
                   /  |  \  \
                  /   |   \  \
             memory  catalog  middleware  xtypes
                                  \
                         integrations/*
```

`*` core keeps cockroachdb/errors + google/uuid — acceptable for a Go SDK.
If you want truly zero, vendor uuid generation and use fmt.Errorf.

## What Users Import

| Use Case | Import | Gets These Deps |
|---|---|---|
| CQRS types only | `core/...` | uuid, errors |
| + in-memory testing | `memory/...` | + core |
| + API docs | `catalog/...` | + core, go-faster/yaml |
| + PostgreSQL store | `integrations/postgres` | + core, pgx |
| + NATS bus | `integrations/nats` | + core, nats.go |
| Everything | all modules | all deps |

Nobody who just wants CQRS types pulls in pgx, nats, or a YAML library.

## What Gets Deleted

| What | Why |
|---|---|
| `catalog/yaml/` (~180 lines) | Replaced by `go-faster/yaml` in catalog's go.mod |
| `pkg/errors/` (~30 lines) | Dead code — never used anywhere |
| `event/memory_store.go` | Moves to `memory/store/` |
| `event/memory_bus.go` | Moves to `memory/bus/` |
| `event/memory_snapshot_store.go` | Moves to `memory/store/` |

## What Gets Fixed (Existing Issues)

| Issue | Fix |
|---|---|
| Query handler missing `context.Context` | Fix in `core/query/` during migration |
| `err113` sentinel errors | Fix in all modules |
| `marshalValue` complexity (14) | Delete the whole file, use go-faster/yaml |
| `catalog/adapters` coverage 66% | Add tests in `catalog/` module |
| Examples not CI-tested | Add to workflow via `go.work` |

## Migration Phases

### Phase 0: Preparation (no breaking changes)

1. Fix query handler signature to include `context.Context`
2. Delete `pkg/errors/` (dead code)
3. Replace custom YAML marshaler with `go-faster/yaml`
4. Fix all `err113` linter warnings
5. Ensure all tests pass clean

### Phase 1: Create go.work + move into subdirectories

1. Create `go.work` at root pointing at current single module
2. Verify everything still works (this is the safe checkpoint)
3. Create `core/` directory, move `command/`, `query/`, `event/`, `aggregate/`, `pkg/` into it
4. Update `core/go.mod` — module path becomes `github.com/larsartmann/go-cqrs-lite/core`
5. Update all internal import paths
6. Run tests, fix until green

### Phase 2: Extract memory implementations

1. Create `memory/` with its own `go.mod`
2. Move MemoryStore, MemoryBus, MemorySnapshotStore from `core/event/` to `memory/`
3. `memory/go.mod` depends on `core`
4. Update `go.work`
5. Run tests, fix until green

### Phase 3: Extract catalog

1. Create `catalog/` with its own `go.mod`
2. Move all catalog code
3. Add `go-faster/yaml` dependency
4. Delete `catalog/yaml/` custom marshaler
5. Update `go.work`
6. Run tests, fix until green

### Phase 4: Extract middleware + xtypes

1. Same pattern — own go.mod, depends on core
2. Run tests, fix until green

### Phase 5: Integration modules (new)

1. Create `integrations/postgres/` — implement `event.Store` backed by pgx
2. Create `integrations/nats/` — implement `event.Bus` backed by nats.go
3. Each has its own go.mod with only its backend dependency
4. Add to go.work

### Phase 6: Tag releases

1. Tag `core/v1.0.0`
2. Tag `memory/v1.0.0`
3. Tag `catalog/v1.0.0`
4. Tag other modules as ready

## Module Path Convention

Go supports two styles for multi-module repos:

```
# Style A: prefixed (traditional)
github.com/larsartmann/go-cqrs-lite/core
github.com/larsartmann/go-cqrs-lite/memory

# Style B: independent (simpler imports)
github.com/larsartmann/go-cqrs-lite-core
github.com/larsartmann/go-cqrs-lite-memory
```

**Recommendation: Style A.** Same repo, clear ownership, standard pattern used by OpenTelemetry and gRPC.

Import example:
```go
import (
    "github.com/larsartmann/go-cqrs-lite/core/command"
    "github.com/larsartmann/go-cqrs-lite/core/event"
    "github.com/larsartmann/go-cqrs-lite/memory/store"
    "github.com/larsartmann/go-cqrs-lite/integrations/postgres"
)
```

## CI Changes

```yaml
# .github/workflows/test.yml
strategy:
  matrix:
    module:
      - core
      - memory
      - catalog
      - middleware
      - xtypes
      - integrations/postgres
      - integrations/nats

steps:
  - name: Test ${{ matrix.module }}
    run: cd ${{ matrix.module }} && go test ./... -race -count=1
```

Each module tests independently. CI catches cross-module breakage via go.work.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Import path churn for existing users | Provide a compatibility shim in root module that re-exports core types |
| Cross-module refactoring pain | go.work makes IDE navigation seamless — same experience as today |
| Version skew between modules | Use replace directives in go.work during dev; tag together initially |
| Lost git history from moves | Use `git mv` — GitHub tracks history across renames |
| Examples break | They're in go.work, tested in CI |

## Open Questions

1. **Should core be truly zero-dep?** Currently depends on cockroachdb/errors + google/uuid. Could vendor UUID generation and use stdlib errors. Tradeoff: lose branded error types and UUID v4 quality.
2. **Should middleware/ be split further?** e.g., `middleware/tracing` with OTel dep vs `middleware/retry` with no deps. Low priority.
3. **Integration module priorities?** PostgreSQL store is the highest-value integration. NATS bus second. Redis optional.
