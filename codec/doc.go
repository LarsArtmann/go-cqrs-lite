// Package codec provides payload encoding and decoding for event sourcing.
//
// The Codec interface abstracts serialization so that stores, snapshots, and
// event construction can work with any encoding format. Four implementations
// are provided:
//
//   - CBORCodec: recommended default — canonical CBOR (deterministic, signing-safe)
//   - JSONCodec: standard encoding/json — universal interop, human-readable
//   - CBORCompactCodec: stricter CBOR with unknown-field rejection (schema drift guard)
//   - RawCodec: passthrough for pre-encoded []byte payloads
//
// # Choosing a Codec
//
// Both JSON and CBOR are fully supported across the library. CBOR is recommended
// for internal serialization: smaller (~19-43%), faster to encode and decode,
// and deterministic (same input always produces the same output bytes — safe for
// cryptographic signing). JSON is the right choice for external interop, HTTP
// APIs, debugging, and any case where human-readability matters. The
// stack.Bundle's DefaultCodec() returns CBORCodec.
//
// # Usage
//
//	codec := codec.JSONCodec{}
//	data, err := codec.Encode(MyPayload{Name: "Alice"})
//	var decoded MyPayload
//	err = codec.Decode(data, &decoded)
//
// # Integration
//
// The Codec is used by event.New (auto-marshal payloads), event.DecodePayload[T]
// (typed decode), and snapshot stores (serialize aggregate state).
//
// The encryption module provides a composable codec wrapper (encryption.NewCodec)
// that wraps any Codec with transparent encrypt-on-encode / decrypt-on-decode.
// It reports its own encoding ("encrypted") and is used with event.WithCodec
// to create events with encrypted payloads.
//
// # CBOR Compact Encoding (toarray)
//
// For maximum payload size reduction (~30-40%), add the cbor:",toarray" struct
// tag to event payload types. This encodes structs as positional CBOR arrays
// instead of keyed maps, eliminating field-name string overhead entirely.
//
//	type UserCreated struct {
//	    _     struct{} `cbor:",toarray"`
//	    Name  string
//	    Email string
//	    Time  int64
//	}
//
// Without toarray (map encoding):  {"Name":"Alice","Email":"a@b.com","Time":1700000000}
// With toarray (array encoding):   ["Alice","a@b.com",1700000000]
//
// The toarray tag works with both CBORCodec and CBORCompactCodec. It is a
// per-type decision — mix array-encoded and map-encoded types freely. Once a
// struct uses toarray, field ORDER is part of the wire format and cannot be
// reordered without breaking existing data. Add new fields only at the end.
//
// For additional compactness, use CBORCompactCodec (CoreDetEncOptions +
// ExtraReturnErrors for schema drift detection). See CBORCompactCodec docs.
//
// # Zero-Allocation Encoding
//
// Codecs that implement the BufferEncoder interface can write directly into a
// caller-provided bytes.Buffer, avoiding the allocation returned by Encode.
// This is useful in hot paths where buffer reuse eliminates GC pressure.
//
//	buf := &bytes.Buffer{}
//	if be, ok := codec.(codec.BufferEncoder); ok {
//	    _ = be.EncodeToBuffer(payload, buf)
//	}
//
// # CBOR Compact Struct Tags
//
// fxamacker/cbor supports two struct tags for further payload optimization:
//
// keyasint — encode struct fields as integer keys instead of string keys.
// This eliminates field-name strings entirely, ideal for high-frequency
// events with many fields.
//
//	type OrderPlaced struct {
//	    _      struct{} `cbor:",keyasint"`
//	    UserID uint64   `cbor:"1,keyasint"`
//	    ItemID uint64   `cbor:"2,keyasint"`
//	    Qty    int      `cbor:"3,keyasint"`
//	}
//
// omitzero — omit fields that are zero-valued. Reduces payload size for
// events where many fields are optional.
//
//	type UserUpdated struct {
//	    Name  string `cbor:"name"`
//	    Email string `cbor:"email,omitempty"`
//	    Bio   string `cbor:"bio,omitzero"`
//	}
//
// Both tags work with CBORCodec and CBORCompactCodec. Once adopted, the
// integer key mapping is part of the wire format — changing key numbers
// breaks existing data.
package codec
