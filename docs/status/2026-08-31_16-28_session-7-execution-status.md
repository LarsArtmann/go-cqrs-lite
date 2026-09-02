# Session 7 Execution Status — Pareto Plan → D3/D8 Close-Out (2026-08-31 16-28)

Honest close-out for the session-7 run (2026-08-30 ~22:00 → 2026-08-31 ~16:30,
with the report written the next afternoon). Baseline: session-5/6 state at
`2e8f2281c`, tree clean, everything pushed. The user directive was: build a
comprehensive ≤12-min-task plan covering ALL open TODOs, report it as a table,
then execute the whole list. Plan committed as
`docs/planning/2026-08-30_22-15-session7-execution-plan.html` (`ad77f5223`,
39 tasks / 4 Pareto tiers). Execution followed it top-down.

Push state at report time: origin/master is at `ad09ec75d` (a concurrent
push/daemon moved it past my early commits); **4 commits unpushed**
(`140409aec..8e7dcddfe`) plus **3 uncommitted files** — see §d.4.

## a) FULLY DONE (verified, committed, gated)

| Task                                                                                                        | Commit(s)                                    | Evidence                                                                                                                                                                                                                                                                                               |
| ----------------------------------------------------------------------------------------------------------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Format the 2 calibration test files that blocked the verify lint gate                                       | `cdc51f488`                                  | golines+nlreturn fixed; module lint 0 issues on both                                                                                                                                                                                                                                                   |
| Session-7 Pareto plan (39 ≤12-min tasks, d2 graph, exclusion list)                                          | `ad77f5223`                                  | HTML report committed; table reported back to the user                                                                                                                                                                                                                                                 |
| CHANGELOG: planned-pushdown completion, RenewLease, float64→DOUBLE fix                                      | `98ffa7bcb`                                  | symbol gate EXIT=0 (129 citations)                                                                                                                                                                                                                                                                     |
| duckdb planned-path float investigation → verified safe by construction; added the missing float-filter pin | `79f7ef7cd`                                  | CGo test suite green; §f 19 closed as verified-no-bug                                                                                                                                                                                                                                                  |
| sqliteengine EXPLAIN QUERY PLAN proof for the planned path (also covers the index-name-drift pin)           | `ca15e925f`                                  | asserts index-backed plan, no bare SCAN, meta_planned_* target                                                                                                                                                                                                                                         |
| Planned-vs-meta_map bench evidence (T07)                                                                    | `558b2d121`                                  | live PG: filtered scan 873.8µs (meta_map) vs 778.9µs (planned); CounterGet equal (287.4 vs 260.0µs) → counters-stay-on-meta_map decision now has numbers                                                                                                                                               |
| `LayoutPlanEvolver` core interface + pg information_schema evolution                                        | `558b2d121`                                  | ADD COLUMN IF NOT EXISTS, ALTER COLUMN TYPE .. USING (42804 needs the explicit cast), applied-actions list                                                                                                                                                                                             |
| mysql evolution + live tests both engines                                                                   | `396b2639e`                                  | information_schema DATABASE()-scoped; MODIFY COLUMN; idempotency + numeric-predicate-over-evolved-column tests green on ephemeral PG + userspace MariaDB                                                                                                                                               |
| Opt-in `BackfillPlannedCollection` + `KeyScanBackend` capability                                            | `6c710e478`                                  | meta_map-direct paged key+value reads; idempotent upsert backfill; live tests on PG + MariaDB; art-dupl accepts (gate green, baseline 131)                                                                                                                                                             |
| Doctor `--- Planned tables ---` section + `PlannedTablesReporter`                                           | `ad09ec75d`                                  | pg+mysql implementations; live row counts; core test pins the no-reporter rendering; api golden 4365→4368 exports                                                                                                                                                                                      |
| `EffectiveDurability()` on badger/bbolt/pebble/sqlite/pg                                                    | `9c32ccce2`                                  | state-derived (sync flags, PRAGMA, DSN tier); registration tests pin the tier round trip incl. live PG; **caught the unpublished-symbol replace class 3 more times** (4 go.mods gained the sibling replace)                                                                                            |
| `RunPlannedOpsMatrix` — D3 slice 4 parity harness (adttest)                                                 | `140409aec`                                  | no-backfill invisibility, backfill, planned visibility, pushdown agreement, MapUpdate, evolution — each sub-capability interface-gated; wired into sqlite/pg/mysql/duckdb; **caught a real duckdb divergence** (MapScan leaked meta_map rows into planned scans) and the fix landed in the same commit |
| Claiming metrics hooks (T17/T18) — option (a) zero-dep                                                      | `84ccb8549`                                  | `ClaimMetrics{Claimed, Renewed, RenewRejected}` + variadic `ClaimOption`; nil-safe; tests pin all three hooks                                                                                                                                                                                          |
| RenewLease renew-vs-claim race test (T19)                                                                   | `84ccb8549`                                  | live PG via ephemeral-pg loop, `-race`, EXIT=0: pollers can't steal a renewing holder's timer; exactly one reclaimer converges after lapse                                                                                                                                                             |
| Claim-token ownership ADR stub (T20)                                                                        | `84ccb8549`                                  | `docs/adr/0134-claim-token-ownership.md` (proposed; additive-overload path sketched)                                                                                                                                                                                                                   |
| Timer-store slot decision (T16)                                                                             | `84ccb8549` (TODO)                           | adttest/enginetest are metaengine-engine harnesses; claiming is a scheduling decorator — tests stay in scheduling/sqlstore; ledger row closed with rationale                                                                                                                                           |
| Docs batch (T21–T26)                                                                                        | `4285defbb`                                  | recipe 2.28 (visibility, backfill, evolution, EXPLAIN proof, measured numbers), modules.md metaengine+duckdb rows, SKILL.md decision-table row, ADR-0124 EXPLAIN-proof requirement, faq planned-visibility entry, pg/mysql README cost tables; **doc-check GREEN, 956 refs / 42 packages**             |
| dgraph health wait made tunable (T27)                                                                       | `8e7dcddfe`                                  | `DGRAPH_HEALTH_TIMEOUT` env, default 60s unchanged; bash -n clean                                                                                                                                                                                                                                      |
| `-shuffle=on` evaluation (T28) — **ADOPT verdict**                                                          | `8e7dcddfe`                                  | first shuffled MariaDB run caught a real order dependence (fixed with `adttest.Factory.PreClean`); green over 2 seeds mysql + sqlite + duckdb targeted suites                                                                                                                                          |
| `SOAK_SKIP_DGRAPH=1` escape hatch + 52s-vs-15min discrepancy documented (T29)                               | `8e7dcddfe` + uncommitted test change (§d.4) | AGENTS Testing section updated                                                                                                                                                                                                                                                                         |
| LSP cache wrapper (T30)                                                                                     | `8e7dcddfe`                                  | `scripts/golangci-lint-lsp-wrapper.sh` force-pins the disk cache; symlinked into ~/.local/bin; crush.json points at it (kills the 150+ phantom diagnostics)                                                                                                                                            |
| nix `#lint-module` app (T31)                                                                                | `8e7dcddfe`                                  | smoke-tested: `nix run .#lint-module -- metaengine/pgengine` → 0 issues                                                                                                                                                                                                                                |
| AGENTS concurrent-session convention (T32)                                                                  | `8e7dcddfe`                                  | file-ownership + shared-ledger re-read rules                                                                                                                                                                                                                                                           |
| Turso CTE-probe test (T34) — **genuine finding**                                                            | uncommitted (§d.4)                           | the turso (libSQL) driver REJECTS recursive CTEs → the construction probe correctly degrades graph traversal to iterative BFS; both the probe-failure and degraded-path contracts now pinned (`cte_probe_test.go`, 2 tests PASS, lint clean)                                                           |
| Post-landing sweep, partial (T33)                                                                           | —                                            | api-stability meta-tests (TestEvery*) green; doc-check green; GOWORK=off standalone builds of 4 consumer modules clean                                                                                                                                                                                 |
| Hygiene closes verified earlier                                                                             | TODO                                         | `.gitignore /t/` confirmed, `.gotmp` has no >7d logs, gh auth VERIFIED working (LarsArtmann, repo scope)                                                                                                                                                                                               |

## b) PARTIALLY DONE

1. **T33 post-landing sweep** — meta-tests + doc-check + pin builds ran; the
   explicit consumer-pin sweep for `record/v4` under GOWORK=off (the original
   ledger item's full shape) was not separately enumerated (covered only for
   4 ad-hoc consumer modules).
2. **Session work committed but the ledger (TODO_LIST) not updated** — the
   session-7 completions (evolution, backfill, Doctor sections, durability,
   parity matrix, duckdb fix, metrics, shuffle adoption, turso finding) are
   still open rows in TODO_LIST.md; only the claiming metrics + timer-slot
   rows were closed.
3. **CHANGELOG coverage of session-7 APIs** — ClaimMetrics has an entry; the
   newer exported surface (`LayoutPlanEvolver`, `KeyScanBackend` +
   `BackfillPlannedCollection`, `PlannedTablesReporter`, `EffectiveDurability`
   implementations, duckdb MapScan visibility fix) is NOT yet in
   [Unreleased]; the symbol gate therefore passes vacuously for them.
4. **Plan HTML statuses** — the committed plan file still shows all tasks
   pending; no post-execution annotation.
5. **Verify-adjacent gates** — `nix fmt` full-repo convergence not run (files
   were formatted with targeted gofumpt/golines); the doc-assertions gate not
   run standalone (it runs inside `#verify`).

## c) NOT STARTED

1. **T35 `.golangci.yml` exclusion audit** — I had just located the three
   broad exclusion blocks (system/ line 466, metaengine/ line 487,
   cmd/cqrs-lint nearby) when the user interrupted; no analysis or edits made.
2. **T36 load-sweep** — not run (required BEFORE `#verify` after this
   session's timing-path adjacencies: planned-scan bench, matrix harness).
3. **T37 full `nix run .#verify`** — not run. Expected GREEN (the two lint
   findings that failed the last run were fixed at session start), but
   unproven — a claim of GREEN would be stale-green fraud.
4. **T38 push** — 4 commits unpushed (see §d.4).
5. **T39 GitHub Releases** — script exists, gh auth verified, zero releases
   created.
6. WS-M remainder (ReadAggregate G1 recalibration for pg/mysql constants) —
   untouched by design (concurrent session's workstream).
7. cqrs-lint ApplyLayout→ApplyLayoutPlan rule — still design-gated.
8. Everything user-gated (tag wave, v5 deletions, billing, macOS, nspawn,
   go-codec push, dead tags) — untouched by design.

## d) TOTALLY FUCKED UP

1. **Committed with a red gate — again** (`558b2d121`): the pgengine lint
   showed `contextcheck: 1` in the SAME command output as the commit; I saw
   it only after. Fixed in the next commit, but this is the exact
   gate-before-commit violation the session-4/5 rules encode. Second
   occurrence across sessions.
2. **The unpublished-symbol sibling-replace gotcha bit me three more times**
   (badger/bbolt/pebble/sqlite go.mods for `DurabilityReporter`, duckdb for
   `RunPlannedOpsMatrix`) despite AGENTS documenting the class twice. The
   failure mode is identical every time: GOWORK=off test-compile fails on an
   undefined symbol that obviously exists — check the go.mod replace list
   FIRST next time.
3. **Glob-formatting swept foreign files into a commit** (`9c32ccce2`): my
   `golines -w metaengine/*/engine*.go`-style wildcard reformatted ~8 files
   owned by the concurrent calibration session (pure formatting, verified
   harmless, all suites green with them) — but they rode inside a commit
   whose message doesn't mention them. Sloppy commit hygiene; should have
   been a separate `style:` commit.
4. **Left session work uncommitted/unpushed at interruption** — the current
   tree carries 3 files of MY session-7 work (SKILL.md decision-table row,
   the `SOAK_SKIP_DGRAPH` test change, the whole turso CTE test file) and 4
   unpushed commits. Root causes: (a) `git add SKILL.md` hit the SYMLINK —
   the target path `.agents/skills/go-cqrs-lite/SKILL.md` is what's tracked,
   so the add was a no-op for content; (b) the hygiene commit's add list
   simply omitted `metaengine/dgraphengine/soak_autocrud_test.go`; (c) the
   turso work happened after the last commit and was interrupted.
5. **The persistent-MariaDB trap caught me despite the AGENTS rule** — my own
   matrix used a fixed collection name with no cleanup; only the user's
   `-shuffle=on` directive surfaced it (the harness fix + PreClean hook is
   the silver lining, but the bug was mine).
6. **Concurrent lint subshells dropped the env** — a `cmd1 & cmd2 &` split
   ran two golangci invocations without the sourced cache env (broken
   /mnt/buildcache errors); wasted a cycle. One long job at a time, per the
   session-5 rules.
7. **Test-first failures I should have caught at design time**: depth-3
   expectation off by one hop (E at depth 4), MapUpdate-leg reading `pre-b`
   on engines where backfill was skipped, item-key casing (Go field names vs
   json tags) in the canonicalizer. Each was cheap to fix but each cost a
   run-test-fix cycle that design care would have avoided.

## e) WHAT WE SHOULD IMPROVE

1. **Gates BEFORE commit, mechanically enforced per task** — twice now the
   pattern "run lint in parallel with commit" cost a follow-up fix commit.
   The `#lint-module` app exists now; the rule should be: no `git commit`
   until the touched modules' lint output has been READ.
2. **After ANY `git add`, run `git status --short` and confirm the diff
   list matches the intent** — the symlink-add trap and the omitted soak
   file were both visible in `git status` and both slipped through.
3. **Check go.mod replace lists BEFORE writing code that extends a core
   interface** — the DurabilityReporter/RunPlannedOpsMatrix replace failures
   were 100% predictable from AGENTS.
4. **Capability-gated test design**: enumerate which engines implement which
   optional interfaces BEFORE writing the expectations table; the matrix
   harness is better for having PreClean/backfilled-flags, but they should
   have been in the first draft.
5. **Single background job, no `&` splitting** — the `&` chain that dropped
   env sourcing is the same failure class as session-5's poller storm.
6. **Ledger closes belong in the task commit** — session-7 landed features
   without their TODO_LIST/FEATURES/CHANGELOG rows; batching ledger updates
   to "later" is how stale-ledger debt accumulates (this report is partially
   caused by it).
7. **New cross-engine harnesses should ship with the engines' own wiring in
   the same commit** — they did here, which is why the duckdb divergence was
   catchable; keep that bar.

## f) NEXT — up to 50 (sorted: do-first at top)

**Immediately (session-7 tails)**

1. Commit the 3 uncommitted files (SKILL.md row, SOAK_SKIP_DGRAPH test,
   turso cte_probe_test.go) — preserve the work; see §d.4.
2. Push the 4+1 unpushed commits after §f.4 goes green.
3. Run `nix run .#load-sweep` (mandatory before #verify — bench/matrix
   timing adjacencies this session).
4. Run full `nix run .#verify` → record verdict (expect GREEN; the two
   original lint findings were fixed in `cdc51f488`).
5. Finish T35: `.golangci.yml` exclusion audit (system/ 466, metaengine/
   487, cmd/cqrs-lint blocks — for each linter, decide "still needed" vs
   "removable now", with a compile/lint probe per removal).
6. TODO_LIST ledger close-out for ALL session-7 completions (§b.2 list).
7. CHANGELOG [Unreleased] entries for `LayoutPlanEvolver`,
   `KeyScanBackend`/`BackfillPlannedCollection`/`ErrBackfillUnsupported`,
   `PlannedTablesReporter`/`PlannedTableInfo`, `EffectiveDurability`
   implementations, duckdb MapScan visibility fix, `RunPlannedOpsMatrix` —
   then re-run the symbol gate.
8. FEATURES.md rows for the same new APIs.
9. Annotate the session-7 plan HTML with executed statuses (point-in-time
   artifact, annotate don't rewrite).
10. `nix fmt` full-repo pass + doc-assertions check before claiming GREEN.

**D3 train continuation (engineering, ungated)**
11. DuckDB: adopt `KeyScanBackend` + backfill (matrix leg currently skips).
12. DuckDB: adopt `LayoutPlanEvolver` (matrix leg currently skips).
13. sqliteengine: adopt `KeyScanBackend` + backfill (matrix leg skips).
14. sqliteengine + duckdb: adopt `PlannedTablesReporter` (Doctor row counts
currently pg/mysql only).
15. sqlite information_schema evolution (PRAGMA table_info path).
16. Planned-vs-meta_map bench at 100K rows (the 2K-row numbers understate
the native-column gap; RTT dominates at small N).
17. Record the planned_vs_metamap bench results in
`docs/benchmarks/` alongside the calibration baseline.
18. `RunPlannedOpsMatrix`: add a mis-type-filter rejection scenario
(ErrPlannedColumnTypeMismatch) to make the classification
cross-engine-pinned too.
19. `RunPlannedOpsMatrix`: add a MapScan-visibility subtest for
MapDelete (delete-visibility is unpinned).
20. Evaluate `-shuffle=on` for the dgraph suite (the last live suite
without a verdict).
21. Roll `-shuffle=on` into ephemeral-pg/mysql/dgraph app invocations at
the next tag wave (per the AGENTS verdict note).

**D8 continuation**
22. Implement ADR-0134 additively (`RenewLeaseWithToken`/
`MarkFiredWithToken`) when a consumer asks or at v5.
23. Claim-storm contention test (>2 pollers, ~20 timers) on live PG.
24. MariaDB SKIP LOCKED re-evaluation (MariaDB 11.8 status may have
changed — verify before relying on ErrClaimingUnsupported).

**Observability/lint**
25. cqrs-lint ApplyLayout→ApplyLayoutPlan rule: the design pass (type-impl
detection or api-stability-fed capability registry) — still scoped-only.
26. C5 remainder: plan-time diagnostic for engines over-declaring `Supports`.
27. C5 remainder: graph BFS fallback `fmt.Sprint` node-key collision
(int(1) vs "1").
28. C5 remainder: OnRecord folds returning Embedding/IndexedText/Point/
MultiEntry/Append get an always-zero Record silently.
29. Doctor: surface `--- Planned tables ---` per-engine ERROR lines when
planned tables exist but COUNT fails (partial: N/A fallback exists).

**Release mechanics (push-gated, user-gated where marked)**
30. [Q1] Tag wave: dgraphengine v4.2.0, sqliteengine v4.3.0,
projectionhost v4.5.0 + the extended D3 wave (pgengine, mysqlengine,
scheduling-sqlstore, metaengine, badger/bbolt/pebble/sqlite/duckdb
engines — the session-7 surface grows it further). Dry-runs were green
for the session-4 subset; the extended wave needs fresh dry-runs.
31. [Q1] Post-wave replace-strip sweep — now includes the 5 new engine
sibling replaces added this session (badger, bbolt, pebble, sqlite,
duckdb go.mods).
32. [Q1] Post-wave GOWORK=off build matrix over all swept modules.
33. [Q2] GitHub Releases for accumulated tags via
`scripts/create-github-releases.sh` (gh auth verified working).
34. Indirect-dep consolidation pass after the wave lands.
35. GitHub Actions billing fix (user action — every paid job red).
36. macOS runner leg for ephemeral-pg (hardware/user).
37. mysql-nspawn full-flow verification (needs root).
38. go-codec F46: commit + tag the UnwrapDecode sniff in ../go-codec, then
update the allocs expectations (user action on the sibling repo).
39. Ratify-or-revert the iroh P99 150ms judgment call (user).
40. Dead eventtest tags: delete remotes or document-and-ignore (user).

**v5 train (gated on the cut decision)**
41. D4–D6 deletion-wave dry-runs on a branch (stack presets, view/
relational, transport, tombstone API, NewStreamRef).
42. Error-code rename aggregate__→stream__ dashboard inventory.
43. Wire-tag rename decode-only fallback reader draft.
44. v5 CHANGELOG assembly from V5-MIGRATION-GUIDE.md.
45. Post-cut `grep -rn "Deprecated:"` empty gate script.

**Hygiene**
46. Sweep stray bbolt .db test files in /home/lars/projects/.gotmp.
47. Consider extracting the repeated "seed → layout → assert visibility"
pattern of the engine planned tests into adttest fully (retire the
per-engine one-off tests that RunPlannedOpsMatrix now covers).
48. doc-check: teach it to resolve `sqlstore.` aliases without a visible
import (cost session-5 a cycle; still open).
49. Consider a `#doc-check` scoped app (parity with `#lint-module`) for
per-task doc gates.
50. Re-verify WS-M ownership: if the concurrent session is dormant, absorb
the pg/mysql CounterGet recalibration (G1) into the next session with
a live-PG/MySQL window.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Push policy**: 4 commits are unpushed and 3 files uncommitted (§d.4).
   My plan: commit the leftovers → load-sweep → full `#verify` → push only
   on GREEN. Confirm, or do you want the leftovers committed and pushed
   immediately (they are gate-green individually) with verify to follow?
2. **Tag wave go/no-go**: the session-7 surface (evolution, backfill,
   Doctor sections, durability reporting, parity matrix, duckdb fix,
   claiming metrics — across metaengine core + 7 engine modules +
   scheduling/sqlstore) makes the pending wave materially bigger than the
   session-4 enumeration. Authorize cutting it now (with fresh dry-runs),
   or hold until the D3 train is "done" by some bar you name?
3. **WS-M handoff**: the concurrent session's ledger shows pg/mysql still
   carry legacy/DIVERGENCE `NsPerAggregate` numbers (only sqlite/duckdb were
   recalibrated onto the CounterGet model). If that session is dormant,
   should I take the pg/mysql CounterGet calibration (needs a live-PG and
   live-MySQL window + bench runs + constant moves), or does it stay theirs?
