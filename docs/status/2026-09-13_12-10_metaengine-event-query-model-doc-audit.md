# Status Report — Event-Query-Model Doc vs Metaengine Truth Audit

**Date:** 2026-09-13 12:10 CEST
**Session scope:** ONE task: audit `docs/planning/event-query-model.md` (2026-07-23, claims "Supersedes all prior meta-engine design docs. This is THE model.") against the current metaengine implementation.
**Nature of work:** Read-only audit. Zero code changes, zero commits, zero test runs (nothing changed; verification was source-reading via grep/view with file:line evidence).
**Format note:** User explicitly requested `.md`; this overrides the status-report skill's HTML default (skill itself defers to explicit user format requests).

---

## TL;DR

The planning doc's **core abstraction shipped and works** (folds-are-the-ADT, per-query projections, cost-based `Plan`, hot-reload plumbing). It is **stale in 5 places** (API examples, engine roster, metadata field names, streaming, cross-projection queries) and **silently aspirational in 1 big place** (§10 command/query/session logs — zero implementation). The doc still claims supremacy ("THE model"), which makes it a live split-brain risk for any consumer or contributor who reads it first.

---

## a) FULLY DONE (verified this session, with evidence)

| #   | Item                                                                                                                                                            | Evidence                                                                                                                             |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| A1  | Read the full 847-line planning doc (all 15 sections)                                                                                                           | `docs/planning/event-query-model.md`                                                                                                 |
| A2  | `metaengine.Query[Q, R](name, folds...)` shipped exactly as designed                                                                                            | `metaengine/query.go:242`                                                                                                            |
| A3  | `metaengine.Plan(engines, args...)` cost-based planner shipped                                                                                                  | `metaengine/planner.go:141`                                                                                                          |
| A4  | Fold-return-type-IS-the-ADT confirmed: `(K,V)`→Map, `K`→Set, `Delta`→Counter, `Edge`→Graph, `Remove[V]()`→delete, `Skip`→no-op                                  | `metaengine/types.go:18-100`, `metaengine/fold.go:13-35`                                                                             |
| A5  | `OnRecord`/`OnRecordTyped` are the canonical fold constructors                                                                                                  | `metaengine/record_fold.go:39,44`                                                                                                    |
| A6  | Read patterns from doc shipped: point_lookup, membership, filtered_scan, aggregate, traversal (+6 more)                                                         | `metaengine/types.go:24-38`                                                                                                          |
| A7  | Per-query independent projections confirmed                                                                                                                     | `metaengine/README.md` "Per-Query Projections"                                                                                       |
| A8  | Hot-reload plumbing shipped: `AddEngine`/`RemoveEngine`, `SwapEngine`, `Replan`, `ReplanLayout`, `SetPriority`, `CheckRouting`, shadow roles (Migration/Backup) | `runtime_backend.go:55,113`, `advanced.go:69`, `store.go:88`, `relayout.go:64`, `priority.go:173`, `store_routing.go:59`, `roles.go` |
| A9  | Cursor pagination, `Volume`, `WithLatencyBudget`, `WithWriteAmplificationBudget`, `Doctor`, `ExecuteTyped` all exist                                            | `cursor.go`, `query.go:27,32`, `planner.go:48`, `explain.go:283`, `execute.go:681`                                                   |
| A10 | Real engine roster enumerated: memory, sqlite, turso, mysql, badger, pebble, duckdb, pg, iroh, dgraph                                                           | `ls metaengine/`                                                                                                                     |
| A11 | Comparison verdict delivered in-chat: shipped-as-designed / shipped-differently / not-shipped, with file:line citations                                         | session output                                                                                                                       |

---

## b) PARTIALLY DONE (works, but diverges from the doc or is only superficially verified)

| #  | Item                                       | What works                                                                                                                                            | What's open                                                                                                                                                                  |
| -- | ------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 | §8 Metadata-as-first-class                 | `OnRecord(rec, e)` exposes Record context: CorrelationID, Cause, Actor, Stamps (`record/record.go:26-80`)                                             | Doc's `rec.MetaData.Timestamp` field name and `metaengine.RangeFilter("timestamp")` API **do not exist as written**; actual API is `WithRange` scan option / `FilterOnField` |
| B2 | Decision 1 (field-path extraction)         | Shipped as a **hybrid**: `FilterOn` typed closures (in-Go eval) + `FilterOnField`/`SortOnField` declarative specs (SQL pushdown) — `query.go:151-193` | NOT the doc's recommended Option B (`r.Field("Status")` descriptor); annotation needed                                                                                       |
| B3 | Decision 2 (streaming)                     | Engine-level `iter.Seq2` exists (`engine.go:384`); `TypedReader.ScanPage` covers pagination                                                           | No `Store.Stream(ctx, input, fn)` unbounded mode anywhere                                                                                                                    |
| B4 | Decision 3 (cross-projection queries)      | Grouped aggregates exist on a single collection (`typed_reader_grouped.go`)                                                                           | Cross-engine read-time fan-out/merge: **unverified, inferred from filenames only**                                                                                           |
| B5 | Decision 4 (replay checkpoints)            | Checkpoint machinery exists (`catchup_state.go`)                                                                                                      | Equivalence to doc's "checkpoint == log tail → cutover" claim: **not traced end-to-end**                                                                                     |
| B6 | §14 hot-reload flow                        | All building blocks exist (A8); `RoleMigration`/`RoleBackup` = shadow replication                                                                     | Doc's "config reload triggers re-plan" + "dual-read during replay → atomic cutover" full loop: **not traced**; triggers look explicitly-manual                               |
| B7 | §7 "zero coordination between projections" | Per-query projections confirmed (A7)                                                                                                                  | `rule_shared_collection.go` exists — a shared-collection planner rule that may contradict or refine the doc's claim. **Not investigated.**                                   |

---

## c) NOT STARTED (verified absent from the codebase)

| #  | Item                                                                                              | Evidence of absence                                                                                                      |
| -- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| C1 | §10 **command log** (`CommandSucceeded`/`CommandRejected` as event streams)                       | Zero hits in `metaengine/*.go` — **CORRECTED 2026-09-13: this claim is WRONG; see the appendix at the end of this file** |
| C2 | §10 **query log** (`QueryExecuted`)                                                               | Zero hits                                                                                                                |
| C3 | §10 **session log** (`SessionStarted`/`SessionEnded`/`SessionRevoked`, sessions-as-event-streams) | Zero hits; external `cqrs-htmx/identity-model` referenced but out of scope here                                          |
| C4 | §12 **YAML engine config** (`engines: { sqlite: { driver, dsn } }`)                               | Engines are Go-constructed (`sqliteengine.NewSQLiteEngineFromDSN`); no config loader                                     |
| C5 | **Neo4j engine**                                                                                  | Doesn't exist; Dgraph (`dgraphengine`) + SQL recursive-CTE fallback (`graph_fallback.go`) replaced it                    |
| C6 | **Bloom filter ADT/backend** for Set queries (§5/§7/§12)                                          | Not in ADT enum, no Bloom backend in any engine                                                                          |
| C7 | Decision 3 **cross-engine fan-out** (Option A or B)                                               | Not started                                                                                                              |

---

## d) TOTALLY FUCKED UP

**D1 — The doc itself is the finding (Severity: HIGH — misleads readers, not runtime).**
`docs/planning/event-query-model.md` still declares itself "THE model. Supersedes all prior design docs" while being wrong about: the execute API (§4), the metadata API (§8), the ADT set (§5), the read patterns (§6), the engine roster (§12: Neo4j/Bloom/YAML all absent), and its largest section (§10: four logs, never built). Anyone reading it first — a consumer, a new contributor, an AI agent — builds on a false model. Root cause: planning docs here have no status-maintenance discipline (ADR-0114 got an implementation-status addendum; this doc got nothing). Workaround: README.md is truthful; but nothing forces a reader to it first. **Not yet fixed — awaiting instruction.**

**D2 — My audit had unverified-confidence gaps (Severity: MEDIUM — process, my fault).**
Four findings (B4, B5, B6, B7) rest on filename inference (`catchup_state.go`, `rule_shared_collection.go`), not on reading the code. I presented them with the same confidence as the line-verified findings (A2-A10). A doc audit that itself contains unverified claims repeats the disease it diagnoses.

**D3 — I did not map the stale doc's blast radius (Severity: MEDIUM).**
I never checked who references `event-query-model.md` (SKILL.md, ADRs, TODO_LIST, other planning docs) nor whether `cmd/doc-check` scans `docs/planning/`. Without the inbound-link map, "annotate in place" vs "supersede" is a guess.

**D4 — Working tree contains 8 modified files I did not author (Severity: INFO for this session, flag it).**
At report time: `README.md`, `command/README.md`, `decider/README.md`, `event/README.md`, `id/README.md`, `flake.nix`, `scripts/test-check-retracts-shipped.sh`, `scripts/test-tag-release.sh`. Per repo safety rules these are not mine to touch; any follow-up doc work must not collide with them. Origin unknown from this session.

**D5 — One wasted round trip (Severity: LOW).**
Malformed `rg` regex (`unclosed group`) burned a tool call mid-audit; caught and fixed on the next call.

**D6 — "Historically accurate for §2–§3" was untestable and I said it anyway (Severity: LOW).**
§1-§3 are philosophy/framing with no code claim; calling them "accurate" was a category error. They're unfalsifiable, which I should have stated.

**Not fucked up (checked and clean):** no build breakage from this session (nothing was modified); no secrets touched; no git operations beyond read-only `status`/`log`.

---

## e) WHAT WE SHOULD IMPROVE

| #  | Improvement                                                                                                                                   | Impact                                          | Concrete fix                                                                                                                                                               |
| -- | --------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| E1 | **Planning-doc status discipline.** Docs claiming "supersedes/THE model" rot into split brains                                                | Every future session risks trusting stale truth | Convention: every `docs/planning/*.md` carries a `**Status:**` header that MUST change when reality diverges; implementation-status addenda in ADR style (like ADR-0114's) |
| E2 | **Audit rigor labeling.** Distinguish "verified by reading source (file:line)" vs "inferred from names" in any comparison output              | Prevents confident-but-unverified claims (D2)   | Two-tier evidence labels in audit reports; make it a habit, not an afterthought                                                                                            |
| E3 | **Blast-radius-first for doc changes.** Check inbound links + doc-check scan set BEFORE proposing annotate/supersede/delete                   | Avoids guessing doc fate                        | `rg -l "event-query-model"` + read `cmd/doc-check` main.go scan list as step 1 of any doc-fate work                                                                        |
| E4 | **Doc examples should compile.** The doc's Go snippets (`store.Execute(ctx, ...)`, `RangeFilter`) never matched the API and nothing caught it | Docs drift silently                             | Extend `cmd/doc-check` or a snippet-compile gate for planning docs (or mark planning docs as explicitly ungated archives — user decision, see Q2)                          |
| E5 | **Read-only sessions still owe a "did I break nothing" statement.**                                                                           | Honesty of reports                              | Always state explicitly: no files modified, no tests applicable, tree state observed and attributed                                                                        |

---

## Self-review (asked explicitly: forgotten / could do better / could still improve)

- **Forgot:** inbound-link sweep (D3); verifying B4-B7 before asserting them (D2); the doc-check question (E4); noting that `AutoInsert/AutoUpdate/AutoCRUDByConvention` and ~25 other features (materialized views, replication, SSE, watchers, vector, spatial, durability, backfill, idempotency, transactions, failover, plan-diff...) exist far beyond anything §3's "the planner derives" table describes — the doc understates the module as much as it overstates some APIs.
- **Could have done better:** read `catchup_state.go`, `rule_shared_collection.go`, and `execute.go`'s inner paths instead of inferring; checked `flake.nix`/`cmd/doc-check` for whether planning docs are gated; delivered the annotated doc addendum as a ready-to-apply artifact instead of an offer.
- **Could still improve:** see E1-E5; and section (f) below.

---

## f) Up to 50 things to get done next

Impact: Critical/High/Medium/Low · Effort: S (<30min) / M (30min-2hr) / L (>2hr)

### Group A — Make the audited doc truthful (annotate `event-query-model.md`)

| #  | Task                                                                                                                                                                               | Impact   | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 1  | Add implementation-status addendum to `event-query-model.md` (per-section DONE/DIFFERENT/NOT-STARTED, ADR-0114 style)                                                              | High     | M      | Documentation |
| 2  | Replace the doc's `**Status:**` header: currently claims "THE model" while diverging from shipped truth                                                                            | High     | S      | Documentation |
| 3  | §4 example: `store.Execute(ctx, FindUser{...})` → `metaengine.ExecuteTyped[FindUser, FindUserResult](ctx, store, ...)` (`execute.go:681`)                                          | Medium   | S      | Documentation |
| 4  | §4/§5 examples: mark `On`/`OnTyped` deprecated → `OnRecord`/`OnRecordTyped` (removed v5, `fold.go:296-309`)                                                                        | Medium   | S      | Documentation |
| 5  | §5 ADT table: add `MultiEntry` (Multimap), `Append` (Log), `EdgeRemoval`; align with ADT enum incl. `stream_log`, `sorted_map` (`types.go:3-15`)                                   | High     | S      | Documentation |
| 6  | §6 read patterns: expand to the shipped 11-value enum (`types.go:24-38`)                                                                                                           | Medium   | S      | Documentation |
| 7  | §8 example: fix `rec.MetaData.Timestamp` to real `CommonMetadata` fields (`record/record.go:26-80`)                                                                                | High     | S      | Documentation |
| 8  | §8 example: replace `metaengine.RangeFilter("timestamp")` with real API (`WithRange` / `FilterOnField`)                                                                            | Medium   | S      | Documentation |
| 9  | Annotate Decision 1 as resolved-by-hybrid: `FilterOn` closures + `FilterOnField`/`SortOnField` pushdown, not Option B (`query.go:151-193`)                                         | High     | S      | Documentation |
| 10 | Annotate Decision 2: no `Store.Stream`; engine-level `iter.Seq2` exists (`engine.go:384`); `ScanPage` covers pagination only                                                       | Medium   | S      | Documentation |
| 11 | Verify then annotate Decision 3 (cross-projection queries): read `typed_reader_grouped.go` + `execute.go` inner paths                                                              | Medium   | S      | Documentation |
| 12 | Verify then annotate Decision 4: trace `catchup_state.go` vs "checkpoint == log tail → cutover" claim                                                                              | Medium   | S      | Documentation |
| 13 | §12 engine examples: SQLite+Pebble+**Dgraph** replaces Neo4j; add real roster of 10 engines                                                                                        | High     | S      | Documentation |
| 14 | Strike or annotate Bloom-filter structures (§5/§7/§12): no Bloom backend exists                                                                                                    | High     | S      | Documentation |
| 15 | §14 flow: rewrite to actual APIs (`AddEngine`/`RemoveEngine`/`SwapEngine`/`Replan`/`ReplanLayout`, shadow roles `roles.go`)                                                        | High     | S      | Documentation |
| 16 | Investigate `rule_shared_collection.go` vs §7's "five independent projections, zero coordination" claim; reconcile the contradiction                                               | High     | M      | Documentation |
| 17 | Map §11's 7 planner steps to source files (`fold_classify.go`, `infer_*.go`, `cost.go`, `layout*.go`, `auto_fold.go`, `typed_reader*.go`, `rules.go`/`plan_audit.go`) and annotate | High     | M      | Documentation |
| 18 | Verify §13 "developer NEVER writes" rows still hold (e.g. "no IndexSpec declarations" vs `layout.go` index planning)                                                               | Medium   | M      | Documentation |
| 19 | §10: annotate command/query/session logs as NOT STARTED; promote to ROADMAP or strike (blocked on Q1 below)                                                                        | Critical | S      | Documentation |
| 20 | §9 auth: mark as scope statement (nothing to implement)                                                                                                                            | Low      | S      | Documentation |
| 21 | §2 table: verify "singleflight, snapshots, state cache" optimization row against decider module                                                                                    | Low      | S      | Documentation |
| 22 | §7: add watcher zero-value delete-notification contract note                                                                                                                       | Low      | S      | Documentation |
| 23 | §11: confirm Volume scale-threshold warnings + write-amp budget default (3) match shipped behavior (`cost.go`, `rule_writeamp.go`)                                                 | Medium   | S      | Documentation |
| 24 | §12: align example planner outputs with actual `PlanResult`/`LogPlan` format                                                                                                       | Low      | S      | Documentation |
| 25 | §1-§3: mark as philosophy (no code claims) — removes the unfalsifiable "accurate" framing                                                                                          | Low      | S      | Documentation |

### Group B — Coverage map: features the doc never mentions (the module grew far beyond §3's table)

| #  | Task                                                                                                                                                                                                                                                                                                                                                                     | Impact | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------ | ------ | ------------- |
| 26 | Produce "features beyond the doc" list: materialized views (`materialized_view*.go`), replication (`replication*.go`, iroh), export/import, SSE (`sse*.go`), watchers, vector search, spatial, graphadapter, durability rules, `demote.go`, `backfill.go`, batch atomicity, `ApplyIdempotent`, `InTransaction`, catchup, priority config, plan-diff/versioning, failover | High   | M      | Documentation |
| 27 | Document `ProjectionRole` (Active/DualUse/Migration/Backup) in metaengine README                                                                                                                                                                                                                                                                                         | Medium | M      | Documentation |
| 28 | README: add `On`/`OnTyped` → `OnRecord` migration note (v5 removal)                                                                                                                                                                                                                                                                                                      | Medium | S      | Documentation |
| 29 | Confirm README fold table covers `FoldEdgeRemove` + `FoldSkip`                                                                                                                                                                                                                                                                                                           | Low    | S      | Documentation |
| 30 | Verify membership-test (Set) pushdown path exists for SQL engines (§12's "UNIQUE index" claim)                                                                                                                                                                                                                                                                           | Medium | S      | Documentation |
| 31 | Verify graph traversal depth semantics (doc's `Depth` field) vs shipped `executeGraphNeighborsUndirected` (`execute.go:414`)                                                                                                                                                                                                                                             | Medium | S      | Documentation |
| 32 | Verify §5 "physical structures" mappings vs `StorageLayout` enum (`layout_type.go`)                                                                                                                                                                                                                                                                                      | Medium | M      | Documentation |
| 33 | Confirm degraded-pattern/write-amp warnings surface as doc §11 Step 7 promises (`doctor_degraded.go`)                                                                                                                                                                                                                                                                    | Medium | S      | Documentation |

### Group C — Process / tooling / hygiene

| #  | Task                                                                                                                | Impact   | Effort | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------- | -------- | ------ | ------------- |
| 34 | Repo-wide inbound-link sweep for `event-query-model.md` before any supersede/delete                                 | Medium   | S      | Documentation |
| 35 | Check whether `cmd/doc-check` scans `docs/planning/`; decide if it should                                           | Medium   | S      | Tooling       |
| 36 | Decide planning-doc policy: living (gated, lockstep) vs archive (annotate once, never maintain) — blocked on Q2     | Critical | S      | Process       |
| 37 | Add planning-doc status convention (Status header + addendum discipline) to AGENTS.md "Project Documentation Files" | Medium   | S      | Process       |
| 38 | Harvest this report's (f) items into `TODO_LIST.md`/`ROADMAP.md` (docs-health HARVEST)                              | High     | S      | Process       |
| 39 | Run docs-health VERIFY over `docs/planning/*.md`                                                                    | Medium   | M      | Process       |
| 40 | Sweep sibling planning docs for the same rot pattern (superseded-claim docs without addenda)                        | Medium   | M      | Process       |
| 41 | Make doc-internal Go examples compile-checkable (or explicitly exempt planning docs)                                | Medium   | L      | Quality       |
| 42 | Record this session's verified file:line map as a references note for future audits                                 | Low      | S      | Documentation |
| 43 | Run `nix run .#verify-fast` before the next doc commit (working tree already dirty — see D4)                        | Medium   | S      | Process       |
| 44 | Investigate the 8 uncommitted working-tree files (D4): confirm intent before any doc work collides                  | Medium   | S      | Process       |
| 45 | File an ADR/addendum for "Dgraph + SQL recursive CTE instead of Neo4j" if none exists                               | Medium   | M      | Documentation |

### Group D — Design decisions (blocked on user answers, see g)

| #  | Task                                                                                                                                                           | Impact   | Effort | Category |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ | -------- |
| 46 | Decide fate of §10 four logs: ROADMAP epic vs struck forever (Q1)                                                                                              | Critical | S      | Decision |
| 47 | Decide whether Bloom-filter Set optimization is ever wanted (Q3)                                                                                               | Low      | S      | Decision |
| 48 | Decide whether a dedicated graph engine beyond Dgraph/CTE is on the roadmap (Q3)                                                                               | Medium   | S      | Decision |
| 49 | Decide final audience for `docs/planning/` — consumers (must be truthful) vs archaeology (can be historical) (Q2)                                              | High     | S      | Decision |
| 50 | If §10 survives: sketch the command-log-as-event-stream design against `record/` + ADR-0117 (command lifecycle as events already exists — check overlap first) | High     | L      | Feature  |

**HARVEST note:** items 1-50 are the primary input for `docs-health` HARVEST into `TODO_LIST.md` (actionable, bounded) and `ROADMAP.md` (items 46-50 + Group D).

---

## g) Top 3 questions I CANNOT figure out myself

**Q1 — What is the intended fate of §10 (command log, query log, session log as event streams)?**
This is the largest unimplemented section of "THE model." I tried: grep across metaengine for `CommandSucceeded|QueryExecuted|SessionStarted` (zero hits), checked `commandlifecycle` (ADR-0117 covers command lifecycle as events — partial overlap), found no ROADMAP entry. I cannot determine intent from code: is this still the north star (→ promote to ROADMAP as an epic, item 50), or abandoned in favor of ADR-0117's narrower command-lifecycle streams (→ annotate as superseded-by-ADR-0117)? The answer changes the annotation direction and 5 report items.

**Q2 — What is the maintenance policy for `docs/planning/*.md`: living documents or point-in-time archives?**
I checked: the repo gates skill references via `cmd/doc-check` with a zero-warning policy, ADRs get implementation-status addenda, but I did not find any rule (AGENTS.md, CONTRIBUTING.md is unexamined for this specific question) stating whether planning docs must be kept in lockstep with code or are allowed to fossilize. `event-query-model.md` behaved like an archive (847 lines, untouched since 2026-07-23) while claiming to be living truth ("THE model"). Which is intended? This decides whether items 1-25 are one-shot annotation or ongoing maintenance obligations.

**Q3 — Are the Bloom-filter Set optimization and a dedicated graph engine (Neo4j-class) still investment targets, or permanently replaced?**
The doc assumes Bloom filters for `CheckEmail` at scale and Neo4j for `FriendsOf`. Truth: no Bloom backend exists; graphs ship via `dgraphengine` + SQL recursive-CTE fallback (`graph_fallback.go`). I cannot tell from code whether these are (a) consciously rejected (Dgraph/CTE won, Bloom's false-positive semantics deemed wrong for existence checks), (b) deferred until scale demands them, or (c) simply never reached. This determines items 14, 47, 48, and whether the doc's §12 production example should be rewritten or kept as a historical target.

---

_Awaiting instructions._

---

## Appendix — Corrections (2026-09-13, post-execution)

- **C1 is WRONG as written.** The command log SHIPPED — not in `metaengine/*.go` (which is all the grep covered), but as `commandlifecycle` (ADR-0117): `command.received/failed/retried/dead-lettered/completed` on `Command/<id>` + `CommandLifecycle/<id>` streams, projections (DLQ, retry count, failure log, processing time, plus per-actor `CommandsByActor` since 2026-09-13), `CommandJournal`/`SeekableCommandJournal`, and `system.WithCommandLifecycle`. The grep was scoped to the wrong module — the miss was acknowledged in the [15:55 deep dive](2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md) and is annotated again here.
- The rest of section (c) stands, with nuances now recorded in the reconciled doc's addendum (C4 no config loader; C5 Dgraph/CTE; C6 Pebble-internal bloom, not an ADT; C7 not shipped).
- Execution outcome of the reconciliation: [18:35 status report](2026-09-13_18-35_event-query-model-truth-reconciliation-execution.md); close-out plan: [`docs/planning/2026-09-13_18-41_SUPERB-reconciliation-close-out.md`](../planning/2026-09-13_18-41_SUPERB-reconciliation-close-out.md).
