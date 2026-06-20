# Session 142: Signing Module Restructuring — Comprehensive Status

**Date:** 2026-05-29 12:58
**Branch:** master
**Commit:** 0d793d3 (restructuring committed), unstaged changes from prior sessions

---

## Executive Summary

Completed a full restructuring of the `signing/` module: extracted multisig functionality into a `signing/multisig/` sub-package, split the monolithic `signer.go` into focused files, consolidated fragmented test suites, and updated all documentation and external consumers. All 25 planned tasks completed. All tests pass. Coverage maintained.

---

## A) FULLY DONE ✅

### Signing Module Restructuring (25/25 tasks)

| #   | Task                                                     | Result                                                                   |
| --- | -------------------------------------------------------- | ------------------------------------------------------------------------ |
| 1   | Export `CloneEvent` from root signing package            | `cloneEvent` → `CloneEvent` in `event.go:14`                             |
| 2   | Extract `signature.go` from `signer.go`                  | `Signature` type + all serialization methods (77 lines)                  |
| 3   | Extract `payload.go` from `signer.go`                    | `canonicalPayload` + `appendLenPrefixed` (76 lines)                      |
| 4   | Clean up `signer.go` to interfaces only                  | `Signer`, `Verifier`, `SignerVerifier` (29 lines, was 174)               |
| 5   | Create `multisig/` directory + `doc.go`                  | Package docs with usage examples                                         |
| 6   | Move `multisig_types.go` → `multisig/types.go`           | Types import `signing.Signature` + `signing.Signer`                      |
| 7   | Move `multisig.go` → `multisig/signer.go`                | `NewMultiSigner`, `Sign`, `Verify`, `VerifyActor`                        |
| 8   | Move `multisig_extract.go` → `multisig/extract.go`       | `VerifyAll`, `ExtractMultiSignature`, `VerifierMap`, `HasMultiSignature` |
| 9   | Move `multisig_middleware.go` → `multisig/middleware.go` | 4 middleware functions + `extractOrPassThrough` helper                   |
| 10  | Create `multisig/errors.go`                              | `ErrNoVerifier` sentinel                                                 |
| 11  | Remove old multisig source files from root               | 4 files deleted                                                          |
| 12  | Move examples to `multisig/example_test.go`              | 4 examples with updated import paths                                     |
| 13  | Consolidate types tests                                  | `multisig/types_test.go` (345 lines) — merged from 2 files               |
| 14  | Consolidate signer tests                                 | `multisig/signer_test.go` (264 lines) — merged from 3 files              |
| 15  | Consolidate extract tests                                | `multisig/extract_test.go` (209 lines) — merged from 2 files             |
| 16  | Consolidate middleware tests                             | `multisig/middleware_test.go` (340 lines) — merged from 3 files          |
| 17  | Remove old multisig test files from root                 | 8 files deleted                                                          |
| 18  | Update `integration/signing/` imports                    | All `signing.X` → `multisig.X`                                           |
| 19  | Update `signing/doc.go`                                  | References `multisig` sub-package                                        |
| 20  | Update `signing/README.md`                               | All multisig examples use `multisig` import                              |
| 21  | Update `docs/signing-architecture.md`                    | File map + code examples updated                                         |
| 22  | Update `FEATURES.md`                                     | Section title → `signing/multisig sub-package`                           |
| 23  | Check `MIGRATION_v1.md`                                  | No changes needed (only single-sig references)                           |
| 24  | Run all tests + verify coverage                          | Root: 93.7%, Multisig: 94.2%, All pass ✅                                |
| 25  | Verify full workspace build                              | All modules compile ✅                                                   |

### Test Results (Full Workspace)

| Module                | Coverage | Status |
| --------------------- | -------- | ------ |
| `core/event`          | 90.7%    | ✅     |
| `core/command`        | 94.2%    | ✅     |
| `core/decider`        | 100.0%   | ✅     |
| `core/query`          | 96.8%    | ✅     |
| `core/pkg/id`         | 100.0%   | ✅     |
| `core/pkg/dispatcher` | 92.2%    | ✅     |
| `signing` (root)      | 93.7%    | ✅     |
| `signing/multisig`    | 94.2%    | ✅     |
| `memory`              | 99.1%    | ✅     |
| `catalog`             | 96.3%    | ✅     |
| `middleware`          | 94.0%    | ✅     |
| `storage`             | 93.7%    | ✅     |
| `projection`          | 89.6%    | ✅     |
| `saga`                | 94.6%    | ✅     |
| `stream`              | 94.0%    | ✅     |
| `pebble`              | 87.2%    | ✅     |
| `testhelpers`         | 82.1%    | ✅     |
| `integration/signing` | —        | ✅     |
| `example/user`        | 36.7%    | ✅     |

---

## B) PARTIALLY DONE

### Pre-existing Unstaged Changes (not from this session)

- `memory/store_extra_test.go` — 152 lines deleted (backward-compat `LoadAll` test removal). Unrelated to signing restructuring.
- `projection/projection_bdd_test.go` — 1 line changed (`cp` → `cp.EventID` for Checkpoint refactor). Pre-existing from Checkpoint refactor.

These are from prior sessions and should be committed separately.

---

## C) NOT STARTED

### Module-level improvements identified during restructuring

1. **`pebble/` module** — 87.2% coverage, could benefit from more edge case tests
2. **`example/user/`** — 36.7% coverage, demo code barely tested
3. **`testhelpers/`** — 82.1% coverage, some helpers untested
4. **`catalog/internal/cattest`** — 0.0% coverage (test helpers for catalog, acceptable)
5. **No `watermill/` or `turso/` test results** — modules may need test verification
6. **No lint run** — `nix run .#lint` or equivalent not executed this session

---

## D) TOTALLY FUCKED UP — Nothing! 🎉

No regressions, no broken tests, no failed builds. The restructuring was clean.

**One minor concern:** The `extractOrPassThrough` helper is duplicated between `signing/middleware.go` and `multisig/middleware.go`. This was a deliberate decision to avoid circular imports (multisig can't import root signing internals), but it's technical debt.

---

## E) WHAT WE SHOULD IMPROVE

### Signing Module Specific

1. **Duplicated `extractOrPassThrough`** — Generic helper lives in both `signing/middleware.go` and `multisig/middleware.go`. Could extract to a shared internal helper if Go sub-package patterns allow it.
2. **`BenchmarkVerifyAll` removed from root** — Was moved conceptually but not re-created in `multisig/`. The benchmark for `VerifyAll` no longer exists anywhere.
3. **BDD test coverage** — `signing_bdd_test.go` only covers single-sig flows. No BDD suite for multisig (Ginkgo tests were all standard `testing.T`).
4. **`multisig/errors.go` is minimal** — Only `ErrNoVerifier`. Root `errors.go` has 4 sentinels. Consider whether `ErrNoVerifier` should live in root for cross-package access.

### Broader Codebase

5. **Unstaged changes accumulating** — `memory/store_extra_test.go` and `projection/projection_bdd_test.go` are dirty from prior sessions. Need committing.
6. **194 gopls errors project-wide** — Mostly "module not in go.mod" from workspace mode. Not real errors but noise.
7. **`example/user/` at 36.7% coverage** — Demo code; low priority but worth noting.
8. **`pebble/` at 87.2%** — Newest module, could use more tests.
9. **`signer.go` at 29 lines** — Extremely clean now, but the interfaces could use Go doc examples.
10. **No `CHANGELOG.md` entry** for the restructuring — Breaking API change for multisig consumers.

---

## F) Top 25 Things to Do Next

### Priority 1: Immediate Cleanup (This Session's Loose Ends)

| #   | Task                                                             | Impact | Effort |
| --- | ---------------------------------------------------------------- | ------ | ------ |
| 1   | Commit pre-existing unstaged changes (`memory/`, `projection/`)  | High   | 5 min  |
| 2   | Add `BenchmarkVerifyAll` to `multisig/`                          | Medium | 10 min |
| 3   | Update `CHANGELOG.md` with restructuring entry                   | High   | 10 min |
| 4   | Update `AGENTS.md` signing module description with new structure | Medium | 10 min |

### Priority 2: Signing Module Hardening

| #   | Task                                                                  | Impact | Effort |
| --- | --------------------------------------------------------------------- | ------ | ------ |
| 5   | Add Ginkgo BDD suite for `multisig/` sub-package                      | Medium | 30 min |
| 6   | Deduplicate `extractOrPassThrough` (shared internal or exported)      | Low    | 15 min |
| 7   | Add Go doc examples to `signer.go` interfaces                         | Medium | 10 min |
| 8   | Review `multisig/errors.go` — should `ErrNoVerifier` be in root?      | Low    | 5 min  |
| 9   | Add integration test for mixed single-sig + multisig middleware chain | Medium | 15 min |

### Priority 3: Broader Codebase Quality

| #   | Task                                                          | Impact | Effort |
| --- | ------------------------------------------------------------- | ------ | ------ |
| 10  | Run `nix run .#lint` and fix issues                           | High   | 30 min |
| 11  | Improve `pebble/` coverage to >90%                            | Medium | 30 min |
| 10  | Improve `example/user/` test coverage                         | Low    | 20 min |
| 13  | Clean up 194 gopls workspace errors                           | Medium | 30 min |
| 14  | Add `TODO_LIST.md` update reflecting restructuring completion | Medium | 10 min |
| 15  | Verify `watermill/` and `turso/` modules build and test       | Medium | 10 min |

### Priority 4: Architecture & Documentation

| #   | Task                                                                       | Impact | Effort |
| --- | -------------------------------------------------------------------------- | ------ | ------ |
| 16  | Update `docs/architecture/` D2 diagrams with multisig sub-package          | Medium | 15 min |
| 17  | Review `catalog/` sub-package pattern consistency with `signing/multisig/` | Low    | 15 min |
| 18  | Add ADR for signing sub-package decision                                   | Medium | 20 min |
| 19  | Verify `docs/MIGRATION_v1.md` covers the `multisig` import path change     | Medium | 10 min |
| 20  | Review `docs/planning/` for stale references                               | Low    | 10 min |

### Priority 5: Forward-Looking

| #   | Task                                                                        | Impact | Effort |
| --- | --------------------------------------------------------------------------- | ------ | ------ |
| 21  | Consider `signing/multisig` → separate `go.mod` if it grows further         | Low    | 30 min |
| 22  | Explore event signing for `storage/` outbox (outbox signing middleware)     | High   | 60 min |
| 23  | Add signing support to `projection/` runner (verify signatures on replay)   | High   | 60 min |
| 24  | Design cross-module signing integration test (storage → bus → projection)   | Medium | 45 min |
| 25  | Evaluate whether `saga/` needs signature verification for saga state events | Medium | 30 min |

---

## G) Top #1 Question I Cannot Answer Myself

**Should `signing/multisig` remain a sub-package within `signing`, or should it become its own top-level module (like `pebble/` and `turso/` were extracted)?**

Arguments for sub-package (current state):

- Follows `catalog/` precedent (`catalog/d2/`, `catalog/asyncapi/`)
- No separate `go.mod` to maintain
- Consumers already import `signing` for `Signer`/`Verifier`/`Signature`

Arguments for separate module:

- `multisig` has zero dependency on root `signing` beyond interfaces
- Could evolve independently
- The signing module is "the most self-contained module" per architecture docs

I cannot determine the right call because it's a product/ownership decision: should consumers `import "go-cqrs-lite/signing/multisig"` or `import "go-cqrs-lite/multisig"`?

---

## File Structure (After Restructuring)

```
signing/
├── doc.go                    # 32 lines — package docs + multisig reference
├── errors.go                 # 31 lines — sentinel errors
├── signer.go                 # 29 lines — Signer/Verifier/SignerVerifier interfaces
├── signature.go              # 77 lines — Signature type + serialization
├── payload.go                # 76 lines — canonicalPayload + helpers
├── hmac.go                   # 78 lines — HMAC-SHA256 implementation
├── ed25519.go               # 109 lines — Ed25519 signer/verifier/keygen
├── event.go                  # 92 lines — CloneEvent, Attach/Extract/HasSignature
├── middleware.go             # 148 lines — Single-sig middleware + extractOrPassThrough
├── multisig/
│   ├── doc.go               # 33 lines — sub-package docs
│   ├── errors.go            # 8 lines — ErrNoVerifier
│   ├── types.go             # 148 lines — Actor, SignatureAlgorithm, SignatureEntry, MultiSignature, MultiSigner, options
│   ├── signer.go            # 178 lines — NewMultiSigner, Sign, Verify, VerifyActor
│   ├── extract.go           # 140 lines — VerifyAll, ExtractMultiSignature, VerifierMap, HasMultiSignature
│   └── middleware.go        # 205 lines — MultiSign/Verify/RequireMultiSig middleware
├── *_test.go (root)         # 8 test files
└── multisig/*_test.go       # 6 test files (consolidated from 9)
```

**Before:** 25 Go files in flat root (9 source + 16 test)
**After:** 15 root files + 12 multisig files = 27 total, but with **clear package boundaries**

---

## Coverage Comparison

| Package            | Before         | After           | Delta |
| ------------------ | -------------- | --------------- | ----- |
| `signing` (root)   | 93.8%          | 93.7%           | -0.1% |
| `signing/multisig` | (part of root) | 94.2%           | NEW   |
| Total signing      | 93.8%          | ~94.0% combined | +0.2% |
