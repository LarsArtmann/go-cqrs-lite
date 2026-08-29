# ALL-TODOS PARETO PLAN — Consolidated Execution Plan v2

> **Date:** 2026-08-27 17:35 CEST · **Session:** "Make a PARETO plan from ALL todos"
> **Supersedes:** `2026-08-27_16-18_GOAL-GAP-CLOSURE-PARETO-PLAN.md` (same strategic spine; this plan folds in the two review follow-up sets that landed AFTER it was written — §"Deep Full-Code Review" (9 items) + §"Deep-Review Gap Wave" (24 items) — and corrects two of its inputs)
> **Companion artifacts:** `2026-08-27_17-35_all-todos-pareto.d2` + `.svg` (execution graph)
> **Inputs:** TODO_LIST.md @ `3f8a24220` (full 72-open-item inventory, 12 sections) · AGENTS.md north star · `docs/planning/2026-08-27_16-18_GOAL-GAP-CLOSURE-PARETO-PLAN.md` · `docs/status/2026-08-27_17-20_deep-review-gap-wave.md` · `docs/reviews/2026-08-27_16-45_full-code-review.html`
> **Preconditions DONE this session:** all review files committed (`607e16e71`), duplication gate GREEN again (`3f8a24220` — art-dupl directives need their own line), tree clean.

---

## 0. Corrections to the 16:18 plan's inputs (status reports are point-in-time)

1. **PG integration isolation moved to §Declined** — T04 of the old plan is dead; do not plan it.
2. **Two review follow-up waves landed after 16:18** — 33 new open items (9 mine, 24 gap-wave), including a 🔥 sextet of P1s and one regression **introduced by today's E10 fix** (system synthetic-engine validation rejects `Before: "projections"`). The correctness floor is now a first-class Pareto tier.
3. **The v5 deletion wave remains the single biggest goal-mover** — unchanged — but it is owner-gated on tags (P07). Everything in Phase 0 below is unblocked RIGHT NOW and raises the trust floor under whatever v5 ships.

## 1. Pareto Breakdown

| Tier                 | Effort share              | Goal-distance closed | What it is                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| -------------------- | ------------------------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **1% → 51%**         | ~3 focused days (P01–P06) | **51%**              | **Unblock + trust floor**: land the stranded tag-chain repair (every future tag re-breaks consumers until then), fix the 🔥🔥 pebble/bbolt standalone-build RED (the documented consumer-breaking incident class), and kill the 🔥 P1 sextet (metaengine data race, replay context loss, watermill at-most-once checkpointing + Close panic, listing tombstone-journal honesty, the E10 synthetic-engine regression). Nothing strategic can ship safely on top of these. |
| **4% → 64%**         | ~2 weeks (P07–P17)        | **+13%**             | **v5 Phase 8 deletion wave**: tags → listing type-driven status → tombstone API → transport/* → Materialize/RunProjections/GraphProjection → view+relational → stack/ → surface sweep + E-items → snapshot wire tags → migration guide → cut v5.0.0. Purely executional; every decision already accepted (ADR-0114/0123/0126/0127). Converts "two stacks" into the goal shape.                                                                                           |
| **20% → 80%**        | ~4 weeks (P18–P27)        | **+16%**             | **Correctness long tail + declare-only DX + ops**: the 20 remaining S/M review follow-ups in four batches (AFTER v5 so deleted surface is never fixed twice), metaengine-gen codegen + planner auto-route (the actual product promise), command sourcing, lifecycle one-call, buses, Dgraph parity, query tree, native backends, flagship + docs.                                                                                                                        |
| **Other 80% → 100%** | the long tail (§5)        | **+20%**             | Release polish (GH Releases, retract-pattern doc, indirect-dep consolidation), lint-config audit, iroh edge convergence, plus environment-blocked items ([user] tags/billing/go-codec/iroh-ratify, [root] nspawn, [hardware] macOS).                                                                                                                                                                                                                                     |

**Execution order:** Phase 0 (now, unblocked) → Phase 1 (owner says "tag") → Phase 2 correctness batches A–D → Phase 3 DX → Phase 4 ops/trust → long tail.
**Why correctness AFTER v5:** fixing code that v5 deletes is double work; the sextet (Phase 0) is the exception because it is trust-critical and touches only surviving code.

## 2. Comprehensive Plan — Medium Granularity (27 tasks, 30–100 min each)

Sorted by importance/impact/effort/customer-value (phase order = priority order). `Was` = task ID in the 16:18 plan. Effort in minutes.

| ID  | Task                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | Phase | Tier | Effort | Impact                                                                       | Customer value                                         | Was / Source                 |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----- | ---- | ------ | ---------------------------------------------------------------------------- | ------------------------------------------------------ | ---------------------------- |
| P01 | Land stranded tag-chain repair: cherry-pick `092b5e8a8` + `4907b6afc` to master, regen api golden fresh, verify command/query pins ≥ metadata v4.5.0                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | 0     | 1%   | 60     | kills the re-break-consumers incident class for every future tag             | published consumers stop breaking                      | T01½ · Release#2             |
| P02 | Fix storage/pebble + storage/bbolt GOWORK=off standalone builds (verified RED) + wire a `#verify-standalone` nix app                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           | 0     | 1%   | 100    | consumer-breaking class eliminated                                           | `go get` builds work first try                         | NEW · Pin#2/3                |
| P03 | metaengine P1 pair: pass Record through invoke closure (recHolder race) + optional `Record` on EventLog entries (replay context loss; Backfill/Demote/Verify divergence)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | 0     | 1%   | 100    | data race + silent fold divergence gone                                      | planner results reproducible                           | NEW · Gap#2/3 🔥             |
| P04 | watermill P1 pair: checkpoint on `msg.Acked()` + last-replayed-ID watermark (replace wrong-invariant ring); never close `outputCh`; guard double-Subscribe                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | 0     | 1%   | 100    | at-most-once loss + panic class gone                                         | reliable catch-up subscriptions                        | NEW · Gap#4/5 🔥             |
| P05 | listing tombstone honesty: correct the lying migration-doc recipe + `WithDeleteTypes` reader option; system: validate shutdown deps against the populated (incl. synthetic) engine set, empty-name → ErrShutdownDependencyInvalid                                                                                                                                                                                                                                                                                                                                                                                                                              | 0     | 1%   | 100    | default TombstoneExclude actually excludes; my E10 regression fixed          | soft-delete listings work; documented example compiles | NEW · Gap#1/6 🔥             |
| P06 | Repo-wide stale-pin sweep (~50 go.mod) + `#verify-ci` mirror run                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | 0     | 1%   | 100    | pin-rot class killed                                                         | standalone builds green                                | T02 · Pin#1                  |
| P07 | **[user]** Cut wave-4 tags (event, metadata, schema, metaengine, irohengine, projectionhost, storage v4.7.2) + transport final v4.x, then replace-drop sweep (~19)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             | 1     | 4%   | 100    | ecosystem unblocked                                                          | every consumer can bump                                | T03 · Release#1/5/8          |
| P08 | listing type-driven status (constructor takes delete/rebirth event types) + migrate example/taskmanager off OnTombstone/OnRebirth + golden                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | 1     | 4%   | 100    | tombstone API becomes deletable; flagship leads migration                    | ADR-0114 completion prereq                             | T05+T06                      |
| P09 | Delete tombstone metadata API: DetectTombstone/MarkTombstone/MarkRebirth/TombstoneStatus/Metadata.Tombstone                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | 1     | 4%   | 60     | deletion is purely event-type-driven                                         | ADR-0114 done                                          | T07                          |
| P10 | Delete transport/http + transport/grpc (go.work, flake, api-stability, doc refs)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | 1     | 4%   | 60     | ADR-0127 done                                                                | watermill + go-sse only path                           | T08                          |
| P11 | Delete stack.Materialize + RunProjections + graph.GraphProjection                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | 1     | 4%   | 60     | one projection runner (projectionhost)                                       | one read API                                           | T09                          |
| P12 | Delete storage/view + storage/relational; absorb concepts as engine internals                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | 1     | 4%   | 100    | v1 read tiers gone                                                           | auto-projection only path                              | T10                          |
| P13 | Delete stack/ entirely: Bundle + 8 presets + bench refs; workspace/flake/api-stability/AGENTS cleanup                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          | 1     | 4%   | 100    | **one composition root**                                                     | system.New is THE entry point                          | T11                          |
| P14 | v5 surface sweep + extended-review E-items: ADR-0126 shells, BuildWhereClause, record.StreamKey rename, breaking NewStreamRef, reserved-config removal; E1/E7/E8/E11/E13/E15                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | 1     | 4%   | 100    | honest API surface; data-model debt cleared                                  | no lying config, cleaner v5                            | T12+T13                      |
| P15 | Snapshot honest wire tags: pebble dual-read rename, bbolt struct tags, SQL ALTER+backfill migrations, integration verify                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       | 1     | 4%   | 100    | aggregate→stream vocabulary done                                             | consistent naming                                      | T14                          |
| P16 | Write v5 migration guide (per-tier before/after incl. relational→metaengine)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   | 1     | 4%   | 100    | consumers can migrate                                                        | trust                                                  | T15                          |
| P17 | **[user]** Cut v5.0.0: CHANGELOG, README/SKILL rewrite, examples audit, exclusive full `#verify`, tag wave                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | 1     | 4%   | 100    | **THE GOAL MILESTONE**                                                       | the unified library ships                              | T16                          |
| P18 | Correctness batch A — storage/engines: pebble dup check under shard lock + fail-open classification; pin stream-not-found + dangling-cursor contract on the interface (align at v5); schema upcaster registry hardening (nil/same-ptr/dup/stamp/validation); snapshot TypedStore.Save via NewSnapshot                                                                                                                                                                                                                                                                                                                                                          | 2     | 20%  | 100    | idempotency + upcast + invariant classes closed                              | backends behave identically                            | NEW · Gap#7/8/9/10           |
| P19 | Correctness batch B — read path: kv.Cache Invalidate/InvalidateAll + cache-aside race docs + DeleteAll blast-radius gate; catalog SchemaFromType anonymous-embed + recursion guard; eventtest fakes (slice alias, OccurredAt order, bus chain lock); record.Stamp `*time.Time` wire fix                                                                                                                                                                                                                                                                                                                                                                        | 2     | 20%  | 100    | generated schemas match wire format; fakes stop lying                        | docs/clients agree with payloads                       | NEW · Gap#11/12/21/22        |
| P20 | Correctness batch C — host/tooling: projectionhost hardening set (ReplayDeadLetters under handleMu, Reset-after-checkpoint, WithBatchSize clamp, Stop→Reset→Start recipe, staleness dead-worker, DLQ family gate, corrupt-row resilience); cqrs-lint C042 Args[3] fix; scenario vacuous-pass guard + doc example; capability interfaces at 3 sites; transport `// Deprecated:` paragraphs                                                                                                                                                                                                                                                                      | 2     | 20%  | 100    | the managed host earns "managed"                                             | DLQ/replay trustworthy                                 | NEW · Gap#13/14/15/16/17     |
| P21 | Correctness batch D — semantics & docs: deriver cycle hazard doc/hops; scheduling multi-instance + MarkFired epoch docs; metaengine routing follow-ups (Calibration mutex, CheckRouting version, hysteresis on assignment, Supports plan-time diagnostic, BFS type-safe dedup, zero-Record fold guard); dispatcher/metadata README truth; id.ActorID/record.Actor asymmetry docs; + my nine: Recorder bounded map, AttemptMiddleware standalone leak, applyWithRetry family skip, projectionhost Stop retry, ReadPressure bound, command/query error drift, cqrs-lint version automation in tag-release.sh, kv miss-path copy, listing cursor (type,id) keying | 2     | 20%  | 100    | long-standing semantic traps documented or closed                            | library stops surprising seniors                       | NEW · Gap#18–24 + Review#1–9 |
| P22 | metaengine-gen (Layer 2 codegen) + planner auto-route Layer 3 (inferred projection as default, folds = override, Doctor explains)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | 3     | 20%  | 100    | declare-only DX core; ADR-0116 completed                                     | zero hand folds for the 80% case                       | T17+T18                      |
| P23 | Command sourcing (CommandAwareFold + journal replay) + system.WithCommandLifecycle one-call + real retry integration + version-tracking fix                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    | 3     | 20%  | 100    | ADR-0112 #4 + ADR-0117 completed                                             | audit + DLQ/retry as events, one call                  | T19+T20                      |
| P24 | DomainConfig ceremony reduction: convention registration, auto-bind decider↔command, quickstart ~50→~15 lines                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | 3     | 20%  | 100    | flagship DX                                                                  | first-run wow                                          | T21                          |
| P25 | NATS + Redis bus registration + Dgraph Snapshot/StreamLog parity + structured query expression tree (Or/And/Not/Gt; RawWhere stays the 5% hatch)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               | 4     | 20%  | 100    | multi-process topologies; graph first-class; planner reasons over predicates | operator choice real; typed filters                    | T22+T23+T24                  |
| P26 | Native search/vector/spatial backends: pg tsvector, PostGIS, DuckDB VSS (+ degraded fallbacks + planner WARN)                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  | 4     | 20%  | 100    | universal ADTs at native speed                                               | no "Memory-only" asterisks                             | T25                          |
| P27 | Flagship: rebuild taskmanager on system.System + operator YAML; SKILL/README refresh; calibration regression CI gate; Turso CTE-probe test                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     | 4     | 20%  | 100    | the vision, demonstrable + measured                                          | proof the promise holds                                | T26+T27 + ADT#1              |

**Phase 0 (6 tasks ≈ 9.3h) → 51%. Phase 1 (11 tasks ≈ 17h) → +13%. Phase 2 (4 batches ≈ 6.7h) + Phase 3 (3 ≈ 5h) + Phase 4 (3 ≈ 5h) → +16%. Long tail → +20%.**

## 3. Detailed Plan — Fine Granularity (≤12 min per task)

### Phase 0 — Unblock & Trust Floor (1% → 51%)

- P01.1 · Verify `092b5e8a8`+`4907b6afc` still absent from master (`git merge-base --is-ancestor`)
- P01.2 · Cherry-pick both onto master; resolve go.mod pins (command/query → metadata v4.5.0)
- P01.3 · Regen api-stability golden fresh; `TestEvery` meta-tests green
- P01.4 · GOWORK=off build command+query+metadata; commit
- P02.1 · Reproduce pebble+bbolt standalone RED; capture exact undefined symbols
- P02.2 · Fix pebble pins/exports; GOWORK=off build green
- P02.3 · Fix bbolt pins/exports; GOWORK=off build green
- P02.4 · Add `#verify-standalone` nix app (GOWORK=off loop over modules)
- P02.5 · Standalone-build CI leg for leaf modules; commit
- P03.1 · Write failing race test: concurrent live Apply + Verify replay on one Fold
- P03.2 · Pass Record as invoke-closure argument; delete recHolder cell
- P03.3 · Add optional `Record` to EventInput/log entries (additive field)
- P03.4 · Thread real Record through Backfill/DemoteEngine catch-up + Verify replay
- P03.5 · Verify compares Record-aware fold results (not just row counts); race test green ×3
- P03.6 · Golden + CHANGELOG + commit
- P04.1 · Write failing test: Nack/crash between handoff and Ack skips events
- P04.2 · Checkpoint on `msg.Acked()` in replay + live phases
- P04.3 · Replace replay dedup ring with last-replayed-ID watermark
- P04.4 · Close(): stop via closeCh only; never close outputCh; guard double-Subscribe
- P04.5 · Regression tests ×3 race; commit
- P05.1 · Correct `docs/migration/tombstone-to-domain-events.md:118-141` recipe (decider save-then-publish reality)
- P05.2 · Add `WithDeleteTypes(...)` reader option to InMemoryStreamReader (type-driven status)
- P05.3 · Table tests: TombstoneExclude/Only with type-driven reader
- P05.4 · system: validate against populated engine set incl. synthetic `"default"`/`"projections"`
- P05.5 · system: empty name → ErrShutdownDependencyInvalid; extend the 3 tests I added today
- P05.6 · Golden + CHANGELOG + commit
- P06.1 · One-shot script: list go.mod pins older than latest sibling tag
- P06.2 · Bump core chain pins (event→metadata→record→id→schema→watermill consumers)
- P06.3 · Bump engine pins (10 engines + benchkit + stack/bench)
- P06.4 · `go mod tidy` + GOWORK=off build per swept module
- P06.5 · `nix run .#verify-ci` matrix mirror; commit

### Phase 1 — v5 Phase 8 Deletion Wave (4% → 64%)

- P07.1 · Pre-tag checklist per module (`#vulncheck`, `#check-arch`, GOWORK=off tests incl. subpackages)
- P07.2 · **[user]** Tag core batch: event, metadata, schema, projectionhost (order constraint)
- P07.3 · **[user]** Tag engine batch: metaengine, irohengine, storage v4.7.2
- P07.4 · **[user]** Tag transport/http + transport/grpc final v4.x
- P07.5 · Replace-drop sweep (~19), tidy, GOWORK=off re-verify each
- P08.1 · Design listing type-driven status API (delete/rebirth types on the reader)
- P08.2 · Implement + replace `DetectTombstone` call at `listing/in_memory.go`
- P08.3 · Golden-parity table tests (old bridge vs type-driven)
- P08.4 · Inventory taskmanager OnTombstone/OnRebirth usage; rewrite folds on `evt.Type()`
- P08.5 · `UPDATE_SNAPS=true` regen example goldens; example tests green
- P08.6 · Golden + doc-check + CHANGELOG; commit
- P09.1 · Delete DetectTombstone/MarkTombstone/MarkRebirth + TombstoneStatus + Metadata.Tombstone + TombstoneMark
- P09.2 · Migrate last internal callers (P08 bridge covers listing)
- P09.3 · Golden + CHANGELOG `### Removed` + skill-refs sweep; commit
- P10.1 · Remove transport/* from go.work `use` + flake testModules + api-stability list (TestEvery green)
- P10.2 · `git rm -r transport/`; drop stale lint/fixtures refs
- P10.3 · Rewrite FAQ/advanced.md transport sections; doc-check; commit
- P11.1 · Delete stack.Materialize + tombstone bridge test; delete RunProjections
- P11.2 · Delete graph.GraphProjection (graphadapter is the path); fix refs
- P11.3 · Golden + CHANGELOG; commit
- P12.1 · Delete storage/view (SQLViewStore, ViewMapper, AutoMapper)
- P12.2 · Delete storage/relational public API; absorb internals
- P12.3 · Sweep consumers (examples, readmodels.md/advanced.md); prune orphaned DDL
- P12.4 · Golden + CHANGELOG; commit
- P13.1 · Delete 8 stack presets + stack core (Bundle, durability docs → system/)
- P13.2 · Migrate/delete stack/bench (fold into cqrs-bench)
- P13.3 · Remove stack/* from go.work, flake, api-stability; TestEvery green
- P13.4 · Fix example/taskmanager + integration/ imports off stack
- P13.5 · Golden + CHANGELOG + AGENTS module-map update; commit
- P14.1 · Delete ADR-0126 shells (schema.VersionedStore, signing.Rejecting*, encryption.ErrInnerStoreNot*, metadata.CustomData)
- P14.2 · Delete storage/sql.BuildWhereClause; record.StreamRef→StreamKey rename sweep
- P14.3 · Breaking record.NewStreamRef validation; migrate call sites
- P14.4 · Remove BusConfig.Mode / InstanceConfig.Subscribe / CacheConfig.Engine
- P14.5 · E1 event Encoding→record.Encoding; E7 RetryConfig rename; E8 Message Kind enum
- P14.6 · E11 AdapterCore.Encode error; E13 SQLTimerStore phantom param; E15 middleware signatures
- P14.7 · Golden + CHANGELOG per module (same-edit rule); commit series
- P15.1 · Pebble snapshot rename w/ dual-read; bbolt struct tag rename
- P15.2 · SQL migrations: ALTER TABLE + backfill per dialect
- P15.3 · `#integration-pg` + mysql (:33061) over renamed schema; round-trip old rows
- P15.4 · Golden + CHANGELOG; commit
- P16.1 · Guide skeleton (per-tier chapters mirroring ADR-0123 table)
- P16.2 · Chapters: stack→system.New; Materialize/RunProjections→auto+projectionhost; view/relational→metaengine; GraphProjection→graphadapter; transport→watermill+go-sse
- P16.3 · doc-check; link from README + SKILL.md; commit
- P17.1 · kvstore SA1019 decision recorded in ADR note
- P17.2 · CHANGELOG v5.0.0 (Breaking/Removed from P09–P15)
- P17.3 · README + SKILL.md v5 rewrite (quickstart = system.New)
- P17.4 · Examples audit: all four on system/
- P17.5 · Exclusive full `nix run .#verify` + `#check-duplication` re-pin on committed tree
- P17.6 · **[user]** Tag v5.0.0 wave via scripts/tag-release.sh + push

### Phase 2 — Correctness Long Tail (20% cluster, post-v5)

- P18.1 · pebble: shard lock around dup check+commit; fail-open → wrapped Infrastructure
- P18.2 · Pin stream-not-found + dangling-cursor contract on event.EventSource/SeekableJournal godoc
- P18.3 · Align pebble/bbolt Load/LoadFromVersion to ErrStreamNotFound (v5 store change)
- P18.4 · schema: guard nil + same-pointer upcast; sort.SliceStable; reject duplicate (type,version)
- P18.5 · schema: verify version stamps post-upcast; implement or delete claimed chain validation
- P18.6 · snapshot: route TypedStore.Save through NewSnapshot (validation + CreatedAt)
- P18.7 · Property test: version-0 snapshot rejected end-to-end
- P18.8 · Golden + CHANGELOG; commit
- P19.1 · kv: add Invalidate/InvalidateAll (additive); document single-writer + TTL recommendation
- P19.2 · kv: gate/document DeleteAll blast radius (no-prefix = whole backend)
- P19.3 · catalog: recurse exported anonymous fields honoring json tags; placeholder-before-recurse guard
- P19.4 · catalog: regen goldens (TestFromType_SkipsAnonymousEmbeddedFields flips)
- P19.5 · eventtest: clone LoadToVersion sub-slice; sort ReadAll/ReadFrom by OccurredAt; lock publishChain
- P19.6 · record: wire `at *time.Time` (nil→unknown); round-trip test zero-time
- P19.7 · Golden + CHANGELOG; commit
- P20.1 · projectionhost: ReplayDeadLetters under handleMu; Reset clears checkpoint-first... state-after
- P20.2 · projectionhost: WithBatchSize clamp ≤0; make Stop→Reset→Start recipe work (re-startable host)
- P20.3 · projectionhost: staleness dead-worker signal; DLQ skips non-retryable families gate option
- P20.4 · projectionhost: corrupt SQLite DLQ row → skip+log, never brick List/Replay
- P20.5 · cqrs-lint: C042 checks Args[3] + catches event.Version(0) conversions; golden
- P20.6 · scenario: t.Cleanup guard for missing terminal Then*; fix doc.go example signature
- P20.7 · Replace private Metadata() clones with command.MetadataCarrier at 3 sites
- P20.8 · transport: `// Deprecated:` paragraphs (SA1019 fires for consumers); commit
- P21.1 · deriver: document cycle hazard on AsHandler + opt-in hops counter
- P21.2 · scheduling: document single-active-instance + MarkFired epoch hazard; errors.Join → all attempts
- P21.3 · metaengine: Calibration setter mutex vs Profile readers
- P21.4 · metaengine: CheckRouting cache key + plan version; hysteresis on assignment not suggestion
- P21.5 · metaengine: plan-time Supports diagnostic + routing penalty; BFS type-safe node key
- P21.6 · metaengine: zero-Record guard for Embedding/IndexedText/Point/MultiEntry folds
- P21.7 · dispatcher/metadata README truth pass (M type, chains, symbols, Tracing snippet)
- P21.8 · id/record: document IsZero asymmetry both sides (unify at v5.1)
- P21.9 · commandlifecycle: bound Recorder versions map (LRU); clear tracker on standalone Attempt path
- P21.10 · projectionhost: applyWithRetry skips !IsRetryable families; Stop timeout re-drain path
- P21.11 · snapshot ReadPressure bound; command/query constructor style unify; kv miss-path copy-on-hit; listing cursor (type,id)
- P21.12 · tag-release.sh bumps cqrs-lint version const; golden + CHANGELOG; commit

### Phase 3 — Declare-Only DX (20% cluster)

- P22.1 · Scaffold cmd/metaengine-gen (AST load, flags)
- P22.2 · Extract event/query struct field metadata (reuse AutoCRUD reflection rules)
- P22.3 · Synthesize folds by event-type suffix convention; emit typed Store methods
- P22.4 · Golden tests: generated code compiles + adttest parity
- P22.5 · Planner Layer 3 skeleton: infer projection shape from query structs
- P22.6 · Wire inference as DEFAULT Plan() path; Override precedence + Doctor diagnostics
- P22.7 · adttest + scenario coverage for inferred projections
- P22.8 · "Declare-only" recipe in SKILL refs + cqrs-lint coaching rule; commit
- P23.1 · CommandAwareFold interface + dispatch in ApplyRecord path
- P23.2 · Command journal replay adapter (commandlifecycle Recorder → fold input)
- P23.3 · system.WithCommandLifecycle(eventSink) wiring + real retry middleware
- P23.4 · Version-tracking fix in commandlifecycle projections
- P23.5 · E2E: failing command → retry → DLQ projection, one call; commit
- P24.1 · Convention registration design (decider name from state type, binding from handler signature)
- P24.2 · RegisterDecider auto-bind; declarative slices alternative to closures
- P24.3 · Rewrite system README quickstart; measure line count; SKILL core.md sync; commit

### Phase 4 — Ops & Trust (20% cluster)

- P25.1 · watermill/natsbackend + redisbackend wrapper modules
- P25.2 · BusConfig driver registry names + construction errors (no silent fallback)
- P25.3 · `#integration-redis` bus suite + NATS ephemeral script
- P25.4 · Dgraph SnapshotBackend spike note → implement
- P25.5 · Dgraph StreamLogBackend + adttest parity + `#integration-dgraph`
- P25.6 · Query expression tree types (Or/And/Not/Gt/Lt/Eq/In)
- P25.7 · SQL pushdown per dialect; LSM/KV decode+filter fallback
- P25.8 · Cross-engine parity property tests; docs; commit
- P26.1 · pgengine tsvector SearchBackend (DDL + query)
- P26.2 · pgengine PostGIS SpatialBackend (extension detection)
- P26.3 · duckdbengine VSS VectorBackend (probe + fallback)
- P26.4 · Capability audit + planner WARN on degraded path; adttest equivalence
- P26.5 · ROADMAP/BENCHMARKS evidence; commit
- P27.1 · Rebuild taskmanager on system.New + deployment YAML (keep ServeSSE)
- P27.2 · SKILL refs sweep to v5-only paths; README story
- P27.3 · Calibration regression script + CI regression job
- P27.4 · Turso CTE-probe test; update LIVE-LATENCY-MODEL evidence
- P27.5 · Full doc-check + api-stability pass; commit

## 4. Execution Graph

Rendered: `2026-08-27_17-35_all-todos-pareto.svg` (D2 source alongside). Structure:

Phase 0 (P01→P02→P06 chain; P03/P04/P05 parallel) → gate [user tags P07] → Phase 1 deletion chain P08→P09→P11→P12→P13→P14→P15→P16 (P10 parallel) → gate [user P17 cut] → Phase 2 batches A→B→C→D → Phase 3 (P22→P23→P24) → Phase 4 (P25→P26→P27).

## 5. Long-Tail Register (the other 80% → last 20%)

| Item                                                                               | Owner/Blocker                     | Effort |
| ---------------------------------------------------------------------------------- | --------------------------------- | ------ |
| GitHub Releases for 2026-08-16/18/21 tag waves (+pkg.go.dev triggers)              | gh auth unverified                | S      |
| Retract-and-republish pattern doc in CONTRIBUTING.md                               | —                                 | S      |
| Consolidate indirect dep refs (~49 go.mod) after tags                              | after P07                         | M      |
| `.golangci.yml` exclusion audit (system 20 / cqrs-lint 17 / metaengine 24 linters) | after P13/P14                     | S      |
| Pre-tag checklist hardening: fold into scripts/tag-release.sh                      | —                                 | S      |
| AGENTS.md data-model conventions memory (T25 carryover)                            | —                                 | S      |
| Post-landing consumer-pin sweep for record/v4 (T24 carryover)                      | after next tags                   | M      |
| iroh graph WriteOp convergence (design spike + impl + suite)                       | —                                 | M/L    |
| macOS ephemeral-PG verification                                                    | [hardware]                        | S      |
| GitHub Actions billing fix                                                         | [user: Billing & plans]           | XS     |
| `#integration-mysql-nspawn` full run                                               | [root]                            | M      |
| go-codec F46: commit+tag UnwrapDecode sniff + alloc pin updates                    | [user repo]                       | XS     |
| Ratify iroh P99 bound 50→150ms                                                     | [user decision]                   | XS     |
| ~~PG integration isolation under explicit DSN~~                                    | **DECLINED — do not re-litigate** | —      |

## 6. Risks & Guardrails

1. **Never break the build** — every deletion task runs workspace build + module GOWORK=off build in the SAME task; golden regen same-edit.
2. **Tags are user-authorized** (P07, P17, and the [user] tail items) — never tag/push modules without explicit instruction.
3. **Deletion order matters** — listing (P08) before tombstone API (P09); taskmanager migration inside P08; stack/ (P13) after view+relational (P12); all before the cut (P17).
4. **Correctness batches AFTER v5** — never fix code v5 deletes (exception: the Phase 0 sextet, surviving code only).
5. **Gates run exclusively** — `#verify` never concurrent with integration suites; Ginkgo modules take `-count=1 -race` ×3 separate runs; art-dupl accept directives on their OWN line (learned today, `3f8a24220`).
6. **Respect §Declined** — no PG-isolation re-litigation, no `#verify-parallel`, no Redis adapter, no composite keys, no WaitReady.
7. **Verschlimmbesser-Protection** — this plan deletes more than it adds; Phase 2 fixes carry tests; Phase 3/4 each backed by accepted ADRs (0116/0112/0117).
8. **Concurrent sessions** — before each phase, `git log --oneline -5` + re-check TODO_LIST for another session's commits; yield modules with foreign dirty files.

## 7. Verification Protocol (per medium task)

1. Workspace build `-tags "goexperiment.jsonv2"` + module GOWORK=off build
2. Module tests GOWORK=off `-count=1` (3× `-race` when thresholds touched)
3. api-stability golden (same edit) + `TestEvery` meta-tests
4. CHANGELOG `[Unreleased]` entry (symbols gate green) + doc-check when refs touched
5. Per phase: `#verify-fast`; pre-cut: exclusive `#verify` + `#check-duplication` on committed tree

---

_Point-in-time snapshot. Living state: TODO_LIST.md. This plan surfaced no NEW tasks beyond TODO_LIST.md — every item maps to an existing open TODO (72/72 accounted). Harvest back via docs-health → HARVEST after execution._
