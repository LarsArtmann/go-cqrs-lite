# ADR-0164: Per-group projection caps (implements #114)

| Field      | Value                                                                        |
| ---------- | ---------------------------------------------------------------------------- |
| Status     | Accepted                                                                     |
| Date       | 2026-11-05                                                                   |
| Deciders   | Lars Artmann                                                                 |
| Related    | #114, #72, ADR-0136 (invertibility ladder), ADR-0141 (temporal engines)      |
| Supersedes | nothing — extends the WithCap policy from #72 with a per-group bound         |

## Context

`metaengine.EnforceCap` keeps the newest N rows of a WHOLE collection (#72).
Consumers whose retention rule is per KEY GROUP (per-owner quotas, per-tenant
windows) still hand-roll check-then-expunge converging loops — the duplication
issue #114 documents at webphone (`call_transcripts` 5000/owner,
`dial_windows` 500/owner): a collection-global cap cannot express either
without letting one owner's flood evict another owner's rows.

Storage reality: Map collections are `meta_map(collection, key, value)` rows
where value is a JSON blob; planned tables carry `key, value` plus extracted
typed columns (layout DDL). Grouping in SQL therefore needs either
`json_extract(value, '$.<col>')` (both table shapes carry a JSON value column)
or the real extracted column (planned tables only).

## Decision

1. **Declaration**: `metaengine.WithCapPerGroup(groupCol string, perGroupRows int)`
   rides on a `WithCap` declaration — WithCap's `orderBy` orders survivors
   WITHIN each group, and `capRows` stays the global ceiling. Sets
   `capGroupCol`/`capGroupRows` on QueryConfig (`CapGroupCol()`/`CapGroupRows()`
   accessors). Grouped validation (`validateCapGroupedConfig`, cap_grouped.go)
   requires: a WithCap declaration (capRows > 0, orderBy non-empty), Map ADT,
   the key extractor, a group-value extractor over the result type (reflect,
   name-or-json-tag matched, case-insensitive — the same matching
   `ExtractFields` applies, so the Go extractor and SQL `json_extract` agree),
   and PLAIN IDENTIFIER syntax for groupCol and orderBy (both flow into SQL
   text; injection is a build-time error, not a runtime escape).
2. **Enforcement**: `Store.EnforceCapGrouped(ctx, queryName)` and
   `Store.EnforceCap` (grouped queries: grouped pass FIRST, then the existing
   global pass — the two caps compose; both under the query's fold lock, so
   `enforceGroupedTrim` itself never locks) and therefore `Store.EnforceCaps`.
   - Engine capability first: `CapGroupTrimmer` (optional interface)
     `TrimGroupedCap(ctx, collection, groupCol, orderBy string, perGroupRows int) ([]any, error)`
     returns the deleted KEYS so the store can notify live subscribers.
     Implemented by sqliteengine via one window-function statement:
     `DELETE ... WHERE rowid IN (SELECT rowid FROM (SELECT rowid,
     ROW_NUMBER() OVER (PARTITION BY <group> ORDER BY <order> DESC) AS rn
     FROM <table> [WHERE collection = ?]) WHERE rn > ?) RETURNING key`.
     Group/order expressions prefer the REAL column on planned tables and fall
     back to `json_extract(value, '$.<col>')` (meta_map rows and unplanned
     fields). Engines that cannot push down for a given collection (cell
     versioning active, non-identifier column) return
     `ErrCapGroupPushdownUnsupported` and the store trims via the Go path.
   - Go fallback (every other engine): one orderBy-DESC page scan
     (`capScanPage`), per-group counters via the reflect group extractor,
     delete overflow per group through the existing `capDelete` path (honors
     temporal engines via `MapDeleteAt`, notifies per row). A newest-first
     global scan makes per-group keep-newest-N a simple count-per-group.
3. `EnforceCaps` dispatches on config: grouped queries get the grouped trim
   plus the global pass, plain queries keep today's behavior exactly.
4. adttest pin: `metaengine/adttest/grouped_cap.go`
   (`AssertGroupedCapContract`) — per-group keep-newest-N, cross-group
   isolation, idempotence — run from every engine suite (pushdown and Go
   paths both exercise the same observable contract). Variadic query options
   let a suite steer the layout (e.g. `WithColumnarLayout` onto the
   real-column pushdown).
5. Capabilities doc regen: `docs/engine-capabilities.md` gains a
   TrimGroupedCap probe column (the census script derives it from source,
   same as the graph/removal/undirected probes).

## Consequences

- webphone's hand-rolled trim loops become `WithCapPerGroup` declarations
  (follow-up consumer change, not part of this landing).
- Replay rule carries over: reset + replay restores full rows; re-run
  EnforceCaps after a rebuild (trim is derived-state maintenance, ADR-0136).
- Engines without the capability pay a Go-side scan (correct, slower) —
  sqlite/turso (the #114 deployment target) get the pushdown; duckdb/pg
  native twins are deferred until a consumer asks (the Go fallback is correct
  for them today).
- Pushdown-returned keys are the engine's native key rendering (SQL engines
  return the stringified key column); notifications carry them as-is.
- NOT in scope: parameterized per-owner enforcement at query time (`capScanPage`
  takes no query args — the grouped DECLARATION is the v4 shape; args wait for v5).

## Implementation map

- metaengine/query_config.go: WithCapPerGroup + CapGroupCol/CapGroupRows
- metaengine/cap_grouped.go (NEW — cap.go is 328/350, no headroom):
  validateCapGroupedConfig (called from validateCapConfig), capGroupExtractor,
  CapGroupTrimmer, ErrCapGroupPushdownUnsupported, EnforceCapGrouped,
  enforceGroupedTrim, EnforceCap composition
- metaengine/sqliteengine/cap_grouped.go (NEW): TrimGroupedCap
  (meta_map json_extract + planned real-column paths; versioning refusal)
- metaengine/{duckdbengine,pgengine}: DEFERRED — Go fallback until a consumer
  asks (ctid for pg when it lands)
- metaengine/tursoengine: delegation coverage via the sqlite path
- metaengine/adttest/grouped_cap.go: contract pin
- docs/adr/0164-per-group-projection-caps.md + docs/adr/README.md index row
- docs/engine-capabilities.md regen; recipes.md fence (+ doc-check catalog);
  CHANGELOG [Unreleased]; FEATURES.md row; comment + close #114
- Gates: api-stability --update, check-untested-exports --update
  (cap_grouped_test.go + sqliteengine test + adttest name the new exports),
  check-engine-capabilities --update
