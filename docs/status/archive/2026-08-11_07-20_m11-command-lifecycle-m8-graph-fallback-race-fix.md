# Status Report: M11 Command Lifecycle, M8 Graph Fallback, M15 Lint, M18 PG Isolation, Race Fix

> **ARCHIVED 2026-08-11 — All work in this report is complete. Open items were resolved by later sessions, captured in TODO_LIST.md, or determined to be minor polish. Original content retained below for historical context.**

**Date:** 2026-08-11 07:20
**Session start:** ~06:50 (resumed from prior session handoff)
**Branch:** master
**Head commit:** `0e8f7ce56` (plus auto-daemon commits)

---

## a) FULLY DONE

### 1. M11: Command Lifecycle as Event Streams (ADR-0117)

**Module:** `commandlifecycle/` + `commandlifecycle/projections/`

Implemented the full command lifecycle tracking module per ADR-0117. Commands
are immutable intents with no status field. Their lifecycle — received, failed,
retried, dead-lettered, completed — is tracked via events appended to a
per-command lifecycle stream.

**Files:**
- `events.go` — 5 lifecycle event types + payloads + stream ref
- `recorder.go` — Recorder with best-effort/strict modes, version tracking
- `middleware.go` — `New(recorder)` returns outer+attempt middleware with shared attempt tracker; standalone `Middleware()` and `AttemptMiddleware()`
- `recorder_test.go` — 19 tests (happy path, failure, retry exhaustion, strict mode, causation links, full retry scenarios)
- `projections/projections.go` — DeadLetterQueue, RetryCount, FailureLog projection declarations for metaengine
- `projections/projections_test.go` — Declaration construction tests

**Registration:** go.work ✓, flake.nix testModules ✓, api-stability ✓ (4050 exports), cqrs-lint catalog ✓

### 2. M8: Graph Fallback for Non-Graph Engines (partial)

**Commit:** `dbf27ee78` (graph fallback) + engine profile updates

Implemented brute-force graph traversal fallback in the Store so that engines
without native graphBackend support (SQLite, MySQL, Pebble, bbolt) can still
serve graph queries via MultimapBackend (O(N) BFS instead of O(degree^depth)).

**Files:**
- `metaengine/graph_fallback.go` — `graphAddEdgeFallback` + `graphNeighborsFallback` (BFS with cycle detection)
- `metaengine/graph_fallback_test.go` — 4 tests (basic traversal, depth-limited, cycle safety, depth-0)
- `metaengine/store.go` — applyFoldEdge falls back to multimap when graphBackend unavailable
- `metaengine/execute.go` — ReadTraversal falls back to BFS when graphBackend unavailable
- `metaengine/engine.go` — Added ADTGraph (degraded, O(N)) to SQLiteEngineProfile
- `metaengine/mysqlengine/engine.go` — Added ADTGraph (degraded, O(N)) to MySQL profile

**Impact:** "If I only give you SQLite, metaengine should deal with all query projections via SQLite. If there are graph queries, it should warn about them being slow." — This invariant is now satisfied for graph queries.

### 3. Data Race Fix: SetCurrentRecord + invoke Must Be Atomic

**Commit:** `0e8f7ce56`

**Root cause:** `Store.applyWithRecord` called `fold.SetCurrentRecord(rec)` in one
loop and `fold.invoke()` in a separate loop. The fold's `recHolder` is shared
mutable state. Two concurrent `Apply` calls could interleave, causing goroutine
A to see goroutine B's record — both a data race AND a correctness bug.

**Fix:** Added `foldMu sync.Mutex` to Store. The SetCurrentRecord + applyFold
pair is now atomic. Verified with `-race -count=1` — zero races detected.

### 4. M15: Lint Exclusion Audit

- **id/actor_id.go** — Fixed `goconst` (extracted kind string constants) and `modernize` (replaced `strings.IndexByte` with `strings.Cut`). Narrowed id/ exclusion from 9 linters to 7 (dropped goconst, modernize).
- **flightrecorder/alias.go** — Documented as permanent exclusion (deprecated re-export module, will be deleted in v5)
- **mysqlengine sqlclosecheck** — Documented as permanent false positive (linter can't track DeferClose indirection; pgengine uses same pattern without exclusion but that's because pgengine's code structure happens to pass)

### 5. ReifyReflect Regression Test

**File:** `metaengine/reify_regression_test.go`

5 tests directly targeting the OnRecord update fold reification path that was
fixed in the prior session (`record_fold.go:115`). The key test
`TestOnRecord_UpdateFold_ReifyMapPrev` creates an OnRecordTyped update fold,
feeds it a `map[string]any` prev value (simulating SQL engine return), and
asserts no panic + correct reification. If someone reverts the `reifyReflect`
fix, this test will panic.

### 6. M18: Per-Test PG Isolation for External DSN

**File:** `testutil/pgtestcontainer/pgtestcontainer.go`

**Root cause:** When using `DATABASE_URL`/`POSTGRES_TEST_DSN` (nix CI path),
all tests shared one database — cross-test interference under `-race`.

**Fix:** `TestMain` now opens an admin connection even with external DSN.
`DSN()` creates per-test databases via `CREATE DATABASE` regardless of DSN
source. The only fallback to shared DSN is when `adminDB == nil` (testcontainer
failed to start).

---

## b) PARTIALLY DONE

### M8 remaining items:
- **StreamLog fallback on Dgraph** — Dgraph doesn't implement StreamLogBackend; needs append-ordered node fallback
- **Recursive CTE optimization on PG** — PG already declares ADTGraph support; could upgrade from brute-force BFS to native recursive CTE for O(degree^depth) instead of O(N)
- **Vector search on memory engine** — Already supported via MemoryVectorIndex; no change needed
- **Benchmarks comparing native vs fallback graph traversal** — Not yet done

---

## c) NOT STARTED

- **M9** — Struct-composition multi-collection (`[]Attachment` → secondary collection)
- **M13** — Calibration benchmarks vs baseline + CI regression check
- **M20** — Tombstone vocab rename (NEEDS USER DECISION — breaking change)
- **M21-M26** — v5 preparation (gated on M9 completion)
- **M27** — Nix apps + infra polish

---

## d) Verify Gate Status

**Full `nix run .#verify`:**
- All 82+ modules pass EXCEPT `cmd/cqrs-bench` (pre-existing version-sequence issue)
- **cmd/cqrs-bench failure:** `benchkit.Truncate`/`TitleCase` were added after `benchkit/v4.3.0` tag. Under `GOWORK=off` (CI per-module testing), the published tag doesn't have them. Fix: tag `benchkit/v4.4.0` and bump `cmd/cqrs-bench` dependency. Can't tag without push access.
- **With workspace** (`go test ./...` without GOWORK=off): cmd/cqrs-bench passes ✓
- **Metaengine race tests:** Pass with `-race -count=1` ✓

---

## e) Key Decisions Made

1. **Graph fallback via MultimapBackend** — Chose multimap (MultiAdd/MultiGet) over map (MapSet/MapGet) because multimap is the natural fit for adjacency lists and more engines implement it.
2. **foldMu serialization scope** — The mutex serializes ALL fold execution across concurrent Apply calls. This is acceptable because fold operations are fast (closure call + backend write). Per-fold mutexes would be more granular but require modifying all fold struct types.
3. **M20 (tombstone rename) deferred to v5** — Breaking change with large blast radius. Aliases would add complexity without clear benefit. Better to do clean rename in v5.
4. **commandlifecycle/projections as separate module** — Follows the multi-module pattern: projections import metaengine which is a heavier dependency than the core commandlifecycle module. Consumers who only need the Recorder don't need metaengine.

---

## f) Commands That Worked

- `cd commandlifecycle && GOWORK=off go test -tags "goexperiment.jsonv2" -count=1 -v ./...` — 19 tests pass
- `cd metaengine && go test -tags "goexperiment.jsonv2" -race -count=1 -timeout 60s -run 'TestMetaengine' .` — All Ginkgo specs pass, zero races
- `cd metaengine && go test -tags "goexperiment.jsonv2" -race -count=1 -run 'TestGraphFallback|TestReifyReflect|TestOnRecord' .` — All pass
- `nix run .#verify` — All modules pass except pre-existing cmd/cqrs-bench version-seq issue

---

## g) What to Do Next

1. **Tag benchkit/v4.4.0** — Include Truncate/TitleCase, update cmd/cqrs-bench dependency
2. **M9: Struct-composition multi-collection** — The next high-impact metaengine feature
3. **M8 remaining: StreamLog fallback on Dgraph** — Dgraph-specific append-ordered fallback
4. **M13: Calibration benchmarks** — Capture baseline numbers, add CI regression check
5. **Documentation:** Update AGENTS.md with graph fallback pattern, foldMu pattern, commandlifecycle module
