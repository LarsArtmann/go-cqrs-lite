# go-cqrs-lite — Consumer Feedback Round 3 (DiscordSync)

**Consumer:** [DiscordSync](https://github.com/LarsArtmann/DiscordSync) — Discord backup bot
**Version used:** commit `f9e0e0bb` (between v3.7.3 and v3.7.4, 15 direct module imports)
**Previous feedback:** [2026-07-10 DiscordSync leverage review](./2026-07-10_DiscordSync_leverage_review.md)
**Date:** 2026-07-12
**Context:** Attempting to eliminate ALL hand-written SQL from DiscordSync by migrating every read model to `view.SQLViewStore`. This report documents the structural gaps that prevent full migration.

---

## Executive Summary

`view.SQLViewStore` is the right primitive for event-sourced projections: it provides typed, queryable materialized views without ORM overhead. But it's currently modeled as a **generic KV-with-columns store** rather than a **projection materialization target**. Five structural gaps force consumers to fall back to hand-written SQL for ~15% of their query surface — and that 15% is where the hardest, most error-prone code lives.

Each gap has a concrete fix that **strengthens the ES/CQRS contract** rather than adding ORM features.

---

## Gap 1: No first-class denormalized blob support (HIGH)

### The problem

Event-sourced projections denormalize by nature. A `MessageProjection` consumes `MessageCreated`, `EmbedAdded`, `StickerAdded`, `PollCreated` events and must materialize a single view containing the message AND all its children. The correct ES pattern is:

```
Events → Projection accumulates state → viewStore.Set(messageID, &materializedView)
```

`SQLViewStore` supports `BLOB` columns via manual `ViewMapper`, but `AutoMapper` — the zero-config path — maps `[]byte` to `TEXT`, not `BLOB`. There is no struct tag hint, no type hint, no way to say "this field is a serialized blob."

### The impact

DiscordSync has 4 child entities (embeds, stickers, polls, attachments) that should live as denormalized blobs inside their parent message view. Without first-class blob support, we're forced to either:

1. Use manual `ViewMapper` for every entity with blobs (defeats AutoMapper's purpose)
2. Keep separate tables + JOINs (the exact pattern ViewStore should eliminate)
3. Store CBOR/JSON as TEXT (works, but column type lies to the database — can't use SQLite's BLOB I/O optimization)

### Proposed fix

**Option A — AutoMapper struct tag:**

```go
type MessageView struct {
    ID        string    `view:"key"`
    ChannelID string    `view:"channel_id"`
    Payload   []byte    `view:"payload,type=BLOB"`  // type hint in tag
}
```

**Option B — AutoMapper type registration:**

```go
// In view/auto.go — add []byte to the type map
case reflect.Slice:
    if rt.Elem() == reflect.TypeFor[byte]() {
        return "BLOB", false
    }
```

Option B is a 3-line change. `[]byte` → `BLOB` is the only sensible mapping.

---

## Gap 2: No composite entity key support (HIGH)

### The problem

The primary key is always `key TEXT PRIMARY KEY` (single column, string). But many event-sourced projections naturally key on **composite entity identity**:

- `GuildMember` = `(guild_id, user_id)` — two independent IDs form the identity
- `VoiceState` = `(guild_id, user_id)` — same pattern
- `Reaction` = `(message_id, user_id, emoji)` — triple composite key
- `MemberRole` = `(guild_id, user_id, role_id)` — junction table

The workaround is encoding composite keys into a string (`"guild:123:user:456"`), but this makes queries on individual key parts impossible without also storing them as separate indexed columns — duplicating data.

### The impact

DiscordSync has 4 entities with composite keys. Each requires:

1. A custom `fmt.Stringer` key type that encodes/decodes the composite
2. All filterable key parts duplicated as data columns anyway
3. Application-level key construction at every call site

### Proposed fix

**Allow multi-column primary keys in `ViewMapper`:**

```go
type ViewMapper[V any] struct {
    Table    string
    KeyColumns []KeyColumn   // NEW: replaces implicit single "key" column
    Columns  []ViewColumn[V]
    // ...
}

type KeyColumn struct {
    Name string
    Type string  // "TEXT", "INTEGER", etc.
}
```

The generated DDL becomes:

```sql
CREATE TABLE guild_members (
    guild_id TEXT NOT NULL,
    user_id  TEXT NOT NULL,
    -- data columns...
    PRIMARY KEY (guild_id, user_id)
)
```

`Set` and `Get` would accept the view struct and extract key columns from it
(via `KeyColumn.Extract` functions, parallel to `ViewColumn.Extract`).

This is a breaking change to the `K fmt.Stringer` type parameter — but the
generic constraint could be relaxed to `K any` with the key extracted from
the value itself (the projection already has the full struct).

**Alternative (non-breaking):** Add `MultiKeyViewStore[V any]` alongside
the existing `SQLViewStore[V, K]`. Same CRUD methods, but `Set`/`Get` take
a `[]any` key slice. This keeps the simple case simple.

---

## Gap 3: Tombstone is bool-only, projections need timestamps (MEDIUM)

### The problem

`TombstoneQuerier` provides `QueryByTombstone(excludeTombstoned, onlyTombstoned bool)` — a boolean flag. But event-sourced projections model deletion as an **event with a timestamp**:

- `MessageDeleted { deletedAt: 2026-07-12T10:00:00Z }` — soft delete
- `MemberLeft { leftAt: ... }` — soft delete
- `VoiceStateRemoved { removedAt: ... }` — hard delete

Consumers need to query "show messages deleted after 2026-07-01" or "exclude messages deleted in the last 24h." The bool-only tombstone can't express any time-based filtering.

### The impact

DiscordSync stores `deleted_at`, `left_at`, `archived_at` timestamps on multiple views. Every query must manually add `WHERE deleted_at IS NULL` or `WHERE deleted_at > ?` conditions — duplicating tombstone logic across every query site, with no library support.

### Proposed fix

**Extend tombstone to support timestamp columns:**

```go
type ViewMapper[V any] struct {
    // ...
    TombstoneColumn string  // existing: bool column name
}
```

Allow the tombstone column to be either:

- `INTEGER` (bool: 0 = active, 1 = tombstoned) — current behavior
- `TEXT` (nullable timestamp: NULL = active, non-null = tombstoned) — NEW

When the tombstone column is `TEXT`/nullable, `QueryByTombstone` generates
`WHERE tombstone_col IS NULL` (active) or `WHERE tombstone_col IS NOT NULL`
(tombstoned). This is a natural generalization — a NULL check subsumes the
bool case (store `"1"` for deleted, `NULL` for active).

Alternatively, add `OpIsNull` and `OpIsNotNull` to the `Operator` enum so
consumers can express nullable-column filtering in `ViewQuery.Conditions`
directly. This is more general and fixes the gap without tombstone-specific
logic:

```go
Conditions: []kv.Condition{
    {Column: "deleted_at", Op: kv.OpIsNull},
}
```

---

## Gap 4: Query conditions are AND-only, no OR (MEDIUM)

### The problem

`ViewQuery.Conditions` are joined with `AND` only. There is no `OR`, no
negation, no parenthesised groups. Event-sourced read models frequently need
disjunctive filters:

- "messages in channel A **OR** channel B" (for multi-channel views)
- "attachments with `download_status = 'pending'` **OR** `download_status = 'failed'`"
  (the existing workaround is `OpIn`, which works for equality but not for
  mixed operators)

### The impact

DiscordSync has ~8 queries that need `OR` or mixed conditions. Each falls
back to hand-written SQL. The `OpIn` workaround covers equality-only cases,
but anything like `(status = 'pending' AND attempts < 5) OR status = 'failed'`
is inexpressible.

### Proposed fix

**Option A — Condition groups:**

```go
type ViewQuery struct {
    Where   WhereClause   // replaces Conditions
    OrderBy string
    Desc    bool
    Limit   int
    Offset  int
}

type WhereClause interface{ isWhereClause() }

type AndClause struct{ Clauses []WhereClause }
type OrClause  struct{ Clauses []WhereClause }
type NotClause struct{ Clause WhereClause }
type Predicate struct{ Column string; Op Operator; Value any }
```

This is the most flexible but requires a builder API to stay ergonomic.

**Option B — Keep Conditions flat, add a `Chain` field:**

```go
type Condition struct {
    Column string
    Op     Operator
    Value  any
    Values []any
    Chain  ChainOp  // new: "AND" (default) or "OR" — how this condition joins to the previous one
}
type ChainOp string
const ( ChainAnd ChainOp = "AND"; ChainOr ChainOp = "OR" )
```

Simpler but can't express parenthesised precedence.

**Option C — Raw WHERE fragment (escape hatch):**

```go
type ViewQuery struct {
    Conditions  []Condition
    RawWhere    string         // NEW: raw SQL fragment, appended after Conditions
    RawArgs     []any          // NEW: args for RawWhere
}
```

This is the pragmatic escape hatch — not type-safe, but unblocks any query
without library changes. Consumers use it sparingly for the 5% of queries
that don't fit the structured API.

---

## Gap 5: No aggregate view pattern for event-driven counters (MEDIUM)

### The problem

Event-sourced systems maintain **incremental aggregates** — counts, totals,
summaries updated on every event. These are read models built from events,
not computed at query time. `SQLViewStore` has `Count()` but no pattern for
maintaining a counter view that's updated incrementally.

DiscordSync has a `GetStats` query that runs 24 correlated `COUNT(*)`
subqueries on every dashboard refresh. The correct ES pattern is a
`StatsProjection` that increments counters on each event:

```
MessageCreated → statsView.Messages++
AttachmentDownloaded → statsView.AttachmentsDownloaded++
```

`SQLViewStore` can store the stats view, but there's no library support for
the **read-modify-write** pattern (load current stats, modify, store). The
consumer must `Get` → mutate → `Set` with no atomicity guarantee.

### The impact

Without atomic read-modify-write, concurrent projections can lose counter
updates. DiscordSync works around this by running all projections through
a single DB connection (`MaxOpenConns=1`), but that's a workaround, not a
solution.

### Proposed fix

**Option A — `Update` method with merge function:**

```go
// Atomically read-modify-write a view value
func (s *SQLViewStore[V, K]) Update(
    ctx context.Context,
    key K,
    merge func(current *V) (*V, error),  // called with current value (or zero value if not found)
) error
```

Internally implemented as: `BEGIN tx → SELECT ... FOR UPDATE → merge() → UPDATE ... → COMMIT`.

This is the ES-native pattern for incremental aggregates — the merge function
applies the delta from a single event.

**Option B — Separate `AggregateStore` type:**

```go
type CounterView struct {
    Key    string `view:"key"`
    Value  int64  `view:"value"`
}

store, _ := storage.NewSQLiteViewStore[CounterView, StringKey](db, mapper)
store.Increment(ctx, "messages_total", 1)  // atomic counter increment
```

Specialized for the counter/aggregator use case. Simpler API, but less
general.

---

## Summary

| #         | Gap                                        | Priority | LOC eliminated in DiscordSync      | Fix size |
| --------- | ------------------------------------------ | -------- | ---------------------------------- | -------- |
| 1         | No first-class BLOB support in AutoMapper  | HIGH     | ~100 (enables denormalization)     | 3 lines  |
| 2         | No composite entity keys                   | HIGH     | ~80 (4 entities)                   | Medium   |
| 3         | Tombstone is bool-only                     | MEDIUM   | ~30 (nullable timestamp filtering) | Small    |
| 4         | Query conditions are AND-only              | MEDIUM   | ~40 (disjunctive filters)          | Medium   |
| 5         | No atomic read-modify-write for aggregates | MEDIUM   | ~50 (stats counters)               | Medium   |
| **Total** |                                            |          | **~300 LOC**                       |          |

### What does NOT need fixing

These were initially on the list but are **not ViewStore gaps** — they're
separate concerns:

- **Full-text search (FTS5/Tantivy)** — Search indexes are a separate
  projection output, not a ViewStore feature. A `SearchProjection` writes
  to FTS5/Tantivy independently.
- **Complex JOINs** — Eliminated by denormalization. The projection owns
  the full view shape (Gap 1's blob support makes this trivial).
- **Schema evolution** — Event-sourced projections evolve by replaying
  events. `DeleteAll()` + replay is the correct pattern. No ALTER TABLE
  needed.
- **PRAGMA/VACUUM** — Infrastructure, not projection concern.

### Relationship to previous feedback

Gaps 1–3 from the [2026-07-10 leverage review](./2026-07-10_DiscordSync_leverage_review.md)
(VersionedSeekableJournal, SSEBroker WithPayloadTransform, SQLite DLQ store)
are about the **event pipeline**. This report's gaps are about the
**projection output**. Together they cover the full CQRS pipeline:

```
Events → [pipeline gaps, round 2] → Projection → [output gaps, this round] → Read Model
```

Fixing all 8 gaps would allow DiscordSync to eliminate 100% of its
hand-written SQL (~229 statements across 37 files).

---

## Appendix: Maintainer Decision (2026-07-12)

**Verdict:** The feedback is high quality — real consumer pain, concrete
numbers, concrete fixes. But its unspoken premise is wrong: it frames
everything as a ViewStore gap, when the library already has a three-tier
projection model
(`Materialize`/`SQLViewStore` → `RelationalProjection` → `GraphProjection`).
Four of the five "gaps" are the consumer reaching for the wrong tier or asking
the library to become a query builder.

### What was shipped

| Change                                                                         | Files           | LOC |
| ------------------------------------------------------------------------------ | --------------- | --- |
| Gap 1: `[]byte` → `BLOB` in AutoMapper (`storage/view/auto.go`)                | 1 prod + 1 test | ~6  |
| SKILL.md: "Which projection tier?" decision table + `RelationalProjection` row | SKILL.md        | ~15 |

That's it. The root cause was discoverability — the tier decision table was
buried in `readmodels.md` reference, not in the core SKILL.md that every
consumer reads. That is now fixed.

### Gap-by-gap verdicts

#### Gap 1: BLOB support in AutoMapper — FIXED

`[]byte` → `BLOB` is the only correct mapping. The old `TEXT` mapping was a
bug. Three-line fix, plus a roundtrip test proving binary data survives
Set/Get.

#### Gap 2: Composite keys — REJECTED (wrong tier)

`RelationalProjection` already supports composite primary keys — including
triple-key junction tables (`guild_id`, `user_id`, `role_id`) — with atomic
multi-table writes. See AGENTS.md's `member_roles` example.

The proposed fix breaks the `K fmt.Stringer` type parameter — the core
abstraction that makes ViewStore composable with `kv.Store`, `kv.Cache`,
`kv.TypedStore`. That blast radius is unacceptable for a convenience feature.

ViewStore is "ONE record, ONE table, ONE key column per event." Composite
keys are relational territory by design.

**Action for DiscordSync:** Use `RelationalProjection` for `GuildMember`,
`VoiceState`, `Reaction`, `MemberRole`.

#### Gap 3: Tombstone timestamps — REJECTED (composition, not a gap)

Nullable timestamp filtering works today via `stack.Materialize`'s
`OnTombstone` hook — store the timestamp in any column you want. The library
should not add nullable-column operators or overload the tombstone concept
(a deliberate boolean, design principle #11) to serve one consumer's
query pattern. If the consumer needs `WHERE deleted_at IS NULL` on
`SQLViewStore`, that's a column like any other — filter via existing
`kv.Condition` operators.

#### Gap 4: OR conditions — REJECTED (ORM creep)

All three proposed options are the same trap: building a query builder. Once
the library ships `OrClause`/`RawWhere`/`Chain`, every consumer wants more
operators, subqueries, CTEs. Principle #1 is "Library, not framework." The
structured `kv.Condition` API covers the 95% case. The 5% that needs `OR`,
parentheses, or mixed operators is either:

- Relational tier (`RelationalProjection` + `RelationalStore.Query` with raw
  SQL fragments), or
- Hand-written SQL via `*sql.DB` directly.

The library already provides both escape valves. Adding a third inside
ViewStore makes the library worse, not better.

#### Gap 5: Atomic read-modify-write — REJECTED (already composable)

`storage/sql.RunInTx` exists for exactly this. The consumer can compose
`Get → mutate → Set` inside a transaction today:
`RunInTx(ctx, db, span, func(tx *sql.Tx) error { ... })`.

Adding `Update()` to ViewStore bloats the API for a pattern that's already
expressible. For SQLite (DiscordSync's database), `SELECT ... FOR UPDATE`
doesn't even work — it serializes the whole database, which is functionally
identical to the consumer's existing `MaxOpenConns=1` workaround.

### Root cause

The feedback exists because the projection tier decision table was buried in
`readmodels.md` reference, not in the core SKILL.md. A consumer who only
reads SKILL.md would never know `RelationalProjection` exists — so they try
to make ViewStore do everything.

**That discoverability gap is now fixed.** The SKILL.md decision matrix now
includes the tier table with explicit "Do NOT use for" guidance. The next
consumer who reaches for ViewStore when they need composite keys or
multi-table writes will see `RelationalProjection` before filing feedback.
