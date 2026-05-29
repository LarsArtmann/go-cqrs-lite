# DerivedEvent — Event Composition / Complex Event Processing

> A `DerivedEvent` is an event produced not from a single command, but from the combination of multiple events or multiple other conditions being met.

## Concept

```
Events[] → derive() → Event
```

A deterministic derivation function that watches multiple event streams and produces a new event when a condition is met. Unlike commands (which are imperative), derived events are **declarative** — they describe *what happened* as a consequence of observed facts.

## Relation to Existing Primitives

| Primitive | Input | Output | Stateful? |
|---|---|---|---|
| **Decider** | Command + State | Events | Yes (aggregate state) |
| **Projection** | Events | Read model | Yes (projection state) |
| **Saga** | Command + State | Commands | Yes (saga state machine) |
| **DerivedEvent** | Events | Events | Possibly (derivation checkpoint) |

go-cqrs-lite already has:
- **`projection`** — watches events, writes to read models, not event streams
- **`saga`** — orchestrates commands across multiple steps, but is command-driven

What's missing is the **"Events → Event"** primitive — a deterministic derivation that doesn't need a saga's state machine.

## Example Use Cases

- "When `OrderPlaced` + `PaymentReceived` → emit `OrderConfirmed`"
- "When 3 `SensorReading` events exceed threshold → emit `AlertRaised`"
- "When `UserCreated` + `EmailVerified` + `ProfileCompleted` → emit `UserOnboarded`"
- "When `InvoiceCreated` and `PaymentFailed` after 3 retries → emit `PaymentDefaulted`"

## Proposed Interface

```go
type Deriver interface {
    // SourceTypes declares which event types this deriver subscribes to.
    SourceTypes() []string

    // Derive inspects a batch of events and returns zero or more derived events.
    // Called with events in aggregate order. May be called multiple times as new events arrive.
    Derive(ctx context.Context, events []event.ImmutableEvent) ([]event.ImmutableEvent, error)
}
```

A runner would subscribe to the declared event types, buffer/order them, and call `Derive` for each relevant batch.

## Key Design Questions

### 1. Where does the derived event land?

| Option | Trade-off |
|---|---|
| Same aggregate/stream | Simple, but couples derivation to aggregate lifecycle |
| Different aggregate/stream | Cleaner separation, but needs cross-aggregate write |

**Recommendation:** Different stream. Derivation results are owned by the deriver, not the source aggregate.

### 2. Idempotency

The deriver must not emit the same derived event twice for the same input events.

| Strategy | How |
|---|---|
| Checkpoint tracking | "Last derived at position N" per deriver |
| Deterministic event ID | Hash input event IDs → derived event ID (idempotent append) |
| Deduplication store | Track `(deriver, source_event_ids) → derived_event_id` |

**Recommendation:** Deterministic event ID from input event IDs. Aligns with the library's append-only, no-delete philosophy.

### 3. Ordering

Must the deriver see events in causal order? If so, it needs a journal/subscription with ordering guarantees.

**Recommendation:** Start with ordered delivery per source type. Cross-type ordering is the consumer's responsibility (same as projection today).

### 4. State

Is the deriver pure (stateless, derives from the event batch alone) or stateful (tracks what it has seen across calls)?

- **Stateless:** Simpler, but can only react to the current batch. Misses multi-batch patterns.
- **Stateful:** Needs a checkpoint or state store. More powerful, but more complex.

**Recommendation:** Start stateless. Stateful derivation can be built on top (consumer manages state in a projection).

## Relationship to Existing Modules

- **`projection`**: A derived event is a "projection that writes events instead of read models." Could share the runner infrastructure.
- **`saga`**: Sagas produce *commands*. Derived events produce *events*. Different semantics — events are facts (already happened), commands are requests (may be rejected).
- **`stream`**: Could use `stream` for aggregate listing and tombstone detection on source aggregates.

## Open Questions

1. Should the deriver support **time windows** ("emit if A and B arrive within 10 minutes")? This would require a scheduler/timer component.
2. Should derived events carry **provenance metadata** (which source events produced them)?
3. How does this interact with **tombstones**? If a source aggregate is tombstoned, should derived events be retracted?
4. What about **derived event chains** (a derived event triggers another derivation)? Needs cycle detection.
