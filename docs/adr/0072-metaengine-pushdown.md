# ADR-0072: Metaengine Pushdown (json_extract SQL pushdown)

|             |                                                                               |
| ----------- | ----------------------------------------------------------------------------- |
| **Status**  | Accepted                                                                      |
| **Date**    | 2026-07-29                                                                    |
| **Context** | Filtered/sorted scans on the SQLite metaengine loaded every row into Go       |

## Context

The metaengine stores every `Map` collection in a single `meta_map(collection,
key, value)` table, where `value` is a JSON blob. The original scan path
(`ScanBackend.MapScan`) loaded **all** rows for a collection into Go, applied a
closure filter in memory, then sorted in Go. For a 100K-row projection this is
O(N) decode + Go-side filter/sort regardless of how selective the query is.

Declarative filters (`FilterOnField`) and sorts (`SortOnField`) carry a field
name and operator but, without pushdown, they were only used by the closure
fallback — the SQL never saw them.

## Decision

Add a `PushdownScan` interface that engines may implement. When a query's
filter/sort accessors are **all** declarative (`canPushdown == true`) and the
chosen engine implements `PushdownScan`, the executor pushes `WHERE`, `ORDER BY`,
and `LIMIT` into SQL via `json_extract(value, '$.field')`:

```sql
SELECT value FROM meta_map
WHERE json_extract(value, '$.status') = ?
ORDER BY json_extract(value, '$.priority')
LIMIT ?
```

This turns an O(N) full scan + Go sort into an O(N) indexed-ish scan with
server-side filtering and ordering. The closure path (`FilterOn`/`SortOn`)
remains the fallback whenever any accessor is not declarative — `canPushdown`
returns false the moment a closure-only accessor appears.

A regression test (`TestPushdownSQL_JSONExtractReachesDB`) pins the proof: the
`meta_map` DDL has **no** `status` column, so a filter on `status` can only
succeed via `json_extract`. The `+1` LIMIT trick signals "has more" for
keyset-cursor pagination.

## Consequences

- SQLite engine gains `PushdownScan`; memory and Pebble do not (they stay
  closure-based — no SQL to push into).
- A query mixing `FilterOnField` (declarative) with a closure sort now correctly
  falls back AND still applies the declarative filter (fixed alongside this ADR
  in `buildFilterPredicates` — previously declarative filters were silently
  dropped in the closure path).
- Pushdown is transparent: the same query returns identical results via either
  path; only the cost changes.

## Alternatives considered

- **Always load + filter in Go**: simplest, but O(N) for every scan. Rejected for
  dashboard/reporting read models.
- **Generated typed read API (sqlc-style)**: the strongest long-term option
  (dedicated indexed tables), but it requires code generation. It is captured
  separately as layout planning (ADR-0073) and the future typed-API work.
