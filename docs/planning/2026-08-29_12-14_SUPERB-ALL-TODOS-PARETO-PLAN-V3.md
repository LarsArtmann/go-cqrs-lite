# ALL-TODOS PARETO PLAN V3 — Trust Floor, Release Spine, Correctness Harvest

> **Date:** 2026-08-29 12:14 CEST · **Session:** "Make a comprehensive Pareto plan from ALL todos"
> **Supersedes:** [`2026-08-27_17-35_ALL-TODOS-PARETO-PLAN.md`](2026-08-27_17-35_ALL-TODOS-PARETO-PLAN.md)
> (same strategic spine; folds in the 2026-08-29 docs-health audit outputs: 29
> verified harvest clusters, the audit's own debt, B1 already cut, P01/P02/P04/
> P05/P06 done).
> **Inputs:** [TODO_LIST.md](../../TODO_LIST.md) @ 94 open items (post-audit
> rebuild: 78 done items removed, 29 harvest clusters added) ·
> [audit status report](../status/2026-08-29_12-10_docs-health-audit-annotate-archive-complete.md)
> · [pending tag-wave plan](2026-08-27_17-30_PENDING-TAG-WAVE-PLAN.md) (B1 CUT
> 2026-08-29: event/v4.9.0, schema/v4.3.1, dedup/v4.2.1, dispatcher/v4.3.1) ·
> AGENTS.md north star.
> **Preconditions:** docs-health tree UNCOMMITTED (11 M / 1060 R) — plan task
> T01 lands it first; doc-check EXIT=0; /mnt/buildcache BROKEN (use /tmp cache
> env per AGENTS gotcha).

---

## 0. Prime directive: NO VERSCHLIMMBESSERUNG

1. The 2026-08-29 audit reorganized 1050 doc paths. **Never** `git checkout`/
   `git restore` the rename set; reversal is `git mv` back, per file. T01
   commits it first so every later task sits on a committed baseline.
2. Stage ONLY files this plan touched. The auto-commit daemon and possible
   concurrent sessions own everything else — `git status` before every stage.
3. Every code task: failing/characterizing test first, scoped tests, then
   `#verify-fast`; api-stability golden regen + CHANGELOG entry in the SAME
   edit (AGENTS rule). Max 350 lines/file, 30 lines/function.
4. Tags, pushes, PRs: only on explicit user instruction (standing rule; B2–B7
   and v5.0.0 are `[user]`-gated in this plan).
5. Gates without pipes: `cmd > /tmp/x.log 2>&1; echo $?` (exit codes after
   pipes lie — AGENTS gotcha).
6. After ANY doc move: rerun the link checker + doc-check (T03 wires this).
7. TODO_LIST.md stays the single living tracker; this plan REFERENCES it,
   never duplicates item text.

## 1. Pareto Breakdown

| Tier                 | Effort share               | Goal-distance closed | What it is                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| -------------------- | -------------------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1% → 51%**         | ~4 focused hours (T01–T07) | **51%**              | **Trust floor**: land the docs-health tree (committed + pushed + gates green — 1050 renames uncommitted is the single biggest current risk), run the full `#verify`/duplication/arch floor (the release-checkpoint claim still cites 08-16), spot-verify a 5% sample of the 400 archive verdicts (kills the audit's own echo-class risk), and fix the three verified XS consumer bugs (`metaJSON` silent discards, `ScanSlice` cap-64, backuptest tag+replace). Everything downstream compounds on this. |
| **4% → 64%**         | ~2 weeks (T08–T19)         | **+13%**             | **Release + v5 spine**: user-gated tag wave B2–B7 (32 tags) → replace-drop sweep → v5 Phase-8 deletion wave (tombstone API, transport, Materialize/RunProjections/GraphProjection, view/relational, stack/, surface sweep + E-items, snapshot wire tags, migration guide) → cut v5.0.0. Every decision already accepted (ADR-0114/0123/0126/0127). B1 done 08-29 moved this forward.                                                                                                                     |
| **20% → 80%**        | ~2 weeks (T20–T28)         | **+16%**             | **Correctness harvest**: the 29 verified clusters from the audit in four batches (metaengine ADT/planner+watcher/iroh, dgraph, storage/SQL) plus the standing P03 race and P18–P21 review batches — AFTER the tag wave so deleted surface is never fixed twice (P03 + the XS trio are the exceptions: trust-critical, surviving code).                                                                                                                                                                   |
| **Other 80% → 100%** | the long tail (T29–T50)    | **+20%**             | Docs/skills debt (CHANGELOG fold, v5-doc coverage, engine READMEs, release docs), DX (metaengine-gen, command sourcing, ceremony reduction), ops (NATS/Redis buses, native backends, flagship), July archive pass, GitHub Releases, external-repo asks.                                                                                                                                                                                                                                                  |

**Execution order:** Phase 0 (now, unblocked) → Phase 1 (`[user]` says "tag")
→ Phase 2 → Phase 3 interleaved by load → Phase 4 long tail.
**Why correctness AFTER v5:** fixing code that v5 deletes is double work; the
Phase-0 trio + P03 are the exceptions (surviving code, trust-critical).

## 2. Comprehensive Plan — Medium Granularity (50 tasks, 30–100 min each)

Sorted by importance/impact/effort/customer-value (phase order = priority).
`Was` = source (master-plan ID / TODO_LIST section / audit report §f).

| ID  | Task                                                                                                                                                                                                                            | Phase | Tier | Effort | Impact                                                      | Customer value                               | Was / Source            |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----- | ---- | ------ | ----------------------------------------------------------- | -------------------------------------------- | ----------------------- |
| T01 | Land the docs-health tree: 2 detailed commits (audit sweep; this plan) + push + post-push doc-check                                                                                                                             | 0     | 1%   | 30     | 1050-file reorg becomes safe, reviewable history            | truthful repo state for every consumer/agent | audit §a6, this session |
| T02 | Full exclusive `nix run .#verify` (/tmp cache env, nothing else running); fix or file anything RED                                                                                                                              | 0     | 1%   | 100    | honest release checkpoint (current claim cites 08-16)       | trust in every gate claim                    | audit §c3               |
| T03 | `#check-duplication` + `#check-arch` + wire link-check into doc routine; scoped fmt over audit files                                                                                                                            | 0     | 1%   | 30     | no clones/arch violations from doc churn                    | stable gates                                 | audit §c3               |
| T04 | Spot-verify 5% random sample of the 400 archive verdicts; `git mv` back any mis-archive; record result                                                                                                                          | 0     | 1%   | 30     | kills the audit's echo-class risk                           | archive trustworthy                          | audit §b2               |
| T05 | Fix `metaJSON, _ :=` silent marshal-error discards (system/adapter_command_serial.go:26, adapter_query_serial.go:24) + tests                                                                                                    | 0     | 1%   | 30     | metadata loss stops silently corrupting envelopes           | durable metadata guarantees                  | TODO harvest            |
| T06 | `ScanSlice` RowCount() pre-size (storage/sql/reconstruction.go:48 cap-64) + test                                                                                                                                                | 0     | 1%   | 30     | large journal reads stop re-growing 64-at-a-time            | read throughput                              | TODO harvest            |
| T07 | backuptest patch tag + drop `=> ../backuptest` replaces in bbolt/pebble + GOWORK=off verify both                                                                                                                                | 0     | 1%   | 30     | published bbolt/pebble builds stop needing sibling replaces | standalone `go get` builds                   | TODO harvest            |
| T08 | **[user]** Tag wave B2–B7 (32 tags, dependency-ordered, cut→push interleave) via scripts/tag-release.sh                                                                                                                         | 1     | 4%   | 100×   | ecosystem unblocked                                         | every consumer can bump                      | T01 Release             |
| T09 | Replace-drop sweep (~19 modules) after tags + tidy + GOWORK=off re-verify each                                                                                                                                                  | 1     | 4%   | 100    | published go.mod hygiene                                    | standalone builds green                      | T06 Release             |
| T10 | Consolidate ~49 indirect `go-cqrs-lite/{codec,retry,…}` dep references post-tags                                                                                                                                                | 1     | 4%   | 100    | lean dependency trees                                       | faster consumer builds                       | T11 Release             |
| T11 | Delete tombstone metadata API: DetectTombstone/MarkTombstone/MarkRebirth/TombstoneStatus/Metadata.Tombstone (ADR-0114 done)                                                                                                     | 1     | 4%   | 60     | deletion purely event-type-driven                           | ADR-0114 complete                            | P09                     |
| T12 | Delete transport/http + transport/grpc (go.work, flake, api-stability, docs) (ADR-0127)                                                                                                                                         | 1     | 4%   | 60     | one delivery path (watermill + go-sse)                      | simpler surface                              | P10                     |
| T13 | Delete stack.Materialize + RunProjections + graph.GraphProjection                                                                                                                                                               | 1     | 4%   | 60     | projectionhost is the only runner                           | one read API                                 | P11                     |
| T14 | Delete storage/view + storage/relational; absorb internals                                                                                                                                                                      | 1     | 4%   | 100    | v1 read tiers gone                                          | auto-projection only                         | P12                     |
| T15 | Delete stack/ entirely (Bundle + presets + bench refs); AGENTS module-map update                                                                                                                                                | 1     | 4%   | 100    | one composition root (system.New)                           | the unified library                          | P13                     |
| T16 | v5 surface sweep + extended-review E-items (ADR-0126 shells, BuildWhereClause, StreamKey rename, breaking NewStreamRef, E1/E7/E8/E11/E13/E15)                                                                                   | 1     | 4%   | 100    | honest API surface                                          | data-model debt cleared                      | P14                     |
| T17 | Snapshot honest wire tags: pebble dual-read, bbolt tags, SQL ALTER+backfill + `#integration-pg`                                                                                                                                 | 1     | 4%   | 100    | aggregate→stream vocabulary done                            | consistent naming                            | P15                     |
| T18 | v5 migration guide (per-tier before/after incl. relational→metaengine)                                                                                                                                                          | 1     | 4%   | 100    | consumers can migrate                                       | trust                                        | P16                     |
| T19 | **[user]** Cut v5.0.0: CHANGELOG, README/SKILL rewrite, examples audit, exclusive `#verify`, tag wave                                                                                                                           | 1     | 4%   | 100    | THE goal milestone                                          | v5 ships                                     | P17                     |
| T20 | metaengine ADT/planner batch: pg/mysql LayoutPlanApplier + schema evolution; DuckDB counter SQL pushdown; mapUpdateReplicationRule FoldMultiInsert/Append; VectorCount + Doctor WARN                                            | 2     | 20%  | 100    | planner routes on honest layout/cost                        | no silent mis-typing                         | TODO harvest            |
| T21 | metaengine watcher + iroh batch: generic watcherEntry, latency bench, WithReificationFailureHook; irohengine HealthChecker; QUIC hardening set (normalizeAny tables, ring-eviction, pooled stress, framing-const dedup, README) | 2     | 20%  | 100    | reactive reads observable + last engine gets health         | operational confidence                       | TODO harvest            |
| T22 | dgraphengine batch: Transactional/RunInTx + ConcurrentTx tests; harness conformance (HealthCheck/Prober/Calibratable tests); empty-collection MapScan                                                                           | 2     | 20%  | 100    | distributed engine parity                                   | Dgraph users                                 | TODO harvest            |
| T23 | storage/SQL batch: injection tail (ORDER BY interpolation guard, fuzz corpus, gosec/nightly CI); relational one-tx-per-event; suite consolidation (storage/sql onto shared suites, querytest self-test)                         | 2     | 20%  | 100    | injection-safe + parity-tested SQL layer                    | security + consistency                       | TODO harvest            |
| T24 | P03 metaengine recHolder race: pass Record through invoke closure; optional Record on EventLog; Backfill/Demote/Verify threading; race ×3                                                                                       | 2     | 20%  | 100    | data race + replay divergence gone                          | reproducible planner results                 | P03 🔥                  |
| T25 | P18 correctness batch A — storage/engines: pebble dup-check shard lock; stream-not-found contract pin; upcaster registry hardening; TypedStore.Save via NewSnapshot                                                             | 2     | 20%  | 100    | backend behavior parity                                     | identical semantics everywhere               | P18                     |
| T26 | P19 correctness batch B — read path: kv Invalidate/InvalidateAll + cache-aside docs; catalog anonymous-embed + recursion guard; eventtest fakes; record.Stamp `*time.Time`                                                      | 2     | 20%  | 100    | generated schemas match wire                                | docs/clients agree                           | P19                     |
| T27 | P20 correctness batch C — host/tooling: projectionhost hardening set; C042 Args[3]; scenario vacuous-pass guard; MetadataCarrier at 3 sites; transport `// Deprecated:`                                                         | 2     | 20%  | 100    | the managed host earns "managed"                            | trustworthy DLQ/replay                       | P20                     |
| T28 | P21 correctness batch D — semantics/docs: deriver cycles; scheduling epochs; metaengine routing set; README truth passes; id/record asymmetry docs; my-nine follow-ups                                                          | 2     | 20%  | 100    | semantic traps documented/closed                            | library stops surprising seniors             | P21                     |
| T29 | Fold `[Unreleased — earlier 2026-08-16 work]` into top Unreleased (CHANGELOG) + changelog-symbol gate run                                                                                                                       | 3     | 20%  | 30     | one unreleased window                                       | honest changelog                             | TODO harvest            |
| T30 | v5-doc coverage set: faq.md deprecation notes; AGENTS Codec-Defaults v5 note; method-level `Deprecated:` decision; durability-tier re-home for storage/pebble (BLOCKS T15); stack/bench decision                                | 3     | 20%  | 100    | v5 unblocks itself                                          | consumers see deprecations everywhere        | TODO harvest            |
| T31 | Fix "~41-byte"→43–46 figure ×4 docs + reconcile SEVEN-TIER-MODEL.md:56 Tier-0 claim vs LAYER=3                                                                                                                                  | 3     | 20%  | 30     | numbers + tier model truthful                               | doc trust                                    | TODO harvest            |
| T32 | AGENTS gotchas: pebble Close() no memtable flush; bbolt mmap quantization; MySQL-VM trio (port-33070 probe, GOWORK=off path, shared-DB isolation)                                                                               | 3     | 20%  | 30     | fewer repeated environment debug sessions                   | agent/consumer productivity                  | TODO harvest            |
| T33 | Engine READMEs ×4 (mysql/sqlite/turso/badger) + pebble engine.go:7 stale comment + metaengine capability-table rows + pg/mysql README rows                                                                                      | 3     | 20%  | 100    | every engine documented                                     | discoverability                              | TODO harvest            |
| T34 | BENCHMARKS.md durability PENDING cell (calm-window run) + modules.md bboltengine row                                                                                                                                            | 3     | 20%  | 30     | honest measured numbers                                     | benchmark trust                              | TODO harvest            |
| T35 | CONTRIBUTING release docs (pin-bump recipe, GOPRIVATE verify) + durability-tier-mapping ADR + Introspection/Doctor durability surfacing                                                                                         | 3     | 20%  | 100    | release process self-service                                | fewer tag incidents                          | TODO harvest            |
| T36 | catalog/docserver follow-ups (css GET test, cId CHANGELOG note, README deps table, templ drift gate decision, CSP nonce, EventCatalog CLI validation)                                                                           | 3     | 20%  | 100    | doc-server polish                                           | docs UX                                      | TODO harvest            |
| T37 | benchmark-regression gate hardening (fixture test for --save+compare, threshold re-tune after live run, baseline runbook, actionlint, `verify --module`)                                                                        | 3     | 20%  | 100    | regression gate trustworthy                                 | perf trust                                   | TODO harvest            |
| T38 | cqrs-lint: C040 projection-handler dead-case detection + doctor/audit polish set (json output, --fix --dry-run diff, enum-warning test)                                                                                         | 3     | 20%  | 100    | linter covers projection handlers                           | consumer signal                              | TODO harvest            |
| T39 | Consumer asks: first-class snapshot encryption (design + EncryptedSnapshotStore + rotation); `retry.DoWithValue[T]` (external go-retry); OTel exporter-lifecycle doc example                                                    | 3     | 20%  | 100    | two consumer blockers resolved                              | adoption                                     | TODO harvest            |
| T40 | Design decisions (write ADR-notes): ULID sharded entropy; Pebble flush-vs-compact calibration basis; command.Bus/MemoryBus removal evaluation                                                                                   | 3     | 20%  | 60     | open questions closed                                       | direction clarity                            | TODO harvest            |
| T41 | July archive pass (174 status + ~40 planning files, same classify/annotate/archive protocol)                                                                                                                                    | 4     | 20%  | 100    | docs/status top level becomes active-only                   | navigable history                            | audit §c2               |
| T42 | Deep per-item annotation pass over archived reports (scripted annotate-prose/rows; **needs user policy answer**)                                                                                                                | 4     | 20%  | 100    | per-item provenance in history                              | audit-grade history                          | audit §b1               |
| T43 | P22 metaengine-gen codegen + planner Layer-3 auto-route                                                                                                                                                                         | 4     | 20%  | 100×   | declare-only DX core                                        | zero hand folds for 80%                      | P22                     |
| T44 | P23 command sourcing (CommandAwareFold + replay) + system.WithCommandLifecycle one-call                                                                                                                                         | 4     | 20%  | 100×   | ADR-0112/0117 complete                                      | audit + DLQ as events                        | P23                     |
| T45 | P24 DomainConfig ceremony reduction (convention registration, quickstart ~50→~15 lines)                                                                                                                                         | 4     | 20%  | 100    | flagship DX                                                 | first-run wow                                | P24                     |
| T46 | P25 NATS + Redis bus registration + Dgraph Snapshot/StreamLog parity + structured query tree                                                                                                                                    | 4     | 20%  | 100×   | multi-process topologies                                    | operator choice real                         | P25                     |
| T47 | P26 native search/vector/spatial backends (pg tsvector, PostGIS, DuckDB VSS + degraded fallbacks)                                                                                                                               | 4     | 20%  | 100×   | universal ADTs at native speed                              | no Memory-only asterisks                     | P26                     |
| T48 | P27 flagship: taskmanager on system.System + operator YAML; calibration CI gate; Turso CTE-probe test                                                                                                                           | 4     | 20%  | 100×   | the vision demonstrable                                     | proof of promise                             | P27                     |
| T49 | Skill-references freshness audit (recipes/modules/faq vs 08-29 reality) + doc-check                                                                                                                                             | 4     | 20%  | 60     | agent-facing docs current                                   | AI-consumer trust                            | audit §c6               |
| T50 | GitHub Releases for 08-16/18/22/29 waves (after billing fix) + feedback-lane naming + annotation-policy note in docs/status/README.md                                                                                           | 4     | 20%  | 60     | release visibility                                          | consumers see highlights                     | T13 Release + audit     |

**Phase 0 (7 tasks ≈ 6h) → 51%. Phase 1 (12 tasks ≈ 2–3 days + [user] gates)
→ +13%. Phase 2 (9 tasks ≈ 3 days) → +16%. Phases 3–4 → the last 20%.**

## 3. Detailed Plan — Fine Granularity (≤12 min per task)

### Phase 0 — Trust Floor (1% → 51%)

- T01.1 · `git status --porcelain` snapshot; confirm no foreign paths staged
- T01.2 · Commit 1: audit sweep (1034 renames + 37 edits + status report) — detailed message (what moved where, why, gates green)
- T01.3 · Commit 2: this plan file — message (supersedes 17:35 plan, B1 done, harvest folded)
- T01.4 · `git push`; verify `origin/master..HEAD` = 0; doc-check re-run EXIT=0
- T02.1 · Export /tmp cache env (GOCACHE/GOMODCACHE/GOPATH/GOLANGCI_LINT_CACHE, GOTOOLCHAIN=auto)
- T02.2 · `nix run .#verify` exclusive; capture full log to /tmp; `echo $?`
- T02.3 · Triage any RED: fix-forward if docs-caused, file TODO if code-caused
- T02.4 · Record GREEN + timestamp in TODO_LIST Release note (replaces 08-16 claim)
- T03.1 · `nix run .#check-duplication` (dirty-tree guard: run post-T01)
- T03.2 · `nix run .#check-arch`
- T03.3 · Scoped gofumpt/goimports over any audit-touched .go (expect zero)
- T03.4 · Save link-checker as `scripts/check-doc-links.sh` + note in docs/status/README.md
- T04.1 · `shuf -n 20` over the archived list; re-run each file's classification spot-check
- T04.2 · `git mv` back any mis-archived file + note the verdict miss
- T04.3 · Record sample result (X/20 correct) in the audit report addendum
- T05.1 · Read system/adapter_{command,query}_serial.go encode paths
- T05.2 · Write failing test: metadata marshal error must surface (not `_, _ =`)
- T05.3 · Fix both sites: handle/propagate marshal error (choose wrap-vs-omit per family rules)
- T05.4 · Scoped tests green; golden regen if surface changed
- T06.1 · Read storage/sql ScanSlice call chain; measure current pre-size behavior
- T06.2 · Add RowCount() pre-size (SELECT COUNT or driver rows-affected path per caller)
- T06.3 · Benchmark/allocation spot-check on a 10k-row read; test green
- T07.1 · Verify backuptest needs no code change for patch tag; `git tag storage/backuptest/v4.0.1` via tag-release.sh flow
- T07.2 · Bump pin + drop replace in storage/bbolt/go.mod; GOWORK=off build
- T07.3 · Same for storage/pebble/go.mod; GOWORK=off build
- T07.4 · CHANGELOG entries; push tag

### Phase 1 — Release & v5 Spine (4% → 64%) — T08/T19 [user]-gated

- T08.1 · [user] Ratify B2–B7 scope (32 tags per pending-tag-wave plan §2)
- T08.2 · Per batch: pre-tag checklist (`#vulncheck`, `#check-arch`, GOWORK=off tests)
- T08.3 · Per batch: cut → push → next (GOPRIVATE VCS interleave)
- T08.4 · Post-wave: GOWORK=off build matrix over all swept modules; CHANGELOG sections
- T09.1 · List ~19 replace-bearing go.mod (grep `=> \.\./`)
- T09.2 · Per module: drop replace → tidy → GOWORK=off build → note
- T09.3 · `#verify-ci` mirror; commit series
- T10.1 · Grep indirect `go-cqrs-lite/{codec,retry,idempotency,flightrecorder}` refs
- T10.2 · tidy per affected module (batched ~10/session); spot GOWORK=off builds
- T11.1 · Inventory remaining DetectTombstone/Mark* callers (post T30 re-home)
- T11.2 · Delete API + migrate internal callers; golden + CHANGELOG Removed
- T11.3 · Skill-refs + example sweep; doc-check
- T12.1 · Final v4.x transport tags (if not in B-wave); drop from go.work/flake/api-stability
- T12.2 · `git rm -r transport/`; lint/fixture sweep; FAQ/advanced.md rewrite
- T13.1 · Delete Materialize + tombstone bridge test + RunProjections
- T13.2 · Delete GraphProjection; graphadapter path verified; golden + CHANGELOG
- T14.1 · Delete storage/view (SQLViewStore, ViewMapper, AutoMapper); sweep consumers
- T14.2 · Delete storage/relational public API; prune orphaned DDL; golden + CHANGELOG
- T15.1 · Migrate durability docs/option homes out of stack/ (depends T30 re-home)
- T15.2 · Delete 8 presets + stack core; fold stack/bench into cqrs-bench
- T15.3 · Remove stack/* from go.work/flake/api-stability (TestEvery green); fix examples/integration imports
- T15.4 · AGENTS module-map + FEATURES matrix rewrite (post-delete census); golden + CHANGELOG
- T16.1 · Delete ADR-0126 shells (VersionedStore, Rejecting*, ErrInnerStoreNot*, CustomData)
- T16.2 · Delete BuildWhereClause; StreamRef→StreamKey rename sweep
- T16.3 · Breaking NewStreamRef validation + call-site migration
- T16.4 · Remove BusConfig.Mode / Subscribe / CacheConfig.Engine reserved fields
- T16.5 · E1 Encoding→record.Encoding; E7 RetryConfig; E8 Kind enum; E11/E13/E15
- T16.6 · Golden regen per module + CHANGELOG per module (same-edit rule)
- T17.1 · Pebble snapshot dual-read rename; bbolt struct-tag rename
- T17.2 · SQL ALTER TABLE + backfill scripts per dialect under storage/sql/migrations
- T17.3 · `#integration-pg` + MariaDB :33061 over renamed schema; old-row round-trip test
- T18.1 · Guide skeleton mirroring ADR-0123 table; per-tier chapters
- T18.2 · Before/after examples incl. relational→metaengine; doc-check; link from README/SKILL
- T19.1 · [user] Final pre-cut: kvstore SA1019 decision note; exclusive `#verify`
- T19.2 · [user] CHANGELOG v5.0.0; README/SKILL v5 rewrite; examples audit; tag wave + push

### Phase 2 — Correctness Harvest (20% → 80%)

- T20.1 · pg/mysql LayoutPlanApplier: planned-column type map + applier impl
- T20.2 · Schema evolution: ALTER TABLE ADD COLUMN on result-type change + tests
- T20.3 · DuckDB CounterGet SQL COUNT / CounterIncrement batch INSERT + filter-builder unify
- T20.4 · mapUpdateReplicationRule: FoldMultiInsert/FoldAppend arms + tests
- T20.5 · VectorCount capability + Doctor/EXPLAIN WARN + adttest scenario
- T21.1 · watcherEntry[V] genericization (metaengine/store.go:330) + API compat check
- T21.2 · Watcher latency bench (memory/sqlite/pebble) into metaengine/bench
- T21.3 · WithReificationFailureHook callback + test
- T21.4 · irohengine HealthChecker implement-or-delegate + test
- T21.5 · QUIC: normalizeAny table tests; dedup.Ring >10K eviction regression
- T21.6 · QUIC: pooled eviction error-injection + 1k-op stress; framing-const dedup package
- T21.7 · quic/README WithStreamPooling row
- T22.1 · dgraphengine RunInTx (Transactional) + ConcurrentTx tests
- T22.2 · Harness conformance: HealthCheck/Prober/TransactMeasurer/Calibratable tests
- T22.3 · Empty-collection MapScan + multimap stress + schema-migration race test
- T23.1 · ORDER BY TimestampColumn ValidateIdentifier guard (journal_reader.go:77,220) + test
- T23.2 · Extend fuzz to multi-condition/ops; persist corpus; gosec CI leg
- T23.3 · Nightly fuzz CI job + `nix run .#fuzz` app (or record CI-owns decision)
- T23.4 · storage/relational: one-tx-per-event batching
- T23.5 · storage/sql command/query stores onto commandtest/querytest.RunStoreSuite
- T23.6 · querytest self-test + LoadToTimestamp subtest in shared suite
- T24.1 · Failing race test: concurrent live Apply + Verify replay on one Fold
- T24.2 · Pass Record via invoke closure; delete recHolder (runtime_backend.go:304, replicator.go:165)
- T24.3 · Optional `Record record.Record` on EventInput/log entries (additive)
- T24.4 · Thread Record through Backfill/Demote/Verify replay; Record-aware comparison
- T24.5 · Race ×3; golden + CHANGELOG + commit
- T25.1–T25.8 · P18 subtasks per master plan (pebble shard lock; contract pin; upcaster hardening ×2; TypedStore.Save; property test; golden)
- T26.1–T26.7 · P19 subtasks (kv Invalidate ×2; catalog ×2; eventtest fakes; Stamp wire; golden)
- T27.1–T27.8 · P20 subtasks (projectionhost ×4; C042; scenario guard; MetadataCarrier ×3 sites; Deprecated)
- T28.1–T28.12 · P21 subtasks (deriver; scheduling; metaengine routing ×4; READMEs; asymmetry; my-nine ×4)

### Phase 3 — Docs/Skills Debt (the 20%)

- T29.1 · Mechanically demote+merge earlier-Unreleased block under top; resolve heading levels
- T29.2 · `scripts/check-changelog-symbols.sh` run; fix any citation fallout
- T30.1 · faq.md deprecation/v5 sweep (zero notes today)
- T30.2 · AGENTS Codec-Defaults table v5 note (line-budget ≤377!)
- T30.3 · Record method-level `Deprecated:` decision (ADR note or ADR-0123 addendum)
- T30.4 · storage/pebble durability re-home: new `storage/pebble.WithDurabilityTier` or metaengine constants; migrate callers; drop stack import (unblocks T15)
- T30.5 · stack/bench deprecation decision recorded (fold vs keep to v5)
- T31.1 · Fix 43–46-byte figure in CHANGELOG/ADR-0124/LAYOUT-MODEL/recipes
- T31.2 · SEVEN-TIER-MODEL.md:56 rewrite to Tier-3-with-ADR-0046-amendment wording
- T32.1 · AGENTS: pebble-close + bbolt-mmap gotchas (2 lines each; watch budget)
- T32.2 · AGENTS: MySQL-VM trio gotcha (1–3 lines)
- T33.1 · mysqlengine README; T33.2 sqliteengine README; T33.3 tursoengine README; T33.4 badgerengine README
- T33.5 · pebble engine.go:7 comment fix; metaengine README capability rows; pg/mysql engine rows
- T34.1 · Run BenchmarkEventAppend{Sync,Async} in a quiet window; fill BENCHMARKS.md cell
- T34.2 · modules.md bboltengine row (mirror pebbleengine row shape)
- T35.1 · CONTRIBUTING: pin-bump-before-tag recipe + GOPRIVATE verify commands
- T35.2 · ADR: durability-tier mapping table (engine × tier → pragma/option)
- T35.3 · metaengine Introspection/Doctor: surface effective durability tier per engine
- T36.1–T36.6 · catalog/docserver set (GET test; cId note; deps table; drift-gate decision; CSP nonce; EventCatalog CLI validation)
- T37.1 · Fixture test pinning benchmark-regression --save+compare
- T37.2 · Post-live-CI threshold re-tune + benchtime decision note
- T37.3 · Baseline-regen runbook in docs/BENCHMARKS.md
- T37.4 · actionlint into devShell; `verify --module <path>` scoped mode
- T38.1 · C040 projection-handler detection + tests + golden
- T38.2 · doctor --audit-suppressions --format json; --fix --dry-run unified diff
- T38.3 · config-loader enum-warning test; explain monetary row
- T39.1 · Snapshot encryption: design note → EncryptedSnapshotStore + KeyResolver wiring + rotation test
- T39.2 · External: go-retry `DoWithValue[T]` PR (separate repo, separate commit)
- T39.3 · otel/ docs: exporter-lifecycle + Shutdown-flush example
- T40.1 · ADR-note: ULID sharded entropy (decision + tradeoff)
- T40.2 · ADR-note: Pebble flush-vs-compact calibration basis
- T40.3 · command.Bus/MemoryBus removal evaluation → Declined or v5 item

### Phase 4 — Long Tail (to 100%)

- T41.1 · Split July files into batches; classify via agents (same protocol)
- T41.2 · Annotate claims; archive; consolidate links; doc-check
- T42.1 · (post [user] policy) Generate specs from archived files' numbered lists; annotate-prose/rows batch runs with dry-run-first discipline
- T43.1–T43.8 · P22 per master plan (scaffold; field extraction; synthesis; goldens; L3 skeleton; default-path wiring; adttest/scenario coverage; skill refs)
- T44.1–T44.6 · P23 per master plan (CommandAwareFold; replay; one-call; retry middleware; version fix; docs)
- T45.1–T45.5 · P24 (convention registration; auto-bind; quickstart rewrite; example update; docs)
- T46.1–T46.8 · P25 (NATS registration; Redis registration; Dgraph Snapshot/StreamLog; query tree Or/And/Gt; planner integration; tests)
- T47.1–T47.6 · P26 (pg tsvector; PostGIS; DuckDB VSS; degraded fallbacks; planner WARNs; benches)
- T48.1–T48.6 · P27 (taskmanager rebuild; operator YAML; SKILL/README refresh; calibration CI gate; Turso CTE test; demo)
- T49.1 · Sweep references/{recipes,modules,faq,advanced,readmodels}.md against 08-29 state; fix drift; doc-check
- T50.1 · (post billing) `gh release create` for 08-16/18/22/29 waves with curated notes
- T50.2 · docs/feedback: rename archive→archived or document divergence
- T50.3 · docs/status/README.md: annotation-depth policy paragraph (post [user] answer)

## 4. Execution Graph

```mermaid
flowchart TD
    A[T01 Land docs tree<br/>commit+push+doc-check] --> B[T02 Exclusive #verify]
    A --> C[T03 duplication+arch+link-check]
    A --> D[T04 5% archive spot-verify]
    B --> E{GREEN?}
    E -- no --> B1[fix-forward / file TODO] --> B
    E -- yes --> F[T05 metaJSON discards]
    E -- yes --> G[T06 ScanSlice pre-size]
    E -- yes --> H[T07 backuptest tag+replaces]
    F --> K[Phase 0 = 51%]
    G --> K
    H --> K
    D --> K
    C --> K
    K --> L{T08 [user] B2-B7 tags}
    L --> M[T09 replace-drop sweep] --> N[T10 indirect deps]
    N --> O[T11-T16 v5 deletions + surface sweep]
    O --> P[T17 snapshot wire tags] --> Q[T18 migration guide]
    Q --> R{T19 [user] cut v5.0.0}
    R --> S[Phase 2: T20-T23 harvest batches]
    K -. exception: trust-critical .-> T24[P03 recHolder race]
    S --> T[T25-T28 P18-P21 batches]
    T --> U[Phase 3: T29-T40 docs/skills debt<br/>interleave by load]
    U --> V[Phase 4: T41-T42 archive depth]
    U --> W[T43-T48 DX + ops + flagship]
    U --> X[T49-T50 freshness + releases]
    V --> Y[100%]
    W --> Y
    X --> Y
```

## 5. Long-Tail Register (the other 80% → last 20%)

Tracked, deliberately unordered: macOS ephemeral-PG hardware verification;
mysql-nspawn root run; per-module `.golangci.yml` split; NATS JetStream adapter
(external, waits for maintained lib); CockroachDB/ScyllaDB/FDB backend spikes
(ROADMAP Themes/Raw Ideas); transactional outbox (ADR-0016); event archival to
S3; docs website; CALM theorem ADR; `CalibrateScanEngine`; ulid entropy
implementation (post-decision); feedback-lane rename; annotation-depth
execution (T42); iroh WriteOp edge convergence (ROADMAP); Windows CI probe.
All live in ROADMAP/TODO_LIST — this plan does not re-triage them.

## 6. Risks & Guardrails

| Risk                                     | Guardrail                                                                                                        |
| ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Rename set half-lands / daemon races     | T01 commits in 2 atomic commits; `git status` before every stage; never stage foreign paths                      |
| Verify RED from doc-era assumptions      | T02 runs exclusive with /tmp cache env; fix-forward or file, never leave RED unrecorded                          |
| Archive verdict errors propagate         | T04 samples 5% before any downstream doc work builds on archived paths                                           |
| Verschlimmbesserung of living docs       | All 6 living docs already gate-checked (doc-check 931 refs); every later doc task re-runs doc-check + link check |
| v5 deletes code still needing fixes      | Correctness batches T25–T28 AFTER T19; only trust-critical survivors (T05–T07, T24) run early                    |
| Tag wave breaks consumers (v4.7.0 class) | T08 per-batch pre-tag checklist + cut→push interleave + post-wave GOWORK=off matrix (hardened script gate)       |
| AGENTS.md 377-line gate                  | T30.2/T32 edits stay within budget; trim a stale line per added line if needed                                   |
| /mnt/buildcache broken                   | /tmp cache env documented in AGENTS; T42-of-report lists repair as env task                                      |

## 7. Verification Protocol (per medium task)

1. Name the test/gate that proves the task (unit, `#verify-fast`, doc-check,
   link check, GOWORK=off build).
2. Run it BEFORE claiming done; capture exit code without pipes.
3. API-surface change → golden regen in the same edit + changelog-symbol gate.
4. Doc move → link checker + doc-check same edit.
5. TODO_LIST checkbox + plan-task ID in the commit message; one logical
   change per commit; `[user]` tasks never self-authorize.

---

_Point-in-time plan at `767545365`, 2026-08-29 12:14 CEST. Inputs: TODO_LIST
94 open items; audit report 12:10; pending-tag-wave plan (B1 cut)._
