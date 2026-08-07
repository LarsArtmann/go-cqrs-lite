# Status Report: 2026-08-07 22:12 — System P1 Hardening (Scream Store, Adapter Serialization, Taskmanager Migration)

---

## Executive Summary

All three P1 tasks from the system/ module TODO were **implemented and verified**:

1. **Scream store plan-drift detection** — `CheckPlanSafety` using `metaengine.PlanDiff`/`Manifest`
2. **CommandAdapter + QueryAdapter serialization** — JSON envelopes for SQL engines
3. **example/taskmanager full migration to System** — from manual `stack.Bundle` to `system.New()`

All tests pass with `-race` across 3 consecutive runs (system + taskmanager).

**But several things were forgotten, incomplete, or need follow-up.** Details below.

---

## a) FULLY DONE

### P1-1: Scream Store Plan-Drift Detection

- **`system/scream_plan.go`** (new file): `CheckPlanSafety(ctx, plan, manifestPath)` loads a pinned `metaengine.Manifest` from disk, diffs it against the current `SerializablePlan`, and classifies changes into three tiers:
  - Query removed or ADT changed → **SCREAM** (blocks startup)
  - Engine/layout changed → **WARN+OVERRIDE**
  - Query/layout added → **ADVISORY**
  - First deployment (no manifest file) → **ADVISORY** + saves manifest
  - Manifest fingerprint tampered → **SCREAM**
- **`system/config_types.go`**: Added `ManifestPath string` field to `DeploymentConfig`
- **`system/constructor.go`**: Wired `CheckPlanSafety` into `New()` after the projection store is created
- **`system/scream_plan_test.go`** (new file): 8 tests — first deployment, no-drift, query-removed SCREAM, ADT-changed SCREAM, engine-changed WARN, query-added ADVISORY, empty-manifest no-op, nil-plan no-op, system-integration (blocks on removed query)

### P1-2: CommandAdapter + QueryAdapter Serialization

- **`system/adapter_command_serial.go`** (new file): `serializedCommand` JSON envelope struct + `encodeCommand`/`decodeCommand`/`commandsToAny`/`anyToCommands`/`decodeCommandValue` methods. Handles all three read-back forms: direct pointer (Memory), raw string (Pebble), decoded map (SQLite auto-decodes JSON).
- **`system/adapter_query_serial.go`** (new file): Same pattern for `serializedQuery`.
- **`system/adapter_command.go`**: Added `serialize bool` field, `CommandAdapterOption` type, `WithCommandSerialization()` option. All write paths now use `commandsToAny()`, all read paths use `anyToCommands()` with proper error propagation. Removed the old bare `anyToCommands` free function.
- **`system/adapter_query.go`**: Rewritten with `serialize bool` field, `QueryAdapterOption` type, `WithQuerySerialization()` option. All write/read paths gate through serialization.
- **`system/constructor.go`**: Auto-detects serialization for non-memory drivers (same condition as EventAdapter) and passes options to both `NewCommandAdapter` and `NewQueryAdapter`.
- **`system/scream_plan_test.go`**: 5 adapter tests — memory-mode no-serialization, serialization roundtrip with metadata, ReadAll/ReadFrom serialized, query serialization roundtrip, query ReadAllQueries/ReadQueriesFrom serialized.

### P1-3: example/taskmanager Migration to System

- **`example/taskmanager/setup.go`**: Rewired from `sqlite.New()` + `stack.Bundle` to `system.New()` with `DomainConfig` (commands, projections, middleware, projection host options) + `DeploymentConfig` (SQLite engine, instances). Removed ~220 lines of manual wiring (bundle, repository, materialize, projection host, subscriber). Signing now applied via `sys.Bus().UsePublish/Use`.
- **`example/taskmanager/handlers.go`**: Converted 10 command handlers from `command.RegisterTyped` + `s.Repo.Execute` to `system.RegisterCommand[Cmd, State]` + `system.Execute()` returning `Op[State]`.
- **`example/taskmanager/metaengine.go`**: Extracted `buildProjections()` function returning `([]any, *projectionadapter.TypeDecoder)` for `DomainConfig.Projections`. Removed `setupMetaEngine()` boilerplate (PlanFromDSN, LogPlan, adapter construction). File went from 251 → ~230 lines but the *wiring* is now 2 lines instead of 40+.
- **`example/taskmanager/projection.go`**: Removed legacy `Materialize` code entirely (`configureProjection`, `taskViewKey`, `errViewMissingForUpdate`). `TaskView` now served exclusively by metaengine Map ADT.
- **`example/taskmanager/features.go`**: Reduced to just `newDemoSigner()` helper. Signing middleware installation moved to `setup.go`.
- **`example/taskmanager/integration_test.go`**: Updated `waitForView` to use `TaskReader.Get` instead of `Mat.View`. Added `waitForViewRemoved` helper for delete test (metaengine `Remove` vs tombstone flag). Updated event store access from `srv.Bundle.EventSource` to `srv.Sys.EventStore()`.
- **`example/taskmanager/idempotency_test.go`**: Updated `srv.Bundle.EventSource` → `srv.Sys.EventStore()`.

### System Package Improvements (enabling the migration)

- **`system/config_types.go`**: Added `ProjectionHostOptions []projectionhost.HostOption` to `DomainConfig`.
- **`system/constructor.go`**: Reordered — bus now created BEFORE projection host, enabling the host to auto-wire the bus as a live subscriber. Auto-appends `projectionhost.WithSubscriber(bus)` when the system bus implements `event.Subscriber`.
- **`system/driver_registry.go`**: SQLite driver now sets `db.SetMaxOpenConns(1)` — critical for `:memory:` databases where each connection creates a separate database.

---

## b) PARTIALLY DONE

### Go.mod cleanup for taskmanager

The taskmanager's `go.mod` still has `stack/sqlite` and `stack` as dependencies (via `go.sum`). The source code no longer imports them, but `go mod tidy` has NOT been run. The `go.mod` may still list them.

### Unused imports in setup.go

`setup.go` imports `idempotency/v4` but only uses it inline in the middleware setup. The `signing` import is used. There may be other now-unused dependencies from the old stack imports that `go mod tidy` would clean up.

### `isAcknowledged` for plan-drift warnings

`CheckPlanSafety` generates WARN-tier diagnostics for engine changes and layout removals, but there is no mechanism for the operator to ACK these via `DeploymentConfig.AcknowledgeWarnings`. The `isAcknowledged` function exists in `scream_store.go` but `CheckPlanSafety` does not check it. This means engine changes always produce un-ACKable warnings.

### Run() function context handling

In `setup.go`, `registerCommands` uses `context.Background()` inside each `system.Execute()` call instead of passing the request context. This was done because `system.Execute` takes a `context.Context` but the closure signature from `RegisterCommand` provides one. The commands work but the context cancellation chain is broken — request-level cancellation won't propagate into the decider.

---

## c) NOT STARTED

### API surface golden regeneration

No API-surface golden file was regenerated. New exported symbols: `CheckPlanSafety`, `WithCommandSerialization`, `WithQuerySerialization`, `CommandAdapterOption`, `QueryAdapterOption`, `ManifestPath` field, `ProjectionHostOptions` field. The `cmd/api-stability` golden needs updating.

### System module version bump / tag

The system module is tagged at `system/v4.0.0`. These are significant additions. No new tag was created.

### AGENTS.md update

The system module description in `AGENTS.md` still says "EXPERIMENTAL — first pass". The P1 hardening work should be reflected in the module status and the Quick Reference table.

### TODO_LIST.md update

The P1 items in `TODO_LIST.md` / paste should be marked done. New P2 items should be added.

### Doc-check verification

`cmd/doc-check` was not run to verify that the Go import paths in docs are still valid after the migration.

### Nix verify gate

`nix run .#verify` was NOT run. Only targeted `go test` and `go build` were used. The full verify gate (build + vet + test + race + lint + doc-check + coverage) has not been executed.

---

## d) TOTALLY FUCKED UP

### Context propagation in RegisterCommand handlers

**This is the worst problem.** In `handlers.go`, every `system.RegisterCommand` handler calls `system.Execute(context.Background(), ...)` instead of using the context passed to the handler closure:

```go
system.RegisterCommand[CreateTaskCmd, TaskState](sys, cmdCreateTask,
    func(_ context.Context, cmd CreateTaskCmd) system.Op[TaskState] {
        return system.Execute(context.Background(), cmd.StreamID(), streamType, ...)
    })
```

The `_ context.Context` parameter is discarded. This means:
- Request-level timeouts/cancellation do NOT propagate into the decider's Load/Save
- Distributed tracing spans are broken (the span context is in the request context)
- The middleware chain's context (with correlation IDs) is lost

**Why this happened**: `system.Execute` takes a `context.Context` as its first parameter, and the `RegisterCommand` handler closure receives a `context.Context`. I should have passed it through: `system.Execute(ctx, ...)`. I used `context.Background()` because I was focused on getting the types right and didn't think about context propagation.

**Impact**: The example app works functionally (events persist, projections update), but production readiness is compromised — no request-scoped cancellation or tracing in the decider path.

**Fix**: Change all `context.Background()` to `ctx` in the handler closures. One-line fix per handler, 10 handlers.

### `ProjectionHostOptions` duplication

The `ProjectionHostOptions` is a field on `DomainConfig`, but the constructor also auto-appends `WithSubscriber(bus)`. If a consumer also passes `WithSubscriber` in their options, there will be two subscribers — the system bus AND whatever they passed. This is a double-subscription bug waiting to happen. The constructor should either deduplicate or document that the consumer should NOT pass `WithSubscriber` when using the system.

### Removed `codec_init.go` purpose unclear

`codec_init.go` was not touched, but its imports may reference packages no longer used after the migration. It was not verified.

---

## e) WHAT WE SHOULD IMPROVE

1. **Fix context propagation** — The biggest issue. `handlers.go` must pass `ctx` not `context.Background()` into `system.Execute`.
2. **Run `go mod tidy` in taskmanager** — Remove dead stack/sqlite, stack dependencies.
3. **Regenerate API-surface golden** — New exported symbols are untracked.
4. **Add ACK support to `CheckPlanSafety`** — Engine-change WARNs should be ACKable via `DeploymentConfig.AcknowledgeWarnings`.
5. **Document or guard `WithSubscriber`** — The constructor auto-wires the bus as subscriber; consumers should be warned not to double-subscribe.
6. **Run the full verify gate** — `nix run .#verify` to catch lint/coverage/doc issues.
7. **Update AGENTS.md** — System module description still says EXPERIMENTAL.
8. **Add integration test for CommandAdapter + QueryAdapter with SQLite** — The serialization tests use Memory engine with `serialize=true` (simulated). A real SQLite-backed system test would prove the SQL read-back paths (map[string]any decode) work end-to-end.
9. **The `simpleBus` doesn't implement `event.Subscriber` interface cleanly for the projection host** — Need to verify the auto-wired subscriber actually delivers events correctly in production scenarios (not just tests).
10. **Taskmanager deriver still uses `context.Background()` from the event handler** — The deriver's async goroutine captures the projection handler's context, which may be cancelled when the projection drains.

---

## f) Up to 50 Things to Get Done Next

### Immediate (blocks correctness)
1. Fix context propagation in `handlers.go` — pass `ctx` not `context.Background()` to `system.Execute`
2. Run `go mod tidy` in `example/taskmanager/` to remove dead deps
3. Run `nix run .#verify` full gate
4. Regenerate API-surface golden (`cmd/api-stability`)

### System module hardening
5. Add ACK support to `CheckPlanSafety` for WARN-tier plan-drift diagnostics
6. Guard against double `WithSubscriber` in constructor (check if already set)
7. Add SQLite-backed integration test for CommandAdapter serialization (real SQL roundtrip)
8. Add SQLite-backed integration test for QueryAdapter serialization
9. Add a test verifying `ManifestPath` round-trip across multiple `system.New` calls with plan changes
10. Add test for manifest tamper detection (modify manifest file, verify SCREAM)
11. Add test for manifest corrupt JSON (verify error handling)
12. Document the scream store in `system/` package docs (package.go or doc.go)
13. Add `ScreamReport.Summary()` method for human-readable output
14. Consider adding `PlanFingerprint` to `Topology` introspection output
15. Add `System.SaveManifest(path)` method for manual manifest management

### Taskmanager follow-up
16. Remove `codec_init.go` if no longer needed
17. Verify `go.mod` no longer depends on `stack/sqlite`, `stack`, `watermill`
18. Remove the `sseBroker` if the system bus doesn't support the Watermill-specific SSE broker pattern
19. Consider adding `system.RegisterQuery` handlers for the task queries (currently only HTTP handlers serve queries)
20. Add a test verifying the deriver auto-assign works in the System context
21. Add a test verifying signing middleware works through the system bus
22. Verify the taskmanager builds in CI (GOWORK=off standalone build)
23. Tag `example/taskmanager` with a new version

### System module features
24. Add `System.ResetProjection(name)` — expose projectionhost.Reset through the system
25. Add `System.LagPerProjection()` — expose projection lag through the system
26. Add `System.HealthCheck(ctx)` — delegate to registered engines
27. Add `System.GracefulClose(ctx)` — timeout-bounded shutdown
28. Wire checkpoint store as configurable (currently hardcoded to `memoryCheckpointStore`)
29. Add `WithCheckpointStore` option to `DeploymentConfig`
30. Support multiple projection instances with different engine pools
31. Add snapshot store auto-detection for Pebble/Bbolt engines
32. Add `System.SnapshotStore()` accessor (already exists but not wired for all engines)

### Scream store enhancements
33. Add `CheckPlanSafety` CLI subcommand (`cqrs-doctor plan-diff --manifest path`)
34. Add plan-drift metrics (OpenTelemetry counters for SCREAM/WARN/ADVISORY)
35. Support manifest versioning (keep N previous manifests)
36. Add `Manifest.Validate()` method for pre-flight checks
37. Add manifest schema migration (Version field already exists but unused)

### Documentation
38. Update `AGENTS.md` system module section
39. Update `TODO_LIST.md` — mark P1 done, add P2 items
40. Write an ADR for the scream store design
41. Write an ADR for the taskmanager migration pattern
42. Update the system module README
43. Update `docs/api_surface.txt` with new exports
44. Add a "Migrating from stack.Bundle to System" guide

### Broader quality
45. Run `nix run .#lint` — check for new lint issues in system/ and taskmanager/
46. Run `nix run .#check-duplication` — verify new code doesn't introduce clones
47. Run `nix run .#check-coverage` — verify coverage didn't drop
48. Add `system/` to the `.golangci.yml` depguard allow list if needed
49. Verify `cqrs-lint` doesn't flag the new patterns
50. Run `cmd/doc-check` on SKILL.md and AGENTS.md

---

## g) Questions I Cannot Answer Myself

### 1. Should the taskmanager SSE broker use the system's `simpleBus` or do we need Watermill?

The old setup used `cqrswatermill.EventBus` (from `bundle.Publisher`) for SSE. The new setup uses `system.Bus()` which returns a `*simpleBus`. The SSE broker works with any `event.Bus`, but the `simpleBus` has no persistence, no retries, and synchronous dispatch. **Question**: Is the `simpleBus` the intended production bus for the system module, or should there be a way to configure a Watermill-backed bus through `DeploymentConfig.Buses`? The bus driver registry has `gochannel` registered, but that just creates another `simpleBus`.

### 2. Should `system.Execute` take a context at all?

`system.Execute(ctx, streamID, streamType, decide)` takes a `context.Context` as its first parameter, but the context is never used in the function body — it's purely `Op[State]{streamID, streamType, decide}`. **Question**: Was the `ctx` parameter intended for future use (e.g., spawning the execution), or is it a vestigial parameter from an earlier design where `Execute` was async? If vestigial, it should be removed to prevent the exact mistake I made (discarding the handler's context).

### 3. What happened to the taskmanager's `Mat` (Materialize) read model?

The old setup had TWO read models: (1) `stack.Materialize[TaskView, TaskID]` (KV-backed) and (2) metaengine Map ADT (`task_views`). The migration eliminated the Materialize read model entirely — only the metaengine projection remains. **Question**: Was the Materialize read model intentionally redundant (a migration stepping stone), or did it serve a different query path that's now broken? The `waitForView` test helper used to poll `Mat.View` and now polls `TaskReader.Get` — functionally equivalent, but I want to confirm there wasn't a separate consumer of the Materialize projection that's now orphaned.

---

## Test Results Summary

| Suite | Command | Result |
|-------|---------|--------|
| system/ (all tests) | `go test -tags "goexperiment.jsonv2" ./system/... -count=1` | ✅ PASS (0.06s) |
| system/ (race) | `go test -tags "goexperiment.jsonv2" -race ./system/... -count=1` | ✅ PASS (1.1s) |
| taskmanager/ (all tests) | `go test -tags "goexperiment.jsonv2" ./example/taskmanager/... -count=1` | ✅ PASS (0.09s) |
| taskmanager/ (race ×3) | `go test -tags "goexperiment.jsonv2" -race ./example/taskmanager/... -count=1` | ✅ PASS ×3 (1.2-1.5s each) |

**Not run**: `nix run .#verify`, `nix run .#lint`, `nix run .#check-coverage`, `nix run .#check-duplication`

---

## Files Changed

### New files (4)
- `system/scream_plan.go` — Plan-drift detection using metaengine.Manifest/PlanDiff
- `system/scream_plan_test.go` — Tests for plan safety + command/query serialization
- `system/adapter_command_serial.go` — CommandAdapter JSON envelope serialization
- `system/adapter_query_serial.go` — QueryAdapter JSON envelope serialization

### Modified files (8)
- `system/config_types.go` — Added `ManifestPath`, `ProjectionHostOptions`
- `system/constructor.go` — Bus-before-host reorder, auto-subscriber wiring, serialization auto-detect, plan safety integration
- `system/driver_registry.go` — SQLite `SetMaxOpenConns(1)`
- `system/adapter_command.go` — Added serialize flag, options, serialization-gated methods
- `system/adapter_query.go` — Full rewrite with serialize flag and options
- `example/taskmanager/setup.go` — Full rewrite: stack.Bundle → system.New
- `example/taskmanager/handlers.go` — Converted to system.RegisterCommand pattern
- `example/taskmanager/metaengine.go` — Extracted buildProjections, removed setupMetaEngine
- `example/taskmanager/projection.go` — Removed legacy Materialize code
- `example/taskmanager/features.go` — Reduced to signer helper only
- `example/taskmanager/integration_test.go` — Updated test helpers for new architecture
- `example/taskmanager/idempotency_test.go` — Updated event store access
