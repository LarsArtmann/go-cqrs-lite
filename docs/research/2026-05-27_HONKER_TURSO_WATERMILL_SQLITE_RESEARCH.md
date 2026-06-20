# Honker, Turso, and Watermill SQLite: Research Summary

> **Date:** 2026-05-27  
> **Scope:** Evaluate SQLite-based pub/sub and task queue options for go-cqrs-lite consumers who want a zero-external-dependency storage path.  
> **Research artifacts:** [russellromney/honker](https://github.com/russellromney/honker), [ThreeDotsLabs/watermill-sqlite](https://watermill.io/pubsubs/sqlite/), [tursodatabase/turso/COMPAT.md](https://raw.githubusercontent.com/tursodatabase/turso/refs/heads/main/COMPAT.md)

---

## TL;DR

| Option                                                         | Single-Machine SQLite Pub/Sub                    | Pure Go / CGO-Free                 | Distributed | Queue / Task Primitives                        | Verdict                      |
| -------------------------------------------------------------- | ------------------------------------------------ | ---------------------------------- | ----------- | ---------------------------------------------- | ---------------------------- |
| **Watermill SQLite** (`wmsqlitemodernc` / `wmsqlitezombiezen`) | Yes                                              | Yes                                | No          | Yes (consumer groups, batching, ack deadlines) | **Recommended**              |
| **Honker**                                                     | Yes (via Rust extension + `PRAGMA data_version`) | **No** (Rust/C extension required) | No          | Yes (durable queue, stream, notify, scheduler) | **Not for Go libraries**     |
| **Turso**                                                      | **No** (no native primitives)                    | N/A                                | Yes         | No                                             | **Incompatible with Honker** |

---

## 1. Honker (russellromney/honker)

### What it is

Honker is a SQLite extension (written in Rust) plus language bindings that add Postgres-style `NOTIFY`/`LISTEN` semantics to SQLite. Ships as a loadable `.so`/`.dylib` and supports Python, Node, Rust, Go, Ruby, Bun, Elixir, C++, .NET, JVM, and Kotlin.

### Architecture

- **Core wake mechanism:** `PRAGMA data_version` polled every 1ms (~3.5 µs/query). SQLite increments this counter on every commit. Monotonic, handles WAL truncation correctly.
- **Queue schema:** `_honker_live` (pending/processing), `_honker_dead` (retry-exhausted), `_honker_notifications` (ephemeral pub/sub). Partial index on `(queue, priority DESC, run_at, id) WHERE state IN ('pending','processing')`.
- **Claim:** single `UPDATE … RETURNING` via partial index; **ack:** single `DELETE`.
- **Transactional outbox by default:** `INSERT INTO orders` + `queue.enqueue(…, tx=tx)` commit or roll back together.

### Strengths

| Feature                             | Why it matters for CQRS                                                                       |
| ----------------------------------- | --------------------------------------------------------------------------------------------- |
| Atomic enqueue with business writes | Event sourcing's holy grail — no lost events between commit and dispatch.                     |
| `PRAGMA data_version` wake          | Beats `stat(2)` and `inotify` — handles WAL truncation, cross-process, ~1–2ms median latency. |
| Partial-index schema                | Claim cost scales with working set, not history size.                                         |
| Cross-language via SQLite extension | Any process using the same `.db` participates regardless of language.                         |
| Streams with per-consumer offsets   | Maps directly to projection runner requirements (`event.GlobalLoader` + `event.Bus`).         |

### Weaknesses

| Concern                         | Impact                                                                                                                                  |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| **Alpha quality**               | README: "better than experimental but not beta-quality yet."                                                                            |
| **Single writer**               | SQLite serializes writes. Event sourcing is append-heavy. Structural ceiling for throughput.                                            |
| **No in-memory DB support**     | `:memory:` doesn't work. Tests need temp files.                                                                                         |
| **Rust/C extension dependency** | Requires Rust toolchain + platform-specific binary artifacts. Not suitable for a Go library that promises "zero external dependencies." |
| **Polling semantics**           | `data_version` every 1ms is cheap polling, but still polling. Fine for projections, not real-time trading.                              |

### Bottom Line for go-cqrs-lite

Honker would make an excellent lightweight storage backend for SQLite consumers, **if** the project could tolerate a Rust/C extension dependency. For a Go SDK library, that dependency is a dealbreaker. The durable stream with per-consumer offsets is the strongest mapping to go-cqrs-lite's projection runner, but the operational cost doesn't justify the benefit when pure-Go alternatives exist.

---

## 2. Turso (tursodatabase/turso)

### What it is

Turso is a **hosted, distributed SQLite platform** built on libSQL. It replicates SQLite databases to the edge. It's SQLite-_compatible_ for queries and transactions, but not SQLite itself.

### The Honker Incompatibility

From [Turso COMPAT.md](https://raw.githubusercontent.com/tursodatabase/turso/refs/heads/main/COMPAT.md), two critical omissions kill any Honker + Turso integration:

| PRAGMA / API            | Turso Status | Honker Depends On                     |
| ----------------------- | ------------ | ------------------------------------- |
| `PRAGMA data_version`   | ❌ No        | Core wake mechanism (1ms poll thread) |
| `load_extension()`      | ❌ No / Stub | Extension loading itself              |
| `sqlite3_commit_hook`   | ❌ No        | Alternate wake path                   |
| `sqlite3_rollback_hook` | ❌ No        | Alternate wake path                   |
| `sqlite3_update_hook`   | ❌ No        | Alternate wake path                   |

**What this means:**

- Honker requires **file-local SQLite** — single process or multi-process sharing one `.db` file on the same filesystem. `PRAGMA data_version` is a per-connection counter SQLite increments on every commit. Turso is distributed and multi-tenant; that counter doesn't exist in the same way.
- You cannot `SELECT load_extension('honker')` on Turso. Full stop.
- Turso is not a drop-in SQLite replacement for anything that depends on C API hooks, extensions, or `PRAGMA data_version`.

### Turso's Queue / Pub/Sub Primitives

**None native.** Turso is a storage layer. If you want queues, streams, or pub/sub, you build it yourself or use a different tool.

### Bottom Line for go-cqrs-lite

- If a consumer uses Turso, they **must** use a separate message broker (NATS, Kafka, etc.) or accept polling-based queue logic in application code.
- Turso does **not** obsolete Honker — they serve completely different architectures. One is a distributed SQLite service; the other is a single-machine pub/sub extension.

---

## 3. Watermill SQLite (ThreeDotsLabs/watermill-sqlite)

### What it is

Watermill's SQLite pub/sub provides two **CGO-free** driver variants for Go:

1. **`wmsqlitemodernc`** — Standard `database/sql` compatible, `modernc.org/sqlite` pure Go implementation. Slower, simpler.
2. **`wmsqlitezombiezen`** — Direct `zombiezen.com/go/sqlite` API, ~6× faster, more stable. Still CGO-free.

Both use the same underlying SQLite engine as Honker (file-local), but implement queue/offset/claim logic in **pure Go** instead of a Rust extension.

### Architecture

- **Schema:** JSON-serialized messages in topic tables + offset tracking tables. Row locking via `unixepoch() + lockTimeout` (since SQLite has no `FOR UPDATE`).
- **Consumer groups:** Supported via `ConsumerGroupMatcher`.
- **Guaranteed order:** Yes.
- **Persistence:** Yes.
- **Exactly-once:** No (at-least-once).
- **Transaction publishing:** Supported — pass a `*sql.Tx` (modernc) or `*sqlite.Conn` (zombiezen) to `NewPublisher()`.

### Strengths vs Honker

| Dimension         | Watermill SQLite            | Honker                                                    |
| ----------------- | --------------------------- | --------------------------------------------------------- |
| Language          | Pure Go                     | Rust extension + thin Go wrapper                          |
| CGO               | **None**                    | Required (or `modernc.org/sqlite` with extension loading) |
| Cross-compilation | **Yes**                     | No (platform-specific `.so`/`.dylib`)                     |
| Build complexity  | `go get`                    | Rust toolchain + extension artifacts                      |
| Consumer groups   | **Yes**                     | No equivalent                                             |
| Maturity          | Beta (stable, tested)       | Alpha                                                     |
| ORM integration   | Via standard `database/sql` | Via custom `db.transaction()` API                         |

### Weaknesses vs Honker

| Dimension                   | Watermill SQLite                               | Honker                                     |
| --------------------------- | ---------------------------------------------- | ------------------------------------------ |
| Wake latency                | Poll-based (configurable interval, default 1s) | ~1–2 ms via `data_version`                 |
| Stream offsets              | Per-consumer-group, not per-consumer-name      | Per-consumer-name (finer-grained)          |
| Scheduler / cron            | **No**                                         | Built-in (`honker_cron_*`, `@every`)       |
| Dead-letter table           | **No**                                         | Native (`_honker_dead`)                    |
| Named locks / rate limiting | **No**                                         | Native                                     |
| Task result storage         | **No**                                         | Native (`honker_result_*`)                 |
| Cross-language interop      | **Go only**                                    | Python, Node, Rust, Go, Ruby, Elixir, etc. |

### Bottom Line for go-cqrs-lite

Watermill SQLite is the **pragmatic choice** for a Go library:

- Zero CGO, zero Rust, zero external runtime dependencies.
- Consumer groups, batch processing, and transaction publishing are already implemented.
- Fits naturally into existing `database/sql` workflows and ORMs.
- Tradeoffs (higher poll latency, no built-in scheduler) are acceptable for a library that defers advanced orchestration to consumers.

---

## 4. What Makes Honker Obsolete (for Go)

Three factors:

1. **Watermill's existing SQLite pub/sub already covers the Go use case.** `wmsqlitemodernc` and `wmsqlitezombiezen` provide durable publish/subscribe with consumer groups, guaranteed order, and transaction publishing — all in pure Go. For Go specifically, Honker adds operational complexity (Rust build, extension loading, platform artifacts) for no functional advantage.

2. **Single-writer is the real ceiling.** Honker's entire value proposition — atomic enqueue with business writes — collapses when you need two machines writing. SQLite serializes writes. If you outgrow one machine, you re-architect anyway (Postgres, NATS, etc.). Honker is a local optimization for a problem with a structural ceiling.

3. **SQLite + triggers + partial indexes is "good enough" in application code.** Honker's schema design is excellent, but it's just SQL. A motivated team can replicate the core queue logic in ~50 lines of schema and Go code. The extension buys convenience, not capability. The `data_version` wake thread is clever but equivalent to `time.Ticker` + `SELECT`.

---

## 5. Recommended Path for go-cqrs-lite

### Option A: Watermill SQLite via `wmsqlitemodernc` (Recommended)

**Effort:** Low  
**Value:** High  
**Why:** Pure Go, no new module needed. Consumers already using Watermill get SQLite pub/sub "for free." Fits the library's "zero external dependencies" philosophy.

### Option B: Build a `storage/turso` module (Future)

**Effort:** High  
**Value:** Medium  
**Why:** Turso is distributed and has no queue primitives, so this requires building a polling-based outbox pattern or integrating a separate broker. Not recommended until Turso adds native pub/sub or change-data-capture (CDC) hooks.

### Option C: Build a `storage/honker` module (Not Recommended)

**Effort:** High  
**Value:** Medium  
**Why:** Adds Rust/CGO burden to a Go library. Schema conflicts with existing `storage.SQLStore`. Cross-language interop is irrelevant for a Go SDK. Honker's alpha status and architectural ceiling make it a risky dependency.

### Option D: Keep Current Architecture (PG for storage, memory for bus)

**Effort:** Zero  
**Value:** Current  
**Why:** Valid. PostgreSQL is battle-tested for event sourcing. Add Watermill SQLite as a documented consumer option for those who need SQLite.

---

## 6. Key Architectural Insight

The gap between **"SQLite-compatible"** (Turso) and **"actually SQLite"** (Honker, Watermill SQLite) is real and unbridgeable:

- **Turso** is a distributed SQLite service. It doesn't support the C API surface that Honker hooks into (`data_version`, extension loading, commit hooks). You cannot run Honker on Turso.
- **Honker** is a single-machine SQLite extension. It cannot scale beyond one machine. It requires CGO/Rust.
- **Watermill SQLite** is a single-machine pure-Go implementation. It sacrifices Honker's sub-millisecond wake latency for the win of zero CGO, zero Rust, and standard `database/sql` compatibility.

**For a Go library, choose pure Go over clever extensions.** The dependency graph and build simplicity matter more than a ~1ms latency improvement on the wake path.

---

## 7. When Each Option Is Correct

| Scenario                                                                            | Right Tool             |
| ----------------------------------------------------------------------------------- | ---------------------- |
| Go application, single machine, zero CGO, SQLite as only database                   | **Watermill SQLite**   |
| Python/Node/Ruby/Elixir application, single machine, shared `.db` file              | **Honker**             |
| Multi-region edge deployment, SQLite-compatible storage, separate broker for queues | **Turso + NATS/Kafka** |
| Production event sourcing, >1 machine writing, complex projections                  | **PostgreSQL + NATS**  |

---

_Research conducted 2026-05-27. Sources: GitHub READMEs, Watermill docs, Turso COMPAT.md._
