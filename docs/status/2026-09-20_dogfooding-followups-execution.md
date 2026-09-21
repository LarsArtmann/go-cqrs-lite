> **RESOLVED-BY-ROUTING — docs-health 10th pass (2026-09-21):** all four items shipped (see the 19-51 self-review, archived alongside). The three "found but NOT fixed" items are verified fixed by later sessions (catalog exclusion in `module_catalog_test.go`; TestStripJSONC green; gci deliberately disabled + hash-golden guard). Companion self-review archived this pass.

# Dogfooding Follow-ups - Execution Status

- **Date:** 2026-09-20
- **Scope:** the four unharvested tail items from the 2026-09-19 dogfooding self-review (owner-unblock paste)
- **Status:** COMPLETE. All four items executed and verified per-module; full `nix run .#verify` still pending a quiet window (unchanged policy).

## What was done

### 1. Tier-0 close-helper decision — RULED + SWEPT (ADR-0144)

- Ruling: `record.DeferClose` is the canonical Tier-0 address ([ADR-0144](../adr/0144-deferclose-lives-in-tier0-record.md).
  `metaengine.DeferClose` stays as a self-contained twin — forwarding to record
  was attempted and REVERTED: it would force a `record` sibling replace into
  every module that compiles metaengine from source (~15 go.mods).
- Sweep: 28 production sites converted to `record.DeferClose` across `kv`,
  `storage`, `storage/pebble`, `storage/turso/indexing`, `scheduling/sqlstore`,
  `projectionhost`, `stack`; plus 8 more sites in `metaengine/sqliteengine`,
  `benchkit`, and 4 examples converted to `metaengine.DeferClose` (helper
  already reachable). `queue/postgres` audited: already clean (bare defers).
  `cmd/cqrs-lint` linecache → bare `defer f.Close()` (tool at 9/9 dep budget;
  documented exception in ADR §4).
- Budget bumps with precedent-style comments in `check-module-layers.sh`:
  kv 3→4, projectionhost 9→10, storage 12→13 (record promoted indirect→direct).
- `kv` moves to `go 1.27.1` (published record v4.5.1 requires it) and
  `go.work` follows to 1.27.1 — which also let the go-directive sweep be
  COMPLETED across the 72 modules still stuck at `go 1.27` (fixes
  `TestEveryModuleGoSumIsTidy`, which was already red before this session).

### 2. Scan/paginate consolidation

- `bboltengine.sortAndPaginateKV` now delegates to `metaengine.SortPaginate`
  (pebble/badger precedent; hand-rolled body + `art-dupl:accept` deleted).
- New `metaengine.ScanScoredVector` + `RowScanner`: the byte-identical
  mysql/sqlite `scanScoredVector` twins and the duckdb JSON variant all
  delegate (duckdb verified green — the []byte scan dest works via
  database/sql convertAssign). `scanJSONValues` twins (mysql/duckdb) stay
  intentional: `art-dupl:accept`-annotated, and unification would change
  sqlite's passthrough error contract.

### 3. Retry-idiom reconciliation — DOCUMENTED (ADR-0145)

- [ADR-0145](../adr/0145-retry-idioms-are-per-concern.md: the four sites are
  four different concern classes (transport op retry / crash-loop damping /
  shadow freshness budget / DB contention retry). No unification; per-class
  rules + the four alignment invariants recorded.

### 4. quic/loopback dedup split brain — CLOSED

- One shared `irohengine.DefaultDedupCapacity` (loopback/quic carry sibling
  replaces to the parent until the next tag wave). quic re-exports the const
  (source-compatible). Parity pinned by `quic/dedup_parity_test.go`
  (`TestDedupParity_SharedCapacityConst` + `TestRing_ProductionCapacity10K`)
  mirroring loopback's `dedup_internal_test.go` contract; loopback/quic green
  under `-race`.

## Gates run

layer+budget ✓ · file-size ratchet ✓ (sqlite_dlq.go shrunk 353→348 by inlining
a single-use helper) · duplication ✓ (0 new groups) · api golden regenerated +
`TestEvery` ✓ · doc-check ✓ (1195 refs) · changelog symbol gate ✓ · treefmt ✓ ·
`GOWORK=off` short tests PASS on all 21 touched modules · loopback+quic `-race` ✓.

## Found but NOT fixed (other sessions' active work)

1. ~~`.golangci.yml` re-adds `gci` as a formatter (commit 96dc20986) —
   contradicts AGENTS.md contract #18 (gci vs treefmt-goimports fight; files I
   never touched fail gci, e.g. `storage/aggregate_projection.go`). Raw
   golangci-lint was NOT used as this session's verdict for that reason;
   treefmt + per-module tests were.~~ FIXED 2026-09-20 (incident-#11 repair; gci disabled + M02 hash-golden tripwire).
2. ~~`testutil/mysqltestcontainer` joined `go.work` today without a
   cqrs-lint `DefaultCatalog` entry → `TestCatalogEveryGoWorkModuleCovered`
   FAIL.~~ FIXED: exclusion entry added (`module_catalog_test.go`); test green 2026-09-21.
3. ~~`cmd/cqrs-lint` `TestStripJSONC` fails (JSONC stripper regression from
   today's committed cqrs-lint work; unrelated to my suppression edit).~~ FIXED: green 2026-09-21 (verified, 10th pass).

## Honest caveats

- `nix run .#verify` (exclusive gate) not run — machine load policy unchanged
  from previous sessions.
- The auto-commit daemon is absorbing this session's work into `chore:`
  commits; no authored history.
