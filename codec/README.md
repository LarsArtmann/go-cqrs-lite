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

The CBOR decoder enforces strict validation:

- **Duplicate map keys** are rejected (not silently overwritten)
- **Unknown struct fields** cause decode errors (not silently ignored)

This catches schema mismatches early — if a producer adds a field that the
consumer doesn't know about, the consumer gets an explicit error instead of
silently dropping data.

### When to Use CBOR vs JSON

| Scenario                               | Recommended Codec | Why                                 |
| -------------------------------------- | ----------------- | ----------------------------------- |
| Event payloads in PebbleDB             | `CBORCodec`       | Deterministic encoding for signing  |
| Interoperability with external systems | `JSONCodec`       | Universal support                   |
| Cryptographic signing of payloads      | `CBORCodec`       | Canonical byte representation       |
| Pre-encoded payloads                   | `RawCodec`        | Zero-copy passthrough               |
| High-throughput event streams          | `CBORCodec`       | Smaller encoded size, faster decode |

### Encoding Metadata

Each codec reports its encoding via the `Encoding()` method:

```go
c := codec.CBORCodec{}
fmt.Println(c.Encoding()) // "cbor"
```

The `Encoding` type constants are `EncodingJSON`, `EncodingCBOR`, and `EncodingRaw`.
