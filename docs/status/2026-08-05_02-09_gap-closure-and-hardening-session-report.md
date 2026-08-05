# Gap Closure & Hardening — Session Status Report

> **Date:** 2026-08-05 02:09
> **Session goal:** Close all critical gaps identified in the prior session report (`2026-08-05_00-38_critical-fixes-hardening-session-report.md`)
> **Outcome:** All 10 planned tasks completed. 2 additional bugs found and fixed. 18 commits (mix of session work + auto-commit daemon). All tests pass with `-race`. Api-stability golden at 3530 exports.

---

## a) FULLY DONE (Verified Working)

### G1: Pebble Restart Safety Test — ✅ DONE

**Files created:** `metaengine/pebbleengine/restart_safety_test.go` (157 lines, 2 tests)

- `TestPebbleRestartSafety_StreamAndJournal`: Opens persistent Pebble DB → appends 3 events + Map + Multimap → closes → reopens same dir → appends 2 more events → verifies ALL 5 events retained (not overwritten), version=5 (not 2), journal has 5 entries, Map data survived, new Map write doesn't overwrite, new Multimap entry doesn't collide.
- `TestPebbleRestartSafety_FromDB`: Same pattern via `NewPebbleEngineFromDB` (caller-owned DB path).
- Both tests pass with `-count=3 -race`.

**Verifies:** The `seedSeqCounters()` implementation from the prior session actually works. Without seeding, reopening would reset all counters to 0 and new writes would overwrite existing keys silently.

### G2: E2E Snapshot Integration Test — ✅ DONE (with 2 bug fixes)

**Files created:** `system/snapshot_e2e_test.go` (285 lines, 3 tests)
**Files modified:** `system/constructor.go`, `system/register.go` (new), `system/snapshot_adapter.go`, `system/system.go`, `system/errors.go`

- `TestSystem_SnapshotAdapterDirect`: Save → Load → Delete roundtrip through System with SQLite engine. Verifies state roundtrips correctly.
- `TestSystem_SnapshotAdapterLoadAtVersion`: Version-bounded snapshot loading. Verifies `LoadAtVersion(15)` finds snapshot at v10, `LoadAtVersion(5)` returns nil.
- `TestSystem_SnapshotE2E_DeciderLifecycle`: Full decider → snapshot → reload cycle. Dispatches `task.create` → verifies snapshot saved at v1 → dispatches `task.complete` → verifies snapshot updated to v2 with correct state.

**Bug #1 found and fixed: `SnapshotAdapter.Save` key mismatch**
- Save used `snap.StreamID.String()` (bare ID: `"01HXY..."`)
- Load/Delete used `ref.StreamKey()` (`"Type:ID"`: `"Task:01HXY..."`)
- These are different strings — snapshots written via Save could never be found via Load.
- **Fix:** Changed Save to construct the key as `snap.StreamType.String()+":"+snap.StreamID.String()` to match `StreamKey()`.

**Bug #2 found and fixed: `RegisterDecider` missing codec wiring**
- Prior session wired `WithSnapshotStore` but not `WithCodec`.
- Without a codec, `snapshot.ShouldSnapshotFor` always returns false (requires strategy + store + codec all non-nil).
- Snapshots were silently never written — the wiring was dead code.
- **Fix:** Added `decider.WithCodec[State](codec.JSONCodec{})` to the repo options when snapStore is present. Also added `WithSnapshotStrategy` option type (`system/register.go`) so consumers can opt into automatic snapshot creation. Added `codec/v4` import to constructor.go.

### G3: PG StreamLog Roundtrip Test — ✅ DONE

**Files created:** `metaengine/pgengine/stream_log_test.go` (120 lines, 2 tests)

- `TestPostgresEngine_StreamLogRoundtrip`: Append to 2 streams, verify StreamRead (3 values), StreamVersion (3), JournalReadAll (4 entries), JournalReadFrom (2 entries after seq 2).
- `TestPostgresEngine_StreamLogAtomicAppender`: Append at v0 (succeeds), v2 (succeeds), v0 stale (fails with ErrVersionConflict). Verifies final version is 3.
- Both tests use `pgDSN(t)` helper which auto-skips when no PG available. Compiles clean.

### G4: Replace `indexOfByte` — ✅ DONE

**File modified:** `metaengine/pebbleengine/seq_seeding.go`

- Replaced `indexOfByte(rest, 0)` with `bytes.IndexByte(rest, 0)`.
- Deleted the 8-line `indexOfByte` function.
- Added `"bytes"` to imports.

### G5: Update metaengine README — ✅ DONE

**File modified:** `metaengine/README.md`

- Updated `NewPebbleEngineFromDB` table entry: changed from `NewPebbleEngineFromDB` to `NewPebbleEngineFromDB(db)` and noted it returns `(Engine, error)` and seeds seq counters.
- Added a blockquote documentation section explaining Pebble seq seeding: when it runs, what it does, the O(N) startup cost, and why `NewPebbleEngineFromDB` returns an error.

### G6: Add `StreamReadFromVersion` to Backend Interface — ✅ DONE

**Files modified:** `metaengine/engine.go`, `metaengine/memory_stream_log.go`, `metaengine/sqlite_stream_log.go`

- Added `StreamReadFromVersion(ctx, collection, streamID, minVersion)` to the `StreamTemporalReader` interface.
- **Memory engine implementation:** Uses slice indexing — `stream[minVersion-1:]` (1-indexed to 0-indexed conversion).
- **SQLite engine implementation:** Uses `LIMIT -1 OFFSET ?` with `minVersion-1` because `seq` is global auto-increment (not per-stream), so OFFSET skips the first N entries within this stream's ordered result set.
- Compile-time assertions verify both engines implement the expanded interface.

### G7: Wire `StreamReadFromVersion` into `LoadFromVersion` — ✅ DONE

**File modified:** `system/adapter_event.go`

- When the backend implements `StreamTemporalReader`, `LoadFromVersion` now delegates to `StreamReadFromVersion` instead of loading the full stream and slicing.
- **Critical semantic fix:** `LoadFromVersion(version)` is 0-indexed (skip N events), but `StreamReadFromVersion(minVersion)` is 1-indexed inclusive. The adapter adds `+1` to convert: `int64(version)+1`.
- Without this conversion, the first `version` events would be INCLUDED instead of skipped, causing version conflicts on the next write (caught by the E2E test).

### G8: Full Workspace Build — ✅ DONE

- `go build -tags "goexperiment.jsonv2" ./...` clean across all 65 modules.
- **Fixed unrelated breakage:** `metaengine/irohengine/engine.go` had an unused `"errors"` import (auto-commit daemon damage). Removed it.

### G9: Full Test Suite + Format + Vet + Api-Stability — ✅ DONE

- **Format:** `gofumpt -w` on all 12 modified files.
- **Vet:** Clean on system, metaengine, pebbleengine, pgengine.
- **Tests:** All pass with `-race`:
  - system: 1.1s
  - metaengine core: 93s
  - metaengine/adttest: 1.0s
  - metaengine/pebbleengine: 1.1s
  - metaengine/duckdbengine: 1.3s
  - metaengine/pgengine: 33s (with testcontainers)
- **Api-stability golden:** Regenerated to 3530 exports (up from 3502).

### G10: Nix Verify Gate — ✅ DONE (with caveat)

- Ran `nix run .#verify` 4 times.
- All tests, lint, doc-checks, vet, race pass.
- **Caveat:** The api-stability check inside the nix sandbox sometimes fails because the auto-commit daemon adds new exports between golden regeneration and the nix build sandbox snapshot. This is a transient timing issue, not a real failure. The api-stability test passes consistently when run directly (`cd cmd/api-stability && GOWORK=off go test`).

---

## b) PARTIALLY DONE

### `constructor.go` at 366 lines (convention is 350)

- Extracted `RegisterDeciderOption`/`registerDeciderConfig`/`WithSnapshotStrategy` into `system/register.go` (18 lines).
- `constructor.go` is 366 lines — 16 over the 350 convention. Not CI-enforced (checked: no lint rule enforces it). The `RegisterDecider` function itself is the bulk (generic function with closures).
- Could extract further but the function is cohesive — splitting would reduce readability.

### Api-stability golden churn from auto-commit daemon

- The daemon committed new exports (`PercentileIdx`, `SortDurations`, irohengine error vars) between my golden regeneration and the nix verify run.
- Had to regenerate the golden 3 times during the session.
- This is a process issue, not a code issue.

---

## c) NOT STARTED

### DuckDB/PG go.mod version drift (still v4.0.0 for metaengine require)

- `duckdbengine/go.mod` and `pgengine/go.mod` both require `metaengine/v4 v4.0.0` (with `replace => ../`).
- The actual metaengine is at v4.4.0. The replace directive masks this in workspace mode.
- External consumers of duckdbengine/pgengine (without replace directives) would get v4.0.0 which lacks `EncodeStreamValue`/`DecodeStreamValue`/`StreamReadFromVersion`.
- Not fixed — needs a metaengine tag release first.

### Tag metaengine v4.5.0

- `EncodeStreamValue`, `DecodeStreamValue`, `StreamReadFromVersion`, `StreamReadFromVersion` are new public API.
- Need a release tag so engine modules can bump their require.
- Not done — tagging is a release process step.

### Meta-test verifying all engines implement AtomicAppender / StreamLogBackend

- Cross-engine contract enforcement meta-test (item #49/#50 from prior report).
- Not written.

---

## d) TOTALLY FUCKED UP

### Nothing is totally fucked up.

But there are issues worth calling out:

### ⚠️ The auto-commit daemon caused significant interference

The daemon committed **18+ times** during this session, including:
- Refactoring my code while I was still working on it (e.g., `system/adapter_event.go` was refactored by the daemon's "share temporal-fast-path logic" commit)
- Adding new exports to `irohengine` that broke the api-stability golden 3 times
- Breaking the build with an unused `errors` import in `irohengine/engine.go`
- Creating status reports and dedup plans I didn't author

This made verification harder — I'd regenerate the golden, then by the time the nix verify gate ran, the daemon had added 2 more exports.

### ⚠️ I didn't catch the 0-indexed vs 1-indexed semantic mismatch on first try

When wiring `StreamReadFromVersion` into `LoadFromVersion`, I initially passed `version` directly. The E2E snapshot test caught the version conflict immediately (because `LoadFromVersion(1)` returned events starting at version 1 instead of skipping it, causing the decider to see stale state). Fixed with `version+1`, but I should have caught this from the interface documentation alone.

---

## e) WHAT WE SHOULD IMPROVE

1. **Tag metaengine v4.5.0** — New public API (`EncodeStreamValue`, `DecodeStreamValue`, `StreamReadFromVersion`) needs a release tag.

2. **Bump engine module go.mod requires** — duckdbengine and pgengine still require metaengine v4.0.0. Bump after tagging.

3. **Write `TestStreamReadFromVersion` unit tests** — I added the method to Memory + SQLite engines but only tested it indirectly through the EventAdapter. Direct unit tests on the engine implementations would catch engine-specific issues (e.g., SQLite OFFSET behavior).

4. **Add `StreamReadFromVersion` to Pebble/DuckDB/PG engines** — Currently only Memory + SQLite implement `StreamTemporalReader`. Pebble could implement it via prefix scan + offset. DuckDB/PG could use SQL OFFSET.

5. **SnapshotAdapter `CreatedAt` is still zero on Load** — The adapter sets `CreatedAt: time.Time{}` because `SnapshotBackend` has no timestamp field. This means snapshot age/lifecycle metadata is lost. Either add a timestamp to `SnapshotBackend.SnapshotSave` or document the limitation in the adapter.

6. **`constructor.go` is 366 lines** — 16 over the 350 convention. Could extract the `RegisterCommand` generic function into a separate file, but it's cohesive.

7. **Consolidate SQLite `encodeStreamValue`** — DuckDB and PG now use shared `metaengine.EncodeStreamValue`. SQLite still uses its local `encodeStreamValue` which wraps `encodeJSON`. Should consolidate for consistency, but SQLite's version has string-passthrough optimization.

8. **The api-stability verify-gate churn** — Consider adding a `nix run .#verify-fast` that skips api-stability (or runs it with `GOWORK=off` directly) to avoid the nix sandbox golden-cache timing issue with the auto-commit daemon.

9. **E2E test uses `EveryNEvents(1)`** — This forces a snapshot after every single event. A more realistic test would use `EveryNEvents(3)` or `EveryNEvents(5)` and verify the snapshot fires at the right threshold.

10. **No test for `LoadFromVersion` fast path directly** — The EventAdapter test should verify that when the backend implements `StreamTemporalReader`, `LoadFromVersion` delegates to `StreamReadFromVersion` (not full-load-then-slice). Currently only tested indirectly through the decider lifecycle.

---

## f) Up to 50 Things We Should Get Done Next

### P0 — Release & Version Alignment

1. **Tag metaengine v4.5.0** — New API needs a release
2. **Bump `duckdbengine/go.mod` metaengine require** from v4.0.0 → v4.5.0
3. **Bump `pgengine/go.mod` metaengine require** from v4.0.0 → v4.5.0
4. **Bump `pebbleengine/go.mod` metaengine require** if stale
5. **Bump `irohengine/go.mod` metaengine require** if stale
6. **Bump `adttest/go.mod` metaengine require** if stale
7. **Bump `projectionadapter/go.mod` metaengine require** if stale

### P1 — Test Coverage

8. **Write direct `TestStreamReadFromVersion` for Memory engine** — append 5, read from v3, verify 3 values
9. **Write direct `TestStreamReadFromVersion` for SQLite engine** — same pattern
10. **Write `TestLoadFromVersion_FastPath`** — verify EventAdapter delegates to StreamReadFromVersion when temporal reader available
11. **Write `TestLoadFromVersion_Fallback`** — verify full-load-then-slice when temporal reader NOT available
12. **Write `TestSnapshotAdapter_MissingSnapshot`** — Load returns nil, nil for unknown stream
13. **Write concurrent `StreamAppendExpected` test for Pebble** — only Memory + SQLite have concurrent tests
14. **Write concurrent `StreamAppendExpected` test for DuckDB** — same
15. **Write concurrent `StreamAppendExpected` test for PG** — same
16. **Add `SnapshotBackend` to `adttest.RunMatrix`** — cross-engine snapshot parity
17. **Write fuzz test for `EncodeStreamValue`/`DecodeStreamValue`** — roundtrip property
18. **Write `TestSnapshotE2E_EveryNEvents3`** — realistic snapshot threshold, not EveryNEvents(1)

### P2 — Feature Completeness

19. **Implement `StreamReadFromVersion` on Pebble engine** — prefix scan + skip first N keys
20. **Implement `StreamReadFromVersion` on DuckDB engine** — SQL OFFSET
21. **Implement `StreamReadFromVersion` on PG engine** — SQL OFFSET
22. **Implement `StreamTemporalReader` on Pebble engine** — already has `StreamReadAsOfVersion`? Check.
23. **Implement `SnapshotBackend` on DuckDB engine** — currently only Memory + SQLite
24. **Implement `SnapshotBackend` on PG engine** — same
25. **Add `CreatedAt` to `SnapshotBackend.SnapshotSave`** — or document the limitation
26. **Add `SnapshotBackend.SnapshotExists` method** — avoids loading just to check presence
27. **Wire `StreamTemporalReader` into `LoadToTimestamp`** — currently iterates all events
28. **Add `System.SnapshotStore()` to introspection** — show in Doctor/Explain output
29. **Add `StreamLogBackend` to metaengine cost model** — planner should know stream log costs
30. **Add `SnapshotBackend` to metaengine Doctor report** — `store.Doctor(ctx)` should show snapshot capabilities

### P3 — Polish

31. **Consolidate SQLite `encodeStreamValue`** into `metaengine.EncodeStreamValue`
32. **Document `SnapshotAdapter` limitation** — `CreatedAt` is zero on load
33. **Add `MultiBus` documentation** — clarify that fan-out buses implement `event.Bus` for direct subscription
34. **Write a migration guide for `NewPebbleEngineFromDB`** — consumers need to handle the new error return
35. **Update `docs/planning/metaengine-redesign.md`** — annotate AtomicAppender + StreamTemporalReader + StreamReadFromVersion support on all engines
36. **Update `AGENTS.md`** — mention StreamReadFromVersion, SnapshotAdapter, WithSnapshotStrategy
37. **Add `RegisterDeciderOption` to cqrs-lint module catalog** — lint should detect system.WithSnapshotStrategy usage
38. **Write a Pebble seq seeding benchmark** — measure O(N) scan cost on large DB (10K+ keys)
39. **Consider lazy seq seeding** — seed on first access per-collection to avoid O(N) startup
40. **Add `context.Context` to `NewPebbleEngineFromDB`** — `seedSeqCounters` does I/O; should accept context
41. **Write meta-test verifying all engines implement `AtomicAppender`** — cross-engine contract
42. **Write meta-test verifying all engines implement `StreamLogBackend`** — same
43. **Consider a `StreamAppendExpected` that returns the new version** — currently returns error only
44. **Run `nix run .#check-layers`** — verify dependency budgets not exceeded by codec import in system/
45. **Run `nix run .#check-coverage`** — verify coverage didn't drop
46. **Run `nix run .#check-duplication`** — verify consolidation reduced duplication score
47. **Add `ErrVersionConflict` to cqrs-lint error catalog** — lint should recognize this error
48. **Extract Pebble seq seeding to a separate file** — `seq_seeding.go` is 140 lines, could split stream/journal/multimap
49. **Write a CGo-fallback stub for DuckDB tests** — `stream_log_cgo_test.go` is invisible without CGo
50. **Document the `afterSeq+1` Pebble JournalReadFrom fix** — add a comment explaining the exclusive lower bound

---

## g) Questions I CANNOT Answer Myself

### Q1: Should I implement `StreamReadFromVersion` on Pebble/DuckDB/PG engines now, or wait?

Currently only Memory + SQLite implement `StreamTemporalReader`. The system EventAdapter type-asserts and falls back to full-load-then-slice for engines that don't implement it. This is correct but means Pebble/DuckDB/PG miss the optimization.

- **(a)** Implement now (consistency across all engines)
- **(b)** Wait until there's a measured performance need (YAGNI)

### Q2: Should the auto-commit daemon be disabled during verify-gate sessions?

The daemon committed 18+ times during this session, causing:
- 3 api-stability golden regenerations (new exports added between regen and verify)
- 1 build breakage (unused import)
- Refactoring of my code while I was still editing it

- **(a)** Disable the daemon during active work sessions
- **(b)** Leave it running and accept the churn (status quo)
- **(c)** Add a "pause daemon" mechanism for verify-gate runs

### Q3: Should `SnapshotBackend.SnapshotSave` gain a `createdAt time.Time` parameter?

The `SnapshotAdapter` currently sets `CreatedAt: time.Time{}` on Load because the backend has no timestamp field. This loses snapshot age metadata. Adding a timestamp parameter would be a breaking change to the `SnapshotBackend` interface.

- **(a)** Add it now (breaking, but only 2 implementations exist: Memory + SQLite)
- **(b)** Document the limitation and leave it
- **(c)** Add a separate `SnapshotMetadata` interface for engines that support timestamps
