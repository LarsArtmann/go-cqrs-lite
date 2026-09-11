# Status Report — Cordis Follow-up Wave: EngineResetter Everywhere + Fold-Write Failover

> **Point-in-time snapshot** — 2026-09-11 05:40 CEST. Scope: THIS session's
> execution of the Cordis spatiotemporal-composability follow-ups
> (TODO_LIST section, 8 items) plus what I observed around it. Written in
> Markdown by explicit user instruction (the status-report skill's HTML
> default was overridden — flagged here per skill policy).
>
> **Context:** a CONCURRENT agent session was executing the same TODO list in
> parallel the entire time. Division of labor ended up: mine = sqlite/turso,
> pg, iroh, fold-write failover, Doctor/Stats surface, C040/E018, goleak,
> tripwire, docs/golden; theirs = pebble, bbolt, badger, mysql, duckdb,
> dgraph (their work reviewed and tested by me where possible; their in-flight
> dgraphengine metrics work untouched).

## a) FULLY DONE (verified green this session)

1. **`sqliteengine.ResetEngine`** (was 🔥): 8 `meta_*` tables + planned-table
   rows (layouts survive — post-Plan semantics) + matviews drop/recreate +
   cached `multiSeq` counters dropped; journal AUTOINCREMENT deliberately
   keeps advancing (resumption tokens stay valid). 4 tests incl. Store-level
   non-partial assert. `metaengine/sqliteengine/reset.go`, `reset_test.go`.
2. **Turso reset** by delegation (tursoengine wraps sqliteengine) — plus a
   LIVE matview-reset test that caught two real defects before they shipped:
   libSQL rejects `DROP MATERIALIZED VIEW` (plain `DROP VIEW` is the syntax)
   and an emptied view made scalar aggregates ERROR instead of returning 0
   (now matches the base path's `DecodeFloat(nil)=0` convention).
   `tursoengine/reset_matview_test.go`, `sqliteengine/materialized_view.go`.
3. **`pgengine.ResetEngine`** — 5 base tables (incl. `meta_vector`) + planned
   tables, one tx, serialized against RunInTx via `mu`. **Verified against a
   live Postgres testcontainer** (113s suite, both tests PASS).
4. **`irohengine.ResetEngine`** — node-local reset through the local engine,
   LWW timestamp map deliberately KEPT as tombstone barriers (stale in-flight
   peer writes rejected post-reset; fresh replay writes win); loud error when
   the local engine isn't resettable. 2 tests. Needed a documented sibling
   replace in `irohengine/go.mod` (unpublished `EngineResetter`).
5. **Fold-write failover (ADR-0137 completion, was L)**: folds reroute around
   quarantined engines in `dispatchFoldsCoreLocked` (same capability-aware
   partition rule as reads, execution-scoped, plan untouched); new
   `Store.CatchUpEngine` = reset + EventLog replay into EXACTLY the
   quarantined engine, quarantine lifted only after a clean rebuild;
   `StartAutoReprobe` prefers catch-up with `ErrCatchUpUnsupported` fallback;
   `ReactivateEngine` documents the stale-read caveat. 5 new tests, all
   pre-existing health tests still green.
6. **Reset observability**: `EngineStats.CanReset` + Doctor `--- Reset ---`
   section with remedy line. 2 tests (`reset_observability.go`,
   `engine_stats.go`, `reset_test.go` additions).
7. **C040/E018 fold-case coverage** (TODO said "E018 fold-case"): C040 now
   catalog-aware (provider parity with E018 + runtime gate — kills the
   imported-event false positive) and fires beside a near-miss when the
   corrected twin is also handled (the `user.creted`-next-to-
   `user.created` hole where C038 is blind). 2 new tests. E018 comment +
   catalog descriptions updated; RULES.md REGENERATED from the generator (my
   manual edit was wrong — see d).
8. **goleak gates** for `metaengine` (ginkgo interrupt-handler ignore
   documented) and `projectionhost` (`//go:build !integration` to avoid
   TestMain clash). Both suites pass with the gates in place.
9. **`[Unreleased]`-position tripwire** in `verify-docs.sh`: first `##`
   section under the header must be `[Unreleased]`; verified positive AND
   negative (orphan case fails).
10. **Docs & gates**: api-stability golden regenerated (+4 symbols: iroh
    ResetEngine, CatchUpEngine, ErrCatchUpUnsupported), CHANGELOG `[Unreleased]`
    gained 4 sections (changelog-symbol gate green after fixing one
    `goleak.VerifyTestMain` citation shape), TODO_LIST Cordis section collapsed
    to just the release-train note, AGENTS.md contracts #22/#23 updated, skill
    references (readmodels.md capability table, recipes.md §2.33) updated,
    doc-check green (1049 refs), `nix fmt` green, golangci-lint green on
    metaengine/sqliteengine/tursoengine/irohengine/pgengine.
11. **Concurrent session's engine resets reviewed + tested where possible**:
    pebble (tag-range batch delete, foreign keys survive), bbolt (bucket
    drop/recreate), badger (DropPrefix) — their `-run Reset` suites run green
    by me; mysql/duckdb/dgraph reviewed by reading (see b for the gap).

## b) PARTIALLY DONE

1. **mysql / duckdb / dgraph reset verification**: implementations exist
   (concurrent session) and I reviewed the code — but I never EXECUTED them:
   mysql needs a live server, dgraph needs a server, duckdb needs CGo
   (compilation itself unverified in this session). Their test files are
   skip-guarded, so a green run on this machine proves nothing. The nix
   integration targets (`#integration-mysql-vm`, `#integration-dgraph`,
   duckdb via CI matrix) remain the honest verification.
2. **CatchUpEngine concurrency**: the catch-up replay is safe against live
   writes in the steady state (live folds reroute away from the quarantined
   engine, so no double-apply), BUT the EventLog snapshot is taken once at
   start — events recorded DURING the replay window are folded onto the
   failover engine and missed by the rebuild, leaving the reactivated engine
   stale for exactly that window. Sequential tests pass; the concurrent hole
   is real (see d/e). Fix designed, not implemented.
3. **`verify-docs.sh` end-to-end**: new check tested standalone; the full
   script (which also runs `nix run .#build`) never executed as a whole this
   session.
4. **projectionhost integration build**: my `!integration` TestMain was never
   compile-checked WITH `-tags integration` (default build is green) — the
   two-TestMain clash risk is handled by the build tag but unproven.

## c) NOT STARTED (known, deliberately)

1. **Release-train note** — standing item, not actionable until the next tag
   wave: `metaengine/projectionadapter` + `metaengine/irohengine` sibling
   replaces stripped, pins bumped, smoke at cut time.
2. `nix run .#verify` (full gate incl. `-race`) — deliberately NOT run: it is
   exclusive, and the concurrent session had uncommitted in-flight work
   (dgraphengine metrics) that would have made a repo-wide run
   unattributable. My dispatch-core refactor touching the hottest write path
   has therefore never seen the race detector (plain tests only).
3. Per-engine catch-up high-water marks / Doctor surfacing of catch-up state
   (see e/f).

## d) TOTALLY FUCKED UP (honest ledger)

1. **THE CATCH-UP SNAPSHOT RACE** — the only real correctness hole I shipped:
   `CatchUpEngine` replays a once-taken `Events()` snapshot; concurrent
   applies during the (potentially long) replay are folded onto the failover
   engine only, so the reactivated engine silently misses that window. Tests
   are sequential and cannot see it. Not "shipped broken" in the default
   sequential path, but the API invites concurrent use. Fix: loop
   (reset+replay) until the log length stops growing between snapshot and
   reactivation, or hold `s.mu` write-locked for the final stabilization
   pass. Should be fast-followed BEFORE the next tag.
2. **I hand-edited RULES.md before remembering it is GENERATED** — the repo
   documents the regen command and `TestRULESMD_Fresh` failed on my run. Lost
   a round trip; the fix (catalog descriptions + regen) was correct.
3. **Initial pg reset test wrote against a nonexistent helper**
   (`pgFindTaskQuery`) and an unused import — caught by vet/build, but I
   guessed instead of checking fixtures first. Same class: my first turso
   matview test assumed `DROP MATERIALIZED VIEW` syntax — WRONG, real turso-go
   parse error (this one was worth it: verified-external-claims in action, the
   error was the finding).
4. **Doc-precision nit I introduced**: AGENTS #22 now says "sequence counters
   deliberately keep advancing across resets" — true for journals/AUTOINCREMENT
   and the KV engines' in-memory seqs, but sqlite's cached `multiSeq` is
   RESTARTED (per the TODO's wording). Both behaviors are individually correct
   and tested; my one-line summary overgeneralizes.
5. **`rg` output mangling** bit me twice (function names displayed as "n",
   `EngineResetter` as "n") — I worked around it, but early on it nearly
   caused me to misjudge what existed (e.g. "dispatchFoldsLocked renamed n").
   Cross-checked with `sed` each time; lesson: never trust filtered tool
   output for existence claims.

## e) WHAT WE SHOULD IMPROVE

1. **Close the catch-up race** (loop-until-stable or write-lock finalize) and
   add a concurrent stress test (apply loop racing CatchUpEngine; assert
   post-reactivation reads see every event).
2. **Race-detector the write-path refactor**: run `-race` on metaengine (and
   `nix run .#verify` once the tree is quiet) — dispatchFoldsCoreLocked is
   the hottest fold path and only saw plain tests.
3. **Compile-check projectionhost with `-tags integration`** (one command).
4. **Run the mysql/duckdb/dgraph reset tests against real backends** via the
   nix integration targets so "EngineResetter everywhere" is verified, not
   reviewed.
5. **Catch-up UX**: surface catch-up state (running/failed/last-caught-up) in
   Doctor + `GetEngineStats`; today it's slog-only.
6. **Per-engine high-water marks** so re-catch-up replays only the tail
   instead of the full journal every reactivation.
7. **AGENTS #22 wording precision** (the multiSeq vs journal-seq distinction).
8. **Coordination with concurrent sessions**: I detected the parallel session
   only via unexplained git-status entries ~40 minutes in. A lightweight
   convention (e.g. a claim line per TODO item, or checking `git log` before
   each file write — which I DID adopt and it prevented two collisions) would
   remove the duplicated-research waste (I fully designed pebble/bbolt/badger
   resets minutes before discovering they were already committed).
9. **routedQuery doc comment** now serves reads AND fold-reroute; update its
   comment to say so (tiny).
10. **Cost of quarantine reroute**: `bestHealthyEngineLocked` runs per fold
    task per event while an engine is quarantined (planner cost estimation per
    task). Cache per-dispatch or per-quarantine-transition if it shows in
    benches.

## f) NEXT — up to 50 things (session-derived, then standing TODO_LIST)

**From this session (highest priority first):**
1. Fix the CatchUpEngine snapshot race (loop-until-stable / write-lock finalize) + concurrent stress test.
2. `-race` run over metaengine (dispatch core refactor).
3. Full `nix run .#verify` once the concurrent session's tree is quiet.
4. `go vet -tags integration ./...` in projectionhost (TestMain clash check).
5. DuckDB reset suite executed (CGo build) — never ran this session.
6. MySQL reset suite via `nix run .#integration-mysql-vm`.
7. Dgraph reset suite via `nix run .#integration-dgraph`.
8. End-to-end `scripts/verify-docs.sh` run (build + all checks).
9. Catch-up state in Doctor/GetEngineStats (running/failed/last event id).
10. Per-engine catch-up high-water marks (tail replay instead of full journal).
11. AGENTS #22 wording fix (multiSeq restart vs journal monotonicity).
12. routedQuery comment: mention fold-reroute reuse.
13. Fold-reroute test with a Transactional engine (memory isn't; RunInTx + reroute path untested).
14. Reroute-cost caching in dispatchFoldsCoreLocked if bench-justified.
15. Grep `references/modules.md` for stale reset-capability prose (doc-check checks symbols, not prose truth).
16. Release train: strip the two sibling replaces, bump pins, tag wave, smoke (`scripts/tag-release.sh`).
17. Review/absorb the concurrent session's dgraphengine metrics work (unverified by me).

**Standing TODO_LIST items observed while editing it (not re-researched):**
18. Turso: file the upstream silent-wrong-results issue (BLOCKED on user approval).
19. Turso: make grouped-spec safety mechanical (guard/flag decision).
20. Turso: single-source the "verified through vX" citation (9-site whack-a-mole).
21. Turso: `ivm_repro_test.go` full three-defect suite behind `-tags ivmrepro`.
22. Turso: sharpen defect-A characterization before filing.
23. Matview v2 surface (planned-table matviews, DropMaterializedView off-boarding, per-view otel counter, …).
24. Routing integration: teach the cost model matview-covered shapes are O(1).
25. cqrs-lint: loose-heuristic-gate follow-ups (deferred 2026-09-11 batch).
26. cqrs-lint: audit cheap-fix + test-gap tail (doc.go drift, dead branches, boundary tests).
27. Doctor-JSON pre-merge semantics ruling (BLOCKED, user decision).
28. Release-policy Q3: severity tightening in a minor (BLOCKED, user decision).
29. Daemon Q2: `.golangci.yml` formatter exclusion (BLOCKED, user decision).
30. F040: branch protection / required checks (BLOCKED, owner decision).
31. 🔥 350-line limit: 58 offending files — split waves + gate-policy decision.
32. 🔥 iroh standalone pin break (`irohengine/v4.2.0` tag or capability-probe skip-guard).
33. Cut `stack/sqlite/v4.3.1` (broken published pseudo-version pin).

**Larger improvement threads (from this session's observations):**
34. Quarantine-aware replan: after a long quarantine, replan onto the failover engine instead of falling back on reactivation (cost-model honesty).
35. CatchUpEngine option: replay into a NEW engine (zero-downtime engine swap riding the same primitive).
36. ResetResult reporting from CatchUpEngine (today it discards the reset outcome detail).
37. EventLog growth bounds/rotation policy for long-lived catch-up-enabled stores (full-journal replay cost).
38. A "claims ledger" convention for parallel agent sessions (claim-before-write on TODO items).
39. Bench: fold-dispatch hot-path p50/p99 before/after the reroute branch (should be identical when nothing is quarantined — prove it).
40. Test: CatchUpEngine failure mid-replay leaves ZERO partial data visible via rerouted reads (spare copy is complete; engine still quarantined).
41. Test: two engines quarantined simultaneously (reroute target selection under multiple failures).
42. Doc: ADR-0137 implementation-status addendum for write failover + catch-up (the ADR text predates it).
43. Skill: add CatchUpEngine to recipes §2.33 code sample (prose updated, code sample doesn't show it).
44. Doc: `docs/agents/gotchas-testing.md` — note the ginkgo+goleak ignore pattern for future suites.
45. Consider `Store.Reset` docs pointing at CatchUpEngine for the one-engine case (discovery).
46. CHANGELOG: verify the concurrent session logged ITS engine entries; if not, fold into mine before the release train.
47. Check `.art-dupl-baseline.json` implications: five near-identical `reset.go`/`rollbackReturning` copies across dep-isolated engine modules — confirm each carries `//art-dupl:accept` or the gate handles it at next `#check-duplication`.
48. api-stability golden: confirm the concurrent session's engine methods are ALL in it (mine verified; theirs assumed).
49. `nix run .#check-arch` after the go.mod changes (iroh replace + goleak deps are test-only, but prove the budget stays green).
50. Self-check: rerun `cmd/doc-check` + `check-changelog-symbols` after the daemon absorbs EVERYTHING (moving-tree risk).

## g) Questions I cannot answer myself

1. **Does the CatchUpEngine snapshot race block the next tag wave?** I can
   fix it as a fast-follow (loop-until-stable, ~1-2h with a race test), but if
   you want it fixed BEFORE anything ships I'll do it immediately — your
   prioritization call.
2. **Who owns verification of the concurrent session's work** (dgraph metrics
   in flight, mysql/duckdb resets)? I deliberately didn't touch their
   in-flight files; should I run their integration gates once their tree is
   quiet, or does that session close its own loop?
3. **When is the next tag wave?** The release-train note (projectionadapter +
   irohengine replaces, metaengine pin bumps) blocks consumers from using
   `EngineResetter`/`CatchUpEngine` from published versions — timing is a
   scheduling decision only you can make.
