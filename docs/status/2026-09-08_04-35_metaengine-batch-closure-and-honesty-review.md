# Status Report — Metaengine Correctness Batch Closure + Session Honesty Review

**Date:** 2026-09-08 04:35 CEST
**Scope of this report:** the metaengine correctness & verification TODO batch (11 items) across the 2026-09-07 evening session + the 2026-09-08 continuation (~00:00–03:30 CEST), plus a brutal self-review of exactly that work. No unrelated subsystems were researched.
**Format note:** written as `.md` per explicit user instruction — the status-report skill's canonical format is styled HTML; this is a flagged one-off override, not a new default.

---

## a) FULLY DONE

**Batch items 1–8 (prior session, verified stable this continuation):**

1. **ApplyBatch honors `EventInput.Record`** — routes through `applyWithRecord`; empty `Record.Type` falls back to the event type. Pinned by `TestApplyBatch_HonorsRecord` + synthetic-record counting test.
2. **`recordAwareEvents` cache invalidation** — `RegisterQuery` invalidates; `Hooks.Logger` advisory path tested (`TestSyntheticRecordAdvisory_LoggerPath`).
3. **`metaengine.SortPaginate[T]` tests** — 6 unit tests, alloc budget (truncation 0 allocs, sort ≤3, upper-bound style), inlined-reference equivalence twin, `BenchmarkSortPaginate_1K`.
4. **keycodec extraction** — `SplitGroupAndSeq`/`SeedSeqMax`/`SeqTailLen(21)` in `metaengine/keycodec`; badger + pebble call sites migrated (badger back under the 350-line CI limit: 355→305); round-trip/rejects/tail-layout tests.
5. **duckdbengine restart-safety adoption + bbolt parity** — cgo-tagged harness with availability probe-skip.
6. **Observe-before-claim set** — Doctor planned-tables row counts (sqlite+duckdb), `RunPlannedOpsMatrix`/`BackfillPlannedCollection` legs, lying-only-engine Apply hard-error correlated with plan WARN, Replan/CheckRouting partition tests (direct-call pattern that defeats the default deadbands).
7. **MySQL/MariaDB claiming completion** — `TestClaimingMySQL_RenewLease` verified live on MariaDB 11.4.12 :33061 (SKIP LOCKED); decision: keep fail-at-first-Due contract, NO version probe; claiming leg wired into both nix vm-mysql runners; README support matrix + version floors (MariaDB 10.6.0 / MySQL 8.0.1, externally sourced).
8. **Dgraph calibration** — skip-guarded pins (350k point / 2.2k scan / 2.7k aggregate / 2.2k filtered-scan + O1 complexity pin); decision: ADTMap→ComplexityO1 only, other OLogN ADTs explicitly deferred; `BenchmarkCalibration_DgraphSearchQuery`; per-engine CALIB_DUMP dump tests.

**Batch items 9–10 (this continuation, end-to-end):**

9. **Planner polish** — gap-aware partition loop (`planner.go:327-334`): documented gap keeps the engine EXCLUDED but suppresses the diagnostic; disable-fix proven (inverted-flip lesson recorded below). Diagnostic names the missing interface (`missingBackendName`, e.g. `metaengine.MapBackend`) with existing substring pins kept. `TestPlan_EqualLatencyTieBreakIsDeterministic` (5 fresh plans + reversed-input sensitivity). Capability-aware partition rule documented in recipes §2.12; doc-check green; api golden regenerated.
10. **iroh test coverage** — `GraphRemoveEdge` sentinel in the graphless-local test; `TestApplyRemoteGraphRemove_RecordsLWWWithoutBackend` pins record-but-skip (disable-fix proven); int-endpoint graph convergence tests for loopback AND quic; `-race -count=3` green on irohengine + loopback + quic; `applyRemote` extracted to `engine_apply.go` (engine.go 334→281 lines).

**Gates run and green this continuation:**

- Per-module golangci on 13 touched modules (3 findings found and fixed: 2× embeddedstructfieldcheck on `honestMapMixin`, 2× golines splits).
- check-duplication: 3 new clone groups detected AND resolved to zero (see §e for how).
- check-coverage within ±2% tolerance; check-changelog-symbols (175 citations honest); doc-check (1016 references valid).
- Full metaengine module suite 2× GREEN (32.5s, 27.1s); targeted re-runs after every lint fix.
- Race: metaengine record-context lock tests `-race -count=3`; bbolt + duckdb restart-safety under `-race`.
- CHANGELOG `[Unreleased]` entries written (Added + Fixed sections for the batch); TODO_LIST batch annotated inline: 10× `[x]` with completion notes, item 11 marked deferred-to-v5.
- Status addendum appended to `docs/status/2026-09-07_22-33_metaengine-correctness-todo-batch.md`.

---

## b) PARTIALLY DONE

1. **verify-ci / CI-matrix surface is BROKEN by my new iroh tests — not fixed.** `nix run .#verify-ci` runs **GOWORK=off per-module build+test** (flake.nix confirmed) and loopback/quic are in `testModules` (enforced by `TestEveryGoModDirIsInTestModules`). Under GOWORK=off they resolve **published `irohengine/v4 v4.1.0`, which predates graph-op replication** (verified: v4.1.0's `RunConvergenceSuite` has no Graph subtests; my int-endpoint test failed against it with a 5s timeout). Workspace-mode runs (`#test`, `#test-race`, `#verify`) pass because go.work resolves the local sibling. So: local gates green, **the GOWORK=off CI matrix will go red on my 2 new tests** (loopback runtime failure confirmed; quic GOWORK=off *build* with the new test file is unverified). This is the single biggest open risk from the batch. Fix options: publish irohengine v4.2.0 (graph ops + capability conformance) and bump loopback/quic pins (proper fix, release-wave territory); or capability-probe skip-guard in the tests (fast but weaker).
2. **Aggregate `nix run .#verify` was never run this session.** All greens are per-module/per-gate. The AGENTS "stale GREEN" rule asks for `#verify`/`#verify-fast` before claiming session GREEN; I under-delivered on the aggregate and only found the verify-ci gap by reading flake.nix while writing this report.
3. **`metaengine/bench` module never built/linted this continuation** despite being on the touched-module list (imports all engines; keycodec/enginetest changes could affect it). Not verified either way.
4. **duckdbengine + sqliteengine full module suites** — only targeted (`-run Restart`, doctor, lint) runs this continuation, not the whole suites.
5. **`.art-dupl-baseline.json` is dirty with an unexplained re-pin** (recordedAt 2026-09-08T02:30:57Z; removes the now-dead pebble/bbolt seq_seeding group — consistent with MY keycodec work — and adds ~21 lines of new entries). I did not run `art-dupl baseline`. Provenance unverified: possibly the concurrent cqrs-lint session, possibly a tool I invoked. Un-owned dirty state on a gated file; the annotation policy says re-pins are for structural shifts only.
6. **Three user questions (§g of the 22:33 report) remain unanswered** — dgraph one-RPC flip scope, CapabilityGaps reach into Doctor, MariaDB :33061 retention. Decisions taken unilaterally where the batch allowed (scoped O1 flip, plan-silence boundary, server left running).
7. **Tie-break test cannot catch a `SliceStable`→`Slice` swap** — at n=2–3 Go's sort is insertion-sort and stable in practice, so no disable-fix can prove it against that specific regression; it pins input-order semantics only. Honest limitation, documented in the test comment.

---

## c) NOT STARTED

1. `errorfamily` rename `aggregate_*` → `stream_*` — deliberately deferred to v5 (TODO_LIST annotated accordingly).
2. `nix run .#verify` full aggregate — not started (see b2).
3. `metaengine/bench` verification — not started (see b3).
4. irohengine v4.2.0 publish + loopback/quic pin bump — not started (fix for b1).
5. HARVEST of the previous 22:33 report's §f (50 items) into TODO_LIST/ROADMAP — the prior session never harvested and I only carried the batch items; the remaining backlog items live only in that timestamped report.
6. CHANGELOG entries for the MySQL claiming docs/nix wiring + RenewLease test — intentionally skipped (tests/docs of a 09-06-featured capability), but the omission is undocumented.
7. CI-green confirmation for the two new quic/loopback test files under the GOWORK=off tag set — not started (this is b1).

---

## d) TOTALLY FUCKED UP

Nothing at the "revert required" level — but three things deserve the red card:

1. **I claimed "all gates green" while a gated surface (verify-ci / CI matrix) is deterministically red by my diff.** I ran the gates the task summary listed, not the gates the repo actually enforces. Reading `flake.nix` for the GOWORK semantics took 2 minutes while writing this report; doing it BEFORE the final report would have either fixed the pin or skip-guarded the tests. The repo's own AGENTS.md warned about exactly this class ("stale GREEN is worse than no claim") and I committed a soft version of it.
2. **My first disable-fix was inverted (`false &&`), and I briefly concluded the test was NOT load-bearing from a passing run.** The sed one-liner netted to a no-op earlier in the same session too (two "ok" runs that proved nothing). Disable-fix discipline works only if the flip direction is reasoned, not pattern-matched. Wasted cycles, nearly drew a wrong conclusion.
3. **The loopback int-endpoint test was written against the wrong resolver mode and failed for environmental reasons**, even though AGENTS.md documents the GOWORK-is-positional class of failure. I then burned a 5s test cycle + debugging to rediscover a documented gotcha, and — worse — recorded the gotcha for loopback/quic but did NOT chase it to its systemic conclusion (which surfaces are GOWORK=off?) until the user demanded a status report. The follow-through was one grep away.

Honorable mention (yellow card): the first draft of the loopback test used three nonexistent APIs (`ctxInt`, `gomega.Ω().ToString()`, `t.HelperIfUnused()`) — sloppy drafting caught only at build time.

---

## e) WHAT WE SHOULD IMPROVE

1. **Gate the gate list, not the habit.** "Per-task gate discipline" is necessary but insufficient: a session that touches a module consumed through MULTIPLE resolution modes (workspace vs GOWORK=off) must check both. Concretely: add "did the diff change any module whose tests resolve sibling modules?" to the pre-claim checklist; if yes, run `#verify-ci` (or the affected `lint-module`/per-module pair) before claiming green.
2. **Disable-fix flips must state direction in the commit-size step.** Write the flip as "make the behavior ALWAYS X → test must fail" in the command/comment before running. The inverted flip produced a false "load-bearing: no".
3. **Duplication-gate annotations: placement is load-bearing.** The pg/mysql `art-dupl:accept` directives sat above the function and suppressed nothing — art-dupl wants them directly on/above the region's first line. AGENTS.md documents this; the earlier session still misplaced them. A tiny `check-dupl-annotations` sanity check (or an art-dupl warning for "accept directive not adjacent to any region") would kill this class.
4. **Prefer root-cause extraction over annotation when the clone is a harness.** The `RunRestartSafetyFromDBTest` extraction (closure params for open/reopenRaw) killed a 3-file clone group and shrinks future engine adoption to ~15 lines. That is the template: annotate only what is genuinely per-module (dump bodies, register.go files).
5. **Test fakes: build them honest from day one.** `honestMapMixin` was needed because live-latency fakes lied structurally (mirrors the 07-43 `nativeMapEngine` precedent). A shared note in enginetest ("fake engines must satisfy engineServesADTNatively for every declared ADT") would stop the next session from re-learning it.
6. **Draft tests against the real API surface.** Three invented helper names in one file is a drafting-process smell: grep the package for the helper before writing it.
7. **The GOWORK resolution split is a recurring foot-gun** (VM tests, protocol benchmarks, now iroh tests). A one-page decision table in AGENTS.md ("which gate runs in which mode; how each module resolves siblings") would convert tribal knowledge into a lookup.
8. **Keep kill-your-children ordering.** The keycodec extraction deleted badger's local helpers and immediately ran `go build` — that worked. Preserve this pattern (it is in AGENTS.md as a cross-cutting lesson; it was applied correctly this session).
9. **Shared-ledger hygiene held but was luck-adjacent.** TODO_LIST/CHANGELOG edits were re-read immediately before each edit and the foreign session's files were never touched — but the dirty `.art-dupl-baseline.json` shows two sessions can still collide on a gated file with no ownership signal. A one-line "owner: <session>" comment convention doesn't exist for JSON; at minimum, re-pin operations should be announced in the shared status file.
10. **Status-report skill default (HTML) vs. practiced reality (md)** — every report in this repo is `.md`; either the skill default should change or the override should stop being silent. (Flagged here per skill contract; not propagated.)

---

## f) UP TO 50 THINGS WE SHOULD GET DONE NEXT

Prioritized; impact-first within each tier. Items 1–8 derive directly from this session; 9–30 carry forward the still-open items from the 2026-09-07 22:33 report §f (same workstream); 31–50 are new observations from this continuation.

**Red — broken/risk (do first):**
1. Resolve the GOWORK=off CI breakage for the 2 new iroh tests: publish irohengine v4.2.0 (graph WriteOp convergence + capability conformance) and bump loopback/quic go.mod pins, or capability-probe skip-guard the tests (decide via §g-style question: publish now vs guard now).
2. Verify quic module `GOWORK=off go build ./...` with the new test file (cheap check; may already fail at compile).
3. Identify the owner of the dirty `.art-dupl-baseline.json` re-pin (02:30Z) and either commit it with a rationale or revert it; re-affirm the annotations-over-repin policy.
4. Run `nix run .#verify` once on a quiet machine to convert the batch's per-module greens into the aggregate green (and catch any cross-module fall-out of the enginetest/keycodec changes).
5. Build + lint `metaengine/bench` (touched-module list member, never verified this session).
6. Run full duckdbengine + sqliteengine module suites once (only targeted tests ran this continuation).
7. Fix the pre-existing gocognit 38>35 in `scheduling/sqlstore/pg_integration_test.go:462` (`TestClaimingPostgres_RenewVsClaimRace` — extract a poll/round helper) so the integration-tag lint surface is clean too.
8. Add the GOWORK-mode decision table to AGENTS.md (which gates run workspace vs per-module; which modules resolve siblings; how loopback/quic/VM/bench differ).

**Orange — carry-forward, same workstream (from 22:33 §f, still open):**
9. Q1: decide dgraph one-RPC complexity flip scope (flip Set/Multimap/Log/StreamLog to O1 in one wave vs incremental per-ADT reassessment).
10. Q2: decide whether documented CapabilityGaps should also silence Doctor's `--- Capability ---` violations (currently plan-silence only).
11. Q3: decide MariaDB :33061 retention (kill vs keep for race runs/nspawn cross-check).
12. Next tag wave: roll `-shuffle=on` into the ephemeral-pg/mysql/dgraph app invocations (adopted 2026-08-30 for live engines; dgraph suite still needs its own shuffled evaluation).
13. MariaDB claiming: run the nspawn-runner cross-check (the nix wiring exists; the run was never executed end-to-end).
14. Claiming metrics hooks: surface `ClaimMetrics` in Doctor or a status endpoint (hooks shipped 08-30, zero consumers).
15. dgraph: per-ADT reassessment benches for the four deferred OLogN ADTs (Set/Multimap/Log/StreamLog point ops).
16. Calibration: fold `BenchmarkCalibration_DgraphSearchQuery` results into the drift-script baseline doc (bench exists, doc not updated).
17. CapabilityAudit gaps for Doctor: thread per-engine `CapabilityGaps` into `capabilityDoctorSection` if Q2 says yes.
18. `RunPlannedOpsMatrix`: add duckdb/sqlite to the ephemeral CI runners if not already scheduled (legs exist; runner wiring to confirm).
19. Consolidate `sameNeighbors`/`sameQuicNeighbors` test helpers (loopback/quic) into an irohengine-exported test helper if a third copy ever appears (two copies are under the dup threshold today).
20. enginetest: document the "fakes must satisfy engineServesADTNatively" contract next to `RunCapabilityConformance`.
21. Consider extracting `sortPaginateReference` twin behind a build tag or doc note explaining why the twin must not drift (it is intentional, but unannotated as to *who* keeps it honest).
22. Check whether the iroh `README.md` claiming/replication matrix mentions the new int-endpoint guarantees (endpoint-type independence) — doc polish.
23. Add the loopback transport's missing int normalization (or document endpoint stringification as the contract) — the test pins behavior; the transport doc should state it.
24. irohengine demo/ directory: stale? (listed in ls, not touched this batch; confirm it still builds or remove).

**Yellow — quality/tooling debt observed this session:**
25. scripts/check-staged-go.sh equivalent for test-only inversions: a tiny `scripts/check-disable-fix.sh` is overkill; instead add the disable-fix direction rule to AGENTS.md testing section.
26. art-dupl feature request (upstream, own project): warn when an `art-dupl:accept` directive is not adjacent to any detected region (would have caught the misplaced pg/mysql annotations at write time).
27. `verify-ci` runtime is per-module serial with 15m timeouts — measure wall time; if the iroh family dominates, consider a `-short`-style skip tier.
28. The flake `test`/`test-race` apps run ALL testModules in ONE `go test` invocation — a single flaky module poisons the aggregate output; consider per-module fail isolation for triage (keep the aggregate gate).
29. `metaengine/probe_live_test.go` fakes: `fakeRemoteEngine` embeds `metaengine.Calibration` AND `honestMapMixin` — consider one `honestRemoteFixture` type so the next fake doesn't re-compose both.
30. `engine_graph_internal_test.go`'s `addOnlyGraphLocal` vs external `graphlessLocal`: fine today; if a third graph fake appears, consolidate into an internal fixtures file.
31. `namedEngines` helper panics instead of taking `*testing.T` — take `t` and `t.Fatalf` (style; gocognit didn't flag).
32. `TestPlan_EqualLatencyTieBreakIsDeterministic`'s stable-vs-unstable blind spot (b7): either accept + document (current) or build an n≥12 shuffled-candidates harness where pdqsort instability is observable.
33. CHANGELOG: decide policy for test/doc-only additions (currently omitted); if omitted-by-rule, write the rule into CONTRIBUTING.md.
34. docs-health HARVEST: actually run it against this report's §f and the 22:33 §f (two backlogs now exist in timestamped files; TODO_LIST should own them).
35. Retire the superseded addendum duplication: the 22:33 report's §f items now partially duplicated here — HARVEST should mark the older list annotated/moved.
36. `docs/research/2026-09-07_turso-go-ivm-commit-failure-issue-draft.md` (foreign workstream): ready-to-file upstream draft — needs the verify-before-filing pass and actual filing; blocked on owner.
37. Turso upstream: track the ~27k IVM commit wall + grouped-view SUM divergence upstream (two known bugs now drafted); add upstream-issue links back into AGENTS gotchas when filed.
38. pebble `layout_planner.go`/`vector.go` gopls "requires go1.27" warnings: benign (toolchain noise) but 3 files trip them every session — a `//go:build` note or toolchain bump in that module would silence the ambient noise.
39. `nix run .#check-lint-config`: run once this week (formatter pin self-heal was exercised by the daemon twice in September; confirm still pinned).
40. Confirm `scripts/check-heap-parallel.sh` passes with the new bench/test files (tripwire never explicitly run this session; expected pass — verify).

**Green — hygiene / small wins:**
41. Replace the two `→` arrows inside `t.Fatalf` format strings in `loopback/graph_int_endpoints_test.go` with ASCII (test output portability).
42. `engine.go` (iroh) at 281 lines — next op kind lands comfortably, but note the next natural split (`engine_set.go`-style grouping) before it re-approaches 350.
43. `keycodec.SeqTailLen` doc comment says 20-digit+NUL layout; the tests pin 21 — re-read both and make comment/test wording identical (they agree numerically; wording says "20-digit" in one place).
44. Add `-count=3 -race` recurrence note for the iroh family to the AGENTS testing section (it passed; record the cadence so it stays a habit).
45. Delete or archive `docs/status/2026-09-07_22-33_*` §f once HARVEST lands (avoid three living backlogs).
46. `.golangci.yml` depguard: no new deps were added this session (verify stayed true — quick `check-arch` run would confirm).
47. Run `nix run .#vulncheck` at the next tag wave (not due now; scheduled reminder).
48. The `manualClock`/`newManualClock` pattern now exists in irohengine_test — consider promoting to a tiny exported testutil if a fourth transport module needs deterministic clocks.
49. README (scheduling/sqlstore): add a one-line example of running the claiming integration test against a userspace MariaDB (the :33061 recipe lives only in AGENTS.md).
50. Celebrate + keep: the direct-call `checkQueryRouting` test pattern (explicit thresholds, no deadband masking) is the reusable idiom for any future hysteresis-gated surface — document it in references/advanced.md.

---

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **iroh pin strategy:** for the verify-ci breakage — do you want the proper fix now (**publish irohengine v4.2.0** with graph-op convergence + capability conformance, bump loopback/quic pins — a mini tag wave with the GOPRIVATE push dance), or the fast **capability-probe skip-guard** in the two tests (CI-safe today, weaker contract) with the publish deferred to the next wave? Publishing also affects the proxy/pin sweep you've been batch into tag waves.
2. **`.art-dupl-baseline.json` re-pin at 02:30Z:** was that you or the concurrent cqrs-lint session? It bakes in a removed pebble/bbolt seq_seeding group (consistent with my keycodec extraction) plus ~21 lines of new entries I can't attribute. I can't tell whether to commit it as-is, split it, or revert and re-pin after the annotations-only policy review.
3. **Aggregate gate cadence:** is `nix run .#verify` (full: build+vet+test+race+lint+doc-check) expected at the END of every session, or is the per-task per-module discipline you codified in AGENTS.md an intentional replacement except before releases? I ask because I followed the per-task rule and still ended up with a red surface the aggregate would have caught — if the answer is "verify every session," the per-task rule needs a carve-out for modules consumed across resolution modes (or verify-ci needs to be in the per-task list for sibling-resolving modules).

---

*Prepared per status-report protocol (md override flagged in the header). Waiting for instructions.*
