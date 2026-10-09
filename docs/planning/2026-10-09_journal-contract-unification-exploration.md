# Journal Contract Unification — Design Exploration (v2 kernel)

> **Status: PROPOSAL / EXPLORATION — not an ADR. No decision taken.**
> Date: 2026-10-09. Origin: owner question ("extract fundamental metaengine interfaces,
> e.g. WAL, into their own generic module projects?"), session
> `docs/status/2026-10-09_02-47_wal-extraction-design-exploration-session-review.md`
> plus its follow-up turn. Blocked on three owner questions (§8).
> If accepted, this becomes an ADR and lands per §7 (v5 core package — NOT a new module).

---

## 1. Problem: one doctrine, three homes (a split brain)

The fleet has three parallel formulations of "ordered, append-only, cursor-readable log",
each with its own position semantics:

| Home                | Surface                                                                               | Position/cursor semantics                                               | Evidence                                                                                                            |
| ------------------- | ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| CQRS read contracts | `event.Journal` / `SeekableJournal` / `StreamingJournal`                              | opaque `id.EventID` cursor; dangling-cursor contract lives in a comment | `event/store.go:110,129`; `event/streaming_source.go:60`; contract comment `event/store.go:122-128`                 |
| Engine contracts    | `metaengine.StreamLogBackend` / `SeqSeekableStreamLog`                                | `int64` seq, gap-tolerant resume tokens                                 | `metaengine/engine.go:468`; `metaengine/seq_seek.go:41`                                                             |
| Generic mechanics   | `LogStore[T,ID]`, `Inserter[T]`/`JournalReader[T]`, `AdapterCore[T]` (ADR-0126 cores) | `afterID` string / "unknown cursor → 0"                                 | `storage/memory/log_store.go:49`; `storage/sql/inserter.go:20`, `journal_reader.go:22`; `system/adapter_core.go:20` |

The mechanics are already shared (ADR-0126). The **contract** is not: a consumer cannot
reason about "what happens when my cursor is unknown/pruned" without reading three
implementations. Same doctrine, three homes, zero types enforcing it.

## 2. Consumer evidence (the payoff test, executed 2026-10-09)

- **cqrs-htmx: ~20 production files** touch journal-position APIs directly —
  `sync_pull.go` (offline sync), `transport/journalsse.go` (deprecated SSE broker,
  ADR-0127), `dashboardui/handlers_audit.go` (audit cursor pagination),
  `usermgmt/*`, `dashboardui/*`, `setup/*`, `systemadapter/*`.
- **go-appkit: zero matches** (thin `system` wrapper — consistent with the
  2026-10-08 consumed-surface analysis in ADR-0152).
- Consumer patterns that must shape the kernel:
  - Capability assertion dance everywhere: `if seekable, ok := journal.(event.SeekableJournal); ok`
    with `ReadAll` + in-memory filter fallback (`sync_pull.go:134`, `transport/journalsse.go:92`).
  - **The `limit+1` dance**: `seekable.ReadFrom(ctx, afterID, limit+1)` to compute
    has-more (`sync_pull.go:222`) and manual `nextCursor = cmds[len-1].ID()` pagination
    (`handlers_audit.go:80,62`). Two consumers re-deriving paging from a raw slice API.
  - **Divergence pain is real**: `handlers_audit.go:92-99` documents command-vs-query
    journal interfaces diverging (`SeekableCommandJournal.ReadFrom` + `id.CommandID` vs
    query equivalents) as accepted duplication.

**Verdict of the payoff test:** there IS a direct fleet consumer of the journal contract
(cqrs-htmx), but it reaches it via `event/` types. Under ADR-0152's v5 topology
(one core module, Tier 0–3 as packages), module-vs-package is moot: `journal` becomes a
**package inside the v5 core**, and `event/`'s journal interfaces become capability
extensions over it. No new go.mod.

## 3. What the ecosystem teaches (verified 2026-10-09)

| System                                              | Position model                                                                                    | Verified facts                                                                                                                                                                                                                             | Teaches us                                                                                                                                                                                                                      |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `github.com/tidwall/wal` v1.2.1                     | caller-supplied `uint64` index, **gapless required** (`ErrOutOfOrder` when index ≠ LastIndex()+1) | `Write(index,data)`, `Read(index)`, `FirstIndex/LastIndex`, `TruncateFront/Back`, `Sync`, `Batch`; sentinels `ErrCorrupt/ErrClosed/ErrNotFound/ErrOutOfOrder/ErrOutOfRange/ErrEmptyLog`                                                    | uint64 positions, explicit bounds reporting, retention, and a sentinel error contract are a proven complete kernel. `Imported by: 49`.                                                                                          |
| etcd `go.etcd.io/etcd/server/v3/storage/wal` v3.7.2 | raft `(term,index)`; segmented `$seq-$index.wal` files, 64 MB cuts                                | cumulative CRC over ALL preceding record protobufs; 8-byte-aligned length fields (torn-write safety); read-mode vs append-mode lifecycle (must `ReadAll` before appending); `ReleaseLockTo(index)` retention; `Repair` truncates torn tail | "WAL" in industry = a **crash-recovery device** (checksums, repair, read-before-append). That is NOT our abstraction — naming must say `journal`. Chained CRC is the reference pattern IF audit-grade integrity is ever wanted. |
| Kafka                                               | per-partition monotonic offsets; retention windows vs consumer lag                                | _(concept-level comparison only — no exact API claims made here)_                                                                                                                                                                          | the retention-vs-catch-up race (§5 I3) is the canonical operational failure mode; resumption policy must be explicit.                                                                                                           |

**Make-vs-buy verdict:** neither library is importable for our shape — tidwall/wal is
file-backed and non-generic (payload `[]byte`, random-access index, gapless-required);
etcd's is raft-typed and recovery-oriented. But their contracts validate the kernel
design below. We adopt the _ideas_ (bounds, sentinels, retention semantics), not the deps
(zero-dep Tier-0 discipline, dep budgets).

## 4. Data model

**Essence:** a totally ordered, immutable, append-only sequence of entries. An entry has
exactly one position; positions only grow; entries never change once appended.

**Invariants (the contract a conformance suite pins):**

1. **I1 — Strict monotonicity, gaps ALLOWED.** Positions strictly increase; unlike
   tidwall/wal we do NOT require gaplessness: ADR-0143 deliberately keeps sequence
   counters advancing across engine resets (so pre-reset resumption tokens never skip
   replayed entries). Therefore: no random access by index — reads are cursor-based
   (`after N`), which both `SeqSeekableStreamLog` and `ReadFrom` already are.
2. **I2 — Immutability.** No API mutates or reorders written entries. Retention (§ kernel)
   removes a prefix; it never rewrites.
3. **I3 — Cursor semantics are total.** For any cursor `c`: entries after `c` return in
   order; `c` beyond head → empty page (not error); `c` pruned or unknown → the log's
   configured **missing-cursor policy** (below), never undefined behavior. This is the
   typed promotion of the `event/store.go:122-128` comment + `fromStartWhenMissing` bool
   - AdapterCore "unknown → 0".
4. **I4 — Durability is explicit.** Appended ≠ durable; durability is the `Syncer`
   capability (memory engines legitimately lack it).
5. **I5 — Batch atomicity.** A single `Append(values...)` is all-or-nothing; its returned
   position is the last written.

**Named missing-cursor policy (replaces the bool):**

```go
type MissingCursorPolicy int

const (
	// ReplayFromStart replays from the beginning when the cursor predates First
	// (event-store semantics: late subscribers must see history; storage/memory
	// LogStore's fromStartWhenMissing=true path).
	ReplayFromStart MissingCursorPolicy = iota
	// StartAtHead returns only entries appended after the (unknown) cursor
	// (command/query store semantics: an unknown cursor means "nothing new yet").
	StartAtHead
	// FailPruned errors with ErrPruned (strict audit semantics: silent replay
	// after data loss is worse than a loud failure).
	FailPruned
)
```

**Axes of variation** (each a type parameter or a capability — never a god-interface):
entry type `T`; ordering scope (global vs per-stream partition); durability tier
(memory/SQL/KV/distributed); sync control; retention; live tailing; time-bounded reads;
integrity (chained checksums — owner question Q3).

**Hard-to-change (decide once):** position token kind (numeric seq vs opaque ID —
numeric wins: engine-native, comparable, already what seq_seek does; opaque event-ID
cursors resolve to seq at the CQRS layer, as AdapterCore already does); missing-cursor
policy shape; batch atomicity; `Page` read shape (kills the `limit+1` dance).

## 5. Proposed kernel (v2 — incorporates all 11 blind-spot findings)

```go
// Package journal is the fleet's unified append-only log contract: the typed
// home of the position/cursor doctrine that event/, metaengine/, and the
// ADR-0126 generic cores currently each formulate separately.
package journal

import (
	"context"
	"time"
)

// Position is an engine-assigned, strictly monotonic sequence value within one
// log. Gaps are permitted (engine resets keep counters advancing, ADR-0143),
// so Position is a cursor token, never a random-access index. Zero is the
// "read from the start" cursor.
type Position uint64

// Entry pairs a value with its position.
type Entry[T any] struct {
	Pos   Position
	Value T
}

// Page is the result of one cursor read: entries in position order, the next
// cursor, and whether more entries exist. This shape exists because two real
// consumers (cqrs-htmx sync_pull.go, dashboardui/handlers_audit.go) currently
// hand-roll it via limit+1 reads and manual last-ID cursors.
type Page[T any] struct {
	Entries []Entry[T]
	Next    Position
	HasMore bool
}

// Log is the minimal kernel. Implementations MUST satisfy I1-I5.
type Log[T any] interface {
	// Append writes values atomically (all-or-nothing) and returns the last
	// written position. Concurrent Append calls are serialized by the
	// implementation; the returned positions are totally ordered.
	Append(ctx context.Context, values ...T) (Position, error)
	// Read returns up to limit entries strictly after the cursor, per the
	// log's configured MissingCursorPolicy when the cursor is pruned/unknown.
	Read(ctx context.Context, after Position, limit int) (Page[T], error)
	// Bounds reports the current [first, last] positions; ok=false when empty.
	// (tidwall/wal's FirstIndex/LastIndex pattern, verified §3.)
	Bounds(ctx context.Context) (first, last Position, ok bool, err error)
}

// Syncer makes durability explicit (I4): appended becomes durable on Sync.
type Syncer interface {
	Sync(ctx context.Context) error
}

// Truncater is retention: drops the prefix strictly before pos (pos becomes
// First). Never rewrites surviving entries (I2). Cursors into the dropped
// prefix hit the MissingCursorPolicy (I3).
type Truncater interface {
	TruncateBefore(ctx context.Context, pos Position) error
}

// Partitioned is per-stream ordering scope: the event-store shape. The global
// log stays authoritative for total order; streams are ordered sub-views.
type Partitioned[T any] interface {
	AppendTo(ctx context.Context, stream string, values ...T) (Position, error)
	ReadStream(ctx context.Context, stream string, after Position, limit int) (Page[T], error)
}

// Tailer is live follow. The channel is bounded by the implementation; on
// overflow the tailer closes the channel with a terminal ErrOverrun entry
// sentinel error path and the client re-tails from the last seen position
// (policy: never silently drop entries — at-least-once, ordered).
type Tailer[T any] interface {
	Tail(ctx context.Context, after Position) (<-chan Entry[T], func())
}

// TimeBounded is time-axis reads (event.EventSource.LoadToTimestamp precedent).
// Position order remains authoritative; time bounds are a filter, not an
// alternative ordering.
type TimeBounded[T any] interface {
	ReadFromTime(ctx context.Context, from time.Time, limit int) (Page[T], error)
}
```

Error contract (sentinels, errorfamily-mapped at the CQRS layer):

```go
import "errors"

var (
	// ErrClosed is returned by every method after Close.
	ErrClosed = errors.New("journal: closed")
	// ErrPruned is returned by Read/Truncater paths under FailPruned when the
	// cursor predates First (I3).
	ErrPruned = errors.New("journal: cursor pruned")
	// ErrOverrun signals a Tailer that fell behind retention or buffer bounds.
	ErrOverrun = errors.New("journal: tail overran")
)
```

Two structural rules complete the kernel (full signatures deferred to the future ADR
since they involve internals): **capability-preserving decoration** —
`journal.Decorate(log, ...)` must forward every detected capability, because ADR-0126's
core lesson is that hand-written wrappers silently drop optional capabilities (the old
`encryptedStore` lost `MultiSink`); and **capability self-description** — implementations
expose their capability set as data (the `Profile()`/`VectorPathReporter` precedent), so
the conformance suite and cqrs-htmx-style UIs can render the matrix mechanically instead
of probing.

## 6. Capability matrix (current backends, honestly)

| Backend                           | Kernel | Partitioned          | Tailer                        | Truncater              | Syncer          | TimeBounded         |
| --------------------------------- | ------ | -------------------- | ----------------------------- | ---------------------- | --------------- | ------------------- |
| `storage/memory.LogStore`         | yes    | yes (`TrackStreams`) | no (in-proc watcher possible) | no                     | no              | filter-only         |
| SQL (`JournalReader`/`Inserter`)  | yes    | yes (stream key)     | no (poll-based)               | DELETE-prefix possible | yes (tx)        | yes (timestamp col) |
| bbolt journal                     | yes    | via keyspace         | no                            | tx delete              | yes             | no                  |
| Pebble                            | yes    | via prefix           | no                            | compaction-ish         | yes (WAL!)      | no                  |
| `system.AdapterCore` over engines | yes    | yes                  | engine Watcher where present  | engine-specific        | engine-specific | no                  |

No backend gets Tailer for free — that is fine: the capability exists so the contract
has a home for `CatchUpSubscriber`-style consumers, not to force implementations.

## 7. Where this lands (ADR-0152-conformant)

- **NOT a new v4 module.** ADR-0152 retired per-train independence for Tier 0–3 (one
  v5 core module, lockstep versioning, drivers stay modular). Adding `wal/v4` now would
  be a new train that must be re-merged at the v5 cut. _(This document therefore
  explicitly REVERSES the session's earlier "Option A: in-repo Tier-0 wal/ module" —
  that option predates reading ADR-0152 at source.)_
- **Lands as `github.com/larsartmann/go-cqrs-lite/v5/journal`** (package in the one
  core module), with `event.Journal`/`SeekableJournal`/`StreamingJournal` re-expressed
  as thin capability wrappers over `journal.Log[record-flavored T]`, and the ADR-0126
  cores keeping their mechanics but satisfying the typed contract. The
  dangling-cursor comment (`event/store.go:122-128`) is promoted into `MissingCursorPolicy`.
- **Interim (v4.x): design stabilization only.** No backport; v4 trains tag fixes only.
- **Conformance:** extend the store-test-suite pattern (`storage/store_testsuite_test.go`)
  into a journal capability suite every implementation runs; pin I1–I5 + the policy
  matrix.

## 8. Open decisions — the three owner questions (blocking)

1. **Q1 — Second consumer?** cqrs-htmx is the only direct consumer, and it consumes via
   `event/`. If no non-CQRS fleet consumer exists, `journal` stays a v5-core package and
   NEVER becomes a sibling repo (ADR-0128 pattern). If one is planned (audit trail?
   monitor365?), say so — it changes the surface ambition.
2. **Q2 — Timing:** land with the v5 core cut (recommended — it is exactly the kind of
   contract the v5 core wants settled early, and the codemod wave touches the consumers
   anyway) or defer post-v5?
3. **Q3 — Audit-grade integrity?** cqrs-htmx already renders an audit UI over command
   journals. If the fleet ever needs tamper-evidence, chained checksums (etcd's
   cumulative-CRC pattern, §3) must be a day-one capability — they cannot be retrofitted
   onto a live log.

## 9. Rejected alternatives (recorded with reasons)

- **New standalone public project "for many people"** — contradicts the 2026-10-08
  fleet-first scope decision (ADR-0152 #1); external adoption is not a goal.
- **Extract `metaengine.StreamLogBackend` as-is** — `[]any` values violate the
  strong-types contract (AGENTS.md #12); it is an engine-internal seam, not a product.
- **Adopt tidwall/wal or etcd's wal** — wrong shape (§3): non-generic payloads,
  gapless-required / raft-typed, recovery-oriented; would drag a dep into a zero-dep
  tier for mechanics the ADR-0126 cores already own.
- **Tie the journal contract to `record.Record`** — drags the CQRS type graph into every
  consumer; `T` stays generic, `event/` adapts at the boundary.
- **Name it `wal`** — "WAL" promises the crash-recovery protocol (fsync-ahead, torn-write
  repair, read-before-append) that lives inside SQLite/Pebble, not in our abstraction.
  `journal` is the honest word and the in-repo vocabulary.

## 10. Verification appendix (per verify-external-claims discipline)

| Claim                                                                                                             | Status                        | Source                                                                                                                                                                                                    |
| ----------------------------------------------------------------------------------------------------------------- | ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| tidwall/wal v1.2.1 API surface (Write/Read/First/Last/Truncates/Sync/Batch, sentinels, gapless `ErrOutOfOrder`)   | verified 2026-10-09           | pkg.go.dev/github.com/tidwall/wal (raw fetch)                                                                                                                                                             |
| etcd wal v3.7.2 (segments 64 MB `$seq-$index.wal`, cumulative CRC, read-before-append, `ReleaseLockTo`, `Repair`) | verified 2026-10-09           | pkg.go.dev/go.etcd.io/etcd/server/v3/storage/wal (raw fetch; path found via pkg.go.dev search after a constructed-URL 404 was caught and discarded)                                                       |
| Kafka row                                                                                                         | concept-level only, hedged    | no exact API strings claimed                                                                                                                                                                              |
| All in-repo file:line citations                                                                                   | verified 2026-10-09           | direct grep/view (`event/store.go:110,129`, `metaengine/engine.go:468`, `storage/memory/log_store.go:49`, `system/adapter_core.go:20`, cqrs-htmx `sync_pull.go:134,222`, `handlers_audit.go:62,80,92-99`) |
| ADR-0151/0152 content                                                                                             | verified at source 2026-10-09 | `docs/adr/0151-*.md`, `docs/adr/0152-*.md`                                                                                                                                                                |
