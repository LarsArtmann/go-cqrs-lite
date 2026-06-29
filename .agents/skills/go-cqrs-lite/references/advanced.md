## 6. Advanced Patterns

### 6.1 Tombstone Soft-Delete & Rebirth

```go
// Delete: emit a tombstone event
marked, _ := event.MarkTombstone(evt)
store.Save(ctx, ref, []event.Event{marked}, expectedVersion)

// Detect: check aggregate status
status := event.DetectTombstone(events) // Active | Tombstoned | Undetermined

// Rebirth: emit a new event after tombstone (tombstone is just metadata)
// See example/user/ for the full tombstone + rebirth cycle
```

### 6.2 Command & Query Persistence (audit trail)

```go
// Persist commands for audit/replay
cmd, _ := command.NewPersistedCommand("user.create", ref, payload)
cmdStore.Save(ctx, ref, cmd)              // CommandSink
cmds, _ := cmdStore.Load(ctx, ref)        // CommandSource
var journal command.CommandJournal = cmdStore        // ReadAll (global)
var seekable command.SeekableCommandJournal = cmdStore // ReadFrom(afterCmdID, limit)

// Persist queries
pq, _ := query.NewPersistedQuery("user.get", payload)
qStore.SaveQuery(ctx, pq)                 // QuerySink
queries, _ := qStore.LoadQueries(ctx, after) // QuerySource
```

### 6.3 Aggregate Listing (read model for all aggregates)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/listing/v3"
    "github.com/larsartmann/go-cqrs-lite/storage/v3"
)

// In-memory reader (consumes a Journal to track aggregate statuses)
reader := listing.NewInMemoryAggregateReader(journal)
builder := listing.NewListBuilder(reader)

// For SQL-backed listing, register the projection with the runner:
// proj, _ := storage.NewAggregateProjection(ctx, db, "aggregate_listing", dialect)
// runner.Register(proj)

// reader.List() → []AggregateListing with Status: Active | Tombstoned
```

### 6.4 Watermill Integration

```go
// Bridge go-cqrs-lite events to a Watermill router
publisher := watermill.NewPublisherAdapter(bus)      // wraps event.Publisher
subscriber := watermill.NewSubscriberAdapter(bus)     // wraps event.Bus
messages, _ := subscriber.Subscribe(ctx, "user.created")
// Use with standard Watermill handler funcs
```

### 6.5 Turso Offline-First

```go
import "github.com/larsartmann/go-cqrs-lite/storage/turso/v3"

// Offline-first: local embedded LibSQL with background sync to Turso cloud
db, _ := turso.OpenSync(ctx, "file:local.db", "libsql://my-db.turso.io", authToken)
backend, _ := turso.NewBackend(db)
// Or without sync: db, _ := turso.Open("file:local.db")
```

### 6.6 Pebble as KV Store

```go
import (
    "github.com/cockroachdb/pebble"
    cqrspebble "github.com/larsartmann/go-cqrs-lite/storage/pebble/v3"
)

db, _ := pebble.Open(dir, &pebble.Options{})               // raw cockroachdb/pebble
kvStore, _ := cqrspebble.NewKVStore(db, cqrspebble.WithSyncWrites())
defer kvStore.Close()
kvStore.Set([]byte("k"), []byte("v"))
val, _ := kvStore.Get([]byte("k"))
```

### 6.7 Code Generation (cqrs-gen)

```bash
go install github.com/larsartmann/go-cqrs-lite/cmd/cqrs-gen/v3@latest
```

Add markers to your types:

```go
//cqrs:command
type CreateUser struct { Name string }

//cqrs:query
type GetUser struct { ID string }
```

Run:

```bash
cqrs-gen -type . -output handlers_gen.go -pkg myapp
```

Generates typed `Register*` boilerplate.

### 6.8 gRPC Transport (remote command/query dispatch)

Expose local dispatchers over gRPC, or dispatch to a remote CQRS server.

```go
import (
    "google.golang.org/grpc"
    cqrsgrpc "github.com/larsartmann/go-cqrs-lite/transport/grpc/v3"
)

// --- Server side: expose your dispatchers over gRPC ---

srv := grpc.NewServer()
cqrsgrpc.RegisterCommandService(srv, cmdDispatcher) // cmdDispatcher: *command.Dispatcher
cqrsgrpc.RegisterQueryService(srv, qDispatcher)     // qDispatcher: *query.Dispatcher

lis, _ := net.Listen("tcp", ":50051")
go srv.Serve(lis)

// --- Client side: dispatch to a remote server ---

conn, _ := grpc.NewClient(
    "localhost:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()), // or TLS
)
defer conn.Close()

// Command dispatch — transparent remote call
cmdClient := cqrsgrpc.NewCommandClient(conn)
err := cmdClient.Dispatch(ctx, myCommand) // same interface as local dispatcher

// Query dispatch — JSON result unmarshaled into your struct
qClient := cqrsgrpc.NewQueryClient(conn)
var result GetUserResult
err := qClient.Ask(ctx, "user.get", &result) // queryType + out pointer
```

Command payloads are carried in metadata (`metadata.Custom["payload"]`); handlers
extract them via `cmd.Metadata().Custom["payload"]`. Query results are JSON-encoded
on the wire. The `CommandClient` implements the same `Dispatch` interface as a local
dispatcher — swap them freely.

### 6.9 Managed Projection Host (crash-restart + checkpoint + DLQ)

The "last loop every consumer rewrites", now a library module. Composes any
`event.SeekableJournal` + `event.CheckpointStore` + your `projection.Projection`s
into a managed lifecycle with per-projection goroutines, exponential-backoff
restarts, persisted checkpoints, and a poison-message dead-letter queue.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/projection/v3"
    "github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
)

// journal: any event.SeekableJournal (MemoryStore, SQLEventStore, pebble.EventStore, ...)
// cpStore: event.CheckpointStore (memory.MemoryCheckpointStore, SQLCheckpointStore, ...)
host, _ := projectionhost.New(journal, cpStore,
    projectionhost.WithBatchSize(100),
    projectionhost.WithDeadLetterStore(projectionhost.NewMemoryDeadLetterStore(), 3), // poison after 3 retries
)
_ = host.Register(&UserProjection{})   // Register returns error; Name() must be unique
_ = host.Register(&OrderProjection{})

go host.Start(ctx)    // one goroutine per projection; crash auto-restart + exponential backoff
defer host.Stop()     // graceful drain (30s timeout)

for _, s := range host.Status() {     // health snapshot per worker
    fmt.Printf("%s: %s processed=%d errors=%d restarts=%d\n",
        s.Name, s.Status, s.Processed, s.Errors, s.Restarts)
}
// Worker states: idle, running, backoff, draining, stopped, failed.
// Reads directly from event.SeekableJournal — NO message-bus dependency.
// For live (push) delivery alongside replay, pair with watermill/CatchUpSubscriber.
```

### 6.10 Scenario-Testing DSL (Given/When/Then)

Fluent BDD for deciders and projections — no store or bus needed, just pure functions.

```go
import "github.com/larsartmann/go-cqrs-lite/scenario/v3"

// Decider: pure fold + pure decide
scenario.Given[incrementCmd, counterState](t, foldCounter, counterState{},
    mustEvent(evtIncremented)).            // pre-existing events folded into state
    When(incrementCmd{}, decideIncrement).  // pure decide function
    Then(evtIncremented)                    // asserts emitted event TYPES
// Variants:
//   .ThenError(target)                 // asserts decide returns an error wrapping target
//   .ThenState(fold, initial, expected)// folds produced events, asserts final state

// Projection: feed events, assert no error
scenario.GivenProjection(t, &UserProjection{}, evt1, evt2).ThenNoError()
scenario.GivenProjection(t, &BrokenProj{}, badEvt).ThenError() // expects >= 1 error
```

### 6.11 Scheduled Commands / Durable Deadlines

Classic ES need — "cancel the order 30 minutes after creation if still unpaid" — as a
library primitive. `TimerStore` persists timers across restarts; `Scheduler` polls and
dispatches. Scheduling is idempotent (same `TimerID` is a no-op), so it is safe to
re-schedule on command retry.

```go
import "github.com/larsartmann/go-cqrs-lite/scheduling/v3"

store := scheduling.NewMemoryTimerStore()
sched := scheduling.New(store, func(ctx context.Context, t scheduling.Timer) error {
    return cmdDispatcher.Dispatch(ctx, t.Payload.(CancelOrderCmd))
},
    scheduling.WithPollInterval(500*time.Millisecond),
    scheduling.WithMaxRetries(5),
)

_ = store.Schedule(ctx, scheduling.Timer{
    ID:      "order-123-timeout",
    FireAt:  time.Now().Add(30 * time.Minute),
    Payload: CancelOrderCmd{OrderID: "123"},
})
_ = store.Cancel(ctx, "order-123-timeout") // order paid → cancel the timeout

go sched.Start(ctx) // polls Due(), dispatches via callback, MarkFired(); retries failures
```

### 6.12 Reactive Command Derivation (deriver)

Derive commands from events — the reactive link that replaces hand-rolled
sagas. A `Deriver` is a pure function: event → zero or more commands. Derivers
compose with `.Then()` (fan-out) and `.Filter()` (event-type matching), and
wire into the event bus as a catch-all subscriber. Safe for at-least-once
delivery: the same event always produces the same commands, so idempotency at
the command handler deduplicates replays.

```go
import "github.com/larsartmann/go-cqrs-lite/deriver/v3"

// A deriver: user.created → send welcome email + sync to CRM
sendWelcomeEmail := deriver.Deriver(
    func(_ context.Context, evt event.Event) ([]command.Command, error) {
        cmd, err := command.New("email.send_welcome", evt.AggregateID())
        if err != nil {
            return nil, err
        }
        return []command.Command{cmd}, nil
    },
)
syncToCrm := deriver.Deriver(
    func(_ context.Context, evt event.Event) ([]command.Command, error) {
        cmd, err := command.New("crm.upsert_user", evt.AggregateID())
        if err != nil {
            return nil, err
        }
        return []command.Command{cmd}, nil
    },
)

// Compose (fan-out) + filter (only user.created) + wire into the bus
composed := sendWelcomeEmail.Then(syncToCrm).Filter("user.created")
bus.SubscribeAll(composed.AsHandler(cmdDispatcher))
// ADR-0040: functional/composable API over a declarative rule registry.
```

### 6.13 Graph Projections (graph)

The third projection tier. Where `stack.Materialize` writes one document per
key and `storage.RelationalProjection` writes across SQL tables, `graph`
merges events into **nodes and edges** — the right shape for variable-depth
traversal, path-finding, adjacency, and connected-component queries (reply
chains, social graphs, causation DAGs, role memberships).

Writes ARE portable across backends (openCypher MERGE semantics shared by
Neo4j, Memgraph, Apache Age, RedisGraph). Reads run native Cypher/Gremlin via
the driver (only `MemoryDriver` offers a Go-native read API).

```go
import "github.com/larsartmann/go-cqrs-lite/graph/v3"

driver := graph.NewMemoryDriver()
proj, _ := graph.NewGraphProjection("discord-graph", driver,
    func(ctx context.Context, evt event.Event, sink graph.GraphSink) error {
        var p MessageCreated
        _ = json.Unmarshal(evt.Payload(), &p)
        msgRef := graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ID}
        sink.MergeNode(msgRef, map[string]any{"created_at": p.CreatedAt})
        // Auto-creates endpoint nodes — handlers need not pre-merge.
        sink.MergeEdge(graph.EdgeRef{Type: "AUTHORED_BY", From: msgRef,
            To: graph.NodeRef{Label: "User", KeyProp: "id", KeyValue: p.AuthorID}}, nil)
        // The recursive edge — relational tier needs WITH RECURSIVE CTE.
        if p.ReplyToMessageID != "" {
            sink.MergeEdge(graph.EdgeRef{Type: "REPLY_TO", From: msgRef,
                To: graph.NodeRef{Label: "Message", KeyProp: "id", KeyValue: p.ReplyToMessageID}},
                map[string]any{"at": p.CreatedAt})
        }
        return nil // atomic: all merges commit or all roll back
    }, []event.Type{"MESSAGE_CREATED"})
// proj implements projection.Projection → register with projectionhost or any runner.

// Schema validation (opt-in, ADR-0039) — catch typos at the sink boundary:
schema := &graph.Schema{
    Nodes: []graph.NodeType{{Label: "User", KeyProp: "id", Properties: []graph.PropertyType{{Name: "name"}}}},
    Edges: []graph.EdgeType{{Type: "AUTHORED_BY", FromLabel: "Message", ToLabel: "User"}},
}
proj, _ = graph.NewGraphProjection("graph", driver, handler, types, graph.WithSchema(schema))

// Read API (MemoryDriver only — Go-native predicates, NOT a query language)
ancestors := driver.Traverse(msgRef, "REPLY_TO", -1)   // BFS unlimited depth
neighbors, edges := driver.Neighbors(centerRef)          // 1-hop adjacency
path, _ := driver.ShortestPath(userA, userB)             // BFS shortest path
```

### 6.14 Prometheus Metrics Export (prometheus)

OTel→Prometheus bridge: expose all CQRS metrics (command/event/query) at the
standard `/metrics` endpoint. Wraps the OTel Prometheus exporter so any
`middleware.NewOTelMetricsRecorder` instruments are automatically exposed.

```go
import (
    "github.com/larsartmann/go-cqrs-lite/prometheus/v3"
    "go.opentelemetry.io/otel"
)

provider, handler, _ := prometheus.Setup() // one call: MeterProvider + /metrics handler
defer provider.Shutdown(context.Background())
otel.SetMeterProvider(provider.AsMeterProvider())

// Expose /metrics for Prometheus scraping
mux.Handle("/metrics", handler)

// Custom registry (multi-tenant / shared)
provider, handler, _ = prometheus.Setup(prometheus.WithRegistry(myRegistry))
```
