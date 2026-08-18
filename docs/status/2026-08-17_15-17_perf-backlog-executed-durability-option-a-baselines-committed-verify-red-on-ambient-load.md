# Status: Perf-Backlog EXECUTED — Durability Option A Landed + False-Sharing Baselines Committed; Verify RED on Ambient Load (2026-08-17 15:17)

**Session scope:** Continuation of
[`2026-08-17_14-23_perf-backlog-review`](2026-08-17_14-23_perf-backlog-review-durability-decision-falsesharing-baselines.md).
The resumed session was instructed to decide the three §g questions autonomously (§h there),
then execute and verify both backlog items one step at a time. This report covers that execution.

**Verdict:** Both TODO_LIST items are **DONE and committed** (durability tier mapping Option A +
benchstat baselines). Module-level tests GREEN. The full `nix run .#verify` gate is **RED on
ambient load**: 9 benchkit timing tests + 1 system/v4 test timed out at ~114-121s while machine
load exploded 3→75 (5-min avg) mid-run with 40 users — **every module this session touched
passed inside that same run**. Verify needs one calm-window re-run before any repo-wide GREEN
claim. No code work remains from the backlog.

---

## a) FULLY DONE

1. **Item 2 — benchstat false-sharing baselines (XS): committed 100%.**
   - Tie-breaker run (g.3 decision: re-run ONLY the anomalous suite): SSEReplaySeq at load ~3.3
     gave adjacent 75.25n ±2% @16 / 77.88n ±17% @32 vs padded 81.42n ±6% / 78.72n ±12% —
     padded +8% SLOWER @16, tied @32. Across three runs there is no reproducible >10% padded
     win; mechanism argument (`record()` pulls both cache lines regardless of layout) holds.
     **NO-PAD confirmed and closed.**
   - `benchmarks/2026-08-17_falsesharing-{sqliteengine,projectionhost,metaengine}.txt` committed
     (the two tight suites from the morning runs; the metaengine file is the tie-breaker run —
     tightest variance of its three runs).
   - Evidence doc (`docs/benchmarks/2026-08-16_false-sharing-contention.md`): tie-breaker outcome
     recorded, "check uptime first + workspace root" protocol hardening in the header, new
     "Committed benchstat baselines" section with all benchstat tables, re-run commands, and the
     NOT-CI-gated rationale (g.2 = evidence-doc only; relative padded-vs-adjacent benches that a
     median gate cannot express, ±56%-under-load cell would flake it).
   - `docs/BENCHMARKS.md`: three micro-path rows updated with baseline pointers + tie-breaker
     result; gate section documents the diagnostic-vs-gate split and the canonical benchstat
     install (`go install golang.org/x/perf/cmd/benchstat@latest` — f.16 resolved by
     documentation; nixpkgs has no attr).
   - TODO_LIST item 2 ticked; AGENTS.md GOWORK gotcha extended (protocol baselines run from
     workspace root; per-module numbers non-comparable; record uptime) — f.17, f.18 done.

2. **Item 1 — durability tier→per-write-sync mapping, Option A (M): implemented, tested, documented.**
   - **§g decisions recorded** (in §h of the prior report): g.1 = bbolt Normal ≡ Strict
     (documented exception; bbolt has no WAL — upstream calls NoSync dangerous; preset keeps
     defaulting Strict so the no-op alias can never silently become a durability drop);
     g.2 = evidence-only baselines; g.3 = tie-breaker-only re-run.
   - **Naming review (the user's explicit gate) applied** per the naming-review skill checklist:
     `pebble.WithBackendAsyncWrites` (+ `BackendOption`) extends the existing
     `With{Command,Query,Snapshot,Checkpoint}AsyncWrites` family (bare `WithAsyncWrites` is
     taken by StoreOption); `pebbleengine.WithAsyncWrites` mirrors storage/pebble for the same
     mechanism; `bboltengine.WithNoSync` is deliberately NOT "async writes" — bbolt has no WAL,
     the name must not imply Pebble-equivalence. Recorded as §h.4.
   - **storage/pebble**: `BackendOption` + `WithBackendAsyncWrites()` threaded through
     `Open`/`NewBackend`/`newBackend` — flips all five record stores AND the shared read-model
     KV store to async; default unchanged (incl. the read-model KV store's historical sync mode).
   - **stack/pebble**: `tierToSettings` maps Strict→sync (also for unknown tiers — safest
     interpretation), Normal→async WAL, Relaxed→DisableWAL+async. **Latent bug fixed**: old
     Relaxed set DisableWAL while stores still wrote with Sync — with the WAL disabled that
     degrades to a memtable flush per write (the slowest Pebble path) inside the tier that
     exists for speed.
   - **stack/durability.go**: doc split brain fixed (line 42 "default flush behaviour" and
     preset.go:38 "same as Strict" both contradicted reality); all four Pebble rows now match
     the implementation; bbolt rows added incl. the Normal≡Strict exception and why the bbolt
     preset defaults Strict.
   - **metaengine/pebbleengine**: `Option` + `WithAsyncWrites()`; 15 hardcoded `pebble.Sync`
     sites (engine.go ×10, stream_log.go ×2, vector.go ×3) replaced by one `writeOptions()`
     seam mirroring storage/pebble's `storeBase`; both `NewPebbleEngine` and
     `NewPebbleEngineFromDB` accept options (per-write sync applies cleanly to a wrapped DB).
   - **metaengine/bboltengine**: `Option` + `WithNoSync()` (sets NoSync + NoFreelistSync on a
     COPY of bolt defaults — the shared global is never mutated; tested); not offered on
     `NewBboltEngineFromDB` (NoSync is fixed at bolt.Open time).
   - **Tests** (4 new files, all GREEN): backend flag-threading tests incl. read-model KV
     store polarity, tier→settings table (incl. unknown-tier fallback), engine option unit +
     functional roundtrip tests, boltOptions mapping + global-mutation guard.
   - **Plumbing**: api-stability golden regenerated (6 new exports, verified diff);
     CHANGELOG `[Unreleased]` Changed + Added entries (honesty gate GREEN after rewording
     third-party `pebble.Sync`/`bolt.DefaultOptions` prose that regex-matched as citations);
     skill `references/modules.md` pebble + pebbleengine rows updated; doc-check 921 refs
     GREEN; TODO_LIST item 1 ticked with honest PENDING-bench note.
   - **Throughput bench added** (`storage/pebble/durability_bench_test.go`:
     `BenchmarkEventAppendSync`/`Async`) — disk-backed (writes under
     `~/.cache/pebble-durability-bench`, `PEBBLE_DURABILITY_BENCH_DIR` override, skips when
     unavailable) because tmpfs would erase the fsync cost being measured.

## b) PARTIALLY DONE

1. **`nix run .#verify` — EXIT=1 at the TEST phase, load-induced.** The run started at load ~4
   and the machine exploded to **38.5/75.8/63.6** (1/5/15-min averages) with 40 users mid-run.
   Casualties, all ~114-121s timeouts: 9 benchkit timing tests (`TestRun_SQLite*`, `TestRun_Pebble`,
   `TestMixedWorkload_SQLite`, `TestRun_AnalyticalJournalScans`, `TestRun_Recovery_Pebble`,
   `TestRun_ClosedStore_ErrorMessage`, …) and `system/v4` `TestSystem_ResetProjection_RestartAndReplay`.
   These are the exact load-sensitive classes AGENTS.md warns about (benchkit Duration=10ms abort
   bound; system test is a 120s timeout). **Inside the same run**: storage/pebble ✓ (252s under
   load), stack/bbolt ✓, stack/pebble ✓, metaengine/bboltengine ✓ (236s), metaengine/pebbleengine ✓,
   scheduling/sqlstore ✓ (foreign module). Race/lint/doc phases were never reached — so the
   foreign session's 2 known lint findings in `scheduling/sqlstore/store.go` (their report:
   "verify-red-on-2-lint") remain unjudged on this tree too. **No repo-wide GREEN claim made.**
2. **Strict-vs-Normal throughput numbers: PENDING.** First measurement attempt (200×5) showed
   sync ≈ async ≈ 2.5 ms/op — a raw `db.Set(..., nil)` isolation bench ALSO measured ~2.5 ms/op,
   proving the device queue itself is saturated (root btrfs 96% full + ambient load), not a code
   issue. Recorded honestly as PENDING in `docs/BENCHMARKS.md`; the scratch isolation bench was
   deleted after diagnosis (its deletion is in the worktree awaiting the daemon).

## c) NOT STARTED

1. **Calm-window `#verify` re-run** (load < ~5): expected to clear the 10 timeout failures; will
   also finally run the lint phase (foreign scheduling/sqlstore findings + my files).
2. **Calm-window durability bench** to fill the PENDING BENCHMARKS.md cell (same conditions).
3. `modules.md` has no `metaengine/bboltengine` row (only pebbleengine among engines I touched
   was listed) — optional table addition, cosmetic.

## d) TOTALLY FUCKED UP (or nearly)

1. **Daemon merge event mid-session.** At 14:59 the auto-commit daemon committed my in-flight
   work as `cd3db3c82` MERGED with a concurrent session's edits to the SAME files
   (pebbleengine engine/stream_log/vector, bboltengine engine, and even my own options_test.go),
   under a commit message claiming work that is NOT in the commit's stat (e.g. "MultiSeq counter
   sizes reduced 12→8 bytes" — no sqliteengine file appears in the diffstat; "SyncWAL option" in
   stack/bbolt — no such symbol exists). Response: re-verified the merged tree immediately —
   build ✓, all 6 touched modules' tests ✓, all durability seams intact (15 writeOptions sites,
   4 option constructors, tierToSettings). Lesson: **with the daemon + live concurrent sessions,
   every GREEN claim has a shelf life — re-verify after any daemon commit before extending it.**
2. **Nearly shipped a lying option.** My first engine.go multiedit draft contained
   `cfg.syncWrites = true && false` (polarity error that would make `WithAsyncWrites` a no-op).
   The edit happened to fail on file-freshness; the corrected version landed. Root cause:
   drafting the option away from the config struct instead of next to its default.
3. **Test-file sloppiness (caught by build):** referenced a nonexistent helper
   (`openRawMemDB`) instead of checking `helper_internal_test.go` first; called `MapSet` on the
   `metaengine.Engine` interface (method lives on the concrete engine — assert internally).
4. **`var _ = fmt.Sprintf` placeholder hack** in the bench file to silence an unused import —
   removed before anything ran; should never have been typed.
5. **CHANGELOG honesty gate × 3:** prose mentions of upstream `pebble.Sync` and
   `bolt.DefaultOptions` regex-matched as `pkg.Symbol` citations and failed the fiction gate.
   Fixed by rewording to plain words rather than extending SKIP_ALIASES (skipping "pebble"
   would weaken the gate for OUR storage/pebble alias). Correct call, but cost 2 gate round-trips.
6. **fsync bench run before checking disk/load conditions.** I checked `df -T` only AFTER seeing
   suspicious sync≈async parity; a 5-second pre-flight (uptime + df) would have saved a bench
   cycle and the scratch diagnosis bench.

## e) WHAT WE SHOULD IMPROVE (process)

1. **Pre-flight for IO-sensitive benches**: `uptime` + `df -T <dir>` before any fsync-adjacent
   measurement; record both next to numbers (the evidence-doc protocol line now says this).
2. **Post-daemon-commit re-verification** is mandatory whenever another session is live —
   the daemon does not segregate sessions and its commit messages are unreliable prose.
3. **Draft option constructors adjacent to their config struct** so defaults and polarity are
   visible in the same view.
4. **Third-party knob names in CHANGELOG prose**: write them as plain words; anything shaped
   `alias.Symbol` will be gate-checked against OUR golden.

## f) NEXT — prioritized

1. **Re-run `nix run .#verify` in a calm window** (load < ~5). Expect the 10 timeout failures to
   clear; the lint phase then judges both my files and the foreign scheduling/sqlstore findings
   (their report owns those fixes if still RED — TODO_LIST "WithActor" backlog, not mine).
2. **Re-run `BenchmarkEventAppendSync/Async` in the same calm window**; fill the PENDING cell in
   `docs/BENCHMARKS.md` (fsync-per-append cost finally made visible) — also finally quantify the
   Normal-tier win the whole backlog item existed for.
3. **Daemon hygiene**: confirm the daemon picks up the worktree deletions/edits (scratch bench
   deletion, CHANGELOG, TODO_LIST, api_surface, skill refs, BENCHMARKS.md).
4. Optional: add the missing `metaengine/bboltengine` row to `references/modules.md`.
5. Nothing else — both backlog items are functionally complete; remaining work is verification
   under honest conditions.

## g) QUESTIONS (cannot be answered from the repo)

1. **Daemon commit `cd3db3c82` mixes two sessions' work and its message describes changes that
   are not in the commit.** History is history (daemon commits are accepted noise per AGENTS),
   but if you want the fiction corrected — e.g. a follow-up note in the next CHANGELOG entry or
   an amend-authoritative summary in a status doc — say which form.
2. **The stack/pebble Normal-tier change ships as Changed/minor (preset default goes fsync-per-write
   → async WAL).** TODO/CHANGELOG already record minor. Confirm minor is intended for the next
   tag batch (vs holding it for a major) — it is the single most consumer-visible default change
   in this wave.
3. **Machine conditions**: root btrfs at 96% full (write latency degradation is likely a
   co-confounder in every IO benchmark today) and ambient load reaching 75. If you want the
   durability numbers + verify GREEN this week, a quieter window (or freeing disk) is needed —
   want me to schedule the re-runs at a specific time, or will you run them?

---

_Environment: verify ran with GOTOOLCHAIN=auto + /tmp cache redirects (GOCACHE/GOMODCACHE/GOPATH

- GOLANGCI_LINT_CACHE); /mnt/buildcache remained untrusted. Load at report time: 38.5 falling
  from a 75.8 peak. Foreign worktree changes (scheduling/, modules.md from another session, the
  other session's status report) untouched throughout._
