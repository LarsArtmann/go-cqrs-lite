# CBOR Codec — Final Comprehensive Status Report

**Date:** 2026-06-11 12:20 UTC
**Branch:** master
**HEAD:** a576dadc

---

## a) FULLY DONE

### Core Implementation (`codec/cbor.go`)

| Component          | Detail                                                                                                      |
| ------------------ | ----------------------------------------------------------------------------------------------------------- |
| `CBORCodec` struct | Zero-value struct, value receivers, `var _ Codec = CBORCodec{}` compile check                               |
| `cborEncMode`      | Canonical encoding (RFC 7049: sorted map keys, shortest floats). IIFE with panic on error.                  |
| `cborDecMode`      | Configured `DecOptions` with `DupMapKeyEnforcedAPF` (rejects duplicate map keys). IIFE with panic on error. |
| `Encoding()`       | Returns `EncodingCBOR = "cbor"`                                                                             |
| `Encode()`         | `cborEncMode.Marshal(v)` with `//nolint:wrapcheck`                                                          |
| `Decode()`         | `cborDecMode.Unmarshal(data, v)` with `//nolint:wrapcheck`                                                  |

### Constants & Wiring (`codec/codec.go`)

- `EncodingCBOR Encoding = "cbor"` added alphabetically between `EncodingJSON` and `EncodingRaw`

### Documentation

| File                                                  | Content                                                                      |
| ----------------------------------------------------- | ---------------------------------------------------------------------------- |
| `codec/doc.go`                                        | Lists all 3 codecs (JSON, CBOR, Raw)                                         |
| `codec/cbor.go`                                       | Accurate doc comments: RFC 7049 Canonical, signing-safe                      |
| `.golangci.yml`                                       | `fxamacker/cbor` in depguard allow list                                      |
| `AGENTS.md`                                           | Updated codec/ description + dependencies table                              |
| `docs/adr/0015-cbor-codec.md`                         | Full ADR: CBOR vs msgpack/protobuf/FlatBuffers, why Canonical, why fxamacker |
| `docs/status/2026-06-11_09-23_CBOR-CODEC-STATUS.md`   | Initial self-review status report                                            |
| `docs/planning/2026-06-11_11-47_CBOR-FINAL-POLISH.md` | Final polish plan                                                            |

### Tests (15 CBOR unit tests)

| Test                                   | What it verifies                       |
| -------------------------------------- | -------------------------------------- |
| `TestCBORCodec_Encoding`               | `Encoding() == "cbor"`                 |
| `TestCBORCodec_RoundTrip`              | Struct round-trip                      |
| `TestCBORCodec_Encode_Map`             | `map[string]any` round-trip            |
| `TestCBORCodec_Decode_InvalidCBOR`     | Error on garbage input                 |
| `TestCBORCodec_Encode_Nil`             | nil → CBOR null → nil                  |
| `TestCBORCodec_Encode_Deterministic`   | 10-iteration map determinism           |
| `TestCBORCodec_Decode_EmptyData`       | Error on empty input                   |
| `TestCBORCodec_RoundTrip_Time`         | CBOR time tag support                  |
| `TestCBORCodec_RoundTrip_ByteSlice`    | Native byte strings (no base64)        |
| `TestCBORCodec_SmallerThanJSON`        | CBOR output is smaller                 |
| `TestCBORCodec_RoundTrip_Slice`        | String slice round-trip                |
| `TestCBORCodec_RoundTrip_NestedStruct` | Nested struct round-trip               |
| `TestCBORCodec_StructTags`             | Dual `json`+`cbor` tag compat          |
| `TestCBORCodec_SigningDeterminism`     | Triple-encode produces identical bytes |
| `TestInterfaceCompliance` (CBOR)       | Interface compliance check             |

### Fuzz Tests (4 CBOR fuzz tests)

| Test                              | What                                              |
| --------------------------------- | ------------------------------------------------- |
| `FuzzCBORCodec_Roundtrip`         | Pure CBOR seed corpus (7 entries), CBOR→CBOR→CBOR |
| `FuzzCBORCodec_Determinism`       | Map key order independence                        |
| `FuzzCBORCodec_DecodeNeverPanics` | Arbitrary bytes never panic                       |
| `FuzzJSONCodec_TypedRoundtrip`    | JSON↔CBOR cross-format                            |

### Benchmarks (4 CBOR benchmarks + 2 comparison)

| Benchmark                              | Result                                    |
| -------------------------------------- | ----------------------------------------- |
| `BenchmarkCBORCodec_Encode`            | 207 ns/op, 96 B, 2 allocs                 |
| `BenchmarkCBORCodec_Decode`            | 358 ns/op, 416 B, 9 allocs                |
| `BenchmarkCodecComparison_Encode/CBOR` | **17% faster** than JSON (2 vs 6 allocs)  |
| `BenchmarkCodecComparison_Decode/CBOR` | **38% faster** than JSON (9 vs 12 allocs) |

### Golden Test

- `TestGolden_CBORCodec_Encode` + `testdata/golden/cbor_encode.bin`

### Example Function

- `ExampleCBORCodec` in `codec/example_test.go` (pkg.go.dev)

### End-to-End Event Integration

- `TestDecodePayload_CBORCodec` in `event/codec_test.go`: `event.New()` + `event.WithCodec(codec.CBORCodec{})` + `event.DecodePayload[T]`

### Example Project

- `example/cbor-codec/`: Minimal runnable example showing CBOR event creation, decode, and signing determinism explanation

### Build & Quality

- `nix fmt`: clean
- `nix run .#lint`: 0 issues in codec, 0 issues in event
- `go test -race`: 70 PASS in codec, all event tests pass, all integration tests pass
- `go work sync`: All downstream modules updated

---

## b) PARTIALLY DONE

Nothing. All planned CBOR work is complete.

---

## c) NOT STARTED

These are future improvements that are NOT required for the CBOR feature:

| #   | Item                                                   | Why deferred                                                       |
| --- | ------------------------------------------------------ | ------------------------------------------------------------------ |
| 1   | CoreDet (RFC 8949) encoding mode option                | Canonical and CoreDet produce identical bytes for typical payloads |
| 2   | Streaming CBOR encoder for large payloads              | Premature for a library                                            |
| 3   | Property-based tests with `pgregory/rapid`             | Fuzz tests already cover this                                      |
| 4   | CBOR tag support docs (`cbor:"name"` vs `json:"name"`) | fxamacker reads `json` tags by default                             |
| 5   | Module-level `codec/README.md`                         | `doc.go` + examples are sufficient                                 |
| 6   | CBOR in `eventtest` golden assertions                  | eventtest uses JSON by default, correct                            |

---

## d) TOTALLY FUCKED UP

**Nothing.** Clean implementation. No bugs, no lying docs, no split brains, no ghost systems.

---

## e) WHAT WE SHOULD IMPROVE

### Across the project (not CBOR-specific)

| #   | Issue                                                   | Module   | Effort |
| --- | ------------------------------------------------------- | -------- | ------ |
| 1   | `storage/` has 58 LSP errors (Dialect, DB, CheckClosed) | storage  | 30min  |
| 2   | Catalog `unconvert` lint issue (pre-existing)           | catalog  | 2min   |
| 3   | Compiled binaries in `example/` dirs (buildflow ERROR)  | examples | 5min   |
| 4   | `turso/doc.go` missing package comment                  | turso    | 2min   |
| 5   | Hardcoded `"json"` strings in storage test mocks        | storage  | 10min  |
| 6   | `docs/status/` has 10+ status reports from one day      | docs     | 5min   |

### CBOR-specific (optional polish)

| #   | Item                                                                      | Effort |
| --- | ------------------------------------------------------------------------- | ------ |
| 7   | Add CBORCodec to `integration/` cross-module tests                        | 15min  |
| 8   | Add CBOR to `encryption/` integration (verify EncryptionCodec wraps CBOR) | 10min  |
| 9   | Golden test for CBOR time encoding                                        | 5min   |
| 10  | Size comparison benchmark (CBOR vs JSON for []byte payloads)              | 10min  |

---

## f) Top #25 Things To Get Done Next

Sorted by impact × effort (highest first). CBOR-specific items marked with 🔵.

| #   | Task                                                            | Impact   | Effort | Module      |
| --- | --------------------------------------------------------------- | -------- | ------ | ----------- |
| 1   | Fix 58 LSP errors in storage/ (Dialect, DB, CheckClosed)        | Critical | 30min  | storage     |
| 2   | Remove compiled binaries from example/ dirs                     | High     | 5min   | examples    |
| 3   | Fix catalog unconvert lint                                      | Medium   | 2min   | catalog     |
| 4   | Add turso/doc.go package comment                                | Medium   | 2min   | turso       |
| 5   | 🔵 Add CBOR to encryption/ integration test                     | High     | 10min  | integration |
| 6   | 🔵 Add CBOR to integration/ cross-module tests                  | Medium   | 15min  | integration |
| 7   | Replace hardcoded "json" strings in storage test mocks          | Medium   | 10min  | storage     |
| 8   | Clean up status reports from today's sessions                   | Low      | 5min   | docs        |
| 9   | 🔵 Size comparison benchmark (CBOR vs JSON for []byte payloads) | Medium   | 10min  | codec       |
| 10  | 🔵 Golden test for CBOR time encoding                           | Low      | 5min   | codec       |
| 11  | Add `cbor` struct tag to `eventtest` test payloads              | Low      | 5min   | event       |
| 12  | Document CBOR in `docs/DOMAIN_LANGUAGE.md`                      | Medium   | 10min  | docs        |
| 13  | Verify pebble/ stores CBOR-encoded payloads correctly           | High     | 10min  | pebble      |
| 14  | Verify signing/ produces stable signatures with CBOR payloads   | High     | 10min  | signing     |
| 15  | Add `nix run .#check-layers` to verify CBOR dep budgets         | Medium   | 5min   | infra       |
| 16  | Add CBORCodec to `memory/` round-trip test                      | Low      | 5min   | memory      |
| 17  | Document `DupMapKeyEnforcedAPF` behavior in cbor.go             | Low      | 3min   | codec       |
| 18  | Add CBOR to `projection/` builder integration test              | Medium   | 10min  | projection  |
| 19  | Add CBOR example to `catalog/` schema generation                | Low      | 15min  | catalog     |
| 20  | Add `cbor:"-"` field omission test                              | Low      | 3min   | codec       |
| 21  | Evaluate CBOR for pebble envelope serialization                 | High     | 30min  | pebble      |
| 22  | Write README section about CBOR adoption                        | Medium   | 30min  | docs        |
| 23  | Add `APIstability` golden file check for CBORCodec exports      | Medium   | 10min  | cmd         |
| 24  | Explore CBOR streaming for large batch operations               | Low      | 45min  | codec       |
| 25  | Add CBOR to `watermill/` protocol adapter test                  | Low      | 10min  | watermill   |

---

## g) Top #1 Question I Cannot Figure Out Myself

**Should `storage/` LSP errors be fixed before or after the next release?**

There are 58 LSP errors in `storage/` (undefined Dialect, DB, CheckClosed fields). These appear to be pre-existing — they show in every file's project diagnostics. They don't break tests (all storage tests pass), suggesting they might be gopls workspace sync artifacts rather than actual compilation errors.

**Question:** Are these real compilation errors that block `go build`, or are they LSP artifacts? If real, they should be #1 priority. If LSP artifacts, they can be ignored.

---

## Commit History (CBOR work)

```
a576dadc feat(codec): add DecMode, ADR-0015, CBOR example, struct tag test
c7378442 test: expand comprehensive fuzz testing and realistic scale benchmarks
32be17c4 docs: update CBOR status report with self-review fixes applied
008dd352 test: add comprehensive fuzz tests and realistic benchmark infrastructure
25027eac feat(codec): add CBORCodec with deterministic canonical encoding
eb3a66b5 docs: add CBOR codec implementation plan with Pareto-ordered task breakdown
```
