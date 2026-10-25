# Turso IVM rewrite verified at source + AVG integer-division fix — 2026-10-25 16:07

Session trigger: "Turso merged some new materialization engine/views; does this fix all our bugs?"
Scope: upstream PR [#10129](https://github.com/tursodatabase/turso/pull/10129) verification, the latent AVG bug it exposed, and the bookkeeping fallout. All claims below were produced this session unless marked historical.

## Headline

Upstream's materialization-engine rewrite (merged 2026-10-19, **in NO tagged release yet**) was verified from source: all three tracked IVM defects are gone. The verification run exposed one latent bug on OUR side (AVG integer division), now fixed. The production pin stays v0.8.2 (defects still live there, by design, until a tagged engine ships).

## a) FULLY DONE

1. **Upstream research, verified first-hand** — PR #10129 body read directly (not secondhand): fixes 18 issues incl. #9391, supersedes #9392 AND #8257; view queries now run through the normal planner/VDBE.
2. **Release-blocker established** — the engine reaches Go only via `turso-go-platform-libs`; newest tag v0.9.0-pre.1 (2026-10-15) predates the merge; platform-libs `main` stale at 2025-12-11. Verified via `go list -m -versions` + repo API + tarball.
3. **Exploratory bindings@main pin** — run and REVERTED clean; proved it silently selects platform-libs v0.8.2 (max-version rule) = old engine, i.e. an inconclusive-by-construction trap for anyone trying the same.
4. **Engine built from source** — core `aa93d669` tarball via `gh api`, `cargo build --profile lib-release --package turso_sync_sdk_kit` (2m19s, gcc 16.2, no clang needed). Tree at `/tmp/turso-engine-verify` (~780 MB, until reboot).
5. **New test seam** — `metaengine/tursoengine/library_strategy_test.go`: `TURSO_TEST_SYSTEM_LIB=1` TestMain pre-inits `InitLibrary` with the SYSTEM strategy (`LD_LIBRARY_PATH`); the driver's later embedded-init calls no-op through `sync.Once`. Negative control (loud panic when lib missing), no-op control (env unset = identical behavior), `go vet` clean.
6. **Defect verification on the rewrite** — A: did NOT reproduce (grouped total exact at 95890.00, the 430.50 signature is gone); B: exact through 27k rows; C: no COMMIT abort through 30k rows. Enforce-mode `TestTursoMatView_GroupedSumDefectAEnvelopeGuard` PASSES.
7. **Latent OUR-side bug found + fixed** — AVG matview serving used bare `agg / cnt`: integer division under real SQLite semantics (both columns INTEGER-typed; `json_extract` yields INTEGER for integral JSON numbers). The old engine's private mini-SQL divided as REAL and masked it for every prior run. Fixed with REAL promotion (`agg * 1.0 / cnt`) in all five serving/EXPLAIN sites of `metaengine/sqliteengine/materialized_view.go`.
8. **Test battery** — full tursoengine package green on BOTH engines (new: 62s / pinned: 30s); pinned `-tags ivmrepro` still green (defects live on the pin, as documented); sqliteengine package green; `go vet` both modules clean.
9. **Gates run** — `check-file-size` ✓, `check-turso-version` ✓ (incl. pin-scan leg), `check-changelog-symbols` ✓ (after it caught my own bad citation — see d), `check-arch` ✓ (dep budgets incl. tursoengine=5 pass with platform-libs now a direct test dep).
10. **Docs/bookkeeping** — CHANGELOG `[Unreleased]`: Added (the seam) + Fixed (AVG). TODO_LIST turso section: two dated receipts, A+B filing row CLOSED MOOT, characterization row CLOSED MOOT, new `[BLOCKED on upstream release]` flip-watch row, routing-row premise updated. `#9391` verification comment drafted + voice-checked at `docs/drafts/2026-10-25_turso-9391-verification-comment.md` — NOT posted (external action, needs approval).
11. **Upstream state confirmed** — #9391 auto-closed by the merge (+9s); #8257 closed superseded; #9392 open (upstream's to close). Our repo has zero turso-related issues.

## b) PARTIALLY DONE

1. **"#10129 fixes our bugs" verification** — A/B/C verified; the #9391-specific zombie-tx READBACK (aborted chunk readable / persistence after reopen / v0.7.2 reopen crash) was NOT directly re-tested — it has no test in our suite (it was a standalone `database/sql` repro in the issue). The defect-C family covers the commit-abort half only.
2. **The #9391 comment draft** — written and voice-checked but carries a misattribution: "the repro suite from this issue" implies I ran #9391's own zombie repro; I ran OUR ivmrepro suite. Must be reworded (or the zombie repro actually run) before posting. See d).
3. **Flip** — everything except the trigger: runbook steps 1–6 fully specified in the flip-watch row; blocked on a platform-libs tag.

## c) NOT STARTED (deliberately deferred)

1. Pin bump / citation bump (`TursoGoIVMVerifiedThrough`) — waits for the tag.
2. Doctor grouped-view WARN removal (runbook step 3).
3. Enforce-guard un-skip, grouped-bench un-skip + `#bench` sweep (steps 2, 5).
4. Drop-and-recreate migration note for long-lived DB files with old-engine views (our `IF NOT EXISTS` will not recreate them).
5. `#verify` full chain (exclusive/heavy; not run this session — targeted gates + per-module suites only).
6. `#check-duplication` (new file + string edits judged low-risk; gate not run).
7. `cmd/doc-check` (no skill-reference fences touched; scan set unchanged).
8. Zombie-tx readback + reopen-crash re-verification (TODO row 997 lists it for flip time).
9. MVCC-mode / `sqlite_sequence` `changes()` / NOCASE-smallest-rule interactions of the rewrite with our surface — unexamined, none expected on our DDL shape.

## d) TOTALLY FUCKED UP (own goals, all recovered)

1. **First-round fatalism** — I answered "impossible to test today; no engine build exists" and stopped at the tag boundary. The system-strategy + build-from-source path was available all along (~30 min total). User push ("Tested all?") was required to find it. The lesson: "no RELEASED artifact" ≠ "cannot verify".
2. **Probe-before-understanding** — ran defect A on bindings@main BEFORE checking which platform-libs it resolves to (one `cat` of the `.mod` would have shown v0.8.2 old-engine = pointless run). Order of operations failure.
3. **Draft overclaim** — the #9391 comment draft says "repro suite from this issue" (we ran OUR suite, not the issue's zombie repro). A verify-before-filing discipline slip in the prose, caught in self-review, unfixed pending instruction.
4. **Changelog symbol citation** — cited `tursogo.InitLibrary` (upstream symbol) in the Added entry; `check-changelog-symbols` correctly FICTION-flagged it; fixed to "the driver's `InitLibrary`".
5. **Daemon noise** — the exploratory pseudo-version pin was absorbed by two `chore: auto-commit` commits before revert; history carries the detour (harmless, go.sum verified clean after).
6. **Missed gate on first pass** — `check-arch` was NOT in my original gate run; only the self-review flagged that platform-libs had become a DIRECT dep in tursoengine's go.mod (budget risk). Run retroactively: passes.

## e) WHAT TO IMPROVE

1. **Escalate verification ladders earlier** — tagged release → any tag → branch pseudo-version → source build. Walk the whole ladder BEFORE declaring "cannot verify".
2. **Pre-flight the artifact identity** — before ANY engine test, confirm which engine binary the test actually loads (version constant, module resolution, or sha). The @main probe would have been skipped.
3. **Regression pins must be engine-independent** — the AVG fix is protected by the property test ONLY on new-engine semantics; the pinned engine cannot catch a regression (it float-divides). Add a serving-SQL string pin (e.g. via `matViewExplainSQL`) so the `* 1.0` survives without needing the rewrite.
4. **Runbook should teach the seam** — `docs/turso-go-ivm-fix-flip-runbook.md` §1 still only knows the pin-bump check; the `TURSO_TEST_SYSTEM_LIB` pre-tag check belongs there (and in gotchas-tooling-build).
5. **Draft comments: verify the ATTRIBUTION, not just the claim** — every noun in an external draft must be traceable to what was actually run.

## f) NEXT (prioritized, ~30 real items)

**Flip window (trigger: platform-libs tag ≥ v0.9.0):**
1. Fix the #9391 draft attribution, then post on approval
2. Run the actual zombie-tx readback repro from #9391 against the tagged engine
3. Re-verify the v0.7.2 reopen-crash (`API misuse: unknown database open flags`) is gone
4. Pin catch-up (`check-turso-version` pin-scan forces the constant bump in the same change)
5. Runbook steps 1–6 in ONE change: repro suite flip signals → enforce-guard un-skip → Doctor WARN removal → citation sweep → grouped-bench un-skip → api-stability regen + CHANGELOG
6. Drop-and-recreate migration note; consider engine-side old-view detection
7. Full `nix run .#bench` sweep on the turso backend before relaxing seed guards
8. Re-run the A+B onset bisect on the tagged build (expect empty divergence)
9. Decide `WithKnownGroupedViewBug` / `ErrGroupedViewBugRefused` fate (deprecate-till-v5 vs immediate removal)
10. Re-check recursive-CTE probe pin on the tagged engine

**Session fallout (now):**
11. Add the engine-independent AVG REAL-division pin test
12. Run `nix run .#verify` over the session's changes
13. Run `nix run .#check-duplication`
14. Add the seam to the flip runbook §1 + gotchas-tooling-build turso block
15. Re-confirm go.mod/go.sum end state on both touched modules (daemon churn)
16. Confirm no CI env sets `TURSO_TEST_SYSTEM_LIB` (should be never)

**Matview v2 surface (existing backlog, unblocked-at-flip):**
17. Grouped routing O(groups) pricing — implement the `AggregateOn` seam (one-pager exists)
18. Planned-table matviews (ordered + backfill via `ApplyLayout`)
19. `DropMaterializedView` off-boarding
20. Per-view IVM write-amp otel counter
21. `system.Introspection()` matview surface
22. cqrs-lint: matview-on-unsupported-driver rule
23. cqrs-lint: matview-plus-planned-table staleness-trap rule
24. `example/materialized-views/`
25. Filtered-view spec variants; multi-aggregate/DISTINCT serving
26. Document the rewrite's reject-list (index-on-view, NOT IN, ORDER BY/LIMIT, windows, recursive CTEs, group_concat) as spec-validation refusals in `MaterializedViewSpec` docs
27. Decide whether the seam generalizes (e.g. a COOKBOOK recipe for consumers pre-verifying engines)

**Watch items:**
28. platform-libs tag feed (v0.9.0-pre.2 / v0.9.0)
29. #9392 upstream closure (their housekeeping)
30. /tmp/turso-engine-verify RAM footprint until reboot (~780 MB tmpfs)

## g) QUESTIONS (cannot resolve myself)

1. **Post the #9391 comment?** After I fix the attribution: post now (source-level verification, pre-release signal to maintainers), wait for the tagged release, or skip commenting entirely?
2. **AVG fix + seam: ride as-is or full-verify first?** I ran targeted gates + both engine suites but NOT the full `#verify` chain — want it green before the next tag wave absorbs these changes, or is the targeted evidence sufficient?
3. **Grouped-view guard fate at flip:** remove `WithKnownGroupedViewBug`/`ErrGroupedViewBugRefused` immediately (breaking) or standard deprecate-till-v5?

— Session receipts: TODO_LIST "Turso materialized views (ADR-0135)" section (4 rows + 5 receipts); CHANGELOG `[Unreleased]` Added + Fixed; `docs/drafts/2026-10-25_turso-9391-verification-comment.md`.
