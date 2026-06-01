# Session 159 — samber/ro Reactive Stream Integration Restored

**Date:** 2026-05-30
**Status:** In Progress

---

## What Happened

Session 156 **deleted** `event/reactive.go` and `command/reactive.go` using the Application Lens ("zero internal consumers = dead code") — the exact anti-pattern called out at the top of AGENTS.md. This session restored the integration with fixes and added `query/reactive.go` (new).

---

## Fully Done

| #   | Task                                                                                                                                                   | Files                                |
| --- | ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------ |
| 1   | Add `samber/ro v0.3.0` to event, command, query go.mod                                                                                                 | 3 go.mod + 3 go.sum                  |
| 2   | Create `event/reactive.go` — EventBus, NewEventBus, FilterEventType, FilterEventTypes, HandlerToObserver (fixed), HandlerToObserverWithContext (fixed) | event/reactive.go (69 lines)         |
| 3   | Create `command/reactive.go` — CommandBus, NewCommandBus, FilterCommandType                                                                            | command/reactive.go (21 lines)       |
| 4   | Create `query/reactive.go` — QueryBus, NewQueryBus, FilterQueryType (NEW — never existed before)                                                       | query/reactive.go (21 lines)         |
| 5   | Create `event/reactive_test.go` — 8 tests covering publish/subscribe, filtering, error propagation, context preservation                               | event/reactive_test.go (383 lines)   |
| 6   | Create `command/reactive_test.go` — 3 tests                                                                                                            | command/reactive_test.go (106 lines) |
| 7   | Create `query/reactive_test.go` — 3 tests                                                                                                              | query/reactive_test.go (86 lines)    |
| 8   | Update `.golangci.yml` — add samber/ro + samber/lo to depguard allow list                                                                              | .golangci.yml                        |
| 9   | Update AGENTS.md — reactive types in module tree, deps, module graph, key patterns, anti-anti-pattern table                                            | AGENTS.md                            |
| 10  | Update CHANGELOG.md — document restoration + improvements                                                                                              | CHANGELOG.md                         |
| 11  | Full test suite passes (32 packages, 0 failures)                                                                                                       | —                                    |
| 12  | Zero lint issues across all 29 modules                                                                                                                 | —                                    |

### Key Fixes Over Deleted Version

| Issue in session 154                                                                   | Fix in this session                                                         |
| -------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `HandlerToObserver` silently dropped errors via `_ = handler(context.Background(), e)` | Explicit `onError func(error)` callback — handler errors flow to the caller |
| Used `context.Background()` — stripped deadlines, tracing, correlation IDs             | Uses `e.Context()` — preserves the event's own context                      |
| `HandlerToObserverWithContext` also dropped errors                                     | Same `onError` callback pattern                                             |
| No `query/reactive.go`                                                                 | Added QueryBus, NewQueryBus, FilterQueryType                                |
| Anti-pattern table was ambiguous ("correct isolation")                                 | Rewrote to "DELETING EXTERNAL-FACING API IS BREAKING THE PRODUCT"           |

---

## Partially Done

None.

---

## Not Started — High-Value samber/ro Improvements

### Consumer-Facing API Gaps

| #   | Gap                                                                    | Why it matters                                                                                                                                                    | Effort |
| --- | ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 1   | **No `NewReplayEventBus(N)`** — ReplaySubject for late subscribers     | Late-joining projections need historical events. ReplaySubject(buffer=N) replays last N events + all future ones. This is the core catch-up subscription pattern. | 10min  |
| 2   | **No `NewBehaviorEventBus(initial)`** — BehaviorSubject for state      | BehaviorSubject replays latest value. Perfect for "current state + future updates" pattern — decider state snapshots, projection state queries.                   | 10min  |
| 3   | **No `Scan` operator** — running state accumulation over event streams | `Scan` IS projection building. `ro.Scan(initialState, accumulator)` over an event stream produces projection state. Core CQRS primitive.                          | 15min  |
| 4   | **No `Map` operator** — transform events in reactive pipelines         | Map events to different shapes, extract payloads, project to read models.                                                                                         | 5min   |
| 5   | **No `Distinct` operator** — idempotent event processing               | Deduplicate events by AggregateID or Type. Critical for at-least-once delivery semantics.                                                                         | 10min  |
| 6   | **No `Tap` operator** — side-effect-free logging/metrics               | Insert logging, OTel tracing, metrics into reactive pipelines without affecting the stream.                                                                       | 5min   |
| 7   | **No `Collect`/`ToSlice`** — materialize observable to slice           | Testing utility — gather all emitted values into a slice for assertion.                                                                                           | 5min   |
| 8   | **No `Retry` + `Catch` operators** — resilient stream processing       | Replace the 60-line manual `handleWithRetry` in projection/runner with declarative operators.                                                                     | 20min  |
| 9   | **No `StartWith` operator** — seed replay before live                  | Prepend replay events to a live stream. Enables seamless replay-then-live handoff for projections.                                                                | 10min  |
| 10  | **No `ToChannel` operator** — bridge reactive → channels               | For consumers using channel-based patterns (select, ranges).                                                                                                      | 5min   |

### Integration & Architecture Gaps

| #   | Gap                                                 | Why it matters                                                                                                                 | Effort |
| --- | --------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ------ |
| 11  | **No integration test for reactive types**          | The reactive bus is not tested in the cross-module integration suite. integration/full_flow_test.go is imperative only.        | 20min  |
| 12  | **No reactive example**                             | examples/ don't demonstrate reactive API usage. Consumers have no reference.                                                   | 30min  |
| 13  | **projection/ runner doesn't use reactive streams** | The runner uses hand-rolled callback-based subscription. It could use ReplaySubject + Scan + Retry for a cleaner architecture. | 60min  |
| 14  | **No reactive middleware**                          | No `ro.Tap`-based logging/metrics middleware for reactive pipelines (distinct from Bus.Use middleware).                        | 15min  |

### Type Model Improvements

| #   | Idea                                                                                     | Why it matters                                                                                                                        | Effort |
| --- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| 15  | **`event.NewReplayEventBus(N)`** + **`event.NewBehaviorEventBus(initial)`** constructors | Expose ReplaySubject and BehaviorSubject as first-class constructors (not just PublishSubject).                                       | 10min  |
| 16  | **`event.ObservableEvent` = `ro.Observable[Event]`** type alias                          | Give consumers a named type for event observables instead of bare `ro.Observable[Event]`.                                             | 2min   |
| 17  | **`event.ScanState[S]` operator** — typed projection scanner                             | `func ScanState[S any](initial S, fold func(S, Event) S) func(ro.Observable[Event]) ro.Observable[S]` — type-safe state accumulation. | 10min  |
| 18  | **`event.DistinctByAggregateID()` operator**                                             | Deduplicate events by AggregateID — common enough to be a named export.                                                               | 5min   |
| 19  | **`command.HandlerToObserver` adapter** (like event's)                                   | Command handlers also need context-preserving, error-forwarding observer adapters.                                                    | 10min  |
| 20  | **`query.DispatchTypedAsync[T]`** — reactive query dispatch                              | Dispatch a query and get a `ro.Observable[T]` back instead of `(T, error)`.                                                           | 15min  |

### Established Lib Improvements

| #   | Idea                                                                                                                                               | Why it matters                                                                                        | Effort       |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ------------ |
| 21  | **Use `ro.Pipe1` consistently** — already done in tests, but reactive.go exports raw `ro.Filter` which returns `func(ro.Observable) ro.Observable` | Consider wrapping Pipe1 usage in exported helpers.                                                    | Low priority |
| 22  | **`ro.Interval` for periodic checkpointing**                                                                                                       | Replace `time.Sleep` in projection runner with `ro.Interval` for scheduled checkpoint writes.         | 15min        |
| 23  | **`ro.ForkJoin` for parallel projection dispatch**                                                                                                 | Replace manual semaphore-based parallel dispatch in projection/runner.                                | 20min        |
| 24  | **`eventtest.FakeReactiveBus`** — test helper for reactive bus                                                                                     | Add a FakeBus-backed reactive subject to eventtest/ for consumer testing.                             | 15min        |
| 25  | **`ro.WithLatestFrom` for event enrichment**                                                                                                       | Enrich events with current state from another stream (e.g., add aggregate version to command events). | 10min        |

---

## Totally Fucked Up

- Session 156's deletion of reactive.go was a textbook violation of the library anti-pattern table. Now fixed with a stronger warning in AGENTS.md.

---

## What We Should Improve

1. **Reactive constructor diversity** — Only PublishSubject is exposed. ReplaySubject and BehaviorSubject are the #1 most valuable additions for CQRS (catch-up subscriptions, state snapshots).
2. **Operator library** — We expose 1 operator (Filter). samber/ro has 50+. We should expose at minimum: Scan, Map, Distinct, Tap, Collect, Retry, Catch.
3. **Projection runner → reactive** — The projection module pulls in `samber/ro` transitively but uses hand-rolled callback loops. It should use reactive streams internally.
4. **Integration test coverage** — No cross-module reactive test. The integration suite should exercise reactive bus → projection → query flow.
5. **Type aliases for readability** — `ro.Observable[Event]` should be `event.Observable` for discoverability.
6. **Command observer adapter** — Command has reactive bus but no `HandlerToObserver` equivalent.

---

## Top 25 Things We Should Get Done Next

Sorted by impact × effort (highest first):

| Priority | Task                                                             | Impact     | Effort |
| -------- | ---------------------------------------------------------------- | ---------- | ------ |
| 1        | Add `NewReplayEventBus(N)` — ReplaySubject constructor           | ⭐⭐⭐⭐⭐ | 10min  |
| 2        | Add `NewBehaviorEventBus(initial)` — BehaviorSubject constructor | ⭐⭐⭐⭐⭐ | 10min  |
| 3        | Add `ScanState[S]` operator — typed projection state builder     | ⭐⭐⭐⭐⭐ | 10min  |
| 4        | Add `Map` operator — event stream transformation                 | ⭐⭐⭐⭐   | 5min   |
| 5        | Add `Distinct` operator — idempotent processing                  | ⭐⭐⭐⭐   | 10min  |
| 6        | Add `Tap` operator — logging/metrics side effects                | ⭐⭐⭐⭐   | 5min   |
| 7        | Add `Collect`/`ToSlice` — test materialization                   | ⭐⭐⭐     | 5min   |
| 8        | Add `DistinctByAggregateID()` — named dedup operator             | ⭐⭐⭐     | 5min   |
| 9        | Add `event.Observable` type alias                                | ⭐⭐⭐     | 2min   |
| 10       | Add `command.HandlerToObserver` adapter                          | ⭐⭐⭐     | 10min  |
| 11       | Add `ToChannel` operator — bridge to channels                    | ⭐⭐⭐     | 5min   |
| 12       | Add tests for all new operators                                  | ⭐⭐⭐⭐   | 20min  |
| 13       | Update AGENTS.md with new operators                              | ⭐⭐       | 5min   |
| 14       | Update CHANGELOG.md with new operators                           | ⭐⭐       | 3min   |
| 15       | Add integration test for reactive bus → projection flow          | ⭐⭐⭐⭐   | 20min  |
| 16       | Add `Retry` + `Catch` operators                                  | ⭐⭐⭐⭐   | 20min  |
| 17       | Add `StartWith` operator — replay seed before live               | ⭐⭐⭐⭐   | 10min  |
| 18       | Refactor projection runner to use ReplaySubject internally       | ⭐⭐⭐⭐⭐ | 60min  |
| 19       | Add `eventtest.FakeReactiveBus` test helper                      | ⭐⭐⭐     | 15min  |
| 20       | Add reactive example to examples/                                | ⭐⭐⭐     | 30min  |
| 21       | Add `query.DispatchTypedAsync[T]`                                | ⭐⭐⭐     | 15min  |
| 22       | Use `ro.Interval` for periodic checkpointing in projection       | ⭐⭐       | 15min  |
| 23       | Use `ro.ForkJoin` for parallel projection dispatch               | ⭐⭐       | 20min  |
| 24       | Add `ro.WithLatestFrom` for event enrichment                     | ⭐⭐       | 10min  |
| 25       | Add `ro.Debounce`/`ro.Throttle` for rate-limiting                | ⭐⭐       | 10min  |

---

## Files Changed This Session

### New files (6)

- `event/reactive.go` — 69 lines
- `event/reactive_test.go` — 383 lines
- `command/reactive.go` — 21 lines
- `command/reactive_test.go` — 106 lines
- `query/reactive.go` — 21 lines
- `query/reactive_test.go` — 86 lines

### Modified files

- `.golangci.yml` — depguard allow samber/ro, samber/lo
- `AGENTS.md` — reactive types, deps, module graph, anti-pattern table
- `CHANGELOG.md` — restoration entry
- 23 `go.mod` + 23 `go.sum` — cascade tidy from samber/ro addition

### Verification

- Full test suite: 32 packages, 0 failures
- Lint: 0 issues across all 29 modules
