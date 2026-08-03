# Status: Metaengine Phase 3 Tasks T19-T27 + Verification

**Date:** 2026-08-03 09:35
**Session:** T19-T27 execution (continuation of Phase 3 Pareto plan)
**Verify Gate:** GREEN (all 11 checks pass)

---

## Completed Tasks

### T19: Soak Test Hardening

- Added `runtime.MemStats.TotalAlloc` delta tracking to `TestSoak_MemoryBounded` — reports allocs/event alongside heap growth
- 3× race verification: stable at 1.85s (50K events, 100 keys, ~273 allocs/event)
- Documented `SOAK_SKIP_10M` env var in AGENTS.md (skips 10M soak, 50K smoke always runs)

### T20: Typed Watcher Convenience Functions

- Added `WatchTyped[V](store, ctx, collection, key) (<-chan V, *Watcher[V])` — free function (Go can't have generic methods on non-generic types)
- Added `WatchTypedWithSeq[V](...)` — SeqValue variant for SSE Last-Event-ID support
- Tests: Memory engine (fast path), SQLite engine (JSON reify fallback), SeqValue variant — all pass with `-race`

### T21: SSE Reconnect Integration Test (SQLite)

- `TestSSE_ReconnectWithSQLite` — end-to-end Last-Event-ID reconnection with SQLite engine
- Verifies the JSON reify fallback path through SSE replay (map[string]any → V)
- Tests: live delivery, disconnect, write more events, reconnect with `Last-Event-ID: 2`, verify replay delivers correct events

### T22: Boundary Key-Type Validation

- Added `ErrKeyTypeMismatch` sentinel — returned when input struct has no field matching the query's declared key type
- Added `checkKeyTypeMatch` helper — extracted to keep `executeQueryInner` under gocyclo limit (30)
- Also extracted `executePointLookup` and `executeMembership` methods for complexity reduction
- Tests: valid input (no error), mismatched input (ErrKeyTypeMismatch), membership path

### T25: Iroh Bridge Evaluation + ADR-0096

- Researched Iroh Go/C binding availability (2026-08-03):
  - No official Go SDK — "Community" status only
  - `iroh-c-ffi` covers networking only, NOT `iroh-docs` (the CRDT KV store)
  - Community `iroh-go` is experimental, Linux-only
- **Decision:** Sidecar process short-term, CGo FFI long-term (once `iroh-docs` stabilizes in C API). Level 2 architecture (Iroh wraps local engine). No implementation now.
- ADR-0096 written with full tradeoff matrix. ADR README index caught up (0083-0096 added).

### T26: gopls Hint Cleanup

- Fixed 7 gopls hints in cmd/cqrs-lint:
  - 6 `infertypeargs` (removed unnecessary type arguments in `NewCommand`/`NewCLI` calls)
  - 1 `writestring` (split string concatenation in `versionVerbose()`)
- Remaining hints are intentional (3 `omitzero` suppressed by `//nolint:modernize`, 5+ `stdversion` false positives from JSON v2 experimental tag)

### T27: cqrs-lint Validation Against Examples

- Ran against all 3 examples (taskmanager, readme-quickstart, getting-started)
- **0 false positives** — all findings are legitimate anti-patterns
- Key findings: C005 (raw json.Unmarshal), A032 (string IDs), B028 (manual goroutine), V006 (version mismatch)
- 2 stale suppressions identified (metaengine.go:148 C025, main.go:143 C027)

---

## Deferred Tasks

### T23: Postgres GIN Containment Indexes

**Status:** Deferred — requires deep pgengine PushdownScan understanding
**What's needed:** Add `@>` operator support to JSONB containment queries, write testcontainers test

### T24: DuckDB LayoutPlanner Follow-ups

**Status:** Deferred — requires deep duckdbengine LayoutPlanner understanding
**What's needed:** explainScan, centralize helpers (extractFields, quoteIdent), benchmark, adttest matrix

---

## Critical Issue Resolved

### v4.4.0 Tag Staleness

- **Was:** Tagged at `ad9bcd6f` BEFORE T14-T18 code (14 commits behind HEAD, never pushed to origin)
- **Fix:** Force-moved to HEAD (`c45b39c8`) — safe because it was never pushed
- **Result:** v4.4.0 now includes Universal ADT (T1-T13) + Replication Polish (T14-T18) + T19-T27 work

### Unpushed Commits

- All commits + v4.4.0 tag remain local (not pushed to origin)
- User must run: `git push origin master && git push origin metaengine/v4.4.0`

---

## API Surface

- **3215 exports** (was 3212 before T19-T27)
- **+3 new:** `WatchTyped`, `WatchTypedWithSeq`, `ErrKeyTypeMismatch`
- Backward compatible — only additions, no removals

---

## Verify Gate Results

| Check                              | Status |
| ---------------------------------- | ------ |
| Build                              | PASS   |
| Vet                                | PASS   |
| Test (all modules)                 | PASS   |
| Race detector                      | PASS   |
| Lint (0 issues)                    | PASS   |
| Check Layers                       | PASS   |
| Check Duplication                  | PASS   |
| Check Coverage (±2.0%)             | PASS   |
| API Stability (3215 exports)       | PASS   |
| Doc Check (1223 refs, 42 packages) | PASS   |
| Documentation assertions           | PASS   |

---

## Files Changed This Session

### New Files

- `metaengine/watcher_typed_test.go` — T20 tests (3 tests: Memory, SQLite, SeqValue)
- `metaengine/boundary_keys_test.go` — T22 tests (2 tests: valid, mismatch)
- `docs/adr/0096-iroh-distributed-engine-bridge-evaluation.md` — T25 ADR

### Modified Files

- `metaengine/soak_test.go` — T19: TotalAlloc tracking
- `metaengine/dx.go` — T20: WatchTyped/WatchTypedWithSeq functions
- `metaengine/errors.go` — T22: ErrKeyTypeMismatch sentinel
- `metaengine/execute.go` — T22: checkKeyTypeMatch, executePointLookup, executeMembership extraction
- `metaengine/sse_replay_test.go` — T21: TestSSE_ReconnectWithSQLite
- `cmd/cqrs-lint/commands.go` — T26: infertypeargs + writestring fixes
- `cmd/cqrs-lint/doctor.go` — T26: infertypeargs fix
- `cmd/cqrs-lint/init.go` — T26: infertypeargs fix
- `cmd/cqrs-lint/main.go` — T26: infertypeargs fix
- `AGENTS.md` — T19: SOAK_SKIP_10M docs; T20/T22: new sentinel + watcher docs
- `docs/README.md` — ADR index: 0096 added, count updated to 94
- `docs/adr/README.md` — ADR index: 0083-0096 added
- `docs/api_surface.txt` — Regenerated (3212→3215)

---

## Resolution (2026-08-03)

T19 (soak `TotalAlloc` + `SOAK_SKIP_10M`), T20 (`WatchTyped[V]`/`WatchTypedWithSeq[V]`), T21 (`TestSSE_ReconnectWithSQLite` `31ec083b`), T22 (`ErrKeyTypeMismatch` `cbc572c8`), T25 (Iroh research + ADR-0096), T26 (7 gopls hints), T27 (cqrs-lint vs 3 examples, 0 false positives). v4.4.0 force-moved to `c45b39c8`. API 3212→3215.

**Deferred:** T23 (Postgres GIN Containment Indexes), T24 (DuckDB LayoutPlanner follow-ups). Captured in TODO_LIST.md.
