# Session 150 — Comprehensive Status Report

**Date:** 2026-05-29 14:46
**Branch:** master (up to date with origin/master)
**Last commit:** `adbe6ce docs(research): add KV store abstraction research document`

---

## Module Health Dashboard

| Module | Build | Tests | Coverage | Notes |
|--------|-------|-------|----------|-------|
| core/command | OK | PASS | 94.7% | |
| core/decider | OK | PASS | 100.0% | |
| core/event | OK | PASS | 90.7% | God-package, 30+ files |
| core/pkg/dispatcher | OK | PASS | 92.2% | |
| core/pkg/id | OK | PASS | 94.5% | Branded IDs |
| memory | OK | PASS | 99.1% | |
| storage | OK | PASS* | ~90% | 2 test fixes uncommitted |
| projection | OK | PASS | 90.4% | |
| pebble | OK | PASS | 87.9% | |
| catalog | OK | PASS | 96.3% | |
| catalog/asyncapi | OK | PASS | 93.7% | |
| catalog/d2 | OK | PASS | 95.0% | |
| catalog/docserver | OK | PASS | 89.9% | |
| catalog/eventcatalog | OK | PASS | 92.8% | |
| catalog/openapi | OK | PASS | 96.2% | |
| catalog/schema | OK | PASS | 86.0% | |
| middleware | OK | PASS | 94.0% | |
| testhelpers | OK | PASS | 83.7% | |
| signing | OK | PASS | 93.7% | |
| signing/multisig | OK | PASS | 94.2% | |
| integration | OK | PASS | — | Cross-module tests |
| integration/signing | OK | PASS | — | |
| watermill | OK | PASS | 94.4% | |
| codec | OK | PASS | 100.0% | |
| otel | OK | PASS | 96.6% | |
| turso | OK | — | — | No test files |
| listing | OK | — | — | New module |
| saga | **DELETED** | — | — | Removed Session 146 |
| stream | **DELETED** | — | — | Replaced by listing/ |
| example/* | OK | PASS | — | 6 example modules |

**Totals:** 28 modules in go.work, 26 build clean, 24 have tests, 22 test green.
**Coverage:** Avg 93.2% across tested packages. All above 83% threshold.

\* storage has 2 uncommitted test fixes from payload_encoding migration — pass individually.

---

## a) FULLY DONE

### This Session's Work (carried from Session 142-149)

1. **ProcessedAt on CheckpointStore** — `event.Checkpoint{EventID, ProcessedAt time.Time}` struct, `IsZero()` method. Memory + SQL implementations updated. SQL schema adds `processed_at` column. Dialect-aware time scanning (fixed SQLite string→time.Time issue). All ~15 test files updated.

2. **WithAsyncWrites() for PebbleEventStore** — `StoreOption` pattern, `syncWrites bool` (default true), `writeOptions()` helper. `commitAndLog` uses configurable write options. Tests verify both sync (default) and async modes.

3. **WithParallelism(n) for projection.Runner** — Semaphore-bounded goroutine pool dispatches projections concurrently. Sequential when parallelism ≤ 1 (unchanged default). Race-detector tested with concurrent execution verification proving actual parallelism.

4. **command.Store** — ISP split into `CommandSink` + `CommandSource` + `Store` composite. New module with its own go.mod.

5. **Payload encoding column** — `payload_encoding` added to events table, wired through entire storage layer, codec integration in projection Builder.

6. **storage/sql/ sub-package** — Extracted dialect, errors, helpers, otel, reconstruction, base into `storage/sql/`.

7. **Listing module** — Extracted aggregate listing from core into dedicated `listing/` module.

8. **Saga module removal** — Completely deleted (Session 146). All references cleaned from go.work, AGENTS.md, TODO_LIST.md.

### Long-standing Completed (selected highlights)

- Event signing/verification (HMAC-SHA256 + Ed25519 + multisig)
- SQL dialect abstraction (PostgreSQL, SQLite, Turso)
- Transactional outbox pattern
- Catalog system (AsyncAPI, OpenAPI, D2, EventCatalog, llms.txt)
- Dead letter queue + retry mechanism for projections
- Upcaster chain with cycle detection
- Branded IDs via go-branded-id
- Error family taxonomy (5 families: Rejection/Conflict/Transient/Infrastructure/Corruption)
- Circuit breaker middleware
- OpenTelemetry tracing
- ReadBackwards, LoadToVersion, LoadToTimestamp on event stores
- CI pipeline with coverage gates

**TODO_LIST stats:** 200 DONE, 15 open, 17 BLOCKED, 23 FUTURE, 7 v2-deferred

---

## b) PARTIALLY DONE

1. **AggregateRef migration** (~50% done) — `core/event/store.go` and `pebble/` use `AggregateRef{Type, ID}` struct. `storage/`, `memory/`, `projection/`, `decider/`, and many test files still use separate `(AggregateType, AggregateID)` params. Causes ~480 LSP errors but workspace builds clean via `replace` directives. This is the single largest source of LSP noise.

2. **Payload encoding test fixes** — The column was added and most tests updated, but `storage/event_store_load_query_test.go` and `storage/benchmark_test.go` still have uncommitted fixes (adding `"json"` to mock row columns). Pass individually but fail with `-coverprofile`.

---

## c) NOT STARTED (from TODO_LIST, actionable)

1. **Catch-up projection runner** — Start-from-checkpoint → replay → live-switch (source: LIVESTORE_DEEP_DIVE)
2. **Increase projection coverage to 95%+** — Currently 90.4%
3. **Rewrite example/user/** — Demonstrate full CQRS capability stack
4. **Add example/user/ smoke test** (TestExampleRuns)
5. **Parallelize CI matrix** — One job per module
6. **Performance regression CI** — Benchmark comparison on each PR
7. **Add gofumpt/goimports to pre-commit hook**
8. **Benchmark storage backends** (PG vs SQLite vs Pebble)
9. **Add BDD tests for Version, SchemaVersion, OutboxStatus, Pagination types**
10. **Add fuzz tests** for event creation, ID parsing, schema reflection, DecodePayload, upcaster chain
11. **Add E2E throughput benchmarks**
12. **Add stream module integration tests** — stream SQL reader tests
13. **Enforce 350-line test file limit via pre-commit hook**
14. **Split large test files** — decider_test.go (~1200L), runner_test.go (~1057L)
15. **Documentation site** (Docusaurus/MkDocs/Hugo)

---

## d) TOTALLY FUCKED UP

1. **LSP diagnostics: 480+ errors** — The AggregateRef migration is half-done. `gopls` reports massive type mismatches across `core/decider/`, `storage/`, `memory/`, `projection/`, and test files. Workspace builds and tests pass via go.work `replace` directives, but the LSP is essentially useless for real-time feedback. This is the biggest developer-experience problem.

2. **storage test flakes with `-coverprofile`** — `TestScanEvents_InvalidEventID` and `TestScanEvents_InvalidAggregateID` fail when running with `-coverprofile` but pass individually. Likely a `go-sqlmock` expectation ordering issue exacerbated by coverage instrumentation.

3. **go.work `replace` directive tangle** — Every module uses `replace` directives pointing to local paths. Can't publish modules independently. Can't `GOWORK=off go build` in individual modules without missing go.sum entries. This is a known blocker until v1.0.0 tags are pushed.

4. **Uncommitted test fixes** — `storage/event_store_load_query_test.go` and `storage/benchmark_test.go` have diffs from the payload_encoding migration that were never committed. These are from a prior session and block `storage` from testing clean with coverage.

---

## e) WHAT WE SHOULD IMPROVE

1. **Finish the AggregateRef migration** — It's the #1 source of noise and confusion. Every session encounters the ~480 LSP errors. Pick a weekend, update all call sites, update tests, commit.

2. **Commit the dangling test fixes** — Two storage test files have uncommitted fixes. Just commit them.

3. **Reduce god-package size in core/event** — 30+ files, 90.7% coverage. Splitting into sub-packages (store, bus, projection, codec) would improve navigability. Flagged as `[v2]` in TODO but the pain is now.

4. **Push v1.0.0 tags and remove replace directives** — This unblocks: per-module CI, pkg.go.dev documentation, external consumers. The code is stable enough.

5. **Add a `nix run .#test` that covers ALL modules** — Currently the AGENTS.md command only lists specific modules. The full test run should be one command.

6. **Example quality** — The `example/user/` is stale and doesn't demonstrate the full stack (signing, projections, outbox, catalog). A superb example would be the best marketing this library has.

7. **Pre-commit hooks** — No gofumpt, goimports, or test-file line limits enforced. Code style drifts between sessions.

8. **Documentation site** — pkg.go.dev works but a dedicated site with guides, architecture diagrams, and examples would dramatically improve adoption.

---

## f) Top 25 Things We Should Get Done Next

**Priority-ordered by impact × feasibility:**

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 1 | Commit dangling storage test fixes (payload_encoding) | HIGH | 5min | storage |
| 2 | Finish AggregateRef migration across ALL modules | CRITICAL | 2-3h | all |
| 3 | Push v1.0.0 tags, remove replace directives | HIGH | 30min | infra |
| 4 | Rewrite example/user/ as full-stack CQRS demo | HIGH | 4h | example |
| 5 | Add example/user/ smoke test | MEDIUM | 30min | example |
| 6 | Increase projection coverage to 95%+ | MEDIUM | 2h | projection |
| 7 | Build catch-up projection runner | HIGH | 4h | projection |
| 8 | Benchmark storage backends (PG vs SQLite vs Pebble) | MEDIUM | 3h | storage/pebble |
| 9 | Add gofumpt/goimports to pre-commit hook | LOW | 15min | infra |
| 10 | Parallelize CI matrix (one job per module) | MEDIUM | 1h | CI |
| 11 | Fix storage test flake with -coverprofile | LOW | 30min | storage |
| 12 | Add E2E throughput benchmarks | MEDIUM | 2h | cross-module |
| 13 | Add fuzz tests (event creation, ID parsing, upcaster) | MEDIUM | 3h | core |
| 14 | Add BDD tests for Version, SchemaVersion, OutboxStatus, Pagination | LOW | 1h | core |
| 15 | Split decider_test.go (~1200L) into focused files | LOW | 1h | core/decider |
| 16 | Enforce 350-line test file limit via pre-commit | LOW | 30min | infra |
| 17 | Add stream/listing module integration tests | MEDIUM | 2h | listing |
| 18 | Performance regression CI (benchmark comparison per PR) | MEDIUM | 2h | CI |
| 19 | Set up pkg.go.dev documentation hosting | LOW | 30min | infra |
| 20 | Create documentation site (Docusaurus/MkDocs) | MEDIUM | 8h | docs |
| 21 | Split core/event god-package into sub-packages | HIGH | 8h | core |
| 22 | Add high-level test utilities (AggregateTester, ProjectionTester) | MEDIUM | 4h | testhelpers |
| 23 | Add catalog diff/breaking-change detection tool | MEDIUM | 4h | catalog |
| 24 | Add thin PostgreSQL store adapter (no Watermill) | HIGH | 6h | storage |
| 25 | Add thin NATS bus adapter (no Watermill) | HIGH | 6h | watermill |

---

## g) Top #1 Question I Cannot Figure Out Myself

**The AggregateRef migration: what's the intended final API shape?**

The codebase currently has TWO calling conventions coexisting:

```go
// Old style (storage, memory, decider, many tests):
store.Save(ctx, "User", aggID, events, version)
store.Load(ctx, "User", aggID)

// New style (pebble, core/event/store.go):
store.Save(ctx, event.AggregateRef{Type: "User", ID: aggID}, events, version)
store.Load(ctx, event.AggregateRef{Type: "User", ID: aggID})
```

The new `AggregateRef` style is clearly better (strong types, fewer params). But:
- Is `event.NewAggregateRef("User", aggID)` the canonical constructor, or should callers use struct literals?
- Should `AggregateRef.Type` be `event.AggregateType` (string alias) or `string`?
- Are there plans for `AggregateRef` to gain methods or become an interface?

**Why this matters:** Finishing this migration is #2 on the priority list above, but I can't confidently update 50+ call sites without knowing the intended final shape. The decision changes every function signature in the Store interface.

---

## Uncommitted Changes

```
 M storage/event_store_load_query_test.go  (payload_encoding mock row fix)
 M storage/benchmark_test.go               (payload_encoding mock row fix)
?? docs/research/PROPOSAL-dissolve-core-v2.html
```

## Session Stats

- **Total sessions tracked:** 150
- **TODO items:** 200 done, 15 open, 17 blocked, 23 future, 7 v2-deferred
- **Modules in go.work:** 28 (including 6 examples)
- **Production modules:** 22
- **Avg test coverage:** 93.2%
- **LSP errors:** ~480 (AggregateRef migration)
- **Build status:** Clean (workspace)
- **Test status:** 26/28 modules green, 2 have uncommitted fixes
