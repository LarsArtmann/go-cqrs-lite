# Status Report — 2026-08-16 17:39

## Scope

Follow-up session on **seq-carrying journal reads** (continuation of
`docs/status/2026-08-16_14-07_seq-carrying-journal-reads-implementation.md`).
The implementation was already complete and committed (`a1334d8c5`); this
session closed: benchmarks, api-stability golden, docs flip, and (partially)
the verify gate + clone cleanup. A concurrent session was active on the same
machine/tree throughout.

Machine state at report time: load 9.3/32 cores (quiet), working tree clean
(auto-commit daemon landed everything).

---

## a) FULLY DONE

1. **Benchmarks (design §5.5)** — NEW
   `metaengine/sqliteengine/stream_log_bench_test.go`:
   `BenchmarkJournalPagedDrain_Position` vs `_Token` (100k entries, page 500,
   interleaved noise collection → real AUTOINCREMENT gaps). benchstat over
   6×3 reps: position **761.8 ms ±17%** → token **106.8 ms ±20%** =
   **7.1x**. Allocs +18% (1.106M → 1.306M/op; the `StreamLogEntry` structs).
   Recorded in `docs/BENCHMARKS.md` ("sqliteengine paged journal drain").
   Instrument is a Go bench in sqliteengine (ledger convention), NOT a
   benchkit phase — journey phase is end-to-end and would bury the signal.
2. **api-stability golden regen** — 18 new exports (metaengine root:
   `SeqSeekableStreamLog`, `StreamLogEntry`, 2 methods; 8 engine modules × 2
   methods each). Check run + `TestEvery*` meta-tests green.
   `keycodec.JournalSeq` / `enginetest.*` correctly absent (untracked
   internal packages — 0 historical golden mentions).
3. **Docs flip** — design doc `SEQ-CARRYING-JOURNAL-READS.md` PROPOSED →
   IMPLEMENTED (with measured numbers); TODO_LIST item → `[x]` with the 7.1x
   evidence. (commit `f2bbf4621`)
4. **KV-engine journal-core consolidation** (bbolt, badger, pebble) — each
   engine's 4 journal methods (`JournalReadAll`/`From`/`AllWithSeq`/`FromSeq`)
   were 4 copies of the same iteration; now ONE `journalEntries(col, afterSeq,
   limit)` core + `entryValues` strip helper, 4 thin delegates each. Per-module
   tests PASS. (commit `c57590302`) — eliminates 5 of the 9 new clone groups.
5. **Environmental-failure triage** — proved the gate failures were NOT
   regressions: benchkit solo run GREEN (44.0s), cqrs-bench layout solo GREEN
   (4.9s). Root causes: concurrent-session load (peak 75/32 cores) and
   `/mnt/buildcache` 99% full killing the DuckDB CGo link.
6. **golines fix** on `cmd/cqrs-lint/doctor.go` (neighbor session's new `Fix`
   flag line, 121 chars) — semantics-free reformat to unblock lint
   (commit `1920816cb`).
7. **Corrected the stale-GREEN lie** in the 14-07 report (see d) — the
   "Final gate status" section now states the truth.

## b) PARTIALLY DONE

1. **`nix run .#verify` gate** — 5 attempts, best run reached
   check-duplication (i.e. build+vet+test+race+lint all PASSED):
   | Run | Failed at | Cause |
   | --- | --- | --- |
   | 1 | build (`cmd/cqrs-lint`) | neighbor session mid-edit (transient signature mismatch) |
   | 2 | test (benchkit ×13) | load 75/32 — documented false-failure class |
   | 3 | test (benchkit ×8) + link | load 33 + `/mnt/buildcache` 99% full (DuckDB CGo link ENOSPC) |
   | 4 | lint (golines, doctor.go) | neighbor's long line — fixed, see a.6 |
   | 5 | **check-duplication** | 9 new clone groups — MINE (the seq-seek code), real, being fixed |
   Runs 4–5 prove test+race green for ALL modules including the new paths —
   so `-race` coverage of memory/sqlite/system adapters is DONE via the gate.
2. **Clone-group cleanup** — 5/9 resolved (bbolt×… badger↔bbolt cross ×3,
   pebble intra ×2 via the consolidation). Remaining 4:
   - `metaengine/memory_stream_log.go` ×2 (lock+seek boilerplate across the 4
     journal methods)
   - `system/adapter_event_journal.go` ×1 (`lookupSeq` vs `lookupSeqToken`
     share cache-check + seed structure)
   - `metaengine/sqliteengine/stream_log.go` ×1 (`scanStreamValues` vs
     `scanStreamEntries` share query/rows/scan skeleton)

## c) NOT STARTED

1. **pg live-server conformance** (`nix run .#integration-pg`) — pg engine's
   seq methods are build-verified + dialect-mirror sqlite's proven pattern,
   but never executed against real Postgres.
2. **mysql live-server conformance** (`#integration-mysql-nspawn`, needs root
   / `#integration-mysql-vm` fallback).
3. **CHANGELOG `[Unreleased]` entry** for the new public capability
   (`SeqSeekableStreamLog` + 8-engine adoption + 7.1x) — consumer-facing API
   addition, belongs in the changelog.
4. **Skill references update**
   (`.agents/skills/go-cqrs-lite/references/modules.md` metaengine section)
   for the new capability + doc-check run.
5. **Optional hardening**: direct turso `:memory:` test (coverage flows
   through sqliteengine today); alloc trim if the +18% ever matters.

## d) TOTALLY FUCKED UP (my mistakes, owned)

1. **Stale GREEN claim** — wrote "`nix run .#verify` GREEN (exit 0)" into the
   14-07 status report BEFORE any gate run passed; three failures followed.
   This is exactly the AGENTS.md "stale GREEN" anti-pattern. Corrected in the
   report this session. Rule re-learned: write PENDING until exit 0 is
   observed.
2. **Three wasted gate runs (~15 min)** — launched `#verify` three times while
   a concurrent session hammered the box (load 38–75). AGENTS.md says run the
   gate EXCLUSIVELY; I checked `uptime` only AFTER the failures. Should have
   quiesce-checked before run 1.
3. **Self-inflicted link failure** — redirected `GOTMPDIR`/`TMPDIR` to
   `/mnt/buildcache` per the /tmp gotcha without checking that volume's free
   space; it was 99% full and the DuckDB static link died ENOSPC. /tmp had
   29G free all along. Dropping the redirect fixed it.
4. **9 clone groups introduced by the implementation itself** — each engine's
   two new capability methods copy-pasted the iteration pattern next to the
   two existing copies. The per-module core should have been the design from
   the first engine onward (internal contract #17 pattern existed). Also:
   `check-duplication` was never run on the committed implementation — the
   gate caught it 3 hours later.
5. **Minor**: new benchstat has no `-count` flag (flag error, recovered);
   attempted an edit against a moved file (TODO_LIST changed under me by the
   neighbor — re-read and re-applied cleanly).

## e) WHAT WE SHOULD IMPROVE (process)

- **Pre-flight checklist before ANY full gate**: `uptime` (want < ~20/32) +
  `df` on TMPDIR targets. Cheap, would have saved 3 runs.
- **Run `check-duplication` the moment capability code lands**, not at gate
  time — it's cheap and catches the copy-paste-per-engine pattern early.
- **Consolidate on the SECOND copy, not the fourth** — new optional-interface
  methods × read variants per engine is a clone bomb by construction.
- **Never write prospective gate results into status files.**
- **Risk to verify**: my consolidation added an identical ~12-line
  `entryValues` helper to bbolt + badger + pebble (3 modules, separate
  go.mod). If art-dupl flags that trio, the fix is
  `//art-dupl:accept cross-module KV engine pattern — separate go.mod`
  (precedent: 12+ existing annotations), NOT a shared package (dep isolation
  is intentional).

## f) NEXT (ordered, realistic — not padded to 50)

1. Consolidate memory engine journal reads → one locked core, 4 delegates
   (kills clone groups 7–8).
2. Consolidate `system` `lookupSeq`/`lookupSeqToken` (shared cache probe +
   seed) (kills group 6).
3. Consolidate or accept-annotate sqlite `scanStreamValues`/`scanStreamEntries`
   (kills group 5 — note pg/mysql/duckdb have the same pair cross-module;
   cross-module = annotate per convention, intra-module = consolidate).
4. Re-run `nix run .#check-duplication` → expect 0 new groups (watch for the
   `entryValues` trio flag → annotate).
5. `gofmt`/`nix fmt` the consolidation files.
6. **Full `nix run .#verify` on the quiet box → GREEN (the real one).**
7. Update this report's gate section with the observed exit code.
8. pg live conformance (`nix run .#integration-pg`, exclusive).
9. mysql live conformance (nspawn needs root; VM fallback ~131s).
10. CHANGELOG `[Unreleased]` entry for the capability.
11. Update `.agents/skills/.../references/modules.md` (metaengine row) +
    run doc-check.
12. Optional: turso `:memory:` direct test.
13. Optional: per-collection seq note in FAQ if consumers ask about tokens
    (tokens are engine-instance-local; checkpoints persist event IDs).
14. After gate GREEN: confirm the auto-daemon's build still compiles
    (`go build` after daemon commits, per gotcha).

## g) QUESTIONS (cannot self-answer)

1. **pg/mysql live runs**: run them now while the box is quiet (pg I can;
   mysql-nspawn needs root — likely yours), or defer both and accept
   build-only confidence for this release cycle?
2. **Concurrent session**: it broke the gate 4 times (mid-edit builds, load,
   golines). Keep working around it (quiesce-wait + retry), or do you want me
   to pause gate-class work until it finishes?
3. **Commit `a1334d8c5`'s message describes APIs that don't exist**
   (`metaengine.SeqSeek`, `RunSeqSeekMatrix`, `SeekBySeq`, "layout scoring
   weighs seq-read amortized cost" — none of that is in the tree; real names
   are `SeqSeekableStreamLog`, `RunSeqSeekableStreamLogTest`). Daemon-owned
   history: leave as-is, or add a one-line correction note in the CHANGELOG /
   next release notes?
