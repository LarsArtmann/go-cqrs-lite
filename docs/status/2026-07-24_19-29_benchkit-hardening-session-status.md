# Benchkit Hardening Session — Comprehensive Status

**Date:** 2026-07-24 19:29
**Session goal:** Execute the entire [Pareto plan](../planning/2026-07-24_17-59_benchkit-hardening-pareto-plan.md) (P1-01 through P1-11)
**Result:** 11/11 tasks executed, 9 fully verified, 2 with caveats

---

## A. FULLY DONE (verified passing)

### P1-01: SQLite pool config fix
- **What:** Added `storage.ConfigureSQLitePool(sqlDB)` after WAL enable in `stack/sqlite/preset.go:136`
- **Root cause:** The function exists, is used by Turso, is called by internal benchmarks, but was missing from the SQLite stack preset
- **Verification:** `cqrs-bench run --backend sqlite --profile small` now succeeds (was: `SQLITE_BUSY`). SQLite handles 4 goroutines correctly.
- **Risk:** Very Low — 1-line wiring fix calling an existing function

### P1-02: Compare-mode disk fix
- **What:** `compareCmd` now collects per-backend `diskPath` via `compareWithDiskPaths()` instead of discarding them with `makeFactory(name, "", "")`
- **Verification:** `cqrs-bench compare --profile dev --format markdown` shows 574KB for pebble, 4.5MB for sqlite (was: all 0 B)
- **Risk:** Low — new helper function, no change to existing API

### P1-03: --version fix
- **What:** Replaced hardcoded `"v4.1.0"` with `runtime/debug.ReadBuildInfo()` + VCS revision fallback
- **Verification:** `cqrs-bench --version` now outputs `cqrs-bench version v0.0.0-20260724...` (was: always `v4.1.0`). Test updated to accept both tagged and devel builds.
- **Risk:** Low — stdlib only

### P1-04: TODO_LIST update
- **What:** Closed 4 done items, added 10 open items. All 29 findings now tracked.
- **Risk:** Zero — documentation only

### P1-05: SKILL.md entry
- **What:** Added benchkit section to root SKILL.md with usage examples + decision matrix (cqrs-bench vs go test -bench). Added `cmd/cqrs-bench` to modules.md table.
- **Verification:** doc-check reports 897 references valid across 34 packages

### P1-06: Lint verification
- **What:** Fixed `unparam` lint issue (removed always-nil error return from `compareWithDiskPaths`). Fixed `wsl_v5` whitespace issues in `report.go`. Fixed `varnamelen` (`ru` → `usage`) in `cpu_unix.go`.
- **Verification:** `nix run .#lint` — **0 issues** across all modules including benchkit

### P1-09: ADR-0060
- **What:** Wrote `docs/adr/0060-benchkit-design-decisions.md` documenting 5 design decisions: codec-aware padding, warmup isolation, ReadRatio-as-passes, SkipPhases, DiskSizer -1 sentinel
- **Cross-ref:** Added link from `cmd/cqrs-bench/README.md` to the ADR

### P1-10: CPU measurement fix
- **What:** Replaced `/proc/self/stat` parsing (10ms tick resolution) with `syscall.Getrusage` (microsecond resolution). Split into `cpu_unix.go` (//go:build unix) and `cpu_other.go` (//go:build !unix).
- **Verification:** `cqrs-bench run --backend memory --profile dev` now reports `CPU: 3.757ms` (was: `CPU: n/a`)
- **Risk:** Low — getrusage is POSIX standard, available on all Unix

---

## B. PARTIALLY DONE (implemented but with caveats)

### P1-07: DiskSizer on Pebble
- **What:** Full 3-layer implementation:
  - `storage/pebble.Backend.DiskUsage()` — computes from level sizes + WAL + obsolete files
  - `stack.WithDiskSize(fn)` option + `stack.Bundle.DiskSize() int64` method (returns -1 when not registered)
  - `benchkit/phases.go durabilityPhase` — checks `>= 0` before using DiskSizer value, falls back to filesystem walk
  - `stack/pebble/preset.go` — wires `WithDiskSize` at construction
- **Caveat:** Pebble v1.1.5 does NOT have `DB.DiskUsage()` method (it was added in a later version). I compute it from `Metrics()` level sizes + WAL physical size + obsolete/zombie table sizes. This is an approximation — it may differ from actual filesystem du because Pebble may have files not tracked in Metrics.
- **Verification:** `TestRun_Pebble_DiskSizerInterface` passes (disk bytes > 0 without DiskPath set). Compare mode shows 574KB for pebble.

### P1-08: Missing tests
- **What:** Added 10 new tests:
  - **benchkit** (3): `TestConfig_ConcurrencyOverride`, `TestRun_SQLite_ReadFromTime`, `TestRun_Pebble_DiskSizerInterface`
  - **cmd/cqrs-bench** (7): `TestCLI_UnknownProfile`, `TestCLI_UnknownBackend`, `TestCLI_CodecCBOR`, `TestCLI_WarmupFlag`, `TestCLI_OutputFile`, `TestCLI_Compare_DiskNonZero`, updated `TestCLI_Version`
- **Test count:** Now 88 total (77 benchkit subtests + 11 CLI tests), up from ~55
- **Caveat:** Did NOT run the full suite with `-race` until the very end. `TestRun_DurationAborts` fails under `-race` (see section D).

### P1-11: Projection benchmark fix
- **What:** Added polling loop in `projectionPhase` that waits for all events to be processed (up to 30s timeout) before calling `host.Stop()`.
- **Verification:** `projectionEvents: 500` for dev profile (was: 0). `projectionLag: 1.5ms`.
- **Caveat:** The projection handler is a no-op (`func(_ context.Context, _ event.Event) error { return nil }`). It measures projection infrastructure throughput, not real projection handler cost. A more realistic projection benchmark would write to a kv.Store.

---

## C. NOT STARTED (from the Pareto plan)

These were explicitly in the "future sessions" tier:

- **P1-12:** Phase 2 durability benchmark (crash recovery, replay-after-restart) — 100 min
- **P1-13:** Phase 6 production replay (replay real event streams) — 100 min
- **P1-14:** Phase 7 `benchtest.RunSuite` (Go `testing.B` wrappers) — 100 min
- **Postgres benchmark tests** — `stack/postgres` skips without `POSTGRES_TEST_DSN`
- **Analytical benchmark profiles** — profiles for read-heavy analytical workloads

---

## D. TOTALLY FUCKED UP (issues found but NOT fixed)

### D1: TestRun_DurationAborts fails under `-race`
- **What:** Under `-race`, the test takes 3.1s instead of <2s, causing a failure.
- **Root cause:** The race detector slows execution ~2-10x. The test uses `Duration: 5*time.Millisecond` with a 100K-stream profile. Under race, the read-model phase runs against partially-written data and gets `kv: key not found` errors. The test expects either success or a clean error, but the read-model phase error propagates as a test failure.
- **Impact:** `go test -race ./benchkit/...` FAILS. This is a pre-existing issue — I didn't introduce it, but I should have caught and fixed it during P1-08.
- **Fix needed:** The read-model phase should either (a) skip when the write phase was duration-aborted, or (b) tolerate missing keys gracefully.

### D2: CHANGELOG.md not updated
- **What:** The CHANGELOG `[Unreleased]` section still says "55 tests (50 benchkit + 5 CLI)" and doesn't mention any of the 6 bug fixes from this session (SQLite pool, compare disk, --version, DiskSizer, CPU measurement, projection benchmark).
- **Impact:** The historical record is wrong. Anyone reading the CHANGELOG won't know these fixes happened.
- **Fix needed:** Add a new entry under `[Unreleased]` with the 6 fixes and updated test count.

### D3: FEATURES.md not updated
- **What:** FEATURES.md still lists benchkit gaps that are now fixed (DiskSizer, CPU measurement, projection benchmark).
- **Impact:** Feature status is stale.

### D4: BuildFlow auto-committed with misleading messages
- **What:** BuildFlow created generic commit messages like "feat(benchkit): add benchmarking toolkit with CPU monitoring" that don't mention the actual fixes (SQLite pool, compare disk, version, etc.). The commits also bundled unrelated concurrent changes (event-size-scaling benchmark, compat alias tests, listing golden test renames) with my changes.
- **Impact:** Git history is muddied. The commit messages don't accurately describe what changed.
- **Can't fix:** Can't rewrite history (safety rule: never git reset).

### D5: Pareto plan not annotated as done
- **What:** `docs/planning/2026-07-24_17-59_benchkit-hardening-pareto-plan.md` still shows all tasks as open. Should be annotated with completion status.
- **Impact:** Future readers won't know which tasks were completed.

### D6: benchkit/README.md not updated
- **What:** I added an ADR cross-reference to `cmd/cqrs-bench/README.md` but didn't check/update `benchkit/README.md` itself. It may have stale information about the DiskSizer interface, CPU measurement, or projection benchmark.

---

## E. WHAT WE SHOULD IMPROVE (process & code)

### Process improvements
1. **Run `-race` early, not last.** I ran tests without `-race` throughout the session and only discovered the race failure at the end. Always run `go test -race` as part of the test cycle, not as an afterthought.
2. **Update CHANGELOG.md as part of the task, not after.** Each bug fix should get a CHANGELOG entry immediately. I forgot this entirely.
3. **Annotate planning docs when tasks complete.** The Pareto plan is a living document — mark tasks done inline, don't leave them stale.
4. **Verify go.work sync after adding new files.** I added `cpu_unix.go` and `cpu_other.go` but never ran `go work sync`.
5. **Don't trust BuildFlow commit messages.** They're generic and often misleading. The actual changes span SQLite fixes, compare-mode fixes, version fixes, DiskSizer, CPU measurement, projection fix, tests, ADR, docs — none of which are captured in the commit messages.

### Code improvements
6. **The `durabilityPhase` -1 sentinel is subtle.** `DiskSize() returns -1` is not obvious to callers. Consider a `HasDiskSize() bool` method or making `diskSizeFn` nil-checkable instead. The current approach works but relies on a magic value.
7. **The projection benchmark is a no-op handler.** It measures projection infrastructure overhead (checkpoint reads, journal scans, batch processing) but not actual projection work. A kv.Store-backed counting projection would be more realistic.
8. **The `cpu_other.go` fallback returns 0.** On Windows, CPU metrics will show "n/a" forever. Consider using `runtime.MemStats` or a platform-specific CPU counter as a fallback.
9. **The `compareWithDiskPaths` function duplicates `benchkit.Compare` logic.** The only difference is per-backend DiskPath injection. Consider adding a `CompareConfig` type or making `Compare` accept per-backend config overrides.

---

## F. Next 50 things to do

### Critical (fix what's broken)
1. Fix `TestRun_DurationAborts` under `-race` — read-model phase should tolerate duration-aborted writes
2. Update CHANGELOG.md `[Unreleased]` with all 6 fixes from this session
3. Update FEATURES.md to reflect benchkit improvements
4. Annotate Pareto plan (`docs/planning/2026-07-24_17-59_benchkit-hardening-pareto-plan.md`) with completion status
5. Check and update `benchkit/README.md` for DiskSizer, CPU, projection changes

### High value (from the plan, not yet started)
6. P1-12: Phase 2 durability benchmark (crash recovery, replay-after-restart)
7. P1-13: Phase 6 production replay (replay real event streams)
8. P1-14: Phase 7 `benchtest.RunSuite` (Go `testing.B` wrappers)
9. Postgres benchmark tests (needs `POSTGRES_TEST_DSN`)
10. Analytical benchmark profiles (OLAP-style queries)

### Test hardening
11. Add race-clean version of `TestRun_DurationAborts` (or fix the underlying issue)
12. Add test for DiskSizer fallback path (DiskSize() returns -1, DiskPath filesystem walk used)
13. Add test for SQLite at `medium` profile (16 goroutines) to verify pool fix at scale
14. Add test for Pebble `DiskUsage()` accuracy vs filesystem `du`
15. Add benchmark test for projection with real kv.Store writes (not no-op handler)
16. Add test for `version()` function with `GOWORK=off` (module version resolution)
17. Add test for `compareWithDiskPaths` with 3 backends + failure isolation
18. Add test for CPU measurement consistency (start < end, delta > 0)
19. Add test for `codec_other.go` build on non-Unix (cross-compile check)
20. Add integration test: full `compare` run produces valid JSON with all fields populated

### Documentation
21. Update benchkit/README.md with DiskSizer section
22. Update benchkit/README.md with CPU measurement explanation (getrusage)
23. Update benchkit/README.md with projection benchmark section
24. Update AGENTS.md module table if benchkit description needs updating
25. Add benchkit to `docs/DOMAIN_LANGUAGE.md` if missing
26. Document the `-1` sentinel convention in DiskSizer docstring more prominently
27. Add ADR for CPU measurement approach (getrusage vs /proc vs runtime)
28. Cross-reference ADR-0060 from benchkit/README.md (currently only from cmd/cqrs-bench/README.md)

### Code quality
29. Consider `HasDiskSize() bool` instead of `-1` sentinel
30. Make projection benchmark handler configurable (allow consumers to plug in their own projection)
31. Add `Config.ProjectionHandler` field so consumers can benchmark their real projections
32. Consider `CompareConfig` type to replace `compareWithDiskPaths` duplication
33. Add Pebble `DiskUsage()` accuracy test (compare to `du -sb`)
34. Verify `cpu_unix.go` getrusage values match old /proc/self/stat values within tolerance
35. Add Prometheus metrics export from benchkit results (for CI regression gates)
36. Add `cqrs-bench run --repeat N` flag for multi-sample averaging (the event-size scaling doc shows ~20% variance)

### Infrastructure
37. Add `nix run .#bench` flake target for one-command benchmarking
38. Add GitHub Actions workflow for performance regression detection
39. Add benchkit to CI matrix (run `cqrs-bench compare --profile dev` in CI)
40. Set up `POSTGRES_TEST_DSN` in CI for Postgres benchmark tests
41. Tag `benchkit/v0.1.0` when API stabilizes (track in TODO_LIST)

### Features
42. Add `--json-schema` flag to export JSON schema of result format
43. Add `cqrs-bench diff` subcommand to compare two result JSON files
44. Add `cqrs-bench history` to track results over time (sqlite-backed)
45. Add memory profiling option (`--mem-profile` to write heap profiles)
46. Add CPU profiling option (`--cpu-profile` to write pprof files)
47. Add `--trace` flag to emit Go execution traces during benchmark
48. Add configurable warmup strategy (current: separate Bundle; alternatives: same-store, skip-reads)
49. Add `cqrs-bench validate` subcommand to check result plausibility (flag impossible numbers)
50. Add multi-codec comparison (auto-run JSON vs CBOR and report delta)

---

## G. Questions I CANNOT figure out myself

### G1: Should the `TestRun_DurationAborts` race failure be fixed by making the read-model phase error-tolerant, or by increasing the test's time budget?

The test uses `Duration: 5*time.Millisecond` with a 100K-stream profile. Under `-race`, the write phase is slow enough that the duration fires before all keys are written. The read-model phase then tries to Get keys that were never Set. Two approaches:
- **(A)** Make the read-model phase use keys that exist (load keys from the write phase's actual output, not the planned set)
- **(B)** Increase the duration to `50ms` or `100ms` and accept that the test is less precise under race
- I lean toward (A) because it's a real bug (the read-model phase assumes all writes succeeded), but it requires knowing which keys were actually written.

### G2: Should the Pebble `DiskUsage()` approximation be documented as approximate, or should I backport the real `DiskUsage()` from a newer Pebble version?

Pebble v1.1.5 (our pinned version) doesn't have `DB.DiskUsage()`. I compute it from Metrics level sizes + WAL. The newer Pebble (v2.x) has the real method. Should we:
- **(A)** Keep the approximation and document it ("computed from Metrics, may differ from `du` by <5%")
- **(B)** Upgrade the Pebble dependency to get the real method
- I lean toward (A) because upgrading Pebble is a bigger change with its own risk.

### G3: Should I squash the BuildFlow commits into a single clean commit before this work goes public?

The 7 auto-commits have misleading messages and bundle unrelated changes. The history reads like a mess. But squashing would require `git reset` (banned by safety rules) or `git rebase -i` (also risky). Should we:
- **(A)** Leave history as-is and rely on CHANGELOG for the accurate record
- **(B)** Create a single "merge" commit that describes the actual work, leaving the messy commits as parents
- **(C)** Something else?
