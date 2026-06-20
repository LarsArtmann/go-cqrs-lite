# Comprehensive Status Report — go-cqrs-lite

**Date:** 2026-06-13 06:02 UTC
**Branch:** `master` (up to date with origin)
**Commits:** 2,044 total (5 since v2.3.0 tag `4e221fbc`)
**Working tree:** CLEAN — zero uncommitted changes
**Go version:** 1.26.3

---

## Test Results

| Metric           | Value                              |
| ---------------- | ---------------------------------- |
| Packages tested  | 42                                 |
| Packages passing | 40                                 |
| Packages failing | 2 (pre-existing golden test drift) |
| Test files       | 351                                |
| Total Go files   | 677                                |
| Race detector    | PASS (all passing packages)        |

**Failing tests (PRE-EXISTING — not caused by recent work):**

| Package         | Test                             | Root Cause                                                       |
| --------------- | -------------------------------- | ---------------------------------------------------------------- |
| `codec/v2`      | `TestGolden_JSONCodec_Encode`    | Golden file expects indented JSON but codec outputs compact JSON |
| `middleware/v2` | `TestGolden_HealthCheckResponse` | Golden file drift — needs `-update`                              |

---

## Coverage Matrix

| Package                  | Coverage | Trend                 |
| ------------------------ | -------- | --------------------- |
| `decider`                | 100.0%   | —                     |
| `catalog/openapi`        | 100.0%   | —                     |
| `catalog/caseutil`       | 100.0%   | —                     |
| `memory`                 | 98.2%    | —                     |
| `dispatcher`             | 98.0%    | —                     |
| `id`                     | 97.5%    | —                     |
| `otel`                   | 97.3%    | ↑ from 73.0% (v2.2)   |
| `command`                | 97.1%    | —                     |
| `listing`                | 94.9%    | —                     |
| `catalog/d2`             | 94.3%    | —                     |
| `watermill`              | 94.3%    | —                     |
| `query`                  | 94.3%    | —                     |
| `signing`                | 94.1%    | —                     |
| `signing/multisig`       | 94.2%    | —                     |
| `catalog/asyncapi`       | 93.9%    | —                     |
| `catalog/eventcatalog`   | 92.8%    | —                     |
| `event`                  | 92.3%    | —                     |
| `integration/simulation` | 92.3%    | —                     |
| `schema`                 | 91.4%    | —                     |
| `projection`             | 91.4%    | —                     |
| `catalog/docserver`      | 90.1%    | —                     |
| `storage`                | 89.3%    | —                     |
| `storage/sql`            | 89.2%    | ↑ from 37.4% (v2.2)   |
| `encryption`             | 88.9%    | —                     |
| `snapshot`               | 88.9%    | —                     |
| `catalog`                | 88.3%    | —                     |
| `pebble`                 | 87.0%    | —                     |
| `catalog/schema`         | 86.0%    | —                     |
| `turso/indexing`         | 75.5%    | NEW                   |
| `turso`                  | 49.1%    | ↑ from 28.6% (v2.2)   |
| `event/eventtest`        | 17.8%    | — (test helpers)      |
| `catalog/cattest`        | 0.0%     | — (test-only helpers) |

**Average coverage (excluding test helpers): 91.4%**

---

## a) FULLY DONE ✅

### Session 2026-06-12 — TODO List Rebuild + Features Audit

Read 80+ docs/status and docs/planning files from `docs/*/2026-06*`. Verified every TODO item against actual source code. Rebuilt TODO_LIST.md and FEATURES.md from scratch.

**Specific items verified as DONE (previously claimed "not done" in docs):**

| Item                                 | What docs claimed          | Actual state                                                    |
| ------------------------------------ | -------------------------- | --------------------------------------------------------------- |
| Godoc examples for decider           | "No runnable examples"     | ✅ `example_test.go` with `ExampleRepository_Execute`           |
| Godoc examples for projection        | "No examples"              | ✅ `example_test.go` with `ExampleNewBuilder`                   |
| Godoc examples for signing           | "No examples"              | ✅ 6 examples across `signing/` and `signing/multisig/`         |
| Godoc examples for schema            | "No examples"              | ✅ `example_test.go` with `ExampleNewUpcaster`                  |
| CBOR cborEncMode error handling      | "Silently drops error"     | ✅ Panics on init failure                                       |
| Pebble unused Backend type           | "Dead state"               | ✅ Already removed                                              |
| ADR README.md only lists 3           | "Only lists ADR-0001–0003" | ✅ Lists all 15 ADRs                                            |
| InMemoryAggregateReader uncached     | "O(n log n) per call"      | ✅ Has cache with invalidation                                  |
| command.Type tests                   | "No tests"                 | ✅ `type_test.go` with full coverage                            |
| query.Type tests                     | "No tests"                 | ✅ `type_test.go` with full coverage                            |
| Ciphertext.Equal() not constant-time | "NOT constant-time"        | ✅ Uses `subtle.ConstantTimeCompare`                            |
| codec fuzz tests                     | "No fuzz_test.go"          | ✅ `codec_fuzz_test.go` with CBOR roundtrip + determinism tests |

### Architecture + Infrastructure (since v2.2.0)

- ✅ Turso indexing sub-package: Advisor, AutoIndexer, Policy, Priority, OTel tracing, benchmarks, checkpoints
- ✅ CBOR codec with deterministic canonical encoding
- ✅ Pebble CBOR envelope with JSON backward compat
- ✅ XChaCha20-Poly1305 + AES-256-GCM encryption
- ✅ otel/ coverage 73% → 97.3%
- ✅ storage/sql coverage 37.4% → 89.2%
- ✅ Zero lint issues across all 28 modules
- ✅ 2044 total commits, all modules tagged v2.3.0

---

## b) PARTIALLY DONE 🟡

| Item                               | Current State                             | What's Missing                                                                                      |
| ---------------------------------- | ----------------------------------------- | --------------------------------------------------------------------------------------------------- |
| **turso/ coverage**                | 49.1% (up from 28.6%)                     | `OpenSync`, `Push`, `Pull`, `Checkpoint`, `Stats` need real Turso server or better mocks to test    |
| **turso/indexing coverage**        | 75.5%                                     | `isUnsupportedPragma` false path, `maybeAnalyze` false path, some option combinations               |
| **`//nolint` justification audit** | event/command/query verified with reasons | Still need to verify: middleware (34+ suppressions), storage, catalog (36 suppressions), encryption |
| **Golden test infrastructure**     | Most modules have golden tests            | `codec/v2` and `middleware/v2` golden files are stale — tests fail                                  |

---

## c) NOT STARTED ⬜

### Documentation

| #   | Task                                              | Effort |
| --- | ------------------------------------------------- | ------ |
| 1   | Add godoc examples for `listing/` package         | 20 min |
| 2   | Document CBOR usage patterns in `codec/README.md` | 20 min |
| 3   | Add README section linking to `docs/benchmarks/`  | 10 min |

### Code Quality

| #   | Task                                                          | Effort |
| --- | ------------------------------------------------------------- | ------ |
| 4   | Add tests for `cmd/api-stability` (0% coverage)               | 30 min |
| 5   | Add `query.BasicQuery` metadata (correlation/tracing context) | 30 min |
| 6   | Clean test deps from 12 production go.mod files               | 2 hr   |
| 7   | Fix 31 `nolint:errcheck` in defer `.Close()` calls            | 1 hr   |
| 8   | Reduce 36 nolint suppressions in `catalog/`                   | 45 min |

### Encryption Module

| #   | Task                                              | Effort |
| --- | ------------------------------------------------- | ------ |
| 9   | Add `StaticKeyResolver` helper (map-based)        | 10 min |
| 10  | Add versioned ciphertext format (prefix byte)     | 30 min |
| 11  | Add `example/encryption/` project                 | 30 min |
| 12  | Add `storage.NewEncryptedEventStore` wrapper      | 2 hr   |
| 13  | Field-level encryption (`encryption/fieldlevel/`) | 4 hr   |

### Turso Indexing (Deferred)

| #   | Task                                     | Effort |
| --- | ---------------------------------------- | ------ |
| 14  | Comparison report generator (CLI tool)   | 2 hr   |
| 15  | Hooks API (`WithIndexingHooks`)          | 1 hr   |
| 16  | Schema evolution/migration integration   | 1 hr   |
| 17  | Health check integration with `listing/` | 1 hr   |

### CI & DevEx

| #   | Task                                       | Effort |
| --- | ------------------------------------------ | ------ |
| 18  | Docker build CI step (linux/amd64 + arm64) | 30 min |
| 19  | Playwright E2E tests for `example/user/`   | 4 hr   |
| 20  | Add `go vulncheck` to CI                   | 30 min |
| 21  | Benchmark regression detection gate        | 2 hr   |

### Phantom Types (Superb Types Sprint leftovers)

| #   | Task                                                    | Effort |
| --- | ------------------------------------------------------- | ------ |
| 22  | Add `String()` + `IsZero()` to 17 catalog phantom types | 30 min |
| 23  | Add `Int()` to `example/todo/domain.Priority`           | 5 min  |
| 24  | Bool→Enum conversions (7 locations)                     | 2 hr   |
| 25  | Split `catalog.Message` into Message+MessageMeta        | 30 min |

### Strategic / v2 Breaking Changes

| #   | Task                                           | Effort | Note              |
| --- | ---------------------------------------------- | ------ | ----------------- |
| 26  | Remove `io.Closer` from core interfaces        | 4 hr   | ADR-0010 accepted |
| 27  | Add global `TransactionID` branded type        | TBD    | ADR needed        |
| 28  | Split `event.Store` into Writer/Reader/Deleter | 8 hr   | Breaking          |
| 29  | Move HTTP code from middleware → `transport/`  | 8 hr   | Breaking          |
| 30  | Make event Core truly immutable                | 4 hr   | opts shallow-copy |

---

## d) TOTALLY FUCKED UP 🔴

### Pre-existing Golden Test Failures

Two test packages fail on every run. These are NOT flaky — they fail deterministically:

1. **`codec/v2/TestGolden_JSONCodec_Encode`** — Golden file at `codec/testdata/golden/json_encode.json` expects indented JSON (`{ "email": ... }`) but `JSONCodec.Encode` outputs compact JSON (`{"email":...}`). Fix: update golden with `-update` flag or change codec to output indented.

2. **`middleware/v2/TestGolden_HealthCheckResponse`** — Golden file at `middleware/testdata/golden/health-check-response.json` has drifted from actual output. Fix: update golden with `-update`.

**Impact:** These mask real regressions in CI. Any new golden test failure would blend in with these known failures.

### No Real Damage

No production code is broken. No data loss. No security vulnerabilities. No panics in production paths. The codebase is genuinely in its best-ever state — 91.4% average coverage, zero lint issues, race-free.

---

## e) WHAT WE SHOULD IMPROVE 🟠

### 1. Fix the Golden Test Drift (URGENT)

The two failing golden tests undermine CI confidence. Every developer who runs `go test ./...` sees `FAIL` and has to mentally filter "is this my change or the known failure?". This is a tax on every future change.

### 2. `cmd/api-stability` Has Zero Tests

The tool that guards against breaking API changes is itself untested. If it breaks silently, we lose our API stability guarantee. A simple smoke test (runs the tool, verifies exit code, checks output format) would take 15 minutes.

### 3. `query.BasicQuery` Has No Metadata

Commands have `Metadata` with correlation IDs, user IDs, etc. Events have the same. Queries have... nothing. This makes distributed tracing through the query path inconsistent. Every observability tool in the stack has a blind spot on queries.

### 4. `turso/` Coverage at 49.1%

The sync functionality (`OpenSync`, `Push`, `Pull`, `Checkpoint`, `Close`) can only be tested with a real Turso server. The local-only paths are well-tested. This is a fundamental limitation — either we accept it or set up integration test infrastructure.

### 5. Catalog Has 36 `//nolint` Suppressions

The worst offender in the codebase. This suggests design issues — either the catalog types are too complex for the linter rules, or the linter rules are too aggressive for catalog's use case. Either way, it deserves investigation.

### 6. 12 Production go.mod Files Have Test-Only Dependencies

Go doesn't support separate test-only `require` blocks, so test deps end up in production `go.mod`. This bloats consumer transitive dependency trees. The fix (extracting `eventtest/` to own module) is a breaking change, so it's deferred to the next major version.

### 7. No Automated Benchmark Regression Gate

Benchmarks exist, baselines exist, CI runs benchmarks — but there's no automated gate that fails when performance regresses. A human has to check. This is a latent risk for a performance-focused library.

---

## f) Top #25 Next Tasks (Priority-Ordered)

| #      | Task                                                         | Impact      | Effort | Why                                                     |
| ------ | ------------------------------------------------------------ | ----------- | ------ | ------------------------------------------------------- |
| **1**  | Fix 2 pre-existing golden test failures (codec + middleware) | 🔴 Critical | 5 min  | Every `go test` run shows FAIL — masks real regressions |
| **2**  | Add smoke test for `cmd/api-stability`                       | 🔴 High     | 15 min | API guard is untested — silent breakage risk            |
| **3**  | Add `StaticKeyResolver` helper to encryption                 | 🟡 Medium   | 10 min | 80% use case for key rotation, trivial to implement     |
| **4**  | Add `query.BasicQuery` metadata                              | 🟡 High     | 30 min | Distributed tracing blind spot on query path            |
| **5**  | Fix ADR-0005 gap (add placeholder or renumber)               | 🟢 Low      | 5 min  | Numbering inconsistency                                 |
| **6**  | Add `String()` + `IsZero()` to catalog phantom types         | 🟡 Medium   | 30 min | Consistency with `event.Type` pattern                   |
| **7**  | Fix `listing/` godoc example                                 | 🟢 Low      | 20 min | Only module without runnable examples                   |
| **8**  | Add CBOR pure→CBOR fuzz test case                            | 🟡 Medium   | 15 min | Current fuzz uses JSON intermediary                     |
| **9**  | Add CBOR DecMode strict configuration                        | 🟡 Medium   | 15 min | Reject unknown fields, duplicate keys                   |
| **10** | Add `go vulncheck` to CI                                     | 🟡 High     | 30 min | Security hardening — zero-cost insurance                |
| **11** | Audit `//nolint` in middleware, storage, catalog, encryption | 🟡 Medium   | 2 hr   | Verify all suppressions are justified                   |
| **12** | Docker build CI step (linux/amd64 + arm64)                   | 🟡 Medium   | 30 min | Multi-arch builds not validated in CI                   |
| **13** | Add benchmark regression CI gate                             | 🟡 Medium   | 2 hr   | Performance is a product promise                        |
| **14** | Document CBOR usage patterns                                 | 🟢 Low      | 20 min | Consumer-facing docs gap                                |
| **15** | Fix 31 `nolint:errcheck` in defer `.Close()`                 | 🟡 Medium   | 1 hr   | Lazy suppressions — use `defer func()` pattern          |
| **16** | Add `example/encryption/` project                            | 🟡 Medium   | 30 min | Only encryption module without standalone example       |
| **17** | Add versioned ciphertext format to encryption                | 🟡 Medium   | 30 min | Future-proof algorithm changes                          |
| **18** | Reduce catalog nolint suppressions (36→20)                   | 🟡 Medium   | 45 min | Worst package — investigate root cause                  |
| **19** | Add `Int()` method to `example/todo/domain.Priority`         | 🟢 Low      | 5 min  | Type requires explicit casts                            |
| **20** | Turso indexing: Hooks API (`WithIndexingHooks`)              | 🟡 Medium   | 1 hr   | Extensibility for pre/post index creation               |
| **21** | Turso indexing: Schema evolution integration                 | 🟡 Medium   | 1 hr   | Cross-module integration with `schema/`                 |
| **22** | Clean test deps from production go.mod files                 | 🟡 Low      | 2 hr   | Bloats consumer transitive deps                         |
| **23** | Add `storage.NewEncryptedEventStore` convenience             | 🟡 Medium   | 2 hr   | Common use case, no wrapper exists                      |
| **24** | Fix `turso/indexing` remaining coverage gaps                 | 🟡 Medium   | 1 hr   | `isUnsupportedPragma`, `maybeAnalyze` false paths       |
| **25** | Add Playwright E2E setup for `example/user/`                 | 🟡 Medium   | 4 hr   | Full-stack consumer confidence                          |

---

## g) Open Question Requiring Owner Decision 🟣

**Should the remaining Must\* panic wrappers be removed from the public API?**

The codebase has ~14 `MustParse*` / `MustNew*` functions remaining (e.g., `id.MustParse[T]`, `command.MustNew`, `event.MustNewEvents`, `snapshot.MustEveryNEvents`). These panic on invalid input. Three options:

1. **Delete entirely** — Replace ~100+ test call sites with `Parse + t.Fatal`. Purest approach but large mechanical change.
2. **Move to `internal/testutil`** — Keep convenience for tests, remove from public API. Consumers can't accidentally call them.
3. **Keep as-is** — Accept that `Must*` is a Go convention (e.g., `template.Must`). Document that they're for test/benchmark use only.

This is an API philosophy decision. The code works either way, but the choice affects every consumer's import experience. I recommend **option 2** — it preserves convenience for internal use while removing footguns from the public API. But this is a breaking change, so it should be bundled into the next major version.

---

## Project Health Dashboard

| Metric                       | Status                                |
| ---------------------------- | ------------------------------------- |
| Build                        | ✅ Clean                              |
| Tests (passing)              | 40/42 packages                        |
| Tests (failing)              | 2 pre-existing golden drift           |
| Lint                         | ✅ Zero issues                        |
| Race detector                | ✅ Clean                              |
| Coverage (avg)               | 91.4%                                 |
| Coverage (lowest production) | 49.1% (turso)                         |
| Dependencies                 | ✅ All tidy                           |
| Security                     | ✅ No known vulns                     |
| API stability                | ✅ Golden file guarded                |
| Documentation                | ✅ All modules have READMEs           |
| ADRs                         | 15 written (gap at 0005)              |
| Fuzz tests                   | 8 modules                             |
| Property tests               | 4 modules (event, command, query, id) |
| Benchmarks                   | 17+ scale benchmarks                  |

---

_This report was generated by reading 80+ docs files, verifying all claims against actual source code, and running the full test suite with coverage._
