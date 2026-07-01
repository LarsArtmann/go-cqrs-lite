# Codec & CBOR Integration — Comprehensive Status Update

> **Date:** 2026-07-01 09:34 (updated 2026-07-01)
> **Scope:** codec/, event/, schema/, stack/, integration/
> **Trigger:** User asked "How can we use codec/ and especially CBOR more/better?"

---

## Executive Summary

We promoted CBOR from "only pebble uses it" to "recommended default for read models, fully supported alongside JSON." Seven commits shipped. Two real bugs were found and fixed during self-review. The codec layer is now genuinely dual-codec — JSON and CBOR are both first-class, validated symmetrically, and proven to coexist in mixed streams.

All quick wins and medium-effort tasks from the original status report are now complete: stale docs fixed, CHANGELOG entry added, `schema.WithCodec` redesigned to accept `codec.Codec`, property-based roundtrip tests added, README reordered with CBOR first.

**Remaining for v4:** blind store encoding stamps, `event.New()` configurable default, composite encoding for encryption codec, transport layer codec injection.

---

## a) FULLY DONE ✅

| #   | What                                                           | Commit     | Evidence                                                                                                             |
| --- | -------------------------------------------------------------- | ---------- | -------------------------------------------------------------------------------------------------------------------- |
| 1   | **stack.DefaultCodec() returns CBORCodec**                     | `2bb6c0a3` | `stack/bundle.go` — read models default to CBOR                                                                      |
| 2   | **stack.WithDefaultCodec() option**                            | `2bb6c0a3` | `stack/options.go` — one-call override for deployers                                                                 |
| 3   | **schema.Validator auto-detects encoding**                     | `2bb6c0a3` | `schema/validator.go` — JSON and CBOR decoders pre-registered, `decoderFor()` picks per event                        |
| 4   | **schema.WithDecoder() per-encoding override**                 | `2bb6c0a3` | `schema/validator.go`                                                                                                |
| 5   | **Symmetric encoding validation**                              | `b531604c` | `event/codec.go:99-108` — JSON no longer bypasses codec check                                                        |
| 6   | **Mixed JSON+CBOR stream test**                                | `d2cbda5e` | `integration/event/mixed_codec_test.go` — both encodings in one store, each decoded correctly, cross-decode rejected |
| 7   | **Realistic benchmarks**                                       | `2bb6c0a3` | `codec/benchmark_test.go` — JSON 277B vs CBOR 224B (-19%) vs CBOR+toarray 158B (-43%)                                |
| 8   | **Example tests: toarray, BufferEncoder, streaming, Diagnose** | `2bb6c0a3` | `codec/example_test.go` — 6 new runnable examples                                                                    |
| 9   | **codec/README.md rewrite — CBOR first**                       | `2bb6c0a3` | Both codecs as first-class, CBOR recommended, Usage section leads with CBOR                                          |
| 10  | **codec/doc.go updated**                                       | `2bb6c0a3` | 4 codecs listed, "Choosing a Codec" section                                                                          |
| 11  | **AGENTS.md patterns updated**                                 | `2bb6c0a3` | toarray, streaming, BufferEncoder, WithDefaultCodec examples                                                         |
| 12  | **SKILL.md cheat sheet recommends CBOR**                       | `2bb6c0a3` | DecodePayload examples use CBORCodec                                                                                 |
| 13  | **v4-WISHLIST entry**                                          | `2bb6c0a3` | `docs/v4-WISHLIST.md` — universal CBOR default tracked                                                               |
| 14  | **kv/typed_options.go doc**                                    | `5f50a3d7` | Documents stack.DefaultCodec override                                                                                |
| 15  | **Misleading json: tags removed (all CBOR examples)**          | `825925db` | `codec/example_test.go` — no json: tags remain on CBOR structs                                                       |
| 16  | **Stale DecodePayload/DecodePayloads docs fixed**              | (this PR)  | `event/codec.go` — removed outdated "when both are non-empty and differ" and "validates once for batch"              |
| 17  | **schema.WithCodec accepts codec.Codec**                       | (this PR)  | `schema/validator.go` — type-safe codec interface, old raw-fn API preserved as deprecated `WithDecodeFunc`           |
| 18  | **Property-based roundtrip tests**                             | (this PR)  | `codec/property_test.go` — 4 rapid tests: JSON/CBOR/CBORCompact roundtrip + CBOR determinism                         |
| 19  | **CHANGELOG.md v3.5.0 entry**                                  | (this PR)  | Full entry covering all codec/CBOR work                                                                              |
| 20  | **API stability golden file updated**                          | (this PR)  | `docs/api_surface.txt` — 1791 exports (was 1736)                                                                     |

### Test Coverage (current)

| Module  | Coverage                                                 |
| ------- | -------------------------------------------------------- |
| codec/  | ~90% (added property tests)                              |
| event/  | 91.9%                                                    |
| schema/ | ~91% (added WithCodec(CBORCodec) + WithDecodeFunc tests) |
| stack/  | 57.6% (accessors only; contract suite is separate)       |

### Benchmark Results (realistic order payload)

| Codec                | Size             | Encode ns/op  | Decode ns/op   | Allocs (decode) |
| -------------------- | ---------------- | ------------- | -------------- | --------------- |
| JSON                 | 277 bytes        | 745 ns        | 4695 ns        | 16              |
| CBOR canonical       | 224 bytes (-19%) | 559 ns (-25%) | 1597 ns (-66%) | 9               |
| CBOR compact+toarray | 158 bytes (-43%) | 488 ns (-34%) | 1305 ns (-72%) | 9               |

---

## b) PARTIALLY DONE 🟡

### CBOR as default — stack level only, not module level

**What's done:** `stack.DefaultCodec()` returns CBORCodec. ReadModel and NewMaterialize accessors use it when caller passes nil codec.

**What's NOT done:** The four "blind" stores still default to JSONCodec:

- `event/event_new.go:42` — `event.New()` defaults to JSONCodec
- `kv/typed_store.go:37` — `kv.NewTypedStore` defaults to JSONCodec
- `snapshot/typed.go:72` — snapshot typed store defaults to JSONCodec
- `command/typed_store.go:45` — command typed store defaults to JSONCodec
- `query/typed.go:34` — query typed store defaults to JSONCodec

**Why not done:** All five are blind stores (no encoding stamped on data). Changing the default would silently break consumers with existing JSON data. Tracked in `docs/v4-WISHLIST.md` as item #8.

---

## c) NOT STARTED ❌

| #   | Item                                   | Impact | Why                                                                                  |
| --- | -------------------------------------- | ------ | ------------------------------------------------------------------------------------ |
| 1   | **event.New() encoding-aware default** | HIGH   | Currently hardcodes JSONCodec; should use a package-level default variable or option |
| 2   | **Blind store encoding tagging**       | HIGH   | kv/snapshot/command/query stores don't stamp encoding — fundamental gap              |
| 3   | **Transport layer codec awareness**    | MED    | gRPC (query_server.go, client.go) hardcodes json.Marshal — no codec injection        |

---

## d) TOTALLY FUCKED UP 💥

### Nothing is catastrophically broken. All 55+ test suites pass, lint is clean.

**However, one thing is deeply misleading:**

### The "CBOR is the default" claim is only half-true

We say "CBOR is recommended" and changed `stack.DefaultCodec()`, but:

- Events are still created as JSON by default (`event.New()`)
- Read models created WITHOUT the stack (direct `kv.NewTypedStore()`) still get JSON
- Snapshots, command stores, query stores all still default to JSON

A consumer reading the README would reasonably expect CBOR everywhere. They get it ONLY for `stack.ReadModel()` and `stack.NewMaterialize()` with nil codec. This split-brain is documented but easy to miss.

---

## e) WHAT WE SHOULD IMPROVE 🎯

### Architecture / Type Model

1. ~~**`schema.WithCodec` should take `codec.Codec`**~~ ✅ DONE — `WithCodec(codec.Codec)` shipped, old API preserved as deprecated `WithDecodeFunc`.

2. **Blind stores need encoding stamps.** The fundamental issue: kv/snapshot/command/query stores serialize values as raw bytes with no format tag. This makes codec migration impossible without data loss. Pebble solved this with an `Encoding` field in its CBOR envelope. SQL stores store encoding in a column. The typed stores are blind by design — they'd need either an envelope type or a codec stamp in the key/value structure.

3. **`event.New()` default codec should be configurable** at minimum via a package-level `var DefaultCodec codec.Codec = JSONCodec{}`. This lets consumers flip to CBOR without v4. Currently the only way is `event.WithCodec(cborCodec)` on every `event.New()` call.

4. **`encryption.Codec` reports "encrypted"** — losing the inner codec's encoding. If inner is CBOR, the event is stamped "encrypted" not "cbor". A mixed stream with unencrypted CBOR events and encrypted CBOR events can't be decoded with one codec path. The encoding should be composite: `"cbor+encrypted"` or the codec should expose both layers.

### Library Hygiene

5. ~~**`codec/example_test.go` still has 3 `json:` tags`**~~ ✅ DONE — all removed.

6. ~~**CHANGELOG.md needs a v3.5.0 entry`**~~ ✅ DONE — full entry covering all codec work.

7. ~~**Doc comments in `event/codec.go` are stale`**~~ ✅ DONE — both fixed.

### Ecosystem Leverage

8. ~~**Property-based codec testing with `rapid`**~~ ✅ DONE — 4 roundtrip + determinism tests.

9. **CBOR diagnostic output** — `codec.Diagnose()` wraps `cbor.Diagnose()`. Good. Could also add `codec.DiagnoseJSON()` for side-by-side comparison.

---

## f) Top Things to Do Next

### Quick Wins (under 15 min each)

| #   | Task                                        | Impact | Work | Risk |
| --- | ------------------------------------------- | ------ | ---- | ---- |
| 1   | ~~Fix stale `DecodePayload` doc~~           | ✅     | DONE |      |
| 2   | ~~Fix stale `DecodePayloads` doc~~          | ✅     | DONE |      |
| 3   | ~~Add CHANGELOG.md v3.5.0 entry~~           | ✅     | DONE |      |
| 4   | ~~Remove last 3 `json:` tags from CBOR ex~~ | ✅     | DONE |      |
| 5   | ~~Reorder README Usage section~~            | ✅     | DONE |      |

### Medium Effort (30-60 min each)

| #   | Task                                                                | Impact | Work  | Risk       |
| --- | ------------------------------------------------------------------- | ------ | ----- | ---------- |
| 6   | ~~Change `schema.WithCodec` to accept `codec.Codec`~~               | ✅     | DONE  |            |
| 7   | ~~Add `schema.WithCodec` deprecation path (keep old func)~~         | ✅     | DONE  |            |
| 8   | ~~Add property-based roundtrip test with `rapid`~~                  | ✅     | DONE  |            |
| 9   | Add `codec.Size(v any) (jsonSize, cborSize int)` helper             | LOW    | 15min | none       |
| 10  | Add CBOR codec to example/deployer-first                            | MED    | 20min | none       |
| 11  | Document the stack-vs-store default asymmetry in AGENTS.md          | HIGH   | 10min | none       |
| 12  | Add `event.DefaultCodec` package var (mutable for CBOR adoption)    | HIGH   | 15min | behavioral |
| 13  | Add stack-level `WithEventCodec()` option for event.New             | HIGH   | 20min | none       |
| 14  | Test: verify schema.Validator rejects encrypted encoding gracefully | MED    | 15min | none       |

### Larger Effort (1-4 hours)

| #   | Task                                                                   | Impact | Work | Risk          |
| --- | ---------------------------------------------------------------------- | ------ | ---- | ------------- |
| 15  | Add codec injection to gRPC transport                                  | MED    | 1h   | API change    |
| 16  | Design encoding stamp for blind stores (envelope or key prefix)        | HIGH   | 2h   | schema change |
| 17  | Add `codec.AutoDetect(data []byte) Encoding` — sniff format from bytes | MED    | 30m  | none          |
| 18  | Add migration tool: scan JSON events, re-encode as CBOR                | MED    | 2h   | data change   |
| 19  | Add `cbor:",keyasint"` example to codec/examples                       | LOW    | 30m  | none          |

### v4 Preparation

| #   | Task                                                        | Impact | Work | Risk          |
| --- | ----------------------------------------------------------- | ------ | ---- | ------------- |
| 20  | Flip blind store defaults to CBOR in a feature branch       | HIGH   | 2h   | v4 breaking   |
| 21  | Design composite encoding for encryption ("cbor+encrypted") | HIGH   | 2h   | schema change |
| 22  | Write migration guide: JSON→CBOR for existing consumers     | HIGH   | 1h   | docs only     |

---

## g) Top #1 Question I Cannot Figure Out Myself

**How should the encryption codec report its encoding?**

Currently `encryption.Codec.Encoding()` returns `"encrypted"`, which discards the inner codec's format (JSON or CBOR). With symmetric validation, an event created via:

```go
event.New(..., event.WithCodec(encryption.NewCodec(codec.CBORCodec{}, enc)))
```

gets stamped `encoding="encrypted"`. On decode, you must pass the exact same `encryptingCodec` — it works, but you lose the information that the plaintext is CBOR. This matters for:

1. **Debugging:** You can't tell from the encoding field whether the plaintext was JSON or CBOR
2. **Mixed streams:** Unencrypted CBOR events stamp `"cbor"`, encrypted CBOR events stamp `"encrypted"` — no common decode path
3. **Migration:** If you want to drop encryption later, you don't know what codec to use for the plaintext

**Options I see but can't decide between:**

- **A:** Composite encoding string: `"cbor+encrypted"` or `"encrypted:cbor"` — expressive but changes the encoding type contract
- **B:** Keep `"encrypted"` and add a separate `InnerEncoding()` method to the codec — doesn't help the event's stamped encoding
- **C:** Don't use `encryption.Codec` for event creation at all — use the EncryptMiddleware/DecryptMiddleware path instead, which preserves the original encoding stamp. This is already the documented recommended path in AGENTS.md.

**Option C** seems cleanest (middleware preserves encoding, codec wrapper is for non-event use cases), but it means the `encryption.NewCodec` function is a footgun when used with `event.WithCodec`. Should we deprecate that combination? Document it? I don't know the right call.
