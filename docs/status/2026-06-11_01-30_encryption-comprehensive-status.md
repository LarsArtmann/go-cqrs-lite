# Encryption Module — Comprehensive Status Report

**Date:** 2026-06-11 01:30
**Module:** `encryption/`
**Status:** Feature-complete, all tests green, 0 lint issues

---

## A) FULLY DONE

| Item | Detail |
|------|--------|
| AES-256-GCM implementation | `NewAES256GCM(key)` — stdlib-only, AES-NI accelerated |
| XChaCha20-Poly1305 implementation | `NewXChaCha20Poly1305(key)` — 24-byte nonce, constant-time, recommended default |
| Core interfaces | `Encrypter`, `Decrypter`, `EncrypterDecrypter` — both algorithms behind same interface |
| Algorithm reporting | `Algorithm()` method on both impls, `AES256GCM`/`XChaCha20Poly1305` constants |
| Ciphertext type | `Ciphertext []byte` with `IsZero`, `Equal` (constant-time), `Bytes`, `String`, JSON serialization |
| Event helpers | `AttachEncryption`, `ExtractCiphertext`, `HasEncryption` — metadata envelope |
| Algorithm identification | `ExtractAlgorithm()`, `AlgorithmKey` metadata — auto-detected by middleware |
| Key ID support | `WithKeyID()`, `WithMiddlewareKeyID()`, `ExtractKeyID()`, `KeyIDKey` metadata — key rotation |
| Middleware | `EncryptMiddleware` (publish), `DecryptMiddleware` (subscribe) — both work with any algorithm |
| Composable codec wrapper | `NewCodec(inner codec.Codec, enc EncrypterDecrypter)` — wraps any codec |
| Sentinel errors | 4 classified errors via go-error-family |
| Self-review fixes | Removed redundant `slices.Clone`, fixed vestigial metadata, constant-time `Equal`, removed signing dep |
| Ciphertext is opaque | No algorithm-specific accessors on Ciphertext type |
| Signing dependency removed | Fixed module graph violation — encryption/ must NOT depend on signing/ |
| BDD tests | 20 Ginkgo specs (AES-GCM, XChaCha20, middleware, codec wrapper) |
| Unit tests | 30+ tests (both algorithms, ciphertext, codec, middleware, algorithm, key ID) |
| Fuzz tests | 5 fuzz tests (AES-GCM roundtrip+corrupt, XChaCha20 roundtrip+corrupt, Ciphertext JSON) |
| Property-based tests | 4 rapid tests (involution AES+XChaCha20, non-determinism, isolation) |
| Golden tests | 6 tests (Ciphertext JSON values, codec wrapper round-trips) |
| Example tests | 4 runnable examples (AES-GCM, XChaCha20, middleware, codec wrapper) |
| Benchmarks | 9 benchmarks (raw encrypt/decrypt/roundtrip for both + codec wrapper) |
| Integration tests | 3 tests (sign+encrypt full flow, codec wrapper, algorithm detection) |
| Package docs | `doc.go` with algorithm comparison, key ID, codec wrapper docs |
| README | Full module README with algorithm comparison table |
| AGENTS.md | Module list, test command, module tree, key patterns, dependencies |
| Build | Full workspace builds cleanly. 0 lint issues in encryption/ and integration/ |

**Test counts:** 20 BDD + 30+ unit + 5 fuzz + 4 property + 6 golden + 4 examples + 3 integration = **76+ tests, all green**

**Benchmarks (1KB payload, AMD Ryzen AI MAX+ 395):**
```
AES256GCM_Encrypt                        585 ns/op    2320 B/op    3 allocs/op
AES256GCM_Decrypt                        307 ns/op    1024 B/op    1 allocs/op
AES256GCM_RoundTrip                      791 ns/op    3344 B/op    4 allocs/op
XChaCha20Poly1305_Encrypt                852 ns/op    2328 B/op    3 allocs/op
XChaCha20Poly1305_Decrypt                602 ns/op    1024 B/op    1 allocs/op
CodecWrapper_XChaCha20_Encode            467 ns/op     280 B/op    6 allocs/op
CodecWrapper_XChaCha20_Decode            575 ns/op     280 B/op    7 allocs/op
CodecWrapper_XChaCha20_RoundTrip        1157 ns/op     545 B/op    12 allocs/op
```

**Module stats:** 25 Go files, ~2916 total lines

## B) PARTIALLY DONE

| Item | What's done | What's missing |
|------|-------------|----------------|
| Deduplication with signing | Identified ~80% overlap, extracted then reverted (premature abstraction) | Shared helpers could be extracted if both modules grow significantly |

## C) NOT STARTED

| Item | Impact | Effort |
|------|--------|--------|
| Field-level encryption sub-package (`encryption/fieldlevel/`) | High | Large |
| Key envelope encryption helper (KMS pattern) | Medium | Medium |
| FIPS 140-2 compliance mode (AES-GCM only, no XChaCha20) | Low | Small |
| Streaming encryption for large payloads | Medium | Large |

## D) TOTALLY FUCKED UP

| Item | Severity | Detail | Status |
|------|----------|--------|--------|
| signing/v2 import leaked into middleware.go | **CRITICAL** | Module graph violation: encryption/ imported signing/ for rejecting middleware helpers | **FIXED** in 34b60025 |
| fuzz_test.go used Encrypter but called .Decrypt() | Medium | Type mismatch from parallel session — helpers needed EncrypterDecrypter | **FIXED** in 34b60025 |
| property_test.go same issue | Medium | Same type mismatch as fuzz_test.go | **FIXED** in 34b60025 |

## E) WHAT WE SHOULD IMPROVE

### 1. Deduplication with signing is premature but real
Both modules have ~80% identical patterns:
- `RejectingPublishMiddleware` / `RejectingHandlerMiddleware` — identical implementations
- `Attach*/Extract*/Has*` — near-identical base64 + metadata envelope pattern
- `Ciphertext`/`Signature` — both are `[]byte` with `Bytes`, `IsZero`, `Equal`, `String`, `MarshalJSON`, `UnmarshalJSON`
- `detectAlgorithm` / `ExtractOrPassThrough` — similar interface detection patterns

We correctly decided NOT to extract shared helpers to `event/` (event/ is Layer 1 core, shouldn't carry middleware helper concerns). The duplication is **intentional isolation**. But if we add a third module with the same pattern, we should reconsider.

### 2. `EncryptionEncoding = "encrypted"` hides inner codec
The codec wrapper returns `"encrypted"` as encoding. Schema exporters and `validateEncodingMatch` won't know the original format. This is a design tradeoff we accepted but it could confuse consumers who try `DecodePayload[T](evt, codec.JSONCodec{})` on encrypted events.

### 3. No key rotation strategy
`ExtractKeyID` is there, but there's no built-in key resolver. Consumers must build their own key lookup table from key ID to decrypter. This is correct for a library (not a framework) but could be documented better.

### 4. No streaming encryption
Both implementations hold the entire plaintext/ciphertext in memory. For multi-GB payloads, this is a non-starter. Would need a streaming AEAD API (e.g., `io.Reader`/`io.Writer` wrappers).

### 5. testdata/golden/ directory is empty
The `testdata/golden/` directory exists but has no fixture files. The golden tests exist but use inline assertions rather than file-based golden comparison.

### 6. No integration with storage/ module
The storage/ module (SQL event store) doesn't know about encryption. Consumers must layer encryption middleware manually. A `storage.NewEncryptedEventStore(store, enc)` wrapper could simplify this.

### 7. Example in example/ directory
No example/ project demonstrating encryption usage. The existing examples in `example_test.go` are basic.

## F) Top 25 Things to Do Next

Sorted by impact × ease (Pareto):

| # | Task | Impact | Effort |
|---|------|--------|--------|
| 1 | Add `encryption/v2` to `go.work` if not present | High | 1m |
| 2 | Update README.md with algorithm ID and key ID examples | Medium | 10m |
| 3 | Add `EncryptionEncoding` documentation to codec/ module | Medium | 10m |
| 4 | Verify all examples in doc.go compile (`go test -run Example`) | Medium | 5m |
| 5 | Add `Algorithm` method to `Encrypter` interface (or document why not) | Medium | 20m |
| 6 | Add `encoding` field to `EncryptionEncoding` to carry inner codec info | Medium | 15m |
| 7 | Consider `encrypted+json` encoding string instead of just `encrypted` | Medium | 10m |
| 8 | Add key resolver interface `type KeyResolver func(keyID string) (Decrypter, error)` | High | 30m |
| 9 | Document key rotation pattern in README | Medium | 15m |
| 10 | Add `encryption/v2` to CI per-module test matrix (buildflow.yml) | Medium | 5m |
| 11 | Add `nix run .#check-layers` for encryption module | Medium | 5m |
| 12 | Create `example/encryption/` demonstrating full sign+encrypt flow | High | 30m |
| 13 | Add storage wrapper: `storage.EncryptedEventStore` | High | 2h |
| 14 | Streaming encryption for large payloads (io.Reader/io.Writer) | High | 4h |
| 15 | Field-level encryption (`encryption/fieldlevel/`) | High | 4h |
| 16 | Key envelope encryption helper (KMS pattern) | Medium | 2h |
| 17 | Add `encryption/v2` to `cmd/api-stability/` golden file | Medium | 10m |
| 18 | Benchmark: compare with encrypt-then-MAC vs AEAD | Low | 30m |
| 19 | Consider `crypto/cipher.Stream` for streaming mode | Low | 1h |
| 20 | Add versioned ciphertext format (prefix byte for algorithm) | Medium | 30m |
| 21 | Consider using `golang.org/x/crypto/nacl/secretbox` as alternative | Low | 30m |
| 22 | Add `Ciphertext.UnmarshalFrom(io.Reader)` for streaming decode | Low | 1h |
| 23 | Investigate `google/tink` for key management integration | Medium | 2h |
| 24 | Add `encryption/v2` to `catalog/` schema exporter | Low | 30m |
| 25 | Push all commits and verify CI passes | Medium | 5m |

## G) Top #1 Question I Cannot Figure Out Myself

**Should `Algorithm()` be on the `Encrypter` interface or remain an optional detection via type assertion?**

Current approach: `detectAlgorithm()` uses a private type assertion to `interface{ Algorithm() Algorithm }`. This means:
- Third-party `Encrypter` implementations won't report their algorithm unless they implement the `Algorithm()` method
- The method isn't part of the public `Encrypter` interface, so it's invisible in docs
- But adding it to `Encrypter` would break the interface segregation (some consumers only need `Encrypt`)

Options:
1. Keep current approach (private type assertion) — works, but invisible
2. Add `Algorithm() Algorithm` to a new `AlgorithmAware` interface — explicit but more types
3. Add `Algorithm() Algorithm` to `Encrypter` interface — cleanest, but breaking change

The question matters because it affects how consumers discover which algorithm encrypted a given event, which is critical for key rotation.

## Module Files

```
encryption/
├── aesgcm.go                    # AES-256-GCM implementation
├── aesgcm_test.go               # AES-GCM unit tests
├── algorithm.go                 # Algorithm type, ExtractAlgorithm, ExtractKeyID
├── algorithm_test.go            # Algorithm and key ID tests
├── benchmark_test.go            # All algorithm + codec wrapper benchmarks
├── ciphertext.go                # Ciphertext type (opaque blob)
├── ciphertext_golden_test.go    # Ciphertext JSON golden tests
├── ciphertext_test.go           # Ciphertext unit tests
├── codec.go                     # Composable codec wrapper (NewCodec)
├── codec_golden_test.go         # Codec wrapper golden tests
├── codec_test.go                # Codec wrapper tests
├── doc.go                       # Package documentation
├── encrypter.go                 # Encrypter/Decrypter interfaces
├── encryption_bdd_suite_test.go # Ginkgo suite entry
├── encryption_bdd_test.go       # BDD specs (20 scenarios)
├── errors.go                    # Sentinel errors
├── event.go                     # Attach/Extract/Has + WithKeyID
├── example_test.go              # 4 runnable examples
├── fuzz_test.go                 # 5 fuzz tests
├── go.mod                       # Module definition (no signing dep)
├── go.sum                       # Dependency checksums
├── middleware.go                 # Encrypt/Decrypt middleware + algorithm detection
├── middleware_test.go            # Middleware unit tests
├── property_test.go             # 4 property-based tests (rapid)
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
| Test-only | `onsi/ginkgo/v2`, `onsi/gomega`, `pgregory.net/rapid` |

## Commits This Session

```
34b60025 fix(encryption): remove leaked signing dependency from middleware.go
8ed9abc7 feat(encryption): add Algorithm enum, codec wrapper, property/fuzz/golden tests, and integration tests
ecfa2429 refactor(encryption): deduplicate rejecting middleware helpers and test helpers
```
