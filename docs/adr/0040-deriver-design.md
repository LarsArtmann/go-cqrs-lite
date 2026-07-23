# ADR-0040: Deriver Module Design

## Status

Accepted — 2026-06-28 (implemented: `deriver/` module)

## Context

The `Deriver` module handles event→command derivation — the stateless saga
pattern where processing one event produces derived commands. This is the
"react to events and dispatch compensating/follow-up commands" use case
that sagas traditionally solve, but expressed as composable pure functions
rather than a stateful orchestration engine.

> **Design inspiration:** TypeDB's `when {...} then {...}` rule model
> informed the key properties we adopted (determinism, idempotency,
> composability). TypeDB's full reasoning engine was not adopted — it would
> be overkill for a library without its own execution engine. See
> Alternatives for what was specifically rejected.

## Decision

**Use a functional/composable Deriver API, not a declarative rule registry.**

### API

```go
// Deriver transforms events into derived commands.
type Deriver func(ctx context.Context, evt event.Event) ([]command.Command, error)

// Then chains derivers: the output of d feeds into next.
func (d Deriver) Then(next Deriver) Deriver

// Filter wraps a deriver to only process matching event types.
func (d Deriver) Filter(types ...event.Type) Deriver
```

### Why functional over declarative

A declarative rule registry (`RegisterRule(name, when, then)`) works for
TypeDB because TypeDB owns the execution engine and can optimize rule
execution. go-cqrs-lite is a library — a declarative registry would add
complexity without the engine benefits. Functional composition is idiomatic
Go: compiler type-checks, easy to test, no hidden execution model.

### Key contracts

1. **Determinism + idempotency** — a Deriver must produce the same output
   for the same event. Re-processing an event must not produce duplicate
   derived commands. The Deriver documentation states this contract.

2. **Forward chaining** — `Deriver.Then(next)` feeds one Deriver's output
   into the next. This is explicit, not engine-optimized backward chaining.

3. **Separation of concerns** — Derivers produce commands; projections
   materialize state. A Deriver is NOT a projection and vice versa.

## Implementation

Implemented in the `deriver/` module: `Deriver` type, `Then`/`Filter`/`Idempotent`
combinators, and `AsHandler` for wiring into the event bus via `bus.SubscribeAll`.
Derived commands carry causation metadata from the source event for idempotency.

## Consequences

- The Deriver module is functional/composable, not declarative.
- No rule registry, no query-time inference, no backward chaining.
- Determinism and idempotency are contracts, not engine-enforced guarantees.
- Recursion termination is the developer's responsibility — documented risk.

## Alternatives Considered

- **Declarative rule registry** (`RegisterRule(name, when, then)`) — rejected: adds complexity without the engine benefits that justify it in TypeDB. A library without its own execution engine gains nothing from declarative rules.
- **Built-in recursion support** — rejected: termination is the developer's responsibility. Document the risk rather than enforcing a terminator.
- **Query-time derivation** — rejected: opposite of event sourcing's materialized-view philosophy.
- **Full TypeDB-style reasoning** — rejected: TypeDB's strengths (symbolic reasoning, rule-as-schema, n-ary relations with typed roles) are database-engine concerns. They don't map to a library running against other databases.
