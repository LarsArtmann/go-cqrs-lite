# Session Status — "FIX ALL BUGS": seq-positional journal reads, benchkit dep-skew, third GREEN gate — 2026-08-15 03:30

> Standing instruction: READ→UNDERSTAND→RESEARCH→REFLECT→execute, one verified
> step at a time. This session executed the bug-fix items from the 02:31
> self-review's f-list. The 3 open questions (tag/push authorization, Go 1.26.6
> direction, SA1019 permanence) remain UNANSWERED — deliberately untouched.

## a) FULLY DONE (verified)

1. **THE headline bug — `JournalReadFrom` raw-seq filtering on all four SQL
   engines** (sqlite, pg, mysql, duckdb). They filtered `seq > afterSeq` on a
   seq counter GLOBAL across collections while the `StreamLogBackend` contract
   (and every caller: `EventAdapter.ReadFrom` → `lookupSeq`, `AdapterCore.
   ReadFromAfter`, the shared harness) passes a POSITION within the
   collection's journal. Any consumer running two collections on one SQL
   engine (e.g. commands-before-events) got **duplicate re-delivery on every
   resume** — projectionhost/catch-up subscribers would double-process.
   Fix: OFFSET over the collection-filtered, seq-ordered result (dialect
   variants: sqlite `LIMIT -1 OFFSET`, pg bare `OFFSET`, mysql
   `LIMIT 2^64-1 OFFSET`, duckdb `LIMIT ALL OFFSET`). memory/dgraph/pebble/
   bbolt/badger were already positional.
2. **Regression tests that PROVABLY catch the bug**:
   - `enginetest.RunStreamLogBackendTest(In)` gained an interleaved-collections
     phase (noise collection written first so target seqs start >1).
   - End-to-end `system.TestEventAdapter_ReadFrom_InterleavedCollections`
     (EventAdapter.ReadFrom cold+warm cache vs sqliteengine) — first-ever
     EventAdapter test coverage.
   - **Proof**: temporarily reverting the sqlite fix made BOTH tests fail with
     exactly the predicted re-delivery (`[a1 a2 a3]` vs `[a2 a3]`; 2 events vs
     1). Restored → green.
3. **Coverage gaps closed**: pgengine now runs the shared harness (replacing
   its hand-rolled subset); sqliteengine gained `TestStreamLogBackend_Contract`;
   badgerengine got its FIRST StreamLog contract test (had zero coverage).
   `engine.go` doc pins the positional contract; `adttest.RunMatrix` doc pins
   the fixed-collection/fresh-database constraint for shared-server engines.
4. **benchkit deterministic failures root-caused** (not flakes!):
   `TestRun_AnalyticalJournalScans` (SQLITE_BUSY 517) +
   `TestRun_Pebble_DiskSizerInterface` (0 bytes). **Dependency skew**:
   benchkit pinned `stack/sqlite` + `stack/pebble` at v4.1.0 — the old
   published tags lack the `SetMaxOpenConns(1)` pool cap and the `WithDiskSize`
   wiring. Passes only inside the workspace (local modules); fails standalone
   (GOWORK=off → stale tags). Bumped both to v4.3.0; full benchkit suite green
   standalone (34s). Proven pre-existing via pre-change-commit worktree.
5. **Live verification of all touched engines**: PG ✓ (incl. new harness),
   MySQL streamlog ✓ (MariaDB pushdown failures = pre-existing, recorded),
   Dgraph ✓ 24/24 parity incl. the new interleave phase (61s).
6. **Reverse LAYER meta-test** (`TestEveryModuleHasLayerEntry`): every go.mod
   dir must have a LAYER entry (81/81 today; only root/examples/integration/
   test-infra exempt). Forward test already existed; the gap direction is now
   covered.
7. **Gate hardening**: verify Test-phase timeout 5m→8m (was tighter than the
   slower Race phase's 8m — backwards asymmetry; duckdb ~76-91s clean but
   ceiling-hugging under load).
8. **Docs**: gate-exclusivity gotcha in AGENTS.md (the 25-min lesson);
   ephemeral-dgraph.sh header no longer lies about direct bash invocation;
   CHANGELOG (3 entries); TODO_LIST (MariaDB compat + seq-carrying perf item).

## b) Gate evidence (final)

- `nix run .#verify`: **EXIT=0, all 18 phases, 0 FAIL lines, 239 `ok` packages,
  `✅ Lint: 76/76 modules clean`, doc-check 1020 refs** (/tmp/verify-final3.log)
- Live: PG pgengine ok 3.1s; MySQL StreamLog roundtrip PASS; Dgraph 24/24 61.1s
- Standalone: benchkit 34.4s, sqliteengine 0.9s, system 0.12s, metaengine 16.4s

## c) Found but deliberately NOT fixed (recorded, needs decisions)

1. **mysqlengine vs MariaDB**: 3 pushdown tests emit MySQL-8 JSON path syntax <- OPEN. TODO_LIST 'Metaengine' (mysqlengine vs MariaDB compatibility, line 160)
   (`>'$.x' = CAST(? AS JSON)`) MariaDB rejects; ADTMatrix/HealthCheck fail
   "invalid connection" in nspawn. TODO_LIST item — dialect support is a
   feature decision.
2. **Seq-carrying journal reads** stays open as a PERFORMANCE item only <- OPEN. TODO_LIST 'Metaengine' (Seq-carrying journal reads perf follow-up, line 143)
   (correctness fixed); OFFSET skip is O(offset) per page on huge journals.
3. The 3 user questions from 02:31 (g1 tag/push, g2 Go 1.26.6, g3 SA1019) — <- OPEN. all three routed: tag/push = TODO_LIST 'Release / Tagging' + ROADMAP 'Open Questions' #1; Go 1.26.6 = OQ #2; SA1019 = TODO_LIST v5 + OQ #3
   still waiting; nothing tagged, nothing pushed, go.mod toolchain untouched.

## d) Process notes

- Applied the gate-exclusivity rule to myself: integrations ran sequentially,
  gate ran alone, report written AFTER the gate (lesson d4).
- Bisect used `git worktree` (never checkout/reset); probe tests deleted after
  use; temp sqlite fix-revert restored from backup copy.

---

## Resolution (2026-08-15)

All session work (a) was committed as `4a95bd04d` and re-verified by the
docs-health pass: reverse LAYER meta-test, interleaved-collections contract
phase, benchkit pin bumps, timeout asymmetry fix, AGENTS gotcha, dgraph
script header all present in the tree. The three deliberately-unfixed items
are routed (MariaDB + seq-carrying perf live in TODO_LIST "Metaengine"; the
three user questions in ROADMAP "Open Questions"). Archived.
