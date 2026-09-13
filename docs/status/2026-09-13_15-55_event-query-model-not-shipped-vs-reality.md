# Research: Event-Query-Model "Not Shipped" Parts vs Current Reality

**Date:** 2026-09-13 15:55 CEST
**Scope:** Deep-dive verification of every "Not shipped (aspirational)" item from the audit of [`docs/planning/event-query-model.md`](../planning/event-query-model.md) (2026-07-23). Follow-up to [`2026-09-13_12-10_metaengine-event-query-model-doc-audit.md`](2026-09-13_12-10_metaengine-event-query-model-doc-audit.md).
**Method:** Repo-wide source verification (grep + read), not just `metaengine/`-scoped. All claims carry file:line evidence as of this date.
**Nature:** Read-only research. No code modified.

> **Corrections included.** This research overturns three verdicts from the earlier
> 12:10 audit and status report: C1 (command log "not started" — wrong at repo
> level), B3 (streaming "engine-level only" — both understated and mischaracterized),
> and C6 (Bloom "no backend" — missed the Pebble-internal reality). The earlier
> snapshot is left unmodified per point-in-time convention; this document supersedes
> those three items.

---

## Summary Table

| # | Doc §10/§12/§15 claim | Verdict | Reality in one line |
|---|----------------------|---------|---------------------|
| 1 | Command log as event stream | **SHIPPED (renamed + reshaped)** | `commandlifecycle` (ADR-0117) + command journals |
| 2 | Query log as event stream | NOT shipped | In-process observability hooks only |
| 3 | Session log as event stream | NOT shipped | Zero adjacent machinery; external `identity-model` owns sessions |
| 4 | §15 Decision 2: `Stream(ctx, input, fn)` | **GHOST-SHIPPED** | Engine capability on 4 engines, zero production callers |
| 5 | §15 Decision 3: cross-projection queries | NOT shipped | Single-collection aggregates only |
| 6 | §12 YAML engine config | NOT shipped | Philosophy honored via Go composition roots |
| 7 | §12 Neo4j engine | NOT implemented, designed-for | `GraphDriver` + `graphtest` extension point exists |
| 8 | §5/§7/§12 Bloom filters | EXISTS, invisible | Pebble-internal filter policy, not an ADT backend |

---

## 1. Command log — SHIPPED (renamed and reshaped)

The doc wanted `CommandSucceeded{Type, Payload, Reason, Metadata, Timestamp}` /
`CommandRejected{...}` appended to a single command log, projected lazily into
queryable shapes. Reality is a **two-stream design that separates immutable intent
from outcomes** — arguably cleaner than the doc's single-log sketch.

### What exists

**ADR-0117 ("command lifecycle as events") implemented as the `commandlifecycle` module:**

- Two streams per command (`commandlifecycle/events.go:10-22`):
  - `Command/<cmd-id>` — the immutable command intent (full command, full payload)
  - `CommandLifecycle/<cmd-id>` — lifecycle transition events
- Five lifecycle event types (`commandlifecycle/events.go:51-65`):
  `command.received`, `command.failed`, `command.retried`, `command.dead-lettered`,
  `command.completed` — mapping onto the doc's `CommandSucceeded`/`CommandRejected`
  with finer granularity (per-attempt failure/retry, not just terminal states).
- `Recorder` + middleware pair; wired in one line via `system.WithCommandLifecycle`
  (`system/lifecycle.go`). Actor/correlation identifiers propagate onto lifecycle
  events through command metadata when present (`commandlifecycle/recorder.go:201-203`).
- **Command journals** (`command/store.go:143-160`):
  - `CommandJournal.ReadAll` — "the command-side equivalent of event.Journal — a
    complete audit trail of every command ever dispatched" with use cases that
    verbatim match doc §10: audit ("who issued what commands and when?"), replay
    debugging, analytics.
  - `SeekableCommandJournal` — incremental command replay from ULID checkpoints
    (`Load`/`LoadToTimestamp`).
- **Projections via metaengine** (`commandlifecycle/projections/projections.go`) —
  the doc's "project lazily" realized literally:
  - `DeadLetterQueue()` — Map ADT query on `command.dead-lettered` (projections.go:68)
  - `RetryCount()` — Delta (Counter ADT) on `command.retried` (projections.go:86)
  - `FailureLog()` — Append (Log ADT) on `command.failed` (projections.go:104)
  All built with `metaengine.OnRecordTyped` + `record.Record` context, keyed by
  `rec.MetaData.Cause.ID` (the command ID).

### Verdict vs doc

| Doc §10 element | Status |
|---|---|
| Commands are append-only logs | Shipped (intent streams + journals) |
| Append first, project lazily | Shipped (event streams + metaengine projections) |
| Full-payload audit records | Shipped at intent-stream level (`Command/<id>`); lifecycle payloads deliberately omit payload (no duplication) |
| Per-user audit projection (`CommandsByUser`) | Possible (actor propagates onto lifecycle events) but **no pre-built projection** |
| "Four logs" framing (command/query/session) | 1 of 3 realized |

### Gaps

- No pre-built per-user/per-actor command-audit projection.
- No `CommandRejected{Reason}`-style rejection event distinct from failures
  (rejections vs retryable failures vs exhaustion are conflated into
  failed/dead-lettered; Rejection-classified errors are not separately modeled).
- Naming/framing: the doc's "command log" vocabulary does not point to
  `commandlifecycle` or `CommandJournal`; a reader of the doc would not find them.

---

## 2. Query log — NOT shipped

Doc: `QueryExecuted{Type, Payload, Duration, ResultHash, Metadata}` as an
append-only log, projected into debugging/auditing shapes.

Verified absent:

- Zero hits for `QueryExecuted` repo-wide (this session's repo-wide grep).
- Adjacent reality is **in-process observability only**:
  - `MetricsRecorder.RecordExecute(col, pattern, d, count, err)` — `metaengine/observability.go:86`
  - `WithSlowQueryLog` (threshold-based), `workloadMeter` counters (`stats.go`),
    `NewCostAccuracyReporter` (cost-model drift).
- No durable query events, no `ResultHash` concept anywhere, no query-replay path.

Gap if wanted: queries execute against engines without persisting an auditable
trace; "what did this query return at time T" is unrecoverable.

---

## 3. Session log — NOT shipped

Doc: `SessionStarted/SessionEnded/SessionRevoked` as event streams with analytics
(concurrent-session counter), audit (time-range), security (revoke-all), and
compliance (access history) use cases.

Verified absent:

- Zero `session` hits in Go code and in all skill references (repo-wide).
- Only foundation present: typed `id.ActorID` attribution end-to-end (AGENTS.md
  data-model conventions #21e).
- The doc itself notes the external `cqrs-htmx/identity-model` repo treats
  sessions as ephemeral runtime objects; that is still the state of the world.

Related-but-different modules that could be confused with sessions:
- `claiming/` (extracted 2026-09-13 from `scheduling/sqlstore`) — lease-claim SQL
  for distributed timers, not user sessions.
- `scheduling/` — durable timers ("cancel order after 30 min"), not sessions.

---

## 4. Streaming (§15 Decision 2) — GHOST-SHIPPED

Doc: each query handler supports `Execute(ctx, input) → output` AND
`Stream(ctx, input, fn)` for unbounded results; "the planner generates both";
streaming uses `iter.Seq2`.

Reality is split across layers — the capability exists, the API does not:

- **Engine capability exists and is implemented 4×:** `StreamingScan` interface —
  `StreamScan(ctx, collection, filters, sort) iter.Seq2[any, error]`
  (`metaengine/engine.go:378-384`). Implementors: `sqliteengine/engine.go`,
  `pebbleengine/stream_scan.go`, `bboltengine/stream_log.go`,
  `badgerengine/stream_log.go`.
- **Zero production callers.** Repo-wide `.StreamScan(` grep: only
  `pebbleengine/stream_scan_test.go` invokes it. Critically, `Store.Export`
  (`metaengine/export_import.go:12`) does NOT use it — despite the interface's own
  doc comment naming "batch processing or export operations" as the use case.
- **No Store/reader API:** no `Store.Stream`, no `TypedReader.Stream`, no
  planner-generated dual-mode handlers. `TypedReader` offers `Scan`/`ScanPage`
  (bounded, cursor-based) only.

Classification: **ghost capability** — shipped interface + implementations with
no consumer. Either wire it (fulfilling Decision 2) or cut it at v5.

---

## 5. Cross-projection queries (§15 Decision 3) — NOT shipped

Doc: "Active users with >5 friends" needs FindUser (Pebble) + FriendsOf (Neo4j);
Option A = new combined projection, Option B = read-time fan-out with merge.

Reality:

- Grouped aggregates (`GroupBy`, `GroupedCount/Sum/Min/Max/Avg`, `MultiAggregate`,
  `typed_reader_grouped.go`) are **single-collection**.
- `executeQueryInner` (`metaengine/execute.go:182`) routes each query to exactly
  one collection's engine; `checkKeyTypeMatch` enforces one key type.
- Cross-engine read-time fan-out/merge: no code path.
- The Option-A workaround (declare a third projection listening to both event
  types) is exactly what the doc recommended and what the fold model supports
  today — so the recommendation is manually achievable, the automation isn't.

---

## 6. YAML engine config (§12) — NOT shipped; philosophy honored in Go

Doc: operator-editable `engines: { sqlite: { driver, dsn } }` YAML.

Reality:

- No YAML config loader anywhere (yaml deps in go.mod files are test-only;
  `catalog/asyncapi` uses yaml for AsyncAPI export roundtrip tests, not config).
- Engines are Go-constructed: `sqliteengine.NewSQLiteEngineFromDSN("app.db")`,
  `metaengine.Plan([]Engine{...}, queries...)`.
- The doc's real intent — "operator picks engines at DEPLOYMENT time, developer
  never worries" — is honored via Go composition roots (`system.New`) and engine
  constructors. `stack/*presets` are deprecated for v5 (ADR-0123), replaced by
  `system` + `metaengine` composition.
- Hot-reload intent ("add/remove engines without restart") is served by the
  runtime engine APIs instead of config reload: `AddEngine`/`RemoveEngine`
  (`runtime_backend.go:55,113`), `SwapEngine` (`advanced.go:69`), `Replan`
  (`store.go:88`), `ReplanLayout` (`relayout.go:64`), shadow roles
  Migration/Backup (`roles.go:8-24`).

---

## 7. Neo4j engine (§12) — NOT implemented, explicitly designed-for

- `graph/graph.go:29`: *"A Neo4j or [other] driver: implement `GraphDriver`
  against their target database; the contract test suite in `graphtest`
  [validates it]"* — the plug-in path is documented and test-supported.
- Shipped graph drivers: `MemoryDriver` (tests), Dgraph (`dgraphengine`, DQL over
  gRPC). SQL recursive-CTE fallback lives in metaengine (`graph_fallback.go`).
- `graphadapter` bridges `GraphDriver` → `metaengine.Engine` (ADR-0113 deleted the
  old GraphBackend; `graph.GraphDriver` implements `metaengine.Engine`).
- Writing a Neo4j driver = implement `GraphDriver`, pass `graphtest` contract
  suite, bridge via `graphadapter`. No design blocker; simply nobody wrote it.

---

## 8. Bloom filters (§5/§7/§12) — EXISTS as invisible internal optimization

- Doc claims: "Bloom filter IS a compressed graph", Set ADT served by "Bloom
  filter (Memory/Pebble)" at scale, `Bloom.Add(email)` / `Bloom.Test(email)` in
  generated handlers.
- Reality: **no Bloom backend in the ADT model** (ADT enum: map/set/counter/graph/
  log/stream_log/sorted_map/multimap — `metaengine/types.go:3-15`), no
  `Bloom.Test` read pattern, no generated Bloom handlers.
- BUT Pebble ships an internal bloom filter policy for LSM point reads:
  `storage/pebble/options.go` (`pebble.FilterPolicy(10 bits/key, ~1% FPR)`),
  shipped by the `stack/pebble` preset; bbolt's preset doc contrasts itself
  against it (`stack/bbolt/preset.go`).
- Net: a Pebble-routed Set/membership query benefits from bloom-accelerated point
  lookups by accident of the storage layer — the doc's physical-structure claim is
  ~half-true, invisible to the ADT model. False-positive semantics (Bloom says
  "maybe" vs Set says "no") were never modeled; correctness for existence checks
  would require an exact structure anyway.

---

## Net assessment

- Of the doc §10 "four logs" vision: **command log shipped** (better-shaped than
  the doc), **query log absent**, **session log absent**.
- Of the doc §15 open decisions: **D1 resolved** (hybrid FilterOn/FilterOnField),
  **D2 ghost-shipped** (engine layer only), **D3 not shipped** (manual Option A
  works), **D4 shipped** (catchup checkpoints).
- Of the doc §12 engine roster: 10 real engines; Neo4j/YAML/Bloom-ADT replaced by
  GraphDriver-extension/Go-composition/Pebble-internals respectively.
- The doc's biggest ongoing risk is not the missing features — it is that its
  §10 vocabulary ("command log", "query log", "session log") does not point a
  reader to `commandlifecycle`, the observability hooks, or `identity-model`,
  while still claiming to be "THE model".

---

## Open questions (need product intent, not code reading)

1. **Command-log scope:** Is the doc's "full comprehensive audit — who did what,
   when, what did it cause" still a target? Concretely: add per-user/actor and
   payload-carrying projections to `commandlifecycle/projections` (actor already
   propagates onto lifecycle events), or is DLQ/RetryCount/FailureLog the finished
   scope? Also: should Rejection-classified errors get a distinct lifecycle event?
2. **StreamingScan ghost:** Wire it (a `Store.Stream`/`TypedReader.Stream` API +
   use it in `Store.Export`, fulfilling Decision 2) or cut it at v5 as an unwired
   capability?
3. **Sessions & the planned `queue/`:** Are sessions-as-event-streams permanently
   delegated to the external `identity-model` repo, or a future go-cqrs-lite
   module? Related: `claiming/` was extracted 2026-09-13 with a note that a
   planned `queue/` module builds on it (`modules.md:65`) — does `queue/`
   intersect the doc's four-logs vision (e.g. query log as a queue), or is it
   unrelated?

---

## Evidence index (file:line)

| Claim | Location |
|---|---|
| Lifecycle stream model | `commandlifecycle/events.go:10-22` |
| 5 lifecycle event types | `commandlifecycle/events.go:51-65` |
| Lifecycle payload shapes (no command payload) | `commandlifecycle/events.go:71-146` |
| Recorder + actor/correlation propagation | `commandlifecycle/recorder.go:79,99-109,201-203` |
| Middleware pair wiring | `commandlifecycle/middleware.go:68-92`, `system/lifecycle.go` |
| DLQ/RetryCount/FailureLog projections | `commandlifecycle/projections/projections.go:68,86,104` |
| CommandJournal / SeekableCommandJournal | `command/store.go:130-160` |
| StreamingScan interface | `metaengine/engine.go:369-384` |
| StreamingScan implementors (4 engines) | `metaengine/sqliteengine/engine.go`, `metaengine/pebbleengine/stream_scan.go`, `metaengine/bboltengine/stream_log.go`, `metaengine/badgerengine/stream_log.go` |
| StreamingScan zero production callers | repo-wide `\.StreamScan\(` grep: only `pebbleengine/stream_scan_test.go` |
| Export does not stream | `metaengine/export_import.go:12` |
| Single-collection aggregation | `metaengine/typed_reader_grouped.go`; single-engine routing `metaengine/execute.go:182` |
| Query observability (not a log) | `metaengine/observability.go:86`, `metaengine/stats.go` |
| Pebble bloom policy | `storage/pebble/options.go`, `stack/pebble/preset.go`, contrast `stack/bbolt/preset.go` |
| Neo4j extension point | `graph/graph.go:22-33,112-130` |
| Hot-reload runtime APIs | `metaengine/runtime_backend.go:55,113`, `metaengine/advanced.go:69`, `metaengine/store.go:88`, `metaengine/relayout.go:64`, `metaengine/roles.go` |
| stack presets deprecated v5 | ADR-0123 (skill core.md routing matrices) |
| No YAML config | repo-wide yaml grep: test-only deps + `catalog/asyncapi` export roundtrip |

*Awaiting instructions.*
