# Status Report — 2026-06-28 06:57 CEST

> **Comprehensive project status for go-cqrs-lite**
> Generated after deep research-reading session + 7 commits of work.

---

## Executive Summary

go-cqrs-lite is a **production-grade CQRS/Event Sourcing library/SDK** for Go
with 46 modules, ~48K LOC, 5 stack presets, and 53 research/brainstorming
documents. The project is in excellent shape — all tests pass, build is clean,
and the "hard problems" (immutability, typed IDs, multi-engine support, deployer-
first architecture) are solved. Active work has shifted to **rounding out the
read-model tiers** (relational/document/graph) and **closing documentation gaps**.

### At a Glance

| Metric                      | Value                                         |
| --------------------------- | --------------------------------------------- |
| Modules                     | 46 `go.mod` files                             |
| Go LOC (non-test)           | ~48,156                                       |
| Research/brainstorming docs | 53 files                                      |
| ADRs                        | 38                                            |
| Stack presets               | 5 (memory, sqlite, pebble, postgres, turso)   |
| Tests                       | All passing (0 failures)                      |
| Open TODO items             | 6 (4 blocked on upstream, 2 design proposals) |
| Last release                | v3.1.0                                        |

---

## A. FULLY DONE (Green)

### Core Architecture (Rock Solid)

- **Event model** — Immutable `*ImmutableEvent`, 19 construction options, typed payloads, versioned streams. The foundation everything builds on.
- **Branded IDs** — 8 phantom-typed IDs (EventID, UserID, CommandID, etc.) backed by ULID. `AggregateID` is string-backed (deliberate — supports SHA-256 derivation + domain-specific names) with strict validation now available.
- **Decider pattern** — Pure `Decider[State]` with `Apply` fold function. `TypedDecider[State, Cmd]` binds command type at compile time.
- **Sink/Source split** — `EventSink` + `EventSource` + `Store` composite (ISP applied across event/command/query/snapshot/checkpoint).
- **Journal** — Cross-aggregate `ReadAll` + `SeekableJournal.ReadFrom` for projection catch-up.
- **Tombstones** — Tri-state (`Active`/`Tombstoned`/`Undetermined`), metadata-based, O(1) detection. No `Delete` on Store.
- **5-family error taxonomy** — Rejection / Conflict / Transient / Infrastructure / Corruption with `go-error-family`.
- **CBOR codec** — Deterministic encoding, ~17% faster encode / ~38% faster decode than JSON, direct signing safety win.

### Storage (Multi-Engine, Deployer-First)

- **5 presets** — `memory.New()`, `sqlite.New()`, `pebble.New()`, `postgres.New()`, `turso.New()` — one-line per engine.
- **Multi-DB split** — SQLite, Turso, and Postgres support `WithEventDB`/`WithQueryDB`/`WithViewDB` for concern isolation.
- **Pebble parity** — Full `SnapshotStore` + `CheckpointStore` + `CommandStore` + `QueryStore` + KV read models via `PebbleBackend`.
- **SQL backend facade** — `NewSQLiteBackend` / `NewSQLBackend` share one `*sql.DB`, lazy store construction.
- **Heterogeneous mixing** — `stack.New(With*)` lets deployers mix engines per concern (Pebble events + SQL views).

### Read-Model Tiers (Three Data-Model Shapes)

- **Relational** — `storage.RelationalProjection`: multi-table, dialect-agnostic, atomic writes. SQL queries (WHERE/ORDER BY/LIMIT).
- **Document/KV** — `kv.TypedStore[T,K]` + `stack.Materialize[V,K]`: one document per key, tombstone-aware projection builder.
- **Graph** — `graph.GraphProjection`: nodes + edges via `GraphSink` (MERGE semantics). In-memory `MemoryDriver` shipped. Neo4j/Memgraph drivers are consumer-pulled sibling modules.

### Cross-Cutting

- **Dead-letter quarantine** — `DeadLetterHandler` callback on `RetryConfig.OnDeadLetter` + `MemoryDeadLetterStore`. Messages no longer silently lost on retry exhaustion.
- **Middleware** — Logging, Retry (with DLQ), Recovery, Validation, Metrics, OTel Tracing+Metrics (command+event+query).
- **Watermill integration** — EventPublisher, CatchUpSubscriber (replay+live+checkpoint), retry middleware, correlation ID.
- **Transport** — SSE (HTTP) with Last-Event-ID reconnection, gRPC (remote command/query dispatch).
- **Catalog** — AsyncAPI, OpenAPI, D2, EventCatalog exporters via reflection.
- **Signing** — HMAC-SHA256, Ed25519, multisig.
- **Encryption** — XChaCha20-Poly1305, AES-256-GCM, codec wrapper, HKDF key derivation.
- **Observability** — OTel helpers (otel/), Prometheus bridge (prometheus/).
- **TypedSnapshot[State]** — Type-safe snapshot adapter (codec-aware).

---

## B. PARTIALLY DONE (Yellow)

### Research → Implementation Gaps

| Feature                              | Status                           | What's Left                                                                                                                                       |
| ------------------------------------ | -------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Metadata pointer→value migration** | ~95% done                        | `*Metadata` → value `Metadata` committed. A few enricher→OTel bridge edge cases may remain.                                                       |
| **Graph module**                     | Phase 1 (in-memory)              | Neo4j/Memgraph driver is "consumer-pulled." No real-backend test exists yet.                                                                      |
| **API surface reduction**            | Partially addressed              | `catalog/` still has 29 string-newtypes (intentional — "good names"). `storage/sql/` still separate package (collapse deferred to major version). |
| **Schema migration system**          | Partial                          | `InitSchema` (CREATE IF NOT EXISTS) + `RelationalSchema.Migrate()`. No versioned migration framework (like goose/atlas).                          |
| **Transport/gRPC**                   | Builds clean, not in go.work     | Blocked on genproto ecosystem conflict (upstream). Works with `GOWORK=off`.                                                                       |
| **sqlc adoption**                    | Phase 1 recommended, not started | Schema extraction to .sql files is highest-value/lowest-regret step.                                                                              |

### Research Documentation

- **19 of 53** research docs are marked RESOLVED/IMPLEMENTED.
- **18 of 53** have no status marker — readers can't tell if proposals are live or historical.
- **16 of 53** are HTML reports (high quality, but not easily greppable for status).

---

## C. NOT STARTED (Red — Flagged in Research)

| Feature                             | Source Doc                                | Impact                                                                                                      |
| ----------------------------------- | ----------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| **Scheduler module**                | `scheduled-event.md`                      | Deadlines, timeouts, cron-style triggers. Infrastructure concern — interface defined, zero implementations. |
| **Deriver module**                  | `derived-event.md`                        | Stateless saga (events → commands). `Deriver` interface proposed, no code.                                  |
| **NATS/Redis transport adapters**   | `nats-jetstream-implementation-plan.html` | ADR-0025 accepted. Zero implementation.                                                                     |
| **Hot-state cache (decider)**       | TODO_LIST                                 | Optional `RepositoryOption` for 100+ cmd/sec aggregates.                                                    |
| **Read-pressure snapshot strategy** | TODO_LIST                                 | Snapshot based on load frequency, not write count.                                                          |
| **Bi-temporal model**               | `time-travel-*.md`                        | `ValidAt` dimension for HR/finance/healthcare. Niche but critical for some domains.                         |
| **Event redaction middleware**      | `event-redaction-design-review.html`      | Design reviewed, pattern accepted, no `redaction/` module.                                                  |
| **Event history visualizer**        | `event-history-visualization-tools.md`    | Redux DevTools for server-side ES. Ecosystem gap.                                                           |
| **Postgres Pebble multi-DB split**  | Deployer audit                            | Pebble has no multi-DB split (column-family isolation). Low priority.                                       |

---

## D. TOTALLY FUCKED UP (Critical Issues)

Honest answer: **nothing is critically broken.** All tests pass, the build is
clean, and no data-loss paths exist. But there are two things that need flagging:

### D1. The `transport/grpc` genproto conflict (BLOCKED UPSTREAM)

`transport/grpc` cannot be added to `go.work` because `cockroachdb/pebble` →
`cockroachdb/errors@v1.14.0` pulls the monolithic `google.golang.org/genproto`,
conflicting with grpc v1.81.1's split `genproto/googleapis/rpc`.

- **Impact**: gRPC transport works but must be tested with `GOWORK=off`.
- **Root cause**: Upstream ecosystem conflict ([cockroachdb/errors#79](https://github.com/cockroachdb/errors/issues/79)).
- **Mitigation**: None possible locally. Blocked until upstream resolves.

### D2. The "unbounded Load" OOM risk (deferred to v3)

`store.Load(ctx, aggType, aggID)` loads ALL events for an aggregate with no
`LIMIT`. For aggregates with millions of events, this is an OOM risk.

- **Impact**: Theoretical OOM on very large aggregates.
- **Root cause**: Adding a `limit` parameter is a breaking API change.
- **Mitigation**: Snapshots reduce replay cost. Deferred to next major version.

### D3. SQLite 6-layout timestamp parsing (deferred to v3)

`parseSQLiteTimestamp` tries 6 different time layouts in a loop on every event
scan. The fix (store as `INTEGER` nanos) is a breaking schema change.

---

## E. WHAT WE SHOULD IMPROVE

### Architecture & Type Model

1. **`AggregateID` string-backing unification** — The one type-safety hole. Strict validation now exists, but the backing type should be ULID at the next major version (with `DeriveAggregateID` producing a ULID-encoded hash).
2. **`Snapshot.State` is still `[]byte`** — `TypedSnapshot[State]` exists but the untyped path is still the default in most code. Promote typed API everywhere.
3. **`Pointer-as-optionality` (`*TombstoneMark`, `*Causation`)** — Nil-means-absent is idiomatic Go but pushes present/absent to runtime. A sealed-interface `Presence` type would make absence unrepresentable (low priority).
4. **`catalog/` string-newtypes** — 29 bare `type X string` aliases. Intentional (good names) but they collide with `event.Version` and `event.ContentType` on import. Consider namespacing or deprecation.

### Operational

5. **No durable Dead Letter Store** — `MemoryDeadLetterStore` exists for testing. A SQL-backed `DeadLetterStore` (for production replay/skip) is the natural next step.
6. **No persistent message bus** — Only in-memory. NATS/Redis/SQS adapters would unlock multi-node deployment.
7. **Outbox stores full event JSON** — Events stored twice (events table + outbox table). Reference-based approach would cut storage ~60-80%.
8. **No schema migration framework** — `InitSchema` is CREATE-IF-NOT-EXISTS only. Versioned migrations (goose/atlas style) needed for evolving schemas.

### Documentation

9. **18 research docs have no status marker** — Readers can't tell which proposals are live vs historical. Stamping RESOLVED/REJECTED/SUPERSEDED on each is mechanical but valuable.
10. **`STORAGE_GUIDE.md` is now fixed** (this session), but other guides (`README.md`, `SKILL.md`) should be audited for staleness with the same rigor.

---

## F. Top 25 Things to Do Next

Sorted by **impact ÷ effort** (Pareto order).

### P0 — Highest Impact, Lowest Effort

1. **Stamp RESOLVED/REJECTED on remaining 18 research docs** — mechanical, prevents confusion. (15 min each)
2. **SQL-backed `DeadLetterStore`** — The in-memory version exists; the SQL version is the production path. Follows existing `SQLCheckpointStore` pattern. (1-2 hr)
3. **`example/deployer-first-heterogeneous` needs a test** — The example runs but has no test. Add a `main_test.go` that asserts the output. (30 min)
4. **Audit `README.md` and `SKILL.md` for stale API references** — Same treatment as `STORAGE_GUIDE.md`. (1 hr)

### P1 — High Impact, Medium Effort

5. **Schema extraction to `.sql` files** (sqlc Phase 1) — Highest-value, lowest-regret step from the sqlc analysis. Valuable even without adopting sqlc. (3-4 hr)
6. **Scheduler module (`scheduler/`)** — Interface defined in research, zero code. Start with `MemoryScheduler` (like `MemoryDeadLetterStore`). (1 day)
7. **NATS JetStream transport adapter** — ADR-0025 accepted. Unlocks multi-node deployments. (2-3 days)
8. **Versioned schema migrations** — `RelationalSchema.Migrate()` exists but needs version tracking. Consider embedding atlas/goose. (1-2 days)
9. **Outbox reference-based storage** — Store event IDs in outbox, JOIN to events table on poll. Cuts storage ~60%. (1 day)

### P2 — Medium Impact, Medium Effort

10. **`Deriver` module** — Stateless saga (events → commands). Interface designed, no code. (1-2 days)
11. **Neo4j/Memgraph graph driver** — `graph/` has in-memory driver. A real backend driver validates the abstraction. (2-3 days)
12. **Event redaction middleware** — Design reviewed in research. Composable, policy-driven, type-preserving. (1-2 days)
13. **Collapse `storage/sql/` into `storage/`** — Un-export ~50 internal-only symbols. Breaking, defer to v3. (1 day)
14. **Promote `TypedSnapshot[State]` as primary in all examples** — Doc updated; examples should follow. (2 hr)

### P3 — Lower Priority / Blocked

15. **Resolve genproto conflict** — Blocked upstream (cockroachdb/errors#79).
16. **Hot-state cache (decider)** — Only matters for 100+ cmd/sec aggregates. Profile first.
17. **Read-pressure snapshot strategy** — Subsumed by hot-state cache.
18. **Bi-temporal model (`ValidAt`)** — Niche. Only finance/HR/healthcare.
19. **Pebble multi-DB split** — Column-family isolation. Low value.
20. **Arena allocation experiment** — Blocked on Go arena API stabilization.
21. **JSON v2 codec experiment** — Blocked on Go stdlib stabilization.
22. **Event history visualizer** — Ecosystem gap, not a library concern. Separate project.
23. **AggregateID ULID unification** — Breaking change, defer to v4.
24. **Pointer-as-optionality → sealed Presence** — Type-safety polish, low ROI.
25. **TigerBeetle/ZFS/btrfs backends** — Thought experiments only. Not practical.

---

## G. Top Question I Cannot Figure Out Myself

**#1: Is `graph/` (in-memory driver) the ship target, or is a real Neo4j/Memgraph
driver required before the next release?**

The `graph/doc.go` says "a Neo4j or Memgraph driver lives in a consumer-pulled
sibling module (e.g. graph/neo4j/)". This implies the in-memory driver is
sufficient for v3.x. But the research doc (`graph-db-event-sourcing.html`) and
the deployer-first audit suggest graph is a **first-class tier** alongside
relational and document.

- If in-memory is sufficient → `graph/` is done, document the driver-extension path.
- If a real driver is needed → it's 2-3 days of work and should be in the next release.

This determines whether graph is a **P0 documentation task** or a **P2
implementation task**.

---

## Session Work (2026-06-28)

7 commits in this session:

| Commit     | Description                                                      |
| ---------- | ---------------------------------------------------------------- |
| `6883def1` | Rewrite stale `STORAGE_GUIDE.md` to match current API            |
| `850f717b` | Remove dead `parseSQLiteTimestamp` wrapper (split-brain cleanup) |
| `f25acb9d` | Add `AggregateID` strict validation + document string-backing    |
| `588c45ae` | Stamp RESOLVED on stale research docs                            |
| `dab49266` | Add dead-letter quarantine for retry exhaustion                  |
| `a6160538` | Promote `TypedStore[State]` as recommended snapshot API          |
| `fe89745f` | Add heterogeneous engine example (Pebble + SQLite)               |

---

_Generated 2026-06-28 06:57 CEST._
