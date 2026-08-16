# Status Report — One Bench System: consolidation + a real regression gate

**Session:** 2026-08-16 18:06 CEST · **Scope:** TODO_LIST.md "Code Quality" → *One bench system*, executed per [`docs/planning/2026-08-16_15-09_one-bench-system-consolidation.md`](../planning/2026-08-16_15-09_one-bench-system-consolidation.md) · **Branch:** master · **Terminal state:** all work committed and pushed (`master` == `origin/master` == `c29d977f4`).

**Context:** the original TODO said "delete `metaengine/bench`"; research proved only 5 of its 36 benchmarks are benchkit-redundant, so the module was **slimmed, not deleted** (user-agreed revision, design decision D4). The session's core deliverable was not the deletions but the **enforcement**: the CI benchmark regression job ran `benchstat … || true` — a gate that cannot fail is theatre. It now runs a median-based threshold check that actually exits 1 on breach.

---

## a) FULLY DONE (verified)

1. **`scripts/benchmark-regression.sh` — complete rewrite** (D2): median ns/op per benchmark via awk insertion sort; flags `--baseline/--current/--save/--threshold/--bench/--dir/--count/--benchtime`; default threshold 25%; exit 1 on breach; new/removed benchmarks reported informationally only; shfmt-clean. Verified with synthetic fixtures: stable → exit 0, +375% regression → exit 1 **and** `--save` file written, missing baseline → exit 0 + save-only warn.
2. **CI regression job enforces for real** (`.github/workflows/benchmarks.yml`): `benchstat … || true` replaced with `./scripts/benchmark-regression.sh --baseline baseline/baseline.txt --current current.txt --threshold 25`; `-run='^$'` added to the bench invocation; refresh+upload steps now `if: ${{ !cancelled() }}` (×2 occurrences) so a failed compare can never freeze the baseline artifact (D6.3). YAML parse verified for both workflows.
3. **Dead warn-only compare removed from `ci.yml`** — lines ~347–367 read the deleted root `benchmark-baseline.txt` via `git show`; found by exhaustive consumer search BEFORE deleting (it was the second, undocumented baseline consumer). Its `fetch-depth: 0` went with it.
4. **24 files deleted via `git rm`** (D5): `benchmarks/` dir (8 v2-era files, June ANSI escapes, `event/v2` refs), root `benchmark-baseline.txt`, 15 integration bench files (`scale_*`, `realistic_*`, `integration/{event,command,query}/benchmark_test.go`). All verified 0 Test funcs.
5. **`integration/simulation/generator_test.go` slimmed** — only `BenchmarkEventGenerator_Generate` stripped; both Test funcs intact.
6. **`metaengine/bench` slimmed from 36 → 32 benchmarks** (D4): deleted `bench_storm_test.go` (whole file), `BenchmarkPromise_ApplyThroughput` + `BenchmarkPromise_ConcurrentApply` (bench_promise_test.go, dropped dead `sync`/`sync/atomic` imports — import-usage checked first), `BenchmarkFilteredScan_Memory` (duckdb columnar), `BenchmarkMultiQuery_EventFanOut` (fanout). Verified via `go test -list`: 32 funcs remain, zero deleted names present (the two grep hits were `*DuckDBApplyThroughput`/`*PebbleApplyThroughput` — different, kept benchmarks). `bench-matrix.sh` metaengine pattern updated.
7. **Fresh committed v4 local baseline** — `benchmarks/benchmark-baseline.txt` recreated (gate set, 100x benchtime, linux/amd64, this machine) replacing the v2-era dump.
8. **Module gates green:** `integration` build+vet+tests `ok`; `metaengine/bench` full `-benchtime=1x` run `PASS` (427s, CGo); `stack/bench` gate set at `-count=5`; `shfmt -d` clean ×37 scripts; api-stability meta-tests `TestEvery*` `ok` (test-file-only edits, no exported API change); doc-check **898 references valid / 41 packages**.
9. **8 docs updated:** TODO_LIST (item `[x]` + revision note), AGENTS.md (Bench quick-ref row), FEATURES.md (removed "17 scale benchmarks" row; re-sourced results line), ROADMAP.md (regression-gate ✓ + consolidation note), docs/performance.md (numbers kept, harness marked removed, successor named), docs/BENCHMARKS.md (new "Regression gate" section), docs/benchmarks/README.md (ARCHIVED), CHANGELOG.md (Unreleased: Changed + Removed). Plan doc: all 12 tasks DONE, checklist ×8, new D6 section.
10. **Committed and pushed.** My explicit commit `12565d918` (fix(bench): enforce median regression gate; harden against false results); four daemon commits swept earlier stages (`49ec023bf`, `139babd4e`, `b30db59d4`, `5a08a96f3`). AGENTS.md Bench row confirmed present in HEAD.

## b) PARTIALLY DONE

1. **Tree-wide `nix run .#verify` not run to completion this session** — module-scoped gates (builds, tests, shfmt, doc-check, api-stability meta-tests) all green individually, but the full gate was impractical: `/mnt/buildcache` (default GOCACHE/GOMODCACHE target) has pre-existing I/O errors and is 99% full, and a parallel session was actively editing shared files. Full `#verify` should get one clean exclusive run (see f/1).
2. **New CI gate never observed live** — the workflow change is pushed but no GitHub run has been witnessed; first run must validate artifact seeding (no baseline → save-only, warn, pass per D3).

## c) NOT STARTED

1. **This report** — demanded before session end; the baseline-job output instead revealed the D6.1 script bug and all remaining time went to fixing and re-validating. Written now (this file).
2. **HARVEST of this report's §f** into TODO_LIST/ROADMAP — deferred: a parallel docs-honesty session has TODO_LIST.md in flight right now; harvesting on top would collide.
3. **benchstat baselines for the 3 new benchmarks** — pre-existing TODO_LIST remainder (line 92), untouched here.

## d) TOTALLY FUCKED UP (honest — all three are D6, all self-caught)

1. **Save-before-compare bug (mine, the headline fuckup):** the rewritten script's `--save` originally overwrote the baseline BEFORE the comparison ran — so every save+compare invocation compared current-against-itself and printed a vacuous "2 stable" PASS. The gate would have **enforced nothing while reporting success** — precisely the theatre this task existed to kill, reborn one layer down. Caught by reading the baseline-job output line by line, NOT by a failing test: my fixture suite covered `--save` and compare as separate scenarios but never their **interaction**. Fix: baseline medians snapshotted before the run; `--save` executes after compare, unconditionally (re-baselining after an intentional perf change must overwrite a "regressed" baseline).
2. **10x benchtime false regression:** at 10x, single samples of the microsecond-scale `BenchmarkFullPipeline_Memory` (~5–12µs) skew up to 2x under CPU steal — observed +91% while gopls was reindexing. Local default moved to 100x (back-to-back runs then compare stable); CI keeps 10x (quiet dedicated runners, preserves the artifact chain). A threshold breach on a dev laptop under load would have been a false alarm with no honest way to distinguish it from a real regression.
3. **Artifact-freeze deadlock risk:** with refresh/upload gated on job success, one failed compare would skip the baseline re-upload forever — the gate would keep comparing against a frozen artifact and either fail eternally or silently accept drift. Fixed with `if: ${{ !cancelled() }}` on both refresh and upload steps.

## e) WHAT WE SHOULD IMPROVE

- **Fixture tests must exercise flag interactions, not just flags** — the D6.1 bug survived a passing fixture suite because `--save` and compare were tested in isolation. Rule: for any script with a mutating flag, add a combined-invocation test (mutate + compare) asserting the compared-against value is the pre-mutation snapshot.
- **Snapshot-before-mutate as a structural default** — any tool that both reads and overwrites a baseline file must parse+pin the old content before any write can occur. Ordering bugs of this class are silent.
- **Artifact-chain CI steps should default to `!cancelled()`** — anything that refreshes state consumed by the next run must run even when the job fails, or the chain deadlocks. Worth an AGENTS.md workflow-pattern note.
- **Microsecond-scale benchmarks need ≥100x benchtime on noisy hosts** — 10x is only defensible on quiet dedicated runners. Encode benchtime guidance in `docs/BENCHMARKS.md` (done) and treat sub-10x local results as noise.
- **Read the actual job/tool output before declaring victory** — the vacuous PASS was visible in the first baseline-job output; the claim "gate done" preceded reading it. Evidence-first, claim-second.
- **Pathspec-limited commits when daemons sweep** — four daemon commits absorbed this session's work mid-flight; explicit commits pinned to only my files avoided double-committing the parallel session's staged files. Keep doing that.

## f) NEXT (Pareto-ordered, ≤50)

1. **Free/fix `/mnt/buildcache`** — 99% full (3.9G/220G free), I/O errors, poisons every default-toolchain `go` command and the bare pre-commit hook. Critical, ops-level (see g/3).
2. **One clean exclusive `nix run .#verify`** — module gates are green but the tree-wide gate has not run since these changes (host blocked it this session).
3. **Observe the first live CI run of the new regression gate** — validate artifact seeding path and the exit-1 path on a deliberately breached synthetic (fork or branch test).
4. **Add a fixture test for `--save`+compare interaction** — regression test pinning D6.1 (assert compare uses pre-save snapshot).
5. **Collect 1–2 weeks of CI artifact-chain data, then re-tune threshold 25%** against observed same-runner noise.
6. **Decide CI benchtime 10x vs 100x** once quiet-runner variance is known (see g/1).
7. **benchstat baselines for the 3 new benchmarks** — open TODO_LIST remainder.
8. **benchkit load-sensitive flake hardening** — TODO_LIST notes flakes discovered "one at a time" (Duration=10ms abort bound under load).
9. **`#verify` per-module timeout headroom** — open TODO_LIST item (quic convergence hit the 8m bound).
10. **Delete repo-root junk** `t/`, `result/` (16MB root-owned), `reports/coverage.out`, `reports/jscpd-report.json` — open TODO_LIST item.
11. **gomod-check replace-directive errors** in integration/middleware/schema flagged by the pre-commit hook — pre-existing, never triaged.
12. **go-structure-linter findings** — pre-commit warnings, backlog unreviewed.
13. **markdown-lint MD013 backlog** in old docs (pre-commit warnings).
14. **GOTOOLCHAIN drift: host 1.26.5 vs go.work ≥1.26.6** — every manual go command needs `GOTOOLCHAIN=auto`; consider pinning in devShell env or bumping host toolchain. Also the source of the 138 gopls/golangci LSP diagnostics (noise, not real errors).
15. **Turso factory entry for `cmd/cqrs-bench`** — noted in TODO_LIST bench section.
16. **Analytical-profile DuckDB run** — noted in TODO_LIST bench section.
17. **Per-module CHANGELOG policy** — root CHANGELOG grows monorepo-wide; decide split vs root-only.
18. **`storage/backuptest`: wire into bbolt/pebble or delete** — open TODO_LIST item.
19. **Tag final v4.x patches of `transport/http` + `transport/grpc`** (ADR-0127 removal path) — blocked TODO_LIST item.
20. **Turso explicit CTE-probe test** — sqliteengine probe covers local drivers; confirm over the remote protocol (TODO_LIST).
21. **Consider widening the gate set** beyond `BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$` after the focused set proves stable for a few weeks.
22. **Check nightly bench-matrix job health** with the slimmed metaengine/bench pattern (32 benchmarks).
23. **Skill references audit** — confirm `.agents/skills/go-cqrs-lite/references/` mention the regression gate script where bench guidance lives; run doc-check.
24. **Add `actionlint` to devShell** — plan noted it is not on host; workflows currently validated by YAML parse only.
25. **Baseline regeneration runbook** — document when to regenerate the committed local baseline (hardware change, benchtime change, gate-set change) in docs/BENCHMARKS.md.
26. **HARVEST this report's §f** into TODO_LIST/ROADMAP once the parallel docs-honesty session's TODO_LIST edit lands.
27. **Trash-empty /tmp when tmpfs pressure returns** (48G shared; existing AGENTS gotcha).
28. **api-stability: root-package-only tracking of cmd modules** — subpackage exports invisible to golden; consider extending if those APIs matter.
29. **Pre-tag replace-directive sweep** before next release (`grep -rn "=> \.\./\|=> /" --include=go.mod .`) — standing release hygiene.
30. **Consider a `verify --module <path>` scoped mode** — first-class scoped verification instead of manual per-module commands (echoed from the 17-43 report).

## g) QUESTIONS (cannot resolve myself)

1. **CI gate benchtime:** keep 10x (fast, quiet runners, ~2 min, preserves artifact chain) or move to 100x like local (10x longer, immune to single-sample steal)? My recommendation: keep 10x and let collected artifact data decide — but you may prefer noise-immunity now over cycle time.
2. **Threshold 25%:** ROADMAP notes ~20–25% run-to-run variance for some benchmarks; is 25% the intended "regression is clearly real" line, or should it be raised (40%?) until the artifact chain gives us empirical same-runner noise percentiles to calibrate against?
3. **Who owns `/mnt/buildcache`?** It is 99% full with I/O errors and breaks every default `go` invocation plus the bare pre-commit hook on this host. Freeing space, remounting, or repointing the default GOCACHE are all fixes — but it looks like shared infrastructure, and I do not know what is safe to delete.

---

**Session verdict:** the consolidation is complete and honest — 5 harnesses → 4 with clear roles (benchkit SDK, cqrs-bench CLI, stack/bench gate entry, metaengine/bench planner internals), 24 dead files gone, and the regression gate now exits nonzero on breach. The one genuine defect (vacuous self-comparison PASS) was self-caught in job output, root-caused, fixed, and re-validated end-to-end; two adjacent hazards (10x noise, artifact freeze) were fixed pre-emptively. All module-scoped gates green; commits pushed (`12565d918` + 4 daemon sweeps). Outstanding: one exclusive tree-wide `#verify`, first live CI observation, and host-level ops (see f/1–3, g/3).
