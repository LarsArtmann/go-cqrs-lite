# ADR-0161: Encoded ingress decodes with the producer's codec, not a schema-free JSON transcode

| Field      | Value                                                                                                 |
| ---------- | ----------------------------------------------------------------------------------------------------- |
| Status     | Accepted                                                                                              |
| Date       | 2026-10-24                                                                                            |
| Deciders   | Lars Artmann                                                                                          |
| Related    | ADR-0062 (metaengine dependency boundary), ADR-0066 (reify), ADR-0111 (Record), ADR-0126 (wrappers)   |
| Supersedes | nothing (re-opens the codec question left open by ADR-0062's 2026-08-06 amendment and ADR-0066)       |

## Context

`metaengine` exposes two encoded entry points: `Store.ApplyEncoded` and
`Store.ApplyEncodedRecord`. They take payload **bytes** rather than a decoded
value, because the store, not the caller, knows each fold's expected Go type.
Every fold decodes the bytes into its own sample before `invoke`
(`metaengine/encoded.go`).

Until 2026-10-23 those entry points accepted JSON only. That is not a corner
case, it is the library's own default:

- `event.New` encodes with `event.DefaultCodec`, which is CBOR
  (`event/codec.go`; pinned by `TestDefaultCodec_DefaultIsCBOR`).
- Therefore the documented bridge `store.ApplyEncoded(ctx, evt.Type(),
  evt.Payload())` could not process the events the library produces.
- The zero-config `projectionadapter` decoded into `map[string]any`
  (`projectionadapter/adapter.go`), and a typed fold cannot consume a map: the
  generated `invoke` closure calls the handler through `reflect.Call`, which
  panics (`using map[string]interface {} as type ...`), poisons the collection,
  and reaches the dead-letter store as `ErrPoisoned`. That path was broken for
  JSON too; CBOR merely failed one step earlier, with a `Corruption` error.

The codec question was never decided against on the merits. ADR-0062 declared a
"zero-dependency core" (2026-07-25) and ADR-0066 chose bare JSON engine values
plus the `reify` round-trip, explicitly rejecting "store type tags alongside
values". ADR-0062's 2026-08-06 addendum declared the zero-dependency premise
**wrong** but only re-admitted `record/`; nobody re-opened the codec question,
so the JSON-only ingress survived as an unexamined consequence.

The forcing evidence is a RED test, not an opinion: a CBOR payload containing a
`time.Time` cannot be normalized by a schema-free transcode. go-codec's
canonical CBOR mode uses `cbor.TimeUnixDynamic`, which writes whole-second times
as **untagged** epoch integers; decoding that generically yields `int64`, and
`encoding/json` then refuses to unmarshal the number into `time.Time`:

```
decode TaskCreated: json decode into TaskCreated:
json: unable to unmarshal JSON number into Go time.Time within "/At"
```

Schema-free transcoding is therefore structurally lossy for any Go value whose
CBOR form is not JSON-representable (time, custom marshaling, map keys that are
not strings). It cannot be the ingress normal form.

## Decision

**Encoded ingress keeps the producer's bytes and decodes them with the producer's
codec, per fold. Schema-free JSON normalization is rejected.**

1. **Accepted encodings** are JSON and CBOR. The codec is taken from
   `Record.Encoding` when known, otherwise detected with `codec.AutoDetect`.
   Encodings outside that pair are an explicit error at ingress; callers with a
   custom codec decode themselves and use `Store.Apply`/`ApplyRecord`.
2. **Two pipeline markers, both carrying the caller's exact bytes.**
   - JSON keeps the historical `jsontext.Value` marker, decoded per fold with
     `encoding/json/v2`. No stored or logged bytes change.
   - CBOR becomes the internal `encodedPayload{raw, encoding}` marker, cloned on
     ingress.
   The marker type is deliberately *not* exported: it crosses the public
   `EventInput.Payload any` boundary, but external callers construct it only by
   calling the encoded entry points.
3. **Per-fold decode with the producer's codec.** `decodeRawFoldPayload`
   dispatches on the marker; `codec.ForEncoding(encoding).Decode(raw, v)` fills
   `reflect.New(fold sample)`. Folds whose sample is a byte slice receive the
   raw bytes undecoded (the pre-existing contract).
4. **A known stamp is trusted.** A `Record.Encoding` that lies about the bytes
   surfaces as a deterministic decode error, never as a silent re-sniff. An
   unknown stamp means "detect".
5. **Decode failures classify as Corruption** (`metaengine.apply.encoded_decode`),
   because the bytes are immutable: retrying cannot fix them, and projection
   hosts must dead-letter rather than crash-loop. This matches the contract the
   adapter already documented.
6. **The zero-config adapter delegates.** With neither a `PayloadDecoder` nor an
   `EventDecoder` configured, `Adapter.Handle` hands the raw payload to
   `Store.ApplyEncodedRecord`. It never builds a `map[string]any` intermediary,
   which no typed fold can consume.
7. **`metaengine` may depend on go-codec** as the Tier -1 codec substrate it
   already shares with `event/`, the storage engines and the projection adapter.
   ADR-0062's zero-dependency premise is superseded by its own 2026-08-06
   addendum; this ADR extends that conclusion to the codec vocabulary so the
   module stops maintaining a private notion of "encoding".

## Alternatives considered

- **Schema-free CBOR→JSON transcode at ingress** (the original plan): rejected
  on the `time.Time` evidence above. It also charges every CBOR payload a
  decode+encode round trip and still cannot represent custom CBOR types.
- **Decode to `map[string]any`, then `Apply`** (the pre-existing adapter
  default): rejected. Typed folds panic in `reflect.Call`; the path was never
  usable, only untested.
- **Leave the entry points JSON-only; callers decode CBOR themselves**: rejected.
  It makes the library's documented bridge incompatible with the library's
  default event codec, and it pushes per-event-type dispatch into every consumer
  (the exact boilerplate `projectionadapter` exists to remove).
- **Store codec stamps alongside engine values** (ADR-0066 reversal): NOT
  decided here. It is orthogonal to ingress, needs a compatibility window, and
  is tracked as an open question (status report 2026-10-24, §7).
- **Trust nothing, always sniff**: rejected. A stamp is a contract; ignoring it
  would make an intentional JSON payload get re-interpreted by a heuristic.

## Consequences

**Positive**

- The library's default event payload flows through its own encoded entry point
  and through a zero-config projection adapter, with dead-letter semantics
  intact.
- No stored-byte change: JSON logs and engine rows keep their representation;
  CBOR payloads are decoded per fold and never reach storage re-encoded.
- Replay is representation-transparent: `Verify`, `Backfill`, engine catch-up
  and failover pass the marker through the same fold funnel (pinned by
  `TestApplyEncoded_CBOR_ReplayThroughVerify` / `..._BackfillReplaysIntoLiveStore`).
- The codec vocabulary is shared with `event/` instead of duplicated.

**Negative / accepted costs**

- **Two markers** exist in the pipeline (`jsontext.Value` and `encodedPayload`).
  They are behaviourally equivalent but structurally different; unifying them is
  a follow-up with its own compatibility story.
- **`AutoDetect` is a heuristic**, not a security boundary (go-codec says so
  explicitly). Shapes are pinned by a matrix and a fuzz target; a stamp is
  always preferred when the caller has one.
- **Only JSON and CBOR** are supported by the encoded entry points. Encrypted or
  custom codecs still require a manual decode plus `ApplyRecord`.
- **One reflection allocation per fold** on the encoded path. The JSON path's
  decode cost is unchanged; the CBOR cost is new work on a path that previously
  failed outright (numbers recorded in the completion plan).
- A lying `Record.Encoding` fails the whole apply. That is intentional, but it
  means a buggy producer sees a Corruption error rather than a best-effort read.

## Implementation status (2026-10-24)

| Surface | Where |
| ------- | ----- |
| Ingress markers + detection | `metaengine/encoded.go` (`ApplyEncoded`, `ApplyEncodedRecord`, `encodePayload`, `encodedPayload`, `decodeEncodedFoldPayload`, `codecEncodingOf`) |
| Adapter delegation | `metaengine/projectionadapter/adapter.go` (`Handle`, no-decoder branch) |
| Error taxonomy | `metaengine.apply.encoded_codec`, `metaengine.apply.encoded_decode`, `metaengine.encoded.empty_payload`, `metaengine.encoded.unknown_encoding` (`docs/error-taxonomy.md`) |
| Dependency | `metaengine/go.mod` requires `github.com/larsartmann/go-codec`; `DEP_BUDGET[metaengine]` = 8 |
| Pins | `metaengine/encoded_codec_test.go`, `encoded_replay_test.go`, `encoded_internal_test.go` (markers, sniff matrix, fuzz, replay), `projectionadapter/adapter_default_codec_test.go` |

## References

- ADR-0062 — Metaengine dependency boundary (amended 2026-08-06)
- ADR-0066 — Cross-engine JSON reification (rejects stamped values; unaffected)
- ADR-0111 — `record.Record` (`Encoding` stamp, closed JSON/CBOR vocabulary)
- ADR-0126 — Store/journal decoration (the reason no ad-hoc wrapper was added)
- `docs/planning/2026-10-24_05-33_SUPERB-metaengine-codec-boundary-completion.md`
- `docs/status/2026-10-24_05-27_metaengine-codec-boundary-repair-status.md`
