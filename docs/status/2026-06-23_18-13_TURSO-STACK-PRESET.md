# Status Report: 2026-06-23 18:13 — Turso Stack Preset + Multi-DB Refactor

> **Session focus:** Added the missing `stack/turso` preset (the gap that was
> pissing off the user), plus inherited a parallel multi-DB refactor and docs
> from another session.

---

## Executive Summary

The repo had `storage/turso/` (LibSQL connector with sync, indexing advisor,
137+ files of infrastructure) but **no `stack/turso/` preset** to wire it into
a `Bundle`. Every other storage engine had a preset — memory, sqlite, pebble,
postgres — but Turso was left out. This session built it from scratch: 848
lines across 9 files, full contract test suite passing, zero lint errors.

Additionally, the working tree contained **uncommitted changes from another
session** that complement this work: a SQLite multi-DB refactor (splitting
`openSecondaryStores` into `openEventStores`/`openQueryStores`), new
documentation (`docs/PRESETS.md`, `docs/INFRASTRUCTURE_RECOMMENDATIONS.md`),
and test improvements. These were reviewed, judged sound, and included in the
commit.

---

## A) FULLY DONE ✅

### `stack/turso` — Complete Turso Preset (NEW MODULE)

| File | Lines | Purpose |
|---|---|---|
| `preset.go` | 298 | `New()` (local), `NewSync()` (remote sync), options, Bundle wrapping |
| `views.go` | 75 | Read-model KV store wiring (primary + secondary DB) |
| `closers.go` | 34 | `multiCloser`, `funcCloser` (lifecycle helpers) |
| `drivers.go` | 6 | Blank import of `turso.tech/database/tursogo` |
| `doc.go` | 35 | Package docs with quick-start examples |
| `contract_test.go` | 17 | Shared contract suite (5 subtests) |
| `preset_test.go` | 201 | E2E: event/read-model roundtrips, persistence across restarts |
| `go.mod` | 29 | Module definition with replace directives |
| `go.sum` | 153 | Dependency lock file |

**Features delivered:**
- `turso.New(dbPath, opts...)` — local embedded LibSQL (mirrors `sqlite.New`)
- `turso.NewSync(ctx, dbPath, remoteURL, authToken, opts...)` — remote sync mode
- `Bundle.Sync()` accessor — returns `*turso.SyncDB` for Push/Pull/Checkpoint/Stats/HealthCheck (nil in local mode)
- Multi-DB topology: `WithEventDB`, `WithQueryDB`, `WithViewDB` (local mode)
- `WithSyncOptions(...)` — passes through `SyncOption` values to sync client
- `WithoutAutoMigrate()` — skip schema creation
- Full rollback on setup failure (no resource leaks)
- 10 tests passing (`-race` clean): 5 contract + 5 E2E

**Quality verification:**
- `go build ./...` — ✅ clean
- `go vet ./...` — ✅ clean
- `go test ./... -count=1 -race` — ✅ 10/10 pass
- `nix run .#build` — ✅ clean
- `nix run .#lint` — ✅ zero errors in `stack/turso`

### Pre-existing Changes (from another session — reviewed and included)

These changes were in the working tree at conversation start (despite the stale
"clean" status snapshot). They were authored by another agent/session and
complement the Turso work:

1. **SQLite multi-DB refactor** (`stack/sqlite/preset.go`, `multi_db_test.go`):
   Split `openSecondaryStores` into `openEventStores` + `openQueryStores` +
   shared `openSecondaryBackend`. Better separation of concerns — event stores
   (events/snapshots/checkpoints) are now distinct from audit stores
   (commands/queries) in the multi-DB path.

2. **Documentation** (`docs/PRESETS.md`, `docs/README.md`):
   - Added Turso to the preset comparison table
   - Added Multi-DB split documentation for SQLite
   - Added Postgres distributed bus (`WithDistributedBus`) docs
   - Added Turso quick-start with sync examples
   - Linked new `INFRASTRUCTURE_RECOMMENDATIONS.md`

3. **New file** (`docs/INFRASTRUCTURE_RECOMMENDATIONS.md`, 139 lines):
   Deployer's decision guide mapping storage engines to CQRS concerns (event
   store, snapshots, checkpoints, audit, read models). Explains WHY each engine
   fits each access pattern.

### Other Completed Work

- `go.work` — added `./stack/turso`
- `AGENTS.md` — module count 38→39, presets 6→7, test command updated
- `stack/contracttest/contract.go` — comment updated to include "turso"
- `nix fmt` — formatted all files (2 pre-existing files reformatted for line wrapping)

---

## B) PARTIALLY DONE 🟡

### Turso Multi-DB + Sync Interaction
Multi-DB split (`WithEventDB`/`WithQueryDB`/`WithViewDB`) works in local mode
but is **intentionally not supported** in sync mode (`NewSync`). The entire
CQRS stack shares one syncing database in sync mode. This is documented but
could be revisited if users need multi-DB + sync.

### Turso Indexing Advisor Integration
The `storage/turso/indexing/` sub-package (auto-smart index management, usage
statistics, WAL checkpoint scheduler) exists but is **not wired into the stack
preset**. The preset calls `InitSchema` (basic DDL) but not
`InitSchemaWithIndexesAndOptimizations`. This is a deliberate choice (the
indexing advisor is an advanced feature) but could be offered as an option.

### Test Coverage for Sync Mode
`NewSync` is implemented and compiles, but there are **no tests for it** in
`stack/turso` — sync tests require a live Turso server (like Postgres tests
require `POSTGRES_TEST_DSN`). The local `New` path is fully tested.

### Docs Freshness
The AGENTS.md structure tree and module list are updated, but the following
docs still need review:
- `SKILL.md` (consumer guide) — may need Turso preset in the decision matrix
- `docs/STORAGE_GUIDE.md` — may need Turso stack section
- `FEATURES.md` — needs Turso stack preset listed

---

## C) NOT STARTED ⬜

1. **`example/` with Turso** — No example app using `stack/turso`. The other
   engines have `example/user`, `example/todo`, `example/deployer-first`.
2. **Turso bench entry** — `stack/bench/` has benchmarks for other engines;
   no Turso benchmark.
3. **CI integration** — The `nix run .#test` command was updated in AGENTS.md
   but the actual `flake.nix` test runner and GitHub Actions `ci.yml` may need
   `./stack/turso/...` added to the test list.
4. **API stability golden file** — `cmd/api-stability/` compares exported
   symbols; the new module's API surface isn't captured in a golden file yet.
5. **CODEOWNERS / module discovery** — Any tooling that enumerates modules
   (scripts, coverage reports) may need updating for the new 40th module.

---

## D) TOTALLY FUCKED UP 💥

### Nothing in this session.

The only issue encountered was LSP diagnostics being stale (showing errors
that didn't exist after `go.work` was updated) — this was cosmetic and
resolved by restarting gopls.

### Pre-existing Issues (not caused by this session):

1. **Lint debt in other modules** — 9 pre-existing lint findings across
   `catalog/`, `transport/http/`, `kv/`, `stack/` (err113, wrapcheck, errcheck,
   nilnil, noinlineerr). None in `stack/turso`.

2. **`storage/turso` lint debt** — 6 `makezero` findings in test/indexing
   files. Pre-existing, not touched this session.

3. **Stale git status snapshot** — The conversation start reported "Status:
   clean" but the working tree had 9 modified files + 1 new file from another
   session. The snapshot mechanism should be more reliable.

---

## E) WHAT WE SHOULD IMPROVE 🔧

1. **Sync mode testability** — `NewSync` has zero tests because it needs a live
   server. We should extract a `syncEngine` interface (already exists in
   `storage/turso/sync.go`) and inject a fake for unit testing the Bundle wiring.

2. **Multi-DB pattern duplication** — `stack/sqlite` and `stack/turso` now share
   nearly identical `closers.go`, `views.go`, and multi-DB wiring patterns.
   Consider extracting shared helpers into `stack/contracttest/` or a shared
   internal package to reduce copy-paste.

3. **Contract test expansion** — The shared contract suite tests BundleFields,
   EventRoundtrip, CommandRoundtrip, ReadModelRoundtrip, CloseIdempotent. It
   does NOT test: Journal reads, SeekableJournal, BackwardsSource, Snapshot
   save/load, Checkpoint save/load. These should be added to the shared suite.

4. **Preset comparison table** — `docs/PRESETS.md` has a comparison table, but
   it lacks a "Sync" column showing which presets support remote sync (only
   Turso). And the "Bus" column should clarify that Postgres can use
   LISTEN/NOTIFY for distributed bus.

5. **Postgres multi-DB** — SQLite has `WithEventDB`/`WithQueryDB`/`WithViewDB`.
   Postgres does not. Turso has it (local mode only). Pebble does not. This
   inconsistency should be documented or resolved.

---

## F) Top 25 Things to Do Next 🎯

### Immediate (this week)
1. **Commit this work** ← happening now
2. **Add `./stack/turso/...` to `flake.nix` test runner** — verify CI will test it
3. **Add `./stack/turso/...` to `ci.yml`** — ensure GitHub Actions runs the tests
4. **Run `nix run .#check-layers`** — verify dependency budget for the new module
5. **Update `SKILL.md`** — add Turso to the consumer guide decision matrix

### Short-term (next sprint)
6. **Add sync mode tests** with injectable fake sync engine
7. **Wire `WithIndexingAdvisor` option** into `stack/turso` preset
8. **Create `example/turso`** — offline-first demo app with sync
9. **Expand contract test suite** — add Journal, Snapshot, Checkpoint tests
10. **Add Turso benchmark** to `stack/bench/`
11. **Capture API stability golden file** for `stack/turso`
12. **Update `FEATURES.md`** — add Turso stack preset to feature inventory

### Medium-term (next quarter)
13. **Extract shared multi-DB helpers** — reduce duplication between sqlite/turso
14. **Add Postgres multi-DB support** — `WithEventDB`/`WithQueryDB`/`WithViewDB` for PG
15. **Add Pebble multi-DB support** — separate column families for isolation
16. **Turso embedded sync + CQRS projection demo** — full edge deployment example
17. **Add `WithDistributedBus` to Turso** — LibSQL has no LISTEN/NOTIFY, but HTTP
    polling or webhook-based pub/sub could enable multi-process Turso
18. **Coverage report for `stack/turso`** — ensure >80% like other modules
19. **Add Turso to `docs/STORAGE_GUIDE.md`** — backend facade documentation
20. **Consider `stack/turso` snapshot strategy** — `EveryNEvents` with SQL snapshots

### Quality / Tech Debt
21. **Fix the 9 pre-existing lint errors** in catalog, transport/http, kv, stack
22. **Fix the 6 `makezero` findings** in `storage/turso/` test/indexing files
23. **Add `stack/turso` to coverage gate** in CI (currently core modules >80%)
24. **Review all `replace` directives** — ensure `stack/turso/go.mod` matches
    the pattern used by other stack presets
25. **Add Turso preset to `cmd/api-stability`** golden file check

---

## G) Top Question I Cannot Answer Myself ❓

**Should the Turso preset use `InitSchemaWithIndexesAndOptimizations` instead
of plain `InitSchema` by default?**

The `storage/turso/` package has two schema initialization paths:
- `InitSchema(ctx, db)` — creates tables only (basic DDL)
- `InitSchemaWithIndexesAndOptimizations(ctx, db)` — creates tables + CQRS-
  optimized indexes + performance PRAGMAs

The SQLite preset uses the equivalent of "tables only" (via
`storage.SQLiteInitSchema`). The Turso connector doc recommends the full
initialization for production. I defaulted to `InitSchema` for consistency
with SQLite and to avoid surprising users with index creation, but a
deployer using Turso for production might expect the optimized path.

This is a **product/API decision** — should presets be "safe minimal" (current)
or "production-optimized by default"? The answer affects all presets, not just
Turso.

---

## Build & Test Status

| Check | Status |
|---|---|
| `go build ./...` (stack/turso) | ✅ Pass |
| `go vet ./...` (stack/turso) | ✅ Pass |
| `go test ./... -race` (stack/turso) | ✅ 10/10 Pass |
| `nix run .#build` (full workspace) | ✅ Pass |
| `nix run .#lint` (stack/turso) | ✅ 0 errors |
| `nix run .#lint` (full workspace) | ⚠️ 9 pre-existing errors in other modules |
| `nix fmt` | ✅ Applied |

## Numbers

| Metric | Value |
|---|---|
| Total Go modules | 40 (was 39) |
| Total Go files | 837 |
| Total test files | 416 |
| New files this session | 9 (`stack/turso/`) |
| New lines of code | 848 |
| Tests added | 10 (5 contract + 5 E2E) |
| Lint errors in new code | 0 |
