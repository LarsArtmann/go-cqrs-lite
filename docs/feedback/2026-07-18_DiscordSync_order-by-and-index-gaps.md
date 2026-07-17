# go-cqrs-lite — Consumer Feedback: ORDER BY, Index Alignment, and Denormalization Gaps

**Consumer:** [DiscordSync](https://github.com/larsartmann/DiscordSync) — Discord backup bot
**Date:** 2026-07-18
**Context:** Adding 9 sort options ("Sort by" dropdown) to the `/attachments` dashboard page. Each sort needs to be index-served for sub-millisecond pagination on 10k+ rows per channel.
**Previous feedback:** [2026-07-12 View-store-as-projection-target](./2026-07-12_DiscordSync_view-store-as-projection-target.md) (covered WHERE clause composition; this report covers the adjacent ORDER BY + index problem)

---

## Executive Summary

Adding user-controlled sorting to a read model exposed a cluster of related gaps in `ViewQuery` and `ViewMapper`. The library correctly identifies the CQRS pattern ("denormalize FK columns in the projection handler") but stops short of providing the tools to execute it well:

- `ViewQuery.OrderBy` accepts a single bare column name — no multi-column tie-breakers, no sort expressions, no mixed ASC/DESC. Stable pagination requires all three.
- `IndexSpec` declares only column names — no per-column direction, no expressions, no partial predicates. So consumers cannot declare the indexes their sorts need.
- The query builder never consults `mapper.Indexes` — the mapper knows the indexes, the query builder ignores them, and nothing warns when a sort regresses to `USE TEMP B-TREE FOR ORDER BY`.
- The library says "denormalize" but provides no helper for the single most common CQRS read-model operation: copying a column from a parent table onto a child for index-served queries.

Each gap has a concrete fix. The fixes are additive (new fields, new helpers) and do not break existing consumers.

**Outcome in DiscordSync:** I stayed with hand-written SQL for this feature. The query is a 3-table JOIN (`attachments → messages → users`) that neither `SQLViewStore` nor `RelationalStore` can express, and even after denormalizing `channel_id` to eliminate the messages JOIN, the ORDER BY requirements (9 sort variants, each with a deterministic tie-breaker, each backed by a direction-specific composite index) exceed what `ViewQuery` can express. This report documents what would need to change for the library to own this query class.

---

## Gap 1: `ViewQuery.OrderBy` cannot express stable pagination (HIGH)

### The problem

`kv/view_store.go:33-54` — `ViewQuery` has `OrderBy string` + `Desc bool`. One column, one direction. `storage/view/query.go:48` renders it as:

```go
fmt.Fprintf(&b, " ORDER BY %s %s", orderCol, dir)
```

This cannot express:

1. **Multi-column tie-breakers** — `ORDER BY size DESC, id DESC`. Stable pagination (LIMIT/OFFSET) requires a deterministic total order; without a tie-breaker, rows with equal sort keys can swap pages between requests. This is not theoretical — it's a correctness bug waiting to happen on any column with duplicate values.

2. **Mixed ASC/DESC** — `ORDER BY size DESC, created_at ASC`. When the tie-breaker's natural order differs from the sort key's direction, the single-`Desc`-bool model breaks. DiscordSync's "smallest size" sort needs `size ASC, id ASC`; "largest" needs `size DESC, id DESC` — two different indexes, two different ORDER BY fragments, both inexpressible.

3. **Sort expressions** — `ORDER BY LOWER(filename)`. Case-insensitive sort is a common product requirement. The current API forces consumers to either sacrifice case-insensitivity (use a bare column to match the index) or sacrifice index alignment (use a raw fragment that defeats `OrderBy` entirely).

### The impact

DiscordSync's 9 sort variants each produce a different ORDER BY fragment:

```go
// internal/db/attachment_sort.go — what the library cannot express
case AttachmentSortOldest:    return "a.created_at ASC, a.id ASC"
case AttachmentSortLargest:   return "a.size DESC, a.id DESC"
case AttachmentSortSmallest:  return "a.size ASC, a.id ASC"
case AttachmentSortFilenameAZ: return "a.filename ASC, a.id DESC"
case AttachmentSortAuthorAZ:  return "LOWER(COALESCE(u.username, '')) ASC, a.id DESC"
```

Every one of these has a multi-column tie-breaker on `id`. Two of them mix directions (`filename ASC, id DESC`). One uses an expression (`LOWER(...)`). None can be expressed via `ViewQuery.OrderBy`.

### Proposed fix

Replace `OrderBy string` + `Desc bool` with `OrderBy []OrderTerm`:

```go
type OrderTerm struct {
    Column   string  // column name (trusted, never user input)
    Desc     bool    // direction
    Expr     string  // optional: raw SQL expression (e.g. "LOWER(filename)") — overrides Column
    NullsLast bool   // optional: NULL handling (SQLite defaults vary by dialect)
}

type ViewQuery struct {
    Conditions []Condition
    OrderBy    []OrderTerm  // was: OrderBy string + Desc bool
    Limit      int
    Offset     int
    RawWhere   string
    RawArgs    []any
}
```

Rendering (in `storage/view/query.go`):

```go
if len(q.OrderBy) == 0 {
    q.OrderBy = []OrderTerm{{Column: keyColumnName}}
}
parts := make([]string, len(q.OrderBy))
for i, t := range q.OrderBy {
    col := t.Expr
    if col == "" { col = t.Column }
    dir := "ASC"
    if t.Desc { dir = "DESC" }
    parts[i] = col + " " + dir
}
fmt.Fprintf(&b, " ORDER BY %s", strings.Join(parts, ", "))
```

**Backward compatibility:** Add a convenience method `q.WithOrder(col string, desc bool)` that sets `OrderBy = []OrderTerm{{Column: col, Desc: desc}}`. Existing callers that set the old `OrderBy` string field would need a migration, but the field is on a struct literal — a compile error points at every call site. The fix is mechanical.

**Why `Expr` is safe:** `Expr` is caller-owned, same trust level as the existing `OrderBy` string column name (already raw-interpolated). The doc comment already says "Column names go in `Condition.Column` (trusted)". `Expr` extends the same trust contract to sort expressions. No injection surface is added — the caller was already responsible for column-name safety.

---

## Gap 2: `IndexSpec` cannot declare direction, expressions, or partial predicates (HIGH)

### The problem

`storage/view/store.go:76-82`:

```go
type IndexSpec struct {
    Name    string
    Columns []string
}
```

This can declare `(channel_id, size)` but cannot declare:

1. **Per-column direction** — `(channel_id, size DESC, id DESC)`. SQLite B-tree indexes are directional; `ORDER BY size DESC` cannot be served from an ASC index without a backward scan (which SQLite supports but is slower on large datasets, and which some engines — including turso/libSQL in some configs — do not support at all).

2. **Expression indexes** — `(channel_id, LOWER(filename))`. SQLite supports `CREATE INDEX ... ON t(LOWER(filename))` but `IndexSpec` has no way to declare it.

3. **Partial indexes** — `CREATE INDEX ... ON t(channel_id) WHERE deleted_at IS NULL`. Partial indexes are dramatically smaller and faster for the common "filter out tombstones" pattern. `IndexSpec` cannot express the `WHERE` clause.

### The impact

DiscordSync declares 6 sort indexes by hand-writing raw DDL in its schema string — outside the library entirely. The `ViewMapper.Indexes` field is unused for this purpose because it cannot express what the sorts need:

```sql
-- What DiscordSync needs (6 indexes, hand-written):
CREATE INDEX idx_attachments_channel_created    ON attachments(channel_id, created_at DESC, id DESC);
CREATE INDEX idx_attachments_channel_size_desc  ON attachments(channel_id, size DESC, id DESC);
CREATE INDEX idx_attachments_channel_size_asc   ON attachments(channel_id, size ASC, id ASC);
CREATE INDEX idx_attachments_channel_filename   ON attachments(channel_id, filename ASC, id DESC);
CREATE INDEX idx_attachments_channel_attempts   ON attachments(channel_id, download_attempts DESC, id DESC);
```

Every one of these has a direction on every column. None can be declared via `IndexSpec`.

### Proposed fix

Extend `IndexSpec`:

```go
type IndexSpec struct {
    Name       string
    Columns    []IndexColumn  // was: []string
    Where      string         // optional: partial-index predicate (trusted SQL)
    Unique     bool           // optional: UNIQUE index
}

type IndexColumn struct {
    Name string  // column name, or expression (e.g. "LOWER(filename)")
    Desc bool    // direction — critical for sort indexes
}
```

Rendering during auto-migrate (`storage/view/store.go:254-265`):

```go
cols := make([]string, len(spec.Columns))
for i, c := range spec.Columns {
    if c.Desc {
        cols[i] = c.Name + " DESC"
    } else {
        cols[i] = c.Name
    }
}
ddl := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s(%s)",
    spec.Name, table, strings.Join(cols, ", "))
if spec.Where != "" {
    ddl += " WHERE " + spec.Where
}
```

**Backward compatibility:** The `Columns []string` field changes to `Columns []IndexColumn`. Existing callers pass `[]string{"email"}` — this becomes `[]IndexColumn{{Name: "email"}}`. A helper `storage.IndexCols("email", "age")` would make the common case (all-ASC, no expressions) as terse as before.

---

## Gap 3: The query builder never consults `mapper.Indexes` (MEDIUM)

### The problem

`storage/view/query.go` — the `Query` method builds SQL from `q.OrderBy` and `s.mapper.Table` but never reads `s.mapper.Indexes`. The mapper declares which indexes exist; the query builder ignores them entirely.

This means:

- No validation that `q.OrderBy` references an indexed column.
- No warning when a sort will produce `USE TEMP B-TREE FOR ORDER BY` (SQLite's marker for an unindexed sort).
- No opportunity to steer the query toward an available index.

### The impact

Consumers who add an index via `ViewMapper.Indexes` get no feedback on whether their queries actually use it. The index may exist but sit unused because the ORDER BY direction doesn't match, or the tie-breaker column isn't in the index, or the planner chose a different path. The consumer discovers this only by running `EXPLAIN QUERY PLAN` manually — which is exactly what DiscordSync had to do this session (see `internal/db/attachment_sort_bench_test.go::TestAttachmentSort_QueryPlans`).

### Proposed fix

Add a development-time (not production) validation hook:

```go
// ValidateQuery checks that the query's ORDER BY can be served by one of the
// declared indexes. Returns nil if OK, or a *PlanWarning describing why the
// sort would fall back to TEMP B-TREE. Intended for tests and CI — not for
// runtime use.
func (s *SQLViewStore[V, K]) ValidateQuery(ctx context.Context, q kv.ViewQuery) (*PlanWarning, error)
```

Where `PlanWarning` carries the EXPLAIN output and a human-readable explanation. Consumers call it in a test:

```go
func TestMessageQuery_IsIndexServed(t *testing.T) {
    warning, err := store.ValidateQuery(ctx, kv.ViewQuery{
        Conditions: []kv.Condition{{Column: "channel_id", Op: kv.OpEq, Value: "ch-1"}},
        OrderBy:    []kv.OrderTerm{{Column: "created_at", Desc: true}},
    })
    require.NoError(t, err)
    require.Nil(t, warning, "query should be index-served: %s", warning)
}
```

This is the "fail CI if my sort regresses to TEMP B-TREE" guardrail that DiscordSync built by hand this session. It belongs in the library.

**Integration with existing tooling:** The `storage/turso/indexing.Advisor` already parses EXPLAIN QUERY PLAN output (`advisor.go:109`). This fix is about (a) making the Advisor available for non-Turso backends (SQLite, Postgres) and (b) wiring it into a `ValidateQuery` method on `SQLViewStore` so consumers don't have to write their own EXPLAIN harness.

---

## Gap 4: No denormalization helper for the #1 CQRS read-model pattern (MEDIUM)

### The problem

The DOMAIN_LANGUAGE doc (line 156) and the RelationalStore doc comment both say: "denormalise FK columns in the projection handler instead of JOINing." This is correct CQRS guidance. But the library provides no helper for executing it.

Denormalizing `parent.channel_id` onto `child.channel_id` requires the consumer to hand-write four things:

1. **Schema column** — `ALTER TABLE child ADD COLUMN channel_id TEXT` (or declare it in `CREATE TABLE`).
2. **Projection handler change** — populate the column on INSERT (`upsertAttachments` now receives `channelID` and writes it).
3. **Backfill migration** — `UPDATE child SET channel_id = (SELECT channel_id FROM parent WHERE parent.id = child.parent_id) WHERE child.channel_id IS NULL`.
4. **Sync-on-parent-change** — when the parent's column changes, propagate to all children (or accept staleness).

DiscordSync did all four by hand this session (`internal/db/migrate.go::backfillDenormalizedFields`, `internal/projection/messages.go::upsertAttachments`, `internal/db/message_attachments.go::InsertAttachment` auto-lookup fallback). Each was a distinct failure mode:

- The schema column was forgotten on two manual test fixtures (`attachment_metadata_helpers_test.go`, `metadata_bench_test.go`) — caught only by compile errors.
- The projection handler needed a signature change (`upsertAttachments` gained a `channelID` parameter) — touched every caller.
- The backfill was idempotent but easy to get wrong (the `WHERE col IS NULL` guard is critical for idempotency).
- The "sync on parent change" step was skipped entirely — DiscordSync accepts that if a message's channel_id ever changes (it can't in Discord, but in general...), the denormalized column drifts.

### The impact

Denormalization is the prescribed solution to the "no JOINs" limitation (Gap 6). But because the library provides no helper, every consumer re-implements the same 4-step pattern, makes the same mistakes, and writes the same backfill SQL. The library's `ViewMapper` is the natural place to declare denormalized columns — it already knows the table, the columns, and the indexes.

### Proposed fix

Add a `Denormalized` field to `ViewColumn`:

```go
type ViewColumn[V any] struct {
    Name     string
    Extract  func(v *V) any
    Index    bool  // existing: create a single-column index
    // DenormalizedFrom names the parent table + column this column mirrors.
    // When set, the store provides:
    //   - A backfill helper: UPDATE child SET col = (SELECT parent.col FROM parent WHERE parent.id = child.<fk>)
    //   - A validation test hook: assert no NULLs remain after backfill
    // The projection handler is still responsible for populating the column on INSERT.
    DenormalizedFrom *DenormSource
}

type DenormSource struct {
    ParentTable  string  // e.g. "messages"
    ParentColumn string  // e.g. "channel_id"
    ForeignKey   string  // e.g. "message_id" — the child column pointing to the parent
}
```

And a helper on `SQLViewStore`:

```go
// BackfillDenormalized populates denormalized columns from their parent tables.
// Idempotent: only touches rows where the column IS NULL. Intended to run
// after Migrate() during schema version bumps.
func (s *SQLViewStore[V, K]) BackfillDenormalized(ctx context.Context) error
```

This doesn't solve the "sync on parent change" problem (that requires either event handlers or triggers, which are out of scope for the store). But it eliminates the first three hand-written steps and makes the denormalization declaration co-located with the rest of the view schema.

---

## Gap 5: `indexing.Advisor` is Turso-only, not wired into the store (LOW)

### The problem

`storage/turso/indexing/advisor.go:109` implements `AnalyzeQuery` — runs `EXPLAIN QUERY PLAN` and detects full-table scans. But:

1. It lives under `storage/turso/` — SQLite and Postgres consumers cannot use it.
2. It is a standalone tool — `SQLViewStore.Query` never calls it.
3. It does not detect `USE TEMP B-TREE FOR ORDER BY` — it focuses on scan types, not sort regressions.

### The impact

DiscordSync wrote its own EXPLAIN harness this session (`internal/db/attachment_sort_bench_test.go`) because the Advisor wasn't available for its SQLite backend and didn't check for the sort-specific regression it needed to catch.

### Proposed fix

1. Move the EXPLAIN-parsing logic from `storage/turso/indexing/` to `storage/` (or `storage/plan/`) so all SQL backends can use it.
2. Extend it to detect `USE TEMP B-TREE FOR ORDER BY` (the sort-regression marker).
3. Expose it via `SQLViewStore.ValidateQuery` (see Gap 3).

---

## Gap 6: No JOIN support means denormalization is mandatory, not optional (CONTEXT)

### The problem

`storage/view/query.go` and `storage/relational/store.go` are both single-table. The RelationalStore doc (`store.go:24-36`) is explicit: "JOINs are not supported."

This is a deliberate, documented design decision — not a bug. But its downstream consequence is worth stating plainly: **every cross-table sort or filter forces the consumer to choose between denormalizing (write amplification, sync complexity) or leaving the library entirely (raw SQL).** There is no middle ground.

### The impact

DiscordSync's attachment query JOINs `attachments → messages → users` for author metadata. Even after denormalizing `channel_id` (to eliminate the messages JOIN for filtering), the users JOIN remains for `u.username`. The two "sort by author" variants cannot be index-served because `username` lives on a different table.

The options are:

1. Denormalize `username` onto `attachments` — write amplification (every username rename updates N attachment rows).
2. Accept `USE TEMP B-TREE FOR ORDER BY` for author sorts — 7× slower than the index-served sorts.
3. Stay with raw SQL and abandon the library for this query.

DiscordSync chose option 2 for now (see status report `docs/status/2026-07-18_00-50_attachment-sort-feature-with-cqrs-denormalization.md`). The library cannot help with this tradeoff.

### No proposed fix

This gap is architectural, not accidental. JOIN support would transform `SQLViewStore` from a materialized-view store into a general-purpose ORM, which is explicitly out of scope. The correct response is to make denormalization easy (Gap 4) and to document the tradeoff explicitly in the DOMAIN_LANGUAGE doc.

**Suggested doc addition** (to DOMAIN_LANGUAGE.md, near line 156):

> **When denormalization isn't worth it:** Not every cross-table query should be denormalized. Columns that change frequently on the parent (e.g. user display names) impose write amplification that may exceed the query-time savings. For these cases, either accept the in-memory sort (if the per-channel row set is bounded — typically <10k rows) or stay with raw SQL outside the store. The store optimizes for the common case (stable parent columns like `channel_id`, `guild_id`); it is not a general-purpose query engine.

---

## What worked well

To end on a positive note — the following aspects of the library were genuinely helpful this session and need no changes:

1. **`Condition` parameterization** — values go through `?` placeholders, column names are caller-trusted. This is the correct security model and it's well-documented at `view_store.go:26-29`.

2. **`RawWhere` escape hatch** — for the predicates `Condition` can't express (OR groups, subqueries). The escape hatch exists and is documented with an example. Consumers aren't trapped.

3. **`ViewMapper` auto-migration** — `CREATE INDEX IF NOT EXISTS` during construction is the right pattern. The index declarations just need to be richer (Gap 2), not the migration mechanism.

4. **The DOMAIN_LANGUAGE doc itself** — it correctly identifies the denormalization pattern and points consumers toward it. The guidance is right; the tooling (Gap 4) is what's missing.

5. **The typed `[V, K]` generics** — `SQLViewStore[MessageView, MessageID]` gives compile-time type safety on keys and values. No `any` in the query path.

---

## Summary table

| Gap                                                   | Severity | Effort | Breaking?                      | Covered by prior feedback?                        |
| ----------------------------------------------------- | -------- | ------ | ------------------------------ | ------------------------------------------------- |
| 1. Multi-column ORDER BY + tie-breakers + expressions | HIGH     | Medium | Field rename (mechanical)      | No (2026-07-12 touched WHERE, not ORDER BY depth) |
| 2. IndexSpec: direction, expression, partial          | HIGH     | Small  | Field type change (mechanical) | No                                                |
| 3. Query builder ignores mapper.Indexes               | MEDIUM   | Medium | No (additive)                  | No                                                |
| 4. No denormalization helper                          | MEDIUM   | Medium | No (additive)                  | No                                                |
| 5. indexing.Advisor is Turso-only                     | LOW      | Small  | No (move + extend)             | No                                                |
| 6. No JOIN support (architectural)                    | CONTEXT  | —      | —                              | Yes (acknowledged in prior docs)                  |

---

## Concrete evidence

All claims verified against go-cqrs-lite source at commit checked out in `/home/lars/projects/go-cqrs-lite/` on 2026-07-18:

- `kv/view_store.go:33-54` — `ViewQuery` struct (single `OrderBy string` + `Desc bool`)
- `storage/view/query.go:48` — `fmt.Fprintf(&b, " ORDER BY %s %s", orderCol, dir)` (single column, raw interpolation)
- `storage/view/store.go:63-82` — `ViewMapper.Indexes []IndexSpec` + `IndexSpec{Name, Columns []string}` (no direction, no expression, no partial)
- `storage/view/query.go` (entire file) — no reference to `s.mapper.Indexes`
- `storage/turso/indexing/advisor.go:109` — `AnalyzeQuery` exists but is Turso-scoped
- `storage/relational/store.go:24-36` — "JOINs are not supported" (documented)
- `docs/DOMAIN_LANGUAGE.md:156` — "denormalise FK columns in the projection handler" (guidance without helper)

DiscordSync's hand-written implementation that this feedback derives from:

- `internal/db/attachment_sort.go` — 9 ORDER BY variants with tie-breakers and expressions
- `internal/db/schema.go` — 6 direction-specific composite indexes declared as raw DDL
- `internal/db/migrate.go::backfillDenormalizedFields` — hand-written denormalization backfill
- `internal/db/attachment_sort_bench_test.go::TestAttachmentSort_QueryPlans` — hand-written EXPLAIN harness
- `docs/status/2026-07-18_00-50_attachment-sort-feature-with-cqrs-denormalization.md` — full session report with benchmark numbers

---

_Feedback generated from a real feature implementation. Every gap was hit in practice, not hypothesized._
