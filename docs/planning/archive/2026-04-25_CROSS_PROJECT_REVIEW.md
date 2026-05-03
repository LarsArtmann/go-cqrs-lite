# Cross-Project Architecture Review: go-cqrs-lite, go-localfirst, go-localsync

**Date:** 2026-04-25
**Status:** Recommendation
**Scope:** Where things should live across the four related projects

---

## Projects at a Glance

| Project                          | Role                                    | Depends On                                                  |
| -------------------------------- | --------------------------------------- | ----------------------------------------------------------- |
| **go-cqrs-lite**                 | Foundational CQRS SDK (zero-dep)        | cockroachdb/errors, google/uuid, go-faster/yaml             |
| **go-localfirst**                | Local-first app framework               | go-cqrs-lite, gin, pebble, casbin, gorilla/websocket, zap   |
| **go-localsync**                 | Provider data sync SDK                  | go-composable-business-types, modernc.org/sqlite, go-github |
| **go-composable-business-types** | Shared business types (ID, Money, etc.) | bojanz/currency, sixafter/nanoid, golang.org/x/text         |

### Dependency Graph (Actual)

```
go-composable-business-types ←── go-localsync (for ID[B,V])
go-cqrs-lite ←── go-localfirst (for CQRS + id.Of[T])

NO dependency between go-localsync and go-localfirst.
```

**Key correction from initial analysis:** go-localsync does NOT import go-localfirst. It has its own `pkg/sync` package with its own LWW implementation, completely independent of go-localfirst's `pkg/sync`.

---

## Finding 1: Two Competing ID Systems

This is the most significant architectural tension across the projects.

### Comparison

| Property             | go-cqrs-lite `id.Of[T any] string`   | go-composable-business-types `id.ID[B any, V comparable] struct{ value V }` |
| -------------------- | ------------------------------------ | --------------------------------------------------------------------------- |
| Underlying type      | Type alias to `string`               | Struct with private `value V` field                                         |
| Type parameters      | 1 (brand only)                       | 2 (brand + value type)                                                      |
| Value types          | String only                          | string, int64, int, uint64, etc.                                            |
| Auto-generation      | `id.New[T]()` → UUID v4              | `id.NewID[B, V](v)` → from explicit value                                   |
| Zero value           | Empty string `""`                    | `struct{ value: zero }`                                                     |
| JSON null            | Supported (`""` → `null`)            | Supported (zero → `null`)                                                   |
| SQL Scan/Value       | Supported                            | Supported                                                                   |
| Binary/Text encoding | Supported                            | Not implemented                                                             |
| `fmt.Formatter`      | Full (`%v`, `%#v`, `%s`, `%q`)       | Partial (`%s`, `%d`, `%v`, `%#v`, `%q`)                                     |
| `Compare`            | Returns int (always works)           | Returns `(int, error)` (ordered types only)                                 |
| Dependencies         | google/uuid, go-json-experiment/json | None (stdlib only)                                                          |
| Used by              | go-cqrs-lite, go-localfirst          | go-localsync, go-composable-business-types                                  |

### Where Each Is Used

**go-cqrs-lite `id.Of[T]`** in go-localfirst:

- `internal/domain/ids.go` — `TodoID`, `PeerID`, `NodeID`, `OperationID`, `TargetID` (all via `id.Of[T]`)
- `internal/cqrs/` — all command/aggregate/event types use `id.AggregateID`, `id.EventID`
- `internal/cqrs/store/pebble_adapter.go` — `CQRSAdapter` uses `id.AggregateID`, `id.ParseAggregateID`

**go-composable-business-types `id.ID[B, V]`** in go-localsync:

- `pkg/types/ids.go` — `EventID = ID[EventBrand, int64]`, `ItemID = ID[ItemBrand, string]`, `ProviderID`, `ActorID`, `RepoID`, `EventTypeID`
- `pkg/provider/provider.go` — `Item` struct uses `types.ItemID`
- `pkg/storage/interface.go` — `Storage` interface uses `types.ItemID`

### The Problem

These two ID systems are **not interoperable at the type level**. If a future feature needs go-localsync and go-localfirst to exchange typed IDs (e.g., syncing events that originate from CQRS aggregates), you must convert via `.String()` and re-parse — losing compile-time safety at the boundary.

### Recommendation: Converge on `id.Of[T] string`

**go-cqrs-lite's `id.Of[T] string`** should become the canonical ID system across all projects.

**Rationale:**

1. **It's already the foundational SDK** — All CQRS types depend on it. Moving away would require rewriting go-cqrs-lite's entire type system.
2. **String-based IDs are the correct default for distributed systems** — Event sourcing, CRDT sync, and HTTP APIs all fundamentally exchange string identifiers. Numeric IDs (like `EventID = ID[EventBrand, int64]`) are database implementation details that leak into domain types.
3. **Zero-value semantics are clearer** — Empty string `""` is a natural sentinel for "unset." Struct-based IDs require `.IsZero()` checks and are harder to use as map keys, struct comparables, and JSON fields.
4. **`go-json-experiment/json` support** — go-cqrs-lite's IDs work with the JSON v2 experiment, which is the future of Go JSON handling.
5. **Full encoding suite** — Binary, Text, SQL, JSON all covered. go-composable-business-types' ID lacks binary/text encoding.
6. **The multi-parameter `ID[B, V]` adds complexity with no practical benefit** — In practice, go-localsync only uses `string` as the value type for all its IDs except `EventID` (which uses `int64`). The `int64` `EventID` is an auto-increment database primary key that should NOT be a branded domain type — it's a storage concern.

**Migration path for go-localsync:**

| Current                                           | Migration                                                                               |
| ------------------------------------------------- | --------------------------------------------------------------------------------------- |
| `EventID = id.ID[EventBrand, int64]`              | Keep as `int64` in storage layer only; use `id.Of[EventMarker] string` for domain types |
| `ItemID = id.ID[ItemBrand, string]`               | `id.Of[ItemMarker] string`                                                              |
| `ProviderID = id.ID[ProviderBrand, string]`       | `id.Of[ProviderMarker] string`                                                          |
| `ActorID = id.ID[ActorBrand, string]`             | `id.Of[ActorMarker] string`                                                             |
| `RepoID = id.ID[RepoBrand, string]`               | `id.Of[RepoMarker] string`                                                              |
| `EventTypeID = id.ID[EventTypeBrand, string]`     | `id.Of[EventTypeMarker] string`                                                         |
| `GithubEventID = id.ID[GithubEventBrand, string]` | `id.Of[GithubEventMarker] string`                                                       |

This adds `google/uuid` as a dependency to go-localsync (transitively through go-cqrs-lite/pkg/id). This is acceptable — `google/uuid` is ubiquitous and tiny.

**go-composable-business-types** keeps its `id.ID[B, V]` for the broader business types use case (Money, ActorChain, etc.), but the ID package should be considered **deprecated for new cross-project use**. Projects that need branded IDs should use `go-cqrs-lite/pkg/id`.

---

## Finding 2: Two Independent `pkg/sync` Packages

### Current State

Both go-localfirst and go-localsync have a `pkg/sync` package, but they are **completely independent**:

|                  | go-localfirst `pkg/sync`                                   | go-localsync `pkg/sync`                             |
| ---------------- | ---------------------------------------------------------- | --------------------------------------------------- |
| **Purpose**      | CRDT primitives (VectorClock, Operation, ConflictResolver) | Sync orchestration (Syncer, ConflictAwareSyncer)    |
| **Dependencies** | Zero (stdlib only)                                         | go-localsync's own pkg/provider, pkg/storage        |
| **Used by**      | go-localfirst internally                                   | go-localsync internally                             |
| **Generic?**     | Yes (`Operation[T any]`, `LWWResolver[T]`)                 | No (operates on `provider.Item`)                    |
| **LWW?**         | `LWWResolver[T]` (generic, timestamp extractor)            | Inline LWW in `ConflictAwareSyncer` (Item-specific) |

These are **different abstractions at different levels**:

- go-localfirst's `pkg/sync` = **CRDT primitives** (building blocks)
- go-localsync's `pkg/sync` = **Sync orchestration** (fetch → store pipeline)

### Recommendation: Extract go-localfirst's `pkg/sync` into `go-cqrs-lite/sync` module

**Rationale:**

1. go-localfirst's `pkg/sync` is explicitly documented as a "shared CRDT primitives SDK" for both projects — but it's trapped inside the application framework.
2. It has **zero external dependencies** — it fits go-cqrs-lite's zero-dep philosophy perfectly.
3. CRDT primitives (VectorClock, ConflictResolver) are **foundational building blocks**, not application-layer code. They belong in the foundational SDK.
4. Extracting it breaks the circular conceptual dependency: go-localsync could use CRDT primitives without importing the entire go-localfirst framework.

**After multi-module monorepo migration**, this becomes a natural `sync/` module:

```
go-cqrs-lite/
├── sync/                    # github.com/larsartmann/go-cqrs-lite/sync
│   └── go.mod               # deps: zero (stdlib only)
│   ├── vectorclock.go       # VectorClock
│   ├── operation.go         # Operation[T any]
│   ├── conflict.go          # ConflictResolver[T], LWWResolver[T], SyncMessage
│   └── doc.go
```

**Short-term** (before multi-module migration): Move `pkg/sync` from go-localfirst into go-cqrs-lite's `pkg/sync/` directory. Both projects already use `replace` directives for local development, so this is seamless.

**Impact on go-localsync:**

go-localsync's `pkg/sync` is **sync orchestration**, not CRDT primitives. It doesn't use VectorClock or LWWResolver from go-localfirst. However, the `ConflictAwareSyncer` implements a simple inline LWW comparison that could be replaced by `LWWResolver[*provider.Item]` from go-cqrs-lite/sync — but this is optional and low priority since the current code is specific to `provider.Item` and works fine.

---

## Finding 3: go-localsync's `pkg/provider/` Should Be Extracted

### Current State

go-localsync's `pkg/provider/` defines:

- `Provider` interface (5 methods: Name, FetchAll, FetchByDateRange, FetchByID, HealthCheck)
- `Item` struct (typed IDs, timestamps, metadata)
- `RateLimitConfig`, `RetryConfig`
- GitHub-specific implementation in `internal/provider/github/`

### Recommendation: Extract as `go-provider` (standalone module)

**Rationale** (per go-localsync's own `PARTS.md`):

1. `Provider` interface has **zero dependency** on pkg/sync, pkg/storage, or go-composable-business-types
2. It's a generic abstraction for "fetch data from an external API" — useful far beyond go-localsync
3. Extraction would let other projects (scrapers, API clients, etc.) use the same Provider interface
4. `Item` struct is well-defined with typed IDs — it could use go-cqrs-lite's `id.Of[T]` after the ID convergence

**Where it should live:** Either as a standalone repo (`go-provider`) or as a module in go-cqrs-lite's monorepo. Since it's not CQRS-specific, a standalone repo is cleaner. But if the multi-module monorepo plan is committed to, a `provider/` module under go-cqrs-lite would keep everything in one place.

**Recommendation: Standalone repo** — `github.com/larsartmann/go-provider`. The Provider abstraction is about external data fetching, not event sourcing. It doesn't belong in a CQRS SDK.

---

## Finding 4: go-cqrs-lite Multi-Module Monorepo Plan — Update Needed

### Current Plan

The existing plan (`docs/planning/2026-04-23_MULTI_MODULE_MONOREPO_PLAN.md`) proposes 10 modules: core, memory, storage, watermill, projection, snapshot, catalog, middleware, xtypes, testutil.

### Recommended Addition: `sync/` Module

Add an 11th module for CRDT/sync primitives:

```
sync/                        # github.com/larsartmann/go-cqrs-lite/sync
└── go.mod                   # deps: zero (stdlib only)
├── vectorclock.go           # VectorClock
├── operation.go             # Operation[T any]
├── conflict.go              # ConflictResolver[T], LWWResolver[T]
└── doc.go
```

**Why it fits in go-cqrs-lite:** CRDT sync is the natural companion to event sourcing in distributed systems. An event-sourced aggregate that needs to sync across nodes needs VectorClock for causal ordering and ConflictResolver for merge. These are as foundational as the event store itself.

**Dependency graph update:**

```
                              core (ulid + errors)
                             / | \  \  \  \  \  \  \
                            /  |  \  \  \  \  \  \  \
                       memory  catalog  middleware  xtypes  testutil  sync
                                  |                                    |
                          ┌───┬───┴───┬────────┐                    (zero deps)
                          │   │       │        │
                      storage  watermill  projection  snapshot
                          │              │        │
                          └──────────────┴────────┘
```

`sync/` has zero dependencies — it doesn't even depend on `core/`. It's purely stdlib.

---

## Finding 5: go-localfirst's CQRS Integration Is Correctly Positioned

### Current State

go-localfirst's `internal/cqrs/` directory:

- `commands/mixin.go` — `CommandHandlerMixin` wrapping go-cqrs-lite's command dispatcher
- `commands/json.go` — JSON marshaling helpers for commands
- `aggregate/todo.go` — Full CQRS aggregate using go-cqrs-lite's `aggregate.Core`
- `store/pebble_adapter.go` — `CQRSAdapter` implementing `event.Store` via Pebble
- `handler/sse_cqrs_bridge.go` — Bridges CQRS events to SSE broadcasting

### Recommendation: Keep as-is

This is **application-layer code** that correctly wraps the foundational SDK. The Pebble adapter is specific to go-localfirst's storage backend. The SSE bridge is specific to go-localfirst's HTTP layer. These should NOT move to go-cqrs-lite.

**One note:** The Pebble adapter could inspire a future `go-cqrs-lite/storage/` module with Pebble support, but that's a separate concern. The current adapter is go-localfirst-specific and should stay.

---

## Finding 6: go-composable-business-types — Keep, But Deprecate ID Package

### Current State

Provides `ID[B, V]`, `ActorChain`, `DataPoint`, `BoundedString`, `Money`, enums. Only the ID package is used cross-project (by go-localsync).

### Recommendation

1. **Deprecate the ID package for cross-project use** — Add a deprecation notice pointing to `go-cqrs-lite/pkg/id`
2. **Keep the other types** — `Money`, `ActorChain`, `BoundedString`, etc. are genuinely business-domain types that don't belong in a CQRS SDK
3. **go-localsync should migrate** to `go-cqrs-lite/pkg/id` per Finding 1

---

## Action Plan

### Priority 1: ID System Convergence (High Impact, Medium Effort)

| Step | Project                      | Action                                                                                                            |
| ---- | ---------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| 1.1  | go-localsync                 | Replace `go-composable-business-types/id` imports with `go-cqrs-lite/pkg/id` in `pkg/types/ids.go`                |
| 1.2  | go-localsync                 | Change `EventID = ID[EventBrand, int64]` to be storage-layer only; add `id.Of[EventMarker] string` for domain use |
| 1.3  | go-localsync                 | Update all branded IDs to `id.Of[TMarker] string`                                                                 |
| 1.4  | go-localsync                 | Update `pkg/provider/`, `pkg/storage/` to use new ID types                                                        |
| 1.5  | go-localsync                 | Remove `go-composable-business-types` from `go.mod`                                                               |
| 1.6  | go-composable-business-types | Add deprecation notice to `id/` package doc                                                                       |

### Priority 2: Extract CRDT Primitives (High Impact, Low Effort)

| Step | Project       | Action                                                                                                            |
| ---- | ------------- | ----------------------------------------------------------------------------------------------------------------- |
| 2.1  | go-cqrs-lite  | Create `pkg/sync/` directory, copy VectorClock, Operation, ConflictResolver, LWWResolver from go-localfirst       |
| 2.2  | go-cqrs-lite  | Add tests for `pkg/sync/` (go-localfirst's `pkg/sync` tests should transfer)                                      |
| 2.3  | go-localfirst | Replace `pkg/sync/` with type aliases to `go-cqrs-lite/pkg/sync` (same pattern as previous `internal/sync` dedup) |
| 2.4  | go-localfirst | Update `pkg/sync/doc.go` to reference go-cqrs-lite as canonical source                                            |
| 2.5  | go-cqrs-lite  | Update multi-module monorepo plan to include `sync/` module                                                       |

### Priority 3: Extract Provider (Medium Impact, Low Effort)

| Step | Project      | Action                                                                                          |
| ---- | ------------ | ----------------------------------------------------------------------------------------------- |
| 3.1  | new repo     | Create `go-provider` with `Provider` interface, `Item` struct, `RateLimitConfig`, `RetryConfig` |
| 3.2  | go-localsync | Replace internal `pkg/provider/` with import from `go-provider`                                 |
| 3.3  | go-localsync | Update `pkg/storage/` to accept `go-provider`'s `Item` type                                     |
| 3.4  | go-localsync | Update `PARTS.md` to mark extraction as complete                                                |

### Priority 4: Multi-Module Monorepo Migration (High Impact, High Effort)

| Step | Project       | Action                                                                                 |
| ---- | ------------- | -------------------------------------------------------------------------------------- |
| 4.1  | go-cqrs-lite  | Execute Phase 0 from the existing plan (fix query handler ctx, delete dead code, etc.) |
| 4.2  | go-cqrs-lite  | Execute Phases 1-2 (core + memory modules)                                             |
| 4.3  | go-cqrs-lite  | Add `sync/` module during Phase 2 (it's zero-dep, easy to extract)                     |
| 4.4  | go-cqrs-lite  | Continue with remaining phases per existing plan                                       |
| 4.5  | go-localfirst | Update imports to `go-cqrs-lite/core/...` after module migration                       |
| 4.6  | go-localsync  | Update imports to `go-cqrs-lite/core/pkg/id` and `go-cqrs-lite/sync`                   |

### Priority 5: Optional — go-localsync CRDT Integration (Low Priority)

| Step | Project      | Action                                                                                                           |
| ---- | ------------ | ---------------------------------------------------------------------------------------------------------------- |
| 5.1  | go-localsync | Consider replacing inline LWW in `ConflictAwareSyncer` with `sync.LWWResolver[*provider.Item]` from go-cqrs-lite |
| 5.2  | go-localsync | Consider adding `VectorClock` to `Operation` for causal ordering of sync operations                              |

These are optional improvements, not blockers. The current code works correctly.

---

## What Should NOT Move

| Code                           | Current Location                        | Why It Stays                                         |
| ------------------------------ | --------------------------------------- | ---------------------------------------------------- |
| Pebble `CQRSAdapter`           | go-localfirst `internal/cqrs/store/`    | Application-specific storage adapter                 |
| SSE event bridge               | go-localfirst `internal/handler/`       | Application-specific HTTP layer                      |
| Command handler mixin          | go-localfirst `internal/cqrs/commands/` | Application-specific handler composition             |
| Sync orchestration (`Syncer`)  | go-localsync `pkg/sync/`                | Provider-data-specific pipeline, not CRDT primitives |
| Storage interface (16 methods) | go-localsync `pkg/storage/`             | SQLite-specific query patterns                       |
| Business types (Money, etc.)   | go-composable-business-types            | Not CQRS-related                                     |

---

## Summary Decision Matrix

| Concern                                    | Canonical Home                                | Consumers                                     |
| ------------------------------------------ | --------------------------------------------- | --------------------------------------------- |
| Branded IDs (`id.Of[T]`)                   | **go-cqrs-lite** `pkg/id/`                    | go-localfirst, go-localsync (after migration) |
| CRDT primitives (VectorClock, LWWResolver) | **go-cqrs-lite** `pkg/sync/` → `sync/` module | go-localfirst, go-localsync (optional)        |
| Provider interface                         | **go-provider** (new repo)                    | go-localsync, future projects                 |
| CQRS core (events, commands, aggregates)   | **go-cqrs-lite** `core/` module               | go-localfirst, any CQRS app                   |
| Memory implementations                     | **go-cqrs-lite** `memory/` module             | Tests everywhere                              |
| SQL event store                            | **go-cqrs-lite** `storage/` module            | Production apps                               |
| Pub/Sub (Watermill)                        | **go-cqrs-lite** `watermill/` module          | Production apps                               |
| Projections                                | **go-cqrs-lite** `projection/` module         | Read model apps                               |
| Catalog/AsyncAPI                           | **go-cqrs-lite** `catalog/` module            | Documentation generation                      |
| Pebble event store adapter                 | **go-localfirst** `internal/cqrs/store/`      | go-localfirst only                            |
| Sync orchestration (fetch→store)           | **go-localsync** `pkg/sync/`                  | go-localsync only                             |
| SQLite storage                             | **go-localsync** `pkg/storage/`               | go-localsync only                             |
| Business types (Money, ActorChain)         | **go-composable-business-types**              | Any business domain app                       |
