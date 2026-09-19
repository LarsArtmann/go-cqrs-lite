# SUPERB Plan: Metaengine Universal Storage Substrate — the ONE disk path

**Date:** 2026-09-18 16:17 CEST
**Type:** Pareto execution plan
**Basis:** 2026-09-18 session (scheduling/ v5 assessment → metaengine integration audit → "ALL disk writes through metaengine ONLY" directive) + [`meta-engine-universal-adt-support.md`](meta-engine-universal-adt-support.md) + [`2026-09-13_durable-work-queue-module.md`](2026-09-13_durable-work-queue-module.md) + [ADR-0123](../adr/0123-v5-unification-single-composition-root.md) + [ADR-0136](../adr/0136-temporal-composability.md)
**Target:** every durable write/read in the library rides metaengine Engine ADTs; operators pick engines for ALL state at deployment time.

---

## Why this plan exists (context)

The owner directive: **metaengine must become as good as possible and the ONE
way we store and retrieve data from disk in the future.**

Session-verified current state:

- **Already on engines (inside `system/`):** the event journal
  (`EventAdapter.Append` → `StreamAppendExpected`/`StreamAppend`,
  system/adapter_event.go:108-157; `JournalReadFrom`), projections
  (collections + planned tables), snapshots (`SnapshotSave` Engine ADT). v5
  (ADR-0123) finishes this half by deleting the stack presets.
- **Satellites that bypass metaengine today:**

| Subsystem                                 | Writes via                                                             | Blocks absorption           |
| ----------------------------------------- | ---------------------------------------------------------------------- | --------------------------- |
| Timers (`scheduling/sqlstore`)            | own dialect SQL + `claiming/` leases                                   | claim/lease ADT             |
| Tasks (`queue/sqlite`, `queue/postgres`)  | own tables, facts-in-tx, claims                                        | claim/lease + facts-in-tx   |
| Dedup (`idempotency/sqlstore`, `kvstore`) | own tables (`Seen`/`Record`/`CheckAndRecord`/`Sweep`)                  | dedup-with-expiry ADT       |
| Projection checkpoints                    | in-memory map default (system/checkpoint.go) / `event.CheckpointStore` | trivial — Map-shaped        |
| Standalone `storage/*` event stores       | own engines (v4 path)                                                  | none — v5 deletes this path |

- **Four missing ADTs** (verified against `metaengine/engine.go`, 60+ methods):
  1. **Claim/lease** — atomic multi-key conditional claim (SKIP LOCKED
     semantics). `MapUpdate` is per-key RMW only. `claiming/` exists outside
     precisely because of this.
  2. **Dedup-with-expiry** — `SetAdd`/`SetContains` has no TTL/CAS window.
  3. **Time-ordered due claims** — `Due(now)`/`NotBefore` gating with lease
     atomicity (planned tables give the read shape, not claim atomicity).
  4. **Facts-in-same-tx** — queue P4/T16's load-bearing invariant.

**Non-goals / guardrails (no verschlimmbessern):**

- **Facade, not rewrite:** `scheduling`, `queue`, `idempotency` keep their
  public APIs forever (library rule: deleting external-facing API is breaking
  the product). Convergence = internals delegate to engine capabilities.
- **Semantics source of truth stays `queue/`** (go-taskqueue's
  production-proven contract — "upstream the PROVEN semantics; do not invent
  new ones"). The ADT is extracted FROM proven stores, not designed fresh
  against imagined needs.
- **v4.x adds capability interfaces** (the `CatchUpEngine` /
  `EngineResetter` / `VectorPathReporter` pattern — runtime-asserted, never
  breaking). Folding them into the universal `Engine` interface is **v5-only**
  (growing core interfaces is BREAKING; same discipline as contract 21g).
- **Universality rule (ADR-0123 §9):** every new ADT lands on all engines or
  carries an honest degraded fallback + SCREAM diagnostic. No
  `errADTNotSupported` dead-ends.
- **ADR-0136 ladder:** every new collection declares its inverse — timers =
  replayable (derived from events → Reset + replay), dedup = replayable,
  task facts = journal (positions advance across resets).
- **Dep budgets + module isolation** (`nix run .#check-arch`); engine modules
  must not gain production deps (`claiming/` is already in-tree for exactly
  this).

**External dependencies (not owned by this plan):** queue/ remainder
(DAG gating T14, claim-tokens T15, `queue/mysql` T17, facts-in-tx T16 —
TODO_LIST "Durable Work Queue module"), the v5 train, tag waves.

---

## Pareto analysis

### The 1% → 51% (do this first)

| Item                                         | Tasks    | Why 51%                                                                                                                                                                                                                                                                                                                                                                                  |
| -------------------------------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **ADR-0142 + the four capability contracts** | T01, T02 | ~3h, zero production-code risk. Pins the decision (capability interfaces in v4.x, universal fold at v5, facades not rewrites), defines `DueClaimer` / `DedupStore` / `FactSink`, classifies each on the ADR-0136 ladder, and aligns with the three existing plans it extends. Without this, every downstream task re-litigates design; with it, everything else is mechanical execution. |

### The 4% → 64% (+13%)

| Item                                                                                                                    | Tasks   | Why +13%                                                                                                                                                                                                                                                                                                                                          |
| ----------------------------------------------------------------------------------------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Conformance suites + reference implementations on sqlite/postgres/memory + the timer adapter that proves absorption** | T03–T08 | Proves the thesis end-to-end on the two most-used engines: timers become a facade over `DueClaimer` (one claim stack, not two), memory gets the degraded reference every other engine copies, and the two documented Scheduler warts (family-blind retry, MarkFired race) die. After this tier the design is unfalsifiable-by-argument — it runs. |

### The 20% → 80% (+16%)

| Item                                                                                                                                                                                                                                                                                                  | Tasks   | Why +16%                                                                                                                                                                                                              |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Full absorption:** queue engines register as drivers, idempotency becomes facades, `system/` grows declarable timers + operator engine picks + persistent checkpoints, all remaining engines implement the capabilities, facts-in-tx lands as an ADT, observability (SCREAM/Doctor) and docs follow | T09–T17 | After this tier, "all writes through metaengine" is simply TRUE: every satellite rides engines, operators choose where timers/tasks/dedup/checkpoints live at deployment time, and the skill/docs teach the one path. |

### The other 20% → 100%

| Item                                         | Tasks   | Why the tail matters                                                                                                                                                                                                                                                                                             |
| -------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Hardening + v5 cutover + ecosystem proof** | T18–T23 | Benchmarks guard the hot paths; the v5 fold makes capabilities universal `Engine` methods and deletes the duplicate SQL stacks; release train ships it; `example/taskmanager` and the go-taskqueue probe prove it against real consumers. Skipping the tail leaves the vision "true but unproven and unshipped". |

---

## Comprehensive plan (medium granularity, 30–100 min each)

Sorted by importance/impact/effort/customer-value (tier order, then dependency order).

| ID  | Task                                                                                                                                                                                                                                                              | Tier | Impact | Effort     | Customer value                                                                 | Deps                     |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---- | ------ | ---------- | ------------------------------------------------------------------------------ | ------------------------ |
| T01 | **ADR-0142 "Universal Storage Substrate"**: decision record — 4 capability ADTs, v4.x capability-interface path, v5 universal fold, facade-not-rewrite guardrails, ADR-0136 ladder classification, alignment matrix vs ADR-0123/universal-ADT/durable-queue plans | 1%   | 🔥🔥🔥 | M (90m)    | Every future session stops re-litigating; the vision gets a contract           | —                        |
| T02 | **Capability contracts in `metaengine/`**: `DueClaimer` (ClaimDue/RenewLease/Release, due-ordered), `DedupStore` (CheckAndRecord/Seen/Sweep + TTL), `FactSink` tx option; assertion helpers; errorfamily codes; Supports entries                                  | 1%   | 🔥🔥🔥 | M (90m)    | The keystone primitives; typed, classified, non-breaking                       | T01                      |
| T03 | **Conformance suites** (`adttest` pattern): DueClaimer (claim/fire-once/lease-expiry reclaim/double-claimer exclusivity/NotBefore gating/ordering) + DedupStore (CAS idempotency/TTL/Sweep), `-race` + restart patterns                                           | 4%   | 🔥🔥🔥 | M (100m)   | One suite pins every engine to identical semantics — the repo's signature move | T02                      |
| T04 | **claimkit SQL runtime + sqlite wiring** (ADR-0142 amendment): ONE shared `database/sql` claim+dedup runtime in `metaengine/claimkit` (via `claiming/`); sqlite engine embeds (constructor + profile only) + conformance green | 4%   | 🔥🔥   | M (100m)   | First real engine; embedded/edge default                                       | T02, T03                 |
| T05 | **postgres wiring**: pg engine embeds the SAME claimkit runtime (CTE `FOR UPDATE SKIP LOCKED` via `claiming/`) + conformance green | 4%   | 🔥🔥   | M (100m)   | Server default; multi-instance claims                                          | T02, T03, T04            |
| T06 | **Map runtimes + memory/KV wiring** (ADR-0142 amendment): `MapDueClaimer`/`MapDedupStore` degraded runtimes over EXISTING Map backends; memory engine embeds + conformance green | 4%   | 🔥🔥   | S (60m)    | Reference for all other engines; test default                                  | T02, T03                 |
| T07 | **`scheduling/engine`: TimerStore[P] facade over DueClaimer** (Schedule=insert+NotBefore, Due=due-scan, MarkFired=complete-with-epoch, Cancel=release) + behavioral parity vs sqlstore + claim-metrics parity                                                     | 4%   | 🔥🔥🔥 | M (100m)   | THE proof: timers absorbed, one claim stack                                    | T04–T06                  |
| T08 | **Scheduler wart fixes (independent quick win)**: errorfamily-aware retry (stop retrying Rejection/Conflict — T17 partitioning), doc-comment truth-up, CHANGELOG                                                                                                  | 4%   | 🔥🔥   | S (45m)    | Production timers stop retry-forever on permanent failures                     | —                        |
| T09 | **queue engines → metaengine drivers**: task-ADT capability on `queue/sqlite`+`queue/postgres`, `RegisterDriver`, cross-conformance, Profile/Doctor entries                                                                                                       | 20%  | 🔥🔥🔥 | M (100m)   | Tasks enter the operator-registry world                                        | T02, T03 (queue T14–T17) |
| T10 | **idempotency → DedupStore facades**: sqlstore + kvstore delegate when an engine is present; API parity tests                                                                                                                                                     | 20%  | 🔥🔥   | M (90m)    | Dedup keys ride engines; one storage story                                     | T04–T06                  |
| T11 | **`system/` declarable timers**: DomainConfig event→timeout declarations, DeploymentConfig timer-engine pick, scheduler lifecycle owned by composition root, coeffect-gate interplay, integration test                                                            | 20%  | 🔥🔥🔥 | M (100m)   | v5 "single composition root" gets no timer-shaped hole                         | T07                      |
| T12 | **Checkpoints as Map collections**: CheckpointStore over Map ADT; persistent-by-default when engine present; restart test                                                                                                                                         | 20%  | 🔥     | S (60m)    | Last trivial satellite absorbed                                                | —                        |
| T13 | **Remaining engines × capabilities**: bbolt/badger/pebble (key-scan claims + TTL iteration), duckdb+mysql (SQL), turso (libSQL path), dgraph/iroh (eval → explicit capability-refusal note); conformance or documented degradation each                           | 20%  | 🔥🔥   | L (3×100m) | Universality rule satisfied — everything works everywhere                      | T03                      |
| T14 | **FactSink-in-tx capability**: same-tx facts on sqlite + postgres (completes queue T16 AS the ADT reference); journal-never-disagrees conformance invariant                                                                                                       | 20%  | 🔥🔥   | M (100m)   | The queue plan's load-bearing invariant becomes an engine capability           | T04, T05                 |
| T15 | **SCREAM + Doctor for the new ADTs**: Supports complexity entries, degradation diagnostics, ExplainPlan rendering                                                                                                                                                 | 20%  | 🔥     | S (60m)    | Honest routing signals for claims/dedup                                        | T04–T06                  |
| T16 | **Reset ladder for new collections**: EngineResetter everywhere new, ResetResult classification, journal-positions-keep-advancing test                                                                                                                            | 20%  | 🔥     | S (60m)    | ADR-0136 compliance; catch-up replay stays correct                             | T04–T06                  |
| T17 | **Docs**: SKILL recipes §2.x (engine-backed timers/queue/dedup), modules.md rows, FEATURES, module-map, FAQ v5 note; doc-check green                                                                                                                              | 20%  | 🔥🔥   | M (90m)    | The one path becomes the documented path                                       | T07, T09–T12             |
| T18 | **Benchmarks + load sweep**: claim/dedup micro-benches vs direct-SQL baseline, `#load-sweep`, regression-gate baseline                                                                                                                                            | tail | 🔥     | S (60m)    | No perf regression smuggled in by abstraction                                  | T04–T07                  |
| T19 | **v5: fold capabilities into universal Engine** (§9): interface unification, base-embed helper, all engines updated, api-stability regen                                                                                                                          | tail | 🔥🔥   | L (100m+)  | Capabilities stop being optional assertions                                    | v5 gate                  |
| T20 | **v5: delete duplicate stacks**: sqlstore/queue internal SQL → facade-only delegation, dead claim paths removed, goldens regen                                                                                                                                    | tail | 🔥🔥   | L (100m)   | One claim/retry/DLQ stack exists, period                                       | T19                      |
| T21 | **v5: release train**: CHANGELOG symbols gate, doc-check, `#verify` + `#vulncheck` + `#check-arch`, tag wave, pin sweep                                                                                                                                           | tail | 🔥     | S (60m)    | Shipped, not just done                                                         | T19, T20                 |
| T22 | **`example/taskmanager` on engine-backed queue**: demo → real consumer                                                                                                                                                                                            | tail | 🔥     | M (90m)    | Executable proof for consumers                                                 | T09                      |
| T23 | **go-taskqueue semantic-diff probe**: ADT vs production contract memo (P5 input)                                                                                                                                                                                  | tail | 🔥     | S (60m)    | External validation gate for the semantics                                     | T09                      |

**Total: 23 tasks, ≈ 31h medium-granularity effort** (excluding v5-gated and externally-dependent work).

---

## Fine-grained breakdown (≤12 min each)

| ID   | Micro-task                                                                                            | Est | Parent |
| ---- | ----------------------------------------------------------------------------------------------------- | --- | ------ |
| T01a | Draft ADR-0142 context+decision: 4 capability ADTs, capability-interface path v4.x, universal fold v5 | 12m | T01    |
| T01b | ADR: ADR-0136 ladder per ADT collection (timers/dedup replayable; task facts journal)                 | 10m | T01    |
| T01c | ADR: alignment matrix vs ADR-0123 §3/§9, universal-ADT plan, durable-queue P0–P5                      | 10m | T01    |
| T01d | ADR: consequences, negatives (dep budgets, perf, framework-risk) + guardrails                         | 10m | T01    |
| T01e | Verify every ADR claim against source (engine.go, queue/store.go, claiming/) + commit                 | 12m | T01    |
| T02a | `DueClaimer` interface: ClaimDue(owner, lease, limit)/RenewLease/Release, due-ordered contract doc    | 12m | T02    |
| T02b | `DedupStore` interface: CheckAndRecord(key, ttl)/Seen/Sweep + ErrAlreadySeen sentinel                 | 10m | T02    |
| T02c | `FactSink` tx-option capability interface + assertion helper (VectorPathReporter pattern)             | 10m | T02    |
| T02d | errorfamily codes/classes for new errors + errors.go + taxonomy doc sync                              | 12m | T02    |
| T02e | Supports map entries + capability probe helpers                                                       | 12m | T02    |
| T02f | Unit tests for helpers; `nix fmt`; per-module build                                                   | 12m | T02    |
| T03a | conformance: claim + fire-once + lease-expiry reclaim                                                 | 12m | T03    |
| T03b | conformance: double-claimer exclusivity + renew + release                                             | 12m | T03    |
| T03c | conformance: NotBefore gating + due ordering determinism                                              | 12m | T03    |
| T03d | conformance: CheckAndRecord idempotency + TTL expiry + Sweep                                          | 12m | T03    |
| T03e | conformance: `-race` stress + restart-durability patterns                                             | 12m | T03    |
| T03f | Wire `adttest.Assert*` helpers + package doc header                                                   | 10m | T03    |
| T04a | sqlite: timers-shape table + `EnsureLeaseColumn` migration via `claiming.Spec`                        | 12m | T04    |
| T04b | sqlite: ClaimDue (single-writer UPDATE..RETURNING) + unit tests                                       | 12m | T04    |
| T04c | sqlite: RenewLease/Release + unit tests                                                               | 12m | T04    |
| T04d | sqlite: DedupStore SQL (seen/record/check-and-record/sweep) + tests                                   | 12m | T04    |
| T04e | sqlite: conformance suite green incl. `-race`                                                         | 12m | T04    |
| T05a | postgres: table + claim spec wiring (CTE SKIP LOCKED)                                                 | 12m | T05    |
| T05b | postgres: ClaimDue/RenewLease/Release + tests                                                         | 12m | T05    |
| T05c | postgres: DedupStore SQL + tests                                                                      | 12m | T05    |
| T05d | postgres: pgtestcontainer conformance green                                                           | 12m | T05    |
| T06a | memory: DueClaimer (mutex map + lease timers)                                                         | 12m | T06    |
| T06b | memory: DedupStore (TTL map + sweep)                                                                  | 12m | T06    |
| T06c | memory: conformance green + `-race`                                                                   | 12m | T06    |
| T07a | `scheduling/engine`: TimerStore[P] mapping (insert+NotBefore/complete/release)                        | 12m | T07    |
| T07b | MarkFired conditional epoch delete across stores                                                      | 12m | T07    |
| T07c | Behavioral parity tests vs sqlstore (Due ordering, idempotent Schedule)                               | 12m | T07    |
| T07d | Claim-metrics parity + README + doc.go                                                                | 12m | T07    |
| T08a | dispatchWithRetry: skip Rejection/Conflict families + tests                                           | 12m | T08    |
| T08b | Doc-comment truth-up (remove fixed-wart warnings) + CHANGELOG entry                                   | 10m | T08    |
| T09a | Task-ADT capability assertion on queue/sqlite + queue/postgres                                        | 12m | T09    |
| T09b | `RegisterDriver("queue-sqlite"/"queue-postgres")` + EngineConfig mapping                              | 12m | T09    |
| T09c | Cross-conformance: queue engines pass DueClaimer suite                                                | 12m | T09    |
| T09d | Profile/Doctor entries + registry test                                                                | 12m | T09    |
| T10a | idempotency/sqlstore: delegate to DedupStore when engine present                                      | 12m | T10    |
| T10b | idempotency/kvstore: delegate via memory-style DedupStore over kv                                     | 12m | T10    |
| T10c | API parity + Close-semantics tests                                                                    | 12m | T10    |
| T11a | DomainConfig: event→timeout declarations (types + validation)                                         | 12m | T11    |
| T11b | DeploymentConfig: timer-engine pick + scheduler lifecycle in system                                   | 12m | T11    |
| T11c | Coeffect-gate interplay: scheduled command must be declared                                           | 12m | T11    |
| T11d | system integration test + doc.go note                                                                 | 12m | T11    |
| T12a | CheckpointStore over Map ADT collection                                                               | 12m | T12    |
| T12b | system: persistent checkpoints default when engine present                                            | 12m | T12    |
| T12c | Restart-durability test                                                                               | 12m | T12    |
| T13a | bbolt: DueClaimer + DedupStore (key-scan claims) + conformance                                        | 12m | T13    |
| T13b | badger: same pattern + conformance                                                                    | 12m | T13    |
| T13c | pebble: same pattern + conformance                                                                    | 12m | T13    |
| T13d | duckdb + mysql: SQL claims + conformance                                                              | 12m | T13    |
| T13e | turso: libSQL path + conformance                                                                      | 12m | T13    |
| T13f | dgraph/iroh: capability eval → explicit refusal note in Supports                                      | 12m | T13    |
| T14a | FactSink on sqlite: same-tx facts as claim transitions                                                | 12m | T14    |
| T14b | FactSink on postgres                                                                                  | 12m | T14    |
| T14c | Conformance: journal-never-disagrees invariant                                                        | 12m | T14    |
| T15a | Supports complexity entries + degradation notes (4 ADTs × engines)                                    | 12m | T15    |
| T15b | ExplainPlan/Doctor rendering for claim/dedup paths                                                    | 12m | T15    |
| T16a | EngineResetter on all new impls + ResetResult classification                                          | 12m | T16    |
| T16b | Test: reset clears timers/dedup; journal positions keep advancing                                     | 12m | T16    |
| T17a | SKILL recipes §2.x engine-backed timers/queue/dedup + modules.md rows                                 | 12m | T17    |
| T17b | Classify new fences in recipes_catalog + doc-check green                                              | 12m | T17    |
| T17c | FEATURES + module-map + FAQ v5 note                                                                   | 12m | T17    |
| T18a | Micro-benches: claim/dedup vs direct-SQL baseline                                                     | 12m | T18    |
| T18b | `#load-sweep` on timing paths + baseline regen if warranted                                           | 12m | T18    |
| T19a | v5: fold capabilities into Engine interface + base-embed helper                                       | 12m | T19    |
| T19b | v5: update all engines + api-stability golden regen                                                   | 12m | T19    |
| T20a | v5: sqlstore/queue internals → facade-only delegation                                                 | 12m | T20    |
| T20b | v5: delete dead claim SQL paths + golden regen + doc-check                                            | 12m | T20    |
| T21a | v5: CHANGELOG symbols gate + `#verify` + `#vulncheck` + `#check-arch`                                 | 12m | T21    |
| T21b | v5: tag wave + pin sweep + GitHub Releases                                                            | 12m | T21    |
| T22a | example/taskmanager on engine-backed queue                                                            | 12m | T22    |
| T22b | README + end-to-end demo run                                                                          | 12m | T22    |
| T23a | go-taskqueue semantic-diff memo (ADT vs production contract)                                          | 12m | T23    |

**Total: 82 micro-tasks.** Each ends with a per-module build (`GOWORK=off go
build ./...` + `-tags "goexperiment.jsonv2"`) and its own tests green; symbol
changes immediately regen the api-stability golden; doc changes run doc-check.

---

## Execution graph (d2)

```d2
direction: right
P0: {title: "1% → 51% — Decide & Contract"}
P1: {title: "4% → 64% — Prove"}
P2: {title: "20% → 80% — Absorb"}
P3: {title: "other 20% → 100% — Harden, Unify, Ship"}

P0.T01: "T01 ADR-0142" {shape: document}
P0.T02: "T02 capability contracts\nDueClaimer · DedupStore · FactSink"
P0.T01 -> P0.T02

P1.T03: "T03 conformance suites"
P1.T04: "T04 sqlite"
P1.T05: "T05 postgres"
P1.T06: "T06 memory (degraded ref)"
P1.T07: "T07 scheduling/engine\nTimerStore facade (THE proof)"
P1.T08: "T08 Scheduler wart fixes" {style.dashed: true}
P0.T02 -> P1.T03
P1.T03 -> P1.T04
P1.T03 -> P1.T05
P1.T03 -> P1.T06
P1.T04 -> P1.T07
P1.T05 -> P1.T07
P1.T06 -> P1.T07

P2.T09: "T09 queue → drivers"
P2.T10: "T10 idempotency facades"
P2.T11: "T11 system declarable timers"
P2.T12: "T12 checkpoints as collections"
P2.T13: "T13 all remaining engines"
P2.T14: "T14 FactSink-in-tx"
P2.T15: "T15 SCREAM + Doctor"
P2.T16: "T16 reset ladder"
P2.T17: "T17 docs (one path, documented)"
P1.T03 -> P2.T09
P1.T07 -> P2.T11
P1.T04 -> P2.T10
P1.T05 -> P2.T10
P1.T06 -> P2.T10
P1.T03 -> P2.T13
P1.T04 -> P2.T14
P1.T05 -> P2.T14
P1.T04 -> P2.T15
P1.T05 -> P2.T15
P1.T06 -> P2.T16
P2.T09 -> P2.T17
P2.T11 -> P2.T17

gate_v5: "v5 gate\n(ADR-0123 train)" {shape: diamond}
P3.T18: "T18 benchmarks"
P3.T19: "T19 fold into Engine"
P3.T20: "T20 delete dup stacks"
P3.T21: "T21 release train"
P3.T22: "T22 example/taskmanager"
P3.T23: "T23 go-taskqueue probe"
P2.T17 -> gate_v5
P1.T07 -> P3.T18
gate_v5 -> P3.T19
P3.T19 -> P3.T20
P3.T20 -> P3.T21
P2.T09 -> P3.T22
P2.T09 -> P3.T23
```

## Verification (per task, per tier)

- Per-module: `cd <mod> && GOWORK=off go build ./... && GOWORK=off go test ./... -count=1` (jsonv2 tag + cache env chain per `docs/agents/gowork-modes.md`)
- Symbol changes: `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`
- Tier gates: `nix run .#verify` (never concurrent with integration suites); `nix run .#check-arch` after any dep move; `nix run .#check-error-taxonomy` after new codes
- Integration legs: `#integration-pg` (T05), `#integration-mysql-vm` (T13d), `#test-integration` before tier completion
