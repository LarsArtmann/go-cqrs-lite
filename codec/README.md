# codec — Payload Encoding for Event Sourcing

[![Go Reference](https://pkg.go.dev/badge/github.com/larsartmann/go-cqrs-lite/codec/v4.svg)](https://pkg.go.dev/github.com/larsartmann/go-cqrs-lite/codec/v4)

Encoding/decoding for event payloads. Provides the `Codec` interface used by stores, snapshots, and event construction.

```bash
go get github.com/larsartmann/go-cqrs-lite/codec/v4
```

## Codecs

| Codec              | Description                                                              |
| ------------------ | ------------------------------------------------------------------------ |
| `CBORCodec`        | **Recommended.** Canonical CBOR (RFC 7049) — deterministic, signing-safe |
| `JSONCodec`        | Standard `encoding/json` — universal interop, human-readable             |
| `CBORCompactCodec` | Stricter CBOR (RFC 8949 Core Deterministic) + unknown-field rejection    |
| `RawCodec`         | Passthrough for pre-encoded `[]byte` payloads                            |

## Interface

```go
type Codec interface {
    Encoding() Encoding
    Encode(v any) ([]byte, error)
    Decode(data []byte, v any) error
}

// Optional zero-allocation interface:
type BufferEncoder interface {
    EncodeToBuffer(v any, buf *bytes.Buffer) error
}
```

## Usage

### CBOR (Canonical) — Recommended

```go
codec := codec.CBORCodec{}
data, _ := codec.Encode(MyPayload{Name: "Alice"})
var decoded MyPayload
_ = codec.Decode(data, &decoded)
```

CBOR produces deterministic output (sorted map keys, shortest floats), making it
safe for content-addressed storage and cryptographic signing. The pebble event
store uses CBOR internally for its on-disk envelope format.

### JSON

```go
codec := codec.JSONCodec{}
data, _ := codec.Encode(MyPayload{Name: "Alice"})
var decoded MyPayload
_ = codec.Decode(data, &decoded)
```

### CBOR Compact (Strict)

```go
codec := codec.CBORCompactCodec{}
data, _ := codec.Encode(MyPayload{Name: "Alice"})
```

`CBORCompactCodec` uses stricter settings than `CBORCodec`:

- **Encoding**: Core Deterministic (RFC 8949) — bytewise-lexically sorted map keys
- **Decoding**: Rejects unknown struct fields as schema drift detection

**Not compatible** with data written by `CBORCodec`. Use only for new event stores.

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

### When to Use CBOR vs JSON

**CBOR is the recommended default** for internal serialization — it is
smaller (19-43%), faster to encode/decode (25-72%), and deterministic (same bytes
every time — safe for signing). JSON is fully supported and remains the right
choice for external interop, debugging, and human-readable payloads. Both codecs
work everywhere in the library; pick one per use case.

| Scenario                                | Recommended Codec  | Why                                 |
| --------------------------------------- | ------------------ | ----------------------------------- |
| **Default for new projects**            | `CBORCodec`        | Smaller, faster, signing-safe       |
| Event payloads in PebbleDB              | `CBORCodec`        | Deterministic encoding for signing  |
| Cryptographic signing of payloads       | `CBORCodec`        | Canonical byte representation       |
| High-throughput event streams           | `CBORCodec`        | Smaller encoded size, faster decode |
| Read models / projections               | `CBORCodec`        | Stack `DefaultCodec()` returns CBOR |
| New event store with schema drift guard | `CBORCompactCodec` | Unknown-field rejection on decode   |
| External system interop / HTTP APIs     | `JSONCodec`        | Universal support                   |
| Debugging / human-readable payloads     | `JSONCodec`        | Readable in logs, curl, DB queries  |
| Pre-encoded payloads                    | `RawCodec`         | Zero-copy passthrough               |

## CBOR Struct Tags for Smaller Payloads

fxamacker/cbor reads `json` struct tags by default, so CBOR works with existing
struct definitions. For additional payload size reduction, use CBOR-specific tags:

### `toarray` — Positional Array Encoding (~30-40% smaller)

Encodes structs as positional CBOR arrays instead of keyed maps, eliminating
field-name string overhead entirely:

```go
type UserCreated struct {
    _     struct{} `cbor:",toarray"`
    Name  string
    Email string
    Time  int64
}
```

Without toarray: `{"Name":"Alice","Email":"a@b.com","Time":1700000000}`
With toarray: `["Alice","a@b.com",1700000000]`

Once a struct uses `toarray`, field **order is part of the wire format** and cannot
be reordered without breaking existing data. Add new fields only at the end.

### `keyasint` — Integer Field Keys

```go
type OrderPlaced struct {
    _      struct{} `cbor:",keyasint"`
    UserID uint64   `cbor:"1,keyasint"`
    ItemID uint64   `cbor:"2,keyasint"`
    Qty    int      `cbor:"3,keyasint"`
}
```

### `omitzero` — Skip Zero-Valued Fields

```go
type UserUpdated struct {
    Name  string `cbor:"name"`
    Email string `cbor:"email,omitempty"`
    Bio   string `cbor:"bio,omitzero"`
}
```

Both tags work with `CBORCodec` and `CBORCompactCodec`. Once adopted, the
integer key mapping is part of the wire format — changing key numbers breaks
existing data.

## Zero-Allocation Encoding (BufferEncoder)

Codecs that implement the `BufferEncoder` interface can write directly into a
caller-provided `bytes.Buffer`, avoiding the allocation returned by `Encode`:

```go
buf := &bytes.Buffer{}
if be, ok := codec.(codec.BufferEncoder); ok {
    _ = be.EncodeToBuffer(payload, buf)
}
```

Both `JSONCodec`, `CBORCodec`, and `CBORCompactCodec` implement `BufferEncoder`.

## Streaming CBOR

For encoding/decoding large event batches without materializing the full byte
slice in memory, use the streaming encoder/decoder:

```go
// Encode a batch to a stream
f, _ := os.Create("events.cbor")
enc := codec.NewCBOREncoder(f)
for _, evt := range events {
    _ = enc.Encode(evt)
}

// Decode a batch from a stream
f, _ := os.Open("events.cbor")
dec := codec.NewCBORDecoder(f)
for {
    var evt MyEvent
    if err := dec.Decode(&evt); err == io.EOF { break }
}
```

The streaming encoder uses the same canonical encoding mode as `CBORCodec`.

## CBOR Diagnostic Notation

Debug corrupt events or inspect raw CBOR payloads in human-readable form:

```go
cborData, _ := codec.CBORCodec{}.Encode(event)
diag, _ := codec.Diagnose(cborData)
log.Printf("CBOR event: %s", diag)
```

## Shared CBOR Modes

Modules that need CBOR encoding identical to `CBORCodec` (e.g., custom storage
backends) should use the exported modes instead of creating their own:

```go
// Same canonical EncMode/DecMode used by CBORCodec internally
data, _ := codec.CBOREncMode().Marshal(payload)
_ = codec.CBORDecMode().Unmarshal(data, &payload)
```

## Time Handling

CBOR codecs use `TimeUnixDynamic` — float64 epoch with sub-second precision (9 bytes).
This preserves nanosecond values in `time.Time` payload fields (within ~165ns float drift).

**Convention:** All `time.Time` in event payloads MUST be `.UTC()` before encoding.
Epoch values carry no timezone; decoded times reconstruct in `time.Local`, not the
original location. Normalizing to UTC at encode time eliminates this ambiguity.

**Wall-clock times** (recurring schedules, business hours) must NOT use `time.Time` —
store wall time components + IANA timezone name instead. See
[docs/TIMEZONE_HANDLING.md](../docs/TIMEZONE_HANDLING.md) for the full guide.

## Related Modules

- [**event**](../event/README.md) — `DecodePayload[T]` accepts a `Codec` to decode payloads
- [**signing**](../signing/README.md) — CBOR's deterministic encoding makes signatures reproducible
- [**encryption**](../encryption/README.md) — `encryption.NewCodec` wraps a codec with encryption
- [**storage/pebble**](../storage/pebble/README.md) — Uses CBOR internally for its on-disk envelope format
- [**kv**](../kv/README.md) — `WithTypedCodec` lets read models use CBOR
