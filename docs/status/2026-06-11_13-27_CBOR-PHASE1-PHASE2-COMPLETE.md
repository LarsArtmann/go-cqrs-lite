# CBOR Phase 1 + Phase 2 — Comprehensive Status

**Date:** 2026-06-11 13:27 UTC
**Branch:** master
**HEAD:** 353c7df7 (1 commit ahead of origin)

---

## a) FULLY DONE

### Phase 1: CBORCodec (shipped at `a576dadc`)

| Component                     | Detail                                                                       |
| ----------------------------- | ---------------------------------------------------------------------------- |
| `codec/cbor.go`               | CBORCodec with canonical EncMode (RFC 7049) + DecMode (DupMapKeyEnforcedAPF) |
| `codec/codec.go`              | `EncodingCBOR = "cbor"` constant                                             |
| `codec/doc.go`                | Lists all 3 codecs (JSON, CBOR, Raw)                                         |
| `codec/go.mod`                | `fxamacker/cbor/v2 v2.9.2` direct dependency                                 |
| `example/cbor-codec/`         | Minimal runnable example                                                     |
| `docs/adr/0015-cbor-codec.md` | Full ADR: CBOR vs msgpack/protobuf/FlatBuffers                               |
| Tests                         | 15 unit, 4 fuzz, 4 benchmarks, 1 golden, 1 example, 1 e2e                    |
| Benchmarks                    | 17% faster encode (2 vs 6 allocs), 38% faster decode (9 vs 12 allocs)        |

### Phase 2: Pebble CBOR Envelope Migration (shipped at `353c7df7`)

| Component                 | Detail                                                                                                                                         |
| ------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `pebble/serialization.go` | `json.Marshal` → `cborEncMode.Marshal` (canonical CBOR). `json.Unmarshal` → `isCBOR()` format sniff + CBOR/JSON decode. Base64 tax eliminated. |
| `pebble/go.mod`           | `fxamacker/cbor/v2` moved from indirect → direct                                                                                               |
| `pebble/doc.go`           | Documented CBOR envelope format + JSON backward compat                                                                                         |
| Backward compat           | `isCBOR()` sniff: CBOR maps (0xa0–0xbf) vs JSON objects (0x7b). Legacy JSON envelopes read transparently.                                      |

**New tests (12 unit + 2 fuzz + 3 benchmarks):**

| Test                                      | What it verifies                            |
| ----------------------------------------- | ------------------------------------------- |
| `TestSerializeEvent_CBORFormat`           | First byte is CBOR major type 5             |
| `TestDeserializeEvent_JSONBackwardCompat` | Hand-crafted JSON envelope reads correctly  |
| `TestDeserializeEvent_CBORRoundTrip`      | Full CBOR encode→decode field fidelity      |
| `TestSerializeEvent_NoBase64`             | Raw bytes survive without base64 encoding   |
| `TestIsCBOR`                              | Format sniff for 9 edge cases               |
| `TestSerializeEvent_Deterministic`        | 10× same event = identical bytes            |
| `TestEventStore_Persistence_CBOR`         | Save → close DB → reopen → load             |
| `TestEventStore_BinaryPayload`            | 0x00–0xFF round-trip through Pebble         |
| `TestDeserializeEvent_EmptyData`          | nil/empty error handling                    |
| `TestSerializeEvent_SmallerThanJSON`      | CBOR < JSON for binary payloads             |
| `TestDeserializeEvent_CBORWithMetadata`   | Correlation/causation IDs survive           |
| `FuzzDeserializeEvent`                    | Arbitrary bytes never panic (7 seed corpus) |
| `FuzzSerializeDeserializeRoundtrip`       | Payload fidelity under fuzz (4 seeds)       |
| `BenchmarkSerializeEnvelope`              | 519 ns/op, 6 allocs                         |
| `BenchmarkDeserializeEnvelope`            | 4214 ns/op, 30 allocs                       |

**All 25 pebble tests pass with `-race`. Zero lint issues.**

### Documentation updates this session

| File                                                      | Change                                                                                                                |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `docs/brainstorming/binary-wire-codec.html`               | Phase 1 banner + Phase 2 DONE marker + measured benchmarks + "What's Done" section + integration point status updates |
| `docs/planning/2026-06-11_PHASE2-PEBBLE-CBOR-ENVELOPE.md` | Full 37-task plan                                                                                                     |
| `AGENTS.md`                                               | pebble/ description updated to "CBOR envelope with JSON backward compat"                                              |
| `pebble/doc.go`                                           | CBOR envelope format + backward compat documented                                                                     |

## b) PARTIALLY DONE

Nothing. All planned CBOR work (Phase 1 + Phase 2) is complete.

## c) NOT STARTED

These are future phases from the serialization blueprint — **not blockers**:

| Phase   | Description                                                                                   | Status                                                                             |
| ------- | --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Phase 3 | Custom binary envelope (Option C) — zero-copy envelope parse via `encoding/binary` + `unsafe` | Not started. Only warranted if CBOR ceiling is too low for replay-heavy workloads. |
| Phase 4 | FlatBuffers codec — opt-in for extreme throughput consumers willing to adopt `.fbs` schemas   | Not started. Available as future opt-in codec.                                     |

## d) TOTALLY FUCKED UP

Nothing. Clean implementation. No bugs, no lying docs, no split brains.

## e) WHAT WE SHOULD IMPROVE

### Cross-project (pre-existing, not CBOR-related)

| #   | Issue                                                    | Module   | Effort |
| --- | -------------------------------------------------------- | -------- | ------ |
| 1   | `storage/` has 58+ LSP errors (Dialect, DB, CheckClosed) | storage  | 30min  |
| 2   | Catalog `unconvert` lint issue                           | catalog  | 2min   |
| 3   | Compiled binaries in `example/` dirs                     | examples | 5min   |
| 4   | `turso/doc.go` missing package comment                   | turso    | 2min   |
| 5   | Hardcoded `"json"` strings in storage test mocks         | storage  | 10min  |

### CBOR integration gaps (optional polish)

| #   | Item                                                                  | Effort |
| --- | --------------------------------------------------------------------- | ------ |
| 6   | Add CBORCodec to `integration/` cross-module tests                    | 15min  |
| 7   | Verify pebble/ stores CBOR payloads correctly with signing middleware | 10min  |
| 8   | Verify encryption/ wraps CBOR payloads through CBOR Pebble envelope   | 10min  |
| 9   | Add CBOR to `projection/` builder integration test                    | 10min  |
| 10  | Document CBOR in `docs/DOMAIN_LANGUAGE.md`                            | 10min  |

## f) Top #25 Things To Get Done Next

| #   | Task                                                                                        | Impact   | Effort | Module      |
| --- | ------------------------------------------------------------------------------------------- | -------- | ------ | ----------- |
| 1   | Push `353c7df7` to origin/master                                                            | Critical | 1min   | infra       |
| 2   | Fix 58 LSP errors in `storage/` (Dialect, DB, CheckClosed)                                  | Critical | 30min  | storage     |
| 3   | Remove compiled binaries from `example/` dirs                                               | High     | 5min   | examples    |
| 4   | Fix catalog `unconvert` lint                                                                | Medium   | 2min   | catalog     |
| 5   | Add `turso/doc.go` package comment                                                          | Medium   | 2min   | turso       |
| 6   | Add CBORCodec to `integration/` cross-module tests                                          | High     | 15min  | integration |
| 7   | Verify signing produces stable signatures with CBOR Pebble envelopes                        | High     | 10min  | signing     |
| 8   | Verify encryption wraps CBOR through CBOR Pebble envelope                                   | High     | 10min  | encryption  |
| 9   | Add CBOR to `projection/` builder integration test                                          | Medium   | 10min  | projection  |
| 10  | Replace hardcoded `"json"` strings in storage test mocks                                    | Medium   | 10min  | storage     |
| 11  | Add `nix run .#check-layers` to verify CBOR dep budgets                                     | Medium   | 5min   | infra       |
| 12  | Document CBOR in `docs/DOMAIN_LANGUAGE.md`                                                  | Medium   | 10min  | docs        |
| 13  | Add CBOR example to `catalog/` schema generation                                            | Low      | 15min  | catalog     |
| 14  | Add `APIstability` golden file check for CBORCodec + Pebble exports                         | Medium   | 10min  | cmd         |
| 15  | Write README section about CBOR adoption                                                    | Medium   | 30min  | docs        |
| 16  | Add `cbor:"-"` field omission test                                                          | Low      | 3min   | codec       |
| 17  | Golden test for CBOR time encoding                                                          | Low      | 5min   | codec       |
| 18  | Size comparison benchmark (CBOR vs JSON for []byte payloads)                                | Low      | 10min  | codec       |
| 19  | Add CBOR to `watermill/` protocol adapter test                                              | Low      | 10min  | watermill   |
| 20  | Evaluate CBOR for pebble envelope serialization (Phase 2 done — evaluate if Phase 3 needed) | Low      | 30min  | pebble      |
| 21  | Add CBOR to `memory/` round-trip test                                                       | Low      | 5min   | memory      |
| 22  | Clean up `docs/status/` — 20+ reports from one day                                          | Low      | 5min   | docs        |
| 23  | Document `DupMapKeyEnforcedAPF` behavior in cbor.go                                         | Low      | 3min   | codec       |
| 24  | Add CBOR to `go.work` sum verification                                                      | Low      | 3min   | infra       |
| 25  | Archive completed status reports to `docs/status/archive/`                                  | Low      | 5min   | docs        |

## g) Top #1 Question I Cannot Figure Out Myself

**Should we tag a v2.3.0 release now that Phase 1 + Phase 2 are done?**

The CBOR work spans two modules:

- `codec/` — CBORCodec (Phase 1, shipped in v2.2.0 cycle)
- `pebble/` — CBOR envelope (Phase 2, new since last tag)

Both are additive (no breaking changes). The question is whether to:

1. Tag now as v2.3.0 (CBOR feature release)
2. Wait for storage/ LSP fixes + integration tests first
3. Accumulate more polish items

This is a product/owner decision — I can't determine the release cadence.

---

## Session Statistics

| Metric               | Value                                                                                                                  |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Commits this session | 3 (`5f6219d9`, `2cd1f76e`, `353c7df7`)                                                                                 |
| Files changed        | 11 files, 825 lines added, 21 lines removed                                                                            |
| New test files       | 2 (`pebble/cbor_test.go`, `pebble/cbor_fuzz_test.go`)                                                                  |
| Total pebble tests   | 25 (13 existing + 12 new)                                                                                              |
| Fuzz tests           | 2 new (7 seed corpus entries)                                                                                          |
| Benchmarks           | 3 new (`BenchmarkSerializeEnvelope`, `BenchmarkDeserializeEnvelope`, `BenchmarkEventStore_LoadToTimestamp` comparison) |
| Lint issues          | 0 in pebble (was 5, all fixed)                                                                                         |
| Race detector        | All 25 tests pass with `-race`                                                                                         |
