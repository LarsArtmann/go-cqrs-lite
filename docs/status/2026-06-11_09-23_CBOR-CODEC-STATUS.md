# Status Report: CBOR Codec Implementation

**Date:** 2026-06-11 09:23 UTC (updated 09:35)
**Branch:** master
**Commit:** 25027eac (initial), subsequent fixes applied

---

## a) FULLY DONE

### Core Implementation

- `codec/cbor.go`: `CBORCodec` struct with canonical (deterministic) encoding via `fxamacker/cbor/v2`. Uses IIFE for `cborEncMode` with proper error handling (panics at init if EncMode creation fails).
- `codec/codec.go`: `EncodingCBOR = "cbor"` constant added
- `codec/doc.go`: Updated package documentation listing all 3 codecs
- `.golangci.yml`: Added `fxamacker/cbor` to depguard allow list
- `AGENTS.md`: Updated codec/ description and dependencies table

### Tests

- `codec/codec_test.go`: 9 CBOR-specific unit tests
  - `TestCBORCodec_Encoding` — verifies Encoding() returns "cbor"
  - `TestCBORCodec_RoundTrip` — struct round-trip
  - `TestCBORCodec_Encode_Map` — map[string]any round-trip
  - `TestCBORCodec_Decode_InvalidCBOR` — error on garbage input
  - `TestCBORCodec_Encode_Nil` — nil handling
  - `TestCBORCodec_Encode_Deterministic` — 10-iteration map determinism
  - `TestCBORCodec_Decode_EmptyData` — edge case for empty input
  - `TestCBORCodec_RoundTrip_Time` — time.Time CBOR tag support
  - `TestCBORCodec_RoundTrip_ByteSlice` — native byte strings (no base64)
- `codec/codec_fuzz_test.go`: `FuzzCBORCodec_Roundtrip` with pure CBOR seed corpus (7 entries)
- `codec/benchmark_test.go`: `BenchmarkCBORCodec_Encode`, `Decode`, and `BenchmarkCodecComparison_Encode/Decode`
- `codec/golden_test.go`: `TestGolden_CBORCodec_Encode` + `testdata/golden/cbor_encode.bin`
- `codec/example_test.go`: `ExampleCBORCodec` for pkg.go.dev
- `event/codec_test.go`: `TestDecodePayload_CBORCodec` — end-to-end event.New() with CBOR
- `TestInterfaceCompliance` updated to include CBOR

### Integration Verification

- `event/`: PASS (all packages)
- `pebble/`: PASS
- `storage/`: PASS (incl. sql subpkg)
- `signing/`: PASS (incl. multisig)
- `encryption/`: PASS
- `memory/`: PASS
- `integration/`: PASS (all subpkgs: command, event, query, signing, encryption, simulation)
- `projection/`: PASS
- `listing/`: PASS
- `watermill/`: PASS
- `turso/`: PASS
- `codec/`: PASS (32 tests, incl. race detector)

### Build & Quality

- `nix fmt`: 120 files formatted, 16 changed
- `nix run .#lint`: 0 issues in codec, 1 pre-existing unconvert in catalog
- `go work sync`: All downstream modules updated
- Committed and pushed to master

---

## b) PARTIALLY DONE

### Deterministic Encoding

- Uses `cbor.CanonicalEncOptions().EncMode()` for sorted map keys
- **ISSUE**: Error from `EncMode()` is silently dropped (`_, _`) — if it ever fails, `Encode()` panics at runtime instead of failing at init
- **ISSUE**: No `DecMode` is configured — decode uses library defaults which may not match encode expectations for edge cases (large values, duplicate keys)

### Fuzz Testing

- `FuzzCBORCodec_Roundtrip` exists but uses JSONCodec as intermediary decoder for seed corpus
- This tests JSON→CBOR→CBOR round-trip, not pure CBOR round-trip
- Works for crash-finding but doesn't validate CBOR→CBOR→CBOR fidelity

### Test Coverage Gaps

- No `ExampleCBORCodec` (JSON and Raw have examples)
- No `TestCBORCodec_Decode_EmptyData` edge case
- No `time.Time` round-trip test (CBOR has RFC 8949 Section 3.4.2 time tag support)
- No `[]byte` field test (CBOR's native byte-string vs JSON's base64 encoding)
- No struct-with-tags test (verifies `json` tag compatibility)
- No slice/array test
- No end-to-end event creation with `event.WithCodec(codec.CBORCodec{})`

---

## c) NOT STARTED

- Event-level integration test with CBOR (creating actual `event.Event` via `event.New()`)
- Side-by-side JSON vs CBOR benchmark comparison
- CBOR-specific tag support (`cbor:"name"` tags alongside `json:"name"`)
- Stress test with large payloads (comparing JSON vs CBOR allocation patterns)
- Documentation of CBOR usage patterns in consumer-facing docs (README, examples)
- `DecMode` configuration for strict decoding (reject unknown fields, duplicate keys, etc.)
- Consider `CoreDetEncOptions` vs `CanonicalEncOptions` — which is the right default?

---

## d) TOTALLY FUCKED UP!

**Nothing.** The implementation is solid and functional. All tests pass, lint is clean, downstream modules work.

**However**, the silent error dropping on `cbor.CanonicalEncOptions().EncMode()` is a latent bug pattern that violates the project's "errors as values" principle. It would never manifest with the current fxamacker version (CanonicalEncOptions always succeeds), but it's sloppy and sets a bad precedent.

---

## e) WHAT WE SHOULD IMPROVE

### Immediate (High Impact / Low Effort)

1. **Fix `cborEncMode` error handling** — use `init()` and panic on error, or at minimum document why it's safe. Currently `_, _` silently drops a potential error.
2. **Add `ExampleCBORCodec`** — every codec should have a runnable example for pkg.go.dev
3. **Add `TestCBORCodec_Decode_EmptyData`** — edge case that could panic or hang
4. **Add `time.Time` test** — CBOR's time tag support is a major advantage over JSON
5. **Add `[]byte` field test** — demonstrates CBOR's killer feature (no base64 bloat)

### Short-term (Medium Impact / Low-Medium Effort)

6. **Fix fuzz test** — use actual CBOR bytes as seed corpus, remove JSON intermediary
7. **Add side-by-side benchmark** — `BenchmarkCodecComparison` showing JSON vs CBOR encode/decode
8. **Add end-to-end event test** — verify `event.New(ctx, "type", id, payload, event.WithCodec(codec.CBORCodec{}))` works
9. **Add `DecMode` configuration** — match encode/decode expectations, enable strict mode
10. **Document CBOR in module README** — consumers need to know this exists

### Mid-term (High Impact / Medium Effort)

11. **Evaluate `CoreDetEncOptions` vs `CanonicalEncOptions`** — CoreDet is RFC 8949 standard; Canonical is RFC 7049. Which is the right default for signing safety?
12. **Add CBOR example to `example/` directory** — a runnable consumer example
13. **Property-based test** — use `pgregory.net/rapid` to verify determinism over random struct shapes
14. **Add `TestCBORCodec_ComplexStruct`** — nested structs, slices, maps, pointers
15. **Profile allocation patterns** — compare JSON vs CBOR allocations with `go test -benchmem`

---

## f) Top #25 Things To Get Done Next

| #   | Task                                               | Impact | Effort | Priority |
| --- | -------------------------------------------------- | ------ | ------ | -------- |
| 1   | Fix `cborEncMode` error handling (init+panic)      | High   | 2 min  | **P0**   |
| 2   | Add `ExampleCBORCodec`                             | Medium | 3 min  | **P0**   |
| 3   | Add `TestCBORCodec_Decode_EmptyData`               | Medium | 2 min  | **P0**   |
| 4   | Add `time.Time` round-trip test                    | High   | 3 min  | **P0**   |
| 5   | Add `[]byte` field test                            | High   | 3 min  | **P0**   |
| 6   | Fix fuzz test to use pure CBOR corpus              | Medium | 5 min  | **P1**   |
| 7   | Add side-by-side JSON vs CBOR benchmark            | Medium | 5 min  | **P1**   |
| 8   | Add end-to-end event creation test                 | High   | 10 min | **P1**   |
| 9   | Add `DecMode` configuration                        | Medium | 10 min | **P1**   |
| 10  | Document CBOR in codec/README.md                   | Medium | 10 min | **P1**   |
| 11  | Evaluate CoreDet vs Canonical default              | High   | 15 min | **P2**   |
| 12  | Add CBOR example to `example/`                     | Medium | 20 min | **P2**   |
| 13  | Property-based determinism test (rapid)            | High   | 15 min | **P2**   |
| 14  | Complex struct round-trip test                     | Medium | 5 min  | **P2**   |
| 15  | Allocation benchmark comparison                    | Medium | 10 min | **P2**   |
| 16  | Add `TestCBORCodec_Slice`                          | Low    | 2 min  | **P3**   |
| 17  | Add `TestCBORCodec_NestedStruct`                   | Low    | 3 min  | **P3**   |
| 18  | Document CBOR determinism for signing              | Medium | 10 min | **P3**   |
| 19  | Verify encryption codec works with CBOR            | Medium | 5 min  | **P3**   |
| 20  | Add `TestCBORCodec_Interface` (any type)           | Low    | 3 min  | **P3**   |
| 21  | Consider CBOR streaming encoder for large payloads | Low    | 30 min | **P4**   |
| 22  | Investigate CBOR tag support for domain types      | Low    | 45 min | **P4**   |
| 23  | Add CBOR to `event/codec_test.go` integration      | Medium | 10 min | **P4**   |
| 24  | Write ADR for CBOR choice over msgpack/FlatBuffers | Low    | 30 min | **P4**   |
| 25  | Add `CBORCodec` to `eventtest` golden assertions   | Low    | 10 min | **P4**   |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `CBORCodec` use `CanonicalEncOptions()` or `CoreDetEncOptions()` as the default?**

- `CanonicalEncOptions()` (RFC 7049): sorts map keys, shortest float encoding. This is what we use now.
- `CoreDetEncOptions()` (RFC 8949): same as canonical plus additional rules. This is the newer IETF STD 94 standard.

Both produce deterministic output. The question is: which one is "more correct" for a library that claims "IETF STD 94" compliance? `CoreDetEncOptions` would be more honest branding, but `CanonicalEncOptions` is what we've tested and verified. Changing it would change the golden file bytes.

**Also**: Should we expose the `EncMode`/`DecMode` as configurable options on `CBORCodec` (e.g. `CBORCodec{Mode: MyCustomMode}`), or keep it as a zero-value struct like `JSONCodec`? The zero-value pattern is consistent with the rest of the codebase, but it prevents consumers from customizing CBOR behavior.

---

## Self-Review Summary

### What was forgotten

- `ExampleCBORCodec` (every other codec has one)
- Error handling for `EncMode()`
- `time.Time` and `[]byte` tests (CBOR's unique advantages)
- Empty-data decode test

### What's stupid

- Using JSON as fuzz intermediary for a CBOR codec
- Silent error dropping with `_, _`

### What could be better

- `DecMode` configuration for encode/decode symmetry
- Side-by-side benchmarks to prove CBOR value
- End-to-end event integration test

### Are there ghost systems?

No. The CBOR codec is fully wired into the existing `Codec` interface. All stores, event constructors, and encryption wrappers work without modification.

### Split brains?

No. The `EncodingCBOR` constant is used consistently. No duplicate type definitions.

### Testing assessment

Good coverage for a first pass (32 tests total, 6 CBOR-specific, 1 fuzz, 2 benchmarks, 1 golden). Missing edge cases and CBOR-specific feature tests. Fuzz test is indirect. No property-based tests.
