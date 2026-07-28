# Status: DuckDB Integration — Brutal Self-Review

**Date:** 2026-07-28 23:03
**Session goal:** Add DuckDB as a storage backend + stack preset
**Verdict:** Shipped working code, but missed lint, docs, and several integration points

---

## A) FULLY DONE (working + verified)

| # | What | Evidence |
|---|------|----------|
| 1 | `DuckDBDialect` in `storage/sql/dialect.go` | Builds, vets, all dialect tests pass |
| 2 | `duckdb.sql` migration + `DuckDBSchemaEmbed()` | Embedded, tested |
| 3 | DuckDB UNIQUE constraint detection in `duplicate.go` | String fallback tested |
| 4 | `NewDuckDBBackend()` + `DuckDBInitSchema()` in storage/ | Builds, compiles |
| 5 | Dialect + duplicate tests in `storage/sql/` | 10+ test cases pass |
| 6 | `stack/duckdb/` module (7 source files) | Full preset: New, multidb, view_models, doc, README |
| 7 | CGo-gated `drivers.go` + `preset_cgo_test.go` | 10/10 tests pass (CGo ON) |
| 8 | `go.work` wiring | Module registered |
| 9 | `flake.nix` testModules + gcc in devShell | Added |
| 10 | `cmd/api-stability` modules list + golden regen | Test passes |
| 11 | `.golangci.yml` depguard allow list | Added `github.com/duckdb/duckdb-go` |
| 12 | `AGENTS.md` updates (modules, tiers, deps, patterns, lint conventions) | Updated |
| 13 | Non-CGo build still works (`CGO_ENABLED=0 go build ./...`) | Verified |
| 14 | Race detector passes | `go test -race` clean |
| 15 | 83.3% test coverage on stack/duckdb | Measured |

---

## B) PARTIALLY DONE (started but incomplete)

| # | What | What's missing |
|---|------|----------------|
| 1 | **Lint compliance** | `golangci-lint` reports 3 issues: 2x `godoclint` (both `doc.go` and `drivers.go` have package comments), 1x `wrapcheck` (`sqlopt.OpenPrimaryBackend` error unwrapped in `preset.go`). I ran `gofumpt` but **never ran `golangci-lint`** — this is inexcusable. |
| 2 | **flake.nix CGO for tests** | Added `gcc` to devShell + goModules, but the `test`/`test-race`/`verify` apps don't set `CGO_ENABLED=1`. So `nix run .#test` would **skip/fail** DuckDB tests because CGo defaults to 0 in Nix environments. The test app needs `CGO_ENABLED=1` in its env or a wrapper. |
| 3 | **AGENTS.md module count** | I wrote "59 go.mod files" and "8 stack presets" but the actual count is 59 go.mod files. The breakdown math (42 library + 8 stack + 3 examples + 5 cmd + 1 root = 59) is correct, but I should verify the "42 library" count is still accurate after adding DuckDB-related changes. |
| 4 | **ROADMAP.md** | Line 138 still says "Phase 2 (`storage/duckdb`, CGO) and Phase 3 (`stack/duckdb`) deferred." — I shipped it but didn't update the roadmap to reflect completion. |

---

## C) NOT STARTED (should have been done)

| # | What | Why it matters |
|---|------|----------------|
| 1 | **Fix lint issues** (godoclint + wrapcheck) | The verify gate will FAIL. This is the #1 priority. |
| 2 | **ADR for CGo introduction** | Introducing CGo to a previously pure-Go project is a **major architectural decision**. Every other significant decision in this project has an ADR. There should be `docs/adr/0070-duckdb-cgo-introduction.md`. |
| 3 | **SKILL.md update** (`.agents/skills/go-cqrs-lite/`) | The AI consumer skill doesn't mention DuckDB at all. The routing table, module table, and recipes are missing DuckDB. |
| 4 | **FEATURES.md** | DuckDB should be listed under DONE features. Currently no mention. |
| 5 | **TODO_LIST.md** | No DuckDB tasks marked as done. |
| 6 | **FOUR-TIER-MODEL.md** | Module graph in `docs/architecture-understanding/FOUR-TIER-MODEL.md` doesn't include `stack/duckdb`. |
| 7 | **Multi-DB contract test** | SQLite has `TestMultiDBContract` in `contract_test.go`. DuckDB should have the same. I only tested multi-DB via `TestNew_MultiDB` (basic wiring), not the full contract suite. |
| 8 | **`view_models_integration_test.go`** | SQLite has comprehensive view model integration tests. DuckDB has none — `SQLViewModel` is exported but untested with DuckDB. |
| 9 | **Golden test for DuckDB schema** | `storage/` has golden tests for postgres/sqlite event/command/snapshot/checkpoint schemas. No DuckDB golden test exists. |
| 10 | **`OpenDuckDB()` + `OpenDuckDBInMemory()` helpers** | `storage/` exports `OpenSQLite()`, `OpenSQLiteInMemory()`. No DuckDB equivalents. |
| 11 | **`ConfigureDuckDBPool()`** | SQLite has `ConfigureSQLitePool()` (caps at 1 connection for WAL). DuckDB may need pool guidance (single-writer or concurrent reader config). |
| 12 | **stack/bench integration** | `stack/bench/` benchmarks backends. DuckDB not included. |
| 13 | **cmd/cqrs-bench integration** | The benchmark CLI doesn't list DuckDB as a backend option. |
| 14 | **`appendDuckDBOptions` unit test** | The DSN query-param appending logic is untested. Edge cases: empty DSN, DSN with existing `?`, DSN with existing `&`, special chars in memory_limit. |
| 15 | **CI pipeline (`ci.yml`)** | GitHub Actions CI doesn't have a CGo-enabled job for DuckDB tests. The existing CI runs with default CGo (likely 0). |
| 16 | **Integration tests** | No test in `integration/` module exercises DuckDB. |
| 17 | **`doc-check` run** | AGENTS.md says to run `cmd/doc-check` after editing docs. I didn't run it to verify DuckDB import paths in markdown. |
| 18 | **Coverage drift check** | `nix run .#check-coverage` not run. |
| 19 | **Deduplication check** | `nix run .#check-duplication` not run. |
| 20 | **ROADMAP.md update** | Phase 2/3 DuckDB entries still say "deferred". |

---

## D) TOTALLY FUCKED UP

| # | What | Impact | Fix |
|---|------|--------|-----|
| 1 | **Never ran `golangci-lint`** | 3 lint failures will break `nix run .#verify`. I ran `gofumpt` and declared "formatting clean" — but `gofumpt` is NOT `golangci-lint`. The `wrapcheck` and `godoclint` issues are real, would have been caught instantly. | Fix the 3 issues, always run lint after changes. |
| 2 | **Claimed "formatting PASS (gofumpt clean)" as if it meant lint passes** | Misleading self-verification. gofumpt only checks formatting, not lint rules. | Never conflate formatting with linting. |
| 3 | **flake.nix test apps don't set CGO_ENABLED=1** | `nix run .#test` will silently skip DuckDB tests (the `//go:build cgo` tag excludes them when CGo is off). The tests I wrote will NEVER RUN in CI/Nix without this fix. This makes the entire test suite I wrote invisible to the standard test pipeline. | Add `CGO_ENABLED=1` to the test app env (or a separate `test-cgo` app). |

---

## E) WHAT WE SHOULD IMPROVE

### Process improvements (for me, the AI)

1. **Always run `golangci-lint` after code changes** — `gofumpt` is formatting only, NOT linting. This is the #1 lesson.
2. **Run `nix run .#verify` (or at least `verify-fast`) before declaring done** — I skipped this entirely and shipped lint failures.
3. **Check CGo propagation in Nix** — writing `//go:build cgo` tests is pointless if the test runner doesn't enable CGo.
4. **Update ALL documentation files** — AGENTS.md is not enough. FEATURES.md, TODO_LIST.md, ROADMAP.md, SKILL.md, FOUR-TIER-MODEL.md all need updates when adding a module.
5. **Write ADRs for architectural decisions** — Introducing CGo is a massive decision that deserves documentation.
6. **Run the existing golden/schema tests** — If storage/ has golden tests for postgres/sqlite schemas, add one for DuckDB in the same change.

### Code improvements

7. **Fix `godoclint`**: `drivers.go` should not have a package comment (only `doc.go` should).
8. **Fix `wrapcheck`**: Wrap the `sqlopt.OpenPrimaryBackend` error return in `preset.go`.
9. **Add `ConfigureDuckDBPool()`**: DuckDB connection pool guidance (like `ConfigureSQLitePool`).
10. **Add DuckDB golden schema tests**: Mirror the postgres/sqlite golden test pattern.
11. **Add multi-DB contract test**: `TestMultiDBContract` with the full `contracttest.RunMultiDBSuite`.
12. **Add `view_models_integration_test.go`**: Test `SQLViewModel` with DuckDB.
13. **Test `appendDuckDBOptions`**: Edge cases for DSN param appending.
14. **Add real DuckDB duplicate-key integration test**: Currently only the string fallback is tested, not an actual DuckDB driver error.

---

## F) Up to 50 Things to Get Done Next

### Critical (blocks verify gate) — P0

| # | Task | Effort |
|---|------|--------|
| 1 | Fix `godoclint`: remove package comment from `drivers.go` | 2 min |
| 2 | Fix `wrapcheck`: wrap `sqlopt.OpenPrimaryBackend` error in `preset.go` | 5 min |
| 3 | Fix flake.nix: add `CGO_ENABLED=1` to test/test-race/verify apps (or separate `test-cgo` app) | 15 min |
| 4 | Run `nix run .#verify` or `verify-fast` to confirm GREEN | 5 min |

### High priority — P1

| # | Task | Effort |
|---|------|--------|
| 5 | Write ADR-0070: DuckDB CGo introduction decision | 30 min |
| 6 | Update SKILL.md (`.agents/skills/go-cqrs-lite/references/modules.md`, `core.md`) with DuckDB | 20 min |
| 7 | Update FEATURES.md: add DuckDB to DONE | 5 min |
| 8 | Update TODO_LIST.md: mark DuckDB tasks done | 5 min |
| 9 | Update ROADMAP.md: mark Phase 2/3 DuckDB as shipped | 5 min |
| 10 | Update FOUR-TIER-MODEL.md: add stack/duckdb to Tier 5 | 5 min |
| 11 | Add `TestMultiDBContract` to stack/duckdb | 20 min |
| 12 | Add `view_models_integration_test.go` to stack/duckdb | 30 min |
| 13 | Add DuckDB golden schema tests to storage/ | 20 min |
| 14 | Run `cmd/doc-check` on AGENTS.md + DuckDB README | 5 min |

### Medium priority — P2

| # | Task | Effort |
|---|------|--------|
| 15 | Add `OpenDuckDB()` + `OpenDuckDBInMemory()` helpers to storage/ | 10 min |
| 16 | Add `ConfigureDuckDBPool()` to storage/ | 10 min |
| 17 | Unit test `appendDuckDBOptions` (DSN edge cases) | 15 min |
| 18 | Add real DuckDB duplicate-key integration test (actual INSERT conflict) | 20 min |
| 19 | Wire DuckDB into `stack/bench/` benchmark suite | 30 min |
| 20 | Wire DuckDB into `cmd/cqrs-bench` CLI | 20 min |
| 21 | Add CGo-enabled job to `ci.yml` GitHub Actions | 20 min |
| 22 | Add DuckDB to `integration/` cross-module tests | 30 min |
| 23 | Run `nix run .#check-coverage` and fix drift | 10 min |
| 24 | Run `nix run .#check-duplication` and update baseline if needed | 10 min |
| 25 | Verify `nix run .#vulncheck` passes for stack/duckdb standalone | 10 min |

### Lower priority — P3

| # | Task | Effort |
|---|------|--------|
| 26 | Add DuckDB preset to `example/taskmanager/` as alternative backend | 1 hr |
| 27 | Document DuckDB-specific performance tuning (columnar scan benefits) | 30 min |
| 28 | Add DuckDB COPY TO Parquet export example to docs | 30 min |
| 29 | Benchmark DuckDB vs SQLite vs Postgres for read-heavy workloads | 2 hr |
| 30 | Add DuckDB-specific `WithReadOnly()` option (access_mode=read_only) | 15 min |
| 31 | Test DuckDB with large payloads (>1MB BLOB) | 30 min |
| 32 | Test DuckDB concurrent read/write behavior | 1 hr |
| 33 | Add DuckDB temp directory configuration option | 15 min |
| 34 | Document binary size impact (~30-50MB) in deployment guide | 15 min |
| 35 | Add DuckDB extension loading support (e.g., `INSTALL httpfs`) | 1 hr |
| 36 | Cross-compile test: DuckDB with CGo on different archs | 1 hr |
| 37 | Add DuckDB version pinning documentation (DuckDB v1.5.5 = duckdb-go v2.10505.0) | 15 min |
| 38 | Consider `storage/duckdb/` as separate module (like pebble/turso) for cleaner isolation | 2 hr |
| 39 | Add DuckDB to the module graph diagram (`docs/perfect-world-modules.svg`) | 30 min |
| 40 | Update `CONTRIBUTING.md` with CGo build instructions for DuckDB | 20 min |
| 41 | Add DuckDB-specific error codes to errorfamily classification | 30 min |
| 42 | Test DuckDB with `scenario/` BDD testing DSL | 30 min |
| 43 | Add DuckDB snapshot store integration test | 20 min |
| 44 | Add DuckDB checkpoint store integration test | 20 min |
| 45 | Test DuckDB with `projectionhost/` (event replay + checkpoint) | 1 hr |
| 46 | Add DuckDB to `cmd/cqrs-lint` feature profile detection | 30 min |
| 47 | Document DuckDB DSN format in storage guide (`docs/storage-guide.md` or similar) | 20 min |
| 48 | Add DuckDB-specific PRAGMA/configuration documentation | 20 min |
| 49 | Consider Arrow integration for DuckDB analytical results | 2 hr |
| 50 | Add `nix run .#test-duckdb` app for CGo-enabled DuckDB-only test run | 15 min |

---

## G) Questions I Cannot Answer Myself

### 1. Should `nix run .#test` set `CGO_ENABLED=1` globally, or should DuckDB tests use a separate `test-cgo` app?

Setting CGO_ENABLED=1 globally would slow down all other module tests (CGo adds overhead) and potentially change behavior. But a separate app means DuckDB tests are invisible to the default test pipeline. The alternative is a build tag like `//go:build cgo` (which I already use) combined with CGO_ENABLED=1 in the default test env — but then ALL modules pay the CGo tax. **What's the preferred Nix pattern: global CGo or a separate test-cgo app?**

### 2. Should DuckDB get its own `storage/duckdb/` module (like pebble/turso) or stay as just a dialect in `storage/sql/`?

Pebble and Turso each have their own `storage/pebble/` and `storage/turso/` modules. DuckDB is currently just a `Dialect` in the shared `storage/sql/` package with no separate `storage/duckdb/` module. This is architecturally simpler (DuckDB uses the same SQL store code as Postgres/SQLite), but breaks the pattern. The research docs originally planned a `storage/duckdb/` module. **Should I follow the existing pattern (just a dialect) or the original plan (separate module)?**

### 3. Should the DuckDB driver dependency (`github.com/duckdb/duckdb-go/v2`) be tagged at `v4.0.0` for the module, or kept at the latest `v2.10505.0`?

The module path is `github.com/larsartmann/go-cqrs-lite/stack/duckdb/v4`, but the DuckDB driver itself is `v2.10505.0`. When publishing the first tagged release of `stack/duckdb/v4`, the `go.mod` requires internal deps at `v4.x.y` but the external DuckDB driver is `v2.x`. This is fine for Go modules (the `/v4` is in OUR path, not the driver's), but I need to know: **should I wait for all internal deps to be tagged at consistent v4 versions before tagging stack/duckdb, or tag immediately?**

---

## Test Results Summary (as of right now)

```
stack/duckdb tests (CGo ON):    10/10 PASS  (0.18s)
stack/duckdb tests (-race):      10/10 PASS  (1.2s)
stack/duckdb coverage:           83.3%
storage/sql tests:               ALL PASS
api-stability test:              PASS
Build (CGo ON):                  PASS — entire workspace
Build (CGo OFF):                 PASS — entire workspace
golangci-lint:                   ❌ 3 FAILURES (godoclint x2, wrapcheck x1)
nix run .#verify:                ❌ NOT RUN (would fail on lint)
```

---

## Files Changed This Session (39 files, +1512 / -53 lines)

**New files (10):**
- `stack/duckdb/` — doc.go, drivers.go, go.mod, go.sum, multidb.go, preset.go, preset_cgo_test.go, README.md, view_models.go
- `storage/sql/migrations/duckdb.sql`

**Modified files (15):**
- `storage/sql/dialect.go` (+126 DuckDBDialect)
- `storage/sql/schema_embed.go` (+7 DuckDBSchemaEmbed)
- `storage/sql/dialect_test.go` (+97 DuckDB tests)
- `storage/sql/duplicate.go` (+3 DuckDB string)
- `storage/sql/duplicate_test.go` (+9 DuckDB cases)
- `storage/sql_backend.go` (+4 NewDuckDBBackend)
- `storage/sqlite_helpers.go` (+4 DuckDBInitSchema)
- `AGENTS.md` (+44/-18 docs)
- `flake.nix` (+7 testModules + gcc)
- `go.work` (+1 stack/duckdb)
- `.golangci.yml` (+1 depguard)
- `cmd/api-stability/main.go` (+1 modules)
- `docs/api_surface.txt` (+21 exports)
- Plus auto-daemon go.mod/go.sum churn in benchkit, codec, encryption, otel, signing, eventtest, transport/http
