# PostgresBus Production Hardening — Execution Plan

> **Date:** 2026-06-19
> **Status:** ✅ EXECUTED AND VERIFIED
> **Commit:** `79d1f6a6`
> **Quality Gate:** lint 0/34 · test 50/50 · layers PASS · API 1852

---

## Context

The prior session wired `storage.PostgresBus` into `stack/postgres` via
`WithDistributedBus(PgxListener)` and fixed a critical `Close()` deadlock.
A brutal self-review identified remaining risks: no deadlock regression
test, no auto-reconnect, dishonest ROADMAP, and missing documentation.

This plan addresses ALL actionable items from the self-review.

---

## Pareto Breakdown

### 1% → 51% of value (quick wins, critical trust)

| # | Task | Impact | Effort | Status |
|---|------|--------|--------|--------|
| T1 | ROADMAP: mark 4 stale `[ ]` items as done (schema validator, prometheus, tracing, logging) | 🔴 HIGH (dishonest docs) | 5m | ✅ |
| T2 | `TestPgxListener_CloseDoesNotDeadlock` regression test with explicit 2s timeout | 🔴 HIGH (prevents critical bug) | 10m | ✅ |

### 4% → 64% of value (production reliability)

| # | Task | Impact | Effort | Status |
|---|------|--------|--------|--------|
| T4+T5 | PgxListener auto-reconnect: `reconnectConfig` + `WithReconnect`/`WithReconnectBackoff`/`WithoutReconnect` options; refactor `receiveLoop` → `receiveOnce` + reconnect loop with exponential backoff | 🔴 HIGH (silent delivery kill) | 25m | ✅ |
| T6 | Reconnect unit tests: backoff calculation (table + rapid property), config options | 🟠 MED | 12m | ✅ |

### 20% → 80% of value (robustness + documentation)

| # | Task | Impact | Effort | Status |
|---|------|--------|--------|--------|
| T3 | Property-based test for `validateChannelName` (3 properties × 100 inputs via rapid) | 🟡 MED | 12m | ✅ |
| T7 | Document graceful drain behavior on `PgxListener.Close()` | 🟡 MED | 8m | ✅ |
| T8 | Document PostgresBus backpressure strategy (channel fullness → server-side queue) | 🟡 MED | 8m | ✅ |

---

## Deferred / Blocked (not actionable now)

| Item | Why Deferred | Effort |
|------|-------------|--------|
| gRPC transport adapter | ADR-0025 accepted. Separate module. Large scope. | 120m |
| NATS/Redis Stream adapter | ADR-0025 accepted. Separate modules. | 90m each |
| jsonv2 codec experiment | Behind build tag. Blocked on Go stdlib. | Blocked |
| Arena allocation experiment | Behind build tag. Blocked on Go stdlib. | Blocked |
| Pebble secondary index (LoadByEventID) | Needs schema migration. Full scan fallback works. | 60m |
| Outbox pattern | Explicitly REMOVED per ROADMAP. Use Watermill. | N/A |
| v3 breaking changes | Deferred to next major version. | 300m |

---

## Execution Graph

```
T1 (ROADMAP freshness) ────────────────────────┐
T2 (deadlock regression test) ─────────────────┤
T3 (property-based channel name test) ─────────┤
T4+T5 (auto-reconnect implementation) ─────────┤── T9 (lint+test+API) ── T10 (commit+push)
T6 (reconnect unit tests) ─────────────────────┤
T7 (graceful drain docs) ──────────────────────┤
T8 (backpressure docs) ────────────────────────┘
```

---

## What Changed

### New Code

- **`stack/postgres/pg_listener.go`** — `reconnectConfig`, `reconnect()`, `receiveOnce()`, `backoffDuration()`, `WithReconnect()`, `WithReconnectBackoff()`, `WithoutReconnect()`
- **`stack/postgres/pg_listener_test.go`** — `TestPgxListener_CloseDoesNotDeadlock`, `TestValidateChannelName_Property` (3 properties), `TestPgxListener_ReconnectConfig` (5 subtests), `TestBackoffDuration` (8 cases), `TestBackoffDuration_Property`

### Modified

- **`storage/pg_bus.go`** — PostgresBus type doc (backpressure paragraph), PgxListener.Close doc (graceful drain)
- **`ROADMAP.md`** — 4 stale `[ ]` items marked `[x]` with evidence
- **`CHANGELOG.md`** — Auto-reconnect, deadlock test, property tests documented
- **`docs/api_surface.txt`** — 1849 → 1852 exports (3 reconnect options)

### Key Design Decisions

1. **Auto-reconnect enabled by default** — A library that silently stops on a network blip is dangerous. Consumers can disable via `WithoutReconnect()`.
2. **Exponential backoff with cap** — 1s → 2s → 4s → … → 30s. Total worst case: ~3 minutes of attempts before giving up.
3. **Context-aware reconnect** — Close() cancels the context, which unblocks backoff waits and pool.Acquire. No deadlock between reconnect and shutdown.
4. **Property-based tests use rapid** — Already a test dependency in event/encryption/decider/id/query modules. Excluded from dep budget per AGENTS.md.
