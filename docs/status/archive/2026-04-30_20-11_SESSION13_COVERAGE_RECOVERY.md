# Session 13 Status Report — Coverage Recovery & Lint Cleanup

**Date:** 2026-04-30 20:11 CEST
**Branch:** master (not yet pushed)
**Commits since origin:** 0 (all changes uncommitted)
**Total Go files:** 121 (64 production, 57 test)
**Total lines:** ~19,472 (5,748 production, 13,724 test)
**Test-to-code ratio:** 2.4:1

---

## a) FULLY DONE ✅

### Coverage Recovery

| Package          | Before | After      | Delta  | How                                                                                        |
| ---------------- | ------ | ---------- | ------ | ------------------------------------------------------------------------------------------ |
| `core/aggregate` | 21.4%  | **95.7%**  | +74.3% | New `repository_test.go` with fake Store/Bus/SnapshotStore/Outbox (no memory dependency)   |
| `core/command`   | 95.0%  | **100.0%** | +5.0%  | `Use` middleware test, closed-dispatcher Register test                                     |
| `core/query`     | 91.0%  | **100.0%** | +9.0%  | `Use` middleware test, `DispatchTyped` success + type-mismatch, closed-dispatcher Register |
| `core/event`     | 98.2%  | **99.1%**  | +0.9%  | `WithCustom` existing-custom-map test                                                      |

### Lint Cleanup

- Added `go-branded-id` to `.golangci.yml` depguard allow list (post-migration fix)
- Removed stale `internal/evtest/` and `internal/testhelpers/helpers.go` exclusions from `.golangci.yml`
- Fixed `wsl_v5` in `core/event/options.go` (missing blank line before `field()`)
- Fixed `wsl_v5` in `catalog/adapters/from_query_dispatcher.go` (2 violations)
- Fixed `nonamedreturns` for `middleware/recovery.go` query recovery (named return required for defer)
- Fixed `golines` formatting in `core/aggregate/repository.go` (long `opError` function signature)
- **Result: 0 lint issues across all 6 modules** (core, memory, catalog, middleware, integration, testhelpers)

### Catalog Schema Coverage

- Added tests for unsigned integer types (`uint8/16/32/64`)
- Added tests for complex types (`complex64/128`)
- Added tests for `any` interface type
- Added tests for fixed-size array type (`[3]int`)

### Documentation

- Updated AGENTS.md coverage table (removed stale "Session 8 Delta" column)
- Updated AGENTS.md known issues: marked aggregate/command/query/memory-snapshot/EventRetry as ✅ FIXED
- Updated AGENTS.md dependencies: `go-composable-business-types` → `go-branded-id`
- Added Session 13 entry to cleanup section

---

## b) PARTIALLY DONE 🔶

### Coverage Gaps Remaining

| Package                | Coverage | Gap                                                                                                                | Severity    |
| ---------------------- | -------- | ------------------------------------------------------------------------------------------------------------------ | ----------- |
| `core/event`           | 99.1%    | `WithCustom` `e.metadata == nil` branch unreachable via `NewEvent` (always initializes metadata)                   | 🟢 Cosmetic |
| `memory`               | 98.9%    | `Ack` loop branch (92.3%), `LoadAtVersion` (92.3%) — defensive path edge cases                                     | 🟢 Low      |
| `catalog`              | 94.4%    | `goTypeToJSON` (64.3%) — `chan/func/UnsafePointer/Invalid` unreachable via structs; `collectionSchema` else branch | 🟢 Low      |
| `catalog/adapters`     | 98.8%    | `addMessageToService` 87.5% — single branch                                                                        | 🟢 Low      |
| `catalog/eventcatalog` | 95.5%    | `writeSchema` nil path, `writeMessage` 91.3%                                                                       | 🟢 Low      |
| `core/aggregate`       | 95.7%    | `opError` helper (only exercised through error paths), `loadFromStore` error wrapping                              | 🟢 Low      |
| `core/pkg/id`          | 97.1%    | Binary encoding error paths                                                                                        | 🟢 Low      |

### Architecture Foundation

| Item                              | Status     | Notes                                            |
| --------------------------------- | ---------- | ------------------------------------------------ |
| `event.Codec` + `JSONCodec`       | ✅ Exists  | In `core/event/codec.go`, already tested         |
| `event.Outbox` interface          | ✅ Exists  | With `MemoryOutboxStore` implementation          |
| `event.SnapshotStore`             | ✅ Exists  | Integrated into `EventSourcedRepository`         |
| `dispatcher.Typed` interface      | ✅ Exists  | Enables generic middleware                       |
| `event.Builder`                   | ✅ Exists  | Fluent API in `core/event/builder.go`            |
| Functional options for Repository | ✅ Exists  | `WithSnapshotStore`, `WithOutbox`                |
| Cached middleware chain           | ✅ Exists  | O(1) dispatch since session 11                   |
| OTel tracing middleware           | ✅ Exists  | `CommandTracing`, `EventTracing`, `QueryTracing` |
| `Projection` interface            | ❌ Missing | No read-model support                            |
| `Upcaster` interface              | ❌ Missing | No event schema versioning                       |
| `CheckpointStore`                 | ❌ Missing | No projection checkpoint tracking                |

---

## c) NOT STARTED ⬜

### Phase 5: Storage Module (`storage/`)

SQL-backed event store using sqlc + pgx. This is the **#1 blocker for production use**.

- PostgreSQL schema (events + outbox tables)
- sqlc config + generated queries
- `event.Store` SQL adapter
- Transactional outbox implementation
- Integration tests with testcontainers

### Phase 6: Watermill Module (`watermill/`)

Pub/sub integration. Blocked on Phase 5.

### Phase 7: Projection Module (`projection/`)

Read-model projections. Blocked on Phase 5+6.

### Phase 8: SQL Snapshot Module (`sqlsnapshot/`)

SQL-backed snapshot store. Blocked on Phase 5.

### Phase 10: Tag Releases

Semantic versioning for all modules. `core/v1.0.0`, `memory/v1.0.0`, etc.

### Type Safety Improvements

- `command.Type` / `query.Type` / `event.Type` are all bare `string` — could use shared `Type[T]` to prevent cross-domain mixing
- `query.Handler` returns `(any, error)` — `DispatchTyped[T]` works but the `any` propagates through all middleware
- `event.Store` is a 5-method god interface — could split into `Writer/Reader/Deleter` (was considered and deferred in session 10)
- `catalog.Message` uses `Kind` as discriminated union — commands lack `Direction`, queries lack `Schema`

### Documentation & DX

- No getting-started guide or working example app
- No API stability guarantees (no semver tags)
- No generated API docs from Go doc comments
- No GitHub Pages with go-import meta tags

---

## d) TOTALLY FUCKED UP 💥

### Nothing is fucked up.

The codebase is in its cleanest state ever:

- **0 lint issues** across all 6 modules
- **0 race conditions** (all tests pass with `-race`)
- **0 test failures** (18/18 packages green)
- **4 packages at 100%** (command, query, dispatcher, middleware)
- **Formatting clean** (`nix flake check` passes)
- **Clean git history** (working tree has 11 modified files, all coherent)

### Self-Criticism for This Session

| #   | Issue                                                                                 | Reflection                                                                                                                                                                                                        |
| --- | ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | Wrote `repository_test.go` with 615 lines — exceeds 250-line guideline                | Should have split into `repository_test.go` (fakes) + `repository_save_test.go` + `repository_load_test.go`. But it's a test file with coherent fake implementations + tests; splitting would reduce readability. |
| 2   | Used `fmt.Errorf` in fake store, then had to replace with `errors.New` for perfsprint | Should have used `errors.New` from the start for static error messages in test code                                                                                                                               |
| 3   | Didn't check if `core/internal/` was already gone                                     | Wasted a task slot on "remove empty core/internal/" that was already done in a previous session                                                                                                                   |
| 4   | Didn't check if MemorySnapshotStore deep copy was already fixed                       | Wasted another task slot — it was already fixed in session 10                                                                                                                                                     |
| 5   | Missed the `go-branded-id` depguard issue initially                                   | Should have run lint FIRST before anything else to identify all issues at once                                                                                                                                    |
| 6   | Didn't consider moving fake Store/Bus/SnapshotStore/Outbox to `testhelpers/`          | The 250+ lines of fake implementations could be shared with integration tests. Currently duplicated (integration uses memory module, core uses inline fakes).                                                     |
| 7   | Did not run `nix fmt` before lint check                                               | Had golines issues that `nix fmt` would have fixed automatically                                                                                                                                                  |

### What I Forgot / Could Have Done Better

1. **Fake implementations should be in `testhelpers/`** — The 250+ lines of `fakeStore`, `fakeBus`, `fakeSnapshotStore`, `fakeOutbox` in `repository_test.go` are general-purpose test doubles that integration tests could also use. They should be extracted to `testhelpers/` as `FakeStore`, `FakeBus`, etc.

2. **`go.work.example` is gone but `go.work` tracking is still inconsistent** — `go.work` is tracked but some docs still reference workspace setup patterns from when it was gitignored.

3. **No benchmark for the new repository paths** — We have benchmarks for aggregate operations in `integration/` but none for the snapshot-aware `Load` or outbox-aware `Save` paths.

4. **Did not check for duplicate code between `repository_test.go` fakes and existing testhelpers** — `testhelpers` already has `NoopCommandHandler`, `FailingCommandHandler`, etc. but no `FakeStore`/`FakeBus`. We should add them there.

5. **Did not update `CHANGELOG.md`** — The coverage improvements and lint fixes should be recorded.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (High impact, low effort)

1. **Extract fake implementations to `testhelpers/`** — Move `fakeStore`, `fakeBus`, `fakeSnapshotStore`, `fakeOutbox` from `repository_test.go` to shared `testhelpers/`. This makes them reusable by integration tests and any future module tests. ~30 min.

2. **Add `event.Codec` usage to `EventSourcedRepository`** — We have `Codec` in `core/event/codec.go` but `EventSourcedRepository` doesn't use it for snapshot serialization/deserialization. `ApplySnapshot` receives raw `[]byte` and the user must manually decode. The repository should accept an optional `Codec` and handle serialization internally. ~1 hr.

3. **Add `Projection` interface to core** — The missing "Q" in CQRS. A simple interface:

   ```go
   type Projection interface {
       Handle(ctx context.Context, event event.Event) error
       Types() []event.Type
   }
   ```

   Plus a `Runner` that subscribes to event types and dispatches to projections. ~2 hr.

4. **Add `Upcaster` interface to core** — Event schema versioning:

   ```go
   type Upcaster interface {
       FromVersion() int
       Upcast(evt event.Event) (event.Event, error)
   }
   ```

   With a `UpcasterRegistry` that applies upcasters during `Load`. ~1 hr.

5. **Tag `v0.1.0-alpha` releases** — The API is stable enough for early adopters. Tags can be prefixed per module: `core/v0.1.0-alpha`, `memory/v0.1.0-alpha`, etc. ~15 min.

### Medium-Term (High impact, medium effort)

6. **Design and implement `storage/` module** — PostgreSQL event store via sqlc. This is the critical path for production use. The `event.Store` interface is already clean and well-tested. We just need a SQL adapter. ~2-3 days.

7. **Split `event.Store` god interface** — 5 methods is too many. Split into `Writer` (Save, AppendBatch), `Reader` (Load, LoadFromVersion), `Deleter` (Delete). Compose back as `Store` interface for backward compatibility. ~1 hr but breaking change.

8. **Use `samber/do` for DI in storage module** — Evaluate `samber/do` (dependency injection container) for wiring SQL connections, outbox publisher, and event bus in the storage module. Could replace manual constructor wiring. ~2 hr research first.

9. **Replace custom `TestMetrics` with OpenTelemetry SDK** — The middleware module has `TestMetrics` struct used in tests. Production metrics should use the real OTel SDK. Tests should use OTel's test utilities. ~2 hr.

10. **Add working example app** — A minimal `example/user/` that demonstrates: create aggregate → record event → save via repository → load → snapshot → query. ~3 hr.

### Architecture / Type Model Improvements

11. **Unified `MessageType` constraint** — `command.Type`, `query.Type`, and `event.Type` are all `type X string`. Create a shared constraint:

    ```go
    type MessageType interface { ~string }
    ```

    This enables generic functions that work across all CQRS kinds. ~30 min.

12. **Generic `aggregate.Root[T]`** — Currently `Core` has no type parameter. Adding `Core[T any]` where `T` is the aggregate's identity type would enable `Repository[T]` with type-safe `Load`/`Save`. Breaking change. ~1 day.

13. **Event payload type safety** — `Event.Payload()` returns `[]byte`. With the existing `Codec`, we could add:
    ```go
    func DecodePayload[T any](e Event, codec Codec) (T, error)
    ```
    This gives typed access to payloads without changing the `Event` interface. ~30 min.

### Established Libraries to Evaluate

14. **`samber/do`** — DI container. Useful for wiring storage module (DB pool + event store + outbox + bus). Lightweight, no code generation.

15. **`pgx/v5` + `sqlc`** — Already planned for storage module. `pgx` for PostgreSQL driver, `sqlc` for type-safe SQL queries. These are the standard Go choices.

16. **`ThreeDotsLabs/watermill`** — Already planned for pub/sub module. Well-established in the Go CQRS/ES community.

17. **`samber/lo`** — Previously evaluated and rejected as "overkill" (session 8). Re-evaluate for storage/projection modules where slice/map operations are common.

18. **`/stretchr/testify`** — Already in use for tests. Consider standardizing on `require`/`assert` in test files for consistency.

19. **`go-playground/validator`** — For struct-tag-based validation in the validation middleware. Currently `CommandValidator` is `func(command.Command) error` — could accept a validator instance.

---

## f) Top #25 Things to Get Done Next

Sorted by impact × effort⁻¹ (highest ROI first):

| #   | Task                                                 | Module         | Effort | Impact   | Notes                                                                                                                                 |
| --- | ---------------------------------------------------- | -------------- | ------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | **Extract fakes to `testhelpers/`**                  | testhelpers    | 30min  | HIGH     | Reusable `FakeStore`, `FakeBus`, `FakeSnapshotStore`, `FakeOutbox` — currently 250+ lines of inline test code in `repository_test.go` |
| 2   | **Add `DecodePayload[T]` helper**                    | core/event     | 15min  | MEDIUM   | Typed payload access using existing `Codec`: `DecodePayload[UserCreated](evt, codec)`                                                 |
| 3   | **Tag `v0.1.0-alpha` releases**                      | Git            | 15min  | MEDIUM   | `core/v0.1.0-alpha`, `memory/v0.1.0-alpha`, `catalog/v0.1.0-alpha`, `middleware/v0.1.0-alpha`                                         |
| 4   | **Add `Projection` interface**                       | core           | 1hr    | HIGH     | Missing "Q" in CQRS. `Projection` + `Runner` + simple in-memory projection runner                                                     |
| 5   | **Add `Upcaster` interface + registry**              | core           | 1hr    | HIGH     | Event schema versioning. `UpcasterRegistry` applies during `Load`.                                                                    |
| 6   | **Split `event.Store` into `Writer/Reader/Deleter`** | core/event     | 1hr    | HIGH     | Breaking change but cleaner. Compose back as `Store` for compat.                                                                      |
| 7   | **Wire `Codec` into snapshot serialization**         | core/aggregate | 30min  | MEDIUM   | `ApplySnapshot` receives `[]byte`; repository should handle decode via `Codec`                                                        |
| 8   | **Design storage module schema**                     | storage/       | 1hr    | CRITICAL | PostgreSQL events + outbox tables. Schema-first design before any code.                                                               |
| 9   | **Create `storage/` module skeleton**                | storage/       | 15min  | CRITICAL | `go.mod`, `go.work` entry, directory structure                                                                                        |
| 10  | **Add sqlc config + generated queries**              | storage/       | 1hr    | CRITICAL | `sqlc.yaml` + `Save`/`Load`/`LoadFromVersion` queries                                                                                 |
| 11  | **Implement `event.Store` SQL adapter**              | storage/       | 2hr    | CRITICAL | `Store` implementation using sqlc-generated queries                                                                                   |
| 12  | **Implement transactional outbox**                   | storage/       | 2hr    | HIGH     | Same-tx `Save` + `Outbox.Append`                                                                                                      |
| 13  | **Add working `example/user/`**                      | example/       | 3hr    | HIGH     | Full CQRS flow: command → aggregate → event → repository → snapshot                                                                   |
| 14  | **Write getting-started guide**                      | docs/          | 2hr    | HIGH     | README is technical; need step-by-step tutorial                                                                                       |
| 15  | **Evaluate `samber/do` for storage DI**              | storage/       | 2hr    | MEDIUM   | Research first, then decide. Don't adopt prematurely.                                                                                 |
| 16  | **Replace `TestMetrics` with OTel SDK**              | middleware     | 2hr    | MEDIUM   | Production-ready observability                                                                                                        |
| 17  | **Add `Unified MessageType` constraint**             | core           | 30min  | LOW      | `type MessageType interface { ~string }` for cross-kind generics                                                                      |
| 18  | **Add storage module design doc**                    | docs/planning/ | 1hr    | MEDIUM   | Before implementation: schema, sqlc, build tags, outbox pattern                                                                       |
| 19  | **Create `watermill/` module**                       | watermill/     | 3hr    | HIGH     | `event.Bus` adapter. Blocked on storage module for outbox publisher.                                                                  |
| 20  | **Create `projection/` module**                      | projection/    | 2d     | HIGH     | `Runner` + `CheckpointStore` + SQL implementation                                                                                     |
| 21  | **Add GitHub Pages go-import tags**                  | docs/          | 15min  | MEDIUM   | Required for `go get` subdirectory modules                                                                                            |
| 22  | **Add E2E throughput benchmarks**                    | integration/   | 30min  | LOW      | Commands/sec, events/sec with real middleware chain                                                                                   |
| 23  | **Add fuzz tests for event creation + ID parsing**   | core/          | 1hr    | LOW      | Already exists for `id.Parse`; extend to event and catalog                                                                            |
| 24  | **Add `Codec` integration to catalog system**        | catalog/       | 1hr    | LOW      | Schema generation for encoded payloads                                                                                                |
| 25  | **Add saga/process manager design doc**              | docs/planning/ | 1hr    | MEDIUM   | Research choreography vs orchestration                                                                                                |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "Should we extract the fake test doubles into `testhelpers/` now, or wait until we need them in the storage module?"

**The Problem:**

The `repository_test.go` I wrote this session has ~250 lines of fake implementations (`fakeStore`, `fakeBus`, `fakeSnapshotStore`, `fakeOutbox`). These are currently inline in a test file and can't be imported by other packages.

Meanwhile, the `integration/` module uses `memory.MemoryStore`, `memory.MemoryBus`, etc. for the same purpose — they're real implementations, not test doubles, but they serve the same role.

When we build the `storage/` module, we'll need test doubles for `event.Store` that can simulate SQL errors (connection lost, constraint violations, etc.). The inline fakes I wrote can be extended for this.

**Options:**

1. **Extract to `testhelpers/` now** — Pro: shared immediately, reusable. Con: adds public API surface to testhelpers (users might depend on fakes).

2. **Keep inline and extract when needed** — Pro: no premature abstraction. Con: duplicated code between `repository_test.go` and future storage tests.

3. **Create `core/testing` package** — Pro: part of core module, no circular deps. Con: increases public API surface.

4. **Use `memory` module for everything** — Pro: already exists, real implementations. Con: can't simulate error conditions (SQL failures, constraint violations).

My inclination is **Option 1** — extract to `testhelpers/` now as unexported types (lowercase `fakeStore`), which keeps them usable within the monorepo but not as public API. The storage module will need `failingStore` variants that are easy to add.

**But I need your input:** Should testhelpers export these as public types (so external consumers can use them in their tests), or keep them unexported (internal monorepo use only)?

---

## Coverage Summary (Current)

| Package                | Coverage   | Status       |
| ---------------------- | ---------- | ------------ |
| `core/command`         | **100.0%** | ✅ Perfect   |
| `core/query`           | **100.0%** | ✅ Perfect   |
| `core/pkg/dispatcher`  | **100.0%** | ✅ Perfect   |
| `middleware`           | **100.0%** | ✅ Perfect   |
| `memory`               | **98.9%**  | ✅ Excellent |
| `catalog/adapters`     | **98.8%**  | ✅ Excellent |
| `core/event`           | **99.1%**  | ✅ Excellent |
| `catalog/asyncapi`     | **97.6%**  | ✅ Excellent |
| `core/pkg/id`          | **97.1%**  | ✅ Excellent |
| `catalog/eventcatalog` | **95.5%**  | ✅ Very good |
| `core/aggregate`       | **95.7%**  | ✅ Very good |
| `catalog`              | **94.4%**  | ✅ Very good |

**Weighted average: ~97.7%**

## Build & Quality Verification

```
go test ./... -count=1 -race   → ALL PASS (18/18 packages)
nix run .#lint                 → 0 issues (all 6 modules)
nix flake check                → PASS (formatting)
```

## Files Changed This Session

| File                                        | Status   | Purpose                                                     |
| ------------------------------------------- | -------- | ----------------------------------------------------------- |
| `.golangci.yml`                             | Modified | Added `go-branded-id` to depguard, removed stale exclusions |
| `AGENTS.md`                                 | Modified | Coverage table, known issues, session 13 entry              |
| `catalog/adapters/from_query_dispatcher.go` | Modified | wsl_v5 blank lines                                          |
| `catalog/schema_test.go`                    | Modified | Unsigned/complex/interface/array schema tests               |
| `core/aggregate/repository.go`              | Modified | golines formatting (long function sig)                      |
| `core/aggregate/repository_test.go`         | **NEW**  | 13 repository unit tests with fake implementations          |
| `core/command/dispatcher_test.go`           | Modified | Use + closed-dispatcher Register tests                      |
| `core/event/event_test.go`                  | Modified | WithCustom existing-custom-map test                         |
| `core/event/options.go`                     | Modified | wsl_v5 blank line                                           |
| `core/query/dispatcher_test.go`             | Modified | Use + DispatchTyped success/mismatch + closed Register      |
| `middleware/recovery.go`                    | Modified | nonamedreturns nolint for query recovery                    |

---

_Generated at 2026-04-30 20:11 CEST by Crush_
