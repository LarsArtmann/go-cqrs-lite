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
