# Innovative CQRS/Event Sourcing Implementations on GitHub

> Research conducted 2026-05-01. Ranked by architectural innovation — not popularity.
> The projects that break from conventional CQRS/ES patterns in fundamental ways.

---

## Table of Contents

- [Tier 1: Genuinely Novel Architectures](#tier-1-genuinely-novel-architectures)
  - [1. Sekiban (.NET) — Dynamic Consistency Boundary](#1-sekiban-net--dynamic-consistency-boundary)
  - [2. Eventide (Ruby) — Autonomous Services on PostgreSQL](#2-eventide-ruby--autonomous-services-on-postgresql)
  - [3. Equinox (F#) — Tip with Unfolds](#3-equinox-f--tip-with-unfolds)
  - [4. Reveno (Java) — Mechanical Sympathy ES](#4-reveno-java--mechanical-sympathy-es)
- [Tier 2: Clever Compositional Innovations](#tier-2-clever-compositional-innovations)
  - [5. canon (Rust) — Proc-Macro Zero-Boilerplate ES](#5-canon-rust--proc-macro-zero-boilerplate-es)
  - [6. Eventuous (.NET) — Gateway as Architecture Seam](#6-eventuous-net--gateway-as-architecture-seam)
  - [7. Commanded (Elixir) — BEAM VM as CQRS Runtime](#7-commanded-elixir--beam-vm-as-cqrs-runtime)
- [Tier 3: Strong Type System Leverage](#tier-3-strong-type-system-leverage)
  - [8. eventually-rs (Rust) — Compile-Time Aggregate Safety](#8-eventually-rs-rust--compile-time-aggregate-safety)
  - [9. Haskell CQRS — Streams as First-Class](#9-haskell-cqrs--streams-as-first-class)
  - [10. zio-event-sourcing (Scala) — Effect-System ES](#10-zio-event-sourcing-scala--effect-system-es)
- [Tier 4: Notable Production Systems](#tier-4-notable-production-systems)
  - [11. Marten (.NET) — PostgreSQL as Universal Store](#11-marten-net--postgresql-as-universal-store)
  - [12. Axon Framework (Java) — The Industry Standard](#12-axon-framework-java--the-industry-standard)
  - [13. Sharpino (F#) — GDPR-First ES](#13-sharpino-f--gdpr-first-es)
- [Innovation Matrix](#innovation-matrix)
- [Cross-Cutting Innovation Themes](#cross-cutting-innovation-themes)
- [Top 3 Recommendations for go-cqrs-lite](#top-3-recommendations-for-go-cqrs-lite)

---

## Tier 1: Genuinely Novel Architectures

These projects break from conventional CQRS/ES patterns in fundamental ways.

---

### 1. Sekiban (.NET) — Dynamic Consistency Boundary

| Field | Value |
|---|---|
| **Repository** | `github.com/J-Tech-Japan/Sekiban` |
| **Stars** | 331 |
| **Language** | C# / .NET |
| **Backends** | Azure Cosmos DB, PostgreSQL, AWS DynamoDB |
| **Integration** | Microsoft Orleans (actor-based clustering) |

#### The Innovation

Replaces the entire aggregate-per-stream + saga pattern with a **single global event stream** and **tag-based dynamic consistency boundaries**.

Instead of static per-aggregate transactional boundaries requiring sagas for cross-entity coordination, Sekiban reserves a *dynamic set of tags* per command and uses optimistic concurrency on those tags.

```
Traditional:  Aggregate A → Saga → Aggregate B → Saga → Aggregate C
Sekiban DCB:  Command reserves {TagA, TagB, TagC} → single event → done
```

#### How DCB Works

- **Single Global Event Stream**: Events are timestamp-ordered via `SortableUniqueId`
- **Tags**: Implement `ITag` and announce participation in consistency via `IsConsistencyTag()`
- **Optimistic Concurrency**: Tag reservations include the last observed `SortableUniqueId`
- **Conflict Resolution**: If a tag's `SortableUniqueId` doesn't match expectations, the operation fails fast

```csharp
public record StudentTag(Guid StudentId) : IGuidTagGroup<StudentTag>
{
    public bool IsConsistencyTag() => true;
    public static string TagGroupName => "Student";
}

// Multi-tag command — atomic across entities
EventOrNone.EventWithTags(
    new StudentEnrolledInClassRoom(command.StudentId, command.ClassRoomId),
    studentTag,
    classRoomTag);
```

#### Why It Eliminates Sagas

1. Cross-entity operations maintain **immediate consistency** through tag reservations
2. Each business fact is **atomic** — no compensating events needed
3. Tag reservations prevent conflicts without complex choreography
4. Commands span multiple tags without intermediate queues or eventual reconciliation
5. Projectors are pure static methods that rebuild state deterministically

#### Orleans Integration

Tags map to grains — one-to-one mapping provides isolation and lifecycle management:

- `TagConsistentGrain` manages reservations
- `TagStateGrain` maintains cached projections
- `MultiProjectionGrain` processes event streams and serves queries
- Hot tags get isolated resources, cold tags cost zero

#### Traditional vs DCB Comparison

| Aspect | Aggregate-Based ES | Dynamic Consistency Boundary |
|---|---|---|
| **Streams** | Per aggregate | Single global stream |
| **Consistency** | Static per aggregate | Per-command dynamic tag set |
| **Cross-entity transactions** | Sagas / eventual consistency | Immediate consistency within reserved tags |
| **Concurrency control** | Aggregate version | Multi-tag optimistic concurrency |
| **Event shape** | Multiple domain-specific events | One business fact per command |

#### Why It Matters

This is the most architecturally radical idea in the list. The DCB concept fundamentally changes how you think about consistency boundaries. Instead of designing aggregates around transactional needs and then building sagas to coordinate across them, you express the consistency you need per operation and the framework enforces it.

---

### 2. Eventide (Ruby) — Autonomous Services on PostgreSQL

| Field | Value |
|---|---|
| **Repository** | `github.com/eventide-project` |
| **Stars** | 3 (understated; production since 2015) |
| **Language** | Ruby |
| **Storage** | PostgreSQL (MessageDB) |
| **Messaging** | PostgreSQL (same instance) |

#### The Innovation

Rejects the entire "smart pipes + separate event store" paradigm. Uses a **single PostgreSQL `messages` table** as both event store and message transport. No separate message broker needed. No ORM. No query services between bounded contexts.

#### MessageDB Schema

```sql
CREATE TABLE message_store.messages (
  id               UUID NOT NULL DEFAULT message_store.gen_random_uuid(),
  stream_name      text NOT NULL,
  type             text NOT NULL,
  position         bigint NOT NULL,
  global_position  bigserial NOT NULL,
  data             jsonb,
  metadata         jsonb,
  time             TIMESTAMP WITHOUT TIME ZONE DEFAULT (now() AT TIME ZONE 'utc') NOT NULL
);
```

One table. Seven columns. Both commands and events live here.

#### Key Radical Decisions

- **No queries between services** — services never query other services, ever. Each owns its data.
- **Behavioral objects, not data objects** — "Tell, Don't Ask" as the fundamental design principle. Objects expose executable interfaces, not data attributes.
- **Category-based routing** — streams organized by category prefix (e.g., all `account-` streams), enabling pub/sub via `correlationStreamName` metadata in PostgreSQL functions.
- **Actor model coordination** — components run as isolated actors with graceful shutdown, no shared state. Each component has its own message store session.
- **Dumb pipes, smart endpoints** — elemental messaging transports rather than "smart pipes" like message brokers or ESBs.
- **No ORM** — objects are not designed around persistence concerns.

#### Pub/Sub via PostgreSQL

- `get_stream_messages()` retrieves messages from a specific stream
- `get_category_messages()` retrieves messages from multiple streams sharing a common prefix
- `correlationStreamName` metadata enables request/response patterns
- Consumer groups via consistent hashing of stream names for horizontal scaling
- Optional SQL filtering for advanced message selection

#### Design Values

- **Useful Objects**: Objects must be useful immediately upon instantiation
- **Substitutes**: Dependencies initialized to safe, inert implementations
- **Telemetry**: Built-in observation capabilities over test doubles
- **Composition**: Independent, standalone units that can be recomposed
- **Immutability**: Events and commands are immutable once written

#### Why It Matters

Achieves CQRS + Event Sourcing + reliable messaging with **zero external infrastructure beyond PostgreSQL**. The "architecture, not infrastructure" philosophy produces systems where services can go offline without cascading failures. The single-table design eliminates the operational complexity of running separate event stores and message brokers.

---

### 3. Equinox (F#) — Tip with Unfolds

| Field | Value |
|---|---|
| **Repository** | `github.com/jet/equinox` |
| **Stars** | N/A (Jet.com internal, open-sourced) |
| **Language** | F# |
| **Backends** | EventStoreDB, CosmosDB, DynamoDB, PostgreSQL (MessageDB), SqlStreamStore |
| **Production** | jet.com since 2017 |

#### The Innovation

Introduces the **"Tip" document pattern** for document databases (CosmosDB, DynamoDB). Instead of reading entire event streams to reconstruct state, each stream maintains a single "Tip" document containing compressed snapshots ("unfolds") using deflate+base64.

```
Traditional:  Read 1000 events → replay → current state
Equinox Tip:  Read 1 document → current state (unfolds are compressed snapshots)
```

#### How the Tip Works

Each stream has a single Tip document that:

- Maintains current stream position/sequence number
- Tracks version (0-based: version = number of events)
- Contains `.u` (unfolds) array field — compressed snapshot data
- Buffers events before batching writes
- Is updated atomically with each write — no separate snapshotting process

#### Codec/Versioning

Uses **FsCodec** for pluggable event serialization:

- `NewtonsoftJson.Codec`: Production-ready, convention-based versioning via TypeShape's `UnionContractEncoder`
- `Box.Codec`: Lightweight non-serializing substitute for testing
- `SystemTextJson.Codec`: .NET System.Text.Json alternative
- Schema evolution with minimal boilerplate
- Multiple co-existing compaction schemas per stream

#### Store Abstractions

| Store | Strategy | Key Optimization |
|---|---|---|
| **CosmosDB** | Tip with Unfolds | Single-document reads, RU cost reduction |
| **DynamoDB** | Tip with Unfolds | Patterned after CosmosStore |
| **EventStoreDB** | Native event streams | gRPC interface, append-only |
| **MessageDB (Postgres)** | SQL event storage | Actively maintained Postgres event store |
| **SqlStreamStore** | SQL-backed | PostgreSQL, MySQL, MSSQL adapters |
| **MemoryStore** | In-memory | ~100 LOC, unit/integration testing |

#### Why It Matters

On CosmosDB (where RU costs dominate), this eliminates the **read amplification** that makes traditional event sourcing expensive. A state reconstruction that would require 50 RU (reading 100 events) becomes a 1 RU single-document read. Yet you retain full event replay capability — the Tip is an optimization layer, not a replacement.

The store-agnostic programming model means the same F# domain logic works identically across all backends, with each backend optimizing for its strengths.

---

### 4. Reveno (Java) — Mechanical Sympathy ES

| Field | Value |
|---|---|
| **Repository** | `github.com/dmart28/reveno` |
| **Stars** | 299 |
| **Language** | Java |
| **Size** | ~300KB |
| **Benchmark** | 1,183,396 TPS, 68μs mean latency (MacBook Pro 2.7GHz i5) |

#### The Innovation

The only CQRS/ES framework designed with **mechanical sympathy** — deep hardware-awareness in every layer. Treats CPU cache behavior, memory access patterns, and GC characteristics as first-class design constraints.

#### Mechanical Sympathy Techniques

- **Off-heap storage**: Data stored directly in off-heap memory, avoiding JVM heap and GC overhead
- **Single-writer principle**: Only one thread modifies specific data structures, eliminating lock contention
- **Sequential memory access**: Predictable access patterns optimize CPU cache utilization
- **False sharing avoidance**: Minimizes cache line contention between CPU cores
- **Natural batching**: Processes operations in batches under high load, amortizing per-operation overhead
- **Preallocated volumes**: Files preallocated as journal containers for consistent low-latency access

#### Zero-Copy Architecture

- Append-only journaling eliminates copy operations during persistence
- Direct memory access via buffers rather than intermediate objects
- Off-heap storage avoids expensive JVM heap operations
- No data copying between memory locations during processing

#### Processing Model

```
Commands → Transaction Actions (state mutators) → Event Dispatch (async) → View Mapping (auto)
```

1. **Commands**: Entry points for business logic and validation
2. **Transaction Actions**: State mutators modifying the in-memory domain model
3. **Event Dispatch**: Asynchronous publication after successful transaction
4. **View Mapping**: Automatic mapping from transactional model to query model

#### CQRS Implementation

- Complete separation between command (write) and query (read) sides
- Independent object models for each side
- Automatic view mapping from transactional to query model
- In-memory query model with automatic updates
- Happens-before guarantees ensure query model consistency

#### Why It Matters

Proves that JVM-based event sourcing can achieve **HFT-tier performance** by treating hardware characteristics as first-class design constraints. The "source of truth in RAM, durability on disk" architecture is something most frameworks don't dare attempt. The 300KB footprint and million+ TPS demonstrate that CQRS/ES doesn't have to be slow.

---

## Tier 2: Clever Compositional Innovations

---

### 5. canon (Rust) — Proc-Macro Zero-Boilerplate ES

| Field | Value |
|---|---|
| **Repository** | `github.com/rjh-mopjones/canon` |
| **Stars** | 0 (new, experimental) |
| **Language** | Rust |
| **Storage** | YugabyteDB (ACID outbox) |
| **Messaging** | Kafka (pluggable) |

#### The Innovation

Uses Rust **proc macros** (`#[aggregate]`, `#[command]`, `#[event]`, `#[command_handler]`) to generate *all* trait implementations and dispatch logic. An entire aggregate with commands, events, handlers, and snapshotting becomes declarative attribute annotations.

#### Declarative Aggregate Definition

```rust
#[aggregate(snapshot_every = 50)]
pub struct Ship {
    status: ShipStatus,
    fuel_level: f32,
}

#[command(Ship, version = 1, produces = [ShipDeparted])]
pub struct DepartForStation { pub destination: StationId }

#[event(Ship, version = 1)]
pub struct ShipDeparted { pub destination: StationId }

#[event_combiner(Ship, version = 1)]
impl ShipDeparted {
    fn combine(&self, state: &mut Ship) {
        state.status = ShipStatus::InFlight;
    }
}

#[command_handler(Ship, version = 1)]
impl DepartForStationHandler {
    type Error = FleetError;
    fn handle(&self, state: &Ship, cmd: DepartForStation) -> Result<ShipDeparted, FleetError> {
        if state.status != ShipStatus::Docked { return Err(FleetError::NotDocked); }
        Ok(ShipDeparted { destination: cmd.destination })
    }
}
```

Auto-discovery: `ServiceBuilder::new().for_aggregate::<Ship>().build()` finds all components through an inventory system.

#### Four-Stage Pipeline

```
Inbox → Command Store → Event Store → Outbound Queue
```

Each stage provides durability guarantees. The outbox pattern with YugabyteDB ACID transactions solves the dual-write problem completely — events are staged within the same transaction as business logic.

#### Key Features

- **Zero boilerplate**: Proc macros generate all trait implementations
- **Pluggable architecture**: Every component sits behind traits — swap Cassandra for DynamoDB, Kafka for Pulsar
- **Counterfactual replay**: Replay historical events in isolation for "what if" scenarios
- **Oversight-gated handlers**: Runtime validation, approval, monitoring, and rate limiting
- **In-memory test harness**: Integration tests in milliseconds with zero external infrastructure
- **Dead letter handling**: Built-in failed message management
- **Snapshotting**: Configurable per-aggregate snapshot frequency
- **Projections with rebuild**: Rebuildable materialized views

#### Why It Matters

"Vibe-coded" (AI-assisted from first principles) but architecturally sound. The proc-macro approach eliminates the boilerplate that makes Rust event sourcing verbose. The four-stage pipeline with guaranteed delivery via ACID outbox is a production-grade pattern that most frameworks leave as an exercise for the user.

---

### 6. Eventuous (.NET) — Gateway as Architecture Seam

| Field | Value |
|---|---|
| **Repository** | `github.com/Eventuous/eventuous` |
| **Stars** | 503 |
| **Language** | C# / .NET |
| **Event Stores** | KurrentDB, PostgreSQL, SQL Server, SQLite |
| **Messaging** | RabbitMQ, Google PubSub, Kafka |

#### The Innovation

The **Gateway component** cleanly separates Event Sourcing from Event-Driven Architecture. Instead of piping domain events directly to message brokers, the Gateway provides an explicit transformation layer.

#### Gateway Architecture

```
EventStore → Subscription → Transform/Filter → Producer (Kafka/RabbitMQ/PubSub)
```

**Why this matters:**

- Avoids using message brokers for direct domain event propagation
- Prevents out-of-order event processing issues
- Eliminates two-phase commit complexities
- Enables safe event replay without affecting external consumers
- Transforms domain events into integration events with different schemas

#### Dual Paradigm: Aggregates AND Functional

**Traditional Aggregate Pattern:**

- `Aggregate<TState>` where TState is immutable
- State changes via `Apply` producing new state instances
- Built-in optimistic concurrency control

**Functional Command Services:**

```csharp
// No aggregate objects needed — pure functions
// Receive: current state + historical events + command → new events
```

Eliminates aggregate objects entirely for scenarios where traditional boundaries feel artificial. Both approaches maintain the same transactional guarantees.

#### Philosophy: "Just Enough"

- Core understandable in 30 minutes
- No interfaces for commands/events (prefers records)
- Explicit ES/EDA boundary via Gateway prevents architectural coupling
- Built-in OpenTelemetry tracing and metrics
- Immutable state by default

#### Comparison with Other .NET Frameworks

| Aspect | Eventuous | EventFlow | Marten |
|---|---|---|---|
| **Philosophy** | "Just enough" | Full framework | Document DB + ES |
| **Abstractions** | Minimal | Extensive interfaces | Moderate |
| **Primary focus** | Event store first | Full CQRS stack | PostgreSQL document store |
| **ES/EDA separation** | Explicit Gateway | Blended | Blended |
| **Aggregate approach** | Dual (object + functional) | Traditional only | Traditional only |

#### Why It Matters

The Gateway is an architecture-level insight that most CQRS frameworks miss. By making the boundary between Event Sourcing and Event-Driven Architecture explicit, it prevents the coupling that makes event-sourced systems fragile when replaying events or changing messaging infrastructure.

---

### 7. Commanded (Elixir) — BEAM VM as CQRS Runtime

| Field | Value |
|---|---|
| **Repository** | `github.com/commanded/commanded` |
| **Stars** | 2k |
| **Language** | Elixir |
| **Event Stores** | PostgreSQL (EventStore), EventStoreDB, in-memory |
| **Runtime** | BEAM VM / OTP |

#### The Innovation

Leverages the BEAM VM's actor model as the **natural CQRS runtime**. Aggregates are GenServers with OTP supervision. Process managers coordinate multi-aggregate workflows with automatic fault tolerance.

#### Aggregate GenServer Approach

```elixir
defmodule BankAccount do
  defstruct [:account_number, :balance]

  # Command handling — enforce business rules
  def execute(%BankAccount{account_number: nil}, %OpenBankAccount{} = command) do
    %BankAccountOpened{account_number: command.account_number, initial_balance: command.initial_balance}
  end

  # State mutation — only through events
  def apply(%BankAccount{} = account, %BankAccountOpened{} = event) do
    %BankAccount{account | account_number: event.account_number, balance: event.initial_balance}
  end
end
```

- State is **only mutated by applying domain events** through `apply/2` functions
- No external GenServer dependency — Commanded manages the process lifecycle
- Aggregates spawned on first command, state rebuilt from events on restart

#### OTP Supervision

```elixir
defmodule Bank.Supervisor do
  use Supervisor

  def init(_arg) do
    children = [
      BankApp,                    # Application
      AccountBalanceHandler,      # Event handler
      TransferMoneyProcessManager,# Process manager
      AccountsProjector,          # Read model projector
    ]
    Supervisor.init(children, strategy: :one_for_one)
  end
end
```

Supervision tree provides self-healing: if an aggregate process crashes, it's restarted and state is rebuilt from events.

#### Process Managers

Process managers are the "opposite" of aggregates — they handle events and dispatch commands to coordinate multiple aggregates:

```elixir
defmodule TransferMoneyProcessManager do
  use Commanded.ProcessManagers.ProcessManager

  # Event routing
  def interested?(%MoneyTransferRequested{transfer_uuid: transfer_uuid}),
    do: {:start, transfer_uuid}

  # Command dispatch in response to events
  def handle(%TransferMoneyProcessManager{}, %MoneyTransferRequested{} = event) do
    %WithdrawMoney{account_number: event.debit_account, amount: event.amount}
  end
end
```

Callbacks: `interested?/1` (routing), `handle/2` (command dispatch), `apply/2` (state update), `error/3` (error handling).

#### Snapshotting

```elixir
config :my_app, MyApp.Application,
  snapshotting: %{
    MyApp.ExampleAggregate => [
      snapshot_every: 10,
      snapshot_version: 1  # Increment when aggregate structure changes
    ]
  }
```

#### Serialization

- Default JSON via Jason library
- Custom serializers supported (MessagePack, etc.)
- `@derive Jason.Encoder` for all event structs
- Custom `JsonDecoder` protocol for complex data types

#### Why It Matters

Most CQRS frameworks bolt supervision on top. Commanded gets it for free from OTP. The aggregate lifecycle (spawn on first command → persist via events → rebuild on restart → snapshot for performance) maps 1:1 to GenServer patterns. The BEAM's "let it crash" philosophy means aggregate consistency failures trigger automatic recovery rather than cascading failures — fundamentally different from try/catch-based error handling in JVM/.NET frameworks.

---

## Tier 3: Strong Type System Leverage

---

### 8. eventually-rs (Rust) — Compile-Time Aggregate Safety

| Field | Value |
|---|---|
| **Repository** | `github.com/get-eventually/eventually-rs` |
| **Stars** | 595 |
| **Language** | Rust |
| **Backends** | PostgreSQL, in-memory |

#### The Innovation

Uses Rust's associated types in the `Aggregate` trait to make mismatched event/command/state combinations a **compile error**.

#### Type-Safe Aggregate Trait

```rust
pub trait Aggregate {
    type Id: Eq;
    type State: Default;
    type Event;
    type Command;
    type Error;

    fn apply(state: Self::State, event: Self::Event) -> Result<Self::State, Self::Error>;
    fn handle<'a, 's>(
        &'a self,
        id: &'s Self::Id,
        state: &'s Self::State,
        command: Self::Command
    ) -> Pin<Box<dyn Future<Output = Result<Option<Vec<Self::Event>>, Self::Error>> + Send + 'a>>;
}
```

#### Compile-Time Guarantees

- **Exhaustive pattern matching**: Add a new event variant → compiler forces you to handle it everywhere
- **Associated types prevent mismatched combinations**: Command A can't produce Event B if the types don't align
- **`Send + Sync` bounds**: Thread-safe event stores guaranteed at compile time
- **Borrow checker**: Prevents mutable aliasing of aggregate state
- **`Result`-based error propagation**: Missing error handling is a compile error

#### Event Store Implementations

| Backend | Thread Safety | Optimistic Concurrency |
|---|---|---|
| **In-Memory** | `HashMap` with `std::sync` | Yes |
| **PostgreSQL** | Production-ready | Yes, with versioning |

#### Projection System

- State folding: Projection state built by applying events sequentially
- Materialized views: Read-optimized models separate from domain aggregates
- Checkpointing: Track progress for reliable restarts
- Asynchronous processing: Projections run independently

#### Subscription Support

```rust
pub trait Subscription<StreamId, Event> {
    type Error;
    type Checkpoint;

    fn resume(&self, checkpoint: &Self::Checkpoint)
        -> SubscriptionStream<'_, Self::Error, StreamId, Event>;
    fn checkpoint(&self) -> Self::Checkpoint;
}
```

Persistent and transient subscriptions with error recovery.

#### Why It Matters

While other frameworks catch event handling errors at runtime, eventually-rs catches them at compile time. The type system makes illegal states unrepresentable — you literally cannot compile code that forgets to handle an event variant or mismatches command/event types.

---

### 9. Haskell CQRS — Streams as First-Class

| Field | Value |
|---|---|
| **Repository** | `github.com/BardurArntsson/cqrs` |
| **Stars** | 48 |
| **Language** | Haskell |
| **Backends** | PostgreSQL, in-memory |

#### The Innovation

Treats event streams as **first-class streaming abstractions** (via `UnliftIO.Streams`) rather than collections loaded from a database. Events are consumed lazily via continuations, never loaded into memory as a list.

#### Streaming Event Streams

```haskell
data EventStream i e = EventStream {
    esReadEventStream :: forall a m . (MonadUnliftIO m) =>
        StreamPosition -> (InputStream (StreamPosition, PersistedEvent' i e) -> m a) -> m a
  , esReadAggregateEvents :: forall a m . (MonadUnliftIO m) =>
        i -> Int32 -> (InputStream (PersistedEvent e) -> m a) -> m a
}
```

Key characteristics:

- **Continuation-based**: Consumers receive an `InputStream` and produce a result
- **Lazy evaluation**: Events processed as streams, not loaded into memory
- **Backpressure handling**: Built-in support for streaming backpressure
- **Parametric polymorphism**: Type-safe stream transformations

#### Type-Safe Stream Transformations

```haskell
transform :: forall e e' i i' . Iso i' i -> Iso e' e -> EventStream i e -> EventStream i' e'
```

Isomorphism-based transformations provide type-safe stream manipulation between different event and identifier types.

#### Monadic Command Handling

- `CommandT` monad transformer for composable command processing
- `UnitOfWorkT` for managing transactional context and side effects
- Pure aggregate actions: `type AggregateAction a e = (Maybe a -> e -> a)`

#### Optimistic Concurrency

```haskell
data PersistedEvent e = PersistedEvent {
    peEvent            :: !e
  , peSequenceNumber   :: !Int32
  , peTimestampMillis  :: !Int64
}
```

Version numbers embedded in the type system for conflict detection.

#### Why It Matters

Most ES frameworks treat event loading as `SELECT * FROM events` → list. This library treats it as what it actually is: an **unbounded stream**. The type system enforces correct consumption patterns. The streaming-first design means memory usage is constant regardless of event stream length.

---

### 10. zio-event-sourcing (Scala) — Effect-System ES

| Field | Value |
|---|---|
| **Repository** | `github.com/holinov/zio-event-sourcing` |
| **Stars** | 39 |
| **Language** | Scala (ZIO) |
| **Backends** | Cassandra/Scylla, RocksDB, file-based, in-memory |

#### The Innovation

All state transformations return `Task[S]` (ZIO effects), making event application **referentially transparent**. The ZIO effect system provides compositional error handling, retry, and resource management.

#### Architecture

```scala
case class AggregateBehaviour[E, S](initialState: S, aggregations: (S, E) => Task[S])

class Aggregate[-E, +S] private[es] (
  key: String,
  aggState: Ref[S],              // ZIO concurrent reference
  aggregations: (S, E) => Task[S],
  persist: (String, E) => Task[Unit]
)
```

#### Effect System Benefits

- **Purely functional operations**: All state transformations use `Task[S]` effects
- **Concurrent state management**: `Ref[S]` for thread-safe state
- **Streaming**: ZIO Streams for event loading with backpressure
- **Compositional error handling**: ZIO's error propagation
- **Resource management**: Automatic cleanup through ZIO ecosystem
- **Effectful aggregation**: Unlike imperative frameworks, every state transformation is effectful and pure

#### Storage Backends

- **In-memory**: Testing and development
- **File-based**: Simple persistence
- **Cassandra/Scylla**: Distributed production storage
- **RocksDB**: Embedded high-performance storage

#### Why It Matters

Proves that effect systems can model event sourcing more safely than imperative approaches. State transformations are effectful but pure — same inputs always produce same outputs, with controlled side effects. The ZIO runtime handles concurrency, error recovery, and resource lifecycle that other frameworks implement manually.

---

## Tier 4: Notable Production Systems

---

### 11. Marten (.NET) — PostgreSQL as Universal Store

| Field | Value |
|---|---|
| **Repository** | `github.com/JasperFx/marten` |
| **Stars** | 3.4k |
| **Language** | C# / .NET |
| **Storage** | PostgreSQL (dual-mode: document + event store) |

#### The Innovation

PostgreSQL as both **document store AND event store simultaneously**. JSONB for documents, append-only event tables for sourcing. No separate database systems needed.

#### Dual-Mode Architecture

| Mode | Storage | Use Case |
|---|---|---|
| **Document Store** | JSONB documents | CRUD-style reads, flexible schema |
| **Event Store** | Append-only event streams | Event sourcing, audit trails |

Both modes share the same PostgreSQL database and ACID transactions.

#### Three Projection Lifecycles

| Lifecycle | Consistency | Execution | Use Case |
|---|---|---|---|
| **Inline** | Strong (same transaction) | Synchronous | Write models needing immediate consistency |
| **Async** | Eventual | Background daemon | Read models for UIs and reporting |
| **Live** | Real-time | On-demand computation | Aggregation without storage overhead |

```csharp
// Live projection — computed on-demand, no storage
await session.Events.AggregateStreamAsync<QuestParty>(questId);
```

#### Additional Features

- **LINQ support**: Full `IQueryable<T>` with PostgreSQL SQL translation
- **Auto-schema migration**: Multiple modes (CreateOnly, CreateOrUpdate, All)
- **Multi-tenancy**: Database-level, schema-level, and sharded-table tenancy
- **Hilo ID generation**: Efficient identity generation without round-trips
- **Schema export**: SQL migration scripts via `dotnet run -- db-patch`

#### Why It Matters

Makes event sourcing accessible to teams already on PostgreSQL by eliminating the "you need a specialized event store" barrier. Document store mode provides a graduation path from CRUD to full ES. The three projection lifecycles let you tune consistency/performance per read model without changing architecture.

---

### 12. Axon Framework (Java) — The Industry Standard

| Field | Value |
|---|---|
| **Repository** | `github.com/AxonIQ/AxonFramework` |
| **Stars** | 3.6k |
| **Language** | Java |
| **Event Stores** | Axon Server, JDBC, JPA, in-memory |
| **Messaging** | Axon Server, Kafka, RabbitMQ, Spring AMQP |

#### Architecture

Not the most architecturally innovative, but the most **complete production ecosystem** for JVM CQRS/ES.

#### Key Components

- **CommandBus**: Dispatch commands to handlers with interceptor chain
- **EventBus**: Publish and subscribe to events
- **EventStore**: Persistent event storage with snapshotting
- **Repositories**: Aggregate loading with event replay
- **Sagas**: Long-running process orchestration with association values

#### Event Processor Types

| Type | Behavior | Use Case |
|---|---|---|
| **Tracking** | Replays from event store, maintains position | Reliable projection building |
| **Subscribing** | Live subscription to event bus | Real-time processing |
| **PCP (Persistent Stream)** | Persistent stream from Axon Server | High-throughput processing |

#### Event Upcasting

Built-in support for migrating old event schemas without breaking existing events. Upcasters transform old event versions to new versions during replay.

#### Axon Server vs Framework

| Component | Role |
|---|---|
| **Axon Framework** | Application library, annotations, abstractions |
| **Axon Server** | Purpose-built event store + message router with gRPC |

Axon Server provides optimized event storage, distributed command routing, and query dispatch beyond what generic databases offer.

#### Annotation-Driven Development

```java
@Aggregate
public class GiftCard {
    @AggregateIdentifier
    private String id;

    @CommandHandler
    public void handle(IssueCardCommand cmd) { /* ... */ }

    @EventSourcingHandler
    public void on(CardIssuedEvent event) { /* ... */ }

    @MessageHandlerInterceptor
    public void intercept(Message message) { /* ... */ }
}
```

#### Why It Matters

The most battle-tested JVM CQRS/ES framework. Enterprise-grade features: event upcasting, snapshotting, distributed processing, saga management, and comprehensive monitoring. If you need CQRS on the JVM and want something that works at scale, this is it.

---

### 13. Sharpino (F#) — GDPR-First ES

| Field | Value |
|---|---|
| **Repository** | `github.com/tonyx/Sharpino` |
| **Stars** | 40 |
| **Language** | F# |
| **Backends** | PostgreSQL (JSON or binary), in-memory |
| **Messaging** | RabbitMQ |

#### The Innovation

The only ES framework with **first-class GDPR compliance** built into the core abstractions.

#### GDPR Compliance

- Replaces events with voided/empty versions on deletion requests
- Replaces snapshots with voided/empty versions
- Updates cache with empty values to reflect GDPR-compliant state
- Partial update support for redacting specific sensitive fields

#### Soft Delete with History Access

- Marks aggregates as deleted (not permanent removal)
- Creates snapshot with "deleted" field set to `true`
- `HistoryStateViewer` accesses historical states including deleted data
- Predicate-based soft deletion (e.g., only delete when reference counters reach zero)

#### Multi-Stream Transactions

```fsharp
// Execute multiple commands across different aggregates in one DB transaction
forceRunNAggregateCommands
// With compensating events (undoers) for rollback
forceRunTwoNAggregateCommands
forceRunThreeNAggregateCommands
```

Cross-stream invariants in a single database transaction, with compensating events for rollback.

#### Caching Architecture

- **StateViewer**: Probes cache first, then event store, applies evolve function
- **HistoryStateViewer**: Includes softly deleted objects
- **L1**: MemoryCache for latest aggregate state
- **L2**: Distributed cache (Azure SQL)
- **RefreshableAsync**: Interface for details cache improvements

#### F# Functional Approach

- Events and commands as discriminated unions
- Pure `evolve` functions for state computation
- Optimistic locking based on `event_id` position
- FsPickler serialization for binary encoding

#### Why It Matters

GDPR compliance is a legal requirement for any system handling EU citizen data, yet most ES frameworks treat it as an afterthought. Sharpino makes it a first-class concern. The multi-stream transaction support with compensating events is also notable — most ES frameworks leave cross-aggregate transactions as an exercise.

---

## Innovation Matrix

| Project | Key Innovation | Language | Stars | Tier |
|---|---|---|---|---|
| **Sekiban** | Dynamic Consistency Boundaries (no sagas) | C# | 331 | 1 |
| **Eventide** | Autonomous services, single Postgres table | Ruby | 3 | 1 |
| **Equinox** | Tip + Unfolds for document DBs | F# | — | 1 |
| **Reveno** | Mechanical sympathy, 1M+ TPS | Java | 299 | 1 |
| **canon** | Proc-macro zero-boilerplate | Rust | 0 | 2 |
| **Eventuous** | ES/EDA Gateway seam, dual paradigm | C# | 503 | 2 |
| **Commanded** | BEAM/OTP as natural CQRS runtime | Elixir | 2k | 2 |
| **eventually-rs** | Compile-time aggregate safety | Rust | 595 | 3 |
| **Haskell cqrs** | Streams as first-class, continuation-based | Haskell | 48 | 3 |
| **zio-event-sourcing** | ZIO effect system for ES | Scala | 39 | 3 |
| **Marten** | PostgreSQL universal store, 3 projection modes | C# | 3.4k | 4 |
| **Axon** | Most complete production ecosystem | Java | 3.6k | 4 |
| **Sharpino** | GDPR-first, multi-stream transactions | F# | 40 | 4 |

---

## Cross-Cutting Innovation Themes

### 1. Eliminating Sagas (Sekiban)

The DCB pattern replaces the entire saga/choreography/orchestration layer with tag-based optimistic concurrency. This is the single most impactful architectural innovation — sagas are the #1 source of complexity in event-sourced systems.

### 2. Single Infrastructure (Eventide, Marten)

Both Eventide (PostgreSQL only) and Marten (PostgreSQL only) demonstrate that you don't need separate event stores, message brokers, and read databases. One database can serve all three roles when designed thoughtfully.

### 3. Mechanical Sympathy (Reveno)

The only framework that treats hardware as a first-class design constraint. Off-heap storage, sequential access, single-writer, and natural batching achieve 1M+ TPS on a laptop. Most ES frameworks ignore hardware entirely.

### 4. Type System Leverage (eventually-rs, Haskell cqrs, Equinox)

Rust's associated types, Haskell's continuation-based streams, and F#'s discriminated unions with codecs — all push safety guarantees into the type system. "If it compiles, it works" applied to event sourcing.

### 5. Effect Systems (zio-event-sourcing, canon)

ZIO in Scala and Rust's proc macros represent two different approaches to the same goal: eliminating boilerplate while maintaining safety. ZIO does it through effect composition, canon does it through compile-time code generation.

### 6. ES/EDA Separation (Eventuous)

The Gateway pattern explicitly separates "storing events as facts" from "distributing events as messages." Most frameworks conflate these two concerns, leading to replay anxiety and ordering problems.

### 7. Actor Model Integration (Commanded, Sekiban)

Commanded uses OTP actors, Sekiban uses Orleans grains. Both demonstrate that the actor model is the natural runtime for event-sourced aggregates — each aggregate instance is an actor with its own lifecycle, state, and supervision.

### 8. Legal Compliance (Sharpino)

GDPR "right to be forgotten" is fundamentally at odds with event sourcing's "never delete events" principle. Sharpino resolves this tension with voiding, partial updates, and history viewers — the only framework to address this directly.

---

## Top 3 Recommendations for go-cqrs-lite

### 1. Sekiban — Dynamic Consistency Boundaries

The DCB concept is the most architecturally radical idea here. Consider whether `go-cqrs-lite` could support optional tag-based consistency instead of requiring aggregate-per-stream. Even as an experimental mode, this could differentiate the library.

### 2. Eventide — Single PostgreSQL Table Philosophy

The MessageDB schema (one table, seven columns, both commands and events) and the "services never query other services" rule are profound constraints that produce simpler systems. The `storage/` module could adopt this schema pattern.

### 3. Reveno — Mechanical Sympathy

The performance techniques (off-heap, sequential access, single-writer, natural batching) are a masterclass in performance-oriented design. For the planned `storage/` and `watermill/` modules, considering access patterns and batching at the design level could yield significant performance benefits.

---

*Research conducted 2026-05-01 across GitHub, Sourcegraph, project documentation, and architecture blogs.*
