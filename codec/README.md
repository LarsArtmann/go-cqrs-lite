# codec — Payload Encoding for Event Sourcing

Encoding/decoding for event payloads. Provides the `Codec` interface used by stores, snapshots, and event construction.

```bash
go get github.com/larsartmann/go-cqrs-lite/codec/v2
```

## Codecs

| Codec       | Description                                             |
| ----------- | ------------------------------------------------------- |
| `JSONCodec` | Standard `encoding/json` marshal/unmarshal              |
| `CBORCodec` | Canonical CBOR (RFC 7049) — deterministic, signing-safe |
| `RawCodec`  | Passthrough for pre-encoded `[]byte` payloads           |

## Interface

```go
type Codec interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

## Usage

### JSON

```go
codec := codec.JSONCodec{}
data, _ := codec.Encode(MyPayload{Name: "Alice"})
var decoded MyPayload
_ = codec.Decode(data, &decoded)
```

### CBOR

```go
codec := codec.CBORCodec{}
data, _ := codec.Encode(MyPayload{Name: "Alice"})
var decoded MyPayload
_ = codec.Decode(data, &decoded)
```

CBOR produces deterministic output (sorted map keys, shortest floats), making it
safe for content-addressed storage and cryptographic signing. The pebble event
store uses CBOR internally for its on-disk envelope format.

### CBOR Strict Decoding

The CBOR decoder enforces strict validation for data integrity:

- **Duplicate map keys** are rejected (not silently overwritten)
- **Unknown struct fields** are silently ignored for forward compatibility

This balances strictness (duplicate keys are always a bug) with forward
compatibility — producers can add fields without breaking consumers that
haven't been updated yet.

### When to Use CBOR vs JSON

| Scenario                               | Recommended Codec | Why                                 |
| -------------------------------------- | ----------------- | ----------------------------------- |
| Event payloads in PebbleDB             | `CBORCodec`       | Deterministic encoding for signing  |
| Interoperability with external systems | `JSONCodec`       | Universal support                   |
| Cryptographic signing of payloads      | `CBORCodec`       | Canonical byte representation       |
| Pre-encoded payloads                   | `RawCodec`        | Zero-copy passthrough               |
| High-throughput event streams          | `CBORCodec`       | Smaller encoded size, faster decode |

### CBOR with Event Signing

CBOR's deterministic encoding makes it ideal for signed event payloads — the same
data always produces the same bytes, so signatures are reproducible:

```go
// Use CBOR for deterministic event payloads
c := codec.CBORCodec{}
data, _ := c.Encode(payload)

// Sign the canonical CBOR bytes (same input → same signature every time)
signer, _ := signing.NewHMAC(secret)
sig, _ := signer.Sign(data)

// Verify on the consumer side
if !signer.Verify(data, sig) {
    return errors.New("signature mismatch")
}
var decoded MyPayload
_ = c.Decode(data, &decoded)
```

### Encoding Metadata

Each codec reports its encoding via the `Encoding()` method:

```go
c := codec.CBORCodec{}
fmt.Println(c.Encoding()) // "cbor"
```

The `Encoding` type constants are `EncodingJSON`, `EncodingCBOR`, and `EncodingRaw`.
