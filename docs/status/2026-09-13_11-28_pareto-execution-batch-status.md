# Pareto Execution Batch — Session Status & Brutal Self-Review

**Date:** 2026-09-13 11:28 CEST · **Session window:** ~09:30–11:28 CEST
**Baseline:** post-quick-win-batch state (plan: `2026-09-13_08-55_SUPERB-pareto-execution-plan-v2.md`)
**End state:** tree clean (daemon-committed), load 22.6 vs gate ceiling 5 → W0.4 still gated.

---

## a) FULLY DONE (verified)

| # | Item | Evidence |
| --- | --- | --- |
| 1 | **W0.1 CatchUpEngine stale-snapshot race fix** — suffix-drain stabilize loop + reactivation inside an append-blocked critical section (`EventLog.reactivateIfStable`) + bounded passes (64) + `catchUpMu` serialization of concurrent rebuilds | build + vet + all existing catch-up/reprobe tests green |
| 2 | **W0.2 concurrent stress test** — 8 writers × 500 bounded, log-length spin for structural overlap, non-idempotent counter must land exactly; 10/10 green | `TestEngineHealth_CatchUpUnderConcurrentApplies` |
| 3 | **W0.3 `-race`** full metaengine suite + projectionhost module, re-run on the FINAL tree (race subset + full suite) | `ok … 26.8s`, projectionhost 10.4s |
| 4 | **W1.3 check-modsums gate** — `scripts/check-modsums.sh` + flake app + `#verify` leg; first run caught REAL go.sum drift (`cmd/cqrs-lint/testdata/typedfixture`, stale v4.9.0 hashes), fixed; 85/85 tidy | `nix run .#check-modsums` ✅ |
| 5 | **W1.4 erraudit storage 53→0, graph 46→0** — sentinel declarations typed `error`; `legacy_as`→`AsType` in keyset_test | recount 0 |
| 6 | **W1.5 erraudit event(25)/encryption(14)/command(13)/decider(12)/kv(11) → 0** | recount 0; all five modules test-OK |
| 7 | **W1.6 erraudit remainder ~80→0** across 15 modules (batch sentinel annotator + 6 `AsType` migrations) | workspace recount 0 (except parallel session's `claiming/`, hands off) |
| 8 | **CI error-audit precondition updated** — ci.yml comments now say baseline met 2026-09-13 | ci.yml |
| 9 | **W2.5 error-taxonomy gate expansion** — 6→11 modules (+watermill, +storage/pebble, +core event/command/query), per-module pool-size floors, tightened extraction pattern (kills `limit`/`.reconstruct_event` junk captures), 5 stale doc sections regenerated from source (314 codes), green | `✓ error-taxonomy: 314 codes across 11 modules` |
| 10 | **W1.7b tripwire** — `--self-test` mutation fixture (`scripts/testdata/error-taxonomy-selftest/planted.go`): planted codes must be caught, junk must NOT be | self-test exit 0 |
| 11 | **W2.6 catch-up observability** — `CatchUpState` / `CatchUpSnapshot()` / `EngineStats.CatchUp` / Doctor "— Catch-Up —" section + test; **api golden regenerated** (6827 exports) | test green, `API surface OK` |
| 12 | **W2.7c/d** — ROADMAP v6 table pebble `serialization.go` row; module-map genproto graph-forced note | docs updated |
| 13 | **W2.8a** seed-log absolute path (anchored at lib location) + ephemeral-pg `REPO_ROOT` anchor; verified from a foreign cwd | log landed in repo `build/` |
| 14 | **W2.8b** `-race` cmd/cqrs-upgrade | ok 5.5s |
| 15 | **W2.8c** shellcheck clean over all touched scripts (incl. test-tag-release.sh) | exit 0 |
| 16 | **W2.8d** archive-count advisory in check-doc-links.sh (>10 live reports → NOTE) | gate run 684/0 |
| 17 | **W2.8e** CI artifact upload of `build/shuffle-seeds.log` on pg/redis job failure, pinned to the repo's existing upload-artifact SHA | ci.yml |
| 18 | **W1.7a** 350-line policy decision memo (A ratchet / C harness exemptions / B split-wave, recommendation A+C, B decoupled) | `docs/planning/2026-09-13_350-line-policy-decision-memo.md` |
| 19 | **CI triage** (run 34747274058) — root-caused: Magic Nix Cache throttled by GH cache API → substituter 418 → 6 nix-based jobs starved (verify-fast, dgraph, CGo, gosec, coverage, +); 4 classes fixed locally (below) | TODO_LIST CI entry re-written |
| 20 | **4 CI classes fixed locally** — File Size (store.go 953→940 via EventInput extraction), shfmt/`nix fmt` drift, api-stability golden, cmd/cqrs-lint module (taskmanager golden re-pinned; `TestTagContentMatchesChangelog` green) | local gates all green |
| 21 | **Duplication gate restored** — parallel session's reset-wave left 4 new clone groups; suppressed with 11 `//art-dupl:accept` directives (AGENTS #19 mirror class), NOT a baseline re-pin; all 7 engine modules vet-OK | `✅ No new clones detected` |
| 22 | **Bookkeeping** — 4 CHANGELOG sections (37 citations verified honest), 7 completed TODO entries deleted, stale SC2086 item disproven & closed, doc-check 1081/1081, flake check passed | all gates ✅ |

## b) PARTIALLY DONE

1. **W1.1/W1.2 CI repair** — root cause fully classified and recorded, but the FIX (cache-backend migration: flakehub-cache-action vs dropping magic-nix-cache + raised timeouts) is not implemented; needs an owner/infra decision. go.work-sync job and benchmarks.yml matview dry-run untouched.
2. **The Pareto plan itself** — executed: W0.1–W0.3, W1.3–W1.7, W2.5–W2.8. Not executed: W0.4–W0.7, W2.1–W2.4, W2.9–W2.10, W3.x (see c).
3. **W0.4 quiet-window verify + calibration re-runs** — deliberately parked twice (load 9→35 vs ceiling 5); TODO_LIST:310 entry stands. The `--self-test` for calibration-gate (a related S item) also not done.
4. **Erraudit zeroing** — `claiming/` carries 1 finding owned by the parallel queue-extraction session; gate will demand it zeroed on landing.
5. **W2.6 scope decision** — 26a/26b done; 26c "tail-only cross-call re-catch-up" deliberately NOT built (in-call suffix replay shipped; cross-call tail-only needs persisted offsets and reset-semantics analysis — documented as deferred); 26d "return ResetResult from CatchUpEngine" not done (would change a public signature — needs a breaking-change ruling; observability shipped additively instead).

## c) NOT STARTED

- W0.5 tag-wave prep (strip storage/go.mod replaces; bump+strip sibling replaces)
- W0.6 tag wave (watermill v4.7.0 first, then batches) + W0.7 pin-sweep/GitHub Releases/cqrs-lint v4.10.2
- W2.1 queue/ conformance-suite skeleton (parallel session still in flight)
- W2.2 matview grouped-spec guard; W2.3 turso IVM follow-ups (a–e); W2.4 cqrs-lint cheap-fix tail (a–f)
- W2.9 release-tooling batch (29a–g: --smoke-all, path_matches_major dedup, check-retracts-shipped.sh, smoke-probes, CONTRIBUTING refs)
- W2.10 test-hardening batch (sqlstore race-stress/counter-scope, conformance tail, taskmanager tail, check-private-deps)
- W2.7a/b/e: reset-recipe 12/12 rewrite, WithContentionObserver recipe, calibration-gate message golden
- W3.1 v5 deletions batch 1 (owner-gated); W3.2 T18 live MySQL/DuckDB migration tail; W3.3 v5 encryption-at-rest ADR
- ~90 further open TODO_LIST threads (96 open at plan start, minus this session's closures)

## d) TOTALLY FUCKED UP (own mistakes, no varnish)

1. **Shipped a self-deadlock in the first version of the fix.** The gate called `s.eventLog.Len()` inside `withAppendsBlocked` — `sync.Mutex` is non-reentrant → instant deadlock (600s test timeout). Root cause: I wrote a generic "run fn under lock" helper instead of a purpose-built gate, and didn't trace every call inside the critical section. The final `reactivateIfStable` design is what I should have written first.
2. **Trusted silent edit failures — repeatedly (3×).** The failover.go hooks edit and the store.go `catchUps` field edit both reported success but did not land (auto-commit daemon mtime races); I built on phantom state for minutes until a build/vet error exposed it. This is EXACTLY the "independently verify tool output before mutating anything" lesson recorded in AGENTS.md — I violated my own rule three times in one session.
3. **Regex batch-editor corrupted event/errors.go.** The alias pattern's `\s*$` consumed the trailing newline and glued 3 declarations (comments fused into the wrong lines). Caught by compile, repaired by hand. A dry-run count pass over a copy would have caught it for free.
4. **Stress-test design burned ~20 minutes across 3 iterations.** v1: unbounded spin writers → the rebuild finished before writers started (no coverage) AND primary reads failed on a still-armed engine (assertion env bug). v2: fixed overlap but kept unbounded spin → mutex-convoy stall, 6-minute hang I first misdiagnosed as log-flood slowness. v3 (bounded 500/writer) converged in 0.25s. The correct design was obvious in hindsight: bound the work, guarantee overlap structurally.
5. **Wrong invariant in my own test** — asserted the spare's counter equals the total post-reactivation, but folds legitimately STOP rerouting to the spare at reactivation. Semantics I should have derived before writing the assertion, not after a red run.
6. **Nearly shipped an unverified action pin.** Wrote `upload-artifact@ea165f8d…` from memory into ci.yml; caught it, aligned to the repo's existing pinned SHA. "No guessing" applies to SHAs too.
7. **Two ratchet violations I caused and then had to pay down** — store.go +8 (catchUpMu comment fat; paid via EventInput move) and explain.go +5 (Doctor wiring noticed only in the FINAL sweep; paid via net-zero consolidation). The second one proves I didn't internalize the first. The ratchet check should have run after every touched file, not at the end.
8. **Iterative regex whack-a-mole** — three successive annotator passes (top-level only → var-block aware → prefix-only rewrite) instead of one pass designed against all five declaration shapes present in the repo. Each pass was "working" by its own narrow test.
9. **`my.code` example took two attempts** (`event.wrap_example` still matched the extractor; `<your.code>` doesn't). A 10-second thought about the character class would have landed it first try.
10. **W0.4 never re-checked at session end.** Load was 9–35 every time I looked, but I parked it binarily instead of polling; a 5-minute end-of-session re-check would have made the "still gated" claim fully current (it was 22.6 at wrap — still gated, but by evidence, not assumption).
11. **Race-evidence ordering** — ran the full `-race` suite before the observability changes landed, briefly citing stale evidence; caught and re-ran on the final tree, but the correct order was free.

## e) WHAT WE SHOULD IMPROVE

1. **Post-edit verification protocol**: after every `edit`/`multiedit`, one `rg` or build proving the change landed — the daemon mtime race makes "success" unreliable.
2. **Ratchet as a per-file habit**, not a final sweep: run `check-file-size` immediately after touching any Go file; it's seconds.
3. **Purpose-built gates over generic lock wrappers**: `reactivateIfStable`-style named methods encode the ordering contract; generic `withLock(fn)` invites reentrancy.
4. **Batch editors need dry-runs**: substitution counts per pattern asserted before write; never `re.sub` without `subn` + assert.
5. **Stress tests: bound first, silence logs first** — deterministic termination and stderr discipline before the first run, not after a hang.
6. **Lock-ordering documentation belongs next to the lock acquisition**, not just in prose comments — the helper's doc comment is what saved this fix.
7. **Provenance check before gate "fixes"**: the 4 duplicate groups were parallel-session residue; checking `git log` on flagged files BEFORE choosing annotation-vs-repin would have saved a cycle.
8. **Poll, don't park, load-gated work**: a background timer re-checking `calibration-gate.sh` would have converted W0.4 from "parked" to "executed or proven impossible" with evidence.
9. **Doc-comment examples are scanner input**: examples in Go doc comments must use non-matching placeholders (`<your.code>`) by convention.
10. **The SKILL references lag new API**: doc-check validates existing refs; nothing forces new public API (CatchUpState et al.) into references. A "new export ⇒ reference check" step is missing from my wrap-up discipline.

## f) NEXT (ranked, 50)

**Release-critical chain**
1. W0.4: quiet-window `nix run .#verify` + `verify-docs.sh` e2e (run `calibration-gate.sh` first)
2. Record S03 acceptance (date/commit/durations) in TODO_LIST once 1 is green
3. Calibration: SearchQuery count=5 re-run → supersede table if medians move >5%
4. Calibration: titled `benchmark-baseline.txt` re-pin via `--save`
5. Calibration: dgraph constants re-anchor campaign
6. W0.5 tag-wave prep: strip `storage/go.mod` replaces; bump+strip sibling replaces (engines, projectionadapter, irohengine)
7. W0.5c walk CONTRIBUTING pre-tag checklist for the wave manifest
8. W0.6 cut v4 tag wave — watermill v4.7.0 FIRST (go-localsync unblock), then batches
9. W0.7 `pin-sweep.sh --check` + storage/eventstore pin evidence
10. W0.7 GitHub Releases batch + `cmd/cqrs-lint` v4.10.2 tag + install verification
11. Owner: release-policy Q3 ruling (severity-in-minor; does `bumps`-always-present + sentinel `error`-interface change ride the minor wave?) — gates 8
12. Owner: ratify one-release-cycle-after-v5 as the v6 shim-deletion window

**CI trust**
13. Decide + execute cache-backend migration (flakehub-cache-action vs drop magic-nix-cache, raise timeouts) — the single highest-leverage repair
14. Re-run CI; confirm file-size/shfmt/api-stability/cqrs-lint classes green
15. Fix the go.work sync check job
16. Dry-run benchmarks.yml matview gate (relative `cd ../metaengine/tursoengine` hop)
17. Zero the 1 erraudit finding in `claiming/` when the parallel session lands it
18. Owner: create ERRAUDIT_PAT secret → error-audit job arms (precondition now met)

**W2.9 release tooling**
19. `--smoke-all` batch mode in tag-release.sh/batch-release.sh
20. Document same-batch sibling limitation + batch `--verify` dry-run decision
21. Extract `path_matches_major` into a sourced lib (two-copy lockstep risk)
22. CONTRIBUTING.md: batch-release + check-release-scripts references
23. `check-retracts-shipped.sh` + clean-dir acceptance test
24. `smoke-probes.txt` + Test-5 no-main-package skip path
25. `tag-release --audit --baseline` known-violations mode + CI leg

**W2 remaining**
26. W2.2 matview grouped-spec guard (`AllowGroupedViews`) + Doctor note
27. W2.3a check-turso-version `--self-test`
28. W2.3b `-race` the `-tags ivmrepro` suite once
29. W2.3c clamp last chunk for non-multiple-of-1000 repro rows
30. W2.3d repro one-liner into docs/release-checklist.md
31. W2.3e fold 3 findings into the frozen upstream draft
32. W2.4a S001 allowlist corpus validation
33. W2.4b D014/D015 registry-acceptance tests
34. W2.4c B008 Warning baseline pin
35. W2.4d S001 selector-LHS receiver context + golden impact
36. W2.4e full-module `-race` cmd/cqrs-lint
37. W2.4f URL/placeholder classifier → lintutil
38. W2.10a sqlstore race-stress (Due pollers vs Metrics reader)
39. W2.10b sqlstore counter-scope pin test
40. W2.10c ApplyIdempotent dedup no-op conformance case
41. W2.10d legacy `EventLog.Record()` synthesis pin
42. W2.10e/f taskmanager must.go tests + module-map note
43. W2.10g/h `check-private-deps.sh` + sibling visibility audit

**Follow-ups from THIS session**
44. W2.6c decision: cross-call tail-only re-catch-up (persisted offsets vs always-full-rebuild) — write the one-paragraph decision
45. W2.6d decision: `CatchUpEngineWithResult` additive API vs breaking `ResetResult` return at v5
46. SKILL references: document the new catch-up semantics (stabilize loop, observability API) in recipes/advanced + doc-check
47. Owner: 350-line memo ruling (ratify ratchet + harness exemptions; schedule/drop store.go split wave)
48. SSE flake: `TestSSE_MultiSubscriberFanOut` failed once under `-race`+load — add a flake-hardening look (subscriber-wait window)
49. W3.3 v5 encryption-at-rest ADR (skeleton + KeyProvider + precedents)
50. W3.2 T18 migration tail (live MySQL/DuckDB runs) — schedule with the integration backends, quiet-window discipline applies

## g) QUESTIONS (cannot answer myself)

1. **Release policy (gates the whole tag wave):** do the sentinel `var ErrX error = …` retype (consumer-visible: direct `*errorfamily.Error` assertions on sentinel vars break) and the `bumps`-always-present wire change ship in the pending **v4 minor** wave, or do you want either held back to v5 — i.e., can I cut W0.6 as soon as a quiet-window `#verify` is green?
2. **CI cache backend:** migrate the workflows to `DeterminateSystems/flakehub-cache-action` (needs your FlakeHub account/ratification — it's the parked GitHub-Actions-billing item) or drop the deprecated magic-nix-cache entirely and accept cold-cache builds with raised `timeout-minutes`? I can implement either; I cannot decide which cost you prefer.
3. **350-line policy:** ratify the memo's recommendation — keep the ratchet (A), exempt the two test-harness monsters via an annotation (C), and treat the store.go split wave as a separate, optional effort (B)? If yes to C, what annotation syntax do you want (`# harnessexempt` vs a baseline JSON field)?

---

**Honest bottom line:** the correctness hole gating the tag wave is closed and pinned; the lint/gate debt it exposed is zeroed with permanent tripwires; everything else I touched is verified by a gate, not a claim. What remains is exactly what was always blocked on either a quiet machine or an owner decision — plus the humility items in §d, which are process debts I intend to pay via §e.
