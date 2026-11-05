# Session Status — Resume, Watcher Armed, Full Pre-Release Prep (2026-11-05 19:08 CET)

Session: resumed 18:14 from the 18:10 park (`docs/status/2026-11-05_18-10_sse-root-cause-ci-triage-verify-interrupted.md`) under the standing "GET SHIT DONE" directive. This report covers 18:14–19:08. **Parked again awaiting instructions** — the critical path (verify attempt 13) is armed but CANNOT launch while the host is saturated (see d/e/g).

## a) FULLY DONE (this session, verified)

1. **Todo reconciliation** — 18 milestone items recreated in the `todos` tool (M04–M26 + hygiene), matching the parked plan state exactly.
2. **State check** — tree clean at resume except the park report (daemon absorbed it: commits `3e57c117e`, `5dc057aed`); `go.work.sum` +1 residue also absorbed; no verify was running; drafts (`build/adr-0164-draft.md`, `build/m23-draft.md`) intact in-repo.
3. **Verify attempt 13 launcher — DETACHED and SIGINT-proof**: `build/verify-launch-13.sh` launched via `setsid nohup` (own session, PID 3359042; survived independent of this crush process). Logic: poll `/proc/loadavg` every 10s until load1+load5 < 15.5 (margin under the `VERIFY_MAX_LOAD=16` guard), then `VERIFY_MAX_LOAD=16 nix run .#verify > build/verify-run13.log 2>&1`, writes `build/verify-run13.rc`. Bounded at 3h (expires ~21:15, rc=12). Status at 19:08: **still waiting — window never opened** (log: iter 0→300, sums 114–1208).
4. **Blocking monitor armed** — background job blocks on `build/verify-run13.rc` appearing; a completion or give-up signal is one `job_output` away.
5. **Interference source identified**: the host is running **10+ concurrent crush sessions** (pts/11, 12, 16, 18=me, 19, 21 + at least 5 more below top-12; 7,404 threads at 19:08). Load sum trajectory this session: 150→332→1208 (load1 862 on 32 cores). Two collide with my queued work directly: one session lives in **cqrs-htmx** (M11 target), one is running **buildflow --fix** with uncommitted changes in **BuildFlow** (M15 target).
6. **#114 design FULLY converged and pre-written** (read-only, tree-safe): studied cap.go, query_config.go, planned.go, temporal.go (versioning flag), extract_helpers.go (name-or-tag matching), buildKeyExtractor, notifyLive/notifyWatchers, sqliteengine encodeKey/encodeValue + meta_map/planned DDL + xd() accessor, QuoteIdent, adttest patterns (vector_dimension.go, cap_test.go, Factory wiring), capabilities-gate probe mechanics. Final design decisions:
   - `WithCapPerGroup(groupCol, perGroupRows)` REQUIRES a WithCap declaration; caps COMPOSE (grouped pass first, then the global pass inside `EnforceCap`; `EnforceCapGrouped` standalone; `EnforceCaps` dispatches on config).
   - `CapGroupTrimmer` capability returns the DELETED KEYS (`DELETE … RETURNING key`) so the Store notifies live SSE subscribers; engines that must not bulk-delete (cell versioning on) return `ErrCapGroupPushdownUnsupported` → Store falls back to its portable Go path.
   - Go fallback: ONE orderBy-DESC page scan + per-group counters (newest-first scan makes per-group keep-newest-N trivial); deletes ride the existing `capDelete` (temporal-correct, per-row notify).
   - Group extractor matches result-type fields by json-tag-or-name case-insensitively — the SAME matching `ExtractFields` uses, so Go grouping and SQL `json_extract` agree.
   - Plain-identifier validation for groupCol/orderBy at declaration time AND engine-side (they flow into SQL text).
   - Planned tables prefer the REAL extracted column, else `json_extract(value,'$.col')` (both table shapes carry the JSON document).
   - **Deadlock caught in design review**: `enforceGroupedTrim` must not take the fold lock when called from `EnforceCap` (already held) — lock moved to the public entry points.
   - adttest math hand-verified (WithCap(8)+PerGroup(2): seed a:6,b:1,c:1 → removed 4 → a{5,6},b{1},c{1}; b-flood → b{4,5}).
7. **Scratch landing kit written** to `build/_114-scratch/` (underscore prefix = invisible to go tooling; verified `go list ./...` ignores it): `adr-0164.md` (final text), `cap_grouped.go` (metaengine core, complete), `sqlite_cap_grouped.go` (pushdown, complete), `adttest_grouped_cap.go` (contract pin, complete), `landing-edits.md` (exact cap.go/query_config.go diffs + gate checklist).
8. **M07 wave versions DERIVED from evidence** (CHANGELOG [Unreleased] + git diff of tag..HEAD): catalog **v4.13.0**, kv **v4.4.0**, id **v4.7.5**, metaengine **v4.21.0** (draft said v4.20.1 — WRONG: `BuildLayoutPlanFromType`/`BuildLayoutPlanForType`/`BuildColumnarLayoutPlan` are new exports since v4.20.0 ⇒ MINOR), sqliteengine **v4.7.0**, cqrs-lint **v4.21.0**. Tag naming: basename + FULL path for `metaengine/sqliteengine/vX` and `cmd/cqrs-lint/vX` (versions.json convention). All 91 metaengine commits are daemon `chore:` blobs — semver from API diff, not messages.
9. **M12 scope CORRECTED**: crm's `.cqrs-lint-baseline.json` is ALREADY EMPTY (`entries: []` — the 11-03 baseline-doctrine session); the plan's "drop 7 waiver-dup baseline rows" is MOOT (errata row for M22). Remaining M12: `scorecard.modules` projectionhost pin (feature verified present: `analyzer.ApplyModulePins`, evidence marker `scorecard.modules`, false-pin rejection) + cqrs-lint pin bump. **crm == "Kith CRM"** (nix package `kith-crm`; the read-storm crm-server binary) — M12+M14 are the same repo, different concerns.
10. **M13 targets located**: webphone pins at catalog v4.12.0 / metaengine v4.20.0 / sqliteengine v4.6.0; `internal/server/apidocs.go` carries hand-rolled `retryAfterClause` text (→ `catalog.WithResponseHeader`) and multipart prose (→ `WithRequestBodyContent`); suppression at `internal/server/wirecontracts.go:130` (`asyncapi.WithServer("webphone","","")`); pagination sites → `WithTypedParam`.
11. **M24 targets located precisely**: warn de-explosion = the two exploded `warn(...)` calls in `cmd/cqrs-lint/baseline_rematch.go` (68–76, 89–98); hand-map split brain = `waiverAdoptionRules` in `cmd/cqrs-lint/waiver_adoption.go` (7 waiver→F-rule pairs duplicating rule-catalog knowledge).
12. **M26 design ready**: `metaengine/sqliteengine/dsl.go:106` `NewSQLiteEngineFromDSNWith` applies defaults (journal_mode=WAL, busy_timeout=5000, cache_size=-65536) BEFORE caller pragmas — golden test pins caller-override-wins on the owned handle (#111 last-write-wins contract).
13. **M21/M23 prep**: lint app loop structure read (GT_LINT_MODULES scoped variant shape); m23-draft.md re-validated (README subsection target after line ~253, fence-ratchet classification, enginetest doc note).
14. **Mirror-lockstep N/A-verified**: `mirrorPairs` covers only the folded trains (command/dedup/dispatcher/event/id/kv/metadata/query/record) — #114's metaengine/sqliteengine exports need NO core/v5 mirror rows (§f item 44 was caution, not obligation).
15. **M22 harvest sources fully collected** (§f of 06:39 + 09:46 + 18:10 reports); todos updated folding in the newly-found obligations: package-surface after #114, godoclint on new symbols, check-release-scripts if batch-release touched, ignore-cascade tripwire, generated-artifact pre-push sweep.

## b) PARTIALLY DONE

- **M04 (verify attempt 13)**: launcher armed and healthy but the quiet window NEVER opened (load sums 114–1208 over 54 min vs threshold <15.5). Watcher expires ~21:15 (rc=12). Prior evidence stands: 88 packages were green in attempt 12 before the external SIGINT; all known blockers fixed.
- **M05–M07**: fully specified and version-derived; hard-blocked on M04 green (receipt-first chain: verify → push → CI → tag).
- **M11–M15**: all targets located and read; blocked on M07 tags (pin bumps) AND session-collision status (see g2).
- **M16–M19**: design + scratch code complete (would land in ~1–2h of tree time); blocked on the M04→M07 chain by choice — piling #114 into the release push would hold the wave hostage to a feature bug and invalidate the derived versions.
- **M21–M26**: all specified at implementation-ready detail; blocked on tree access (verify exclusivity — no tree edits while the watcher can fire any moment).

## c) NOT STARTED (blocked, by design)

- M04 completion → M05 CI receipt (push + watch MySQL/Dgraph/benchmarks legs) → M06 (changed-set tests over 12 modules + fleet census) → M07 (batch-release, manifest, tag push, go-get smoke) — the entire receipt-first chain.
- #114 landing (M16–M19), M21, M23, M24, M26 tree edits.
- M22 final harvest + errata; improvement rows; final full verify; completion report superseding 06:39/09:46/18:10.

## d) TOTALLY FUCKED UP (honest ledger)

1. **The quiet-window strategy is starving under today's host reality.** I armed a <15.5 gate calibrated to the documented 8–45 service floor — but the box now runs 10+ concurrent agent sessions (load sum peaked 1208 on 32 cores). Result: 54 minutes of armed waiting, zero verify progress, and a 21:15 give-up deadline. The threshold was not wrong for the documented floor; the floor's ASSUMPTION is wrong for today. This is the single thing that cost the session its critical-path progress, and I chose waiting over escalating to the owner (the parked g2 question said "fallback: justified ceiling per owner answer" — the owner had not answered, so I waited; with hindsight I should have surfaced the starvation explicitly at ~18:45 when sum first exceeded 300 sustained).
2. **Nothing else broke** — no tree edits, no test runs, no load contributed by me beyond reads; the scratch-kit placement mistake (Go files in `build/114-scratch/` could poison root `go build ./...`) was caught and fixed within minutes (renamed `build/_114-scratch/`, verified invisible to `go list`).
3. Carried over (not mine, still open): the three 18:10 questions (g1 interference policy, g2 load-burn policy, g3 MySQL QEMU recurrence bar) remain UNANSWERED — g2 now subsumed by my question 1 below; g3 unanswerable until the post-push CI run exists.

## e) WHAT WE SHOULD IMPROVE (structural)

1. **Load policy must be re-ruled for multi-session reality** (see g1): a fixed <15.5 gate under a 10-session host is a starvation trap; the honest options are a raised justified ceiling with triage duty, or scheduled exclusive windows.
2. **`verify-launch-13.sh` needs a deadline disposition**: today it gives up (rc=12). It should either launch-with-burn at deadline or auto-requeue, per the owner's g1 answer.
3. **Cross-session repo locking is missing**: two of my five consumer targets are being touched by other live sessions RIGHT NOW (cqrs-htmx session active; BuildFlow tree dirty from a live buildflow run). A per-repo advisory lock (e.g., `.session-lock` with PID+timestamp) would prevent double-agent edits — cheap insurance versus a clobbered tree.
4. **Verify-launcher deserves promotion** to `scripts/` as the standing improvement row already notes (poll-interval/margin flags, deadline mode, rc file convention) — this session proved the setsid-detach + repo-side-log + rc-file pattern works; formalize it once, reuse forever.
5. **Scratch discipline**: pre-writing complete landing kits in `build/_<name>/` (underscore-invisible) converted ~2h of blocked wait into near-zero-cost landing time — adopt as the standard pattern for tree-blocked windows.
6. **The 8–45 floor documented in the park context is stale** — today's honest floor is "8–45 + N concurrent sessions × their compile storms"; any future gate calibration should measure, not assume (docs/agents/gotchas-tooling-build.md candidate row).

## f) NEXT (ordered, up to 50 — resumable from this exact state)

**Critical path (M04→M07):**
1. Resolve the load-policy question (g1) → either the window opens (watcher fires) or execute the owner's chosen fallback.
2. Watch verify attempt 13 to completion (`build/verify-run13.rc`, log `build/verify-run13.log`); triage any finding by class (known flake classes all fixed; timing-under-load → rerun specific test at a quiet moment).
3. If watcher expires rc=12 first: relaunch per owner ruling (immediate burn / overnight / next quiet hour).
4. On green: `git push origin master` (daemon commits are local; CI cancel-in-progress supersedes run 41978263864).
5. M05 receipt: poll the new run's jobs (gh api); watch MySQL leg (flake recurrence → g3 bar), Dgraph leg (tx-aborts), benchmarks duration (2h+ legs), actionlint, verify-fast, flaky reruns.
6. M06: changed-set `GOWORK=off go test ./... -count=1` over catalog, kv, id, metaengine, metaengine/sqliteengine, cmd/cqrs-lint, middleware, event(+eventtest), storage(+pebble/bbolt/eventstore/relational), core/v5, system, systemscenario.
7. M06: `bash scripts/census-fleet.sh` + read the zero-consumer report.
8. M07: `./scripts/batch-release.sh "catalog v4.13.0 …" "kv v4.4.0 …" "id v4.7.5 …" "metaengine v4.21.0 …" "metaengine/sqliteengine v4.7.0 …" "cmd/cqrs-lint v4.21.0 …"` (FULL-path tags for the last two).
9. M07: versions-manifest `--update` + README manifest + `check-versions-manifest --check`.
10. M07: push tags; `go get` smoke per tag from a clean dir; `nix run .#check-release-scripts` if batch-release.sh was touched.
11. M07: CHANGELOG cut (move [Unreleased] → versioned sections; changelog-symbols gate).

**Consumers (M11–M15):**
12. M11 cqrs-htmx: delete `usermgmt/view_cache.go` (~182 lines) → `kv.NewCacheForViewStore` (+`WithCacheCopyFunc` choice), kv pin → v4.4.0, suite green, hand-commit at boundary — **after confirming the other cqrs-htmx session's work** (g2).
13. M12 crm: add `scorecard.modules: {"projectionhost": true}` (+graph if wiring shows it), update the "stays unwaived" comment, bump cqrs-lint → v4.21.0, run cqrs-lint verify, commit.
14. M13 webphone: `WithResponseHeader` on 429 lanes; `WithRequestBodyContent` multipart `/api/transcribe`; `WithTypedParam` pagination; drop `WithServer("","","")`; catalog pin → v4.13.0; export diff + tests + commit.
15. M14 crm/Kith: wire `sqliteengine.WithMaxOpenConns(n)` at engine construction (pin → v4.7.0), re-measure the read storm (ps showed crm-server at 73% CPU, 435 CPU-min accumulated — the storm is live), record numbers.
16. M15 BuildFlow: `go get` cqrs-lint @ v4.21.0 (3 modules), vendorHash.nix re-pin, build+test — **after the other session's buildflow --fix run commits** (g2).

**#114 (M16–M19, scratch ready in `build/_114-scratch/`):**
17. M16: `nix run .#check-adr-numbering` (before), land `docs/adr/0164-per-group-projection-caps.md` from scratch, docs/README.md index row, numbering gate again.
18. M17: query_config.go fields+option+accessors; cap_grouped.go (validation, extractor, CapGroupTrimmer, EnforceCapGrouped, Go fallback); cap.go edits (validate dispatch + grouped pre-pass in EnforceCap) — exact diffs in `landing-edits.md`.
19. M18: sqliteengine cap_grouped.go + interface pin; memory path covered by core tests; turso delegation test; duckdb/pg TrimGroupedCap twins (ctid for pg).
20. M18: adttest/grouped_cap.go + suite wiring (sqlite meta_map + planned + versioned; turso; duckdb; pg).
21. M19: capabilities-gate TrimGroupedCap probe + `--update` regen; recipes.md fence + doc-check catalog entry; CHANGELOG (cite `metaengine.WithCapPerGroup`, `metaengine.EnforceCapGrouped`, `metaengine.CapGroupTrimmer`, `metaengine.ErrCapGroupPushdownUnsupported`); FEATURES row.
22. M19: `api-stability --update`; `check-untested-exports --update`; `check-package-surface`; godoclint on new symbols; comment+close #114 citing ADR-0164.

**Hygiene (M21–M26):**
23. M21: scoped-lint flake app reading `GT_LINT_MODULES` (default all lintModules), failure summary shape, dry-run/self-test, AGENTS quick-ref row.
24. M23: README SeqSeekableStreamLog section from `build/m23-draft.md`; readme fence-ratchet classification; enginetest seqseek.go doc note; capabilities regen no-op check.
25. M24: `baseline_rematch.go` warn de-explosion (string-concat form); `waiverAdoptionRules` generated from the rule catalog (or documented hand-map policy if generation is worse); cqrs-lint suite + README count pins if rules docs change.
26. M26: pragma-override golden test (caller cache_size/busy_timeout wins over defaults; synchronous capture still recorded).
27. M22: TODO_LIST harvest — route every un-done §f item above, tick done ones, errata rows (go-jose no-op; crm-baseline-already-empty; metaengine v4.20.1→v4.21.0 draft correction).
28. Improvement rows: verify-launcher promotion; ignore-cascade tripwire (`git ls-files` vs disk pre-push); generated-artifact pre-push freshness sweep (api golden, capabilities, taxonomy, docs-ui.css); wedged-run watchdog gotcha; `nix run .#erraudit` wrapper; CI benchmarks `timeout-minutes`; sse_test.go redundant RaceEnabled cleanup.
29. Final: full `nix run .#verify` green on the post-#114 tree (second quiet window needed — plan it per the g1 ruling).
30. Final: completion report superseding 06:39/09:46/18:10; wait for instructions.

## g) QUESTIONS (cannot figure out myself)

1. **Load policy under multi-session reality (supersedes parked g2):** the host ran 10+ concurrent agent sessions today (load sum 114→1208 on 32 cores; my verify gate needs <15.5). Waiting has already cost ~1h of critical-path progress and the 3h watcher expires ~21:15. Choose: (a) keep waiting for a natural window (possibly overnight), (b) justified ceiling burn now (e.g. `VERIFY_MAX_LOAD=45–60`, accepting that timing-sensitive legs may false-red and need rerun triage), or (c) a scheduled exclusive window you reserve (tell me when the other sessions sleep). I cannot know the other sessions' schedules or your risk preference for a false-red verify burn.
2. **Session-collision policy (extends parked g1):** a live session is working in cqrs-htmx (my M11 target) and BuildFlow carries another session's uncommitted buildflow run (my M15 target). Proceed there anyway (risk conflicting edits/lost work), skip-and-note for later, or wait for those sessions to finish? I cannot see their intent or completion times from here.
3. **Unattended overnight verify:** if the window only opens overnight, should the watcher-launched verify run unattended (a red attempt 14 at 03:00 blocks nothing until triage, but the daemon may absorb unrelated drift into the same state), or should verify only run in attended hours? This is a judgment call about burn-vs-staffing I can't make for you.

— Parked at 19:08 CET. Watcher (PID 3359042) and monitor job (04C) remain armed; `build/verify-run13.rc` is the signal file. All scratch reusable: `build/_114-scratch/`.
