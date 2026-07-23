# The Event-Query Model: The Core Abstraction

> **Two primitives. That's all.** Events (the source of truth across time) and Queries
> (the read intent). Everything else — data structures, engines, indexes, projections,
> cost models — is derivation from the relationship between these two.

**Status:** Foundational Design (2026-07-23)
**Supersedes:** All prior meta-engine design docs for the API model. This is THE model.

---

## Table of Contents

1. [The Graph-At-Three-Levels](#1-the-graph-at-three-levels)
2. [Why Event + Query Is Sufficient](#2-why-event--query-is-sufficient)
3. [The Developer API](#3-the-developer-api)
4. [The Fold Return Type IS the ADT](#4-the-fold-return-type-is-the-adt)
5. [The Query Input Type IS the Read Pattern](#5-the-query-input-type-is-the-read-pattern)
6. [Each Query Has Its Own Independent Projection](#6-each-query-has-its-own-independent-projection)
7. [What the Planner Derives Automatically](#7-what-the-planner-derives-automatically)
8. [Concrete Examples](#8-concrete-examples)
9. [What the Developer Writes vs. What They Never Write](#9-what-the-developer-writes-vs-what-they-never-write)
10. [Open Design Decisions](#10-open-design-decisions)

---

## 1. The Graph-At-Three-Levels

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

## 2. Why Event + Query Is Sufficient

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

## 3. The Developer API

### Three Things the Developer Writes

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

type ListActiveUsers        struct { Limit int; After *metaengine.Cursor }
type ListActiveUsersResult  struct { Users []FindUserResult; Next *metaengine.Cursor }

type CountByStatus        struct{}
type CountByStatusResult  struct { Active, Suspended, Deleted int64 }

type FriendsOf        struct { ID UserID; Depth int }
type FriendsOfResult  struct { IDs []UserID }

// ════════════ 3. QUERIES (event → result relationship) ════════════

findUser := metaengine.Query[FindUser, FindUserResult]("find_user",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{
            ID: e.ID, Name: e.Name, Email: e.Email,
            Status: "active", Country: e.Country, JoinedAt: e.At,
        }
    }),
    metaengine.On(UserSuspended{}, func(e UserSuspended, prev FindUserResult) FindUserResult {
        prev.Status = "suspended"
        return prev
    }),
    metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]()),

    metaengine.Volume(1_000_000), // optional cardinality hint
)

checkEmail := metaengine.Query[CheckEmail, CheckEmailResult]("check_email",
    metaengine.On(UserCreated{}, func(e UserCreated) string {
        return e.Email // just the key — this is a Set
    }),
    metaengine.On(UserDeleted{}, metaengine.Remove[string]()),
)

listActive := metaengine.Query[ListActiveUsers, ListActiveUsersResult]("list_active",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) {
        return e.ID, FindUserResult{..., Status: "active", JoinedAt: e.At}
    }),
    metaengine.On(UserSuspended{}, func(e UserSuspended, prev FindUserResult) FindUserResult {
        prev.Status = "suspended"
        return prev
    }),
    metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]()),
    // Read intent: filter by Status="active", sort by JoinedAt DESC, paginate
    // The planner derives this from the Query input type (see section 5)
)

countByStatus := metaengine.Query[CountByStatus, CountByStatusResult]("count_by_status",
    metaengine.On(UserCreated{}, func(e UserCreated) metaengine.Delta {
        return metaengine.Delta{"active": +1}
    }),
    metaengine.On(UserSuspended{}, func(e UserSuspended) metaengine.Delta {
        return metaengine.Delta{"active": -1, "suspended": +1}
    }),
    metaengine.On(UserDeleted{}, func(e UserDeleted) metaengine.Delta {
        return metaengine.Delta{"suspended": -1, "deleted": +1}
    }),
)

friendsOf := metaengine.Query[FriendsOf, FriendsOfResult]("friends_of",
    metaengine.On(Friendship{}, func(e Friendship) metaengine.Edge {
        return metaengine.Edge{From: e.From, To: e.To}
    }),
)
```

### How Queries Are Called

```go
// Each query is an independently callable function. No "store" object.
store, _ := metaengine.Plan(engines, findUser, checkEmail, listActive, countByStatus, friendsOf)

user, _   := store.Execute(ctx, FindUser{ID: userID})          // → FindUserResult
taken, _  := store.Execute(ctx, CheckEmail{Email: "a@b.com"}) // → CheckEmailResult
page, _   := store.Execute(ctx, ListActiveUsers{Limit: 50})   // → ListActiveUsersResult
counts, _ := store.Execute(ctx, CountByStatus{})               // → CountByStatusResult
network, _ := store.Execute(ctx, FriendsOf{ID: userID, Depth: 2}) // → FriendsOfResult
```

Each `Execute` dispatches to the engine the planner chose for that specific query. The
developer doesn't know or care which engine serves which query.

---

## 4. The Fold Return Type IS the ADT

The developer never declares "I need a Map" or "I need a Counter." The fold function's return
type IS the declaration. The planner inspects it at startup.

### The Return Type → ADT → Structure Mapping

```go
// ══ MAP ADT ══
metaengine.On(Event{}, func(e Event) (Key, Value) { ... })
// Returns (key, value) → planner infers Map<Key, Value>
// Physical structures: hash index (Pebble, Memory), B-tree table (SQLite)

// ══ SET ADT ══
metaengine.On(Event{}, func(e Event) Key { ... })
// Returns just a key → planner infers Set<Key>
// Physical structures: hash set (Memory), Bloom filter (Memory/Pebble), UNIQUE index (SQL)

// ══ COUNTER ADT ══
metaengine.On(Event{}, func(e Event) metaengine.Delta { ... })
// Returns Delta{key: ±n} → planner infers Counter
// Physical structures: atomic counter (Memory), rollup table (SQLite), HyperLogLog (approx)

// ══ GRAPH ADT ══
metaengine.On(Event{}, func(e Event) metaengine.Edge { ... })
// Returns Edge{From, To} → planner infers Graph
// Physical structures: adjacency list (Memory), graph DB (Neo4j), recursive CTE (SQL)

// ══ REMOVE signal ══
metaengine.On(Event{}, metaengine.Remove[Value]())
// Returns Remove → signals deletion of this key from whatever ADT it's in
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
   metaengine.On(Event{}, func(e Event) (Key, Value) { ... })  // return type = Map
   metaengine.On(Event{}, func(e Event) metaengine.Delta { ... }) // return type = Counter
   → The fold function's signature IS the ADT declaration. No strings. No manual planning.
```

---

## 5. The Query Input Type IS the Read Pattern

Just as the fold return type declares the write-side ADT, the query input type declares the
read-side access pattern. The planner inspects both.

### How the Input Shape Maps to Read Patterns

```go
// ══ POINT LOOKUP ══
type FindUser struct { ID UserID }
// Input has a single typed key field → planner infers: read by key from the Map

// ══ MEMBERSHIP TEST ══
type CheckEmail struct { Email string }
// Input has a single typed key, result is a boolean → planner infers: test membership in the Set

// ══ FILTERED SCAN ══
type ListActiveUsers struct { Limit int; After *metaengine.Cursor }
// Input has pagination but no key → planner infers: scan with filter + sort
// The filter/sort criteria come from the query name + the fold function's data shape
// (e.g., if the fold stores Status="active", and this query returns active users,
//  the planner knows it needs an index on Status to serve this efficiently)

// ══ AGGREGATE READ ══
type CountByStatus struct{}
// Input is empty, result is counts → planner infers: read from Counter

// ══ GRAPH TRAVERSAL ══
type FriendsOf struct { ID UserID; Depth int }
// Input has a node ID + depth → planner infers: traverse the Graph
```

### The Tension: How Does the Planner Know the Filter/Sort Criteria?

For point lookups, membership tests, aggregates, and traversals, the read pattern is obvious
from the input type.

For filtered/ordered scans (`ListActiveUsers`), the planner needs to know: **filter on what?
sort by what?** Two options:

**Option A: The query input type carries the filter as a typed field:**

```go
type ListByStatus struct {
    Status string   // ← the planner sees this field and knows: filter by Status
    Limit  int
    After  *metaengine.Cursor
}
```

The field name in the query input maps to the field name in the fold result. Fully typed,
no strings. The planner creates an index on that field.

**Option B: The query declares its filter via a typed accessor:**

```go
listActive := metaengine.Query[ListActiveUsers, ListActiveUsersResult]("list_active",
    folds...,
    metaengine.FilterOn(func(r FindUserResult) string { return r.Status }, "active"),
    metaengine.SortOn(func(r FindUserResult) time.Time { return r.JoinedAt }),
)
```

The field accessor `r.Status` is typed Go. The planner extracts the field path and creates
an index. No strings.

**Recommendation:** Option A for simple cases (field in query input = filter column). Option B
for complex cases (multi-field filters, computed filters). The planner supports both.

---

## 6. Each Query Has Its Own Independent Projection

This is the critical architectural decision. **There is no shared "UserView" that serves all
queries.** Each query gets its own projection, its own data shape, its own engine.

### Why

```
FindUser needs:      {ID → full user record}           → Map<UserID, UserRecord>
CheckEmail needs:    {Email → exists}                  → Set<Email>
ListActiveUsers needs: {users WHERE status=active}     → SortedMap indexed on Status
CountByStatus needs: {status → count}                  → Counter
FriendsOf needs:     {UserID → [friend IDs]}           → Graph adjacency
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
    FindUser        CheckEmail      ListActive      CountBy         FriendsOf
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

### Consequence: No "Store" Object

There is no `UserStore` with 6 methods. Each query is an independent handler. A consumer that
only needs `CheckEmail` depends only on the `CheckEmail` query — not on `FindUser`,
`CountByStatus`, or anything else. This is ISP applied to the read side.

---

## 7. What the Planner Derives Automatically

Given the event types, query types, and fold functions, the planner derives everything:

```
INPUT (from developer):
  - Event types (Go structs)
  - Query types (Go structs: input + result)
  - Fold functions (event → value / delta / edge / remove)
  - Optional: Volume(N) cardinality hint, latency budget

INPUT (from operator):
  - Available engines with cost profiles

PLANNER DERIVATION:

Step 1: Classify each query's write-side ADT
  FindUser:     fold returns (UserID, FindUserResult) → Map
  CheckEmail:   fold returns string → Set
  CountByStatus: fold returns Delta → Counter
  ListActive:   fold returns (UserID, FindUserResult) → Map (+ needs index for read)
  FriendsOf:    fold returns Edge → Graph

Step 2: Classify each query's read pattern
  FindUser:     {ID} → point lookup on Map
  CheckEmail:   {Email} → membership test on Set
  CountByStatus: {} → aggregate read on Counter
  ListActive:   {Limit, After} → filtered scan on Map (needs index on Status)
  FriendsOf:    {ID, Depth} → traversal on Graph

Step 3: Assign each query to the cheapest engine
  FindUser → Pebble (O(1) hash) [or SQLite O(logN) if Pebble not available]
  CheckEmail → Bloom filter at scale, hash set if small
  CountByStatus → SQLite rollup table (O(1))
  ListActive → SQLite table + index on Status (O(logN))
  FriendsOf → Neo4j if available, else SQLite CTE (degraded)

Step 4: Plan physical structures per engine
  Pebble: users_by_id keyspace (FindUser)
  Bloom: emails set (CheckEmail) [if scale justifies]
  SQLite: users table + idx_status (ListActive), user_status_counts rollup (CountByStatus)
  Neo4j: User nodes + FRIENDS_WITH edges (FriendsOf)

Step 5: Generate projection handlers (event → engine writes)
  UserCreated →
    Pebble.Set(userID, record)         [FindUser]
    Bloom.Add(email)                   [CheckEmail]
    SQLite.Upsert(users, record)       [ListActive]
    SQLite.Increment(status_counts, "active", +1)  [CountByStatus]
  (FriendsOf handler ignores UserCreated — only listens to Friendship events)

Step 6: Generate typed read handlers
  FindUser(ctx, FindUser{ID}) → Pebble.Get(ID) → FindUserResult
  CheckEmail(ctx, CheckEmail{Email}) → Bloom.Test(Email) → CheckEmailResult{Taken}
  CountByStatus(ctx, CountByStatus{}) → SQLite.Scan(status_counts) → CountByStatusResult
  ListActive(ctx, ListActiveUsers{Limit}) → SQLite.Query(users, WHERE status='active', LIMIT) → ListActiveUsersResult
  FriendsOf(ctx, FriendsOf{ID, Depth}) → Neo4j.Traverse(ID, Depth) → FriendsOfResult

Step 7: Validate + warn
  - Every query has an assigned engine ✓
  - Check for degraded patterns (Graph on SQL CTE → warn)
  - Check write amplification (5 projections per UserCreated → warn if >3)
  - Check memory constraints (Bloom filter size vs available RAM)

OUTPUT:
  - Projection plan (which engine, which structure, which indexes)
  - Auto-generated write handlers (event → engine writes)
  - Auto-generated read handlers (query → engine reads → result)
  - Startup diagnostics (warnings, degradation, costs)
```

---

## 8. Concrete Examples

### Example 1: Single SQLite (Development)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/app.db
```

```
Planner plan for all 5 queries:

  FindUser:      SQLite table users_by_id, PK index          O(logN) ✓
  CheckEmail:    SQLite UNIQUE index on email                O(logN) ✓
  ListActive:    SQLite table users + idx_status             O(logN) ✓
  CountByStatus: SQLite rollup table status_counts           O(1)    ✓
  FriendsOf:     SQLite junction table + recursive CTE       O(N)    ⚠ DEGRADED

⚠ FriendsOf: using SQL recursive CTE for graph traversal. O(N) per query.
  Add Neo4j or a graph engine for deep traversal at scale.
```

### Example 2: SQLite + Pebble + Neo4j (Production)

```yaml
engines:
  sqlite:
    driver: sqlite
    dsn: /data/indexes.db
  pebble:
    driver: pebble
    dsn: /data/kv
  neo4j:
    driver: neo4j
    dsn: bolt://graph:7687
```

```
Planner plan for all 5 queries:

  FindUser:      Pebble hash index users_by_id               O(1)    ✓ OPTIMAL
  CheckEmail:    Pebble hash set emails                      O(1)    ✓ OPTIMAL
  ListActive:    SQLite table users + idx_status             O(logN) ✓ OPTIMAL
  CountByStatus: SQLite rollup table status_counts           O(1)    ✓ OPTIMAL
  FriendsOf:     Neo4j User nodes + FRIENDS_WITH edges       O(d)    ✓ OPTIMAL

ALL queries at optimal complexity. Zero degradation.
Three engines. Five projections. One event stream. Zero projection code by the developer.
```

### Example 3: Memory Only (Testing/CI)

```yaml
engines:
  memory: {}
```

```
Planner plan for all 5 queries:

  FindUser:      Memory map[UserID]FindUserResult            O(1)    ✓
  CheckEmail:    Memory map[string]struct{}                  O(1)    ✓
  ListActive:    Memory slice + in-memory filter             O(N)    ⚠ DEGRADED (fine for tests)
  CountByStatus: Memory map[string]int64                     O(1)    ✓
  FriendsOf:     Memory adjacency list                       O(d)    ✓

⚠ ListActive: in-memory filter (O(N)). Fine for tests. Add SQLite for production filtering.
```

---

## 9. What the Developer Writes vs. What They Never Write

### The Developer Writes (3 things):

1. **Event types** — pure Go structs, domain vocabulary
2. **Query types** — input + result structs, named as domain questions
3. **Fold functions** — event → result mapping, pure functions, no storage types

### The Developer NEVER Writes:

| Never                                                          | Why                                          |
| -------------------------------------------------------------- | -------------------------------------------- |
| `database/sql` imports                                         | Engines are operator-provided                |
| `storage/` imports                                             | No storage packages in consumer code         |
| `*sql.DB` handling                                             | Connection management is in engine plugins   |
| SQL DDL (CREATE TABLE, CREATE INDEX)                           | Planner auto-generates from fold result type |
| ViewMapper with column types                                   | No column type declarations at all           |
| IndexSpec declarations                                         | Planner auto-creates from read patterns      |
| kv.ViewStore implementations                                   | Engine plugins implement these               |
| Projection tier selection (Materialize vs Relational vs Graph) | Planner selects                              |
| Engine selection per query                                     | Planner selects based on cost                |
| "Entity" types                                                 | No entities — events + queries only          |
| "Store" interfaces                                             | No stores — each query is independent        |
| Stringly-typed column names in declarations                    | All typed Go                                 |

---

## 10. Open Design Decisions

### Decision 1: How Does the Planner Learn Filter/Sort Criteria for Scans?

For `ListActiveUsers`, the planner needs to know: filter on Status, sort on JoinedAt.

**Option A:** Query input carries typed filter fields:

```go
type ListByStatus struct {
    Status string   // ← field name maps to result field name → planner creates index
    Limit  int
}
```

**Option B:** Query declares via typed field accessors:

```go
metaengine.FilterOn(func(r FindUserResult) string { return r.Status }),
metaengine.SortOn(func(r FindUserResult) time.Time { return r.JoinedAt }),
```

**Option C:** Code generation extracts field paths from query handler functions.

**Recommendation:** Start with Option B (typed accessors). It's explicit, compiler-safe, and
the planner can extract the field path from the closure's field access via reflection on the
function signature.

### Decision 2: How Are Multiple Events Handled in One Projection?

A projection like `FindUser` listens to UserCreated, UserSuspended, UserDeleted. These are
three different event types with three different fold functions. The API already supports this:

```go
metaengine.Query[FindUser, FindUserResult]("find_user",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, FindUserResult) { ... }),
    metaengine.On(UserSuspended{}, func(e UserSuspended, prev FindUserResult) FindUserResult { ... }),
    metaengine.On(UserDeleted{}, metaengine.Remove[FindUserResult]()),
)
```

The `On` for UserSuspended receives the previous value (for update semantics). The `On` for
UserDeleted returns Remove (for deletion semantics). The `On` for UserCreated returns a new
value (for insertion semantics).

**Open question:** How does the projection handler atomically update the engine? For Pebble:
one `Set` call. For SQLite: one `UPSERT`. For the counter: one `Increment`. The planner
generates the right call based on the ADT + engine.

### Decision 3: Streaming vs. Slicing for Large Results

`ListActiveUsers` might return 1M users. The query input has `Limit` and `After` (cursor)
for pagination. But what about bulk operations (export all, analytics scan)?

**Proposal:** Each query handler supports two modes:

- `Execute(ctx, input) → output` for bounded results (point lookup, count, paginated list)
- `Stream(ctx, input, fn func(output) error)` for unbounded results (full scans, exports)

The planner generates both. The developer picks which to call. Streaming uses Go iterators
(`iter.Seq2[Output, error]`) under the hood.

### Decision 4: Hot-Reload (Adding/Removing Engines at Runtime)

From the "Perfect Software Architecture" requirements (#2, #49): the planner must support
adding/removing engines WITHOUT restart.

**Flow:**

1. Operator adds a Pebble engine to a running app (via config reload or API call)
2. Planner re-plans: FindUser could now be O(1) on Pebble instead of O(logN) on SQLite
3. Planner creates new Pebble projection for FindUser
4. Background replay: events replayed from the log into the new Pebble projection
5. While replaying: reads continue from SQLite (old projection)
6. When caught up: atomic cutover — reads switch to Pebble
7. Old SQLite projection for FindUser optionally torn down

**This requires:**

- The planner to be re-plannable (not one-shot)
- Projections to support live cutover (dual-read during transition)
- The projection host to support background replay while serving live reads
- The plan to be a live runtime object, not a startup artifact

### Decision 5: What About Queries That Need Data From Multiple Projections?

Example: "Active users who have >5 friends" needs:

- Status from the FindUser projection (Pebble)
- Friend count from the FriendsOf projection (Neo4j)

**Option A:** Declare a new query with a fold that combines both:

```go
metaengine.Query[ActiveUsersWithFriends, Result]("active_with_friends",
    metaengine.On(UserCreated{}, func(e UserCreated) (UserID, Result) { ... }),
    metaengine.On(Friendship{}, func(e Friendship, prev Result) Result {
        prev.FriendCount++; return prev  // denormalized into this projection
    }),
)
```

This creates a third projection that maintains both status and friend count. No cross-engine
read at query time. Write amplification: this projection listens to both UserCreated AND
Friendship events.

**Option B:** The query handler fans out at read time:

```go
// Planner generates a handler that queries both engines and merges
func(ctx, ActiveUsersWithFriends) {
    activeUsers := findUserProjection.Filter(status="active")    // Pebble
    for each user := range activeUsers {
        count := friendsOfProjection.Degree(user.ID)             // Neo4j
        if count > 5 { results = append(results, user) }
    }
}
```

Correct but potentially slow (N cross-engine reads). If engines are local, this is fine. If
remote, the planner warns and suggests Option A (denormalize).

**Recommendation:** Option A (declare a combined projection). It's explicit, no hidden fan-out
costs, and the developer controls the tradeoff. Option B as a fallback for ad-hoc queries.
