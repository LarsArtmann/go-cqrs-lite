# ADR-0079: SSE consolidation — two implementations, two layers

|             |                                                                   |
| ----------- | ----------------------------------------------------------------- |
| **Status**  | Accepted                                                          |
| **Date**    | 2026-07-31                                                        |
| **Context** | Two SSE implementations exist: transport/http.SSEBroker and metaengine.ServeSSE |

## Context

The codebase has two Server-Sent Events implementations:

1. **`transport/http.SSEBroker`** — bridges an `event.Bus` to HTTP clients.
   Subscribes to domain events (raw event stream) and pushes them to
   connected SSE clients. Supports Last-Event-ID reconnection via an
   optional `event.SeekableJournal`, byte-budgeted replay, payload
   transforms (e.g. CBOR-to-JSON), and dedup. This is the **event-bus
   push** layer.

2. **`metaengine.ServeSSE`** — watches a metaengine Store collection for
   mutations and pushes materialized query results to SSE clients. Supports
   collection-level and key-level subscriptions, replay from a journal,
   and timeout. This is the **read-model push** layer.

## Decision

**Keep both. They serve fundamentally different push semantics.**

- `transport/http.SSEBroker` pushes **raw domain events** — the client
  sees every event that flows through the bus, regardless of projections.
  Use case: real-time event dashboards, audit feeds, event replay.

- `metaengine.ServeSSE` pushes **materialized query results** — the client
  sees changes to a specific query's projection (e.g., "task #42 changed
  from pending to active"). Use case: reactive UIs that need current state,
  not event history.

Consolidating them would require either:
- Making SSEBroker aware of projections (violates transport/http's
  dependency boundary — it only depends on event/)
- Making ServeSSE independent of metaengine (defeats its purpose)

Neither is desirable. The separation is architecturally correct.

## When to Use Which

| Need | Use |
|------|-----|
| Stream raw domain events to clients | `transport/http.SSEBroker` |
| Stream materialized read-model changes to clients | `metaengine.ServeSSE` |
| Simple CRUD without projections | `transport/http.SSEBroker` |
| Reactive UI synced to query results | `metaengine.ServeSSE` |

## Consequences

- Both implementations are documented and maintained.
- The doc comments in both files cross-reference each other, explaining
  the layer difference.
- No consolidation planned — the architectural boundary is intentional.
