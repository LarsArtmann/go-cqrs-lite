# CI-Heal Round 3 — Honest Closeout Status: Campaign WON, Green Receipt Still Chasing (2026-11-22 14:53)

> Session: continuation of the SUPERB closeout plan (M01–M12), executed 2026-11-21 ~20:00 through 2026-11-22 ~02:00, with an overnight Autoretry re-run of the last CI candidate.
> Predecessors: [13-13 sweep](2026-11-21_13-13_ci-heal-and-lint-campaign-first-sweep.md) · [17-02 mass execution](2026-11-21_17-02_ci-heal-round2-lint-campaign-mass-execution.md) · [23-59 round-3 report](2026-11-21_23-59_ci-heal-round3-closeout-green-chain.md) (committed mid-session; this file is the full a–g accounting the owner asked for).

## a) FULLY DONE (verified, committed, pushed)

| Item                                                                                                                                                                                                                                                                                                                                      | Evidence                                     |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| **The 47-module lint campaign: COMPLETE — full `#lint` matrix 89/89 modules at ZERO findings** (measured twice)                                                                                                                                                                                                                           | cde46eb97; CHANGELOG row updated to COMPLETE |
| duckdbengine → lint 0 (sentinels, exhaustive annotations, SELECT/filter extractions, test splits; plus two off-by-one placeholder fixes I had introduced, caught by tests)                                                                                                                                                                | 3178ee607, 9abcfc200                         |
| storage/turso → lint 0; **the inherited "5 REAL sqlclosecheck bugs" were verified FALSE POSITIVES** (every Rows closed via `record.DeferClose`; the linter can't see through the helper); dead exclusion row revived with corrected path, trimmed 13 linters → 1                                                                          | 46750c502                                    |
| pgengine → lint 0 (PGNsPer* priors, stride consts, 5 sentinels, writeAggregateSelect)                                                                                                                                                                                                                                                     | 9c4be2a19                                    |
| irohengine → lint 0 (checked as*Backend assertions, generic localBackend accessor naming wiring bugs, suite fixtures, demo phase extraction gocognit 48→~10)                                                                                                                                                                              | e331b6e4f                                    |
| irohengine/quic → lint 0 (error-return demo so defers run, checked asserts, peer renames, named consts)                                                                                                                                                                                                                                   | 4abf8405c                                    |
| **M09 trust tier**: coverage gate GREEN with repairs, zero re-baselines (graphadapter +8.0, stack/pebble 65.6→79.7 — real tests); erraudit 104/104; package-surface, test-runtime, untested-exports all green; 8 engines' priors frozen by TestCalibrationPriorsPinned                                                                    | 77040d720                                    |
| Local gates: `nix flake check` PASS (incl. NixOS VM test); api golden regen; shfmt 3.14.1 heredoc fix; 4 orphaned-nolint stragglers swept (queue/*, mysqltestcontainer)                                                                                                                                                                   | cde46eb97, 0945ea6bc                         |
| **M10 live backends (local)**: PG ephemeral ✓, Dgraph full 120s ✓, pebble 42s ✓, badger ✓, bbolt matrix ✓ + **790s soak ✓**, NATS watermill suite ✓ (2.4s)                                                                                                                                                                                | session log                                  |
| **Root-caused and fixed (CI)**: scheduled-Benchmarks calibration leg — THREE stacked causes: ambient-Go toolchain (job installs only nix; now setup-go from go.mod), the 2026-11-10 /v4 path move broke BOTH the dump loop and the bench loop of calibration-drift.sh (fixed in two commits — see §d), stderr now captured into the error | 559d10a5b, 9c868d760                         |
| SSE flake (2 CI occurrences): root cause = global SSEDroppedEvents delta false-attributing the parallel eviction unit tests' deliberate evictions; assertion now tests only what it can measure                                                                                                                                           | 0945ea6bc                                    |
| **The NATS CI leg never ran a single test since it was added** — bare `nix run .#integration-nats` hit the script's interactive wait-for-Ctrl+C branch; now mirrors Redis (no args = run the suite); verified end-to-end locally                                                                                                          | feec9bc74                                    |
| MySQL 1213-deadlock under the 40-claimer conformance shard: retry ladder 3→8 (25ms→2s jittered) — the documented InnoDB remedy; CI verification pending                                                                                                                                                                                   | c91c3affc                                    |
| quic transport.go 352→338 lines (limits.go split) — un-reddened File Size + Verify + flake-check at once                                                                                                                                                                                                                                  | 2f633621d                                    |
| NATS job timeout 10→20 + duplicate-key dedup (see §d)                                                                                                                                                                                                                                                                                     | 39a307e92, cccf8bca2                         |
| **M12 hygiene**: session report + README rows (via the new self-tested `scripts/add-status-row.sh` — its first production use), parallel session's dangling report landed, M21 TODO_LIST unblock, overlap audit clean (c2363922f rows absorbed by supersets), doc gates green (readme-links, canonical-facts, md-go, doc-check 1397 refs) | 9589163bc, 2e532d5b0                         |
| Runtime-gate rows for two environment-mode flips (doc-check cold-proxy 45.6→50, pgengine Docker-present skip→run 0.068→24) with named causes                                                                                                                                                                                              | 0945ea6bc                                    |
| Pushes: 8 total under the grant, all fast-forward, no force                                                                                                                                                                                                                                                                               | 8e02a4b48 → feec9bc74                        |

## b) PARTIALLY DONE

| Item                  | State                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **M08 green receipt** | NOT obtained after a chain of 8 CI candidates. The last (44972829752 on feec9bc74, Autoretried overnight as attempt 2) failed on exactly two legs: Verify-fast = the docserver-css `/v4` path bug (fix exists but is **uncommitted in my tree** — see §c), and Nix Flake Check = a **NEW, undiagnosed failure at "Error-taxonomy behaviour ratchet"** (step 19; everything before it — fmt, build, vet, test, race, lint, api-stability, untested-export, sleeps, skips, matrix, adoption — green on that job). |
| M11 scheduled triage  | Calibration Drift leg: all three root causes fixed; **one dispatch (44964504774) went green on that leg** after the second path fix; a final dispatch on the last SHA is pending. Gates Rollup = red-CI aggregator, self-heals on green. SSE-flake fix landed; systemscenario's one-off "renamed view never streamed" occurrence got a failure-adjacent drop-delta diagnostic (test-only import, harness dep budget intact) — waiting for a recurrence to classify.                                             |
| M10 mysql local       | Both `#integration-mysql-vm` runs died at the suite tail on server-teardown races ("connection reset/invalid connection" — tests that connected passed; the CI MySQL legs are authoritative and were green except the deadlock shard, which the ladder fix addresses).                                                                                                                                                                                                                                          |

## c) NOT STARTED / UNCOMMITTED RIGHT NOW

1. **`scripts/build-docserver-css.sh` /v4 fix + rebuilt `docs-ui.css` sit UNCOMMITTED in my tree** — this is the Verify-fast leg's root cause; the next receipt run cannot go green until this lands. (The rebuild also refreshed tailwind 4.3.3→4.3.5 output — a 2-line generated diff that rides the same commit.)
2. The flake check's Error-taxonomy-behaviour-ratchet failure: zero diagnosis so far (first seen in the overnight attempt-2; likely a new documented code or a baseline-row code that left the doc — the gate's two known classes).
3. Final Benchmarks dispatch + Calibration Drift green receipt on the last SHA.
4. M21's TODO_LIST row still awaits the actual green run URL.
5. CHANGELOG/FEATURES rows for the CI-repair wave (calibration, NATS, mysql ladder) — the ritual was not run for these.

## d) TOTALLY FUCKED UP (mine, honestly)

1. **quic transport.go 352 lines**: my named-const comments pushed the file past the 350 ratchet and reddened THREE CI legs at once — a locally-runnable gate (`#check-file-size`) I simply didn't run before pushing.
2. **Duplicate `timeout-minutes` in ci.yml**: I inserted the new key at the job header and left the original trailing key — GitHub rejects the whole workflow (0-second "workflow file issue"). actionlint exists locally and caught it in one second — AFTER the push.
3. **Dangling README row**: I indexed the parallel session's report without committing the report file; my local readme-links run passed because the gate can't see tracked-vs-untracked — CI caught what my local run structurally could not.
4. **The /v4 path bug fixed twice**: I fixed calibration-drift.sh's dump loop, pushed, and only then discovered the bench loop carried the identical pre-move path. The full-pattern sweep I eventually ran (`grep 'metaengine/\$mod'`) would have caught both in the first edit. Same class, different script, for docserver-css (`$REPO_ROOT/catalog`) — found only because a CI run tripped it.
5. Smaller self-inflicted wounds: overlapping python replaces mangled `mmb(`→`mmapBackend` and `evictPooledStream`'s param (both caught by build, both pure carelessness); the pin-test generator emitted `Name=Name` args; a stash dance around the README row briefly lost my own row.
6. **Chasing a self-test that doesn't exist**: I "ran" `calibration-drift.sh --self-test` twice and debugged its load-gate abort before noticing the script has no self-test branch — my invocation was silently running the real path.
7. Meta-lesson of the night: **8 CI candidates to (still) not-green** — every push surfaced the next latent failure, most of which (file size, yaml dup, dangling link) were locally detectable. I iterated on the slowest available feedback loop.

## e) WHAT WE SHOULD IMPROVE (process, from this session's scars)

1. **Pre-push CI-candidate checklist** (mechanical, ~2 min): `#check-file-size`, `actionlint` on any workflow diff, `check-readme-links`, `check-generated-artifacts`, api golden — run ALL before a receipt-candidate push.
2. **actionlint in a gate** (`check-lint-config` sibling or pre-commit) — one duplicate YAML key cost a run.
3. **Sweep-don't-spot for path classes**: when a path-move bug is found, grep the whole scripts/ tree for the pattern in the SAME edit (the /v4 class bit four+ times across three sessions).
4. **check-readme-links should flag rows targeting untracked files** — the local/CI asymmetry hid the dangling link.
5. **Runtime gate design**: environment-mode rows (Docker-present runners, cold proxies) don't fit factor-based widening; consider per-environment baselines or an explicit CI-mode row type.
6. **A first-failure index for CI legs** — the recurring classes (file-size, yaml, links, pre-/v4 paths, ambient toolchain) each cost a 25–100 min run to rediscover; one doc page of "failed leg → known classes → local repro command" would pay for itself immediately.
7. **Autoretry** re-ran the last candidate overnight — useful, but it re-ran a SHA with known-unpushed fixes; consider teaching it to skip when a newer master SHA exists.

## f) NEXT — up to 50, ranked

1. Commit the docserver-css `/v4` fix + rebuilt docs-ui.css (in tree, verified locally) — blocking the receipt.
2. Diagnose the flake check's Error-taxonomy-behaviour-ratchet failure (pull that step's log from 44972829752 attempt 2; likely a doc/baseline/code drift of the two known classes).
3. Push; watch the next CI run — the true green-receipt candidate.
4. On green: record the run URL in M21's TODO_LIST row (T01.5 rule: push-run only) + addendum link in the 23-59 report.
5. Final Benchmarks dispatch on the green SHA; confirm Calibration Drift green end-to-end (first fully-green scheduled-class leg since 11-10).
6. Confirm Gates Rollup goes green on its next schedule after CI green.
7. Verify NATS leg's first real green run (suite passed locally in 2.4s; timeout raised to 20).
8. Verify MySQL QEMU leg green with the deadlock ladder (the 28/40 shard).
9. Verify systemscenario SSE leg stays green (drop-delta diagnostic awaits any recurrence).
10. CHANGELOG `[Unreleased]` rows for: calibration /v4 + setup-go chain, NATS interactive-wait fix, mysql deadlock ladder, runtime-row raises, SSE attribution fix, add-status-row.sh.
11. FEATURES.md rows for the same (module-registration-without-inventory is the known drift start).
12. `docs/agents/gotchas-tooling-build.md`: the /v4 script-path class + sweep rule; ambient-Go runner class + setup-go pattern; actionlint-before-workflow-push rule.
13. Add actionlint to a checkable gate (check-lint-config sibling) — §e.2.
14. check-readme-links: flag untracked link targets — §e.4.
15. Sweep ALL repo scripts for remaining pre-/v4 module paths (`$REPO_ROOT/<module>` constructions) — one grep, close the class.
16. Check `check-generated-artifacts` covers docs-ui.css freshness (if not, wire it).
17. Refresh the 6 STALE-UP coverage rows (sqliteengine 72.8→75.3, tursoengine 88.6→91.0, doc-check, core/v5/id, scheduling, systemscenario, cqrs-lint/rules/version) via `--update` — ratchet-UP is the good direction.
18. Align local shfmt to 3.14.1 (nix-provided) — kill the local-vs-CI disagreement on probe-proxy-tags.sh.
19. mysql-vm local harness: add a suite-drain wait before server teardown (or document CI legs as authoritative).
20. Runtime gate: design per-environment baselines (local vs CI-mode rows) — owner input wanted (§g).
21. systemscenario SSE occurrence: if it recurs, use the new drop-delta log line to split backpressure vs fold-side; only then change code.
22. graphadapter Close error-branch is unreachable with the memory driver (no-op Close) — either a test double seam or a documented exemption; decide.
23. quic demo's `backendOf` duplicates irohengine's — tiny shared-shape note or leave (dep isolation is the point).
24. Consider moving DefaultALPN next to the other quic limits (stayed in transport.go for the gochecknoglobals row path-scope).
25. Verify the Benchmark Regression Detection leg un-skips on the next Benchmarks dispatch (it skipped on the failed runs).
26. Confirm the calibration baseline cache wasn't poisoned by the failed dispatches (green-only refresh is the design — verify next scheduled run compares sanely).
27. Autoretry: teach it (or gate it) to skip re-running SHAs superseded by a newer master push.
28. The push-triggered Benchmarks run (44961863632 succeeded) — confirm push-triggering benchmarks is intended; if not, gate to schedule/dispatch.
29. docserver-css check runs LAST in verify-fast (~40 min in) — reorder for faster feedback if cheap.
30. TODO_LIST harvest of this report's §c/§f residue (docs-health HARVEST mode).
31. Land the parallel session's still-uncommitted files (docs/research/README.md row; TODO_LIST double-D34 numbering collision to reconcile).
32. `mesh-demo/` untracked dir (present since session start, unclaimed) — owner question §g.
33. Fleet census: parallel sessions may hold unpushed commits; reconcile before any release wave.
34. The pin tests: strengthen to profile==consts equality where engines are constructible without servers (4 engines today).
35. Coverage: bboltengine soak is excluded from #test (SOAK_SKIP_BOLT) — confirm the runtime budget row reflects matrix-only timing.
36. api-sability golden: the M07 regen was additive (new consts); run TestEvery to confirm meta-gates.
37. The Verify-fast SSE+docserver ordering: consider moving check-docserver-css into flake check where the other generated-artifact gates live.
38. Write the "CI first-failure index" doc (§e.6).
39. Update the SUPERB plan file with a dated status banner per the planning-doc ritual (executed M01–M12; deviations noted).
40. quic transport_test named-returns: the un-named forms could re-trigger gofumpt churn — none observed, leave.
41. Check `check-test-sleeps` accepted the two new annotations on CI (it did locally; the flake-check sleep step was green on attempt 2 — confirmed).
42. Consider a scheduled canary that runs `nix run .#integration-nats` weekly (the leg was silent-broken for weeks; a scheduled probe would have caught it).
43. The exhaustruct pattern entry for iroh-go EndpointOptions — verify it matches under the 2.15.0 canary (flake-check lint step was green on attempt 2 — confirmed).
44. docs/status has 69 live reports — archive pass (the canonical-facts NOTE keeps nagging).
45. Session-end: confirm the auto-commit daemon hasn't absorbed unreviewed parallel-session work into my pushed commits (spot-check the 8 push commit diffs).
46. The mysql claim deadlock: longer-term, ordered locking (claim candidates sorted) would make deadlocks structurally impossible vs the retry ladder — a design row for the queue backlog.
47. Consider `--self-test` for calibration-drift.sh (env-hooked like CALIB_FAKE_TMPFS_TYPE) to satisfy the gate-script policy the AGENTS file states.
48. Add `check-docserver-css` to the pre-push checklist from §e.1 once its /v4 fix lands.
49. Re-run `#check-coverage` once after the CI-repair commits (the mysql ladder added a retry loop — statement counts moved).
50. Close-out: mark M08/M11 complete in the plan's revision log only when the receipt URL exists — no premature done-markers.

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Push-grant scope**: the 19:50 grant named "this campaign's boundary commits." The campaign since grew a CI-repair sub-campaign (8 pushes so far, the docserver-css receipt push pending, plus a final green-closeout commit). Confirm the grant extends to these remaining pushes — or name a stopping point.
2. **The mysql deadlock ladder (3→8)** is engine-behavior tuning I shipped unilaterally under the fix-it mandate. Alternative root fixes exist (ordered claim-candidate locking, §f.46). Retro-confirm the ladder, or direct the structural fix?
3. **Two owner-orphaned items**: the untracked `mesh-demo/` directory (present since session start; not mine to touch) and the parallel session's uncommitted TODO_LIST D34-row collision + research-README edit — yours to disposition, or may I absorb them into the closeout commit?

---

**Bottom line**: the campaign itself is DONE and verified (89/89 lint-zero, all local gates green, live backends green); the green CI receipt is one commit + one undiagnosed flake-check step away, blocked on nothing but the two items in §c.
