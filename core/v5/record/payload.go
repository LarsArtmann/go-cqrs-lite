package record

// Payload is the record's encoded payload with its codec stamp fused into
// one value (review R4): payload presence becomes honest states instead of
// one conflated zero-value pair.
//
//	State     Data     Enc                 Meaning
//	absent    empty    EncodingUnknown     no payload at all (the zero Payload)
//	opaque    set      EncodingUnknown     bytes without a codec stamp at this layer
//	wrapped   set      EncodingEnvelope    ADR-0044 envelope; the envelope carries its own codec stamp
//	stamped   set      EncodingJSON/CBOR   self-describing payload
//
// Construct through [PayloadFrom]. The absent state is checkable with
// [Payload.IsZero]; the wrapped state is stamped by producers that KNOW the
// bytes are an envelope (the typed command/query stores always WrapEncode) —
// bridges that cannot know leave the stamp opaque.
type Payload struct {
	// Data is the encoded payload bytes. Empty in the absent state.
	Data []byte

	// Enc is the codec stamp. EncodingUnknown with empty Data means absent;
	// EncodingUnknown with Data set means opaque (no stamp at this layer).
	Enc Encoding
}

// PayloadFrom fuses payload bytes and their codec stamp into a Payload.
// A nil/empty data slice yields the absent state regardless of the stamp.
func PayloadFrom(data []byte, enc Encoding) Payload {
	if len(data) == 0 {
		return Payload{}
	}

	return Payload{Data: data, Enc: enc}
}

// IsZero reports the absent state: no payload data. A stamp without data is
// meaningless, so presence is a fact about the bytes alone.
func (p Payload) IsZero() bool {
	return len(p.Data) == 0
}
