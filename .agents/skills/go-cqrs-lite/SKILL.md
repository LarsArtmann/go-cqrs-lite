---
name: go-cqrs-lite
description: Build Go applications with go-cqrs-lite — the composable CQRS + Event Sourcing library (event-sourced aggregates, deciders, projections, read models, SQL/Pebble/Turso storage, snapshots, schema evolution, signing, encryption, scheduling, deriver sagas, graph projections, catalog docs). Use this skill whenever a project imports any github.com/larsartmann/go-cqrs-lite/*/v4 module, OR the user asks how to build CQRS/event-sourcing systems in Go, dispatch commands/queries, build read models or projections, use event stores or buses, or work with any go-cqrs-lite module (event, command, query, decider, id, codec, storage, stack, kv, listing, projection, projectionhost, schema, signing, encryption, middleware, otel, catalog, watermill, scheduling, deriver, graph, prometheus, scenario) — even when the user does NOT name the library explicitly (e.g. "set up event sourcing", "build a read model", "dispatch a command", "snapshot an aggregate", "replay events", "idempotent commands", "soft-delete aggregate", "project events to SQL").
user-invocable: true
metadata:
  tags: cqrs, event-sourcing, go, decider, projection, read-model, event-store, domain-driven-design
---

# go-cqrs-lite — AI Consumer Guide

A **library, not a framework**: import only the modules you need; compose them. Core loop: Command→Dispatcher→Handler→Decider(load→fold→decide→save→publish)→EventStore+Bus→Projection→ReadModel→Query.

## Read the matching guide

| Need                                                                              | Guide                                       |
| --------------------------------------------------------------------------------- | ------------------------------------------- |
| Mental model, quickstart, decision matrix, conventions, cheat sheet               | [`core.md`](references/core.md)             |
| Recipes (event sourcing, persistence, snapshots, signing, encryption, OTel, docs) | [`recipes.md`](references/recipes.md)       |
| Read models (projections, SQL views, tier selection)                              | [`readmodels.md`](references/readmodels.md) |
| All 28 modules: imports + one-liners                                              | [`modules.md`](references/modules.md)       |
| Advanced (tombstone, watermill, gRPC, projectionhost, scheduling, graph, SSE)     | [`advanced.md`](references/advanced.md)     |
| Pitfalls & FAQ                                                                    | [`faq.md`](references/faq.md)               |

Read [`core.md`](references/core.md) first — it has the decision matrix.
