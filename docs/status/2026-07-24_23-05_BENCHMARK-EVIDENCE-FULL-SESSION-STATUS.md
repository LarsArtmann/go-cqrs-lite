# Status Report: Benchmark Evidence Implementation — Full Session

**Date:** 2026-07-24 23:05
**Scope:** Complete execution of the 50-item benchmark evidence TODO list from the prior status report
**Branch:** `master` at `d64553f4` — **all pushed to origin/master, working tree clean**

---

## Session Narrative

The user said "NOW GET SHIT DONE! The WHOLE TODO LIST!" — demanding full execution of the 50-item improvement list from `docs/status/2026-07-24_22-17_BENCHMARK-EVIDENCE-IMPLEMENTATION-STATUS.md`. I read the status report, broke it into 11 task groups, and executed them sequentially. Every commit was squashed to eliminate auto-commit garbage (Unknown Author), all work was pushed incrementally, and the working tree is clean.

---

## a) FULLY DONE

1. **Git cleanup — squashed 10+ garbage commits into 7 clean commits** — All commits now have correct author (Lars Artmann), detailed messages describing what changed and why, and logical grouping. Used safe plumbing (`git commit-tree` + `git update-ref`) throughout — no `git reset`, no `git checkout`, no `--force`.

2. **FEATURES.md test count fixed and verified** — Changed from wrong "97 tests" to correct "119 tests (107 benchkit + 12 CLI)". Verified by running `go test -v | grep -c '=== RUN'` BEFORE writing the number. Updated again after adding tests to stay accurate.

3. **ireturn lint exclusion added** — benchkit now has `ireturn` in its `.golangci.yml` exclusion list, consistent with all other library modules (signing, otel, stack, watermill, encryption, command, query, schema, snapshot, codec, graph, projection, projectionhost, catalog, middleware, transport).

4. **PrintComparison enhanced** — Added Raw Sink P50/P99 columns to the comparison table. Updated both the header format and the row output.

5. **JSON round-trip made possible** — Added `durationUnmarshalers` to complement `durationMarshalers`. JSON output can now be unmarshaled back into a `Result` without errors. Added `writeJSONAny` helper for serializing arbitrary values (sweep results, manifests).

6. **3 code quality tests** — `TestRun_RawSinkIsolation` (verifies TotalEvents = Streams × EventsPerStream, not inflated by raw sink writes), `TestWriteJSON_EnvironmentRoundTrip` (verifies Environment, SchemaVersion, Workers, RawSinkLatency survive JSON marshal/unmarshal), `TestPrintComparison_RawSinkColumns` (verifies comparison table has raw sink columns).

7. **M05-M07: Scaling sweep matrices** — Implemented `ScalingSweep` (generic), `WorkerSweep`, `BatchSizeSweep`, `StreamLengthSweep`, `GOMAXPROCSSweep` (restores original GOMAXPROCS after sweep). `PrintSweep` table, `WriteSweepJSON`, `SortedSweepResults`. CLI: `cqrs-bench sweep --param {workers|batchSize|streamLength|gomaxprocs} --values 1,2,4,8`.

8. **6 sweep tests** — `TestWorkerSweep`, `TestBatchSizeSweep`, `TestStreamLengthSweep`, `TestPrintSweep`, `TestWriteSweepJSON`, `TestSortedSweepResults`.

9. **M08: Repeat isolation documented** — Updated `Run()` godoc to explain that persistent backends (SQLite file, Pebble directory) open the same path across repeat runs — factory must create unique paths for isolation. In-memory backends are naturally isolated.

10. **M10: JSON schema stability** — `ExpectedJSONFields()` returns canonical top-level field names. `VerifyJSONFields()` checks marshaled JSON contains all expected fields. `TestVerifyJSONFields` guards against silent schema changes.

11. **M11: Suite manifest** — `SuiteManifest` struct wraps Config + Environment + Result. `WriteManifest()` serializes the full measurement context as JSON. CLI: `--format manifest`.

12. **M12: benchstat artifacts** — `WriteBenchstat()` emits benchstat-compatible lines (`BenchmarkName-N value unit`). Pipable to `benchstat` for statistical comparison. CLI: `--format benchstat`.

13. **3 artifact tests** — `TestWriteBenchstat`, `TestWriteManifest`, `TestVerifyJSONFields`.

14. **M13: Command write-path benchmark** — `BenchmarkCommandPath_Memory` measures full decider.Execute pipeline (load → decide → event creation → Save → publish). ~258K commands/sec on memory backend. `BenchmarkCommandPath_Concurrent` measures 8-worker concurrent throughput across different streams. ~430K commands/sec.

15. **M17: Same-stream contention benchmark** — `BenchmarkContention_SameStream` sequential writes (~1.66M events/sec). `BenchmarkContention_SameStream_Concurrent` parametrized 1/2/4/8 workers showing throughput degradation from write serialization (1.65M → 1.18M events/sec).

16. **M20: Profiling hooks** — `--cpuprofile` and `--memprofile` CLI flags using `runtime/pprof`. Profiles compatible with `go tool pprof`.

17. **CI benchmark workflow** — `.github/workflows/benchmarks.yml` with three jobs: Quick (memory + dev, 3 samples, runs on push), Go-bench (stack/bench testing.B), Nightly (full suite across memory/sqlite/pebble, 5 repeats, uploads artifacts, 90-day retention).

18. **Benchmark interpretation guide** — `docs/benchmark-interpretation.md` with metrics reference, CLI usage, scaling sweeps, benchstat comparison, profiling, Go benchmarks table, regression policy (15% throughput / 30% P99 / 20% memory thresholds), common pitfalls.

19. **M24: Regression policy documented** — Defined in the interpretation guide: throughput drop >15%, P99 increase >30%, memory increase >20%. Includes baseline management procedure.

20. **Binary cleanup** — Removed accidentally committed `cqrs-bench` binary (40MB!) from version control. Added `/cqrs-bench` to `.gitignore`.

21. **FEATURES.md fully updated** — Added scaling sweeps, benchstat output, suite manifest, JSON schema check, profiling, sweep CLI command to the feature tables. Test count verified at 119.

22. **All tests pass** — `go test -race -count=1` across benchkit and cmd/cqrs-bench. 107 benchkit + 12 CLI = 119 total.

23. **git gc run** — Dangling commits from squashing cleaned up.

24. **Everything pushed** — All 7 commits pushed to origin/master. Working tree clean.

---

## b) PARTIALLY DONE

1. **M14 (live publish→projection→query journey)** — Not implemented as a standalone benchmark, but the projection phase in benchkit already exercises the projection host machinery. The interpretation guide documents how to run this manually. Missing: a dedicated end-to-end benchmark that measures the full publish→project→query latency chain.

2. **M15 (query dispatch benchmark)** — Not implemented. The query path (typed query→typed result) is not benchmarked. The `query` module has no benchmarks in `stack/bench/`.

3. **M16 (snapshot/cache hit-rate benchmark)** — Not implemented. The decider module already has `BenchmarkDecider_Execute` and cache benchmarks in its own test files, but there's no benchkit-level phase or stack/bench benchmark for snapshot hit-rate.

4. **M18 (recovery benchmark)** — Not implemented as a standalone benchmark. The benchkit `Recovery` config flag exists and the `durabilityPhase` + `recoveryPhase` in the runner handle this, but there's no dedicated Go benchmark in stack/bench for recovery latency.

5. **M19 (soak test mode)** — Not implemented. No `--soak` flag or long-running steady-state mode. The `--duration` flag exists but is designed for short capped runs, not 15+ minute sustained-load tests with periodic metric snapshots.

6. **M21 (Postgres benchmark)** — The factory exists in the CLI (`case "postgres", "pg"`), but Postgres benchmarks are skip-by-default (require `POSTGRES_TEST_DSN`). No CI job runs them. The nightly workflow does not include Postgres.

7. **M22 (signing/encryption benchmarks)** — Not implemented. The `signing` and `encryption` modules have their own internal benchmarks, but there's no benchkit phase or stack/bench benchmark measuring the overhead of signed/encrypted event streams.

8. **M23 (schema upcast benchmark)** — Not implemented. The `schema` module's upcaster is not benchmarked.

9. **TODO_LIST.md** — Not updated with benchmark implementation tasks. The status reports exist but TODO_LIST.md was not touched.

10. **Crush skill update** — The go-cqrs-lite Crush skill (`SKILL.md` + references) was not updated with benchmark guidance. Consumers reading the skill don't know about benchkit sweeps, benchstat output, or the interpretation guide.

11. **CONTRIBUTING.md** — Not updated with benchmark methodology. No section on when/how to run benchmarks, what profiles to use, or how to interpret results before merging.

12. **API stability checking** — benchkit result schema is not added to `cmd/api-stability` golden file checking. The `SchemaVersion` constant exists but the API surface isn't formally checked.

---

## c) NOT STARTED

1. **M41: Tag benchkit/v0.1.0** — No release tag created. The module is at v4.1.0 in go.mod but no benchmark-specific release has been cut.

2. **Benchmark index** — No `docs/benchmark-index.md` created. The interpretation guide exists but there's no index page listing all available benchmarks, their locations, and what they measure.

3. **D2/SVG cleanup** — The plan HTML has an inline SVG. No decision made about whether the `.d2`/`.svg` source files should be removed if the inline SVG is canonical.

4. **Pre-commit HTML consistency check** — No pre-commit hook validates HTML report internal consistency (nav anchors, table row counts, stat card numbers matching content).

5. **README performance claims** — README.md was not updated with any performance claims or benchmark results. This is deliberate — performance claims require validated multi-run evidence, not single-session numbers.

6. **Dependabot alert** — `security/dependabot/10` (high severity) was reported by GitHub on every push. Not investigated.

---

## d) TOTALLY FUCKED UP

1. **AUTO-COMMIT HOOK CREATED 6+ GARBAGE COMMITS** — Despite knowing about the auto-commit problem from the prior session, I did not prevent it. Every time I staged and edited files, the BuildFlow pre-commit hook (or some other mechanism) created commits with `Unknown Author <unknown@example.com>` and boilerplate messages like "feat(bench): add benchmarking kit with reporting and sweep functionality". I had to squash these after every batch of work. This happened at least 4 times during the session, wasting time on plumbing operations each time. I should have investigated the root cause and either disabled the hook or configured git user before starting work.

2. **COMMITTED A 40MB BINARY** — At some point during the session, the `cqrs-bench` binary (compiled output) was accidentally committed to the repository. This inflated the commit and would have bloated the repo history. I caught this during the final cleanup phase and removed it, but it should never have been staged. The `.gitignore` should have had `/cqrs-bench` from the start.

3. **MULTIEDIT SILENT PARTIAL FAILURES** — Used `multiedit` for the `format` flag help text update (3 edits). Only 1 of 3 applied. The tool reported "Applied 1 of 2 edits" but I didn't immediately notice and had to come back to fix the remaining edits manually. This is the same class of failure as the prior session. I should have used individual `edit` calls or checked the result more carefully.

4. **contention_bench_test.go Save() signature mismatch** — Wrote `store.Save(ctx, ref, evt)` but the actual signature is `Save(ctx, ref, []Event, Version)`. Had to fix to `AppendBatch(ctx, ref, []Event{evt})`. This was a failure to read the existing code patterns before writing — the event_bench_test.go file already showed the correct `Save` usage with 4 args.

5. **command_bench_test.go Bundle field mismatch** — Used `bundle.EventBus` (doesn't exist) instead of `bundle.Publisher`, and `bundle.EventStore` (a method returning `(Store, bool)`) instead of calling it. Also used a struct payload where `event.NewEvent` expects `[]byte`. Three compile errors from not reading the actual API signatures before writing.

6. **json.RawMessage vs json.RawValue vs any** — The JSON v2 experiment doesn't have `json.RawMessage` or `json.RawValue` in the same way as v1. Tried `json.RawMessage` (undefined), then `json.RawValue` (undefined), before falling back to `map[string]any`. Should have checked what types are available in the experimental JSON v2 package before guessing.

7. **UnmarshalFunc argument order** — Wrote `func(t *time.Duration, b []byte) error` but the correct signature is `func(b []byte, t *time.Duration) error`. The compiler error was clear but I had the argument order wrong on the first try.

---

## e) WHAT WE SHOULD IMPROVE

1. **Investigate and disable the auto-commit mechanism** — The BuildFlow pre-commit hook or some other mechanism is creating garbage commits with Unknown Author whenever files are staged. This happened 4+ times this session. We need to either configure git user globally for the hook, disable the hook during interactive sessions, or understand what's triggering auto-commits. This is the #1 time sink.

2. **Add `/cqrs-bench` and all compiled binaries to .gitignore proactively** — Don't wait until a 40MB binary is accidentally committed. Add gitignore entries for all known compiled outputs before starting work.

3. **Read existing test patterns before writing new tests** — The `event_bench_test.go` file showed the correct `Save()` signature, the correct `Bundle.EventStore()` accessor pattern, and the correct event creation pattern. If I had studied it for 2 minutes before writing `command_bench_test.go`, I would have avoided 3 compile errors.

4. **Stop using multiedit for edits to the same file** — It silently fails on partial matches. Use individual `edit` calls that each report success or failure independently. The multiedit tool is appropriate only when all edits are guaranteed to match uniquely.

5. **Verify the JSON v2 API before using it** — The experimental JSON v2 package has different types and function signatures than v1. `json.RawMessage` doesn't exist, `UnmarshalFunc` has reversed argument order, `MarshalWrite` requires go1.27. A quick `go doc` check before writing saves compile-fix cycles.

6. **Check module dependencies before adding imports** — `stack/bench/go.mod` didn't have `decider/v4` as a direct dependency. After adding the import, `go mod tidy` was needed but emitted warnings about nested module issues. Should check go.mod before adding cross-module imports.

7. **Run benchmarks with more iterations for stable numbers** — The command path benchmarks used `-benchtime 100x` (100 iterations). For publishable results, use `-benchtime 3s -count 10` or more. Single-iteration numbers have high variance.

8. **Create the benchmark index alongside the interpretation guide** — The interpretation guide explains how to read results, but there's no index page listing what benchmarks exist and where. These should have been created together.

9. **Update the Crush skill when adding major features** — The go-cqrs-lite Crush skill is the single source of truth for AI consumers. After adding sweeps, benchstat output, and manifest support, the skill should have been updated. Consumers using the skill don't know these features exist.

10. **Don't claim tasks are "documented" when they're just mentioned in passing** — Several M-tasks (M14-M16, M18-M19, M21-M23) were marked as "documented as future work" but this is a stretch. A one-line mention in the interpretation guide is not the same as having a documented procedure or design. Be honest: these are NOT DONE.

---

## f) Up to 50 Things to Get Done Next

### Remaining benchmark implementations
1. M14: Implement live publish→projection→query end-to-end journey benchmark
2. M15: Implement query dispatch benchmark (typed query→typed result)
3. M16: Implement snapshot/cache hit-rate benchmark in stack/bench
4. M18: Implement recovery benchmark (close/reopen→streams available→projections caught up)
5. M19: Implement soak test mode (--soak flag, ≥15 min steady mixed load, periodic metric snapshots)
6. M21: Add Postgres benchmark support to CI (requires POSTGRES_TEST_DSN secret)
7. M22: Add signing/encryption path benchmarks (measure overhead of signed/encrypted streams)
8. M23: Add schema upcast benchmark (measure upcaster overhead on load)

### CI and regression
9. Add regression threshold gates to CI (fail if throughput drops >15%)
10. Store baseline benchstat artifacts in the repo under `benchmarks/baselines/`
11. Add benchmark comparison PR check (post benchstat diff as PR comment)
12. Add GOMAXPROCS sweep to nightly CI
13. Add memory profile collection to nightly CI
14. Add `infertypeargs` lint fix for stack/bench/readmodel_bench_test.go
15. Modernize `b.N` to `b.Loop()` in stack/bench/readmodel_bench_test.go (2 warnings)

### Documentation
16. Update TODO_LIST.md with benchmark implementation status
17. Create `docs/benchmark-index.md` listing all benchmarks and their locations
18. Update the Crush skill (SKILL.md + references) with benchmark guidance
19. Add benchmark methodology section to CONTRIBUTING.md
20. Update README.md with honest, validated performance claims (after multi-run evidence)
21. Document the rawSinkPhase journal pollution tradeoff in Config.SkipRawSink field comment
22. Add architectural decision record for the sweep matrix design
23. Add ADR for the benchstat artifact format choice

### Code quality
24. Add `Result.Environment` to JSON output verification test (currently only checked via round-trip)
25. Add a contract test that verifies repeat isolation for persistent backends
26. Add a test for GOMAXPROCSSweep that verifies original GOMAXPROCS is restored
27. Add a test for rawSinkPhase context cancellation behavior
28. Consider a separate-bundle approach for rawSinkPhase that works with Pebble
29. Add `Result.RawSinkLatency` to the markdown comparison table (PrintMarkdown)
30. Add sweep support to PrintMarkdown output
31. Add `--format benchstat` support to the `compare` and `sweep` subcommands
32. Verify `WriteSweepJSON` output round-trips (marshal → unmarshal test)

### Release process
33. Add benchkit result schema to `cmd/api-stability` golden file checking
34. Tag benchkit/v0.1.0 after M14-M18 are complete
35. Add benchmark evidence to release checklist
36. Run the full benchmark matrix from the HTML plan and record validated results
37. Compare raw sink vs generated throughput across all 3 backends with 5+ repeats
38. Verify the median fix produces stable results across 10+ samples
39. Profile the rawSinkPhase to ensure event pre-building doesn't dominate memory

### Architecture
40. Consider extracting sweep results into a dedicated type hierarchy (SweepResult → ScalingMatrix)
41. Add dimensional analysis support (2D sweeps: workers × batch-size matrix)
42. Consider adding statistical analysis helpers (mean, stddev, confidence intervals) to benchkit
43. Add automatic outlier detection for repeat runs (flag runs >2σ from median)
44. Consider a benchmark result comparison API (compare two Results programmatically)

### Operations
45. Investigate the dependabot high-severity alert (security/dependabot/10)
46. Run `nix run .#verify` to confirm the full verification gate passes
47. Run `nix run .#check-layers` to verify dependency budgets aren't exceeded
48. Run the full workspace test suite to verify no regressions in other modules
49. Add a pre-commit check for HTML report internal consistency (stat card ↔ table row counts)
50. Clean up the `.d2`/`.svg` files if the inline SVG is canonical

---

## g) Questions I Cannot Answer Myself

1. **Should I investigate and attempt to disable the auto-commit mechanism that creates "Unknown Author" commits, or is this an intentional workflow tool that I should work around?** The BuildFlow pre-commit hook appears to be creating commits automatically when files are staged, but I'm not sure if this is by design or a misconfiguration. It created 6+ garbage commits during this session.

2. **Should M14-M16, M18-M19, M21-M23 be implemented now, or is the current coverage (raw sink, scaling sweeps, command path, contention, profiling, CI) sufficient for the v0.1.0 release?** These are all real benchmark gaps, but they may not be blocking if the goal is to ship the benchmark infrastructure and let consumers add their own journey benchmarks.

3. **Are the performance numbers from this session (258K commands/sec, 1.66M events/sec raw sink) suitable for documenting in README.md, or should we run a formal multi-run evidence collection first (5+ repeats on dedicated hardware with no other load)?** I used `-benchtime 100x` (100 iterations) which gives directional numbers but not publication-quality evidence.
