# Status Report — 2026-06-01 18:46

## Executive Summary

**Build:** ✅ Clean | **Lint:** ✅ Zero issues | **Tests:** ✅ All 33 packages pass

**v2.0.0 readiness:** 75% — all P0/P1 bugs fixed, major refactoring complete, test gaps partially filled. Remaining work is P3–P5 (tests, examples, CI, architecture polish).

---

## a) FULLY DONE ✅

### Code Quality & Bug Fixes

| Item                                                     | Commit                 |
| -------------------------------------------------------- | ---------------------- |
| Replace panics with error returns in multisig middleware | `8c5736c0`             |
| Immutability bug — NewEvent doesn't clone payload        | `728023de`             |
| Data race in CatalogDispatcher.RegisterHandlerMeta       | `728023de`             |
| XSS in catalog/docserver/html.go                         | `728023de`             |
| MustParse panics in watermill/protocol.go                | `3c30b12a`             |
| Context leak in ImmutableEvent.Context()                 | `5322bd06`             |
| time.After timer leak in projection runner               | `bfe13c75`             |
| Uncapped exponential backoff in projection runner        | `bfe13c75`             |
| SQL column list duplication across storage               | `bdeee731`             |
| Turso doc.go import hack                                 | `9e958455`             |
| cqrs-gen broken query handler template                   | `5c984038` (committed) |
| Unused metricName constants in middleware                | `5c984038` (committed) |

### Refactoring & Deduplication

| Item                                                                                    | Commit      |
| --------------------------------------------------------------------------------------- | ----------- |
| Extract shared event reconstruction logic                                               | `c657ad0d`  |
| Consolidate test helpers into event/eventtest                                           | `442c56f8`  |
| Extract generic `sortedCopy` + `copyPtr` in catalog (14× → 2)                           | `afdd8362`  |
| Extract `checkClosed` helper in command/query dispatchers                               | uncommitted |
| Pebble stuttering rename: `PebbleEventStore` → `EventStore`, etc. + aliases             | `770f18bd`  |
| Turso stuttering rename: `TursoSyncDB` → `SyncDB`, `OpenTurso` → `Open`, etc. + aliases | `2e88aaf6`  |
| Pebble redundant backend switch removed (both cases identical → single error)           | `770f18bd`  |
| ID phantom type: `ULID(Of[struct{}])` → generic `ULID[T any]`                           | uncommitted |

### Reactive Streams

| Item                                             | Status      |
| ------------------------------------------------ | ----------- |
| samber/ro integration across event/command/query | ✅ Complete |
| EventBus, ReplayEventBus, BehaviorEventBus       | ✅          |
| FilterEventType, ReplayFilter, HandlerToObserver | ✅          |
| Map, ScanState, Tap operators                    | ✅          |

### Test Coverage Added This Session

| File                                | Tests Added  | Coverage Area                                                    |
| ----------------------------------- | ------------ | ---------------------------------------------------------------- |
| `event/slice_test.go`               | 8 tests      | SliceFromVersion, SliceToVersion, FilterByTimestamp + edge cases |
| `event/context_test.go`             | 6 tests      | deadlineCtx Deadline/Done/Err/Value                              |
| `dispatcher/catalog_test.go`        | 3 tests      | DispatcherWithCatalog Init/Inner + concurrent dispatch           |
| `memory/checkpoint_closed_test.go`  | 2 tests      | Save/Load after Close                                            |
| `watermill/subscriber_test.go`      | 3 tests      | Subscribe error, Close with closable bus, idempotent Close       |
| `example/saga-pattern/main_test.go` | 2 tests      | Compile check + file existence                                   |
| `example/listing/main_test.go`      | 2 tests      | Compile check + file existence                                   |
| `cmd/cqrs-gen/main_test.go`         | 5 assertions | Query template verifies [R any], query.Query param               |

### Documentation & Planning

| Item                                                        | Status            |
| ----------------------------------------------------------- | ----------------- |
| `docs/planning/2026-06-01_V2.0.0-RELEASE-EXECUTION-PLAN.md` | ✅ 67 micro-tasks |
| Saga module removed, example/saga-pattern added             | ✅                |
| Command store ISP split (Sink/Source/Store)                 | ✅                |

---

## b) PARTIALLY DONE ⚠️

| Item                       | What's Done                                      | What's Missing                                           |
| -------------------------- | ------------------------------------------------ | -------------------------------------------------------- |
| TODO_LIST.md cleanup       | ~25 items verified done in code                  | Not marked `[x]` in the file — still shows `[ ]`         |
| Test coverage — turso      | —                                                | Zero test files. Entire module untested.                 |
| Test coverage — projection | 88.3%                                            | Target 95%+, needs ~7% more                              |
| Test coverage — storage    | 72.7%                                            | Lowest coverage of any library module                    |
| Test coverage — schema     | 77.6%                                            | Below 80% gate                                           |
| Test coverage — event      | 84.5%                                            | Below 90%, SliceFuncs + deadlineCtx now tested           |
| Example/user rewrite       | Smoke test exists, catalog write tested          | Not rewritten to demonstrate full CQRS stack             |
| CI improvements            | Benchmark + file-size-gate + coverage gate exist | No matrix parallelism, no govulncheck, no mod tidy check |

---

## c) NOT STARTED ❌

### From Execution Plan (P3–P5)

| #   | Task                                                           | Priority | Est. |
| --- | -------------------------------------------------------------- | -------- | ---- |
| T48 | BDD tests for Version, SchemaVersion, OutboxStatus, Pagination | P3       | 20m  |
| T50 | Fuzz tests for NewEvent, ID parsing, schema reflection         | P3       | 12m  |
| T51 | E2E throughput benchmarks                                      | P3       | 12m  |
| T52 | Storage backend benchmarks (SQLite vs Pebble)                  | P3       | 10m  |
| T53 | Increase projection coverage to 95%+                           | P3       | 8m   |
| T54 | Split integration/event BDD test (479L)                        | P3       | 10m  |
| T56 | Fix example/user/catalog.go semantic misuse                    | P4       | 5m   |
| T57 | Fix example/user/main.go writes to working dir                 | P4       | 5m   |
| T58 | Add example/user smoke test                                    | P4       | 10m  |
| T61 | Rewrite example/user for full CQRS showcase                    | P4       | 12m  |
| T64 | CI matrix parallelism (per-module jobs)                        | P4       | 12m  |
| T65 | Add gofumpt/goimports to CI                                    | P4       | 8m   |
| T66 | Add govulncheck to CI                                          | P4       | 8m   |
| T67 | Add go mod tidy -diff check to CI                              | P4       | 5m   |

### Architecture (P5)

| #      | Task                                                    | Est. |
| ------ | ------------------------------------------------------- | ---- |
| T30    | Fix decider → memory layer violation                    | 10m  |
| T31    | Fix storage → listing coupling                          | 12m  |
| T32    | Accept ctx in projection/health.go:IsRunning            | 5m   |
| T33-34 | Move cross-module test assertions to integration/       | 20m  |
| T27    | Optimize pebble/save.go checkVersion O(n) → O(1)        | 12m  |
| T28    | Optimize listing/in_memory.go — only keep last N events | 8m   |

### Not Even in Plan

| Task                                                                                    | Priority |
| --------------------------------------------------------------------------------------- | -------- |
| Update TODO_LIST.md — mark ~25 done items as `[x]`                                      | P2       |
| Remove unused test helpers (5 functions in pebble/storage)                              | P3       |
| Modernize watermill tests (t.Context, maps.Copy)                                        | P3       |
| Split files over 250 lines (decider 258L, dispatcher 253L, catalog/schema/reflect 263L) | P3       |

---

## d) TOTALLY FUCKED UP 💥

### Session Execution Problems

1. **Auto-commits happened without my knowledge.** The git stash/pop cycle during the session caused intermediate work to be committed automatically (commits `5c984038`, `770f18bd`, `2e88aaf6`, `408495fd`). I was unaware commits were happening behind the scenes. This means I can't accurately describe the commit history.

2. **Working tree was left dirty.** The session ended with 7 modified + 7 untracked files uncommitted. This creates risk of lost work.

3. **TODO_LIST.md not updated.** Despite verifying ~25 items as done, I never actually updated the file to mark them `[x]`. This means the next session will re-audit the same items.

4. **The execution plan had errors.** Claimed `copyPtr` was "extracted but not applied" — it IS applied. Claimed `cqrs-gen` was fixed — it was, but the fix was committed mid-session without clear attribution. Several "already done" items from the plan's audit section were wrong or incomplete.

5. **Test quality concerns.** The example smoke tests (`saga-pattern`, `listing`) just check compilation + file existence — they don't verify behavior. The `watermill/subscriber_test.go` mock bus is minimal. Several tests could be more thorough.

6. **Did not run `go vet` or race detector separately.** Only relied on `nix run .#test` which may not enable `-race` in all configurations.

---

## e) WHAT WE SHOULD IMPROVE 🎯

### Type Model Improvements

1. **`Metadata.Custom` values are untyped `string`.** Could introduce `MetadataValue` branded type for consistency with `MetadataKey`. Low impact but improves the type story.

2. **`query.Handler` returns `any`.** This is a known Go limitation — well-documented with the "typed bookend" pattern via `RegisterTyped[T]` / `DispatchTyped[T]`. Not fixable until Go has proper higher-kinded types.

3. **`event.NewEvent` takes `payload any`.** Unavoidable for JSON marshaling. Could potentially use generics (`NewEvent[T any](...)`) but the current design with `any` is idiomatic Go.

### Architecture Improvements

1. **Layer violations are the #1 architecture debt.** `decider → memory` and `storage → listing` violate the module layer graph. Fix requires interface extraction.

2. **~22 replace directives across 22 modules.** Standard for Go monorepos but blocks publishing. Requires tag push to resolve.

3. **File sizes — 4 files exceed 250-line standard.** `decider/decider.go` (258L), `dispatcher/dispatcher.go` (253L), `catalog/schema/reflect.go` (263L), `catalog/internal/cattest/builders.go` (354L).

### Dependency Improvements

4. **No action needed on dependencies.** All production deps are justified. `samber/lo` and `golang.org/x/exp` are transitive only. No over-engineered libraries. No missing libraries.

### Process Improvements

5. **Commit hygiene.** Should commit after each self-contained change with clear messages. Session auto-committed mid-work, creating unclear history.

6. **TODO_LIST.md must stay synchronized.** The gap between TODO_LIST.md and actual code is ~25 items. Every session should update it.

---

## f) TOP 25 THINGS TO GET DONE NEXT (Sorted by Impact/Effort)

| #   | Task                                                                   | Impact                | Effort | Ratio      |
| --- | ---------------------------------------------------------------------- | --------------------- | ------ | ---------- |
| 1   | Commit current uncommitted work (7 modified + 7 new files)             | 🔴 Prevents data loss | 2min   | ⭐⭐⭐⭐⭐ |
| 2   | Update TODO_LIST.md — mark ~25 done items as `[x]`                     | 🟠 Accuracy           | 10min  | ⭐⭐⭐⭐⭐ |
| 3   | Fix decider → memory layer violation (accept interface, remove import) | 🟡 Architecture       | 10min  | ⭐⭐⭐⭐   |
| 4   | Fix storage → listing coupling (move AggregateReader to shared)        | 🟡 Architecture       | 12min  | ⭐⭐⭐⭐   |
| 5   | Add turso connector tests (in-memory SQLite delegation)                | 🟠 Coverage gap       | 15min  | ⭐⭐⭐⭐   |
| 6   | Accept ctx in projection/health.go IsRunning                           | 🟡 Correctness        | 5min   | ⭐⭐⭐⭐   |
| 7   | Fix example/user/catalog.go semantic misuse                            | 🟠 API honesty        | 5min   | ⭐⭐⭐⭐   |
| 8   | Fix example/user/main.go writes eventcatalog to working dir            | 🟠 Side effect        | 5min   | ⭐⭐⭐⭐   |
| 9   | Add go mod tidy -diff to CI                                            | 🟢 Dep hygiene        | 5min   | ⭐⭐⭐     |
| 10  | Add govulncheck to CI                                                  | 🟢 Security           | 8min   | ⭐⭐⭐     |
| 11  | Split decider/decider.go (258L → 2 files)                              | 🟡 File size          | 8min   | ⭐⭐⭐     |
| 12  | Split catalog/schema/reflect.go (263L → 2 files)                       | 🟡 File size          | 8min   | ⭐⭐⭐     |
| 13  | Optimize pebble/save.go checkVersion O(n) → O(1)                       | 🟡 Perf               | 12min  | ⭐⭐⭐     |
| 14  | Increase storage coverage 72.7% → 80%+                                 | 🟠 Coverage           | 15min  | ⭐⭐⭐     |
| 15  | Increase schema coverage 77.6% → 80%+                                  | 🟡 Coverage           | 10min  | ⭐⭐⭐     |
| 16  | Increase projection coverage 88.3% → 95%+                              | 🟡 Coverage           | 10min  | ⭐⭐⭐     |
| 17  | Rewrite example/user for full CQRS showcase                            | 🟡 Consumer value     | 30min  | ⭐⭐       |
| 18  | Add BDD tests for Version, SchemaVersion, OutboxStatus                 | 🟢 Test quality       | 20min  | ⭐⭐       |
| 19  | Split integration/event BDD test (479L)                                | 🟡 File size          | 10min  | ⭐⭐       |
| 20  | Add fuzz tests for event creation, ID parsing                          | 🟢 Robustness         | 12min  | ⭐⭐       |
| 21  | E2E throughput benchmarks                                              | 🟢 Perf baseline      | 12min  | ⭐⭐       |
| 22  | Storage backend benchmarks (SQLite vs Pebble)                          | 🟢 Perf comparison    | 10min  | ⭐⭐       |
| 23  | CI matrix parallelism (per-module jobs)                                | 🟢 CI speed           | 12min  | ⭐⭐       |
| 24  | Move cross-module test assertions to integration/                      | 🟡 Module cycles      | 20min  | ⭐         |
| 25  | Optimize listing/in_memory.go — keep only last N events                | 🟢 Memory             | 8min   | ⭐         |

---

## g) TOP #1 QUESTION I CANNOT FIGURE OUT MYSELF

**How were intermediate commits created during the session?**

I see 4 commits (`5c984038`, `770f18bd`, `2e88aaf6`, `408549fd`) that I did not explicitly create. They appeared during a `git stash` / `git stash pop` cycle. The stash pop showed only 2 modified files, but the full set of refactoring changes (catalog generics, pebble rename, turso rename, formatting) were already committed in those intermediate commits.

**Questions:**

1. Is there a git hook or automation that auto-commits on certain triggers?
2. Was there another process (e.g., a CI bot or editor save action) committing changes?
3. The commit messages are well-formed (`refactor(catalog):...`, `refactor(store):...`) — who/what generated them?

This matters because I cannot give an accurate accounting of the commit history, which affects the release notes and changelog.

---

## Module Coverage Summary

| Module                    | Coverage  | Status            |
| ------------------------- | --------- | ----------------- |
| codec                     | 100.0%    | ✅                |
| decider                   | 100.0%    | ✅                |
| catalog/internal/caseutil | 100.0%    | ✅                |
| memory                    | 99.1%     | ✅                |
| otel                      | 96.4%     | ✅                |
| catalog/openapi           | 96.2%     | ✅                |
| watermill                 | 96.0%     | ✅                |
| catalog                   | 95.9%     | ✅                |
| catalog/d2                | 95.0%     | ✅                |
| id                        | 94.5%     | ✅                |
| middleware                | 94.5%     | ✅                |
| signing/multisig          | 94.1%     | ✅                |
| signing                   | 93.9%     | ✅                |
| listing                   | 93.8%     | ✅                |
| catalog/asyncapi          | 93.7%     | ✅                |
| snapshot                  | 92.3%     | ✅                |
| catalog/eventcatalog      | 92.8%     | ✅                |
| catalog/docserver         | 90.1%     | ✅                |
| cmd/cqrs-gen              | 89.9%     | ✅                |
| pebble                    | 88.4%     | ✅                |
| projection                | 88.3%     | ⚠️ Target 95%+    |
| catalog/schema            | 86.1%     | ✅                |
| event                     | 84.5%     | ⚠️ Could improve  |
| query                     | 97.1%     | ✅                |
| command                   | 94.9%     | ✅                |
| dispatcher                | 97.0%     | ✅                |
| **storage**               | **72.7%** | 🔴 Below 80% gate |
| **schema**                | **77.6%** | 🔴 Below 80% gate |
| **turso**                 | **0.0%**  | 🔴 Zero tests     |

---

## Key Metrics

| Metric                         | Value                                                                                 |
| ------------------------------ | ------------------------------------------------------------------------------------- |
| Total production LOC           | 23,594                                                                                |
| Total test packages            | 33                                                                                    |
| Packages passing tests         | 33/33 (100%)                                                                          |
| Lint issues                    | 0                                                                                     |
| Build status                   | Clean                                                                                 |
| Replace directives             | 22 across 22 modules                                                                  |
| Files > 250 lines (production) | 4 (decider 258L, dispatcher 253L, catalog/schema/reflect 263L, cattest/builders 354L) |
| Commits since v1.7.1           | 57                                                                                    |
| Uncommitted files              | 7 modified + 7 untracked                                                              |
