# Metaengine Cookbook

Practical recipes for common metaengine patterns. Each recipe is copy-paste
ready and includes the query declaration, fold handlers, and reader usage.

## Payload Encodings

Events reach the store in one of two ways:

```go
// 1. Decoded value — the caller already knows the type.
_ = store.Apply(ctx, "task.created", TaskCreated{ID: "t1"})

// 2. Encoded bytes — the store decodes per fold sample.
_ = store.ApplyEncoded(ctx, "task.created", evt.Payload())
```

`ApplyEncoded` / `ApplyEncodedRecord` accept **JSON and CBOR**. The codec is
taken from `Record.Encoding` when known and detected otherwise, and each fold
decodes the bytes with the codec that produced them; a JSON normalization step
would lose values that CBOR represents differently (for example `time.Time`,
which canonical CBOR writes as an epoch integer). See
[ADR-0161](../docs/adr/0161-encoded-ingress-producer-codec.md).

Because `event.New` encodes CBOR by default, path 2 is the one to use when
bridging a raw `event.Event` into a projection. Decode failures classify as
Corruption, so a projection host with a dead-letter store quarantines poison
payloads instead of retrying them forever.

### Stored values (map and stream-log rows)

Engine rows hold `any` values that `MapBackend`, `LogBackend`, and
`StreamLogBackend` serialize as text. The stored representation is
**JSON-typed**:

- strings are stored verbatim and read back verbatim;
- every other value is JSON-encoded, so integers read back as `float64`,
  structs as `map[string]any`, and `time.Time` as an RFC 3339 string.

One consequence is not escapable at the read side: a **string that is itself
valid JSON** (`{"a":1}`, `123`, `true`, `null`) is stored verbatim and reads
back as the PARSED value, because the row carries no type tag.
`DecodeStreamValue` therefore keeps its raw-string fallback instead of growing
an error-returning twin — it cannot tell a plain string from corruption. Store
values in a JSON-native shape when the Go type must survive, or wait for the
stamped engine-value envelope (the ADR-0066 reversal tracked as T16 in
[`docs/planning/2026-10-24_05-33_SUPERB-metaengine-codec-boundary-completion.md`](../docs/planning/2026-10-24_05-33_SUPERB-metaengine-codec-boundary-completion.md)).

SQL engine write paths (`sqliteengine`/`turso`, `pgengine`, `duckdbengine`,
`mysqlengine`, `dgraphengine`) reject a value JSON cannot represent (channel,
function, cyclic structure) with a Rejection-classified error **before** any row
is written. `sqliteengine` (and `turso`) previously stored a
`fmt.Sprintf("%v")` placeholder; that fallback is gone there.

The LSM engines (`pebbleengine`, `bboltengine`, `badgerengine`) reject too:
their 21 production value write sites encode through `keycodec.EncodeJSONErr`
(Rejection code `keycodec.encode_value`), and the cross-engine contract is
pinned by `adttest.AssertValueEncodeRejection`. `keycodec.EncodeJSON` keeps its
`fmt.Sprintf("%v")` fallback for KEY and display-shaped paths, where a
placeholder is identity rather than data. The `memory` engine stores Go values
as-is, so it cannot lose them.

**Closed (2026-10-25):** `bigtableengine` now rejects too — its cell-value
writer encodes through `keycodec.EncodeJSONErr` (code `keycodec.encode_value`,
pinned by the same `adttest.AssertValueEncodeRejection` harness against the
in-process `bttest` fake). The graph encoders (`mysqlengine`, `duckdbengine`,
`pgengine`, `badgerengine`) were VERIFIED value-free: graph tables/keys carry
only `(collection, from_node, to_node)` identity — their `encodeJSONString` /
`graphNodeKeyJSON` helpers stringify node IDENTITY, the same documented
lenient contract as the index-key helpers below, so there is no value path to
migrate. Index-key helpers (`sqliteengine.encodeKey`, `pebbleengine`'s
planned-layout `indexKeyValue`) are identity, not payload, and keep the `%v`
form deliberately.

**Where the bytes are pinned** (literal expectations read back through raw
engine reads — copy these when adding an engine or a write path):
`sqliteengine/stored_bytes_pin_test.go` +
`sqliteengine/stored_bytes_write_paths_pin_test.go` (planned table + MapUpdate
re-encode), and `stored_bytes_pin_test.go` in `pebbleengine` / `bboltengine` /
`badgerengine` (LSM: `RawValueReader`/`FromDB` bucket/txn reads of the
`keycodec` key layout). Mutation proofs for the encoder seams live with the
session receipts (`docs/status/2026-10-25_14-06`, plan §11).

## Counter Patterns

### Status Count Dashboard

Track counts by status for a dashboard endpoint. O(1) aggregate reads.

```go
type countInput struct{}

statusCounts := metaengine.Query[countInput, map[string]int64]("status_counts",
    metaengine.On(itemCreated{}, func(e itemCreated) metaengine.Delta {
        return metaengine.Delta{"open": +1}
    }),
    metaengine.On(itemCompleted{}, func(e itemCompleted) metaengine.Delta {
        return metaengine.Delta{"open": -1, "completed": +1}
    }),
    metaengine.On(itemDeleted{}, func(e itemDeleted) metaengine.Delta {
        return metaengine.Delta{"completed": -1}
    }),
)

// Read:
counts, _ := metaengine.ExecuteTyped[countInput, map[string]int64](
    ctx, store, countInput{})
// counts["open"], counts["completed"]
```

### Rate Limiter / Dedup Counter

Track request counts per user for rate limiting.

```go
type rateInput struct{ UserID string }

rateLimit := metaengine.Query[rateInput, map[string]int64]("rate_limit",
    metaengine.On(requestEvent{}, func(e requestEvent) metaengine.Delta {
        return metaengine.Delta{e.UserID: +1}
    }),
)
```

## Map Patterns

### CRUD Projection (Filtered + Sorted Scan)

The most common pattern: a read model with filterable, sortable fields.

```go
type listInput struct{ Status string }
type itemView struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    Status   string `json:"status"`
    Priority int    `json:"priority"`
}

items := metaengine.Query[listInput, itemView]("items",
    metaengine.On(createdEvt{}, func(e createdEvt) (string, itemView) {
        return e.ID, itemView{ID: e.ID, Title: e.Title, Status: "open"}
    }),
    metaengine.On(updatedEvt{}, func(e updatedEvt, prev itemView) itemView {
        prev.Title = e.Title
        return prev
    }),
    metaengine.On(deletedEvt{}, metaengine.Remove[itemView]()),
    metaengine.FilterOnField[itemView]("status", metaengine.FilterEq),
    metaengine.SortOnField[itemView]("priority", true),
)

// Reader:
reader := metaengine.NewReader[itemView](store, "items")
openItems, _ := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(50))

// Point lookup:
item, found, _ := reader.Get(ctx, "item-123")
```

### EventDecoder with eventWithID Wrapper

For Map queries keyed by entity ID, the fold handler needs the stream ID.
Use EventDecoder to bridge event-sourced stream IDs to Map keys. The key is
the BARE stream value (`StreamID.Get()` — what commands dispatch on), not
the `StreamID.String()` display form:

```go
// skip-validate
type eventWithID[P any] struct {
    ID      string
    Payload P
}

func myDecoder(evt event.Event) (any, error) {
    id := evt.StreamID().Get()
    switch evt.Type() {
    case "created":
        var p createdPayload
        _ = json.Unmarshal(evt.Payload(), &p)
        return eventWithID[createdPayload]{ID: id, Payload: p}, nil
    default:
        return nil, fmt.Errorf("no fold for %q", evt.Type())
    }
}

adapter := projectionadapter.New("items", store, nil,
    projectionadapter.WithEventDecoder(myDecoder))
```

### Bounded Collections: Keep the Newest N Rows (`WithCap`)

Transcripts, audit tails, faxes, and token tables all need the same rule:
"after this append, at most N rows survive, oldest expunged first". Declare it
once on the query and let the store do the trimming:

```go
type transcriptSegment struct {
    ID        string    `json:"id"`
    CallID    string    `json:"call_id"`
    CreatedAt time.Time `json:"created_at"`
}

segments := metaengine.Query[struct{}, transcriptSegment]("segments",
    metaengine.OnRecord(segmentAdded{}, func(_ record.Record, e segmentAdded) (string, transcriptSegment) {
        return e.ID, transcriptSegment{ID: e.ID, CallID: e.CallID, CreatedAt: e.CreatedAt}
    }),
    // Keep the newest 5,000 rows by `created_at` (descending order keeps the
    // largest timestamp values; everything older is trimmed).
    metaengine.WithCap(5_000, "created_at"),
)
```

Enforce the cap after a batch append or on a timer — not on every insert:

```go
// All capped collections in one call (query names sorted; per-query errors joined):
removed, err := store.EnforceCaps(ctx)
if err != nil {
    log.Printf("cap enforcement removed %d rows with errors: %v", removed, err)
}

// Or one collection:
removed, err = store.EnforceCap(ctx, "segments")
```

How it works and what to know:

- `WithCap` is **declaration only**; `EnforceCap`/`EnforceCaps` is the trim.
  Enforcement is explicit because a per-insert trim would scan on every write
  (memory and KV engines materialize the whole collection to sort it).
- Enforcement runs under the query's fold lock, then reads the newest `N` rows
  (the keep page) and deletes every older page in batches of 500. No converging
  loop is needed: descending order makes the keep set a prefix.
- Requirements: a Map projection (insert fold) whose result type carries the
  query's key type, and a plain sortable column name.
- **Replay caveat:** trimming deletes projection rows outside the event log.
  `Reset` + replay restores the full, uncapped collection, so run
  `EnforceCaps` again after a rebuild. Treat it as compaction, not as a domain
  event.
- A write that lands after a trim simply makes the next pass remove one more
  row; concurrent runs converge.

### Seeing Scan Truncation Instead of Guessing (`WithExact`)

`Scan` returns at most 100 rows without a limit, and the result shape cannot
tell "exactly these rows" from "the first N of many". Ask for exactness when
the difference matters:

```go
rows, err := reader.Scan(ctx, metaengine.WithLimit(500), metaengine.WithExact())
if errors.Is(err, metaengine.ErrScanTruncated) {
    // There are more than 500 matching rows. Either raise the limit,
    // paginate with ScanPage, or scan unbounded with WithLimit(0).
    return fmt.Errorf("result set too large: %w", err)
}
```

`WithExact` composes with `WithLimit`: it answers "was this the whole result?",
not "return more rows". With `WithLimit(0)` (unbounded) it can never fail. Use
`ScanPage` when you want the continuation cursor instead of an error.

## Multi-Query Patterns

### Fan-Out (Multiple Read Patterns on Same Data)

Declare multiple queries on the same event types. The planner assigns each
to the best engine:

```go
store, _ := metaengine.Plan(
    []metaengine.Engine{metaengine.NewMemoryEngine(), sqliteEng},
    statusCounts,  // Counter → Memory (O(1) reads)
    items,         // Map with FilterOnField → SQLite (pushdown scans)
)
```

### Cursor Pagination

```go
page1, _ := reader.Scan(ctx,
    metaengine.WithFilter("status", metaengine.FilterEq, "open"),
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(20))

// Use ScanPage for cursor-based pagination:
page1, cursor1, _ := reader.ScanPage(ctx,
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(20))

page2, _, _ := reader.ScanPage(ctx,
    metaengine.WithSort("priority", true),
    metaengine.WithLimit(20),
    metaengine.WithCursor(cursor1.Value))
```

## Watcher Patterns

### Reactive Map Updates

Watch a Map query for live updates. This is the read-model push path that backs
`ServeSSE`:

```go
watcher := metaengine.NewWatcher[itemView](store, "items")
ch := watcher.Watch(ctx, nil) // nil = all keys

for v := range ch {
    if v.ID == "" {
        // Delete notification: Remove[itemView]() delivers the zero value.
        log.Printf("item %s deleted", lastKey)
        continue
    }
    log.Printf("item updated: %+v", v)
}
```

### Delete Notifications

A `Remove[V]()` fold emits a watcher value with the zero value of `V`. Do not
interpret "no value" as "deleted" — the zero value is the delete signal. Use
an explicit tombstone field or compare against the zero value to distinguish
updates from deletions:

```go
if v == (itemView{}) {
    // handle delete
}
```

### Cross-Engine Watcher Semantics

The watcher pipeline reifies engine-specific value representations back to the
declared type `V`:

- **Memory engine:** returns typed Go values directly (fast path, no alloc).
- **SQLite/Postgres/DuckDB:** stored JSON decodes to `map[string]any` (or to
  raw `jsonValue` for pushdown paths). The watcher JSON round-trips to `V`
  transparently.

The same watcher consumer works unchanged across engines.

## Engine Selection Patterns

### When to Use Each Engine

| Engine   | Best For                                       | Persistence        | Filter Pushdown                 |
| -------- | ---------------------------------------------- | ------------------ | ------------------------------- |
| Memory   | Counters, small datasets, testing              | Volatile           | O(N) Go-side                    |
| SQLite   | Filtered scans, point lookups with persistence | Persistent         | json_extract WHERE/ORDER BY     |
| Pebble   | Ultra-fast point lookups (LSM)                 | Dynamic (dir/mem)  | Raw value scan + closure filter |
| DuckDB   | Analytical queries, GROUP BY aggregations      | Dynamic (file/mem) | json_extract WHERE/ORDER BY     |
| Postgres | Remote persistent storage, JSONB + B-tree      | Persistent         | JSONB operator WHERE/ORDER BY   |

> **Persistence** is now a first-class type on `EngineProfile` (ADR-0098).
> The planner emits WARN when a query routes to a volatile engine with no
> persistent alternative, and INFO when a persistent alternative exists.

### TieredStore (Read Replicas)

Fan out writes to multiple stores, read from primary:

```go
primary, _ := metaengine.Plan([]metaengine.Engine{sqliteEng}, query)
replica, _ := metaengine.Plan([]metaengine.Engine{memEng}, query)
tiered := metaengine.NewTieredStore(primary, replica)

// Writes go to both:
tiered.Apply(ctx, "created", payload)
// Reads use primary:
reader := metaengine.NewReader[itemView](primary, "items")
```

### SwapEngine (Zero-Downtime Upgrade)

Replace an engine at runtime without stopping the service:

```go
// Warm up a new SQLite engine with data, then swap:
err := store.SwapEngine("memory", "sqlite", sqliteEng)
// All queries previously on "memory" now use "sqlite"
```

### QueryBuilder (Fluent API)

```go
qb := metaengine.NewQueryBuilder[itemView](reader)
results, _ := qb.
    Where("status", metaengine.FilterEq, "open").
    OrderBy("priority", true).
    Limit(50).
    Execute(ctx)
```

## Graph Patterns

Relations as events, traversals as queries. Fold follow/unfollow facts into
`metaengine.Edge` records; the planner classifies the query as the Graph ADT
and routes depth-limited reachability to the deployed engine. Full
composition-root wiring (system.New, decider guards, BDD suite):
[`example/graph-native`](../example/graph-native) and the skill's
[recipes §2.44](../.agents/skills/go-cqrs-lite/references/recipes.md).

### Follow Network with Retraction (Edge + EdgeRemoval)

```go
type Followed struct{ Follower, Followee string }
type Unfollowed struct{ Follower, Followee string }

// Node field = start node; Depth defaults to 1 when zero; Undirected is
// optional engine capability (HasUndirectedGraphSupport).
type Reachability struct {
    Node  string
    Depth int
}

followGraph := metaengine.Query[Reachability, []string]("follow_graph",
    metaengine.On(Followed{}, func(e Followed) metaengine.Edge {
        return metaengine.Edge{From: e.Follower, To: e.Followee}
    }),
    // ADR-0114 deletion-as-event: the retraction fold removes exactly the
    // edge the retracted fact added (engines without GraphRemoveEdge fail
    // loudly at apply time — there is no degraded fallback).
    metaengine.On(Unfollowed{}, func(e Unfollowed) metaengine.EdgeRemoval {
        return metaengine.EdgeRemoval{From: e.Follower, To: e.Followee}
    }),
)

// Apply facts, read reachability (typed; raw ExecuteCtx returns []any):
reach, _ := metaengine.ExecuteTyped[Reachability, []string](
    ctx, store, Reachability{Node: "alice", Depth: 2})
```

### Which Engine Serves the Traversal

The per-engine matrix (graph / undirected / edge removal / CGo — e.g. duckdb
traverses but has no `GraphRemoveEdge` and no undirected support) is
GENERATED from source and gated:
[docs/engine-capabilities.md](../docs/engine-capabilities.md). Traversal
paths in short: sqlite/turso recursive CTE (iterative fallback), postgres and
duckdb a single WITH RECURSIVE, mysql WITH RECURSIVE 8.0+ with probed
fallback, dgraph native `n(depth:)`, badger prefix-scan BFS, memory/iroh
BFS/passthrough; pebble and bbolt do not serve graph queries at all.

> Node identity is payload-derived by convention: `Edge.From`/`To` are
> untyped endpoints, and stream-ID display strings are brand-prefixed
> (`"StreamMarker:alice"`) — carry clean node names in the event payload.
> The in-memory `metaengine/graphadapter` bridge additionally flattens all
> nodes to one label; engine-backed graphs are unaffected (its package doc
> details the `Driver()` escape hatch).
