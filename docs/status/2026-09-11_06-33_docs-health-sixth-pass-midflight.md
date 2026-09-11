# Status Report — Docs-Health 6th Pass (Mid-Flight Interrupt): Living Docs Rebuilt, Annotation Sweep Half-Done

> **When:** 2026-09-11 05:55–06:33 CEST (~38 min) · **Mandate:** "View ALL `**/2026-0*` files! Execute the docs-health SKILL! PROPERLY! FUCKING SUPERBLY!!! TODO_LIST/CHANGELOG/AGENTS/README/ROADMAP/FEATURES must be all SUPERB! Archive FULLY done and UPDATED (inline strikethrough) .md files!" — interrupted by this status request at ~06:31.
> **Mode:** docs-health AUDIT (BUILD + HARVEST + VERIFY + ANNOTATE + ARCHIVE) over the 26 non-archived `2026-0*` markdown targets + the six living docs.
> **Skill discipline held:** SKILL.md + 4 references loaded BEFORE acting; the skill's `annotate-rows.py` asset used with `--dry-run` first (the mandated 2026-08-18 lesson).

---

## a) FULLY DONE (each verified against code/tree at 06:33)

1. **Full inventory + read of the active set:** all 8 active `docs/status/2026-09-11_*` reports read in full; TODO_LIST (619 ln), ROADMAP (711 ln), README (222 ln), CHANGELOG `[Unreleased]` (40 sections enumerated), FEATURES (structure + every targeted row). Non-status 2026-0* files classified (research/benchmarks = LEAVE ALONE per the standing exemption; SUPERB plan = live; ADR proposals = fully resolved, archive-ready; analytics-rollup ×2 = resolved, archive-ready; turso drafts = live upstream handoffs).
2. **VERIFY with mechanical evidence (not report trust):** `find -name go.mod | wc -l` → **84** ✓; `git tag -l 'watermill*'` → **v4.7.0 still absent** (tag-wave row stays) ✓; rule count recounted from the generated `cmd/cqrs-lint/RULES.md` summary table → **206** (ROADMAP said 204 — STALE, fixed); `bash scripts/check-file-size.sh` → **GREEN** (58 baselined, no new offenders, no growth); api golden carries all engine `ResetEngine` exports (19 refs) ✓; shipped-symbols greps confirmed: turso reachability docs (README:29), `dgraph.type` conflict docs + `TestIsContentionError` pin, `TestEnsureEdgeSchema_InsideRunInTx`, `doMutate` narrowing, bench modernization (10× `b.Loop` + `atomic.Uint64`), and ALL 8 sections of the 2026-08-17 system-v4 ADR proposals addressed (`ExecuteTypedByName`, `PublisherFor`/`PublisherAt`, role wiring, reserved-config honesty comments, `system/durability.go`, crash-window docs, 09-08 re-tag, dep-diet verdict).
3. **TODO_LIST truth pass (S01-final):** **10 done/stale rows deleted** (2 `[x]` rows per the no-completed-items policy, turso-encryption-breadth [all 5 sub-items verified shipped], dgraph.type docs + isContentionError pin, contention-retry observability [= WithContentionObserver], skip-vs-fail [honest-loud shipped], ensureEdgeSchema in-tx pin, doWrite review, bench modernization, check-coverage wrapper fix); **3 rows updated** (350-line → ratchet-shipped/ratification-pending state; matview routing → S28 `AggregateOn` seam + scalar-first v1; contention-fix verify → the quiet-window S03 composed-GREEN item incl. `-race` + verify-docs e2e); **~20 rows HARVESTED in** (each with source citations, deduped against existing rows): 🔥 CatchUpEngine snapshot race + stress test, catch-up observability + high-water marks + ResetResult return, Doctor per-entry-point synthetic-feed counters, legacy-log synthesis pin, conformance tail (ApplyIdempotent dedup + applyFold micro-bench + live PG/MySQL ClaimMetrics runs), recipes `ApplyEncodedRecord` snippet, calibration gate v2 (load1+load5), scheduling/sqlstore hardening tail (9 items consolidated), cqrs-lint cheap-fix tail (6 items), error-taxonomy gate extension (7 modules), cqrs-upgrade strict-gate holes (5), self-lint false-green root fix, private-dep mechanical guard + visibility audit + policy, release-tooling follow-ups (6), taskmanager tail (3), turso IVM session tail (5), skip-vs-fail classifier spread, projectionhost `-tags integration` vet, cec9248da work record, skill reset-recipe all-engines update, status-README index upkeep. Result: **0 `[x]` rows, 90 open rows, ~745 lines.**
4. **ROADMAP:** 204→**206** ×2 (mechanical recount); **OQ 10 struck** (skip-vs-fail ANSWERED: honest-loud, implemented); **OQ 8 updated** (`WithContentionObserver` shipped — knobs question remains); **Release History `[Unreleased]` row refreshed** with the full 2026-09-11 second wave (reset ladder 12/12, fold failover + CatchUpEngine, ApplyEncodedRecord + sweep, observer, file: DSN fix, ivm repro + citation gate, taxonomy gate, cqrs-upgrade hardening, C040 parity, goleak, tripwire, release tooling, ratchet, coverage fix); Theme 1 short-term remainder updated; **Raw Ideas write-back DONE (13 new items)** — the S12 "dropped brainstorm fuel" the 5th pass banner-noted but never wrote (quarantine-aware replan, CatchUpEngine-into-new-engine, EventLog bounds, span-name gate, cqrs-upgrade CI dogfood, codec-defaults gate, canonical-fact gate pattern, golden-profile harness, examples-out-of-workspace, public/private helper-repo contradiction, claims-ledger convention).
5. **FEATURES:** reset row rewritten (ALL 12 engines + `CanReset` + Doctor `--- Reset ---`); ADR-0137 row updated (reads AND folds reroute; catch-up; stale-read caveat); **2 new rows** (fold-write failover + catch-up — with the known snapshot-race fast-follow noted; encoded-apply record context + 13-entry-point sweep); dgraphengine row extended (contention-hardened + observer + ResetEngine); Turso "Local DB" row documents the `file:` DSN normalization. The file-size ratchet was deliberately NOT added (contributor tooling — wrong owner file per doc-ownership).
6. **CHANGELOG:** **2 missing `[Unreleased]` entries added** (file-size baseline+ratchet gate; coverage vacuous-0.0%-drift fix) — both shipped by the 05:51 session and previously unlogged.
7. **AGENTS.md:** contract #1 rewritten to the actual ratchet mechanism (was "(CI-enforced)" — decorative since 08-08); contract #22 wording precision fix (journal positions monotonic vs per-collection seq caches like sqlite's `multiSeq` restart — the 05-40 §d4 overgeneralization).
8. **Two on-sight fixes:** `docs/error-taxonomy.md` watermill paragraph reworded ("shuts the replay down silently" → Debug shutdown log, 2026-09-11 semantics — 05-26 §f27); `gotchas-tooling-build.md` gained the `go build -o` binary-pollution gotcha (05-34 §f37).
9. **ANNOTATE (2 of 8 reports):** 04-35 fifth-pass report — **25 §f rows struck inline** via the skill's `annotate-rows.py` (dry-run first; shape-verified read-back) + RESOLVED-BY-ROUTING banner; 05-12 report — 4 rows struck (ratchet, RULES.md regen, reset ladder, harvest).
10. **Gate truth established for annotation verdicts:** `nix run .#check-duplication` run to completion — **RED: 5 new clone groups (baseline 54)**, the reset-wave `*engine/reset*.go` clones still unannotated/unbaselined (see d/e) — so 05-21 §f1 and 05-40 §f47 stay OPEN, not struck.

## b) PARTIALLY DONE

1. **Annotation sweep: 6 of 8 reports remain** (05-21, 05-26, 05-34, 05-38, 05-40, 05-51) — per-row verdict lists are fully prepared (which rows ship-strike vs routed-open), commands not yet executed.
2. **05-12 banner written but not applied** — the edit hit the mtime guard (my own `annotate-rows.py` write had bumped it after my last read); banner text ready.
3. **ARCHIVE: nothing moved yet.** Planned and verified: 8 status reports → `docs/status/archived/`; ADR proposals (all 8 sections resolved) → `docs/adr/archived/` (dir to create); analytics-rollup ×2 → `docs/feedback/reviewed/archived/`. SUPERB plan = stays live, gets a progress banner + ✅/◐ row markers (verdicts derived from the two execution reports); book-insights gets one inline correction (SQL idempotency store shipped).
4. **docs/status/README.md index** — not yet updated for the 2026-09-11 batch (harvested as a standing TODO row too).
5. **Final gates not yet run post-edits:** `check-doc-links`, `check-changelog-symbols`, `cmd/doc-check`, `nix fmt`, re-run `check-file-size`. (Pre-edit baselines were green.)
6. **Health report (Accuracy + Fitness with visible math)** — findings table is mentally compiled; not yet delivered.

## c) NOT STARTED

1. ADR-proposals annotation (verification DONE — see a2; edits not made).
2. SUPERB-plan progress annotation (S05–S07, S10–S11, S13–S14, S17, S20–S21, S23–S25 = done; S01/S12/S15/S16/S18/S19/S22 = half; S02–S04, S08-redo, S09, S26–S30 = open).
3. book-insights inline correction.
4. SKILL.md `references/` reset-recipe all-engines update (deliberately routed to TODO_LIST instead — code-adjacent skill edit, not a living-doc fix).

## d) TOTALLY FUCKED UP (this session's own failures, no varnish)

1. **Three edit-tool failures from reconstructed (not freshly-viewed) old_strings in TODO_LIST:** (i) a multiedit corrupted a row header into a blockquote merge — repaired; (ii) an over-wide deletion span silently swallowed TWO still-open rows (`go mod tidy integration/`, `unify ephemeral passthrough`) — caught by follow-up grep and restored; (iii) the tool's "Applied 1 of 2 (1 failed)" messages twice misdescribed reality — one "failed" edit HAD applied, leaving a `PLACEHOLDER_NEVER_MATCHES` token I only found because I grepped for it. Root cause: batching long-span deletions from memory + trusting partial-failure reporting. Policy going forward: one row per edit, fresh View immediately before, grep-verify after every batch.
2. **Self-inflicted mtime-guard rejections ×2:** my own `annotate-rows.py` write bumps the mtime, so the immediately-following `edit` banner call is refused (CHANGELOG once — daemon's absorb; 05-12 once — my own script). Correct safety behavior; my sequencing was wrong. Rule: re-View after ANY script write to a file I'm also editing.
3. **The check-duplication background job's verdict was truncated to its footer** in the first read (pipe captured only the art-dupl help lines); I re-ran it synchronously rather than striking rows on an assumed green — that caution is why 05-21 §f1/05-40 §f47 correctly remain open (gate is RED, see a10).

## e) WHAT WE SHOULD IMPROVE

1. **Finish the sweep in 6 single commands** — the per-row verdicts are prepared; `annotate-rows.py` proved exact on this table shape (25/25 + 4/4 rows, shape-verified).
2. **The mtime dance is mechanical now:** script-write → re-View → edit-tool banner. Or: banners FIRST, annotations second.
3. **The RESOLVED-BY-ROUTING banner pattern works** — it states what shipped, what was harvested where, what stays user-gated; keep it as the house banner.
4. **`#check-duplication` RED (5 groups) blocks every clean `#verify`/tag-wave claim** — the reset-wave `reset*.go` clones need `//art-dupl:accept` directives (per AGENTS §14: annotate, don't re-pin) or a deliberate baseline re-pin on a committed tree. This is now the top Quality item (f1).
5. Edit discipline rule worth writing into gotchas: "the edit tool's partial-failure reporting is not trustworthy for multiedit — verify with grep after every batch" (this session burned 3 round trips re-learning it).

## f) Up to 50 things we should get done next (impact-sorted; 1–14 finish THIS pass)

| #  | Task                                                                                                                                                 | Effort |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1  | Annotate remaining 6 status reports (verdicts prepared) + apply the 05-12 banner                                                                     | S      |
| 2  | ARCHIVE: 8 reports → `status/archived/`; ADR proposals → `docs/adr/archived/` (create); analytics ×2 → `reviewed/archived/` (`git mv`)                | S      |
| 3  | SUPERB-plan progress banner + inline ✅/◐ row markers (keep live — S26–S30 + gated items remain)                                                      | S      |
| 4  | book-insights inline correction (SQL idempotency store shipped) + `docs/status/README.md` index update for the batch day                              | XS     |
| 5  | Final gates: `check-doc-links`, `check-changelog-symbols`, `cmd/doc-check`, `nix fmt`, `check-file-size` — then the inline Accuracy+Fitness report    | S      |
| 6  | 🔥 Resolve the RED `#check-duplication` (5 reset-wave clone groups): `//art-dupl:accept` per group (iterative until 0 new) or baseline re-pin on committed tree | S/M |
| 7  | 🔥 CatchUpEngine snapshot race fix + concurrent stress test (TODO_LIST row; pre-tag-wave)                                                             | M      |
| 8  | S08 REDO: verify-ci go.sum probe (edit tool, exact context) + `TestEveryModuleSumComplete` + mutation proof                                          | M      |
| 9  | S09: `lint-module` tag arg + CI leg for `*_integration_test.go` modules                                                                              | S/M    |
| 10 | Quiet-window exclusive `#verify` composed GREEN + `-race` metaengine + `verify-docs.sh` e2e (record date/commit/durations)                            | M      |
| 11 | MySQL live reset run (VM/nspawn) + dgraph `-race` seeds 42/7/1234 + composite shuffle evals + seed-persistence log                                   | S      |
| 12 | 🔥 S02 tag wave (runbook: dependency-ordered manifest, smoke probes, go-localsync `watermill/v4.7.0` unblock check) — user authorization pending     | M      |
| 13 | S26 sweep §4 remainder (watermill keys dual-read, SQL columns, benchkit key, bbolt tags, v6 markers, T18 tail, V5-MIGRATION-GUIDE)                   | L      |
| 14 | S28 `AggregateOn(fn, column, group)` design one-pager → routing v1 (scalar-covered O(1), grouped O(N) + Doctor note)                                  | M      |
| 15 | Error-taxonomy gate extension: watermill, pebble, event/command/query, view, stack, deriver (+ pool floor, explicit extraction asserts)              | S/M    |
| 16 | cqrs-lint cheap-fix tail: S001 corpus validation, D014/D015 registry tests, B008 Warning baseline, receiver-context message, full-module `-race`     | S each |
| 17 | cqrs-upgrade strict-gate holes (rep.Error fails strict, NoPins scanned, `bumps` always-present, `schemaVersion`, E2E fixture test)                    | S      |
| 18 | Self-lint root fix (`IsLibrarySelfLint` treats `example/*` as consumers) + analyzed-assert; V007 typed detection decision pre-v5                     | M      |
| 19 | check-turso-version `--self-test` mode; ivmrepro `-race`; `TURSO_IVM_REPRO_ROWS` clamp; release-checklist repro step; fold 3 findings into draft      | S      |
| 20 | Private-dep guard: `check-private-deps.sh` + flake app + CI leg; sibling-repo visibility audit into module-map.md; owner policy                      | S/M    |
| 21 | Release-tooling follow-ups: `--smoke-all`, batch limitation doc, `--verify` mode, CONTRIBUTING refs, `path_matches_major` lib, gate into `#verify`    | S      |
| 22 | taskmanager tail: must.go unit tests, what-runs-in-0.080s check, module-map notes                                                                    | S      |
| 23 | Calibration gate v2 (load1 AND load5) + shellcheck; quiet-window count=5 SearchQuery re-run + titled baseline re-pin + dgraph re-anchor (one window) | S/M    |
| 24 | Doctor: per-entry-point synthetic-feed counters (runtime visibility the sweep only gives in tests)                                                    | M      |
| 25 | Legacy-log-entry synthesis pin (one legacy `EventLog.Record()` case in the conformance sweep)                                                        | S      |
| 26 | Conformance tail: ApplyIdempotent dedup case, applyFold micro-bench, live PG/MySQL ClaimMetrics runs                                                 | S/M    |
| 27 | Catch-up observability: Doctor/GetEngineStats state + per-engine high-water marks + ResetResult return                                               | M      |
| 28 | 🔥 CI triage wave 2 (S04): verify-fast, go.work sync, Nix Flake Check, CGo, Security Scan, Minimum Coverage, ephemeral FlakeHub bisect, matview gate dry-run | M-L |
| 29 | S19 tail: per-finding lint attribution, `aggregate_*` permanent mutation fixture, pre-commit gates remainder                                          | S      |
| 30 | skill references: reset recipe all-engines + `WithContentionObserver` recipe + `ApplyEncodedRecord` recipes.md snippet                                | S      |
| 31 | cec9248da orphaned-work record (short annotated report)                                                                                              | XS     |
| 32 | Scheduling/sqlstore hardening tail (race-stress, counter-scope pin, property, fuzz, examples, RenewLease, process-start ts) — one slice per PR       | M      |
| 33 | fold-reroute test with a Transactional engine; reroute-cost caching if bench-justified; `routedQuery` comment                                         | S      |
| 34 | taskmaster golden policy (must.go panics + V006 pin) — confirm whether the 05-34 fix closed it; triage D013/E003/S010/C023/C026                       | S      |
| 35 | #verify-fast membership check (exists? includes taxonomy gate?) + probe-negative rule into gotchas-testing                                            | XS     |
| 36 | Annotate archived 02-06 report's false-green "examples scan green" claim (non-destructive correction)                                                | XS     |
| 37 | Repo-health code items noticed in passing: `graph.sink.unknown_node_label` relocation, middleware deadletter prefix (v5), quickstart lint findings ×5, stale compiled binary in example dir | S |
| 38 | S29 program tail: loose cqrs-lint gates one-per-PR, calibration-drift redesign, ephemeral passthrough unification, T23 skill pass                     | L      |
| 39 | S27 v5 train phase B (deletions + `NewStreamRef` validation + E-items) then cut v5.0.0 — owner-gated                                                 | L      |
| 40 | S30 decision bundle (turso filing + 3 driver issues, DSN policy, sync/embedded, dgraph one-RPC, CapabilityGaps→Doctor, Doctor-JSON, Q3 severity, daemon Q2, F040, dead-path OQ 11, iroh P99, macOS PG, nspawn, CV bump, archived/ shard, 5th-pass 3 rulings) | decision |
| 41 | GitHub Releases for outstanding tags (`create-github-releases.sh` — gh auth verified)                                                                | S      |
| 42 | Indirect-dep consolidation after the next tag wave (~49 consumer go.mod files) + standing pin-sweep step                                             | M      |
| 43 | Watch tonight's sentinel + post-push CI legs (file-size/verify-ci acceptance) once pushed                                                            | XS     |
| 44 | Relational + pebble dotted/underscore code-spelling unification (v5 sweep; ~10 + ~40 pairs)                                                          | M      |
| 45 | Skip-vs-fail classifier spread to pg/mysql live helpers                                                                                              | S      |
| 46 | projectionhost `go vet -tags integration ./...` compile check (TestMain clash)                                                                       | XS     |
| 47 | `go mod tidy` in integration/ (gopls unused genproto — still flagged)                                                                                | XS     |
| 48 | ROADMAP Experimental: jsonv2 tag removal tracking (Go 1.27+); turso MVCC upstream watch                                                              | XS     |
| 49 | Nightly-dogfood + green-recency signal hygiene (job-summary table separating known-red from new-red)                                                 | S      |
| 50 | Claims-ledger / CLAIMS.md convention trial on the next multi-session execution window (ROADMAP Raw Idea)                                             | S      |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **The three 5th-pass §g rulings are still open and this pass hit all three again:** (a) archive rule — I follow the repo precedent (harvest → annotate → archive, even with open-but-routed items) over the skill's "EVERY item resolved" letter; (b) CHANGELOG path repoints after archiving (I again chose not to touch released sections — no new cases this pass); (c) the HTML/`.txt` exemption (again inventoried by title, never opened). Ratify all three as standing policy?
2. **Same three owner gates carried from the 05:51 report:** (a) is the parallel agent session yours and deliberate — keep lane-splitting or claim whole waves?; (b) authorize the S02 tag wave (consumers see nothing until tags move; go-localsync blocked on `watermill/v4.7.0`)?; (c) does the 350-line baseline+ratchet stand as POLICY (vs full split waves / harness exemptions)?
3. **`#check-duplication` is RED (5 reset-wave clone groups) with a clean tree — annotate or re-pin?** The five near-identical `*engine/reset*.go` implementations are dep-isolated by design (the `register.go` precedent: `//art-dupl:accept` per module). I can annotate them live (iterative until 0 new groups) or re-pin the baseline on a committed tree — annotate is the AGENTS §14 default, but a re-pin also captures the 5 groups in one move. Your call on which, or I annotate.

---

**Receipts:** 84 go.mods recounted · 206 rules recounted from generated RULES.md · watermill v4.6.0 latest tag (v4.7.0 absent) · `check-file-size` GREEN · `#check-duplication` RED (5 new groups, baseline 54 — verified synchronously) · api golden 19 ResetEngine refs · annotate-rows.py 25+4 rows, shape-verified · TODO_LIST 0 `[x]` / 90 open · tree clean at daemon `1fe241b54` (all edits absorbed).

_Generated 2026-09-11 06:33 CEST. Point-in-time snapshot — annotate, don't rewrite. WAITING FOR INSTRUCTIONS — annotation/archive sweep (f1–f5) resumes on "continue" or per your routing._
