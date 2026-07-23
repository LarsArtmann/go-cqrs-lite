# ADR-0001: Decider Pattern Over OO Aggregate

**Status:** Accepted  
**Date:** 2026-05-03

## Context

The library originally provided `core/aggregate/` with a traditional OO approach: `Root` interface with 9 methods (`ApplyEvent`, `LoadEvents`, `Changes`, etc.), mutable state, and inheritance-based design. Consumers had to implement a concrete aggregate struct embedding `Core`, manage mutable state, and implement the full interface.

This creates friction:

- Mutable state is hard to test in isolation
- 9-method interface is verbose for simple aggregates
- Embedding `Core` couples domain logic to library infrastructure
- No compile-time safety that `ApplyEvent` handles all event types

## Decision

Introduce `core/decider/` as the recommended pattern for new consumers:

- `Decider[State]` struct holds `Initial` state and a pure `Fold` function
- `DecideFunc[State]` takes `(state, version) → ([]events, error)`
- `Repository[State]` handles load → fold → decide → save → publish
- Zero infrastructure knowledge in domain logic
- Full testability without mocking — pass initial state, fold, decide as functions

The OO `aggregate` package was removed in Session 99 (2026-05-03). Only the Decider pattern remains. See [ADR-0058](0058-rename-aggregate-to-stream.md) for the subsequent identity-type rename (`Aggregate*` → `Stream*`).

## Consequences

**Positive:**

- Pure functions — testable without mocks, deterministic, no side effects
- Generic state type — compile-time type safety for aggregate state
- Fewer concepts — `Decider` + `Repository` vs `Root` + `Core` + `Repository` + `EventSourcedRepository`
- Better alignment with functional programming principles (the library's stated preference)

**Negative:**

- ~~Two aggregate patterns exist (`aggregate` + `decider`)~~ — `aggregate/` was removed in Session 99
- ~~`SnapshotStrategy` is duplicated~~ — resolved when `aggregate/` was removed
- Consumers familiar with OO aggregates may need to learn the functional approach

**Neutral:**

- Both patterns shared the same `event.Store` and `event.Bus` infrastructure (historical)
