# DerivedEvent — Event Composition / Complex Event Processing

> A `DerivedEvent` is an event produced not from a single command, but from the combination of multiple events or multiple other conditions being met.

**Date:** 2026-05-29
**Status:** Research — refined after command-event causality analysis

## Core Invariant

> **Every event traces to a command. No exceptions.**

This means derived events don't bypass the command→event pipeline — they produce **derived commands** instead.

## Concept: Derived Commands, Not Derived Events

### Old model (rejected)

```
Events[] → derive() → Event     ← breaks command→event invariant
```

### New model

```
Events[] → derive() → Command → Decider → Events
```

The derivation layer is a **command factory**. It observes events and emits commands, not events. The command goes through the normal pipeline: validation, middleware, metrics, decider. Events are only ever produced by deciders processing commands.

### Why commands, not events?

- **Rejection is a feature**: By the time the derived command executes, state may have changed. The decider validates against current reality. A raw derived event would bypass that.
- **Reuses everything**: Middleware, tracing, metrics, logging — all already wired for the command→event path.
- **Command carries intent**: `ConfirmOrder` (a command) is clearer than `OrderConfirmed` (an event asserted without validation).
- **Sagas already do this**: `Event → Saga → Command → Events`. A derivation is a simpler, stateless version of the same pattern.
- **Preserves 1:1 invariant**: Every event type is produced by exactly one command type.

## The 1:1 Command–Event Invariant

Every event type maps to exactly one command type:

```
CreateTodo          → TodoCreated
UpdateTodo          → TodoUpdated
DeleteTodo          → TodoDeleted
ConfirmOrder        → OrderConfirmed        ← derived command, same shape
ReceiveWebhook      → ExternalEventIngested ← external, same shape
ScheduleTimeout     → TimeoutExpired        ← timer, same shape
```

Consequences:

- **No CommandType metadata needed** — structurally derivable from the event type itself
- **Catalog is trivial** — one table, two columns, no ambiguity
- **Event naming is deterministic** — `DoX` → `XDone`
- **Audit trail is complete** — "which command produced this event?" is always answerable
- **Derived events don't break the rule** — they go through a derived command

### What about commands that produce multiple events?

Two positions:

| Position        | Rule                                                                              | Trade-off                                             |
| --------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------- |
| **Strict 1:1**  | One command → one event. Split `CreateOrder` into `CreateOrder` + `AddOrderLines` | Purest invariant, more commands, very granular events |
| **1:N bounded** | One command → N events, but each event type belongs to exactly one command type   | Pragmatic, reverse direction still unique             |

Strict 1:1 is the aspirational target. 1:N bounded is acceptable if the aggregate truly needs atomic multi-event writes.

## Unified Primitive Table

| Primitive      | Input           | Output     | Stateful?                |
| -------------- | --------------- | ---------- | ------------------------ |
| **Decider**    | Command + State | Events     | Yes (aggregate state)    |
| **Projection** | Events          | Read model | Yes (projection state)   |
| **Saga**       | Events          | Commands   | Yes (saga state machine) |
| **Derivation** | Events          | Commands   | No (stateless)           |

Note: both **Saga** and **Derivation** produce commands, not events. The only difference is statefulness. A derivation is a stateless saga — or equivalently, a saga is a stateful derivation.

## Example Use Cases

- "When `OrderPlaced` + `PaymentReceived` → dispatch `ConfirmOrder` command"
- "When 3 `SensorReading` events exceed threshold → dispatch `RaiseAlert` command"
- "When `UserCreated` + `EmailVerified` + `ProfileCompleted` → dispatch `OnboardUser` command"
- "When `InvoiceCreated` and `PaymentFailed` after 3 retries → dispatch `DefaultPayment` command"

## Proposed Interface

```go
type Deriver interface {
    // SourceTypes declares which event types this deriver subscribes to.
    SourceTypes() []string

    // Derive inspects a batch of events and returns zero or more commands.
    // Commands are dispatched through the normal command pipeline (validation, middleware, decider).
    Derive(ctx context.Context, events []event.Event) ([]command.Command, error)
}
```

A runner subscribes to the declared event types, buffers/orders them, calls `Derive` for each batch, and dispatches returned commands through the command dispatcher.

### Derived command metadata

A derived command sets:

- `Source: "derivation"` or `"derived:OrderConfirmationRule"`
- `CausationID`: hash of the triggering event IDs (links back to provenance)
- `CorrelationID`: propagated from the source events

This makes the full audit trail: source events → derived command → decider → derived event. Every link is traceable.

## Key Design Questions

### 1. Where does the derived command target?

The derived command targets an aggregate — same as any command. The derivation decides _which_ aggregate based on the source events.

- Same aggregate as source events → simple, natural for invariants within one aggregate
- Different aggregate → cross-aggregate invariant, the decider on the target validates independently

**Recommendation:** No restriction. The deriver specifies the target `AggregateID` in the returned command.

### 2. Idempotency

The deriver must not dispatch the same command twice for the same input events.

| Strategy                 | How                                                           |
| ------------------------ | ------------------------------------------------------------- |
| Checkpoint tracking      | "Last derived at position N" per deriver                      |
| Deterministic command ID | Hash input event IDs → command ID (deduplicate at dispatcher) |
| Consumer-side dedup      | Decider rejects if event already exists (version conflict)    |

**Recommendation:** Deterministic command ID from input event IDs + decider's natural idempotency (version check). Belt and suspenders.

### 3. Ordering

Must the deriver see events in causal order?

**Recommendation:** Start with ordered delivery per source type. Cross-type ordering is the consumer's responsibility (same as projection today).

### 4. State

Is the deriver pure (stateless) or stateful?

- **Stateless:** Simpler. Can only react to the current batch. Misses multi-batch patterns.
- **Stateful:** Needs a checkpoint or state store. More powerful. Equivalent to a saga.

**Recommendation:** Start stateless. Stateful derivation is a saga — use the existing `saga` module for that case.

## Relationship to Existing Modules

- **`projection`**: Derivation is a "projection that dispatches commands instead of writing read models." Could share the runner infrastructure.
- **`saga`**: Sagas produce commands statefully. Derivations produce commands statelessly. Same output type, different lifecycle. Derivation is strictly simpler.
- **`stream`**: Could use `stream` for aggregate listing and tombstone detection on source aggregates.
- **`command`**: Derived commands are regular commands. No special type needed — just metadata to mark provenance.

## Impact on command-event-causality.md

This refinement eliminates the original "Option C" problem (events requiring commands at construction):

- Replay/rehydration reads events — never produces new ones → unaffected
- Derived events now go through commands → no bypass
- External ingestion already modeled as commands → already aligned
- Timer triggers modeled as commands → already aligned

See `docs/research/command-event-causality.md` for the full options analysis.

## Open Questions

1. Should the deriver support **time windows** ("dispatch if A and B arrive within 10 minutes")? This requires a scheduler/timer component — essentially a stateful saga.
2. How does this interact with **tombstones**? If a source aggregate is tombstoned, should derived commands be suppressed?
3. What about **derivation chains** (a derived event triggers another derivation)? Since derived events go through commands→decider, the chain is: `Events → Deriver → Command → Decider → Event → Deriver → ...`. Cycles are prevented naturally if the decider rejects (state unchanged).
4. Should derivation be a **separate module** (`derivation`) or live inside `saga` as a stateless variant?
