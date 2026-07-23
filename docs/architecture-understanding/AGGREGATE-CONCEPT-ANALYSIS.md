# The "Aggregate" Concept in go-cqrs-lite

> A deep analysis of what "Aggregate" means in this library, why the name persists despite the OO concept being dismantled, and how it relates to Events, State, Projections, Queries, and the broader aggregateless event sourcing movement.

---

## Table of Contents

1. [The Decomposition](#1-the-decomposition)
2. [Why State Is Never Stored](#2-why-state-is-never-stored)
3. [How the Decider Makes It Practical](#3-how-the-decider-makes-it-practical)
4. [Relationship to Queries and Projections](#4-relationship-to-queries-and-projections)
5. [AggregateRef as Partition Key](#5-aggregateref-as-partition-key)
6. [The Naming Question](#6-the-naming-question)
7. [Consumer Usage Patterns](#7-consumer-usage-patterns)
8. [Aggregateless Event Sourcing: What Was Researched and Rejected](#8-aggregateless-event-sourcing-what-was-researched-and-rejected)
9. [The Honest Assessment](#9-the-honest-assessment)

---

## 1. The Decomposition

In traditional DDD, an Aggregate is a monolithic concept that bundles three concerns into one object:

```
Traditional DDD Aggregate = Identity + State + Behavior
```

This library decomposed that god concept into three clean pieces, each living in a different module:

| Concern | Traditional DDD | This Library | Type / Location |
|---|---|---|---|
| **Identity** | Aggregate Root holds its own ID | `AggregateRef{Type, ID}` | `id/aggregate_type.go` |
| **Behavior** | Aggregate methods (ApplyEvent, LoadEvents, Changes...) | Pure `Decider[State]` with `Apply` function | `decider/decider.go` |
| **State** | Mutable fields on the aggregate object | Folded from events on demand, never persisted | App-defined `State` type parameter |

ADR-0001 (`docs/adr/0001-decider-over-aggregate.md`) documents this decision. The original `core/aggregate/` package had a 9-method OO interface (`Root` with `ApplyEvent`, `LoadEvents`, `Changes`, etc.) that coupled domain logic to infrastructure. It was replaced by:

```go
type Decider[State any] struct {
    Initial State
    Apply   func(state State, evt event.Event) (State, error)
}
```

Two fields. One pure function. Zero infrastructure knowledge.

The old `aggregate/` package was flagged for deprecation — its 70% structural overlap with `decider/` was documented in `docs/architecture-understanding/archive/2026-05-21_00-20-SESSION_84_ARCHITECTURE_REVIEW.md:81`.

---

## 2. Why State Is Never Stored

**Events are the integral; state is the derivative.** The library stores the complete, append-only history (the integral) and computes the current state (the derivative) on demand.

If you store state, you lose:

| Lost | Consequence |
|---|---|
| **How you got here** | No audit trail — you see *what is* but never *why* |
| **When things changed** | No temporal queries — can't reconstruct state at time T |
| **The ability to evolve** | Can't re-derive state with new business rules — the old logic is baked in |
| **Debuggability** | Can't replay to find exactly which event caused a bad state |
| **Concurrency simplicity** | Read-modify-write races; optimistic concurrency becomes hard |

An event is a **fact** — it happened, it's immutable, it's undeniable. State is an **interpretation** of facts. Persisting interpretation over facts is storing the derivative and throwing away the integral. You can always re-derive, but you can never recover discarded history.

`DOMAIN_LANGUAGE.md:331` encodes this as an anti-pattern:

| Instead of | We say | Why |
|---|---|---|
| "State" (mutable) | "Folded state" | State is always reconstructed from events via `Apply`, never directly mutated |

### The Snapshot Exception

Snapshots do NOT violate this principle. A snapshot is cached folded state at version N, so you can fold from N instead of 0. The events remain the source of truth. Throw the snapshot away, re-fold from events, get the same state. A snapshot is a performance optimization, not a second source of truth.

Storage key pattern: `cqrs_snapshot:{aggregateType}:{aggregateID}` — one snapshot per stream, overwritten by newer versions.

---

## 3. How the Decider Makes It Practical

Without a Decider, "never store state" means manually replaying events everywhere — unworkable. The Decider makes state computation a **pure function** — a left fold:

```
Initial → Apply(Initial, Event₁) → Apply(State, Event₂) → ... → Stateₙ → discard
```

State exists for **microseconds** — long enough to make a decision, then it's gone.

### What Is `State`?

`State` is an unconstrained generic type parameter: `type Decider[State any]`. The library doesn't know or care what it is. The consumer defines the shape:

```go
type UserState struct {
    Name  string
    Email string
}

d := decider.Decider[UserState]{
    Initial: UserState{},
    Apply: func(state UserState, evt event.Event) (UserState, error) {
        switch evt.Type() {
        case "UserCreated":
            state.Email = string(evt.Payload())
        case "UserNameChanged":
            state.Name = string(evt.Payload())
        }
        return state, nil
    },
}
```

The library's entire relationship with `State`:
- Hold the `Initial` value
- Feed events through `Apply` one at a time
- Pass the resulting state to the consumer's decide function
- Discard state after the command completes

This gives you:
- **Testability** — pass in events, check the resulting state. No database, no mocks.
- **Determinism** — same events, same state, every time.
- **Evolution** — change `Apply`, re-fold all events, get a new state shape from old data.

### The Command Lifecycle

Every command execution follows this cycle:

```
Load events → fold through Apply → get current state → decide → emit new events → append → discard state
```

The `Repository[State]` orchestrates this (`decider/decider.go:36`):

1. `Load(ctx, ref)` — fetches all events for the `AggregateRef` (or loads from snapshot + delta)
2. `foldEvents(Initial, events)` — folds through `Apply`
3. Consumer's decide function inspects state + command, returns new events
4. `Save(ctx, ref, newEvents, expectedVersion)` — appends atomically with optimistic concurrency check
5. State is discarded

---

## 4. Relationship to Queries and Projections

The relationship is **strictly one-directional** through the event stream. Aggregates, Projections, and Queries form the write-to-derive-to-read pipeline that gives CQRS its name.

### The Data Flow

```
Command → Decider (Aggregate) → Events → Journal/Bus → Projection → Read Model ← Query
         WRITE SIDE                          DERIVE         READ SIDE
```

Each stage is decoupled and knows nothing about the next.

### The Two Folds

There are **two folds of the same events**, for two different purposes:

| | Write-side fold | Read-side fold |
|---|---|---|
| **Who** | `Decider.Apply` | `Projection.Handle` |
| **Purpose** | Make decisions (validate invariants) | Answer queries (O(1) reads) |
| **Shape** | Normalized domain state | Denormalized, query-optimized |
| **When** | Synchronous, on every command | Async, eventually consistent |
| **Persisted?** | **NO** — computed and discarded | **YES** — materialized in KV/SQL/graph |

Materialized views ARE the read-side fold. The library provides three tiers of materialized view builders, chosen by query shape:

| Tier | Builder | Store shape | Best for |
|---|---|---|---|
| **Document/KV** | `stack.Materialize[V,K]` | One record per key | Single-entity views (user profile, todo item) |
| **Relational** | `storage.RelationalProjection` | Multiple related SQL tables | Multi-table atomic writes (messages + attachments + junction tables) |
| **Graph** | `graph.GraphProjection` | Nodes + edges | Variable-depth traversal, adjacency, path-finding |

Each projection is a **different denormalization** of the same event stream, optimized for a different query pattern.

### What Connects Them (and What Doesn't)

| Relationship | Exists? | Mechanism |
|---|---|---|
| Aggregate → Projection | **Indirect** | Aggregate produces events to Journal. Projection consumes from Journal. No direct reference. |
| Aggregate → Query | **None** | Completely decoupled. A query never touches an aggregate's event stream directly. |
| Projection → Query | **Implicit** | Projection writes a read model; a query handler reads it. Consumer wires them together. |
| Query → Aggregate | **None** | Queries never mutate. If a query result demands a change, a consumer dispatches a *Command*. |

### The Projection Interface

```go
// projection/projection.go:23
type Projection interface {
    Name() string
    Handle(ctx context.Context, evt event.Event) error
    EventTypes() []event.Type
}
```

Projections read from the **cross-aggregate Journal** (`SeekableJournal.ReadFrom`), not from individual aggregate streams. They transform events into read-optimized shapes.

### Consistency Model

- **Per-aggregate** strong consistency: `expectedVersion` provides single-writer-per-aggregate semantics on the write side.
- **Per-projection** eventual consistency: projections lag behind the log, tracked by `projectionhost.LagDuration()` / `LagPerProjection()`.
- **Read-your-writes is NOT provided** — after a command succeeds, the read model may not yet reflect it. The consumer must poll lag or use returned events for optimistic UI updates.
- **Bounded staleness is consumer-implemented** — check `LagDuration()` before querying and reject if lag exceeds a threshold.

### The AggregateListing Bridge

`listing.AggregateListing` is a specialized projection that lists *aggregates themselves* with their tombstone status — bridging aggregate identity back to the read side. Implementations: `listing.InMemoryAggregateReader` (from Journal), `storage.SQLAggregateReader`.

### Six Rules for Getting Materialized Views Right

1. **Projections must be idempotent and replayable** — the same event WILL be processed multiple times (crash recovery, rebuilds, catch-up). Use upserts keyed by event/aggregate ID, never blind appends.

2. **You must be able to throw away any projection and rebuild from events** — if you can't, you've silently made the projection a source of truth. Never store data in a projection that can't be derived from events. `projectionhost.Host.Reset(ctx, "users")` wipes and replays from zero.

3. **Design read models for the query, not for the domain** — denormalize freely. Duplicate data across projections. One projection per query pattern.

4. **Read-your-writes is YOUR problem** — three strategies: return emitted events for optimistic UI updates; poll `LagDuration()` for bounded staleness; or check if the projection includes the event ID you just wrote.

5. **Measure and bound lag** — wire `LagDuration()` and `LagPerProjection()` to Prometheus. Unmonitored eventual consistency is inconsistency you haven't noticed yet.

6. **Schema evolution is first-class** — when you change `Apply` or `Projection.Handle`, old events still exist with old payload shapes. Use `schema.Upcaster` to migrate payloads on load.

---

## 5. AggregateRef as Partition Key

`AggregateRef` is functionally a **partition key** for event streams. The code itself acknowledges this — the method is literally called `StreamKey()`:

```go
// id/aggregate_type.go:46
func (r AggregateRef) StreamKey() string {
    return r.String()  // returns "Type:ID"
}
```

`AggregateRef` plays exactly three mechanical roles:

| Role | What it does |
|---|---|
| **Stream partition key** | Groups events: `Load(ctx, ref)` returns all events for that key, ordered by version |
| **Concurrency boundary** | `expectedVersion` is checked per-key — one writer at a time per partition |
| **Snapshot key prefix** | `cqrs_snapshot:{Type}:{ID}` — snapshot stored per partition |

That is what a partition key IS. The word "Aggregate" adds zero semantic information beyond "the key that partitions events into ordered, independently-versioned streams."

### The Three Identity Types

The library ships three identity types orbiting a concept that has no behavioral representation in code:

| Type | Definition | Purpose |
|---|---|---|
| `AggregateID` | `cbid.ID[AggregateMarker, string]` | Identifier for a specific stream instance. The only non-ULID-backed ID — string-backed so it accepts `DeriveAggregateID` (SHA-256) and legacy values. |
| `AggregateType` | `type AggregateType string` | Category label (`"User"`, `"Order"`) — namespacing streams. |
| `AggregateRef` | `struct{Type AggregateType; ID AggregateID}` | Composite key passed to all Store methods. |

You never hold an "aggregate" in your hand. You hold its identity, its type, or its ref.

---

## 6. The Naming Question

### Why "Aggregate" Is a Debatable Name for What Remains

1. **The English word "aggregate" means "a collection" or "to gather into a mass."** That tells you nothing about "the unit of write consistency."

2. **The library already killed the thing the name referred to.** The OO Aggregate Root is gone, replaced by the Decider. Keeping the name "Aggregate" for the *leftover identity key* creates a split brain — two concepts (`Decider` = behavior, `Aggregate` = identity) where there used to be one.

3. **There is no `Aggregate` type.** Only `AggregateID`, `AggregateType`, `AggregateRef` — three identity types orbiting a concept that has no representation in code.

4. **The word carries baggage.** People coming from DDD expect a stateful root object with encapsulated rules. Here it's just a stream partition key. The `DOMAIN_LANGUAGE.md:330` anti-pattern table has to actively *un-teach* this expectation.

### The StreamRef / StreamKey Alternative

The rename proposal would be:

| Current | Proposed | Reasoning |
|---|---|---|
| `AggregateID` | `StreamID` | The identifier of a specific event stream |
| `AggregateType` | `StreamCategory` or `StreamType` | The category label (`"User"`, `"Order"`) |
| `AggregateRef` | `StreamRef` | `{Category, ID}` — the composite key passed to Store methods |
| `NewAggregateID()` | `NewStreamID()` | Generate a new stream identity |
| `NewAggregateRef()` | `NewStreamRef()` | Construct the composite ref |

The argument: EventStoreDB and Marten literally use the term "stream" for this exact concept. `AggregateRef.StreamKey()` already exists in the codebase, so the code itself acknowledges the stream framing.

### Why It Was Rejected

Three reasons the name was kept despite the critique:

1. **Industry convention.** EventStoreDB, Axon, Marten, NEventStore all use "Aggregate" for this concept. Renaming breaks the shared vocabulary.

2. **Domain communication.** DDD practitioners recognize "Aggregate" as "the consistency boundary." The term maps to a concept domain experts understand ("an Order," "a User"), even though the implementation is radically different from traditional DDD.

3. **Breaking change cost.** This is a published library with 14+ active consumers. Renaming `AggregateID`, `AggregateType`, `AggregateRef` is a breaking change to every consumer's imports, type signatures, and error messages. The cost is enormous for marginal naming improvement.

### Alternative Names Considered and Rejected

| Candidate | Problem |
|---|---|
| **TraceKey** | Collides with tracing vocabulary (`CorrelationID`, `CausationID`, OTel spans). Suggests observability, not consistency. |
| **IdentityKey** | Same vagueness as "Entity." Every ID in the system is an "identity." Conveys zero domain meaning. |
| **OriginKey** | Implies provenance. Overlaps with `CausationID`. The aggregate key isn't about origin — it's about grouping and consistency. |
| **StreamRef** | Best candidate. Accurately describes the function. But still a breaking rename for industry-conventional naming. |

All alternatives share one flaw: **they name the mechanism (it's a key), not the role (it's a consistency boundary).** Per the project's own naming principles: *"Name for the role, not the mechanism."*

---

## 7. Consumer Usage Patterns

A survey of 14 active consumer projects (across `/home/lars/projects/`) reveals a spectrum of engagement with the "Aggregate" concept:

### The DDD Engagement Spectrum

| Project | DDD Engagement | Nature |
|---|---|---|
| **cqrs-htmx** | **Deep** | Explicit aggregate boundaries (User vs Membership), snapshot strategy, `aggregateRepositories` struct |
| **Standup-Killer** | **Deep** | Named aggregates (Team/Member/Checkin), state-as-aggregate docs, ID bridging layer |
| **bank-sync** | **Moderate** | "one aggregate per (balanceID, provider)", aggregate mapping projection |
| **StopTube** | **Moderate** | `BlockingSession` explicitly documented as "an aggregate" |
| **crush-daily** | **Moderate** | `DailyReportState` documented as "aggregate state" |
| **timesheets** | **Mechanical** | `aggID` is always opaque stream key; no conceptual engagement |
| **DiscordSync** | **Mechanical** | 23 `Aggregate*` constants used only as `Emit()` partition keys; no state/invariants |
| **invoices** | **Mechanical** | SQLC boilerplate; `aggregate_id` as DB column |
| **Code-Quality-Agent** | **Mechanical** | SQLC boilerplate; `aggregate_id` as DB column |

### Usage Pattern Summary

| Pattern | Prevalence |
|---|---|
| `AggregateType` as `event.AggregateType` constant | ALL 14 projects |
| `NewAggregateID` for fresh ULIDs | ALL 14 projects |
| `c.AggregateID()` on commands | Most projects (some embed `BasicCommand`, some manual) |
| `evt.AggregateID()` on events in projections | Most projects |
| `ParseAggregateID` from branded/domain IDs | 9 of 14 projects |
| `DeriveAggregateID` for deterministic IDs | 6 of 14 projects |
| `NewAggregateRef` to construct refs for store methods | 8 of 14 projects |
| `event.AggregateRef` struct literals `{ID, Type}` | 3 of 14 projects |
| `AggregateMarker` for custom branded types | 1 of 14 (browser-history) |
| `AggregateTimestamp` | 0 of 14 projects |

### The Key Finding

The majority of "aggregate" occurrences across consumers are **mechanical API calls** — `id.NewAggregateID()`, `c.AggregateID()`, `AggregateType` constants passed as arguments. Only **cqrs-htmx and Standup-Killer** demonstrate genuine DDD aggregate design thinking (aggregate boundaries, invariants, consistency reasoning). **DiscordSync and timesheets** use the types purely as opaque stream keys without engaging with aggregate invariants, consistency boundaries, or state lifecycle.

This confirms the conceptual critique: the word "Aggregate" is used as a vocabulary convention, not because consumers are engaged in DDD aggregate design.

---

## 8. Aggregateless Event Sourcing: What Was Researched and Rejected

### What Aggregateless Event Sourcing Is

Coined by **Rico Fritzsche** (2025), building on **Sara Pellegrini's** *"Killing the Aggregate"* (2023) and **Ralf Westphal's** *Command Context Consistency*. The core idea:

> **Eliminate aggregates entirely.** Events are standalone facts in one flat table. No streams, no `aggregate_id`, no `aggregate_type`. Each *command* dynamically declares its own consistency boundary by querying the exact events it needs — and the append atomically re-checks that same query.

Instead of:
```sql
SELECT * FROM events WHERE aggregate_id = 'order-42'   -- traditional
```

You do:
```sql
SELECT * FROM events
  WHERE event_type IN ('DeviceRegistered', 'AssetCreated')
    AND payload->>'assetId' = 'asset-7'                -- dynamic context
```

The atomic write uses a CTE — if any event matching that filter was appended since you read, the write is rejected:

```sql
WITH context AS (
  SELECT MAX(sequence_number) AS max_seq
  FROM events
  WHERE <your command's context filter>
)
INSERT INTO events (event_type, payload)
SELECT ... FROM context
WHERE COALESCE(max_seq, 0) = <expected_max_seq>;
```

### The Advantage That Matters

**No false conflicts.** Two commands touching the same "aggregate" but different concerns (rename an item vs. check out inventory) never block each other. The consistency boundary is exactly as wide as the command needs — no wider.

### The Criticism

Gokce Yalcin: *"A stream with an expected-revision check is still an aggregate, just not entity-centric. I'd call it dynamic aggregates rather than aggregate-less."*

### What the Repo Researched

Three documents in `docs/research/archive/`:

| Document | Status | Verdict |
|---|---|---|
| `2026-05-01_AGGREGATELESS_EVENT_SOURCING_DEEP_DIVE.md` | **RESOLVED** | *"Aggregate identity retained; fold/decider ideas absorbed into ADR-0001"* |
| `2026-05-01_HYBRID_ARCHITECTURE_BEST_OF_BOTH_WORLDS.md` | **SUPERSEDED** | Proposed additive context queries; never shipped |
| `2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md` | Reference | Called it "the most radical architectural innovation" but rated it for future consideration |

### What Was Absorbed

Two ideas from aggregateless ES made it into the library:

1. **Pure-function fold** — the `Decider.Apply` pattern. Instead of a mutable aggregate object with methods, fold events through a pure function. This is straight from Fritzsche's "functional core."

2. **Feature slices** — each decider is self-contained with its own minimal state, not a god aggregate. The `deriver.Deriver` extends this for event-to-command derivation.

### What Was Rejected

| Aggregateless Idea | Why Rejected |
|---|---|
| Single flat events table (no streams) | Lose aggregate identity, versioning, snapshots |
| No `aggregate_type` / `aggregate_id` columns | Lose stream-based load (`Load(ctx, ref)`) |
| Raw JSONB without typed events | Lose compile-time safety (Go's typed payloads) |
| No snapshotting | *"Has no concept of snapshots because it has no concept of aggregates"* |
| No push-based subscriptions | Library already has `event.Bus` + projections |
| No schema enforcement | Library has `Codec` + Go types + `schema.Upcaster` |

### The Deep Dive's Bottom Line

> *"Aggregateless ES is a thought-provoking critique of aggregate-based design. Its core insight — that consistency boundaries can be defined by queries rather than aggregate identity — is valuable even if you keep aggregates."*
>
> — `docs/research/archive/2026-05-01_AGGREGATELESS_EVENT_SOURCING_DEEP_DIVE.md:1129`

And from the innovations survey:

> *"The aggregate is no longer sacred."*
>
> — `docs/research/archive/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md:501`

---

## 9. The Honest Assessment

### The Library Is Aggregateless-Adjacent

ADR-0001 did 80% of what aggregateless ES advocates:

- Killed the OO aggregate (mutable state, 9-method interface, entity hierarchy)
- Replaced it with a pure function — exactly what Fritzsche advocates
- State is never stored, only computed from events

The only thing retained from traditional DDD is the **stream partition key** (`AggregateRef`) for operational reasons: stream-based loading, optimistic concurrency versioning, snapshot keys.

### The Middle Position

```
Traditional DDD:     Aggregate = Identity + State + Behavior + Boundaries
Aggregateless ES:    No Aggregate — just events + dynamic context queries
This library:        AggregateRef = Identity only (stream partition key)
                     Decider[State] = Behavior (pure function, no stored state)
                     Event Stream = State (folded on demand)
```

The aggregateless folks would say: *"You're 80% of the way here. The only thing left is the stream partition key, and you're keeping that for operational reasons."*

They're right. The resolution was pragmatic: keep the name for industry familiarity, adopt the pure-function ideas, document the compromise.

### The Open Question

The word "Aggregate" persists not because the design needs aggregates — it doesn't. ADR-0001 proved that. It persists because the **infrastructure** (stream partitioning, optimistic concurrency, snapshot keys) needs a grouping identifier, and "Aggregate" is the industry-standard name for that grouping.

Whether that's the right long-term call is a legitimate open question — one the repo has flagged in its research documents but not closed. The aggregateless alternative (dynamic context queries with CTE-based atomic appends) was evaluated and rejected for concrete operational reasons. Those reasons are sound for a Go library that prioritizes type safety and operational maturity. But the conceptual critique stands, and the name remains the most debatable decision in the entire architecture.

---

## References

- [ADR-0001: Decider Over Aggregate](../adr/0001-decider-over-aggregate.md)
- [DOMAIN_LANGUAGE.md](../DOMAIN_LANGUAGE.md) — Core Concepts, Anti-Patterns
- [Aggregateless ES Deep Dive](../research/archive/2026-05-01_AGGREGATELESS_EVENT_SOURCING_DEEP_DIVE.md)
- [Hybrid Architecture Proposal](../research/archive/2026-05-01_HYBRID_ARCHITECTURE_BEST_OF_BOTH_WORLDS.md)
- [CQRS Event Sourcing Innovations](../research/archive/2026-05-01_CQRS_EVENT_SOURCING_INNOVATIONS.md)
- [Aggregate ID Design Review](../planning/archive/2026-05-25_AGGREGATE_ID_DESIGN_REVIEW.md)
- `id/aggregate_type.go` — `AggregateRef`, `AggregateType`, `StreamKey()`
- `id/aggregate_id.go` — `AggregateID`, `AggregateMarker`, constructors
- `decider/decider.go` — `Decider[State]`, `Repository[State]`
- `decider/example_test.go` — Working example of `Decider[UserState]`
- `projection/projection.go` — `Projection` interface
- Rico Fritzsche: [Aggregateless Event Sourcing](https://ricofritzsche.me/aggregateless-event-sourcing/)
- Sara Pellegrini: "Killing the Aggregate" (2023)
- Ralf Westphal: [Command Context Consistency](https://ralfwestphal.substack.com/p/command-context-consistency)
