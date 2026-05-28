# Session 117 — Comprehensive Status Report

**Date:** 2026-05-28 11:20 UTC+2
**Scope:** Signing module audit, full codebase quality sweep
**Previous session:** 116 (signing refactoring complete)

---

## a) FULLY DONE ✅

### Signing Module
- `Signer`/`Verifier`/`SignerVerifier` interface split
- `Signature` type with `String()`, `MarshalJSON()`, `UnmarshalJSON()`, `Equal()`, `Bytes()`, `IsZero()`
- `Actor` branded type replacing raw `string`
- `NewMultiSigner` returns `(*MultiSigner, error)` with validation
- `MultiVerifyMiddlewareFor(actor, verifier)` convenience function
- `SignatureEntry.Validate()` method
- `RequireMultiSigMiddleware` nil-event guard
- Compile-time interface assertions (`assertions_test.go`)
- 3 `Example*` functions (`example_test.go`)
- `BenchmarkCanonicalPayload` (`benchmark_test.go`)
- `FuzzSignature_Roundtrip` fuzz test
- Canonical payload edge case tests (nil, empty, 1MB payload)
- Signing coverage: **93.6%**
- All production files under 250 lines (multisig.go at 281 — needs split)
- All tests pass, 0 lint issues

### Infrastructure
- Catalog golden file tests fixed (3 files updated)
- Integration `go.mod` fixed (added missing `saga` replace directive)
- 3 `nlreturn` lint issues fixed in catalog tests
- Signing module added to `docs/perfect-world-modules.d2`
- `example/user/main.go` already has signing demo (sign+verify middleware chain)

---

## b) PARTIALLY DONE 🔶

### `multisig.go` — 281 lines (limit is 250)
- Needs split into `multisig_types.go` (types + consts + MultiSignature methods) and `multisig.go` (MultiSigner logic)

### `SignatureEntry.Validate()` — exists but never called internally
- `MultiSigner.Sign()` and `VerifyAll` should validate entries but don't

### `NewMultiSigner` validation — partial
- Validates: empty actor, nil signer, nil clock
- Missing: nil verifier guard (Verify will panic), algorithm validation

---

## c) NOT STARTED ⬜

### Critical Bugs Found
1. **`NewMultiSigner` allows nil verifier** → `(*MultiSigner).Verify()` panics at `m.verifier.Verify()` (multisig.go:249). Ed25519 signers without `WithVerifier` option hit this.
2. **Corrupted signatures bypass verification** — `HasSignature`/`HasMultiSignature` return `false` for corrupt JSON, so `VerifyMiddleware`/`MultiVerifyMiddleware` pass tampered events through as "unsigned" instead of rejecting them.

### Dead Code
3. **`middleware/common.go`** — all 3 functions (`commandErrMiddleware`, `eventErrMiddleware`, `queryErrMiddleware`) are unused. Entire file (34 lines) is dead code.

### `go.mod` Issues
4. **`saga/go.mod`** — orphaned `memory => ../memory` replace (saga doesn't require memory)
5. **`example/user/go.mod`** — signing version is `v0.0.0-20260528054122-...` instead of `v1.6.0`

### Storage Module Broken (pre-existing)
6. **`storage/event_store_mock_test.go`** — 3 call sites with wrong arg count to `testEventWithAggID`
7. **`storage/event_store_test.go`** — calls `store.ReadAll` which doesn't exist on `*SQLEventStore` (it's on `Journal` interface)
8. **`projection/runner.go`** and **`storage/event_store.go`** — reference `event.Journal`/`event.SeekableJournal` which gopls can't resolve (likely a workspace resolution issue, tests pass via nix)

### Code Quality
9. **`multisig.go` `Verify`/`VerifyActor` duplication** — near-identical 25-line methods that could share a private helper
10. **No nil guard on middleware constructors** — `SignMiddleware(nil)`, `VerifyMiddleware(nil)`, `MultiSignMiddleware(nil)` all panic at runtime instead of returning an error
11. **`RequireSignatureMiddleware` double-extracts** signature (calls `HasSignature` then delegates to `VerifyMiddleware` which re-extracts)
12. **`RequireMultiSigMiddleware` double-extracts** multi-signature (extracts for `HasActor`, then `VerifyAll` re-extracts)

### Test File Bloat
13. **`signing/multisig_test.go`** — 1072 lines (worst offender)
14. **`signing/signing_test.go`** — 933 lines

### Minor
15. **`signing/benchmark_test.go`** — `b.N` → modernize to `b.Loop()`
16. **`signing/README.md`** badge URL has inconsistent casing

---

## d) TOTALLY FUCKED UP 💣

None in signing module. The code is solid architecturally. Issues are edge cases and missing guards, not design flaws.

**However**, the storage module has pre-existing broken tests (items 6-7 above) that are masked because gopls workspace resolution differs from `nix run .#test` which uses `GOWORK=off` per-module. These need fixing.

---

## e) WHAT WE SHOULD IMPROVE

### Architecture
- **Unexport signing structs**: `HMACSigner`, `Ed25519Signer`, `Ed25519Verifier`, `MultiSigner` should have unexported underlying types. Currently `HMACSigner{}` bypasses `NewHMAC` validation and panics on use. Only constructors should create instances.
- **Corrupted signature handling**: `HasSignature`/`HasMultiSignature` should distinguish "no signature" from "corrupt signature". Return an error or a tri-state. This is security-relevant.

### Type Model
- **`Signature` is `[]byte`** — consider making it an opaque struct with a private byte slice. This prevents zero-initialization and allows validation on construction.
- **`SignatureAlgorithm` is a bare string** — validate against known values in `NewMultiSigner`. Consider a registry if we add RSA/ECDSA later.
- **`Clock` pattern** (`func() time.Time`) works well but isn't documented in a central place. Other modules could benefit.

### Library Usage
- **`crypto/hmac`** and **`crypto/ed25519`** are stdlib — good, no third-party crypto deps.
- **Consider `golang.org/x/crypto`** for future algorithm support (argon2 for key derivation, bcrypt for key storage).

---

## f) TOP 25 NEXT ACTIONS (sorted by impact × effort)

| # | Impact | Effort | Action |
|---|--------|--------|--------|
| 1 | 🔴 HIGH | S | Fix `NewMultiSigner` nil verifier panic — validate verifier non-nil or add explicit error in `Verify()` |
| 2 | 🔴 HIGH | S | Fix corrupted signature bypass — make `HasSignature`/`HasMultiSignature` return tri-state or check extract error in middlewares |
| 3 | 🟡 MED | S | Delete dead `middleware/common.go` |
| 4 | 🟡 MED | S | Remove orphaned `memory` replace from `saga/go.mod` |
| 5 | 🟡 MED | S | Fix `example/user/go.mod` signing version |
| 6 | 🟡 MED | M | Split `multisig.go` into types + logic files (under 250 lines each) |
| 7 | 🟡 MED | M | Extract `verifyActorEntry` helper to deduplicate `Verify`/`VerifyActor` |
| 8 | 🟡 MED | M | Add nil guards to all middleware constructors |
| 9 | 🟡 MED | S | Call `SignatureEntry.Validate()` in `MultiSigner.Sign()` |
| 10 | 🟡 MED | S | Add `algorithm` validation in `NewMultiSigner` |
| 11 | 🟡 MED | M | Unexport signing struct types (`HMACSigner` → `hmacSigner`, etc.) |
| 12 | 🟢 LOW | S | Fix `signing/README.md` badge casing |
| 13 | 🟢 LOW | S | Modernize `b.Loop()` in `benchmark_test.go` |
| 14 | 🟢 LOW | M | Simplify `RequireSignatureMiddleware` to avoid double-extract |
| 15 | 🟢 LOW | M | Simplify `RequireMultiSigMiddleware` to avoid double-extract |
| 16 | 🟢 LOW | L | Split `signing/multisig_test.go` (1072 lines) into focused files |
| 17 | 🟢 LOW | L | Split `signing/signing_test.go` (933 lines) into focused files |
| 18 | 🟡 MED | L | Fix `storage/event_store_mock_test.go` wrong arg counts |
| 19 | 🟡 MED | L | Fix `storage/event_store_test.go` `ReadAll` calls |
| 20 | 🟢 LOW | M | Fix `catalog/eventcatalog/writer_frontmatter_test.go` unused field writes |
| 21 | 🟢 LOW | S | Fix `core/event/outbox.go:17` unnecessary type args |
| 22 | 🟢 LOW | M | Document `Clock` pattern centrally (e.g., in `core/event` or a design doc) |
| 23 | 🟢 LOW | L | Consider making `Signature` an opaque struct instead of `[]byte` |
| 24 | 🟢 LOW | L | Fix `docs/adr/README.md` TODO about `init()` registration |
| 25 | 🟢 LOW | L | Consider Algorithm registry for extensibility |

---

## g) TOP #1 UNRESOLVED QUESTION

**How should corrupted signatures be handled?**

Currently `HasSignature`/`HasMultiSignature` return `false` for corrupt JSON, which causes verify middlewares to pass events through as unsigned. This is a security concern: an attacker who corrupts a signature byte can bypass verification entirely.

Two approaches:

1. **Tri-state return**: `HasSignature` → `SignatureStatus` enum (`Present`, `Absent`, `Corrupt`). Middlewares reject on `Corrupt`, pass through on `Absent`, verify on `Present`.
2. **Error-based**: Change `HasSignature` → `CheckSignature` that returns `(bool, error)`. Middlewares reject on error. Simpler but breaks the boolean API.

Option 1 is more explicit but adds an enum type. Option 2 is simpler but less clear. Which do you prefer?

---

## Module Health Summary

| Module | Tests | Lint | Coverage | Notes |
|--------|-------|------|----------|-------|
| core | ✅ | ✅ | 84-100% | gopls workspace issues with Journal type |
| memory | ✅ | ⚠️ dupl | ~95% | 1 dupl + 2 nolintlint in tests |
| catalog | ✅ | ✅ | ~90% | Golden files updated |
| middleware | ✅ | ✅ | ~90% | `common.go` is dead code |
| signing | ✅ | ✅ | **93.6%** | 2 latent panics, 281-line file |
| storage | ✅ | ✅ | ~85% | 3 mock_test.go broken call sites, 3 ReadAll calls |
| projection | ✅ | ✅ | ~90% | gopls Journal resolution issue |
| saga | ✅ | ✅ | ~85% | Orphaned replace directive |
| integration | ✅ | ✅ | N/A | Fixed go.mod |
| testhelpers | ✅ | ✅ | ~95% | Clean |
| watermill | ✅ | ✅ | N/A | Clean |
| cmd/cqrs-gen | ✅ | ✅ | N/A | Clean |
