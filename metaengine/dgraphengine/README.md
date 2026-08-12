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

## Testing

Tests require a running Dgraph instance. Set `DGRAPH_ADDR` (default:
`localhost:9080`). Tests skip gracefully when Dgraph is unavailable.

```bash
# Start Dgraph (e.g., via Docker)
docker run -d -p 9080:9080 dgraph/dgraph:latest dgraph alpha

# Run tests
DGRAPH_ADDR=localhost:9080 go test -tags "goexperiment.jsonv2" ./...
```

Cross-engine parity is verified via `adttest.RunMatrix` against the
memory engine.
