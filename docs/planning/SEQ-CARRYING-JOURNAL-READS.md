# Seq-Carrying Journal Reads — Design for `StreamLogEntry{Seq, Value}`

> Perf follow-up to the 2026-08-15 `JournalReadFrom` positional-contract fix.
> Correctness is now guaranteed everywhere; this design removes the remaining
> **O(offset) per page** cost on SQL engines and the **cursor-reconstruction
> scans** in the `system` adapters. Design-only — no code written for this
> proposal beyond the referenced current-state facts.
>
> **Status:** PROPOSED (TODO_LIST "Seq-carrying journal reads (perf follow-up)").
> **Date:** 2026-08-16.

---

## 1. TL;DR

`StreamLogBackend.JournalReadFrom` treats `afterSeq` as a **position within
the collection** and SQL engines implement the skip as
`LIMIT ? OFFSET ?` over `WHERE collection = ? ORDER BY seq`. Every resumed
page re-scans all skipped index rows: a paged full drain of an N-entry journal
at page size P costs O(N²/P) row visits.

The fix is to make journal reads **carry their resume token**:
`StreamLogEntry{Seq, Value}` returned by new capability methods
(`JournalReadAllWithSeq`, `JournalReadFromSeq`), where `Seq` is the engine's
own **append sequence — an opaque, strictly-increasing, per-collection-stable
cursor**. Resume becomes `WHERE collection = ? AND seq > ? ORDER BY seq LIMIT ?`
— a pure index range seek on the `(collection, seq)` composite index that
**already exists** on every SQL engine (`idx_stream_log_journal`). KV engines
(pebble/bbolt) already seek by embedded journal key and get the token for free.

Bonus correctness hardening: the current position semantics silently assume
**dense** journals; any future journal-entry deletion breaks
position = seq on KV engines. Token-based resume is gap-tolerant by
construction.

---

## 2. Current State (verified facts)

| Engine | `afterSeq` semantics | Skip mechanism | Per-page cost | Evidence |
| --- | --- | --- | --- | --- |
| sqlite / turso | position | `LIMIT -1 OFFSET ?` | O(offset) | `sqliteengine/stream_log.go:59` |
| pg | position | OFFSET over ordered rows | O(offset) | `pgengine/stream_log.go:92+` |
| mysql / duckdb | position | same OFFSET pattern | O(offset) | dialect twins of the above |
| pebble | per-collection dense seq | `NewIter(LowerBound: journalKey(col, afterSeq+1))` | O(log n) seek | `pebbleengine/stream_log.go:129-141` |
| bbolt | per-collection dense seq | `Cursor.Seek(journalKey(col, afterSeq+1))` | O(log n) seek | `bboltengine/stream_log.go:196-226` |
| memory | per-collection seq | linear `journal[start].seq <= afterSeq` scan | O(n) worst | `memory_stream_log.go:113+` |

Two additional costs hidden above the engine layer:

1. **`system.AdapterCore.ReadFromAfter`** (`system/adapter_core.go:133-165`)
   resolves an item ID to a position by calling `ReadAll` and linearly
   indexing the full slice — O(N) + O(N) allocations per cold call.
2. **`system.EventAdapter`** maintains a `seqCache` (eventID → position) and
   populates it with `afterSeq + i + 1` arithmetic
   (`system/adapter_event.go:247`); a cache miss triggers a full
   `JournalReadAll` (`adapter_event.go:268+`).

The interface contract itself explains why positions exist
(`metaengine/engine.go:481-491`): SQL engines share one global
AUTOINCREMENT/BIGSERIAL counter across collections, so a raw `seq > X`
filter was (correctly) judged unsafe *when X is a position*. The design below
keeps the safety argument and removes the scan.

## 3. The Design

### 3.1 Core type and capability interface

```go
// metaengine

// StreamLogEntry pairs a journal value with its engine resume token.
type StreamLogEntry struct {
    // Seq is the engine's append sequence for this entry: an opaque,
    // strictly-increasing, never-reused token, stable within a collection.
    // Callers must treat it as a cursor (feed it back to
    // SeqSeekableStreamLog.JournalReadFromSeq), never as a position or a
    // count. Values start at 1; 0 means "from the beginning".
    Seq int64
    // Value is the decoded journal payload (same content JournalReadAll returns).
    Value any
}

// SeqSeekableStreamLog is the token-resumable journal capability. Engines
// implement it when they can resume a journal read via an index seek on the
// entry's own sequence, instead of skipping ahead by position.
type SeqSeekableStreamLog interface {
    // JournalReadAllWithSeq is JournalReadAll with each entry's Seq attached.
    JournalReadAllWithSeq(ctx context.Context, collection string) ([]StreamLogEntry, error)

    // JournalReadFromSeq returns up to limit entries with Seq > afterSeq
    // (0 = from the start), in journal order. limit <= 0 means no limit.
    JournalReadFromSeq(ctx context.Context, collection string, afterSeq int64, limit int) ([]StreamLogEntry, error)
}
```

Optional-capability pattern (same as `Transactional`, `AtomicAppender`,
`Prober`): engines adopt incrementally; callers type-assert.

### 3.2 The cursor contract (why raw seq is now safe)

The 2026-08-15 hazard was **semantic mixing**: interpreting a position as a
raw seq (or computing one) across collections with a shared counter
re-delivers or skips entries. The token contract eliminates mixing by
construction:

1. Every entry's `Seq` comes from the **same collection's** prior read
   (or 0). Monotonicity + uniqueness of the append counter mean
   `{e ∈ collection : e.seq ≤ cursor}` is exactly the set already
   delivered.
2. Interleaved appends to *other* collections are invisible: the
   `collection = ?` predicate is part of the seek.
3. Gaps (deletions, failed transactions burning sequence values) are
   harmless to `seq > cursor` — unlike position arithmetic
   (`afterSeq + i + 1` in `adapter_event.go:246`), which silently
   mis-cursors on any non-dense journal.

Per-engine token sources:

| Engine | Token | Monotonicity guarantee | Seek implementation |
| --- | --- | --- | --- |
| sqlite/turso | `INTEGER PRIMARY KEY AUTOINCREMENT` | AUTOINCREMENT never reuses | range scan on `idx_stream_log_journal(collection, seq)` |
| pg | `BIGSERIAL PRIMARY KEY` | sequence, never reused | same index shape (`pgengine/engine.go:133`) |
| mysql | `AUTO_INCREMENT` PK | same | same |
| duckdb | seq column (verify identity semantics) | append-only table ⇒ monotonic | same |
| pebble/bbolt | seq embedded in journal key | per-collection append counter | existing `Seek`/`NewIter` — additionally return the key's seq |
| memory | existing `streamJournal[col][i].seq` field | append counter under mutex | binary search on seq (upgrade from linear) |

All SQL engines already create the required `(collection, seq)` index —
**no schema migration is needed for the seek itself.**

### 3.3 Adapter integration (`system`)

`AdapterCore` and `EventAdapter` gain a fast path:

```go
if seeker, ok := c.Backend.(metaengine.SeqSeekableStreamLog); ok { ... }
```

- `EventAdapter.ReadFrom` populates `seqCache[eventID] = entry.Seq` from the
  returned entries — exact tokens, no position arithmetic, no cold-miss
  full scan after the first page (the first `JournalReadFromSeq(0, P)`
  already seeds the cache with true seqs).
- `AdapterCore.ReadFromAfter` keeps its ID-resolution scan as fallback;
  with the capability it can optionally cache ID→Seq the same way
  (follow-up; not required for the perf win on hot paths).
- Engines without the capability keep today's position-based
  `JournalReadFrom` unchanged. Nothing breaks, nothing is deprecated.

### 3.4 Projectionhost / projectionadapter

`projectionadapter` reads through the `system` adapters, so it inherits the
win transparently. No interface change at the `event.SeekableJournal` layer:
`ReadFrom(afterEventID, limit)` keeps translating ID → cursor internally,
now via the seq cache backed by true tokens.

## 4. Expected impact

Paged full drain of N entries, page size P, on SQL engines:

- Today: Σₖ O(kP) row visits = **O(N²/P)** + O(N) cursor-resolution work in
  adapters.
- After: (N/P) index seeks + N row reads = **O(N + (N/P)·log N)**.

This mirrors the wave-3 event-journal keyset-pagination fix (285x on a
200k-entry SQLite journal, `storage/v4.7.0`) — same pathology, same cure,
different layer. KV engines see no perf change but gain gap-tolerance.

## 5. Rollout plan

1. **Core types + conformance tests** in `metaengine` (`enginetest` matrix):
   - tokens strictly increasing within a collection;
   - `JournalReadFromSeq(cursor of entry k)` ≡ suffix of
     `JournalReadAllWithSeq` after k — including with **interleaved
     collections** and **deleted entries** (gap tolerance);
   - equivalence with position-based `JournalReadFrom` on dense journals.
2. **SQL engines** (sqlite first, then pg/mysql/duckdb/turso): one new query
   each (`WHERE collection = ? AND seq > ? ORDER BY seq LIMIT ?`) + return
   `(seq, value)`.
3. **KV engines**: decode seq from the journal key already being iterated;
   memory engine switches its linear skip to `sort.Search` on seq.
4. **Adapters**: `EventAdapter` token-backed cache; `AdapterCore` fast path.
5. **Benchmarks**: benchkit journal-drain phase @ 100k+ entries, SQL engine,
   before/after with benchstat; record in `docs/BENCHMARKS.md`.

Estimated effort: M (one query + row-scan change per SQL engine; KV engines
trivial; adapters localized).

## 6. Alternatives considered

- **`ROW_NUMBER() OVER (ORDER BY seq)` positional filter**: still scans to
  compute row numbers; no seek. Rejected.
- **Dense per-collection renumbering column** (position materialized as a
  column): schema migration + write-path contention (max+1 per append per
  collection). Rejected — the global seq with collection filter already
  seeks.
- **Change `JournalReadFrom` signature to return tokens**: breaking change
  for all engines at once; the capability-interface path gets the same
  result incrementally. Rejected for v4.
- **Do nothing**: correctness is fine today; but large-journal restarts
  (browser-history-style 200k+ journals) pay quadratic drains on every SQL
  backend. The wave-3 fix proved this class of win is worth taking.

## 7. Risks

- **DuckDB seq semantics** (step 2): verify its seq column is
  never-reused-increasing across transactions before enabling; guard with
  the conformance test, not by reading docs.
- **Dgraph**: journal model differs (predicate-based); capability stays
  unimplemented there until its journal backend exists — the optional
  interface makes that a non-event.
- **Token leakage**: if a caller persists a Seq across engine *migrations*
  (e.g. sqlite → pg), tokens are meaningless. Document: tokens are
  engine-instance-local, unlike positions which are at least
  engine-family-portable on dense journals. Checkpoint stores that persist
  cursors long-term (projectionhost checkpoint) must persist the
  event-ID-based cursor (as today), not the raw Seq.
