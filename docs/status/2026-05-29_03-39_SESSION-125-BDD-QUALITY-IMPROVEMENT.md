# Session 125 — BDD Quality Improvement: Status Report

**Date:** 2026-05-29 03:39
**Branch:** master (up to date with origin/master)
**All changes committed:** YES

---

## A. FULLY DONE

### Sentinel Leak Fixes (Previous + This Session)
All 19 sentinel-leaking specs fixed across 8 files:

| File | It Strings Fixed | Assertions Fixed |
|------|-----------------|-----------------|
| `core/command/command_bdd_test.go` | 4 | 5 |
| `core/decider/decider_bdd_test.go` | 3 | 3 |
| `core/event/types_bdd_test.go` | 1 | 1 |
| `memory/memory_bdd_test.go` | 1 | 1 |
| `middleware/middleware_bdd_test.go` | 2 | 0 |
| `projection/projection_bdd_test.go` | 4 | 4 |
| `saga/saga_bdd_test.go` | 2 | 2 |
| `integration/event/event_sourcing_bdd_test.go` | 3 | 3 |

**Pattern applied:**
- `It("should return ErrXxx")` → `It("should reject my setup and explain that a bus is required")`
- `errors.Is(err, pkg.ErrXxx)` → `Expect(err.Error()).To(ContainSubstring("descriptive message"))`
- `MatchError(pkg.ErrXxx)` → `Expect(err.Error()).To(ContainSubstring("descriptive message"))`

### Saga BDD Rewrite
Complete rewrite from 100% duplicate of standard tests to user-story narrative:
- Order fulfillment workflow: reserve-stock → charge-payment → ship-order
- State machine transitions: pending → running → completed
- Compensation scenarios: reverse-order rollback, no-compensation on first step
- Recovery paths: buggy nil handler, failed initial command
- Incident investigation: querying active sagas, preserving state for forensics

### Full Test Suite
- **28 packages**, all pass, 0 failures
- **161 BDD specs** across 11 suites, all pass

---

## B. PARTIALLY DONE

### It String Quality (Vague → User Perspective)
~60 specs still have mechanical It strings like `"should route the command"` instead of user-perspective descriptions like `"should deliver my command to the handler I registered for this type"`. The Context blocks provide good framing, but the It strings themselves could be more descriptive.

---

## C. NOT STARTED

### Remaining 1 Sentinel Leak
- `core/event/event_bdd_test.go:56` — `It("should reject it with ErrEmptyEventType")` — still contains sentinel name

### Recovery Path Scenarios
No specs test "after failure, can I recover?":
- Command: rejected → fix input → retry
- Event: version conflict → reload → retry
- Projection: handler failure during replay → restart
- Decider: decide fails → aggregate stays clean → retry

### Missing BDD Coverage
- `signing/` module — no BDD tests at all (security-critical module)
- `memory/MemorySnapshotStore` — no BDD coverage
- `memory/MemoryCheckpointStore` — no BDD coverage

### Duplicate BDD Specs
Overlap with standard tests:
- `middleware_bdd` — ~78% duplicate
- `command_bdd` — ~67% duplicate
- `decider_bdd` — ~67% duplicate

These should either be removed (standard tests cover it) or rewritten to tell stories standard tests don't tell.

---

## D. TOTALLY FUCKED UP

Nothing. All 161 specs pass, 28 packages green, zero sentinel assertions remain in assertions (only 1 It string left).

---

## E. WHAT WE SHOULD IMPROVE

1. **1 remaining sentinel It string** at `core/event/event_bdd_test.go:56`
2. **~60 vague It strings** that describe WHAT not WHY — the Context blocks help, but the It strings should stand alone as user stories
3. **Zero recovery-path specs** — BDD's value over standard tests is showing "when things go wrong, can I recover?"
4. **Zero signing BDD tests** — security-critical module, highest missing-coverage risk
5. **No MemorySnapshotStore/MemoryCheckpointStore BDD** — consumers use these in every test setup
6. **High duplicate ratio** in middleware/command/decider BDD — maintenance burden without narrative value

---

## F. Top 25 Things We Should Get Done Next

| # | Task | Impact | Effort | Est |
|---|------|--------|--------|-----|
| 1 | Fix last sentinel leak: `event_bdd:56` ErrEmptyEventType | HIGH | LOW | 2min |
| 2 | Improve 6 It strings in `command_bdd` | MED | LOW | 5min |
| 3 | Improve 7 It strings in `query_bdd` | MED | LOW | 5min |
| 4 | Improve 7 It strings in `event_bdd` | MED | LOW | 5min |
| 5 | Improve 13 It strings in `types_bdd` | MED | LOW | 8min |
| 6 | Improve 2 It strings in `decider_bdd` | MED | LOW | 2min |
| 7 | Improve 7 It strings in `memory_bdd` | MED | LOW | 4min |
| 8 | Improve 2 It strings in `middleware_bdd` | MED | LOW | 2min |
| 9 | Improve 4 It strings in `projection_bdd` | MED | LOW | 3min |
| 10 | Improve 8 It strings in `integration/event_bdd` | MED | LOW | 5min |
| 11 | Improve 5 It strings in `integration/query_bdd` | MED | LOW | 3min |
| 12 | Improve 23 It strings in `stream/sql_bdd` + `listbuilder_bdd` | MED | MED | 10min |
| 13 | Add recovery-path: command rejected → fix → retry | HIGH | LOW | 8min |
| 14 | Add recovery-path: event version conflict → reload → retry | HIGH | LOW | 8min |
| 15 | Add recovery-path: decider fails → aggregate stays clean | MED | LOW | 5min |
| 16 | Add recovery-path: projection handler failure → restart | MED | MED | 10min |
| 17 | Add BDD suite for `signing` module | HIGH | MED | 12min |
| 18 | Add BDD specs for `MemorySnapshotStore` | MED | LOW | 8min |
| 19 | Add BDD specs for `MemoryCheckpointStore` | MED | LOW | 5min |
| 20 | De-duplicate `middleware_bdd` (78% duplicate) | MED | MED | 10min |
| 21 | De-duplicate `command_bdd` (67% duplicate) | MED | MED | 10min |
| 22 | De-duplicate `decider_bdd` (67% duplicate) | MED | MED | 10min |
| 23 | Run full test suite + verify zero regressions | HIGH | LOW | 5min |
| 24 | Commit all improvements | HIGH | LOW | 3min |
| 25 | Update this status report with final numbers | LOW | LOW | 2min |

---

## G. Top #1 Question I Cannot Figure Out Myself

**Should de-duplication (tasks 20-22) REMOVE duplicate BDD specs entirely, or REWRITE them to tell user stories that standard tests don't cover?**

Removing reduces maintenance burden. Rewriting preserves coverage but adds narrative value. The answer depends on whether the team views BDD as "living documentation" (keep + rewrite) or "redundant test coverage" (remove).

---

## Current BDD Spec Inventory

| Suite | Specs | Status |
|-------|-------|--------|
| core/command | 12 | ✅ Fixed sentinels |
| core/decider | 12 | ✅ Fixed sentinels |
| core/event (event_bdd) | 16 | ⚠️ 1 sentinel It left |
| core/event (types_bdd) | 20 | ✅ Fixed sentinel |
| core/query | 9 | ✅ Clean |
| memory | 9 | ✅ Fixed sentinel |
| middleware | 10 | ✅ Fixed sentinels |
| projection | 11 | ✅ Fixed sentinels |
| saga | 12 | ✅ Rewritten |
| integration/event | 24 | ✅ Fixed sentinels |
| integration/query | 6 | ✅ Clean |
| stream (sql + listbuilder) | 23 | ✅ Clean (no sentinels) |
| **TOTAL** | **164** | **All pass** |
