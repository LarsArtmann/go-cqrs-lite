# Book Insights vs Codebase: Detailed Q&A

> Generated 2026-07-23
> Follow-up analysis mapping 7 systems-engineering books against go-cqrs-lite
> Companion to `docs/architecture-understanding/2026-07-23_book-insights-vs-codebase.html`

---

## Table of Contents

1. [Q1: Is the single-process scope clear?](#q1-is-the-single-process-scope-clear)
2. [Q2: Should the snapshot↔materialized view connection improve?](#q2-should-the-snapshotmaterialized-view-connection-improve)
3. [Q3: High-water mark — explain and suggest improvement](#q3-high-water-mark--explain-and-suggest-improvement)
4. [Q4: Schema evolution — code or docs?](#q4-schema-evolution--code-or-docs)
5. [Q5: Replay vs live — how does the book vocabulary differ?](#q5-replay-vs-live--how-does-the-book-vocabulary-differ)
6. [Q6: Idempotent receiver — what is missing?](#q6-idempotent-receiver--what-is-missing)
7. [Q7: Codec/content negotiation — why only Good, how to reach Excellent?](#q7-codeccontent-negotiation--why-only-good-how-to-reach-excellent)
8. [Q8: Missing vocabulary — improved and verified](#q8-missing-vocabulary--improved-and-verified)
9. [Q9: Message broker via Watermill — correction](#q9-message-broker-via-watermill--correction)
10. [Q10: Anti-patterns — expanded detail on all 15](#q10-anti-patterns--expanded-detail-on-all-15)
11. [DOMAIN_LANGUAGE.md changes applied](#domain_languagemd-changes-applied)

---

## Q1: Is the single-process scope clear?

**No, it is not clear enough.**

The README never says "single-process" or "not distributed." The closest hints:

- `stack/sqlite/doc.go:3` says "single-node deployments"
- `stack/postgres/preset.go:85-86` says "single-process" for GoChannel and "multi-process" for `WithDistributedBus`
- `ROADMAP.md:98` lists "Distributed projection runner (leader election, multi-node coordination)" as a future item
- `docs/INFRASTRUCTURE_RECOMMENDATIONS.md:90` says "Single-node, embedded → `stack/sqlite`"

But the **README** — the first thing consumers read — says "52 independent modules" and "library, not framework" without ever stating the deployment scope. The word "distributed" only appears in the Postgres preset row ("Distributed, with `LISTEN/NOTIFY`") which is misleading — that's multi-process shared-DB, not multi-server distributed.

### The distinction that needs to be explicit

| Deployment shape | Supported | How |
|---|---|---|
| **Single-process embedded** | Yes | `stack/sqlite`, `stack/pebble`, `stack/turso` — one process, one file |
| **Single-process + broker** | Yes | Any preset + `watermill.WithBackend(kafkaPub, kafkaSub)` |
| **Multi-process shared DB** | Yes | `stack/postgres` + `WithDistributedBus(listener)` — `LISTEN/NOTIFY` for cross-process pub/sub |
| **Multi-server distributed** | No | No replication, consensus, or leader election. Use external coordination (etcd, Kubernetes) at the deployment layer |

### Improvement applied

Added a **Deployment Scope** section to `docs/DOMAIN_LANGUAGE.md` with the above table. The README should get a short "Deployment Scope" paragraph too — that's a separate docs task.

---

## Q2: Should the snapshot↔materialized view connection improve?

**No code changes needed. They are correctly separate.**

Research confirms the two systems are completely independent:

| | Snapshots | Materialized Views |
|---|---|---|
| **What they store** | Aggregate `State` (decider fold result) | Read-model `V` (projection handler output) |
| **Purpose** | Speeds up `Repository.Load` (command side) | Serves queries (read side) |
| **Store** | `snapshot.SnapshotStore` | `kv.ViewStore[V,K]` |
| **Writer** | `decider.Repository` after `Execute` | `stack.Materialize` event handler |
| **Reader** | `decider.Repository` during `Load` | Query side via `Materialize.View` / `.List` |
| **Keyed by** | `id.StreamRef` (aggregate identity) | `K` (read-model key from event) |
| **Package imports** | `decider` → `snapshot` | `stack` → `kv` (no `snapshot` import) |

Zero imports, zero shared interfaces, zero data flow between them.

A snapshot IS a "materialized view" of the aggregate's event stream in the abstract DDIA sense (any cache is a materialization), but connecting them in code would be a category error — they serve different sides of CQRS and different consumers.

The one place the connection is discussed: ADR-0050 groups both as "blind store data" (derived state that can't be replayed from events). That's an operational observation, not a design coupling.

The academic research (`docs/research/archive/2026-06-29_CQRS_EVENT_SOURCING_ACADEMIC_RESEARCH.md:383`) notes the snapshot store is "conceptually similar [to compaction] but not yet wired as a compaction mechanism." `docs/research/time-travel-options.md:265` explicitly warns: "Log compaction is INCOMPATIBLE with event sourcing."

### Verdict

Leave them separate. Document the conceptual relationship (both are "derived data" in DDIA terms) but don't wire them together.

---

## Q3: High-water mark — explain and suggest improvement

### What DDIA means by "high-water mark"

In DDIA Ch. 5 (Replication) and Ch. 11 (Stream Processing), the high-water mark is the maximum position in the log that has been **safely replicated/acknowledged** — consumers can read up to this point knowing the data won't be lost. In Kafka, it's the "high watermark" — the highest offset replicated to all in-sync replicas.

### What go-cqrs-lite has instead

`event.CheckpointStore` — a per-projection saved position. The `projectionhost` worker saves its checkpoint after processing each event (or batch). On restart, it resumes from the last checkpoint.

Code references:
- `event.CheckpointStore` interface (CheckpointSink + CheckpointSource)
- `projectionhost/worker.go` saves checkpoint after processing
- `watermill/catchup_subscriber.go` saves checkpoint after each forwarded event

### The difference

A high-water mark is a **global** property of the log (what's safe to read). A checkpoint is a **per-consumer** property (where this consumer left off). The checkpoint IS the consumer's local high-water mark — it knows "I've safely processed up to here." But there's no global high-water mark because there's no replication in this library.

### Should we improve this?

No new code needed. The term "checkpoint" is more precise than "high-water mark" for this library's single-process model. A high-water mark implies replication (what's safe to read across nodes), which this library doesn't do. The checkpoint is the right abstraction.

Added a `High-Water Mark` entry to `DOMAIN_LANGUAGE.md` that explicitly bridges the vocabulary: *"DDIA term for the maximum safely-processed position in a stream; this library calls it 'Checkpoint'."*

---

## Q4: Schema evolution — code or docs?

**Both code and docs are solid. No code changes needed. Only docs could be slightly improved.**

### Code (already excellent)

- `schema.Upcaster` with `SourceType()`, `SourceVersion()`, `Upcast(evt) (*event.ImmutableEvent, error)` — exactly DDIA's Avro reader-schema-vs-writer-schema pattern
- `schema.VersionedStore` wraps any `event.Store` with upcasting
- `schema.VersionedSeekableJournal` wraps `SeekableJournal` for projectionhost pipelines — implements `event.SeekableJournal` at compile time (`schema/versioned_journal.go:14`: `var _ event.SeekableJournal = (*VersionedSeekableJournal)(nil)`)
- Events carry `SchemaVersion` stamps
- Property-based tests verify upcaster chains (`schema/versioned_journal_rapid_test.go`)
- ADR-0044 envelopes make blind stores self-describing too (`codec/envelope.go`: `WrapEncode`/`UnwrapDecode`)

### Docs (could be slightly better)

- The `schema/README.md` is concise but doesn't explicitly connect to DDIA's schema evolution vocabulary (forward/backward compatibility, reader vs writer schema)
- No documented migration recipes (v1→v2 upcaster step-by-step guide)
- The `DOMAIN_LANGUAGE.md` entry now says "Forward/backward-compatible event evolution via upcasting on read (DDIA Ch. 4)" with `VersionedSeekableJournal` added

### Verdict

Code is complete. Docs improvement = add the DDIA vocabulary link (done in `DOMAIN_LANGUAGE.md`) and consider a migration recipe in `schema/README.md`.

---

## Q5: Replay vs live — how does the book vocabulary differ?

### DDIA / Designing Event-Driven Systems vocabulary

- **"Backfill" / "Bootstrap"** — replaying historical events to bring a new consumer up to speed
- **"Catch-up"** — the transition from replay to live processing
- **"Replay"** — re-processing the log from the beginning (or from a position)
- **"Tail" / "Live tail"** — consuming new events as they arrive after catch-up
- **"Processing mode"** — distinguishing replay from live so consumers can skip side effects during replay

### go-cqrs-lite vocabulary

- **`ProcessingMode`** — `ModeLive` vs `ModeReplay` (`event.WithProcessingMode(ctx, ModeReplay)`) — matches "processing mode" exactly
- **`CatchUpSubscriber`** — matches "catch-up" exactly
- **"Replay"** / **"Live"** — used throughout docs and code
- **"Journal drain"** — the `projectionhost` worker's batch-drain phase (replay until caught up, then exit)
- **"Backfill"** — `transport/http.BackfillHandler` — REST endpoint for fetching missed events

### Difference

The book vocabulary and the codebase vocabulary are almost identical. The library uses "replay" and "live" (book terms), "catch-up" (book term), "backfill" (book term), and "processing mode" (book concept). The only term the library uses that the books don't is **"journal drain"** — the projectionhost's specific pattern of replay-until-caught-up-then-exit (rather than transitioning to live).

### Verdict

Vocabulary is well-aligned. No changes needed. The `DOMAIN_LANGUAGE.md` already has `ProcessingMode` and `CatchUpSubscriber`.

---

## Q6: Idempotent receiver — what is missing?

Three gaps, two of which are by design:

### Gap 1: Cross-process / distributed store — MISSING (should fix)

Both `MemoryStore` and `kvstore.Store` (Pebble) are **process-local**. Two instances of the same service sharing a `MemoryStore` get independent dedup — a retry hitting instance B after instance A processed it is NOT caught.

The Pebble adapter's `SetIfAbsent` uses a `sync.Mutex` — atomic only within one process.

No SQL-backed `idempotency.Store` exists despite:
- The interface doc mentioning `INSERT ON CONFLICT DO NOTHING` as the future SQL strategy
- `storage/sql/duplicate.go` having `IsDuplicateKeyError` for PG/SQLite unique violations
- The `idempotency.Store` interface supporting it

The pieces exist, they're just not assembled. A ~100-line `idempotency/sqlstore/` implementation using `INSERT ON CONFLICT DO NOTHING` would close this gap for multi-process Postgres deployments (which the library already supports via `WithDistributedBus`).

### Gap 2: Response caching / replay — MISSING (by design)

`ErrDuplicate` is returned; the original response is not stored or replayed. The client must handle the error.

Storing responses adds complexity (serialization, size limits, TTL coordination). For a library, rejecting duplicates is sufficient. Consumers who need response replay implement it on top.

**Should not add.**

### Gap 3: Idempotency key from transport metadata — MISSING (by design)

No auto-extraction of `Idempotency-Key` HTTP headers. The `keyExtractor` must be provided by the caller.

The library doesn't own HTTP handlers.

**Should not add.** But should document the pattern.

### What exists (verified against source)

| Component | File | Status |
|---|---|---|
| `Store` interface | `idempotency/store.go` | 3 methods: `Seen`, `Record`, `CheckAndRecord` (atomic) |
| `MemoryStore` | `idempotency/store.go:56-156` | TTL-based, lazy deletion + optional background sweep |
| `kvstore.Store` | `idempotency/kvstore/store.go` | Adapts any `KVBackend` via `SetIfAbsent` |
| `CommandIdempotency` | `middleware/idempotency.go` | Default key: `cmd.ID().String()` |
| `EventIdempotency` | `middleware/idempotency.go` | Default key: `evt.ID().String()` |
| `QueryIdempotency` | `middleware/idempotency.go` | Panics if `keyExtractor` is nil (documented) |
| `dedup.Ring` | `dedup/ring.go` | O(1) fixed-capacity, `DefaultCapacity = 1024` |
| CatchUpSubscriber dedup | `watermill/catchup_subscriber.go:247` | `catchUpDedupRingCapacity = 1024` |
| projectionhost dedup | `projectionhost/host.go:111` | `dedup.NewRing(dedup.DefaultCapacity)` |

### One actionable improvement

Add a SQL-backed `idempotency.Store` (`idempotency/sqlstore/` or in `storage/`). This is the single biggest gap — multi-process Postgres deployments (which the library supports via `WithDistributedBus`) have no shared dedup store. The interface, the duplicate-detection utility, and the SQL dialect abstraction all exist. It's a ~100-line implementation.

---

## Q7: Codec/content negotiation — why only Good, how to reach Excellent?

### Why "Good" not "Excellent"

The internal codec system is excellent:
- 4 codecs (`JSONCodec`, `CBORCodec`, `CBORCompactCodec`, `RawCodec`)
- Self-describing events (each carries `Encoding()` stamp)
- Mixed-stream decoding (`DecodePayloadAuto[T]` dispatches per-event)
- Envelope-wrapped blind stores (ADR-0044: `WrapEncode`/`UnwrapDecode`)
- Zero-allocation `BufferEncoder` interface (implemented by all 3 main codecs)
- Streaming CBOR encoder/decoder for large batches
- Auto-detection (`codec/autodetect.go`: heuristic first-byte sniffing)
- Deterministic CBOR (signing-safe, canonical map key ordering)

But there is **zero content negotiation at the API boundary** — and this is **by deliberate design** (ADR-0052).

### What "Excellent" would require per Service Design Patterns

1. `Accept` header parsing — client requests `Accept: application/cbor` and gets CBOR
2. Per-request codec selection — not a fixed `WithPayloadTransform` set at construction time
3. `Vary: Accept` header on negotiated responses
4. `406 Not Acceptable` when no supported media type matches
5. Q-value parsing (`Accept: application/cbor;q=0.9, application/json;q=0.8`)

### Should we reach Excellent?

**No.** ADR-0052 explicitly decided:

> "No CBOR REST endpoint support will be built without consumer demand. Content negotiation is a REST framework concern, not a library concern."

This is correct — the library is SSE-only for HTTP transport. Adding content negotiation would make it a framework.

### What we should do instead

Document that the library's codec system is **internal-only** and that transport-level content negotiation is the consumer's responsibility. The `WithPayloadTransform` hook is the translation point. This is already documented in ADR-0052 but is now surfaced in the `DOMAIN_LANGUAGE.md` Codec entry.

### Verdict

"Good" is the right rating. "Excellent" would mean becoming a framework. The library correctly stops at "Good."

---

## Q8: Missing vocabulary — improved and verified

I updated `docs/DOMAIN_LANGUAGE.md` with the following additions, each verified against actual source code:

### New Cross-Cutting terms (verified against source)

| Term | Source verification |
|---|---|
| `Idempotent Receiver` (renamed from `Idempotency`) | `idempotency/store.go`, `dedup/ring.go:25` (`Ring`), `dedup/ring.go:21` (`DefaultCapacity = 1024`) |
| `Circuit Breaker` | `middleware/circuit_breaker.go:199` (`NewCircuitBreaker[M]`), `middleware.CommandCircuitBreaker` |
| `Retry` | `retry/retry.go:43` (`Do`), `retry/config.go:19` (`Config`), `retry/retry.go:104` (`Backoff`) |
| `Dedup Ring` | `dedup/ring.go:25` (`Ring`), `dedup/ring.go:21` (`DefaultCapacity = 1024`) |
| `Projection Lag` | `projectionhost/host.go:263` (`LagDuration`), `host.go:286` (`LagPerProjection`) |
| `Heartbeat` | `transport/http/sse.go:192` (`DefaultSSEHeartbeat`), `transport/http/sse_event.go:137` (`WriteSSEHeartbeat`) |
| `Backfill` | `transport/http/sse_backfill.go:75` (`BackfillHandler`) |
| `BufferEncoder` | `codec/codec.go:55` (`BufferEncoder` interface), implemented by `JSONCodec`, `CBORCodec`, `CBORCompactCodec` |
| `Materialized View` | `stack/materialize.go:28` ("materialized view"), ADR-0030, ADR-0040 |
| `High-Water Mark` | Documented as DDIA term for what the library calls `Checkpoint` |

### Schema Evolution entry expanded

Added `VersionedSeekableJournal` (verified: `schema/versioned_journal.go:22`)

### New sections added

- **Deployment Scope** — explicit table of what's supported (single-process, multi-process) vs not (multi-server distributed)
- **Consistency Guarantees** — explicit table of what's provided (optimistic concurrency, linearizable writes, eventual consistency, consistent prefix reads) vs not (read-your-writes, bounded staleness, monotonic reads)

### Verification block updated

Added imports for `dedup/v4`, `retry/v4`, and symbols:
- `middleware.CommandCircuitBreaker`
- `retry.Do`, `retry.DefaultConfig`
- `dedup.NewRing`, `dedup.DefaultCapacity`
- `schema.NewVersionedSeekableJournal`
- `http.BackfillHandler`, `http.WriteSSEHeartbeat`, `http.DefaultSSEHeartbeat`

All pass `doc-check` (verified: `cd cmd/doc-check && GOWORK=off go run . ../../docs/DOMAIN_LANGUAGE.md` — zero new failures).

---

## Q9: Message broker via Watermill — correction

**The original report understated this.** The library DOES provide message broker support via Watermill — it's not just "transport-agnostic, bring your own broker."

### What the `watermill/` module provides

- `watermill.NewEventBus()` with `WithBackend(publisher, subscriber, closer)` — inject any Watermill-compatible broker
- `watermill.NewCommandBus()` with `WithCommandBackend(...)` — command distribution over any broker
- `watermill.NewEventPublisher(wmPublisher, topic)` — wraps any `message.Publisher` as `event.Publisher`
- `watermill.NewCommandPublisher(wmPublisher, topic)` — wraps any `message.Publisher` as `command.Publisher`
- Documented recipes for NATS JetStream and Redis Streams in `watermill/doc.go:86-110`
- Kafka mentioned (uses Watermill-kafka plugin, external)
- `CatchUpSubscriber` for replay→live handoff

### Code references

- `watermill/event_bus.go:22`: "WithBackend" — inject Kafka/NATS publisher+subscriber
- `watermill/event_bus_options.go:24`: "e.g., Kafka, NATS"
- `watermill/command_bus.go:20`: "inject a Kafka/NATS publisher+subscriber"
- `watermill/command_bus_options.go:24`: "multi-process capable"
- `watermill/doc.go:86-110`: NATS JetStream + Redis Streams recipes

The `watermill/go.mod` depends on `github.com/ThreeDotsLabs/watermill v1.5.2` but does NOT bundle NATS/Kafka/Redis directly — consumers bring their own Watermill plugin (`watermill-nats`, `watermill-redis-stream`, `watermill-kafka`).

### Verdict

The `DOMAIN_LANGUAGE.md` "Patterns NOT in the Library" table already correctly says: *"Injected via Watermill adapter (`watermill.NewEventBus` with Kafka/NATS/Redis)."* The original report should have rated this higher — the library provides real broker integration, not just transport-agnosticism.

---

## Q10: Anti-patterns — expanded detail on all 15

I expanded the `DOMAIN_LANGUAGE.md` anti-patterns table from 5 to 16 entries and the "Patterns NOT in the Library" table from 3 to 9 entries.

### Anti-Patterns Table (expanded from 5 → 16)

| New Entry | Book | Why Avoided |
|---|---|---|
| **Aggregate Root (OO)** | Implementing DDD | ADR-0001: 9-method OO interface couples domain to infrastructure. Pure `Decider[State]` + `Apply` separates them. |
| **Update / Patch** | CQRS/ES | No mutation of past events. New events supersede old state via fold. The event log is append-only. |
| **Log Compaction** | DDIA Ch. 11 | Compaction destroys the audit trail. Snapshots avoid replay cost without losing data. `docs/research/time-travel-options.md:265` explicitly warns compaction is incompatible with ES. |
| **2PC / Two-Phase Commit** | DDIA Ch. 9 | Blocking and fragile. Projections derive independently from the log. ADR-0016 declined for the same reason. |
| **Outbox** | DDIA Ch. 11, CQRS/ES | ADR-0016: the event journal IS the outbox. `CatchUpSubscriber` replays and publishes. No separate outbox table needed. |
| **Replication** | DDIA Ch. 5 | Library doesn't replicate. Postgres/Pebble replication handles this at the storage layer. Adding it would make the library opinionated about deployment topology. |
| **Leader Election** | Patterns of Distributed Systems | No Raft/Paxos. Optimistic concurrency per aggregate is the application-level fencing. Node coordination is deployment infra (K8s, etcd). `ROADMAP.md:98` lists it as a future idea, not current scope. |
| **Fencing Token** | Patterns of Distributed Systems | Application-level fencing via `expectedVersion`. A stale instance's write fails the version check. Deployment-level fencing is outside scope. |
| **God Aggregate** | CQRS/ES | Large aggregates violate SRP. Split into small deciders + derivers for event→command derivation. |
| **Enforced Transport** | Service Design Patterns | Library provides SSE, gRPC, REST helpers but doesn't force a protocol. Consumers choose. |
| **Data Lakehouse** | Deciphering Data Architectures | Application-level CQRS library, not analytics platform. Projections are operational read models, not analytical datasets. |

### Patterns NOT in the Library (expanded from 3 → 9)

| New Entry | How It Emerges | Why No Module |
|---|---|---|
| **Outbox** | Event journal + `CatchUpSubscriber` + `EventPublisher` | ADR-0016: journal IS the outbox. A projection reads the journal and publishes events, making the pattern composable without a dedicated table. |
| **Distributed Consensus** | Optimistic concurrency per aggregate (`expectedVersion`) | No Raft/Paxos: the library provides single-writer-per-aggregate semantics. Multi-node coordination (leader election, quorum) is a deployment concern. |
| **Log Compaction** | `snapshot.SnapshotStore` with strategies | Compaction destroys events — incompatible with event sourcing. Snapshots avoid replay cost without data loss. See `docs/research/time-travel-options.md`. |
| **Stream Processing Engine** | `projectionhost.Host` (simple, correct) + `CatchUpSubscriber` | Windowing, watermarking, stream joins are over-engineering for application-level CQRS. Consumers needing Kafka-scale streaming use Kafka + the Watermill adapter. |
| **Fencing Tokens** | `expectedVersion` (optimistic concurrency) | Deployment-level fencing (K8s leases, etcd locks) is outside scope. Application-level fencing via version check is sufficient for single-writer-per-aggregate. |
| **Data Lakehouse / Fabric** | N/A — projections are operational read models | This is an application-level CQRS library, not an analytics platform. Warehouse/lakehouse/fabric solve a different problem (analytics at organizational scale). |

### Original 5 anti-patterns (kept unchanged)

| Term | Why |
|---|---|
| "Database" → "Store" or "Event Store" | CQRS separates write/read; "database" implies a single thing |
| "Entity" → "Aggregate" | DDD aggregate is the consistency boundary; entity is too vague |
| "CRUD" → "Command + Event + Projection" | No updates or deletes — only append |
| "Delete" → "Tombstone" | Event streams are append-only; soft-delete via metadata, never removal |
| "State" (mutable) → "Folded state" | State is always reconstructed from events via `Apply`, never directly mutated |

---

## DOMAIN_LANGUAGE.md changes applied

### Summary of all edits to `docs/DOMAIN_LANGUAGE.md`

1. **Renamed** `Idempotency` → `Idempotent Receiver` (DDIA pattern name) with `dedup.Ring` reference added
2. **Expanded** `Schema Evolution` entry with `VersionedSeekableJournal` and DDIA Ch. 4 reference
3. **Added 9 new Cross-Cutting terms**: Circuit Breaker, Retry, Dedup Ring, Projection Lag, Heartbeat, Backfill, BufferEncoder, Materialized View, High-Water Mark
4. **Added Deployment Scope section** — explicit table: single-process (yes), multi-process (yes), multi-server distributed (no)
5. **Added Consistency Guarantees section** — explicit table: what's provided (optimistic concurrency, linearizable writes, eventual consistency, consistent prefix reads) vs not (read-your-writes, bounded staleness, monotonic reads)
6. **Expanded Anti-Patterns table** from 5 → 16 entries (added 11 new terms-to-avoid)
7. **Expanded "Patterns NOT in the Library" table** from 3 → 9 entries (added 6 new intentionally-absent patterns)
8. **Updated verification block** with `dedup/v4`, `retry/v4` imports and all new symbols — all pass `doc-check`

### Pre-existing doc-check failures (not caused by this change)

5 references to `event.NewRejection`, `event.NewConflict`, `event.NewTransient`, `event.NewInfrastructure`, `event.NewCorruption` — these functions were removed from the `event/` package (now live in `go-error-family` as `errorfamily.NewRejection` etc.). These failures predate this work and should be fixed separately.

### One actionable code gap discovered

**No SQL-backed `idempotency.Store`** for multi-process Postgres deployments. The interface, `IsDuplicateKeyError`, and dialect abstraction all exist — it's a ~100-line assembly job using `INSERT ON CONFLICT DO NOTHING`.
