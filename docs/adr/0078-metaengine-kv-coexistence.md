# ADR-0078: Metaengine and kv.ViewStore coexistence

|             |                                                                                      |
| ----------- | ------------------------------------------------------------------------------------ |
| **Status**  | Accepted                                                                             |
| **Date**    | 2026-07-31                                                                           |
| **Context** | Two query layers exist: kv.ViewStore (simple KV) and metaengine (cost-based planner) |

## Context

The codebase has two read-model query layers:

1. **`kv.ViewStore[V,K]`** via `stack.Materialize` — a simple key-value
   projection: one document per key, point lookup via `Get`, full scan via
   `List`. No server-side filtering, sorting, or pagination. Filtering is
   done in Go after `List` returns all rows. Simple, predictable, sufficient
   for small collections.

2. **`metaengine/`** — a cost-based query planner with 7 ADTs (Map, Set,
   Counter, Graph, Multimap, Log, SortedMap), declarative filter/sort
   pushdown (`FilterOnField`, `SortOnField`), and multi-engine support
   (Memory, SQLite, Pebble). The planner assigns each query to the cheapest
   engine. Supports server-side filtered scans via SQL `json_extract()`
   pushdown.

## Decision

**Coexist. Do not replace kv.ViewStore with metaengine.**

The two layers serve different complexity tiers:

- **kv.ViewStore** is the right choice for simple read models where the
  consumer needs point lookups or small full-scan lists. It has zero
  planner overhead, no engine management, and works with any KV backend.
  Most CRUD applications use this tier.

- **metaengine** is the right choice when the consumer needs server-side
  filtering, sorting, pagination, aggregate counts, or multi-engine cost
  optimization. It adds declaration complexity (fold handlers, query config)
  but eliminates O(N) Go-side filtering.

## Migration Path

Consumers can migrate incrementally:

1. Start with `stack.Materialize` + `kv.ViewStore` for all read models.
2. When a query becomes slow (O(N) filter on large collections), declare
   a metaengine `Query` with `FilterOnField` for that specific read model.
3. Run both projections in parallel during migration (the taskmanager
   example demonstrates this pattern: `mat` + `meAdapter` both registered
   with the projection host).
4. Once the metaengine projection is verified, switch the handler to use
   `TypedReader.Scan` and keep the old projection as a fallback.

## Consequences

- Both layers are first-class, documented, and tested.
- The taskmanager example demonstrates the parallel-run migration pattern.
- Consumers choose based on query complexity, not a forced migration.
- No timeline for deprecating kv.ViewStore — it remains the recommended
  default for simple use cases.
