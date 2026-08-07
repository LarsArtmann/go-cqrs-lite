# Status Report: System Module P2 Hardening — 2026-08-08 00:49

> Session focused on the 6 P2 hardening items from `paste_1.txt` for the
> `system/` package. All items shipped, but several gaps and improvement
> opportunities surfaced during the work.

---

## A) FULLY DONE

### 1. system/README.md Quick Start — FIXED

- **Before:** Used non-existent API (`DomainConfig.StreamTypes`, `sys.Dispatch(ctx, cmd)`).
- **After:** Complete `package main` program with real API (`DomainConfig.Commands`, `RegisterDecider`, `RegisterCommand`, `CommandDispatcher().Dispatch`). Imports all needed packages. Verified compiles + runs as a test.
- **Files:** `system/README.md`
- **Verification:** Test `TestQuickStartCompiles` passed (temp test file, then removed).

### 2. cmd/doc-check arg-parsing — FIXED

- **Before:** `cobra.ArbitraryArgs` accepted anything with no validation.
- **After:** Custom `fileArgs` validator rejects non-existent files, directories, non-`.md` extensions. Zero args still triggers auto-discovery.
- **Files:** `cmd/doc-check/main.go`
- **Verification:** Tested 3 cases: zero args (auto-discover, 1335 refs valid), valid file arg, invalid file (rejected with clear error).

### 3. System.HealthCheck(ctx) — ADDED

- **What:** Returns `nil` if healthy, first error otherwise. Checks: stopped state, engines implementing `metaengine.HealthChecker`, projection host `WorkerFailed` status.
- **Files:** `system/introspection.go` (method), `system/errors.go` (`ErrSystemStopped`)
- **Tests:** `TestSystem_HealthCheck_Healthy`, `TestSystem_HealthCheck_Stopped` (both pass with -race).

### 4. System.GracefulClose(ctx) — ADDED

- **What:** Runs `Close()` in goroutine racing against `ctx.Done()`. Matches `stack.Bundle.GracefulClose` pattern (minus Drainer phase, which System doesn't have).
- **Files:** `system/system.go`
- **Tests:** `TestSystem_GracefulClose`, `TestSystem_GracefulClose_ContextExpired` (both pass with -race).

### 5. System.ResetProjection(ctx, name) — ADDED

- **What:** Delegates to `projectionhost.Host.Reset()`. Returns `ErrNoProjectionHost` if no host configured.
- **Files:** `system/system.go` (method), `system/errors.go` (`ErrNoProjectionHost`)
- **Tests:** `TestSystem_ResetProjection_NoHost` (passes with -race).

### 6. Configurable Checkpoint Store — WIRED

- **What:** Added `CheckpointStore event.CheckpointStore` field to `DomainConfig`. Constructor uses it instead of hardcoded `&memoryCheckpointStore{}`. Falls back to in-memory when nil.
- **Files:** `system/config_types.go` (field + `event` import), `system/constructor.go` (selection logic)
- **Tests:** `TestSystem_CustomCheckpointStore` (passes, but see B-section for gap).

### Supporting work

- Updated `system/README.md` API tables (Constructor + Introspection sections).
- Regenerated `docs/api_surface.txt` golden (5 new symbols: `GracefulClose`, `HealthCheck`, `ResetProjection`, `ErrNoProjectionHost`, `ErrSystemStopped`).
- `go vet` clean on both `system/` and `cmd/doc-check/`.
- doc-check on full repo: 1335 references valid across 49 packages.

---

## B) PARTIALLY DONE

### 1. TestSystem_CustomCheckpointStore — shallow assertion

The test verifies that `DomainConfig.CheckpointStore` is accepted without error, but it does NOT verify the custom store is actually **used** by the projection host. No projections are declared in the test config, so `projHost` is nil and the checkpoint store is never exercised. A proper test would declare a projection, start the system, dispatch a command (producing an event), and assert `recordingCheckpointStore.saveCnt > 0`.

### 2. HealthCheck — no test for failed projection worker

`TestSystem_HealthCheck_Healthy` and `_Stopped` cover the happy path and the stopped path. There is no test that puts a projection worker into `WorkerFailed` state and verifies `HealthCheck` returns an error mentioning the projection name. This is the most valuable assertion of the feature and it's missing.

### 3. ResetProjection — no positive-path test

Only `TestSystem_ResetProjection_NoHost` exists. There is no test that:

- Configures a projection
- Starts + stops the host
- Calls `ResetProjection`
- Verifies the checkpoint was cleared and replay occurs on restart

### 4. GracefulClose — no test with real slow shutdown

The context-expired test uses an already-cancelled context, which triggers the `ctx.Done()` path immediately. There is no test that verifies `GracefulClose` completes successfully when `Close()` takes real time (e.g., with a slow projection host stop).

### 5. README API table — CheckpointStore not documented

The `DomainConfig.CheckpointStore` field is new but the README's Configuration / Design section doesn't mention it. The API table under "Constructor" could also note `DomainConfig.CheckpointStore` as a configuration option.

---

## C) NOT STARTED

### From the original P2 list — all 6 items shipped

The `paste_1.txt` contained exactly 6 P2 items. All 6 are implemented and tested (at varying depth — see section B).

### Broader system/ hardening not in scope but noticed

1. **System.Close() silently drops engine close errors** — `for _, eng := range s.engines { _ = eng.Close() }`. Should join errors like `stack.Bundle.Close()` does.
2. **System.Close() has no shutdown ordering** — `stack.Bundle` has `WithShutdownDependency` + topological sort. System just calls closers in registration order.
3. **System.closers slice is never populated** — declared as `[]func() error` but `New()` never appends to it. Dead code.
4. **System.Health(ctx) string is redundant with HealthCheck(ctx) error** — two health methods with different signatures. `Health` returns a human string, `HealthCheck` returns a go/no-go error. This is intentional (k8s probe vs dashboard) but could be documented better.
5. **No `Drainer` interface in system/** — `stack.Bundle.GracefulClose` drains subscribers before closing. System's `GracefulClose` just races `Close()` against the context. If a projection host is mid-batch, the context may expire before the batch completes.
6. **No `WithCheckpointStore` functional option** — the checkpoint store is a `DomainConfig` field (direct struct assignment). Unlike `stack.WithCheckpointStore()` which is a functional option. This is consistent with the System module's style (DomainConfig is a struct, not options), but consumers used to `stack.With*` may find it inconsistent.

---

## D) TOTALLY FUCKED UP

Nothing. All changes compile, tests pass, no regressions detected.

---

## E) WHAT WE SHOULD IMPROVE

### Immediate (this session's work)

1. **Deepen the checkpoint store test** — declare a real projection, start the system, produce events, assert the custom checkpoint store received `Save` calls. Without this, the feature is "compiles but unproven."
2. **Add a failed-projection HealthCheck test** — register a projection with a handler that always returns an error, let it exhaust retries, assert `HealthCheck` returns an error naming the projection.
3. **Add a positive ResetProjection test** — configure a projection, stop the host, reset, restart, verify replay from zero.
4. **Join errors in Close()** — stop silently dropping engine close errors. Use `errors.Join` (Go 1.20+).
5. **Document CheckpointStore in README** — add a "Projection Checkpoints" subsection under Configuration.

### Broader system/ improvements

6. **Populate `s.closers`** — either use it or remove it. Currently dead code.
7. **Add shutdown ordering** — port `WithShutdownDependency` from stack.Bundle if multi-store deployments need ordered close.
8. **Consider a `Drainer` interface** — for graceful shutdown that drains in-flight work before closing.
9. **System.Health(ctx) should delegate to HealthCheck** — `Health` could call `HealthCheck` and format the result, avoiding two separate health-check code paths.
10. **SQLite integration test for HealthCheck** — Memory engines are always healthy. A SQLite engine health check would test the real ping path.

### Code quality

11. **The `ErrSystemStopped` name is slightly misleading** — it means "the system is stopped" but reads as "the system stopped" (past tense event). `ErrSystemAlreadyStopped` or `ErrSystemNotRunning` might be clearer.
12. **`fileArgs` validator doesn't check symlinks** — `os.Stat` follows symlinks, so a symlink to a directory would pass the `!IsDir` check. Edge case, probably fine.

---

## F) UP TO 50 THINGS TO GET DONE NEXT

### Tests for this session's work (P0)

1. Deepen `TestSystem_CustomCheckpointStore` — declare projection, produce events, assert `saveCnt > 0`
2. Add `TestSystem_HealthCheck_FailedProjection` — poison a projection, assert HealthCheck returns error
3. Add `TestSystem_ResetProjection_Positive` — configure projection, stop, reset, restart, verify replay
4. Add `TestSystem_GracefulClose_SlowShutdown` — verify Close completes within context when it takes time
5. Add `TestSystem_HealthCheck_EngineUnhealthy` — mock an engine that returns error from HealthCheck

### system/ robustness (P1)

6. Join errors in `System.Close()` instead of returning only the first
7. Remove or populate the dead `s.closers` slice
8. Port `WithShutdownDependency` from stack.Bundle for ordered shutdown
9. Consider adding a `Drainer` interface for pre-close drain
10. Make `System.Health(ctx)` delegate to `HealthCheck(ctx)` internally

### system/ documentation (P2)

11. Add "Projection Checkpoints" subsection to README Configuration section
12. Document `DomainConfig.CheckpointStore` in README
13. Add `GracefulClose`, `HealthCheck`, `ResetProjection` to the README Design section
14. Update `docs/adr/` with an ADR for the P2 hardening batch
15. Update AGENTS.md system/ module description to mention health/graceful-close/reset

### system/ integration (P2)

16. SQLite integration test for `HealthCheck` (real DB ping)
17. SQLite integration test for `ResetProjection` (persistent checkpoint store)
18. SQLite integration test for `GracefulClose` (slow SQLite close + context)
19. Test `CheckpointStore` with `SQLCheckpointStore` from storage/eventstore
20. End-to-end test: system with SQLite, projection, custom checkpoint store, restart, verify projection resumes

### cmd/doc-check (P3)

21. Add unit tests for `fileArgs` validator (non-existent file, directory, non-md, valid md, zero args)
22. Add test for the auto-discover path when no args are given
23. Consider adding `--strict` flag that treats warnings as errors
24. Consider adding `--exclude` flag to skip specific files

### Broader system/ features (P3)

25. Add `System.LagDuration()` delegating to `projectionhost.Host.LagDuration()`
26. Add `System.LagPerProjection()` delegating to `projectionhost.Host.LagPerProjection()`
27. Add `System.Status()` delegating to `projectionhost.Host.Status()`
28. Wire `projectionhost.WithDeadLetterStore` as a `DomainConfig` option
29. Wire `projectionhost.WithFlightRecorder` as a `DomainConfig` option
30. Add `System.DeadLetterCount()` for DLQ monitoring

### Code quality (P3)

31. Rename `ErrSystemStopped` to `ErrSystemNotRunning` for clarity
32. Consider `ErrNoProjectionHost` → `ErrProjectionHostNotConfigured` for consistency with `ErrSeekableJournalMissing`
33. Add `//nolint:lll` comments where needed for long error wrapping lines
34. Run `nix run .#lint` on the changed files to catch any lint issues
35. Run `nix run .#check-duplication` to verify no new duplication was introduced

### API surface (P3)

36. Consider adding `WithCheckpointStore` as a functional option on `New()` for ergonomic parity with `stack.WithCheckpointStore`
37. Consider adding `WithHealthCheck` option for custom health check functions
38. Consider adding `WithGracefulCloseTimeout` option for default timeout when `ctx` has no deadline

### Release (P3)

39. Tag `system/v4.1.0` with the new methods (minor version bump for new API)
40. Update `CHANGELOG.md` with the P2 hardening entries
41. Update `FEATURES.md` system/ section to mark health-check/graceful-close as DONE
42. Update `ROADMAP.md` to reflect the P2 hardening completion
43. Run `nix run .#verify` as the final gate before tagging

### Cross-module (P4)

44. Consider a shared `health.Checker` interface across stack/ and system/ for consumer-side abstraction
45. Consider a shared `graceful.Closer` interface across stack/ and system/
46. Add `system` to the `cmd/cqrs-lint` module catalog for adoption scoring
47. Add `system` to the `example/taskmanager` migration path documentation
48. Consider a `system/testutil` package with helpers for testing System-based apps
49. Add `system` to the `benchkit` benchmark suite
50. Consider a `system/helm` chart for Kubernetes deployment (liveness/readiness probes)

---

## G) QUESTIONS I CANNOT ANSWER MYSELF

### 1. Should `System.Close()` join all errors or return only the first?

The current implementation returns only the first error (inherited from the original code). `stack.Bundle.Close()` joins all errors. Changing this is technically a behavioral change — callers doing `if err := sys.Close(); err != nil { ... }` would see different error values. Should I align with `stack.Bundle` (join all) or preserve the current first-error behavior?

### 2. Should `DomainConfig.CheckpointStore` be a functional option instead of a struct field?

`stack.Bundle` uses `stack.WithCheckpointStore()` as a functional option. `DomainConfig` uses direct struct fields (no options). I chose the struct-field pattern for consistency with `DomainConfig`, but consumers coming from `stack` may expect `system.WithCheckpointStore()`. Should I add a functional option wrapper too?

### 3. Should the dead `s.closers` slice be populated or removed?

`System.closers` is declared as `[]func() error` but `New()` never appends to it. It's dead code. Should I wire it (e.g., add SQLite engine close functions, bus close functions) or remove it? Wiring it would make `Close()` properly close resources that currently rely on engine `Close()`. Removing it simplifies the struct. I can't tell if this was intentional (engines handle their own close) or a forgotten TODO.
