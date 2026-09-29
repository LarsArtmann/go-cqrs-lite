# Pebble/bbolt versioned cells — scope decision one-pager (M18.4)

> Status: DECISION NOTE for the owner (prep-only; no implementation before a
> ruling). Harvested from 14:07 §f31 via the 2026-09-28 post-wave Pareto plan.

## Question

Should `metaengine/pebbleengine` and `metaengine/bboltengine` join the
versioned-cells engines (today: memory, sqlite/turso, bigtable), implementing
the ADR-0141 temporal surface (`VersionedWriter.MapSetAt/MapDeleteAt`,
`VersionedStorage.MapGetAsOf/MapExistsAsOf`, `CellHistoryReader.MapHistory`,
retention)?

## What they bring natively

Both are ordered KV stores whose cores already do prefix-range machinery:

- **bbolt**: buckets + `Cursor.Seek` byte-order scans; the journal already
  uses a secondary-index bucket + Seek-based reads (`storage/bbolt`
  `cqrs_journal_idx`). A version chain is
  `{col}/{key}/{020d_unixnano}` → value: one bucket-range scan per as-of
  read, binary-searchable by key construction (the timestamp IS the sort key).
- **pebble**: key-prefix iteration with `Iter.SeekGE` — same layout shape.

That makes the READ side cheap and natural. The hard parts are elsewhere
(below).

## Effort estimate (S/M/L per engine)

| Piece                                                                   | bbolt | pebble | Notes                                                                                                                                                                                                                                            |
| ----------------------------------------------------------------------- | ----- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Chain layout + MapSetAt/MapDeleteAt                                     | S     | S      | append-only keyed writes                                                                                                                                                                                                                         |
| MapGetAsOf (seek-to-prefix, read backwards one step)                    | S     | S      | the natural fit                                                                                                                                                                                                                                  |
| Retention (MaxVersions/MaxAge, never-prune-newest)                      | M     | M      | batched range deletes; must mirror `trimRetentionLocked` semantics exactly (incl. same-ts LWW ties and the newest-entry guarantee — now property-pinned by `temporal_property_test.go`, so a divergence fails tests instead of rotting silently) |
| Same-ts LWW collapse                                                    | S     | S      | duplicate `{ts}` keys need a tiebreak byte (application seq) to make "last applied wins" durable — unlike memory, KV has no insert-order column                                                                                                  |
| `EngineResetter` interaction (ADR-0143: journal survives, chains clear) | S     | S      | chains are derived data → clear on reset, keep journal                                                                                                                                                                                           |
| adttest conformance legs                                                | S     | S      | the versioned matrix runner already exists                                                                                                                                                                                                       |

Total: ~M per engine, mostly shared shape (a third SQL-vs-KV dialect split
like the scan-family work in t4).

## Tradeoffs

- **Pro**: embedded-durable temporal reads without SQL; closes the
  "versioned = memory|sql|cloud" asymmetry; the property suite (M18.1-18.3)
  already encodes the contract engine-neutrally.
- **Con**: retention = range deletes on every write (write amplification on
  pebble; tx size on bbolt); the tiebreak byte bakes application order into
  the key format (a one-way door — pick once); demand so far is speculative
  (no consumer has asked for durable-as-of on embedded KV).

## Recommendation

**Defer to the next wave with a demand trigger**: implement when a consumer
asks for durable as-of reads on an embedded engine, or when the differential
memory-vs-X temporal harness (shipped 2026-09-28) grows a KV leg organically.
The layout one-way door (tiebreak byte) deserves a deliberate decision, not a
wave-tail landing.

## Owner ruling requested

A) defer (recommended) · B) scope into the next wave · C) bbolt-only (it
already carries the Seek precedent) · D) drop the row.
