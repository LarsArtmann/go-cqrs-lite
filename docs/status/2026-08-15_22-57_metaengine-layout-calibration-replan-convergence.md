# Status: Metaengine Layout Calibration + Replan Convergence — 2026-08-15 22:57

Task: execute the five open items from `TODO_LIST.md` §"Metaengine — Layout
Planning (Phase 6b)" + §"Layout roles". Progress: 3 of 5 code items done and
verified GREEN; 2 remain (multi-engine integration test, DemoteEngine), plus
docs + full verify gate.

## DONE (verified)

### 1. DuckDB (Columnar) calibration — TODO item closed

- New bench: `metaengine/bench/bench_layout_calibration_columnar_cgo_test.go`
  (cgo-gated). Embed side via the engine MapBackend API (meta_map point
  read/RMW); normalize side via parent/child tables in a separate DuckDB file
  (LEFT JOIN read, O(1) child insert); storage via standalone files (os.Stat).
- Measured on file-backed DuckDB (AMD Ryzen AI MAX+ 395, 2026-08-15,
  ~5s/op ≈ 60s+ per engine run):
  - read ratio (normalize/embed): **2.62x** (point lookups favor embed)
  - write ratio: **0.20x** (DuckDB UPDATE ≈ 5x an insert)
  - storage ratio: **0.59x** (columnar compression absorbs duplication)
- `layout_scoring.go` Columnar cell recalibrated: embed 0.62/2.23/1.30,
  normalize 1.62/0.45/0.77 (geomean-centered: pair product = 1.0 per dimension).
- **The exact-tie cell is gone**: Columnar × ReadSpeed was 2.65 vs 2.65
  (float-comparison fragility); it is now a measured 0.08-margin Embed win
  (3.345 vs 3.425). `layout_matrix_test.go` updated; all 16 cells PASS.

### 2. SQLite/Postgres/MySQL (Row) calibration — TODO item closed

- New benches: `bench_layout_calibration_row_test.go` (timing, 4 ops ×
  engines) + `bench_layout_calibration_storage_test.go` (storage sizes +
  shared chunked-seed helpers). PG gated on `POSTGRES_TEST_DSN`, MySQL on
  `MYSQL_TEST_DSN` (skip when unset); SQLite always runs.
- Executed: SQLite (file-backed), Postgres 16 via `nix run .#integration-pg`,
  MySQL via `nix run .#integration-mysql-vm` (QEMU, ratio-grade numbers).
- Measured normalize/embed ratios:

  | Engine | read | write | storage |
  |---|---|---|---|
  | SQLite | 1.95x | 0.66x | 0.327 |
  | Postgres 16 | 1.00x | 0.375x | 0.326 |
  | MySQL (VM) | 1.06x | 0.56x | 0.413 |
  | **Row geomean** | **1.27x** | **0.52x** | **0.35x** |

- `layout_scoring.go` Row cell recalibrated: embed 0.89/1.39/1.68, normalize
  1.13/0.72/0.59. Notable sign-flip vs the old analytical estimate: normalize
  reads are NOT cheaper than JSON-column reads (old guess 0.8; measured
  1.95x on SQLite, ≈1.0x on server engines). Decisions unchanged: Row stays
  Normalize in all 4 priority cells (write+storage dominate).
- `metaengine/bench/go.mod` gained pgengine/mysqlengine (local `replace`) +
  direct `go-sql-driver/mysql` + `jackc/pgx/v5/stdlib` driver imports.
- Bug found+fixed along the way: MySQL `information_schema` sizes are stale
  until `ANALYZE TABLE` (first storage run reported 54.3x — actually 0.41x).

### 3. ReplanLayout converged into Store.Replan — TODO item closed

- `metaengine/relayout.go`: `ReplanLayout(ctx, pc)` now (a) applies `pc` as
  the store priority config when non-nil, (b) funnels through the ONE replan
  path `replanWithTrigger` (audited `priority-change`/`manual`), (c) returns
  diffs computed old-plan vs new-plan (both produced by `planQuery`, which
  already records `Layout` — the duplicate scoring/priority-resolution copy
  in relayout.go is deleted).
- Orphaned `currentLayoutForQuery` removed. Signature unchanged → no API
  surface change. Semantic change (documented): ReplanLayout now APPLIES the
  config instead of being a pure what-if — equivalent to SetPriority+Replan.
- Existing Ginkgo + convergence tests pass unchanged; full metaengine
  package: `ok 15.890s` (count=1).

## PENDING (not started)

1. **Multi-engine integration test** (two live backends, AddEngine+Backfill,
   verify both serve correct results) — planned home: `metaengine/bench`
   (SQLite + Pebble, both CGo-free, fixtures already there).
2. **DemoteEngine** (Active → shadow role transition, LAYOUT-ROLES §4.4
   "future API"). Design worked out in-session: skip-set on `replicate` to
   close the demotion double-apply window + targeted catch-up replay for
   collections the demoted engine never served. Not yet written.
3. Docs: CHANGELOG, TODO_LIST check-offs, ADR-0124 addendum (calibration
   provenance + replan convergence), `METAENGINE-LAYOUT-PLANNING-MODEL.md`
   numbers, skill `references/recipes.md` if constants are quoted there.
4. Gates not yet run: `nix fmt` (new files untouched by treefmt so far),
   api-stability golden regen (expected no-op), `check-arch` (bench go.mod
   grew deps), `.golangci.yml` depguard allow-list for the two new driver
   imports, doc-check, race, and the exclusive full `nix run .#verify`.

## What I forgot / could do better

1. **Pre-existing calibration benches have a measurement flaw I did not fix**:
   the memory + LSM benches (`layout_calibration_bench_test.go`,
   `bench_layout_calibration_disk_test.go` EmbedWrite) append a child on every
   iteration → the value grows unboundedly during the run (SQLite embed-write
   drifted 41µs → 85µs before I made MY new benches size-stable). The
   published KV/LSM constants were derived from drifting measurements. Worth
   a TODO: make those benches size-stable and re-derive KV/LSM constants.
2. My first draft of the storage bench shipped a broken stub
   (`catalogStorageSizes` returning 0,0) and a DuckDB duplicate-key failure
   (repeated `$1,$2` placeholders bind the same row values — DuckDB/PG
   positional semantics; each row needs distinct `$n`). Both caught by
   running the benches before touching constants — but the first pass would
   not have survived review.
3. `sed -i` bulk renames on files while the auto-commit daemon runs — one
   edit failed mid-flight because the file changed between read and write
   (mtime bump). Re-read + re-applied. Use multiedit on freshly-read files.
4. Foreign uncommitted changes exist in the tree (~40 files: command/event/id
   asrecord + actor_id work) — NOT touched, owned by another session.
5. MySQL absolute latencies are QEMU-RTT-inflated (~300µs floor); only ratios
   feed the constants. Documented in the constants' provenance comment.
6. Storage model measures embed as 3 copies (summary/history/search) — same
   convention as the KV calibration; an approximation, stated in the bench.
7. DuckDB embed reads (~185µs) are dominated by per-statement overhead at
   N=1000 — an OLAP engine's point-read cost is honest but dataset is small;
   no disk pressure exercised for reads (writes/updates do hit disk).

## Questions

1. The TODO says "Run 60s disk bench" for DuckDB. I ran ~5s/op (≈60s+ total
   per engine). Do you want a literal `-benchtime=60s` single-op
   re-confirmation before the constants are considered locked?
2. MySQL ran over the QEMU port-forward. Accept ratios as-is, or re-run via
   `integration-mysql-nspawn` (needs sudo, ~15s) for lower-RTT confirmation?
3. DemoteEngine: implement now with the skip-set + targeted-replay design
   (above), or write the design addendum to METAENGINE-LAYOUT-ROLES.md first
   and wait for your approval before code?

— next step (unless redirected): item 3 (multi-engine integration test) in
`metaengine/bench`, then DemoteEngine per your answer to Q3.
