# Tier 1% Execution — Red-to-Green Trust (2026-11-03 21:15 CET)

> **Scope — what this report VERIFIED:** every claim below was verified this session by
> running the actual command or reading the actual output: repro-test red ×5 / green runs
> (`go test -run TestCachedEventStore_CheckSetGapWrite… -count=N`), `-race ./system/...` ×3
> (logs `/tmp/race-run-{1,2,3}.log`, zero `DATA RACE`), cqrs-lint + catalog full suites
> (`GOWORK=off go test -count=1 ./...`), file-size gate (`nix run .#check-file-size`),
> changelog-symbols + versions-manifest + release-script self-tests + example-standalone
> `--build` (0 findings), vet on system, api-stability golden check (6,423 exports),
> proxy probes for both tags (`tag-release.sh --smoke`, attempt-1 serves), remote tag
> existence (`git ls-remote`).
> **What this report did NOT verify:** the composed `#verify` gate (M10's job, needs a
> quiet window), `-race` on catalog/cmd-cqrs-lint (plain suites only — matches the plan's
> pre-tag proof definition), golangci-lint on the changed files (not run — see d6),
> `#check-md-go` (broken since 10-24, M07; today's doc edits are inside that unvalidated
> exposure class), bank-sync's build state (not touched — deferred, see b3), and the two
> doc commits are local-only (not pushed; only the two tags were pushed).

Predecessors this session: done-row sweep (114 rows → CHANGELOG audit, 19:09–19:41
report), Pareto plan ([`2026-11-03_20-04_SUPERB-red-to-green-trust-pareto-plan.md`](../planning/2026-11-03_20-04_SUPERB-red-to-green-trust-pareto-plan.md),
pushed as `be330c7da`). This phase executed the plan's **1% tier (M01–M04)** end-to-end.

---

## a) FULLY DONE

1. **M01 — the `CachedEventStore` stale-read TOCTOU is FIXED.** Root cause: `Load`'s
   generation check and `cache.Set` were not atomic with the write path's invalidations —
   a write completing entirely between check and Set stranded a pre-save snapshot forever.
   Fix: check+`Set` now one critical section (`mu`), `beginWrite`/`endWrite` bump+invalidate
   likewise (`system/cache.go:88-99,111-135`); store reads stay outside the lock.
2. **Deterministic regression test shipped** (`system/cache_gap_internal_test.go`,
   internal package + a nil-default `gapHook` test seam): fails RED in <10 ms against the
   old ordering (`got 4 events, want 5` — the exact `#verify` race-leg shape), green on the
   fix. Verified red ×4 consecutive, then green.
3. **`-race ./system/...` green ×3**, zero data races (F04).
4. **M02 — file-size ratchet confirmed GREEN** with zero work needed: gate says "no new
   offenders, no growth"; the three named offender files were already split by a parallel
   session (273/230/164 lines, none baselined). The plan's offender lists were stale —
   erratum 4 ("re-derive from the gate") proven correct in the wild.
5. **M03 — `cmd/cqrs-lint/v4.19.0` cut, pushed, proxy-served** (attempt 1), clean-dir
   `go install` probe green. Pre-tag proofs: full cqrs-lint suite green (after fixing a
   stale `scratch/counterkey` module-catalog exclusion — one-line fix with reason),
   `check-release-scripts` self-tests green, `check-example-standalone --build` 0 findings.
6. **M04 — `catalog/v4.12.0` cut, pushed, proxy-served.** Inventoried first: 27-file
   `catalog/compat` train (executable schema-compat policy, ADR-0163, complete and
   CHANGELOG-documented since 11-02 — finished work awaiting a tag, NOT in-flight
   parallel work) + `Registry.MarkDemo`. Tagged from a detached worktree because a
   parallel session held the main tree dirty (sanctioned gotcha-mechanic 2b).
7. **mesh-demo healed — the incident's exact repro closes.** Pin v4.11.0 → v4.12.0,
   temporary relative replace + its go.mod comment deleted, standalone allowance removed;
   `GOWORK=off go build` + test suite green **against the proxy-served module** (the build
   literally downloaded catalog v4.12.0 from proxy.golang.org), `check-example-standalone
   --build` green with mesh-demo proxy-pure.
8. **CHANGELOG wave sections written for both tags** (`v4.19.0 satellite-SQL advisory
   wave`, `catalog/v4.12.0 executable schema-compat wave`): moved E023, C044-hardening,
   explain-IDs, F032, and the compat row out of `[Unreleased]`; wrote the missing
   **MarkDemo** bullet (coverage gap — it had NO changelog entry at all) and the mesh-demo
   heal as the catalog wave's Fixed row. `check-changelog-symbols` green after every move;
   `versions.json` manifest refreshed + `--check` green.
9. **TODO_LIST truth updates:** TOCTOU 🔴 BLOCKING row deleted (→ CHANGELOG Fixed), E023
   release row rewritten (tag half done, bank-sync half pending with the library-lineage
   migration rationale), MarkDemo train row deleted, the "gate examplePaths" row rewritten
   to its true remainder (wire the existing `--build` leg into CI — the script is wired
   NOWHERE, discovered this session).
10. **Commits:** `d7c014550` (E023 row state), `5fb0d1a46` (catalog wave receipts); both
    tags verified on origin via `git ls-remote`.

## b) PARTIALLY DONE

1. **M03's consumer half — bank-sync pin bump.** Deferred deliberately: bank-sync's
   `go-cqrs-lite` flake input serves the **library lineage** (verified in flake.nix
   comments: the `mkPreparedSource` replace compiles bank-sync against the pin tree), so
   `d15e0a48` → `6de79f5997f8` is a three-week API migration (vendorHash dance, coherence
   `--strict`, possible compile fixes), not a tool bump — AND a parallel session held
   uncommitted bank-sync changes (`cmd/bank-sync/import.go` +2 files) at snapshot time.
   Recorded on the row with effort re-graded S→M.
2. **M02's bookkeeping tail.** The gate is green (the actual goal), but the TODO_LIST rows
   that demand offender work (L794/L810/L1538/L2145 region) were NOT closed — they are now
   stale ("offenders exist" is false). F10 (`adttest/pagination_conformance.go` harness-dir
   exemption, owner's call) skipped entirely, not even rowed as a question.
3. **M04's gate-class follow-up.** The MarkDemo *incident* is healed, but the *class*
   stays ungated: `check-example-standalone.sh` is referenced by no workflow, no flake app,
   no `#verify` leg. Re-rowed as "wire `--build` into CI" (Effort S) rather than done.
4. **M01's composed-proof share.** Module `-race` ×3 done per F04; the un-forced
   **composed `#verify` re-record** belongs to M10 and is untouched.
5. **Push state.** Tags are on origin; the doc commits (`d7c014550`, `5fb0d1a46` + several
   daemon commits) are local-only. Consistent with push rules (only the plan's tag pushes
   were authorized) but origin's master trails what the CHANGELOG wave sections describe.

## c) NOT STARTED

1. **4% tier (M05–M11):** hygiene gate (M05), duplication campaign (M06, ~190 groups),
   md-go FOD repair + nix-hash guard extension (M07 — `#check-md-go` still un-runnable,
   every doc change since 10-24 including today's unvalidated), lint-config guards (M08),
   fleet go.sum sweep (M09), `#verify-serial` + composed re-record (M10), stale-premise
   sweep + detector (M11).
2. **20% tier (M12–M21):** json/v2 determinism (KEYS first), idempotency-retirement ADR,
   cqrs-lint consumer-fix pack, Wave III tag trains, arch quick wins, A9 keycodec pebble
   leg, quiet-window batch, generator truth fixes, cursor issuance protocol, docs/recipes
   truth pack.
3. **80% tier (M22–M27 + F130–F139):** EventCatalog pack, consumer asks, metaengine tails,
   BDD/graph tails, spike kickoffs, v5 kickoff; all standing/blocked clusters.
4. **This report's own HARVEST** — section (f) items that are NOT already plan rows need
   routing into TODO_LIST before session end (the skill's closing-loop rule; explicitly
   deferred here because the instruction is report-then-wait).

## d) TOTALLY FUCKED UP

1. **The repro-test handshake took three failed rounds (~12 min wasted).** Round 1:
   double-`close(atGap)` panic (hook ran twice from an interleaving I hadn't simulated).
   Round 2: 10-MINUTE test timeout — I double-received on the buffered `writeDone` channel
   (select consumed it, then `<-writeDone` waited forever). Round 3: correct. Root cause: I
   wrote channel choreography from a mental model instead of simulating BOTH interleavings
   (buggy-ordering and fixed-ordering) on paper first. The production fix itself was right
   on the first attempt — all flailing was in my own test scaffolding.
2. **FEATURES.md was never updated for the wave.** The close-a-wave rule is explicit
   (CHANGELOG row + FEATURES.md upsert at session end); I did CHANGELOG and TODO_LIST but
   NOT the FEATURES.md row for `catalog/compat` + `MarkDemo`. Same drift class the
   2026-10-10 lesson warned about, committed fresh today.
3. **F032 nearly shipped out of its wave section.** A parallel session added F032 to
   `[Unreleased]` between my wave-section edit and the tag cut; the tag carries F032
   (verified: `git show cmd/cqrs-lint/v4.19.0^{commit}:…/f032.go` exists). I caught it by
   LUCK while scanning Unreleased for the catalog pass — not by any systematic post-tag
   reconciliation step. One hour of timing difference and CHANGELOG would have lied about
   v4.19.0's content.
4. **M02's stale rows left open** (see b2) — the exact "stale row" class this session's
   own sweep phase (19:09) had just cleaned 114 of.
5. **`rg -rn` hook violation THREE TIMES** — the same replace-flag trap, the hook's own
   message says third strike on this host. Also one malformed `sed -e` and one `rg -h`
   misuse (help flag read as no-filename). Tool-call hygiene regression under pace.
6. **Edit-precision slips:** the `])` vs `)` anchor miss on mesh-demo/go.mod; an "edit"
   that only appended a newline instead of removing the replace block; a stray
   placeholder edit accidentally included in a multiedit batch (failed harmlessly — but it
   should never have been in the batch).
7. **golangci-lint never ran on the changed files** (system's cache.go + new test,
   cmd/cqrs-lint's catalog test, mesh-demo go.mod). Pre-commit ran treefmt (clean) and
   `go vet` passed on system, but the lint gate for the changed set is unproven.
8. **Detached worktree left dirty** (needed `--force` to remove — the tagger's restore
   state). No damage, sloppy cleanup.
9. **The 1-second timeout tax.** The gap test burns a full 1 s on EVERY green run (fixed
   path detection), paid by every future `-race`/CI run of the system module forever. A
   deliberate determinism-vs-latency trade I made without flagging it for review.

## e) WHAT WE SHOULD IMPROVE

1. **Simulate concurrency handshakes before writing them** — for channel-orchestrated
   tests, write both interleavings' step tables (who holds what, which channel fires)
   before typing code. Would have saved the entire d1 flail.
2. **Make post-tag CHANGELOG reconciliation a step, not luck** — after any tag cut,
   systematically re-scan `[Unreleased]` for rows the tag now carries (candidate:
   extend `tag-release.sh --smoke` or `check-changelog-symbols.sh` with a
   tag-content-vs-wave-section diff; goes in (f) #32).
3. **Close stale rows the moment their premise dies** — M02 proved gate-green; its rows
   should have been closed in the same hour (docs-health discipline, same as the sweep).
4. **Wire what you rely on** — the example-standalone gate existed for weeks as a
   manual-only script; "a gate that nobody runs" is a ledger entry, not a gate.
5. **Blast-radius lint step** — add "golangci-lint on changed modules" to the personal
   done-checklist; vet + treefmt is not the lint gate.
6. **Tool-call discipline under pace** — the `-rn` trap struck three times because I kept
   typing grep habits; slow down on flags, not on thinking.
7. **Test-latency budgets deserve an explicit note in the test's doc comment** when a
   timeout is load-bearing for correctness detection — so the next reader knows why 1 s
   and what breaks if tuned.

## f) 50 things to get done next

> Brainstorm, not commitment (skill rule). Items marked 📋 are already plan rows
> (M/F numbering from [`the Pareto plan`](../planning/2026-11-03_20-04_SUPERB-red-to-green-trust-pareto-plan.md))
> — they need NO new TODO_LIST row; unmarked items are session finds that DO need
> HARVEST routing with the extra brainstorm rigor.

1. 📋 M05 — build `check-todo-list-hygiene` gate + self-tests + wire (F22–F25).
2. 📋 M06 — duplication-campaign decision + execution (~180 mirror twins + stragglers, F26–F30).
3. 📋 M07 — repair `#check-md-go` FOD; flake.lock-coherence check; extend `check-nix-hashes` to all `*.nix` (F31–F34).
4. 📋 M08 — `.buildflow.yml` hash-golden tripwire + lint-config canaries (F35–F37).
5. 📋 M09 — fleet go.sum tidy sweep + sqliteengine pseudo-version pin (F38–F39).
6. 📋 M10 — build `#verify-serial` tier, then ONE composed `#verify` re-record on quiet window (F40–F43).
7. 📋 M11 — stale-premise sweep (L512 etc.) + cqrs-htmx battery + "blocked on tag X" detector (F44–F47).
8. Close M02's stale file-size rows (L794/L810/L1538/L2145 region) — premise dead, gate green.
9. Row + decide F10: `adttest/pagination_conformance.go` harness-dir exemption (owner's call).
10. bank-sync pin bump + `cqrs-lint-disable-audit` re-verification (blocked; see g1).
11. 📋 M12 — json/v2 determinism wave, KEYS first (F48–F53).
12. 📋 M13 — idempotency-retirement ADR (F54–F57).
13. 📋 M14 — cqrs-lint consumer-fix pack: #79 fixture, `--show-suppressed`, #83 hardening, 2.0s→7.4s perf (F58–F64).
14. 📋 M15 — Wave III delta-audit tag trains (F65–F68).
15. 📋 M16 — arch P0/P1 quick wins: memoryengine extraction + DOMAIN_LANGUAGE seams (F69–F73).
16. 📋 M17 — A9 keycodec-v2 pebble reference leg (F74–F78).
17. 📋 M18 — calibration/quiet-window batch (F79–F85).
18. 📋 M19 — taxonomy-generator sentinel fix + `core/v5/.go-arch-lint.yml` (F86–F89).
19. 📋 M20 — compound-cursor issuance protocol (F90–F93).
20. 📋 M21 — docs/recipes truth pack (F94–F99).
21. 📋 M22 — EventCatalog tail pack (F100–F103).
22. 📋 M23 — consumer-ask pack (F104–F108).
23. 📋 M24 — metaengine capability tails (F109–F116).
24. 📋 M25 — BDD harness + graph tails (F117–F122).
25. 📋 M26 — spike kickoffs: DuckLake pins, journal dump⇄load DESIGN, AsyncAPI response schemas (F123–F125).
26. 📋 M27 — v5 kickoff: delete-at-zero memo, deriver-ABSORB row, RawQuery→DeclaredQuery draft, `cqrs-upgrade` scaffold (F126–F129).
27. **FEATURES.md row for the catalog/v4.12.0 wave** (compat + MarkDemo) — the d2 miss, close it.
28. **Wire `check-example-standalone --build` into CI** — row already rewritten today (Effort S); this is the incident-class kill.
29. **Triage GitHub dependabot's 2 high vulnerabilities** on the default branch (noticed in the 19:09 phase, still unaddressed).
30. **Push the local doc commits** (`d7c014550`, `5fb0d1a46` + daemon commits) so origin's master matches the CHANGELOG wave sections the tags reference.
31. **Run golangci-lint on today's changed modules** (system, cmd/cqrs-lint, example/mesh-demo) — close d7.
32. **Post-tag reconciliation step** — extend a gate to diff wave sections vs tag content (the d3 process fix; `check-changelog-symbols` neighbor).
33. **Trim or document the gap test's 1 s timeout** (d9) — shorten to ~200 ms with margin data, or `//go-sleep:accept`-style annotation.
34. **Update the Pareto plan's status header** — mark the 1% tier executed with today's receipt (plans get dated banners, not silent edits).
35. **Annotate the 19:41 status report** — its three owner questions are partially answered by this phase (docs-health ANNOTATE).
36. **docs-health HARVEST of this report** — route unmarked (f) items into TODO_LIST/ROADMAP before session end.
37. **AGENTS.md procedure update** — add "post-tag Unreleased re-scan" + "detached-worktree tag flow" notes to the release gotchas (memory maintenance for the next session).
38. **Fleet census re-run** after today's tags (timestamps in `docs/evidence/fleet-census-*.json` are pre-wave; bank-sync still pins pre-E023 — the zero-consumer math should be re-checked for cqrs-lint consumers).
39. **Pre-compute bank-sync's migration surface offline** — `git diff d15e0a48..6de79f599 --stat` filtered to bank-sync's vendored modules, so the bump window (g1) is minutes not hours.
40. **Nightly-gate watch for the two new tags** — `check-versions-manifest --check` is green locally; confirm tonight's nightly agrees (fresh-manifest freshness gate).
41. **Decide `scratch/counterkey`'s fate** — a parallel session put it in go.work (L75) and I excluded it from the linter catalog; scratch modules should not live in go.work indefinitely.
42. **CHANGELOG `[Unreleased]` hygiene** — ~20 infra/testing bullets there ride no module tag; decide their recording home (root-level wave section extension or leave-until-v5).
43. **`#check-md-go` exposure note** — until M07 lands, add a one-line warning to the close-a-wave procedure: doc edits are parse-unvalidated.
44. **Gap-hook seam polish** — consider a build-tag-internal constructor instead of a struct field (design taste; zero behavior change).
45. **Verify the two tags from a /tmp scratch consumer** (the 2026-09-28 lesson: install probe alone proves compile, a scratch module proves symbol resolution end-to-end — worth doing once for MarkDemo + E023 since both are consumer-visible).
46. **cqrs-htmx + go-appkit v5 pin sweep check** — today's two tags may unblock companion-repo bumps rowed in the v5-transition model (dq: which companions track catalog/cqrs-lint trains).
47. **`example/taskmanager` systemcatalog wiring** — F032's field-proof follow-up, already rowed by its session; candidate for M14's pack.
48. **Time-budget check of the gap test under `-race` in CI** — confirm the +1 s/test is invisible in the system module's runtime budget (`check-test-runtime` baseline: system's package budget may need `--update` if the suite crossed tolerance).
49. **Pair M05's hygiene gate with a docs-health pass** — one gate + one sweep closes the 114-row class permanently.
50. **Owner-question follow-ups from the 19:41 report** — its (g) section asked three things; fold any answers into M06/M10/M11 decisions.

## g) Three questions I cannot figure out myself

1. **bank-sync window:** a parallel session held uncommitted changes in bank-sync
   (`import.go`, `import_wise_statement.go`, `aggregate_mapping.go`) when I deferred the
   pin bump. Is that session still live, and do you want the `d15e0a48` → `v4.19.0`
   library-lineage migration done in a specific window (it will touch vendorHashes and
   possibly need compile fixes across three weeks of go-cqrs-lite API drift)?
2. **M06's ONE decision:** the ~180 ADR-0152 mirror-twin duplication groups — do you want
   the mechanical per-twin trailing `//art-dupl:accept` directives (keeps the baseline
   frozen, documents each twin), or ONE end-of-campaign full `art-dupl baseline` re-pin
   after the v5 copy-forward twins die (riskier mid-campaign, cleaner end-state)? The row
   says one decision is owed; the campaign's shape follows from your call.
3. **Composed proof strategy:** M10's honest `#verify` re-record needs load < 10 on this
   shared box and ~30–60 min of exclusive runtime. Build `#verify-serial` FIRST and run it
   regardless of load (proof tonight, slower but deterministic), or wait for a natural
   quiet window tonight/tomorrow for the canonical composed run (matches the gate's
   intended semantics, timing uncertain)?
