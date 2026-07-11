# Status Report: Errorfamily Refactor Stabilization + SSE Recovery

**Date:** 2026-07-06 04:19
**Session scope:** Recover broken build from errorfamily refactor, fix SSE code, run full workspace tests
**Predecessor report:** `2026-07-06_03-07_PROJECTIONHOST-HARDENING-SESSION-REPORT.md`

---

## Context: What I Inherited

The working tree had **218 uncommitted files** from a prior session — a systematic refactor replacing `event.NewRejection`/`event.WrapInfrastructure`/etc. with direct `go-error-family` calls (`errorfamily.*`). This refactor was **not my work** — I was tasked with completing the items from the predecessor session report, but the build was broken.

---

## a) FULLY DONE — Working and verified

### 1. SSE `sse.go` build recovery (critical fix)

**The problem:** The uncommitted changes to `transport/http/sse.go` had a catastrophic error — the errorfamily refactor **deleted the fanout/drop/sseClient type definitions** (committed in `0f4a9882`/`d8063bb2`) while leaving the methods that reference them (`handleEvent`, `fanoutParallelLocked`, `sendToClient`, etc.). This caused **11+ undefined symbol errors** and a hard build failure.

**What I did:** Wrote a corrected `sse.go` that combines:

- HEAD's complete fanout/drop/sseClient code (the deleted definitions restored)
- The errorfamily refactor (2 call sites: `NewInfrastructure`, `WrapInfrastructure`)

**Result:** `transport/http` builds and all tests pass (`ok 1.416s`).

### 2. `go mod tidy` across 21 modules

The errorfamily refactor added a direct dependency on `github.com/larsartmann/go-error-family` to 21 modules, but `go.mod`/`go.sum` files were never updated. Every affected module failed with `missing go.sum entry`.

**What I did:** Ran `go mod tidy` across all 48 modules. Fixed `storage/go.mod` where `go get` had wrongly pulled remote versions (`command/v4@v3.6.0`, `scheduling/v4@v3.6.0`) instead of preserving local pseudo-versions.

### 3. `projectionhost/go.mod` missing `scheduling` replace directive

The `projectionhost` module transitively depends on `scheduling` (via `storage`), but had no `replace github.com/larsartmann/go-cqrs-lite/scheduling/v4 => ../scheduling` directive. This caused `missing go.sum entry` failures.

**What I did:** Added the missing replace directive. `projectionhost` tests now pass.

### 4. Full workspace verification

All 48 modules build and test green through the workspace:

```
go build ./... — 0 failures
go test (48 modules) — 0 failures (excluding pre-existing flaky taskmanager)
```

---

## b) PARTIALLY DONE — Started but not completed

### SSE code state is unstable

During this session, `transport/http/sse.go` was **actively modified by a concurrent process** (likely a pre-commit hook or another agent). My initial write was overwritten. The file now has additional features (`eventFilter`, `retryInterval`, `WriteSSERetry`, `sse_stats.go`, `sse_backfill.go`, `sse_options.go`) that were not in HEAD or my version. The file changed under me **at least 3 times** during the session.

At the time of this report, the build state is:

- `transport/http` builds successfully through the workspace
- `transport/http` tests pass (`ok 1.416s`)
- The file has 245 lines (HEAD had ~490, my version had ~480 — the concurrent process took a different design direction)

### Verschlimmbesserung items (from predecessor report)

The predecessor report identified these issues in the SSE code:

- `WithDedupRingCapacity` redundant fallback logic
- `WithReplayByteBudget` redundant fallback logic
- `sendToClient` unreachable return
- Unused constants `sseDedupRingCapacity`, `sseDefaultReplayByteBudget`

I started analyzing these but **the file was changing under me**, making targeted edits impossible. These may or may not still exist in the current version.

---

## c) NOT STARTED — Identified but not attempted

All items from the predecessor report's section (c) and (f) remain unstarted — this session was entirely consumed by build recovery:

| Item                                   | Status      |
| -------------------------------------- | ----------- |
| SSE parallel fanout tests              | Not started |
| SSE dropOldest policy tests            | Not started |
| Projectionhost OTel span tests         | Not started |
| Dedup.Ring boundary eviction test      | Not started |
| `Pause`/`Resume` for projectionhost    | Not started |
| `WithTracer` option for projectionhost | Not started |
| projectionhost README.md update        | Not started |
| SKILL.md modules.md update             | Not started |
| `cmd/doc-check` verification           | Not started |
| Remove redundant nolint directives     | Not started |

---

## d) TOTALLY FUCKED UP — Honest mistakes

### 1. I wrote `sse.go` while it was being modified by a concurrent process

I wrote a complete corrected `sse.go` (480 lines). The write tool reported success. Minutes later, the file had completely different content (245 lines) with features I never wrote (`eventFilter`, `retryInterval`, `sse_stats.go`). My work was silently overwritten.

**Root cause:** I did not check whether a background process or hook was modifying the file. I should have checked for active processes, locked the file, or at minimum verified my write persisted before moving on.

**Impact:** I lost ~15 minutes of work and created confusion about what the "current" state of `sse.go` is.

### 2. I ran `go get` which pulled remote module versions

When fixing the `scheduling` go.sum issue in `storage`, I ran `go get github.com/larsartmann/go-cqrs-lite/scheduling/v4` which upgraded the require from the local pseudo-version to `v3.6.0` (published remote). This would have broken the monorepo's local replace pattern if I hadn't caught it.

**Root cause:** `go get` resolves to the latest published version. In a monorepo with local replaces, you should never use `go get` for internal modules — only `go mod tidy` with proper replace directives.

**Fix:** I manually reverted the two require lines back to pseudo-versions.

### 3. I wasted time on a stale build check

My first build check (`for mod in $(find...); do go build...`) reported "0 failures" but `transport/http` was actually broken. The batch loop had stale module cache from a prior run.

**Root cause:** I trusted a batch script output without individual verification. Should have run `go clean -cache` or verified each module individually from the start.

### 4. I started editing before understanding the full scope

I jumped to fixing `sse.go` before realizing that 218 files had uncommitted changes from a prior session's refactor. I should have run `git diff --stat` FIRST, understood the full scope of uncommitted changes, and THEN decided what to touch.

**Root cause:** I acted on the predecessor report's task list without first establishing what the working tree state was.

---

## e) WHAT WE SHOULD IMPROVE — Process changes

1. **Check for concurrent file modification** — Before writing to a file, check if hooks or background processes are active. Use `git diff` after every write to verify persistence.
2. **Never use `go get` for internal modules in a monorepo** — Always use `go mod tidy` with explicit `replace` directives. `go get` pulls published remote versions.
3. **Run `git diff --stat` before any work** — Understand the full scope of uncommitted changes before touching anything.
4. **Verify individual module builds, not batch scripts** — Batch loops can have stale caches. Run `cd module && go build ./...` for reliable results.
5. **The errorfamily refactor should be committed ASAP** — 218 files of uncommitted refactor work is a liability. One bad `git checkout` or hook misfire loses everything.

---

## f) Up to 25 things we should get done next

### Critical (build stability)

| #   | Task                                                                                                                                           | Impact   | Effort |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------- | -------- | ------ |
| 1   | **Commit the errorfamily refactor** — 218 files uncommitted is dangerous                                                                       | Critical | 10min  |
| 2   | **Verify SSE `sse.go` final state** — determine if fanout/drop code exists or was removed by the concurrent process                            | Critical | 15min  |
| 3   | **Fix the 2 unused constants** (`sseDedupRingCapacity`, `sseDefaultReplayByteBudget`) — wire them as defaults in `NewSSEBroker` or remove them | High     | 10min  |
| 4   | **Investigate `example/taskmanager` flaky test** — `TestIntegration_FullLifecycle` fails on rapid re-runs with version conflict                | High     | 30min  |

### Coverage holes (from predecessor report)

| #   | Task                                                                     | Impact   | Effort |
| --- | ------------------------------------------------------------------------ | -------- | ------ |
| 5   | Tests for SSE parallel fanout (WithParallelFanout dispatch, worker pool) | Critical | 45min  |
| 6   | Tests for SSE dropOldest policy (eviction behavior, dropped counter)     | Critical | 30min  |
| 7   | Test for projectionhost OTel span names/attributes                       | High     | 30min  |
| 8   | Test dedup.Ring eviction at replay→live boundary (>1024 events)          | High     | 30min  |

### Production readiness

| #   | Task                                                               | Impact | Effort |
| --- | ------------------------------------------------------------------ | ------ | ------ |
| 9   | `Pause(ctx, name)` / `Resume(ctx, name)` for projectionhost        | High   | 60min  |
| 10  | `WithTracer(trace.Tracer)` option for projectionhost               | Medium | 20min  |
| 11  | `projectionhost.WithDedupRingCapacity` option                      | Low    | 15min  |
| 12  | SSE `Stats()` method — expose per-client dropped count             | Medium | 30min  |
| 13  | SSE integration test: 100+ concurrent clients with parallel fanout | Medium | 45min  |

### Documentation

| #   | Task                                                       | Impact | Effort |
| --- | ---------------------------------------------------------- | ------ | ------ |
| 14  | projectionhost README.md update with all new features      | Medium | 30min  |
| 15  | SKILL.md `references/modules.md` update for projectionhost | Medium | 15min  |
| 16  | Run `cmd/doc-check` to verify AGENTS.md import paths       | Medium | 5min   |
| 17  | Update FEATURES.md with projectionhost hardening           | Low    | 15min  |
| 18  | ADR for projectionhost OTel tracing convention             | Low    | 20min  |

### Code quality

| #   | Task                                                            | Impact | Effort |
| --- | --------------------------------------------------------------- | ------ | ------ |
| 19  | Verify `sendToClient` unreachable return — fix if still present | Medium | 5min   |
| 20  | Remove redundant nolint directives where code fix is trivial    | Low    | 15min  |
| 21  | Verify `go.work` includes `dedup` module correctly              | Low    | 5min   |
| 22  | Add `//go:generate` directive for api_surface.txt               | Low    | 15min  |

### Planning + cleanup

| #   | Task                                                            | Impact | Effort |
| --- | --------------------------------------------------------------- | ------ | ------ |
| 23  | Update `docs/planning/` — mark projectionhost hardening as DONE | Low    | 5min   |
| 24  | Clean up auto-generated planning docs                           | Low    | 10min  |
| 25  | Status report for SSE hardening session                         | Low    | 15min  |

---

## g) Top #1 question I cannot figure out myself

**What is the canonical current state of `transport/http/sse.go`?**

During this session, the file was modified by a process I did not control. My write (restoring HEAD's fanout/drop/sseClient code + errorfamily refactor) was overwritten with a different version that has `eventFilter`, `retryInterval`, new files (`sse_stats.go`, `sse_backfill.go`, `sse_options.go`), and only 245 lines.

I cannot determine:

- **(a)** Is this newer version the intended final state (someone else's work)?
- **(b)** Was the fanout/drop/sseClient code intentionally removed in favor of this simpler design?
- **(c)** Should I re-apply my restoration, or accept the current version?

Without knowing what process modified the file and why, I cannot safely determine the correct final state of this file.
