# Comprehensive Project Status Report

**Date:** 2026-04-26 17:30 CEST
**Branch:** master (1 commit ahead of origin)
**Last 3 commits:**
- `7888d3b` fix(id): remove broken NewWithPrefix, fix lint issues, fix stale comments
- `d311e9f` fix(id): correct MarshalJSON double-encoding and UnmarshalBinary size check
- `a9c833a` chore: migrate all test fixtures from human-readable IDs to ULID-formatted IDs

---

## A) FULLY DONE ✅

### Multi-Module Monorepo Migration (Phases 0–4)
| Phase | Description | Status |
|-------|-------------|--------|
| 0 | Fix query handler ctx, delete pkg/errors, replace custom YAML | ✅ Done |
| 1 | go.work + move into `core/` subdirectory | ✅ Done |
| 2 | Extract `memory/` module | ✅ Done |
| 3 | Extract `catalog/` module | ✅ Done |
| 4 | Extract middleware + xtypes | ✅ Done |

### Branded Return Types Migration
All core interfaces now return branded ID types instead of `string`:
- `Event.ID()` → `id.EventID`
- `Event.AggregateID()` → `id.AggregateID`
- `Root.ID()` → `id.AggregateID`
- `Command.AggregateID()` → `id.AggregateID`

### ULID Migration (core/pkg/id)
- `id.Of[T]` changed from type alias `= cbid.ID[T, string]` to wrapper struct around `cbid.ID[T, ulid.ULID]`
- All serialization reimplemented locally (MarshalJSON, UnmarshalJSON, Scan, Value, MarshalBinary, UnmarshalBinary, MarshalText, UnmarshalText, Compare, String)
- ~120 test fixture strings replaced with valid 26-char Crockford base32 ULIDs
- `NewWithPrefix` removed (prefix incompatible with ULID, silently discarded the parameter)
- `PrefixString` type removed
- All 5 workspace modules + both example apps have `go-composable-business-types v0.1.0` replace directives

### Code Quality
- All tests pass (15 packages, 0 failures)
- `go vet` clean on all modules
- `golangci-lint run ./...` → 0 issues
- Both example apps build cleanly (GOWORK=off)
- Coverage: 75.7% total (8 packages >85%)

### Bug Fixes (All Sessions)
| Bug | Fix | Commit |
|-----|-----|-------|
| Retry dead cancellation | `context.Background().Done()` → `ctx.Done()` | `5ad0356` |
| Aggregate version desync | Removed fallback loop; `Load()` requires `HistoryLoader` | `1862eae` |
| Wrong error sentinel (dispatcher) | `ErrHandlerNotFound` → `ErrDispatcherClosed` | `5ad0356` |
| Slice mutation (MemoryStore) | Defensive copies from `Load()`/`LoadFromVersion()` | `d5ea811` |
| Wrong error sentinel (snapshot) | `ErrSnapshotNotFound` → `ErrSnapshotStoreClosed` | `8e5150c` |
| MarshalJSON double-encoding | Bare ULID text → proper JSON string with quotes | `d311e9f` |
| UnmarshalBinary wrong size | `ulid.EncodedSize` (26) → `ulidBinarySize` (16) | `d311e9f` / `7888d3b` |
| Value() returning bytes | `[]byte` → `string` for SQL driver compatibility | `d311e9f` |
| Compare() on non-Ordered | `cmp.Compare` → `ulid.ULID.Compare()` | `a9c833a` |
| NewWithPrefix ignoring prefix | Deleted function (no production callers) | `7888d3b` |
| Benchmark using UUID string | Changed to valid ULID | `7888d3b` |

### Dead Code Removed
- `query.Result[T]`, `Streamer` interface, `store_config.go`, `internal/testutil`, `evtest.GenerateUUID`
- Unused error sentinels: `ErrEventNotFound`, `ErrInvalidEventType`, `ErrCommandValidation`, `ErrQueryValidation`
- `NewWithPrefix`, `PrefixString` type, `TestNewWithPrefix`, `BenchmarkNewWithPrefix`

### Documentation Updated
- `AGENTS.md`: google/uuid → oklog/ulid, coverage updated, ULID migration noted, deferred items marked done
- `README.md`: google/uuid → oklog/ulid in dependency tables
- `event.go:17`: stale comment fixed (google/uuid → oklog/ulid)
- Multiple status reports in `docs/status/`

---

## B) PARTIALLY DONE 🟡

### Test Coverage
| Package | Coverage | Target | Gap |
|---------|----------|--------|-----|
| `pkg/id` | 73.1% | >85% | New ULID marshaling methods need more test cases |
| `middleware` | 64.8% | >80% | `EventRetry` untested, some error paths uncovered |
| `internal/dispatcher` | 73.8% | >80% | Edge cases in lifecycle/close paths |
| `aggregate` | 89.7% | >90% | Minor gap |

### Replace Directives
- All workspace modules have `go-composable-business-types` replace → `v0.1.0` ✅
- `middleware/go.mod` and `xtypes/go.mod` still have `replace` directives pointing to `../../go-composable-business-types` (local path) — these are necessary for now since the repo is private and the modules need it for transitive deps

---

## C) NOT STARTED 🔴

### Migration Phases 5–10
| Phase | Description | Priority |
|-------|-------------|----------|
| 5 | Storage module (sqlc event store) | HIGH |
| 6 | Watermill module (pub/sub) | MEDIUM |
| 7 | Projection module (samber/ro internally) | MEDIUM |
| 8 | Snapshot module (SQL-backed) | MEDIUM |
| 9 | Test utilities module | LOW |
| 10 | Tag releases | HIGH (but depends on storage module) |

### Missing Test Coverage
- `EventRetry` middleware — zero test coverage
- `pkg/id` — new ULID-backed methods (MarshalBinary roundtrip, Scan edge cases, Value nil receiver)
- Query middleware (`QueryRetry`, `QueryRecovery` error paths)

### gopls LSP Issues
- `example/catalog/` shows BrokenImport errors in gopls (stale cache). `go build` works fine. LSP needs restart.

---

## D) TOTALLY FUCKED UP 💥

**Nothing is totally fucked up.** The codebase compiles, all tests pass, lint is clean, vet is clean, both examples build. No broken features, no data loss risks, no security issues.

---

## E) WHAT WE SHOULD IMPROVE

### High Impact
1. **Storage module (Phase 5)** — The project has no persistent event store. `memory/` is only for testing. A `sqlc`-backed PostgreSQL store is the #1 missing piece for production use.
2. **`pkg/id` test coverage** — Dropped from 85.4% to 73.1% after ULID migration. New marshaling methods (MarshalBinary roundtrip, SQL Scan/Value edge cases, zero-value handling) need tests.
3. **Middleware test coverage** — 64.8% is too low. `EventRetry` has zero tests. `QueryRetry`/`QueryRecovery` error paths untested.
4. **Remove unnecessary type args** — gopls reports 6 `infertypeargs` warnings in `core/pkg/id/id.go`. Easy cleanup.
5. **`go.work` version mismatch** — go.work says `go 1.26` but modules require `go 1.26.0`. Should run `go work sync`.

### Medium Impact
6. **`MemoryBus.Publish` RLock during handler execution** — Subscribers block publishers. Acceptable for test utility but should be documented.
7. **`xtypes.TypedCommand.Command()` allocation** — Creates new `command.Core` on every call. Could cache.
8. **`toDotAddress` number handling** — "Get3DView" → "get.3.d.view" instead of "get.3d.view". Edge case in catalog.
9. **EventRetry tests** — Only middleware with zero test coverage.
10. **Replace directives cleanup** — Once `go-composable-business-types` v0.1.0 is published, local replace directives can be removed.

### Low Impact
11. **Aggregate coverage** — 89.7%, close to 90% target. One or two more test cases.
12. **Dispatcher coverage** — 73.8%, lifecycle/close edge cases missing.
13. **Example modules not in go.work** — `example/user` and `example/catalog` are standalone, which is correct, but could be documented.
14. **Unnecessary `memory` replace in `example/catalog/go.mod`** — Harmless but unnecessary.

---

## F) TOP 25 THINGS TO DO NEXT

| # | Task | Impact | Effort | Priority |
|---|------|--------|--------|----------|
| 1 | Storage module: sqlc PostgreSQL event store (Phase 5) | 🔴 Critical | 2-3 days | P0 |
| 2 | Add `pkg/id` test coverage (MarshalBinary roundtrip, Scan edge cases, zero-value Value) | 🔴 High | 30min | P0 |
| 3 | Add `EventRetry` tests | 🟡 High | 20min | P0 |
| 4 | Fix `go.work` version mismatch (`go work sync`) | 🟡 Medium | 5min | P1 |
| 5 | Remove unnecessary type args in `id.go` (6 gopls warnings) | 🟡 Medium | 10min | P1 |
| 6 | Improve middleware test coverage to >80% | 🟡 Medium | 1hr | P1 |
| 7 | Improve `internal/dispatcher` coverage to >80% | 🟡 Medium | 30min | P1 |
| 8 | Watermill module (Phase 6) — pub/sub | 🟡 Medium | 2-3 days | P2 |
| 9 | Projection module (Phase 7) | 🟡 Medium | 2-3 days | P2 |
| 10 | Snapshot module (Phase 8) — SQL-backed | 🟡 Medium | 1-2 days | P2 |
| 11 | Cache `TypedCommand.Command()` allocation | 🟢 Low | 15min | P3 |
| 12 | Fix `toDotAddress` number handling | 🟢 Low | 15min | P3 |
| 13 | Add aggregate coverage to >90% | 🟢 Low | 15min | P3 |
| 14 | Document `MemoryBus.Publish` RLock behavior | 🟢 Low | 10min | P3 |
| 15 | Remove unnecessary `memory` replace in `example/catalog/go.mod` | 🟢 Low | 5min | P3 |
| 16 | Tag v0.1.0 releases for all modules | 🔴 High | 30min | P1 (after storage) |
| 17 | Test utilities module (Phase 9) | 🟢 Low | 1 day | P4 |
| 18 | Add integration test with full CQRS flow (command → event → aggregate → query) | 🟡 Medium | 2hr | P2 |
| 19 | Add Go doc examples (playground-runnable) for `id.New`, `id.Parse`, `event.NewEvent` | 🟡 Medium | 1hr | P2 |
| 20 | Remove `replace` directives once `go-composable-business-types` is published | 🟢 Low | 15min | P4 |
| 21 | Add `go work sync` to CI pipeline | 🟢 Low | 15min | P3 |
| 22 | Add benchmark for ULID generation vs old UUID generation | 🟢 Low | 15min | P3 |
| 23 | Add `.goreleaser.yml` for multi-module releases | 🟡 Medium | 1hr | P2 |
| 24 | Explore `go-json-experiment/json` v2 for struct-level marshaling of `id.Of[T]` | 🟢 Low | 30min | P4 |
| 25 | Add CHANGELOG.md tracking breaking changes per module | 🟡 Medium | 30min | P2 |

---

## G) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**Should the Storage module (Phase 5) use `sqlc` with hand-written SQL, or an ORM like `pgx`+`sqlx`, or a higher-level library like `goose`+`pgx`?**

The migration plan says "sqlc event store" but:
- `sqlc` generates Go code from SQL — great for queries, but event stores need complex transaction patterns (optimistic concurrency via version checks, append-only with version validation)
- `pgx` gives direct PostgreSQL access with better transaction control
- The event store interface needs `Append(ctx, aggregateID, events)` and `Load(ctx, aggregateID)` with version-gated optimistic concurrency — this maps better to hand-written SQL than `sqlc` generates

I need guidance on: **What's the desired approach for the SQL-backed event store — `sqlc`, `pgx` raw queries, or something else?** This determines the entire architecture of Phase 5+.

---

## Test Coverage Summary (Current)

| Package | Coverage |
|---------|----------|
| catalog/adapters | 98.8% |
| memory | 94.7% |
| xtypes | 95.7% |
| catalog/asyncapi | 97.6% |
| query | 91.4% |
| catalog | 87.0% |
| aggregate | 89.7% |
| event | 88.0% |
| catalog/eventcatalog | 89.7% |
| command | 84.4% |
| middleware | 64.8% |
| core/pkg/id | 73.1% |
| internal/dispatcher | 73.8% |
| **Total** | **75.7%** |

## Quality Gates (All Passing ✅)

| Gate | Status |
|------|--------|
| `go test ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...` | ✅ All 15 packages pass |
| `go vet ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...` | ✅ Clean |
| `golangci-lint run ./...` | ✅ 0 issues |
| `go build` (all workspace modules) | ✅ Clean |
| `go build .` (example/user, GOWORK=off) | ✅ Clean |
| `go build .` (example/catalog, GOWORK=off) | ✅ Clean |
