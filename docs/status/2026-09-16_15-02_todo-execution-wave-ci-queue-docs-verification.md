# Status Report — TODO-Execution Wave: CI-Red Fixes, W4 Gates, Queue Family, Verification Sweep

> **Point-in-time snapshot:** 2026-09-16 15:02 CEST. Single session, mandate: "execute the whole
> TODO_LIST, plan-first, sorted by impact". Entry state: the 2026-09-16 docs-health 7th pass had
> just rebuilt TODO_LIST (95 open rows) and the 13:05 skill-docs pass had closed its 5 items —
> so the pasted-in-chat TODO was already stale and the ON-DISK list was treated as the queue.
> Machine: 32 cores at load 30–40 all session (quiet-window items correctly NOT attempted).
> **Format note:** `.md` per explicit user instruction (5th consecutive override of the
> status-report HTML default — flagged, not propagated).

**One-line verdict:** 13 items fully done with gate evidence, 2 real red-class bugs fixed
(repo-wide stale go.sum drift + both still-open CI job defects), 3 verification rows
closed as double-verified, 6 own-goal failures honest-scored — and the `#verify`/quiet-window
legs, CHANGELOG bookkeeping, and ~20 planned items remain.

---

## a) FULLY DONE (each verified this session)

| #  | Work                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 | Verification                                                                                                                                                                                                            |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | **File-size ratchet RED fixed** — `metaengine/store.go` 954→**944** (≤945 baseline): the 09-16 07:15 formatter reflow of `ApplyIdempotent`'s two 103-col one-liners counter-flattened into one behavior-identical guard (`eventID != "" && CheckAndRecord`) + a new `applyIdempotentFeed` helper with formatter-stable line widths                                                                                                                                                                                                   | `nix run .#check-file-size` GREEN (944 ≤ 945); metaengine `-run 'Idempotent\|Apply'` tests pass                                                                                                                         |
| 2  | **Repo-wide go.sum drift found + fixed (NEW bug, not on any list)** — `TestEveryModuleGoSumIsTidy` failed across ~90 modules: the `go-error-family v0.10.0→v0.10.1` bump wave (commit df0d62c6a) left stale v0.10.0 hashes in go.sum everywhere. Bulk `GOWORK=off go mod tidy` over all 91 modules (0 failures, ~59 files),                                                                                                                                                                                                          | `TestEvery` meta-test RED → **GREEN** (24.6s); `go mod tidy -diff` in claiming showed the exact stale-hash diff class                                                                                                   |
| 3  | **W4 gates (command-side plan) — every gate green**: per-module tests (decider, command, commandlifecycle, schema), api-stability golden (7,092 exports verified), `TestEvery`, `check-changelog-symbols` (86 citations honest), `cmd/doc-check` (1,142 refs / 49 pkgs valid), `#check-arch` (decider zero new deps), `#check-duplication`                                                                                                                                                                                           | All run this session; only the composed `nix run .#verify` leg remains (quiet-window-gated, see b1) — W4 is otherwise closable                                                                                          |
| 4  | **CI job `go-work-sync` root-caused + fixed** — the job installed the nix stack but NO Go toolchain (`go: command not found`) AND rode the throttled magic-nix-cache path; replaced with plain `actions/setup-go@v5` `go-version-file: go.mod` (the proven modsums-job pattern); committed go.work verified clean of `../` entries so the gate logic can actually pass                                                                                                                                                               | ci.yml edit; gate script itself unchanged; committed go.work: 0 external use-entries, go 1.26.7                                                                                                                         |
| 5  | **CI `benchmarks.yml` matview-gate defect fixed** — the single-`cd` form hopped `../metaengine/tursoengine` from `stack/bench` (nonexistent `stack/metaengine/...`) and teed to `stack/current.txt` while the compare step reads root `current.txt`. Restructured into subshells with root-relative tee                                                                                                                                                                                                                              | BOTH legs verified LIVE with real runs: stack/bench suite emits both benchmarks; `BenchmarkMatViewRead/agg=[A-Z]+/scale=1k` matches SUM_GROUPED/SUM_VIA_GROUPED and produces parsable bench output                      |
| 6  | **queue/postgres conformance on the repo's OWN ephemeral PG** — first execution on this leg (previously pgtestcontainer-only), via `PG_MODULES="queue/postgres" nix run .#integration-pg`                                                                                                                                                                                                                                                                                                                                            | Full `TestConformance` suite PASS (Claims/Retry/Dedup/Journal/Reads incl. the `lifecycle_cancel.go` split paths), 1.57s, clean server shutdown                                                                          |
| 7  | **Queue family docs tail** — new `queue/README.md` (contract-in-one-minute, engine table, SQLite+Postgres quickstarts); partial-literal semantics documented (`task.New`/`task.Task`/`Filter`/`facts.Fact` exhaustruct exemptions are design, with zero-value reasoning); errcheck dead short forms `(*sql.{DB,Rows,Stmt}).Close` → fully-qualified `(*database/sql.…)`; `queue/mysql` doc mentions struck (store.go, conformance/doc.go) per the row's "implement or strike" fork (strike chosen; MySQL engine = M, consumer-gated) | Every README claim verified against source before shipping (DedupKey sqlite enqueue.go:27, deps gating in claim SQL, `Claim.ID()`, `Store.Close()`, `ErrNoTaskDue`); `check-lint-config` GREEN incl. exhaustruct canary |
| 8  | **Set-membership pushdown — verified & TODO ticked** (audit item 30): independent re-check CONFIRMS the 2026-09-15 doc banner — SQLite is the only SQL engine with the Set ADT; `SetContains` is a pushed-down point lookup (`SELECT 1 FROM meta_set WHERE collection=? AND key=?`, sqliteengine/engine.go:130); uniqueness = composite PRIMARY KEY (engine.go:92-95), not a literal separate UNIQUE index; pg/mysql declare ADTSet degraded with no meta_set DDL                                                                    | Two independent sources agree; TODO row ticked with citations                                                                                                                                                           |
| 9  | **Graph traversal depth — verified & TODO ticked** (audit item 31): `FriendsOf{Depth}` ships as `GraphNeighbors` with within-≤depth-hops semantics on ALL FOUR graph engines (sqlite CTE, pg, mysql, memory BFS), dedup, start excluded, negative=unlimited                                                                                                                                                                                                                                                                          | Two independent sources agree; TODO row ticked with citations                                                                                                                                                           |
| 10 | **V007 deprecation verification + table extension** — the open question is ANSWERED: V007 is a CURATED TABLE, not marker-driven — new `// Deprecated:` markers do NOT auto-surface. Extended `deprecatedV5Symbols` with `On`, `OnTyped`, `Infer`, `InferFromNamedEvents`; the drift meta-test then caught 2 stale entries (On's "removed in **the** v5.0.0 release" phrasing misses the scanner's signal list) → reworded both fold.go doc phrases to the canonical "removed in v5"                                                  | `cmd/cqrs-lint` version package tests GREEN; drift gate + table now in lockstep                                                                                                                                         |
| 11 | **Recipe §2.36 "Watch Dgraph Contention Retries"** — the `WithContentionObserver` entry the TODO asked for: construction-time hook, dependency-budget rationale, escalation tie-in to §2.35 health machinery; catalog compile-scaffold entry added                                                                                                                                                                                                                                                                                   | `TestRecipes` ok 12.7s (new fence compiled + catalog-covered); doc-check vet clean                                                                                                                                      |
| 12 | **pin-sweep `--check`** — standing post-release check executed on demand                                                                                                                                                                                                                                                                                                                                                                                                                                                             | "All sibling pins are at their latest tags (source: local refs)"                                                                                                                                                        |
| 13 | **Reset-recipe TODO row — verified STALE** — readmodels.md §"Revert & rebuild" ALREADY documents the full 12/12 `EngineResetter` ladder (2026-09-11); no "memory-only" text exists in SKILL.md/references; the row's second half (WithContentionObserver recipe) shipped as #11 → row fully closable                                                                                                                                                                                                                                 | grep + readmodels.md:319-337 evidence                                                                                                                                                                                   |

## b) PARTIALLY DONE

1. **W4's final leg**: composed `nix run .#verify` never ran — box at load 30–40 all session
   (the standing quiet-window row owns it). Every individually-runnable W4 gate is green (#3).
2. **TODO_LIST bookkeeping**: only the Set/graph rows ticked so far. Rows now closable with
   evidence but not yet updated: queue docs tail (#7), queue/postgres leg (#6), CI(d) items
   (#4/#5), file-size ratchet RED (#1), V007 verification (#10), reset recipe (#13), pin-sweep
   (#12), W4 progress (#3).
3. **CHANGELOG**: no `[Unreleased]` entries written for this wave (CI fixes, queue README,
   V007 table extension, §2.36, fold.go doc rewording). `check-changelog-symbols` untouched.
4. **The two CI workflow fixes** are committed but UNVERIFIED BY CI — pushing is author-gated
   and GitHub Actions billing is broken (paid jobs die in 3–7s), so green can only be proven
   post-push post-billing-fix. The benchmarks legs were verified locally instead (#5).
5. **V007 detection philosophy**: I extended the curated table (in-lockstep, drift-gated);
   making detection marker-DRIVEN (auto-derive from `// Deprecated:` repo-wide) is a design
   decision deliberately left to the owner.
6. **Plan HTML report** (pareto-planning Step 4): not written; this mandate's status report
   supersedes it for today.

## c) NOT STARTED (planned this session, deliberately deferred behind the executed items)

ClaimMetrics + scheduling/sqlstore integration vs live PG (I ran only `PG_MODULES="queue/postgres"`);
ADR-0140 vector distance-semantics draft; vector verification gaps (a)(b) (irohengine passthrough,
system tests + quickstart); skip-vs-fail classifier spread to pg/mysql helpers; error-taxonomy
drift-gate extension beyond its 5 modules; cqrs-upgrade residuals (NoPins deprecation scan, E2E
`run()` fixture test); pre-commit hook hardening batch (a)–(d); `.golangci.yml` corruption
root-cause + depguard auto-restore; recipes-gate CI posture grep; nightly gate cron workflow;
self-lint false-green fix (example/* as consumers); ResetProjection stall repro under current
load (load 30–40 matched the stall's regime, but the suite storm was not attempted — time went
to the red-class fixes); benchkit SDK polish batch; AggregateOn QueryDecl one-pager; watermill
skill tail (cross-links, upstream verify, NATS leg); md-go-validator real gate; turso defect-A
onset characterization; turso/badger contention-retry review; ephemeral-script passthrough
unification; T23 skill-maintenance pass; Go 1.27 availability check.

## d) TOTALLY FUCKED UP (own goals, no varnish)

1. **Paraphrase-memory edits, AGAIN — the documented repeat offense.** Three consecutive
   `edit` rejections (store.go, ci.yml, benchmarks.yml) because I composed `old_string` from
   my reading summary instead of extracting exact bytes first; the tool's read-before-edit
   gate caught me each time. The 13:03 report (§d4) documented this exact failure class
   TWO HOURS earlier. Extract-then-edit after that: zero failures.
2. **Appended after the closing brace.** My `cat >>` for the §2.36 catalog entry landed AFTER
   the map literal's `}`, producing broken syntax. Caught it by viewing the tail before the
   build (the same "verify the raw state" rule that caught worse), fixed with a brace-merge
   edit. A structured `edit` against the final `},\n}` anchor would have been correct first try.
3. **Extended the V007 table before reading the gate that owns its contract.** The drift
   meta-test (`TestV007_TableEntriesHaveLiveMarkers`) defines exactly what a valid entry is;
   I added 4 entries, ran the test, and iterated twice on the On/OnTyped phrasing mismatch.
   Reading v007_drift_test.go FIRST would have made it one cycle.
4. **No authored commits at task boundaries — 4th consecutive session.** The daemon absorbed
   the whole wave into `chore:` commits, including the 60-file bulk tidy mixed with unrelated
   CI/doc edits. I hold the no-unprompted-commits rule but never ASKED for authorization,
   which is the actionable miss the last three self-reviews already flagged.
5. **README written before fully verified.** I drafted queue/README.md's quickstart and claims,
   THEN verified each (DedupKey, Deps gating, `Claim.ID()`, `Close()`) and corrected. Correct
   outcome, wrong order — verify-first costs the same time and removes the risk.
6. **Spent the first minutes reconciling the STALE pasted TODO against the repo** before
   reading the on-disk TODO_LIST — the 13:03/13:05 reports at the top of the tree already
   announced the 7th-pass rebuild. Newest-artifact-first would have saved a detour.

## e) WHAT WE SHOULD IMPROVE

1. **Extract-then-edit, mechanically**: byte-exact `old_string` from the file, never from
   memory — even for files read minutes earlier (three tool-gate saves today).
2. **Read the owning gate before extending the gated artifact** (V007 drift test, recipes
   catalog spec) — the gate IS the contract.
3. **Commit per task when authorized** — ask at phase boundaries; the daemon's heuristic
   commits destroy the wave's history shape.
4. **Environment-gated items need an explicit gate-check line in the plan** (quiet window,
   billing, root) so they're skipped with evidence, not silently.
5. **The stale-TODO class keeps costing**: three rows this session were already done or
   stale (reset recipe, Set, graph). The docs-health VERIFY pass is working; execution
   sessions should still re-verify row truth before starting (today's cost: ~10 min).

## f) NEXT (up to 50, sorted by impact)

~~1. Update TODO_LIST with this wave's 8 closable rows (b2) + CHANGELOG `[Unreleased]` entries (b3).~~ done 2026-09-16 — CHANGELOG entries
2. Quiet-window exclusive `nix run .#verify` composed GREEN (the standing 🔥 row; unblocks W4 close + release train).
3. Push + observe the two fixed CI jobs (`go-work-sync`, benchmarks matview-gate) once billing allows.
4. Fix GitHub Actions billing (user action; blocks ALL paid CI verification).
~~5. Run scheduling/sqlstore + storage ClaimMetrics integration vs live PG (`PG_MODULES="scheduling/sqlstore storage"`).~~ done 2026-09-16 — PG half (TODO_LIST row)
~~6. ResetProjection stall repro under the current load regime (system suite `-parallel` + soakers; the exact load band 30–52 was present all session — a missed window).~~ done 2026-09-19 — ADR-0143 root cause
~~7. Extend error-taxonomy drift gate beyond its 5 modules (watermill, pebble, core event/command/query, view, stack, deriver, storage-facade + pool-size floors).~~ done 2026-09-16 — TODO_LIST [x]
~~8. Root-cause `.golangci.yml` config-corruption loop + add depguard auto-restore to `check-lint-config`.~~ done 2026-09-18 — TODO_LIST [x]
~~9. Pre-commit hook hardening batch (one canonical hook, `.githooks/` tracked, `.golangci.yml` staged trigger, scoped fmt gate).~~ done 2026-09-18 — TODO_LIST [x] + CHANGELOG
~~10. Self-lint false-green fix (`example/*` as consumers + analyzed-assert file counts).~~ done 2026-09-18 — TODO_LIST [x]
11. cqrs-upgrade residuals: NoPins deprecation scan + E2E `run()` fixture test.
~~12. ADR-0140: vector distance-semantics contract (semantics already pinned in AGENTS #26).~~ done 2026-09-16 — TODO_LIST [x]
~~13. Vector gaps (a)(b): verify irohengine vector passthrough + CHANGELOG enumeration; system tests + metaengine-quickstart run.~~ done 2026-09-16 — TODO_LIST [x] a–h
14. AggregateOn(fn, column, group) on QueryDecl — the planner-readable declarative seam (design one-pager).
~~15. Skip-vs-fail classifier spread to pg/mysql live helpers.~~ done 2026-09-16 — TODO_LIST [x] + CHANGELOG
16. Turso defect-A onset-boundary characterization (rows × groups × tx bisect) for the upstream draft.
17. Decide + implement grouped-matview mechanical guard (blocked on upstream timeline, but the flag design can land).
~~18. Nightly gate cron workflow (check-lint-config + modsums + script harnesses) + calibration-baseline artifact loop.~~ done 2026-09-18 — nightly-gates.yml
19. md-go-validator real gate (`--init` config + baseline + flake app + P2 skip-validate sweep).
~~20. benchkit SDK polish batch (per-metric MIN, LoadAvg1 drift report, zero-value audit).~~ done 2026-09-19 — TODO_LIST [x]
~~21. benchkit compare + serialization tail (noisy-metric column, markdown variation footer, manifest runs[]).~~ done 2026-09-19 — TODO_LIST [x]
22. Watermill skill tail: cross-links from SKILL.md/advanced.md, upstream plugin verify, 3 trigger-eval prompts.
23. NATS JetStream roundtrip leg + `#integration-nats` flake app.
~~24. queue/mysql engine behind the shared conformance suite (T17; consumer pull exists via PapDashboard).~~ done 2026-09-19 — M4, live MariaDB
~~25. Queue T14: DAG dep-gating additions (enqueue validation, cycle rejection, unblock-bump).~~ done 2026-09-19 — M4
~~26. Queue T15: owner-bearing claims + claim-token ADR-0134.~~ done 2026-09-19 — M4
~~27. Queue T16: FactSink-in-tx + watermark API completion.~~ done 2026-09-19 — M4
28. RenewLease ownership/claim tokens (scheduling/sqlstore; design-gated comment in code).
29. Backport contention-retry review to turso/badger engines.
30. Unify ephemeral-script passthrough conventions (pg positional / dgraph TEST_ARGS / redis raw).
31. Shuffle eval + adoption for test-integration.sh/test-all-backends.sh (gated on ROADMAP OQ #9).
~~32. T23 skill-maintenance pass (read-prior-reports + copy-template steps in review skills).~~ done 2026-09-19 — 94 modules go 1.27.1 + jsonv2 graduation
~~33. Go 1.27 upgrade wave — own session: nixpkgs go_1_27 availability, 91 go.directives, flake pin, CI, doc command chains, verify + bench sweep.~~ done 2026-09-18 — TODO_LIST [x]
34. Pin the recipes-gate CI posture (grep cmd/doc-check in testModules; decide #verify cold-cost stance).
35. V007 marker-driven detection decision (curated table now in lockstep vs auto-derive).
36. Consolidate indirect dep references after the next tag wave publishes (~49 consumer go.mods).
37. Create GitHub Releases for outstanding tags (script exists; owner-gated external action).
38. Tag cmd/cqrs-lint v4.10.2 (buildinfo reporting) in the next authorized wave.
39. Tag claiming + queue + queue/sqlite + queue/postgres v4.0.0 (dry-runs READY) in the next wave.
40. Next v4 tag wave: strips sibling replaces (storage go.mod, matview family, watermill v4.7.0 typed causation…).
41. Calibration quiet-window re-runs: SearchQuery count=5 + benchmark-baseline re-pin + dgraph constant campaign.
42. Green MySQL-VM shuffled suite in the quiet window (replay build/shuffle-seeds.log seeds).
43. Watch dgraph+redis CI jobs (~10 shuffled runs) for rare-ordering failures.
~~44. dgraph "Transaction has been aborted" flake investigation (failing on CI since 09-15; retryOnContention gap?).~~ done 2026-09-16 — TODO_LIST [x]
~~45. Dgraph version-floor decision: feature-detect/lazy vector schema vs documented v24+ floor.~~ done 2026-09-16 — TODO_LIST [x]
46. Matview routing v1: scalar-covered shapes priced O(1) after the AggregateOn seam (#14).
47. cqrs-lint loose-heuristic gates batch (V001/V004/V005 import-scope, B018, A015-A019 FPs — needs FP analysis per rule).
48. 350-line split waves after owner policy ratification (store.go 944 is the largest growth-gated file; typed_reader 1127 tops the split queue).
49. Storage/eventstore pin health evidence (companion to #12's green pin-sweep).
50. Write the Pareto plan HTML the skill mandates (today superseded by this report; next planning pass should emit it).

## g) QUESTIONS (cannot answer myself)

1. **Commit authorization**: may I make authored, per-task commits for execution-wave work
   (like today's), or should every wave keep landing via the auto-commit daemon? Four
   consecutive self-reviews flag this; the fix is one sentence of policy.
2. **queue/mysql engine**: strike-and-wait (what I did) or invest the M-effort now? PapDashboard
   and go-taskqueue are the named consumers — if either needs MySQL today, I'll build it behind
   the conformance suite next session; if not, the struck doc mention is the honest state.
3. **CI push policy**: the two workflow fixes are proven only locally. Do you want workflow-only
   commits pushed immediately to validate on real runners (billing still gates the nix jobs,
   but both fixed jobs are now nix-free), or do pushes ride your normal cadence?
