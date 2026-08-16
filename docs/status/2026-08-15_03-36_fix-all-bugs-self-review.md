# Session Self-Review + Status — FIX ALL BUGS run: what I did, forgot, and should do next — 2026-08-15 03:36

> Brutal self-review of THIS session only (the "FIX ALL BUGS" continuation of
> the standing READ→UNDERSTAND→RESEARCH→REFLECT→execute loop). Predecessor:
> `2026-08-15_03-30_fix-all-bugs-seq-positional-benchkit-skew-green-gate.md`
> (the work report; this adds the review layer the user asked for).
> Scope: only this session's work and what I noticed during it.

## a) FULLY DONE (verified, gate green)

1. **JournalReadFrom raw-seq filtering fixed on all four SQL engines**
   (sqliteengine, pgengine, mysqlengine, duckdbengine). They filtered
   `seq > afterSeq` on a GLOBAL-across-collections counter while the contract
   and every caller (`EventAdapter.lookupSeq`, `AdapterCore.ReadFromAfter`,
   harness) passes a collection-local POSITION → duplicate re-delivery on
   resume whenever collections interleaved (e.g. commands written before
   events on one engine). OFFSET-based positional skip per dialect (sqlite
   `LIMIT -1 OFFSET`, pg bare `OFFSET`, mysql `LIMIT 18446744073709551615
   OFFSET`, duckdb `LIMIT ALL OFFSET`). Contract documented in engine.go.
2. **Regression tests PROVEN to catch the bug**: interleaved-collections phase
   in `enginetest.RunStreamLogBackendTest(In)` + end-to-end
   `system.TestEventAdapter_ReadFrom_InterleavedCollections` (first-ever
   EventAdapter test). Proof protocol: temporarily reverted the sqlite fix →
   both tests failed with exactly the predicted re-delivery → restored →
   green. Not "tests that pass", tests that FAIL on the old code.
3. **Live verification of every touched engine**: ephemeral PG ✓ (pgengine
   3.1s incl. new harness phases), MySQL nspawn ✓ (StreamLog roundtrip incl.
   the unlimited-LIMIT branch), Dgraph ✓ 24/24 (61s, includes new interleave
   phase via parity collection), duckdb in-process CGO ✓, sqlite/bbolt/pebble/
   badger/metaengine/system local ✓.
4. **benchkit deterministic failures root-caused as dependency skew** — pins
   `stack/sqlite`+`stack/pebble` at v4.1.0 lacked the pool cap (SQLITE_BUSY
   517) and WithDiskSize (0 bytes). Bumped to v4.3.0; full suite green
   standalone (34s). Proven pre-existing via pre-change-commit worktree
   (git worktree, never checkout/reset).
5. **Coverage gaps closed**: pgengine onto the shared harness (replacing a
   hand-rolled subset); sqliteengine contract test; badgerengine's FIRST-ever
   StreamLog contract test; `adttest.RunMatrix` doc pins the
   fixed-collection/fresh-database constraint for shared-server engines.
6. **Reverse LAYER meta-test** (`TestEveryModuleHasLayerEntry`) — every go.mod
   dir must have a LAYER entry (81/81 today). New-module omission now fails
   the api-stability suite instead of silently skipping layer enforcement.
7. **Gate hardening + docs**: verify Test-phase timeout 5m→8m (was tighter
   than the slower Race phase — backwards); AGENTS.md gate-exclusivity gotcha
   (the prior session's 25-min lesson, now written where future sessions
   look); ephemeral-dgraph.sh header no longer lies about direct bash
   invocation; CHANGELOG (3 entries); TODO_LIST updated (MariaDB compat,
   seq-carrying now perf-only).
8. **Gate**: `nix run .#verify` **EXIT=0, all 18 phases, 0 FAIL lines, 239 ok
   packages, lint 76/76, doc-check 1020 refs** (/tmp/verify-final3.log). Ran
   exclusively (rule applied to myself); report written AFTER the gate.

## b) PARTIALLY DONE

1. **MariaDB vs mysqlengine**: classified (3 pushdown tests emit MySQL-8 JSON
   path syntax MariaDB rejects; ADTMatrix/HealthCheck fail "invalid connection"
   in nspawn) and TODO'd — but no repro isolation, no dialect probe, no
   decision. The failures could ALSO partially be a nspawn-env artifact
   ("invalid connection" ≠ syntax error — two distinct failure classes I
   lumped together without separating).
2. **benchkit stale pins**: only the two BREAKING pins bumped (stack/sqlite,
   stack/pebble). benchkit still pins decider v4.3.0, event v4.6.0,
   sqliteengine v4.0.1 (pre-my-fix!), metaengine v4.10.0, stack v4.3.0…
   Nothing else broke TODAY, but the skew CLASS is unaddressed repo-wide (see
   f1). Notably sqliteengine v4.0.1 pin means benchkit standalone still tests
   the OLD JournalReadFrom — only the workspace replace saves it.
3. **Diagnostics hygiene**: the 30-error gopls flood (go.work 1.26.6 bump,
   prior session) was ignored all session as "known noise". Correct for
   builds, but gopls was never restarted (prior f6 item), so ALL LSP
   diagnostics this session were untrustworthy — I verified everything by
   build/vet/test instead, which worked but leaves the editor broken for the
   next human.

## c) NOT STARTED (deliberately deferred / blocked)

1. Tagging — the JournalReadFrom fix is INVISIBLE to external consumers until
   sqliteengine/pgengine/mysqlengine/duckdbengine (+ metaengine, system) get
   new tags. Everything sits uncommitted-to-tags in the workspace (blocked on
   user question g1, now MORE urgent than before).
2. The two other standing questions: Go 1.26.6 repo-wide adoption (g2) and
   SA1019 exclusion permanence (g3, third session open).
3. From the 02:31 f-list untouched this session by choice (scope): cqrs-lint
   per-module regression tests; `.golangci.yml` exclusion audit; DuckDB/Row
   calibration benches + CI regression check; v5 Phase 8 deletions + migration
   guide; docs-health HARVEST pass over 2026-08-14/15 reports; watermill real
   broker edges; ephemeral-pg macOS; #check-lint-config/#verify-ci apps; #sweep
   wiring; register.go consolidation; two-live-engine test; vector brute force
   on LSM engines; recursive CTE graph dispatch; layout long-horizon items.
4. CI status never checked — I never looked at whether GitHub Actions was red
   on benchkit before my fix (CI runs GOWORK=off per-module, so it plausibly
   WAS red on those two tests for a while and nobody noticed — the gate's
   workspace resolution masked it locally). Unverified hypothesis.

## d) TOTALLY FUCKED UP (honest ledger, this session)

1. **Five-iteration probe-test dance in stack/pebble** — I wrote the DiskSize
   probe from memory (wrong `id.NewStreamRef` arg order, wrong `event.New`
   signature, testutil dep not present, StreamID type) instead of reading
   `preset_test.go` FIRST. Wasted 4 round trips; the 5th attempt just copied
   the file's own idiom. Lesson already in AGENTS.md ("read similar code for
   patterns"); I skipped it because "it's just a throwaway probe".
2. **mysqlengine live-run invocation error**: `nix run .#integration-mysql-nspawn
   -- test -tags ...` — the script prepends its own `test`, producing
   `go test test ...` → "package test is not in std". Should have read the
   script's arg handling (or noticed its `Running: go test $*` echo) before
   burning a ~40s nspawn boot. Second script-invocation lesson in two
   sessions.
3. **Wrote the reverse LAYER meta-test with a type error**
   (`excluded[rel] || infra[rel]` — bool vs struct{}) and shipped the first
   test run through `| grep` piping (EXIT echoed grep's status — the EXACT
   AGENTS.md anti-pattern) before re-running with proper log capture. The
   type error I then diagnosed via a separate vet call; both fixes were
   quick but both were self-inflicted round trips.
4. **edit-without-read**: attempted to edit sqliteengine/engine.go before
   viewing it (tool refused). I had grep'd the target lines but "read the
   relevant context" means View, not grep. One wasted call, zero damage —
   the tool saved me from myself.
5. **Sloppy failure triage on the first PG integration run**: benchkit failed
   inside `#integration-pg` and my first thought path was "did I break it?"
   — correct instinct — but I initially ran the isolated tests WITHOUT first
   checking whether integration-pg was even supposed to run benchkit (it
   runs a fixed module set including benchkit). Understanding the runner's
   scope would have predicted the failure surface immediately.
6. **`go mod tidy` in system/ without diffing the result** — tidy after adding
   the test's blank sqlite import; I never inspected what changed in
   go.mod/go.sum (dep budget is gate-checked, and check-arch passed, but I
   didn't KNOW what I added when I moved on — budget compliance by luck of
   the gate, not by intent).

## e) WHAT WE SHOULD IMPROVE (process, from this session's scars)

1. **Read the sibling test file BEFORE writing any new test in a package**
   — the 5-attempt probe would have been 1 attempt. Cheapest lesson on this
   list and I violated it first thing.
2. **Pin-drift detection is missing as a gate**: two classes of skew found in
   two sessions (system's temporary replaces; benchkit's stale pins). A
   meta-test comparing each in-repo module's required versions of sibling
   modules against `git tag -l '<module>/v4*' | sort -V | tail -1` would catch
   both classes automatically at test time instead of via broken CI or
   manual archaeology. (Concrete: f2 below.)
3. **The gate masks standalone breakage by design** — `#verify` resolves the
   workspace (local modules), CI runs GOWORK=off. Anything that passes the
   gate but fails standalone (benchkit's case) is invisible locally. Either
   add a periodic `#verify-standalone` (GOWORK=off per module, mirroring CI)
   or accept CI as the only standalone signal and CHECK CI after gates.
4. **Failure taxonomy discipline**: "MariaDB rejects syntax" and "invalid
   connection" got one TODO item. Different root causes need different items
   — lumping them creates a TODO nobody can start.
5. **gopls restart belongs in session STARTUP after any go.work change**, not
   as an ignorable noise footnote — otherwise every diagnostic all session is
   dead weight and the LSP tools (rename, symbols) are unusable.

## f) NEXT — ordered by leverage (not all 50; these are real, deduped)

**Release (blocked on g1):**

1. Tag engine fixes: sqliteengine/pgengine/mysqlengine/duckdbengine v4.0.2+ <- OPEN. TODO_LIST 'Release / Tagging' + ROADMAP 'Open Questions' #1 (fix invisible to consumers until tagged)
   (+ metaengine, system, stack/sqlite, stack/pebble consumers as needed) —
   otherwise the re-delivery fix never reaches consumers.
2. Same pass: engines v4.0.2 (badger/pebble) + watermill/v4.5.0 from the prior <- OPEN. TODO_LIST 'Release / Tagging'
   queue; then drop system's 5 temporary replaces; standalone re-verify.
3. `go mod tidy` sweep of ~49 stale indirect refs; `nix run .#vulncheck` + <- OPEN. TODO_LIST 'Release / Tagging' (tidy sweep + pre-tag checklist)
   `#check-arch` as pre-tag checklist.

**Skew/drift:**
4. Pin-drift meta-test (required-version vs latest tag per sibling module). <- OPEN. TODO_LIST 'Pin & Standalone-Build Hygiene' (pin-drift meta-test, flagged urgent); gated on ROADMAP 'Open Questions' #4
5. Repo-wide stale-pin sweep for ALL modules (benchkit still pins <- OPEN. TODO_LIST 'Pin & Standalone-Build Hygiene' (repo-wide stale-pin sweep, flagged urgent)
sqliteengine v4.0.1 = the pre-fix code, decider v4.3.0, event v4.6.0…).
~~6. Check CI (gh run list) for red benchkit runs predating the fix — establish~~ done - checked 2026-08-15 (docs-health pass): CI 'Benchmarks' job red, 3 recent failures, consistent with standalone skew; the predating window was not bisected further
how long standalone was broken.
7. `#verify-standalone` nix app (GOWORK=off per module) or explicit decision <- OPEN. TODO_LIST 'Pin & Standalone-Build Hygiene' (#verify-standalone)
that CI owns that signal.

**Correctness follow-ups:**
8. Seq-carrying journal reads (perf): OFFSET skip is O(offset) per page; <- OPEN. TODO_LIST 'Metaengine' (seq-carrying perf follow-up)
`JournalReadAllWithSeq`/`StreamLogEntry{Seq,Value}` enables index-seek
resume. Correctness is done; this is the large-journal performance item.
9. mysqlengine/MariaDB: separate the two failure classes (JSON-path syntax vs <- OPEN. TODO_LIST 'Metaengine' (MariaDB compatibility)
invalid connection); decide dialect support vs MySQL-8-only + test-backend
swap.
10. Two-live-engine integration test (AddEngine + Backfill correctness). <- OPEN. TODO_LIST 'Metaengine' (two live backends)
11. Vector ADT brute-force on Pebble/bbolt. <- OPEN. in flight - concurrent metaengine session writing bbolt/pebble vector backends now
12. Recursive CTE graph dispatch for PG/MySQL; recursive CTE optimization for <- OPEN. in flight - pgengine/mysqlengine/sqliteengine graph work in the concurrent session's tree
deep SQLite traversals.

**Toolchain / editor:**
13. Restart gopls (go.work bump noise) at next session start. <- NOT-DO - session-start operational habit (gopls restart), no repo artifact
14. Go 1.26.6 decision + alignment sweep (go.mod files, CI, nix pin, <- OPEN. ROADMAP 'Open Questions' #2
.go-version) — blocked on g2.
~~15. Add `nix fmt` to the end-of-session checklist (I only gofumpt'd changed~~ WONT - session checklist habit; lint gate already enforces formatting
dirs; lint covered it this time, convention says treefmt).

**Hardening / tests:**
16. cqrs-lint per-module regression tests (F004, F007, F009, F012, F017, <- OPEN. TODO_LIST 'cqrs-lint' (per-module regression tests)
F023–F029, B030).
17. `.golangci.yml` exclusion audit (system/ 20 linters, cmd/cqrs-lint/ 17, <- OPEN. TODO_LIST 'Code Quality' (.golangci.yml exclusion audit)
metaengine/ 24).
18. Watermill-redisstream real-broker edge tests (redelivery duplicates, <- OPEN. TODO_LIST 'Code Quality' (Wire broker tests into CI)
consumer-group rebalance, message size limits).
19. duckdbengine suite split or budget raise (76-91s observed; now 8m ceiling <- OPEN. TODO_LIST 'Code Quality' (duckdbengine suite split)
— healthier, still worth splitting the soak).
20. ephemeral-pg.sh macOS verification. <- OPEN. TODO_LIST 'Code Quality' (macOS ephemeral-pg)
21. Doc-check 0-warning CI tripwire (regression guard for the count). <- OPEN. TODO_LIST 'Code Quality' (Doc-check 0-warning tripwire)

**Infrastructure polish:**
22. `#check-lint-config` + `#verify-ci` nix apps mirroring GH Actions. <- OPEN. TODO_LIST 'Code Quality' (Infrastructure polish)
23. Wire `#sweep` to pre-commit/cron. <- OPEN. TODO_LIST 'Code Quality' (Infrastructure polish)
24. Consolidate engine `register.go` boilerplate (7 modules). <- OPEN. TODO_LIST 'Code Quality' (Infrastructure polish)
25. Calibration-baseline CI regression check; DuckDB 60s disk calibration; <- OPEN. TODO_LIST 'Metaengine' (calibration benchmarks + CI regression check)
SQLite/PG/MySQL Row-layout calibration.

**Docs/debt:**
~~26. docs-health HARVEST: open items from 2026-08-14/15 reports → TODO_LIST.~~ done - TODO_LIST rebuilt + reports annotated/archived by the docs-health audit 2026-08-15
27. v5 Phase 8 sequence: delete stack.Materialize → RelationalProjection/view <- OPEN. TODO_LIST 'v5 Unification Phase 8' (each step is its own item)
→ graph.GraphProjection → Bundle+8 presets → RunProjections → ADR-0126
shells → transport/http+grpc registry drop → migration guide → cut v5.0.0.
28. go-codec repo scaffolding (sibling session's lane — do not start here).

## g) QUESTIONS (cannot figure out myself)

1. **Tag + push authorization** (standing, now critical): the re-delivery fix
   only exists in the workspace until sqliteengine/pgengine/mysqlengine/
   duckdbengine (+ metaengine/system) are tagged — consumers on v4.0.1 are
   still double-processing on resume. Authorize a tagging pass (which
   modules, which versions), and is pushing tags + master allowed? (Never
   tag/push without explicit instruction.)
2. **Go 1.26.6 direction** (standing): adopt repo-wide (go.mod sweep + CI +
   nix go pin + .go-version), or hold at 1.26.5 and ask the sibling go-codec
   session to revert its uncommitted bump? The half-state (go.work only)
   keeps gopls noisy and CI unaligned.
3. **Stale-pin sweep policy** (new): may I bump ALL sibling-module pins
   repo-wide to latest tags (~50 go.mod files, mechanical, gate-verified), or
   should pin bumps happen only when something breaks (today's benchkit
   case)? A yes also greenlights the pin-drift meta-test (f4) failing CI on
   staleness instead of on breakage.

## Gate evidence (final, unchanged since the work report)

- `nix run .#verify`: EXIT=0, all 18 phases, 0 FAIL, 239 ok packages,
  `✅ Lint: 76/76 modules clean`, doc-check 1020 refs (/tmp/verify-final3.log)
- Live: PG pgengine ok; MySQL StreamLog PASS; Dgraph 24/24 (61.1s)
- Standalone: benchkit 34.4s green after pin bump

---

## Resolution (2026-08-15)

27 of 28 items carry verdicts (item 28, go-codec scaffolding, is the sibling
repo's lane and stays open unrouted). Everything closable was closed by
`4a95bd04d` (the four-engine positional fix + meta-tests + pin bumps) and
verified by this pass. The release chain (1-3) is the critical open path -
consumers on old tags still double-process on resume; tracked in TODO_LIST
"Release / Tagging" and ROADMAP Open Questions #1/#4. Stays active for item 28.
