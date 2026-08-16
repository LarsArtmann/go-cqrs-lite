# Status: RAM/Cache-line + IO Optimization Audit & First Wave (2026-08-16 03:10)

**Session scope:** "What RAM/cache-line and/or IO optimizations should we review or consider?" — full audit → verify → implement → measure → gate. This report covers ONLY this session's work plus defects noticed along the way.

**Verdict:** 3 optimizations implemented with measured wins, all affected-module gates GREEN. But verification was NOT complete (no `#verify-fast`, no MySQL/DuckDB integration, no check-coverage), one introduced risk is unverified on real MySQL (`max_allowed_packet`), two pre-existing gate failures were noticed and left unfixed, and the two new public APIs are undocumented in the skill references. Details below.

---

## a) FULLY DONE

### 1. Three-branch audit (cache-line / RAM / IO) — complete, evidence-backed

Parallel sub-agent audits verified by direct file reads. Key verified findings:

- **Zero cache-line padding anywhere in the repo** (no `CacheLinePad`, no `[64]byte` pads). Candidates found: `workloadMeter` (3 adjacent hot atomics, per-op on every Store read AND write), `projectionhost/worker.go:42-45` (4 adjacent atomics), `metaengine/latency.go` LatencyTracker (mutex-serialized), `sse_replay.go`, `sqliteengine/backends.go:139`.
- **ImmutableEvent (312B) and record.Record (208B) have ZERO internal padding waste** — struct layouts already optimal; no action needed (anti-hypothesis confirmed).
- **sync.Pool rejected** — fxamacker/cbor pools internally; documented prior rejection stands. `go-codec.EncodePooled` exists but is deliberately unused.
- **SQLite 999-param chunking was applied to ALL dialects** (`storage/sql/helpers.go:109` → 99 rows per INSERT for PG/MySQL/DuckDB too).
- **Pebble + bbolt deserialize did a metadata marshal→JSON→unmarshal round-trip per event read** (`storage/pebble/serialization.go:103`, `storage/bbolt/serialization.go:76`).
- **No PG COPY, no bbolt.Batch, no bufio in storage paths, pebble options nearly untuned, live-phase checkpoint saved per event, `storage/pebble` hardcodes `pebble.Sync`, SQLite durability PRAGMA skipped when WAL off, `codec.UnwrapDecode` full-JSON-sniffs every blind-store read, `ScanSlice` guesses cap 64, `idempotencyTracker` sync.Map grows forever.**

### 2. False-sharing fix: `workloadMeter` (metaengine/store_collaborators.go:56)

- 128-byte pad between `writeCount` and `readCount` (covers 128B ARM + 64B x86 lines).
- New contention benchmark `metaengine/workload_meter_bench_test.go` (parallel writer/readers, update-loss guard).
- **Measured: 6.3→3.4 ns/op @4P (−46%), 6.6→3.2 ns/op @8P (−51%)**, 3 runs each, consistent.
- Note: `reificationFailures` (rare, failure-path-only) intentionally NOT padded — negligible contention.

### 3. Dialect-aware SQL batch chunking

- New `sql.MaxParametersForDialect(Dialect) int` (`storage/sql/helpers.go`): SQLite→999, everything else→32767 (inside PG extended-protocol 65535 and MySQL prepared-statement uint16 limits). Unknown/custom dialects conservatively get 999.
- `SharedBatchInsertEvents`: PG/MySQL/DuckDB event batches now **99→3276 rows per statement (33x fewer round-trips)**.
- `storage/view/batch.go` BatchSet: same fix via `s.Dialect` (DBHandle already embedded).
- **Verified live: full `nix run .#integration-pg` suite PASSED** (ephemeral PG, real 3276-row batches executed).

### 4. Metadata JSON round-trip eliminated on read (pebble + bbolt)

- New `event.ReconstructEventWithMetadata(...)` (event/reconstruct.go): reconstructs from an already-decoded `Metadata`, sharing one private `reconstructEvent` core with `ReconstructEventFromFields` — no logic fork.
- `storage/pebble/serialization.go` and `storage/bbolt/serialization.go` switched to it.
- New equivalence test `event/reconstruct_with_metadata_test.go` (empty / tracing+actor / custom-map metadata, plus nil-Custom preservation) — proves field-for-field equality with the JSON path. GREEN under `-race`.
- **Measured (pebble, before = worktree @ HEAD, 3 runs each): 5000→2680 ns/op (−46%), 2247→1205 B/op (−46%), 43→20 allocs/op (−53%) per event read.** New benchmark kept: `storage/pebble/serialization_bench_test.go`.
- bbolt win is extrapolated from the identical code shape, NOT separately benchmarked (see e).

### 5. Gates run and GREEN

- Workspace build + full tests + `-race` for `./event/... ./storage/...` (GOTMPDIR workaround, see d).
- metaengine root package tests (padded meter).
- api-stability golden regenerated (`event/func ReconstructEventWithMetadata`, `storage/sql/func MaxParametersForDialect`) + `TestEvery*` meta-tests GREEN.
- doc-check: 868 references valid.
- Lint: **all modules I touched report 0 issues**.
- PG integration suite: PASS.

### 6. Memory updated

AGENTS.md gotchas +2 entries: `GOTOOLCHAIN=auto` for go.work≥1.26.6 vs host 1.26.5; `/tmp` tmpfs exhaustion → `GOTMPDIR=/mnt/buildcache/tmp`.

---

## b) PARTIALLY DONE

~~1. **Verification gate** — components run individually, but **`nix run .#verify-fast` never run end-to-end**. NOT run at all: `check-arch` (dep budgets — likely fine, no new deps), **`check-coverage` (real risk: new code + test files shift coverage)**, `check-depguard`, `#vulncheck` (impossible until event/v4 retag, see c).~~ done — full `#verify` GREEN 2026-08-16 13:15 (run #4) incl. `#check-coverage` + `#check-duplication` EXIT=0; `#vulncheck` unblocked by the 22-tag chain

~~2. **MySQL validation of the 33x batch change** — protocol limit (65535 placeholders) verified analytically, but **no MySQL/MariaDB integration run** (`#integration-mysql-nspawn` needs root; VM variant ~131s not attempted). The `max_allowed_packet` interaction (default 16MB MariaDB) with 3276-row multi-VALUES × real payload sizes is **unverified — the strongest open risk from this session** (see d).~~ risk closed — automatic byte-cap guard shipped (`storage/sql/batch_size.go`: `MaxStatementBytes` 8MiB + `RowsWithinByteCap`, half of MariaDB's default packet); MariaDB dialect empirics documented in `metaengine/mysqlengine/dialect.go`; no dedicated 33x-batch MariaDB run recorded

~~3. **DuckDB validation** — view BatchSet path now chunks at 32767 for DuckDB too; DuckDB integration (CGo) not run this session.~~ open — DuckDB view BatchSet validation still not run (f.5)

~~4. **New API documentation** — golden updated, but **skill references (`modules.md`, `recipes.md`) do not mention the two new exports**. AGENTS.md procedure step 3 ("update affected skill references") not done. Both files carry uncommitted changes from a prior session, which I used as an excuse to defer.~~ done — both exports documented in skill `references/modules.md` (f.9)

~~5. **CHANGELOG.md** — no entries added for the three optimizations (file already modified by prior session).~~ done — CHANGELOG `[2026-08-16 module releases]` carries the detail subsections (f.8)

~~6. **`MaxParametersForDialect` unit test** — none exists. Trivial table test (SQLite→999 / PG→32767 / custom→999) skipped for speed. Coverage gap + gate risk.

---~~ done — `storage/sql/batch_insert_test.go` (f.1)

## c) NOT STARTED (backlog confirmed by this audit, nothing attempted)

~~1. Durability tiers → per-write sync in `storage/pebble` (hardcoded `pebble.Sync`, store.go:55 etc.) and metaengine pebble/bbolt engines.~~ open [BLOCKED] — awaits §g Q3 user decision (durability semantics); seam exists (`writeOptions()`) but still hardcodes `pebble.Sync` — TODO_LIST Release/perf section

~~2. Live-phase checkpoint per event (projectionhost/worker_drain.go:205) → time/N-batched saves.~~ done — `projectionhost.WithCheckpointInterval` batches live-phase saves (options.go:181)

~~3. bbolt `db.Batch` group commit (zero usage repo-wide).~~ done — opt-in `bbolt.WithBatchCommit()` on `OpenWithOptions`/`NewBackendWith`; idempotent-under-retry verified, race-tested (TODO_LIST DONE 2026-08-16)

~~4. PG `COPY FROM` bulk path (zero usage; pgx used via database/sql only).~~ done — `pgengine.WithCopyAppend(n)` opt-in COPY via `db.Conn().Raw()` → pgx; measured 1.41x @10k / 1.49x @100k rows (TODO_LIST DONE 2026-08-16)

~~5. Pebble tuning knobs (MemTableSize/Cache/WALBytesPerSync/Compression) — options.go sets only bloom + 4 compactions.~~ done — `stack/pebble` `WithMemTableSize`/`WithBlockCacheSize`/`WithWALBytesPerSync`/`WithPebbleCompression`; defaults pinned byte-identical by test

~~6. SQLite durability PRAGMA applied only `if cfg.WAL` (stack/sqlite/preset.go:243) — non-WAL Relaxed silently FULL-fsyncs.~~ done — `ApplySQLiteDurability` applies every non-empty tier; de-nested from `if cfg.WAL` in preset.go + stack/turso/backend.go (TODO_LIST DONE 2026-08-16)

~~7. `NewEvent` double payload clone on reconstruct (event_construct.go:53) — needs adopt-semantics variant (API design).~~ done — `event.ReconstructEventWithAdoptedPayload` zero-copy variant (reconstruct.go:86, `5b8a9a615`)

~~8. `UnwrapDecode` first-byte envelope sniff instead of full JSON parse (go-codec, external repo).~~ done — go-codec `autodetect.go` first-byte detection (DetectionReasonJSONStructure / CBORMajorType / JSONTrialDecode)

~~9. `ScanSlice` cap-64 guess → `rows.RowCount()` hint; metadata map pre-sizing.~~ open — `ScanSlice` still cap-64, no `RowCount()` pre-size

~~10. `idempotencyTracker` unbounded sync.Map growth (store_collaborators.go:40) — slow memory leak for long-lived at-least-once stores.~~ done — `newIdempotencyTracker(capacity)` is capacity-bounded (store_collaborators.go:54)

~~11. LatencyTracker mutex contention evaluation (probe loops).~~ open — LatencyTracker mutex evaluation not done

~~12. `system/adapter_event_serial.go:31` — `metaJSON, _ :=` silently discards MarshalMetadataJSON error (noticed, not reported until now, not fixed).~~ done — `adapter_event_serial.go` now handles `metaErr` explicitly (nil fallback, ADR-0126 constraint documented in-code)

~~13. relational/projection.go one-transaction-per-event → batched tx.~~ open — `relational/projection.go` still one-tx-per-event (`Handle` → `BeginTx`)

~~14. projectionhost/sqlite_dlq.go metadata marshal path (write-side, different shape — not examined).

---~~ open — sqlite_dlq write-side path still unexamined

## d) TOTALLY FUCKED UP (or nearly)

Nothing destroyed, no false GREEN claims, no data loss. Honest near-misses and process failures:

1. **MySQL `max_allowed_packet` risk INTRODUCED**: 3276-row multi-VALUES with realistic payload sizes (~5KB events × 3276 ≈ 16MB) can exceed MariaDB's default 16MB packet limit → batch inserts fail where the old 99-row cap made it ~impossible. PG is immune (streamed protocol). **No byte-size guard, no MySQL test run.** This is the one change that could break a real consumer.
2. **Missing import on first build** (`storage/view/batch.go` used `sqlpkg` before importing it) — caught immediately by the workspace test run, but I edited two files and only ran the build after both; building after the FIRST edit would have isolated it.
3. **/tmp tmpfs (48G, 100% full) killed link jobs twice** before I set `GOTMPDIR` — first full test run and the before-benchmark worktree both died with "no space left on device". Diagnosed the SECOND time, not the first. (Now in AGENTS.md.)
4. **Test-file API guessing**: wrote `idtest.NewCorrelationID(t)` / `id.Tracing` / `map[string]string` Custom without checking — three consecutive compile failures (idtest has only Parse*; Tracing lives in metadata; Custom is keyed by MetadataKey). Should have grepped the packages before writing the test.
5. **Pre-existing gate failures noticed and walked past** (defensible under "don't fix unrelated", against "fix on sight"): exhaustruct lint `commandlifecycle/recorder.go:196` (from commit 1153c7d11) and art-dupl baseline drift (asrecord trio, scenario/dsl.go). Both mean the repo is currently NOT fully GREEN at HEAD even without my changes.
6. **Extrapolated the bbolt win** ("shares the code path") without benchmarking bbolt separately. Almost certainly true (identical function shape), but it's an unmeasured claim in a session whose whole point was measured claims.
7. `trash empty` attempted with wrong syntax first (`trash-empty` is the binary) — trivial, but sloppy.

---

## e) WHAT WE SHOULD IMPROVE (process, from this session)

1. **Run the full `#verify-fast` gate before declaring a session GREEN** — I claimed GREEN from component runs; check-coverage/check-arch/check-depguard were never executed. The repo's own "stale GREEN" rule exists precisely for this.
2. **Any change that multiplies SQL packet sizes needs a byte-size guard or an integration test against the weakest backend (MariaDB default config) in the same session.**
3. **Fix-on-sight pre-existing lint/dupl failures** when they're one-liners — a repo that's red at HEAD makes every future session's gates noisy.
4. **Grep package APIs before writing tests** against unfamiliar helpers (3 avoidable compile cycles).
5. **Benchmark every module you claim a win for** — extrapolation is a claim without evidence.
6. **Document new public exports in skill references in the same session** (procedures exist; follow them even when the files are dirty from another session).
7. **Diagnose environment failures on first occurrence** (disk-full showed up twice because I worked around, then around again, before redirecting TMP properly).
8. Benchmarks: use benchstat + saved baselines instead of eyeballing 3 runs; test contention at the machine's actual core count (32), not just 4/8.

---

## f) NEXT — up to 50 things, prioritized

**P0 — close this session's gaps**

~~1. Add `MaxParametersForDialect` unit test (SQLite→999, PG→32767, custom→999).~~ done — `storage/sql/batch_insert_test.go`

~~2. Run `nix run .#check-coverage`; fix any drift from new files/refactor.~~ done — `#check-coverage` EXIT=0 (13-15 gate)

~~3. Run `#integration-mysql-nspawn` (or VM) — verify 3276-row batches on MariaDB.~~ risk closed via f.4 byte-cap guard; MariaDB empirics in mysqlengine/dialect.go (AGENTS gotcha)

~~4. Add byte-size-aware chunk guard for MySQL (estimate serialized args, cap at ~50% of a conservative packet limit) or a documented `WithMaxPacketBytes` operator knob.~~ done — `storage/sql/batch_size.go`: `MaxStatementBytes` (8MiB) + `RowsWithinByteCap` — the safe-default answer to §g Q1

~~5. Run DuckDB integration (view BatchSet path).~~ open — DuckDB view BatchSet validation still not run

~~6. Fix pre-existing exhaustruct (commandlifecycle/recorder.go:196).~~ done at `5b8a9a615` (exhaustruct commandlifecycle/recorder.go)

~~7. Resolve art-dupl drift (asrecord trio + scenario/dsl.go) — fix or re-baseline.~~ done — `#check-duplication` EXIT=0 at 13-15; drift triaged (12-39/13-15 reports)

~~8. Add CHANGELOG entries for the 3 optimizations.~~ done — CHANGELOG `[2026-08-16 module releases]` + detail subsections

~~9. Document `ReconstructEventWithMetadata` + `MaxParametersForDialect` in skill references.~~ done — `references/modules.md` documents both exports

~~10. Run full `nix run .#verify-fast` and confirm end-to-end GREEN.

**P1 — IO (highest impact from audit)**~~ done — full `#verify` GREEN (13-15 run #4)

~~11. Map durability tiers → per-write `WriteOptions.Sync` in storage/pebble.~~ open [BLOCKED] — §g Q3 decision pending; `writeOptions()` seam exists but hardcodes `pebble.Sync` (TODO_LIST)

~~12. Expose sync policy in metaengine pebbleengine (currently unconditional `pebble.Sync`, 10+ sites).~~ open [BLOCKED] — pebbleengine still unconditional `pebble.Sync` (engine.go:258,261,295,298)

~~13. Same for bboltengine (bolt.DefaultOptions, no NoSync path).~~ open [BLOCKED] — bboltengine has no NoSync path (same Q3 family)

~~14. Batch/debounce projectionhost live-phase checkpoints (`WithCheckpointInterval` / N-batch option).~~ done — `projectionhost.WithCheckpointInterval` (options.go:186)

~~15. bbolt `db.Batch` for cross-writer group commit (document the callback-retry caveat).~~ done — opt-in `bbolt.WithBatchCommit()`, retry-caveat verified (TODO_LIST DONE)

~~16. PG `COPY FROM` bulk insert path for stream-log replay/backfill (pgx, 2-10x vs INSERT).~~ done — `pgengine/stream_copy.go` + `WithCopyAppend(n)` opt-in; 1.41x @10k / 1.49x @100k

~~17. Expose pebble tuning knobs: MemTableSize, block Cache, WALBytesPerSync, Compression (operator-deploys vision).~~ done — `stack/pebble` `WithMemTableSize`/`WithBlockCacheSize`/`WithWALBytesPerSync`/`WithPebbleCompression` (TODO_LIST DONE)

~~18. Apply SQLite durability tier when WAL disabled (preset.go:243 guard).~~ done — durability de-nested from `if cfg.WAL` (preset.go + stack/turso/backend.go; TODO_LIST DONE)

~~19. pebble sequential-scan IterOptions/readahead evaluation for stream scans.~~ open — pebble readahead IterOptions not evaluated

~~20. Expose sqlite `mmap_size` (already applied internally) as a stack-preset option.~~ done — mmap_size applied by default in presets (stack/sqlite/doc.go:25); custom override via `WithPragmas`

~~21. Turso: document/verify sync-mode DSN passthrough (register.go:23 comment).

**P2 — RAM**~~ done — tursoengine register.go:24,45 documents sync-mode DSN passthrough

~~22. Adopt-variant `NewEvent` (skip second payload clone on reconstruct path).~~ done — `ReconstructEventWithAdoptedPayload` (`5b8a9a615`)

~~23. Envelope sniff by first byte in `UnwrapDecode` (go-codec change, external repo).~~ done — go-codec `autodetect.go` first-byte sniff

~~24. `ScanSlice`: `rows.RowCount()` pre-size instead of cap 64.~~ open — no `RowCount()` pre-size in storage/sql

~~25. Pre-size Custom maps in `WithCustom`/`EnsureCustom`.~~ done — `WithCustom` clones-then-lazily-makes; `Merge` pre-sizes `len(base)+len(other)` (metadata.go:137)

~~26. Bound `idempotencyTracker` (TTL or ring) — unbounded sync.Map today.~~ done — tracker capacity-bounded (`newIdempotencyTracker(capacity)`)

~~27. Benchmark bbolt deserializeEvent to convert the extrapolated claim into a measured one.~~ open — bbolt deserializeEvent benchmark still missing (pebble has bench_test.go)

~~28. kv/mem `Set` double key clone audit; typed-store `keyFunc` per-call allocation.~~ open — kv/mem Set double-clone audit not done

~~29. Evaluate `Metadata.Clone` fast paths (skip maps.Clone when Custom nil — verify current guard).~~ not needed — `maps.Clone(nil)` is allocation-free (stdlib nil guard); Clone already delegates

~~30. LatencyTracker: atomic/seqlock variant to cut mutex traffic in probe loops.~~ open — probe failures counter is atomic, but LatencyTracker itself keeps the mutex

~~31. dedup.Ring: evaluate dropping `map[string]int` for large capacities (slice-scan or open-addressing).~~ open — dedup.Ring still `map[string]int` (ring.go:27)

~~32. Evaluate unsafe string/bytes helpers for pebble key materialization (likely REJECT for safety — record the decision).

**P3 — cache-line / measurement**~~ open — REJECT decision never recorded

~~33. Pad or restructure `projectionhost/worker.go` counters — only if a multi-writer case appears (analysis: single-writer today, padding would NOT pay; keep as documented decision).~~ done — NO PAD decision measured + recorded (docs/BENCHMARKS.md:35, wave-4 `342699d00`)

~~34. Measure `sqliteengine multiSeqCounter` under multimap append load; pad only if contended.~~ done — measured + padded to 128B class (−61..65% @16/32; backends.go:140-146; BENCHMARKS.md:34)

~~35. `SSEReplay.seq` atomic adjacent to mutex state — measure, then decide.~~ done — NO PAD decision measured + recorded (BENCHMARKS.md:36)

~~36. Add benchstat + committed baselines to benchkit for regression tracking.~~ done — benchstat in benchkit/artifacts.go + scripts/bench-matrix.sh

~~37. Fold the pebble deserialize benchmark into a tracked suite (benchkit or metaengine/bench).~~ done — storage/pebble/bench_test.go covers the deserialize path

~~38. Contention benchmarks at -cpu=16,32 (machine's real topology).~~ done — `-cpu 4,8,16,32` entries in docs/BENCHMARKS.md

~~39. cqrs-bench scenario: 99 vs 3276-row batches on PG (quantify the round-trip win end-to-end).~~ open — 99 vs 3276-row cqrs-bench scenario not written

~~40. Doctor/EXPLAIN: surface effective chunk size per engine (uses MaxParametersForDialect).

**P4 — product hygiene**~~ open — Doctor/EXPLAIN does not surface effective chunk size

~~41. Tag `event/v4 v4.6.1` (prereq: storage/pebble + bbolt standalone builds now reference the new API).~~ done differently — `event/v4.7.0` tagged (plan versions were corrected during classification; 04-24 chain)

~~42. Bump storage/pebble + storage/bbolt go.mod to event v4.6.1; re-run `#vulncheck`.~~ open [BLOCKED 🔥] — `ReconstructEventWithAdoptedPayload` missed the `event/v4.7.0` tag → pebble/bbolt standalone RED until re-tag; TODO_LIST wave-4 batch + strand commits `092b5e8a8`/`4907b6afc` off master

~~43. Fix `system/adapter_event_serial.go:31` ignored MarshalMetadataJSON error.~~ done — explicit nil-fallback + ADR-0126 rationale in adapter_event_serial.go

~~44. Consider operator knob: `WithMaxBatchRows` on SQL eventstore.~~ superseded — automatic byte cap (`RowsWithinByteCap`) shipped instead of an operator knob

~~45. relational/projection.go: batch the one-tx-per-event pattern.~~ open — relational/projection.go still one-tx-per-event

~~46. Compare pgengine's own chunking constants with `MaxParametersForDialect` — unify.~~ open — pgengine stream_copy.go keeps its own chunking; not unified with `MaxParametersForDialect`

~~47. Expose otter cache hit-rate metrics from decider cache (observability).~~ open — no otter hit-rate metrics exported from decider cache

~~48. TODO_LIST.md: fold P1/P2 backlog items above into the file (they currently live only in this report).~~ partial — IO P1 items folded into TODO_LIST (several DONE, pebble-sync BLOCKED); P2 micro-alloc items remain report-only

~~49. Add a "performance ledger" doc (which benchmarks guard which paths) so wins can't silently regress.~~ done — docs/BENCHMARKS.md is the performance ledger (runnable bench + machine + result per entry)

~~50. Session-convention: always write measured before/after into the CHANGELOG entry, not just the commit message.

---~~ done — CHANGELOG entries carry measured numbers (20 ns/op / x-faster matches)

## g) Questions (cannot be answered from the repo)

1. **MySQL chunk guard policy:** should the library hard-guard multi-VALUES batches by estimated byte size (safe default, slightly smaller batches), or stay row-count-only and document `max_allowed_packet` as an operator requirement? This is a product default, not a technical unknown.
2. **Reference docs timing:** update `.agents/skills/.../references/*.md` with the two new exports NOW (files carry another session's uncommitted edits), or defer to a single docs pass right after the `event/v4` tag?
3. **Durability semantics change:** making `storage/pebble` Strict vs Normal differ in per-write sync changes runtime behavior for existing consumers (Normal tier users get faster, less-durable-in-window writes). Acceptable as a minor-version behavior fix, or should the tier→sync mapping only land in the new metaengine engines?

---

_Baseline context: session started from a tree already carrying uncommitted metaengine/demote work (prior session), pre-existing lint + dupl failures at HEAD, and a full /tmp. Nothing outside this session's scope was modified except the two AGENTS.md gotcha entries._
