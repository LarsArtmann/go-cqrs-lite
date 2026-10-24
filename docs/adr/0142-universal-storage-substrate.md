# ADR-0142: Universal Storage Substrate (Every Durable Write Rides an Engine)

**Date:** 2026-09-18
**Status:** Accepted
**Supersedes:** N/A (extends [ADR-0123](0123-v5-unification-single-composition-root.md); operationalizes the "universal ADT" direction of [`meta-engine-universal-adt-support.md`](../planning/meta-engine-universal-adt-support.md))
**Related:** ADR-0123 §3/§9/§10 (driver registry, universality, batch atomicity), ADR-0136 (invertibility ladder), ADR-0140 (capability-degradation precedent), [`2026-09-18 SUPERB plan`](../planning/2026-09-18_16-17_SUPERB-metaengine-universal-storage-substrate.md), [`durable-work-queue-module.md`](../planning/archived/2026-09-13_durable-work-queue-module.md)

> Numbered 0142, not 0141 as the SUPERB plan says: the 0141 slot was taken
> ([native temporal versioned cells](0141-native-temporal-versioned-cells.md)) between planning and execution.

## Context

Owner directive (2026-09-18): **metaengine must become as good as possible and
the ONE way data is stored and retrieved from disk in the future.** Today that
is half-true, and the half that is false is structural:

- **Already on engines (inside `system/`)** the event journal rides engine ADTs
  (`EventAdapter.Append` → `StreamAppendExpected`/`StreamAppend`,
  `system/adapter_event.go`; `JournalReadFrom`), projections ride collections
  and planned tables, snapshots ride `SnapshotSave`. ADR-0123 finishes this
  half at v5 by deleting the stack presets.
- **Satellites bypass metaengine** — each carries its own SQL/schema and its
  own claim protocol:

| Subsystem                                 | Writes via                                            | Missing engine capability            |
| ----------------------------------------- | ----------------------------------------------------- | ------------------------------------ |
| Timers (`scheduling/sqlstore`)            | own dialect SQL + `claiming/` leases                  | atomic claim/lease with due-ordering |
| Tasks (`queue/sqlite`, `queue/postgres`)  | own tables, facts-in-tx, claims                       | claim/lease + facts-in-same-tx       |
| Dedup (`idempotency/sqlstore`, `kvstore`) | own tables (`Seen`/`Record`/`CheckAndRecord`/`Sweep`) | dedup-with-expiry (TTL + CAS window) |
| Projection checkpoints                    | in-memory map default (`system/checkpoint.go`)        | none — trivially Map-shaped          |

Verified against `metaengine/engine.go` (60+ methods across 12 backend
interfaces): the Engine surface has NO claim/lease primitive (`MapUpdate` is
per-key RMW only), NO dedup-with-TTL (`SetAdd`/`SetContains` has no expiry or
check-and-set window), NO time-ordered due-claim (`Due`/`NotBefore` gating with
lease atomicity), and NO cross-ADT transaction option. `claiming/` exists
outside metaengine precisely because of the first gap.

Two prior plans already point here without deciding it: the universal-ADT
design exploration (declare every ADT everywhere, degrade honestly) and the
durable-queue plan (queue semantics upstreamed from go-taskqueue, `claiming/`
extracted as its SQL core). This ADR is the decision that binds them: which
new ADTs the Engine world grows, how they land without breaking v4, and what
absorbs the satellites.

## Decision

### 1. Four missing capabilities, landing as three v4.x capability interfaces

| Capability (concern)                        | v4.x interface (in `metaengine/`)                                                                        | Semantics source of truth                                                                                                                    |
| ------------------------------------------- | -------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| Claim/lease **and** time-ordered due claims | `DueClaimer` — `ClaimDue(collection, owner, lease, limit)` / `RenewLease` / `Release`, due-ordered       | `queue.Store.ClaimDue` + `claiming/` SQL (both production-proven; scheduling/sqlstore ran this SQL across SQLite/PG/MySQL before extraction) |
| Dedup-with-expiry                           | `DedupStore` — `CheckAndRecord(key, ttl)` / `Seen` / `Sweep`, returning `ErrAlreadySeen` on the CAS miss | `idempotency/sqlstore` (`CheckAndRecord` shape)                                                                                              |
| Facts-in-same-tx                            | `FactSink` — tx-scoped append option so a state mutation and its journal fact commit atomically          | `queue.Store` invariant #1 ("a state change without its fact did not happen")                                                                |

`DueClaimer` merges the claim/lease and time-ordered-due concerns because they
are one operation in every real consumer: a due-claim IS a claim gated on a
timestamp with lease fencing — splitting them would recreate the exact
`Schedule→Due→MarkFired` non-atomicity the scheduler documents as a wart.

The ADTs are extracted FROM proven stores, not designed fresh: the semantics
are already running in production shapes (go-taskqueue's `internal/queue.Store`,
scheduling/sqlstore, idempotency/sqlstore). New engines implement the
interface; the reference SQL lives in `claiming/` (in-tree, zero new deps).

### 2. Capability-interface path in v4.x; universal fold at v5

The interfaces follow the established pattern (`EngineResetter`,
`CatchUpEngine`, `VectorPathReporter`): optional, runtime type-asserted,
additive. Growing the core `Engine`/backend interfaces is BREAKING (the
contract-21g discipline) and waits for the v5 gate, where they fold into the
universal Engine exactly as ADR-0123 §9 folds the existing 12 backends.

### 3. Facades, not rewrites

`scheduling`, `queue`, and `idempotency` keep their public APIs forever
(deleting external-facing API is breaking the product). Convergence means their
INTERNALS delegate to engine capabilities: `scheduling/engine` is a
`TimerStore[P]` facade over any `DueClaimer` engine; idempotency stores
delegate to `DedupStore` when an engine is present; queue engines register as
metaengine drivers. The proven SQL is not discarded — engines embed `claiming/`
for it.

### 4. Universality rule applies (ADR-0123 §9)

Every first-party engine implements `DueClaimer` + `DedupStore` natively or via
degraded fallback (KV engines: key-scan claims + TTL iteration; SQL engines:
`claiming/` statements), with honest `Supports`/`DegradedADTs` entries and
SCREAM/Doctor rendering — no `errADTNotSupported` dead-ends. Engines that
genuinely cannot (evaluated: dgraph, iroh replication wrapper) record an
explicit capability refusal note in `Supports`, never silence.

### 5. ADR-0136 ladder classification for the new collections

| Collection           | Rung           | Reset semantics                                                                                                  |
| -------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------- |
| Timers               | 1 — replayable | Derived from timeout-declaration + events; Reset clears, replay re-derives                                       |
| Dedup keys           | 1 — replayable | Derived window over recent facts; Reset clears, replay rebuilds                                                  |
| Task facts / journal | 3 — facts      | Append-only; reset NEVER deletes — journal positions keep advancing (same split as engine journals, contract 22) |
| Task state rows      | 1+3 hybrid     | State is replayable from facts; the fact stream is the journal                                                   |

### 6. Alignment matrix

| Prior artifact                                                                | Relationship                                                                                                                                                                                            |
| ----------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ADR-0123 §3 (driver registry in metaengine, engines self-register)            | T09 registers queue engines as drivers — the first non-storage engine family to enter the registry world; validates §3 before v5                                                                        |
| ADR-0123 §9 (every engine every ADT)                                          | This ADR extends the ADT set the v5 fold must cover; capability interfaces are the v4.x on-ramp                                                                                                         |
| ADR-0123 §10 (batch boundary = event)                                         | Once timers/dedup are collections on the same engine as the journal, append-event + schedule-timer + record-dedup becomes one tx — the outbox problem structurally dissolved; FactSink is the primitive |
| `meta-engine-universal-adt-support.md` (declare everywhere, degrade honestly) | Adopted as rule 4; this ADR supplies the mechanism (DegradedADTs + SCREAM) it lacked                                                                                                                    |
| Durable-queue plan P0–P5                                                      | P0 (`claiming/`) is reused verbatim; P4/T16 facts-in-tx IS `FactSink` (the ADT becomes its reference); queue T14/T15/T17 remain queue-internal and gate only T09's queue-driver leg                     |
| ADR-0140 (degrade-everywhere precedent)                                       | Same pattern extended: capability + honest profile + conformance parity                                                                                                                                 |

## Amendment (2026-09-18, owner review): one runtime per storage class, not per engine

The original task breakdown read as "implement DueClaimer on sqlite (T04),
again on postgres (T05), again on memory (T06)" — owner pushback ("I hate that
we need to reimplement it") was correct: that shape would copy the same SQL and
the same claim loop per engine. Amended execution structure:

| Storage class      | Shared runtime                                                                                                                                                                                                                                                       | Consumed by                                                                                  |
| ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| SQL engines        | `metaengine/claimkit` — ONE payload-agnostic `database/sql` runtime (table DDL, dialect-dispatched claims via `claiming/`, owner-fenced renew, epoch delete, same-tx facts, SQL dedup) implementing the metaengine interfaces directly                               | sqlite, postgres, mysql, duckdb, turso — each embeds it (constructor + profile entries only) |
| Map-shaped engines | `metaengine.MapDueClaimer` + `metaengine.MapDedupStore` — ONE pure-Go degraded runtime over the EXISTING `MapBackend`+`MapUpdater`+`ScanBackend` interfaces (per-key RMW claims under engine write serialization; honest `DegradedADTs` entry, the ADR-0140 pattern) | memory, pebble, bbolt, badger — each embeds it                                               |

Per-engine work drops to wiring: construct the runtime with the engine's
handle, add the `Supports`/`DegradedADTs` profile entries, run the conformance
suite. No engine hand-writes claim SQL or a claim loop. `claiming/` stays
statement-builders (charter unchanged); claimkit extends it with a richer
claim builder (owner column, collection filter, LIMIT) — old builders remain
for `scheduling/sqlstore`.

## Realized capability matrix (2026-09-19, T13 complete)

The universality rule (§4) as landed — every first-party engine is one of:
native (claimkit SQL), degraded (Map runtimes, honestly declared), or
explicitly refused (`RefusedADTs` with reason, pinned by the
`capability_audit` test in `metaengine`).

| Engine   | ADTDueClaim     | ADTDedup              | Mechanism / refusal reason                                                                                        |
| -------- | --------------- | --------------------- | ----------------------------------------------------------------------------------------------------------------- |
| memory   | O(N), degraded  | O(logN)-ish, degraded | `MapDueClaimer` + `MapDedupStore` reference runtimes (ADR-0140 pattern)                                           |
| pebble   | O(N), degraded  | degraded              | Map runtimes over KV key-scan claims + TTL iteration                                                              |
| bbolt    | O(N), degraded  | degraded              | Map runtimes over KV key-scan claims + TTL iteration                                                              |
| badger   | O(N), degraded  | degraded              | Map runtimes over KV key-scan claims + TTL iteration                                                              |
| sqlite   | O(logN), native | O(logN), native       | `metaengine/claimkit` (single-statement upsert CAS)                                                               |
| turso    | O(logN), native | O(logN), native       | delegates to sqliteengine — pinned by a local-file-DSN conformance run (libSQL driver path)                       |
| postgres | O(logN), native | O(logN), native       | claimkit (CTE `FOR UPDATE SKIP LOCKED` via `claiming/`)                                                           |
| mysql    | O(logN), native | O(logN), native       | claimkit (two-statement claim; lock-free dedup CAS — `INSERT IGNORE` + conditional UPDATE, no gap-lock deadlocks) |
| duckdb   | O(logN), native | O(logN), native       | claimkit; writes serialized by claimkit's dialect-conditional mutex (DuckDB ON CONFLICT limitation)               |
| dgraph   | **REFUSED**     | **REFUSED**           | DQL upserts cannot atomically fence concurrent claimers (no SKIP LOCKED / CAS-with-expiry)                        |
| iroh     | **REFUSED**     | **REFUSED**           | CRDT/eventual replication — a lease on one replica is not a lease anywhere                                        |
| bigtable | **REFUSED**     | **REFUSED**           | no atomic arbitrary-value RMW (`MapUpdate`) or collection scan yet (needs CheckAndMutate CAS)                     |

## Consequences

**Positive**

- The directive becomes testable: "all durable writes ride engines" is a
  conformance matrix, not an aspiration. Timers absorbed by `DueClaimer` is
  the keystone proof (`scheduling/engine` facade over one claim stack).
- One claim/retry/DLQ story survives: `claiming/` SQL serves the engines and
  the satellites delegate — no second claim protocol is ever written.
- v4.x consumers get engine-backed timers/dedup/queue without any API change;
  operators gain deployment-time engine choice for ALL state.

**Negative / accepted costs**

- Engine modules grow implementation surface; `nix run .#check-arch` budgets
  stay enforced (`claiming/` in-tree keeps production deps at zero).
- Degraded paths (KV key-scan claims) are O(N)-ish; the profile declares it
  and the planner/Doctor surface it (same cost ADR-0140 accepted for vectors).
- Framework-risk: three new optional interfaces today means a v5 fold with
  migration notes; deferring them entirely would leave satellites forked
  forever — the larger cost.
- The facade rule means duplicate write paths coexist until the v5 deletion
  (T20); until then `scheduling/sqlstore` remains the default and
  `scheduling/engine` is opt-in.

## Verification

- `adttest.AssertDueClaimer` / `AssertDedupStore` conformance suites run per
  engine (claim/fire-once, lease-expiry reclaim, double-claimer exclusivity,
  NotBefore gating, due-ordering determinism, CAS idempotency, TTL expiry,
  Sweep, `-race`, restart durability).
- Behavioral parity: `scheduling/engine` vs `scheduling/sqlstore` (Due
  ordering, idempotent Schedule, claim metrics).
- Capability audit: profiles declare the new ADTs on every first-party engine
  — no over- or under-declaration (the ADR-0140 audit pattern).
- Journal-never-disagrees invariant (FactSink): task-state mutation without
  its fact must be unobservable.

## Amendment (2026-10-24, owner directive): scope extended to ALL data I/O; A/B gate formalized

The 2026-09-18 directive said "every durable WRITE rides an engine." The owner
reaffirmed and widened it: **ALL data I/O — reads included — goes solely
through metaengine.** For any missing capability, a two-step gate now applies
before any new I/O is written:

- **(A) Good-reason check.** Is there a good reason metaengine lacks the
  capability? Transport is not durable state (watermill brokers and the SSE
  wire layer stay at their delivery layer — their acks/redelivery/ordering
  are broker law, not storage). Build artifacts are not application state
  (catalog/docserver/benchkit output). Legacy `storage/*` is already
  sentenced to v5 removal (ADR-0123). Where the answer is yes, the feature is
  NOT added — preferring no feature over forking the substrate is the
  intended outcome, not a failure.
- **(B) Design-with-owner.** If the capability should exist, it is designed
  WITH the owner BEFORE implementation, following this ADR's amendment
  pattern: ONE shared runtime per storage class (metaengine core, `claimkit`
  for SQL, Map runtimes for KV-shaped engines) — never a per-engine
  reimplementation, never satellite-owned SQL. The feature is then built on
  top of the landed engine capability.

Ledger as of 2026-10-24: journal, projections/read models, snapshots
(`system.SnapshotAdapter` over `SnapshotBackend`), checkpoints (engine Map
collection default via `resolveCheckpointStore`: custom > engine >
memory-only fallback), and dedup (`DedupStore`; `idempotency/*` deprecated
per ADR-0158) ride engines. Open items under the widened scope: timers
(`scheduling/engine` opt-in; `scheduling/sqlstore` remains the v4 default,
flip v5-gated T20), queue task tables (engines are metaengine drivers; task
tables stay engine-owned because their ClaimDue predicate — priority + age
bonus + dependency gating + dedup keys — is richer than `DueClaimer`'s), and
`kv` (no engine adapter; only v5-removed legacy backends implement
`kv.Store`).

### Verdicts (2026-10-24, owner decisions on the widened scope)

| Open item | Verdict | Consequence |
| --- | --- | --- |
| Timers default | **Deprecate `scheduling/sqlstore` now** (was: v5-gated T20 default-flip) | Every package-level symbol carries a `// Deprecated:` notice pointing at `scheduling/engine.NewTimerStore` over a `metaengine.DueClaimer` engine; module deleted at v5.0.0. In-repo exempt demo: `example/scheduler-otel-status` (demonstrates the sqlstore-specific `ClaimMetrics` hooks; banner added, engine-path observability demo replaces it before v5). |
| Queue task tables | **A-verdict: stay engine-owned**, with a revisit note | Task semantics (priority + age bonus + dependency gating + attempt budgets + token-fenced finalizes) are not a generic ADT; queue engines already ARE metaengine drivers exposing `DueClaimer`/`DedupStore`/`FactSink`. Revisit trigger recorded in TODO_LIST: if a SECOND consumer needs priority-ordered due-claims on engine collections, reopen the DueClaimer-v2 design. |
| `kv` durable backends | **Under evaluation** | The owner asked for the original design intent of `kv.Store` before deciding between an engine Map adapter and demoting `kv` to in-memory/cache-only. Origin research (ADR-0022/0032 lineage) pending; decision open. |
| Enforcement | **Approved + shipped: cqrs-lint E023 `satellite-owned-sql`** | Advisory (info/high) rule flags `database/sql` + SQL-driver imports in packages that also import go-cqrs-lite; engine-tier prefixes exempt, test files exempt. Wired into register/catalog/RULES.md/README/V007 tables (which also gained the `scheduling/sqlstore` whole-module entry covering its 21 new deprecation markers) + api-stability golden. |


