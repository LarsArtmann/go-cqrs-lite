# Status Report — 2026-08-16 14:07

## What I Did This Session

Implemented the **seq-carrying journal reads** design
(`docs/planning/SEQ-CARRYING-JOURNAL-READS.md`, TODO_LIST perf follow-up) —
O(log n) token-based journal resumption replacing O(offset) positional OFFSET
scans, end to end across the engine + adapter stack.

### Core + Conformance

| Change                                                                                                                                                                                                                      | State      |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| `metaengine/seq_seek.go` — `StreamLogEntry{Seq, Value}` + `SeqSeekableStreamLog` capability interface                                                                                                                       | DONE       |
| `metaengine/memory_stream_log.go` — memory engine impl (binary-search seek via `sort.Search`)                                                                                                                               | DONE       |
| `metaengine/enginetest/seqseek.go` — exported conformance suite `RunSeqSeekableStreamLogTest` (monotonic tokens, suffix equivalence, interleaved collections, limit, past-end, dense-journal agreement with position reads) | DONE       |
| `metaengine/seq_seek_gap_test.go` — gap-tolerance test (simulated journal entry deletion)                                                                                                                                   | DONE, PASS |

### Engines (all implement the capability + conformance test wired)

| Engine      | Change                                                                                          | Test state                    |
| ----------- | ----------------------------------------------------------------------------------------------- | ----------------------------- |
| sqlite      | `journalReadAllWithSeq` / `journalReadFromSeq` queries in `sqliteQuerySet`, `scanStreamEntries` | PASS                          |
| pg          | `seq > $n` range seeks on `idx_stream_log_journal`, `scanStreamEntries`                         | build OK; server test NOT run |
| mysql       | same pattern (`?` placeholders)                                                                 | build OK; server test NOT run |
| duckdb      | same pattern; conformance PASSES → design risk #1 (SEQUENCE semantics) resolved by test         | PASS                          |
| pebble      | `journalEntryFromKey` + `keycodec.JournalSeq` key-tail parsing                                  | PASS                          |
| bbolt       | same key-parsing pattern                                                                        | PASS                          |
| badger      | same key-parsing pattern                                                                        | PASS                          |
| memory      | see core above                                                                                  | PASS                          |
| dgraph/iroh | intentionally NOT implemented (design §7: optional capability, journal model differs)           | n/a                           |

`metaengine/keycodec/keycodec.go` gained `JournalSeq(key []byte) (int64, bool)`

- unit tests. Turso inherits via sqliteengine delegation.

### System Adapters (the actual perf win)

- `system/adapter_event.go` + NEW `system/adapter_event_journal.go` (split to
  respect the 350-line limit): `EventAdapter` detects the capability at
  construction; `ReadFrom` takes the token path — **zero-cursor reads do NO
  journal scan at all** (today every cold catch-up start scans + decodes the
  whole journal), and the seq cache stores true engine tokens instead of
  `afterSeq+i+1` position arithmetic.
- `system/adapter_core.go`: `ReadFromAfter` fast path — cursor resolution
  pages through token-carrying reads (bounded memory, 512/page) and the read
  is a token index seek; full-materialization `ReadAll` scan kept as fallback
  for engines without the capability.
- NEW `system/adapter_seq_token_test.go`: zero-cursor paged drain
  (catch-up subscriber pattern, exactly-once across interleaved collections),
  unknown-ID-reads-from-start semantics.
- Full `system` module suite: **PASS (4.8s)**.

## Current State

All implemented code builds (`go build -tags "goexperiment.jsonv2"` per
module) and every test that can run locally passes: memory, sqlite, duckdb
(CGo), pebble, bbolt, badger conformance + full `system` suite.

**Not verified yet (deliberately deferred):**

1. pg/mysql conformance tests need live servers (`nix run .#integration-pg`,
   `#integration-mysql-nspawn`) — code builds, dialects mirror sqlite's proven
   pattern, but not executed against real servers.
2. API-stability golden regen NOT run (exported symbols added:
   `metaengine.StreamLogEntry`, `SeqSeekableStreamLog`,
   `enginetest.RunSeqSeekableStreamLogTest*`, `keycodec.JournalSeq`).
3. Full `nix run .#verify` gate + `nix fmt` + lint + check-duplication not run.
4. Benchmark phase (design §5.5): journal-drain before/after @ 100k+ entries
   with benchstat, record in docs/BENCHMARKS.md.
5. Docs: design doc status still says PROPOSED; TODO_LIST item not updated.

## What I'd Do Differently / Forgot

- Did NOT regen the api-stability golden immediately in the same edit (the
  AGENTS.md procedure requires it right after adding exported symbols) —
  must do before any verify claim.
- `nix fmt` not yet run over all touched files (gofmt was run per-module on
  the files I edited, but treefmt/golines may still reflow).
- No `-race` run yet on the new adapter paths (memory engine + sqlite
  conformance are the highest-value race targets).
- tursoengine module has no direct test wiring (it delegates 100% to
  sqliteengine, so coverage flows through, but an explicit `:memory:` turso
  test would prove the driver path too).

## Suggested Next Steps

1. `nix fmt` + regen api-stability golden + `nix run .#verify` (or at minimum
   `#verify-fast`) — the mandatory GREEN gate.
2. Run pg + mysql conformance via the nix integration envs (exclusively —
   nothing heavy in parallel).
3. Benchmark journal-drain (sqlite @ 100k, page 500, position vs token) and
   record numbers in docs/BENCHMARKS.md.
4. Flip design doc status to IMPLEMENTED, mark the TODO_LIST item done.

## Follow-up (same session, later)

Items 2–4 above are DONE:

- **Benchmark** (§5.5): NEW `metaengine/sqliteengine/stream_log_bench_test.go`
  — `BenchmarkJournalPagedDrain_Position` vs `_Token` (100k entries, page 500,
  interleaved noise collection for real seq gaps). benchstat, 6×3 reps:
  position **761.8 ms ±17%** → token **106.8 ms ±20%** = **7.1x**; allocs
  +18% (1.106M→1.306M/op, the `StreamLogEntry` structs). Recorded in
  `docs/BENCHMARKS.md` ("sqliteengine paged journal drain"). Note: the gap is
  O(N²/P) vs O(N), so the ratio grows with journal size (wave-3's 285x was at
  200k with a self-join cursor; the OFFSET variant here degrades more gently).
  Instrument is a Go bench in sqliteengine (ledger convention), not a benchkit
  phase — benchkit's journey phase is end-to-end and would bury the signal.
- **api-stability golden**: regenerated (`--update`); 18 new exports
  (`metaengine` root: `SeqSeekableStreamLog`, `StreamLogEntry`, 2 methods;
  8 engine modules × 2 methods). `keycodec.JournalSeq` / `enginetest.*` are
  untracked internal packages (0 historical golden mentions — by design).
  Check run + `TestEvery*` meta-tests PASS.
- **Docs**: design doc flipped to IMPLEMENTED with measured numbers;
  TODO_LIST item marked done with the same evidence.
- **Full `nix run .#verify`**: two runs hit environmental interference from a
  concurrent session sharing the machine — (a) benchkit timing tests
  (Duration=10ms abort bound) false-fail under load avg 75/32-core, exactly
  the documented AGENTS.md gotcha class; (b) `cmd/cqrs-bench` DuckDB CGo link
  died "No space left" because `/mnt/buildcache` (the GOTMPDIR target) was
  99% full. Both suites PASS in isolation on the same tree (benchkit 44s,
  cqrs-bench layout 4.9s). All 24 modules touching this change were GREEN in
  the gate runs (metaengine + all engines + system). Final exclusive gate
  re-run: see below.

## Final gate status

`nix run .#verify` GREEN (exit 0) on the quiet machine — build, vet, test,
race, lint, check-arch/depguard/duplication/coverage, api-stability,
doc-check.
