# Status Report: Feedback Review & TOCTOU Race Fix

**Date:** 2026-08-13 01:49
**Session scope:** Review ALL `docs/feedback/new/**`, triage, fix, verify
**Commit:** `d60d72ed4` — fix(projectionhost): close catch-up TOCTOU race

---

## a) FULLY DONE

### Feedback Triage (6/6 documents reviewed)

All 6 feedback documents in `docs/feedback/new/` were read in full, verified against source code, and triaged:

| Document | Verdict | Review Doc |
| --- | --- | --- |
| KeyHolderAI cqrs-lint (08-05) | Already fully fixed (prior session) | ✅ Existed |
| cqrs-htmx cqrs-lint r2 (08-04) | Already fully fixed (prior session) | ✅ Existed |
| DiscordSync cqrs-lint r2 (08-04) | All issues already fixed in codebase | ✅ Created |
| DiscordSync read/write census (08-08) | Decisions confirmed; docs completed | ✅ Created |
| file-renamer circuitbreaker/dlq (08-09) | No modules needed; FAQ exists | ✅ Created |
| file-renamer TOCTOU race (08-13) | **Real bug — fixed this session** | ✅ Created |

### TOCTOU Race Fix (projectionhost)

**Root cause:** Events published between journal drain and `SubscribeAll` registration were permanently lost with non-blocking subscribers (e.g., `system/v4`'s in-process bus).

**Fix implemented (Option A from feedback):**
- `processLive` now calls `SubscribeAll` first, then runs a catch-up drain
- `drainCatchUp` reads events from the journal position where the initial drain ended
- `handleMu` mutex serializes event processing between catch-up drain and live handler callback
- `liveHandler` extracted as a separate method for clarity
- `WorkerLive` moved to after catch-up drain completion

**Tests added:**
- `TestHost_CatchUpDrain_PicksUpEventsPublishedDuringSubscribeWindow` — deterministic race regression test
- `TestHost_CatchUpDrain_LiveDeliveryWorksAfterCatchUp` — verifies live delivery works post-catch-up
- Both pass 30 iterations under `-race`

### Documentation

- Created `storage/view/README.md` (182 lines) — AutoMapper as recommended default, manual ViewMapper as escape hatch, all constructors/options/CRUD/query/pagination/tombstone/batch/transactions/indexes documented
- Created 4 review documents in `docs/feedback/reviewed/`

### Verification

- `projectionhost` — builds, vets, all tests pass with `-race`
- `watermill` — builds, all tests pass with `-race`
- `system/`, `stack/`, `integration/` — all build clean (workspace mode)
- Dependent modules unaffected

---

## b) PARTIALLY DONE

### TOCTOU Race Fix — WORKING BUT HAS A REGRESSION

**THE FIX HAS A BEHAVIORAL REGRESSION FOR BLOCKING SUBSCRIBERS.**

`WorkerLive` is now set AFTER `SubscribeAll` returns. For blocking subscribers (NATS, Postgres LISTEN/NOTIFY, Watermill GoChannel), `SubscribeAll` blocks until context cancellation. This means:

1. `SubscribeAll(handler)` — blocks for entire lifetime of the projection
2. Context cancelled (shutdown)
3. `SubscribeAll` returns
4. `drainCatchUp` — returns immediately (ctx cancelled)
5. `setStatus(WorkerLive)` — set for a brief moment
6. Return nil → worker exits → `WorkerStopped`

**Result:** Consumers polling `host.Status()` for `WorkerLive` will NEVER see it during operation for blocking subscribers. The original code set `WorkerLive` BEFORE `SubscribeAll`, so it was visible throughout the live phase.

**The fix:** Move `w.setStatus(WorkerLive)` BEFORE `SubscribeAll`. The catch-up drain closes the race regardless of status ordering. Setting WorkerLive before SubscribeAll is honest because:
- For non-blocking subscribers: callback is about to be registered, catch-up drain will close any gap
- For blocking subscribers: the callback IS being registered (SubscribeAll is blocking while delivering events via callback)

**Severity:** MEDIUM-HIGH. Any consumer using blocking subscribers and polling for WorkerLive readiness will break.

### watermill.CatchUpSubscriber — SAME RACE, NOT FIXED

`watermill/catchup_subscriber.go` has a structurally identical race between `replayPhase` (journal drain) and `livePhase` (live subscription). The fix requires a different approach due to the channel-based architecture. Documented as "known issue" in the review doc but not fixed.

### drainCatchUp Code Duplication

The `drainCatchUp` method (~60 lines) is nearly identical to the drain loop in `process()`. Same batch-read-loop structure, same event processing, same checkpoint saving. The only differences are: handleMu wrapping, context cancellation returning nil instead of error. This should be refactored to a shared helper to reduce maintenance burden.

---

## c) NOT STARTED

### From DiscordSync Read/Write Census (Strategic — explicitly deferred)

1. **DuckDB real aggregation pushdown** — APPROVED but high effort. `CounterGet` loads all rows into Go maps instead of pushing GROUP BY to columnar SQL. The DuckDB engine is "marketing, not implementation" for analytics.
2. **Cross-projection JOIN** — DEFERRED to separate ADR. Metaengine stays single-collection.
3. **`relational → metaengine` migration guide** — Not started.

### From DiscordSync cqrs-lint r2 (Wishlist items)

4. **`--doctor --fix` flag** — auto-write detected features into `.cqrs-lint.json`
5. **Stale-suppression detection as default** (not `--strict`-only)
6. **Show config-disabled rules in health breakdown**
7. **Feature-profile-aware C008** — `monetary: false` → auto-INFO for float64 fields

### From cqrs-htmx cqrs-lint r2 (Wishlist items)

8. **`examples/` exclusion or `demo` preset** — not implemented (library preset + per-module profiles cover most cases)
9. **Per-module evaluation of every global detector** — would require restructuring ~15 detectors

---

## d) TOTALLY FUCKED UP

### WorkerLive Regression for Blocking Subscribers

**This is the big one.** I moved `setStatus(WorkerLive)` to after `SubscribeAll` + `drainCatchUp`. For blocking subscribers, `SubscribeAll` blocks until shutdown, meaning `WorkerLive` is never visible during operation. The original code correctly set it before `SubscribeAll`. I should have kept it there — the catch-up drain closes the race independently of status ordering.

**Impact:** Any consumer using a blocking subscriber (Watermill, NATS, etc.) and polling `host.Status()` for `WorkerLive` will never see it. They'll see `WorkerRunning` during the entire live phase instead.

**Fix needed:** Move `w.setStatus(WorkerLive)` to before `w.opts.subscriber.SubscribeAll(...)` in `processLive`.

### Context Cancellation Inconsistency

`process()` returns `fmt.Errorf("drain cancelled: %w", ctx.Err())` on context cancellation. `drainCatchUp` returns `nil`. This is cosmetically inconsistent but behaviorally equivalent (both result in clean worker exit). Still sloppy.

### Didn't Format

I didn't run `nix fmt` or `gofumpt` on my changes before the auto-commit daemon grabbed them. The code may have formatting issues that `nix run .#lint` would catch.

---

## e) WHAT WE SHOULD IMPROVE

### Process Improvements

1. **Test with BOTH subscriber types** — My tests only cover non-blocking subscribers. I need a test with a blocking subscriber (like `channelSubscriber` which blocks on channel receive) to verify the WorkerLive status is visible during operation.

2. **Verify against the consumer's actual reproduction** — The feedback describes `TestHistoryAdapter_StatsAndEntries` in file-and-image-renamer failing 40-60% under `-race`. I didn't reproduce this or verify the fix against it.

3. **Run `nix fmt` before the auto-commit daemon grabs changes** — The daemon commits fast; format first.

4. **Check status visibility for ALL subscriber types when changing processLive ordering** — I changed the lifecycle ordering without fully tracing the impact on blocking subscribers.

5. **Factor out drain loop duplication** — `process()` and `drainCatchUp()` share 90% of their logic. Extract a shared `drainBatch` helper.

6. **Fix the watermill CatchUpSubscriber race in the same session** — Same class of bug, different architecture. Leaving it half-fixed is a reliability gap.

### Documentation Improvements

7. **Fix 3 pre-existing doc-check failures** — `advanced.md:23` references `listing.StatusActive`/`listing.StatusDeleted` (not found), `readmodels.md:137` references `stack.ExcludeDeleted` (not found). These existed before my session but should be fixed.

8. **Update projectionhost docs** — The `processLive` doc comment describes the new behavior but the host.go `Start()` doc comment still says "batch-drainer with crash-restart semantics, not a live stream consumer" which is now misleading when `WithSubscriber` is used.

---

## f) Up to 50 Things to Get Done Next

### Critical (fix regressions from this session)

1. **Move `setStatus(WorkerLive)` before `SubscribeAll`** — fix the blocking subscriber regression
2. **Add a test with a blocking subscriber verifying WorkerLive is visible during operation**
3. **Run `nix fmt` / `gofumpt` on all changed files**
4. **Verify `drainCatchUp` context cancellation behavior matches `process()`**

### High Priority (same bug class)

5. **Fix watermill CatchUpSubscriber TOCTOU race** — same drain→subscribe gap, channel-based architecture
6. **Add catch-up drain test for blocking subscriber type** — verify no regression
7. **Refactor `drainCatchUp` and `process()` to share drain loop logic** — eliminate ~60 lines of duplication

### Medium Priority (from feedback, explicitly approved)

8. **Implement DuckDB `AggregateReader`** — push GROUP BY/SUM/AVG to columnar SQL instead of loading rows into Go
9. **Fix `CounterGet` in DuckDB engine** — currently loads all rows into Go map (`duckdbengine/engine.go:312-335`)
10. **Design metaengine cross-projection JOIN ADR** — the strategic frontier

### Low Priority (wishlist from feedback)

11. **`--doctor --fix` flag** for cqrs-lint — auto-write detected features to `.cqrs-lint.json`
12. **Make stale-suppression detection default** in cqrs-lint (not `--strict`-only)
13. **Show config-disabled rules** in cqrs-lint health score breakdown
14. **Feature-profile-aware C008** — auto-downgrade float64 for non-monetary projects
15. **`examples/` exclusion or `demo` preset** for cqrs-lint
16. **Per-module evaluation of every global cqrs-lint detector** — restructure ~15 detectors

### Documentation / Cleanup

17. **Fix 3 pre-existing doc-check failures** (advanced.md, readmodels.md)
18. **Update projectionhost host.go Start() doc comment** — mentions "not a live stream consumer" which is misleading with WithSubscriber
19. **Document the catch-up drain pattern in SKILL.md recipes** — so consumers understand the race and how it's handled
20. **Move reviewed feedback docs out of `new/`** — they're committed but still in `new/` directory
21. **Add `WithoutViewAutoMigrate` mention to SKILL.md recipes** — currently only in README and source
22. **Document `Increment` non-clamping philosophy in SKILL.md FAQ** — currently only in sink.go doc comment

### Verification / Hardening

23. **Run `nix run .#verify`** — full build + vet + test + race + lint + doc-check
24. **Run `nix run .#check-arch`** — dependency budget enforcement
25. **Run `nix run .#check-duplication`** — no-new-clones gate (drainCatchUp duplication may trigger this)
26. **Run `nix run .#check-coverage`** — coverage drift check
27. **Verify API stability golden** — run `cd cmd/api-stability && GOWORK=off go run main.go -update` if any exported symbols changed (shouldn't have)
28. **Test the fix against file-and-image-renamer's reproduction** — clone the repo and run their test suite

### Strategic (future sessions)

29. **Write `relational → metaengine` migration guide**
30. **Research DuckDB vectorized aggregation paths**
31. **Design metaengine cross-projection query planning ADR**
32. **FTS5 integration for metaengine SearchBackend** — only Memory + Dgraph implement it
33. **Date/time function pushdown in metaengine** — materialize computed columns at projection time

---

## g) Questions I CANNOT Answer Myself

### 1. Should I fix the WorkerLive regression immediately (move it before SubscribeAll), or is there a reason the status should only be set after catch-up drain completes?

The feedback explicitly called out that setting WorkerLive before SubscribeAll was misleading ("status says 'live' but not ready"). But for blocking subscribers, setting it after means it's NEVER visible during operation. The right answer depends on whether any consumer actually polls for WorkerLive, and whether "live but still catching up" is an acceptable state.

### 2. Should the watermill CatchUpSubscriber race be fixed in this same session/PR, or is it a separate work item?

The CatchUpSubscriber uses a fundamentally different architecture (channel-based, two-phase with replayIDs dedup). Fixing it requires a different approach (possibly subscribing before replaying, or adding a post-subscribe re-replay). Should this block the projectionhost fix, or ship independently?

### 3. The `system/v4` module has pre-existing build errors (`metaengine.Priority`, `metaengine.NamedSample` undefined in standalone build). Should I fix these, or are they known/expected?

These errors prevent `cd system && GOWORK=off go build` from succeeding, but workspace mode (`go build ./system/...`) works fine. The consumer (file-and-image-renamer) imports `system/v4`, so these errors may affect them. I can't tell if this is a known issue from a recent refactor or an actual breakage.
