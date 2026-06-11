// Package codec provides payload encoding and decoding for event sourcing.
//
// The Codec interface abstracts serialization so that stores, snapshots, and
// event construction can work with any encoding format. Two implementations
// are provided:
//
//   - JSONCodec: standard encoding/json marshal/unmarshal
//   - RawCodec: passthrough for pre-encoded []byte payloads
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
package codec
