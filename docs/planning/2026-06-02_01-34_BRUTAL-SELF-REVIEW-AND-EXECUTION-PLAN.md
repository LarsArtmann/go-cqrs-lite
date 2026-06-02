# Brutal Self-Review & Execution Plan

Date: 2026-06-02

## Self-Review Answers

### 1. What did you forget?

- **command/reactive.go and query/reactive.go are ghost systems** — zero production usage outside their own test files. `NewCommandBus`, `NewQueryBus`, `FilterCommandType`, `FilterQueryType`, `Observable` aliases — all dead code with tests that prove they work but nothing wires them.
- **event/reactive.go is a split brain** — contains both dead reactive primitives (`NewEventBus`, `FilterEventType`, `Map`, `Observable`) AND genuinely useful infrastructure (`HandlerToObserver`, `ReplayFilter`, `ScanState`, `Tap`).
- **command.Metadata is a 100% field-level duplicate** of event.Metadata — 4 shared fields (CorrelationID, CausationID, UserID, RequestID) copy-pasted.
- **Three separate `ErrHandlerNotFound`** sentinels across command/query/dispatcher + **three separate `ErrDispatcherClosed`**. Same semantic, different error codes, different packages.
- **Testify in 5 files** — banned library per project policy. `integration/chaos_test.go`, `otel/otel_test.go`, `otel/logging_test.go`, `projection/health_test.go`, `middleware/tracing_logging_test.go`.
- **command/aggregate_ref.go re-exports event types** — boundary violation, couples command to event internals.
- **schema.VersionedStore exposes embedded event.Store publicly** — consumers can bypass versioning.
- **Integration test directory contains misplaced unit tests** — `integration/command/command_test.go` and `integration/query/query_test.go` test single-module behavior, not cross-module integration.

### 2. What's stupid that we do anyway?

- **Middleware 3× fan-out** — 27 nearly-identical public functions across 9 files. Every bug fix applied in triplicate. ~500 lines of duplication. This is the single biggest code quality problem.
- **event/errors.go re-exports the entire errorfamily surface** — non-event code imports `event.WrapRejection` which is semantically wrong.
- **ImmutableEvent lives in event.go** alongside 4 other concerns (Type, AggregateType, Event interface, parse helpers). File is within the 250-line limit but mixes responsibilities.

### 3. What could you have done better?

- The CatalogDispatcher removal was clean but I should have caught the `wrapcheck` and `gci` issues before the first test run.
- I should have noticed that command/reactive.go and query/reactive.go are ghost systems during the reactive extension discussion.

### 4. What could you still improve?

See Execution Plan below — 14 actionable steps.

### 5. Did you lie to you?

No. All findings are backed by `grep` and source code analysis. The gopls errors for `dispatcher/catalog_test.go` are phantom cache — the file was deleted and committed.

### 6. How can we be less stupid?

- **Unify error sentinels** — one `ErrHandlerNotFound` in dispatcher/, re-exported by command/ and query/.
- **Unify Metadata** — extract shared fields to a base type, or let command embed event.Metadata.
- **Generic middleware** — collapse 27 functions to ~9 generics.

### 7. Ghost systems?

- **command/reactive.go** — ghost. Zero production usage. Plan: integrate `ro.Subject[Command]` into command.Dispatcher.
- **query/reactive.go** — ghost. Zero production usage. Plan: integrate `ro.Subject[Query]` into query.Dispatcher.
- **event/reactive.go** — split brain. Parts are ghost (buses, filters), parts are real (HandlerToObserver, ReplayFilter). Plan: integrate into dispatcher; keep HandlerToObserver as utility.
- **saga-pattern/** — NOT a ghost. Active example with tests.

### 8. Scope creep trap?

Yes risk. The middleware 3× dedup is a large refactor. Must be surgical — generic per concern, thin adapters for backward compat, don't change public API names.

### 9. Did we remove something useful?

No. CatalogDispatcher was verified dead — zero non-test consumers, catalog/ has its own independent registration system.

### 10. Split brains?

| Split Brain | Location | Severity |
|-------------|----------|----------|
| Metadata fields | command/metadata.go vs event/metadata.go | HIGH |
| Error sentinels | command/errors.go vs query/errors.go vs dispatcher/errors.go | HIGH |
| Error re-export | event/errors.go exports errorfamily for all modules | MEDIUM |
| Type string pattern | event.Type, command.Type, query.Type — identical pattern | LOW (acceptable isolation) |
| Reactive ghost | command/reactive.go, query/reactive.go — unused reactive types | MEDIUM |
| Boundary violation | command/aggregate_ref.go re-exports event types | HIGH |

### 11. Tests?

- 84-100% coverage across modules
- Testify in 5 files (banned) — should migrate to gomega
- 2 integration test files are misplaced unit tests
- turso module has zero test coverage
- Missing: fuzz tests, E2E throughput benchmarks

---

## Execution Plan

Sorted by **impact / effort** (Pareto: 1%→51%, 4%→64%, 20%→80%).

### Phase 1: Quick Wins (1% → 51% impact, ~30min each)

| # | Task | Impact | Effort | Files |
|---|------|--------|--------|-------|
| 1 | Replace testify with gomega in 5 test files | Policy compliance | 30min | 5 test files + 2 go.mod |
| 2 | Consolidate ErrHandlerNotFound + ErrDispatcherClosed into dispatcher/, re-export from command/query | Eliminates 4 duplicate sentinels | 30min | 6 error files + callers |
| 3 | Fix command/aggregate_ref.go boundary violation — remove event type re-exports | Clean module boundary | 15min | 1-2 files |

### Phase 2: Type Model Fixes (4% → 64% impact, ~30-60min each)

| # | Task | Impact | Effort | Files |
|---|------|--------|--------|-------|
| 4 | Extract shared Metadata — command.Metadata embeds event.Metadata or shared base | Eliminates split brain | 45min | 3-4 files |
| 5 | Fix schema.VersionedStore — hide embedded event.Store | Prevents bypass | 15min | 1-2 files |
| 6 | Extract ImmutableEvent to immutable.go (from event.go) | Single responsibility | 15min | 2 files |

### Phase 3: Middleware Generics (20% → 80% impact, ~2-3h)

| # | Task | Impact | Effort | Files |
|---|------|--------|--------|-------|
| 7 | Create generic Middleware[H] per concern in middleware/ — retry, recovery, logging, validation, metrics | Removes ~300 lines of duplication | 2h | 9 files |

### Phase 4: Reactive Integration (post-v2, ~1-2h)

| # | Task | Impact | Effort | Files |
|---|------|--------|--------|-------|
| 8 | Embed ro.Subject[M] in Dispatcher[H,M] — emit on every dispatch | Eliminates ghost system | 1h | 3-5 files |
| 9 | Remove standalone reactive.go type aliases from command/ and query/ | Dead code removal | 15min | 2-4 files |
| 10 | Clean up event/reactive.go — keep utilities, remove dead bus/filter aliases | Split brain fix | 30min | 1-2 files |

### Phase 5: Test Quality (~1h)

| # | Task | Impact | Effort | Files |
|---|------|--------|--------|-------|
| 11 | Move misplaced integration/command + integration/query tests to their home packages | Test hygiene | 15min | 4 files |
| 12 | Extract withRLock/withWLock helpers in memory/ | DRY | 15min | ~5 files |

### Phase 6: Documentation & Final Cleanup

| # | Task | Impact | Effort | Files |
|---|------|--------|--------|-------|
| 13 | Update AGENTS.md with new architecture state | Memory freshness | 15min | 1 file |
| 14 | Update TODO_LIST.md — mark completed items | Tracking accuracy | 15min | 1 file |

---

## Pareto Summary

- **Phase 1** (3 tasks, ~75min): Eliminates banned dependency, duplicate sentinels, boundary violation
- **Phase 2** (3 tasks, ~75min): Fixes metadata split brain, VersionedStore leak, SRP violation
- **Phase 3** (1 task, ~2h): Removes 300+ lines of middleware duplication
- **Phase 4** (3 tasks, ~2h): Integrates reactive into dispatchers (post-v2)
- **Phase 5** (2 tasks, ~30min): Test hygiene
- **Phase 6** (2 tasks, ~30min): Documentation

**Total: ~7h of focused work.**

## D2 Execution Graph

```d2
phase_1: Phase 1 — Quick Wins {
  shape: rectangle
  testify: Replace testify with gomega
  errors: Consolidate error sentinels
  boundary: Fix aggregate_ref boundary

  testify -> errors -> boundary
}

phase_2: Phase 2 — Type Model Fixes {
  shape: rectangle
  metadata: Extract shared Metadata
  versioned: Fix VersionedStore exposure
  immutable: Extract ImmutableEvent

  metadata -> versioned -> immutable
}

phase_3: Phase 3 — Middleware Generics {
  shape: rectangle
  generics: Generic Middleware[H] per concern
}

phase_4: Phase 4 — Reactive Integration {
  shape: rectangle
  embed: Embed ro.Subject in Dispatcher
  cleanup: Remove ghost reactive files
  event_clean: Clean event/reactive.go
}

phase_5: Phase 5 — Test Quality {
  shape: rectangle
  move_tests: Move misplaced integration tests
  lock_helpers: Extract withRLock helpers
}

phase_6: Phase 6 — Documentation {
  shape: rectangle
  agents: Update AGENTS.md
  todos: Update TODO_LIST.md
}

phase_1 -> phase_2 -> phase_3 -> phase_4 -> phase_5 -> phase_6
```
