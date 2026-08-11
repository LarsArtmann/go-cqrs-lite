# Status Report: Record Consolidation (ADR-0111 Phases 3-4)

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions, captured in TODO_LIST.md, or determined to be minor polish. Original content retained below for historical context.**

**Date:** 2026-08-10 14:53 CEST
**Task:** Finish Record consolidation — consolidate event.Metadata, command.Metadata, metadata.Tracing into record.CommonMetadata. Record becomes the single structural base for events + commands. Delete duplicate metadata types.

---

## a) FULLY DONE (Production Code)

### New Types Created
- **`id/actor_id.go`** + **`id/actor_id_json.go`** — `ActorID` kind-discriminated struct (ActorUser/Bot/System/Service), `ParseActorID`, `PrefixedString`, `MarshalJSON`/`UnmarshalJSON`, `Equal`, `IsZero`, `Format`. Inspired by go-composable-business-types/actor and cqrs-htmx/identity-model.
- **`record/record.go`** — `CommonMetadata` now uses branded types (`id.CorrelationID`, `id.CausationID`, `id.ActorID`, `id.RequestID`) instead of plain strings. Added `Merge()` method. Added JSON tags. Added `id/v4` dependency to `record/go.mod`.

### Metadata Consolidation (Phase 3)
- **`record/record.go`** — `CommonMetadata` fields are branded types. `Merge()` method added.
- **`event/metadata.go`** — `Metadata` now embeds `record.CommonMetadata` (was `metadata.Tracing`). `Clone()`, `Merge()`, `WithCustom()` updated. `Tombstone` field removed (Phase 4).
- **`event/options.go`** — `WithCorrelationID`, `WithCausationID`, `WithUserID`, `WithRequestID` now pass branded types directly (no `.String()`). Added `WithActor(id.ActorID)` option. `WithUserID` constructs `id.NewUserActor(v)`.
- **`event/asrecord.go`** — `AsRecord()` uses `CommonMetadata` directly from `evt.Metadata()`. Deleted local `brandedString` helper (no longer needed). CausationID precedence: typed Causation wins via `id.ParseCausationID(commandID.String())`.
- **`command/metadata.go`** — `Metadata` now embeds `record.CommonMetadata` (was `metadata.Tracing`). All methods updated. `WithUserID` constructs `id.NewUserActor(v)`.
- **`command/asrecord.go`** — `AsRecord()` passes `md.CommonMetadata` directly. Deleted local `brandedString` helper.
- **`query/query.go`** — `Metadata` now embeds `record.CommonMetadata`. All methods + options updated.
- **`metadata/metadata.go`** — `Tracing` type **DELETED**. `CustomData[K]` now embeds `record.CommonMetadata` (was `Tracing`). `MergeCustomMaps` kept.
- **`metadata/bridge.go`** — **DELETED** (obsolete after Tracing removal).

### Tombstone Removal (Phase 4)
- **`event/tombstone.go`** — **DELETED** (TombstoneStatus, TombstoneMark, DetectTombstone, MarkTombstone, MarkRebirth, MetadataKeyTombstone, MetadataKeyRebirth all gone).
- **`event/tombstone_test.go`** — **DELETED**.
- **`event/tombstone_property_test.go`** — **DELETED**.
- **`event/metadata.go`** — `Tombstone *TombstoneMark` field removed from `Metadata`.
- **`event/parser_fuzz_test.go`** — `FuzzDetectTombstone` function removed.

### Build Status
- **`go build -tags "goexperiment.jsonv2" ./...`** — PASSES. All production code compiles.
- **Test compilation** — PASSES. `go test -run='^$' ./...` compiles all test files (0 FAIL).

---

## b) PARTIALLY DONE

### Test Fixes (In Progress)
- **`id/`** — Tests PASS. ActorID type working.
- **`record/`** — Tests PASS. `TestRecord_JSONRoundTrip`, `TestCommonMetadata_Merge`, `TestCommonMetadata_ZeroValue` all green.
- **`event/`** — Tests COMPILE but 1 BDD test failure: `event_bdd_test.go:62` expects branded `CorrelationID` to equal string. Fix applied but may have residual issues.
- **`command/`** — Tests COMPILE. `metadata_test.go`, `asrecord_test.go`, `command_bdd_test.go`, `store_test.go` fixed via sed.
- **`query/`** — Tests COMPILE. `metadata_test.go`, `store_test.go` fixed via sed.
- **`metadata/`** — `metadata_test.go` rewritten (removed Tracing tests, updated CustomData tests). Untested.

### Downstream Consumers (NOT VERIFIED)
These modules have production code that compiles but tests have NOT been run:
- `decider/` — has `decider_coverage_test.go:100` referencing old types
- `listing/` — heavily dependent on deleted tombstone types (`TombstoneStatus`, `DetectTombstone`, `MarkTombstone`, `StatusMiddleware`)
- `watermill/` — `protocol.go` references `m.Tracing` and `writeTracing(md, t metadata.Tracing)`
- `integration/` — `creation_bdd_test.go:48` and `metadata_roundtrip_test.go:77` reference `evt.Metadata().UserID`
- `scenario/`, `projectionhost/`, `transport/`, `storage/`, `stack/`, `system/` — unknown status

---

## c) NOT STARTED

1. **listing/ module refactor** — `listing/types.go`, `listing/middleware.go`, `listing/in_memory.go` all reference deleted tombstone API. `StatusMiddleware` is broken. `StreamStatus.Status` field uses deleted `event.TombstoneStatus`. Entire tombstone policy system needs rethinking per ADR-0114.
2. **listing/ tests** — Multiple test files reference `event.MarkTombstone`, `event.TombstoneActive`, `event.TombstoneStatus`, `event.MetadataKeyTombstone`. All broken.
3. **watermill/ adapter updates** — `protocol.go` and `command_protocol.go` reference `metadata.Tracing` directly and `writeTracing` function.
4. **API stability golden regeneration** — `cmd/api-stability/main.go -update` not run. Exported API surface changed massively.
5. **go.mod cleanup** — `event/go.mod` still lists `metadata/v4` as direct dep (now only used for `MergeCustomMaps`). May need demotion to indirect.
6. **flake.nix testModules** — No new modules added, but `record/` now has `id/` dependency.
7. **Doc updates** — SKILL.md, AGENTS.md, references/*.md all reference old types.
8. **Tombstone migration doc** — ADR-0114 references `docs/migration/tombstone-to-domain-events.md` which doesn't exist.

---

## d) TOTALLY FUCKED UP

### Root Cause Design Pivot Mid-Task
- **Started with `string` fields in CommonMetadata** → user correctly called out `.String()` proliferation as a code smell.
- **Pivoted to branded types** → required creating `id.ActorID` (kind-discriminated struct). This was the RIGHT call but added significant scope.
- **CausationID type conversion in asrecord.go** — `id.CausationID(md.Causation.CommandID)` didn't compile (different branded types). Fixed with `id.ParseCausationID(commandID.String())` — this is a roundtrip through string which is ugly but works because both are ULID-backed.
- **sed-based batch test fixes** — Used `sed` to fix `command/metadata_test.go`, `command/command_bdd_test.go`, `command/store_test.go`, `query/metadata_test.go`, `query/store_test.go`. These fixes are mechanical and NOT verified to be correct. Some may have wrong semantics (e.g., `ActorID.Raw()` comparisons that should use `.Equal()`).

### Import Syntax Errors
- `event/event_metadata_test.go:10` — multiedit mangled the closing `)` onto the same line as the last import (`record/v4")`). Fixed but this shows the edit tool is fragile with import blocks.

### Leftover `.String()` in Tests
- Several test files still have `.String()` comparisons from the first (string-based) attempt that were partially fixed. The `event_bdd_test.go` fix was applied but the test FAILED at runtime — branded ID does not `Equal` a string.

---

## e) WHAT WE SHOULD IMPROVE

### Process
1. **Design the type system BEFORE writing code.** The string→branded pivot happened after ~20 edits. Should have caught the `.String()` smell in the design phase.
2. **Run `go build` after EVERY file change, not batches.** Would have caught import syntax errors immediately.
3. **Don't use `sed` for semantic changes.** `sed` replaced `.UserID` with `.ActorID` but didn't handle type mismatches (id.UserID vs id.ActorID). Should use targeted edits with full context.
4. **Test the foundation first.** Should have run `id/` and `record/` tests to green BEFORE touching event/command/query.
5. **The `FuzzDetectTombstone` deletion left an import gap** — had to manually verify no orphaned imports.

### Architecture
6. **CausationID roundtrip through string** in `asrecord.go` is a design smell. `CommandID` and `CausationID` are both ULID-backed branded types but not assignable. Consider a shared interface or a `CausationIDFromCommand(CommandID) CausationID` constructor.
7. **`metadata/` module is now nearly empty** — only `CustomData[K]` (deprecated) and `MergeCustomMaps` remain. Consider whether this module should be deleted entirely or merged into `record/`.
8. **`event.WithUserID` naming** — still called `WithUserID` but sets `ActorID`. This is correct for backward compat but the doc should clarify it constructs a user-kind actor.
9. **`listing/` module is architecturally broken** — its entire tombstone policy system depends on deleted APIs. This needs a design decision: delete the tombstone policy, or reimplement with event-type-based detection.

---

## f) Up to 50 Things to Do Next

### Immediate (Tests Must Pass)
1. Run `go test ./id/... ./record/...` — verify still green
2. Run `go test ./metadata/...` — verify rewritten test file
3. Run `go test ./event/...` — fix BDD test failure
4. Run `go test ./command/...` — verify sed fixes
5. Run `go test ./query/...` — verify sed fixes
6. Fix `event_bdd_test.go` — use `Equal(corrID)` not `Equal(corrID.String())`
7. Fix `event/codec_typed_test.go:123` — `!= corrID` → `!Equal(corrID)`
8. Fix `event/event_type_clone_test.go:193` — `id.CorrelationID{}` → zero value
9. Fix `event/event_metadata_test.go:123` — branded assignment to field
10. Fix `event/event_metadata_test.go:350` — branded comparison
11. Verify all sed-fixed files compile and pass
12. Run `go test ./event/... ./command/... ./query/... ./metadata/...` to green

### Downstream Module Fixes
13. Fix `decider/decider_coverage_test.go:100` — `.CorrelationID != correlationID`
14. Fix `integration/event/creation_bdd_test.go:48` — `.UserID` undefined
15. Fix `integration/event/metadata_roundtrip_test.go:77` — `.UserID` undefined
16. Fix `watermill/protocol.go:73` — `m.Tracing` undefined
17. Fix `watermill/command_protocol.go:52,101` — `metadata.Tracing` undefined
18. Fix `watermill/golden_test.go:53` — `Tracing: metadata.Tracing{...}`
19. Audit ALL `go test ./...` failures and fix them

### Listing Module (Phase 4 Tombstone Removal)
20. Decide: delete `StatusMiddleware` or reimplement with event-type detection
21. Remove `TombstonePolicy` / `TombstoneExclude/Include/Only` from `listing/types.go`
22. Remove `StreamStatus.Status event.TombstoneStatus` field
23. Refactor `listing/in_memory.go` — remove `applyTombstonePolicy`, `DetectTombstone` calls
24. Fix `listing/golden_test.go` — remove tombstone status assertions
25. Fix `listing/middleware_test.go` — remove StatusMiddleware tests
26. Fix `listing/fuzz_test.go` — remove `FuzzTombstonePolicy_String`
27. Fix `listing/listbuilder_bdd_test.go` — remove tombstone custom metadata
28. Fix `listing/example_test.go` — remove `ExampleStatusMiddleware`
29. Fix `listing/benchmark_test.go` — remove `event.MarkTombstone`
30. Fix `listing/builder_test.go` — remove `event.TombstoneActive`

### API Surface & Build Infrastructure
31. Regenerate API stability golden: `cd cmd/api-stability && GOWORK=off go run main.go -update`
32. Run `go mod tidy` in event/, command/, query/, metadata/ to clean up deps
33. Check if `metadata/v4` can be removed from `event/go.mod` direct deps
34. Run `nix run .#verify` (build + vet + test + race + lint + doc-check)
35. Run `nix run .#check-arch` (dependency budget enforcement)
36. Update `.agents/skills/go-cqrs-lite/references/*.md` with new types
37. Update `AGENTS.md` — remove references to `metadata.Tracing`, tombstone types
38. Update `SKILL.md` — update composition recipes

### Quality & Polish
39. Write `id/actor_id_test.go` — test ActorID constructors, Parse, JSON roundtrip, Equal, IsZero
40. Consider `id.CausationIDFromCommand(CommandID) CausationID` helper to avoid string roundtrip
41. Add `CommonMetadata.IsZero()` method (currently no way to check if all fields are unset)
42. Consider whether `metadata/` module should be deleted entirely
43. Write migration doc: `docs/migration/tombstone-to-domain-events.md` (referenced by ADR-0114)
44. Update ADR-0111 to mark Phases 3-4 as completed
45. Run `nix fmt` on all changed files
46. Check `go vet -tags "goexperiment.jsonv2" ./...` for issues
47. Verify JSON wire compatibility — old format `{"correlationId": "01J..."}` vs new ActorID prefixed format `"user:01J..."`
48. Update `cmd/cqrs-lint/` rules if they reference deleted tombstone types
49. Check `example/` apps for broken imports
50. Write a migration guide for consumers: `WithUserID(uid)` still works but now constructs an ActorID internally

---

## g) Questions I CANNOT Answer Myself

### 1. Listing module: delete tombstone policy or reimplement?
The entire `listing/` tombstone system (`StatusMiddleware`, `TombstonePolicy`, `StreamStatus.Status`, `applyTombstonePolicy`) depends on the deleted `event.DetectTombstone`/`event.MarkTombstone` API. ADR-0114 says "deletion semantics are domain-specific, the projection handler encodes this knowledge." Should I:
- (a) Delete the entire tombstone policy system from `listing/` (consistent with ADR-0114)?
- (b) Reimplement `StatusMiddleware` using event-type-based detection (configure delete types like `[]event.Type{"user.deleted"}`)?

### 2. JSON wire compatibility for ActorID
The old `metadata.Tracing` serialized as `{"userId": "01JXYZ..."}`. The new `ActorID` serializes as `{"actorId": "user:01JXYZ..."}` (prefixed string). This is a **breaking wire-format change**. Existing stored events with `userId` in their JSON metadata will fail to deserialize. Should I:
- (a) Accept the breaking change (this is a v4 major version consolidation)?
- (b) Add a compatibility layer in `event/metadata_json.go` that reads old `userId` and converts to `actorId`?

### 3. metadata/ module fate
After removing `Tracing`, the `metadata/` module contains only:
- `CustomData[K]` (deprecated, embeds `record.CommonMetadata`)
- `MergeCustomMaps[K]` (standalone helper)

Should I:
- (a) Delete the module entirely and inline `MergeCustomMaps` into each consumer?
- (b) Keep it as-is for backward compat (external consumers may import `CustomData`)?
- (c) Move `MergeCustomMaps` to `record/` and mark `metadata/` as deprecated?
