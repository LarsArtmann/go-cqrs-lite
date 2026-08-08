# dgraphengine

Dgraph-backed [metaengine](../) Engine for go-cqrs-lite.

## Why Dgraph?

Dgraph is a distributed graph database with native graph traversal and
full-text search. This engine makes **GraphBackend** and **SearchBackend**
first-class citizens — no degradation, no emulation.

| ADT       | Complexity      | Degraded? | Notes                        |
| --------- | --------------- | --------- | ---------------------------- |
| Map       | O(logN)         | No        | @index(exact) point lookup   |
| Counter   | O(1)            | No        | Atomic read-modify-write     |
| Graph     | O(degree^depth) | **No**    | **Dgraph's native strength** |
| Set       | O(logN)         | No        | @index(exact) membership     |
| SortedMap | O(N)            | Yes       | Scan + Go-side sort          |
| Search    | O(logN)         | **No**    | **@index(term) full-text**   |

## Implemented Backends

- `MapBackend` — key-value storage via Dgraph nodes with @index(exact)
- `CounterBackend` — atomic counters via transactional read-modify-write
- `ScanBackend` — filtered/sorted scans (Go-side filter/sort)
- `GraphBackend` — **native graph edges with @reverse, O(degree^depth) traversal**
- `SetBackend` — set membership via @index(exact)
- `SearchBackend` — **full-text search via @index(term) + anyofterms()**

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
        metaengine.On(MyEvent{}, func(e MyEvent) (string, Output) {
            return e.ID, Output{...}
        }),
    ),
)
```

## Profile

- **Persistence**: Persistent (survives restarts)
- **Replication**: Single-leader (RAFT consensus per group)
- **NsPerOp**: 10,000 ns (gRPC + WAL fsync)
- **NsPerRead**: 8,000 ns (gRPC + index lookup)

Pure Go (no CGo): uses the [dgo v240](https://github.com/dgraph-io/dgo)
gRPC client.

## Performance (benchmark against Dgraph 25.4.0, single-node, localhost, `-benchtime=3s`)

| Operation                 | Latency   | Allocs   | Notes                                    |
| ------------------------- | --------- | -------- | ---------------------------------------- |
| MapSet                    | 2.7 ms    | 194      | gRPC + RAFT consensus commit             |
| MapGet                    | 344 us    | 146      | read-only txn, no RAFT                   |
| CounterIncrement          | 2.4 ms    | 288      | upsert with conditional mutation         |
| CounterGet                | 3.4 ms    | 1,143    | returns all counters in collection       |
| SetAdd                    | 2.1 ms    | 179      | upsert with @index(exact)                |
| GraphAddEdge              | 2.8 ms    | 360      | 2-step upsert: nodes + bidirectional     |
| GraphNeighbors (depth 1)  | 420 us    | 193      | direct adjacency, read-only txn          |
| GraphNeighbors (depth 3)  | 963 us    | 463      | @recurse multi-hop (killer feature)      |
| SearchInsert              | 2.5 ms    | 192      | upsert into @index(term) full-text index |
| SearchQuery               | 882 us    | 157      | anyofterms() over 500-doc corpus         |

Writes are dominated by RAFT consensus (leader proposes, followers ack):
MapSet, GraphAddEdge, and SearchInsert all land at 2.1-2.8 ms.
Reads bypass RAFT via `NewReadOnlyTxn()`, so they are 3-7x faster:
MapGet (344 us), GraphNeighbors depth-1 (420 us), SearchQuery (882 us).

Depth-3 graph traversal (@recurse) at 963 us is Dgraph's killer feature —
the same query in SQL would require recursive CTEs. The 100-node graph
with 300 edges reaches 39+ nodes at depth 3.

### Mixed Workload Performance

These benchmarks combine multiple backends per iteration — the patterns real
consumers use. They answer: "Can one Dgraph instance serve a polyglot workload?"

| Workload                       | Latency   | Allocs | Pattern                                    |
| ------------------------------ | --------- | ------ | ------------------------------------------ |
| GraphRAG (search + expand)     | 2.7 ms    | 1,098  | search 5 docs + depth-2 graph per hit      |
| Graph Write/Read Mix           | 4.8 ms    | 1,255  | 1 edge add + 3 neighbor queries (25/75)    |
| Map Read/Write Mix             | 671 us    | 152    | 80% MapGet + 20% MapSet (read-heavy)       |
| Full Triad (Map + Graph + Search) | 1.0 ms | 459   | 1 read from each backend per iteration     |

**GraphRAG pipeline** at 2.7 ms: full-text search (882 us) + 5 graph depth-2
traversals (~360 us each) = the end-to-end retrieval latency an LLM agent
experiences per query. No separate vector DB + graph DB needed.

**Full Triad** at 1.0 ms: MapGet + SearchQuery + GraphNeighbors in one
iteration — all reads bypass RAFT via `NewReadOnlyTxn()`. A single Dgraph
instance serves all three ADT needs.

## GraphRAG: The Killer Use Case

GraphRAG (Graph Retrieval-Augmented Generation) is the pattern where Dgraph's
dual GraphBackend + SearchBackend capability becomes a unique differentiator.
No other metaengine implements both at full parity.

**Pipeline:**

1. **Index** — insert entities with text descriptions (SearchInsert) AND build
   a knowledge graph connecting them (GraphAddEdge).
2. **Retrieve** — search for entities matching a text query (SearchQuery).
3. **Expand** — traverse the graph neighborhood of each hit (GraphNeighbors).
4. **Assemble** — deduplicate into a context window for an LLM.

```go
eng, _ := dgraphengine.New("localhost:9080")
gb := eng.(metaengine.GraphBackend)
sb := eng.(metaengine.SearchBackend)

// 1. Search for relevant entities.
results, _ := sb.SearchQuery(ctx, "knowledge-base", "golang performance", 5)

// 2. Expand each hit via 2-hop graph traversal.
context := make(map[string]bool)
for _, r := range results {
    context[r.ID] = true
    neighbors, _ := gb.GraphNeighbors(ctx, "entity-graph", r.ID, 2)
    for _, n := range neighbors {
        context[fmt.Sprint(n)] = true
    }
}
// 3. Feed context to your LLM.
```

On Memory/SQLite/Pebble, you'd need two separate engines and glue code to
join search results with graph relationships. Dgraph handles both natively.

Validated by `TestGraphRAG_SearchThenGraphTraverse` and
`TestGraphRAG_DifferentQueries` (see `graphrag_test.go`).

## Testing

Tests require a running Dgraph instance. Set `DGRAPH_ADDR` (default:
`localhost:9080`). Tests skip gracefully when Dgraph is unavailable.

The recommended way to test is with the ephemeral Dgraph script:

```bash
# Start ephemeral Dgraph (Zero + Alpha from nixpkgs, no Docker)
nix run .#ephemeral-dgraph -- go test -tags "goexperiment.jsonv2" -v ./...

# Or start manually:
dgraph zero --my=localhost:5080 &
dgraph alpha --zero=localhost:5080 &
DGRAPH_ADDR=localhost:9080 go test -tags "goexperiment.jsonv2" ./...
```

Cross-engine parity is verified via `adttest.RunMatrix` against the
memory engine. All 6 implemented ADTs (Map, Set, Counter, Graph, Search,
SortedMap) pass at full parity.

**GraphRAG integration tests** (`graphrag_test.go`) validate the combined
SearchBackend + GraphBackend pipeline — the pattern no other engine can serve
in a single instance.

**Mixed workload benchmarks** (`mixed_bench_test.go`) measure combined
read/write and cross-ADT performance patterns.

### Dgraph 25.x delete behavior

Dgraph 25.x does **not** delete all predicates when `DeleteJson` contains
only `{"uid": "..."}`. Each predicate must be explicitly set to `null`.
This engine's `MapDelete` handles this correctly via the upsert pattern
with explicit null-predicate deletion.
