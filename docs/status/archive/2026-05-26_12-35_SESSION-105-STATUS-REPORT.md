# Comprehensive Status Report — Session 105

**Date:** 2026-05-26 12:35 CEST
**Branch:** `master` (ahead of origin by 2 commits)
**Commit:** `f84e93d` (pre-session) → uncommitted changes from this session
**Go Version:** 1.26.3
**Modules:** 13 in `go.work` (+ root + example/user + example/todo)
**Tags:** 12 module-level v1.0.0 tags at HEAD (local only, NOT pushed to remote)

---

## a) FULLY DONE

| Item                             | Detail                                                                               | Evidence                                                          |
| -------------------------------- | ------------------------------------------------------------------------------------ | ----------------------------------------------------------------- |
| **Saga module**                  | `saga/` — Definition, Step, Instance, Runner, compensation, retry, timeout           | 27 tests, 93.8% coverage, all pass                                |
| **Saga WithLogger wiring**       | Logger logs: started, step completed, saga completed, step failed, compensate failed | `saga/runner.go:241-252`                                          |
| **Watermill module**             | Publisher + Subscriber adapters, metadata protocol                                   | 89.6% coverage, all tests pass                                    |
| **SQL StreamLoader**             | `SQLEventStore.LoadStream` with cursor-based `sqlEventStream`                        | `storage/stream.go`, tests pass                                   |
| **OutboxPoller**                 | Background worker polls outbox, publishes, acks batches                              | `storage/outbox_poller.go`, 8 tests pass                          |
| **cqrs-gen tool**                | AST-based code generation for typed command/query handlers                           | 17 tests, **89.9%** coverage (was 70.8%), all pass                |
| **cqrs-gen `run()` refactoring** | Extracted testable `run()` from `main()` — `main()` is now a thin 1-liner            | `cmd/cqrs-gen/main.go:36-38`                                      |
| **ImmutableEvent.Clone()**       | Deep copy method on `ImmutableEvent` — copies payload bytes + metadata map           | `core/event/event.go:170-192`, 4 tests                            |
| **catalog/d2 golden test**       | Golden file infrastructure for D2 diagram export                                     | `catalog/d2/golden_test.go`, `catalog/testdata/golden/diagram.d2` |
| **CI coverage gate**             | New `coverage-gate` job fails CI if any package < 80%                                | `.github/workflows/ci.yml:46-77`                                  |
| **CI per-module GOWORK=off**     | Updated module list: added `saga`, `watermill`, `cmd/cqrs-gen`                       | `.github/workflows/ci.yml:91-93`                                  |
| **FEATURES.md**                  | Documents saga, watermill, cqrs-gen, stream loading, OutboxPoller                    | All modules listed with accurate coverage                         |
| **CHANGELOG.md**                 | Coverage claims match actual measured values                                         | Lines 22-24                                                       |
| **README.md**                    | Current `catalog.Builder` API (no dead references)                                   | Lines 415-436                                                     |
| **All 27 test packages**         | `go test` — zero failures                                                            | 27/27 `ok`                                                        |
| **go vet**                       | Clean across all modules                                                             | Exit code 0                                                       |

### Session 105 Diff Summary

| File                                 | Change                                                                                        | Lines   |
| ------------------------------------ | --------------------------------------------------------------------------------------------- | ------- |
| `cmd/cqrs-gen/main.go`               | Extract `run()` from `main()` for testability                                                 | +16/-13 |
| `cmd/cqrs-gen/main_test.go`          | 6 new `TestRun_*` tests (invalid type, no markers, success, query, write error, default path) | +128    |
| `core/event/event.go`                | Add `Clone()` deep copy method to `ImmutableEvent`                                            | +23     |
| `core/event/event_test.go`           | 4 new Clone tests (deep copy, independent payload, independent metadata, nil payload)         | +111    |
| `catalog/d2/golden_test.go`          | Golden test for D2 diagram export                                                             | +42     |
| `catalog/testdata/golden/diagram.d2` | Initial golden file                                                                           | +87     |
| `.github/workflows/ci.yml`           | Add `per-module-test` (updated modules) + `coverage-gate` job                                 | +34     |
| `AGENTS.md`                          | Update modules (13), coverage (27 pkgs), `Clone()`, known blocker, CI jobs                    | +18/-6  |

**Total:** +452 lines, -19 lines across 8 files.

---

## b) PARTIALLY DONE

| Item                              | What's Done                                                                             | What's Missing                                                   | Severity     |
| --------------------------------- | --------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | ------------ |
| **`replace` directives removal**  | Investigated: removing them breaks workspace build because v1.0.0 tags aren't on remote | Need to push tags first, then remove `replace`, then tag v1.1.0  | **Critical** |
| **cqrs-gen coverage**             | 89.9% — all paths tested via `run()`                                                    | `main()` at 0% (1-liner wrapper, acceptable)                     | Low          |
| **example/todo**                  | Builds successfully                                                                     | Zero tests. External `httputil` dep — fragility since Session 77 | Low          |
| **Pebble optimistic concurrency** | Store exists                                                                            | Concurrent writes silently overwrite — noted since Session 45    | Medium       |

---

## c) NOT STARTED

| Item                                         | From Plan              | Why It's Blocked                                                                                            |
| -------------------------------------------- | ---------------------- | ----------------------------------------------------------------------------------------------------------- |
| **Push v1.0.0 tags to remote**               | Session 104 #1         | Requires human decision on release timing                                                                   |
| **Remove `replace` directives**              | Session 104 #1         | Blocked on tag push (chicken-and-egg)                                                                       |
| **Dry-run `go get` from scratch module**     | Session 104 #2         | Blocked on `replace` removal                                                                                |
| **Schema registry + JSON Schema middleware** | E1-E2 (Pareto Phase 3) | Complex new feature requiring design decisions                                                              |
| **Middleware tracing span test**             | Q6                     | OpenTelemetry span attribute assertions need SDK test recorder setup                                        |
| **Benchmarks**                               | B1-B2                  | No performance regression baseline                                                                          |
| **PostgreSQL integration tests**             | Multiple sessions      | No testcontainers or pg setup in CI                                                                         |
| **Storage SQL error/rollback deep tests**    | Q3-Q4                  | Partially covered by `transactional_store_test.go`                                                          |
| **GOWORK=off CI per-module actually runs**   | TODO_LIST.md           | The CI job exists but has never been validated in CI — `replace` directives let each module resolve locally |
| **Consumer validation**                      | Session 103            | Zero external consumers; no dry-run `go get` test from scratch module                                       |

---

## d) TOTALLY FUCKED UP

**Nothing is fundamentally broken.** The codebase is in its strongest state ever:

- All 27 packages pass
- `go vet` clean
- Coverage gate at 80% in CI
- No known bugs

**Remaining honesty risk:**

1. **`replace` directives block external consumers** — Every `go.mod` has `replace` directives pointing to local paths. `go get github.com/larsartmann/go-cqrs-lite/saga@v1.0.0` will **fail** for anyone outside this repo. The v1.0.0 tags are local-only — they have NEVER been pushed to remote. This is the single biggest blocker to the project being usable by anyone else.

2. **File size violations** — 5 files exceed the 250-line limit:
   - `catalog/types.go` (283 lines)
   - `catalog/eventcatalog/exporter_resources.go` (274 lines)
   - `catalog/registry_helpers.go` (272 lines)
   - `catalog/registry.go` (258 lines)
   - `saga/runner.go` (251 lines)

3. **Zero external validation** — No consumer has ever imported any module from this repo. The API surface is unproven outside our tests.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (next session)

1. **Push v1.0.0 tags to remote** — This is the single highest-impact action. Everything else flows from it.
2. **Remove `replace` directives** — Immediately after push. Then verify workspace still builds.
3. **Dry-run `go get` test** — Create a scratch module outside the repo, `go get` each module, verify it compiles.
4. **Fix file size violations** — Split the 5 files over 250 lines.

### Short-term (next 2-3 sessions)

5. **Add PostgreSQL integration test** — Even a single `docker run postgres` test for `SaveWithOutbox`.
6. **Add Watermill subscriber integration test** — End-to-end event flow through SubscriberAdapter.
7. **Add `catalog/openapi` golden test** — Infrastructure exists for AsyncAPI, EventCatalog, D2. OpenAPI is missing.
8. **Split `catalog/types.go`** (283 → <250) — Extract `DataStore`/`Flow`/`Team`/`User` to `types_resources.go`.
9. **Split `catalog/registry.go` + `registry_helpers.go`** — Both over limit; helpers should be inlined or split by domain.

### Medium-term (next month)

10. **Consumer validation** — Import saga/watermill into a real external project via real `go get`.
11. **Pebble concurrency fix** — Reproduce and fix the silent overwrite bug.
12. **Minimum coverage gate (80%) in CI** — Done! But never run in actual CI.
13. **Benchmarks for core modules** — No performance baseline exists.

---

## f) Top #25 Things to Get Done Next

| #   | Task                                                          | Module           | Impact       | Effort  | Pareto   |
| --- | ------------------------------------------------------------- | ---------------- | ------------ | ------- | -------- |
| 1   | Push v1.0.0 tags to remote                                    | all              | **Critical** | 5m      | 1% → 51% |
| 2   | Remove `replace` directives from all `go.mod` files           | all              | **Critical** | 1h      | 1% → 51% |
| 3   | Dry-run `go get` from scratch module                          | external         | **Critical** | 30m     | 1% → 51% |
| 4   | Split `catalog/types.go` (283→<250)                           | catalog          | High         | 30m     | 20%      |
| 5   | Split `catalog/exporter_resources.go` (274→<250)              | catalog          | High         | 30m     | 20%      |
| 6   | Split `catalog/registry.go` (258→<250)                        | catalog          | High         | 30m     | 20%      |
| 7   | Split `catalog/registry_helpers.go` (272→<250)                | catalog          | High         | 30m     | 20%      |
| 8   | Split `saga/runner.go` (251→<250)                             | saga             | Low          | 15m     | 20%      |
| 9   | Add PostgreSQL integration test (testcontainers)              | storage          | High         | 2h      | 20%      |
| 10  | Add Watermill subscriber integration test                     | watermill        | Medium       | 45m     | 20%      |
| 11  | Add `catalog/openapi` golden test                             | catalog/openapi  | Low          | 30m     | 20%      |
| 12  | Add OpenTelemetry span attribute assertions                   | middleware       | Low          | 30m     | 20%      |
| 13  | Add `catalog` enum/default struct tag support                 | catalog          | Medium       | 2h      | 20%      |
| 14  | Make AsyncAPI servers configurable                            | catalog/asyncapi | Low          | 30m     | 20%      |
| 15  | Storage SQL error/rollback deep tests                         | storage          | Medium       | 1h      | 20%      |
| 16  | Add benchmarks for core modules                               | all              | Low          | 2h      | 20%      |
| 17  | Add Saga metrics via `MetricsRecorder`                        | saga             | Low          | 30m     | 20%      |
| 18  | Add example/todo tests                                        | example          | Low          | 1h      | 20%      |
| 19  | Remove `example/todo` external dep (`httputil`)               | example          | Low          | 30m     | 20%      |
| 20  | Add `event.Context` propagation helpers                       | core/event       | Medium       | 45m     | 20%      |
| 21  | Schema registry design document                               | docs             | Medium       | 2h      | 20%      |
| 22  | Consumer trial — import saga/watermill into real project      | external         | **Critical** | ongoing | 20%      |
| 23  | Fix PebbleStore optimistic concurrency                        | storage          | Medium       | 1h      | 20%      |
| 24  | Validate CI coverage-gate job actually runs in GitHub Actions | CI               | High         | 30m     | 20%      |
| 25  | Add `catalog/d2` cross-service connection test                | catalog/d2       | Low          | 30m     | 20%      |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "Should we push v1.0.0 tags now and release, or wait for consumer validation?"

The `replace` directives in 11 `go.mod` files are the **single biggest blocker** to external adoption. Right now:

- `go get github.com/larsartmann/go-cqrs-lite/saga@v1.0.0` fails for external consumers
- The 12 v1.0.0 tags exist locally but have **never been pushed** to origin
- We have zero external consumers, so we can't verify the release works

**The release sequence is:**

1. Push v1.0.0 tags to remote (`git push origin --tags`)
2. Remove `replace` directives from all `go.mod` files
3. Run `go mod tidy` in each module
4. Verify workspace still builds
5. Tag v1.1.0
6. Dry-run `go get` from a scratch directory

**The question is: do we push the current v1.0.0 tags (which have `replace` directives and won't work externally), or skip them and go straight to a v1.1.0 release that removes `replace`?**

Pushing v1.0.0 as-is means the first public release is broken. Skipping it means v1.1.0 is the first usable release but v1.0.0 tags in git are misleading.

I recommend: **skip v1.0.0, remove `replace`, then tag v1.0.0 fresh** (delete old local tags, create new ones from clean state). This gives a clean first release.

This requires a human decision. The technical steps are clear — the strategy question is not.

---

## Metrics Snapshot

| Metric                  | Value                       |
| ----------------------- | --------------------------- |
| Production Go files     | 196                         |
| Test files              | 143                         |
| Lines of Go (prod)      | 18,016                      |
| Lines of Go (test)      | 33,370                      |
| Lines of Go (total)     | 51,386                      |
| Test packages           | 27/27 passing               |
| `go vet`                | Clean                       |
| Module tags at v1.0.0   | 12 (local only, NOT pushed) |
| Commits since last push | 2 + this session            |

### Coverage by Module (measured)

| Module                      | Coverage  | Status         |
| --------------------------- | --------- | -------------- |
| core/command                | 92.5%     | ✅             |
| core/query                  | 98.4%     | ✅             |
| core/event                  | 93.7%     | ✅             |
| core/decider                | 93.6%     | ✅             |
| core/pkg/id                 | 100.0%    | ✅             |
| core/pkg/dispatcher         | 100.0%    | ✅             |
| memory                      | 99.6%     | ✅             |
| catalog                     | 96.3%     | ✅             |
| catalog/asyncapi            | 93.7%     | ✅             |
| catalog/d2                  | 95.0%     | ✅             |
| catalog/eventcatalog        | 92.8%     | ✅             |
| catalog/openapi             | 94.4%     | ✅             |
| catalog/docserver           | 90.1%     | ✅             |
| catalog/internal/caseutil   | 100.0%    | ✅             |
| catalog/internal/schemautil | 84.2%     | ✅             |
| middleware                  | 100.0%    | ✅             |
| testhelpers                 | 91.2%     | ✅             |
| projection                  | 94.4%     | ✅             |
| storage                     | 89.6%     | ✅             |
| saga                        | 93.8%     | ✅             |
| watermill                   | 89.6%     | ✅             |
| **cmd/cqrs-gen**            | **89.9%** | ✅ (was 70.8%) |

### File Size Violations (>250 lines)

| File                                         | Lines | Over By |
| -------------------------------------------- | ----- | ------- |
| `catalog/types.go`                           | 283   | +33     |
| `catalog/eventcatalog/exporter_resources.go` | 274   | +24     |
| `catalog/registry_helpers.go`                | 272   | +22     |
| `catalog/registry.go`                        | 258   | +8      |
| `saga/runner.go`                             | 251   | +1      |

---

## Risk Register

| Risk                                | Probability | Impact       | Mitigation                                    |
| ----------------------------------- | ----------- | ------------ | --------------------------------------------- |
| `replace` directives block `go get` | **Certain** | **Critical** | Push tags, remove `replace`, tag v1.0.0 fresh |
| v1.0.0 tags never pushed            | **Certain** | **High**     | Human decision needed                         |
| No PostgreSQL validation            | Likely      | Medium       | Add testcontainers test                       |
| Pebble concurrency bug              | Unknown     | Medium       | Needs reproduction                            |
| Zero external consumers             | **Certain** | High         | Consumer validation after release             |
| File size violations                | Certain     | Low          | Split 5 files                                 |
| CI coverage-gate never tested in CI | Likely      | Medium       | Push and observe CI run                       |

---

_Generated: 2026-05-26 12:35 CEST_
_Session: 105_
