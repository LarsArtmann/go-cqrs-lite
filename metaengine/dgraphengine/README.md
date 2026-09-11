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

## Concurrency & contention

Every SetJson mutation writes Dgraph's `dgraph.type` predicate, so ALL
parallel writers against one Alpha are conflict partners — concurrent
committers abort each other's transactions, and a schema Alter is rejected
with "Pending transactions found" while transactions are in flight. The
engine absorbs this class instead of leaking it:

- **Standalone ops** retry via `retryOnContention` (6 attempts, exponential
  backoff 15ms→240ms plus jitter). Retriable errors are transaction aborts
  and pending-transaction Alter rejections; anything else surfaces
  immediately. Construction-time schema applies (`New`, `ensureEdgeSchema`)
  retry the same way — they are idempotent.
- **`RunInTx`** serializes transactions (one active transaction per engine,
  nesting rejected). Ops inside a transaction are NOT retried in place: an
  abort surfaces to the caller, who re-runs the whole transaction — partial
  side effects are discarded atomically.

The matcher and retry schedule are pinned by `transaction_retry_test.go`
(`TestIsContentionError`, `TestRetryOnContention_*`).

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
