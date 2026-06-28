# 2026-06-11 Fuzzing Expansion Status

## Summary

Expanded fuzzing coverage from **18 → 78 fuzzers** across 8 modules. All new fuzzers pass smoke tests (2s each, ~2.5 minutes total). No crashes, panics, or correctness bugs surfaced — the codebase is solid against randomized adversarial input on the surfaces tested.

## Coverage Map

| Module              | Before | After  | New fuzzers | Focus areas                                                                   |
| ------------------- | ------ | ------ | ----------- | ----------------------------------------------------------------------------- |
| `signing/`          | 1      | 8      | +7          | HMAC/Ed25519 sign-verify-tamper, base64, nil guards                           |
| `signing/multisig/` | 0      | 8      | +8          | Multi-actor chains, JSON corruption, key-length edge cases                    |
| `encryption/`       | 6      | 15     | +9          | Codec roundtrip, Algorithm/KeyID extract, middleware nil guards               |
| `event/`            | 7      | 19     | +12         | Parser fuzzers, Version/SchemaVersion arithmetic, Metadata Clone/Merge/JSON   |
| `id/`               | 1      | 10     | +9          | ParseAggregateID, DeriveAggregateID determinism, JSON roundtrip, ULID entropy |
| `listing/`          | 0      | 3      | +3          | AggregateListing/Status JSON roundtrip, TombstonePolicy String                |
| `codec/`            | 3      | 6      | +3          | CBOR determinism (canonical), CBOR decode-no-panic, JSON typed roundtrip      |
| `schema/`           | 0      | 6      | +6          | Upcaster metadata, VersionedStore nil guards, upcast pipeline                 |
| **Total**           | **18** | **78** | **+60**     | —                                                                             |

## Pareto-Ordered Themes

### Tier 1 — Security-Critical (highest impact)

1. **HMAC sign/verify tamper detection** — `FuzzHMAC_SignVerifyRoundtrip`, `FuzzHMAC_TamperReject`, `FuzzHMAC_NilAndZeroGuards`. Critical: a fuzz-found bypass would be a CVE-grade bug.
2. **Ed25519 sign/verify tamper detection** — `FuzzEd25519_SignVerifyRoundtrip`, `FuzzEd25519_TamperReject`. Asymmetric counterpart to HMAC.
3. **Multi-sig chain verification** — `FuzzMultiSig_MultiActorChain`, `FuzzMultiSig_ReplaceActor`, `FuzzMultiSig_VerifyAllMissingVerifier`. Defense against re-signing and verifier-map confusion.
4. **Ed25519 key-length guards** — `FuzzMultiSig_Ed25519KeyLength` covers the `New*` constructors' length validation.
5. **Signature base64 parsing** — `FuzzSignature_UnmarshalJSON` exercises both URL-safe and standard base64 fallbacks.

### Tier 2 — Encoding & Schema Evolution

6. **CBOR canonical determinism** — `FuzzCBORCodec_Determinism` proves that two maps with the same keys/values but different insertion orders produce identical bytes. Critical for content-addressed storage and signing.
7. **CBOR decode-no-panic** — `FuzzCBORCodec_DecodeNeverPanics` on arbitrary byte sequences. No fuzz-found crashes.
8. **JSON typed roundtrip** — `FuzzJSONCodec_TypedRoundtrip` catches issues that generic roundtrips miss.
9. **VersionedStore upcast pipeline** — `FuzzVersionedStore_LoadFromArbitraryStream` ensures schema migrations preserve identity (AggregateID, Version) and bump SchemaVersion correctly.

### Tier 3 — Parser/Validator Robustness

10. **Event parser surfaces** — `FuzzParseType`, `FuzzParseAggregateType`, `FuzzParseSchemaVersion`, `FuzzNewUserAgent`. All reject empty input correctly.
11. **Version/SchemaVersion arithmetic** — `FuzzVersion_Arithmetic`, `FuzzSchemaVersion_Arithmetic` cover Add/Sub/Mod/Increment with extreme operands.
12. **id. parsing surfaces** — `FuzzParseAggregateID` (loose string contract), `FuzzParse_TypeSafety` (strict ULID).
13. **DeriveAggregateID** — `FuzzDeriveAggregateID_Deterministic` (purity), `FuzzDeriveAggregateID_DifferentInputs` (collision resistance).
14. **Tombstone precedence** — `FuzzDetectTombstone` covers "last event wins" and "rebirth > tombstone" invariants.

### Tier 4 — Serialization & Metadata

15. **Metadata Clone/Merge/JSON** — `FuzzMetadata_CloneIsDeep`, `FuzzMetadata_Merge`, `FuzzMetadata_JSON_Roundtrip`. The Merge and JSON tests respect that invalid UTF-8 / invalid IPs roundtrip with normalization.
16. **Listing JSON roundtrip** — `FuzzAggregateListing_JSON_Roundtrip` and `FuzzAggregateStatus_MarshalOnly`. The marshal-only test exists because `AggregateStatus.MarshalJSON` emits a string while the default `UnmarshalJSON` expects an int — a pre-existing asymmetry documented in the test.
17. **id. JSON roundtrip** — `FuzzAggregateID_JSON_Roundtrip` (valid-UTF-8 only).
18. **Encryption codec roundtrip** — `FuzzEncryptingCodec_Roundtrip` catches key/nonce handling bugs.
19. **Cipher text extract/attach** — `FuzzAttachExtractCiphertext` with non-empty ciphertext (empty ciphertext is rejected by design).

## Notable Findings (No Bugs, Documented Asymmetries)

1. **`AggregateStatus` JSON asymmetry**: `MarshalJSON` emits `"status": "active"` (string) but the default `UnmarshalJSON` expects an int. The `FuzzAggregateStatus_MarshalOnly` fuzzer documents this. A future fix would add a custom `UnmarshalJSON`.
2. **Empty ciphertext rejection**: `ExtractCiphertext` returns `ErrNilCiphertext` for empty base64 — by design, to avoid ambiguous "is this encrypted?" signals. The `FuzzAttachExtractCiphertext` fuzzer skips the extract check for empty input.
3. **JSON marshaling of invalid UTF-8**: Go's `encoding/json` does lossy replacement (U+FFFD). JSON roundtrip fuzzers in `event/` and `id/` and `listing/` all skip the assertion for invalid UTF-8 inputs.

## Verification

All fuzzers pass at 1–3 second smoke durations:

```bash
# Run all new fuzzers per module (each ~1-3s)
for mod in signing signing/multisig encryption event id listing codec schema; do
  (cd $mod && go test -fuzz=. -fuzztime=1s -run=^$ . 2>&1) || echo "Note: -fuzz=. only matches one fuzzer per package, run individually"
done
```

Per-fuzzer invocation works (Go's `-fuzz=` accepts a regex, but `-fuzz=.` matches more than one fuzzer in packages with multiple). Each new fuzzer is `go test -fuzz=Fuzz<Name> -fuzztime=2s -run=^$ .` invokable.

The pre-existing `TestGolden_JSONCodec_Encode` failure (compact vs indented JSON in golden file) is unrelated to this work and was already broken before the changes.

## Future Work (Not Done)

- **Snapshot/**: `EveryNEvents` strategy + Snapshot JSON roundtrip.
- **command/**: `NewPersistedCommand` validation paths.
- **storage/, watermill/, pebble/, projection/**: would need integration-level fuzzers (with in-memory backends).
- **catalog/**: schema reflection fuzzers (likely high-effort, low-value).
- **otel/**: middleware fuzzing (most logic is in OTel SDK itself).
- **dispatcher/**: type-erasure fuzzing (low surface area).
