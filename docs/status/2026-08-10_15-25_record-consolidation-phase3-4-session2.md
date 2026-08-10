# Status Report: Record Consolidation (ADR-0111 Phases 3-4) — Session 2

**Date:** 2026-08-10 15:25 CEST
**Task:** Finish Record consolidation — consolidate metadata types, remove tombstone (Phase 4 per ADR-0114), fix all downstream consumers.

---

## a) FULLY DONE

### Core Metadata Consolidation (ADR-0111 Phase 3)
- **`record/record.go`** — `CommonMetadata` uses branded types (`id.CorrelationID`, `id.CausationID`, `id.ActorID`, `id.RequestID`). `Merge()` method. JSON tags. `id/v4` dependency.
- **`event/metadata.go`** — Embeds `record.CommonMetadata`. `Clone()`, `Merge()`, `WithCustom()` updated. `Tombstone` field removed.
- **`event/options.go`** — Branded type options. `WithActor(id.ActorID)` added. `WithUserID` constructs `id.NewUserActor(v)`.
- **`event/asrecord.go`** — Uses `CommonMetadata` directly. `brandedString` helper deleted.
- **`command/metadata.go`** — Embeds `record.CommonMetadata`. All methods updated.
- **`command/asrecord.go`** — Passes `md.CommonMetadata` directly. `brandedString` helper deleted.
- **`query/query.go`** — Embeds `record.CommonMetadata`. All methods + options updated.
- **`metadata/metadata.go`** — `Tracing` type DELETED. `CustomData[K]` embeds `record.CommonMetadata`.
- **`metadata/bridge.go`** — DELETED (obsolete after Tracing removal).
- **`id/actor_id.go`** + **`id/actor_id_json.go`** — `ActorID` kind-discriminated struct, constructors, `ParseActorID`, JSON marshaling.
- **`id/actor_id_test.go`** — Comprehensive tests: constructors, parsing, JSON roundtrip, equality, formatting, edge cases (9 test functions, ~40 sub-cases).

### Tombstone Removal in Core (ADR-0114 Phase 4)
- **`event/tombstone.go`** — DELETED (all tombstone types/functions gone).
- **`event/tombstone_test.go`**, **`event/tombstone_property_test.go`** — DELETED.
- **`event/parser_fuzz_test.go`** — `FuzzDetectTombstone` removed.

### Listing Module Refactored (ADR-0114)
- **`listing/types.go`** — Local `Status` type (Active/Deleted) replacing `event.TombstoneStatus`. `IsDeleted()` method. `StreamStatus.Status` now `listing.Status`.
- **`listing/middleware.go`** — `StatusMiddleware` DELETED. `CacheInvalidationMiddleware` kept.
- **`listing/in_memory.go`** — `WithDeleteTypes(event.Type...)` option. `detectStatus()` checks last event's type against configured delete types. `applyTombstonePolicy` uses `IsDeleted()`.
- **`listing/aggregate_reader.go`** — Updated doc comments.
- **`listing/doc.go`** — Rewritten for event-type-based detection.
- **All listing test files** — Updated for new API. Golden files regenerated.

### Watermill Adapter Fixed
- **`watermill/command_protocol.go`** — `writeTracing` → `writeCommonMetadata(record.CommonMetadata)`. Metadata import → record import.
- **`watermill/protocol.go`** — `writeCommonMetadata` call. Tombstone serialization/deserialization removed. `UserID` → `ActorID` wire mapping via `id.NewUserActor()`.
- **All watermill test files** — Updated.

### Integration Tests Fixed
- **`integration/event/creation_bdd_test.go`** — `.UserID` → `.ActorID` with `id.NewUserActor()`.
- **`integration/event/metadata_roundtrip_test.go`** — Same fix.

### Infrastructure
- API stability golden files regenerated (`docs/api_surface.txt` — 3874 exports).
- API stability meta-tests pass.
- `go mod tidy` clean in: event/, command/, query/, metadata/, watermill/, listing/, record/.
- All auto-committed by daemon as commit `8b8303299`.

---

## b) PARTIALLY DONE

### Verification — FALSE GREEN
- The session claimed "all tests pass" based on `go test -tags "goexperiment.jsonv2" ./...` from workspace root.
- **THIS WAS WRONG.** The workspace root has no `go.mod`, so `./...` matches ZERO packages. The command tested nothing.
- The modules that DO pass when tested individually: id/, record/, event/, command/, query/, metadata/, listing/, watermill/, decider/, scenario/, integration/event/.

---

## c) NOT STARTED

### Broken Modules (DO NOT COMPILE) — 16 modules fail `go build`
These modules reference deleted tombstone types (`event.TombstoneStatus`, `event.MarkTombstone`, `md.Tombstone`, `event.MetadataKeyTombstone`, etc.):

1. **`storage/`** — `aggregate_projection.go` (8 references), `sql_aggregate_reader.go` (1 reference). The `detectStatusFromMetadata()` function and `SQLAggregateReader` both depend on tombstone types.
2. **`stack/`** — `materialize.go` (5 references). `handleEvent()` checks `md.Tombstone` and branches on `TombstoneTombstoned`/`TombstoneActive`/`TombstoneUndetermined`.
3. **`transport/grpc/`** — `event_server.go` (2 references). Checks `md.Tombstone` in SSE stream.
4. **Cascading failures** (depend on stack/ or storage/):
   - `benchkit/`, `cmd/cqrs-bench/`, `example/getting-started/`
   - ALL `stack/*` presets: bbolt, duckdb, memory, mysql, pebble, postgres, sqlite, turso
   - `storage/turso/`

### Broken Test Files (in modules that compile)
5. **`storage/sql_aggregate_reader_test.go`** — `event.MarkTombstone`, `event.TombstoneTombstoned` (3 references)
6. **`stack/sqlite/view_models_integration_test.go`** — `md.Tombstone`, `event.MarkTombstone` (5 references)
7. **`stack/turso/view_models_integration_test.go`** — `md.Tombstone` references
8. **`cmd/cqrs-lint/`** — Compiles but rules reference deleted APIs in suggestions (`f001.go`, `a009_a013.go`, `catalog_extra.go`). The lint rules will produce incorrect guidance.

### Design Decisions Needed
9. **`stack/materialize.go` tombstone handling** — The `OnTombstone`/`OnRebirth` callback system depends on `md.Tombstone`. This is the core projection materialization path. Needs redesign: either event-type-based callbacks, or remove tombstone special-casing entirely (let `OnUpdate` handle delete events).
10. **`storage/aggregate_projection.go`** — `detectStatusFromMetadata()` function needs redesign to event-type-based detection (like listing/) or removal.

### Documentation
11. `SKILL.md` — references old types.
12. `.agents/skills/go-cqrs-lite/references/*.md` — references old types.
13. `AGENTS.md` — references `metadata.Tracing`, tombstone types.
14. `docs/migration/tombstone-to-domain-events.md` — referenced by ADR-0114, doesn't exist.
15. ADR-0111 not marked as completed.

---

## d) TOTALLY FUCKED UP

### False Verification (CRITICAL PROCESS FAILURE)
- **`go test ./...` from workspace root tests NOTHING.** There is no `go.mod` at root. The `./...` pattern matches zero packages.
- I ran this command MULTIPLE TIMES, saw "no output", and declared "ALL TESTS PASS" each time.
- The correct way to verify is: (a) test each module explicitly: `go test ./event/... ./command/...`, (b) use `nix run .#test` which iterates `testModules` from flake.nix, or (c) `go work` commands.
- **Impact:** I proceeded to "finish" the task and write a completion summary while 16 modules were broken. The auto-commit daemon committed this broken state as `8b8303299`.
- **Root cause:** I didn't verify that `./...` actually matched packages. I should have checked `go list ./... | wc -l` after the first run.

### Orphaned Constants in watermill/protocol.go
- `metaTombstoneStatus` and `metaTombstoneReason` constants are still defined (lines 34-35) but no longer used. Dead code.

### Stale Comment in listing/fuzz_test.go
- Line 103 still references "TombstoneStatus" in a comment. Harmless but sloppy.

### cqrs-lint Rules Give Incorrect Guidance
- `f001.go` tells users to "Use event.MarkTombstone" — a function that no longer exists.
- `a009_a013.go` suggests "Use event.DetectTombstone(events)" — also deleted.
- `catalog_extra.go` describes tombstone soft-delete as best practice.
- These compile fine (string literals) but will mislead users.

---

## e) WHAT WE SHOULD IMPROVE

### Process
1. **NEVER trust `./...` in a workspace without a root go.mod.** Always verify with `go list ./... | head` that packages are actually matched. Better: use `nix run .#test` which properly iterates `testModules`.
2. **Run `go build` in EACH module, not just from root.** The workspace root build is a no-op when there's no root go.mod.
3. **Check for ALL consumers before declaring done.** A simple `grep -rn "DeletedSymbol" --include="*.go"` would have found the 16 broken modules immediately.
4. **The AGENTS.md explicitly warns about this:** "Always `go build` immediately after deleting a package, before editing dependents." I deleted `event/tombstone.go` but never ran `go build` in storage/, stack/, transport/grpc/.

### Architecture
5. **The tombstone removal has a much larger blast radius than listing/ and watermill/.** The `stack/materialize.go` `OnTombstone`/`OnRebirth` callback system is the primary projection materialization path — it's used by ALL stack presets. This needs a proper redesign, not just mechanical fixes.
6. **`storage/aggregate_projection.go` has its own tombstone detection** (`detectStatusFromMetadata`) that duplicates the deleted `event.DetectTombstone` logic. This should follow the same event-type-based pattern as listing/.
7. **The cqrs-lint rules need updating** to stop recommending deleted APIs and start recommending event-type-based deletion.

---

## f) Up to 50 Things to Do Next

### Fix Broken Production Code (CRITICAL — 16 modules broken)
1. Fix `storage/aggregate_projection.go` — redesign `detectStatusFromMetadata()` to event-type-based detection or remove it
2. Fix `storage/sql_aggregate_reader.go:161` — `event.TombstoneStatus` reference
3. Fix `stack/materialize.go:146-182` — redesign `handleEvent()` tombstone branch to event-type-based callbacks or remove special-casing
4. Fix `transport/grpc/event_server.go:158-159` — remove `md.Tombstone` check
5. Verify ALL `stack/*` presets compile after stack/ fix
6. Verify `benchkit/`, `cmd/cqrs-bench/`, `example/getting-started/` compile
7. Run `go build` in EVERY workspace module to confirm zero failures

### Fix Broken Test Files
8. Fix `storage/sql_aggregate_reader_test.go` — `event.MarkTombstone`, `event.TombstoneTombstoned`
9. Fix `stack/sqlite/view_models_integration_test.go` — tombstone references
10. Fix `stack/turso/view_models_integration_test.go` — tombstone references
11. Fix `stack/materialize_test.go` — may need tombstone test cases removed/updated
12. Run `go test` in EVERY workspace module

### Fix cqrs-lint Rules
13. Update `cmd/cqrs-lint/pkg/rules/adoption/f001.go` — stop recommending `event.MarkTombstone`
14. Update `cmd/cqrs-lint/pkg/rules/api/a009_a013.go` — stop recommending `event.DetectTombstone`
15. Update `cmd/cqrs-lint/pkg/rules/catalog_extra.go` — remove tombstone best-practice suggestion
16. Update `cmd/cqrs-lint/pkg/rules/adoption/f001_f009_test.go` — remove `TestF001_NoFindingWithMarkTombstone`
17. Run cqrs-lint test suite

### Cleanup
18. Remove orphaned `metaTombstoneStatus`/`metaTombstoneReason` constants from `watermill/protocol.go`
19. Fix stale comment in `listing/fuzz_test.go:103`
20. Check `example/taskmanager/setup.go:113` — `[]any` vs `[]system.ProjectionDeclaration` (may be pre-existing)

### Design: stack/materialize.go Tombstone Handling
21. Decide: remove `OnTombstone`/`OnRebirth` callbacks entirely, or redesign for event types
22. If removing: document that delete events should be handled in `OnUpdate` (the projection handler decides what "deleted" means)
23. If redesigning: add `DeleteTypes []event.Type` config to `Materialize`, check `evt.Type()` in `handleEvent`
24. Update all stack preset consumers if the Materialize API changes
25. Update `stack/materialize_test.go` for new approach

### Design: storage/aggregate_projection.go
26. Decide: remove `detectStatusFromMetadata()` or redesign for event types
27. If redesigning: add configurable delete types (like listing/ `WithDeleteTypes`)
28. Update `SQLAggregateReader` to use new status type
29. Update `storage/sql_aggregate_reader_test.go`

### Verification
30. Run `nix run .#test` (proper full test suite via flake.nix)
31. Run `nix run .#build` (proper full build)
32. Run `nix run .#lint` (will catch remaining issues)
33. Run `nix run .#verify` (build + vet + test + race + lint + doc-check)
34. Run `nix run .#check-arch` (dependency budget enforcement)

### Documentation
35. Update `AGENTS.md` — remove tombstone references, update listing/ module description
36. Update `SKILL.md` — update composition recipes
37. Update `.agents/skills/go-cqrs-lite/references/*.md`
38. Write `docs/migration/tombstone-to-domain-events.md` (referenced by ADR-0114)
39. Update ADR-0111 to mark Phases 3-4 as completed
40. Update ADR-0114 with migration guidance

### Polish
41. Run `nix fmt` on all changed files
42. Run `go vet -tags "goexperiment.jsonv2"` in every module
43. Regenerate API stability goldens after all fixes
44. Update `cmd/api-stability` if any new modules were added
45. Check `flake.nix` `testModules` list is current
46. Consider `id.CausationIDFromCommand(CommandID) CausationID` helper to avoid string roundtrip in asrecord.go
47. Consider whether `metadata/` module should be deleted entirely (only CustomData + MergeCustomMaps remain)
48. Add `CommonMetadata.IsZero()` method
49. Update listing/README.md for new event-type-based detection
50. Write a migration guide for consumers

---

## g) Questions I CANNOT Answer Myself

### 1. stack/materialize.go: Remove OnTombstone/OnRebirth or redesign for event types?
The `Materialize` struct has `OnTombstone` and `OnRebirth` callbacks that fire when `md.Tombstone` is set. With tombstone metadata gone (ADR-0114), these callbacks can never fire. Options:
- **(a) Remove them entirely** — delete events go through `OnUpdate` like any other event. The projection handler decides what "deleted" means. Simplest, most consistent with ADR-0114 ("deletion semantics are domain-specific").
- **(b) Redesign with event types** — add `DeleteTypes []event.Type` to `Materialize`, check `evt.Type()` in `handleEvent`. More ergonomic but adds configuration burden.

### 2. storage/aggregate_projection.go: Same question for detectStatusFromMetadata()
The `SQLAggregateReader` returns `StreamStatus` with `event.TombstoneStatus`. This duplicates the listing/ pattern but for SQL-backed reads. Should it:
- **(a) Use the listing.Status type** (from listing/types.go)?
- **(b) Define its own local status type** (like listing/ does)?
- **(c) Remove status tracking entirely** from the SQL reader?

### 3. Should this broken state be committed (it already is) or should we revert?
The auto-commit daemon committed the broken state as `8b8303299`. 16 modules don't compile. Options:
- **(a) Keep the commit, fix forward** — the broken state is checkpointed, we fix the remaining modules in subsequent commits.
- **(b) Revert and redo properly** — go back to the pre-session state and redo the entire migration with correct verification. More work but cleaner history.
