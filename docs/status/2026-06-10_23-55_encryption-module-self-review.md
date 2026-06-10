# Encryption Module — Status Report

**Date:** 2026-06-10 23:55
**Module:** `encryption/` (new)
**Author:** Crush (AI)

---

## Executive Summary

The `encryption/` module was built as a sibling to `signing/`, providing AES-256-GCM event payload encryption with middleware for automatic encrypt-on-publish / decrypt-on-handle. All 27 tests pass (14 BDD + 11 unit + 2 examples). Benchmarks: ~700ns encrypt, ~500ns decrypt for 1KB payloads.

**However**, the self-review reveals significant issues: ~80% code duplication with `signing/`, a redundant `slices.Clone` in the hot path, stale go.mod dependencies, missing README, missing doc.go examples, no integration test, and an architecture that could be cleaner with shared abstractions.

---

## A) FULLY DONE

| Item | Status | Notes |
|------|--------|-------|
| Module scaffolding | Done | `go.mod`, `go.work`, builds cleanly |
| Core interfaces | Done | `Encrypter`, `Decrypter`, `EncrypterDecrypter` |
| AES-256-GCM implementation | Done | `NewAES256GCM`, stdlib-only, correct nonce handling |
| Ciphertext type | Done | `Ciphertext []byte` with Nonce/Data/Equal/JSON serialization |
| Event helpers | Done | `AttachEncryption`, `ExtractCiphertext`, `HasEncryption` |
| Middleware | Done | `EncryptMiddleware` (publish), `DecryptMiddleware` (subscribe) |
| Sentinel errors | Done | 4 classified errors via go-error-family |
| BDD tests | Done | 14 specs via Ginkgo/Gomega, all pass |
| Unit tests | Done | 11 table-driven tests, all pass |
| Example tests | Done | 2 runnable examples, all pass |
| Benchmarks | Done | 3 benchmarks (encrypt/decrypt/round-trip) |
| Package doc | Done | `doc.go` with usage examples |

## B) PARTIALLY DONE

| Item | Status | What's missing |
|------|--------|----------------|
| Deduplication with signing | Partial | Both modules share ~80% structure but have separate implementations. No shared extraction between them. |
| go.mod hygiene | Partial | Stale indirect deps (spew, difflib) and unused snapshot replace directive. Needs `go mod tidy`. |
| Architecture documentation | Partial | No README.md in encryption/ (signing has one) |

## C) NOT STARTED

| Item | Impact | Effort |
|------|--------|--------|
| Extract shared metadata helper to `event/` or shared internal | High | Medium |
| Field-level encryption (`encryption/fieldlevel/` sub-package) | High | Large |
| XChaCha20-Poly1305 alternative implementation | Medium | Small |
| Integration test with signing (sign + encrypt round-trip) | Medium | Small |
| README.md for encryption/ | Medium | Small |
| Golden test for ciphertext serialization | Low | Small |
| Key rotation helper / envelope encryption | Medium | Medium |
| fuzz test for Decrypt | Medium | Small |
| Doc.go examples for middleware pipeline composition | Low | Small |

## D) TOTALLY FUCKED UP

| Item | Severity | Detail |
|------|----------|--------|
| Redundant `slices.Clone(plaintext)` in `aesgcm.go:85` | Minor | `aead.Open(nil, ...)` already returns a fresh allocation. The clone is wasted memory + CPU. |
| `CloneEvent` exported but only used internally | Minor | Should be unexported, or better: extracted to shared package |
| `Ciphertext.Equal()` is NOT constant-time | Security | Uses byte-by-byte loop. For ciphertext comparison this is acceptable (not a secret), but the signing module uses `hmac.Equal` (constant-time) for `Signature.Equal`. Inconsistent approach could confuse consumers about whether `Ciphertext` values are sensitive. |
| Decrypted event retains vestigial `"event.encrypted": ""` metadata | Smell | `WithCustom(key, "")` sets empty string, doesn't delete key. Not a bug (`HasEncryption` correctly returns false), but it's unclean — the decrypted event carries a ghost entry. |
| `gcmNonceSize` const duplicated between `ciphertext.go` and `aesgcm.go` | Minor | Both files define `gcmNonceSize = 12` independently |

## E) WHAT WE SHOULD IMPROVE

### Critical Path — Deduplication

The biggest issue is code duplication with `signing/`. Both modules implement the same pattern:

1. **Opaquely-typed `[]byte` values** — `Signature` and `Ciphertext` share identical `IsZero`, `Bytes`, `String`, `MarshalJSON`, `UnmarshalJSON` methods
2. **Metadata envelope** — Both use `event.MetadataKey` to attach/extract base64-encoded blobs
3. **Middleware pattern** — Both have identical rejecting middleware helpers and near-identical publish/subscribe middleware

**Shared code should live in `event/` as unexported helpers** (both signing and encryption already depend on event). The typed `[]byte` value pattern (base64 JSON serialization, defensive clone, etc.) could become a shared internal type.

### Architecture Improvements

1. **`event.CloneEvent(evt, key, value)` should exist in `event/`** — Both signing and encryption need to clone an event while changing one metadata field. This is a generic operation that belongs in the event package.

2. **`event.AttachBlob(evt, key, blob) → Event`** — Both signing and encryption do: serialize blob → base64 → store in metadata → return cloned event. This is a single function.

3. **`event.ExtractBlob(evt, key) → ([]byte, error)`** — Both do: check metadata → base64 decode → return bytes. Single function.

4. **Rejecting middleware helpers** — Could live in `event/` as unexported helpers (or as a public utility since they're useful for any middleware).

### Type Model Improvements

The `Ciphertext` type has a `Nonce()` / `Data()` accessor pattern that's AES-GCM specific. If we add XChaCha20-Poly1305 (24-byte nonce), the split is different. Consider:

- `Ciphertext` stays as opaque `[]byte` (nonce+data concatenated)
- `Nonce()` / `Data()` are moved to an AES-GCM-specific helper, not on the generic type
- Or: `Ciphertext` only has `Bytes()` and `IsZero()` — nonce extraction is implementation detail

### Established Libraries to Consider

- **`golang.org/x/crypto/chacha20poly1305`** — XChaCha20-Poly1305 for the 24-byte nonce variant. Zero risk, this is `golang.org/x/crypto` (quasi-stdlib).
- **`filippo.io/age`** — Age encryption format. Well-designed, but too heavy for our use case (we need raw AES-GCM, not a full file encryption format).
- **`google/tink/go`** — Google's crypto library. Way too heavy for this module. Good for reference patterns though.

---

## F) Top 25 Things to Do Next

Sorted by **impact × ease** (Pareto):

### Tier 1: Quick Wins (High Impact, Low Effort)

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 1 | Remove redundant `slices.Clone(plaintext)` in `aesgcm.go:85` | Perf | 1 min | Free performance win in hot path |
| 2 | `go mod tidy` to remove stale deps | Hygiene | 1 min | Removes spew, difflib, snapshot |
| 3 | Unify `gcmNonceSize` const to single location | Cleanliness | 2 min | Duplicated in ciphertext.go and aesgcm.go |
| 4 | Unexport `CloneEvent` — it's internal | Cleanliness | 2 min | Not used outside package |
| 5 | Add `encryption/README.md` | Docs | 10 min | Every module has one |
| 6 | Add fuzz test for Decrypt | Robustness | 10 min | Standard practice for crypto code |
| 7 | Fix vestigial `"event.encrypted": ""` on decrypted events | Cleanliness | 15 min | Delete key from Custom map instead of setting empty string |

### Tier 2: Architecture Cleanup (High Impact, Medium Effort)

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 8 | Extract `CloneEvent` to `event/` as shared helper | Dedup | 30 min | Signing + encryption both need it |
| 9 | Extract `AttachBlob` / `ExtractBlob` to `event/` | Dedup | 30 min | Identical pattern in both modules |
| 10 | Extract rejecting middleware helpers to `event/` | Dedup | 20 min | Exact duplication |
| 11 | Add integration test: sign + encrypt round-trip | Confidence | 30 min | Proves middleware composition works |
| 12 | Refactor `Ciphertext.Equal()` to use `subtle.ConstantTimeCompare` | Security | 5 min | Matches signing's approach; consistency |
| 13 | Move `Nonce()`/`Data()` off `Ciphertext` type | Architecture | 20 min | These are AES-GCM specific; type should be opaque |

### Tier 3: Feature Work (High Impact, High Effort)

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 14 | Add XChaCha20-Poly1305 implementation | Feature | 2h | Better nonce space, no hardware dependency |
| 15 | Add field-level encryption sub-package | Feature | 4h | High-value: selective encryption preserving event structure |
| 16 | Add key envelope encryption helper | Feature | 2h | KMS integration pattern: encrypt DEK with KEK |
| 17 | Add encryption to AGENTS.md module list | Docs | 5 min | Keep project docs current |

### Tier 4: Polish (Medium Impact, Low-Medium Effort)

| # | Task | Impact | Effort | Why |
|---|------|--------|--------|-----|
| 18 | Add golden test for ciphertext JSON serialization | Regression | 20 min | Matches signing's golden test pattern |
| 19 | Add doc.go examples for middleware composition | Docs | 15 min | Show encrypt+sign pipeline |
| 20 | Add `testdata/golden/` fixtures | Regression | 15 min | Standard practice |
| 21 | Add `event_test.go` file for event helper coverage | Coverage | 15 min | Split test file by concern |
| 22 | Verify nix build/lint passes for new module | CI | 10 min | Must work in CI pipeline |
| 23 | Add encryption to `.buildflow.yml` CI config | CI | 5 min | Already partially done |
| 24 | Add encryption to flake.nix if applicable | Build | 10 min | Nix-based CI |
| 25 | Property-based test: encrypt/decrypt is involutory | Robustness | 30 min | Using rapid or gopter |

---

## G) Top #1 Question I Cannot Figure Out Myself

**Should `CloneEvent` and the metadata blob helpers live in `event/` as public API, or as shared internal code?**

Arguments for public `event.CloneEvent(evt, key, value)`:
- Both signing and encryption need it, and future modules might too
- It's a generic event manipulation primitive
- Consumers might want to attach their own metadata blobs

Arguments against:
- It adds surface area to `event/`'s public API (this is a library — we can't remove it)
- It might encourage consumers to do low-level metadata manipulation instead of using signing/encryption middleware
- The current internal-only approach is safer

**My recommendation**: Add them as unexported helpers in `event/` (both signing and encryption already depend on event), NOT as public API. This preserves the "library, not framework" principle while eliminating duplication. Consumers use signing/encryption middleware, not raw metadata helpers.

---

## Test Results

```
14 BDD specs — PASS
11 unit tests — PASS
2 example tests — PASS
3 benchmarks — PASS

Benchmarks (1KB payload):
  AES256GCM_Encrypt    693ns/op   2320 B/op   3 allocs/op
  AES256GCM_Decrypt    504ns/op   2048 B/op   2 allocs/op
  AES256GCM_RoundTrip  1106ns/op  4368 B/op   5 allocs/op

Build: PASS (full workspace)
Signing tests: PASS (no regression)
```

## Module Graph Position

```
Layer 0: id/, dispatcher/, codec/
Layer 1: event/, command/, query/
Layer 2: schema/, snapshot/
Layer 3: decider/
Layer 4: memory/, signing/, otel/, encryption/  ← NEW
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
```

## Files Changed

```
encryption/             (NEW — 14 files)
go.work                 (added ./encryption)
```
