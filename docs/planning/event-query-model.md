# The Event-Query Model: The Core Abstraction

> **Status update — 2026-09-13: reconciled against source. This is a historical design record, not API reference.**
>
> The core abstraction below **shipped** (typed queries, folds-as-the-ADT, per-query projections,
> cost-based planning) — but the document was written 2026-07-23, before the first implementation,
> and several sections carry API examples that no longer match the code. Nothing in the original
> design text below has been deleted; corrections and per-section verdicts live in the
> [Implementation-Status Addendum](#implementation-status-addendum-2026-09-13) at the end.
>
> - **Shipped:** `Query[Q,R]`, `OnRecord` folds, `Plan`, `ExecuteTyped`, per-query projections,
>   cost-based planning, hot-reload APIs (`AddEngine`/`SwapEngine`/`Replan`), command-lifecycle log.
> - **Not shipped:** query log and session log (§10), multi-projection reads (§15 D3),
>   dual-read cutover (§14). (`StreamingScan` was wired 2026-09-13 — see §15 D2.)
> - **Current truth:** [`metaengine/README.md`](../../metaengine/README.md) (API surface),
>   [`commandlifecycle/`](../../commandlifecycle/) (the command log, shipped as ADR-0117).
> - **Audit trail:** [12:10 audit](../status/2026-09-13_12-10_metaengine-event-query-model-doc-audit.md) ·
>   [15:55 deep dive](../status/2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md) ·
>   [T02 source-verification notes](../status/2026-09-13_17-40_event-query-model-t02-verification-notes.md).

> **Two primitives drive everything.** Events (the source of truth across time) and Queries
> (the read intent). Commands propose events. Metadata travels with all three. Sessions are
> event streams too. Everything else — data structures, engines, indexes, projections, cost
> models — is derivation from the relationship between events and queries.

**Status:** Foundational Design (2026-07-23)
**Supersedes:** All prior meta-engine design docs for the API model. This is THE model.

---

## Table of Contents

1. [The Graph-At-Three-Levels](#1-the-graph-at-three-levels)
2. [The Three Messages: Command, Event, Query](#2-the-three-messages-command-event-query)
3. [Why Event + Query Is Sufficient](#3-why-event--query-is-sufficient)
4. [The Developer API](#4-the-developer-api)
5. [The Fold Return Type IS the ADT](#5-the-fold-return-type-is-the-adt)
6. [The Query Input Type IS the Read Pattern](#6-the-query-input-type-is-the-read-pattern)
7. [Each Query Has Its Own Independent Projection](#7-each-query-has-its-own-independent-projection)
8. [Metadata Is First-Class](#8-metadata-is-first-class)
9. [Auth Is Upstream's Concern](#9-auth-is-upstreams-concern)
10. [Commands and Queries As Event Streams](#10-commands-and-queries-as-event-streams)
11. [What the Planner Derives Automatically](#11-what-the-planner-derives-automatically)
12. [Concrete Examples](#12-concrete-examples)
13. [What the Developer Writes vs. What They Never Write](#13-what-the-developer-writes-vs-what-they-never-write)
14. [Hot-Reload: Zero-Downtime Engine Changes](#14-hot-reload-zero-downtime-engine-changes)
15. [Open Design Decisions](#15-open-design-decisions)
16. [Implementation-Status Addendum (2026-09-13)](#implementation-status-addendum-2026-09-13)

---

## 1. The Graph-At-Three-Levels

> **Scope note (2026-09-13) — PHILOSOPHY.** This section is the project's north star; it makes
> no API claims and needs no reconciliation. The three levels remain conceptually accurate.

Data is a graph. It exists at three levels, plus the dimension of time.

```
TIME (event log)
│   The graph across time. Append-only, ordered, immutable.
│   Each event is a delta — a mutation of the graph at a point in time.
│   This is the single source of truth.
│
├──▶ DISK (projections)
│       The graph materialized on disk in shapes optimized for specific queries.
│       A hash table IS a graph (one edge type: key → value).
│       A B-tree IS a graph (directed acyclic, ordered traversal).
│       A SQL table IS a graph (rows = nodes, FKs = edges, columns = attributes).
│       A Bloom filter IS a compressed graph (membership = reachable?).
│       A counter IS a graph aggregate (node property = sum of edges).
│
├──▶ RUNTIME (in-memory)
│       The graph in RAM, being actively queried right now.
│       Go structs with references = graph nodes with edges.
│       A query traverses this graph to produce an answer.
│
└──▶ QUERY (read intent)
        "I want this slice of the graph, filtered/ordered/aggregated this way."
        The query describes a TRAVERSAL PATTERN on the runtime graph.
```

**The meta-engine's job:** keep these representations in sync.

- Event arrives (time dimension) → update disk projections (optimized shapes) → query at
  runtime (in-memory traversal) → answer.
- The operator picks the disk engines. The developer declares events and queries. The
  meta-engine derives the optimal disk shapes and wires the runtime traversal.

**Why this works:** Because event sourcing makes projections disposable and independent.
Each projection is one "view" of the graph, optimized for one traversal pattern. Adding or
removing a projection doesn't affect any other projection or the event log. The graph can be
re-sliced at any time.

---

## 2. The Three Messages: Command, Event, Query

> **Scope note (2026-09-13) — PHILOSOPHY / DONE.** Messaging model; no API examples. The
> decider/query-fold symmetry is the shipped architecture (shared `record.Record` base,
> ADR-0111).

Three temporal roles, one graph:

```
Command   = "should"   (future intent — proposes a graph mutation)
Event     = "did"      (past fact — the applied mutation, immutable)
Query     = "is"       (present state — traversal of the materialized graph)
```

The **Event is the pivot.** Commands produce events. Queries consume events. Commands and
Queries never see each other.

```
Command (intent)                    Query (intent)
     │                                   │
     ▼                                   ▼
┌──────────┐                       ┌──────────┐
│ Decider  │                       │  Query   │
│ (pure)   │                       │  fold    │
│          │                       │ (pure)   │
│ state +  │                       │ event →  │
│ command  │                       │ result/  │
│ → events │                       │ delta/   │
│          │                       │ edge     │
└────┬─────┘                       └────▲─────┘
     │                                  │
     ▼                                  │
┌──────────────────────────────────────┐│
│          EVENT LOG (truth)            ││
│          append-only, ordered         ││
└────────────────────┬─────────────────┘│
                     │                  │
                     └──────────────────┘
                     replay into projections
```

### The Symmetry

Both decider and query fold are pure functions over the same event log:

|                  | Decider fold                         | Query fold                     |
| ---------------- | ------------------------------------ | ------------------------------ |
| **Scope**        | One aggregate's events               | All events (cross-aggregate)   |
| **Output**       | Aggregate state (for decision)       | Query result (for reading)     |
| **Optimization** | Singleflight, snapshots, state cache | Cost-based structure selection |
| **Lives in**     | go-cqrs-lite (existing)              | meta-engine (new)              |

The decider doesn't know its events will be projected into a Bloom filter. The query fold
doesn't know its events came from a `SuspendUser` command. Both are pure functions. The event
log is the only shared state.

### Relationships Live in Events

Relationships are not a separate concept — they're data in event payloads connecting entities:

```go
type Friendship   struct { From, To UserID; At time.Time }     // peer relationship
type TaskAssigned struct { TaskID; AssigneeID }                // one-to-many
type ReplyPosted  struct { MessageID; ReplyToID }              // tree/hierarchy
```

The fold function extracts the relationship shape. `Friendship` becomes a Graph Edge.
`TaskAssigned` becomes a Multimap key-value pair. The return type tells the planner which ADT
to use — same principle as all other folds.

---

## 3. Why Event + Query Is Sufficient

> **Scope note (2026-09-13) — DONE (model).** The derivation claims are verified in §5, §6,
> and §11; "nothing else is needed" is v1 scope, not a prohibition (typed accessors and
> latency budgets exist as optional extras).

### The Claim

The developer provides exactly two things:

1. **Events** — the mutations (what happened, in order)
2. **Queries** — the read intents (what they want to know, and how they want it returned)

From these two inputs, the planner derives:

| Derived                             | How                                                                        |
| ----------------------------------- | -------------------------------------------------------------------------- |
| What data to store                  | Query result type (each query declares its own result type)                |
| How events update the data          | Fold functions (event → typed value/delta/edge)                            |
| What access pattern the query needs | Query input type (key lookup, scan, aggregate, traverse)                   |
| What ADT the projection is          | Fold return type (Map, Set, Counter, Graph, Log)                           |
| What data structure to use          | ADT × cardinality × available engines                                      |
| Which engine serves the query       | Cost-based optimization                                                    |
| What indexes to create              | Read pattern × engine index capabilities                                   |
| Whether to denormalize              | Cross-query dependency analysis (rare — each query has its own projection) |

### Why Nothing Else Is Needed

There is no "View" type because each query defines its own result shape. There is no "Store"
interface because each query is served independently. There is no "Entity" because state is
derived, not primary. There are no "Filter/Sort/Count" declarations because the fold return
type and query input shape imply the access pattern.

The only thing the developer MUST provide that cannot be derived is **the fold functions** —
the domain knowledge of how events relate to query results. That `UserSuspended` means
"status becomes suspended" is domain logic, not infrastructure.

---

## 4. The Developer API

### Three Things the Developer Writes

> **Corrected 2026-09-13 (API drift).** These examples predate the implementation. Three things
> changed: (1) `On`/`OnTyped` are deprecated — `OnRecord` is canonical and the `record.Record`
> first parameter is required (`record_fold.go:39`); (2) query execution is
> `metaengine.ExecuteTyped[Q, R](ctx, store, input)` (`execute.go:681`), not
> `store.Execute(ctx, input)`; (3) `:=` declarations were converted to `var` so the snippet is
> valid at package level — the `Query` constructor is designed for package-level declarations
> (`query.go:236-241`).

```go
// ════════════ 1. EVENTS (pure domain types — already exist) ════════════

type UserCreated   struct { ID UserID; Email, Name, Country string; At time.Time }
type UserSuspended struct { ID UserID; At time.Time }
type UserDeleted    struct { ID UserID; At time.Time }
type Friendship     struct { From, To UserID; At time.Time }

// ════════════ 2. QUERY TYPES (each query = input + result) ════════════

type FindUser        struct { ID UserID }
type FindUserResult  struct {
    ID UserID; Name, Email, Status, Country string; JoinedAt time.Time
}

type CheckEmail        struct { Email string }
type CheckEmailResult  struct { Taken bool }

type ListByStatus        struct { Status string; Limit int; After *metaengine.Cursor }
type ListByStatusResult  struct { Users []FindUserResult; Next *metaengine.Cursor }

type CountByStatus        struct{}
type CountByStatusResult  struct { Active, Suspended, Deleted int64 }

type FriendsOf        struct { ID UserID; Depth int }
type FriendsOfResult  struct { IDs []UserID }

// ════════════ 3. QUERIES (event → result relationship) ════════════

var findUser = metaengine.Query[FindUser, FindUserResult]("find_user",
    metaengine.OnRecord(UserCreated{}, func(_ record.Record, e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{
            ID: e.ID, Name: e.Name, Email: e.Email,
            Status: "active", Country: e.Country, JoinedAt: e.At,
        }
    }),
    metaengine.OnRecord(UserSuspended{}, func(_ record.Record, e UserSuspended, prev FindUserResult) FindUserResult {
        prev.Status = "suspended"
        return prev
    }),
    metaengine.OnRecord(UserDeleted{}, metaengine.Remove[FindUserResult]()),

    metaengine.Volume(1_000_000), // optional cardinality hint
)

var checkEmail = metaengine.Query[CheckEmail, CheckEmailResult]("check_email",
    metaengine.OnRecord(UserCreated{}, func(_ record.Record, e UserCreated) string {
        return e.Email // just the key — this is a Set
    }),
    metaengine.OnRecord(UserDeleted{}, metaengine.Remove[string]()),
)

var listByStatus = metaengine.Query[ListByStatus, ListByStatusResult]("list_by_status",
    metaengine.OnRecord(UserCreated{}, func(_ record.Record, e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{ID: e.ID, Status: "active", JoinedAt: e.At}
    }),
    metaengine.OnRecord(UserSuspended{}, func(_ record.Record, e UserSuspended, prev FindUserResult) FindUserResult {
        prev.Status = "suspended"
        return prev
    }),
    metaengine.OnRecord(UserDeleted{}, metaengine.Remove[FindUserResult]()),

    // Filter/sort declared via TYPED field accessors — no strings, no column names
    metaengine.FilterOn(func(r FindUserResult) string { return r.Status }),
    metaengine.SortOn(func(r FindUserResult) time.Time { return r.JoinedAt }),
)

var countByStatus = metaengine.Query[CountByStatus, CountByStatusResult]("count_by_status",
    metaengine.OnRecord(UserCreated{}, func(_ record.Record, e UserCreated) metaengine.Delta {
        return metaengine.Delta{"active": +1}
    }),
    metaengine.OnRecord(UserSuspended{}, func(_ record.Record, e UserSuspended) metaengine.Delta {
        return metaengine.Delta{"active": -1, "suspended": +1}
    }),
    metaengine.OnRecord(UserDeleted{}, func(_ record.Record, e UserDeleted) metaengine.Delta {
        return metaengine.Delta{"suspended": -1, "deleted": +1}
    }),
)

var friendsOf = metaengine.Query[FriendsOf, FriendsOfResult]("friends_of",
    metaengine.OnRecord(Friendship{}, func(_ record.Record, e Friendship) metaengine.Edge {
        return metaengine.Edge{From: e.From, To: e.To}
    }),
)
```

### How Queries Are Called

```go
store, _ := metaengine.Plan(engines, findUser, checkEmail, listByStatus, countByStatus, friendsOf)

user, _    := metaengine.ExecuteTyped[FindUser, FindUserResult](ctx, store, FindUser{ID: userID})
taken, _   := metaengine.ExecuteTyped[CheckEmail, CheckEmailResult](ctx, store, CheckEmail{Email: "a@b.com"})
page, _    := metaengine.ExecuteTyped[ListByStatus, ListByStatusResult](ctx, store, ListByStatus{Status: "active", Limit: 50})
counts, _  := metaengine.ExecuteTyped[CountByStatus, CountByStatusResult](ctx, store, CountByStatus{})
network, _ := metaengine.ExecuteTyped[FriendsOf, FriendsOfResult](ctx, store, FriendsOf{ID: userID, Depth: 2})
```

Each call dispatches to the engine the planner chose for that specific query. The developer
doesn't know or care which engine serves which query. When several queries share one input type,
`ExecuteTyped` resolves to the most recently registered query — use
`metaengine.ExecuteTypedByName[Q, R](ctx, store, queryName, input)` (`execute.go:716`) to address
one query by name.

---

## 5. The Fold Return Type IS the ADT

The developer never declares "I need a Map" or "I need a Counter." The fold function's return
type IS the declaration. The planner inspects it at startup.

### The Return Type → ADT → Structure Mapping

> **2026-09-13 status — DIFFERENT (extended).** The shipped ADT enum has 8 values: `map`, `set`,
> `counter`, `graph`, `log`, `stream_log`, `sorted_map`, `multimap` (`types.go:6-15`), and the
> sentinel returns below extend beyond `Delta`/`Edge`/`Remove`/`Skip` with `MultiEntry`,
> `Append`, `EdgeRemoval`, `Embedding`, `IndexedText`, and `Point` (`types.go:50-95`). The
> "physical structures" comments describe engine-internal choices; the planner's own model is
> the 4 abstract layouts `row`/`columnar`/`lsm`/`kv` (`layout_type.go:9-26`).

```go
// ══ MAP ADT ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) (Key, Value) { ... })
// Returns (key, value) → planner infers Map<Key, Value>
// Physical structures: hash index (Pebble, Memory), B-tree table (SQLite)

// ══ SET ADT ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) Key { ... })
// Returns just a key → planner infers Set<Key>
// Physical structures: hash set (Memory), bloom filter (Pebble-internal policy), UNIQUE index (SQL)

> **[VERIFIED 2026-09-15 — audit item 30]** Set-membership pushdown on SQL engines:
> TRUE for SQLite, and SQLite is the only SQL engine with the Set ADT.
> `Engine.SetContains` (`metaengine/engine.go:453`) is implemented by exactly two
> engines: Memory (Go map hash set, `memory_backends.go:24`) and SQLite
> (`sqliteengine/backends.go:25` → `SELECT 1 FROM meta_set WHERE collection = ?
> AND key = ?`, `sqliteengine/engine.go:131`) — a direct index lookup. The
> "UNIQUE index" wording is substantively right but not literal: `meta_set`
> declares `PRIMARY KEY (collection, key)` (`sqliteengine/engine.go:93-96`),
> which SQLite materializes as an automatic unique index; no explicit
> `CREATE UNIQUE INDEX` is emitted. pg, MySQL, Turso, and DuckDB do not
> implement `SetAdd`/`SetContains` at all, so the planner cannot route Set
> queries there. The Bloom mention stays Pebble-internal only (audit item 14).

// ══ COUNTER ADT ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.Delta { ... })
// Returns Delta{key: ±n} → planner infers Counter
// Physical structures: atomic counter (Memory), rollup table (SQLite), HyperLogLog (approx)

// ══ GRAPH ADT ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.Edge { ... })
// Returns Edge{From, To} → planner infers Graph
// Physical structures: adjacency list (Memory), graph DB (Dgraph), recursive CTE (SQL)

// ══ MULTIMAP ADT ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.MultiEntry { ... })
// Returns MultiEntry{Key, Value} → planner infers Multimap (one key → many values)

// ══ LOG ADT ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.Append { ... })
// Returns Append{Value} → planner infers Log (ordered, append-only)
// Also: EdgeRemoval (edge retraction on tombstone events), Embedding (vector_search),
// IndexedText (full_text_search), Point (spatial_range) — see types.go:50-95

// ══ REMOVE signal ══
metaengine.OnRecord(Event{}, metaengine.Remove[Value]())
// Returns Remove → signals deletion of this key from whatever ADT it's in

// ══ SKIP signal ══
metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.Skip { return metaengine.Skip })
// Returns Skip → this event doesn't apply to this projection (no-op)
```

### Why This Is Better Than Explicit ADT Declaration

```
❌ EXPLICIT (old design):
   projection.Declare[V,K]("users").
       PointLookup().      // developer says "I want a Map"
       Filter("status").   // developer says "I want a SortedMap" (stringly-typed!)
       Count("status")     // developer says "I want a Counter"
   → Developer is doing the planner's job manually. Column names as strings. Leaky.

✅ DERIVED (this design):
   metaengine.OnRecord(Event{}, func(_ record.Record, e Event) (Key, Value) { ... })  // return type = Map
   metaengine.OnRecord(Event{}, func(_ record.Record, e Event) metaengine.Delta { ... }) // return type = Counter
   → The fold function's signature IS the ADT declaration. No strings. No manual planning.
```

---

## 6. The Query Input Type IS the Read Pattern

Just as the fold return type declares the write-side ADT, the query input type declares the
read-side access pattern. The planner inspects both.

### How the Input Shape Maps to Read Patterns

> **2026-09-13 status — DIFFERENT (expanded).** The shipped read-pattern enum has 11 values
> (`types.go:20-32`): `point_lookup`, `membership`, `filtered_scan`, `aggregate`, `traversal`,
> `scan`, `multi_lookup`, `log_tail`, `vector_search`, `full_text_search`, `spatial_range`.
> Inference is broader than input-shape matching alone: field-name prefixes
> (`Min`/`Max`/`Since`/`Until`/`From`/`To`/`Start`/`End`/`Before`/`After` map to comparison
> operators, `infer_filters.go`), composite filters (`infer_composite.go`), named queries
> (`infer_named.go`), and sort inference (`infer_sort.go`).

```go
// ══ POINT LOOKUP ══
type FindUser struct { ID UserID }
// Input has a single typed key field → planner infers: read by key from the Map

// ══ MEMBERSHIP TEST ══
type CheckEmail struct { Email string }
// Input has a single typed key, result is a boolean → planner infers: test membership in the Set

// ══ FILTERED SCAN ══
type ListByStatus struct { Status string; Limit int; After *metaengine.Cursor }
// Input carries a filter value + pagination → planner infers: scan with filter + sort
// The Status field in the input maps to the FilterOn accessor in the fold declaration
// → planner creates an index on Status

// ══ AGGREGATE READ ══
type CountByStatus struct{}
// Input is empty, result is counts → planner infers: read from Counter

// ══ GRAPH TRAVERSAL ══
type FriendsOf struct { ID UserID; Depth int }
// Input has a node ID + depth → planner infers: traverse the Graph

// ══ ALSO SHIPPED (added 2026-09-13) ══
// scan (unbounded scan), multi_lookup (batch keys), log_tail (log subscription),
// vector_search (Embedding folds), full_text_search (IndexedText folds),
// spatial_range (Point folds) — see types.go:20-32
```

### Filter/Sort: Typed Accessors, Never Strings

The old `FilterOn(func(r FindUserResult) string { return r.Status })` mechanism shipped as
`FilterOn`/`SortOn` closures (`query.go:151,167`), joined by the declarative pair
`FilterOnField[R](field, op)` / `SortOnField[R](field, desc)` (`query.go:180,193`) that carries
the column name and operator explicitly for pushdown to SQL-aware engines (`json_extract`).
Closure bodies are not reflected on: a closure-only filter is applied by calling the closure
at read time and matching the query-input field by TYPE, never by name (`query.go:142-150`).
Index/layout inference is driven separately from the query-input field names, including the
prefix conventions above (`infer_filters.go`). The query input can also carry filter values
(the `Status string` field in `ListByStatus`), which map onto the same inferred index. All
mechanisms stay type-anchored to the result type `R`; only the explicit `FilterOnField` path
names a column, and that name cannot point at a different result type.

---

## 7. Each Query Has Its Own Independent Projection

This is the critical architectural decision. **There is no shared "UserView" that serves all
queries.** Each query gets its own projection, its own data shape, its own engine.

### Why

```
FindUser needs:        {ID → full user record}           → Map<UserID, UserRecord>
CheckEmail needs:      {Email → exists}                  → Set<Email>
ListByStatus needs:    {users WHERE status=?}            → SortedMap indexed on Status
CountByStatus needs:   {status → count}                  → Counter
FriendsOf needs:       {UserID → [friend IDs]}           → Graph adjacency
```

Forcing all five through one `UserView` struct and one projection means:

- CheckEmail materializes full user records just to check existence (over-fetching)
- CountByStatus loads user records just to count them (over-fetching)
- All queries share one engine even if different engines are optimal for each
- One change to the "view" affects all queries

### Instead: One Event Stream, Five Independent Projections

```
                EVENT LOG (source of truth)
                     │
     ┌───────────────┼───────────────┬──────────────┬──────────────┐
     ▼               ▼               ▼              ▼              ▼
FindUser        CheckEmail      ListByStatus    CountBy         FriendsOf
projection      projection      projection      projection      projection
     │               │               │              │              │
Pebble hash     Bloom filter    SQLite table    SQLite rollup   Neo4j graph
(by UserID)     (email set)     + idx_status    (status→count)  (adjacency)
     │               │               │              │              │
O(1) lookup      O(k) test       O(logN) scan    O(1) read       O(degree^d)
```

When `UserCreated` arrives, ALL FIVE projections update independently — each in its own
optimal shape, each potentially on a different engine, each with zero coordination with the
others. This is possible because events are immutable and projections are disposable.

> **Reconciliation note (2026-09-13).** §7 remains the shipped architecture. The opt-in
> shared-child collection (ADR-0124, `rule_shared_collection.go`) is a layout-level
> normalization for result types that embed a declared-shared child: it forces
> `LayoutNormalize` within that query and warns when a shared type spans multiple collections
> ("without a shared collection these copies drift independently"). It does not introduce a
> shared projection or cross-query coordination. Diagram note: "Bloom filter" and "Neo4j" are
> illustrative physical choices; shipped engines express the Set as a set index and the Graph
> as Dgraph or SQL CTE (see §12 and the addendum).

### Consequence: No "Store" Object

There is no `UserStore` with 6 methods. Each query is an independent handler. A consumer that
only needs `CheckEmail` depends only on the `CheckEmail` query — not on `FindUser`,
`CountByStatus`, or anything else. This is ISP applied to the read side.

---

## 8. Metadata Is First-Class

Metadata travels WITH events today — since ADR-0111 the shared base is `record.CommonMetadata`
inside `record.Record` (`record/record.go:26-108`): correlation ID, typed `Cause`/`Actor`,
`Created`/`Received`/`Stored` presence-explicit `Stamp`s, and `SchemaVersion`. The meta-engine
treats metadata as **first-class query fields, not a side channel.**

```go
type UserCreated struct { ID UserID; Email, Name string }
// This event also carries: rec.MetaData.CorrelationID, rec.MetaData.Cause / .Actor,
// rec.MetaData.Created / .Received / .Stored (Stamps), rec.MetaData.SchemaVersion.

// A query fold can use metadata exactly like payload fields:
var userAuditTrail = metaengine.Query[AuditTrail, AuditResult]("user_audit_trail",
    metaengine.OnRecord(UserCreated{}, func(rec record.Record, e UserCreated) (time.Time, AuditEntry) {
        at := rec.MetaData.Received.Time() // Stamp.Time(); check Stamp.IsZero() when presence matters
        return at, AuditEntry{
            Action:        "created",
            At:            at,
            CorrelationID: rec.MetaData.CorrelationID,
        }
    }),
    // Sort/filter are declared on RESULT fields via typed accessors (or declarative field specs);
    // read-time ranges are scan options: metaengine.WithRange("at", low, high) (scan_options.go:40).
    metaengine.SortOn(func(r AuditEntry) time.Time { return r.At }),
)
```

With `OnRecord`, the fold receives the full `record.Record` as its first parameter — metadata is
not a side channel and no second argument is needed (`record_fold.go:26-41`). Fields the fold
copies out of `rec.MetaData` become ordinary result fields, and the planner indexes and filters
them like any other field. **Corrected 2026-09-13:** the original example used
`rec.MetaData.Timestamp` and `metaengine.RangeFilter("timestamp")`, neither of which exists;
`CausationID` is also deprecated in favor of `Cause` (removal v5).

This means queries like "who triggered this change" (causation), "show me everything in this
transaction" (correlation), and "what was the state at time T" (timestamp) are **just queries
with filter/access patterns on metadata fields.** No special machinery.

---

## 9. Auth Is Upstream's Concern

> **Scope note (2026-09-13) — PHILOSOPHY.** No auth surface exists in the meta-engine;
> unchanged design intent.

Auth is not the meta-engine's problem. The meta-engine stores identity projections like any
other data — it doesn't know or care that a field is an auth scope. Authentication (who are
you?), command authorization (can you do this?), and enforcement (RBAC/ABAC) all live upstream
in the transport layer and command handler middleware. The meta-engine just stores and queries
data.

---

## 10. Commands and Queries As Event Streams

Commands and Queries are messages with a type, payload, metadata, and timestamp. **They are
append-only logs.** The meta-engine already knows how to optimize append-only logs.

```
FOUR LOGS flow through the system:

1. Command log:    CommandSucceeded{Type, Payload, Metadata, Timestamp}
                   CommandRejected{Type, Payload, Reason, Metadata, Timestamp}

2. Query log:      QueryExecuted{Type, Payload, Duration, ResultHash, Metadata}

3. Domain event log: UserCreated{...}, UserSuspended{...}, Friendship{...}

4. Session log:    SessionStarted{ActorID, Token, Origin, IPAddress, At}
                   SessionEnded{ActorID, Token, At, Reason}
                   SessionRevoked{ActorID, Token, At, By}
```

All three are the same shape (Log ADT). All three get the same treatment — they can be
projected into queryable shapes using the same fold mechanism.

> **2026-09-13 status — PARTIAL: 1 of the 3 new logs shipped.**
>
> **Command log — SHIPPED, better than designed (ADR-0117).** `commandlifecycle` records
> `command.received`, `command.failed`, `command.retried`, `command.dead-lettered`, and
> `command.completed` (`commandlifecycle/events.go:51-64`) on `Command/<id>` and
> `CommandLifecycle/<id>` streams (`events.go:42-48`). Shipped projections: dead-letter queue,
> retry count, failure log, processing time, and per-actor commands (`CommandsByActor`, added
> 2026-09-13; `commandlifecycle/projections/projections.go`).
> Durable journals: `CommandJournal` / `SeekableCommandJournal` (`command/store.go:141-160`).
> Wiring: `system.WithCommandLifecycle(store)` (`system/lifecycle.go:50`).
>
> **Query log — NOT SHIPPED.** No `QueryExecuted` stream exists; the only query instrumentation
> is in-process observability (metrics/traces via `observability.go`).
>
> **Session log — NOT SHIPPED.** No session events exist in this repo; sessions live in
> `cqrs-htmx/identity-model` as ephemeral runtime objects (see the section below).
>
> **Framing note:** the list above has four entries — the domain event log is the pre-existing
> truth; "all three" refers to the three new logs (command, query, session). Of those, one
> shipped. The "full comprehensive audit" claim further down is therefore a target, not current
> state. Open scope questions are tracked in the T17/T18 memos of the reconciliation plan.

### Example: Auditing Commands

```go
type CommandsByUser struct { UserID UserID; Limit int }
type CommandsByUserResult struct { Commands []CommandRecord; Next *metaengine.Cursor }

var commandsByUser = metaengine.Query[CommandsByUser, CommandsByUserResult]("commands_by_user",
    metaengine.OnRecord(CommandSucceeded{}, func(rec record.Record, c CommandSucceeded) (UserID, CommandRecord) {
        return extractUser(c.Payload), CommandRecord{
            Type: c.Type, Timestamp: rec.MetaData.Received.Time(), Payload: c.Payload,
        }
    }),
    metaengine.OnRecord(CommandRejected{}, func(rec record.Record, c CommandRejected) (UserID, CommandRecord) {
        return extractUser(c.Payload), CommandRecord{
            Type: c.Type, Rejected: true, Reason: c.Reason, Timestamp: rec.MetaData.Received.Time(),
        }
    }),
)
```

### Example: Causation Chain (Command → Events → Commands)

```go
type WhatDidThisCommandCause struct { CommandID string }
type WhatDidThisCommandCauseResult struct { Events []EventRecord; Commands []CommandRecord }

var causationChain = metaengine.Query[WhatDidThisCommandCause, WhatDidThisCommandCauseResult]("causation_chain",
    metaengine.OnRecord(CommandSucceeded{}, func(rec record.Record, c CommandSucceeded) metaengine.Edge {
        if rec.MetaData.CausationID != "" {
            return metaengine.Edge{From: rec.MetaData.CausationID, To: c.ID}
        }
        return metaengine.Skip{}
    }),
    metaengine.OnRecord(UserCreated{}, func(rec record.Record, e UserCreated) metaengine.Edge {
        return metaengine.Edge{From: rec.MetaData.CausationID, To: e.ID}
    }),
)
```

### Write Efficiency: Append First, Project Lazily

Storing every command and query is a LOT of writes. The solution: the command/query logs are
**Logs first** (O(1) append), projected **lazily** (async, batched by projection host).

```
Command arrives
  → Append to command log (O(1), always)           ← fast path, never blocks
  → Emit CommandSucceeded event to bus              ← async
  → Projection host picks it up                     ← async, batched
  → Updates audit projections in background         ← O(1) per projection, batched
```

The command/query logs are Logs first (the ADT). The debugging/auditing projections are
derived from them, just like domain read models are derived from domain events.

This means a FULL COMPREHENSIVE audit log — "who did what, when, and what did it cause" — is
solved by default. The command log, query log, and session log ARE that comprehensive log,
projected into queryable shapes automatically.

> **2026-09-13 correction.** Only the command half shipped (as `commandlifecycle`, in a
> different shape — see the status box above). The query and session logs have no implementation,
> so "solved by default" is not current state. The `CommandsByUser` and `causation_chain`
> examples above are illustrative; they compile against no shipped `CommandSucceeded` type. The
> per-actor idea shipped 2026-09-13 as `projections.CommandsByActor` (keyed by the record's typed
> Actor, not by payload extraction).

### Sessions as Event Streams

Sessions are not special — they're another event stream. Event-streaming sessions enables
analytics that ephemeral runtime sessions cannot provide:

> **2026-09-13 status — NOT SHIPPED.** No session event types exist repo-wide. The section
> below is a design argument; the identity-model project still treats sessions as ephemeral
> runtime state (as the text itself acknowledges below).

```go
type SessionStarted struct { ActorID ActorID; Token string; Origin string; At time.Time }
type SessionEnded   struct { ActorID ActorID; Token string; At time.Time; Reason string }
type SessionRevoked struct { ActorID ActorID; Token string; At time.Time; By ActorID }
```

Benefits of event-streaming sessions:

- **Analytics:** "How many concurrent sessions right now?" (Counter projection: +1 on start, -1 on end)
- **Audit:** "Who was logged in when the data breach happened?" (Time-range query on session log)
- **Security:** "Revoke all sessions for suspended user" (projection reads SessionRevoked events)
- **Patterns:** "User logs in from 2 countries simultaneously" (Set/Graph projection on IPAddress)
- **Compliance:** "Show me the complete access history for this user" (Scan on session log, filtered by ActorID)

The identity-model project (`cqrs-htmx/identity-model`) treats sessions as ephemeral runtime
objects. But event-streaming them is strictly more useful — the projection cost is near-zero
(it's just another fold), and the analytics/audit/security benefits are significant.

---

## 11. What the Planner Derives Automatically

Given the event types, query types, and fold functions, the planner derives everything:

```
INPUT (from developer):
  - Event types (Go structs)
  - Query types (Go structs: input + result)
  - Fold functions (event → value / delta / edge / remove)
  - Optional: Volume(N) cardinality hint, latency budget, typed FilterOn/SortOn accessors

INPUT (from operator):
  - Available engines with cost profiles

PLANNER DERIVATION:

Step 1: Classify each query's write-side ADT
  FindUser:       fold returns (UserID, FindUserResult) → Map
  CheckEmail:     fold returns string → Set
  CountByStatus:  fold returns Delta → Counter
  ListByStatus:   fold returns (UserID, FindUserResult) → Map (+ FilterOn/SortOn → needs indexes)
  FriendsOf:      fold returns Edge → Graph

Step 2: Classify each query's read pattern
  FindUser:       {ID} → point lookup on Map
  CheckEmail:     {Email} → membership test on Set
  CountByStatus:  {} → aggregate read on Counter
  ListByStatus:   {Status, Limit, After} → filtered scan on Map (needs index on Status + JoinedAt)
  FriendsOf:      {ID, Depth} → traversal on Graph

Step 3: Assign each query to the cheapest engine
  FindUser → Pebble (O(1) hash) [or SQLite O(logN) if Pebble not available]
  CheckEmail → set projection; Pebble-internal bloom policy at scale
  CountByStatus → SQLite rollup table (O(1))
  ListByStatus → SQLite table + composite index (tenant, status, joined_at)
  FriendsOf → Dgraph if available, else SQLite CTE (degraded)

> **[VERIFIED 2026-09-15 — audit item 31]** Graph traversal depth semantics: the
> `{ID, Depth}` query shape above ships as `Engine.GraphNeighbors(ctx, collection,
> node, depth)` on all four graph engines, with consistent semantics (all nodes
> within ≤depth hops, deduplicated, origin excluded):
> - Memory: BFS frontier loop `for d := 0; d < depth` (`metaengine/memory_graph.go:43`, BFS loop :64).
> - Postgres: recursive CTE with the depth parameter; `depth <= 0` returns empty
>   (`pgengine/graph.go:57-94`).
> - SQLite: `WITH RECURSIVE walk(node, depth)` (`sqliteengine/graph.go:40`) with a
>   construction-time `WITH RECURSIVE` capability probe and fallback
>   (`sqliteengine/graph.go:49-51`) — the "degraded" path here is the non-CTE
>   fallback, not lost functionality.
> - Dgraph: native `GraphNeighbors` (depth-1 and depth-3 measured, `dgraphengine/engine.go:43-44`).
> The example query in §1 (`FriendsOf{ID: userID, Depth: 2}`) maps 1:1 onto this API.

Step 4: Plan physical structures per engine
  Pebble: users_by_id keyspace (FindUser)
  Pebble: emails set (CheckEmail)
  SQLite: users table + idx_status + idx_joined (ListByStatus), user_status_counts rollup (CountByStatus)
  Dgraph: User nodes + FRIENDS_WITH edges (FriendsOf)

Step 5: Generate projection handlers (event → engine writes)
  UserCreated →
    Pebble.Set(userID, record)         [FindUser]
    Set.Add(email)                     [CheckEmail]
    SQLite.Upsert(users, record)       [ListByStatus]
    SQLite.Increment(status_counts, "active", +1)  [CountByStatus]
  (FriendsOf handler ignores UserCreated — only listens to Friendship events)

Step 6: Generate typed read handlers
  FindUser(ctx, FindUser{ID}) → Pebble.Get(ID) → FindUserResult
  CheckEmail(ctx, CheckEmail{Email}) → Set.Test(Email) → CheckEmailResult{Taken}
  CountByStatus(ctx, CountByStatus{}) → SQLite.Scan(status_counts) → CountByStatusResult
  ListByStatus(ctx, ListByStatus{Status:"active"}) → SQLite.Query(...) → ListByStatusResult
  FriendsOf(ctx, FriendsOf{ID, Depth}) → Dgraph.Traverse(ID, Depth) → FriendsOfResult

Step 7: Validate + warn
  - Every query has an assigned engine ✓
  - Check for degraded patterns (Graph on SQL CTE → warn)
  - Check write amplification (5 projections per UserCreated → warn if >3)
  - Check memory constraints against available RAM

OUTPUT:
  - Projection plan (which engine, which structure, which indexes)
  - Auto-generated write handlers (event → engine writes)
  - Auto-generated read handlers (query → engine reads → result)
  - Startup diagnostics (warnings, degradation, costs)
```

> **2026-09-13 verification — DONE (all 7 steps have source counterparts).** Step 1 →
> `fold_classify.go:10` (`classifyADT`) · Step 2 → `infer_filters.go`, `infer_sort.go`,
> `infer_composite.go`, `infer_named.go` · Step 3 → `cost.go:70` (`estimateCost`) with
> `rules.go:54` (`defaultRules`) · Step 4 → `layout.go:49` (`BuildLayoutPlan`), `:116` (`DDL()`) ·
> Step 5 → `auto_fold.go` plus the applyFold pipeline (`store.go:573-922`) · Step 6 →
> `typed_reader*.go`, `execute.go:681` (`ExecuteTyped`) · Step 7 → `rules.go`, `plan_audit.go`,
> `explain.go:283` (`Doctor`). The derivation model shipped; only concrete engine/structure
> names in the walkthrough were illustrative (corrected above: Dgraph, no Bloom ADT).

---

## 12. Concrete Examples

> **2026-09-13 corrections.** (1) There is no YAML engine-config format in the shipped library —
> engines are Go values composed with `metaengine.Plan([]metaengine.Engine{...})` or the
> `PlanFromMemory` convenience; operators still pick engines at deployment, as designed.
> (2) There is no Neo4j engine; the shipped graph engine is Dgraph
> (`dgraphengine.New(addr)`) with a SQL recursive-CTE fallback (`graph_fallback.go:14,36`) —
> decided in [ADR-0119](../adr/0119-dgraph-engine.md), transactional support deferred per
> [ADR-0129](../adr/0129-dgraph-engine-transactional-deferred.md).
> (3) Bloom filters are a Pebble-internal policy (10 bits/key), not a projection shape or ADT.
> (4) Real engine roster: in-process memory plus badger, bbolt, dgraph, duckdb, iroh, mysql,
> pebble, pg, sqlite, turso (each its own `metaengine/*engine` module). The YAML blocks below
> are illustrative pseudo-config, kept for intent.

### Example 1: Single SQLite (Development)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/app.db
```

The shipped form (Go composition):

```go
eng, _ := sqliteengine.NewSQLiteEngineFromDSN("/data/app.db")
store, _ := metaengine.Plan([]metaengine.Engine{eng}, findUser, checkEmail, listByStatus, countByStatus, friendsOf)
```

```
Planner plan for all 5 queries:

  FindUser:       SQLite table users_by_id, PK index          O(logN) ✓
  CheckEmail:     SQLite UNIQUE index on email                O(logN) ✓
  ListByStatus:   SQLite table users + idx_status             O(logN) ✓
  CountByStatus:  SQLite rollup table status_counts           O(1)    ✓
  FriendsOf:      SQLite junction table + recursive CTE       O(N)    ⚠ DEGRADED

⚠ FriendsOf: using SQL recursive CTE for graph traversal. O(N) per query.
  Add Dgraph (dgraphengine.New) or another graph engine for deep traversal at scale.
```

### Example 2: SQLite + Pebble + Dgraph (Production)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/indexes.db
  pebble:
    driver: pebble
    dsn: /data/kv
  dgraph:
    driver: dgraph
    dsn: dgraph:9080
```

```
Planner plan for all 5 queries:

  FindUser:       Pebble hash index users_by_id               O(1)    ✓ OPTIMAL
  CheckEmail:     Pebble hash set emails                      O(1)    ✓ OPTIMAL
  ListByStatus:   SQLite table users + idx_status             O(logN) ✓ OPTIMAL
  CountByStatus:  SQLite rollup table status_counts           O(1)    ✓ OPTIMAL
  FriendsOf:      Dgraph User nodes + FRIENDS_WITH edges      O(d)    ✓ OPTIMAL

ALL queries at optimal complexity. Zero degradation.
Three engines. Five projections. One event stream. Zero projection code by the developer.
```

### Example 3: Memory Only (Testing/CI)

```yaml
engines:
  memory: {}
```

The shipped form:

```go
store, _ := metaengine.PlanFromMemory(findUser, checkEmail, listByStatus, countByStatus, friendsOf)
```

```
Planner plan for all 5 queries:

  FindUser:       Memory map[UserID]FindUserResult            O(1)    ✓
  CheckEmail:     Memory map[string]struct{}                  O(1)    ✓
  ListByStatus:   Memory slice + in-memory filter             O(N)    ⚠ DEGRADED (fine for tests)
  CountByStatus:  Memory map[string]int64                     O(1)    ✓
  FriendsOf:      Memory adjacency list                       O(d)    ✓

⚠ ListByStatus: in-memory filter (O(N)). Fine for tests. Add SQLite for production filtering.
```

---

## 13. What the Developer Writes vs. What They Never Write

### The Developer Writes (3 things):

1. **Event types** — pure Go structs, domain vocabulary
2. **Query types** — input + result structs, named as domain questions
3. **Fold functions** — event → result mapping, pure functions, no storage types

### The Developer NEVER Writes:

| Never                                                          | Why                                                           |
| -------------------------------------------------------------- | ------------------------------------------------------------- |
| `database/sql` imports                                         | Engines are operator-provided                                 |
| `storage/` imports                                             | No storage packages in consumer code                          |
| `*sql.DB` handling                                             | Connection management is in engine plugins                    |
| SQL DDL (CREATE TABLE, CREATE INDEX)                           | Planner auto-generates from fold result type                  |
| ViewMapper with column types                                   | No column type declarations at all                            |
| IndexSpec declarations                                         | Planner auto-creates from read patterns                       |
| kv.ViewStore implementations                                   | Engine plugins implement these                                |
| Projection tier selection (Materialize vs Relational vs Graph) | Planner selects                                               |
| Engine selection per query                                     | Planner selects based on cost                                 |
| "Entity" types                                                 | No entities — events + queries only                           |
| "Store" interfaces                                             | No stores — each query is independent                         |
| Stringly-typed column names in declarations                    | All typed Go                                                  |
| Auth enforcement                                               | Auth is upstream's concern — the meta-engine just stores data |

---

## 14. Hot-Reload: Zero-Downtime Engine Changes

From the "Perfect Software Architecture" requirements: the operator must be able to add or
remove engines WITHOUT restarting the application.

### The Flow

```
1. Operator adds a Pebble engine to a running app (config reload or API call)
2. Planner detects new engine, re-plans
3. FindUser could now be O(1) on Pebble instead of O(logN) on SQLite
4. Planner creates new Pebble projection for FindUser
5. Background replay: events replayed from the log into the new Pebble projection
6. While replaying: reads continue from SQLite (old projection)
7. When caught up: atomic cutover — reads switch to Pebble
8. Old SQLite projection for FindUser optionally torn down
9. ZERO DOWNTIME
```

> **2026-09-13 status — PARTIAL.** The runtime APIs shipped: `AddEngine`/`RemoveEngine`
> (`runtime_backend.go:55,113`), `SwapEngine` (`advanced.go:69`), `Replan` (`store.go:88`),
> `ReplanLayout` (`relayout.go:64`), `CheckRouting` (`store_routing.go:59`), plus shadow roles
> (`RoleMigration`/`RoleBackup`, `roles.go:11-21`) for cutover-by-role. Steps 5-8 as drawn
> (background replay into a new projection, dual-read cutover, teardown) are NOT shipped as an
> orchestrator; `CatchUpEngine` covers replay for quarantine recovery only (`failover.go:60`).

### What This Requires

- The planner must be **re-plannable** (not one-shot at startup)
- Projections must support **live cutover** (dual-read during transition)
- The projection host must support **background replay** while serving live reads
- The plan must be a **live runtime object**, not a startup artifact

> **2026-09-13 scorecard:** re-plannable ✓ (`Store.Replan`), live plan object ✓ (`PlanAudit`
> history, `SerializablePlan`), background replay ~ (engine catch-up only), live cutover ✗.

### Symmetric Removal

```
1. Operator removes the graph engine (cost savings)
2. Planner re-plans FriendsOf: now served by SQLite recursive CTE (degraded)
3. Planner creates new SQLite junction table projection for FriendsOf
4. Background replay from event log
5. When caught up: cutover to SQLite, disconnect the graph engine
6. ⚠ WARNING: "FriendsOf now using SQL CTE (O(N)). Was O(d) on the graph engine.
     Query latency will increase at scale. Consider re-adding a graph engine."
```

> **2026-09-13 note:** same partial status as the flow above; the degraded-pattern warning
> exists via rules (`rule_degraded_adt.go`) and re-plan diagnostics. "Neo4j" in the original
> text was never a shipped engine — read "Dgraph or another graph engine".

---

## 15. Open Design Decisions

### Decision 1: How Does the Planner Extract Field Paths from Typed Accessors?

`FilterOn(func(r FindUserResult) string { return r.Status })` — the planner needs to know this
means "field Status on type FindUserResult."

**Options:**
A. Reflection on the closure at startup (Go reflection can inspect function signatures but
not closure bodies — would need a code-generation step or a convention)
B. The accessor returns a named field descriptor:
`FilterOn(func(r FindUserResult) metaengine.Field { return r.Field("Status") })`
C. Code generation at build time (a `go generate` step that extracts field paths from
accessor functions)

**Recommendation:** Option B (named field descriptor). It's fully typed, requires no
reflection or codegen, and the `Field()` method returns a `metaengine.Field` type that carries
the field name and type for the planner.

> **2026-09-13 resolution — hybrid shipped, not Option B.** No `metaengine.Field` type and no
> codegen. `FilterOn`/`SortOn` closures filter and sort at read time by TYPE-matching the query
> input (`query.go:142-150`); the declarative pair `FilterOnField`/`SortOnField`
> (`query.go:180,193`) carries explicit field names for pushdown; index inference reads the
> query-input field names and prefixes (`infer_filters.go`, `infer_sort.go`).

### Decision 2: Streaming vs. Slicing for Large Results

`ListByStatus` might return 1M users. The query input has `Limit` and `After` (cursor) for
pagination. But what about bulk operations (export all, analytics scan)?

**Proposal:** Each query handler supports two modes:

- `Execute(ctx, input) → output` for bounded results (point lookup, count, paginated list)
- `Stream(ctx, input, fn func(output) error)` for unbounded results (full scans, exports)

The planner generates both. Streaming uses Go iterators (`iter.Seq2[Output, error]`).

> **2026-09-13 status — WIRED (option A executed).** `Store.StreamCollection`
> (`stream_collection.go`) streams via the `StreamingScan` capability (`engine.go:369-384`;
> sqlite, pebble, bbolt, badger) and falls back to `ScanBackend.MapScan`; `Store.Export` streams
> each collection row-by-row instead of materializing it. The query-level
> `Stream(ctx, input, fn)` form remains future work.

### Decision 3: What About Queries That Need Data From Multiple Projections?

Example: "Active users who have >5 friends" needs Status from FindUser (Pebble) + friend
count from FriendsOf (Dgraph).

**Option A (recommended):** Declare a new query with a fold that combines both. This creates
a third projection that maintains both status and friend count. No cross-engine read at query
time. Write amplification: this projection listens to both UserCreated AND Friendship events.

**Option B:** The query handler fans out at read time (query both engines, merge in memory).
Correct but potentially slow. If engines are local, this is fine. If remote, the planner
warns and suggests Option A.

> **2026-09-13 status — NOT SHIPPED.** Reads are single-collection; grouped/aggregate reads
> exist (`typed_reader_grouped.go`, `typed_reader_aggregates.go`) but no cross-projection
> fan-out. Option A remains the design direction if multi-projection queries are pursued;
> no decision has been made.

### Decision 4: How Does Hot-Reload Track Replay Progress?

During background replay (hot-reload step 5), the system needs to track: "how many events
have been replayed into the new Pebble projection?"

**Proposal:** Each projection maintains a checkpoint (last processed event ID), exactly like
the existing `projectionhost` checkpoint store. The planner reads the checkpoint to determine
catch-up progress. When checkpoint == event log tail, the projection is "caught up" and ready
for cutover.

> **2026-09-13 status — PARTIAL (different mechanism).** Replay progress is tracked for
> quarantine catch-up: `CatchUpState.Replayed`/`CompletedAt` (`catchup_state.go:11-29`), driven
> by `CatchUpEngine` (`failover.go:60`), surfaced in `EngineStats` and the Doctor report.
> projectionhost separately maintains subscriber checkpoints. Checkpoint-driven CUTOVER
> orchestration does not exist.

---

## Implementation-Status Addendum (2026-09-13)

> Added by the truth-reconciliation pass
> ([plan](2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md)).
> The design text above is preserved verbatim; every row below was verified against source in
> September 2026. Status vocabulary: **DONE** = shipped as designed · **DIFFERENT** = shipped in
> a different shape · **PARTIAL** = some shipped, some not · **NOT SHIPPED** = no implementation ·
> **PHILOSOPHY** = design intent, not a code claim.

| §  | Section                           | Status     | What actually shipped                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| -- | --------------------------------- | ---------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Graph-at-three-levels             | PHILOSOPHY | Conceptual frame; still the north star, no code artifact of its own.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| 2  | Three messages                    | DONE       | `record.Record` is the shared base (ADR-0111); decider fold and query fold are both pure over the event log.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
| 3  | Event + Query sufficient          | DONE       | Planner derives the ADT from the fold return type (`fold_classify.go:10`) and the read pattern from input inference (`infer_filters.go`, `infer_sort.go`, `infer_composite.go`, `infer_named.go`).                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| 4  | Developer API                     | DIFFERENT  | `Query[Q,R]` and `OnRecord` folds (`record_fold.go:39`) are canonical; `On`/`OnTyped` are deprecated (removal v5). Execution is package-level `ExecuteTyped[Q,R](ctx, store, input)` (`execute.go:681`) — there is no `store.Execute(ctx, input)` method. Examples corrected inline.                                                                                                                                                                                                                                                                                                                                                                |
| 5  | Fold return type = ADT            | DIFFERENT  | ADT enum has 8 values: map, set, counter, graph, log, stream_log, sorted_map, multimap (`types.go:6-15`). Beyond `Delta`/`Edge`/`Remove`/`Skip`, sentinel returns include `MultiEntry`, `Append`, `EdgeRemoval`, `Embedding`, `IndexedText`, `Point` (`types.go:50-95`). Physical structures are abstracted to 4 layouts: row, columnar, lsm, kv (`layout_type.go:9-26`).                                                                                                                                                                                                                                                                           |
| 6  | Query input type = read pattern   | DIFFERENT  | 11 read patterns shipped (`types.go:20-32`): point_lookup, membership, filtered_scan, aggregate, traversal, scan, multi_lookup, log_tail, vector_search, full_text_search, spatial_range. Inference includes field-name prefixes (Min/Max/Since/Until..., `infer_filters.go`), composite filters, named queries, and sort inference.                                                                                                                                                                                                                                                                                                                |
| 7  | Independent projections           | DONE       | One collection per query; no shared view object. The opt-in shared-child collection (ADR-0124) is layout-level normalization only and warns rather than coordinating (`rule_shared_collection.go:85-96`).                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| 8  | Metadata first-class              | DIFFERENT  | Real metadata: `record.CommonMetadata` (`record/record.go:26-108`) — CorrelationID, Cause/Actor (typed; CausationID/ActorID deprecated), Created/Received/Stored `Stamp`s, SchemaVersion. There is no `rec.MetaData.Timestamp` and no `RangeFilter` API; ranges are `WithRange(column, low, high)` (`scan_options.go:40`) or `FilterOnField` (`query.go:180`). Example corrected inline.                                                                                                                                                                                                                                                            |
| 9  | Auth upstream                     | PHILOSOPHY | Unchanged intent; metaengine has no auth surface.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| 10 | Commands/queries as event streams | PARTIAL    | **Command log: SHIPPED, better than designed** — `commandlifecycle` (ADR-0117): 5 event types (`command.received/failed/retried/dead-lettered/completed`) on `Command/<id>` + `CommandLifecycle/<id>` streams, projections for DLQ/retry-count/failure-log/processing-time plus per-actor `CommandsByActor` (2026-09-13), plus `CommandJournal`/`SeekableCommandJournal` (`command/store.go:141-160`) and `system.WithCommandLifecycle` (`system/lifecycle.go:50`). **Query log: NOT SHIPPED** — in-process observability hooks only (`observability.go:86`). **Session log: NOT SHIPPED** — sessions remain external (`cqrs-htmx/identity-model`). |
| 11 | Planner derivation                | DONE       | All 7 steps have source counterparts; mapping annotated inline.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| 12 | Concrete examples                 | DIFFERENT  | No Neo4j engine; graph = Dgraph (`metaengine/dgraphengine/`; [ADR-0119](../adr/0119-dgraph-engine.md), [ADR-0129](../adr/0129-dgraph-engine-transactional-deferred.md)) or SQL CTE fallback (`graph_fallback.go:14,36`). No YAML config format — engines are composed in Go at deployment. Bloom filters exist only as a Pebble-internal policy (10 bits/key), not an ADT. Real engine roster: in-process memory plus badger, bbolt, dgraph, duckdb, iroh, mysql, pebble, pg, sqlite, turso (each its own `metaengine/*engine` module). Examples annotated inline.                                                                                  |
| 13 | What the developer never writes   | DONE       | DDL, column types, and indexes are derived (`layout.go:116` `DDL()`, `:169` `inferColumnType`, `:199` `BuildLayoutPlanFromType`; index inference in `infer_*.go`). Boundary: classic modules (`storage/relational`, `storage/view`, `graph`) still expose explicit schema/`IndexSpec` APIs for consumers not using auto-projection.                                                                                                                                                                                                                                                                                                                 |
| 14 | Hot-reload                        | PARTIAL    | Runtime APIs exist: `AddEngine` (`runtime_backend.go:55`), `RemoveEngine` (`:113`), `SwapEngine` (`advanced.go:69`), `Replan` (`store.go:88`), `ReplanLayout` (`relayout.go:64`), `CheckRouting` (`store_routing.go:59`), shadow roles Migration/Backup (`roles.go:11-21`). The dual-read + atomic-cutover orchestration drawn above is NOT shipped.                                                                                                                                                                                                                                                                                                |
| 15 | Open decisions                    | SEE BELOW  | D1 resolved; D2 WIRED 2026-09-13 (`Store.StreamCollection` + streaming `Export`); D3 not shipped; D4 partial. Details below.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |

### §15 decision resolutions

- **Decision 1 (field paths from typed accessors) — RESOLVED, hybrid.** `FilterOn`/`SortOn`
  closures (`query.go:151,167`) plus declarative `FilterOnField`/`SortOnField` (`query.go:180,193`)
  with inferred indexes (`infer_filters.go`, `infer_sort.go`). The closure body is not reflected
  on; the declarative pair carries the field name explicitly.
- **Decision 2 (streaming vs slicing) — WIRED 2026-09-13.** `Store.StreamCollection`
  (`stream_collection.go`) uses `StreamingScan` (`engine.go:369-384`) when available and falls
  back to `ScanBackend.MapScan`; `Store.Export` streams row-by-row. Query-level
  `Stream(ctx, input, fn)` remains future work.
- **Decision 3 (multi-projection queries) — NOT SHIPPED.** Reads are single-collection;
  grouped/aggregate reads exist (`typed_reader_grouped.go`, `typed_reader_aggregates.go`) but no
  cross-projection fan-out. Option A remains the design direction.
- **Decision 4 (replay progress) — PARTIAL.** `CatchUpState`/`CatchUpEngine` track quarantine
  rebuild progress (`catchup_state.go`, `failover.go:60`); projectionhost maintains subscriber
  checkpoints. No checkpoint-driven cutover orchestrator.

### Where current truth lives

- [`metaengine/README.md`](../../metaengine/README.md) — module overview and API surface.
- [`commandlifecycle/`](../../commandlifecycle/) — command-log implementation (ADR-0117).
- Audits: [12:10 audit](../status/2026-09-13_12-10_metaengine-event-query-model-doc-audit.md) ·
  [15:55 deep dive](../status/2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md) ·
  [T02 verification notes](../status/2026-09-13_17-40_event-query-model-t02-verification-notes.md).

### Features beyond this document (coverage map)

> Shipped capabilities the design text above does not describe. Verified 2026-09-13, one
> `file:line` per feature. Not exhaustive — this is the delta a reader would otherwise miss.

**Data layer and lifecycle**

- Materialized views: `MaterializedViewSpec` (`materialized_view.go:24`), versioning
  (`materialized_view_versions.go`), engine reporting (`materialized_view_doctor.go`).
- Replication metadata and lag accounting (`replication.go:16,44,60`) with a per-engine
  replicator (`replicator.go:61`); durability tiers validated per driver (`durability.go:50,79`).
- Export/import of collections (`export_import.go:12,79`) — export streams via
  `Store.StreamCollection` + `StreamingScan` since 2026-09-13 (§15 D2).
- Hot/cold demotion with preflight and shadow replay (`demote.go:65,198,311`); planned-collection
  backfill (`backfill.go:48`).
- Batch atomicity: `Store.ApplyBatch` (`store.go:429`) plus the engine capability interface
  `Transactional.RunInTx` (`transaction.go:15`).
- Engine reset for replay-safe rebuilds (`reset.go:66`).

**Query and read extras**

- Vector search: `Embedding` folds, in-memory index (`vector_search.go:162`), binary embeddings
  (`vector_binary.go`), execute path (`vector_search_execute.go`).
- Spatial range: `Point` folds plus in-memory spatial index with haversine distance
  (`spatial.go:56,99`).
- Aggregations: `AggregateReader`/`GroupedAggregateReader` (`aggregations.go:20,65`), grouped
  pushdown (`typed_reader_grouped.go:85,94`), aggregate reads (`typed_reader_aggregates.go`).
- Cursors and keyset pagination (`cursor.go:30,45`, `sort_paginate.go`), unbounded scans
  (`typed_reader_scan.go:10`).
- Time travel on the memory engine: `MapGetAsOf`/`MapExistsAsOf` (`memory_versioned.go:66,93`).

**Transport and streaming**

- SSE serving of typed watchers (`sse.go:104`, `dx.go:65`) with replay-from-cursor
  (`sse_replay.go:29,75`, `WithSSEReplayLimit`).
- Trace replay tooling: `ReadTrace`/`ReplayTrace` plus store sink
  (`trace_player.go:39,78,102`).

**Operations and plan control**

- Live latency: `ProbeEngine` background probing (`probe.go:222`), `LatencyTracker`
  (`latency.go:97`), calibration precedence, hysteresis/min-delta routing options.
- Health-driven quarantine and catch-up recovery (`engine_health.go`, `failover.go:60`,
  `catchup_state.go`).
- Priority overrides (`priority.go:173`), plan audit history (`plan_audit.go:49,57`), plan diff
  (`plan_diff.go:47`), EXPLAIN/Doctor diagnostics (`explain.go:283`, `inspect.go:12,36`).
- Capability auditing per engine (`capability_audit.go:74,188`), per-query fold locks
  (`fold_locks.go:17`), consistency event log for tests (`consistency.go:17`).
- Projection roles Active/DualUse/Migration/Backup (`roles.go:11-21`) with shadow routing
  semantics.
