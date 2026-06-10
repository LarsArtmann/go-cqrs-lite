# Session 119 — Comprehensive Status Report

**Date:** 2026-05-28 12:21
**Branch:** `master` (commit `d20cd9c`, pushed to origin; uncommitted: 4 files)
**Scope:** `VerifierMap` convenience function, README fix, signing module ergonomics
**Previous:** Session 118 (`d20cd9c`)

---

## Executive Summary

Session 119 implemented the outcome of the `VerifyAll([]*MultiSigner)` vs `map[Actor]Verifier` PRO/CONTRA analysis. Decision: **keep `map[Actor]Verifier`** (interface segregation wins for a library) but add a `VerifierMap` convenience function to eliminate boilerplate. Also fixed the `map[string]` → `map[signing.Actor]` bug in README examples.

**Result:** Signing module at 94.2% coverage (up from 94.1%), 0 lint issues, all 28 packages pass.

---

## A. FULLY DONE

### Session 119 Work (this session)

| #   | Item                                                                           | Files                         | Status                                                          |
| --- | ------------------------------------------------------------------------------ | ----------------------------- | --------------------------------------------------------------- |
| 1   | `VerifierMap(signers ...*MultiSigner) map[Actor]Verifier` convenience function | `signing/multisig_extract.go` | Done — panics on nil input, works with zero args, 100% coverage |
| 2   | `ExampleVerifierMap` example test                                              | `signing/example_test.go`     | Done                                                            |
| 3   | Update `ExampleVerifyAll` to use `VerifierMap`                                 | `signing/example_test.go`     | Done — removed manual map construction                          |
| 4   | Fix README `map[string]` → `signing.VerifierMap()`                             | `signing/README.md`           | Done — both "Verify All" and "Require All" sections fixed       |
| 5   | `TestVerifierMap`, `TestVerifierMap_NilPanics`, `TestVerifierMap_Empty`        | `signing/multisig_test.go`    | Done — 3 new tests                                              |

### Carried Over (Sessions 117–118, fully done)

- Session 118: 15 improvements to signing module (nil guards, unexported types, format version, corrupt sig handling, file split, validation, coverage)
- Session 117: Journal naming migration, storage refactor
- Sessions 112–116: Full lint sweep, deduplication, formatting, code quality

---

## B. PARTIALLY DONE

Nothing partially done — all items in this session were completed.

---

## C. NOT STARTED

### Signing Module

| #   | Item                                 | Priority | Notes                                             |
| --- | ------------------------------------ | -------- | ------------------------------------------------- |
| 1   | Push signing v1.0.0 tag              | HIGH     | Code is ready, just needs tag + push              |
| 2   | Write `docs/signing-architecture.md` | MEDIUM   | Architecture decision doc for signing             |
| 3   | Split `signing_test.go` (1028L)      | MEDIUM   | Pre-commit warns at 350L for prod, 350L for tests |
| 4   | Split `multisig_test.go` (1338L)     | MEDIUM   | Same — well over limit                            |
| 5   | Add HMAC benchmarks                  | LOW      | `benchmark_test.go` exists but only has Ed25519   |
| 6   | Add Ed25519 benchmarks               | LOW      | Already present but could be expanded             |
| 7   | Add `VerifyAll` benchmark            | LOW      | Multi-actor verification perf baseline            |

### Cross-Module

| #   | Item                                                    | Priority | Notes                                                                      |
| --- | ------------------------------------------------------- | -------- | -------------------------------------------------------------------------- |
| 8   | Fix LSP errors in `projection/runner.go` and `storage/` | HIGH     | `event.Journal` undefined — needs API alignment after Session 117 refactor |
| 9   | Fix `storage/event_store_mock_test.go` errors           | HIGH     | `ReadFrom`/`ReadAll` undefined — mock out of sync                          |
| 10  | Update `TODO_LIST.md` to reflect signing work           | MEDIUM   | Still references pre-signing state                                         |
| 11  | `example/user/go.mod` signing version to v1.0.0         | MEDIUM   | Currently v1.6.0, should track tagged release                              |
| 12  | Move `example/todo` to own repository                   | BLOCKED  | Requires manual repo creation                                              |

### Catalog

| #   | Item                                            | Priority | Notes             |
| --- | ----------------------------------------------- | -------- | ----------------- |
| 13  | `catalog/internal/schemautil` coverage at 84.2% | LOW      | Lowest in catalog |
| 14  | `catalog/docserver` coverage at 90.1%           | LOW      |                   |

### Core

| #   | Item                                            | Priority | Notes                     |
| --- | ----------------------------------------------- | -------- | ------------------------- |
| 15  | Query handler `any` → generic `TypedHandler[T]` | v2       | Breaking change, deferred |
| 16  | `io.Closer` removal from core interfaces        | v2       | Breaking change, deferred |
| 17  | Global `TransactionID` branded type             | v2       | Breaking change, deferred |

---

## D. TOTALLY FUCKED UP

**Nothing in this session.** Clean execution.

**Historical note from Session 118:** The `middleware/common.go` "dead code" item was investigated and proved to be actively used by `retry.go` and `circuit_breaker.go`. This was correctly caught before deletion. Lesson: always verify before deleting.

---

## E. WHAT WE SHOULD IMPROVE

### Signing Module Maturity

1. **API surface is solid but docs are thin.** We have a README but no architecture decision record explaining WHY the signing module makes the choices it does (unexported structs, canonical format versioning, interface segregation for `VerifierMap`). ADR would help future contributors.

2. **Test files are obese.** `signing_test.go` at 1028L and `multisig_test.go` at 1338L are well beyond the 350L guideline. These should be split into focused test files (e.g., `hmac_test.go`, `ed25519_test.go`, `middleware_test.go`, `multisig_middleware_test.go`, `multisig_extract_test.go`).

3. **`attachMultiSignature` at 71.4% coverage.** The JSON marshal error path is untested. Low risk but worth a test.

4. **`hmacSigner.Verify` at 80.0% coverage.** The `s.Sign()` error path inside Verify is untested. Should add a test with a corrupted HMAC key state.

5. **No cross-module signing integration test.** The signing module is tested in isolation. No test exercises the full pipeline: `event.NewEvent` → `MultiSigner.Sign` → `event.Bus.Publish` → `RequireMultiSigMiddleware` → handler. The integration/ package could host this.

### Project-Level

6. **LSP errors in projection and storage** (12 errors total). These are pre-existing from the Session 117 Journal naming migration that wasn't fully completed. They don't block tests (Go workspace resolves them) but they confuse IDEs and LSP tooling.

7. **`TODO_LIST.md` is stale.** Last reconciled "Session 112b final cleanup" — doesn't mention signing at all. Needs a full reconciliation.

8. **No signing benchmarks.** `benchmark_test.go` has only Ed25519. HMAC and multi-sig verification perf is unmeasured.

---

## F. TOP 25 NEXT ACTIONS

### Immediate (do now)

| #   | Action                                                                  | Impact | Effort |
| --- | ----------------------------------------------------------------------- | ------ | ------ |
| 1   | Push signing v1.0.0 tag                                                 | HIGH   | 5 min  |
| 2   | Fix LSP errors in `projection/runner.go` (`event.Journal` undefined)    | HIGH   | 1 hr   |
| 3   | Fix `storage/event_store_mock_test.go` (`ReadFrom`/`ReadAll` undefined) | HIGH   | 30 min |
| 4   | Split `signing_test.go` (1028L → 4 files under 350L)                    | MED    | 1 hr   |
| 5   | Split `multisig_test.go` (1338L → 4 files under 350L)                   | MED    | 1 hr   |

### Short-term (this week)

| #   | Action                                             | Impact | Effort |
| --- | -------------------------------------------------- | ------ | ------ |
| 6   | Write `docs/signing-architecture.md` ADR           | MED    | 1 hr   |
| 7   | Update `TODO_LIST.md` with signing items           | MED    | 30 min |
| 8   | Add `attachMultiSignature` JSON marshal error test | LOW    | 15 min |
| 9   | Add `hmacSigner.Verify` error path test            | LOW    | 15 min |
| 10  | Add HMAC benchmarks to `benchmark_test.go`         | LOW    | 30 min |
| 11  | Add `VerifyAll` benchmark                          | LOW    | 30 min |
| 12  | Add cross-module signing integration test          | MED    | 2 hr   |

### Medium-term (next sessions)

| #   | Action                                                            | Impact | Effort |
| --- | ----------------------------------------------------------------- | ------ | ------ |
| 13  | Update `example/user/go.mod` to signing v1.0.0                    | LOW    | 5 min  |
| 14  | Improve `catalog/internal/schemautil` coverage (84.2% → 90%+)     | LOW    | 1 hr   |
| 15  | Improve `testhelpers` coverage (82.1% → 90%+)                     | LOW    | 1 hr   |
| 16  | Optimize Pebble LoadToTimestamp (full scan → timestamp bounds)    | MED    | 2 hr   |
| 17  | Add catalog diff/breaking-change detection tool                   | FUTURE | Large  |
| 18  | Add high-level test utilities (AggregateTester, ProjectionTester) | FUTURE | Large  |
| 19  | v2: Query handler generic `TypedHandler[T]`                       | HIGH   | Large  |
| 20  | v2: `io.Closer` removal from core interfaces                      | MED    | Medium |
| 21  | v2: Global `TransactionID` branded type                           | MED    | Medium |

### Infrastructure

| #   | Action                                            | Impact | Effort            |
| --- | ------------------------------------------------- | ------ | ----------------- |
| 22  | Move `example/todo` to own repository             | LOW    | Manual            |
| 23  | Publish go-composable-business-types as Go module | LOW    | Manual            |
| 24  | Modularize ActaFlow                               | LOW    | Different project |
| 25  | Evaluate nix flake migration from justfile        | LOW    | 4 hr              |

---

## G. TOP QUESTION

**Should the signing module be its own repository?**

The signing module has zero imports from any other `go-cqrs-lite` module (it only depends on `core/event` and `core/pkg/id`). It's the most self-contained module in the monorepo. Arguments for extracting:

- Independent versioning (v1.0.0 tag without tying to the monorepo)
- Smaller dependency footprint for consumers who only need signing
- Clear ownership boundaries

Arguments against:

- Currently benefits from shared CI, linting, and nix build
- The `go.work` workspace makes cross-module development easy
- Extraction adds release overhead

The answer determines whether we tag v1.0.0 in this repo or set up a new one. I cannot decide this autonomously.

---

## Module Health Dashboard

| Module                      | Coverage | Lint Issues | Notes                                 |
| --------------------------- | -------- | ----------- | ------------------------------------- |
| core/command                | 94.3%    | 0           |                                       |
| core/decider                | 91.1%    | 0           |                                       |
| core/event                  | 92.4%    | 0           |                                       |
| core/pkg/dispatcher         | 100.0%   | 0           |                                       |
| core/pkg/id                 | 100.0%   | 0           |                                       |
| core/query                  | 98.4%    | 0           |                                       |
| memory                      | 99.6%    | 0           |                                       |
| catalog                     | 96.3%    | 0           |                                       |
| catalog/asyncapi            | 93.7%    | 0           |                                       |
| catalog/d2                  | 95.0%    | 0           |                                       |
| catalog/docserver           | 90.1%    | 0           |                                       |
| catalog/eventcatalog        | 92.8%    | 0           |                                       |
| catalog/internal/caseutil   | 100.0%   | 0           |                                       |
| catalog/internal/schemautil | 84.2%    | 0           |                                       |
| catalog/openapi             | 94.4%    | 0           |                                       |
| middleware                  | 93.7%    | 0           |                                       |
| testhelpers                 | 82.1%    | 0           |                                       |
| integration/command         | —        | 0           | No statements                         |
| integration/event           | —        | 0           | No statements                         |
| integration/query           | —        | 0           | No statements                         |
| projection                  | 96.0%    | 12 LSP      | `event.Journal` undefined             |
| storage                     | 90.1%    | 12 LSP      | `ReadFrom`/`ReadAll` mock out of sync |
| saga                        | 93.4%    | 0           |                                       |
| watermill                   | 94.4%    | 0           |                                       |
| signing                     | 94.2%    | 0           |                                       |

**Totals:** 28 packages, 0 lint issues, 12 LSP errors (pre-existing, 2 modules), 41,539 total lines of Go code.

---

## Signing Module File Map

```
signing/                          3,882 total lines
├── signer.go                     171L  — Signature type, canonicalPayload, Signer/Verifier interfaces
├── multisig_middleware.go        172L  — 4 middleware functions with nil guards
├── multisig_types.go             154L  — Actor, SignatureEntry, MultiSignature, options
├── multisig.go                   153L  — NewMultiSigner, Sign, Verify, verifyActorEntry
├── example_test.go               150L  — 4 example functions
├── ed25519.go                    105L  — ed25519Signer, ed25519Verifier (unexported)
├── middleware.go                 108L  — SignMiddleware, VerifyMiddleware, RequireSignatureMiddleware
├── multisig_extract.go          123L  — VerifyAll, VerifierMap, ExtractMultiSignature, attach
├── event.go                      85L   — cloneEvent, ExtractSignature, HasSignature
├── hmac.go                       78L   — hmacSigner (unexported)
├── errors.go                     31L   — Sentinel errors
├── doc.go                        27L   — Package doc
├── benchmark_test.go             18L   — Ed25519 benchmarks
├── assertions_test.go            13L   — Type assertion tests
├── README.md                    128L   — Usage guide with VerifierMap examples
├── signing_test.go            1,028L   — ⚠️ Over 350L limit
└── multisig_test.go           1,338L   — ⚠️ Over 350L limit
```

---

## Change Log This Session

| File                          | Change                                                                                                     |
| ----------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `signing/multisig_extract.go` | Added `VerifierMap(signers ...*MultiSigner) map[Actor]Verifier` — convenience builder with nil panic guard |
| `signing/example_test.go`     | Added `ExampleVerifierMap`; updated `ExampleVerifyAll` to use `VerifierMap`                                |
| `signing/README.md`           | Fixed `map[string]` → `signing.VerifierMap()` in "Verify All" and "Require All" sections                   |
| `signing/multisig_test.go`    | Added `TestVerifierMap`, `TestVerifierMap_NilPanics`, `TestVerifierMap_Empty`                              |

**+113 lines, -18 lines across 4 files.**
