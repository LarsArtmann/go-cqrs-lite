# codec — Payload Encoding for Event Sourcing

Encoding/decoding for event payloads. Provides the `Codec` interface used by stores, snapshots, and event construction.

```bash
go get github.com/larsartmann/go-cqrs-lite/codec/v2
```

## Codecs

| Codec       | Description                                   |
| ----------- | --------------------------------------------- |
| `JSONCodec` | Standard `encoding/json` marshal/unmarshal    |
| `RawCodec`  | Passthrough for pre-encoded `[]byte` payloads |

## Interface

```go
type Codec interface {
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}
```

## Usage

```go
codec := codec.JSONCodec{}
data, _ := codec.Encode(MyPayload{Name: "Alice"})
var decoded MyPayload
_ = codec.Decode(data, &decoded)
```
