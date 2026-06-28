# ADR-0040: Deriver Module Design (TypeDB Rule Model Reference)

## Status

Accepted — 2026-06-28 (implemented: `deriver/` module)

## Context

The `Deriver` module is a long-planned but unbuilt feature (TODO C11 in multiple status reports). It handles event→event and event→command derivation — the stateless saga pattern where processing one event produces derived commands or events.

Research into TypeDB's symbolic reasoning engine surfaced a design reference for the Deriver API. TypeDB uses declarative `when {...} then {...}` rules that generate new facts from existing data at query time. Key properties:
- Rules are part of the schema (declarative, not imperative)
- Results are deterministic and idempotent
- Termination follows from "each derived instance produced at most once"
- Rule chaining: one rule's output feeds another rule's input
- Rule recursion: rules can trigger themselves (e.g., transitive closure)

## Decision

**Use a functional/composable Deriver API, not a declarative rule registry.**

TypeDB's declarative rules work because TypeDB owns the engine and can optimize rule execution. go-cqrs-lite is a library — a declarative rule registry would add complexity without the execution-engine benefits.

### Proposed API

```go
// Deriver transforms events into derived commands.
type Deriver func(ctx context.Context, evt event.Event) ([]command.Command, error)

// Then chains derivers: the output of d feeds into next.
func (d Deriver) Then(next Deriver) Deriver

// Filter wraps a deriver to only process matching event types.
func (d Deriver) Filter(types ...event.Type) Deriver
```

### Why functional over declarative

| Approach | TypeDB | go-cqrs-lite |
| --- | --- | --- |
| Declarative rule registry | Works — engine optimizes execution | Overkill — no execution engine to optimize |
| Functional composition | N/A | Idiomatic Go — compiler type-checks, easy to test |
| Determinism | Enforced by engine | Enforced by contract (pure functions) |
| Termination | Engine guarantees | Developer responsibility (document, don't enforce) |
| Chaining | Rule chaining (backward) | `Deriver.Then(other)` (forward, explicit) |

### What we steal from TypeDB

1. **Determinism + idempotency** — a Deriver must produce the same output for the same event. Re-processing an event must not produce duplicate derived commands. The Deriver documentation must state this contract.

2. **Composability** — TypeDB's rule chaining maps to `Deriver.Then(next)`. The output of one Deriver feeds into the next. This is forward chaining (explicit), not backward chaining (TypeDB's engine optimization).

3. **Separation of concerns** — TypeDB separates rules (derivation logic) from data (facts). We separate Derivers (derivation logic) from projections (materialized state). A Deriver produces commands; a projection materializes the results.

### What we do NOT steal

- **Query-time evaluation** — TypeDB derives facts at query time. We derive at write time (event arrives → Deriver produces commands → commands dispatched). No query-time inference.
- **Recursion termination guarantees** — TypeDB's engine guarantees termination. We rely on developer discipline + documentation. A recursive Deriver that doesn't terminate is a bug, not a framework concern.
- **Rule-as-schema** — TypeDB rules are part of the schema. Our Derivers are application code, not schema declarations.

## Implementation plan (deferred)

1. Define `Deriver` type + `Then`/`Filter` combinators (~100 LOC)
2. Wire into event bus: `bus.SubscribeAll(deriver.AsHandler(dispatcher))` (~50 LOC)
3. Add idempotency check: derived commands carry causation metadata from the source event
4. Example: `example/deriver-demo/` showing event→command derivation

## Consequences

- The Deriver module, when built, will be functional/composable, not declarative.
- No rule registry, no query-time inference, no backward chaining.
- Determinism and idempotency are contracts, not engine-enforced guarantees.
- This ADR locks in the design direction so future implementation doesn't cargo-cult TypeDB's full reasoning model.

## Alternatives Considered

- **Declarative rule registry** (`RegisterRule(name, when, then)`) — rejected: adds complexity without the engine benefits that justify it in TypeDB.
- **Built-in recursion support** — rejected: termination is the developer's responsibility. Document the risk.
- **Query-time derivation** — rejected: opposite of event sourcing's materialized-view philosophy.
