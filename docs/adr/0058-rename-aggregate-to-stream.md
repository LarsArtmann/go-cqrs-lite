# ADR-0058: Rename Aggregate* to Stream*

**Status:** Proposed
**Date:** 2026-07-23
**Supersedes:** The identity-type naming from ADR-0001 (behavioral decision still stands)

## Context

ADR-0001 decomposed the traditional DDD Aggregate (Identity + State + Behavior) into three clean pieces: `Decider[State]` for behavior, `AggregateRef` for identity, and events for state. The OO aggregate root was killed. But the name "Aggregate" survived on the identity types.

This creates a split brain. The library dismantled the concept the word refers to — there is no aggregate object, no aggregate state, no aggregate behavior. What remains is a **stream partition key**: an identifier that groups events into an ordered, append-only, independently-versioned sequence. The code itself acknowledges this: `AggregateRef.StreamKey()` is the method that returns the canonical stream key.

The word "Aggregate" is actively misleading because:

1. **It promises a concept that isn't there.** DDD practitioners expect a stateful root object with encapsulated rules. This library has none of that.
2. **It has to be actively un-taught.** `DOMAIN_LANGUAGE.md:330` carries an anti-pattern table entry: "Aggregate Root (OO) → Decider (pure functions)." The name creates a cognitive burden the library must repeatedly counter.
3. **Consumers don't engage with it.** A survey of 14 active consumer projects shows the majority use the types mechanically as opaque stream keys — only 2 of 14 (cqrs-htmx, Standup-Killer) demonstrate genuine DDD aggregate design thinking. 7 of 14 use them purely as partition keys (DiscordSync, timesheets, invoices, etc.).

Meanwhile, the library already researched **Aggregateless Event Sourcing** (Rico Fritzsche, 2025) in three documents under `docs/research/archive/`. The conclusion was: _"The aggregate is no longer sacred."_ The pure-function fold was absorbed into ADR-0001. Only the stream partition key was retained — for operational reasons (loading, versioning, snapshots).

### What the Type Actually Does

`AggregateRef` plays three mechanical roles, all of which describe a stream partition key:

| Role                 | What it does                                                                        |
| -------------------- | ----------------------------------------------------------------------------------- |
| Stream partition key | Groups events: `Load(ctx, ref)` returns all events for that key, ordered by version |
| Concurrency boundary | `expectedVersion` is checked per-key — one writer at a time per partition           |
| Snapshot key prefix  | `cqrs_snapshot:{Type}:{ID}` — snapshot stored per partition                         |

## Decision

Rename the identity types from `Aggregate*` to `Stream*`:

| Current           | New            | Role                                                              |
| ----------------- | -------------- | ----------------------------------------------------------------- |
| `AggregateMarker` | `StreamMarker` | Phantom type for compile-time branding (zero bytes, zero methods) |
| `AggregateID`     | `StreamID`     | Identifier for a specific event stream instance                   |
| `AggregateType`   | `StreamType`   | Category label (`"User"`, `"Order"`) — the stream namespace       |
| `AggregateRef`    | `StreamRef`    | `{Type, ID}` composite passed to all Store methods                |

### Why "Stream"

**1. It's what the thing IS.** A stream of events. Ordered, append-only, versioned. That is the literal definition of the abstraction.

**2. Industry consensus.** The three most influential event sourcing infrastructures use "stream":

- **EventStoreDB** — streams are the primary abstraction (`User-123` is a stream)
- **Marten** — `streamId` internally
- **NEventStore** — `streamId`
- **Kafka** — partition = ordered stream (topic + key = type + ID)

Axon Framework uses "aggregate" — but Axon has ACTUAL aggregate objects with state and behavior. go-cqrs-lite killed those (ADR-0001). Using "aggregate" for what remains is inherited vocabulary from a pattern no longer implemented.

**3. The codebase already acknowledges it.** `AggregateRef.StreamKey()` exists today (`id/aggregate_type.go:46`). The method that returns the canonical key is already called `StreamKey`. The code knows what it is.

**4. Reads naturally in every API position:**

```go
// Construction
const UserType id.StreamType = "User"
streamID := id.NewStreamID()
ref := id.NewStreamRef(UserType, streamID)

// Events
evt.StreamID()     // "which stream does this event belong to?"
evt.StreamType()   // "what type of stream?"

// Commands
c.StreamID()       // "which stream does this command target?"

// Store
store.Load(ctx, ref)              // "load this stream"
store.Save(ctx, ref, events, v)   // "append to this stream"

// Versioning
expectedVersion    // "expected version of the stream" — already correct!
```

**5. Serves both consumer types.** DDD-engaged consumers: "the User's event stream" — natural. Mechanical consumers: "stream `discord.message:123`" — exactly what they mean.

**6. Doesn't lie.** "Aggregate" promises a DDD concept that isn't there. "Stream" promises an ordered sequence of events — which is exactly what's there.

**7. Survives refactoring.** Tied to the fundamental nature of event sourcing, not to a design pattern that was dismantled.

### Why Exactly Three Real Types (Not Four)

The marker is plumbing, not a type:

```
StreamID     = concept (which stream)
StreamType   = concept (what kind)
StreamRef    = bundle  ({Type, ID} — ergonomic, prevents bugs)
StreamMarker = plumbing (zero-byte compile-time tag, invisible)
```

**StreamID** and **StreamType** must be separate because they have different lifecycles: StreamID is generated or derived (permanent, opaque), StreamType is a human-chosen label (could be renamed, queryable, a SQL column, a Pebble prefix namespace). Merging them means renaming `"User"` to `"Account"` changes every ID.

**StreamRef** exists because both values are always passed together. Without it, every store method takes two string-ish args (swappable = silent bug), every map key doubles in arity, and there is no single value to validate.

### Full Rename Map

| Current                           | New                            |
| --------------------------------- | ------------------------------ |
| `AggregateMarker`                 | `StreamMarker`                 |
| `AggregateID`                     | `StreamID`                     |
| `AggregateType`                   | `StreamType`                   |
| `AggregateRef`                    | `StreamRef`                    |
| `NewAggregateID()`                | `NewStreamID()`                |
| `NewAggregateRef()`               | `NewStreamRef()`               |
| `ParseAggregateID()`              | `ParseStreamID()`              |
| `ParseAggregateIDStrict()`        | `ParseStreamIDStrict()`        |
| `DeriveAggregateID()`             | `DeriveStreamID()`             |
| `AggregateIDFrom()`               | `StreamIDFrom()`               |
| `AggregateTimestamp()`            | `StreamTimestamp()`            |
| `IsAggregateIDULID()`             | `IsStreamIDULID()`             |
| `ParseAggregateType()`            | `ParseStreamType()`            |
| `ErrEmptyAggregateType`           | `ErrEmptyStreamType`           |
| `event.NewAggregateRef()`         | `event.NewStreamRef()`         |
| `evt.AggregateID()`               | `evt.StreamID()`               |
| `evt.AggregateType()`             | `evt.StreamType()`             |
| `c.AggregateID()`                 | `c.StreamID()`                 |
| `listing.AggregateListing`        | `listing.StreamListing`        |
| `listing.AggregateStatus`         | `listing.StreamStatus`         |
| `listing.InMemoryAggregateReader` | `listing.InMemoryStreamReader` |
| `storage.SQLAggregateReader`      | `storage.SQLStreamReader`      |
| `AggregateRef.StreamKey()`        | stays (already correct)        |
| `Decider[State]`                  | stays (already correct)        |
| `Repository[State]`               | stays (already correct)        |
| `expectedVersion`                 | stays (already correct)        |

## Alternatives Considered

### Keep "Aggregate" (status quo)

Pros: No migration cost. Matches Axon Framework's vocabulary.

Cons: Actively misleading. The library killed the OO aggregate (ADR-0001), researched and absorbed aggregateless ES ideas, and documented the aggregate as "no longer sacred." Keeping the word creates cognitive dissonance that must be repeatedly un-taught. Consumer surveys confirm most don't engage with the DDD meaning anyway.

### Other names (TraceKey, IdentityKey, OriginKey, PartitionKey)

All rejected — they name the mechanism ("it's a key") rather than the abstraction ("it's a stream of events"). Per the project's naming principles: name for the role, not the mechanism.

### Merge to fewer types

Rejected — StreamID, StreamType, and StreamRef each serve distinct roles with different lifecycles (see [Why Exactly Three Real Types](#why-exactly-three-real-types-not-four)).

## Consequences

**Positive:**

- Honest naming: the types describe what they actually are
- Eliminates the split brain between ADR-0001 (killed the aggregate) and the type names (kept the word)
- Aligns with EventStoreDB, Marten, NEventStore vocabulary
- Natural API readability: `store.Load(ctx, streamRef)` reads as intent, not jargon
- Consumers using the types as opaque keys (the majority) get a clearer mental model
- `StreamKey()` method name and `StreamRef` type align — no vocabulary mismatch

**Negative:**

- Breaking change for all 14+ active consumers — every import, type reference, and method call using `Aggregate*` must change
- Migration effort: mechanical find-and-replace across consumer codebases, but semantically straightforward (1:1 mapping, no behavior change)
- Type aliases for backward compatibility can ease migration but add cleanup debt
- `DOMAIN_LANGUAGE.md` requires significant revision (anti-pattern table entries, identity section, consistency guarantees section)
- SQL schemas using `aggregate_id` / `aggregate_type` column names remain unchanged (storage-level concern, not API-level)

**Neutral:**

- The `aggregate/` package was already removed in Session 99 (referenced in ADR-0001). No legacy package to rename.
- ADR-0001's behavioral decision (Decider over OO aggregate) is unaffected. This ADR only renames identity types.

## Migration Strategy

The rename is a 1:1 mechanical mapping with no behavior change. Options:

1. **Hard break (next major):** rename all types and methods, provide a migration guide. Consumers run a project-wide find-and-replace using the table above.
2. **Type aliases (transition period):** keep `type AggregateID = StreamID` as an alias, mark as deprecated. Consumers migrate incrementally. Remove aliases in the following major version.

Option 1 is recommended — the transition period adds complexity and the mapping is purely mechanical. The rename should ship in the next major version with a migration guide generated from the rename map above.

## References

- [ADR-0001: Decider Pattern Over OO Aggregate](0001-decider-over-aggregate.md) — the behavioral decision this ADR completes
- [AGGREGATE-CONCEPT-ANALYSIS](../architecture-understanding/AGGREGATE-CONCEPT-ANALYSIS.md) — full analysis of the Aggregate concept in this library
- [Aggregateless ES Deep Dive](../research/archive/2026-05-01_AGGREGATELESS_EVENT_SOURCING_DEEP_DIVE.md) — research that informed the pure-function approach
- [CQRS Event Sourcing Innovations](../research/archive/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md) — "The aggregate is no longer sacred" (line 501)
- [DOMAIN_LANGUAGE.md](../DOMAIN_LANGUAGE.md) — anti-pattern table and identity definitions (to be updated)
- `id/aggregate_type.go:46` — `StreamKey()` method that already uses the correct name
- Rico Fritzsche: [Aggregateless Event Sourcing](https://ricofritzsche.me/aggregateless-event-sourcing/)
- EventStoreDB: [Streams](https://developers.eventstore.com/server/v23.10/docs/streams/)
