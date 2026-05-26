# Comprehensive Status Report — Session 104

**Date:** 2026-05-26 09:11 CEST
**Branch:** `master` (ahead of origin by 12 commits)
**Commit:** `7d57e6a` (pre-session) → uncommitted changes from this session
**Go Version:** 1.26.3
**Modules:** 13 in `go.work` (+ root + example/user + example/todo)
**Tags:** 12 module-level v1.0.0 tags at HEAD

---

## a) FULLY DONE

| Item | Detail | Evidence |
|------|--------|----------|
| **Saga module** | `saga/` — Definition, Step, Instance, Runner, compensation, retry, timeout | 27 tests, 93.8% coverage, all pass |
| **Saga WithLogger wiring** | Logger now logs: started, step completed, saga completed, step failed, compensate failed | `saga/runner.go:241-252` — `logInfo`/`logError` helpers; all lifecycle events logged |
| **Watermill module** | Publisher + Subscriber adapters, metadata protocol | 89.6% coverage, all tests pass |
| **SQL StreamLoader** | `SQLEventStore.LoadStream` with cursor-based `sqlEventStream` | `storage/stream.go`, tests pass |
| **OutboxPoller** | Background worker polls outbox, publishes, acks batches | `storage/outbox_poller.go`, 8 tests pass |
| **cqrs-gen tool** | AST-based code generation for typed command/query handlers | 17 tests, 70.8% coverage, all pass |
| **FEATURES.md** | Documents saga, watermill, cqrs-gen, stream loading, OutboxPoller. PLANNED section cleaned. Maturity matrix updated | All modules listed with accurate coverage |
| **CHANGELOG.md** | Coverage claims match actual measured values: Saga 93.8%, Watermill 89.6% | Lines 22-24 |
| **README.md** | Replaced dead `catalog/adapters` references with current `catalog.Builder` API | Lines 415-436 |
| **All 27 test packages** | `go test` + `nix run .#test` — zero failures | 27/27 `ok` |
| **go vet** | Clean across all modules | Exit code 0 |
| **Git tags v1.0.0** | 12 tags: root + core, memory, catalog, middleware, testhelpers, integration, storage, projection, saga, watermill, cqrs-gen | `git tag -l '*v1.0.0'` |

### Session 104 Diff Summary

| File | Change | Lines |
|------|--------|-------|
| `FEATURES.md` | Added saga/watermill/cqrs-gen sections, updated storage rows, updated maturity matrix, cleaned PLANNED | +72 |
| `README.md` | Replaced `catalog/adapters` with `catalog.Builder` API | +18/-16 |
| `CHANGELOG.md` | Fixed coverage claims (Saga 93.8%, Watermill 89.6%) | +2/-2 |
| `saga/runner.go` | Wired `WithLogger` into all lifecycle events | +25 |
| `saga/saga_test.go` | 9 new tests: logger wiring, compensate failure, nil-compensate, timeout cancellation, edge cases | +451 |
| `cmd/cqrs-gen/main_test.go` | 12 new tests: error paths, scan edge cases, multi-path, test file skip, marker on TypeSpec | +238 |

**Total:** +795 lines, -29 lines across 6 files.

---

## b) PARTIALLY DONE

| Item | What's Done | What's Missing | Severity |
|------|-------------|----------------|----------|
| **cqrs-gen coverage** | 70.8% — scan/generate/extractMarker fully tested | `main()` CLI entry point not tested (flag parsing, os.Exit paths, file writing) | Low |
| **Outbox transaction co-participation** | `TransactionalStore.SaveWithOutbox` works | `SQLOutbox.Append` and `SQLEventStore.Save` still run in separate transactions when called individually — consumer must wire correctly | Low |
| **example/todo** | Builds successfully (`go build ./...` clean) | Zero tests. Uses external `github.com/larsartmann/httputil` dep — fragility noted since Session 77 | Low |
| **Pebble optimistic concurrency** | Store exists | Concurrent writes silently overwrite — noted since Session 45, never addressed | Medium |

---

## c) NOT STARTED

| Item | From Plan | Why It's Blocked |
|------|-----------|------------------|
| **Schema registry + JSON Schema middleware** | E1-E2 (Pareto Phase 3) | Complex new feature requiring design decisions on schema versioning |
| **Middleware tracing span test** | Q6 | OpenTelemetry middleware exists (`tracing.go`) but span attribute assertions need SDK test recorder setup |
| **Benchmarks** | B1-B2 | No performance regression baseline established |
| **PostgreSQL integration tests** | Multiple sessions | No testcontainers or pg setup in CI |
| **catalog d2 golden test** | D1 | No golden file infrastructure in `catalog/d2/` |
| **Storage SQL error/rollback deep tests** | Q3-Q4 | Partially covered by `transactional_store_test.go` but not exhaustive |
| **schemautil edge tests** | Q1-Q2 | `SchemaToAny` error path (marshal/unmarshal failure) not tested |
| **GOWORK=off CI matrix** | TODO_LIST.md | Would catch version drift but adds CI time |
| **Remove replace directives** | TODO_LIST.md | `go.work` is the single source, but `replace` in `go.mod` files still exist for local dev — blocks `go get` for external consumers |
| **Consumer validation** | Top question from Session 103 | Zero external consumers; no dry-run `go get` test from scratch module |

---

## d) TOTALLY FUCKED UP

**Nothing is fundamentally broken.** The honesty gaps from Session 103 have been resolved:

1. ~~**FEATURES.md lies about Saga, Watermill, cqrs-gen**~~ → **FIXED.** All three now have full sections with accurate coverage numbers.
2. ~~**CHANGELOG coverage claims inflated**~~ → **FIXED.** Saga: 93.8% (was claimed ~92%), Watermill: 89.6% (was claimed ~85%). Now uses measured values.
3. ~~**README references deleted `catalog/adapters`**~~ → **FIXED.** Replaced with current `catalog.Builder` API.
4. ~~**Saga `options.go` is dead code**~~ → **FIXED.** `WithLogger` now wired into Runner lifecycle. `WithRetryPolicy` and `WithRetryMultiplier` were already wired.
5. ~~**`core/aggregate` "deprecated" but first-class**~~ → **RESOLVED.** Package was deleted in Session 99.

**Remaining honesty risk:**

6. **`replace` directives block external consumers** — Every `go.mod` file (except root) has `replace` directives pointing to local paths. `go get github.com/larsartmann/go-cqrs-lite/saga@v1.0.0` will **fail** for anyone outside this repo. The v1.0.0 tags are published but not installable.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (this session — if time permits)

1. **Remove `replace` directives from all `go.mod` files** — This is the single highest-impact change for external usability. Without it, v1.0.0 tags are decorative.
2. **Dry-run `go get` test** — Create a scratch module outside the repo, `go get` each module, verify it compiles.

### Short-term (next 2-3 sessions)

3. **Get cqrs-gen coverage to >80%** — Test CLI via `exec.Command` of the built binary.
4. **Add PostgreSQL integration test** — Even a single `docker run postgres` test for `SaveWithOutbox` would validate the SQL dialect.
5. **Add Watermill subscriber integration test** — Test end-to-end event flow through SubscriberAdapter.
6. **Add `catalog/d2` golden test** — Infrastructure already exists for AsyncAPI and EventCatalog.

### Medium-term (next month)

7. **Consumer validation** — Import saga/watermill into a real external project via real `go get`.
8. **Pebble concurrency fix** — Reproduce and fix the silent overwrite bug.
9. **GOWORK=off CI matrix** — Catch version drift early.
10. **Minimum coverage gate (80%) in CI** — Prevent regression.

---

## f) Top #25 Things to Get Done Next

| # | Task | Module | Impact | Effort | Pareto |
|---|------|--------|--------|--------|--------|
| 1 | Remove `replace` directives from all `go.mod` files | all | **Critical** | 1h | 1% → 51% |
| 2 | Dry-run `go get` from scratch module | external | **Critical** | 30m | 1% → 51% |
| 3 | Add PostgreSQL integration test (testcontainers) | storage | High | 2h | 20% |
| 4 | Add cqrs-gen CLI test via `exec.Command` | cmd/cqrs-gen | Medium | 30m | 4% |
| 5 | Add Watermill subscriber integration test | watermill | Medium | 45m | 20% |
| 6 | Add `catalog/d2` golden test | catalog/d2 | Low | 30m | 20% |
| 7 | Add `SchemaToAny` marshal-failure test | catalog/schemautil | Low | 10m | 20% |
| 8 | Add context cancellation to `SQLOutbox` methods | storage | Medium | 30m | 20% |
| 9 | Fix PebbleStore optimistic concurrency | storage | Medium | 1h | 20% |
| 10 | Add `GOWORK=off` CI matrix job | CI | Medium | 30m | 20% |
| 11 | Add minimum coverage gate (80%) to CI | CI | Medium | 15m | 20% |
| 12 | Add OpenTelemetry span attribute assertions | middleware | Low | 30m | 20% |
| 13 | Add `catalog` enum/default struct tag support | catalog | Medium | 2h | 20% |
| 14 | Make AsyncAPI servers configurable | catalog/asyncapi | Low | 30m | 20% |
| 15 | Add `event.Event.Clone()` defensive copy method | core/event | Medium | 30m | 20% |
| 16 | Storage SQL error/rollback deep tests | storage | Medium | 1h | 20% |
| 17 | Add `eventcatalog` writer split (408→3 files) | catalog/eventcatalog | Low | 1h | 20% |
| 18 | Add benchmarks for core modules | all | Low | 2h | 20% |
| 19 | Add Saga metrics via `MetricsRecorder` | saga | Low | 30m | 20% |
| 20 | Add example/todo tests | example | Low | 1h | 20% |
| 21 | Remove `example/todo` external dep (`httputil`) | example | Low | 30m | 20% |
| 22 | Add `event.Context` propagation helpers | core/event | Medium | 45m | 20% |
| 23 | Schema registry design document | docs | Medium | 2h | 20% |
| 24 | Consumer trial — import saga/watermill into real project | external | **Critical** | ongoing | 20% |
| 25 | Update AGENTS.md with Session 104 learnings | docs | Low | 15m | 20% |

---

## g) Top #1 Question I Cannot Figure Out Myself

### "Should we remove `replace` directives now and publish v1.1.0, or wait for consumer validation?"

The `replace` directives in 11 `go.mod` files are the **single biggest blocker** to external adoption. Right now:
- `go get github.com/larsartmann/go-cqrs-lite/saga@v1.0.0` fails for external consumers
- The 12 v1.0.0 tags are published but effectively unusable
- We have zero external consumers, so we can't verify that removing `replace` + bumping versions will actually work

**The question is: do we:**

- **A)** Remove `replace`, bump to v1.1.0, and do a dry-run `go get` from a scratch directory (risk: we discover issues after publishing)
- **B)** Create a realistic example app that imports via real `go get` first, then publish (risk: more internal code, still not real consumer validation)
- **C)** Leave `replace` in place, accept that v1.0.0 is dev-only, and focus on consumer validation first (risk: the tags are misleading)

This requires a human decision. I recommend **A** — the dry-run is low-risk and the payoff is immediate usability.

---

## Metrics Snapshot

| Metric | Value |
|--------|-------|
| Production Go files | 196 |
| Test files | 142 |
| Lines of Go (prod+test) | 51,080 |
| Test packages | 27/27 passing |
| `go vet` | Clean |
| Module tags at v1.0.0 | 12 |
| Commits since last push | 12 + this session |

### Coverage by Module (measured)

| Module | Coverage | Status |
|--------|----------|--------|
| core/command | 92.5% | ✅ |
| core/query | 98.4% | ✅ |
| core/event | 93.6% | ✅ |
| core/decider | 93.6% | ✅ |
| core/pkg/id | 100.0% | ✅ |
| core/pkg/dispatcher | 100.0% | ✅ |
| memory | 99.6% | ✅ |
| catalog | 96.3% | ✅ |
| catalog/asyncapi | 93.7% | ✅ |
| catalog/d2 | 95.0% | ✅ |
| catalog/eventcatalog | 92.8% | ✅ |
| catalog/openapi | 94.4% | ✅ |
| catalog/docserver | 90.1% | ✅ |
| middleware | 100.0% | ✅ |
| testhelpers | 91.2% | ✅ |
| projection | 94.4% | ✅ |
| storage | 89.6% | ✅ |
| **saga** | **93.8%** | ✅ (was 80.3%) |
| watermill | 89.6% | ✅ |
| **cmd/cqrs-gen** | **70.8%** | ⚠️ (was 46.1%) |

---

## Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| `replace` directives block `go get` | **Certain** | **Critical** | Remove before any external release |
| cqrs-gen CLI untested for real use | Likely | Low | Add exec.Command test |
| No PostgreSQL validation | Likely | Medium | Add testcontainers test |
| Pebble concurrency bug | Unknown | Medium | Needs reproduction |
| Zero external consumers | **Certain** | High | Consumer validation needed |

---

_Generated: 2026-05-26 09:11 CEST_
_Session: 104_
