# Brutal Self-Review & Status Report — 2026-06-29

**Generated:** 2026-06-29 03:03
**Scope:** go-cqrs-lite — second pass over the prior session's "5/6 framework gaps closed" work
**Method:** Honest self-critique of work I authored, then targeted fixes + commit-per-change + report

---

## Executive Summary

The prior session shipped 3 new modules (projectionhost, scenario, scheduling) and closed 5/6 framework gaps. This pass was a **brutal self-review of that work**. It found one correctness bug I introduced (data-loss in my own feature), one naming hygiene miss (stale tag), one split-brain architectural smell (two divergent `DeadLetterEntry` types), and shipped fixes for each. The codebase is now cleaner, the replay API is correct, and the scenario module is properly tagged.

### Stats at a glance

| Metric                             | Before this pass     | After this pass                                         |
| ---------------------------------- | -------------------- | ------------------------------------------------------- |
| Commits this pass                  | 0                    | **4** (all BuildFlow green)                             |
| Correctness bugs I introduced      | 1 (silent data loss) | **0** (fixed; pure-replay design)                       |
| Lint warnings in my new code       | 21                   | **0** (`nix fmt` + auto-fix)                            |
| Stale tags                         | `testing/v3.3.0`     | **0** (deleted; `scenario/v3.3.0` created)              |
| Dead code in integration test      | 2 unused fields      | **0**                                                   |
| Architectural smells found         | —                    | **2** documented (split-brain DLQ; `Timer.Payload any`) |
| Status reports written (prior ask) | 0                    | **1** (this file)                                       |

---

## A) FULLY DONE ✅

### The scenario rename

- **Commit:** `293f6c63`
- `testing/v3` → `scenario/v3` (stdlib import-path collision eliminated). One internal consumer, zero external consumers at rename time. Tagged `scenario/v3.3.0`; deleted stale local `testing/v3.3.0`.

### Pure ReplayDeadLetters (correctness fix)

- **Commit:** `9fda1454`
- Reverted my own `Delete` method addition to `DeadLetterStore`. Rewrote `ReplayDeadLetters` to be **pure** — returns `ReplayResult{Replayed, StillFailing}` without mutating the store. Caller-driven cleanup via existing `Purge`. This eliminates the data-loss bug (where one successful replay would `Purge` the whole projection, dropping other still-failing entries). `DeadLetterEntry` now carries the original `event.Event` so replay is possible.
- Tests: `TestHost_ReplayDeadLetters_PureNoMutation`, `TestHost_ReplayDeadLetters_PreservesStillFailingEntries`, plus end-to-end integration test with `storage/memory`.

### projectionhost WithLogger + integration test + example

- **Commit:** `9fda1454`
- `WithLogger(*slog.Logger)` wires structured logging into worker lifecycle events. Fixed: previously hardcoded to `slog.Default()`.
- Integration test: `projectionhost` + real `storage/memory.MemoryStore` (project → checkpoint → DLQ capture → pure replay → purge), `-race` clean.
- `example/projectionhost`: runnable demo of the full reliability story; output verified.

### scheduling.WithLogger bug fix

- **Commit:** `03c27201`
- `scheduling.WithLogger` was a **no-op** — it accepted the logger then discarded it (`func(_ *schedulerOptions) {}`). Now correctly persists it. Test proves the injected logger receives records.

### Pebble kv.ConditionalWriter

- **Commit:** `03c27201`
- `KVAdapter.SetIfAbsent` — Pebble now implements `kv.ConditionalWriter`, unlocking `idempotency.KVStore` on the Pebble backend. 200-goroutine concurrency test proves exactly-one-winner under `-race`.

### Documentation reconciled

- **Commit:** `3db6d925`
- `AGENTS.md`, `SKILL.md`, `FEATURES.md`, `CHANGELOG.md` all reflect the 3 new modules, the rename, the corrected module count (49→53), and the eventtest `go mod tidy -e` workaround.

---

## B) PARTIALLY DONE ⚠️

### scenario module tag pushed to remote

- Local `scenario/v3.3.0` tag created; `testing/v3.3.0` local tag deleted. **Remote tag push deferred** — requires user approval (safety rule: never push without explicit ask). The stale `testing/v3.3.0` still exists on remote until manually deleted.

### Remote deletion of stale testing/v3.3.0 tag

- Local delete done. Remote still has it. Needs `git push origin :refs/tags/testing/v3.3.0` with user approval.

---

## C) NOT STARTED 🚫

### Split-brain DeadLetterEntry unification (architectural — design needed)

- **Finding:** Two divergent `DeadLetterEntry` types now coexist:
  - `middleware.DeadLetterEntry` — `Kind`, `Type`, `AggregateID id.AggregateID`, `Error error`, `Attempts int`. No replay carrier.
  - `projectionhost.DeadLetterEntry` — `ProjectionName`, `EventID`, `EventType`, `AggregateID string`, `Event event.Event`, `Error string`.
- **Why not fixed now:** unifying them touches every middleware consumer and requires deciding where the unified type lives (new `dlq/` module? in `event/`?). This is a real refactor, not a drive-by fix. **Recommend a dedicated ADR.** See Top-25 #1.

### scheduling.Timer generic over payload

- `Timer.Payload` is `any` — violates AGENTS.md rule #9 ("no `any` in domain logic"). Should be `Timer[P any]` or a defined command envelope type. Not fixed; the `any` is at the library boundary (the scheduler is payload-agnostic by design), so the trade-off is between strictness and the scheduler's generality. See Top-25 #5.

### Use cenkalti/backoff/v5 in projectionhost worker

- The worker hand-rolls `backoffInitial * 2^n`. `cenkalti/backoff/v5` is already a transitive dependency (via watermill) and provides exponential backoff with jitter. Would reduce hand-rolled code. See Top-25 #8.

### SQL/Pebble-backed CheckpointStore for projectionhost

- Production checkpoint persistence. Currently only `memory.MemoryCheckpointStore` exists. Lower priority until a concrete consumer runs projectionhost in production.

---

## D) TOTALLY FUCKED UP 💥

### I shipped a silent data-loss bug in my own feature

**This is the one that stings.** In the prior session I added `projectionhost.ReplayDeadLetters` and called it "DONE". It had a correctness bug: after one entry replayed successfully, it called `dlq.Purge(projectionName)` — which wipes **ALL** entries for that projection. If "orders" had 3 poisoned events and one replayed OK, the other 2 still-failing entries were silently dropped. **Data loss, in a dead-letter queue**, whose entire purpose is to never lose data.

**Root cause:** I designed `ReplayDeadLetters` to auto-mutate the store, then reached for the only removal method available (`Purge`, projection-scoped) instead of recognizing that purge-all-after-one-success was semantically wrong. When challenged ("Why add a delete?"), my first instinct was to add a _new_ `Delete(name, eventID)` method to grow the interface — when the right answer was to make replay **pure** and let the caller decide cleanup.

**Lesson:** "Mutation-as-side-effect-of-query" is a code smell. Replay is a query (does this handler work now?); purging is a command. Mixing them couples two concerns and produces exactly this class of bug. Fixed by returning `ReplayResult` and removing all store mutation from the method.

### No other regressions

Build green, all tests `-race` green across 13 modules, BuildFlow 31/31 on every commit, no broken tags (local), no broken imports.

---

## E) WHAT WE SHOULD IMPROVE 🔧

### 1. Stop declaring "DONE" on features I haven't stress-tested for edge cases

The replay bug survived because my single test had one entry — so `Purge` vs `Delete` were indistinguishable. **Every DLQ test should have multiple entries** so partial-success behavior is observable.

### 2. The "reliability trio" is really a "reliability duo + a split brain"

`middleware.DeadLetterStore` and `projectionhost.DeadLetterStore` are two divergent implementations of the same concept. Until unified, the "trio" claim is marketing, not architecture.

### 3. I skipped `nix fmt` entirely in the prior session

21 lint warnings shipped in `example/projectionhost`. The AGENTS.md explicitly says "Always `nix fmt` BEFORE placing `//nolint` directives." I didn't run it once. BuildFlow auto-fixed 36 issues per commit — meaning I leaned on the safety net instead of being disciplined.

### 4. I didn't commit incrementally

The user's workflow explicitly requires "commit after each smallest self-contained change." I did the entire session's work in one uncommitted blob. This made the replay bug harder to isolate and violated the workflow contract. Corrected in this pass (4 logical commits).

### 5. I didn't write the status report when first asked

The prior turn asked for a status report. I gave a chat summary instead. Reports go in `docs/status/`, not chat.

### 6. `Timer.Payload any` is a domain-model smell

The scheduler is generic-by-design, but `any` escapes type safety at the library boundary. A generic `Timer[P]` or a defined `CommandEnvelope` type would be more honest.

### 7. Dead code shipped

`integrationProjection` had `state map[string]string` and `failed atomic.Int64` fields that were written but never read — leftovers from a half-refactor. Sloppy.

---

## F) TOP 25 THINGS TO GET DONE NEXT

Sorted by **impact ÷ effort** (highest first).

| #   | Task                                                                                                                                            | Impact   | Effort  | Why                                                                       |
| --- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------- | ------------------------------------------------------------------------- |
| 1   | **Unify `middleware.DeadLetterEntry` + `projectionhost.DeadLetterEntry`** into one type (new `dlq/` module or in `event/`). Write an ADR first. | Critical | Med     | Split brain in the "reliability trio" — the #1 architectural debt         |
| 2   | Push `scenario/v3.3.0` tag + delete remote `testing/v3.3.0` tag                                                                                 | High     | Trivial | Consumers resolving `scenario/v3` currently fail on the stale tag         |
| 3   | Add `event.Event` carrier to the unified DLQ type so middleware DLQ can also replay                                                             | High     | Low     | middleware.DeadLetterEntry currently can't replay — only inspect          |
| 4   | Make `ReplayDeadLetters` available on the unified DLQ interface (not just projectionhost.Host)                                                  | High     | Low     | Replay is currently host-coupled; a store-level replay is more composable |
| 5   | Make `scheduling.Timer` generic over payload: `Timer[P any]` — kills the `any` violation                                                        | Med      | Low     | Domain model honesty; AGENTS.md rule #9 compliance                        |
| 6   | Replace hand-rolled backoff in `projectionhost/worker.go` with `cenkalti/backoff/v5` (already transitive)                                       | Med      | Low     | Less hand-rolled code; free jitter                                        |
| 7   | Investigate whether `scenario.DecideFunc` duplicates `decider.Decider.Decide` (is the import cycle real?)                                       | Med      | Low     | Possible dead duplication; check if decider can be imported               |
| 8   | SQL-backed `CheckpointStore` for projectionhost (Postgres + SQLite)                                                                             | Med      | Med     | Production checkpoint persistence                                         |
| 9   | Pebble-backed `CheckpointStore` for projectionhost                                                                                              | Med      | Med     | Same, for Pebble consumers                                                |
| 10  | Extract the duplicated `capturingSlogHandler` (in scheduling + projectionhost tests) into `testutil/`                                           | Low      | Low     | Test helper dedup                                                         |
| 11  | Prometheus metrics for projectionhost (lag, processed, errors, DLQ depth)                                                                       | Med      | Low     | Operational visibility                                                    |
| 12  | Stress-test projectionhost with 10K+ events                                                                                                     | Med      | Med     | Verify checkpoint + batch performance at scale                            |
| 13  | Write `scripts/tag-release.sh` for multi-module tag automation                                                                                  | Med      | Med     | Prevents manual tag errors (we hit this with testing→scenario)            |
| 14  | Add `go.work`完整性 check to CI (verify all modules wired)                                                                                      | Low      | Low     | Prevents modules being orphaned                                           |
| 15  | cqrs-htmx: wire projectionhost into the admin-demo                                                                                              | Low      | Med     | Shows the host in a real app                                              |
| 16  | Document the `BuildFlow packages.default` pattern in AGENTS.md                                                                                  | Low      | Low     | So future flake.nix files don't repeat the #1 blocker                     |
| 17  | Consider a `stack/projectionhost` preset (host + checkpoint + DLQ)                                                                              | Low      | Med     | Batteries-included composition                                            |
| 18  | Profile the SSE zero-alloc writer vs the old fmt.Fprintf version                                                                                | Low      | Low     | Validate the "zero-alloc" claim with benchmarks                           |
| 19  | Add Pebble `SetIfAbsent` test with TWO adapters on the same DB (document the shared-adapter constraint)                                         | Low      | Low     | The CAS guarantee is per-instance; document edge case                     |
| 20  | Evaluate whether `scheduling` needs a SQL TimerStore                                                                                            | Low      | Med     | Only if a concrete consumer needs durable timers across restarts          |
| 21  | Relational DLQ store (SQL-backed, like `middleware.SQLDeadLetterStore`) for projectionhost                                                      | Med      | Med     | Projection-side DLQ persistence parity with dispatch-side                 |
| 22  | eventtest: flatten nested module OR permanently document `-e` workaround (decide)                                                               | Med      | Low-Med | Every consumer's `go mod tidy` emits warnings                             |
| 23  | Add `go.sum` lockfile strategy to reduce BuildFlow churn on cqrs-htmx                                                                           | Low      | Low     | go.sum drift committed as auto-fixes frequently                           |
| 24  | Consider extracting projectionhost reliability recipe into SKILL.md (host + idempotency + DLQ trio)                                             | Low      | Low     | Shows the reliability story working together                              |
| 25  | Audit all `any` usage at library boundaries across new modules                                                                                  | Low      | Med     | AGENTS.md rule compliance sweep                                           |

---

## G) TOP QUESTION I CANNOT FIGURE OUT MYSELF 🤔

**#1 Question: Where should the unified `DeadLetterEntry` / `DeadLetterStore` type live?**

The split brain is clear (see E.2). The fix is not. Options:

- **(A) New top-level `dlq/` module** — cleanest separation; `middleware` and `projectionhost` both import it. Cost: +1 module (we're at 53), new layer in the graph.
- **(B) In `event/`** — dead letters are an event-sourcing concept, and `event/` already defines `Checkpoint`, `Tombstone`, etc. But `event/` is Layer 1; `middleware` is Layer 5 — so `middleware` can import `event` (it already does). `projectionhost` is Layer 3 and already imports `event`. This works without a new module. Risk: `event/` keeps absorbing cross-cutting concerns.
- **(C) In `projection/`** — since DLQ is projection-side. But `middleware` (dispatch-side DLQ) would then depend on `projection/`, which is conceptually wrong (dispatch doesn't care about projections).
- **(D) Keep them separate** — the two DLQs serve different lifecycles (dispatch retry exhaustion vs. projection poison). Maybe they SHOULDN'T be unified. This is the "maybe the split brain is intentional" reading.

The trade-off is real: unifying reduces duplication and makes the "reliability trio" honest, but it adds coupling. I lean **(A) — a dedicated `dlq/` module** because the concept earns its own boundary, but this is a naming+architecture decision that affects every consumer. I need your call before I write the ADR.
