# Status Report: Flight Recorder Integration (Go 1.25)

**Date:** 2026-08-01 19:42
**Session scope:** Leveraging Go 1.25's `runtime/trace.FlightRecorder` in go-cqrs-lite
**Trigger:** User shared https://go.dev/blog/flight-recorder and asked how to leverage it

---

## Executive Summary

Created a new **`flightrecorder/`** module (Tier 0, zero-dependency) that wraps Go 1.25's `runtime/trace.FlightRecorder` with composable trigger conditions, plus CQRS middleware integration (`Command/Event/QueryFlightRecorder`). The module lets consumers capture execution trace snapshots when operations go wrong (slow commands, failed event handlers, erroring queries) for offline analysis with `go tool trace`.

**35 tests passing, 85.4% coverage, clean with `-race -count=3`.**

---

## a) FULLY DONE

| #   | Item                                                                                                              | Verification                         |
| --- | ----------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| 1   | `flightrecorder/` module created (`go.mod`, zero deps, stdlib only)                                               | `GOWORK=off go build ./...` ✓        |
| 2   | `Recorder` type: `New`, `Start`, `Stop`, `Enabled`, `Snapshot`, `SnapshotToFile`, `SnapshotIf`, `Reset`, `Writer` | 25 core tests ✓                      |
| 3   | `TriggerFunc` system: `OnLatency`, `OnError`, `OnErrorOrLatency`, `OnAlways`, `OnAny`, `OnAll`                    | 10 trigger tests ✓                   |
| 4   | Options: `WithMinAge`, `WithMaxBytes`, `WithWriter`, `WithFile` (lazy file creation)                              | Tested ✓                             |
| 5   | Once-semantics (first `Snapshot` wins; `Reset` for re-arm)                                                        | Tested with concurrent goroutines ✓  |
| 6   | Thread-safety: `sync.Mutex` around `WriteTo` to prevent `Stop`/`Snapshot` races                                   | `-race` clean ✓                      |
| 7   | CQRS middleware: `NewFlightRecorder[M]`, `CommandFlightRecorder`, `EventFlightRecorder`, `QueryFlightRecorder`    | 10 middleware tests ✓                |
| 8   | Middleware preserves original error, captures async (goroutine), logs on snapshot failure                         | Tested ✓                             |
| 9   | `go.work` updated with `./flightrecorder`                                                                         | Workspace resolves ✓                 |
| 10  | `middleware/go.mod` updated with replace directive for untagged module                                            | `go build` ✓                         |
| 11  | `cmd/api-stability/main.go` modules list updated                                                                  | `TestEveryGoModDirIsInModulesList` ✓ |
| 12  | `docs/api_surface.txt` regenerated (3117 exports, 29 new flightrecorder symbols)                                  | `go run .` ✓                         |
| 13  | `AGENTS.md` updated: Modules list, Monorepo tree, Tier 0 graph, Key Patterns example                              | Verified ✓                           |
| 14  | `flightrecorder/README.md` with quickstart, trigger table, CQRS middleware example                                | Written ✓                            |
| 15  | `flightrecorder/doc.go` with package-level usage docs                                                             | Written ✓                            |
| 16  | `gofumpt` + `goimports` clean on all new files                                                                    | Checked ✓                            |
| 17  | `go vet` clean on both modules                                                                                    | Checked ✓                            |

---

## b) PARTIALLY DONE

| #   | Item                             | What's missing                                                                                                                                                                                                                                                                                                                                                               |
| --- | -------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **AGENTS.md test command**       | `./flightrecorder/...` was supposed to be added to the `go test` command string in the Quick Reference table. The `multiedit` reported "Applied 2 of 3 edits" — this edit **FAILED silently**. The test command string does NOT include `./flightrecorder/...`.                                                                                                              |
| 2   | **middleware/go.sum**            | `flightrecorder` entry is missing from `go.sum` (0 matches). `go mod tidy` ran but didn't add it because the workspace `replace` directive resolves it locally. This works in the workspace but **breaks standalone `GOWORK=off` builds** of the middleware module if the local replace isn't present. Not critical (replace is in go.mod), but go.sum should still be tidy. |
| 3   | **AGENTS.md Dependencies table** | Not updated, but `flightrecorder` has zero production deps (stdlib only), so the table is technically correct without it. Still, it should be documented as "stdlib only" for discoverability.                                                                                                                                                                               |

---

## c) NOT STARTED

| #   | Item                                                     | Impact                                                                                                                                                                                                                                                                                                        |
| --- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **ADR for flight recorder design**                       | No ADR documenting the design decision (once-semantics, trigger composition, why zero-dep core + middleware integration split). Last ADR is 0088. Should be 0089.                                                                                                                                             |
| 2   | **SKILL.md + `.agents/skills/go-cqrs-lite/` references** | The AI consumer skill (SKILL.md + references/) has ZERO mentions of flightrecorder. Consumers asking AI agents about diagnostics won't know it exists. Need to add to `references/modules.md`, `references/core.md` (decision matrix), `references/advanced.md` (advanced patterns), `references/recipes.md`. |
| 3   | **`doc-check` verification**                             | AGENTS.md says to run `cmd/doc-check` after editing docs to verify Go import paths. Not run. The new AGENTS.md code examples reference `flightrecorder` and `middleware` packages — these need validation.                                                                                                    |
| 4   | **`.golangci.yml` depguard allowlist**                   | No `.golangci.yml` found in repo (grep returned 0). Not applicable, but should verify the lint config (wherever it lives) doesn't flag the new import.                                                                                                                                                        |
| 5   | **Integration test**                                     | No test showing the end-to-end workflow: start recorder → dispatch slow command → verify `.trace` file → verify it's parseable by `go tool trace` (or at least has valid trace header bytes).                                                                                                                 |
| 6   | **`projectionhost` integration**                         | No hook for projection failures (the `WithOnFailed` callback, DLQ threshold, worker restart) to trigger a flight recorder snapshot. This is a natural fit — projection poison messages are exactly the kind of problem flight recording diagnoses.                                                            |
| 7   | **`stack.Bundle` integration**                           | No `WithFlightRecorder` option on `stack.Bundle` for one-call setup. Consumer must wire it manually.                                                                                                                                                                                                          |
| 8   | **`cqrs-lint` F-series adoption rule**                   | No F-series rule coaching users toward flight recording (like the existing rules for tombstone, catalog, OTel, etc.).                                                                                                                                                                                         |
| 9   | **Example in `example/taskmanager`**                     | The flagship example doesn't demonstrate flight recording.                                                                                                                                                                                                                                                    |

---

## d) TOTALLY FUCKED UP

| #   | Issue                                                           | Severity | Status                                                                                                                                                                                                                                                                                                                                                                                          |
| --- | --------------------------------------------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **`Snapshot(ctx)` ignores `ctx`**                               | Medium   | The `context.Context` parameter on `Snapshot()` and `SnapshotIf()` is **never used**. It's accepted for future cancellation support but currently does nothing. This is a **lying API** — callers expect context cancellation to abort the snapshot, but it won't. Either implement context-aware cancellation or remove the parameter. The Go blog example doesn't use context at all.         |
| 2   | **`lazyFile.Close()` is unreachable**                           | Low      | The `lazyFile` type has a `Close()` method that is never called by any public API path. `WithFile` creates a `lazyFile` as the writer, but `Recorder` has no lifecycle hook to close it. The file handle leaks until process exit. For a single-snapshot use case this is tolerable, but it's still a resource leak.                                                                            |
| 3   | **Initial test attempt had parallel `runtime/trace` conflicts** | Resolved | First test run failed because Go only allows ONE active `runtime/trace.FlightRecorder` per process. Tests were using `t.Parallel()`. Fixed by serializing with `sync.Mutex`, but the **root constraint is not documented** in the package docs. Consumer code that creates two `Recorder` instances and calls `Start()` on both will get a confusing `"flight recorder already enabled"` error. |
| 4   | **Race condition in first iteration**                           | Resolved | `Snapshot()` initially called `r.Enabled()` (which acquires `r.mu`) then `r.fr.WriteTo()` (without holding `r.mu`), racing with `Stop()`. Fixed by holding `r.mu` for the entire `WriteTo` call. The fix is correct but the initial design was wrong — should have held the lock from the start.                                                                                                |

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Remove or implement `ctx` parameter** — `Snapshot(ctx)` and `SnapshotIf(ctx, ...)` accept but ignore `context.Context`. Either wire it into `WriteTo` cancellation (hard — Go's `FlightRecorder.WriteTo` doesn't accept context) or remove it from the signature (breaking change but more honest). Recommended: keep for API stability, document that it's reserved for future use.

2. **Document the process-global constraint** — `doc.go` should prominently warn that only ONE `runtime/trace.FlightRecorder` can be active per process. The `"flight recorder already enabled"` error from `Start()` should be wrapped with a clear message.

3. **`Recorder` should implement `io.Closer`** — `Stop()` exists but there's no `Close()`. For consistency with the rest of the codebase (`stack.Bundle.Close()`, `MultiCloser`, `FuncCloser`), `Recorder` should implement `io.Closer` so it can participate in shutdown ordering.

4. **`Writer()` method is unused** — The `Writer()` accessor exposes the configured writer but no test or consumer uses it. YAGNI candidate — remove unless needed.

5. **`SnapshotToFile` duplicates `Snapshot` logic** — Both methods implement the `once.Do` + lock + check-enabled pattern independently. Extract a shared `captureSnapshot(w io.Writer) error` helper.

### Architecture

6. **No projection host hook** — `projectionhost.Worker` has `WithOnFailed`, DLQ threshold, worker restart backoff, and `MetricsRecorder.EventErrored()`. None of these can trigger a flight recorder snapshot. The most valuable integration point (projection poison messages causing worker failure) is unwired.

7. **No stack.Bundle integration** — Every other diagnostic capability (OTel, Prometheus, health checks) has a one-call `stack.With*` option. Flight recorder doesn't. Consumer must manually create, start, and wire it.

8. **No decider/repository integration** — `decider.Repository[State].Execute()` is the hot path for event-sourced systems. A slow `Load` or `Save` call is a prime flight recorder trigger candidate, but there's no hook.

### Documentation

9. **SKILL.md is stale** — The AI consumer skill has no knowledge of flightrecorder. This is the single source of truth for AI agents using the library.

10. **No ADR** — Design decisions (once-semantics, trigger composition, zero-dep core + middleware split) are undocumented.

11. **Test command in AGENTS.md is incomplete** — `./flightrecorder/...` missing from the `go test` command string (edit failed silently).

---

## f) Up to 50 Things to Do Next

### Immediate (this session's loose ends)

1. Fix AGENTS.md test command — add `./flightrecorder/...`
2. Run `go mod tidy` on middleware and verify go.sum
3. Add flightrecorder to SKILL.md + references/modules.md
4. Add flightrecorder to references/core.md decision matrix
5. Add flightrecorder to references/advanced.md (diagnostics pattern)
6. Add flightrecorder to references/recipes.md (copy-paste recipe)
7. Run `cmd/doc-check` to verify all import paths in docs
8. Write ADR-0089: Flight Recorder Design
9. Document process-global constraint in doc.go
10. Wrap `Start()` error with clearer message on "already enabled"

### Code improvements

11. Extract shared `captureSnapshot` helper (dedup Snapshot/SnapshotToFile)
12. Remove unused `Writer()` method (YAGNI)
13. Consider implementing `io.Closer` on `Recorder`
14. Consider removing `ctx` from `Snapshot`/`SnapshotIf` (or document as reserved)
15. Handle `lazyFile.Close()` — add lifecycle hook or document the leak
16. Add `ErrAlreadyEnabled` sentinel error for double-`Start()`
17. Add `Recorder.SnapshotToWriter(w io.Writer)` for one-off writer override
18. Add integration test verifying trace file is valid (parseable header)
19. Add benchmark: snapshot capture latency overhead
20. Add benchmark: middleware overhead when trigger doesn't fire (should be ~0)

### Deeper integrations

21. Add `projectionhost.WithFlightRecorder(recorder, trigger)` option
22. Wire `projectionhost.OnFailed` → flight recorder snapshot
23. Wire `projectionhost.DLQ` threshold → flight recorder snapshot
24. Add `decider.WithFlightRecorder[State](recorder, trigger)` repository option
25. Add `stack.WithFlightRecorder(recorder, trigger)` Bundle option
26. Add `middleware.OTelBundle` extension: `WithFlightRecorder(recorder)`
27. Add `scheduling.Scheduler` flight recorder hook for failed timer dispatches
28. Add `metaengine.WithFlightRecorder` hook for slow queries (complements `WithSlowQueryLog`)
29. Add `transport/http` SSE broker hook for slow event delivery

### cqrs-lint adoption rules

30. F-series rule: detect missing flight recorder on production middleware chains
31. F-series rule: coach `OnErrorOrLatency` as the recommended default trigger
32. F-series rule: detect `projectionhost` without flight recorder on critical projections

### Testing & quality

33. Add `flightrecorder` to `nix run .#verify` gate verification
34. Add `flightrecorder` to coverage drift check (`scripts/check-coverage.sh`)
35. Add `flightrecorder` to duplication baseline if needed
36. Run `nix fmt` on all new files (scoped format)
37. Add `flightrecorder` to the flake.nix module list if one exists
38. Verify `nix run .#build` includes flightrecorder
39. Verify `nix run .#lint` passes on flightrecorder
40. Add `flightrecorder/race_on.go` + `race_off.go` if timing-sensitive tests need relaxed thresholds under `-race`

### Documentation & examples

41. Add flight recording section to example/taskmanager README
42. Add flight recording to the README.md top-level feature list
43. Add to FEATURES.md
44. Add to TODO_LIST.md (mark flight recorder as DONE, integrations as PLANNED)
45. Add to CHANGELOG.md
46. Write a blog-style guide: "Diagnosing CQRS latency with Go 1.25 flight recording"
47. Add to docs/architecture-understanding/ if diagnostic tier docs exist
48. Add D2 diagram showing trigger → snapshot → trace analysis flow

### Future considerations

49. Explore `go tool trace` programmatic parsing (Go issue #62627) for automated analysis
50. Consider `gotraceui` integration for non-browser trace analysis

---

## g) Questions I Cannot Answer Myself

### 1. Should `Snapshot` keep the `ctx` parameter?

`runtime/trace.FlightRecorder.WriteTo()` does NOT accept `context.Context`, so I cannot implement real cancellation. The parameter is currently a lie. Options:

- **A)** Remove `ctx` from `Snapshot`/`SnapshotIf` — honest but breaking if I add it back later
- **B)** Keep `ctx`, document as "reserved for future use when Go's API supports it"
- **C)** Implement partial cancellation — check `ctx.Done()` before/after `WriteTo` but not during

I chose B by default, but this is a design decision that affects the public API surface permanently.

### 2. Should flight recorder snapshot files include structured metadata (timestamp, trigger reason, operation type)?

Right now `Snapshot()` just dumps the raw trace bytes. But a consumer debugging a slow command might want to know WHICH command triggered the capture, what the duration was, and what error occurred. Options:

- **A)** Keep it simple — raw trace only (current)
- **B)** Add `SnapshotWithMetadata(ctx, TriggerContext)` that writes a JSON sidecar file
- **C)** Wrap the trace in a container format (trace + metadata header)

This affects the consumer workflow significantly.

### 3. How should projection host integration work — automatic or opt-in?

`projectionhost` has multiple failure signals: per-event error (retryable), DLQ threshold exceeded, worker restart, terminal failure. Should flight recording:

- **A)** Fire on ALL errors (could be very noisy with poison messages)
- **B)** Fire only on terminal failure (`OnFailed` callback — rare, high-signal)
- **C)** Fire on DLQ threshold (first poison message that exceeds retry budget)
- **D)** Let the consumer configure the trigger (most flexible, most work for consumer)

This determines whether it's a `WithFlightRecorder(recorder, trigger)` option or hardcoded to a specific signal.

---

## Files Created/Modified This Session

| File                                | Action       | Lines                                         |
| ----------------------------------- | ------------ | --------------------------------------------- |
| `flightrecorder/go.mod`             | Created      | 3                                             |
| `flightrecorder/doc.go`             | Created      | 37                                            |
| `flightrecorder/recorder.go`        | Created      | 178                                           |
| `flightrecorder/options.go`         | Created      | 108                                           |
| `flightrecorder/trigger.go`         | Created      | 108                                           |
| `flightrecorder/recorder_test.go`   | Created      | 410                                           |
| `flightrecorder/trigger_test.go`    | Created      | 189                                           |
| `flightrecorder/README.md`          | Created      | 65                                            |
| `middleware/flightrecorder.go`      | Created      | 103                                           |
| `middleware/flightrecorder_test.go` | Created      | 352                                           |
| `middleware/go.mod`                 | Modified     | +2 lines (require + replace)                  |
| `go.work`                           | Modified     | +1 line                                       |
| `cmd/api-stability/main.go`         | Modified     | +1 line                                       |
| `docs/api_surface.txt`              | Regenerated  | 3117 exports                                  |
| `AGENTS.md`                         | Modified     | +15 lines (tree, tier, patterns, module list) |
| **Total**                           | **16 files** | **~1600 lines**                               |

---

## Resolution (2026-08-03)

`flightrecorder/` module shipped (Recorder, TriggerFunc, options, once-semantics). CQRS middleware (command/event/query), decider/projectionhost/stack integration all wired. ADR-0089 written. SKILL.md + all reference files updated. Coverage 85.4%→92.5%.

**Open items from this report — resolved by `20-44`:** `ErrAlreadyEnabled` sentinel added, `io.Closer` on Recorder, `ctx` pre-check in Snapshot, unused `Writer()` removed, `lazyFile.Close()` leak fixed. ADR-0089 written. End-to-end integration test added.
