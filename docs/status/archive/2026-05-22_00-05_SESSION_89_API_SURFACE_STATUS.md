# Comprehensive Status Report — Session 89

**Date:** 2026-05-22 00:05
**Branch:** master
**Last Commit:** 9417085 fix(docserver): serve static assets with correct MIME types
**Previous Status:** 2026-05-21_20-31_SESSION_88_SHIP_READY_STATUS.md

---

## Executive Summary

Session 89 focused on **API surface hygiene for `core/event`**. The audit identified ~110 exported symbols, of which ~40 had zero external consumers. Commit `a6496e5` privatized all internal types, collapsing the surface to ~75 stable exports. Coverage rose from 89.3% to 92.1%. A minor regression was caught and reverted (accidental re-export of `OutboxPublisher` during this session). All 27 test packages pass, zero races, all 8 modules build independently.

---

## a) FULLY DONE ✅

### Session 89 Deliverables (commit a6496e5)

| #   | Task                                | Files                                       | Impact                                                                                                                                                                                                                                           |
| --- | ----------------------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Delete `CatalogMeta`**            | `catalog.go` (deleted)                      | Removed dead deprecated type                                                                                                                                                                                                                     |
| 2   | **Delete `Version.Decrement()`**    | `types.go`                                  | Zero callers, dead method                                                                                                                                                                                                                        |
| 3   | **Delete `NewTypedProjection`**     | `projection.go`                             | Zero callers, dead generic                                                                                                                                                                                                                       |
| 4   | **Unexport Builder subsystem**      | `builder.go`                                | `Builder` → `builder`, `NewBuilder` → `newBuilder`                                                                                                                                                                                               |
| 5   | **Unexport OutboxPublisher**        | `outbox_publisher.go`                       | `OutboxPublisher` → `outboxPublisher`, all options + constructors                                                                                                                                                                                |
| 6   | **Unexport Upcaster subsystem**     | `upcaster.go`, `upcaster_registry.go`       | `Upcaster` → `upcaster`, `UpcasterRegistry` → `upcasterRegistry`                                                                                                                                                                                 |
| 7   | **Unexport Enricher subsystem**     | `enricher.go`                               | `ContextEnricher` → `contextEnricher`, `CompositeEnricher`, `EnrichEvent`                                                                                                                                                                        |
| 8   | **Unexport batch codec**            | `codec_batch.go`                            | `NewEvents` → `newEvents`, `MustNewEvents`, `DecodePayloads`                                                                                                                                                                                     |
| 9   | **Unexport ProjectionFunc**         | `projection.go`                             | `ProjectionFunc` → `projectionFunc`, `NewProjection` returns `Projection` interface                                                                                                                                                              |
| 10  | **Unexport HandleParallel**         | `runner.go`                                 | `HandleParallel` → `handleParallel`                                                                                                                                                                                                              |
| 11  | **Unexport DefaultClock**           | `types.go`, `event.go`, `publish_helper.go` | `DefaultClock` → `defaultClock`                                                                                                                                                                                                                  |
| 12  | **Unexport ParseSchemaVersion**     | `types.go`, `event.go`                      | Only used internally                                                                                                                                                                                                                             |
| 13  | **Delete Wrap\* funcs (7)**         | `errors.go`                                 | `Wrap`, `WrapRejection`, `WrapConflict`, `WrapTransient`, `WrapCorruption`, `WrapInfrastructure`, `WrapFrom` — all unused after privatization                                                                                                    |
| 14  | **Unexport RegisterClassification** | `errors.go`                                 | Zero external callers                                                                                                                                                                                                                            |
| 15  | **Unexport 15 error sentinels**     | `errors.go`                                 | `ErrMismatchedSlices`, `ErrPayloadMarshal`, `ErrInvalidSnapshotInterval`, `ErrDuplicateProjection`, `ErrNilProjection`, `ErrNilCheckpointStore`, `ErrNilOutbox`, `ErrNilBus`, `ErrAlreadyStarted`, `ErrPublisherClosed`, `ErrProjectionPanicked` |
| 16  | **Convert test packages**           | Multiple `*_test.go`                        | `builder_test.go`, `enricher_test.go`, `upcaster_test.go` converted from `event_test` to `event` package                                                                                                                                         |
| 17  | **Fix clock_test references**       | `clock_test.go`                             | Replaced `event.NewBuilder` and `event.NewEvents` with `event.NewEvent`                                                                                                                                                                          |
| 18  | **Remove ExampleNewBuilder**        | `example_test.go`                           | Unexported builder has no public example                                                                                                                                                                                                         |
| 19  | **Update golden files**             | `catalog/testdata/golden/*`                 | AsyncAPI YAML, EventCatalog config, package.json                                                                                                                                                                                                 |

### Session 89 Additional Work (this session)

| #   | Task                            | Detail                                                                                                           |
| --- | ------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| 20  | **Revert accidental re-export** | Caught regression where `OutboxPublisher` was re-capitalized; reverted to committed state                        |
| 21  | **AGENTS.md updates**           | Coverage 89.1→92.1%, remove `DefaultClock` from key types, add session 89 milestone, fix CatalogMeta known issue |

### Pre-existing Session 89 Commits

| Commit    | Description                                                                |
| --------- | -------------------------------------------------------------------------- |
| `9417085` | fix(docserver): serve static assets with correct MIME types                |
| `a6496e5` | refactor(event): privatize all internal implementation types               |
| `94f910f` | docs(planning): add library quality and consumer friction elimination plan |
| `fbac220` | docs(planning): add library design audit and consumer pain analysis        |

---

## b) PARTIALLY DONE 🔶

| Task                         | Status            | Detail                                                                                         |
| ---------------------------- | ----------------- | ---------------------------------------------------------------------------------------------- |
| `testhelpers` coverage       | 10.5% — known low | These are test utilities, not production code. Low coverage is acceptable but could be higher. |
| `catalog/docserver` coverage | 90.0%             | Close to threshold, minor improvement possible                                                 |
| `catalog` core coverage      | 90.5%             | Could improve with more edge-case tests                                                        |

---

## c) NOT STARTED ⬜

| #   | Task                                                                                                 | Priority | Effort                                       |
| --- | ---------------------------------------------------------------------------------------------------- | -------- | -------------------------------------------- |
| 1   | Split `core/event` into sub-packages (runner → `projection/`, outbox publisher → `outbox/`)          | LOW      | Large refactor, current flat structure works |
| 2   | Converge `InMemoryRunner` (core/event) with `Runner` (projection/) — two parallel implementations    | MEDIUM   | Design decision needed                       |
| 3   | `testhelpers` coverage improvement (10.5% → 50%+)                                                    | LOW      | Test utility code                            |
| 4   | Fix `ParseUserAgent` inconsistency — returns bare value while all other `Parse*` return `(T, error)` | LOW      | Breaking change if signature changes         |
| 5   | Generate API surface docs from exports (godoc or custom)                                             | LOW      | Documentation                                |
| 6   | Remove remaining `CatalogMeta` from `command` and `query` packages                                   | LOW      | Two packages still have their own copies     |

---

## d) TOTALLY FUCKED UP 💥

| Issue                               | Severity  | Detail                                                                                                                                                                                                                                                                                                                            |
| ----------------------------------- | --------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Accidental re-export regression** | **FIXED** | During this session, I re-wrote `outbox_publisher.go`, `errors.go`, and `outbox_publisher_test.go` with capitalized exports, reverting the privatization from commit `a6496e5`. Caught and reverted. Root cause: didn't check `git diff` before writing files — assumed files needed modification when they were already correct. |

---

## e) WHAT WE SHOULD IMPROVE

### API Surface Quality

The `core/event` export audit revealed a clear pattern: **subsystems were fully exported "just in case" but had zero external consumers.** The remaining 75 exports are genuinely used across 2+ packages. The lesson:

> **Export nothing until a consumer needs it.** Go's package-private (lowercase) is the default for a reason.

### Remaining `core/event` Concerns

| Concern                                                 | Detail                                                                                      |
| ------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| **`InMemoryRunner` still exported**                     | Used by 1 external package (integration tests). Could move those tests to internal package. |
| **`SubscribesTo` still exported**                       | Used by `projection/runner.go`. Could duplicate the 7-line function.                        |
| **`InMemoryRunner` overlaps `projection.Runner`**       | Two separate projection runners with similar but different APIs                             |
| **`Clock` type exported but `DefaultClock` unexported** | Correct: consumers need the type for `WithClock` but don't need the default var             |

### Cross-project Quality

| Metric             | Value                                  | Assessment      |
| ------------------ | -------------------------------------- | --------------- |
| Files >250 lines   | 1 (`testhelpers/fake_store.go` at 263) | ⚠️ Should split |
| TODO/FIXME in prod | 0                                      | ✅              |
| Race detector      | Clean                                  | ✅              |
| GOWORK=off builds  | 8/8                                    | ✅              |
| Lint               | Clean                                  | ✅              |

---

## f) Top #25 Things to Do Next

### High Impact (Ship Value)

| #   | Task                                                                    | Impact                  | Effort |
| --- | ----------------------------------------------------------------------- | ----------------------- | ------ |
| 1   | **Version bump and tag release** — core/v1.5.1 with API surface cleanup | Consumers get clean API | 15 min |
| 2   | **README update** — document which types are public API vs internal     | Consumer clarity        | 30 min |
| 3   | **Split `testhelpers/fake_store.go`** (263→<250 lines)                  | Code quality gate       | 15 min |
| 4   | **Storage module improvements** — increase coverage from 88.1% to 90%+  | Reliability             | 2 hr   |
| 5   | **Converge InMemoryRunner + projection.Runner** — eliminate duplication | Architecture            | 4 hr   |

### Medium Impact (Library Quality)

| #   | Task                                                                                  | Impact               | Effort |
| --- | ------------------------------------------------------------------------------------- | -------------------- | ------ |
| 6   | **Move integration projection tests to internal** → unexport `InMemoryRunner`         | API surface          | 30 min |
| 7   | **Duplicate `SubscribesTo` into projection package** → unexport from event            | API surface          | 10 min |
| 8   | **Fix `ParseUserAgent` return type** — match other `Parse*` pattern                   | Consistency          | 15 min |
| 9   | **Remove `command.CatalogMeta` and `query.CatalogMeta`** — replace with direct struct | Dead code            | 1 hr   |
| 10  | **Add example/ for Outbox pattern** — show outbox publisher usage                     | Consumer DX          | 1 hr   |
| 11  | **Add integration test for outbox publisher with real bus**                           | Reliability          | 1 hr   |
| 12  | **Increase `catalog/docserver` coverage to 92%+**                                     | Quality              | 30 min |
| 13  | **Increase `catalog` core coverage to 92%+**                                          | Quality              | 1 hr   |
| 14  | **Add code examples to godoc** for all exported types                                 | Consumer DX          | 2 hr   |
| 15  | **Benchmark projection.Runner vs InMemoryRunner**                                     | Performance baseline | 1 hr   |

### Lower Impact (Polish)

| #   | Task                                                                                | Impact       | Effort |
| --- | ----------------------------------------------------------------------------------- | ------------ | ------ |
| 16  | **Generate API surface documentation** from exports                                 | Docs         | 2 hr   |
| 17  | **Add `example/` showing schema evolution with upcasters**                          | Consumer DX  | 1 hr   |
| 18  | **Increase `testhelpers` coverage to 30%+**                                         | Quality      | 1 hr   |
| 19  | **Add OpenAPI 3.1 exporter for queries** (complement AsyncAPI for events)           | Feature      | 4 hr   |
| 20  | **Add `event.Validate()` for consumer-side event validation**                       | Feature      | 2 hr   |
| 21  | **Add `sync/` module tests** — currently at 92.2%                                   | Quality      | 30 min |
| 22  | **Document versioning strategy** — semver guarantees, what's stable vs experimental | Docs         | 1 hr   |
| 23  | **Add CHANGELOG.md** — auto-generate from commits                                   | Docs         | 1 hr   |
| 24  | **CI pipeline for coverage regression** — fail if coverage drops >1%                | Quality      | 2 hr   |
| 25  | **Explore `internal/` sub-packages** for event implementation types                 | Architecture | 4 hr   |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `InMemoryRunner` and `OutboxPublisher` remain in `core/event` or move to separate sub-packages?**

The current flat structure puts concrete implementations (goroutines, mutexes, polling) next to pure interfaces (`Store`, `Bus`, `Projection`). The alternatives are:

1. **Keep flat** — `core/event` is the single import for all event types. Simple for consumers. The `internal/` directory pattern isn't possible across module boundaries in a multi-module workspace.
2. **Split into sub-packages** — `core/event/runner/`, `core/event/outbox/`, `core/event/upcaster/` — but this creates import fragmentation (`import "github.com/.../core/event/runner"`) and Go module complexity (sub-packages within a module share a `go.mod`).
3. **Move to separate modules** — `runner/` module, `outbox/` module — but these have been tried (the `projection/` module already exists as a separate runner) and created the duplication problem we'd be solving.

The tradeoff is between **consumer convenience** (one import) and **architectural purity** (interfaces separate from implementations). What's the right call?

---

## Project Vital Signs

| Metric              | Value                                  | Status     |
| ------------------- | -------------------------------------- | ---------- |
| Total LOC           | 47,612 (15,841 prod + 31,771 test)     | ✅ Healthy |
| Production files    | 179                                    | ✅         |
| Test files          | 131                                    | ✅         |
| Benchmark functions | 59 across 13 files                     | ✅         |
| Go modules          | 9                                      | ✅         |
| Test packages       | 27/27 pass                             | ✅         |
| Race detector       | Clean                                  | ✅         |
| GOWORK=off builds   | 8/8 pass                               | ✅         |
| Files >250 lines    | 1 (`testhelpers/fake_store.go` at 263) | ⚠️         |
| TODO/FIXME in prod  | 0                                      | ✅         |
| Total commits       | 946                                    | ✅         |

## Coverage Summary

| Package                | Coverage  | Change                  |
| ---------------------- | --------- | ----------------------- |
| `core/query`           | 100.0%    | —                       |
| `core/pkg/dispatcher`  | 100.0%    | —                       |
| `middleware`           | 100.0%    | —                       |
| `catalog/adapters`     | 100.0%    | —                       |
| `memory`               | 99.6%     | —                       |
| `core/pkg/id`          | 97.8%     | —                       |
| `core/aggregate`       | 95.9%     | —                       |
| `catalog/d2`           | 95.0%     | —                       |
| `core/command`         | 94.7%     | —                       |
| `catalog/openapi`      | 94.4%     | —                       |
| `catalog/asyncapi`     | 93.7%     | —                       |
| `projection`           | 93.9%     | —                       |
| `core/decider`         | 93.3%     | —                       |
| `core/event`           | **92.1%** | **↑ from 89.3%**        |
| `sync`                 | 92.2%     | —                       |
| `catalog/eventcatalog` | 91.3%     | —                       |
| `catalog`              | 90.5%     | —                       |
| `catalog/docserver`    | 90.0%     | —                       |
| `core/aggregate`       | 95.9%     | —                       |
| `storage`              | 88.1%     | —                       |
| `testhelpers`          | 10.5%     | ⚠️ Low (test utilities) |

## `core/event` API Surface: Before → After

| Metric                   | Before (Session 88) | After (Session 89) | Delta                   |
| ------------------------ | ------------------- | ------------------ | ----------------------- |
| Production files         | 23                  | 22                 | -1 (catalog.go deleted) |
| Exported symbols         | ~110                | ~75                | **-35**                 |
| Coverage                 | 89.3%               | 92.1%              | **+2.8%**               |
| Dead code (zero callers) | 4 symbols           | 0                  | All removed             |

### Symbols Removed from Public API

**Deleted entirely (dead code):**

- `CatalogMeta`, `Version.Decrement()`, `NewTypedProjection`

**Privatized (zero external consumers):**

- `Builder`, `NewBuilder`, `MustBuild`
- `OutboxPublisher`, `NewOutboxPublisher`, `OutboxPublisherOption`, `WithPollInterval`, `WithBatchSize`
- `Upcaster`, `UpcasterFunc`, `NewUpcaster`, `UpcasterRegistry`, `NewUpcasterRegistry`
- `ContextEnricher`, `CompositeEnricher`, `EnrichEvent`
- `NewEvents`, `MustNewEvents`, `DecodePayloads`
- `ProjectionFunc` (returned as `Projection` interface)
- `HandleParallel` → `handleParallel`
- `DefaultClock` → `defaultClock`
- `ParseSchemaVersion` → `parseSchemaVersion`
- `Wrap`, `WrapRejection`, `WrapConflict`, `WrapTransient`, `WrapCorruption`, `WrapInfrastructure`, `WrapFrom` (deleted — unused)
- `RegisterClassification`
- 15 error sentinels (`ErrMismatchedSlices`, `ErrPayloadMarshal`, `ErrInvalidSnapshotInterval`, `ErrDuplicateProjection`, `ErrNilProjection`, `ErrNilCheckpointStore`, `ErrNilOutbox`, `ErrNilBus`, `ErrAlreadyStarted`, `ErrPublisherClosed`, `ErrProjectionPanicked`)
