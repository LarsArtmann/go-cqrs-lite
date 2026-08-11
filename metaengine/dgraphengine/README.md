# dgraphengine

**The only metaengine that serves GraphRAG (Graph Retrieval-Augmented Generation) from a single instance.**

GraphRAG combines full-text search with knowledge-graph traversal in one pipeline:
search for relevant entities by text, then expand context by traversing their
relationships. Every other metaengine implements either graph dispatch **or**
SearchBackend — forcing you to run two databases and glue them together.
Dgraph implements both natively, at full parity, with zero degradation.

```
┌─────────────────────────────────────────────────────────────┐
│                     GraphRAG Pipeline                        │
│                                                              │
│  SearchQuery("golang performance")                           │
│      ↓  882 µs (anyofterms + @index(term))                  │
│  [entity-42, entity-117, entity-3, ...]                      │
│      ↓                                                        │
│  GraphNeighbors("entity-42", depth=2)                        │
│      ↓  420 µs per hop (@recurse, O(degree^depth))          │
│  [entity-43, entity-44, entity-45, entity-46, ...]           │
│      ↓                                                        │
│  Context Window → LLM                                        │
│                                                              │
│  Single Dgraph instance. No vector DB. No glue code.         │
└─────────────────────────────────────────────────────────────┘
```

## GraphRAG Performance (the headline numbers)

**Concurrent stress test**: 200 entities, ~600 edges, 16 goroutines,
320 GraphRAG pipeline queries (search + depth-2 graph expansion per hit).

| Metric      | Normal    | With -race |
| ----------- | --------- | ---------- |
| Throughput  | 2,955 q/s | 1,460 q/s  |
| Latency p50 | 5.3 ms    | 10.7 ms    |
| Latency p99 | 6.5 ms    | 13.1 ms    |
| Latency max | 6.7 ms    | 13.5 ms    |

Each "query" = SearchQuery (limit 5) + 5 x GraphNeighbors (depth 2).
That's 6 gRPC round-trips per pipeline query, all completing in <7ms p99.

**Single-pipeline latency**: 2.7 ms per GraphRAG query (search + 5 graph
expansions). Full Triad (Map + Graph + Search reads): 1.0 ms.

### Why not just use two engines?

Without Dgraph, GraphRAG requires a polyglot stack:

| What you need      | Polyglot approach           | Dgraph            |
| ------------------ | --------------------------- | ----------------- |
| Full-text search   | Elasticsearch / Meilisearch | @index(term)      |
| Graph traversal    | Neo4j / Memgraph            | @recurse edges    |
| KV projections     | Redis / PostgreSQL          | @index(exact)     |
| **Engines to run** | **3 separate databases**    | **1**             |
| **Glue code**      | **join search IDs → graph** | **none**          |
| **Consistency**    | **eventual (cross-DB)**     | **transactional** |

Every additional database is a deployment, a failure mode, a consistency
boundary, and a maintenance burden. Dgraph collapses the polyglot stack.

### GraphRAG Code Example

```go
import dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"

eng, _ := dgraphengine.New("localhost:9080")
defer eng.Close()

// Dgraph implements graph dispatch + full-text search natively (ADR-0113).
type graphDispatch interface {
	GraphAddEdge(ctx context.Context, collection string, edge metaengine.Edge) error
	GraphNeighbors(ctx context.Context, collection string, node any, depth int) ([]any, error)
}

gb := eng.(graphDispatch)
sb := eng.(metaengine.SearchBackend)

// 1. INDEX: insert entities with text + relationships.
for _, e := range entities {
    sb.SearchInsert(ctx, "docs", metaengine.IndexedText{ID: e.ID, Content: e.Desc})
    gb.GraphAddEdge(ctx, "graph", metaengine.Edge{From: e.ID, To: e.RelatedID})
}

// 2. RETRIEVE: search for relevant entities.
results, _ := sb.SearchQuery(ctx, "docs", "golang performance", 5)

// 3. EXPAND: traverse graph neighborhood of each hit.
context := make(map[string]bool)
for _, r := range results {
    context[r.ID] = true
    neighbors, _ := gb.GraphNeighbors(ctx, "graph", r.ID, 2)
    for _, n := range neighbors {
        context[fmt.Sprint(n)] = true
    }
}
// 4. ASSEMBLE: feed context window to your LLM.
```

Validated by:

- `TestGraphRAG_SearchThenGraphTraverse` — correctness (8-entity graph)
- `TestGraphRAG_DifferentQueries` — multi-query (5 microservices)
- `TestGraphRAG_ConcurrentStress` — 320 concurrent queries, 16 goroutines
- `BenchmarkDgraph_GraphRAG_SearchThenExpand` — 2.7ms per pipeline query

---

## ADT Support

| ADT        | Complexity      | Degraded? | Notes                               |
| ---------- | --------------- | --------- | ----------------------------------- |
| **Graph**  | O(degree^depth) | **No**    | **Native — Dgraph's core strength** |
| **Search** | O(logN)         | **No**    | **@index(term) full-text**          |
| Map        | O(logN)         | No        | @index(exact) point lookup          |
| Counter    | O(1) incr       | No        | Batched multi-key increments        |
| Set        | O(logN)         | No        | @index(exact) membership            |
| Multimap   | O(logN)         | No        | One node per (key, value) pair      |
| Log        | O(logN)         | No        | Append-only, timestamp-ordered      |
| SortedMap  | O(N)            | Yes       | Scan + Go-side sort                 |

## Implemented Backends

- **Graph dispatch** — native graph edges with @reverse, O(degree^depth) traversal
- **`SearchBackend`** — full-text search via @index(term) + anyofterms()
- `MapBackend` — key-value storage via Dgraph nodes with @index(exact)
- `CounterBackend` — batched multi-key increments, 1 RAFT commit per Delta
- `ScanBackend` — filtered/sorted scans (Go-side filter/sort)
- `SetBackend` — set membership via @index(exact)
- `MultimapBackend` — one node per (key, value) pair with @index(exact)
- `LogBackend` — append-only nodes ordered by nanosecond timestamp

## Profile

- **Persistence**: Persistent (survives restarts)
- **Replication**: Single-leader (RAFT consensus per group)
- **NsPerOp**: 2,500,000 ns (2.5ms — gRPC + WAL fsync, calibrated)
- **NsPerRead**: 600,000 ns (600µs — read-only txn, calibrated)
- **NsPerWrite**: 2,500,000 ns (2.5ms — RAFT consensus, calibrated)

Pure Go (no CGo): uses the [dgo v240](https://github.com/dgraph-io/dgo)
gRPC client.

## Usage

```go
import dgraphengine "github.com/larsartmann/go-cqrs-lite/metaengine/dgraphengine/v4"

// Connect to a Dgraph Alpha server
eng, err := dgraphengine.New("localhost:9080")
if err != nil {
    log.Fatal(err)
}
defer eng.Close()

// Use with the metaengine planner
store, err := metaengine.Plan([]metaengine.Engine{eng},
    metaengine.Query[Input, Output]("my_query",
        metaengine.OnRecord(MyEvent{}, func(_ record.Record, e MyEvent) (string, Output) {
            return e.ID, Output{...}
        }),
    ),
)
```

## Full Performance Table

All benchmarks against Dgraph 25.4.0, single-node, localhost, `-benchtime=3s`.

### Single-ADT Operations

| Operation                | Latency | Allocs | Notes                                    |
| ------------------------ | ------- | ------ | ---------------------------------------- |
| MapSet                   | 2.7 ms  | 194    | gRPC + RAFT consensus commit             |
| MapGet                   | 344 us  | 146    | read-only txn, no RAFT                   |
| CounterIncrement         | 2.4 ms  | 288    | upsert with conditional mutation         |
| CounterGet               | 3.4 ms  | 1,143  | returns all counters in collection       |
| SetAdd                   | 2.1 ms  | 179    | upsert with @index(exact)                |
| GraphAddEdge             | 2.8 ms  | 360    | 2-step upsert: nodes + bidirectional     |
| GraphNeighbors (depth 1) | 420 us  | 193    | direct adjacency, read-only txn          |
| GraphNeighbors (depth 3) | 963 us  | 463    | @recurse multi-hop (killer feature)      |
| SearchInsert             | 2.5 ms  | 192    | upsert into @index(term) full-text index |
| SearchQuery              | 882 us  | 157    | anyofterms() over 500-doc corpus         |

### Mixed Workload Benchmarks

| Workload                          | Latency    | Allocs    | Pattern                                   |
| --------------------------------- | ---------- | --------- | ----------------------------------------- |
| **GraphRAG (search + expand)**    | **2.7 ms** | **1,098** | **search 5 docs + depth-2 graph per hit** |
| Graph Write/Read Mix              | 4.8 ms     | 1,255     | 1 edge add + 3 neighbor queries (25/75)   |
| Map Read/Write Mix                | 671 us     | 152       | 80% MapGet + 20% MapSet                   |
| Full Triad (Map + Graph + Search) | 1.0 ms     | 459       | 1 read from each backend per iteration    |

Writes are dominated by RAFT consensus (2.1-2.8 ms).
Reads bypass RAFT via `NewReadOnlyTxn()` (344 us - 963 us).

## Testing

Tests require a running Dgraph instance. Set `DGRAPH_ADDR` (default:
`localhost:9080`). Tests skip gracefully when Dgraph is unavailable.

```bash
# Start ephemeral Dgraph (Zero + Alpha from nixpkgs, no Docker)
nix run .#ephemeral-dgraph -- go test -tags "goexperiment.jsonv2" -v ./...

# Run just the GraphRAG stress test:
nix run .#ephemeral-dgraph -- go test -tags "goexperiment.jsonv2" -v \
    -run TestGraphRAG_ConcurrentStress -race ./...
```

### Test inventory

| File                  | Tests                                    | What it validates                    |
| --------------------- | ---------------------------------------- | ------------------------------------ |
| `graphrag_test.go`    | 2 GraphRAG functional tests              | Search + Graph pipeline correctness  |
| `stress_test.go`      | Concurrent stress (16 goroutines, 320 q) | Throughput, p50/p99, race-safety     |
| `bench_test.go`       | 10 single-ADT benchmarks                 | Per-operation cost calibration       |
| `mixed_bench_test.go` | 4 mixed-workload benchmarks              | Real-world combined-ADT performance  |
| `adt_matrix_test.go`  | Cross-engine parity (6 ADTs vs Memory)   | No semantic drift from Memory engine |
| `injection_test.go`   | DQL injection source-code scan           | No string-interpolated DQL           |

### Dgraph 25.x delete behavior

Dgraph 25.x does **not** delete all predicates when `DeleteJson` contains
only `{"uid": "..."}`. Each predicate must be explicitly set to `null`.
This engine's `MapDelete` handles this correctly via the upsert pattern
with explicit null-predicate deletion.
