# Encryption Module — Sprint Status Report

**Date:** 2026-06-11 00:47
**Module:** `encryption/`
**Sprint:** XChaCha20-Poly1305 + Codec Wrapper + Self-Review Fixes

---

## A) FULLY DONE

| Item | Detail |
|------|--------|
| AES-256-GCM implementation | `NewAES256GCM(key)` — stdlib-only, AES-NI accelerated |
| XChaCha20-Poly1305 implementation | `NewXChaCha20Poly1305(key)` — 24-byte nonce, constant-time, recommended default |
| Core interfaces | `Encrypter`, `Decrypter`, `EncrypterDecrypter` — both algorithms behind same interface |
| Ciphertext type | `Ciphertext []byte` with `IsZero`, `Equal` (constant-time), `Bytes`, `String`, JSON serialization |
| Event helpers | `AttachEncryption`, `ExtractCiphertext`, `HasEncryption` — metadata envelope |
| Middleware | `EncryptMiddleware` (publish), `DecryptMiddleware` (subscribe) — both work with any `Encrypter`/`Decrypter` |
| Composable codec wrapper | `NewCodec(inner codec.Codec, enc EncrypterDecrypter)` — wraps any codec with encrypt/decrypt |
| Sentinel errors | 4 classified errors via go-error-family |
| Self-review fixes | Removed redundant `slices.Clone`, fixed vestigial metadata, constant-time `Equal`, unexported `CloneEvent`, removed signing dep |
| Ciphertext made opaque | Removed `Nonce()`/`Data()` — algorithm-specific concern, not generic type |
| BDD tests | 20 Ginkgo specs (AES-GCM, XChaCha20, middleware, codec wrapper) |
| Unit tests | 22 tests (both algorithms, ciphertext, codec, middleware) |
| Example tests | 4 runnable examples (AES-GCM, XChaCha20, middleware, codec wrapper) |
| Benchmarks | 6 benchmarks (encrypt/decrypt/round-trip for both algorithms) |
| Package docs | `doc.go` with algorithm comparison and usage examples |
| README | Full rewrite with algorithm comparison table, codec wrapper section |
| AGENTS.md | Module list, test command, module tree, Layer 4 all updated |
| Build | Full workspace builds cleanly. All tests pass. |

**Test counts:** 20 BDD + 22 unit + 4 examples = **46 tests, all green**

**Benchmarks (1KB payload, AMD Ryzen AI MAX+ 395):**

```
AES256GCM_Encrypt              497ns/op    2320 B/op    3 allocs/op
AES256GCM_Decrypt              272ns/op    1024 B/op    1 allocs/op
AES256GCM_RoundTrip            850ns/op    3344 B/op    4 allocs/op
XChaCha20Poly1305_Encrypt      791ns/op    2328 B/op    3 allocs/op
XChaCha20Poly1305_Decrypt      590ns/op    1024 B/op    1 allocs/op
XChaCha20Poly1305_RoundTrip   1556ns/op    3352 B/op    4 allocs/op
```

**Commits this session:**
```
3161334b feat(encryption): add XChaCha20-Poly1305, remove signing dependency, make Ciphertext opaque
8f0aba7  feat(encryption): add composable codec wrapper — encryption.NewCodec(inner, enc)
e15e744  feat(encryption): add XChaCha20-Poly1305 algorithm, composable codec wrapper, and full comparison
```

## B) PARTIALLY DONE

| Item | What's done | What's missing |
|------|-------------|----------------|
| Deduplication with signing | Identified ~80% overlap, removed erroneous signing import | Shared helpers not yet extracted to `event/` |

## C) NOT STARTED

| Item | Impact | Effort |
|------|--------|--------|
| Fuzz tests for AES-GCM + XChaCha20 Decrypt | Medium | Small |
| Integration test: sign + encrypt round-trip | Medium | Small |
| Integration test: sign + encrypt via codec wrapper | Medium | Small |
| Golden test for Ciphertext JSON serialization | Low | Small |
| Property-based test (rapid): encrypt/decrypt is involutory | Medium | Medium |
| Field-level encryption sub-package (`encryption/fieldlevel/`) | High | Large |
| Key envelope encryption helper (KMS pattern) | Medium | Medium |
| Extract shared helpers to `event/` (attachBlob/extractBlob/rejecting middleware) | High | Medium |
| Refactor signing/ to use shared event/ helpers | High | Small |
| Verify nix build/lint passes for encryption module | Medium | Small |
| `testdata/golden/` fixtures for codec wrapper | Low | Small |

## D) TOTALLY FUCKED UP

| Item | Severity | Detail |
|------|----------|--------|
| Nothing broken | — | All 46 tests pass, workspace builds cleanly, no regressions |

## E) WHAT WE SHOULD IMPROVE

1. **Deduplication with signing** — Both modules have identical rejecting middleware helpers, near-identical `Extract*`/`Attach*`/`Has*` patterns, and identical `[]byte` type methods. Should be extracted to `event/` as unexported helpers.

2. **`EncryptionEncoding = "encrypted"` hides inner codec** — The codec wrapper returns `"encrypted"` as encoding, but the inner codec is JSON. Schema exporters and `validateEncodingMatch` in event/codec.go won't know the original format. This is a design tradeoff we accepted (composable, not combinable) but it could confuse consumers.

3. **No key rotation or key ID support** — Consumers need to track which key encrypted which event. Should add optional key ID to the metadata envelope alongside the ciphertext.

4. **No algorithm identification in ciphertext** — Ciphertext is opaque bytes. There's no way to tell if a ciphertext was produced by AES-GCM or XChaCha20 without trying both. Should consider a prefix byte or metadata field.

5. **AGENTS.md not fully committed** — The AGENTS.md changes (encryption in test command, module tree, Layer 4) are staged but the commit was absorbed by a parallel session's buildflow run. Need to verify it landed.

## F) Top 25 Things to Do Next

Sorted by impact × ease (Pareto):

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Extract shared rejecting middleware helpers to `event/` (unexported) | High | 20m |
| 2 | Extract shared `attachBlob`/`extractBlob`/`hasBlob` to `event/` (unexported) | High | 30m |
| 3 | Refactor `signing/` to use shared `event/` helpers | High | 30m |
| 4 | Refactor `encryption/` to use shared `event/` helpers | High | 20m |
| 5 | Integration test: sign + encrypt round-trip through middleware | High | 30m |
| 6 | Add algorithm identification to ciphertext (1-byte prefix or metadata field) | Medium | 30m |
| 7 | Add key ID support to encryption metadata envelope | Medium | 30m |
| 8 | Fuzz test for AES-GCM Decrypt | Medium | 10m |
| 9 | Fuzz test for XChaCha20 Decrypt | Medium | 5m |
| 10 | Integration test: sign + encrypt via codec wrapper | Medium | 20m |
| 11 | Property-based test: encrypt/decrypt is involutory (rapid) | Medium | 30m |
| 12 | Verify nix build/lint passes for encryption module | Medium | 10m |
| 13 | Field-level encryption sub-package (`encryption/fieldlevel/`) | High | 4h |
| 14 | Key envelope encryption helper (KMS pattern) | Medium | 2h |
| 15 | Golden test for Ciphertext JSON serialization | Low | 15m |
| 16 | `testdata/golden/` fixtures for codec wrapper | Low | 15m |
| 17 | Update HTML design review in docs/status/ to reflect XChaCha20 addition | Low | 15m |
| 18 | Add codec wrapper to pkg.go.dev examples in doc.go | Low | 10m |
| 19 | Add `encryption/v2` to flake.nix if applicable | Low | 10m |
| 20 | Add encryption to `.buildflow.yml` CI per-module test matrix | Low | 5m |
| 21 | Consider `Nonce()`/`Data()` as package-level functions with algorithm parameter | Low | 20m |
| 22 | Benchmark: encrypting codec wrapper overhead vs raw encrypt | Low | 10m |
| 23 | Add `EncryptionEncoding` documentation to codec/ module | Low | 10m |
| 24 | Consider `errors.Is` for codec wrapper decrypt failures | Low | 5m |
| 25 | Push all commits and verify CI passes | Medium | 5m |

## G) Top #1 Question I Cannot Figure Out Myself

**Should the codec wrapper's `Encoding()` return `"encrypted"` or something like `"encrypted+json"`?**

Returning `"encrypted"` means `event.validateEncodingMatch()` will fail when comparing against a `"json"` codec. This is correct behavior (the payload IS encrypted, not JSON anymore) but it means consumers can't use `DecodePayload[T](evt, codec.JSONCodec{})` on encrypted events — they must use the encrypting codec.

The alternative `"encrypted+json"` would carry the inner codec's identity but creates a new encoding namespace that nothing else understands.

Current choice: `"encrypted"`. This is the right tradeoff — if the payload is encrypted, the encoding IS encrypted, not JSON. Consumers must decrypt first (via middleware or the encrypting codec). But I'm not 100% sure this won't surprise consumers who expect `"json"` to work after decryption.

---

## Module Files

```
encryption/
├── aesgcm.go                    # AES-256-GCM implementation
├── aesgcm_test.go               # AES-GCM unit tests
├── benchmark_test.go            # Both algorithm benchmarks
├── ciphertext.go                # Ciphertext type (opaque blob)
├── ciphertext_test.go           # Ciphertext unit tests
├── codec.go                     # Composable codec wrapper (NewCodec)
├── codec_test.go                # Codec wrapper tests
├── doc.go                       # Package documentation
├── encrypter.go                 # Encrypter/Decrypter interfaces
├── encryption_bdd_suite_test.go # Ginkgo suite entry
├── encryption_bdd_test.go       # BDD specs (20 scenarios)
├── errors.go                    # Sentinel errors
├── event.go                     # Attach/Extract/Has encryption helpers
├── example_test.go              # 4 runnable examples
├── go.mod                       # Module definition
├── go.sum                       # Dependency checksums
├── middleware.go                 # Encrypt/Decrypt middleware
├── middleware_test.go            # Middleware unit tests
├── README.md                    # Module README
├── testdata/golden/             # Golden test fixtures (empty)
├── test_helpers_test.go         # Test helpers
├── xchacha20.go                 # XChaCha20-Poly1305 implementation
└── xchacha20_test.go            # XChaCha20 unit tests
```

## Module Graph Position

```
Layer 0: id/, dispatcher/, codec/
Layer 1: event/, command/, query/
Layer 2: schema/, snapshot/
Layer 3: decider/
Layer 4: memory/, signing/, encryption/, otel/  ← HERE
Layer 5: middleware/, storage/, projection/, listing/, watermill/, pebble/, turso/
```

## Dependencies

| Category | Packages |
|----------|----------|
| Production | `event/v2`, `id/v2`, `codec/v2`, `golang.org/x/crypto/chacha20poly1305` |
| Test-only | `onsi/ginkgo/v2`, `onsi/gomega` |
