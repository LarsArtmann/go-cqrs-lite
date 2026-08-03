---
name: go-cqrs-lite
description: Build Go applications with go-cqrs-lite — the composable CQRS + Event Sourcing library (event-sourced streams, deciders, projections, read models, SQL/Pebble/Turso storage, snapshots, schema evolution, signing, encryption, scheduling, deriver sagas, graph projections, catalog docs). Use this skill whenever a project imports any github.com/larsartmann/go-cqrs-lite/*/v4 module, OR the user asks how to build CQRS/event-sourcing systems in Go, dispatch commands/queries, build read models or projections, use event stores or buses, or work with any go-cqrs-lite module (event, command, query, decider, id, codec, storage, stack, kv, listing, projection, projectionhost, schema, signing, encryption, middleware, otel, catalog, watermill, scheduling, deriver, graph, prometheus, scenario, flightrecorder) — even when the user does NOT name the library explicitly (e.g. "set up event sourcing", "build a read model", "dispatch a command", "snapshot a stream", "replay events", "idempotent commands", "soft-delete stream", "project events to SQL").
user-invocable: true
metadata:
  tags: cqrs, event-sourcing, go, decider, projection, read-model, event-store, domain-driven-design
---

# go-cqrs-lite

A **library, not a framework**: import only the modules you need; compose them.

Core loop: Command→Dispatcher→Handler→Decider(load→fold→decide→save→publish)→EventStore+Bus→Projection→ReadModel→Query.

**Read [`core.md`](references/core.md) first** — decision matrix, conventions, cheat sheet, anti-patterns.

- [`recipes.md`](references/recipes.md) — event sourcing, persistence, snapshots, signing, encryption, OTel
- [`readmodels.md`](references/readmodels.md) — projections, SQL views, tier selection
- [`modules.md`](references/modules.md) — all modules: imports + one-liners
- [`advanced.md`](references/advanced.md) — tombstone, watermill, gRPC, projectionhost, scheduling, graph, SSE
- [`faq.md`](references/faq.md) — pitfalls & common mistakes

### Benchmarking

`cqrs-bench` is the CLI front-end for the `benchkit` library. It runs synthetic event
workloads (write, read, read-model, durability phases) against any stack preset and
reports latency percentiles, throughput, heap, and disk metrics.

```bash
go build -o cqrs-bench ./cmd/cqrs-bench/
./cqrs-bench run --backend sqlite --profile small           # single backend
./cqrs-bench compare --profile small --format markdown      # all 3 backends side-by-side
./cqrs-bench run --backend pebble --profile medium --codec cbor  # CBOR vs JSON
```

| Use `cqrs-bench` when...                        | Use `go test -bench` (stack/bench) when...  |
| ----------------------------------------------- | ------------------------------------------- |
| Comparing backends (memory vs sqlite vs pebble) | Micro-benchmarks inside your own test suite |
| Measuring latency percentiles at scale          | Measuring a single operation's ns/op        |
| Checking codec impact (JSON vs CBOR)            | Checking allocation counts                  |
| CI performance regression gates                 | Quick local iteration during development    |

### Routing Decision Matrices

#### SSE: Which implementation?

go-cqrs-lite has **two SSE implementations** (ADR-0091: kept separate — different layers, different data sources). **Both consume [`go-sse`](https://github.com/larsartmann/go-sse) internally** for wire-format serialization (`sse.WriteEvent`, `sse.SetHeaders`, `sse.WriteHeartbeat`) — the duplicated `fmt.Fprintf`/byte-append serializer was eliminated in ADR-0097. They are NOT merged: each preserves its own fan-out, replay, and feature set.

| You want to push…                                             | Module           | Function            | Source                                | Replay                                           | Key features                                                                                                           |
| ------------------------------------------------------------- | ---------------- | ------------------- | ------------------------------------- | ------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------- |
| **Raw domain events** to browser/HTTP client                  | `transport/http` | `SSEBroker`         | `event.Bus` + `event.SeekableJournal` | Journal-backed (**durable**)                     | Event filter, CBOR→JSON transform, byte budget, REST `BackfillHandler`, OTel replay metrics, `Last-Event-ID` reconnect |
| **Materialized query results** (read-model values) to browser | `metaengine`     | `ServeSSE[V]`       | `Watcher[V]` (Store collection)       | In-memory ring (`SSEReplay[V]`, **recent-only**) | Heartbeat keepalive, `Last-Event-ID` reconnect, drop-old backpressure, timeout                                         |
| **Events to a server-side worker/projection** (not a browser) | `watermill`      | `CatchUpSubscriber` | `event.SeekableJournal` + live sub    | Checkpoint store (**durable**)                   | Crash-restart, routes through any broker (NATS/Kafka/Redis), ordered delivery                                          |

**Rule of thumb:**

- Browser needs the **event log** (audit feed, notification stream, event-sourced UI) → `transport/http.SSEBroker`.
- Browser needs the **current read-model state** (live-updating table/dashlet built from a `metaengine.Store`) → `metaengine.ServeSSE`.
- Server-side projection/integration that must survive crashes → `watermill.CatchUpSubscriber` + `projectionhost.Host`.

**Do NOT merge the two browser-facing implementations** — `SSEBroker` replays from a durable journal (survives restart, cross-process), while `ServeSSE` replays from an in-process ring buffer (cheap, recent-only, lost on restart). See `advanced.md` §6.15–6.16 for the full comparison and code.

#### Read models: Which tier?

| Data shape                                 | Query pattern            | Recommended tier                           |
| ------------------------------------------ | ------------------------ | ------------------------------------------ |
| One document per key                       | Get/Set by key           | `kv.ViewStore[V,K]` or `stack.Materialize` |
| Multi-table, joins, relations              | SQL WHERE/ORDER BY/LIMIT | `storage.RelationalProjection`             |
| Variable-depth traversal, adjacency, paths | N-hop queries            | `graph.GraphProjection`                    |
| Event-folded aggregations, counters        | Cost-planned queries     | `metaengine` Store + `projectionadapter`   |

#### Dead-letter handling: Which layer?

| Scenario                           | Module           | Mechanism                                                       |
| ---------------------------------- | ---------------- | --------------------------------------------------------------- |
| Event projection poison messages   | `projectionhost` | `WithDeadLetterStore(dlq, retries)` — per-worker retry + poison |
| Command/event/query dispatch retry | `middleware`     | `middleware.Retry` + `middleware.Recovery`                      |
| Idempotent delivery dedup          | `middleware`     | `middleware.{Command,Event,Query}Idempotency`                   |

#### Dedup: Which store?

| Scenario                    | Module                 | Store                                 |
| --------------------------- | ---------------------- | ------------------------------------- |
| In-process, fast, ephemeral | `idempotency`          | `MemoryStore`                         |
| SQL-backed, persistent      | `idempotency/sqlstore` | `NewSQLiteStore` / `NewPostgresStore` |
| KV-backed (Pebble, etc.)    | `idempotency/kvstore`  | `KVStore` with any `kv.Store`         |
