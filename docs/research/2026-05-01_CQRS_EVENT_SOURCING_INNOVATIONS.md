# State of the Art: CQRS & Event Sourcing Innovations (2024–2026)

A research report on the most significant and innovative developments in CQRS and Event Sourced systems.

---

## Table of Contents

1. [Paradigm Shifts](#1-paradigm-shifts)
2. [Aggregateless Event Sourcing](#2-aggregateless-event-sourcing)
3. [AI + Event Sourcing Convergence](#3-ai--event-sourcing-convergence)
4. [Database & Platform Innovations](#4-database--platform-innovations)
5. [Type-Safe Functional Approaches](#5-type-safe-functional-approaches)
6. [The Decider Pattern](#6-the-decider-pattern)
7. [Actor Model + Event Sourcing](#7-actor-model--event-sourcing)
8. [Modern Framework Landscape](#8-modern-framework-landscape)
9. [Event Schema Evolution](#9-event-schema-evolution)
10. [Innovation Matrix](#10-innovation-matrix)

---

## 1. Paradigm Shifts

The CQRS/Event Sourcing landscape is undergoing three fundamental shifts:

### Shift 1: From Objects to Functions

Traditional event sourcing was deeply coupled to OOP aggregates (Greg Young's original .NET implementation). The cutting edge has moved decisively toward **functional core / imperative shell**:

- **Decider pattern** — Pure `decide(state, command) → events` function, no object, no mutation
- **Fold pattern** — Pure `fold(state, event) → state` for reconstruction
- **Eventium** (Haskell) — Entire domain logic is IO-free, testable without mocks
- **Sharpino** (F#) — Type-safe functional event sourcing with pure projections

### Shift 2: From Aggregates to Contexts

The traditional "find the right aggregate boundary" problem is being challenged:

- **Aggregateless event sourcing** (Rico Fritzsche) — No aggregates at all. Events are standalone facts. Consistency defined by query-based contexts.
- **Feature slices** — Organize by business feature, not aggregate. Each slice defines its own context via event queries.
- **CTE-based atomic operations** — PostgreSQL CTEs for check-and-insert atomicity without aggregate locks.

### Shift 3: From Data Stores to AI Foundations

Event sourcing is being repositioned as the **natural data layer for AI systems**:

- **eventsourcing.ai** — Event streams as AI-ready data products with temporal ordering and domain language
- **LLM-Event Store interfaces** — Natural language queries over event streams
- **Continuous learning loops** — Predictions become events, outcomes feed back as training data
- **Reproducible AI pipelines** — Event streams enable point-in-time reconstruction for model training

---

## 2. Aggregateless Event Sourcing

**Source**: [Rico Fritzsche](https://ricofritzsche.me/aggregateless-event-sourcing/)

The most radical architectural innovation in this research. It eliminates the central concept of event sourcing — the aggregate.

### Core Idea

Events are **standalone facts**, not aggregate-bound state transitions. Consistency is defined by **query-based contexts**, not aggregate boundaries.

### Architecture

```
Traditional:     Command → Aggregate.decide() → Events → Aggregate.apply()
Aggregateless:   Command → Load Context (SQL query) → Pure Function → Atomic Insert
```

### Technical Implementation

**Event Store Schema** (single table, no stream concept):

```sql
CREATE TABLE events (
  sequence_number BIGSERIAL PRIMARY KEY,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'
);
```

**Context-Based Consistency** (CTE atomic check-and-insert):

```sql
WITH context AS (
  SELECT MAX(sequence_number) AS max_seq
  FROM events
  WHERE event_type IN ('DeviceRegistered', 'AssetRegistered', 'DeviceBoundToAsset')
    AND (payload->>'deviceId' = $1 OR payload->>'assetId' = $2)
)
INSERT INTO events (event_type, payload, metadata)
SELECT unnest($3::text[]), unnest($4::jsonb[]), unnest($5::jsonb[])
FROM context
WHERE COALESCE(max_seq, 0) = $6
```

**Feature Slice Pattern**:

```
Feature Slice = {
  context query:  defines which events are relevant
  fold function:  reconstructs only needed state
  decide function: pure business logic
  atomic append:  CTE-based optimistic locking
}
```

### Tradeoffs

| Aspect              | Traditional Aggregates   | Aggregateless             |
| ------------------- | ------------------------ | ------------------------- |
| Boundary decisions  | Hard, often wrong        | Eliminated                |
| Cross-aggregate ops | Complex sagas            | Natural query composition |
| Schema simplicity   | Stream per aggregate     | Single events table       |
| Tooling maturity    | Rich ecosystem           | Early stage               |
| Learning curve      | DDD knowledge needed     | SQL + functional thinking |
| Performance         | Aggregate-level batching | Feature-specific queries  |

### When to Consider

- Business domains where aggregate boundaries are genuinely ambiguous
- Systems with many cross-cutting business processes
- Teams comfortable with SQL and functional programming
- Greenfield projects without legacy aggregate commitments

---

## 3. AI + Event Sourcing Convergence

**Source**: [eventsourcing.ai](https://eventsourcing.ai) (the native web GmbH)

### Vision

Event sourcing becomes the **foundation for AI-ready systems**, not just an architectural pattern for auditability.

### Key Innovations

#### 3.1 LLM as Event Store Interface

Instead of building projections first, LLMs query event stores directly:

```
User: "Which genres had the most late returns last quarter?"
LLM:  → Queries LateFeeIncurred events
     → Analyzes patterns
     → "Graphic novels had 40% increase during school holidays"
```

**Why this works**: Events are expressed in business language (`BookBorrowed`, `LateFeeIncurred`). LLMs understand this natively — no translation layer needed.

#### 3.2 Analytical Projections as AI Foundation

```
Raw Events → Analytical Projections → ML Features → Models
              (structured,          (labeled,      (predictions)
               queryable,           temporal,
               rebuildable)         reproducible)
```

#### 3.3 Continuous Learning Feedback Loop

```
Events → Predictions → Actions → Outcomes → (new Events)
                                    ↑                    |
                                    └────────────────────┘
                                     (feedback loop)
```

Predictions become events (`LateReturnPredicted`). Actions become events (`ReminderSent`). Outcomes feed back into training data. Complete chronological record enables "what-if" simulations.

#### 3.4 Practical Benefits

- **Rapid prototyping** — Test analytical hypotheses in minutes via conversation
- **Incident investigation** — Walk through event sequences interactively
- **Non-technical access** — Domain experts explore data without SQL or BI tools
- **Audit compliance** — Every AI decision traceable to source events

### Implications for go-cqrs-lite

The `catalog` module's auto-documentation (AsyncAPI, EventCatalog) aligns perfectly with this vision — structured event metadata is exactly what LLMs need to reason about event streams.

---

## 4. Database & Platform Innovations

### 4.1 KurrentDB (formerly EventStoreDB)

**Latest**: KurrentDB 26.1.0 (April 2026)

**Major innovations since rebrand**:

| Feature                         | Description                                                 | Impact                                                   |
| ------------------------------- | ----------------------------------------------------------- | -------------------------------------------------------- |
| **Archiving**                   | Upload chunk files to S3, transparent reads through archive | 10x storage cost reduction for long-lived streams        |
| **Multi-Stream Atomic Appends** | Atomic writes across multiple streams                       | Eliminates need for sagas/process managers in many cases |
| **Custom Indices**              | Virtual views organized by any property                     | Flexible querying without modifying event log            |
| **Kafka Source Connector**      | Native ingestion from Kafka topics                          | Bridge between event streaming and event sourcing        |
| **Relational Sink**             | Auto-sync to PostgreSQL/SQL Server                          | Built-in CQRS read model synchronization                 |
| **Arrow Flight SQL**            | SQL querying protocol                                       | Direct analytical queries over event data                |
| **Single Package**              | Merged OSS + Enterprise into one binary                     | Simplified operations                                    |
| **DuckDB Integration**          | High-performance analytical processing                      | Up to 10x projection replay speed                        |

**Rebranding timeline**:

- Dec 2024: EventStoreDB 24.10 (last under old name)
- Mar 2025: KurrentDB 25.0 (first under new name)
- Apr 2026: KurrentDB 26.1.0 (current)

### 4.2 EventSourcingDB

**Source**: [eventsourcingdb.io](https://www.eventsourcingdb.io) (the native web GmbH)

A **purpose-built database** for event sourcing, positioned as:

- Event-native from the ground up (not a general-purpose DB with ES bolted on)
- AI-ready with direct LLM integration interfaces
- Focused on event sourcing as foundation for AI systems
- From the creators of `cqrs.com` and the `eventsourcing.ai` initiative

### 4.3 Key Platform Trends

1. **Unified packages** — KurrentDB merged OSS/Enterprise; single binary deployment
2. **Cloud-native connectors** — Built-in Kafka, MongoDB, PostgreSQL, RabbitMQ sinks
3. **Analytical bridges** — Arrow Flight SQL, DuckDB integration
4. **Archive tiering** — Cold storage for historical events without losing queryability
5. **Serverless SDKs** — All major platforms now offer lightweight client libraries

---

## 5. Type-Safe Functional Approaches

### 5.1 Eventium (Haskell)

**Source**: [Eventium Blog Post](https://www.sidorenko.me/blog/2026/04/eventium-event-sourcing-library-for-haskell/)

The most **type-theoretically rigorous** event sourcing framework discovered.

#### Key Innovations

**Pure Domain Logic**: Both `decide` (command handler) and `react` (process manager) are pure functions — no IO, no monads, trivially testable.

**Type-Safe Compensation**: The `IssueCommandWithCompensation` type carries a failure handler as part of its structure:

```haskell
data ProcessManagerEffect command
  = IssueCommand UUID command
  | IssueCommandWithCompensation UUID command (RejectionReason -> [ProcessManagerEffect command])
```

No separate compensation service. The entire decision tree lives in the type.

**Template Haskell for Events**: `constructSumType` generates sum types from event records automatically — exhaustive pattern matching guaranteed.

**TypeEmbedding**: `mkSumTypeEmbedding` lifts aggregate-specific events to application-wide types. Unrecognized events are silently skipped, enabling safe composition.

#### Core Abstractions

| Type                                     | Purpose                                                      |
| ---------------------------------------- | ------------------------------------------------------------ |
| `Projection state event`                 | Fold function with seed — composable state reconstruction    |
| `CommandHandler state event command err` | Pure `decide` + `Projection`                                 |
| `ProcessManager state event command`     | Pure `react` + `Projection` for cross-aggregate coordination |
| `ReadModel m event`                      | First-class queryable views                                  |
| `EventStoreWriter/Reader`                | Polymorphic backend abstractions (STM, SQLite, PostgreSQL)   |

**Implication**: When your type system can express compensation logic, saga correctness becomes a compile-time guarantee, not a runtime hope.

### 5.2 Sharpino (F#)

F# event sourcing with:

- Pure functional domain modeling
- Type-safe event definitions and projections
- Optimized for real-time state updates

**F# Performance Benchmarks** (100,000 students):

| Read Model Strategy            | Time       |
| ------------------------------ | ---------- |
| State rebuilding (cold start)  | 23 seconds |
| Continuously updated in-memory | 652ms      |
| Message-driven (RabbitMQ)      | 28ms       |

**Lesson**: Message-driven projections achieve near-constant-time access — the "project ahead of time" pattern is critical for production.

---

## 6. The Decider Pattern

**Source**: [Jeremie Chassaing](https://thinkbeforecoding.com/), Equinox framework

The Decider is the **core architectural pattern** behind modern functional event sourcing.

### Definition

A Decider is a tuple of three pure functions:

```
Decider = {
  decide:  (Command, State) → Event list     -- command handling
  evolve:  (State, Event) → State            -- state reconstruction
  initialState: State                        -- starting point
}
```

### How It Differs from Traditional Aggregates

| Aspect         | Traditional Aggregate    | Decider                      |
| -------------- | ------------------------ | ---------------------------- |
| Encapsulation  | State + behavior bundled | State and behavior separated |
| Purity         | Side effects in methods  | Pure functions, IO in shell  |
| Testing        | Mock repositories        | Direct function calls        |
| Composition    | Inheritance/composition  | Function composition         |
| Persistence    | Active Record style      | Event stream + fold          |
| State mutation | In-place mutation        | Immutable, event-driven      |

### Usage in Modern Frameworks

- **Equinox** (.NET) — Decider as first-class abstraction
- **Eventium** (Haskell) — `CommandHandler` is a Decider
- **go-cqrs-lite** — `event.Core` + `aggregate.Core` follow this pattern

### Why It Matters

1. **Testability** — No infrastructure needed. Call `decide()` with inputs, assert on outputs.
2. **Composability** — Deciders compose via function composition, not inheritance.
3. **Portability** — Same domain logic works with any storage backend.
4. **Reasoning** — Pure functions are referentially transparent — easy to reason about.

---

## 7. Actor Model + Event Sourcing

### Akka Persistence

Actors persist events for recovery and migration. Each entity has a **single representation** across the cluster (virtual actors), preventing race conditions.

**Key benefits**:

- Database independence — actors operate in-memory with async writes
- Lifecycle management — actors optimized by business-defined lifecycles
- Location transparency — actors move across cluster nodes seamlessly

### Erlang/Elixir Ecosystem

- **Commanded** — Production-grade CQRS/ES for Elixir, leveraging BEAM VM's fault tolerance
- **Gleam's Signal** — Emerging event-sourcing library for the Gleam language on BEAM

### Why Actors + ES Work Together

1. **Single writer per entity** — Actor model guarantees single-threaded execution per actor, eliminating concurrency conflicts
2. **Natural lifecycle mapping** — Actors can be loaded/unloaded based on event stream activity
3. **Fault tolerance** — Actor supervision + event sourcing = self-healing systems
4. **Distribution** — Virtual actors solve the "which node owns this aggregate?" problem

---

## 8. Modern Framework Landscape

### By Language

#### Go

| Framework        | Innovation                                                                 | Maturity         |
| ---------------- | -------------------------------------------------------------------------- | ---------------- |
| **go-cqrs-lite** | Library-first, branded IDs, auto-doc generation (AsyncAPI, EventCatalog)   | Production-ready |
| **goes**         | Streaming-first APIs (channels), multi-backend (MongoDB, PostgreSQL, NATS) | Active           |
| **EventHorizon** | Multi-backend (Redis, AWS, Kafka, MongoDB, NATS, GCP), DDD built-in        | Mature           |
| **Watermill**    | Event-driven applications with multiple message broker support             | Mature           |

#### Rust

| Framework         | Innovation                                   | Maturity |
| ----------------- | -------------------------------------------- | -------- |
| **cqrs-es**       | Serverless-optimized, zero-cost abstractions | Active   |
| **eventually-rs** | DDD-focused, macro-based compile-time safety | Pre-1.0  |

**Rust advantages for event sourcing**:

- Memory safety without GC pauses during high-throughput event processing
- Ownership model prevents data races in concurrent event handling
- Zero-cost abstractions for event serialization
- Enum-based event modeling ensures exhaustive handling
- Serde ecosystem for efficient serialization

#### .NET

| Framework          | Innovation                                                         | Maturity            |
| ------------------ | ------------------------------------------------------------------ | ------------------- |
| **Equinox**        | Store-neutral, CosmosDB-optimized "Tip" mechanism, Decider pattern | Production-hardened |
| **Axon Framework** | Enterprise CQRS/ES, full DDD support                               | Very mature         |

#### Haskell

| Framework    | Innovation                                                         | Maturity   |
| ------------ | ------------------------------------------------------------------ | ---------- |
| **Eventium** | Type-safe compensation, pure domain logic, Template Haskell events | New (2026) |

#### Elixir

| Framework     | Innovation                                      | Maturity   |
| ------------- | ----------------------------------------------- | ---------- |
| **Commanded** | BEAM VM fault tolerance, real-time capabilities | Production |

#### Python

| Framework         | Innovation                                           | Maturity |
| ----------------- | ---------------------------------------------------- | -------- |
| **eventsourcing** | Comprehensive Python library, modern Python features | Mature   |

### Cross-Cutting Tooling

| Tool                          | Purpose                                                  |
| ----------------------------- | -------------------------------------------------------- |
| **Apache EventMesh**          | Serverless event middleware for distributed applications |
| **Redpanda Connect**          | Cloud-native stream processing with ES capabilities      |
| **ksqlDB**                    | Real-time SQL queries over Kafka streams for projections |
| **Confluent Schema Registry** | Schema evolution and compatibility enforcement           |

---

## 9. Event Schema Evolution

### Modern Approaches

#### Upcasting (Still Gold Standard)

Transform old event versions to new format during replay:

```
v1 event → Upcaster → v2 event → v2 handler
```

#### Schema Registry (Confluent)

- **Backward compatibility** — New schema can read old data
- **Forward compatibility** — Old schema can read new data
- **Full compatibility** — Both directions
- **Avro/Protobuf/JSON Schema** — Multiple format support

#### Contract Testing

- **Pact** — Consumer-driven contract testing for event-driven architectures
- **Schema validation at boundaries** — Validate events against schema at event bus entry/exit
- **Compatibility modes** — Automated compatibility checking in CI/CD pipelines

#### Emerging Patterns

1. **Weak Schema** (Aggregateless approach) — JSONB payload with GIN indexes. Schema evolves by adding new fields. Old consumers ignore unknown fields.
2. **Event Interoperability** — Multiple event versions coexist. Consumers declare which versions they understand.
3. **Schema-on-Read** — Apply schema at consumption time, not storage time. Enables multiple interpretations of the same event.

---

## 10. Innovation Matrix

### Impact × Novelty Assessment

| Innovation                       | Impact         | Novelty       | Maturity      | Direction            |
| -------------------------------- | -------------- | ------------- | ------------- | -------------------- |
| Aggregateless ES                 | 🔴 High        | 🔴 Very High  | 🟡 Early      | Paradigm shift       |
| AI + Event Sourcing              | 🔴 High        | 🔴 Very High  | 🟡 Early      | Emerging rapidly     |
| Decider Pattern                  | 🟠 Medium-High | 🟠 Medium     | 🟢 Mature     | Best practice        |
| Eventium (Haskell)               | 🟠 Medium      | 🔴 High       | 🟡 Early      | Type safety frontier |
| KurrentDB Archiving              | 🟠 Medium-High | 🟠 Medium     | 🟢 Production | Cost optimization    |
| Multi-Stream Atomic Appends      | 🔴 High        | 🟠 Medium     | 🟢 Production | Simplifies sagas     |
| Functional Core/Imperative Shell | 🟠 Medium-High | 🟠 Medium     | 🟢 Mature     | Best practice        |
| Message-Driven Projections       | 🟠 Medium-High | 🟡 Low-Medium | 🟢 Production | Performance critical |
| Actor + Event Sourcing           | 🟠 Medium      | 🟡 Low        | 🟢 Mature     | Proven pattern       |
| Rust Event Sourcing              | 🟡 Medium      | 🟡 Low        | 🟡 Growing    | Performance niche    |

### Trends Summary

```
2024: Functional patterns mature (Decider, Fold, pure functions)
2025: AI convergence begins (LLM-ES interfaces, analytical projections)
2026: Aggregateless approaches emerge, type-safe sagas (Eventium)
      KurrentDB rebrand + cloud-native connectors mature
```

### What This Means for go-cqrs-lite

| Trend                      | Relevance             | Action                                                     |
| -------------------------- | --------------------- | ---------------------------------------------------------- |
| Decider pattern            | ✅ Already follows it | Document as architectural decision                         |
| AI + ES                    | 🟡 Emerging           | Catalog module's structured metadata is AI-ready           |
| Aggregateless ES           | 🟠 Interesting        | Could inform future projection/query architecture          |
| Message-driven projections | ✅ Partially done     | `InMemoryRunner` exists; consider persistent subscriptions |
| Type-safe sagas            | 🔴 High value         | Process manager / saga support would be impactful          |
| Multi-backend              | ✅ Already done       | Store/Bus interfaces are backend-agnostic                  |

---

## Key Takeaways

1. **The aggregate is no longer sacred** — Aggregateless event sourcing challenges the fundamental unit of traditional ES. Whether it wins or not, the critique is valuable.

2. **AI is the killer app for event sourcing** — Not auditability, not temporal queries, but AI. Event streams are the perfect training data: temporal, domain-expressed, immutable, reproducible.

3. **Pure functions won** — The Decider pattern (pure `decide` + `evolve`) is now the standard across Haskell, F#, .NET, and even Go frameworks. OOP aggregates are legacy.

4. **Type-safe sagas are the frontier** — Eventium's approach of encoding compensation in the type system is the most innovative pattern discovered. Compile-time saga correctness.

5. **Databases are converging** — KurrentDB's archive tier, analytical SQL, and connector ecosystem blur the line between event store, stream processor, and analytical database.

6. **Go is well-positioned** — The Go ecosystem (goes, EventHorizon, Watermill, go-cqrs-lite) has strong coverage. Rust is emerging for performance-critical niches.

---

_Research conducted May 2026. Sources include project documentation, blog posts, conference talks, and technical articles from the event sourcing and CQRS communities._
