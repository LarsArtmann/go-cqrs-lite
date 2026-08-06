# Session: SQLite Benchmark Fairness + CGo Driver Support

Date: 2026-08-06 14:06
Session goal: Make SQLite benchmarks realistic (enable optimizations) and add mattn/go-sqlite3 CGo driver for comparison

---

## What Was Done

### a) FULLY DONE

1. **Root cause analysis of SQLite vs Pebble gap** — Identified three factors: (1) `MaxOpenConns(1)` serializes all writers, (2) pure-Go `modernc.org/sqlite` driver is 3-5x slower per operation, (3) `Optimize: false` in `defaultConfig()` left SQLite with its builtin 2MB cache vs Pebble's bloom filters + concurrent compactions.

2. **`WithDriverName` option in `stack/sqlite/preset.go`** — New exported option that overrides the `database/sql` driver name used by `sql.Open`. Default is `"sqlite"` (modernc.org/sqlite, pure-Go). Set to `"sqlite3"` to use `mattn/go-sqlite3` (CGo, 3-5x faster). Skips modernc-specific `EnsureSQLiteDSNBusyTimeout` DSN rewriting for non-default drivers. Updates `CGoRequired` on `stack.Capabilities` to `true` when driver != `"sqlite"`.

3. **Driver name threading in `stack/sqlite/multidb.go`** — `openSecondaryDB` now uses `cfg.driverName` for `sql.Open` and conditionally applies the modernc DSN busy_timeout helper.

4. **CGo/non-CGo driver registration in `cmd/cqrs-bench/`** — Two new files following the DuckDB pattern:
   - `factory_sqlite_cgo.go` (`//go:build cgo`) — blank-imports `github.com/mattn/go-sqlite3`, sets `sqliteCgoAvailable = true`
   - `factory_sqlite_nocgo.go` (`//go:build !cgo`) — `sqliteCgoAvailable = false` sentinel

5. **`sqlite-cgo` backend in `cmd/cqrs-bench/factory.go`** — Merged with existing `sqlite` case. Accepts `sqlite-cgo` and `sq3` aliases. Checks `sqliteCgoAvailable` and gives a helpful error if CGo is disabled. Passes `sqlite.WithDriverName("sqlite3")`.

6. **Optimizations enabled by default for ALL SQLite bench variants** — Both `sqlite` and `sqlite-cgo` backends now apply `sqlite.WithPragmas(sqlopt.WithOptimizations())` which sets `cache_size=-65536` (64MB), `temp_store=MEMORY`, `mmap_size=268435456` (256MB). This replaces SQLite's conservative 2MB default cache.

7. **Help text updates** — `main.go` longDesc now lists `sqlite-cgo` backend with tradeoff description. `flags.go` updated `Backend` and `Backends` flag help strings. Unknown-backend error message updated.

8. **`mattn/go-sqlite3` promoted to direct require** — `go mod tidy` with `CGO_ENABLED=1` promoted the dependency from `// indirect` to direct in `cmd/cqrs-bench/go.mod`.

9. **API stability golden regenerated** — `docs/api_surface.txt` updated to include `stack/sqlite/func WithDriverName` (3556 exports).

10. **Build verification** — Both `CGO_ENABLED=0 go build` and `CGO_ENABLED=1 go build` pass. `stack/sqlite` and `cmd/cqrs-bench` tests pass. Manual benchmark run verified:
    - `sqlite-cgo` is 2.2x faster on writes at `small` profile (188µs vs 409µs)
    - Write amplification dropped from 44.3x to 11.3x with optimizations enabled
    - `sqlite-cgo` has lower GC pause (528µs vs 8.5ms) and lower P99 tail (1.1ms vs 2.6ms) vs pure-Go sqlite

### b) PARTIALLY DONE

1. **mattn/go-sqlite3 error classification compatibility** — `storage/sql/duplicate.go` uses a structural `sqliteCodeError` interface with `Code() int`. mattn's `sqlite3.Error` type returns `Code() ExtendedSQLiteErrorCode` (a different type). This means mattn errors may NOT match the structural interface for duplicate-key detection. There is a string-matching fallback (`"UNIQUE constraint failed"`) that should work, but this is fragile and untested. The bench tool doesn't exercise duplicate-key paths, so this is not blocking for benchmarking but IS blocking for production use of mattn as the driver.

2. **mattn DSN parameter compatibility** — modernc.org/sqlite and mattn/go-sqlite3 use different DSN parameter syntax. The code conditionally skips `EnsureSQLiteDSNBusyTimeout` for non-modernc drivers, but mattn also needs `_busy_timeout=5000` in the DSN (or a PRAGMA exec). Currently mattn gets busy_timeout via the PRAGMA exec in `SQLiteEnableWAL` which sets it per-connection. With `MaxOpenConns(1)` this is fine. With multi-DB mode (where secondary DBs don't cap the pool), it could be a problem. Not tested.

### c) NOT STARTED

1. **No unit test for `WithDriverName`** — The new option has no test verifying it threads correctly through `openBackend` and `openSecondaryDB`.

2. **No test for the CGo path** — Only verified via manual benchmark run. No automated test that `sqlite-cgo` backend produces a working bundle.

3. **No test for optimization effect** — No test verifying that `WithOptimizations()` actually applies the pragmas (cache_size, temp_store, mmap_size).

4. **Full `nix run .#verify` gate not run** — Only ran targeted tests for `stack/sqlite` and `cmd/cqrs-bench`. The full verify gate (build + vet + test + race + lint + doc-check + api-stability) was not executed.

5. **`Capabilities.Backend` still hardcoded to `"sqlite"`** — Even when using the mattn driver, the Bundle's `Capabilities.Backend` field reports `"sqlite"`. Should report `"sqlite3"` or `"sqlite-cgo"` for accurate introspection.

6. **AGENTS.md not updated** — The module table doesn't mention `sqlite-cgo`. The bench help text section doesn't reflect the new backend.

7. **cmd/cqrs-bench/README.md not updated** — No documentation of the sqlite-cgo backend or the optimization change.

### d) TOTALLY FUCKED UP

Nothing. All changes build (both CGo and non-CGo), tests pass, and the benchmark runs correctly with all four backends.

---

## What We Should Improve

### Immediate (this session's gaps)

1. **mattn error classification gap** — `storage/sql/duplicate.go`'s `sqliteCodeError` interface expects `Code() int`, but mattn returns `Code() ExtendedSQLiteErrorCode`. Need to verify whether the structural interface match fails, and if so, add a mattn-specific path or broaden the interface. This is a silent correctness issue for any production code that uses mattn with the library.

2. **No test for the CGo path** — The `sqlite-cgo` backend works (verified by manual benchmark), but there's no automated test. A test should verify: (a) the bundle opens, (b) events round-trip, (c) duplicate-key detection works. This is especially important because the CGo path has a different error type.

3. **mattn busy_timeout DSN** — mattn/go-sqlite3 accepts `_busy_timeout=N` as a DSN query param. We skip `EnsureSQLiteDSNBusyTimeout` for non-modernc drivers, but we should add an equivalent for mattn to ensure pool-level coverage (relevant for multi-DB mode where secondary DBs don't cap the pool at 1).

### Broader improvements

4. **Compare default should auto-include sqlite-cgo when available** — Currently `--backends` defaults to `memory,sqlite,pebble`. When `sqliteCgoAvailable` is true, the default could be `memory,sqlite,sqlite-cgo,pebble` so users automatically see the comparison.

5. **`--codec cbor` should be tested with sqlite-cgo** — Different drivers may handle BLOB columns differently. mattn stores binary data as `[]byte`; modernc may differ.

---

## Up to 50 Things We Should Get Done Next

### Correctness & Testing (1-10)

1. Verify mattn/go-sqlite3 error classification works with `storage/sql/duplicate.go` — run a duplicate-key insert test with CGo enabled
2. Add unit test for `WithDriverName` option — verify driver name threads through to `sql.Open`
3. Add integration test for `sqlite-cgo` backend — full event round-trip (Save → Load → verify payload)
4. Add test for `WithOptimizations()` — verify pragmas are applied (query `PRAGMA cache_size` after setup)
5. Run `nix run .#verify` to confirm the full gate passes with all changes
6. Run `nix run .#verify` with `CGO_ENABLED=1` — the CI may only test CGo-free builds
7. Add mattn busy_timeout DSN helper (equivalent to `EnsureSQLiteDSNBusyTimeout` but for `_busy_timeout=` syntax)
8. Test multi-DB mode with mattn driver (event/query/view DB separation)
9. Add `sqlite-cgo` to the `cmd/cqrs-bench/main_test.go` backend list if it has one
10. Verify `Capabilities.Backend` reports the correct driver name, not always `"sqlite"`

### Performance & Benchmarking (11-20)

11. Run `medium` or `large` profile comparison to see if the CGo advantage scales beyond `small`
12. Benchmark with `--codec cbor` to see if encoding is a bottleneck for sqlite-cgo
13. Benchmark `read-heavy` profile with sqlite-cgo to compare read path
14. Add a `--batch-size` sweep for sqlite-cgo to find the optimal batch size
15. Compare `synchronous=NORMAL` vs `synchronous=OFF` (durability relaxed) for sqlite-cgo
16. Profile sqlite-cgo with `--cpuprofile` to identify remaining bottlenecks
17. Test if `PRAGMA wal_autocheckpoint` tuning improves sqlite-cgo performance
18. Benchmark with `--payload-sizes 64,256,4096,16384` to see payload size effect
19. Add `mmap_size` sweep — test 0, 64MB, 256MB, 1GB to find optimal for the workload
20. Compare modernc sqlite with optimizations vs without — quantify the pragma impact

### Documentation (21-25)

21. Update `cmd/cqrs-bench/README.md` with sqlite-cgo backend description and examples
22. Update `AGENTS.md` module table — add `sqlite-cgo` to the bench backends
23. Add a "Driver Selection Guide" section to cqrs-bench README (when to use sqlite vs sqlite-cgo vs pebble)
24. Document the mattn error classification compatibility note in `storage/sql/duplicate.go`
25. Update `docs/api_surface.txt` after any further API changes

### Architecture (26-30)

26. Consider making `WithDriverName` validate that the driver is actually registered (sql.Open succeeds but fails on first query if driver missing)
27. Consider a `WithMattnDriver()` convenience option that sets driver name + registers the driver
28. Consider extracting DSN parameter helpers into a `storage/sql/dsn.go` per-driver module
29. Evaluate whether `ConfigureSQLitePool` (`SetMaxOpenConns(1)`) is correct for mattn — mattn supports concurrent access better than modernc in some configs
30. Consider adding `sqlite-cgo` to the `compare` command's auto-detect logic (include if CGo available)

### CI & Integration (31-35)

31. Add a CI job variant that runs benchmarks with `CGO_ENABLED=1` to exercise the sqlite-cgo path
32. Ensure `go mod tidy` in CI runs with both `CGO_ENABLED=0` and `CGO_ENABLED=1` to keep dep classification correct
33. Verify the api-stability `TestEveryGoModDirIsInModulesList` still passes
34. Run the `cmd/doc-check` tool against updated docs
35. Add `mattn/go-sqlite3` to the dependency budget check (`nix run .#check-layers`)

### Lower Priority (36-50)

36. Explore `sqlite3_preupdate_hook` (mattn supports it, modernc does not) for change data capture
37. Explore mattn's `_txlock=immediate` DSN param for write-heavy workloads
38. Consider WAL mode + shared cache for mattn (`?cache=shared`) to enable read concurrency with pool > 1
39. Benchmark mattn with `MaxOpenConns > 1` + WAL — WAL allows concurrent readers
40. Explore SQLite's `PRAGMA mmap_size` with larger values (1GB+) on systems with enough RAM
41. Consider `PRAGMA page_size=4096` (or larger) at DB creation time for better I/O alignment
42. Test if `PRAGMA journal_size_limit` reduces WAL file growth and improves performance
43. Explore mattn's `RegisterAggregator` for custom SQL functions (not available in modernc)
44. Consider a `sqlite-native` backend alias that auto-detects CGo and picks the best driver
45. Benchmark the `raw sink` phase specifically for sqlite-cgo to isolate storage write cost
46. Compare Turso (libSQL) vs sqlite-cgo — both are native SQLite forks
47. Explore `PRAGMA optimize` at connection close for long-running connections
48. Test concurrent read-during-write (`--skip-mixed=false`) with mattn + pool > 1
49. Add a "Driver" column to the comparison table output
50. Consider environment variable `CQRS_BENCH_DRIVER=sqlite3` as an alternative to `--backend sqlite-cgo`

---

## Questions

1. **Should `ConfigureSQLitePool` (`SetMaxOpenConns(1)`) be conditional on the driver?** mattn/go-sqlite3 with WAL mode supports concurrent readers. Setting `MaxOpenConns(1)` for mattn unnecessarily serializes reads. The current code applies it unconditionally. Should we skip the pool cap for mattn, or make it configurable? This affects production use, not just benchmarks.

2. **Should the `compare` command auto-include `sqlite-cgo` when CGo is available?** Currently the default is `memory,sqlite,pebble`. Auto-including `sqlite-cgo` when `sqliteCgoAvailable == true` would give users the full picture automatically, but adds ~30s to every compare run (sqlite-cgo is slower than memory/pebble but faster than sqlite).

3. **Is the mattn error classification gap blocking enough to fix before tagging?** The structural interface `sqliteCodeError` expects `Code() int`, but mattn returns `ExtendedSQLiteErrorCode`. Duplicate-key detection has a string fallback (`"UNIQUE constraint failed"`), so it should work in practice. But the typed classification path silently fails. Should we fix this now or defer to a follow-up?
