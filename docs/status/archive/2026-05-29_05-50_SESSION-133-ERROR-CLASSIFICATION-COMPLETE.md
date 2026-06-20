# Session 133 — Full Comprehensive Status Report

**Date:** 2026-05-29 05:50 CEST
**Branch:** master
**Commits since last report:** 1 (session 132 status doc)
**Test Suite:** 29/29 packages PASS
**Build:** CLEAN

---

## a) FULLY DONE (This Session)

### 1. Signing Error Classification — COMPLETE ✅

**Scope:** 36 bare `fmt.Errorf` calls → 0 across 8 production files

All signing/ production errors now use the `event.*` error-family classification system.

| File                             | Calls | Pattern Used                                                                | Classification                                    |
| -------------------------------- | ----- | --------------------------------------------------------------------------- | ------------------------------------------------- |
| `signing/event.go`               | 2     | `event.WrapInfrastructure`                                                  | Event reconstruction, base64 decode               |
| `signing/signer.go`              | 2     | `event.WrapInfrastructure`, `event.Newf(Infrastructure)`                    | JSON unmarshal, base64 fallback                   |
| `signing/hmac.go`                | 1     | `event.Wrapf(Rejection)`                                                    | Key length validation (preserves `ErrInvalidKey`) |
| `signing/ed25519.go`             | 3     | `event.Wrapf(Rejection)`, `event.WrapInfrastructure`                        | Key validation + key generation                   |
| `signing/middleware.go`          | 8     | `event.WrapInfrastructure`, `event.Newf(Rejection)`                         | Sign/verify/corrupt, missing signature            |
| `signing/multisig.go`            | 6     | `event.WrapInfrastructure`, `event.WrapRejection`, `event.Wrapf(Rejection)` | Multi-actor sign/verify/validate                  |
| `signing/multisig_extract.go`    | 5     | `event.WrapInfrastructure`, `event.Newf(Rejection)`                         | JSON encode/decode, verifier lookup               |
| `signing/multisig_middleware.go` | 12    | `event.WrapInfrastructure`, `event.Newf(Rejection)`, `event.WrapRejection`  | All middleware operations                         |

**Impact:** `event.IsRetryable()` now works correctly for all signing errors. Crypto/infrastructure errors are Transient (retryable). Validation errors are Rejection (not retryable).

### 2. Stream Error Classification — COMPLETE ✅

**Scope:** 10 `fmt.Errorf` calls → 0 across 5 production files

| File                   | Pattern                                          | Details                        |
| ---------------------- | ------------------------------------------------ | ------------------------------ |
| `stream/types.go`      | `event.NewRejection`                             | Table prefix validation        |
| `stream/in_memory.go`  | `event.WrapInfrastructure`                       | Journal read errors            |
| `stream/middleware.go` | `event.WrapInfrastructure`                       | Tombstone/rebirth marking      |
| `stream/sql_reader.go` | `event.WrapInfrastructure`, `event.NewRejection` | SQL query/scan + Type required |
| `stream/projection.go` | `event.WrapInfrastructure`                       | Table creation                 |

### 3. Middleware Error Classification — COMPLETE ✅

**Scope:** 4 `fmt.Errorf` calls → 0 across 2 production files

| File                       | Pattern                   | Details                                                       |
| -------------------------- | ------------------------- | ------------------------------------------------------------- |
| `middleware/recovery.go`   | `event.Wrapf(Corruption)` | Panic recovery wrapping `ErrPanicRecovered`                   |
| `middleware/validation.go` | `event.Wrapf(Rejection)`  | Command/Event/Query validation wrapping `ErrValidationFailed` |

### 4. Interface Compliance Checks Added — COMPLETE ✅

| Type                             | Interface                | File                      |
| -------------------------------- | ------------------------ | ------------------------- |
| `saga.MemoryStore`               | `saga.Store`             | `saga/memory_store.go:17` |
| `stream.InMemoryAggregateReader` | `stream.AggregateReader` | `stream/in_memory.go:19`  |
| `stream.SQLAggregateReader`      | `stream.AggregateReader` | `stream/sql_reader.go:20` |

Previously verified as already correct:

- `memory.MemoryBus` → `event.Bus` ✅
- `memory.MemoryStore` → `event.Store`, `Journal`, `SeekableJournal`, `BackwardsSource`, `io.Closer` ✅

### 5. Deprecated Interface Assertions Removed — COMPLETE ✅

Removed `event.GlobalLoader` and `event.PositionalLoader` assertions from:

- `memory/store.go` (2 lines removed)
- `storage/event_store.go` (4 lines removed, multiline format)

Zero deprecated interface references remain in the entire codebase.

### 6. Storage SQLite Test Duplication — Already Resolved ✅

The `sqlite_integration_test.go` monolith was already deleted in a prior session. The LSP diagnostics were stale. Verified: build and tests pass cleanly.

---

## b) PARTIALLY DONE

### fmt.Errorf Classification Across Project — ~65% Complete

- **Done:** `signing/` (36 → 0), `stream/` (10 → 0), `middleware/` (4 → 0), `storage/` (already classified)
- **Remaining:** `core/` (4 calls), `catalog/` (29 calls), `projection/` (2 calls), `saga/` (1 call)

Remaining `fmt.Errorf` calls by module:
| Module | Count | Nature |
|--------|-------|--------|
| `core/pkg/id/` | 3 | ULID parsing — these are validation errors wrapping `errEmptyString` |
| `core/decider/` | 1 | `Load` error formatting — intentional formatted error builder |
| `catalog/` | 29 | File I/O in exporters — wrapping filesystem operations |
| `projection/` | 2 | Health check — wrapping checkpoint store / journal errors |
| `saga/` | 1 | Health check — nil store guard |

---

## c) NOT STARTED (from TODO_LIST.md)

### High Priority

- [ ] **Wire example/user/aggregate.go to use catalog-aware event constructors**
- [ ] **Add ProcessedAt to CheckpointStore** — store (EventID, time.Time) not just EventID
- [ ] **Add event.Context propagation** — thread ctx through NewEvent, PublishChanges

### Medium Priority

- [ ] **Build catch-up projection runner** (start-from-checkpoint → replay → live-switch)
- [ ] **Add background polling for InMemoryRunner** (currently push-model only)
- [ ] **Increase projection coverage to 95%+**
- [ ] **Add WithAsyncWrites() option for PebbleEventStore**
- [ ] **Add projection parallel processing — goroutine pool**
- [ ] **Parallelize CI matrix** — one job per module

### Low Priority

- [ ] **Rewrite example/user/** to demonstrate full CQRS capability stack
- [ ] **Add example/user/ smoke test** (TestExampleRuns)
- [ ] **Performance regression CI** — benchmark comparison on each PR
- [ ] **Add gofumpt/goimports to pre-commit hook**
- [ ] **Add BDD tests** for Version, SchemaVersion, OutboxStatus, Pagination types
- [ ] **Add fuzz tests** for event creation, ID parsing, schema reflection, DecodePayload, upcaster chain
- [ ] **Add E2E throughput benchmarks**
- [ ] **Add stream module integration tests**
- [ ] **Add stream SQL reader tests**
- [ ] **Enforce 350-line limit on test files via pre-commit hook**
- [ ] **Split large test files:** decider_test.go (~1200L), runner_test.go (~1057L)
- [ ] **Benchmark storage backends** (PG vs SQLite vs Pebble)

---

## d) TOTALLY FUCKED UP

### 1. codec/ Module Is Untracked

The `codec/` directory with 4 files (`codec.go`, `json.go`, `raw.go`, `go.mod`, `codec_test.go`) exists on disk but:

- Not tracked by git
- Not listed in `go.work`
- Referenced from `core/event/codec.go` which has a `Codec` interface and `JSONCodec` implementation
- Appears to be a new module extraction that was never committed
- **Risk:** Lost work if disk wiped; confusion about which Codec to use

### 2. Pre-existing Uncommitted Changes from Prior Sessions

17 modified files in the working tree that are NOT from this session:

- `core/decider/benchmark_test.go` — benchmark refactoring
- `core/decider/decider_coverage_test.go` — test cleanup
- `core/event/event.go`, `core/event/options.go` — new functionality (Encoding field?)
- `core/event/event_bdd_test.go` — BDD test updates
- `core/event/outbox_publisher_*.go` — outbox test cleanup
- `core/go.mod` — dependency changes
- `core/query/query_bdd_test.go` — BDD test updates
- `docs/status/SESSION-132-*.md` — status doc update
- `go.work` — workspace changes
- `integration/chaos_test.go` — chaos test update
- `saga/saga_bdd_test.go` — BDD test updates
- `storage/event_store_loadall_test.go` — test update
- `stream/listbuilder_bdd_test.go`, `stream/sql_bdd_test.go` — BDD test updates

These changes are all from prior sessions and were never committed. They need to be committed separately.

### 3. LSP Diagnostic Noise

The gopls diagnostics show 390+ "go mod tidy" errors for BDD test files that need `ginkgo`/`gomega` in `go.mod`. These are **false positives** — the test files compile and run fine because `GOWORK=off go test` resolves deps correctly. The LSP is confused by the workspace mode.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture

1. **Error classification in catalog/** — 29 unclassified `fmt.Errorf` calls in file I/O exporters. These should use `event.WrapInfrastructure` for filesystem operations.
2. **Error classification in core/pkg/id/** — 3 ULID parsing errors should be `event.WrapRejection` (validation failures, not retryable).
3. **Health check errors** — `projection/health.go` and `saga/health.go` should use `event.WrapInfrastructure` for consistency.
4. **codec/ module** — Needs to be committed and wired into `go.work`, or the work should be abandoned and deleted.

### Code Quality

5. **Test file size** — Multiple test files still exceed 350-line convention: `decider_test.go` (~1200L), `runner_test.go` (~1057L), `saga_bdd_test.go` (~650L).
6. **CI parallelization** — One job per module would cut CI time from ~10min to ~2min.
7. **Pre-commit hook** — Add gofumpt + goimports enforcement.

### Process

8. **Commit hygiene** — Prior sessions left 17 uncommitted files. The BuildFlow pre-commit hook creates a commit cycle (3-5 amend rounds) which discourages frequent commits.
9. **Stale LSP diagnostics** — The "go mod tidy" noise from gopls should be suppressed via gopls settings or the test files should use build tags.

---

## f) Top 25 Things to Get Done Next

| #   | Task                                                                  | Impact | Effort | Module     |
| --- | --------------------------------------------------------------------- | ------ | ------ | ---------- |
| 1   | **Commit the 17 pre-existing uncommitted files** (prior session work) | HIGH   | LOW    | all        |
| 2   | **Commit or delete the codec/ module**                                | HIGH   | LOW    | codec      |
| 3   | **Classify catalog/ errors** (29 `fmt.Errorf` → classified)           | MED    | MED    | catalog    |
| 4   | **Classify core/pkg/id/ errors** (3 calls → `event.WrapRejection`)    | MED    | LOW    | core       |
| 5   | **Classify projection/health.go errors** (2 calls)                    | LOW    | LOW    | projection |
| 6   | **Classify saga/health.go error** (1 call)                            | LOW    | LOW    | saga       |
| 7   | **Push signing/v1.0.0 tag** if not already done                       | HIGH   | LOW    | signing    |
| 8   | **Remove replace directives** from go.mod files (after v1.0.0 tags)   | HIGH   | MED    | all        |
| 9   | **Add ProcessedAt to CheckpointStore**                                | MED    | MED    | core       |
| 10  | **Build catch-up projection runner**                                  | HIGH   | HIGH   | projection |
| 11  | **Add stream module integration tests**                               | MED    | MED    | stream     |
| 12  | **Add stream SQL reader tests**                                       | MED    | MED    | stream     |
| 13  | **Add example/user/ smoke test**                                      | MED    | LOW    | example    |
| 14  | **Rewrite example/user/ for full CQRS demo**                          | HIGH   | HIGH   | example    |
| 15  | **Add fuzz tests** for event creation, ID parsing, upcaster chain     | MED    | MED    | core       |
| 16  | **Parallelize CI** — one job per module                               | MED    | MED    | CI         |
| 17  | **Split large test files** (decider_test.go, runner_test.go)          | LOW    | MED    | tests      |
| 18  | **Enforce 350-line limit via pre-commit hook**                        | LOW    | MED    | CI         |
| 19  | **Add gofumpt/goimports to pre-commit**                               | LOW    | LOW    | CI         |
| 20  | **Benchmark storage backends** (PG vs SQLite vs Pebble)               | MED    | HIGH   | storage    |
| 21  | **Add E2E throughput benchmarks**                                     | MED    | MED    | tests      |
| 22  | **Add background polling for InMemoryRunner**                         | MED    | MED    | core       |
| 23  | **Add projection parallel processing** (goroutine pool)               | MED    | HIGH   | projection |
| 24  | **Performance regression CI** (bench comparison on PRs)               | MED    | MED    | CI         |
| 25  | **Suppress gopls "go mod tidy" noise** for BDD test files             | LOW    | LOW    | tooling    |

---

## g) Top #1 Question I Cannot Figure Out Myself

**What is the intended future of the `codec/` module?**

The `codec/` directory exists on disk with a `Codec` interface, `JSONCodec`, `RawCodec`, and tests — but it's untracked by git and not in `go.work`. Meanwhile, `core/event/codec.go` already has its own `Codec` interface and `JSONCodec`. This creates a split-brain:

- Is `codec/` meant to **replace** `core/event/codec.go` (extracted as standalone module)?
- Or is it an **alternative** for a different use case (e.g., catalog serialization)?
- Or was it an **abandoned experiment** that should be deleted?

I need the owner to clarify before committing, wiring, or deleting.

---

## Session Stats

| Metric                            | Value                                             |
| --------------------------------- | ------------------------------------------------- |
| Production files modified         | 13                                                |
| Test files modified               | 0 (from this session)                             |
| `fmt.Errorf` eliminated (prod)    | 50                                                |
| Interface compliance checks added | 3                                                 |
| Deprecated assertions removed     | 4                                                 |
| Packages passing                  | 29/29                                             |
| Packages failing                  | 0                                                 |
| Remaining `fmt.Errorf` in prod    | 36 (core: 4, catalog: 29, projection: 2, saga: 1) |
