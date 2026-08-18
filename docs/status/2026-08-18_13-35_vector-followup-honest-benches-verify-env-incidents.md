# Status Report — Vector Binary Phase 0 Follow-Up: Honest Benches, Three-Engine Numbers, Verify Environment Incidents

**Date:** 2026-08-18 13:35 (+02:00)
**Scope:** This session only — executing the follow-up backlog (f-1..f-7, f-11, f-12) from
`docs/status/2026-08-17_14-25_vector-binary-phase0-and-depth1-graph-shortcircuit.md` after the
prior session ended awaiting user answers. Includes the full-verify campaign and the shell
environment incidents observed while running it. No unrelated research.

**Commit state:** Working tree CLEAN. All session work (bench files, doc corrections, status
addendum) captured by the auto-commit daemon inside `c2720df6a`
"feat(metaengine/vector): batch-vector binary codec + graph-shortcircuit honesty" and
subsequent doc commits (53d1a154b, 2b062037d are a concurrent session's v5-deprecation work).

---

## a) FULLY DONE

### 1. f-1 — Forced-mode graph bench honesty fix (the prior session's known defect)

- `metaengine/mysqlengine/graph_bench_test.go`: the forced-CTE arm of
  `runGraphNeighborsBench` now calls `e.graphNeighborsCTE` directly, bypassing the
  production depth-1 dispatch short-circuit. The forced-iterative arm keeps the public
  `GraphNeighbors` entry point (a no-CTE server genuinely takes short-circuit + BFS).
  The file header documents this as the "honesty rule" for future bench authors.
  `BenchmarkGraphNeighbors_CTE/.../depth-1` measures the actual CTE again.
- Vet green; full mysqlengine suite re-run green after the change.

### 2. Honest depth-1 measurement (f-1/f-2 evidence)

- Live against the relaunched userspace MariaDB 11.4.12 (20x, 1k/10k/100k graphs):
  **short-circuit 53-59µs vs true CTE 94-129µs — a real 1.6-2.4x win**, consistent with
  the original §9 crossover table's 2-4x claim. The prior session's "both forced modes
  converge" observation is now definitively documented as a bench artifact (both modes
  were executing the same direct SQL). Depth-2+ rows reproduce the known shape
  (CTE depth-flat, iterative depth-linear, crossover at depth 2).

### 3. f-3 — All three LSM vector engines measured (was pebble-only)

- New `metaengine/bboltengine/vector_bench_test.go` +
  `metaengine/badgerengine/vector_bench_test.go` (art-dupl:accept-annotated mirrors of
  pebble's; badger had NO vector bench at all before).
- Results (D=128, k=10, cosine, 20x): **pebble 457.4µs / 5.23ms, bbolt 425.7µs /
  5.79ms, badger 646.7µs / 5.85ms** (1K / 10K). All three in the same ~430-650ns/vector
  band — the Phase-0 win is the shared binary format, not a pebble artifact.
- Per-engine `TestVectorSearch_LegacyJSONPayloadReadable` re-verified green in all three.

### 4. f-4 — Filtered k-NN bench

- `BenchmarkPebbleVectorSearchFiltered_1K` (half the collection matching a parity
  filter): **1034.5µs vs 457.4µs unfiltered** — per-row metadata read + filter
  evaluation ≈ 2.2x over the bare scan.

### 5. f-5 — Codec micro-bench

- New `metaengine/vector_binary_bench_test.go`: decode **196ns binary vs 8.51µs JSON
  @D=128 (43x)**, **1.81µs vs 110.1µs @D=1536 (61x)**; 1 alloc vs 8-13. Encode:
  152ns / 1.4µs, 1 alloc. This pins the pure format cost independent of LSM noise —
  the reference point for future int8 work.

### 6. f-7 — G115 house pattern

- `metaengine/vector_binary.go`: inline `//nolint:gosec` replaced by the extracted
  `vectorBinaryDim` helper (mirroring `transport/grpc/event_version.go`). metaengine
  golangci-lint: 0 issues.

### 7. f-2 — All misleading wording corrected with honest numbers

- `METAENGINE-LIVE-LATENCY-MODEL.md` §9 finding #2: now carries the true
  short-circuit-vs-CTE numbers + an explicit note that the earlier "convergence" was a
  bench artifact. Historical table left intact (append-only; superset-safe under prior
  open question 3 either way).
- `CHANGELOG.md` both entries: vector entry now cites all three engines + the codec
  micro-bench; graph entry cites the honest 53-59µs vs 94-129µs measurement.
- Spike doc §2: follow-up table with the three engines, filtered k-NN, and codec
  numbers; bench-file inventory updated.
- `TODO_LIST.md` Phase-0 done-note extended with the follow-up bench facts.
- Prior session's status report: §h addendum appended documenting this follow-up.

### 8. f-11 — MariaDB hygiene

- `cqrs_test` dropped and recreated — **11 accumulated stale test tables pruned**; the
  `cqrs` TCP user re-provisioned; DSN verified end-to-end. Full mysqlengine suite:
  31 top-level tests, 82 runs, 0 skips, GREEN.

### 9. f-12 — Benchmark regression gate checked

- `scripts/benchmark-regression.sh` gate set is
  `BenchmarkFullPipeline_Memory|BenchmarkBenchkitSuite_Memory$` in `stack/bench`;
  `benchmarks/benchmark-baseline.txt` contains **0 references** to
  GraphNeighbors/VectorSearch series. **No re-pin needed.**

### 10. Module-level gates (all GREEN)

- `nix fmt` clean (no reformat needed on my files).
- golangci-lint **0 issues** on all 5 touched modules (metaengine, mysqlengine,
  bboltengine, badgerengine, pebbleengine).
- `check-changelog-symbols.sh`: 39 citations verified.
- `cmd/doc-check`: 921 references valid across 42 packages.
- `#check-duplication`: 0 new clones (baseline 111).
- `#load-sweep`: all timing-assertion tests survived CPU soakers.

### 11. Prior open question 1 answered by events

- The concurrent session's CHANGELOG entry ("stale id/v4.4.0 pins silently dropped
  ActorID in CBOR — all 59 modules bumped to v4.5.0") documents the go.mod sweep as an
  intentional bugfix. No action needed from this workstream.

---

## b) PARTIALLY DONE

### Full `nix run .#verify` — NOT banked green in a single shot; every failure root-caused environmental

Three attempts, each failing at a different stage for a different environmental reason;
every failing package passed in isolation afterwards:

1. **Attempt 1** — failed at Test: `cmd/cqrs-bench` tests (which each shell out to
   `go build` a DuckDB-static binary) hit **19× "No space left on device"**: /tmp at
   94% (45G/48G used by 40+ concurrent sessions' gocaches + multiple concurrent
   `/tmp/go-link-*` jobs). Fix: `GOTMPDIR=/mnt/buildcache/tmp` (now writable, 32G free).
   Isolation re-run: GREEN in 376s.
2. **Attempt 2** (with the TMPDIR fix) — failed at Test: **benchkit timing trio**
   (`TestRun_Pebble`, `TestRun_Recovery_Pebble`, `TestRun_AnalyticalJournalScans`) —
   the documented load-sensitive class; the machine was anything but exclusive.
   Isolation re-run: GREEN in 46.9s.
3. **Race stage** (manual mirror over the exact 76-module gate list): 4 package
   failures, all isolation-green afterwards:
   - `metaengine/bboltengine`: 12m timeout — **my mirror forgot `SOAK_SKIP_BOLT=1`**
     (the real gate sets it; the bbolt AutoCRUD soak runs 8-20m under load).
   - `cmd/cqrs-lint`, `cmd/cqrs-bench`: 12m timeout — cold race cache + parallel-load
     first build; GREEN with 25m headroom (cqrs-bench 325.9s).
   - `system/v4`: 2 tests (60.6s / 1.5s) — load flake; GREEN in 1.4s in isolation.
   - 116 packages green on the first pass; NO data races anywhere.
4. **Check stages after Test/Race never ran at app level** — the attempt to run them
   collided with the shell breakage (see d-1). Individually banked green this session:
   check-duplication, doc-check, changelog-symbols (module-level). **NOT run this
   session: `#lint` (app), `#check-arch`, `#check-depguard`, `#check-docserver-css`,
   `#check-coverage`, `#check-api-stability`.**

The honest summary: build + vet + test (except load-flakes) + race (except
env-timeouts) green across the workspace, with the remaining six check apps unverified
since the shell died mid-campaign and the user asked for a report.

---

## c) NOT STARTED

- **f-8** — `VectorCount` optional capability + Doctor/EXPLAIN WARN advisory (spike §5
  "cheap to add anytime" row).
- **f-9** — Phase 1 (int8 quantization) trigger evaluation: with all three engines at
  ~430-650ns/vector, is p99 still above budget at real N? (Data now exists to decide.)
- **f-10** — pgengine BYTEA dual-read migration sketch (deliberate deferral unless PG
  scan cost matters in practice).
- **f-13** — benchkit timing-flake isolation/quarantine (now THREE gate cycles burned
  across two sessions on this class).
- **f-14** — `#integration-mysql-nspawn` root blocker (pre-existing).
- **F46** — go-codec tagging vs workspace alloc pins (pre-existing).
- **`/mnt/buildcache` repair** (pre-existing; caches still redirected to /tmp — which
  has now itself become the gate-killer; see f-15/f-16 below).
- Prior open questions 2 and 3 remain formally user-gated (both now superset-safe —
  the strictest readings are satisfied by this session's data).

---

## d) TOTALLY FUCKED UP (honest ledger)

1. **The tool shell's coreutils vanished mid-session** — after the background
   check-stages loop, `tail`/`cat`/`grep` became "executable file not found", and even
   absolute `/run/current-system/sw/bin/tail` returned ENOENT. The check-stages run
   itself produced garbage as a result. Recovered on its own by next session day
   (2026-08-18 13:35 everything resolves again) — almost certainly a concurrent
   `nixos-rebuild switch` / profile swap / nix GC by another of the 40+ sessions,
   but from inside this session it killed the verify campaign's tail. Consequence:
   six check apps unverified (b-4).
2. **I violated the documented "exit codes after pipes lie" rule** — the check-stages
   loop piped each app through `| tail -2; echo "EXIT=$?"` and recorded EXIT=0 for a
   lint line that was actually an error ("golangci-lint: No such file or directory" —
   a vanished store path `yl3hlickshlf7smf4iql5lalmhi6hhqg`). This exact anti-pattern
   is called out in AGENTS.md; I read it at session start and did it anyway.
3. **My manual race-stage mirror omitted `SOAK_SKIP_BOLT=1`** — burned a 12-minute
   timeout + diagnosis cycle on the bbolt soak that the real gate explicitly skips.
4. **My first race-stage attempt used `./...`** from the repo root — covers only the
   root module — and briefly reported a false `RACE=0`. Caught because 0 `ok` lines is
   impossible; fixed by extracting the real 76-module `testModules` list from flake.nix.
5. **Bench pattern capitalization bug** (`Benchmark${m^}` produced
   `BenchmarkBboltengine`, matching nothing) — the first "all engines measured" log
   showed only `ok` lines with zero Benchmark rows; caught on inspection before any
   claim was recorded.
6. One CHANGELOG edit hit a stale read (daemon had modified the file between read and
   edit) — recovered by re-reading; cost one cycle.

---

## e) WHAT WE SHOULD IMPROVE

- **Gate output must go to a file, exit code checked separately** — no `| tail`
  anywhere near a gate command. Known rule; this session proves the failure mode is
  one tired refactor away even when you know it.
- **Bake `GOTMPDIR` into the flake verify/test apps** (or export it in the devShell):
  /tmp at 94% with 40+ sessions made ENOSPC the #1 gate-killer this wave; /mnt has
  32G free and is writable again.
- **A `#verify-race-only` app (or stage-restart flag)** would have saved this session
  ~90 minutes of manual stage mirroring — the mirror is where SOAK_SKIP_BOLT and the
  module list got hand-rebuilt (and one got missed).
- **Manual mirrors of flake gates are a smell** — read the app's script from flake.nix
  and copy its env verbatim, or don't mirror at all.
- **benchkit timing flakes now cost 3 gate cycles across 2 sessions** — the
  quarantine/isolate decision (f-13) has a measurable price tag; prioritize it.
- **The tool shell can lose coreutils mid-session** (concurrent rebuild/GC). Defensive
  habit: for gates, prefer dedicated tools (view/grep/glob) and file-redirected logs;
  the log files survived and were readable with the `view` tool even while the shell
  was dead.
- **Daemon commit mixing again** — this session's follow-up landed inside
  `c2720df6a` named for the codec work (acceptable fit) but interleaved with a
  concurrent session's v5-deprecation docs commits. Recurring archaeology cost.
- **badger's 1K number (646.7µs) is ~1.4-1.5x pebble/bbolt** — not investigated this
  session; possibly iterator overhead. Cheap follow-up candidate.

---

## f) NEXT — prioritized backlog (this session's fallout first)

1. Re-run the six unverified check apps (`#lint`, `#check-arch`, `#check-depguard`,
   `#check-docserver-css`, `#check-coverage`, `#check-api-stability`) — shell is
   healthy again as of 2026-08-18 13:35.
2. Then bank ONE clean single-shot full `nix run .#verify` (exclusive window,
   `GOTMPDIR=/mnt/buildcache/tmp`) to close b) properly.
3. Investigate the vanished golangci-lint store path (`yl3hlick…`) — nix GC race?
   Rebuild the lint app input if it recurs.
4. f-15 (new): bake `GOTMPDIR=/mnt/buildcache/tmp` into flake verify/test apps or
   devShell — kills the ENOSPC class permanently.
5. f-16 (new): prune or relocate the giant /tmp caches (19G gocache-verify, 4.2G
   golangci-lint-cache, 3G gomod-verify) now that /mnt is writable.
6. f-13: benchkit timing flakes — isolate/quarantine/relax (3 cycles burned).
7. f-8: `VectorCount` capability + Doctor/EXPLAIN WARN advisory.
8. f-9: Phase 1 (int8) trigger decision — the three-engine + codec data now exists.
9. badager 1K vs pebble/bbolt gap investigation (iterator overhead?).
10. Filtered k-NN benches for bbolt/badger twins (only if filtered path matters there).
11. f-10: pgengine BYTEA dual-read sketch (stays deferred unless PG cost matters).
12. Record the "manual verify mirror must copy the app env (SOAK_SKIP_BOLT=1)"
    gotcha in AGENTS.md alongside the existing verify guidance.
13. f-14: `#integration-mysql-nspawn` root blocker (unchanged).
14. F46 go-codec tagging → workspace alloc-pin updates (unchanged).
15. `/mnt/buildcache` full repair (unchanged; only /tmp workaround in place).
16. Consider a `#verify-race-only` flake app to make stage-level re-verification cheap.
17. MariaDB: add periodic prune of `cqrs_test` test tables to the relaunch recipe
    (11 tables had accumulated; will re-accumulate).

Beyond 17: nothing new observed this session; the prior backlog items (cqrs-lint
section, metaengine live-latency P2/P3, TODO_LIST F-items) remain authoritative.

---

## g) QUESTIONS (cannot be answered from the repo)

1. **Did you (or another session) run a `nixos-rebuild switch` / `nix store gc`
   around 2026-08-17 ~17:00?** The tool shell's coreutils vanished mid-verify
   (`/run/current-system/sw/bin/tail` → ENOENT, golangci-lint store path 404s) and
   self-healed by next day. If another session owns that event I will not touch nix
   profile state; if not, I should verify the system profile integrity before the
   next verify run.
2. **What is the GREEN-claim standard for this wave?** Every verify failure was
   environmental (ENOSPC / load flakes / cold-cache timeouts) with isolation greens
   and all module-level gates green — but a single clean full `#verify` has not been
   banked. Do you accept "environment-explained RED + isolation greens" as verified
   for this wave, or do you require the clean single-shot run (next-step 2) before
   any GREEN claim?
3. **Close the two standing Phase-0 questions?** With the three-engine numbers and
   the honest depth-1 measurement now recorded (both prior readings are supersets of
   what you were offered): (a) is Phase 0 now fully "done-done" in TODO_LIST, and
   (b) is the §9 append-plus-historical-table treatment what you wanted, or should
   the table be rewritten to current-truth instead?
