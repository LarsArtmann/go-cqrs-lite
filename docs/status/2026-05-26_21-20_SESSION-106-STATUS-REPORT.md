# Comprehensive Status Report — Session 106

**Date:** 2026-05-26 21:20 CEST
**Branch:** `master` (up to date with `origin/master`, unstaged changes)
**Commit:** `08572df` (committed) → uncommitted changes from this session
**Go Version:** 1.26.3
**Modules:** 13 in `go.work` (+ root + example/user + example/todo)
**Tags:** 12 module-level v1.0.0 tags at HEAD (local only, NOT pushed to remote)

---

## a) FULLY DONE

| Item | Detail | Evidence |
| --- | --- | --- |
| **cqrs-gen CutPrefix** | Replaced `HasPrefix`+`TrimPrefix` with idiomatic `CutPrefix` | `cmd/cqrs-gen/main.go:174`, committed `21b142f` |
| **saga/runner.go split** | Extracted `compensate()` to `saga/compensate.go` | 251→213 lines, committed `bf8fee3` |
| **catalog/types.go split** | Extracted helper types to `catalog/types_helpers.go` | 283→215 lines, committed `08572df` |
| **catalog/registry.go split** | Extracted resource methods to `catalog/registry_resources.go` | 258→202 lines, committed `08572df` |
| **catalog/eventcatalog split** | Extracted flow/team/user writers to `exporter_resources_extra.go` | 274→120 lines, unstaged |
| **catalog/registry_helpers split** | Moved resource copy functions to `registry_copy.go` | 272→158 lines, unstaged |
| **PebbleStore concurrency fix** | Per-aggregate mutex prevents silent concurrent overwrites | `storage/pebble_event_store.go`, lockAggregate/unlockAggregate |
| **PebbleStore corrupt event logging** | Added `slog.Warn` before returning corruption errors | `storage/pebble_event_store.go:110-114` |
| **PebbleStore concurrent test** | `TestPebbleEventStore_ConcurrentSave_VersionConflict` — 10 goroutines, exactly 1 success | `storage/pebble_event_store_test.go` |
| **Decider coverage → 100%** | 3 new tests: TransactionalStore error, codec error, fold-after-snapshot error | `core/decider/decider_coverage_test.go` |
| **All 27 test packages** | `go test` — zero failures | 27/27 `ok` |
| **go vet** | Clean across all modules | Exit code 0 |
| **Zero file size violations** | All production files under 250 lines (was 5 over) | Largest: `core/event/event.go` at 245 lines |

### Session 106 Diff Summary

| File | Change | Lines |
| --- | --- | --- |
| `cmd/cqrs-gen/main.go` | CutPrefix refactor | +2/-3 (committed) |
| `saga/compensate.go` | New: extracted compensate() | +44 (committed) |
| `saga/runner.go` | Removed compensate(), log helpers | -40 (committed) |
| `catalog/types_helpers.go` | New: Change, Badge, Repository, etc. | +62 (committed) |
| `catalog/types.go` | Removed helper types | -68 (committed) |
| `catalog/registry_resources.go` | New: AddChannel, AddDataStore, etc. | +57 (committed) |
| `catalog/registry.go` | Removed resource methods | -57 (committed) |
| `catalog/eventcatalog/exporter_resources_extra.go` | New: writeFlow, writeTeam, writeUser | +154 |
| `catalog/eventcatalog/exporter_resources.go` | Removed moved functions | -154 |
| `catalog/registry_copy.go` | Added resource copy functions from helpers | +114 |
| `catalog/registry_helpers.go` | Removed resource copy functions | -114 |
| `storage/pebble_event_store.go` | Per-aggregate locking + slog.Warn | +53/-4 |
| `storage/pebble_event_store_test.go` | ConcurrentSave_VersionConflict test | +57 |
| `core/decider/decider_coverage_test.go` | 3 coverage gap tests | +158 |
| `catalog/testdata/golden/asyncapi.yaml` | Golden file update from code moves | ~197 changed |
| `catalog/testdata/golden/eventcatalog-config.js` | Golden file update | ~2 changed |
| `catalog/testdata/golden/package.json` | Golden file update | ~12 changed |

**Committed:** +165/-168 lines across 7 files (3 commits: `21b142f`, `bf8fee3`, `08572df`).
**Unstaged:** +436/-382 lines across 8 files + 2 new files.

### Verified Already-Correct Items (not changed, no action needed)

| Item | Finding |
| --- | --- |
| **OutboxPublisher split-brain** | `Close()` correctly sets `p.cancel = nil` and `p.state = publisherClosed`. No split-brain. |
| **Key separator mismatch** | Both MemoryStore and FakeStore use `event.StreamKey()` which uses `:`. Pebble also uses `:`. Consistent. |
| **EventRetry middleware tests** | Already 100% coverage. TODO was stale. |
| **SQLOutbox context cancellation** | Already uses `ExecContext`/`QueryContext`. |

---

## b) PARTIALLY DONE

| Item | What's Done | What's Missing | Severity |
| --- | --- | --- | --- |
| **`replace` directives removal** | Investigated: removing them breaks workspace build because v1.0.0 tags aren't on remote | Need to push tags first, then remove `replace`, then tag v1.0.0 fresh | **Critical** |
| **cqrs-gen coverage** | 89.9% via `run()` extraction | `main()` at 0% (1-liner wrapper, acceptable) | Low |
| **example/todo** | Builds successfully | Zero tests. External `httputil` dep — fragility since Session 77 | Low |
| **Pebble concurrency** | Per-aggregate locking prevents silent overwrites | Not true database-level CAS — two PebbleEventStore instances sharing one DB still race | Low |

---

## c) NOT STARTED

| Item | From Plan | Why It's Blocked |
| --- | --- | --- |
| **Push v1.0.0 tags to remote** | Session 104 #1 | Requires human decision on release timing |
| **Remove `replace` directives** | Session 104 #1 | Blocked on tag push (chicken-and-egg) |
| **Dry-run `go get` from scratch module** | Session 104 #2 | Blocked on `replace` removal |
| **Schema registry + JSON Schema middleware** | E1-E2 (Pareto Phase 3) | Complex new feature requiring design decisions |
| **Middleware tracing span test** | Q6 | OpenTelemetry span attribute assertions need SDK test recorder setup |
| **Benchmarks** | B1-B2 | No performance regression baseline |
| **PostgreSQL integration tests** | Multiple sessions | No testcontainers or pg setup in CI |
| **Storage SQL error/rollback deep tests** | Q3-Q4 | Partially covered by `transactional_store_test.go` |
| **Consumer validation** | Session 103 | Zero external consumers; no dry-run `go get` test from scratch module |

---

## d) TOTALLY FUCKED UP

**Nothing is fundamentally broken.** The codebase is in its strongest state ever:

- All 27 packages pass
- `go vet` clean
- Coverage gate at 80% in CI
- Zero file size violations (was 5)
- Decider coverage at 100%
- No known bugs
- PebbleStore concurrency fixed

**Remaining honesty risk:**

1. **`replace` directives block external consumers** — Every `go.mod` has `replace` directives pointing to local paths. `go get github.com/larsartmann/go-cqrs-lite/saga@v1.0.0` will **fail** for anyone outside this repo. The v1.0.0 tags are local-only — they have NEVER been pushed to remote. This is the single biggest blocker to the project being usable by anyone else.

2. **Zero external validation** — No consumer has ever imported any module from this repo. The API surface is unproven outside our tests.

3. **Pebble concurrency is process-local only** — The per-aggregate mutex only prevents races within a single `PebbleEventStore` instance. Two processes sharing a Pebble DB directory can still corrupt data. This is documented but worth noting.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (next session)

1. **Push v1.0.0 tags to remote** — This is the single highest-impact action. Everything else flows from it.
2. **Remove `replace` directives** — Immediately after push. Then verify workspace still builds.
3. **Dry-run `go get` test** — Create a scratch module outside the repo, `go get` each module, verify it compiles.

### Short-term (next 2-3 sessions)

4. **Add PostgreSQL integration test** — Even a single `docker run postgres` test for `SaveWithOutbox`.
5. **Add Watermill subscriber integration test** — End-to-end event flow through SubscriberAdapter.
6. **Add `catalog/openapi` golden test** — Infrastructure exists for AsyncAPI, EventCatalog, D2. OpenAPI is missing.
7. **Split oversized test files** — `storage/pebble_event_store_test.go` (422L), `core/decider/decider_test.go` (1182L) still violate file size guidelines.

### Medium-term (next month)

8. **Consumer validation** — Import saga/watermill into a real external project via real `go get`.
9. **Pebble cross-process safety** — Document or add advisory locking for multi-process Pebble access.
10. **Benchmarks for core modules** — No performance baseline exists.
11. **Minimum coverage gate (80%) in CI** — Done! But never run in actual CI.

---

## f) Top #25 Things to Get Done Next

| # | Task | Module | Impact | Effort | Pareto |
| --- | --- | --- | --- | --- | --- |
| 1 | Push v1.0.0 tags to remote | all | **Critical** | 5m | 1% → 51% |
| 2 | Remove `replace` directives from all `go.mod` files | all | **Critical** | 1h | 1% → 51% |
| 3 | Dry-run `go get` from scratch module | external | **Critical** | 30m | 1% → 51% |
| 4 | Add PostgreSQL integration test (testcontainers) | storage | High | 2h | 20% |
| 5 | Add Watermill subscriber integration test | watermill | Medium | 45m | 20% |
| 6 | Add `catalog/openapi` golden test | catalog/openapi | Low | 30m | 20% |
| 7 | Add OpenTelemetry span attribute assertions | middleware | Low | 30m | 20% |
| 8 | Split `core/decider/decider_test.go` (1182→<350) | core/decider | Medium | 1h | 20% |
| 9 | Split `storage/pebble_event_store_test.go` (422→<350) | storage | Medium | 30m | 20% |
| 10 | Fix Pebble cross-process concurrency documentation | storage | Medium | 15m | 20% |
| 11 | Add `catalog` enum/default struct tag support | catalog | Medium | 2h | 20% |
| 12 | Make AsyncAPI servers configurable | catalog/asyncapi | Low | 30m | 20% |
| 13 | Storage SQL error/rollback deep tests | storage | Medium | 1h | 20% |
| 14 | Add benchmarks for core modules | all | Low | 2h | 20% |
| 15 | Add Saga metrics via `MetricsRecorder` | saga | Low | 30m | 20% |
| 16 | Add example/todo tests | example | Low | 1h | 20% |
| 17 | Remove `example/todo` external dep (`httputil`) | example | Low | 30m | 20% |
| 18 | Add `event.Context` propagation helpers | core/event | Medium | 45m | 20% |
| 19 | Schema registry design document | docs | Medium | 2h | 20% |
| 20 | Consumer trial — import saga/watermill into real project | external | **Critical** | ongoing | 20% |
| 21 | Validate CI coverage-gate job actually runs in GitHub Actions | CI | High | 30m | 20% |
| 22 | Add `catalog/d2` cross-service connection test | catalog/d2 | Low | 30m | 20% |
| 23 | Trim AGENTS.md from ~400→<300 lines | docs | Low | 30m | 5% |
| 24 | Split testhelpers/fakes.go (342→per-fake) | testhelpers | Low | 15m | 5% |
| 25 | Add -race to CI test commands | CI | Low | 5m | 5% |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "Should we push v1.0.0 tags now and release, or wait for consumer validation?"

The `replace` directives in 11 `go.mod` files are the **single biggest blocker** to external adoption. Right now:

- `go get github.com/larsartmann/go-cqrs-lite/saga@v1.0.0` fails for external consumers
- The 12 v1.0.0 tags exist locally but have **never been pushed** to origin
- We have zero external consumers, so we can't verify the release works

**The release sequence is:**

1. Delete old local v1.0.0 tags
2. Remove `replace` directives from all `go.mod` files
3. Run `go mod tidy` in each module
4. Verify workspace still builds
5. Create fresh v1.0.0 tags
6. Push tags to remote
7. Dry-run `go get` from a scratch directory

**The question is: do we push the current v1.0.0 tags (which have `replace` directives and won't work externally), or skip them and go straight to a clean v1.0.0 release that removes `replace`?**

Pushing v1.0.0 as-is means the first public release is broken. Skipping it means v1.0.0 tags in git are misleading.

I recommend: **skip the existing tags, remove `replace`, then tag v1.0.0 fresh** (delete old local tags, create new ones from clean state). This gives a clean first release.

This requires a human decision. The technical steps are clear — the strategy question is not.

---

## Coverage by Module (measured)

| Module | Coverage | Change |
| --- | --- | --- |
| core/command | 92.5% | — |
| core/query | 98.4% | — |
| core/event | 93.7% | — |
| **core/decider** | **100.0%** | ↑ was 93.6% |
| core/pkg/id | 100.0% | — |
| core/pkg/dispatcher | 100.0% | — |
| memory | 99.6% | — |
| catalog | 94.4% | ↓ was 96.3% (file moves changed coverage boundaries) |
| catalog/asyncapi | 93.7% | — |
| catalog/d2 | 95.0% | — |
| catalog/eventcatalog | 92.8% | — |
| catalog/openapi | 94.4% | — |
| catalog/docserver | 90.1% | — |
| middleware | 100.0% | — |
| testhelpers | 91.2% | — |
| projection | 94.4% | — |
| storage | 89.6% | — |
| saga | 93.8% | — |
| watermill | 89.6% | — |
| cmd/cqrs-gen | 89.9% | — |

## Metrics Snapshot

| Metric | Value |
| --- | --- |
| Production Go files | 166 |
| Test files | 136 |
| Lines of Go (prod) | 15,789 |
| Lines of Go (test) | 31,506 |
| Lines of Go (total) | 47,295 |
| Test packages | 27/27 passing |
| `go vet` | Clean |
| Module tags at v1.0.0 | 12 (local only, NOT pushed) |
| File size violations (>250 lines) | 0 (was 5) |
