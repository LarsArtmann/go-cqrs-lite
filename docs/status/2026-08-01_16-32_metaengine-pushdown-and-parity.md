# Status Report: Metaengine Pushdown + Cross-Engine Parity

**Date:** 2026-08-01 16:32
**Session scope:** pgengine + duckdbengine PushdownScan, LayoutPlanner, adttest matrix wiring, doc overclaim fixes

---

## A. FULLY DONE

1. **pgengine adttest.RunMatrix wiring** — `adt_matrix_test.go` created. Map, Counter, SortedMap pass with memory-engine parity. 7 unimplemented ADTs (Set, Graph, Log, Multimap, Vector, Search, Spatial) auto-skip via reflection.
2. **duckdbengine adttest.RunMatrix wiring** — `adt_matrix_cgo_test.go` created. Same 3 ADTs pass parity, 7 auto-skip.
3. **pgengine PushdownScan** (`pushdown.go`) — Filter/sort/cursor/limit pushed into Postgres `WHERE`/`ORDER BY`/`LIMIT` using `value->'field'` JSONB operators. Parameters cast as `$N::jsonb` for type-correct comparisons. FilterIn operator supported (`value->'field' IN ($2::jsonb, $3::jsonb, ...)`).
4. **pgengine LayoutPlanner** (`pushdown.go`) — `ApplyLayout` creates partial expression indexes: `CREATE INDEX IF NOT EXISTS ... ON meta_map ((value->'field')) WHERE collection = '...'`. Idempotent (tracks applied layouts in a map).
5. **duckdbengine PushdownScan** (`pushdown.go`) — Filter/sort/cursor/limit pushed into DuckDB `WHERE`/`ORDER BY`/`LIMIT` using `json_extract(value, '$.field')`. Parameters cast as `$N::json`.
6. **pgengine pushdown tests** (6 tests) — filter, sort, filter+sort+limit, cursor pagination, FilterIn, ApplyLayout idempotency + pushdown integration.
7. **duckdbengine pushdown tests** (5 tests) — filter, sort, filter+sort+limit, cursor pagination, FilterIn.
8. **Documentation overclaim fixes** — pgengine doc.go: removed false "GIN indexes" claim, replaced with accurate PushdownScan/LayoutPlanner description. duckdbengine doc.go: removed false "vectorized GROUP BY for O(1) Counter reads" claim, replaced with honest description. scan.go comments in both engines updated (removed "Future enhancement" notes since pushdown is now implemented).
9. **AGENTS.md + ROADMAP.md updated** — module descriptions reflect actual capabilities. ROADMAP items updated to show what shipped vs what remains.
10. **API stability verified** — 3086 exports verified, zero drift.
11. **Both modules vet-clean** — `go vet` passes with `goexperiment.jsonv2` build tag.

---

## B. PARTIALLY DONE

1. **pgengine JSONB operators** — `value->'field'` (JSONB access) is implemented and working. The `@>` containment operator and `?` existence operator are NOT implemented. These are needed for "tags contains X" or "metadata has key Y" query patterns. The `->>` text-extraction operator is also not used (we use `->` which returns JSONB, preserving types).
2. **pgengine LayoutPlanner** — Expression indexes (partial B-tree on JSONB paths) are implemented. GIN indexes (for `@>` containment queries) are NOT implemented. GIN would benefit full-text-search and array-membership patterns, but those query types aren't yet declared in the metaengine FilterSpec system.
3. **duckdbengine PushdownScan** — `json_extract` filter/sort pushdown works. DuckDB-specific columnar optimizations (vectorized GROUP BY, column pruning, zone maps) are NOT leveraged. The scan still reads the `value` column as a JSON string; DuckDB's columnar advantage is minimal when data is stored as JSON blobs.
4. **duckdbengine CounterGet** — Still a row-by-row `SELECT key, value FROM meta_counter WHERE collection = $1` scan. No vectorized `SUM...GROUP BY` pushdown. The doc comment was corrected to stop claiming this.
5. **Cross-engine pushdown parity** — The pushdown tests are per-engine (pgengine tests against Postgres, duckdbengine tests against DuckDB). There is no shared pushdown test matrix that asserts both engines produce identical pushdown results for the same FilterSpec/SortSpec inputs (unlike the adttest.RunMatrix which covers the backend interfaces).
6. **10M-event soak test** — Not addressed in this session. The memory engine's existing soak test caps at 50K events.

---

## C. NOT STARTED

1. **GIN index support in pgengine** — Would require new FilterSpec operators (e.g., `FilterContains`, `FilterExists`) that translate to `@>` and `?` JSONB operators. Currently FilterSpec only has comparison operators (=, !=, <, <=, >, >=, IN).
2. **DuckDB columnar-native storage** — Storing data in actual columnar format (one column per declared field) instead of JSON blobs in a VARCHAR column. This would require LayoutPlanner support in duckdbengine (not implemented) and a different table schema.
3. **`metaengine-gen` code generator** — Not touched. CLI tool for typed Store methods from query declarations.
4. **DuckDB vectorized CounterGet** — Would require a `SELECT key, SUM(value) FROM meta_counter WHERE collection = $1 GROUP BY key` rewrite. Trivial change but not done.
5. **DuckDB LayoutPlanner** — Not implemented. DuckDB supports `CREATE INDEX` but the columnar engine doesn't benefit from B-tree indexes the same way row-oriented engines do. Zone maps (DuckDB's automatic min/max statistics per row group) would be the right optimization, but those are automatic and don't need explicit DDL.
6. **PushdownScan integration test via metaengine.Plan** — The pushdown tests call `PushdownMapScan` directly. No test verifies the full `Plan → Apply → ExecuteTyped` path with `FilterOnField`/`SortOnField` options against pgengine or duckdbengine (the existing `TestPostgresEngine_MetaenginePlan` and `TestDuckDBEngine_MetaenginePlan` use counter queries, not filtered scans).

---

## D. TOTALLY FUCKED UP

Nothing catastrophic. But several things worth calling out honestly:

1. **Test assertion bug — limit+1 confusion** — First test run failed because I wrote `expected 3` when `PushdownMapScan` returns `limit+1` rows (for has-more detection). I should have known this from reading the SQLite engine's `PushdownMapScan` implementation, which explicitly emits `LIMIT ?` with `limit+1`. I caught and fixed it immediately, but it's a sign I didn't fully internalize the convention before writing tests.
2. **Unused import** — Left `"sync"` imported in pushdown.go after moving the layout mutex to engine.go. Caught by `go vet`, fixed immediately. Sloppy.
3. **Overly complex interface** — Initial `scanDuckDBJSONValues` used a convoluted anonymous interface with nested anonymous interfaces for the `*sql.DB` subset. Over-engineered for no reason. Replaced with plain `*sql.DB`. Same issue in the first draft of pgengine's `queryExecer` interface — deleted entirely.
4. **pgDSN called outside factory** — Initially called `pgDSN(t)` once at the top of `TestPostgresADTMatrix` and passed the DSN string to all subtests. This caused all subtests to share the same database, and parallel CREATE TABLE conflicts caused the Counter test to silently skip. Fixed by moving `pgDSN(t)` inside each factory closure so each subtest gets its own database. This is documented behavior in the testcontainer pattern — I should have followed it from the start.

---

## E. WHAT WE SHOULD IMPROVE

1. **No shared pushdown parity test** — adttest.RunMatrix covers backend interfaces (MapSet/MapGet, CounterIncrement/CounterGet, MapScan). But PushdownScan is not covered by any shared test harness. A `RunPushdownMatrix` that seeds identical data and asserts identical FilterSpec/SortSpec results across engines would catch pushdown divergences. Currently each engine has its own ad-hoc pushdown tests with different data sets.
2. **FilterSpec lacks JSONB-native operators** — The metaengine FilterSpec only has comparison operators (`=`, `!=`, `<`, `<=`, `>`, `>=`, `IN`). Postgres JSONB has powerful operators that have no metaengine equivalent: `@>` (contains), `<@` (contained by), `?` (key exists), `?|` (any key exists), `?&` (all keys exist), `#>` (path access). Without these, the pgengine can't push down the kinds of queries where JSONB + GIN indexes shine most.
3. **DuckDB engine stores JSON as VARCHAR** — The `meta_map` table uses `value VARCHAR NOT NULL`. DuckDB has a native `JSON` type (alias for VARCHAR with validation) and a `STRUCT` type for columnar data. Using VARCHAR means DuckDB must parse JSON on every query, negating the columnar advantage. Storing as STRUCT (or using LayoutPlanner to extract columns) would unlock DuckDB's real power.
4. **No `Explain` integration for pgengine/duckdbengine** — The SQLite engine has `explainStandard`/`explainPlanned` that show the SQL plan. Neither pgengine nor duckdbengine hooks into the `Explain` system. Operators can't see what SQL is being generated for their queries.
5. **Error handling in pushdown.go** — Both engines silently ignore `json.Marshal` errors with `jb, _ := json.Marshal(value)`. If a filter value can't be marshaled to JSON, the query parameter is empty and the SQL comparison silently produces wrong results. Should return an error.
6. **No RawValueReader/RawScanReader for either engine** — SQLite and Pebble engines implement these to skip JSON decode on hot paths. Neither pgengine nor duckdbengine does. For pgengine, `value::text` already returns the raw JSON string; for DuckDB, the VARCHAR column is already raw. Adding these interfaces would eliminate redundant decode in scan paths.
7. **No StreamingScan for either engine** — SQLite has `StreamScan` (iter.Seq2 for OOM-safe lazy iteration). Neither pgengine nor duckdbengine implements it. For large collections, both engines materialize all rows into a `[]any` slice.
8. **LayoutPlanner only in pgengine** — DuckDB doesn't get LayoutPlanner. While DuckDB's columnar engine benefits less from B-tree indexes, explicit column extraction (via CREATE TABLE with extracted columns or DuckDB's generated columns) would still help filter pushdown by avoiding JSON parsing.
9. **SQL injection surface in ApplyLayout** — The `escapeSQLString(collection)` function escapes single quotes, but the index name is sanitized by replacing non-alphanumeric characters. The collection name is interpolated into the DDL via `WHERE collection = '...'` — if collection contained a backslash or other escape sequence, behavior depends on Postgres's `standard_conforming_strings` setting. Should use parameterized DDL or `quote_ident()`.
10. **No benchmark tests** — No `BenchmarkPushdownMapScan` in either engine. The whole point of pushdown is performance, but there are no tests that measure whether pushdown is actually faster than the Go-side MapScan fallback.
11. **Test helper duplication** — `seedProducts` (pgengine) and `seedDuckDBProducts` (duckdbengine) are identical functions with different names. They should be shared, either in adttest or a new test helper package.
12. **No MapDelete test in duckdbengine** — The existing duckdbengine test suite has no `TestDuckDBEngine_MapDelete`. pgengine has one. This gap existed before this session but was not addressed.

---

## F. Next 50 Things to Get Done

### Metaengine Engines (pgengine + duckdbengine)
1. Add `RunPushdownMatrix` to adttest — shared pushdown parity test across engines
2. Add `FilterContains` / `FilterExists` operators to FilterSpec for JSONB `@>`/`?` support
3. Implement GIN indexes in pgengine for `@>` containment queries
4. Fix silent `json.Marshal` errors in both pushdown implementations — return error
5. Add RawValueReader to pgengine (skip JSON decode on point reads)
6. Add RawValueReader to duckdbengine
7. Add StreamingScan to pgengine (iter.Seq2 for large collections)
8. Add StreamingScan to duckdbengine
9. Add `BenchmarkPushdownMapScan` to pgengine — compare pushdown vs Go-side MapScan
10. Add `BenchmarkPushdownMapScan` to duckdbengine
11. Rewrite duckdbengine CounterGet to use `SELECT key, value FROM meta_counter WHERE collection = $1` (already does this — verify it's optimal or add `SUM(value) GROUP BY key`)
12. Add DuckDB LayoutPlanner — extract JSON fields into typed columns for columnar-native scans
13. Change duckdbengine `meta_map.value` from VARCHAR to DuckDB native JSON type
14. Add `Explain` integration for pgengine (show generated SQL plan)
15. Add `Explain` integration for duckdbengine
16. Add shared test helper to eliminate seedProducts/seedDuckDBProducts duplication
17. Add MapDelete test to duckdbengine test suite
18. Add pushdown integration test via `metaengine.Plan` + `FilterOnField` + `ExecuteTyped` for pgengine
19. Add pushdown integration test via `metaengine.Plan` + `FilterOnField` + `ExecuteTyped` for duckdbengine
20. Harden ApplyLayout SQL injection surface — use parameterized DDL or `quote_ident()`
21. Add concurrent write/read stress test for pgengine (like TestSoak_SQLiteSustainedWrites)
22. Add concurrent write/read stress test for duckdbengine
23. Add `FilterLike` / regex operator to FilterSpec (Postgres `~`, DuckDB `regexp_matches`)
24. Add multi-field sort support to SortSpec (currently only single sort field)
25. Add compound index support to LayoutPlanner (multi-column indexes)

### Metaengine Core
26. Implement 10M-event soak test for memory engine (verify O(keys) memory bound)
27. Add `FilterNull` / `FilterNotNull` operators (Postgres `IS NULL` / `IS NOT NULL`)
28. Add `FilterBetween` operator (Postgres `BETWEEN`, DuckDB `BETWEEN`)
29. Add `SortOnField` multi-field companion — `SortOnFields([]SortSpec)`
30. Consider a `PlannedTable` abstraction shared across SQL engines (SQLite, PG, DuckDB share the same pattern)

### Soak/Scale Tests
31. Add 1M-event soak test for pgengine (JSONB pushdown at scale)
32. Add 1M-event soak test for duckdbengine (columnar scan at scale)
33. Add memory-bounded scan test for pgengine (verify no unbounded growth during scan)
34. Add memory-bounded scan test for duckdbengine

### Documentation
35. Add ADR for pgengine PushdownScan + LayoutPlanner design decisions
36. Add ADR for duckdbengine PushdownScan design decisions
37. Document the JSONB operator choice (`->` vs `->>`) and why type preservation matters
38. Update the meta-engine-design.md planning doc to reflect pushdown is shipped for both engines
39. Add code examples to AGENTS.md showing FilterOnField + pgengine/duckdbengine usage

### cqrs-lint Remaining Work
40. Address ~29 open items in cqrs-lint improvement backlog (Pareto plan)
41. Run cqrs-lint against real consumer projects (Kernovia, Standup-Killer, bank-sync, cqrs-htmx, DiscordSync)
42. Add migration paths to cqrs-lint findings (L1.16)
43. Add doc links to cqrs-lint findings (L1.17)
44. Add domain-based severity calibration to cqrs-lint (L1.5)

### metaengine-gen
45. Design CLI interface for `metaengine-gen` (typed Store methods from query declarations)
46. Implement AST parsing of query declarations
47. Generate typed Scan/Get methods from FilterOnField/SortOnField declarations

### Cross-cutting
48. Add `nix run .#check-layers` to verify dependency budgets for new pgengine/duckdbengine imports
49. Run `nix run .#verify` to get the full CI gate result (build + vet + test + race + lint + doc-check)
50. Tag new versions of pgengine/v4 and duckdbengine/v4 (PushdownScan + LayoutPlanner are new exported API)

---

## G. Questions I Cannot Answer Myself

1. **Should DuckDB engine store data as native columnar (STRUCT columns) instead of JSON VARCHAR?** This is an architectural decision: the current JSON-in-VARCHAR approach is consistent with the SQLite/Postgres pattern but wastes DuckDB's columnar advantage. Switching to STRUCT would make LayoutPlanner mandatory for DuckDB (you can't query undeclared fields in a STRUCT), which is a breaking change in how queries are declared. I don't know if you want DuckDB to be "SQLite but columnar" or "a fundamentally different analytical engine."

2. **Should the `limit+1` convention be part of the PushdownScan contract documentation?** The SQLite engine returns `limit+1` rows for has-more detection, and I matched this in both new engines. But the `PushdownScan` interface doc doesn't mention this — a new engine implementor would have to read the SQLite source to discover it. Should I add this to the interface contract, or is it intentionally undocumented?

3. **Should pgengine get GIN indexes now, or wait until there are FilterSpec operators that use them?** GIN indexes only help with `@>`/`?`/`?|`/`?&` operators, which don't exist in FilterSpec yet. Adding GIN without the operators is dead code. But adding the operators without GIN means full-table scans on Postgres. Which comes first — the operators or the indexes?
