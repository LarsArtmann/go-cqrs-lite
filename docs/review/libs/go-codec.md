# go-codec — Utilization Audit · go-cqrs-lite

> Audited 2026-11-22 · Library `github.com/larsartmann/go-codec` @ 98bb1bc (latest local tag v0.3.3) · Consumer go-cqrs-lite @ feec9bc74 · Group G1

**Verdict:** UNDERUSED-MODERATE
**Score:** 8/17 capabilities fully leveraged = 47% (low)

The serialization spine is fully adopted: the `Codec` contract rides through
event, command, query, kv, snapshot, signing, encryption, storage, watermill,
and both core generations, and the ADR-0044 blind-store envelope is used at
every typed-store boundary. The misses are the performance and diagnostics
periphery the library advertises as first-class: the zero-allocation
`BufferEncoder` path, the buffer pool, the `DeterministicCodec` compile-time
guard for signing, streaming codecs, transcoding, and size diagnostics —
none referenced once. The repo also hand-rolls deterministic JSON at 36 call
sites because the library never exported that capability.

## 1. What the library is

The serialization contract at the center of the event-sourcing stack:
`Codec` over canonical `CBORCodec` (recommended default), `JSONCodec`
(dual-build v1/v2), strict `CBORCompactCodec`, and `RawCodec`; encoding
stamps + `ForEncoding`; self-describing payload envelopes (`WrapEncode` /
`UnwrapDecode` / `DecodeEnvelopeOrLegacy`); `AutoDetect`/`AutoDetectDebug`
format inference; zero-alloc `BufferEncoder` + `sync.Pool` helpers; the
`DeterministicCodec` marker for signing safety; COSE (Sign1/Encrypt0)
structure helpers; base64-JSON wire helpers; streaming encoders/decoders;
`TranscodeToJSON`; `Diagnose` (CBOR diagnostic notation); `Size`; and an
`ObservableCodec` metrics decorator. Extracted from this repo by ADR-0128
(2026-08-14), which deleted the in-repo `codec/v4` shim and forbids
re-implementing it here.

Sources: `go-codec/README.md:37-45` (capability diagram),
`go-codec/codec.go:56-89` (Codec/BufferEncoder/DeterministicCodec).

## 2. Usage census in go-cqrs-lite (evidence)

- **86 go.mod files pin v0.3.3** (31 direct — `event/v4/go.mod:8`,
  `kv/v4/go.mod:6`, `query/v4/go.mod:6`, `signing/v4/go.mod:7`,
  `encryption/v4/go.mod:6`, `metaengine/v4/go.mod:8`, `watermill/v4/go.mod:10`,
  `storage/v4/go.mod:10`, `snapshot/v4/go.mod:7`, `system/v4/go.mod:10`,
  `storage/bbolt/v4/go.mod:6`, `cmd/cqrs-bench/v4/go.mod:7`, … — 55 indirect).
- Prod symbol census: `codec.Codec` 77, `CBORCodec` 43, `JSONCodec` 33,
  `Encoding`/`EncodingJSON`/`EncodingCBOR` 56, `ForEncoding` 12,
  `EncodeJSONErr` 22, plus the whole keycodec family (`MapKey`, `VectorKey`,
  `StreamKey`, `JournalKey`, `Escape`, `Unescape`, `MapKeyV2`, …) inside
  `metaengine/v4/keycodec`.
- **ADR-0044 envelope adopted at every typed store** (v4 and v5):
  wrap on write — `command/v4/typed_store.go:61`, `query/v4/typed.go:47`,
  `kv/v4/typed_store.go:89`, `snapshot/v4/typed.go:91`,
  `scheduling/sqlstore/v4/store.go:118`, and the v5 twins
  (`core/v5/command/typed_store.go:62`, `core/v5/query/typed.go:48`,
  `core/v5/kv/typed_store.go:90`); legacy-fallback decode on read —
  `codec.DecodeEnvelopeOrLegacy` at `command/v4/typed_store.go:171`,
  `query/v4/typed.go:104`, `kv/v4/typed_store.go:62`,
  `snapshot/v4/typed.go:180`, and the v5 twins; raw envelope strip at
  `scheduling/sqlstore/v4/envelope.go:90` (`codec.UnwrapDecode`).
- Canonical CBOR modes are delegated, not duplicated:
  `storage/pebble/v4/serialization.go:18,36` and
  `storage/bbolt/v4/serialization.go:15,19` marshal through
  `codec.CBOREncMode()/CBORDecMode()` so all modules share one mode;
  `snapshot/v4/wire.go:62` decodes snapshot wire the same way. CBOR is the
  documented stack default (`kv/v4/typed_options.go:24`).
- COSE and base64 helpers carry the crypto layer:
  `signing/v4/cose_sign1.go:30-69` (PrepareCOSESetup/SigStructure/
  COSESign1), `signing/v4/cose.go:89,120,158` (algorithm constants),
  `signing/v4/signature.go:68,78` (MarshalBase64JSONWithModule /
  AssignBase64JSON).
- Mixed-format inference: `metaengine/v4/encoded.go:112` falls back to
  `codec.AutoDetect(payload)` when an engine payload carries no encoding
  stamp.

## 3. Capability inventory

| #  | Capability                                                               | Grade    | Evidence / reason                                                                                                                                                                                                                                                   |
| -- | ------------------------------------------------------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1  | Codec contract + `CBORCodec`/`JSONCodec`                                 | **Full** | CBOR default (`kv/v4/typed_options.go:24`), 77 `codec.Codec` references                                                                                                                                                                                             |
| 2  | `CBORCompactCodec` (CoreDet + unknown-field rejection)                   | Partial  | selectable via `kv/v4/typed_options.go:26` and benched in `benchkit/v4/runner.go:82`, but no store or default uses it — the schema-drift guard ships unused                                                                                                         |
| 3  | `RawCodec` passthrough                                                   | Missed   | 0 constructions; `metaengine/v4/encoded.go` re-wraps raw payloads via AutoDetect + custom `rawJSON` instead                                                                                                                                                         |
| 4  | Encoding stamps + `ForEncoding` + `EncodeJSONErr`                        | **Full** | 56 stamp references, 12 `ForEncoding`, 22 `EncodeJSONErr`; `record/v4/encoding.go:25` mirrors the stamp names                                                                                                                                                       |
| 5  | ADR-0044 envelope (`WrapEncode`/`UnwrapDecode`/`DecodeEnvelopeOrLegacy`) | **Full** | 10 wrap sites, 11 legacy-tolerant decode sites across v4+v5 typed stores (§2)                                                                                                                                                                                       |
| 6  | `AutoDetect` / `AutoDetectDebug`                                         | Partial  | one production use of `AutoDetect` (`metaengine/v4/encoded.go:112`); the explainable `AutoDetectDebug` variant is never used for triage                                                                                                                             |
| 7  | `BufferEncoder` zero-alloc encode                                        | Missed   | 0 `EncodeToBuffer` call sites despite `go-codec/README.md:40` billing it as the zero-allocation hot path                                                                                                                                                            |
| 8  | Buffer pool (`GetBuffer`/`PutBuffer`/`EncodePooled`)                     | Missed   | 0 call sites; every encode allocates a fresh `[]byte`                                                                                                                                                                                                               |
| 9  | `DeterministicCodec` compile-time guard                                  | Missed   | `SignMiddleware` signs `canonicalPayload(evt)` — the bytes of whatever codec built the event (`signing/v4/middleware.go:62`, `signing/v4/payload.go:16`) — with no determinism assertion; tests sign `JSONCodec` events directly (`signing/v4/event_test.go:39,88`) |
| 10 | Canonical CBOR modes (`CBOREncMode`/`CBORDecMode`)                       | **Full** | `storage/pebble/v4/serialization.go:18,36`, `storage/bbolt/v4/serialization.go:15,19`, `snapshot/v4/wire.go:62`                                                                                                                                                     |
| 11 | COSE structure helpers                                                   | **Full** | `signing/v4/cose_sign1.go:30,42,60`, `encryption/v4/cose.go`                                                                                                                                                                                                        |
| 12 | Base64-JSON wire helpers                                                 | **Full** | `signing/v4/signature.go:68,78`                                                                                                                                                                                                                                     |
| 13 | Keycodec helpers (escape/key builders, V2)                               | **Full** | `metaengine/v4/keycodec/` (`v2.go`, `escape.go`), prefixes across vector/stream/journal/multimap keys                                                                                                                                                               |
| 14 | Streaming encoders/decoders (`NewCBOREncoder`, …)                        | Missed   | 0 call sites; large-payload paths encode into memory wholesale                                                                                                                                                                                                      |
| 15 | `TranscodeToJSON`                                                        | Missed   | 0 call sites; `metaengine/v4/encoded.go` hand-rolls a `rawJSON` passthrough for JSON folds instead of transcoding CBOR→JSON for consumers that need JSON                                                                                                            |
| 16 | `Diagnose` (CBOR diagnostic notation)                                    | Missed   | 0 call sites; corruption triage (`WrapCorruption`, 151 sites) prints raw bytes, not the diagnostic form                                                                                                                                                             |
| 17 | `ObservableCodec` metrics                                                | N/A      | observability seam is engine-level (`metaengine/otelobserver`) and cqrsotel by doctrine; a codec-level decorator would duplicate that layer                                                                                                                         |
| 18 | `Size` payload diagnostics                                               | Missed   | 0 call sites; `benchkit` hand-measures encoded sizes when sizing stories                                                                                                                                                                                            |

Score derivation: rows N=18, N/A row 17 → **A=17**. Full=7 (rows 1, 4, 5,
10, 11, 12, 13), Partial=2 (rows 2, 6 → 1.0), Missed rows 3, 7, 8, 9, 14,
15, 16, 18 → 0. Score = round(100 × 8.0 / 17) = **47% → low band**.

## 4. Gap analysis

1. **The signing path trusts whatever codec built the event.**
   `SignMiddleware(signer)` signs `canonicalPayload(evt)` — the payload
   bytes produced by whatever `Codec` constructed the event
   (`signing/v4/middleware.go:62`, `signing/v4/payload.go:16`) — and the
   suite exercises signing `JSONCodec` events (`signing/v4/event_test.go:39,88`).
   Under the default v1 JSON build, map key ordering is non-deterministic,
   so a JSON payload containing a map can sign one byte sequence and verify
   against a re-encoded other. go-codec exists to make that a compile error
   at the sign/verify seam:

   ```go
   // skip-validate
   func SignPayload(payload any, c codec.DeterministicCodec, s Signer) (Signature, error) {
       data, err := c.Encode(payload)
       if err != nil {
           return Signature{}, err
       }
       return s.Sign(data)
   }
   ```

   Today only CBORCodec/CBORCompactCodec (and opt-in v2 JSONCodec) satisfy
   the interface (`go-codec/codec.go:80-89`), which is exactly the guard the
   README's signing section tells consumers to install.

2. **Deterministic JSON is hand-rolled 36 times.** `json.Marshal(v,
   json.Deterministic(true))` appears at 36 non-test sites —
   `system/v4/adapter_query_serial.go:24,38`,
   `system/v4/checkpoint_engine.go:116`, `catalog/v4/schema/reflect.go:69,80`,
   `event/v4/metadata_json.go:34`, `signing/v4/multisig/extract.go:111`,
   `catalog/v4/message_config.go:192`, and more. go-codec has the logic
   (`jsonMarshalDet`, `go-codec/json_compat_v2.go:16-19`) but keeps it
   unexported and offers no `JSONDeterministic` option or codec. The honest
   fix is upstream: export the deterministic JSON mode, then delete 36 local
   variants. Until then this is sanctioned duplication — but it is the
   single largest codec-shaped clone in the repo.

3. **Zero-allocation path unused on the hot path.** Event append/publish and
   kv writes are the fleet's throughput ceiling; all of them call
   `Codec.Encode` (allocating) rather than `EncodeToBuffer` + the pool,
   which `go-codec` benchmarks exist to serve. A `benchkit` A/B run would
   size the win before touching the stores.

4. **`CBORCompactCodec`'s drift guard ships as a dead option.** It is
   offered in `kv/v4/typed_options.go:26` and measured in
   `benchkit/v4/runner.go:82` but never selected by any preset, so
   unknown-field schema drift in stored payloads is detected nowhere.

## 5. Version currency

- Pinned **v0.3.3 in all 86 go.mod files** — identical to the latest local
  tag v0.3.3 (2026-10-29). **ON LATEST**, zero drift.
- v0.3.3's headline fix (CBOR decode accepting text strings for
  `encoding.TextUnmarshaler` types, `go-codec/CHANGELOG.md` [v0.3.3]) lands
  directly in this consumer's lane: branded-ID journal replay across
  pre-branding wire shapes — the pin already protects the fleet.
- Stale sibling note: `go-codec/TODO_LIST.md` item #24 still asks to "sweep
  the remaining ~57 modules from v0.3.1 → v0.3.2" — the consumer finished
  that sweep (and one more minor) long ago; the item is dead weight in the
  library's list.
- [Unreleased] in `go-codec/CHANGELOG.md` is CI-lint-only; no pending API
  the consumer needs to chase.

## 6. The fleet argument

"I have MANY projects and limited time; if every project re-implements or
reintegrates everything I will only fix bugs all day long." The 36
hand-rolled deterministic-JSON sites are that sentence in code: every one is
a place where a future JSON-engine subtlety (ordering, escapes, number
format) must be fixed 36 times across fleet projects, while one exported
option in go-codec fixes it once. Same for the unused buffer pool and
streaming codecs — they are performance fixes the whole fleet inherits the
day the hot paths adopt them. The extraction doctrine (ADR-0128) already
paid off once this quarter: the v0.3.3 replay bug was fixed in the library
and reached all 86 module pins by a version bump, with zero changes here.

## 7. Ranked opportunities

1. **Assert `DeterministicCodec` at the sign/verify seam in `signing/v4`**
   (high, small): a typed helper or middleware option; v1 JSON callers
   break at compile time exactly as intended.
2. **Export a deterministic JSON mode in go-codec, then sweep the 36 local
   `json.Deterministic` sites** (high; upstream-first).
3. **A/B the pooled `BufferEncoder` path in `benchkit`** and adopt in
   `kv`/`event` hot loops if the alloc win is real (medium).
4. **Make `CBORCompactCodec` a real option**: a stack preset or a documented
   recipe for new stores that want unknown-field rejection (medium).
5. **Use `AutoDetectDebug` in corruption errors** where raw bytes are logged
   today, and `Diagnose` for CBOR payloads (low, operator QoL).
6. **Delete stale item #24 from `go-codec/TODO_LIST.md`** (trivial).

## 8. Reconsider when

- Any store grows payloads large enough that whole-buffer encode shows in
  profiles (streaming codecs flip from Missed to the obvious tool).
- A consumer needs JSON views of CBOR-stored events (HTTP projections,
  docserver exports): `TranscodeToJSON` becomes the one-liner.
- `GOEXPERIMENT=jsonv2` becomes the fleet default build: JSONCodec turns
  deterministic and the `DeterministicCodec` assertion loosens — re-run this
  census then.
- go-codec ships an exported deterministic JSON codec (opportunity 2) — the
  36-site sweep becomes mechanical.

## 9. How to verify

```
cd /home/lars/projects/go-cqrs-lite
rg 'go-codec v' -g go.mod | sort | uniq -c          # 86 × v0.3.3
rg -n 'codec\.WrapEncode' --type go -g '!*_test.go'  # 10 sites
rg -n 'codec\.DecodeEnvelopeOrLegacy|codec\.UnwrapDecode' --type go -g '!*_test.go'
rg -n 'codec\.AutoDetect' --type go -g '!*_test.go'  # metaengine/v4/encoded.go:112
rg -n 'RawCodec|EncodeToBuffer|GetBuffer|EncodePooled|TranscodeToJSON|Diagnose|ObserveCodec' --type go   # 0 prod
rg -n 'json\.Deterministic' --type go -g '!*_test.go' | wc -l   # 36
sed -n '18,36p' storage/pebble/v4/serialization.go
sed -n '24,26p' kv/v4/typed_options.go
sed -n '60,69p' signing/v4/cose_sign1.go
head -12 docs/adr/0044-blind-store-encoding-stamps.md 2>/dev/null || ls docs/adr | grep 0044
git -C /home/lars/projects/go-codec log -1 --format='%H'
git -C /home/lars/projects/go-codec tag --list | sort -V | tail -1   # v0.3.3
```

Sibling reports in this directory: `go-error-family.md`, `go-retry.md`,
`go-idempotency.md`.
