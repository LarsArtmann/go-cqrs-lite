# ADR-0091: SSE Consolidation Decision

**Date:** 2026-08-02
**Status:** Accepted

## Context

The codebase has two SSE (Server-Sent Events) implementations that appear to
overlap:

1. **`metaengine.ServeSSE`** — pushes materialized query result changes from
   a `Watcher[V]` (collection-level, typed values `V`).
2. **`transport/http.SSEBroker`** — pushes raw domain events from an
   `event.Bus` (stream-level, `event.Event`).

A previous TODO item ("SSE consolidation") asked whether these should be merged
into a single implementation. This ADR documents the decision and rationale.

## Decision

**Keep both implementations separate. They serve different layers and should
not be merged.**

### Layer Comparison

| Aspect                | `metaengine.ServeSSE`               | `transport/http.SSEBroker`        |
| --------------------- | ----------------------------------- | --------------------------------- |
| **Data source**       | `Watcher[V]` (collection updates)   | `event.Bus` (domain event stream) |
| **Payload**           | Typed value `V` (JSON-encoded)      | Raw `event.Event` payload bytes   |
| **Replay source**     | In-memory ring buffer (`SSEReplay`) | `event.SeekableJournal`           |
| **Replay key**        | Sequence number (`uint64`)          | Event ID (`Last-Event-ID`)        |
| **Dedup mechanism**   | Seq-number ring                     | `dedup.Ring` (event IDs)          |
| **Module dependency** | `metaengine` (Tier 0)               | `transport/http` (Tier 4)         |
| **Event filtering**   | No (collection is already scoped)   | Yes (`WithEventFilter`)           |
| **Payload transform** | No (values are already JSON)        | Yes (`WithPayloadTransform`)      |
| **Byte budget**       | No                                  | Yes (`WithReplayByteBudget`)      |
| **REST backfill**     | No                                  | Yes (`BackfillHandler`)           |
| **Graceful shutdown** | No                                  | Yes (`CloseWithGrace`)            |
| **OTel spans**        | No                                  | Yes (fanout + replay)             |
| **Target consumer**   | Browser `EventSource` (read model)  | Browser or service (event stream) |

### Rationale

1. **Different abstraction levels.** `ServeSSE` operates at the read-model
   layer — it pushes the _result_ of a projection (e.g., "user count is 42").
   `SSEBroker` operates at the event-sourcing layer — it pushes the _events_
   that produce those results (e.g., "user.created"). Merging them would
   collapse two distinct architectural layers into one.

2. **Module boundary preservation.** `metaengine` is a Tier 0 primitive with
   zero internal dependencies (stdlib + `database/sql` only). It cannot
   import `event/` or `transport/http/`. Merging the SSE implementations
   would either:
   - Pull `event.Bus` into `metaengine` (breaks the dependency boundary), or
   - Push `Watcher` into `transport/http` (couples transport to a specific
     storage engine).

3. **Different replay strategies.** `ServeSSE` uses an in-memory ring buffer
   (`SSEReplay[V]`) because collection updates are cheap to buffer and the
   consumer only needs recent values. `SSEBroker` uses `event.SeekableJournal`
   because domain events are the source of truth and must survive process
   restarts. A merged implementation would need to support both strategies,
   adding complexity rather than reducing it.

4. **Different feature sets.** `SSEBroker` has features that only make sense
   for raw event streams: event-type filtering, payload transforms (CBOR→JSON
   for browsers), byte-budgeted replay (events can be large), and REST
   backfill. `ServeSSE` has features that only make sense for read models:
   heartbeat-based keepalive and collection-scoped subscription. Bundling
   all features into one type would create a god-object with most features
   unused by either consumer.

5. **Composition over merging.** A consumer who needs both can compose them:
   serve read-model updates via `ServeSSE` on one endpoint and raw events via
   `SSEBroker` on another. The two are designed to coexist, not compete.

## Consequences

- Two SSE implementations coexist. This is intentional, not accidental.
- Adding a third SSE use case (e.g., projection-level SSE for a specific
  read-model store) should follow the pattern: implement it in the module
  that owns the data source, not in a shared "SSE utility" package.
- The `metaengine` module's zero-dependency boundary is preserved.
- Consumers who want both read-model push and event push import both
  `metaengine` and `transport/http` — no conflict, no duplication.

## Alternatives Considered

> **Update 2026-08-03:** ADR-0097 supplements this decision with the finding
> that `github.com/larsartmann/go-sse` already exists as a standalone SSE
> primitives library. ADR-0091 was written without knowledge of go-sse. The
> consumption of go-sse wire-format primitives does NOT reverse this decision —
> both implementations remain separate and serve different layers.

### Merge into `transport/http`

Rejected: `transport/http` would need to import `metaengine` for `Watcher[V]`,
adding a Tier 0 → Tier 4 reverse dependency. The metaengine module's
zero-dependency principle (ADR-0062) would be violated.

### Merge into `metaengine`

Rejected: `metaengine` would need to import `event.Bus` and
`event.SeekableJournal`, pulling the event-sourcing layer into the storage
planner. This couples two independent subsystems.

### Shared SSE utility package

Rejected: A `sseutil` package with shared framing logic (SSE header parsing,
`Last-Event-ID` extraction, keepalive) would save ~20 lines of code per
implementation but add a new module dependency for both `metaengine` and
`transport/http`. The shared code is trivial Go stdlib (`http.Flusher`,
`fmt.Fprintf`); extracting it adds coupling without meaningful deduplication.
