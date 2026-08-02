
> **Date:** 2026-08-02 21:19 CEST
> **Session scope:** Resume watcher reification fix, run full verify gate, add
>   cross-engine regression tests, documentation, metric tracking, CHANGELOG.
> **Honesty mode:** Brutal.

---

## a) FULLY DONE

1. **Verify gate doc-assertion failure fixed** — ADR-0091 (`sse-consolidation-decision`)
   was committed to `docs/adr/` but never indexed in `docs/README.md`. Added the
   index row and updated the ADR count text from "78 ADRs" to "89 ADRs".
   `scripts/verify-docs.sh` now passes.

2. **API stability golden regenerated** — Pre-existing stale golden from the
   DuckDB LayoutPlanner work (4 new exports: `ApplyLayout`, `BuildColumnarLayoutPlan`,
   `WithColumnarLayout`, `LayoutPlanApplier`). Ran `cd cmd/api-stability &&
   GOWORK=off go run . -update` to sync from 3187→3192 exports.

3. **cqrs-lint go.mod tidy** — `go mod tidy` was needed (stale `go-finding`
   pseudo-version). After tidy + commit, all 12 cqrs-lint rule sub-packages
   pass. The F009/F015/F017 "0 findings want 1" failures from verify were
   caused by the stale go.mod, not by my code.

4. **Pebble watcher regression tests** —
   `metaengine/pebbleengine/watcher_test.go` (2 tests):
   - `TestPebbleWatcher_DeleteNotificationDeliversZeroValue` — verifies delete
     delivers zero value, not silent drop.
   - `TestPebbleWatcher_WithReplayRecordsTypedValue` — verifies replay journal
     captures typed value, not seq=0.
   Both PASS. Required introducing a local `watcherTaskID` branded type to
   resolve the "ambiguous key" panic from `Remove[V]()` key inference.

5. **DuckDB watcher regression tests** —
   `metaengine/duckdbengine/watcher_cgo_test.go` (2 tests, `//go:build cgo`):
   - `TestDuckDBWatcher_DeleteNotificationDeliversZeroValue` — exercises the
     `map[string]any` reify fallback path (DuckDB returns decoded JSON maps).
   - `TestDuckDBWatcher_WithReplayRecordsTypedValue` — verifies replay journal
     records correctly via reify, not silent seq=0.
   Both PASS.

6. **Postgres watcher regression tests** —
   `metaengine/pgengine/watcher_test.go` (2 tests, testcontainers):
   - `TestPostgresWatcher_DeleteNotificationDeliversZeroValue`
   - `TestPostgresWatcher_WithReplayRecordsTypedValue`
   Both PASS against postgres:16-alpine via testcontainers.

7. **jsonValue fast-path test** — `TestReifyWatcherValue_JSONValueFastPath`
   in `metaengine/watcher_typesafe_test.go`. Feeds raw JSON bytes through
   `reifyWatcherValue[testTask]` and verifies single-pass decode to V. PASS.

8. **metaengine README updated** — Added two blockquote notes under the Watcher
   section: (a) delete notifications deliver zero value of V, (b) cross-engine
   representation (Memory typed, SQL `map[string]any`/`jsonValue`).

9. **metaengine COOKBOOK updated** — New "Watcher Patterns" section with three
   recipes: reactive map updates, delete notifications, cross-engine semantics.

10. **CHANGELOG entry added** — "Metaengine watcher reification and delete
    notifications" under `[Unreleased] → Fixed`. Covers: delete no longer
    silently dropped, cross-engine reify, replayShim fix, regression tests
    across 5 engines, documentation updates.

11. **projectionadapter reviewed** — Confirmed no dependency on old silent-drop
    behavior. The adapter calls `store.Apply()` and does not use `Watcher[V]`
    at all. No changes needed.

12. **Reification failure metric** — Added `reificationFailures atomic.Int64`
    to `workloadMeter` with `IncReificationFailure()` and `ReificationFailures()`
    accessors. Wired into both `Watch` and `WatchWithSeq` goroutines in `dx.go`
    — when `unwrapWatcherValue`/`unwrapWatcherSeqValue` returns `!ok`, the
    meter increments instead of silently continuing. The auto-commit daemon
    already committed this (`da48e988`).

---

## b) PARTIALLY DONE

1. **Reification failure test** — I wrote the test body
   (`TestWorkloadMeter_ReificationFailures`) but the final `edit` call to
   append it to `watcher_typesafe_test.go` failed because I used an empty
   `old_string` on an existing file (wrong tool pattern — should have used
   `view` first, then `edit` with the last line as anchor). The test was
   **never added**. The production code is committed and working; only the
   test is missing.

2. **Full verify gate re-run** — I ran verify once early in the session (found
   the ADR index + API golden + cqrs-lint issues). After fixing those and
   adding all the new tests + metric code, I did **not** re-run `nix run
   .#verify` to confirm the final state. The metaengine, pebbleengine,
   duckdbengine, and pgengine test subsets all pass individually, but the
   full gate (lint, race, doc-check, api-stability) was not re-verified
   end-to-end in this session after the metric wiring change. The daemon
   committed a "verify green" status report (`aca6274b`), but I cannot
   personally confirm that run included my latest changes.

3. **Type assertion audit** — Started by grepping for `.(V)` and `.(T)` patterns
   in the engine packages. Found that the main risk sites (`reify`, `reifyReflect`,
   `reflectCall1`) all use the `reify` fallback pattern already. The audit is
   incomplete for `duckdbengine/`, `pgengine/`, and `pebbleengine/` scan/pushdown
   paths which may have their own `.(map[string]any)` assertions.

---

## c) NOT STARTED

1. **`watcherEntry` generic refactor** — The original follow-up list suggested
   refactoring `watcherEntry` to `watcherEntry[V]` to eliminate `chan any`
   entirely. This would cascade into `subscriberHub` (non-generic) and `Store`
   (non-generic). Never started — correctly identified as a larger architectural
   change out of scope for the immediate fix.

2. **SSE + SQLite Last-Event-ID reconnect test** — A test named
   `TestSSE_LastEventID_Reconnect_SQLite` already exists at
   `sse_replay_test.go:725` (from a prior session). I verified it passes but
   did not write a new one. The original follow-up item was redundant.

3. **Benchmark for watcher notification latency** — Never attempted. The
   reify fallback adds a JSON round-trip on SQL engines; this should be
   measured but was not.

4. **Export `ReificationFailures()` on the public `Store` type** — Currently
   the method is on the unexported `workloadMeter`. Consumers cannot observe
   the counter without it being promoted to `Store` (or exposed via
   `WorkloadStats`). This was not done.

---

## d) TOTALLY FUCKED UP

1. **Failed edit on test file** — The very last action of the session was
   trying to append `TestWorkloadMeter_ReificationFailures` using `edit` with
   `old_string=""` (empty). The tool correctly rejected this ("file already
   exists"). I should have read the end of the file and used the last function
   as an anchor for the edit. This left a test gap — production code tracks
   failures but no test pins the counter behavior.

2. **dx.go indentation corruption** — When I used `multiedit` to restructure
   the `if ok {` blocks into `if !ok { continue }` early-return style, the
   nested `select` blocks ended up with wrong brace nesting (extra closing
   brace). `gofmt -w` fixed it silently, but I should have verified the
   structure with `go build` before moving on. The daemon commit `da48e988`
   includes the gofmt'd version, so the committed code is correct — but I
   almost shipped a syntax error.

3. **Trusted the daemon's "verify green" without re-running** — Commit
   `aca6274b` claims "verify green" at 21:17, but I was still making changes
   at 21:19. The daemon's status report may or may not include my latest
   metric wiring. This is the "stale GREEN" anti-pattern documented in
   AGENTS.md. I should have run `nix run .#verify` myself as the final action.

---

## e) WHAT WE SHOULD IMPROVE

1. **Always finish test before production** — The reification failure metric
   is production code without a test. This violates the testing mandate. The
   test body is trivially simple; it just needs to be properly appended.

2. **Use `view` before `edit` every time** — The failed edit was caused by
   not reading the end of the file first. The workflow says "read before you
   write" and I skipped it on the last step.

3. **Run `go build` after every structural edit** — The dx.go brace corruption
   would have been caught immediately by `go build`. I relied on gofmt to fix
   it silently, which is fragile.

4. **Never trust daemon verify claims** — The daemon commits "verify green"
   reports that may not reflect the current working tree. Always run the gate
   yourself in the same session that made changes.

5. **Promote `ReificationFailures()` to `Store` or `WorkloadStats`** — The
   counter is invisible to consumers. It should be either a `Store` method or
   a field in `WorkloadStats` so monitoring dashboards can alert on it.

6. **Consider `log.Warn` on reification failure** — A counter is passive; a
   log warning would make failures visible in operator logs without requiring
   a metrics dashboard. The `metaengine` package currently has no logger
   dependency (zero-dep core), so this would need an optional callback.

---

## f) Next Steps (up to 50)

### Immediate (this session's loose ends)
1. Add `TestWorkloadMeter_ReificationFailures` test (the one that failed to append)
2. Run `nix run .#verify` to confirm green after all changes
3. Verify the daemon's `aca6274b` "verify green" claim matches reality

### Short-term (next session)
4. Promote `ReificationFailures()` to `Store` method or add to `WorkloadStats`
5. Audit `.(map[string]any)` assertions in `duckdbengine/scan.go` and `pushdown.go`
6. Audit `.(map[string]any)` assertions in `pgengine/scan.go` and `pushdown.go`
7. Audit type assertions in `pebbleengine/raw_reader.go` and `scan.go`
8. Add `log.Warn` or callback hook for reification failures (optional logger)
9. Benchmark watcher notification latency: Memory vs SQLite vs Pebble
10. Consider `WithReificationFailureCallback(fn func(collection string, key any, value any, err error))` option

### Watcher hardening
11. Refactor `watcherEntry` to generic `watcherEntry[V]` to eliminate `chan any`
12. Make `subscriberHub` generic or split by collection type parameter
13. Add overflow counter for dropped notifications (consumer too slow)
14. Add per-collection watcher count metric
15. Document the "drop if consumer is slow" semantics in README
16. Add a `WatchBlocking` variant that blocks instead of dropping
17. Consider `WatchFiltered(ctx, key, filterFunc)` for predicate-based filtering

### Cross-engine parity
18. Add DuckDB watcher test with `FilterOnField` pushdown path
19. Add Postgres watcher test with `FilterOnField` pushdown path
20. Add Pebble watcher test with `RawValueReader` fast path
21. Add cross-engine test: same events → same watcher values across all engines
22. Test watcher behavior under concurrent `Apply` from multiple goroutines
23. Test watcher behavior under `SwapEngine` (does the old engine's watcher break?)

### SSE / Replay hardening
24. Add SSE reconnect test with delete notification in the replay window
25. Add SSE reconnect test with `jsonValue` fast-path values
26. Test `SSEReplay` ring buffer eviction under high write rate
27. Test `WithReplayByteBudget` interaction with watcher notification timing
28. Add `ServeSSE` test with `WithPayloadTransform` + CBOR events
29. Test SSE connection drop + reconnect with concurrent `ApplyBatch`

### Metaengine data model
30. Consider `ScanResult[T]` generic to replace `[]any` (breaking, needs major bump)
31. Add boundary key-type validation at Store boundary (not just fold time)
32. Consider `metaengine-gen` CLI for typed Store methods from query declarations
33. Add `Store.Validate()` method that checks all queries at registration time
34. Consider versioned schema for `jsonValue` wrapper (add a version byte?)

### Observability
35. Export watcher notification count per collection
36. Export replay journal depth and capacity utilization
37. Add OTel span for watcher notification dispatch (currently no tracing)
38. Add histogram for reify round-trip latency on SQL engines
39. Consider health check: `Store.HealthCheck()` includes reification failure rate

### Documentation
40. Document watcher lifecycle: create → watch → close → re-watch
41. Add ADR for watcher reification contract (delete = zero value)
42. Update SKILL.md references with watcher delete semantics
43. Add architecture diagram: event → fold → engine → notify → watcher → SSE
44. Document `reifyWatcherValue` contract in the package doc comment

### Cleanup
45. Remove unused `layoutComplexity` function (gopls diagnostic)
46. Remove unused `op` type in `property_test.go` (gopls diagnostic)
47. Fix 6 `gopls stdversion` warnings in `reify.go` (json v2 requires go1.27)
48. Address 3 `gopls waitgroupgo` hints (modernize to `wg.Go`)
49. Address 2 `gopls testingcontext` hints (modernize to `t.Context()`)
50. Address 2 `gopls infertypeargs` hints in `features4_test.go`

---

## g) Questions

1. **Should `ReificationFailures()` be promoted to the public `Store` type?**
   It's currently on the unexported `workloadMeter`, making it invisible to
   consumers. Promoting it would add a public API surface symbol (requires
   golden regen). Alternatively, it could be a field in `WorkloadStats` (no
   new method). I cannot decide this without knowing your preference for API
   surface growth vs. stats-struct extension.

2. **Should the watcher drop-on-slow-consumer behavior be configurable?**
   Currently both `Watch` and `WatchWithSeq` silently drop notifications when
   the consumer's buffered channel (cap=1) is full. An alternative is a
   blocking variant. I cannot infer whether consumers need blocking semantics
   or whether dropping is the right production default without your input.

3. **Is the `watcherEntry[V]` generic refactor worth the cascading scope?**
   Eliminating `chan any` requires making `subscriberHub` generic or splitting
   it by type parameter, which ripples into `Store.registerWatcher`. This is a
   medium-sized architectural change. I cannot determine if you want to pay
   that cost now or accept the `chan any` boundary as a permanent tradeoff.
