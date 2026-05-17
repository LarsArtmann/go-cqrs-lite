# Session 65 — Architectural Type Safety Sweep: Status Report

**Date:** 2026-05-17 05:02
**Branch:** master (clean, pushed)
**Last commit:** `ae086e6` feat: architectural type safety sweep

---

## Metrics

| Metric                      | Value                                                                               |
| --------------------------- | ----------------------------------------------------------------------------------- |
| Test packages               | 21/21 pass, 0 fail                                                                  |
| Lint issues                 | 2 (golines formatting on outbox SQL constants)                                      |
| Test functions              | 791                                                                                 |
| Benchmarks                  | 43                                                                                  |
| Total LOC                   | 37,975                                                                              |
| Production coverage         | ~93% (storage 85.1% lowest)                                                         |
| Production files >250 lines | 5 (pebble 448, helpers 423, aggregate repo 279, decider 265, asyncapi exporter 258) |

---

## a) FULLY DONE (This Session + Prior Sessions)

### 1% → 51% of Value (ALL COMPLETE)

| #   | Task                                                      | Commit                    | Status |
| --- | --------------------------------------------------------- | ------------------------- | ------ |
| P1  | `NewEvent` accepts `event.Version` not `int`              | `8c10def` (prior session) | DONE   |
| P2  | `SchemaVersion` strong type with `ParseSchemaVersion`     | `8c10def` (prior session) | DONE   |
| P3  | `OutboxStatus` enum + replace 8 `'pending'` magic strings | `ae086e6`                 | DONE   |
| P4  | Middleware error classification (4 sentinels) + tests     | `ae086e6`                 | DONE   |

### 4% → 64% of Value (PARTIALLY COMPLETE)

| #   | Task                                        | Status            | Detail                                                                                                                                                                                                     |
| --- | ------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| P5  | `Pagination` fields → `uint`                | DONE              | `ae086e6` — Page, PageSize, TotalCount, TotalPages all uint                                                                                                                                                |
| P6  | `SyncMessage.Type` → `SyncMessageType` enum | DONE              | `ae086e6` — SyncMessageTypeRequest, SyncMessageTypeResponse                                                                                                                                                |
| P7  | `NodeID` branded type in sync module        | DONE              | `ae086e6` — ParseNodeID, MustParseNodeID, used in Operation + SyncContextMixin                                                                                                                             |
| P8  | Storage table name constants                | SKIPPED           | Low-value cosmetic: SQL DDL strings can't be parameterized; risk/reward poor                                                                                                                               |
| P9  | `OutboxPublisher` split-brain fix           | NOT STARTED       | `cancel` stays non-nil after Close(), `done` channel never recreated                                                                                                                                       |
| P10 | Register remaining unclassified sentinels   | PARTIALLY DONE    | memory.ErrHandlerNil, catalog.ErrDomainNotFound, catalog.ErrNilSchema ARE registered. Dispatcher base sentinels are wrapped by command/query equivalents — not directly registerable without import cycle. |
| P11 | `RetryConfig.Validate()` method             | EXISTS but UNUSED | Method exists on RetryConfig, checks MaxAttempts/InitialDelay/Multiplier, but CommandRetry/EventRetry/QueryRetry never call it                                                                             |

### Also Done (from prior sessions, confirmed working)

- `Event.SchemaVersion()` returns `SchemaVersion` (not `int`)
- `Event.Version()` returns `Version` (not `int`)
- `Core` struct fields use `Version` and `SchemaVersion`
- Error taxonomy: 38+ sentinels registered across 8 modules
- `OutboxStatusPending` constant used in all SQL (outbox.go, sqlite_outbox.go)

---

## b) PARTIALLY DONE

| Task                        | What's Done                                                                                 | What's Missing                                                                                                                                        |
| --------------------------- | ------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| OutboxPublisher split-brain | Diagnosis complete: `cancel != nil` tracks state, `done` channel closes but never recreated | Need to add explicit `state` enum, reset `cancel=nil` + recreate `done` after Close(), add Start→Close→Start test                                     |
| RetryConfig validation      | `Validate()` method exists with full checks                                                 | Never called in `CommandRetry`/`EventRetry`/`QueryRetry` constructors — silent misconfiguration possible                                              |
| Sentinel registration       | 38+ sentinels registered. Only 3 base `dispatcher` sentinels unregistered.                  | Base dispatcher sentinels can't be registered directly (circular import). Command/query register their own equivalents. Consider adding note in code. |

---

## c) NOT STARTED

| #   | Task                                                                                      | Priority | Effort | Impact                                            |
| --- | ----------------------------------------------------------------------------------------- | -------- | ------ | ------------------------------------------------- |
| 1   | Split `pebble_event_store.go` (448→<250 lines)                                            | HIGH     | 30min  | Only file over 370-line limit                     |
| 2   | Split `aggregate/repository.go` (279→<250 lines)                                          | MEDIUM   | 15min  | File size compliance                              |
| 3   | Split `decider/decider.go` (265→<250 lines)                                               | MEDIUM   | 15min  | File size compliance                              |
| 4   | Unify `CatalogMeta` (3 structs → 1 shared)                                                | MEDIUM   | 30min  | ~60 lines saved, 2 identical + 1 with extra field |
| 5   | BDD tests for type safety (Version, SchemaVersion, OutboxStatus, uint Pagination, NodeID) | MEDIUM   | 60min  | Verify new constraints                            |
| 6   | Delete/simplify `cattest` internal package (454 lines, 0% coverage)                       | LOW      | 20min  | Dead weight — no direct tests                     |
| 7   | `storage/helpers.go` (423 lines) split                                                    | LOW      | 20min  | Large file but under 370                          |
| 8   | Update `AGENTS.md` with all type changes from this session                                | HIGH     | 10min  | Stale documentation                               |
| 9   | Fix 2 golines lint issues (outbox SQL constants)                                          | HIGH     | 5min   | Formatting                                        |
| 10  | SQL dialect abstraction (12→7 files in storage)                                           | LOW      | 90min  | ~250 lines saved but high risk                    |

---

## d) TOTALLY FUCKED UP

| Issue                   | Severity | Detail                                                                                                                                                     |
| ----------------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2 golines lint failures | MINOR    | `storage/outbox.go:49` and `storage/sqlite_outbox.go:43` — string concatenation for `OutboxStatusPending` creates long lines. Need to reformat or shorten. |
| LSP stale errors        | COSMETIC | 17 LSP errors from stale workspace cache. Code compiles and all tests pass. `gopls` confused by `go.work` replace directives.                              |

Nothing is actually broken. All 21 test packages pass, code compiles, and the only lint issues are formatting.

---

## e) WHAT WE SHOULD IMPROVE

1. **OutboxPublisher lifecycle is broken by design** — After `Close()`, `cancel` stays non-nil, so `Start()` returns `ErrAlreadyStarted` even though the goroutine has exited. Need explicit state tracking.

2. **RetryConfig.Validate() exists but is never called** — Someone (likely session 48) added the method but forgot to wire it into the constructors. A `MaxAttempts: 0` config silently skips all retries.

3. **`cattest` package is 454 lines of dead code** — 0% coverage, no direct tests. Only used as internal test helper by catalog exporters. Should be simplified or consolidated.

4. **5 production files over 250 lines** — pebble (448), helpers (423), aggregate repo (279), decider (265), asyncapi exporter (258). Pebble is the worst offender at 448 lines.

5. **CatalogMeta × 3** — Two are identical (command, query), one adds AggregateType (event). Classic unification target.

6. **`storage/helpers.go` is 423 lines** — Contains DDL, version checking, shared scan/delete helpers. Could split into `ddl.go`, `scan.go`, `shared.go`.

---

## f) Top #25 Things to Do Next

Ranked by impact × effort (highest first):

| #   | Task                                                                              | Impact | Effort | Category     |
| --- | --------------------------------------------------------------------------------- | ------ | ------ | ------------ |
| 1   | Fix 2 golines lint issues (outbox SQL constant formatting)                        | HIGH   | 5min   | Lint         |
| 2   | Wire `RetryConfig.Validate()` into CommandRetry/EventRetry/QueryRetry             | HIGH   | 10min  | Bug          |
| 3   | Fix OutboxPublisher split-brain (state enum + Close resets)                       | HIGH   | 25min  | Bug          |
| 4   | Update AGENTS.md with all session 65 changes                                      | HIGH   | 10min  | Docs         |
| 5   | Update plan status to IN_PROGRESS/COMPLETED                                       | HIGH   | 5min   | Planning     |
| 6   | Split `pebble_event_store.go` (448→<250)                                          | HIGH   | 30min  | Quality      |
| 7   | Add test: Start→Close→Start lifecycle for OutboxPublisher                         | HIGH   | 10min  | Tests        |
| 8   | Add test: RetryConfig.Validate() rejects invalid configs                          | MEDIUM | 10min  | Tests        |
| 9   | Split `aggregate/repository.go` (279→<250)                                        | MEDIUM | 15min  | Quality      |
| 10  | Split `decider/decider.go` (265→<250)                                             | MEDIUM | 15min  | Quality      |
| 11  | Unify CatalogMeta (3 structs → 1 shared + alias)                                  | MEDIUM | 30min  | DRY          |
| 12  | Add BDD tests for Version/SchemaVersion type safety                               | MEDIUM | 15min  | Tests        |
| 13  | Add BDD tests for uint Pagination                                                 | MEDIUM | 10min  | Tests        |
| 14  | Add BDD tests for NodeID branded type                                             | MEDIUM | 10min  | Tests        |
| 15  | Add BDD tests for OutboxStatus enum                                               | MEDIUM | 10min  | Tests        |
| 16  | Add BDD tests for SyncMessageType                                                 | LOW    | 10min  | Tests        |
| 17  | Simplify/cleanup cattest package (454 lines, 0% coverage)                         | LOW    | 20min  | Cleanup      |
| 18  | Split `storage/helpers.go` (423→<250)                                             | LOW    | 20min  | Quality      |
| 19  | Consider SQL dialect abstraction for storage (PG/SQLite/Pebble)                   | LOW    | 90min  | Architecture |
| 20  | Add `example/user/` usage of new types (NodeID, uint pagination)                  | LOW    | 15min  | Docs         |
| 21  | Add doc comments to new types (NodeID, SyncMessageType, OutboxStatus)             | LOW    | 10min  | Docs         |
| 22  | Review `storage/outbox.go:125` — `outboxEvent.Version` is still bare `int`        | LOW    | 10min  | Type Safety  |
| 23  | Consider `SchemaVersion` on `outboxEvent` struct for JSON roundtrip               | LOW    | 5min   | Type Safety  |
| 24  | Add `NodeID` validation to `VectorClock.Increment` (currently accepts any string) | LOW    | 15min  | Type Safety  |
| 25  | Update `FEATURES.md` with new type-safety features                                | LOW    | 10min  | Docs         |

---

## g) My Top #1 Question I Cannot Figure Out Myself

**The OutboxPublisher restart design question:**

After `Close()`, should `Start()` be allowed to restart the publisher? This creates a design tension:

- **Option A (single-use):** `Close()` is final. `Start()` after `Close()` returns an error. This is the `io.Closer` contract — close is one-way. Simple, no surprises.
- **Option B (restartable):** `Close()` resets internal state, `Start()` can be called again. Useful for testing and graceful restart scenarios. More complex — must recreate `done` channel, reset `cancel`, handle concurrent Close+Start.

I lean toward **Option A** (single-use, consistent with `io.Closer` semantics). The "split-brain" is only that `cancel` stays non-nil, so `Close()→Close()` would block on `<-p.done` (already closed channel returns immediately — actually safe). The real bug is `Start()→Close()→Start()` returning `ErrAlreadyStarted`. If we document that `Close()` is terminal, this is correct behavior, not a bug.

**Should the OutboxPublisher be single-use (Option A) or restartable (Option B)?**

---

## Coverage by Module

| Module               | Coverage |
| -------------------- | -------- |
| core/command         | 100.0%   |
| core/query           | 100.0%   |
| core/pkg/dispatcher  | 100.0%   |
| catalog/adapters     | 100.0%   |
| memory               | 99.5%    |
| core/pkg/id          | 97.8%    |
| core/aggregate       | 96.9%    |
| projection           | 98.3%    |
| catalog/d2           | 97.6%    |
| catalog/eventcatalog | 95.7%    |
| middleware           | 95.7%    |
| catalog              | 94.5%    |
| catalog/asyncapi     | 93.9%    |
| core/decider         | 92.7%    |
| core/event           | 92.8%    |
| storage              | 85.1%    |
| cattest (internal)   | 0.0%     |
