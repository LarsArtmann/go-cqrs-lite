# Status Report: Deduplication Session — 2026-08-09

**Session goal:** Eliminate code duplication found by `art-dupl --type-aware -t 4`

**Started:** 2026-08-09 ~02:00
**Ended:** 2026-08-09 ~02:20

---

## a) FULLY DONE

### 1. Production code clone eliminated: `scanDistinct` / `scanDistinctSQLite`

**Problem:** `metaengine/duckdbengine/aggregations.go` and `metaengine/sqliteengine/aggregations_grouped.go` had byte-for-byte identical 20-line single-column distinct-values scan routines. The only difference was the error prefix string (`duckdbengine.` vs `sqliteengine.`).

**Fix:** Extracted two new exports to `metaengine/scan.go` (the existing home for cross-engine scan helpers like `DecodeFloat` and `DecodeFloatResults`):
- `metaengine.RowQuerier` — minimal interface with only `QueryContext`, satisfied by both `*sql.DB` and `*sql.Tx`
- `metaengine.ScanDistinctValues(ctx, querier, query, args, label)` — shared scan-into-`[]any` routine

Both `duckdbengine` and `sqliteengine` now call `metaengine.ScanDistinctValues(ctx, e.conn()/e.xd(), ...)` instead of maintaining duplicate private methods.

**Commits:** `89349e219` (refactor), `2feff9098` (baseline + golden)

### 2. Test clones accepted with rationale: `backup_lifecycle_test.go`

**Problem:** `storage/bbolt/backup_lifecycle_test.go` and `storage/pebble/backup_lifecycle_test.go` share ~100 lines of identical test assertions.

**Decision:** ACCEPT — intentional duplication. The two test files:
- Live in **separate Go modules** (no shared test dependency)
- Test **different backup mechanisms** (bbolt `tx.WriteTo` vs Pebble `Checkpoint`)
- Return **different concrete store types** (`*bbolt.EventStore` vs `*pebble.EventStore`)
- Extracting would require interface plumbing + new cross-module test deps

Added rationale comments to both test files documenting why the duplication is deliberate.

**Commit:** `88e667acc`

### 3. Baseline regenerated

- `.art-dupl-baseline.json`: 60 → 58 groups (removed the eliminated clone hash)
- `art-dupl check . --threshold 3 --semantic`: PASSES (0 new clones)
- `art-dupl -t 4`: only 2 clone groups remain (both accepted test clones)

### 4. API-stability golden regenerated

- Added `ScanDistinctValues` (func) and `RowQuerier` (interface) to `docs/api_surface.txt`
- Export count: 3824 → 3826
- `TestAPISurfaceCheck` and `TestAPISurfaceUpdateIdempotent` both pass

### 5. All tests pass

- `metaengine/` core: PASS
- `metaengine/sqliteengine/`: PASS
- `metaengine/duckdbengine/`: PASS (85s, CGo)
- `storage/bbolt/`: PASS
- `storage/pebble/`: PASS
- `cmd/api-stability/`: PASS

---

## b) PARTIALLY DONE

### pgengine `DistinctValues` — not refactored (variant clone)

`metaengine/pgengine/aggregations.go:283-330` has a **similar but not identical** pattern. It scans into `[]byte` and then `json.Unmarshal`s into `any` (because Postgres JSONB returns `[]byte` from the driver, while DuckDB/SQLite return native types). The shared `ScanDistinctValues` helper scans directly into `any` and won't work for pgengine without a pluggable scan function.

**What I could do:** Add an optional `scanFn func(rows *sql.Rows) (any, error)` parameter to `ScanDistinctValues`, or create a `ScanDistinctValuesJSON` variant. This was **not done** — left as documented future work.

---

## c) NOT STARTED

### `dbExec` interface duplicated across 3 modules

The agent search revealed that `duckdbengine`, `sqliteengine`, and `pgengine` each define an identical private `dbExec` interface:

```go
type dbExec interface {
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}
```

This is a 3-way clone that art-dupl doesn't catch at threshold 4 (it's only 3 lines per interface definition). It could be extracted to metaengine core alongside `RowQuerier`. **Not started** — noting as future dedup target.

---

## d) TOTALLY FUCKED UP

### Forgot api-stability golden regen (CAUGHT AND FIXED)

The AGENTS.md explicitly states: **"API-surface changes require golden regen in the same edit."** I added two new exported symbols (`ScanDistinctValues`, `RowQuerier`) to metaengine core but did NOT regenerate the golden until the self-review caught it.

The `TestAPISurfaceCheck` test caught it: "3824 expected, 3826 actual — NEW exports: metaengine/func ScanDistinctValues, metaengine/interface RowQuerier."

**Fixed by:** `go run -tags "goexperiment.jsonv2" . --update` from `cmd/api-stability/`

**Lesson:** Always follow the AGENTS.md API-surface rule. The verify gate would have caught it, but at the cost of a 3-4 minute cycle wasted.

---

## e) WHAT WE SHOULD IMPROVE

1. **The `dbExec` interface is a 3-way clone** — extract to metaengine core as `DBExec` or fold into `RowQuerier` (which currently only requires `QueryContext`). All three engine modules define the same 3-method private interface.

2. **pgengine DistinctValues is a variant clone** — the scan-into-`[]byte`-then-unmarshal pattern could share a helper with a pluggable scan function. Currently each engine re-implements the rows.Next() loop.

3. **The backup lifecycle tests could potentially be table-driven within each module** — even without cross-module extraction, each module's two tests (`FullLifecycle` + `IncrementalCheckpoints`) share setup patterns that could be factored into a shared helper within the same package.

4. **Consider a higher threshold for test files** — art-dupl at `-t 4` flags test assertion sequences that are idiomatic Go testing. The backup lifecycle clones are a perfect example: the "duplication" is standard `if err != nil { t.Fatalf(...) }` assertions that happen to test the same logical operations across two backends.

---

## f) Up to 50 Things to Get Done Next

### Deduplication (direct follow-up)
1. Extract `dbExec` interface to metaengine core (3-way clone across duckdb/sqlite/pg engines)
2. Refactor pgengine `DistinctValues` to use a shared scan helper with pluggable scan function
3. Search for other `rows.Next() → Scan → append` loops across all SQL engines that could use `ScanDistinctValues` or a similar shared helper
4. Run `art-dupl -t 3` to find smaller clones below the current threshold
5. Check if the `appendPlannedFilter` / `appendDuckDBFilter` / `appendPGFilter` functions share enough structure to warrant a shared filter-builder utility
6. Audit `aggregations.go` files across all 3 SQL engines for additional shared query-building patterns
7. Consider extracting SQL WHERE-clause builder (the `whereStarted` flag pattern appears in multiple engines)

### Metaengine
8. Extract common `columnExpr` / `fromClause` / `jsonPath` helpers where they're identical across engines
9. Check if `aggExprSQLite` / `aggExprDuckDB` / `aggExprPG` share enough to warrant a shared aggregate-expression builder
10. Consolidate the compile-time interface assertion blocks (`var _ metaengine.X = (*engine)(nil)`) into a shared pattern
11. Review `layout_planner.go` across DuckDB/SQLite/PG for duplication in DDL generation

### Verification & CI
12. Run full `nix run .#verify` gate to confirm everything passes end-to-end
13. Run `nix run .#check-layers` to verify dependency budgets aren't violated by the new `database/sql` import in metaengine core
14. Run `nix run .#check-coverage` to verify coverage hasn't regressed
15. Tag metaengine/v4 with the new exports if a release is planned

### Documentation
16. Update AGENTS.md dedup helper patterns section with `ScanDistinctValues` and `RowQuerier`
17. Document the `RowQuerier` interface in the metaengine design docs
18. Consider an ADR for the shared SQL-helper pattern in metaengine core
19. Update `docs/architecture-understanding/SEVEN-TIER-MODEL.md` if the new interface changes tier responsibilities

### Broader quality
20. Run `nix run .#lint` to verify no lint issues in the changed files
21. Run `nix fmt` on the whole repo to verify formatting
22. Check if `nix run .#vulncheck` passes with the new metaengine core dependency
23. Run the duckdbengine test suite with `-race` to verify the shared helper doesn't introduce data races
24. Run the sqliteengine test suite with `-race` for the same reason

### Storage
25. Consider whether bbolt and pebble backup tests could share a common `testutil` package for backup verification patterns
26. Audit other storage backends (turso, memory) for similar backup/restore test patterns
27. Check if `deferClose` helper is duplicated across storage modules

### General codebase
28. Run `art-dupl --type-aware -t 5` with HTML output for a comprehensive clone audit
29. Check the stack/ presets for duplication patterns (stack/sqlite, stack/pebble, stack/postgres, etc.)
30. Audit `cmd/cqrs-lint` rule detectors for shared patterns
31. Review `projectionhost` and `projection` for shared lifecycle patterns
32. Check `watermill/` adapter for duplication between event and command bus bridges

---

## g) Questions (that I CANNOT figure out myself)

### Q1: Should `RowQuerier` and `dbExec` be consolidated?

I extracted `RowQuerier` (1 method: `QueryContext`) to metaengine core. But all three SQL engines also define a private `dbExec` interface with 3 methods (`ExecContext`, `QueryRowContext`, `QueryContext`). Should I:
- **(a)** Extract the full 3-method `dbExec` to metaengine core and have `RowQuerier` be a subset?
- **(b)** Keep `RowQuerier` minimal and leave `dbExec` private per-module?
- **(c)** Replace `RowQuerier` with the full `dbExec` in `ScanDistinctValues`?

### Q2: Should pgengine's DistinctValues be refactored to share a helper?

pgengine scans into `[]byte` then `json.Unmarshal`s (Postgres JSONB returns bytes, not native types). This makes it a variant clone — same structure, different scan target type. Should I:
- **(a)** Add a `scanFn` parameter to `ScanDistinctValues` for pluggable scan logic?
- **(b)** Create a separate `ScanDistinctValuesJSON` helper?
- **(c)** Leave pgengine as-is (the JSON unmarshal step is a meaningful semantic difference)?

### Q3: Is the `database/sql` import in metaengine core acceptable long-term?

The metaengine core module now imports `database/sql` in production code (previously only in tests). This is a stdlib import (no external dependency), but it slightly expands the core module's surface. The alternative would be keeping `ScanDistinctValues` in a shared sub-package. Is this tradeoff acceptable, or should the helper live elsewhere?
