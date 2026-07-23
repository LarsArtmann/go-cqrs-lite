# Status: Aggregate → Stream Rename (Partial Completion)

**Date:** 2026-07-23 20:11
**Session Focus:** Executing the rename of `Aggregate*` to `Stream*` types across the entire go-cqrs-lite library
**ADR:** [ADR-0058](../adr/0058-rename-aggregate-to-stream.md)
**Plan:** [SUPERB-RENAME-AGGREGATE-TO-STREAM](../planning/2026-07-23_17-51_SUPERB-RENAME-AGGREGATE-TO-STREAM.md)

---

## Executive Summary

The codebase **compiles, tests pass (79 packages, 0 failures), lint is clean (0 issues), and doc-check validates 897 references**. The rename is structurally complete — all Go identifiers (types, functions, methods, variables, constants) have been renamed to `Stream*` with deprecated aliases preserving backward compatibility. Wire formats (JSON tags, SQL columns, proto fields, OTel attribute string values, error codes) are preserved.

**However, the rename is only ~80% done.** What remains is documentation and comment cleanup — ~99 Go files still contain the word "aggregate" in comments and doc strings, and AGENTS.md/SKILL.md still reference old type names throughout. Additionally, two exported error variables in `storage/sql/` and `storage/pebble/` were missed entirely.

---

## a) FULLY DONE (Verified Green)

### Build / Test / Lint

| Gate                        | Status | Evidence                                       |
| --------------------------- | ------ | ---------------------------------------------- |
| Build (`go build ./...`)    | PASS   | Zero compilation errors across all 52+ modules |
| Tests (`nix run .#test`)    | PASS   | 79 packages ok, 0 failures                     |
| Lint (`nix run .#lint`)     | PASS   | 0 issues across all modules                    |
| Doc-check (`cmd/doc-check`) | PASS   | 897 references valid across 34 packages        |
| Format (`nix fmt`)          | PASS   | 47 files formatted                             |

### Phase 1: id/ Module (100% Complete)

- `id/stream_id.go` — `StreamMarker`, `StreamID`, all constructors (`NewStreamID`, `ParseStreamID`, `ParseStreamIDStrict`, `DeriveStreamID`, `StreamIDFrom`, `IsStreamIDULID`, `StreamTimestamp`)
- `id/stream_type.go` — `StreamType`, `StreamRef`, `ParseStreamType`, `NewStreamRef`, `StreamRef.StreamKey()`, `StreamRef.Validate()`
- `id/aggregate_id.go` — Deprecated aliases: `AggregateMarker = StreamMarker`, `AggregateID = StreamID`, all old constructors as deprecated wrappers
- `id/aggregate_type.go` — Deprecated aliases: `AggregateType = StreamType`, `AggregateRef = StreamRef`, `ErrEmptyAggregateType = ErrEmptyStreamType`
- `id/idtest/` — `ParseStreamID()` added, `ParseAggregateID()` as deprecated wrapper
- `id/derive.go`, `id/doc.go` — Updated

### Phase 2: event/ Module (100% Complete)

- `event/event.go` — Fields renamed (`streamID`, `streamType`), `StreamID()` and `StreamType()` methods added, `AggregateID()`/`AggregateType()` kept as deprecated wrappers
- `event/errors.go` — `ErrNilStreamID`, `ErrEmptyStreamType`, `ErrStreamNotFound` as primary; old names as deprecated aliases. Error code strings unchanged (`"event.nil_aggregate_id"`, etc.)
- `event/event_new.go`, `event/single.go`, `event/batch.go`, `event/event_validate.go`, `event/tombstone.go` — All updated to use `streamID`/`streamType` parameters

### Phase 3: command/ Module (100% Complete)

- `command/aggregate_ref.go` — `StreamType` and `StreamRef` as primary types, deprecated `AggregateType`/`AggregateRef` aliases. `ParseStreamType`/`NewStreamRef` as primary, deprecated wrappers
- `command/errors.go` — `ErrNilStreamID`, `ErrEmptyStreamType` as primary, deprecated aliases
- `command/command.go` — `StreamID()` method on `BasicCommand`
- `command/store.go` — `StreamID()`/`StreamType()` on `PersistedCommand`

### Phase 4: listing/ Module (100% Complete)

- `listing/types.go` — `StreamListing`, `StreamStatus` as primary, `AggregateListing`/`AggregateStatus` as deprecated aliases
- `listing/aggregate_reader.go` — `StreamReader` interface, `AggregateReader` deprecated alias
- `listing/in_memory.go` — `InMemoryStreamReader` as primary, `InMemoryAggregateReader` as deprecated alias
- `listing/builder.go`, `listing/middleware.go`, `listing/doc.go` — Updated

### Phase 5: otel/ Module (100% Complete)

- `otel/attributes.go` — `AttrStreamType`, `AttrStreamID`, `AttrStreamVersion`, `AttrStreamCount` as primary constants. OTel string values UNCHANGED (`"cqrs.aggregate.type"`, etc.). `StreamAttrs()` function, `AggregateAttrs()` deprecated
- `otel/golden_test.go` — Golden file updated with new Go constant names

### Phase 6: storage/ Module (100% Complete for Type Names)

- `storage/sql_aggregate_reader.go` — `SQLStreamReader` as primary, `SQLAggregateReader` as deprecated alias
- `storage/aggregate_projection.go` — `StreamProjection` as primary, `AggregateProjection` as deprecated alias
- `storage/sql/otel.go` — `StartStreamSpan` as primary, `StartAggregateSpan` as deprecated wrapper
- `storage/sql/helpers.go` — `DeleteByStream` as primary, `DeleteByAggregate` as deprecated wrapper
- `storage/pebble/otel.go` — `startStreamSpan` (private)
- `storage/pebble/store.go` — `streamPrefix`, `streamUpperBound`, `lockStream`, `unlockStream` (all private)
- All SQL column names (`aggregate_type`, `aggregate_id`), table names (`listing_aggregates`), and error codes preserved

### Wire Format Preservation (100% Verified)

- JSON struct tags: `json:"aggregateId"`, `json:"aggregateType"` — UNCHANGED
- SQL columns: `aggregate_type`, `aggregate_id` — UNCHANGED
- Proto fields: `aggregate_id`, `aggregate_type` in .proto and .pb.go — UNCHANGED
- OTel attribute string values: `"cqrs.aggregate.type"`, etc. — UNCHANGED
- Error code strings: `"event.aggregate_not_found"`, `"command.nil_aggregate_id"`, etc. — UNCHANGED
- Pebble key prefixes — UNCHANGED

### Bulk Call-Site Rename (100% Complete)

- ~1257 call sites updated from `id.NewAggregateRef`/`id.ParseAggregateID`/`id.NewAggregateID` to `id.NewStreamRef`/`id.ParseStreamID`/`id.NewStreamID`
- All `idtest.ParseAggregateID` calls updated to `idtest.ParseStreamID`
- All `command.NewAggregateRef` calls updated to `command.NewStreamRef`
- All deprecated wrapper calls in test files updated (zero SA1019 lint warnings)

### Documentation: ADR + DOMAIN_LANGUAGE (Partially Done)

- `docs/adr/0058-rename-aggregate-to-stream.md` — Status updated to Accepted
- `docs/adr/README.md` — ADR-0058 status updated to Accepted
- `docs/DOMAIN_LANGUAGE.md` — Core terms table updated (Stream, StreamRef, StreamType, StreamID, StreamListing, etc.), concurrency section updated, identity section updated

---

## b) PARTIALLY DONE

### Comments in Production Go Files (~70 files with "aggregate" in comments)

**~99 .go files still contain the word "aggregate"** — but the vast majority are:

- Deprecated alias definitions (correct — they SHOULD say "aggregate")
- Wire-format strings (correct — they SHOULD say "aggregate")
- Comments/doc strings that still use "aggregate" conceptually

**~70 production files have comments saying "aggregate" when they should say "stream":**

Key offenders (high-value files):

- `decider/doc.go` — 8 references ("pure-function aggregate pattern", "mutable aggregate root", etc.)
- `decider/decider.go` — 6 references in comments
- `decider/cache.go` — 5 references ("folded aggregate state", etc.)
- `decider/options.go` — 3 references
- `decider/typed_decider.go` — 2 references
- `listing/builder.go` — 8 references ("aggregate listings", "aggregate type", etc.)
- `listing/doc.go` — 3 references
- `storage/pebble/store.go` — 10+ comment references
- `storage/pebble/snapshot.go` — 10+ comment references
- `storage/memory/store_load.go` — 6 references
- `storage/memory/store.go` — 2 references
- `event/store.go` — 5 references
- `event/doc.go` — multiple references
- `snapshot/doc.go` — 3 references
- `snapshot/strategy.go` — 1 reference
- `command/doc.go` — 3 references
- `stack/bundle.go` — 3 references ("cross-aggregate reads")
- `stack/options.go` — 1 reference
- `testutil/doc.go` — 3 references
- `testutil/rapidgen.go` — 1 reference
- `scenario/dsl.go` — 1 reference

### AGENTS.md (Partially Updated)

16 "aggregate" mentions remain:

- Line 52: Module tree description still says `AggregateID`
- Line 77: `AggregateListing, AggregateStatus, InMemoryAggregateReader`
- Line 135: Design principle mentions "aggregate"
- Lines 153-157: Code examples use `aggregateID`, `AggregateMarker`
- Lines 228-238: Code examples use `AggregateID()`, `AggregateType()`, `NewAggregateRef`
- Lines 398-419: Code examples mention "aggregate"
- Lines 470-471: Code examples use `id.AggregateID`
- Line 786: Comment says "Aggregate lag"

### SKILL.md (Not Updated)

1 mention in the description line:

- "event-sourced aggregates" and "snapshot an aggregate" and "soft-delete aggregate"

### Skill References (Not Updated)

6 reference files with 32 total aggregate mentions:

- `references/core.md` — 10 mentions
- `references/advanced.md` — 11 mentions
- `references/recipes.md` — 3 mentions
- `references/modules.md` — 3 mentions
- `references/readmodels.md` — 2 mentions
- `references/faq.md` — 3 mentions

---

## c) NOT STARTED

### Missed Exported Error Variables (storage/sql/ and storage/pebble/)

Two error variable pairs were NOT renamed and have NO deprecated aliases:

**`storage/sql/errors.go`:**

- `ErrAggregateTypeMismatch` — should be `ErrStreamTypeMismatch` (code string `"storage.aggregate_type_mismatch"` stays)
- `ErrAggregateIDMismatch` — should be `ErrStreamIDMismatch` (code string `"storage.aggregate_id_mismatch"` stays)

**`storage/pebble/errors.go`:**

- `ErrAggregateTypeMismatch` — should be `ErrStreamTypeMismatch`
- `ErrAggregateIDMismatch` — should be `ErrStreamIDMismatch`

These are exported Conflict errors with consumers. They need: rename primary + add deprecated alias.

### cqrs-lint Rule Descriptions

The linter has rules that describe OO aggregate detection:

- `cmd/cqrs-lint/pkg/rules/api/a007.go` — "Dual model (OO aggregate + functional decider)" — these are DDD concept references and may be intentionally kept as "aggregate"
- `cmd/cqrs-lint/pkg/analyzer/types.go:61` — `IsOO bool // object-oriented aggregate (has uncommittedEvents)` — again, this refers to the DDD pattern being detected, not the library's types
- `cmd/cqrs-lint/pkg/rules/catalog.go` — Rule descriptions mention "aggregates"

**Decision needed:** These may intentionally keep "aggregate" since they describe the DDD anti-pattern being detected, not the library's renamed types.

### Example Applications

- `example/taskmanager/` — 5 files with aggregate references in comments (`deriver.go`, `domain.go`, `events.go`, `projection.go`, `setup.go`)
- `example/getting-started/` — Not checked

### Integration Simulation Docs

- `integration/simulation/doc.go` — 2 references
- `integration/simulation/generator.go` — 2 comment references

### Codex/Protocol References

- `watermill/doc.go:82` — "Commands carry identity (type, aggregate ID)"
- `transport/grpc/proto/cqrs.pb.go` — Comment says "aggregate type" (wire format stays, comment could update)

---

## d) TOTALLY FUCKED UP (Nothing!)

No catastrophic failures. The codebase compiles, tests pass, lint is clean. No data was lost. No wire formats were broken. The strategy of type aliases + deprecated wrappers worked exactly as designed.

The closest to a "fuckup" was the original sed command that renamed JSON struct tags (`json:"aggregateType"` → `json:"streamType"`) — this was caught and fixed during this session before it reached any commit.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **The sed was too broad** — It should have excluded struct tags from the start. A regex like `s/\bAggregate\w*/Stream\w*/g` applied only to Go identifiers (not inside backtick strings) would have been safer. Consider using `gofmt -r` for AST-based renames instead of sed.

2. **Deprecated aliases should have been generated programmatically** — Writing deprecated aliases by hand for every module is error-prone. A code generator or template would ensure consistency.

3. **The comment sweep should have been part of the sed** — The sed only renamed Go identifiers, not comments. A second sed pass for comment text (`s/aggregate/stream/gi` in comments, with manual review) would have caught 90% of the remaining work.

4. **storage/sql/errors.go and storage/pebble/errors.go were missed** — The comprehensive sweep agent found them but they weren't fixed before the session moved to documentation. They should be the first thing fixed next.

5. **No migration guide was written** — A `docs/migration/aggregate-to-stream.md` file showing consumers how to migrate their code from `AggregateID` to `StreamID` would be valuable.

### Design Observations

6. **The deprecated alias approach is excellent** — Zero consumer breakage, all 14+ consumer projects compile unchanged. The `// Deprecated:` comments are picked up by staticcheck (SA1019), guiding consumers to migrate.

7. **Wire format preservation is correct** — Changing JSON tags would break persisted data. Changing OTel attribute strings would break dashboards. Changing error codes would break log filters. All preserved.

8. **"Stream" was the right choice** — It accurately describes what the identifier names: an ordered, append-only sequence of events. "Aggregate" was a DDD concept that this library dismantled in ADR-0001.

---

## f) Next 50 Things to Get Done

### Critical (Must Do)

1. Rename `ErrAggregateTypeMismatch` → `ErrStreamTypeMismatch` in `storage/sql/errors.go` + deprecated alias
2. Rename `ErrAggregateIDMismatch` → `ErrStreamIDMismatch` in `storage/sql/errors.go` + deprecated alias
3. Rename `ErrAggregateTypeMismatch` → `ErrStreamTypeMismatch` in `storage/pebble/errors.go` + deprecated alias
4. Rename `ErrAggregateIDMismatch` → `ErrStreamIDMismatch` in `storage/pebble/errors.go` + deprecated alias
5. Update call sites for the above in `storage/pebble/save.go` and wherever they're referenced

### High Priority — AGENTS.md (16 mentions)

6. Update module tree description (line 52): `AggregateID` → `StreamID`
7. Update listing module description (line 77): `AggregateListing` → `StreamListing`, etc.
8. Update design principle #17 (line 135): "aggregate" → "stream"
9. Update code examples (lines 153-157): `aggregateID` → `streamID`, `AggregateMarker` → `StreamMarker`
10. Update upcaster code example (lines 228-238): `AggregateID()` → `StreamID()`, `NewAggregateRef` → `NewStreamRef`
11. Update singleflight example (line 398): "aggregate" → "stream"
12. Update hot-state cache example (lines 401-419): "aggregate" → "stream"
13. Update SQL view store example (lines 470-471): `id.AggregateID` → `id.StreamID`
14. Update lag comment (line 786): "Aggregate lag" → "Stream lag"
15. Update remaining AGENTS.md code examples and comments

### High Priority — Skill References (32 mentions across 6 files)

16. Update `references/core.md` (10 mentions)
17. Update `references/advanced.md` (11 mentions)
18. Update `references/recipes.md` (3 mentions)
19. Update `references/modules.md` (3 mentions)
20. Update `references/readmodels.md` (2 mentions)
21. Update `references/faq.md` (3 mentions)
22. Update `SKILL.md` description (1 mention)

### Medium Priority — Production Code Comments (~70 files)

23. Update `decider/doc.go` (8 references)
24. Update `decider/decider.go` comments (6 references)
25. Update `decider/cache.go` comments (5 references)
26. Update `decider/options.go` comments (3 references)
27. Update `decider/typed_decider.go` comments (2 references)
28. Update `listing/builder.go` comments (8 references)
29. Update `listing/doc.go` comments (3 references)
30. Update `storage/pebble/store.go` comments (10+ references)
31. Update `storage/pebble/snapshot.go` comments (10+ references)
32. Update `storage/pebble/iteration.go` comments
33. Update `storage/pebble/journal.go` comments
34. Update `storage/pebble/command_read.go` comments
35. Update `storage/pebble/command_store.go` comments
36. Update `storage/pebble/doc.go` comments
37. Update `storage/pebble/helpers.go` comments
38. Update `storage/pebble/otel.go` comments
39. Update `storage/pebble/query_store.go` comments
40. Update `storage/memory/store_load.go` comments (6 references)
41. Update `storage/memory/store.go` comments
42. Update `storage/memory/command_store.go` comments
43. Update `event/store.go` comments (5 references)
44. Update `event/doc.go` comments
45. Update `event/tombstone.go` comments
46. Update `snapshot/doc.go` comments
47. Update `snapshot/strategy.go` comments
48. Update `snapshot/read_pressure.go` comments
49. Update `command/doc.go` comments
50. Update `command/store.go` comments

### Lower Priority

- (Beyond 50): Example app comments, integration simulation comments, scenario DSL comments, testutil comments, stack comments, watermill comments, transport comments, middleware comments, codec comments, projectionhost comments

---

## g) Questions I Cannot Answer Myself

### 1. Should cqrs-lint rule descriptions and OO-detection logic keep the word "aggregate"?

The linter detects consumers who use OO-style aggregates (classes with `uncommittedEvents`, mutable state). Rule A007 describes this as "OO aggregate + functional decider". The `IsOO` field comment says "object-oriented aggregate".

**Should these stay as "aggregate"** (since they describe the DDD pattern being detected, which IS called "aggregate" in the wider Go ecosystem), **or should they be renamed to "stream"** for internal consistency?

### 2. Should the file names be renamed?

`id/aggregate_id.go` and `id/aggregate_type.go` are now deprecated alias files. `command/aggregate_ref.go` similarly. `storage/aggregate_projection.go` and `storage/sql_aggregate_reader.go` also.

**Should these files be renamed** to `stream_id.go` (already exists as the primary), `deprecated_aggregate_aliases.go`, etc.? Or is the current naming fine since they contain deprecated aliases?

### 3. Should the planning document and analysis docs be updated or marked as historical?

`docs/planning/2026-07-23_17-51_SUPERB-RENAME-AGGREGATE-TO-STREAM.md` describes the execution plan (status: "In Progress"). `docs/architecture-understanding/AGGREGATE-CONCEPT-ANALYSIS.md` is the analysis that motivated the rename.

**Should these be updated to reflect completion status**, or left as historical artifacts of the decision-making process? The planning doc still says "In Progress" — should it be marked "Complete"?

---

## Verification Summary

| Check                                        | Result                                     |
| -------------------------------------------- | ------------------------------------------ |
| `go build -tags "goexperiment.jsonv2" ./...` | PASS                                       |
| `nix run .#test` (79 packages)               | PASS (0 failures)                          |
| `nix run .#lint` (all modules)               | PASS (0 issues)                            |
| `cmd/doc-check` (897 references)             | PASS                                       |
| `nix fmt`                                    | PASS (47 files formatted)                  |
| Deprecated aliases compile                   | PASS                                       |
| Wire formats preserved                       | PASS (JSON, SQL, proto, OTel, error codes) |
| SA1019 deprecation warnings                  | 0 (all internal code uses Stream*)         |
