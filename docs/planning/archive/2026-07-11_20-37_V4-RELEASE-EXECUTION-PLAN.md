# V4 Release Execution Plan

<!-- historical-artifact-banner -->

> **Historical session artifact.** This is a point-in-time snapshot from a past
> session. Many items marked TODO / Open / Not Started / Broken have since been
> resolved. See [CHANGELOG.md](../../../CHANGELOG.md) and
> [TODO_LIST.md](../../../TODO_LIST.md) for current state.
> Last documentation health audit: 2026-07-16.

> **Created:** 2026-07-11 20:37
> **Status:** EXECUTION COMPLETE — all phases done + storage split + shutdown fix. Awaiting `git tag v4.0.0`.
> **Total effort:** ~2.5 hours actual (estimated 4.5 hrs)
> **Risk level:** LOW. The code breaking changes were already done pre-execution. Path migration + storage split were mechanical. Shutdown fix was additive testing.

---

## What Actually Happened (Execution Retrospective)

| Planned                                  | Actual                                                                     | Delta             |
| ---------------------------------------- | -------------------------------------------------------------------------- | ----------------- |
| Fix envelope magic string                | **DROPPED** by user — `"$"` JSON key handles collision avoidance           | -1 blocker        |
| `BackfillHandlerWithTransform` delegates | `BackfillHandlerWithTransform` **removed entirely** — consolidated         | Cleaner           |
| Storage/ split deferred to v4.1          | **PULLED INTO v4.0.0** by user request — done with backward-compat aliases | +1 phase          |
| `WithShutdownDependency` test deferred   | **FIXED** by user request — integration tests through real `stack.New()`   | +1 fix            |
| `goexperiment.arenas` tag                | Removed — dead code was deleted in a previous session                      | Tag count reduced |
| `git tag v4.0.0` + push                  | **Awaiting user approval**                                                 | Pending           |

---

## Phase Results

### Phase 1: Safety Nets — ✅ DONE

| Task                                       | Status      | Notes                                                         |
| ------------------------------------------ | ----------- | ------------------------------------------------------------- |
| ~~Fix envelope magic string~~              | **DROPPED** | User decided `"cqrs"` is fine                                 |
| Backward-compat test: raw JSON → CBOR read | ✅          | `kv.TestTypedStore_Migration_OldRawJSON_ReadByNewCBORDefault` |
| Backward-compat test: mixed envelope + raw | ✅          | `kv.TestTypedStore_Migration_MixedOldAndNewData`              |

### Phase 2: BackfillHandler Breaking Change — ✅ DONE

| Task                                                   | Status | Notes                                              |
| ------------------------------------------------------ | ------ | -------------------------------------------------- |
| Change signature: `BackfillHandler(broker *SSEBroker)` | ✅     | Added `Journal()` + `PayloadTransform()` accessors |
| Remove `BackfillHandlerWithTransform`                  | ✅     | Consolidated                                       |
| Update tests                                           | ✅     | 5 tests including NoJournalReturns503              |
| Update AGENTS.md + SKILL.md + FEATURES.md              | ✅     |                                                    |

### Phase 3: Path Migration `/v3` → `/v4` — ✅ DONE

| Task                                     | Status | Notes                      |
| ---------------------------------------- | ------ | -------------------------- |
| sed all go.mod files                     | ✅     | Also fixed pseudo-versions |
| sed all .go import paths                 | ✅     | 750 files                  |
| `git mv event/v3 event/v4`               | ✅     |                            |
| Fix external dep false positives         | ✅     | 4 external deps reverted   |
| Update tool code (doc-check, otel/spans) | ✅     |                            |
| Update scripts                           | ✅     |                            |
| Full build + test                        | ✅     | 60+ packages pass          |

### Phase 4: Documentation — ✅ DONE

| Task                 | Status | Notes                                         |
| -------------------- | ------ | --------------------------------------------- |
| ADR-0053             | ✅     | `docs/adr/0053-unified-codec-default-flip.md` |
| CHANGELOG `[v4.0.0]` | ✅     | 5 breaking changes                            |
| FEATURES.md          | ✅     | Including storage split sub-packages          |
| MIGRATION-GUIDE.md   | ✅     | All 5 breaking changes                        |

### Phase 5: Polish — ✅ DONE

| Task                                       | Status | Notes                                                |
| ------------------------------------------ | ------ | ---------------------------------------------------- |
| `HealthCheck` on `OwnedDBHandle`           | ✅     | All SQL stores inherit                               |
| `WithShutdownDependency` integration tests | ✅     | Through real `stack.New()` with close-order tracking |
| Update FEATURES.md                         | ✅     |                                                      |

### Phase 6: Release — ✅ DONE (tag pending)

| Task                 | Status | Notes                  |
| -------------------- | ------ | ---------------------- |
| Backfill v3 tags     | ✅     | v3.0.0–v3.7.1          |
| api-stability golden | ✅     | 2206 exports           |
| `git tag v4.0.0`     | ⏳     | Awaiting user approval |

### Phase 7: Storage Split — ✅ DONE (added by user request)

| Task                                  | Status | Notes                                                 |
| ------------------------------------- | ------ | ----------------------------------------------------- |
| Extract `storage/eventstore/`         | ✅     | 8 files: event_store\*.go, snapshot.go, checkpoint.go |
| Extract `storage/readmodel/`          | ✅     | 1 file: kv_sql.go                                     |
| Backward-compat aliases in `storage/` | ✅     | Type aliases + constructor re-exports                 |
| Update `sql_backend.go`               | ✅     | Imports from new packages                             |
| Full workspace build + test           | ✅     | 60+ packages pass                                     |
| Update AGENTS.md tree                 | ✅     | New packages in directory tree                        |
| Update FEATURES.md                    | ✅     | New sub-package rows                                  |
| Update CHANGELOG                      | ✅     | Added to [v4.0.0] Added section                       |
| Update storage split proposal         | ✅     | Status → DONE                                         |

---

## Remaining Work Before Tagging

1. **Commit all 20 uncommitted files**
2. **Run `nix run .#lint`** — not yet executed
3. **Run `go test -race`** on full workspace — only ran on stack shutdown tests
4. **Update AGENTS.md Modules row** — flat module list missing `storage/eventstore`, `storage/readmodel`
5. **Consolidate `errStoreClosed`** — duplicate sentinel (same error code, different variables)
6. **`git tag v4.0.0`** + push — awaiting user approval

---

## Risk Assessment (post-execution)

| Risk                                              | Status          | Notes                                                      |
| ------------------------------------------------- | --------------- | ---------------------------------------------------------- |
| Path migration breaks hidden import               | ✅ Eliminated   | No hidden imports found                                    |
| External deps caught by global sed                | ✅ Fixed        | 4 external deps reverted                                   |
| Storage split breaks consumers                    | ✅ Eliminated   | Type aliases provide full backward compat                  |
| Shutdown ordering pointer mismatch                | ✅ Tested       | Integration tests prove pointer identity works             |
| Storage/ split attempted alongside path migration | ✅ Not an issue | Done after migration, separate concern                     |
| v3 tag backfill points to wrong commit            | ⚠️ Minor        | v3.7.0 and v3.7.1 point to same commit (docs-only release) |

---

## What NOT to Do (validated by execution)

1. ✅ Do NOT manually edit import paths — used global sed + targeted revert
2. ✅ Do NOT skip the envelope backward-compat test — migration tests prove old data reads
3. ✅ Do NOT touch the event/ module structure — not split
4. ✅ Do NOT refactor working code during path migration — mechanical only
5. ✅ Do NOT bundle storage/ split WITH path migration — done as separate phase after migration was verified

---

## Post-v4 Roadmap

| Version | Content                                                     | Effort                       |
| ------- | ----------------------------------------------------------- | ---------------------------- |
| v4.1    | Parquet journal (`storage/parquet`)                         | ~3-4 days                    |
| v4.2    | DuckDB materializations (`storage/duckdb` + `stack/duckdb`) | ~4-5 days                    |
| Post-v4 | License swap + git history scrub                            | ~1 hr (user approval needed) |
| Post-v4 | README "sales page" rewrite + Postgres CI                   | ~2 hrs                       |
