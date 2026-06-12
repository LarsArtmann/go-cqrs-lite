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
