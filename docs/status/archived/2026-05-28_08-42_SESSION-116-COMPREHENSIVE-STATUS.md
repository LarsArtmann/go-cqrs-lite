# Session 116 Comprehensive Status Report

**Date:** 2026-05-28 08:42 UTC+2
**Branch:** `master`
**Total Commits:** 1,264
**Go Version:** 1.26.3
**Modules:** 14 (go.work)
**Go Files:** 389 (20,967 production LOC + 37,434 test LOC)

---

## a) FULLY DONE ✅

### Signing Module (All Original Audit Issues Resolved)

The signing module was the primary focus of Sessions 115–116. Every issue identified in the critical audit has been addressed and pushed to `master`.

| Commit    | Description                                                                                |
| --------- | ------------------------------------------------------------------------------------------ |
| `4a7fc2d` | Split `Signer` into `Signer` + `Verifier` + `SignerVerifier` interfaces                    |
| `989a151` | Add `Signature.String()`, `MarshalJSON()`, `UnmarshalJSON()` with URL-safe base64          |
| `c2a4e0f` | Extract `VerifyAll` to standalone function + add `WithClock` for deterministic `SignedAt`  |
| `360eeb2` | Extract `cloneEvent` helper to eliminate duplicate `event.NewEvent` reconstruction         |
| `ea5e989` | Add `Signature.Equal()` for constant-time comparison (uses `hmac.Equal`)                   |
| `93f2827` | Remove dead `ErrAlgorithmMismatch` sentinel error                                          |
| `043be95` | Fix README stale `Signer` reference → `Verifier` in `VerifyAll` example                    |
| `d4a55e2` | **🔒 Security fix:** `RequireMultiSigMiddleware` now cryptographically verifies signatures |

**Current signing metrics:**

- **Tests:** All pass (race-clean)
- **Coverage:** 88.9% (up from initial 83.4%)
- **Lint:** 0 issues
- **File sizes:** All production files ≤350 lines (241 `multisig.go`, 152 `signer.go`)

### All Other Modules (Passing)

| Module                | Tests | Lint | Coverage |
| --------------------- | ----- | ---- | -------- |
| `core/command`        | ✅    | ✅   | 94.3%    |
| `core/decider`        | ✅    | ✅   | 91.1%    |
| `core/event`          | ✅    | ✅   | 92.4%    |
| `core/pkg/dispatcher` | ✅    | ✅   | 100.0%   |
| `core/pkg/id`         | ✅    | ✅   | 100.0%   |
| `core/query`          | ✅    | ✅   | —        |
| `memory`              | ✅    | ✅   | 99.6%    |
| `catalog` (main)      | ✅    | ✅   | 96.3%    |
| `catalog/d2`          | ✅    | ✅   | 95.0%    |
| `catalog/docserver`   | ✅    | ✅   | 90.1%    |
| `catalog/openapi`     | ✅    | ✅   | 94.4%    |
| `middleware`          | ✅    | ✅   | 93.7%    |
| `testhelpers`         | ✅    | ✅   | 91.2%    |
| `storage`             | ✅    | ✅   | 90.1%    |
| `projection`          | ✅    | ✅   | 96.0%    |
| `saga`                | ✅    | ✅   | 93.4%    |
| `watermill`           | ✅    | ✅   | 94.4%    |
| `signing`             | ✅    | ✅   | 88.9%    |

### Docs (Fresh)

- `AGENTS.md` — Updated with signing patterns, upcasting patterns, 135 lines
- `TODO_LIST.md` — Reconciled, 271 lines, 183 done / 20 open / 15 blocked / 22 future
- `FEATURES.md` — Current at 496 lines

---

## b) PARTIALLY DONE ⚠️

### `catalog/asyncapi` — 3 Golden File Failures

```
FAIL catalog/asyncapi  — AsyncAPI YAML mismatch
FAIL catalog/eventcatalog — eventcatalog-config.js mismatch
FAIL catalog/eventcatalog — package.json mismatch
```

These are **golden file mismatches**, not code bugs. The test outputs differ from checked-in golden files. They have been failing consistently and are being tracked across sessions. Root cause: the golden files need to be regenerated with the `-update` flag. This is a known maintenance burden.

**Impact:** Medium — these are documentation/test artifacts, not runtime failures.

### `integration` Module — `go mod download` Broken

```
go: github.com/larsartmann/go-cqrs-lite/saga@v1.6.0:
    reading .../saga/go.mod at revision saga/v1.6.0: unknown revision
```

The `integration` module cannot resolve remote versions because the project relies on `replace` directives in `go.work`. Running `GOWORK=off` in the integration directory fails because the tags are not pushed to remote. This is expected behavior for a local-only workspace but it means the integration module cannot be tested in isolation without `go.work`.

**Impact:** Medium — the CI uses `go.work`, so this only affects local isolation testing.

### Signing Module — `testhelpers` in `go.mod`

`signing/go.mod` lists `testhelpers` as a direct dependency. Go modules include test dependencies in the main `require` block — there is no clean `test` qualifier. Consumers importing `signing` will transitively pull in `testhelpers` and its dependencies (ulid, go-branded-id). This is a cosmetic dependency graph issue but not functionally harmful.

**Impact:** Low — cosmetic. No clean Go mechanism to exclude.

---

## c) NOT STARTED 📋

From the TODO list, the following remain genuinely open (not blocked/future):

- `[v2]` Fix query.Handler returns `any` → generic `TypedHandler[T]` returning `(T, error)`
- `[v2]` Add global `TransactionID` branded type for cross-aggregate consistency
- `[v2]` `io.Closer` removal from core interfaces
- `[BLOCKED]` Publish `go-composable-business-types` as Go Module
- `[BLOCKED]` Modularize ActaFlow
- `[FUTURE]` Add catalog diff/breaking-change detection tool
- `[FUTURE]` Add high-level test utilities — `AggregateTester`, `ProjectionTester`, `BusTester` fluent API

---

## d) TOTALLY FUCKED UP! 🔴

### `catalog/asyncapi` + `catalog/eventcatalog` — Golden Files Perpetually Out of Date

These tests fail on **every** full test run (`nix run .#test`). They are not "bugs" — the golden files drift because the catalog output format changes during feature work. The current process requires manual `-update` flag runs to refresh them.

**Why this is fucked up:**

- It creates noise on every CI run / `nix run .#test`
- The failures are indistinguishable from real bugs at a glance
- The team has been ignoring these failures for multiple sessions
- `golangci-lint` on catalog has 3 `nlreturn` issues in `eventcatalog/exporter_test.go` that should be auto-fixable

**Fix needed:** Either (a) add `//go:generate` scripts to auto-update golden files, or (b) make the golden tests accept a date-stamped tolerance, or (c) separate golden tests into their own CI job.

### `integration` Module — Cannot Test in Isolation

The module is broken for standalone use (`GOWORK=off`) because it depends on remote tags that don't exist. This makes it impossible to verify that the module graph is valid for external consumers.

**Why this is fucked up:**

- The module graph is technically un-testable for publishing
- `go.work` masks the problem locally
- CI passes because it uses `go.work`, but external consumers won't have `go.work`
- The `replace` directives are a known blocker until v1.0.0 tags are pushed

---

## e) WHAT WE SHOULD IMPROVE! 💡

### 1. Signing Module: Coverage to 90%+

Current: 88.9%. Missing paths:

- `VerifyActor` (cross-actor verification method)
- Backward-compatible standard base64 in `UnmarshalJSON`
- `Signature.Equal` with mismatched lengths
- `cloneEvent` error path (nil event already guarded upstream)

### 2. Signing Module: `MultiVerifyMiddleware` API

Currently takes `*MultiSigner` but only needs actor name + `Verifier`. Consider:

```go
// Current:
func MultiVerifyMiddleware(signer *MultiSigner) event.Middleware
// Could be cleaner:
func MultiVerifyMiddleware(actor string, verifier Verifier) event.Middleware
```

This would decouple the middleware from the `MultiSigner` type and allow verification without needing a full `MultiSigner` instance (e.g., verifying someone else's signature).

### 3. Golden File Tests: Automated Refresh

Add a `justfile`/`flake.nix` target that runs:

```bash
cd catalog/asyncapi && go test ./... -update
cd catalog/eventcatalog && go test ./... -update
```

And run it as a pre-commit step or weekly cron job.

### 4. `integration` Module: Fix Standalone Testability

Either:

- Add `go mod download` with workspace context to the test script
- Or add a CI job that runs integration tests WITH `go.work`
- Or document that integration is workspace-only and move it to a separate test job

### 5. `catalog/eventcatalog/exporter_test.go` Lint

3 `nlreturn` issues remain. Run `gofumpt -w` or add blank lines.

### 6. `Signature` Type: Consider Value Semantics

`Signature` is `[]byte` (reference type). The `Equal` method uses `hmac.Equal` which is timing-safe for bytes, but since `Signature` is a slice, two `Signature` values with the same content are not `==` comparable. This is correct behavior for crypto but worth documenting.

### 7. `canonicalPayload`: SchemaVersion Not Part of First Version

The `canonicalPayload` function includes `SchemaVersion` in the signed content. This means events with the same payload but different schema versions will have different signatures. This is correct for security but may confuse consumers who expect signatures to be stable across schema migrations. Document this explicitly.

### 8. Module Count Discrepancy

`go.work` lists 14 modules but some directories (like `example/user/`) have their own `go.mod` files. The module count in AGENTS.md says 14 but should explicitly list all modules with their paths.

---

## f) Top #25 Things We Should Get Done Next! 🚀

Sorted by impact × effort:

1. **[HIGH]** Fix catalog golden file tests — auto-update or separate CI job (blocks clean CI)
2. **[HIGH]** Fix `integration` module standalone testability (blocks publishing confidence)
3. **[HIGH]** Add `Signature.Equal` test coverage and edge cases
4. **[MEDIUM]** Add `MultiVerifyMiddleware(actor, Verifier)` overload for cleaner API
5. **[MEDIUM]** Fix 3 `nlreturn` lint issues in `catalog/eventcatalog/exporter_test.go`
6. **[MEDIUM]** Add signing module to full workspace `nix run .#test` coverage
7. **[MEDIUM]** Document `canonicalPayload` schema version inclusion rationale
8. **[MEDIUM]** Add signing examples to `example/user` (basic signing/verify demo)
9. **[MEDIUM]** Consider removing `testhelpers` from signing `go.mod` via `_test.go` only usage audit
10. **[MEDIUM]** Add `SignerVerifier` compile assertion tests for `HMACSigner`
11. **[MEDIUM]** Add `MultiSigner` option validation (empty actor name, nil clock, etc.)
12. **[MEDIUM]** Document `Clock` pattern more prominently (reusable across modules)
13. **[LOW]** Consider `Signature` branded ID pattern like `id.Of[SignatureMarker]`
14. **[LOW]** Add `MultiSignature.Entries` nil-safety guards in methods
15. **[LOW]** Add `Event` metadata key collision detection (reserved keys: `event.signature`, `event.multisig`)
16. **[LOW]** Add benchmark for `canonicalPayload` (perf-critical for high-volume events)
17. **[LOW]** Consider `sync.Pool` for `canonicalPayload` buffer reuse
18. **[LOW]** Add `signing` module to `docs/planning/` architecture diagrams
19. **[LOW]** Add property-based tests (fuzz) for `Signature` marshal/unmarshal roundtrip
20. **[LOW]** Verify `Event` payload hash covers edge cases (empty, nil, very large)
21. **[LOW]** Add `SignatureEntry` validation (non-empty actor, known algorithm)
22. **[LOW]** Consider `Algorithm` registry for extensibility (RSA, ECDSA, etc.)
23. **[LOW]** Add `go doc` examples for `NewMultiSigner`, `VerifyAll`
24. **[LOW]** Add middleware composition example (Sign → Publish → Verify → Handle)
25. **[LOW]** Review `Replace` directives across all `go.mod` files for v1.0.0 readiness

---

## g) Top #1 Question I Cannot Figure Out Myself ❓

**How do we handle the `replace` directive problem for v1.0.0 publishing?**

The project has `replace` directives in every module's `go.mod` (e.g., `signing` replaces `core` with `../core`). This works perfectly with `go.work` for local development. But for publishing:

1. When we push v1.0.0 tags to GitHub, external consumers won't have the `replace` directives
2. The modules must resolve each other via the Go proxy
3. But the current version tags (`v1.6.0`) are synthetic — they don't exist on GitHub
4. The `integration` module cannot even `go mod download` because it references non-existent remote tags

The question is: **What is the exact release sequence?**

- Do we push all tags simultaneously? (risk of partial failure)
- Do we remove `replace` directives first, commit, then push? (breaks local dev)
- Do we use a release script that swaps `replace` → version, pushes, then reverts?
- Do we need `go work sync` before publishing?
- What about modules that depend on each other cyclically (or near-cyclically) during the tag push?

This is a real blocker. The current state works for local development but the path to publishing is unclear. A clear release runbook or script needs to be created before v1.0.0.

---

## Appendix: Signing Module File Inventory

| File                     | Lines     | Type | Purpose                                                                     |
| ------------------------ | --------- | ---- | --------------------------------------------------------------------------- |
| `doc.go`                 | 27        | Doc  | Package documentation                                                       |
| `errors.go`              | 31        | Code | 4 sentinel errors (was 5)                                                   |
| `hmac.go`                | 73        | Code | `HMACSigner` (Sign+Verify)                                                  |
| `ed25519.go`             | 105       | Code | `Ed25519Signer` (Sign), `Ed25519Verifier` (Verify)                          |
| `event.go`               | 85        | Code | `AttachSignature`, `ExtractSignature`, `cloneEvent`                         |
| `middleware.go`          | 87        | Code | `SignMiddleware`, `VerifyMiddleware`, `RequireSignatureMiddleware`          |
| `multisig_middleware.go` | 96        | Code | `MultiSignMiddleware`, `MultiVerifyMiddleware`, `RequireMultiSigMiddleware` |
| `multisig_extract.go`    | 103       | Code | `VerifyAll`, `ExtractMultiSignature`, `attachMultiSignature`, `removeActor` |
| `multisig.go`            | 241       | Code | Types, `MultiSigner`, options, methods                                      |
| `signer.go`              | 152       | Code | `Signer`, `Verifier`, `SignerVerifier`, `Signature`, `canonicalPayload`     |
| `signing_test.go`        | 813       | Test | Single-sig tests                                                            |
| `multisig_test.go`       | 707       | Test | Multi-sig tests                                                             |
| **Total**                | **2,510** |      | **13 files**                                                                |
