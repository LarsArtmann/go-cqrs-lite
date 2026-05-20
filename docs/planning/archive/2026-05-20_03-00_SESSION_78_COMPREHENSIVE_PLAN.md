# Session 78 — Comprehensive Execution Plan

**Created:** 2026-05-20 03:00 · **Author:** Crush (with full codebase analysis)
**Scope:** go-cqrs-lite library + SEC consumer friction audit
**Total:** 48 tasks across 9 phases, each ≤12 min

---

## Executive Summary

The library is internally excellent (0 lint, 23/23 tests pass, 90%+ coverage, strong types, sentinel errors).
But SEC (the only real consumer, ~13K LOC) reveals **API design friction** that internal quality work cannot fix:

1. **`query.Handler` returns `any`** — SEC writes `assertCmd[T]` workarounds, type assertions everywhere
2. **No typed command dispatch** — `corecmd.Handler` takes `corecmd.Command`, forcing manual type assertion per handler
3. **`event.Event` payload is `[]byte`** — SEC maintains a 100-line encode/decode pipeline per aggregate
4. **`DomainEvent.Payload` is `any`** — 18 type assertions in SEC, silent failures on wrong type
5. **String-typed event dispatch** — dual switch statements (decode + fold) with zero compile-time checks
6. **GameID ↔ AggregateID conversion** scattered across 4 files

This plan addresses: (a) consumer-facing API improvements, (b) remaining correctness bugs, (c) quality gaps, (d) release readiness.

---

## Phase 1: Critical Bug Fixes (RED — Must Do First)

**Why:** Data loss / correctness bugs. No point improving API on top of broken foundations.

| #   | Task                                                                        | Effort | Impact | Files                                   |
| --- | --------------------------------------------------------------------------- | ------ | ------ | --------------------------------------- |
| 1   | Fix retry middleware timer leak: add `defer timer.Stop()` before select     | 5min   | HIGH   | `middleware/retry.go:104`               |
| 2   | Fix aggregate snapshot with nil codec: skip snapshot save when codec is nil | 8min   | HIGH   | `core/aggregate/load_helpers.go:93-122` |
| 3   | Fix Pebble Store: add optimistic concurrency check in Save                  | 10min  | HIGH   | `storage/pebble_event_store.go:48-102`  |
| 4   | Fix decider Execute dual %w wrapping                                        | 5min   | MEDIUM | `core/decider/decider.go:113`           |
| 5   | Add nil TimestampFunc guard in sync.NewLWWResolver                          | 5min   | MEDIUM | `sync/conflict.go:40`                   |
| 6   | Add nil type check in catalog.SchemaFromType for interface types            | 5min   | MEDIUM | `catalog/schema.go:25-29`               |

---

## Phase 2: Consumer API Improvements (HIGHEST CUSTOMER VALUE)

**Why:** SEC is writing 200+ lines of boilerplate that the library should eliminate. Every future consumer will hit the same walls.

| #   | Task                                                                                                           | Effort | Impact     | Files                                  |
| --- | -------------------------------------------------------------------------------------------------------------- | ------ | ---------- | -------------------------------------- |
| 7   | Add `command.TypedHandler[T]` + `command.RegisterTyped[T]` — type-safe handler that receives `T` not `Command` | 10min  | 🔥 HIGHEST | `core/command/typed.go` (new)          |
| 8   | Add tests for `command.TypedHandler[T]` — table-driven, type mismatch, dispatch                                | 10min  | HIGH       | `core/command/typed_test.go` (new)     |
| 9   | Add `query.TypedHandler[T any]` returns `(T, error)` — eliminate `any` return                                  | 10min  | 🔥 HIGHEST | `core/query/typed.go` (new)            |
| 10  | Add tests for `query.TypedHandler[T]` — result extraction, type mismatch                                       | 10min  | HIGH       | `core/query/typed_test.go` (new)       |
| 11  | Add `event.TypedProjection[T any]` — fold function receives decoded `T` not `[]byte`                           | 10min  | HIGH       | `core/event/typed_projection.go` (new) |
| 12  | Add `event.EncodeEvents` helper — `[]DomainEvent → []Event` with auto-marshal                                  | 10min  | HIGH       | `core/event/encode.go` (new)           |
| 13  | Add `event.DecodePayloads` helper — batch decode `[]Event → []T`                                               | 8min   | HIGH       | `core/event/decode.go` (extend)        |
| 14  | Update example/user to use TypedHandler + EncodeEvents                                                         | 10min  | HIGH       | `example/user/*.go`                    |

---

## Phase 3: Observability & Error Improvements (HIGH)

**Why:** Silent failures in production are unacceptable. SEC can't tell when events are corrupt.

| #   | Task                                                                               | Effort | Impact | Files                                   |
| --- | ---------------------------------------------------------------------------------- | ------ | ------ | --------------------------------------- |
| 15  | Add slog.Warn for corrupt IDs in Pebble deserialization                            | 8min   | HIGH   | `storage/pebble_serialization.go:76-88` |
| 16  | Return error from Pebble iterateEvents instead of silently skipping corrupt events | 8min   | HIGH   | `storage/pebble_event_store.go:120-123` |
| 17  | Add slog.Warn in OutboxPublisher.publishPending for failed publishes               | 5min   | HIGH   | `core/event/outbox_publisher.go:221`    |
| 18  | Add duplicate projection name check in Runner.Register                             | 5min   | MEDIUM | `projection/runner.go:66-74`            |
| 19  | Add clock injection option `WithClock(func() time.Time)` to NewEvent               | 8min   | MEDIUM | `core/event/event.go:226`               |

---

## Phase 4: Storage Quality (MEDIUM-HIGH)

**Why:** Storage is the lowest-coverage module (88.1%). SEC uses it for Turso persistence.

| #   | Task                                                                                  | Effort | Impact | Files                                          |
| --- | ------------------------------------------------------------------------------------- | ------ | ------ | ---------------------------------------------- |
| 20  | Add Pebble deserialization split: extract `deserializeMetadata`, `deserializeOptions` | 10min  | MEDIUM | `storage/pebble_serialization.go`              |
| 21  | Add storage DDL onto Dialect interface (Schema/OutboxSchema methods)                  | 10min  | MEDIUM | `storage/dialect.go`, `storage/event_store.go` |
| 22  | Add storage integration test sketch: Turso in-memory save→load→delete roundtrip       | 12min  | HIGH   | `storage/turso_integration_test.go` (new)      |
| 23  | Bump testhelpers to v1.2.0 — update Version types to `event.Version`                  | 8min   | HIGH   | `testhelpers/go.mod`, helpers                  |
| 24  | Remove replace directives from all go.mod files (go.work makes them redundant)        | 10min  | LOW    | all go.mod files                               |

---

## Phase 5: Architecture Cleanup (MEDIUM)

**Why:** Reduce maintenance burden. Duplicated patterns increase bug surface.

| #   | Task                                                                      | Effort | Impact | Files                                                                        |
| --- | ------------------------------------------------------------------------- | ------ | ------ | ---------------------------------------------------------------------------- |
| 25  | Extract shared `event.WalkMessages(cat, fn)` helper for catalog exporters | 10min  | MEDIUM | `catalog/walk.go` (new)                                                      |
| 26  | Unify ErrNilBus/ErrNilStore sentinels into `core/event/errors.go`         | 8min   | MEDIUM | `core/aggregate/errors.go`, `core/decider/errors.go`, `projection/errors.go` |
| 27  | Delete or deprecate `core/event/catalog.go` with migration comment        | 5min   | LOW    | `core/event/catalog.go`                                                      |
| 28  | Standardize version references across go.mod files (v0.0.0 vs v1.1.0)     | 8min   | LOW    | all go.mod files                                                             |
| 29  | Add projection.Runner position-based optimization sketch (interface only) | 10min  | MEDIUM | `projection/runner.go`                                                       |

---

## Phase 6: Documentation & Release Prep (MEDIUM)

**Why:** Consumers need onboarding docs. Currently README is a reference manual.

| #   | Task                                                                    | Effort | Impact     | Files                     |
| --- | ----------------------------------------------------------------------- | ------ | ---------- | ------------------------- |
| 30  | Write getting-started README section: "Your first CQRS app in 30 lines" | 12min  | 🔥 HIGHEST | `README.md`               |
| 31  | Write API migration guide: `query.Handler any → TypedHandler[T]`        | 8min   | HIGH       | `docs/MIGRATION.md` (new) |
| 32  | Update FEATURES.md with example/todo in Module Maturity Matrix          | 5min   | LOW        | `FEATURES.md`             |
| 33  | Update TODO_LIST.md: mark Phase 1-3 items as completed                  | 5min   | LOW        | `TODO_LIST.md`            |
| 34  | Write CONTRIBUTING.md — architecture guidelines                         | 10min  | MEDIUM     | `CONTRIBUTING.md` (new)   |

---

## Phase 7: Tagged Release (HIGH IMPACT)

**Why:** 77 sessions of work. No tagged release. SEC is importing `v0.4.0` of catalog but we're far past that internally.

| #   | Task                                                                   | Effort | Impact     | Files                            |
| --- | ---------------------------------------------------------------------- | ------ | ---------- | -------------------------------- |
| 35  | Audit all go.mod module paths for consistency                          | 5min   | HIGH       | all go.mod                       |
| 36  | Tag core v1.4.0 (breaking: IdempotencyKey added, Version type changed) | 5min   | 🔥 HIGHEST | git tag                          |
| 37  | Tag memory v1.2.0                                                      | 3min   | HIGH       | git tag                          |
| 38  | Tag catalog v0.5.0 (breaking: MessageID → GetID, SchemaType added)     | 3min   | HIGH       | git tag                          |
| 39  | Tag middleware v1.1.0                                                  | 3min   | HIGH       | git tag                          |
| 40  | Tag testhelpers v1.2.0                                                 | 3min   | HIGH       | git tag                          |
| 41  | Tag projection v1.0.0                                                  | 3min   | HIGH       | git tag                          |
| 42  | Tag storage v0.2.0                                                     | 3min   | HIGH       | git tag                          |
| 43  | Tag sync v0.1.0                                                        | 3min   | MEDIUM     | git tag                          |
| 44  | Update SEC to use new tags and verify build                            | 8min   | 🔥 HIGHEST | `/home/lars/projects/SEC/go.mod` |

---

## Phase 8: Example Quality (MEDIUM)

**Why:** Examples are the first thing consumers see. Both are stale.

| #   | Task                                                                    | Effort | Impact | Files                    |
| --- | ----------------------------------------------------------------------- | ------ | ------ | ------------------------ |
| 45  | Rewrite example/user to use TypedHandler + Decider pattern + full stack | 12min  | HIGH   | `example/user/`          |
| 46  | Fix example/todo build (stale storage API references from TODO_LIST)    | 10min  | HIGH   | `example/todo/`          |
| 47  | Add example/user README with architecture diagram                       | 8min   | MEDIUM | `example/user/README.md` |

---

## Phase 9: Future-Looking (LOW — Park for Later)

| #   | Task                                                             | Effort | Impact | Status                |
| --- | ---------------------------------------------------------------- | ------ | ------ | --------------------- |
| 48  | VectorClock.Compare: return enum (Before/After/Equal/Concurrent) | 10min  | LOW    | PLANNED               |
| 49  | Saga/Process Manager implementation                              | 18h    | MEDIUM | PLANNED (design done) |
| 50  | PostgreSQL integration tests for storage                         | 2h     | MEDIUM | PLANNED               |
| 51  | Watermill pub/sub adapter module                                 | 8h     | MEDIUM | PLANNED               |
| 52  | Consolidate CatalogMeta across event/command/query               | 4h     | LOW    | PLANNED               |

---

## Priority Ranking (All 48 Tasks, Sorted by Impact × Urgency / Effort)

| Rank | #   | Task                         | Phase | Effort | Impact | Cust Value                   |
| ---- | --- | ---------------------------- | ----- | ------ | ------ | ---------------------------- |
| 1    | 1   | Fix retry timer leak         | 1     | 5min   | HIGH   | 🔥 Production bug            |
| 2    | 2   | Fix nil codec snapshot       | 1     | 8min   | HIGH   | 🔥 Data loss                 |
| 3    | 3   | Fix Pebble concurrency check | 1     | 10min  | HIGH   | 🔥 Data loss                 |
| 4    | 7   | command.TypedHandler[T]      | 2     | 10min  | 🔥     | 🔥 Eliminates #1 boilerplate |
| 5    | 9   | query.TypedHandler[T]        | 2     | 10min  | 🔥     | 🔥 Eliminates `any` return   |
| 6    | 30  | Getting-started README       | 6     | 12min  | 🔥     | 🔥 First impression          |
| 7    | 12  | event.EncodeEvents helper    | 2     | 10min  | HIGH   | 🔥 Eliminates 100 LOC glue   |
| 8    | 4   | Fix dual %w wrapping         | 1     | 5min   | MEDIUM | Error handling               |
| 9    | 17  | OutboxPublisher slog.Warn    | 3     | 5min   | HIGH   | Observability                |
| 10   | 18  | Duplicate projection check   | 3     | 5min   | MEDIUM | Correctness                  |
| 11   | 5   | LWW nil guard                | 1     | 5min   | MEDIUM | Panic prevention             |
| 12   | 6   | SchemaFromType nil check     | 1     | 5min   | MEDIUM | Panic prevention             |
| 13   | 8   | TypedHandler tests           | 2     | 10min  | HIGH   | Trust                        |
| 14   | 10  | TypedHandler query tests     | 2     | 10min  | HIGH   | Trust                        |
| 15   | 11  | event.TypedProjection[T]     | 2     | 10min  | HIGH   | Type safety                  |
| 16   | 13  | event.DecodePayloads batch   | 2     | 8min   | HIGH   | DX                           |
| 17   | 14  | Update example/user          | 2     | 10min  | HIGH   | Demo quality                 |
| 18   | 15  | Pebble corrupt ID warning    | 3     | 8min   | HIGH   | Observability                |
| 19   | 16  | Pebble corrupt event error   | 3     | 8min   | HIGH   | Correctness                  |
| 20   | 19  | Clock injection option       | 3     | 8min   | MEDIUM | Testability                  |
| 21   | 23  | Bump testhelpers v1.2.0      | 4     | 8min   | HIGH   | Build fix                    |
| 22   | 36  | Tag core v1.4.0              | 7     | 5min   | 🔥     | 🔥 Release                   |
| 23   | 37  | Tag memory v1.2.0            | 7     | 3min   | HIGH   | Release                      |
| 24   | 38  | Tag catalog v0.5.0           | 7     | 3min   | HIGH   | Release                      |
| 25   | 44  | Update SEC to new tags       | 7     | 8min   | 🔥     | 🔥 Consumer validation       |

---

## Summary Statistics

| Metric                            | Value           |
| --------------------------------- | --------------- |
| Total tasks                       | 48 (+ 5 future) |
| Estimated total effort            | ~8.5 hours      |
| Critical bugs                     | 6               |
| Consumer API improvements         | 8               |
| Observability fixes               | 5               |
| Storage quality                   | 5               |
| Architecture cleanup              | 5               |
| Documentation                     | 5               |
| Release tasks                     | 10              |
| Example quality                   | 3               |
| Future (parked)                   | 5               |
| Phase 1 (bugs) estimated          | ~38 min         |
| Phase 2 (consumer API) estimated  | ~78 min         |
| Phase 3 (observability) estimated | ~34 min         |
| Phase 4 (storage) estimated       | ~48 min         |
| Phase 5 (architecture) estimated  | ~41 min         |
| Phase 6 (docs) estimated          | ~40 min         |
| Phase 7 (release) estimated       | ~36 min         |
| Phase 8 (examples) estimated      | ~30 min         |

---

## Key Decisions for Lars

1. **Phase 2 (TypedHandler) is the highest-value work** — SEC has 200+ lines of boilerplate that vanish with typed handlers. This is the difference between "library that works" and "library that's a joy to use."
2. **Phase 7 (tagging) should happen AFTER Phase 2** — the TypedHandler additions are breaking API additions; better to release once with them included.
3. **Phase 9 items are explicitly parked** — Saga, Watermill, PostgreSQL integration tests are future work, not this session.
