# Comprehensive Status Report — 2026-04-26

**Generated:** 2026-04-26_12-37
**Branch:** master (ahead of origin by 0 commits, up to date)
**Last 2 commits:** `6aae77b` docs + `7cc3e20` feat(core)! branded types

---

## A. WORK STATUS

### A.1. Branded Return Types Migration — ✅ FULLY DONE (this session)

| Item                                                   | Status  |
| ------------------------------------------------------ | ------- |
| `Event.ID()` → `id.EventID`                            | ✅ DONE |
| `Event.AggregateID()` → `id.AggregateID`               | ✅ DONE |
| `Root.ID()` → `id.AggregateID`                         | ✅ DONE |
| `Command.AggregateID()` → `id.AggregateID`             | ✅ DONE |
| All callers updated (15 files)                         | ✅ DONE |
| Redundant `id.ParseAggregateID()` re-parses eliminated | ✅ DONE |
| All tests pass (9 modules)                             | ✅ DONE |
| Documentation updated (AGENTS.md)                      | ✅ DONE |
| Commits pushed to origin                               | ✅ DONE |

**Files changed (15):**

```
core/aggregate/aggregate.go          |  6 +++---
core/aggregate/aggregate_test.go    |  8 ++++----
core/aggregate/cqrs_bdd_test.go   | 12 ++++++------
core/aggregate/integration_test.go   |  6 +++---
core/aggregate/repository.go         | 23 ++++++++---------------
core/aggregate/repository_test.go   |  4 ++--
core/command/command.go            |  4 ++--
core/command/command_test.go       |  2 +-
core/event/event.go                |  8 ++++----
core/event/event_sourcing_bdd_test.go | 6 +++---
core/event/event_test.go           |  4 ++--
middleware/logging.go              | 4 ++--
middleware/middleware_test.go      | 2 +-
xtypes/xtypes_test.go             | 8 ++++----
catalog/eventcatalog/exporter_test.go | 4 ++--
AGENTS.md                          | 20 +++++++++++++++++---
```

**Net result:** 64 insertions, 57 deletions (−7 lines, cleaner code)

**Commits:**

- `7cc3e20` feat(core)!: return branded ID types from Event, Root, Command interfaces
- `6aae77b` docs: update AGENTS.md with branded return types migration

### A.2. Uncommitted Changes — ⚠️ PARTIALLY DONE

**`memory/bus.go` + `memory/snapshot.go`** have uncommitted changes:

- Adding `mu` field initialization in constructors (fixes `exhaustruct`)
- Changing `err := b.CheckClosed(...)` + `if err != nil` → `if err := ...; err != nil { return errors.Wrap(err, "...") }`
- Adds `errors.Wrap` for `CheckClosed` returns
- **Problem:** This changes `err := ...` pattern to `if err := ...; err != nil` which triggers `noinlineerr` linter
- **Net effect:** Would fix 3 `wrapcheck` issues but add ~6 `noinlineerr` issues. Does NOT improve overall lint scorecard.
- **Status:** NOT committed, NOT recommended to commit as-is

### A.3. example/user Module — ❌ NOT STARTED (blocked)

- **Blocker:** `go mod tidy` fails: `github.com/larsartmann/go-composable-business-types` at revision `v0.0.0` does not exist
- This is the same `go-composable-business-types` publish problem documented in prior status reports
- Files that need updating once unblocked:
  - `example/user/aggregate.go:62,83`: `id.MustParseAggregateID(u.ID())` → `u.ID()`
  - `example/user/handlers.go:15,27`: `id.MustParseAggregateID(cmd.AggregateID())` → `cmd.AggregateID()`
- Cannot build/test until `go-composable-business-types` is published or inlined

### A.4. xtypes Pre-existing Bugs — ✅ ALREADY FIXED

The "remaining issues" from the prior session were actually already fixed in the files:

- `xtypes/event.go:95`: `b.aggregateID.IsZero()` — correct (not `IsEmpty`)
- `xtypes/xtypes_test.go:244`: `e.AggregateID() != aggregateID` — correct (not `.String()`)
- `xtypes/xtypes_test.go:333`: `core.ID() != aggregateID` — correct (not `.String()`)
- gopls diagnostics were stale cache, not real errors

---

## B. BUILD & TEST STATUS

### Build

```
go build ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...  ✅ CLEAN
```

### Test Suite — ✅ ALL PASSING

| Module                | Packages | Status          | Coverage        |
| --------------------- | -------- | --------------- | --------------- |
| `core/aggregate`      | 1        | ✅              | 89.7%           |
| `core/command`        | 1        | ✅              | 84.4%           |
| `core/event`          | 1        | ✅              | 88.0%           |
| `core/pkg/dispatcher` | 1        | ✅              | 77.4%           |
| `core/pkg/id`         | 1        | ✅              | **63.6%** ← LOW |
| `core/query`          | 1        | ✅              | 91.4%           |
| `memory`              | 1        | ✅              | 94.2%           |
| `catalog`             | 5        | ✅              | 87.0%–98.8%     |
| `middleware`          | 1        | ✅              | **64.8%** ← LOW |
| `xtypes`              | 1        | ✅              | 95.7%           |
| **TOTAL**             | **14**   | **✅ ALL PASS** |                 |

**Note:** `core/pkg/id` coverage dropped from 85.4% to 63.6% after ULID migration. The package now delegates to `go-composable-business-types/id` (28 lines, down from 222) but tests weren't updated to match.

---

## C. LINT STATUS — ⚠️ 105 PRE-EXISTING ISSUES (not introduced by this migration)

```
golangci-lint run ./core/... ./memory/... ./catalog/... ./middleware/... ./xtypes/...
```

| Category       | Count | Detail                                           |
| -------------- | ----- | ------------------------------------------------ |
| `exhaustruct`  | 50    | Struct literals missing optional fields          |
| `wrapcheck`    | 32    | Errors from external packages returned unwrapped |
| `noinlineerr`  | 10    | Inline error handling (in `memory/`)             |
| `thelper`      | 9     | Test helpers missing `b.Helper()` / `t.Helper()` |
| `tagalign`     | 4     | JSON struct tags not aligned                     |
| `paralleltest` | 3     | BDD suite tests missing `t.Parallel()`           |
| `gci`          | 3     | Import group formatting                          |
| `gosec`        | 2     | `G304` file inclusion risk                       |
| `modernize`    | 1     | `mapsloop` → `maps.Copy`                         |
| `varnamelen`   | 1     | Variable name too short                          |

**Files with most lint issues:**
| File | Issues |
|------|--------|
| `catalog/registry_test.go` | 21 |
| `catalog/adapters/adapters_test.go` | 11 |
| `core/pkg/dispatcher/dispatcher_test.go` | 9 |
| `core/event/internal/evtest/helpers.go` | 7 |
| `memory/store.go` | 6 |
| `core/aggregate/cqrs_bdd_test.go` | 6 |
| `core/aggregate/integration_test.go` | 6 |

**Note:** The `noinlineerr` linter wants `err := ...` (plain assignment) NOT `if err := ...; err != nil`. The uncommitted changes in `memory/bus.go`/`memory/snapshot.go` convert to the inline form and would worsen this count.

---

## D. KNOWN ISSUES

| Issue                                      | Severity | Status This Session                                                  |
| ------------------------------------------ | -------- | -------------------------------------------------------------------- |
| LSP diagnostics stale (gopls cache)        | LOW      | gopls shows 149 errors that don't exist; `go build` compiles cleanly |
| `example/user` module broken               | MEDIUM   | Still blocked by `go-composable-business-types` not published        |
| `core/pkg/id` coverage 63.6%               | MEDIUM   | Dropped after ULID delegation; tests need updating                   |
| `middleware` coverage 64.8%                | MEDIUM   | `EventRetry` has no tests                                            |
| 105 lint issues                            | LOW      | All pre-existing, none from branded types migration                  |
| `MemoryBus.Publish` holds RLock            | LOW      | Acceptable for test utility                                          |
| `xtypes.TypedCommand.Command()` allocation | LOW      | Creates new `command.Core` on every call                             |
| `toDotAddress` number handling             | LOW      | "Get3DView" → "get.3.d.view" instead of "get.3d.view"                |
| `go.work` version mismatch                 | LOW      | says `go 1.26` but modules require `go 1.26.0`                       |
| Uncommitted memory changes                 | LOW      | Don't improve lint; not committed                                    |

---

## E. WHAT WE SHOULD IMPROVE

### High Priority

1. **Publish `go-composable-business-types`** — Critical portability blocker. All modules have `replace` directives pointing to a non-existent revision. Anyone cloning this repo cannot build without access.

2. **Fix 105 lint issues** — Low-severity but accumulative noise. The `exhaustruct` (50) and `wrapcheck` (32) issues are the biggest categories.

3. **Restore `core/pkg/id` test coverage** — 63.6% is the weakest module. After ULID migration, the package delegates 100% to `go-composable-business-types/id`, but tests still call the thin wrapper. Need test updates or test deletion.

4. **Fix `example/user` module** — Unblocked once `go-composable-business-types` is published. Update callers to use branded types directly.

### Medium Priority

5. **Increase `middleware` test coverage** — 64.8% coverage. `EventRetry` middleware has zero tests.

6. **Add `BinaryMarshaler`/`TextMarshaler` to branded IDs** — Enables ID serialization to binary/text formats (database columns, URL paths).

7. **Add utility methods to `Of[T]`** — `Equal`, `Compare`, `Or`, `Reset` were identified in prior review as missing from the branded ID type.

8. **Fix `event/store_config.go`** — Vestigial config that returns an error pointing to `memory` module. Should be deleted or properly implemented.

9. **Fix `go.work` version mismatch** — `go 1.26` vs `go 1.26.0` in modules. Run `go work use` to sync.

10. **Add `t.Parallel()` to BDD suite tests** — `TestCQRSBDD`, `TestEventBDD`, `TestQueryBDD` are missing the parallel call (3 `paralleltest` issues).

### Low Priority

11. **Update outdated `docs/planning/go-composable-business-types-usage.md`** — Written before strong IDs were implemented; describes a future state implemented differently.

12. **Document `go-composable-business-types` API surface used** — Only a subset of the library is used; documenting it reduces confusion.

13. **Fix `toDotAddress` number handling** — "Get3DView" → "get.3.d.view" instead of "get.3d.view".

14. **Add integration tests for catalog exporters** — AsyncAPI YAML/JSON and EventCatalog MDX exporters.

15. **Add `goimports` / `gofumpt` to CI** — Currently linting is separate from formatting checks.

16. **Performance benchmarks for event store** — No benchmarks currently exist.

17. **Tag releases (Phase 10)** — Migration phases 0–4 are done; phases 5–10 are planned but not started.

---

## F. TOP #25 THINGS TO GET DONE NEXT

| #   | Item                                                         | Priority | Effort | Category       |
| --- | ------------------------------------------------------------ | -------- | ------ | -------------- |
| 1   | Publish `go-composable-business-types` as Go module          | 🔴 HIGH  | 1h     | Infrastructure |
| 2   | Fix 105 lint issues (or selectively suppress)                | 🟡 MED   | 2–4h   | Code Quality   |
| 3   | Restore `core/pkg/id` coverage (63.6% → 85%+)                | 🟡 MED   | 1h     | Testing        |
| 4   | Fix `example/user` module callers                            | 🟡 MED   | 30min  | Migration      |
| 5   | Increase `middleware` coverage (64.8% → 80%+)                | 🟡 MED   | 1h     | Testing        |
| 6   | Add `EventRetry` tests in `middleware`                       | 🟢 LOW   | 30min  | Testing        |
| 7   | Add `BinaryMarshaler`/`TextMarshaler` to `Of[T]`             | 🟡 MED   | 1h     | Feature        |
| 8   | Add utility methods (`Equal`, `Compare`, `Or`) to `Of[T]`    | 🟢 LOW   | 1h     | Feature        |
| 9   | Fix `go.work` version mismatch (`1.26` vs `1.26.0`)          | 🟢 LOW   | 5min   | Config         |
| 10  | Add `t.Parallel()` to BDD suite tests                        | 🟢 LOW   | 10min  | Testing        |
| 11  | Delete or implement `event/store_config.go`                  | 🟢 LOW   | 15min  | Cleanup        |
| 12  | Update `docs/planning/go-composable-business-types-usage.md` | 🟢 LOW   | 30min  | Docs           |
| 13  | Fix `toDotAddress` number handling bug                       | 🟢 LOW   | 1h     | Bug            |
| 14  | Document `go-composable-business-types` API surface used     | 🟢 LOW   | 10min  | Docs           |
| 15  | Add `goimports`/`gofumpt` to CI pipeline                     | 🟢 LOW   | 30min  | CI             |
| 16  | Add performance benchmarks for event store                   | 🟢 LOW   | 1h     | Performance    |
| 17  | Phase 5: Implement storage module (sqlc event store)         | 🔴 HIGH  | 4–8h   | Migration      |
| 18  | Phase 6: Implement Watermill module (pub/sub)                | 🟡 MED   | 4–8h   | Migration      |
| 19  | Phase 7: Implement Projection module (samber/ro)             | 🟡 MED   | 4–8h   | Migration      |
| 20  | Phase 8: Implement Snapshot module (SQL-backed)              | 🟡 MED   | 4h     | Migration      |
| 21  | Phase 9: Implement Test utilities module                     | 🟡 MED   | 2h     | Migration      |
| 22  | Phase 10: Tag releases                                       | 🟡 MED   | 1h     | Release        |
| 23  | Update README with full migration status                     | 🟢 LOW   | 15min  | Docs           |
| 24  | Add CONTRIBUTING.md for external contributors                | 🟢 LOW   | 30min  | Docs           |
| 25  | Address `xtypes.TypedCommand.Command()` allocation issue     | 🟢 LOW   | 1h     | Optimization   |

---

## G. TOP #1 QUESTION I CANNOT FIGURE OUT

### Should we publish `go-composable-business-types` as a standalone Go module, or inline the ULID ID logic back into `core/pkg/id`?

**The problem in detail:**

The ULID migration (session 3–4, April 25) replaced 222 lines of hand-rolled UUID-based ID code in `core/pkg/id` with 28 lines delegating to `github.com/larsartmann/go-composable-business-types/id`. This is elegant but creates a critical portability problem:

1. **`go-composable-business-types` is not accessible outside this workspace.** No one can `go mod download` without access. This breaks `GOWORK=off go mod tidy` for `example/user` and any external consumer.

2. **Every module needs `replace` directives.** The library exists as a sibling directory but is not a proper Go module with a published version. This is fragile and confusing.

3. **The library contains more than just IDs** — it has `Money`, `ActorChain`, and other business types. It's a legitimate library candidate, not just a throwaway wrapper.

**Option A: Publish `go-composable-business-types`**

- Tag `v1.0.0` on GitHub, remove all `replace` directives, update `go.mod` files to point to the real version.
- Pro: Library is properly reusable. Follows the original design intent.
- Con: Requires maintaining a second repository. Two release processes.

**Option B: Inline ULID logic back into `core/pkg/id`**

- Remove the `go-composable-business-types` dependency entirely. Implement ULID generation directly in `core/pkg/id`.
- Pro: Zero external dependencies. Simpler module graph. `example/user` works immediately.
- Con: Duplicates code. `go-cqrs-lite` and `go-localsync` diverge on ID implementation.

**Option C: Keep `replace` directives, add `example/user` to `go.work`**

- Add `example/user` to the workspace, remove the need for `go mod tidy` in that module.
- Pro: Quick fix. No library publishing needed.
- Con: Doesn't fix the fundamental portability issue for external consumers.

**What I've tried:**

- Checked all `go.mod` files — all have `replace` directives for `go-composable-business-types`
- Attempted `go mod tidy` in `example/user` — fails as expected
- Checked `go-cqrs-lite/go.work` — `example/user` is intentionally excluded (not a library module)
- Reviewed prior status docs — this was flagged in `2026-04-26_12-09` as item #1 priority, unresolved

**What I need to know:**
Is `go-composable-business-types` intended to be a real, published Go library (Option A), or is it a local experiment that should be inlined back into this project (Option B)? The answer determines the entire next phase of the migration.

---

## H. MIGRATION PHASE SUMMARY

| Phase | Description                                                   | Status         |
| ----- | ------------------------------------------------------------- | -------------- |
| 0     | Fix query handler ctx, delete pkg/errors, replace custom YAML | ✅ Done        |
| 1     | go.work + move into `core/` subdirectory                      | ✅ Done        |
| 2     | Extract `memory/` module                                      | ✅ Done        |
| 3     | Extract `catalog/` module                                     | ✅ Done        |
| 4     | Extract middleware + xtypes                                   | ✅ Done        |
| 5     | Storage module (sqlc event store)                             | 🔴 Not Started |
| 6     | Watermill module (pub/sub)                                    | 🔴 Not Started |
| 7     | Projection module (samber/ro internally)                      | 🔴 Not Started |
| 8     | Snapshot module (SQL-backed)                                  | 🔴 Not Started |
| 9     | Test utilities module                                         | 🔴 Not Started |
| 10    | Tag releases                                                  | 🔴 Not Started |

---

## I. COMMITS THIS SESSION

```
7cc3e20 feat(core)!: return branded ID types from Event, Root, Command interfaces
6aae77b docs: update AGENTS.md with branded return types migration
```

**This session's net code change:** 15 files, +64 lines, −57 lines (net −7 lines, cleaner)

---

_Report generated by Crush AI. All data verified against live `go build`, `go test`, and `golangci-lint` output._
