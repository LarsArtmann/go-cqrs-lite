# CBOR Phase 2 — Pebble Envelope Migration Status

**Date:** 2026-06-11 12:59 UTC
**Branch:** master
**HEAD:** 3641cffc

---

## a) FULLY DONE

### Phase 1: CBORCodec (shipped)

| Component | Detail |
|-----------|--------|
| `codec/cbor.go` | CBORCodec with canonical EncMode (RFC 7049) + DecMode (DupMapKeyEnforcedAPF) |
| `codec/codec.go` | `EncodingCBOR = "cbor"` constant |
| `codec/doc.go` | Lists all 3 codecs (JSON, CBOR, Raw) |
| `codec/go.mod` | Added `fxamacker/cbor/v2 v2.9.2` + `x448/float16 v0.8.4` |
| `example/cbor-codec/` | Minimal runnable example |
| `docs/adr/0015-cbor-codec.md` | Full ADR |
| Tests | 15 unit, 4 fuzz, 4 benchmarks, 1 golden, 1 example, 1 e2e (event.New + DecodePayload) |
| Benchmarks | 17% faster encode (2 vs 6 allocs), 38% faster decode (9 vs 12 allocs) |
| Integration | `event.WithCodec(codec.CBORCodec{})` works end-to-end |
| Signing | Deterministic encoding verified via `TestCBORCodec_SigningDeterminism` |
| Encryption | `encryption.NewCodec(codec.CBORCodec{}, enc)` works — codec-agnostic wrapper |

### Documentation updates this session

| File | Change |
|------|--------|
| `docs/brainstorming/binary-wire-codec.html` | Updated with Phase 1 completion: status banner, measured benchmarks, DONE markers on phases, "What's Done" section, file structure updates |
| `docs/planning/2026-06-11_PHASE2-PEBBLE-CBOR-ENVELOPE.md` | Phase 2 plan created |

## b) PARTIALLY DONE

Nothing. Phase 1 is fully complete. Phase 2 plan is written but no code started.

## c) NOT STARTED

### Phase 2: Pebble CBOR Envelope Migration

Full 37-task plan in `docs/planning/2026-06-11_PHASE2-PEBBLE-CBOR-ENVELOPE.md`.

**Scope:** Replace `json.Marshal`/`json.Unmarshal` in `pebble/serialization.go` with CBOR encoding for the event envelope. This eliminates the 33% base64 tax on `[]byte` payload fields.

**Key insight:** Only `pebble/serialization.go` changes behavior (2 lines: `json.Marshal` → `cborEncMode.Marshal`, `json.Unmarshal` → format-sniff + decode). Everything else is tests, benchmarks, and docs.

**Migration strategy:** Format sniff — CBOR maps start with byte `0xa0`–`0xbf` (major type 5). Old JSON envelopes start with `{` (0x7b). `deserializeEvent` tries CBOR first, falls back to JSON for backward compat.

## d) TOTALLY FUCKED UP

Nothing. Clean codebase, clean working tree.

## e) WHAT WE SHOULD IMPROVE

| # | Issue | Module | Effort |
|---|-------|--------|--------|
| 1 | `storage/` has 58+ LSP errors (Dialect, DB, CheckClosed) — pre-existing | storage | 30min |
| 2 | Catalog `unconvert` lint issue (pre-existing) | catalog | 2min |
| 3 | Compiled binaries in `example/` dirs (buildflow ERROR) | examples | 5min |
| 4 | `turso/doc.go` missing package comment | turso | 2min |
| 5 | Hardcoded `"json"` strings in storage test mocks | storage | 10min |

## f) Top #25 Things To Get Done Next

| # | Task | Impact | Effort | Module |
|---|------|--------|--------|--------|
| 1 | Phase 2: Record baseline Pebble benchmarks (before CBOR) | Critical | 10min | pebble |
| 2 | Phase 2: Move fxamacker/cbor from indirect → direct in pebble/go.mod | Critical | 3min | pebble |
| 3 | Phase 2: Add pebbleEncMode/pebbleDecMode IIFE vars in serialization.go | Critical | 5min | pebble |
| 4 | Phase 2: Replace json.Marshal with cborEncMode.Marshal in serializeEvent | Critical | 2min | pebble |
| 5 | Phase 2: Add isCBOR format sniff helper | Critical | 5min | pebble |
| 6 | Phase 2: Rewrite deserializeEvent with format sniff + CBOR/JSON fallback | Critical | 8min | pebble |
| 7 | Phase 2: Run all pebble tests to verify zero regressions | Critical | 5min | pebble |
| 8 | Phase 2: Add CBOR format verification test | High | 8min | pebble |
| 9 | Phase 2: Add JSON backward compat test (hand-crafted JSON envelope) | Critical | 10min | pebble |
| 10 | Phase 2: Add CBOR round-trip fidelity test | High | 8min | pebble |
| 11 | Phase 2: Add no-base64 verification test | High | 8min | pebble |
| 12 | Phase 2: Add format sniff unit tests | Medium | 8min | pebble |
| 13 | Phase 2: Add deterministic envelope encoding test | Medium | 8min | pebble |
| 14 | Phase 2: Add persistence test with CBOR envelope | High | 10min | pebble |
| 15 | Phase 2: Add binary payload round-trip test | High | 10min | pebble |
| 16 | Phase 2: Update golden test + fixtures | Medium | 10min | pebble |
| 17 | Phase 2: Add envelope encode/decode benchmarks (JSON vs CBOR) | High | 10min | pebble |
| 18 | Phase 2: Re-run LoadToTimestamp benchmark against baseline | High | 10min | pebble |
| 19 | Phase 2: Add disk size comparison benchmark | Medium | 10min | pebble |
| 20 | Phase 2: Add FuzzDeserializeEvent (crash safety) | High | 10min | pebble |
| 21 | Phase 2: Add FuzzSerializeDeserializeRoundtrip | Medium | 10min | pebble |
| 22 | Phase 2: Update pebble/doc.go + AGENTS.md + ADR | Medium | 10min | docs |
| 23 | Phase 2: nix fmt + lint + full test suite | High | 10min | infra |
| 24 | Phase 2: Verify signing stability with CBOR Pebble envelope | High | 10min | signing |
| 25 | Phase 2: Verify encryption + projection integration | Medium | 10min | integration |

## g) Top #1 Question I Cannot Figure Out Myself

**Should the Pebble CBOR migration be a breaking change (new Pebble store, fresh data) or must it read old JSON envelopes?**

The format-sniff approach (try CBOR first, fall back to JSON) gives backward compat with zero migration cost. But it adds a small decode branch to every read. For a library, backward compat seems right. But the consumer may not have any persisted Pebble data yet — the format is internal.

**Recommendation:** Implement format sniff. It costs ~3 lines and one `if` on the read path. If performance ever matters, remove the JSON fallback in a future major version.
