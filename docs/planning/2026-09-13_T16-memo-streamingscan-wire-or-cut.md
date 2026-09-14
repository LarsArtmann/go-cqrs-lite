# T16 Decision Memo: StreamingScan — Wire or Cut?

**Date:** 2026-09-13
**Status:** Proposal for decision (no code changed)
**Parent plan:** [`2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md`](2026-09-13_16-01_SUPERB-event-query-model-truth-reconciliation.md) (T16)
**Evidence:** [T02 verification notes](../status/2026-09-13_17-40_event-query-model-t02-verification-notes.md) · [15:55 deep dive](../status/2026-09-13_15-55_event-query-model-not-shipped-vs-reality.md)

---

## The situation

`StreamingScan` (`metaengine/engine.go:369-384`) is an optional engine capability:

```go
StreamScan(ctx, collection string, filters []FilterSpec, sort *SortSpec) iter.Seq2[any, error]
```

- Implemented by 4 engines: sqlite, pebble, bbolt, badger (covered by `pebbleengine/stream_scan_test.go` at minimum).
- **Zero production callers.** `Store.Export` (`export_import.go:12`) uses `MapScan`, which materializes every row in memory (`result.Items`) before writing. `Import` rebuilds via per-row `MapSet`.
- The interface's own doc comment says it exists for "batch processing or export operations" — the intended consumer was never wired.
- The design doc's Decision 2 (§15) proposed a query-level `Stream(ctx, input, fn)` mode with `iter.Seq2`; the engine-level half shipped, the query-level half did not.

## Options

| # | Option                                                      | Effort             | Risk                                              | Outcome                                                               |
| - | ----------------------------------------------------------- | ------------------ | ------------------------------------------------- | --------------------------------------------------------------------- |
| A | **Wire it** — add `Store.Stream`, use it in `Export`, tests | ~3-4h              | Low (additive API; fallback preserves behavior)   | Decision 2 honored; memory-safe exports; capability no longer a ghost |
| B | **Cut at v5** — delete `StreamingScan` + 4 engine impls     | ~1h + engine churn | Medium (removes tested code; abandons Decision 2) | Smaller surface; export stays O(all-rows-in-RAM)                      |
| C | **Keep dormant** — document as capability, no caller        | ~15min doc         | Low now, debt later                               | Ghost persists; every future reader asks why it exists                |

## Recommendation: **A — Wire it**

Rationale: the capability is already implemented and tested on 4 engines; the missing piece is one consumer. Export is a real OOM vector (a 10M-row collection today materializes fully). The wiring is additive, fallback-safe, and closes the doc's Decision 2 in the same move.

### Ready-to-execute wiring steps (for T23)

1. **`Store.Stream(ctx context.Context, queryName string, fn func(any) error) error`** — no generics in the first cut (keeps API small); resolves query → collection + configured filters/sort, then:
   - if `eng.(StreamingScan)` → iterate `StreamScan(...)`, call `fn` per item, stop on `fn` error;
   - else fallback → `MapScan` once and iterate items (identical observable behavior, no new failure mode).
2. **`Export` integration** — per collection, prefer `StreamingScan` and write rows incrementally; fall back to `MapScan`. Export output format stays byte-identical (key order: collection iteration order unchanged; row order was already engine-defined).
3. **Tests** — (a) capability path via sqlite/pebble engine, (b) fallback path via memory engine, (c) `Export` golden equality between before/after for a mixed store, (d) `fn`-error propagation, (e) canceled context mid-iteration.
4. **API golden** — `cd cmd/api-stability && GOWORK=off go run -tags "goexperiment.jsonv2" . --update`.
5. **Docs** — announce `Store.Stream` in the metaengine README and mark §15 D2 resolved in `event-query-model.md`.

### If the decision is B (cut)

Remove `StreamingScan` from `engine.go`, delete the 4 engine implementations and their tests, and update §15 D2 to "cut at v5 — Export uses bounded scans" in the same change. Do not leave a half-removed interface.

## Decision

- [ ] A — wire it (recommended)
- [ ] B — cut at v5
- [ ] C — keep dormant

---

## Update 2026-09-13 (executed)

Option A was implemented the same day: `Store.StreamCollection`
(`metaengine/stream_collection.go`) streams via the `StreamingScan` capability and falls back to
`ScanBackend.MapScan`; `Store.Export` now streams each collection row-by-row with byte-identical
output (existing export tests pass). Five new tests cover capability preference, fallback,
unknown collection, fn-error propagation, and export equality. The query-level
`Stream(ctx, input, fn)` form remains future work.
