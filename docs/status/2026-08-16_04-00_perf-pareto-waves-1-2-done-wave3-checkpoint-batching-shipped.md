# Status Report — Perf Pareto Waves 1–2 DONE, Wave 3 checkpoint batching shipped (2026-08-16 04:00)

> **Session:** Execute `docs/planning/2026-08-16_03-18_PERF-PARETO-SAFETY-FIRST-EXECUTION.md`
> (W1 safety → W2 proof & green → W3 IO wins → W4 remainder; F24–F27 tagging stays BLOCKED on user).
> **Stop reason:** Time-box reached 04:00 while Wave 3 item 1 (checkpoint batching) had just gone
> green; verify-fast re-run and Waves 3-remainder/4 not yet executed.

---

## (a) Fully done

### Wave 1 — SAFETY (the 1% that delivers 51%) — COMPLETE, all green

- **F1 guard strategy**: byte-estimate cap at 8 MiB (`MaxStatementBytes` = 50% of MariaDB's
  default 16 MB `max_allowed_packet`), minimum chunk = 1 row (oversized single events still
  insert; operator must size packet to largest single event). `mysqlengine/dialect.go` read —
  JSON-SQL dialect only, packet risk is purely in the chunking.
- **F2/F3 dual cap implemented**:
  - New `storage/sql/batch_size.go`: `MaxStatementBytes` (8 MiB, exported), `bytesPerEventOverhead`
    (256 B: ID/type/stream/version/time fields + placeholder syntax + driver framing),
    `RowsWithinByteCap(start, n, maxRows, rowBytes)` — clamps `maxRows` to ≥1, always returns ≥1
    row for a non-empty range, splits when cumulative estimate EXCEEDS the cap (boundary: total
    == cap stays in one statement).
  - `SharedBatchInsertEvents`: metadata now marshaled **once per Save** (was once per chunk —
    bonus waste removal), then chunked by BOTH dialect param cap AND byte cap.
  - **`storage/view/batch.go` `BatchSet` got the same guard** (plan only scoped events, but the
    prior session's 99→8191-row raise made view rows with large doc columns the same defect
    class): `estimateItemBytes` (key + per-column `estimateArgBytes` + 64 B row overhead),
    `Extract` double-call documented safe (pure field projection).
  - `Inserter.InsertAll` audited: row-by-row by design — no exposure.
- **F4/F5 tests**: new `storage/sql/batch_insert_test.go` (dialect table incl. unknown + nil → 999;
  7-case `RowsWithinByteCap` table incl. boundary/oversized-single/zero-maxRows clamp; 9×1 MiB
  behavioral chunk test; 250-event SQLite param-cap test), `storage/view/batch_large_test.go`
  (6×2 MiB view values chunk safely; `estimateArgBytes` unit test). All green.
- **Defect fixed during F5**: `MaxParametersForDialect` doc promised "unknown dialects keep the
  conservative SQLite limit" but the impl returned 32767 for everything non-SQLite. Now a type
  switch: SQLite→999, Postgres/MySQL/DuckDB→32767, unknown/nil→999 (custom dialects can inherit
  the wider limit by embedding a known one). Verified only 4 in-repo Dialect impls exist.
- **F6/F7 MySQL VM integration — GREEN** (after killing a stale orphaned QEMU from Aug 15 holding
  port 33070 that invalidated the first run): full `stack/mysql` suite incl. new
  `TestEventSinkSave_LargeBatchWithinPacketLimit` — **2000 events × 8 KiB (~16 MiB total, would
  exceed a default MariaDB packet as one 3276-row statement) saved through the guard and loaded
  back complete on real MariaDB in the QEMU VM.**
- **F8 decision recorded**: pgengine `calibration_bench_test.go` chunk constants (500/1000) are
  calibration targets, not protocol chunking (1000×3 params < limits); unification would change
  what the benchmark measures. Skipped deliberately.
- **F10 DuckDB**: `stack/duckdb` (CGo) + entire `storage/...` tree green with the new chunking.

### Wave 2 — PROOF & GREEN (partial but the two RED gates are fixed)

- **F13 exhaustruct**: `commandlifecycle/recorder.go:196` — full explicit `event.Metadata` literal.
- **F15/F16 art-dupl — 0 new clone groups, closed by REAL dedup, no baseline inflation**:
  - asrecord trio: `brandedString`/`actorString` moved to new `metadata/ids.go` as exported
    `metadata.BrandedString[T]`/`metadata.ActorString(Tracing)` (metadata owns `Tracing` and
    already depends on `id/`; `record/` is zero-dep and cannot host it). Three copies deleted,
    ~10 call sites rewired, compile-check `var _` moved with it.
  - `scenario/dsl.go`: shared `Then`/`ThenEvents` prologue extracted to `decideEvents(method)`
    (also fixes the misleading `When:` error prefix — now names the actual step).
  - `nix run .#check-duplication`: **0 clone groups** (baseline 99, nothing new).
- **F17–F19 CHANGELOG**: one "Changed — storage perf" section with the 33x + guard numbers,
  cache-line pad ns/op, deserialize ns/B/allocs.
- **F20–F22 skill refs**: modules.md event row (+`ReconstructEventWithMetadata`) and storage row
  (+`MaxParametersForDialect`/`MaxStatementBytes`/`RowsWithinByteCap` with the packet-safety
  one-liner); WithActor TODO verified already documented (core.md §3.8, recipes §2.21, modules
  row) and checked off in TODO_LIST.
- **Golden regen + doc-check**: api_surface.txt at 4093 exports (+`ReconstructEventWithMetadata`,
  +3 sql exports, +2 metadata exports); doc-check 868 references valid.

### Wave 3 item 1 — projectionhost checkpoint batching (F28–F30) — IMPLEMENTED, module-green

- **F28 options** (opt-in, defaults byte-identical): `WithCheckpointEvery(n)` (persist after
  every n live events; n<2 ignored), `WithCheckpointInterval(d)` (persist when d elapsed since
  last save, checked on next event; combine — first threshold wins). Default: save after EVERY
  live event, unchanged.
- **F29 live handler batching**: new `projectionhost/worker_checkpoint.go`
  (`recordLiveCheckpoint` stages + flushes per cadence; `shouldFlushCheckpoint`;
  `flushCheckpoint`; `flushPendingCheckpoint`); worker fields `cpPending/cpHasPending/
  cpSinceSave/cpLastSave` guarded by `handleMu` (same lock domain as the subscriber callback);
  `run()` defers a flush; **`Host.Stop()` flushes all workers' pending checkpoints after wg.Wait**
  — required because with non-blocking subscribers the worker goroutine exits after catch-up
  while live events keep flowing through bus callbacks.
- **F30 tests** (`checkpoint_cadence_test.go`, 4 tests, all green): default saves every event
  (5 = 1 catch-up batch + 4 live); every(3) → flushes at live events 3/6 + shutdown flush at
  Stop; crash window bounded (every(100): only the catch-up save is durable mid-stream);
  interval(120ms) burst/sleep/flush cadence. Semantics documented: delivery stays at-least-once
  (same contract as the replay→live overlap); catch-up/replay drain unchanged (per-batch saves).

## (b) Partially done

~~- **F11 verify-fast**: first run executed while I was mid-edit on projectionhost (see (d)),
  hit a transient unused-import build failure, then failed `system/v4
  TestSystem_ResetProjection_RestartAndReplay`. **Proven PRE-EXISTING**: fails at pre-session
  baseline `f836c7f1c` in a clean worktree (and at `06e046c2f`). A CONCURRENT session is already
  on it (staged `system/system_hardening_test.go`/`system/testdsn_test.go` +
  `docs/status/2026-08-16_03-44_withactor-resume-gate-investigation-two-defects.md`). Clean
  end-to-end verify-fast re-run still owed.~~ done — frozen-tree `#verify` GREEN 2026-08-16 13:15 (run #4), api-surface green after the parallel wave's golden regen; the replay failure was the 03-44 fixture bug, fixed upstream `5d66308c3`

~~- **F14 lint gate**: exhaustruct fix is compile+test green, but `nix run .#lint` itself not re-run.~~ done — `#lint` 76/76 GREEN (13-15 run #4)

~~- **F31 docs for checkpoint batching**: recipes.md note + modules.md projectionhost row MISSING.~~ done — readmodels.md checkpoint note + modules.md projectionhost row document both options

~~- **Golden for the 2 new projectionhost options NOT regenerated** (same-edit rule violated — top
  of next-50).~~ done — golden regenerated with the parallel wave; api-surface phase green at 13-15

~~- **F9**: MySQL/DuckDB results recorded in CHANGELOG but not yet in a status follow-up (this file
  closes that gap).~~ closed — this file + CHANGELOG `[2026-08-16 module releases]` record the MySQL/DuckDB results

## (c) Not started

~~- F24–F27 tag chain ([BLOCKED:user] — now a FOUR-module chain, see Q1).~~ partial — the 22-tag chain shipped (metadata `v4.5.0`, event `v4.7.0`, storage `v4.7.0`→retracted→`v4.7.1`); projectionhost checkpoint-options tag + master pin bumps (command/query→metadata v4.5.0, pebble/bbolt→event, stack/mysql→storage) still pending — TODO_LIST wave-4 batch + strand commits `092b5e8a8`/`4907b6afc` off master

~~- Wave 3 remainder: F32–F34 bbolt `db.Batch` opt-in path; F35–F36 PG COPY spike+bench; F37–F38
  pebble knobs + preset wiring; F39 perf ledger `docs/BENCHMARKS.md`; F40 TODO_LIST harvest of
  Wave 3/4 leftovers.~~ done — bbolt `WithBatchCommit`, PG COPY `WithCopyAppend` (1.41x/1.49x), pebble knobs (defaults pinned byte-identical), `docs/BENCHMARKS.md` ledger, TODO_LIST harvest

~~- All of Wave 4 (F41–F60).~~ done — F55–F59 shipped (`9541df676`, `921147a01`), F60 conformance skeleton GREEN (`30711eb79b`), capability audit + false-sharing pads measured (`342699d00`); only F41 (bbolt deserialize bench) still open

~~- `-race` pass on projectionhost with the new mutex-guarded state.~~ done — race stage GREEN (13-15 run #4)

## (d) Totally fucked up / near-misses (honest ledger)

1. **Ran verify-fast while actively editing projectionhost** — the gate compiled a half-edited
   tree (unused `errorfamily` import break) and muddied the failure signal. Freeze edits during
   repo-wide gates.
2. **First MySQL VM run wasted + state-dir corruption**: an orphaned `--keep-alive` QEMU from a
   prior session (16 h old) held port 33070; the new VM couldn't bind, and the new driver's
   state-dir deletion yanked the disk from under the OLD VM mid-test (resets + stale rows).
   Always check `ss -tlnp | grep 33070` + stray qemu before `#integration-mysql-vm`.
3. **Same-edit golden rule violated** for `WithCheckpointEvery`/`WithCheckpointInterval`.
4. First cadence-test draft forgot the catch-up drain's own batch save (+1 to every count) —
   caught by reasoning before running, expectations rewritten.
5. Python rewrite regex `\nbrandedString\(` missed indented call sites → transient build break;
   fixed with `\b` matching.
6. Nearly misattributed the system test failure to my change — the worktree-at-baseline check
   saved a wild goose chase; that technique is now the default for "did I break this?".

## (e) Improvements made beyond the letter of the plan

1. Packet guard extended to `view.BatchSet` (same defect class the 33x raise created).
2. `MaxParametersForDialect` doc/impl contradiction fixed (unknown dialects truly conservative).
3. Metadata marshaled once per Save instead of once per chunk.
4. Dupl gate closed by real dedup into `metadata/` rather than baseline inflation.
5. `Host.Stop()` flush covers the non-blocking-subscriber topology (worker-defer alone would
   have silently skipped the flush where the worker goroutine exits before live traffic).

## (f) Next 50 (prioritized; 1–10 are the resume wave)

~~1. Regen api-stability golden (+`WithCheckpointEvery`/`WithCheckpointInterval`).~~ done — golden regen landed with the parallel wave

~~2. `nix run .#lint` (F14 close-out).~~ done — lint 76/76 (13-15)

~~3. Frozen-tree `nix run .#verify-fast` end-to-end (F11 proper).~~ done — full `#verify` GREEN (13-15 run #4)

~~4. `nix run .#check-coverage` (F12); add tests for uncovered new code.~~ done — `#check-coverage` EXIT=0 (13-15)

~~5. `projectionhost` `-race -count=3` on the new checkpoint state.~~ done — projectionhost green under the race stage (13-15); explicit `-count=3` not recorded

~~6. F31: recipes.md §readmodels checkpoint-tuning note.~~ done — readmodels.md checkpoint-tuning note present

~~7. modules.md projectionhost row: + both checkpoint options.~~ done — modules.md projectionhost row lists `WithCheckpointEvery`/`WithCheckpointInterval`

~~8. CHANGELOG entry for checkpoint batching (Wave 3 ship note).~~ done — CHANGELOG "Wave-3 IO wins" section (line 289)

~~9. Coordinate on `TestSystem_ResetProjection_RestartAndReplay` with the concurrent session
   (my baseline-proof at `f836c7f1c` is useful input; their fixes are staged, not mine to touch).~~ done — resolved upstream: fixture bug fixed by the concurrent session `5d66308c3` (03-44 §h.2)

~~10. Re-run `#check-duplication` + `#check-arch` after the above (pre-tag standing gates).~~ partial — `#check-duplication` EXIT=0 (13-15); `#check-arch` run not recorded

~~11. [BLOCKED:user Q1] Tag `metadata/v4 v4.5.0` (BrandedString/ActorString).~~ done — `metadata/v4.5.0` tagged (04:15)

~~12. [BLOCKED:user Q1] Tag `event/v4 v4.6.1` (ReconstructEventWithMetadata).~~ done differently — shipped as `event/v4.7.0` (04:16), not v4.6.1; `ReconstructEventWithMetadata` included

~~13. [BLOCKED:user Q1] Tag `storage/v4 v4.7.0` (batch_size exports + packet-safe chunking).~~ done — `storage/v4.7.0` tagged, retracted, superseded by `v4.7.1`

~~14. [BLOCKED:user Q1] Tag `projectionhost/v4` next patch (checkpoint options).~~ open [BLOCKED] — `projectionhost/v4.3.0` predates the options; wave-4 batch pending (TODO_LIST)

~~15. [BLOCKED:user] Bump command/query/event pins → metadata v4.5.0 (standalone builds).~~ open [BLOCKED] — master command/query still pin metadata `v4.4.0`; the pins live in strand commits `092b5e8a8` off master

~~16. [BLOCKED:user] Bump storage/pebble + storage/bbolt pins → event v4.6.1.~~ open [BLOCKED 🔥] — pebble/bbolt reference the unpublished adopt API; standalone RED until event re-tag (TODO_LIST)

~~17. [BLOCKED:user] Bump stack/mysql pin → storage v4.7.0.~~ open [BLOCKED] — stack/mysql still pins storage `v4.6.0`

~~18. [BLOCKED:user] `#vulncheck` (per-module standalone builds resolve the new APIs).~~ open — `#vulncheck` run not recorded

~~19. F32: bbolt `WithBatchCommit` design; read `db.Batch` fn-retry semantics.~~ done — `bbolt.WithBatchCommit()` shipped with retry semantics verified

~~20. F33: bbolt opt-in batch-write path (single-writer default unchanged).~~ done — opt-in path, single-writer default unchanged

~~21. F34: bbolt batch tests (concurrent writers → identical journal; fn retry-idempotence).~~ done — concurrent-writers journal identity + fn retry-idempotence verified (TODO_LIST DONE)

~~22. F35: pgx COPY FROM spike behind interface (stream_log backfill only).~~ done — `pgengine/stream_copy.go` + `WithCopyAppend(n)`

~~23. F36: COPY vs multi-VALUES bench at 10K/100K rows; keep winner behind option.~~ done — measured 1.41x @10k / 1.49x @100k rows

~~24. F37: pebble `WithMemTableSize/WithCacheSize/WithWALBytesPerSync/WithCompression`.~~ done — `WithMemTableSize`/`WithBlockCacheSize`/`WithWALBytesPerSync`/`WithPebbleCompression`

~~25. F38: wire knobs through stack/pebble; defaults byte-identical test.~~ done — wired through stack/pebble; defaults pinned byte-identical by test

~~26. F39: `docs/BENCHMARKS.md` perf ledger.~~ done — `docs/BENCHMARKS.md` is the ledger

~~27. F40: harvest Wave 3/4 leftovers into TODO_LIST.~~ done — TODO_LIST perf section carries the wave-3/4 outcomes

~~28. F41: bbolt deserialize benchmark (extrapolated → measured).~~ open — bbolt deserialize benchmark still missing

~~29. F42: `reconstructEventAdopt` design + equivalence/race tests.~~ done — `event.ReconstructEventWithAdoptedPayload` + equivalence/race tests (`5b8a9a615`)

~~30. F43: wire adopt path into pebble+bbolt; bench delta.~~ done in-tree, release-blocked — pebble/bbolt call the adopt API but `event/v4.7.0` lacks it; standalone RED until re-tag (same 🔥 as f.16)

~~31. F44: idempotencyTracker TTL bound option (default-unbounded decision documented).~~ done — tracker is capacity-bounded (`newIdempotencyTracker(capacity)`)

~~32. F45: tracker eviction + 1M-ID memory-bound test.~~ done — eviction + memory-bound coverage in `metaengine/idempotency_tracker_test.go`

~~33. F51: fix ignored `MarshalMetadataJSON` error (`system/adapter_event_serial.go:31`).~~ done — explicit nil-fallback with ADR-0126 rationale in adapter_event_serial.go

~~34. F50: SQLite durability tier outside WAL guard (`stack/sqlite/preset.go:243`) + Relaxed≠FULL test.~~ done — durability de-nested from `if cfg.WAL`; `WithoutWAL` table + `RelaxedWithoutWAL` tests (TODO_LIST DONE)

~~35. F52: ScanSlice RowCount pre-size; Custom map size hints.~~ open — ScanSlice still cap-64

~~36. F47: benchstat baselines committed; contention benches at `-cpu=16,32`.~~ done — benchstat in benchkit; `-cpu 4,8,16,32` entries in BENCHMARKS.md

~~37. F48: measure-then-pad worker counters @32P (pad only if >10% delta; record either way).~~ done — measured: NO PAD (padded mirror ~58% slower; single-writer) — BENCHMARKS.md:35

~~38. F49: measure-then-pad `multiSeqCounter` + `SSEReplay.seq`.~~ done — multiSeqCounter padded (−61..65%); SSEReplay NO PAD — BENCHMARKS.md:34,36

~~39. F53: pin-drift meta-test (required-vs-latest tags, fail on staleness).~~ done — `8961bb6c3` pin-drift guard; `cmd/api-stability/pin_drift_test.go`

~~40. F54: `system/integration` DuckDB standalone replace-directive/driver guard.~~ done — system no longer imports stack/duckdb (driver name is a config string; go.mod has no duckdb dep)

~~41. F55: `DecorateJournal` for VersionedSeekableJournal (ADR-0126 completion).~~ done — `ca64b3517` `event.DecorateJournal` (ADR-0126); deprecated shells await v5 removal

~~42. F56: singleflight leader-ctx capture fix (`decider/load.go`).~~ done — `9541df676` (F55–F59 sweep)

~~43. F57: turso DSN leak in errors (`register.go:69`).~~ done — `921147a01` turso DSN redaction

~~44. F58: sqliteengine/tursoengine Close() leak.~~ done — `9541df676` Close() ownership fixes

~~45. F59: seq-carrying journal reads — `StreamLogEntry{Seq,Value}` design.~~ done — `a1334d8c5` seq-carrying reads across all engines; flipped IMPLEMENTED today (`f2bbf4621`)

~~46. F60: engine capability conformance test skeleton.~~ done — `30711eb79b` capability audit + conformance skeleton GREEN (09-13)

~~47. F46: go-codec `UnwrapDecode` first-byte sniff (external repo PR).~~ done — go-codec `autodetect.go` first-byte detection

~~48. AGENTS.md gotcha: stale QEMU/port 33070 check before MySQL VM runs.~~ open — port-33070 gotcha not added to AGENTS.md

~~49. AGENTS.md gotcha: `#integration-mysql-vm` default path = GOWORK=off published tags; use
    `-- go test ...` (repo root, workspace mode) to verify uncommitted storage changes.~~ open — `#integration-mysql-vm` path gotcha not added to AGENTS.md

~~50. AGENTS.md gotcha: VM mode (MYSQL_TEST_DSN) shares ONE database — no per-test isolation;
    count assertions must tolerate cross-test state.~~ open — VM single-database isolation gotcha not added to AGENTS.md

## (g) Questions for you

1. **Tag chain approval (blocks 8 standing items):** this session grew the unpublished-API set to
   FOUR modules — `metadata/v4` (new exports), `event/v4`, `storage/v4`, `projectionhost/v4` —
   and standalone (GOWORK=off) builds of command/query/event/pebble/bbolt/stack/mysql stay broken
   until they are tagged and consumers re-pinned. Approve the chain (metadata v4.5.0 → event
   v4.6.1 → storage v4.7.0 → projectionhost patch) + pin bumps, per the plan's F24–F27?
2. **Durability gate (still open from the plan):** tier→sync mapping in storage/pebble +
   metaengine NoSync paths is a real IO win but a behavior change for existing Normal-tier
   consumers on a minor version. Approve for this cycle, or defer to v5?
3. **The pre-existing system-test failure:** `TestSystem_ResetProjection_RestartAndReplay` fails
   at the pre-session baseline (proven in a clean worktree) and a concurrent session has staged
   fixes for it. Should I fold their fix into my gate runs once it lands, or leave that work
   stream entirely to them and keep my verify-fast scoped to my changes?

---

### Environment / resume notes

- `GOTOOLCHAIN=auto` on every go command (host 1.26.5 vs go.work ≥1.26.6); `GOTMPDIR=
  /mnt/buildcache/tmp` for builds/tests; `-tags "goexperiment.jsonv2"`.
- MySQL VM verification of UNCOMMITTED changes: `GOTOOLCHAIN=auto nix run .#integration-mysql-vm
  -- go test -tags "goexperiment.jsonv2" -count=1 ./stack/mysql/...` (the `-- go` form runs from
  repo root in workspace mode; the default path cd's into stack/mysql with GOWORK=off = published
  tags).
- Concurrent session active: DO NOT touch `system/system_hardening_test.go`,
  `system/testdsn_test.go`, `storage/eventstore/event_store_stream.go`,
  `storage/sql/journal_reader.go`, `integration/go.mod`,
  `docs/planning/MADVISE-HUGEPAGE-ANALYSIS.md`, `docs/status/2026-08-16_03-44_*`.
- `/tmp/wt-head` worktree removed after the baseline check.
- Machine: AMD Ryzen AI MAX+ 395, 32 logical cores, linux/amd64.
