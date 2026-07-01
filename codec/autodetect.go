package codec

// cborMinMajorType is the minimum byte value for CBOR major types 4 and 5
// (arrays 0x80-0x9f, maps 0xa0-0xbf). These never start valid JSON.
const cborMinMajorType byte = 0x80

// AutoDetect inspects raw payload bytes and returns the most likely [Encoding].
// It distinguishes JSON from CBOR by examining the structural first byte:
//
//   - JSON objects/arrays/strings start with '{', '[', or '"' (ASCII).
//   - CBOR maps (major type 5) start with 0xa0–0xbf; arrays (major type 4)
//     with 0x80–0x9f. These ranges never overlap with valid JSON leading bytes.
//
// For ambiguous leading bytes (e.g. bare numbers, booleans) the function falls
// back to a trial decode: JSON first, then CBOR.
//
// Empty input returns [EncodingRaw]. Unknown data returns [EncodingRaw].
//
// AutoDetect is a best-effort heuristic for diagnostics and tooling — it is NOT
// a security boundary. Never use it to skip encoding validation; always pair
// detected data with the matching codec's Decode for authoritative parsing.
func AutoDetect(data []byte) Encoding {
	if len(data) == 0 {
		return EncodingRaw
	}

	first := data[0]

	// CBOR major types 4-7 (0x80-0xff) never start valid JSON.
	if first >= cborMinMajorType {
		return EncodingCBOR
	}

	// Unambiguous JSON structural starts.
	switch first {
	case '{', '[', '"':
		return EncodingJSON
	}

	// JSON keywords and numbers (ASCII letters/digits/signs). These overlap
	// with CBOR major types 0-3, so try JSON decode first.
	if isJSONStart(first) {
		var v any
		if (JSONCodec{}).Decode(data, &v) == nil {
			return EncodingJSON
		}

		return EncodingCBOR
	}

	// Remaining bytes are either CBOR scalars or unrecognised.
	var v any
	if err := (CBORCodec{}).Decode(data, &v); err == nil {
		return EncodingCBOR
	}

	return EncodingRaw
}

// isJSONStart reports whether b is a valid first byte for a JSON value
// (per RFC 8259): object, array, string, number, true, false, null.
func isJSONStart(b byte) bool {
	switch b {
	case '{', '[', '"', '-', 't', 'f', 'n':
		return true
	}

	return b >= '0' && b <= '9'
}
