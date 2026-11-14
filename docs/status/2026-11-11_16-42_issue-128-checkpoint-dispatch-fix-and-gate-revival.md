# Status Report: Issue #128 (checkpoint-failure-fails-dispatch) + gate-chain revival

**Session:** 2026-11-11, ~15:00–16:42 CET · Crush on glm-5.3 · branch master (auto-commit daemon absorbed all work)

**Scope:** Fix GitHub issue #128 and everything its verification surfaced. This report covers ONLY this session's run.

---

## a) FULLY DONE (verified green this session)

### The issue-#128 fix (primary ask)

- **`projectionhost/v4/worker_drain.go` liveHandler:** a failed live checkpoint save now logs a warn and returns nil — it no longer fails the subscriber handler, the bus publish, and the command dispatch of an already-journaled+applied event. The staged checkpoint (`cpHasPending`) retries on the next live event; `flushPendingCheckpoint` at Stop gives the final retry (shutdown path already modeled warn-and-continue).
- **Tests:** `TestLiveCheckpoint_SaveFailureDoesNotFailHandler` (handler returns nil, warn logged, next event's flush persists evt2), `TestLiveCheckpoint_SaveFailureFlushesAtShutdown` (staged checkpoint carries forward through persistent failures, Stop flush persists the LAST staged event), `TestHost_LiveCheckpointFailure_DoesNotStopWorker` (rewritten — it previously PINNED the bug: live cp failure → worker failed/stopped). Full projectionhost suite green with `-race` (18.2s).

### The busy_timeout root cause (issue #128's secondary observation)

- **Root-caused, not guessed:** `NewFromDSN`/`NewSQLiteEngineFromDSNWith` executed `PRAGMA busy_timeout=5000` via one `db.ExecContext` — per-connection settings that reached exactly ONE pooled connection. Under `WithMaxOpenConns` every fresh connection opened with busy_timeout=0 → immediate SQLITE_BUSY. **Independently reproduced** with a standalone program: `conn1 busy_timeout=5000, conn2 busy_timeout=0` under the old pattern.
- **Fix:** new `metaengine/sqliteengine/v4/pragma_connector.go` — pool opens through a connector wrapper over `sqlite.NewConnector` (the driver-documented interposition path) applying pragmas in `Connect` on every new connection; eager `Ping` preserves fail-fast pragma validation. Both DSN constructors rewired (`dsl.go`); `pool.go`/`NewFromDSN` docs updated.
- **Test:** `TestNewFromDSN_PragmasHoldOnEveryPooledConnection` (two held-open distinct conns, both read busy_timeout=5000, cache_size=-65536, journal_mode=wal). Full sqliteengine suite green (10.9s).

### Blast radius (per repo procedure)

- Consumers tested green: `system/v4`, `systemtest/v4`, `stack/sqlite/v4`, `systemscenario/v4`, `metaengine/tursoengine/v4`.

### The dead gate chain found because procedure REQUIRED doc-check

The 2026-11-10 module-dir move (0e4fbb3d2) had silently broken EVERY layer of doc-check; push CI had never completed before 2026-11-09, so nothing caught it. 702 references failing as "not found" on clean HEAD (proven via a detached worktree before touching anything):

- `importDir` trimmed the `/v4` suffix → pointed at versionless dirs (`event/`) with no Go files → every symbol "not found" (1389 refs in the skill corpus) while the string-pure `TestImportDir_StripsVersionSuffixes` kept passing. **Fixed** (canonical path keeps the version segment; `fullDir` keeps a legacy-layout fallback; live-tree regression test `TestFullDir_ResolvesLiveTreeLayout` added — the pin that was missing).
- `mirrorAliasDirs` rejected the post-move mirror shape (`event/v4` ↔ `core/v5/event`) as UNKNOWN ambiguity. **Fixed** (+ test cases).
- Three test files hardcoded `../..` repo roots and one read `metaengine/enum_validation.go` at its pre-move path. **Fixed** via `findRepoRootFromPath(".")`.
- Both flake invocations (`#verify` inline + `#doc-check` app) passed `../../` file args that resolve into `cmd/` since the tool moved to `cmd/doc-check/v4`. **Fixed** (`../../../`); AGENTS.md's three documented invocations updated too.
- Real drift the dead gate had accumulated, repaired: `docs/METAENGINE_DOMAIN_LANGUAGE.md` stale `metaengine/adttest`/`metaengine/enginetest` import paths; `system/v4/README.md` unscoped `schema.` alias (block now imports `schema/v4`).
- **Result: `nix run .#doc-check` green end-to-end — 2794 references valid across 84 packages.**

### More gate-chain fallout repaired

- **api-stability golden** regenerated — also repaired a daemon-created drift (a committed cqrs-lint `CommandInfo→CommandMetadata` rename with no golden regen; verified the rename is committed code, not in-flight edits, before absorbing).
- **`cmd/api-stability/v4/main.go`** was a NEW 430-line file-size offender (committed post-move, uncaught by the never-completing CI). Split: golden-file half → `golden.go`; no signature changes; 300+139 lines. File-size gate green.

### Gates green this session

lint-scope 4/4 modules · file-size ratchet · duplication (0 new clones) · coverage floors (all 89 modules + 170 packages) · changelog-symbols (12 citations honest) · doc-check full corpus · md-go (1532 blocks) · api-stability meta-tests · doc-check module suite · CHANGELOG 4 entries · readmodels.md contract sentence added.

---

## b) PARTIALLY DONE

~~- **Verification of the fix:** every gate my changes touch ran individually, but the **composed `#verify` receipt is missing** (skipped for the exclusivity rule + runtime; every component is green, the composition is not proven in one run).~~ routed — M34 chain #4 armed (log build/verify-chain-20261114-053846.log); receipt rows TODO_LIST:331/495
~~- **Issue #128 closure:** fix + tests + changelog done; **no closing comment posted on the GitHub issue** (deliberate — owner reviews AI filings per the issue template's checkbox — but uncoordinated with you).~~ done — issue closed (gh issue list empty since 2026-11-13 close-out); fix+tests+changelog+docs landed
~~- **doc-check invocation split brain:** flake + AGENTS.md both updated, but the same invocation still lives in TWO places that can drift again (they did — that's how this broke).~~ routed — TODO_LIST harvest row "Doc-check single-source invocation" (open; names the split brain)

## c) NOT STARTED

~~- **Consumer validation in the CRM:** the issue's reproducer is `LarsArtmann/crm` `board_stage_matrix_test.go` (24 concurrent flows, one sqlite file, `-race`). Fixing the library without re-running the actual reproducer (pin bump + suite run in that repo) leaves the end-to-end claim unproven.~~ routed — TODO_LIST harvest row "(repo: crm) end-to-end proof" (rides release wave M35)
~~- **Sibling audits for the same pragma bug class:** `tursoengine` (delegates to sqlite) and `queue/sqlite` (modules.md documents its own "MaxOpenConns(1) + WAL + busy_timeout" wiring) were never inspected for the exec-once-per-pool pattern. Their tests ran green — absence of inspection, not absence of bugs.~~ done — DONE 2026-11-14 (M37/F229-F230): tursoengine CLEAN, queue/sqlite queueDSN extracted; TODO row closed with receipts
~~- `check-test-runtime` (new tests add wall-clock; budget/STALE rows unchecked), `check-canonical-facts` (AGENTS.md edited; counts untouched but gate unrun), `check-untested-exports` (new file pragma_connector.go unexported-only, believed safe, unverified), `doc-check doctor` subcommand, soak/fuzz legs.~~ done — canonical-facts rides chain #4 preflight (passed); test-runtime covered by existing package budgets; untested-exports: no new exported symbols; doctor leg rowed below

## d) TOTALLY FUCKED UP (nothing catastrophic — waste, not damage)

- **3 failed edit attempts from stale reads/approximate whitespace** (pool.go comment wrap, pragma_test import block twice) + **1 silently-non-matching perl multiline substitution** (caught only because I counted matches after). Each caught immediately; ~5 wasted round trips total.
- **2 doc-check runs with wrong relative paths** (ran the documented AGENTS command before checking the layout it describes — the doc was the stale thing I was there to check).
- **Initial test-name collision** (`failingCheckpointStore` already existed in host_test.go — grepped for it AFTER writing the file; should have grepped fixtures first).

## e) WHAT WE SHOULD IMPROVE (honest critique, worst first)

~~1. **I didn't mutation-verify my own projectionhost tests.** I proved the sqliteengine premise with a standalone repro, but argued the liveHandler tests fail-against-old-code "by construction" without actually reverting to demonstrate it. This repo's own policy: "new goldens must be mutation-tested before they count as pins." My pins are probably good; "probably" is not the standard.~~ routed — DONE 2026-11-14 (M37/F227): 3 tests fail against restored return saveErr, restored green
~~2. **The eager-ping error path (`openPragmaPool` failing construction on a bad pragma) has NO test** — I preserved fail-fast behavior without pinning it. A regression there fails silently at first query instead.~~ routed — DONE 2026-11-14 (M37/F228): both DSL constructors pinned
~~3. **Procedure miss: no FEATURES.md row upsert.** "Close a Wave" says CHANGELOG AND FEATURES for anything user-visible; I did CHANGELOG only. Same session that re-learned "module registration in 6 gates with zero feature-inventory presence is how drift starts."~~ routed — DONE 2026-11-14 (M37/F237): all six rows landed incl. state_shape caveat
~~4. **Unbounded warn-storm case undocumented:** with a persistently failing checkpoint store, the live path now warns forever while the reprocessing window grows unbounded; nothing surfaces in `WorkerState.LastError`. Correct trade-off (the old behavior was worse), but the operational ceiling deserves a TODO_LIST row and (v5) an additive metric hook.~~ routed — TODO_LIST "Checkpoint-failure observability family" row (open, M36 ruling first)
~~5. **Ghost gate revival is a bandage on a systemic hole:** the daemon commits code without regenerating goldens, and gates that would catch drift die silently when CI never completes. This session revived doc-check; nothing prevents the next structural move from killing it again for a week.~~ routed — TODO_LIST "Golden-drift daemon guard" row (regen-or-refuse on .go commits)
~~6. **`pragmaConnector.Connect` shows up in the api golden** (`metaengine/sqliteengine/v4/method Connect`) — an exported-name method on an UNEXPORTED type being captured by the golden scanner. Unexplained artifact I walked past; might be intended golden semantics, might be scanner imprecision.~~ routed — TODO_LIST "api-stability golden scanner semantics" row (investigate)
~~7. **`:memory:` + raised pool footgun** now more attractive: per-conn pragmas make `WithMaxOpenConns` look safe, but `:memory:` under a raised ceiling still yields SEPARATE databases per connection (SQLite semantics). My pool.go note doesn't warn about this.~~ routed — pool.go + gotchas-testing rows landed 2026-11-14 (F235); faq.md half open in the footgun row
~~8. **Prior-review continuity skipped** (per your no-research instruction): this self-review didn't cross-reference earlier brutal-self-review episodes; open items from those may overlap.~~ routed — not rowed (process note)

## f) NEXT: up to 50 things, roughly Pareto-sorted

~~1. Run composed `nix run .#verify` for the full receipt.~~ routed — M34 chain #4 armed; receipt lands TODO_LIST:331/495 on green
~~2. Post closing comment on issue #128 (after your manual review) + close.~~ routed — issue closed 2026-11-13 wave (gh list empty)
~~3. Validate against the real consumer: bump `LarsArtmann/crm` pin (replace → local tree), run `board_stage_matrix_test.go` under `-race`.~~ routed — TODO_LIST crm row (rides M35)
~~4. Mutation-verify: temporarily restore `return saveErr` in liveHandler, prove the 3 new tests fail, restore.~~ routed — DONE 2026-11-14 (M37/F227)
~~5. Add test: invalid pragma fails `NewFromDSN`/`NewSQLiteEngineFromDSNWith` at construction (eager ping path).~~ routed — DONE 2026-11-14 (M37/F228)
~~6. FEATURES.md rows: projectionhost checkpoint semantics, sqliteengine per-conn pragmas, doc-check revival, api-stability split.~~ routed — DONE 2026-11-14 (M37/F237)
~~7. Harvest this report's c/e items into TODO_LIST.md (docs-health HARVEST mode).~~ routed — M31/F194 harvest DONE 2026-11-13
~~8. Audit `tursoengine` for the same once-per-pool pragma pattern.~~ routed — DONE 2026-11-14 (M37/F229)
~~9. Audit `queue/sqlite` for the same (its own DSN/pool wiring per modules.md).~~ routed — DONE 2026-11-14 (M37/F230)
~~10. Run `nix run .#check-test-runtime` — new tests' package budgets.~~ routed — existing package budgets cover (no new packages); gate judges at next #test
~~11. Run `nix run .#check-untested-exports` over sqliteengine/projectionhost/doc-check.~~ routed — no new exported symbols this arc; gate green at last battery
~~12. Run `bash scripts/check-canonical-facts.sh` (AGENTS.md was edited).~~ routed — chain #4 preflight passed (canonical-facts in battery)
~~13. Single-source the doc-check invocation (one script flake+AGENTS reference) to kill the split brain permanently.~~ routed — r
~~14. Add a CI/flake leg that runs doc-check with NO file args from repo root; fix the default corpus path if it's also CWD-relative.~~ routed — r
~~15. Make the auto-commit daemon (or a post-commit hook) run `api-stability --update` check-or-fail on `.go` commits so golden drift can't be committed silently again (the CommandMetadata incident).~~ routed — r
~~16. Investigate why golden scanner lists methods of unexported types (`Connect`); tighten or document.~~ routed — r
~~17. TODO_LIST row: bounded-warn-storm — surface persistent live-checkpoint failure in WorkerState/metrics (v5-additive MetricsRecorder hook).~~ routed — r
~~18. Document `:memory:` + raised MaxOpenConns footgun in pool.go/WithMaxOpenConns.~~ routed — TODO_LIST ":memory: + raised MaxOpenConns footgun doc" row (faq.md half open)
~~19. `doc-check doctor` run; wire stale-invocation detection (paths that resolve nowhere) into it.~~ routed — not rowed (doctor subcommand leg)
~~20. Add `TestFullDir` legacy-layout fallback case (dir without /v4) to pin the pre-move branch I preserved.~~ routed — r
~~21. Sweep other `cd cmd/<tool>` invocations in flake/AGENTS/docs for the same one-level-short v4-move breakage (api-stability? cqrs-lint? cqrs-upgrade?).~~ routed — r
~~22. Check `cmd/doc-check` README for stale paths (readme_compile_test now passes, but prose may lie).~~ routed — r
~~23. Run `#check-package-surface` (new files in sqliteengine/doc-check packages).~~ routed — h
~~24. Run the gosec gate over changed modules (`#check-gosec`).~~ routed — r
~~25. Consider turso DSN constructors adopting the shared connector pattern (extract to metaengine core if both engines need it — contract 27 pattern).~~ routed — r
~~26. CHANGELOG: verify the 4 entries render correctly on the next release-notes cut.~~ routed — r
~~27. Nightly bench sanity: pragma connector adds per-connect execs — confirm connect-latency-sensitive benchmarks unaffected.~~ routed — declined-with-reason (harvest ledger: pool pre-warm by construction + nightly bench)
~~28. Re-run `check-lint-config` (depgraph untouched, but cheap).~~ routed — M34 close-out gate battery (post-chain)
~~29. `git log` review of the daemon's 12 commits from this session for anything absorbed mid-edit (I verified files exist; per-hunk review not done).~~ routed — not rowed; commits long absorbed
~~30. Ask owner about cqrs-lint parallel session (see questions) — possible golden race.~~ routed — 16-42 verified the rename was committed code, not in-flight; no race materialized
~~31. systemtest: add a scenario pinning "checkpoint store failure does not fail dispatch" at the SYSTEM tier (systemscenario), not just the host tier.~~ routed — r
~~32. Consider degrading DRAIN-phase checkpoint failures to warn+retry too (worker restart is self-healing but noisy; separate decision, needs owner).~~ routed — r
~~33. Read prior brutal-self-review episodes and cross-reference open items (skipped this session by instruction).~~ routed — not rowed
~~34. The `errors.New("database is locked (5) (SQLITE_BUSY)")` test fixture — consider asserting the errorfamily classification of the wrapped error in production code (Infrastructure) with a doc pointer.~~ routed — r
~~35. Update `.agents/skills/go-cqrs-lite/references/faq.md` with the SQLITE_BUSY-under-raised-pool pitfall (one paragraph, consumer-facing).~~ routed — r
~~36. Sweep skill references for other stale pre-move import paths (corpus is green, but prose outside fences isn't symbol-checked).~~ routed — r
~~37. Verify pkg.go.dev renders the new doc comments correctly after the next tag wave (go-release checklist).~~ routed — declined-with-reason (ledger: go-release ritual)
~~38. Tag-wave prep: both fixed modules need releases for the CRM to consume (projectionhost, sqliteengine) — batch-release smoke after #verify.~~ routed — r
~~39. Consider `WithCheckpointFailureHook` (additive option) for consumers needing more than a warn log.~~ routed — r
~~40. Pin `runImmediateTx` + pragmaConnector interplay: pragmas apply BEFORE any tx on each conn — add a comment-level cross-link in transaction.go.~~ routed — r
~~41. Add `#check-md-go` case for import-only fences (README edit introduced one).~~ routed — r
~~42. Check whether `check-readme-links.sh`/`check-readme-deprecated.sh` (nightly) cover the system README edit.~~ routed — nightly gates (check-readme-links/deprecated) cover README edits automatically
~~43. Consider teaching doc-check to FAIL when a file argument doesn't exist (it errored, but as a broken-reference count — make it a distinct loud error).~~ routed — r
~~44. The 54-vs-702 failure counts in my entry: they're accurate per-run; keep the report numbers straight if anyone quotes them.~~ routed — h
~~45. Look at whether projectionhost should ALSO retry the flush on a timer when idle (today retry only happens on next event / Stop — a quiet stream with a failed save stays stale until the next event).~~ routed — r
~~46. Confirm `dedup`-heavy paths unaffected by pragma connect latency (pool pre-warm at construction covers it — one Ping).~~ routed — declined-with-reason (ledger: pool pre-warm)
~~47. Review whether `pragmaConnector` should implement `driver.Validator`/session-reset semantics modernc exposes (pool reset after errors).~~ routed — r
~~48. Add the sqliteengine fix to the engine-capabilities doc ONLY if owner wants per-conn pragmas as a stated capability (contract: generated doc, don't hand-edit).~~ routed — owner-conditional (stated capability?) — not rowed
~~49. Skim `docs/agents/gotchas-module-management.md` for an entry about the v4-move gate breakage class (one paragraph so the next mover runs the gates).~~ routed — r
~~50. If everything above is green: close the loop by updating this report's header with the verify receipt timestamp.~~ routed — M34 close-out (TODO_LIST:331/495)

## g) Questions I cannot answer myself

~~1. **Issue #128:** may I post the closing comment + close it now, or do you want to tick the "MANUALLY REVIEWED" box first and have me hold? (The fix, tests, changelog, and docs are in; only the GitHub thread is untouched.)~~ routed — closed 2026-13 wave; gh issue list empty
~~2. **CRM validation:** do you want me to bump `LarsArtmann/crm` against this tree (replace directive) and run their `board_stage_matrix_test.go` under `-race` as the real end-to-end proof, or is that yours?~~ routed — TODO_LIST crm row rides M35
~~3. **cqrs-lint golden race:** the daemon committed a `CommandInfo→CommandMetadata` rename without regenerating the api golden (I repaired the golden). Is another session mid-flight on cqrs-lint that my regen could have stepped on — and should the daemon/hook learn to regen-or-refuse on `.go` commits so committed golden drift becomes impossible?~~ routed — race resolved (committed code); regen-or-refuse = Golden-drift daemon guard row (open)

---

*Generated by Crush (glm-5.3) at the owner's request, 2026-11-11 16:42 CET. Point-in-time snapshot; not a backlog home — f-items 1–7 are the harvest candidates for TODO_LIST.md.*
