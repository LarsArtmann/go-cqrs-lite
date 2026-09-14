# T02 Verification Notes: event-query-model.md vs Source

**Date:** 2026-09-13 17:40 CEST
**Parent plan:** [`2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md`](../planning/2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md) (task T02)
**Purpose:** source-verified answers for the 7 inferred claims that T01..T22 annotations depend on. Every claim below was read from source in this session; file:line citations are exact as of `93cc2be6c` + working tree.

---

## T01 results (inbound links + gate scope)

- **Inbound doc links (13 files, all documentation):** `docs/reviews/2026-09-13_md-go-validator-review.md`, `docs/planning/future-typespec-extension.md` (calls it "Prerequisite reading"), `docs/planning/archived/2026-07-23_22-17_meta-engine-build-plan.md` ("THE specification"), `docs/planning/archived/2026-07-23_21-55_metaengine-plan-alignment.md`, both 2026-09-13 status reports, one planning plan, and 6 archived status docs.
- **Zero links from `AGENTS.md`, `SKILL.md`, or `.agents/`.** The doc's blast radius is docs-only.
- **doc-check default scan set does NOT include `docs/planning/`** (`cmd/doc-check/main.go:103-119`): SKILL.md, AGENTS.md, `docs/DOMAIN_LANGUAGE.md`, `docs/METAENGINE_DOMAIN_LANGUAGE.md`, `.agents/skills/*/references/*.md`. Planning docs are only checked when explicitly passed as args. So the broken snippet at `event-query-model.md:182` is currently ungated.
- **Pre-existing validator finding:** `docs/reviews/2026-09-13_md-go-validator-review.md:227` flags `event-query-model.md:182` (block 2): "Go syntax error: expected operand, found '...' — snippet mixes package-level declarations with function-body statements." Feeds T25.

## T02.1 — §7 "zero coordination" vs `rule_shared_collection.go`

**Verdict: §7 is NOT contradicted.** The shared-collection rule (`metaengine/rule_shared_collection.go:10-19`) is:

- **Opt-in** — only applies to child Go types explicitly declared shared via `WithSharedCollection` (ADR-0124, `docs/adr/0124-operator-driven-layout-planning.md`).
- **Layout-level, not cross-projection coordination** — it forces `LayoutNormalize` on a query whose result embeds a declared-shared child type (`rule_shared_collection.go:37-47`), because embedding would duplicate the shared child inside that one collection.
- **v1 is scoring/diagnostics only** — "physical child-collection materialization does not exist yet" (`rule_shared_collection.go:17-19`).
- **It reinforces independence by default**: a shared type spanning 2+ collections emits `DiagLevelWarn`: "without a shared collection these copies drift independently" (`rule_shared_collection.go:85-96`).

Annotation for T11: each query still gets its own independent projection (unchanged); the opt-in shared collection is a future dedup capability, currently a layout normalization + warning.

## T02.2 — Decision 4 (hot-reload replay progress) vs `catchup_state.go`

**Verdict: PARTIALLY shipped, different mechanism.** `CatchUpState` (`metaengine/catchup_state.go:11-29`) tracks the ADR-0137 quarantine rebuild (write-failover recovery): `Running`, `LastError`, `Replayed` (events folded at the stability gate), `CompletedAt` — surfaced via `Store.CatchUpSnapshot`, `EngineStats.CatchUp`, and the Doctor report. The rebuild itself is `Store.CatchUpEngine` (`metaengine/failover.go:60`).

The §14 flow (dual-read while replaying, atomic cutover, teardown) is **not** implemented as described. What exists on the hot-reload axis:

- `Store.Replan` (`store.go:88`), `Store.ReplanLayout` (`relayout.go:64`), `Store.CheckRouting` (`store_routing.go:59`)
- `Store.AddEngine` (`runtime_backend.go:55`), `RemoveEngine` (`runtime_backend.go:113`), `SwapEngine` (`advanced.go:69`)
- Shadow roles `RoleMigration`/`RoleBackup` (`roles.go:11-21, 173`) as the cutover primitive
- projectionhost checkpoint stores (event-sourced subscribers) exist in the projection module, independent of metaengine.

Annotation for T09: Decision 4's checkpoint idea shipped in two forms, neither being the §14 cutover orchestrator.

## T02.3 — Decision 3 (multi-projection queries) vs `typed_reader_grouped.go`

**Verdict: NOT shipped.** All grouped/aggregate reads are single-collection: `TypedReader[V].Distinct` (`typed_reader_grouped.go:12-16`), `GroupBy` (`:56-60`), `GroupedCount`/`GroupedSum` (`:84-99`) all operate on `r.collection`. A repo-wide grep for `fanout|multi.collection|cross.query|cross.projection` in metaengine production files returns only two layout-scoring comments about duplicated data, no fan-out path. Option A (combined fold) is what the doc recommends and remains a design proposal.

## T02.4 — §13 "never writes" rows vs layout code

**Verdict: TRUE on the metaengine path, with a boundary caveat.**

- DDL IS auto-generated: `LayoutPlan.DDL()` (`metaengine/layout.go:116`), `BuildLayoutPlan` (`:49`), `BuildLayoutPlanFromType` (`:199`), `inferColumnType` (`:169`).
- Index inference from read patterns is real: `infer_filters.go` (prefix → FilterOp table), `infer_sort.go`, `infer_composite.go`, `infer_named.go`.
- **Caveat:** classic lower-level modules still expose explicit schema APIs — `storage/relational/schema.go`, `storage/view/mapper.go`, `graph/schema.go` all define `IndexSpec`; `cmd/cqrs-lint/pkg/rules/version/v007_tables.go` lints them. The "never writes" claim holds for consumers using the auto-projection metaengine; consumers using classic modules still write schemas.

## T02.5 — §11 planner steps 1-7 → real source counterparts

| Step | Claim                        | Source counterpart (verified)                                                                  |
| ---- | ---------------------------- | ---------------------------------------------------------------------------------------------- |
| 1    | Classify write-side ADT      | `metaengine/fold_classify.go:10` `classifyADT`                                                 |
| 2    | Classify read pattern        | `metaengine/infer_filters.go`, `infer_sort.go`, `infer_composite.go`, `infer_named.go`         |
| 3    | Assign cheapest engine       | `metaengine/cost.go:70` `estimateCost`, `rules.go:54` `defaultRules`                           |
| 4    | Plan physical structures     | `metaengine/layout.go:49` `BuildLayoutPlan`, `:116` `DDL()`                                    |
| 5    | Generate projection handlers | `metaengine/auto_fold.go` (field mapping), fold pipeline (`store.go:573-922` applyFold family) |
| 6    | Generate typed read handlers | `metaengine/typed_reader*.go`, `execute.go:681` `ExecuteTyped`                                 |
| 7    | Validate + warn              | `metaengine/rules.go`, `plan_audit.go`, `explain.go:283` `Doctor`                              |

All 7 steps have a real counterpart; the doc's abstractions map cleanly, though step 4/5 details differ (see T02.6).

## T02.6 — §5 physical-structure claims vs `StorageLayout`

**Verdict: DIFFERENT representation.** The shipped planner reasons over 4 abstract layouts — `LayoutRow`, `LayoutColumnar`, `LayoutLSM`, `LayoutKV` (`metaengine/layout_type.go:9-26`) — scored per engine in `layout_scoring.go`. The doc's per-ADT "Physical structures" prose (hash index, B-tree table, Bloom filter, adjacency list, Neo4j, recursive CTE) is a mixture of engine-internal concrete structures (not part of the planning model) and engines that don't exist (Neo4j). Concrete structures are engine-private; Bloom exists only as a Pebble-internal filter policy (10 bits/key, `storage/pebble/options.go`), not an ADT or layout. SQL graph traversal exists as `graph_fallback.go:14,36` (CTE fallback) and Dgraph is the graph engine (`metaengine/dgraphengine/`).

## Exact signatures for corrected snippets (T05-T08)

| API                                                                                                       | Location                                                                                                                   |
| --------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| `ExecuteTyped[Q any, R any](ctx, store, input) (R, error)`                                                | `metaengine/execute.go:681`                                                                                                |
| `ExecuteTypedByName[Q, R](ctx, store, queryName, input)`                                                  | `metaengine/execute.go:716`                                                                                                |
| `Store.Execute(input any) (any, error)` — NO ctx                                                          | `metaengine/execute.go:11`                                                                                                 |
| `Store.ExecuteCtx(ctx, input) (any, error)`                                                               | `metaengine/execute.go:36`                                                                                                 |
| `Store.ExecuteQueryByName(ctx, queryName, input)`                                                         | `metaengine/execute.go:62`                                                                                                 |
| `FilterOn[R,T](accessor func(r R) T)`                                                                     | `metaengine/query.go:151`                                                                                                  |
| `SortOn[R,T](accessor func(r R) T)`                                                                       | `metaengine/query.go:167`                                                                                                  |
| `FilterOnField[R](field string, op FilterOp)`                                                             | `metaengine/query.go:180`                                                                                                  |
| `SortOnField[R](field string, desc bool)`                                                                 | `metaengine/query.go:193`                                                                                                  |
| `WithRange(column string, low, high any)`                                                                 | `metaengine/scan_options.go:40`                                                                                            |
| `Volume(n int64)` / `WithLatencyBudget(ms int64)`                                                         | `metaengine/query.go:27` / `:32`                                                                                           |
| `OnRecord[E](sample E, handler any)` / `OnRecordTyped[E]`                                                 | `metaengine/record_fold.go:39` / `:44`                                                                                     |
| ADT enum (8 values)                                                                                       | `metaengine/types.go:6-15`                                                                                                 |
| ReadPattern enum (11 values)                                                                              | `metaengine/types.go:20-32`                                                                                                |
| Sentinels `Delta`/`Edge`/`EdgeRemoval`/`MultiEntry`/`Append`/`Skip`/`Cursor`                              | `metaengine/types.go:34-95`                                                                                                |
| `CommonMetadata` real fields (CorrelationID, Cause, Actor, Created/Received/Stored Stamps, SchemaVersion) | `record/record.go:26-108`                                                                                                  |
| `Record` real fields (ID, Type, Payload, Encoding, StreamID, StreamType, Version, MetaData)               | `record/record.go:114-150`                                                                                                 |
| `Stamp` (presence-explicit timestamp)                                                                     | `record/stamp.go:17-25`                                                                                                    |
| commandlifecycle event types (received/failed/retried/dead-lettered/completed)                            | `commandlifecycle/events.go:51-64`                                                                                         |
| commandlifecycle projections (DeadLetter/RetryCount/FailureLog/ProcessingTime)                            | `commandlifecycle/projections/projections.go:42-124`                                                                       |
| `system.WithCommandLifecycle`                                                                             | `system/lifecycle.go:50`                                                                                                   |
| `CommandJournal` / `SeekableCommandJournal`                                                               | `command/store.go:141` / `:150`                                                                                            |
| Projection roles (Active/DualUse/Migration/Backup)                                                        | `metaengine/roles.go:11-21`                                                                                                |
| `AddEngine`/`RemoveEngine`/`SwapEngine`/`Replan`/`ReplanLayout`/`CheckRouting`/`CatchUpEngine`            | `runtime_backend.go:55`/`:113`, `advanced.go:69`, `store.go:88`, `relayout.go:64`, `store_routing.go:59`, `failover.go:60` |
| `StreamingScan` capability                                                                                | `metaengine/engine.go:369-384`                                                                                             |

**Engine roster (dirs):** badgerengine, bboltengine, dgraphengine, duckdbengine, irohengine, mysqlengine, pebbleengine, pgengine, sqliteengine, tursoengine + in-process memory (`memory_engine.go`). Confirm README framing at T08/T15.

## T22 sibling sweep (2026-09-13)

Rot-pattern grep over `docs/planning/*.md` (`supersedes|THE model|THE specification`) found:

- `event-query-model.md` — the only doc WITHOUT a status banner; reconciled this session (fixed).
- `keep-apps-off-db-layer.md` — its "Supersedes" line describes consolidating two sibling storage docs; that is a scope statement, not a stale truth claim. No action.
- Older meta-engine design docs (`meta-engine-design.md`, `meta-engine-assumptions-and-query-planning.md`, `meta-engine-project-definition.md`) already carry `STATUS: ASPIRATIONAL` headers + 2026-08-06 v2 addenda — prior hygiene was good; no action.
- `future-typespec-extension.md` — its prerequisite-reading pointer now flags the reconciliation banner (edited).

Verdict: no additional sibling annotations required.

## Execution outcome (same day)

- T23 executed (option A from the T16 memo): `Store.StreamCollection` + streaming `Export`, 5 new tests, existing export tests green.
- T24 executed (option B from the T17 memo): `projections.CommandsByActor` (Multimap keyed by typed Actor), tests updated, `projections.All()` now 5 entries.
- API golden regenerated (6855 exports); api-stability meta-tests green; changelog-symbol gate green (48 citations).
- Decisions still open: `command.rejected` event (T17 deferred half), session-log boundary (T18), query-level `Stream(ctx, input, fn)` (T16 follow-up).
