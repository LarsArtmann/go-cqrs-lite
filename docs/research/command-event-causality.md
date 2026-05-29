# Command-Event Causality Research

**Date:** 2026-05-29
**Status:** Research — no implementation yet

## Problem Statement

Events and commands in `core/event` and `core/command` are structurally disconnected. No event knows which command produced it. This violates a core CQRS invariant: **every event traces back to intent**.

## Current State: Events and Commands are Strangers

| | Command | Event |
|---|---|---|
| Has | `Type`, `AggregateID`, `Metadata` | `Type`, `AggregateID`, `AggregateType`, `Version`, `Payload`, `Metadata` |
| Metadata | `CorrelationID`, `CausationID`, `UserID`, `RequestID` | Same + `Source`, `IPAddress`, `UserAgent`, `Custom` |

### What's Missing

1. Events have **no `CommandType`** — you can't answer "which command produced this event?"
2. Events have **no `CommandID`** — you can't trace back to the specific command instance
3. `DecideFunc` receives `(State, Version)` — **no command reference at all** (`core/decider/decider.go:83`)
4. In the example, `CreateTodoHandler.Handle()` calls `aggregate.DecideCreate()` — command metadata (UserID, CorrelationID) is **never propagated** to the produced events
5. `CausationID` exists but is optional and semantically vague — it could be anything, not specifically "the command that caused this event"

### The Problem in Practice

From `example/todo/commands/create_todo.go:56-72`:

```go
func (h *CreateTodoHandler) Handle(ctx context.Context, cmd command.Command) error {
    return h.execute(
        ctx, createCmd.AggregateID(),
        aggregate.DecideCreate(           // ← loses all command context
            createCmd.AggregateID(),
            createCmd.Title, ...
        ),
    )
}
```

`DecideCreate` returns a `DecideFunc` that creates events with zero knowledge of the command. No command type, no command ID, no UserID propagation. The link is severed.

## Why Every Event Must Trace to a Command

Every event in a CQRS system traces back to intent:

1. **Direct**: `Command → Decide → Events` (the happy path)
2. **Saga-chained**: `Event → Saga → Command → Events` (still command→event at each step)
3. **External**: External system event → modeled as `ReceiveExternalEvent` command → events
4. **Time-triggered**: Cron → modeled as `ScheduledCommand` → events
5. **Replay/rehydration**: Events were historically caused by commands — the causation is a historical fact

The only exception is **projections/read models**, which consume events but don't produce new domain events. They're not part of this concern.

## Design Options

### Option A: Lightweight — CommandType in Event Metadata

Add `MetadataKeyCommandType` and `MetadataKeyCommandID` to event metadata. An enricher in the handler layer propagates automatically.

- **Pros**: Minimal change, backward compatible, no API breakage
- **Cons**: Not structurally enforced — easy to forget, no compile-time guarantee
- **Scope**: `core/event/options.go` + a new middleware/enricher

### Option B: Structural — DecideFunc receives Command

Change the decider contract:

```go
// Now:
DecideFunc[State any] func(state State, version event.Version) ([]event.Event, error)

// Proposed:
DecideFunc[State any] func(state State, version event.Version, cmd command.Command) ([]event.Event, error)
```

The repository's `Execute` takes a `command.Command` and auto-propagates `CommandType`, `CommandID`, `CorrelationID`, `UserID` to all produced events.

- **Pros**: Enforces the invariant at the decider layer (where events are produced), metadata propagation is automatic, decider function stays pure (receives command as data)
- **Cons**: Breaking change to `DecideFunc` signature, affects all consumers of `decider.Repository.Execute()`
- **Scope**: `core/decider/decider.go`, all `DecideFunc` callers

### Option C: Strong — Event requires Command at construction

`event.New()` requires a `command.Command` (or `command.Reference`) as the first argument.

- **Pros**: Makes the invariant impossible to violate at the lowest level
- **Cons**: Most opinionated, breaks the most code, couples `core/event` to `core/command` (currently independent packages), problematic for event replay/rehydration where no command exists
- **Scope**: `core/event/event_new.go`, all event creation sites

### Option D: Command-aware Decider (new type)

Keep existing `Decider` for backward compat, add a `CommandDecider` that takes a Command and returns events with automatic metadata propagation. Consumers opt in.

- **Pros**: No breaking changes, gradual adoption
- **Cons**: Two parallel APIs, more complexity in `core/decider`, doesn't enforce the invariant universally
- **Scope**: New type in `core/decider/`, new `Execute` overload

## Trade-off Matrix

| Criterion | A: Metadata | B: DecideFunc | C: Required arg | D: New type |
|---|---|---|---|---|
| Structural enforcement | No | Yes | Yes | Opt-in |
| Breaking change | None | Medium | Large | None |
| core/event ↔ core/command coupling | None | Weak (interface) | Strong | Weak |
| Replay/rehydration safe | Yes | Yes | Problematic | Yes |
| Automatic propagation | No (manual) | Yes | Yes | Yes |

## Preliminary Recommendation

**Option B** is the sweet spot:

- Enforces the invariant at the decider layer (where events are actually produced)
- `command.Command` is a small interface — `Type()`, `AggregateID()`, `Metadata()` — no heavy coupling
- The decider function stays pure (receives command as data, doesn't dispatch)
- Metadata propagation (CorrelationID, UserID, CommandType) becomes automatic
- Replay/rehydration is unaffected — it uses `Load`, not `Execute`

The `command.Command` interface would need to be importable by `core/decider`. Currently `core/decider` doesn't depend on `core/command`. This is a new dependency edge — but `core/command` is a leaf with no transitive deps beyond `core/pkg/id`, so the coupling is acceptable.

## Open Questions

1. Should `command.Command` grow an `ID()` field for per-instance traceability?
2. Should derived events (from sagas) carry the *original* command type or the saga's internal command type?
3. How does this interact with `event.Builder` — should it also require command context?
4. Is a `CommandReference` type (just Type + ID + Metadata, no payload) useful to avoid importing the full `command.Command`?
