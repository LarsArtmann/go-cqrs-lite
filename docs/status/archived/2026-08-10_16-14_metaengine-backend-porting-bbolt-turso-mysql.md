# Status: Metaengine Backend Porting — bbolt, turso, mysql

> **ARCHIVED 2026-08-11 — Substantive work complete. Remaining open items (engine test parity gaps, compile-time assertion gaps, bench fold reflect.Call panic, pebble CounterIncrement benchmark) harvested into TODO_LIST.md Phase 7. Original content retained below for historical context.**

**Date:** 2026-08-10 16:14
**Session scope:** Phase 4 — Port remaining metaengine backend drivers (bbolt, turso, mysql)

---

## Executive Summary

Ported 3 new metaengine engine modules: `bboltengine`, `tursoengine`, `mysqlengine`.
All compile, pass `go vet`, pass tests (including `-race`), and pass the ADT matrix parity
tests. 6 of 8 planned engines already existed (pebble, bbolt was partially done as `storage/bbolt`
but NOT as a metaengine engine, postgres, duckdb, badger, dgraph). This session created the
3 missing ones. All 9 driver names now self-register: memory, sqlite, pebble, bbolt, duckdb,
postgres, mysql, badger, dgraph, turso.

**However**, the implementation has real gaps — missing compile-time assertions, missing test
coverage categories, no skill docs update, and no calibration benchmarks. Details below.

---

## a) FULLY DONE

### `metaengine/bboltengine/` — KV engine (B+tree)

- [x] `go.mod` with `go.etcd.io/bbolt v1.5.0` dependency
- [x] `engine.go` — struct, `NewBboltEngine(dir)`, `NewBboltEngineFromDB(db)`, Profile,
      Close, HealthCheck, keycodec aliases, compile-time assertions for 10 interfaces
- [x] `seq_seeding.go` — restart-safe seq counter seeding (log/multimap/stream/journal)
      using bbolt cursor prefix scans, modeled after pebbleengine
- [x] `map_backends.go` — MapBackend, MapUpdater (atomic RMW via single bbolt tx),
      ScanBackend with sort+paginate
- [x] `backends.go` — SetBackend, CounterBackend, MultimapBackend, LogBackend
- [x] `stream_log.go` — StreamLogBackend, AtomicAppender, StreamingScan
- [x] `register.go` — `RegisterDriver("bbolt", ...)` self-registration
- [x] Tests: ADT matrix (11 ADTs, 8 pass + 3 skip correctly), healthcheck (healthy + closed),
      record stamping, soak (45K events, 0 errors) — ALL PASS including `-race`
- [x] go.work, flake.nix testModules, api-stability modules list — registered
- [x] api-stability golden regenerated (53 new exports across 3 modules)
- [x] `go vet` clean, `go build` clean

### `metaengine/tursoengine/` — libSQL engine (SQLite wire-compatible)

- [x] `go.mod` with `turso.tech/database/tursogo v0.7.2` + sqliteengine replace
- [x] `register.go` — `New(dsn)` opens via tursogo driver, delegates to
      `sqliteengine.NewSQLiteEngine(db)`, `RegisterDriver("turso", ...)`
- [x] Tests: ADT matrix (all pass), profile verification, driver registration — ALL PASS
- [x] Correct architecture: Turso/libSQL is SQLite wire-compatible, so all SQL operations
      delegate to sqliteengine. No SQL duplication needed.
- [x] go.work, flake.nix, api-stability registered

### `metaengine/mysqlengine/` — SQL engine (MySQL dialect)

- [x] `go.mod` with `github.com/go-sql-driver/mysql v1.10.0`
- [x] `engine.go` — struct, `New(dsn)`, `NewFromDB(db)`, DDL with MySQL-specific types
      (VARCHAR(255) PK, JSON columns, AUTO_INCREMENT, backtick-escaped `key`), Profile,
      conn/inTx helpers
- [x] `backends.go` — MapBackend (`?` placeholders, ON DUPLICATE KEY UPDATE),
      CounterBackend (batch UPSERT)
- [x] `scan.go` — ScanBackend (Go-side filter/sort/limit)
- [x] `stream_log.go` — StreamLogBackend, AtomicAppender
- [x] `pushdown.go` — PushdownScan (`value->'$.field'` JSON path operators),
      LayoutPlanner (functional indexes, duplicate-index tolerance)
- [x] `transaction.go` — RunInTx (Transactional interface)
- [x] `register.go` — `RegisterDriver("mysql", ...)`
- [x] Tests: driver registration (PASS), profile + healthcheck + ADT matrix (skip without
      `MYSQL_TEST_DSN` — integration tests need a running MySQL instance)
- [x] go.work, flake.nix, api-stability registered

### Module Registration

- [x] `go.work` — all 3 modules added in alphabetical position
- [x] `flake.nix` testModules — all 3 added (feeds both `#test` and `#lint`)
- [x] `cmd/api-stability/main.go` modules slice — all 3 added
- [x] Meta-tests `TestEveryGoModDirIsInModulesList` + `TestEveryGoModDirIsInTestModules` — PASS
- [x] `docs/api_surface.txt` — regenerated (3928 exports verified)
- [x] AGENTS.md module map updated with new engine names

---

## b) PARTIALLY DONE

### bboltengine

- **Compile-time assertions incomplete**: Has `HealthChecker` method but NO `_ metaengine.HealthChecker = (*bboltEngine)(nil)` assertion. Also missing `StreamingScan` assertion (has the method, no assertion).
- **No persistence test**: badgerengine and pebbleengine both have `persistence_test.go` verifying data survives Close+reopen. bboltengine doesn't.
- **No disk-backed test**: pebbleengine has `disk_backed_test.go` verifying on-disk operation. bboltengine doesn't.
- **No restart safety test**: pebbleengine has `restart_safety_test.go` verifying seq counters survive restart. bboltengine doesn't.
- **No calibration benchmark**: badgerengine/pebbleengine have `calibration_bench_test.go` for cost model calibration. bboltengine uses estimates (5000 ns/op) without measurement.

### mysqlengine

- **Missing `Calibratable` compile-time assertion**: Has `SetCalibration` method but no `_ metaengine.Calibratable = (*mysqlEngine)(nil)` assertion.
- **Missing `HealthChecker` compile-time assertion**: Has `HealthCheck` method but no assertion.
- **No integration test coverage**: ADT matrix, healthcheck, and profile tests all skip without `MYSQL_TEST_DSN`. No MySQL nix integration test exists (unlike pgengine which has testcontainers + nix VM).
- **No calibration benchmark**: Uses PG_NsPerOp estimates copied from pgengine without measurement.
- **No stream_log test**: pgengine has `stream_log_test.go` with specific StreamLog/AtomicAppender tests. mysqlengine doesn't.
- **No pushdown test**: pgengine has `pushdown_test.go` testing JSONB WHERE/ORDER BY pushdown. mysqlengine doesn't.

### tursoengine

- **No record stamp test**: sqliteengine has one, tursoengine should too (though it delegates to sqliteengine, the import path differs).
- **No soak test**: sqliteengine and badgerengine have soak tests.
- **No healthcheck test**: The engine delegates to sqliteengine, but the test should verify HealthChecker is satisfiable through the tursoengine import.

---

## c) NOT STARTED

1. **Skill docs update**: `SKILL.md` and `.agents/skills/go-cqrs-lite/references/*.md` have zero mentions of bboltengine, tursoengine, or mysqlengine. The module map in references/modules.md is stale.
2. **Calibration benchmarks**: None of the 3 new engines have measured cost models. bboltengine uses 5000 ns/op (estimate), mysqlengine uses 12000 ns/op (copied from pgengine), tursoengine inherits sqliteengine's 7000 ns/op.
3. **Dedup baseline update**: The `.art-dupl-baseline.json` was NOT updated. The new modules use `//art-dupl:accept` comments correctly, but `nix run .#check-duplication` may flag new clone patterns if the baseline isn't regenerated.
4. **MySQL Nix integration test**: pgengine has `nix run .#integration-pg` and `nix run .#integration-pg-vm`. MySQL has `nix run .#integration-mysql-nspawn` and `nix run .#integration-mysql-vm` for the STORAGE layer, but there's no metaengine-specific MySQL integration test.
5. **bboltengine `LayoutPlanner`**: badgerengine doesn't have it, but pebbleengine does. bboltengine could support secondary indexes via separate buckets, but this wasn't implemented.
6. **`gofumpt` formatting pass**: Files were written with `edit`/`write` tools, not formatted with `nix fmt` or `gofumpt`. May have formatting inconsistencies.
7. **`nix run .#verify`**: The full verification gate was NOT run. Only individual `go build`, `go test`, `go vet` were run per-module.

---

## d) TOTALLY FUCKED UP

**Nothing is totally fucked up.** All 3 modules compile, pass tests, and don't break existing
engines. But there are real quality gaps that would fail `nix run .#verify`:

1. **`nix fmt` NOT run** — `golines` (max-len: 120) will reformat long lines and may break
   `//nolint` directives. Per AGENTS.md: "Always `nix fmt` BEFORE placing `//nolint` directives."
   The bboltengine `engine.go` has a `//nolint:unused` on `setKey` that could get moved.
2. **`nix run .#check-duplication` NOT run** — new code has heavy structural similarity to
   badgerengine/pebbleengine (especially seq_seeding.go and stream_log.go). The
   `//art-dupl:accept` comments are placed but the baseline golden was NOT regenerated.
3. **`nix run .#check-arch` NOT run** — dependency budget enforcement not verified for the
   3 new modules.
4. **`nix run .#lint` NOT run** — golangci-lint with 202 rules not run on new code.
5. **MySQL `ExplainableScan`/`ExplainableAggregate` NOT implemented** — pgengine has these
   (explain.go), mysqlengine doesn't. Both are part of the metaengine optional interface surface.

---

## e) WHAT WE SHOULD IMPROVE

### Code Quality

1. **Add missing compile-time assertions**: mysqlengine needs `Calibratable` + `HealthChecker`;
   bboltengine needs `HealthChecker` + `StreamingScan`.
2. **Run `nix fmt`** before any `//nolint` placement, then verify nolint positions survive.
3. **Run `nix run .#verify`** (or at minimum `#verify-fast`) before claiming GREEN.
4. **Update `.art-dupl-baseline.json`** via `art-dupl baseline . --threshold 3 --semantic`.
5. **Add MySQL `ExplainableScan`/`ExplainableAggregate`** for parity with pgengine.
6. **Add bboltengine persistence/restart-safety tests** matching pebbleengine coverage.

### Architecture

7. **Consider extracting shared SQL engine helpers** — mysqlengine/backends.go, scan.go,
   stream_log.go, transaction.go are 80%+ identical to pgengine equivalents. The
   `//art-dupl:accept cross-module SQL engine pattern` comments acknowledge this, but a
   shared `metaengine/sqldsl` or `metaengine/sqlengine` package could eliminate the drift
   permanently. This is a bigger refactor but would pay off when adding more SQL engines.
8. **Consider extracting shared KV engine helpers** — bboltengine's seq_seeding.go is
   nearly identical to pebbleengine's. A shared `metaengine/kvseq` package could serve both.
9. **tursoengine should expose a distinct Profile name** — currently returns "sqlite" via
   delegation. Should return "turso" for operator visibility, even though cost profile is
   identical. This means wrapping the engine to override Profile().

### Testing

10. **Add MySQL integration test infrastructure** — either testcontainers (like pgengine)
    or nix-based (like storage layer). Currently zero integration coverage.
11. **Add calibration benchmarks** for all 3 engines — cost models are guesses/estimates.
12. **Add `ginkgo`/`gomega` BDD tests** for mysqlengine stream_log and pushdown (matching
    pgengine's test style).

### Documentation

13. **Update SKILL.md** with the 3 new engines in the module map and decision matrix.
14. **Update `references/modules.md`** with per-module entries.
15. **Add MySQL-specific gotchas** to AGENTS.md (JSON path syntax, `key` reserved word,
    no `CREATE INDEX IF NOT EXISTS`).

---

## f) Up to 50 Things to Get Done Next

#### P0 — CI-blocking (would fail `nix run .#verify`)

1. Run `nix fmt` on all 3 new modules
2. Run `nix run .#lint` and fix all lint findings
3. Run `nix run .#check-duplication` and update `.art-dupl-baseline.json`
4. Run `nix run .#check-arch` and verify dependency budgets
5. Add `_ metaengine.HealthChecker` assertion to bboltengine engine.go
6. Add `_ metaengine.HealthChecker` assertion to mysqlengine engine.go
7. Add `_ metaengine.Calibratable` assertion to mysqlengine engine.go
8. Add `_ metaengine.StreamingScan` assertion to bboltengine engine.go
9. Verify `//nolint` directives survive `nix fmt` reformatting
10. Run full `nix run .#verify` gate

#### P1 — Completeness gaps

11. Add bboltengine `persistence_test.go` (data survives Close+reopen)
12. Add bboltengine `restart_safety_test.go` (seq counters survive restart)
13. Add bboltengine `disk_backed_test.go` (on-disk operation)
14. Add bboltengine `calibration_bench_test.go` (measure real ns/op)
15. Add mysqlengine `stream_log_test.go` (StreamLog + AtomicAppender contract)
16. Add mysqlengine `pushdown_test.go` (JSON WHERE/ORDER BY pushdown)
17. Add mysqlengine `explain.go` (ExplainableScan + ExplainableAggregate)
18. Add mysqlengine `calibration_bench_test.go`
19. Add tursoengine `record_stamp_test.go`
20. Add tursoengine `soak_autocrud_test.go`
21. Add tursoengine `healthcheck_test.go`

#### P2 — Documentation

22. Update `SKILL.md` module map with 3 new engines
23. Update `references/modules.md` with per-module entries
24. Update `references/recipes.md` with engine selection guidance
25. Add MySQL dialect gotchas to AGENTS.md Gotchas section
26. Add bbolt single-writer model note to AGENTS.md
27. Update `docs/METAENGINE_DOMAIN_LANGUAGE.md` with 3 new engine names
28. Document tursoengine delegation pattern (why it's thin)

#### P3 — Integration testing

29. Add MySQL testcontainer support (like pgengine's testcontainer_test.go)
30. Add `nix run .#integration-mysql-metaengine` (metaengine-specific MySQL test)
31. Wire mysqlengine ADT matrix to use testcontainers when available
32. Add bboltengine to `metaengine/bench/` cross-engine benchmark suite
33. Add tursoengine to `metaengine/bench/` cross-engine benchmark suite
34. Add mysqlengine to `metaengine/bench/` (needs running MySQL)
35. Add cross-engine parity test including all 3 new engines simultaneously

#### P4 — Architecture improvements

36. Extract shared SQL engine helpers into `metaengine/sqldsl` or similar
37. Extract shared KV seq seeding into `metaengine/kvseq` package
38. Override tursoengine Profile() to return "turso" instead of "sqlite"
39. Add bboltengine LayoutPlanner (secondary index buckets)
40. Add mysqlengine SetBackend (meta_set table)
41. Add mysqlengine MultimapBackend (meta_multimap table)
42. Add mysqlengine LogBackend (meta_log table)
43. Consider mysqlengine MapUpdater (atomic RMW via SELECT FOR UPDATE)

#### P5 — Polish

44. Add `const MySQLNsPerWrite` (currently only Op and Read)
45. Add `const BboltNsPerWrite` usage in Profile (currently set but not differentiated)
46. Add doc.go for each new engine package (package-level docs exist but could be richer)
47. Add example_test.go showing how to use each engine
48. Verify `nix run .#vulncheck` passes for new modules
49. Check `nix run .#check-coverage` for coverage drift
50. Tag releases for the 3 new modules (`scripts/tag-release.sh`)

---

## g) Questions (things I genuinely cannot determine)

### Q1: Should mysqlengine implement SetBackend, MultimapBackend, and LogBackend?

pgengine does NOT implement these (only MapBackend, CounterBackend, ScanBackend, PushdownScan,
LayoutPlanner, StreamLogBackend, AtomicAppender, Transactional). I matched pgengine parity.
But sqliteengine DOES implement SetBackend/MultimapBackend/LogBackend. Should mysqlengine
be a full-featured SQL engine matching sqliteengine, or a minimal engine matching pgengine?

**Context**: This affects whether we need `meta_set`, `meta_multimap`, `meta_log` tables in
the MySQL DDL. My decision was "match pgengine" but I'm not sure if pgengine is intentionally
minimal or just not yet complete.

### Q2: Should tursoengine override Profile() to return "turso"?

Currently tursoengine delegates entirely to sqliteengine, so `Profile().Name` returns "sqlite".
This is technically correct (same cost profile) but operationally misleading — an operator
querying registered engines would see "turso" in the driver list but "sqlite" in the profile.
Should I wrap the engine to override just the Name field? This adds complexity for cosmetic benefit.

### Q3: Is the pre-existing `enginetest/record_stamp.go` compile error known?

The file references `id.NewSystemActor` (undefined) and uses `id.NewCorrelationID()` as a
string when it returns a struct type. This breaks all engines' record_stamp tests in
`GOWORK=off` mode (standalone module builds). In workspace mode the workspace go.mod
resolves the version mismatch. This is pre-existing (not introduced by my changes) but
blocks standalone module testing for ALL engines, not just the new ones.
