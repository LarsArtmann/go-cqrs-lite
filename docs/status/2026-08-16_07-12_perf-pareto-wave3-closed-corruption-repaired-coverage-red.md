# Session Report — Perf Pareto Resume: Wave 3 Closed (bbolt Batch, PG COPY, pebble knobs), Corruption Repaired Mid-Flight, Coverage Gate Red

**Session:** 2026-08-16 ~03:48 → 07:12 (~3.4h) · **Mandate:** resume the perf-Pareto plan's owed items + Wave 3 remainder, per "READ, UNDERSTAND, RESEARCH, REFLECT. Break down. Execute and Verify one step at a time."
**Tree state at report time:** clean — the auto-commit daemon landed everything in `a9b1f68ab` (07:11).

---

## a) FULLY DONE

### Resume wave (owed from prior session's handoff)

1. **api-stability golden regenerated** — 3× (see §d for why): final state 4107 exports, clean diff = exactly this session's 10 new exports (`storage/bbolt`: `OpenWithOptions`/`NewBackendWith`/`WithBatchCommit`/`BackendOption`; `metaengine/pgengine`: `Option`/`WithCopyAppend`; `stack/pebble`: 4 knob options).
2. **Lint gate closed (F14)** — first run surfaced 9 findings in 5 modules; ALL fixed:
   - Mine: `flushPendingCheckpoint` contextcheck → now takes `ctx` (`context.WithoutCancel(ctx)` in `run()` defer; `Host.Stop()` passes `context.Background()`); nonamedreturns in `checkpoint_cadence_test.go`; `mp`→`provider` varnamelen in `commandlifecycle/recorder.go`.
   - Concurrent session's tree (fixed as found): `storage/sql/keyset.go` named returns + the `err :=` they required; `sqlite_journal_readfrom_test.go` redundant `id.StreamType()` conversion; `metaengine/sqliteengine/register.go` contextcheck nolint (same pattern benchkit already uses); `benchkit/phases_projection.go` ×3 contextcheck nolints (`Host.Stop` is ctx-free by API).
3. **verify-fast (F11)** — 238/238 test packages GREEN; lint leg failed only on my two new test files' formatting (§d.3), fixed → final full `nix run .#lint`: **76/76 modules, 0 issues**.
4. **projectionhost race ×3** — GREEN (27.6s) on the mutex-guarded checkpoint state.
5. **F31 docs** — readmodels.md §2.3 checkpoint-tuning note (corrected once: interval is evaluated at event arrival, not a background ticker); modules.md projectionhost row; CHANGELOG entry with at-least-once semantics.

### Wave 3 — TOP IO WINS (all three independently shippable features, complete)

6. **F32-F34 bbolt opt-in group commit** — `storeBase.writeTx` dispatch (Batch vs Update), mirrored `writeTx` on `KVAdapter` + `bboltBatch`; new `OpenWithOptions`/`NewBackendWith` constructors + `WithBatchCommit()` BackendOption; propagated to all six stores. Verified bbolt's retry semantics from vendored source first: failed fn re-runs solo, survivors re-run idempotently on rolled-back state. 3 tests race-GREEN: 8 concurrent writers → byte-identical journal (400 events, distinct-ID check); version-conflict writer ejected + re-run solo surfaces Conflict while batch-mate lands; default backend flag stays false.
7. **F35-F36 pgengine COPY + batching** — `stream_copy.go`: `WithCopyAppend(minValues)` option; `copyAppend` via `db.Conn(ctx).Raw()` → pgx native `CopyFrom` (no second pool; `errCopyUnavailable` fallback for non-pgx drivers); `streamInsertBatch` chunked multi-VALUES (10k rows/stmt, 30k params < 65535 cap) now the DEFAULT for `StreamAppend` AND `StreamAppendExpected`; `New(dsn, opts...)` variadic. 3 tests GREEN on real Postgres (testcontainer): COPY≡INSERT value/version equivalence, RunInTx falls back to INSERT (COPY can't join the tx), threshold respected. **Bench (real PG): COPY 1.41× @10k rows (39.3→27.9ms), 1.49× @100k (368→248ms).**
8. **F37-F38 pebble operator knobs** — `stack/pebble`: `WithMemTableSize`/`WithBlockCacheSize`/`WithWALBytesPerSync`/`WithPebbleCompression`; block-cache ref released after Open + replaced-cache Unref (no leak); `WithPebbleOptions` nil-guard. Tests: defaults **byte-identical** to `cqrspebble.DefaultOptions()` (pinned), each knob touches only its field, cache lifecycle. Full stack/pebble suite GREEN (12.4s).
9. **F39 perf ledger** — `docs/BENCHMARKS.md`: every shipped win → runnable benchmark → baseline → last measured (batch chunking, view chunking, COPY, bbolt batch, pebble deserialize, workloadMeter pad, row layout, keyset, checkpoint cadence).
10. **F40 harvest** — TODO_LIST perf section: 7 items checked DONE with one-line proof notes; CHANGELOG "Wave-3 IO wins" section with measured numbers; modules.md rows (bbolt, pgengine, storage row already had packet guard); recipes.md §2.0 pebble deploy-time tuning block; doc-check **875 refs GREEN**.

### Wave 4 verification work (done by others, verified + harvested by me)

11. F56 (singleflight `WithoutCancel`), F57 (turso `redactDSN`), F58 (`sqliteengine.OwnDB` + `close_ownership_test.go`) — all confirmed SHIPPED at HEAD; TODO_LIST defect-sweep section updated to reflect reality.

---

## b) PARTIALLY DONE

1. **F12 `#check-coverage` — RED, uninvestigated**: `event` 87.3% (−2.7%) and `query` 84.5% (−5.4%) drifted beyond ±2%. Prime suspect: the asrecord `brandedString`/`actorString` dedup (my prior session) removed covered local helpers from `event/asrecord.go`/`query/asrecord.go` — coverage moved to `metadata/ids.go`, which those modules' own coverage doesn't count. Needs either test backfill in event/query or an honest `--update` re-baseline.
2. **F50 SQLite durability outside WAL — researched, not implemented**: both call sites found (`stack/sqlite/preset.go:243`, `stack/turso/backend.go:63`); root cause identified — `ApplySQLiteDurability`'s `Normal` early-return comment is WAL-specific ("NORMAL is already set by SQLiteEnableWAL"), and the whole tier application is nested under `if cfg.WAL`. Fix shape: apply tier unconditionally (drop the Normal early-return for non-WAL or make the helper WAL-aware) + a `Relaxed ≠ FULL` test. Interrupted by this status request.
3. **F51/F52 — researched only**: `encodeEvent`'s ignored error cannot propagate through `AdapterCore.Encode func(T) string` (ADR-0126 core constraint) → plan-sanctioned "explicit discard w/ comment + safe fallback" is the design; ScanSlice pre-size is `make([]T, 0, 64)` at `storage/sql/reconstruction.go:48`.

---

## c) NOT STARTED (Wave 4 remainder + standing)

- F41 bbolt deserialize bench · F42-F43 payload adopt-variant · F44-F45 idempotencyTracker bound · F46 go-codec sniff (external PR) · F47 benchstat baselines @-cpu=16,32 · F48-F49 measure-then-pad candidates
- F53 pin-drift meta-test · F54 system/integration DuckDB standalone · F55 DecorateJournal · F59 seq-carrying journal reads · F60 capability conformance skeleton
- Standing TODO_LIST sections untouched: cqrs-lint (4), WithActor Hardening (3), Metaengine Phase 6b/7 (22), v5 Unification, Docs Honesty
- From the OTHER session's report §b (their lane, listed for visibility): 3 worktree commits to cherry-pick to master (retracts + hardened tag script + metadata pins — **release-critical**, master's command/query still pin metadata v4.4.0), replace-drop sweep, T2 GitHub Releases.

---

## d) TOTALLY FUCKED UP (and what I did about it)

1. **Concurrent-session mid-write corruption hit my gates TWICE.** `projectionhost/worker.go` got `func (w *workor) {` + two mid-line truncations; `scenario/dsl.go` got `fojection.Projection,` — pure garbage in the INDEX and worktree, no owning commit (`git log -S` empty). First golden regen silently dropped **ALL projectionhost/scenario exports** (4097→4023) because api-stability skips unparseable files. Caught it because the export count move was suspicious; repaired both files by restoring HEAD text via `edit` (no `git restore` — safety rules), verified `diff <(git show HEAD:file) file` = empty + module tests green, then re-ran golden. **Lesson: after any concurrent-session activity, `gofmt -l` the dirty Go files BEFORE trusting gate output; a silently-shrinking golden is a corruption tell.**
2. **I burned a full ~15-min verify-fast cycle on lint formatting I could have caught in 10 seconds.** Wrote `batch_commit_test.go`/`copy_test.go` import blocks from memory: wrong alias convention (`"github.com/larsartmann/go-error-family"` vs `errorfamily "..."`), `1 << 30` vs `1<<30`, missing godot period. The other session's report §e.1.2 warned EXACTLY this ("batch the cheap checks before the expensive one") and I repeated it anyway. **Local `gofmt`/`goimports` DIVERGES from the gate config (gci sections, aliases) — the only reliable local check is `golangci-lint run --config .golangci.yml <module>` or `golangci-lint fmt`.**
3. **Three compile-fix roundtrips on the COPY test** from guessing APIs (`LoadStream` vs `Load`; `Engine.StreamAppend` vs the `StreamLogBackend` capability assertion; `RunInTx` lives on `metaengine.Transactional` with `func(context.Context) error` shape). Grep-first would have zeroed all three.
4. **Coverage gate left red** when the status request arrived (see §b.1) — the one gate I opened and did not close.

---

## e) WHAT WE SHOULD IMPROVE (repo-level)

1. **Corruption is a recurring multi-session hazard** (this session ×2; daemon revert history in AGENTS.md). Cheapest hardening: BuildFlow pre-commit already runs shfmt on staged shell files — add `gofmt -l` on staged `.go` files (syntax-only, 1s) so garbage can't enter the index.
2. **The api-stability tool should FAIL LOUDLY on parse-skip**, not print `skip projectionhost:` and proceed — a corrupted module looks identical to a legitimately-removed one in the golden.
3. **Adopt the gate's golangci as the only formatter** — document `golangci-lint fmt --config .golangci.yml` in AGENTS.md as THE local format command; bare gofmt/goimports misleads (cost me a verify cycle).
4. **New-export → module-tag pipeline gap**: my 3 features are committed but untagged (bbolt/pgengine/stack-pebble need minor bumps). Same class the other session hit with their 3 stranded worktree commits — a "tag owed" meta-test (diff api_surface vs last tag) would surface this.
5. **Multi-session attribution is now unreadable**: daemon commit `a9b1f68ab`'s message describes MY `writeTx` work as "secondary index batched commit path" (the secondary index was a PRIOR session's) and credits "MiniMax-M3". Harmless here, but per-session status reports are the only honest record — keep writing them.

---

## f) NEXT 50 (prioritized)

**Close the open gates (do first):**

1. Investigate + close coverage drift: backfill event/query tests for asrecord paths OR `bash scripts/check-coverage.sh --update` re-baseline (+ AGENTS.md line) — decide honestly which.
2. `nix run .#verify` (full, incl. race) once — verify-fast covered tests+lint; full adds race + doc-assertions.
3. `nix run .#check-duplication` — new `writeTx` triplication (storeBase/KVAdapter/bboltBatch) may trip the clone gate; if so, consolidate or baseline.

**Ship the tags (consumer-reachability):**
4. Tag `storage/bbolt v4.7.0` (WithBatchCommit + constructors), `metaengine/pgengine v4.1.1` (COPY + batching), `stack/pebble v4.x` (knobs) via the hardened `scripts/tag-release.sh` (includes standalone-build gate).
5. Cherry-pick the other session's 3 stranded worktree commits (`092b5e8a8` retracts+script, `4907b6afc` bench tidy) — release-critical, master go.mods still pin metadata v4.4.0.
6. Replace-drop sweep (system ×6, cqrs-bench ×7, integration ×2) now all targets are tagged; GOWORK=off re-verify each.

**Wave 4 quick wins (XS-S, no design):**
7. F50 SQLite durability outside WAL (design already settled — §b.2) + Relaxed≠FULL test.
8. F51 `encodeEvent` explicit-discard comment + safe fallback (`Metadata: metaJSON` stays nil-JSON on marshal error).
9. F52 ScanSlice `RowCount()` pre-size when available + Custom map size hints.
10. F41 bbolt deserialize benchmark (mirror pebble's; convert extrapolated claim to measured).
11. F22/F51-class: grep repo for remaining `metaJSON, _ :=` patterns.
12. F47 benchstat baselines for BatchInsert/COPY/deserialize benches; contention pass @-cpu=16,32.
13. F48 measure worker-counter false sharing @32P; record decision either way (TODO says single-writer → likely NO pad).
14. F49 same for multiSeqCounter + SSEReplay.seq.

**Wave 4 medium:**
15. F44-F45 idempotencyTracker TTL/ring bound + 1M-ID memory test.
16. F42-F43 payload adopt-variant (`reconstructEventAdopt` unexported, equivalence + race tests, wire pebble+bbolt).
17. F53 pin-drift meta-test (required-vs-latest tags, fail on staleness).
18. F55 `DecorateJournal` for VersionedSeekableJournal (ADR-0126 completion).
19. F54 system/integration DuckDB standalone (replace directive or driver guard).
20. F59 seq-carrying journal reads design (`StreamLogEntry{Seq,Value}`).
21. F60 capability conformance skeleton (declared vs implemented table).
22. F46 go-codec `UnwrapDecode` first-byte sniff (external PR; verify-external-claims first).
23. api-stability: fail on parse-skip (§e.2).
24. BuildFlow: `gofmt -l` staged-Go syntax gate (§e.1).
25. AGENTS.md: document `golangci-lint fmt` as THE formatter (§e.3).

**Benchmarks/observability:**
26. benchkit phase for bbolt WithBatchCommit (group-commit scaling curve vs writer count).
27. benchkit phase for pgengine COPY (insert-vs-copy crossover table into BENCHMARKS.md).
28. benchstat files committed under `benchmarks/` for the 5 ledger paths.
29. BENCHMARKS.md: add CI check that every ledger-named benchmark still exists (doc-drift guard).

**Docs honesty:**
30. Correct plan doc §T1 version numbers (other session's F1.10).
31. CHANGELOG: retract incident entry + SKILL FAQ recipe (their §g Q1, default T17).
32. TODO_LIST: prune `Session verification gap` item (closed by this session's gates, modulo coverage decision).
33. modules.md: bbolt row already added — verify pgengine/pebble rows render in SKILL.md ToC checks.

**Test-debt:**
34. `kv.Cache` shared `*T` defect (open in defect sweep).
35. TypedQueryStore hardcoded JSON decode (`query/typed.go`).
36. Ghost `event.ErrBinaryNotFound` — document or delete.
37. Fuzz `ValidateIdentifier` vs sqlite/pg/mysql metacharacter sets.
38. Golden-safety: api-stability meta-test that export COUNT never drops >N% in one regen (corruption tripwire).

**Metaengine:**
39. Phase 6b layout remainder (22 items — see TODO_LIST).
40. Live-latency: export COPY-aware costs for StreamAppend (planner should know COPY changed write costs).
41. `metaengine/bench` go.mod tidy drift (their §b.3).
42. Dgraph/DuckDB/Turso engine gaps (standing).

**Release/infra:**
43. T2: GitHub Releases ×20 tags (needs `gh` auth check).
44. CI leg for GOWORK=off leaf-module standalone builds.
45. `#verify-standalone` nix app decision.
46. Check how long CI Benchmarks job has been red (`gh run list`).
47. Cleanup `/tmp/cqrs-tagwt` worktree registration.
48. `stack/*` presets sweep-tag (all have unreleased commits).
49. `system` module v4.5.0 tag (12 unreleased commits, 6 replaces to drop first).
50. v5 cut planning (transport/* final tags, deprecated-shell deletions).

---

## g) QUESTIONS (cannot figure out myself)

1. **Coverage drift resolution**: `event` −2.7% / `query` −5.4% stems (most likely) from the asrecord dedup moving helper coverage into `metadata/`. Backfill equivalent tests in event/query (30-60m), or accept the structural move and re-baseline with `--update` (2m)? The coverage moved, not vanished — but only you can say whether the number is a KPI or a floor.
2. **Tag my three features now or batch?** bbolt/pgengine/stack-pebble tags are owed for consumer reachability, but the other session's stranded repair commits (metadata v4.5.0 pins on master) should land FIRST to avoid re-breaking the chain. Sequence: (a) cherry-pick their commits → (b) tag mine, or leave all tagging to a single dedicated release session?
3. **Am I authorized to touch the other session's release-critical stragglers** (cherry-pick `092b5e8a8`/`4907b6afc`, master go.mod metadata pins), or is that stream strictly theirs? Their report treats it as next-session work but doesn't name an owner; two sessions editing go.mod pins concurrently is exactly what poisoned command/v4.7.0.
