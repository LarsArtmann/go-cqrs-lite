# V4 Release Execution Plan

> **Created:** 2026-07-11 20:37
> **Status:** EXECUTION COMPLETE — all phases done, awaiting `git tag v4.0.0`
> **Total effort:** ~2 hours actual (estimated 4.5 hrs)
> **Risk level:** LOW. The 3 code breaking changes were already done pre-execution. This was mostly mechanical path migration + safety nets + documentation.

---

## What Actually Happened (Execution Retrospective)

| Planned                                                       | Actual                                                                                                                              | Delta                               |
| ------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| Fix envelope magic string (`"cqrs"` → `"cqrs-envelope-v1"`)   | **DROPPED** by user — `"$"` JSON key already provides 99% of collision avoidance. Extra bytes per record for near-zero benefit.     | -5 min saved + 1 blocker eliminated |
| `BackfillHandlerWithTransform` delegates to `BackfillHandler` | `BackfillHandlerWithTransform` **removed entirely** — consolidated into `BackfillHandler(broker)` which reads transform from broker | Cleaner, fewer functions            |
| `goexperiment.arenas goexperiment.jsonv2` tags                | Only `goexperiment.jsonv2` — `goexperiment.arenas` was removed when `arena_experiment.go` dead code was deleted                     | Tag count reduced                   |
| Phase 5.4: Shutdown ordering integration test                 | **Deferred** — current struct-literal tests cover the logic; non-blocking                                                           | Moved to post-v4 backlog            |
| `git tag v4.0.0` + push                                       | **Awaiting user approval** — all code, tests, and docs are ready                                                                    | Pending                             |

---

## Pareto Analysis (what actually delivered the value)

### The 1% that delivered 51%

**Envelope safety net + path migration.** These tasks produced a safe, tagged-ready v4:

|     | Task                                                                 | Status                                    |
| --- | -------------------------------------------------------------------- | ----------------------------------------- |
| 1   | Envelope backward-compat integration test (raw JSON → CBOR default)  | ✅ Done — `kv.TestTypedStore_Migration_*` |
| 2   | `/v3` → `/v4` module path migration (49 go.mod files, 750 .go files) | ✅ Done — all tests pass                  |
| 3   | `git tag v4.0.0`                                                     | ⏳ Awaiting user approval                 |

### The 4% that delivered 64%

**Above + documentation + the small breaking change:**

|     | Task                                                         | Status  |
| --- | ------------------------------------------------------------ | ------- |
| 4   | ADR-0053 for codec default flip                              | ✅ Done |
| 5   | CHANGELOG `[v4.0.0]` section                                 | ✅ Done |
| 6   | BackfillHandler → `*SSEBroker` (approved v4 breaking change) | ✅ Done |
| 7   | Backfill missing v3 git tags (v3.0.0–v3.7.1)                 | ✅ Done |

### The 20% that delivered 80%

**Above + polish:**

|     | Task                                                           | Status                                          |
| --- | -------------------------------------------------------------- | ----------------------------------------------- |
| 8   | HealthCheck on `OwnedDBHandle` (inheritable by all SQL stores) | ✅ Done                                         |
| 9   | Update FEATURES.md for v4                                      | ✅ Done                                         |
| 10  | Update migration guide with `/v3` → `/v4` path note            | ✅ Done                                         |
| 11  | Final build + test + api-stability + doc-check verification    | ✅ Done — 57+ packages pass, 880 doc refs valid |
| 12  | Test `WithShutdownDependency` through real `sqlite.New()`      | ⏳ Deferred (non-blocking)                      |

### Deferred items (NOT in v4)

|                                         | Task    | When                                                                                 | Why deferred |
| --------------------------------------- | ------- | ------------------------------------------------------------------------------------ | ------------ |
| Storage/ split (3 packages)             | v4.1    | Structural, not behavioral. Doing it in the same cut as path migration doubles risk. |
| Parquet/DuckDB modules                  | v4.1    | Additive, no breaking changes. Design complete at `docs/research/`.                  |
| License swap (PROPRIETARY → Apache-2.0) | Post-v4 | User decision, irreversible, explicitly after v4.                                    |
| Git history scrub                       | Post-v4 | User decision, irreversible, explicitly after v4.                                    |
| Postgres CI matrix                      | Post-v4 | Nice-to-have, not blocking.                                                          |
| README polish to "sales page"           | Post-v4 | Cosmetic.                                                                            |
| NATS/ValKey transport                   | Future  | New modules, no breaking changes.                                                    |

---

## Strategic Decision: Storage/ Split is DEFERRED to v4.1

**Reasoning:** The `/v3` → `/v4` path migration touches all 49 `go.mod` files and every import path in every `.go` file. The storage/ split ALSO touches import paths (creates `storage/eventstore/`, `storage/readmodel/`). Doing both simultaneously means debugging two independent sets of import failures at once. That's how you verschlimmbessern a system.

**The smart sequence:**

1. **v4 (now):** Path migration + 3 code breaking changes + BackfillHandler. Clean, mechanical, debuggable.
2. **v4.1 (next):** Storage/ split with deprecated re-exports. Can even ship as v4.0.1 if the re-exports make it non-breaking.

---

## Execution Phase Results

### Phase 1: Safety Nets — ✅ DONE

| #       | Task                                       | Status      | Notes                                                                      |
| ------- | ------------------------------------------ | ----------- | -------------------------------------------------------------------------- |
| ~~1.1~~ | ~~Fix envelope magic string~~              | **DROPPED** | User decided `"cqrs"` is fine — `"$"` JSON key handles collision avoidance |
| 1.2     | Verify existing envelope tests pass        | ✅          | All codec tests pass                                                       |
| 1.3     | Backward-compat test: raw JSON → CBOR read | ✅          | `kv.TestTypedStore_Migration_OldRawJSON_ReadByNewCBORDefault`              |
| 1.4     | Backward-compat test: mixed envelope + raw | ✅          | `kv.TestTypedStore_Migration_MixedOldAndNewData`                           |
| 1.5     | Full codec + kv test suites                | ✅          | All pass                                                                   |

### Phase 2: BackfillHandler Breaking Change — ✅ DONE

| #   | Task                                                   | Status | Notes                                                                                                    |
| --- | ------------------------------------------------------ | ------ | -------------------------------------------------------------------------------------------------------- |
| 2.1 | Change signature: `BackfillHandler(broker *SSEBroker)` | ✅     | Added `Journal()` + `PayloadTransform()` accessors on SSEBroker                                          |
| 2.2 | Remove `BackfillHandlerWithTransform`                  | ✅     | Consolidated — broker's transform is used directly                                                       |
| 2.3 | Update tests                                           | ✅     | 5 tests: ReturnsEvents, MissingAfterParam, LimitsTo1000, PayloadTransformFromBroker, NoJournalReturns503 |
| 2.4 | Update AGENTS.md + SKILL.md + FEATURES.md references   | ✅     | All doc references updated                                                                               |

### Phase 3: Path Migration `/v3` → `/v4` — ✅ DONE

| #    | Task                                                                        | Status | Notes                                                                                                                      |
| ---- | --------------------------------------------------------------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------- |
| 3.1  | `sed` all go.mod files: module paths + require versions                     | ✅     | Also fixed `v3.0.0-` pseudo-versions → `v4.0.0-`                                                                           |
| 3.2  | `sed` all .go import paths                                                  | ✅     | 750 files updated                                                                                                          |
| 3.3  | `git mv event/v3 event/v4` (directory rename)                               | ✅     |                                                                                                                            |
| 3.4  | Update go.work                                                              | ✅     | `./event/v4/eventtest`                                                                                                     |
| 3.5  | `go mod tidy -e` all modules                                                | ✅     | No errors                                                                                                                  |
| 3.6  | Fix external dep false positives (yaml/v3, shortuuid/v3, semver/v3, etc.)   | ✅     | sed caught `go.yaml.in/yaml/v3`, `lithammer/shortuuid/v3`, `Masterminds/semver/v3`, `go-task/slim-sprig/v3` — all reverted |
| 3.7  | Update `cmd/doc-check/exports.go` hardcoded `/v3`                           | ✅     | Now strips `/v4`                                                                                                           |
| 3.8  | Update `otel/spans.go` `ComponentTracer` format string                      | ✅     | Now `/%s/v4`                                                                                                               |
| 3.9  | Update scripts (check-api-stability-sync.sh, check-workspace-sync.sh, etc.) | ✅     |                                                                                                                            |
| 3.10 | Update `docs/api_surface.txt` via `api-stability -update`                   | ✅     | 2243 exports                                                                                                               |
| 3.11 | Full workspace build                                                        | ✅     | Zero errors                                                                                                                |
| 3.12 | Full workspace test (57+ packages)                                          | ✅     | All pass                                                                                                                   |
| 3.13 | `go vet`                                                                    | ✅     | Clean                                                                                                                      |
| 3.14 | `cmd/doc-check`                                                             | ✅     | 880 references valid across 34 packages                                                                                    |

### Phase 4: Documentation — ✅ DONE

| #   | Task                         | Status | Notes                                                         |
| --- | ---------------------------- | ------ | ------------------------------------------------------------- |
| 4.1 | ADR-0053: Codec default flip | ✅     | `docs/adr/0053-unified-codec-default-flip.md`                 |
| 4.2 | CHANGELOG `[v4.0.0]` section | ✅     | 4 breaking changes, migration steps                           |
| 4.3 | Update FEATURES.md           | ✅     | Envelope wrapping, codec flip, health checks, BackfillHandler |
| 4.4 | Update MIGRATION-GUIDE.md    | ✅     | All 4 breaking changes + path migration                       |

### Phase 5: Polish — ✅ DONE (1 deferred)

| #   | Task                                                           | Status      | Notes                                                        |
| --- | -------------------------------------------------------------- | ----------- | ------------------------------------------------------------ |
| 5.1 | `HealthCheck` on `OwnedDBHandle`                               | ✅          | `storage/sql/base.go` — all SQL stores inherit via embedding |
| 5.2 | Remove redundant `HealthCheck` from `*SQLEventStore`           | ✅          | Inherited now                                                |
| 5.3 | Verify `HealthCheck` works on all SQL store types              | ✅          | Existing health tests pass                                   |
| 5.4 | Shutdown ordering integration test through real `sqlite.New()` | ⏳ Deferred | Non-blocking, struct-literal tests cover logic               |

### Phase 6: Release — ✅ DONE (tag pending)

| #   | Task                                                    | Status | Notes                                                  |
| --- | ------------------------------------------------------- | ------ | ------------------------------------------------------ |
| 6.1 | Find commit SHAs for v3.0.0–v3.7.1 from CHANGELOG dates | ✅     | All identified via `git log --before`                  |
| 6.2 | Tag missing v3 releases                                 | ✅     | v3.0.0, v3.3.0, v3.4.0, v3.5.0, v3.6.0, v3.7.0, v3.7.1 |
| 6.3 | `cmd/api-stability -update` for final golden            | ✅     | 2243 exports, captures BackfillHandler change          |
| 6.4 | `git tag v4.0.0`                                        | ⏳     | Awaiting user approval                                 |
| 6.5 | `git push origin master --tags`                         | ⏳     | Awaiting user approval                                 |

---

## Risk Assessment (post-execution)

|                                                   | Risk   | Probability | Impact                                         | Mitigation                             | Status |
| ------------------------------------------------- | ------ | ----------- | ---------------------------------------------- | -------------------------------------- | ------ |
| Path migration breaks hidden import               | Medium | Medium      | Phased sed passes + full test after each phase | ✅ No hidden imports found             |
| Envelope magic string change breaks consumer data | N/A    | N/A         | **DROPPED** — magic string unchanged           | ✅ Risk eliminated                     |
| `go mod tidy` pulls unexpected deps               | Low    | Low         | Workspace-mode tidy, check diff                | ✅ Only workspace refs updated         |
| Storage/ split attempted alongside path migration | HIGH   | HIGH        | **DEFERRED to v4.1**                           | ✅ Not attempted                       |
| v3 tag backfill points to wrong commit            | Medium | Low         | Verified commit dates match CHANGELOG          | ✅ All dates verified                  |
| External deps caught by global sed                | HIGH   | Medium      | Post-sed audit + selective revert              | ✅ Found + fixed 4 external `/v3` deps |

---

## What NOT to Do (validated by execution)

1. **Do NOT bundle storage/ split with path migration.** ✅ Followed — deferred to v4.1.
2. **Do NOT manually edit import paths.** ✅ Followed — used global sed, then fixed external dep false positives.
3. **Do NOT skip the envelope backward-compat test.** ✅ Followed — `TestTypedStore_Migration_*` proves old data reads.
4. **Do NOT touch the event/ module structure.** ✅ Followed — not split.
5. **Do NOT do Parquet/DuckDB in v4.** ✅ Followed — deferred to v4.1.
6. **Do NOT refactor working code during the path migration.** ✅ Followed — mechanical sed only.

---

## Post-v4 Roadmap

|         | Version                                                     | Content                      | Effort |
| ------- | ----------------------------------------------------------- | ---------------------------- | ------ |
| v4.1    | Storage/ split (3 packages with deprecated re-exports)      | ~4 hrs                       |
| v4.2    | Parquet journal (`storage/parquet`)                         | ~3-4 days                    |
| v4.3    | DuckDB materializations (`storage/duckdb` + `stack/duckdb`) | ~4-5 days                    |
| Post-v4 | License swap + git history scrub                            | ~1 hr (user approval needed) |
| Post-v4 | README "sales page" rewrite + Postgres CI                   | ~2 hrs                       |
