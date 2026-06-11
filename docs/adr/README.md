# Architecture Decision Records

## ADR-0001: Decider Pattern over Aggregate Root

**Date:** 2026-04-29
**Status:** Accepted
**Supersedes:** Traditional DDD Aggregate Root pattern

### Context

The traditional DDD Aggregate Root pattern requires a 9-method interface, mutable state, and tight coupling between infrastructure and domain logic. This makes testing difficult and creates unnecessary abstraction layers.

### Decision

Adopt the **Decider pattern** — a pure-function approach where:

- `Decider[State]` holds `Initial` state and `Fold func(State, Event) (State, error)`
- `DecideFunc` takes a command and state, returns events (pure function)
- `Repository[State].Execute` does load → fold → decide → save → publish
- No mutable state, no 9-method interface, zero-infrastructure testing

### Consequences

- **+** Pure functions are trivially testable without mocks
- **+** Single generic `Repository[State]` replaces per-aggregate repositories
- **+** `ExecuteWithResult` returns typed results alongside events
- **-** Consumers must think in terms of state + fold, not OOP-style objects
- The deprecated `core/aggregate` package was removed in Session 99

---

## ADR-0002: Error Taxonomy via go-error-family

**Date:** 2026-05-03
**Status:** Accepted

### Context

Without classified errors, consumers cannot distinguish transient failures from business rule violations from data corruption. This leads to blanket retry strategies that mask real problems.

### Decision

Adopt a **5-family error taxonomy** via `go-error-family`:

| Family             | Constructor         | Meaning                      | Retry? |
| ------------------ | ------------------- | ---------------------------- | ------ |
| **Rejection**      | `NewRejection`      | Business rule violation      | No     |
| **Conflict**       | `NewConflict`       | Concurrency/version conflict | No     |
| **Transient**      | `NewTransient`      | Temporary failure            | Yes    |
| **Infrastructure** | `NewInfrastructure` | System-level failure         | Maybe  |
| **Corruption**     | `NewCorruption`     | Data integrity violation     | No     |

Each family has `Is*()` predicates and `Wrap*()` wrapping functions. Middleware uses `event.IsRetryable(err)` to decide retry behavior.

### Consequences

- **+** Middleware can make informed retry/recovery decisions
- **+** Error classification is explicit, not heuristic-based
- **+** 38 sentinel errors classified across all modules
- **-** Requires `init()` registration for cross-package sentinels (TODO: explicit setup)

---

## ADR-0003: Multi-Module Monorepo with go.work

**Date:** 2026-04-24
**Status:** Accepted

### Context

A single `go.mod` creates tight coupling between all packages. Changes to the catalog module require re-testing storage, even though they're unrelated. External consumers can't import individual packages.

### Decision

Split into **16 workspace modules** with independent `go.mod` files, tied together by `go.work`:

```
core/          — Zero external deps (ulid, branded-id, error-family)
memory/        — In-memory test implementations
catalog/       — Documentation generation (AsyncAPI, D2, EventCatalog, OpenAPI)
middleware/    — Cross-cutting CQRS middleware
testhelpers/   — Shared test utilities
projection/    — Projection runner with replay
storage/       — SQL + Pebble backends
saga/          — Long-running process orchestration
watermill/     — Watermill protocol adapter
```

### Consequences

- **+** Each module has isolated dependencies
- **+** External consumers import only what they need
- **+** CI can test modules in parallel and in isolation (`GOWORK=off`)
- **-** `replace` directives required until v1.0.0 tags are pushed to remote
- **-** Version management across 16 modules requires discipline
- **-** `golangci-lint` doesn't work well with `go.work` (pre-existing tooling issue)

---

## Index

| ADR | Title | Date | Status |
|-----|-------|------|--------|
| [0001](0001-decider-over-aggregate.md) | Decider Pattern over Aggregate Root | 2026-04-29 | Accepted |
| [0002](0002-error-taxonomy.md) | Error Taxonomy via go-error-family | 2026-05-03 | Accepted |
| [0003](0003-multi-module-monorepo.md) | Multi-Module Monorepo with go.work | 2026-04-24 | Accepted |
| [0004](0004-saga-process-manager.md) | Saga/Process Manager Pattern | — | Accepted |
| 0005 | *(gap — no ADR-0005 was issued)* | — | — |
| [0006](0006-sink-source-split.md) | Sink/Source ISP Split | — | Accepted |
| [0007](0007-gopls-workspace-workaround.md) | gopls Workspace Workaround | — | Accepted |
| [0008](0008-typed-handler-signature.md) | Typed Handler Signature | — | Accepted |
| [0009](0009-pebble-scope-event-store-only.md) | Pebble Scope: Event Store Only | — | Accepted |
| [0010](0010-remove-io-closer-from-interfaces.md) | Remove io.Closer from Core Interfaces | — | Accepted |
| [0011](0011-unify-err-dispatcher-closed.md) | Unify ErrDispatcherClosed | — | Accepted |
| [0012](0012-split-catalog-modules.md) | Split Catalog into Sub-Modules | — | Accepted |
| [0013](0013-zero-copy-payload-for-decode.md) | Zero-Copy Payload for Decode | — | Accepted |
| [0014](0014-test-only-dependencies-in-go-mod.md) | Test-Only Dependencies in go.mod | — | Accepted |
| [0015](0015-cbor-codec.md) | CBOR Codec | — | Accepted |
