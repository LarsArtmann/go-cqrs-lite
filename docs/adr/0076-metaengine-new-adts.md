# ADR-0076: Metaengine New ADTs (Vector, Search, Spatial)

**Date:** 2026-08-01
**Status:** Accepted
**Supersedes:** N/A
**Related:** [ADR-0075](0075-metaengine-layered-architecture.md) (layered architecture), [ADR-0062](0062-metaengine-dependency-boundary.md)

## Context

The metaengine originally shipped with 7 ADTs: Map, Set, Counter, Graph, Log, SortedMap, Multimap. These cover traditional CRUD-style projections but miss three high-value access patterns:

1. **Vector similarity search (k-NN)** — the RAG/semantic search primitive. No Go CQRS library offers planner-routed vector projections.
2. **Full-text search** — inverted index for content queries. Removes the need for external Elasticsearch for simple use cases.
3. **Spatial range queries** — geographic proximity (haversine distance). Unblocks geo apps.

All three follow the existing 9-touchpoint ADT extension pattern: interface design, fold kind, classification, call helper, applyFold case, execute path, typed execute, memory impl, tests.

## Decision

Add three new ADTs to the metaengine, each following the existing pattern:

### Vector ADT (`ADTVector`)

- **Fold kind:** `FoldVector` — handler returns `Embedding{ID, Values []float32}`
- **Backend interface:** `VectorBackend` — `VectorInsert(ctx, col, emb)` + `VectorSearch(ctx, col, query, k, metric)`
- **Read pattern:** `ReadVectorSearch`
- **Typed execute:** `VectorExecuteTyped[Q](ctx, store, input)` → `[]VectorResult`
- **Memory impl:** `MemoryVectorIndex` — brute-force k-NN with cosine/euclidean/dot metrics
- **Classification:** `classifyADT()` returns `ADTVector` when any fold has `FoldVector` kind

### Search ADT (`ADTSearch`)

- **Fold kind:** `FoldSearch` — handler returns `IndexedText{ID, Content}`
- **Backend interface:** `SearchBackend` — `SearchInsert(ctx, col, doc)` + `SearchQuery(ctx, col, query, limit)`
- **Read pattern:** `ReadFullTextSearch`
- **Typed execute:** `SearchExecuteTyped[Q](ctx, store, input)` → `[]SearchResult`
- **Memory impl:** `MemorySearchIndex` — TF-IDF inverted index
- **Classification:** `classifyADT()` returns `ADTSearch` when any fold has `FoldSearch` kind

### Spatial ADT (`ADTSpatial`)

- **Fold kind:** `FoldSpatial` — handler returns `Point{ID, X, Y}`
- **Backend interface:** `SpatialBackend` — `SpatialInsert(ctx, col, pt)` + `SpatialRange(ctx, col, x, y, radius, limit)`
- **Read pattern:** `ReadSpatialRange`
- **Typed execute:** `SpatialExecuteTyped[Q](ctx, store, input)` → `[]SpatialResult`
- **Memory impl:** `MemorySpatialIndex` — brute-force haversine range query
- **Classification:** `classifyADT()` returns `ADTSpatial` when any fold has `FoldSpatial` kind

### ADT classification priority

The `classifyADT()` function uses a priority order. The new ADTs are checked BEFORE the existing ones because they are more specific (an Embedding-returning fold is unambiguously a Vector query, not a Map insert):

```
Vector → Search → Spatial → Graph → Counter → Multimap → Log → Set → Map
```

## Consequences

### Positive

- **Three modern access patterns** available via the planner — RAG, full-text search, and geo queries
- **Zero new production dependencies** in the metaengine core — all brute-force implementations use stdlib only (math, sort, strings)
- **Mechanical extension** — each ADT followed the exact 9-touchpoint pattern, adding zero risk to existing ADTs
- **Memory engine gains 3 new backend interfaces** — VectorBackend, SearchBackend, SpatialBackend, all delegating to the standalone index types

### Negative

- **Brute-force memory implementations** — O(N) per query for all three. Production scale requires ANN (HNSW/PQ for Vector), BM25/trigram for Search, or R-tree for Spatial. These will be added as separate engine modules.
- **No SQL engine support yet** — SQLite and DuckDB engines don't implement VectorBackend/SearchBackend/SpatialBackend. This is by design — these backends require specialized index structures that don't map naturally to SQL.

### Neutral

- **ADT classification priority changed** — Vector/Search/Spatial are now checked first in `classifyADT()`. This is safe because the return types (`Embedding`, `IndexedText`, `Point`) are distinct sentinel types that cannot be confused with Map/Set/Counter return types.

## Usage Example

```go
// Vector similarity search — RAG/semantic search
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine()},
    metaengine.Query[SemanticSearch, metaengine.VectorResult]("semantic_search",
        metaengine.On(DocEmbedded{}, func(e DocEmbedded) metaengine.Embedding {
            return metaengine.Embedding{ID: e.ID, Values: e.Values}
        }),
    ),
)
store.Apply(ctx, "DocEmbedded", DocEmbedded{ID: "doc1", Values: []float32{0.9, 0.1, 0}})
results, _ := metaengine.VectorExecuteTyped[SemanticSearch](ctx, store,
    SemanticSearch{Vector: []float32{1, 0, 0}, K: 5, Metric: "cosine"})
```
