# Status Report: Flight Recorder — Lint & Cleanup Session

**Date:** 2026-08-01 22:00
**Session scope:** Execute the remaining gap-closure items from `2026-08-01_20-44_flight-recorder-gap-closure.md` — rename misleading methods, split oversized file, update stale README, fix struct alignment, run formatters, fix all lint findings, fix a race condition found by the verify gate.
**Previous session:** Created the flight recorder module + integrations but left formatting/lint/race issues.

---

## a) FULLY DONE

| #   | Item                                                                                                                                                              | Verification                          |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| 1   | **Renamed `captureLocked`/`captureToFileLocked`** → `capture`/`captureToFile`                                                                                     | Build + tests pass ✓                  |
| 2   | **Split `stack/options.go`** (354→334 lines) — moved `WithDiskSize` + `DiskSize()` to `accessors.go`                                                              | Build + stack tests pass ✓            |
| 3   | **Updated `flightrecorder/README.md`** — `Close()` instead of `Stop()`, `ErrAlreadyEnabled`, `SnapshotToFile(ctx, path)`, lifecycle table, snapshot methods table | Content verified ✓                    |
| 4   | **Fixed struct field alignment** — gofumpt aligned `flightRecorder`/`flightRecorderTrigger` in decider + projectionhost                                           | `nix fmt` clean ✓                     |
| 5   | **Ran `gofumpt -w` + `goimports -w`** on all modified files                                                                                                       | No diff after formatting ✓            |
| 6   | **Ran `nix fmt`** twice (whole repo) — 11 files changed first run, 2 second run, 0 third                                                                          | Clean ✓                               |
| 7   | **Ran `nix run .#verify`** — found 1 race condition in my code + pre-existing metaengine failures                                                                 | Race fixed, verify re-run ✓           |
| 8   | **Fixed projectionhost race condition** — moved `captureFlightRecorder` before `setStatus(WorkerFailed)` so snapshot completes before test observes status        | Stable across 10× `-count=10 -race` ✓ |
| 9   | **Fixed all lint issues in flightrecorder** (10→0) — sentinel errors for err113, wrapped `os.Create` for wrapcheck, gosec nolint                                  | `nix run .#lint`: 0 issues ✓          |
| 10  | **Fixed all lint issues in decider** (2→0) — moved `//nolint:nonamedreturns` to directive position above func                                                     | `nix run .#lint`: 0 issues ✓          |
| 11  | **Fixed all lint issues in projectionhost** (3→0) — passed `ctx` parameter (contextcheck), renamed `fr`→`recorder` (varnamelen), exhaustruct nolint               | `nix run .#lint`: 0 issues ✓          |
| 12  | **Fixed all lint issues in middleware** (1→0) — inline nolint for named return                                                                                    | `nix run .#lint`: 0 issues ✓          |
| 13  | **Regenerated API surface golden** (3122→3161 exports — metaengine exported new symbols from pre-existing changes)                                                | `api-stability`: 3161 OK ✓            |
| 14  | **All flight recorder tests pass** with `-race` across flightrecorder, decider, projectionhost, middleware, stack                                                 | Full test run ✓                       |
| 15  | **Coverage maintained at 91.7%**                                                                                                                                  | `go test -cover` ✓                    |

---

## b) PARTIALLY DONE

| #   | Item                             | What's missing                                                                                                                                                                                                                                                                                                                                                              |
| --- | -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **`nix run .#verify` full pass** | The integrated gate was run but does NOT pass due to **pre-existing metaengine failures** (71 of 161 Ginkgo specs failing with `queryMeta` interface errors — from uncommitted metaengine changes by the auto-commit daemon, NOT from my flight recorder work). My modules all pass individually. The metaengine failures existed before this session and are out of scope. |
| 2   | **Lint gate**                    | `nix run .#lint` has 0 issues in ALL my modules (flightrecorder, decider, projectionhost, middleware, stack). However, **benchkit has 32 pre-existing lint issues** and **metaengine has 11 pre-existing lint issues** — both from auto-commit daemon changes, not mine.                                                                                                    |

---

## c) NOT STARTED

These items were in the previous status report's "Up to 50 Things" list and remain unaddressed (correctly deprioritized — they're enhancements, not blockers):

| #   | Item                                                                                | Impact                       |
| --- | ----------------------------------------------------------------------------------- | ---------------------------- |
| 1   | Integration test: verify captured `.trace` file has valid trace header bytes        | Confidence                   |
| 2   | Benchmarks: snapshot capture latency, middleware overhead when trigger doesn't fire | Performance characterization |
| 3   | Additional decider tests: latency trigger path, `OnErrorOrLatency` composite        | Coverage                     |
| 4   | `CHANGELOG.md` / `FEATURES.md` / `TODO_LIST.md` entries                             | Living docs                  |
| 5   | `scheduling.Scheduler` flight recorder hook                                         | Deeper integration           |
| 6   | `transport/http` SSE broker hook                                                    | Deeper integration           |
| 7   | cqrs-lint F-series adoption rules for flight recorder                               | Coaching                     |
| 8   | `example/taskmanager` flight recorder demo                                          | Example                      |
| 9   | `flightrecorder` added to `nix run .#check-coverage` coverage drift check           | CI gate                      |

---

## d) TOTALLY FUCKED UP

| #   | Issue                                                                                                                                                                                                                                                                          | Severity                                  | How I Fixed It                                                                                                                                                                                                                                          |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Projectionhost race condition** — `captureFlightRecorder` was called AFTER `setStatus(WorkerFailed)`, so the test could observe `WorkerFailed` and check the buffer before the snapshot completed. This caused `TestHost_FlightRecorder_CapturesOnTerminalFailure` to flake. | **High** — test flaked on the verify gate | Moved `captureFlightRecorder(ctx, err)` before `setStatus(WorkerFailed)`. Verified stable across 10× runs with `-race`. **This was a real bug in the previous session's code that I should have caught during code review, not waited for CI to find.** |
| 2   | **Lint nolint placement on decider** — I initially put `//nolint:nonamedreturns` inline on the closing paren `) (execErr error) {`. But golangci-lint reports the issue on the `func` line (line 111), so the inline nolint on line 116 was "unused" (nolintlint).             | Low                                       | Moved to a directive comment above the function. **I should have known golangci-lint directive placement rules.**                                                                                                                                       |
| 3   | **Initial flightrecorder lint fixes used too many nolint directives** — first pass slapped `//nolint:err113` on dynamic errors instead of properly extracting sentinel error variables. This is a code smell — the linter exists for a reason.                                 | Medium                                    | Second pass properly extracted `errMinAgeMustBePositive` and `errMaxBytesMustBePositive` sentinels, and wrapped `os.Create` errors. **I should have done this right the first time instead of reaching for nolint as a quick fix.**                     |

---

## e) WHAT WE SHOULD IMPROVE

### Process failures this session

1. **Did not catch the projectionhost race condition during initial code review** — The previous session wrote `setStatus(WorkerFailed)` then `captureFlightRecorder(err)`. This is a textbook TOCTOU race: the test waits for `WorkerFailed`, then checks the buffer, but the snapshot hasn't run yet. I should have caught this when reviewing the previous session's code, not when the verify gate caught it. **The ordering is now: capture → setStatus → recordMetric → onFailed → logger. This is the correct order — capture the diagnostic before announcing the failure.**

2. **Reached for nolint directives before trying the right fix** — First lint pass used `//nolint:err113`, `//nolint:wrapcheck`, `//nolint:gosec` as quick patches. Second pass properly fixed the root causes (sentinel errors, error wrapping). This wasted a full lint cycle (~3 minutes). **Lesson: always try the proper fix first, nolint is the last resort.**

3. **Did not verify the `nix run .#verify` gate in the previous session** — The race condition existed since the previous session created the projectionhost integration. If verify had been run then, the race would have been caught immediately. The "stale GREEN" anti-pattern from AGENTS.md is relevant here.

### Architecture observations

4. **The projectionhost `captureFlightRecorder` method signature changed from `(failedErr error)` to `(ctx context.Context, failedErr error)`** — This is an improvement (passes the worker's context instead of creating a new `context.Background()`), but it means the snapshot can be cancelled if the worker context is cancelled. In practice this is fine since terminal failure means the worker is shutting down anyway, and the snapshot's `WriteTo` is fast.

5. **The sentinel error pattern in flightrecorder/options.go** (`errMinAgeMustBePositive`, `errMaxBytesMustBePositive`) follows the project convention but these are unexported. Consumers can't `errors.Is` them. This is intentional for validation errors that are programmer bugs (not runtime conditions), but worth noting.

6. **The `stack/options.go` split was the simplest possible** — moved `WithDiskSize` + `DiskSize()` (19 lines) to `accessors.go`. This brought options.go from 354→334 lines. An alternative would have been to split lifecycle options (`WithMetaEngine`, `WithFlightRecorder`, `WithCloser`, `WithDrainer`) into a separate `options_lifecycle.go`. But the simple move is lower risk and keeps all options in one mental location.

---

## f) Up to 50 Things to Do Next

### Immediate (blocking full verify GREEN)

1. **Fix pre-existing metaengine test failures** — 71 of 161 Ginkgo specs fail with `query does not implement queryMeta` errors. The `queryMeta` interface in `metaengine/query.go` was changed by the auto-commit daemon (uncommitted working tree changes to `metaengine/advanced.go`, `metaengine/query.go`, `metaengine/rule_layout.go`, `metaengine/stats.go`, `metaengine/temporal.go`). These changes broke the `QueryDecl` type's interface conformance. This is NOT flight-recorder-related.
2. **Fix pre-existing metaengine lint issues** (11 issues: 10 godot + 1 interfacebloat)
3. **Fix pre-existing benchkit lint issues** (32 issues: modernize, wsl_v5, err113, etc.)
4. **Run `nix run .#verify` again** once metaengine is fixed — should achieve full GREEN

### Testing improvements

5. Add integration test: capture trace → verify file has valid trace header bytes
6. Add benchmark: snapshot capture latency overhead (WriteTo cost)
7. Add benchmark: middleware overhead when trigger doesn't fire (should be ~0)
8. Add decider test for latency trigger path (not just error trigger)
9. Add decider test for `OnErrorOrLatency` composite trigger
10. Add stack test that verifies `Start()` → `Close()` lifecycle (not just `Close()` on unstarted recorder)
11. Add test: double `Start()` returns `ErrAlreadyEnabled` then first recorder still works
12. Add test: `Reset()` + `Snapshot()` fires again (once-semantics re-arm)

### Documentation

13. Add flight recorder to `CHANGELOG.md`
14. Add flight recorder to `FEATURES.md` (DONE status)
15. Add flight recorder to `TODO_LIST.md` (mark integrations as PLANNED: scheduling, transport)
16. Verify `references/faq.md` doesn't need a flight recorder entry
17. Add flight recorder section to `example/taskmanager` README
18. Add flight recording demo to `example/taskmanager` code

### Deeper integrations

19. Add `scheduling.Scheduler` flight recorder hook for failed timer dispatches
20. Add `transport/http` SSE broker hook for slow event delivery
21. Add `middleware.OTelBundle` extension: `WithFlightRecorder(recorder)`
22. Add `metaengine.WithFlightRecorder` hook for slow queries
23. Explore `deriver` flight recorder hook for derivation failures

### cqrs-lint adoption rules

24. F-series rule: detect missing flight recorder on production middleware chains
25. F-series rule: coach `OnErrorOrLatency` as the recommended default trigger
26. F-series rule: detect `projectionhost` without flight recorder on critical projections
27. D-series rule: detect inconsistent trigger usage across command/event/query middleware

### Operational concerns

28. Document snapshot file rotation strategy (once-semantics means one file per process without Reset)
29. Consider `SnapshotToFileWithMetadata` that writes a JSON sidecar with trigger context
30. Consider structured trace naming: `trace_<timestamp>_<projection>_<error>.trace`
31. Consider `gotraceui` integration for non-browser trace analysis

### Quality gates

32. Add `flightrecorder` to `nix run .#check-coverage` coverage drift check
33. Verify `nix run .#test` includes `./flightrecorder/...` in the test command
34. Add flight recorder contract test pattern (like `contracttest.RunSuite`)

### Future design questions

35. Should the recorder auto-reset on `Snapshot` for periodic capture mode?
36. Should `stack.WithFlightRecorder` auto-`Start()` the recorder?
37. Should there be a `MultiRecorder` to work around process-global constraint?
38. Consider exposing `runtime/trace.WithRegion` / `runtime/trace.WithTask` for user-code trace annotations

---

## g) Questions I Cannot Answer Myself

### 1. Should the pre-existing metaengine failures be fixed in this session or a separate session?

The metaengine `queryMeta` interface changes are from the auto-commit daemon and are **uncommitted working tree changes** (`metaengine/advanced.go`, `metaengine/query.go`, `metaengine/rule_layout.go`, `metaengine/stats.go`, `metaengine/temporal.go`). They predate my session and are unrelated to flight recording. Fixing them would require understanding the intended `queryMeta` interface evolution, which I don't have context on. **Should I attempt to fix these, or leave them for a session that has metaengine context?**

### 2. Should the `errMinAgeMustBePositive` / `errMaxBytesMustBePositive` sentinels be exported?

Currently they're unexported because they represent programmer errors (invalid configuration), not runtime conditions consumers should match on. But the project convention in AGENTS.md says "Sentinel errors: `errors.New` in `errors.go` files" — these are in `options.go`, not `errors.go`. **Should these move to an `errors.go` file, or is `options.go` correct since they're validation-specific?**

### 3. Should `stack.WithDiskSize` + `DiskSize()` have stayed in `options.go`?

The split was necessary to get under the 350-line limit, but `WithDiskSize` is an Option function and `DiskSize()` is a Bundle method — semantically they belong with options and accessors respectively. Moving them to `accessors.go` groups them with other Bundle accessors, which is arguably more correct. But it splits "all Option functions" across two files. **Is this the right split, or should I have extracted lifecycle options (`WithMetaEngine`, `WithFlightRecorder`, `WithCloser`) to a new file instead?**

---

## Files Modified This Session

All changes are **uncommitted** (auto-commit daemon may pick them up):

| File                           | Changes                                                                                                                     | Lines changed |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------- | ------------- |
| `flightrecorder/recorder.go`   | Renamed `captureLocked`→`capture`, `captureToFileLocked`→`captureToFile`; removed unused nolint directive                   | 2             |
| `flightrecorder/options.go`    | Extracted sentinel errors (`errMinAgeMustBePositive`, `errMaxBytesMustBePositive`); wrapped `os.Create` error in `openFile` | 17            |
| `flightrecorder/README.md`     | Full rewrite: `Close()` instead of `Stop()`, `ErrAlreadyEnabled`, `SnapshotToFile(ctx, path)`, lifecycle + snapshot tables  | Full file     |
| `decider/decider.go`           | Added `//nolint:nonamedreturns` directive above `Execute` (moved from inline)                                               | 2             |
| `middleware/flightrecorder.go` | Added inline `//nolint:nonamedreturns` on closure signature                                                                 | 2             |
| `projectionhost/worker.go`     | Fixed race: `captureFlightRecorder` before `setStatus`; added `ctx` parameter; renamed `fr`→`recorder`; exhaustruct nolint  | 12            |
| `stack/options.go`             | Removed `WithDiskSize` + `DiskSize()` (moved to accessors.go)                                                               | -19           |
| `stack/accessors.go`           | Added `WithDiskSize` + `DiskSize()`                                                                                         | +19           |

**Total: 5 modified + 1 rewritten + 2 split = 8 files touched**
