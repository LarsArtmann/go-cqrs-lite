package codec

// Size encodes v with both JSON and CBOR and returns the byte sizes. This is
// useful for deciding whether switching a payload type from JSON to CBOR is
// worthwhile before committing to a format change.
//
// If a codec fails to encode v, that size is reported as -1.
//
//	type UserCreated struct { Name string; Email string }
//	jsonSize, cborSize := codec.Size(UserCreated{Name: "Alice", Email: "a@b.c"})
//	savings := float64(jsonSize-cborSize) / float64(jsonSize) * 100 // e.g. -19
func Size(v any) (int, int) {
	jsonData, err := (JSONCodec{}).Encode(v)
	if err != nil {
		cborData, cborErr := (CBORCodec{}).Encode(v)
		if cborErr != nil {
			return -1, -1
		}

		return -1, len(cborData)
	}

	cborData, err := (CBORCodec{}).Encode(v)
	if err != nil {
		return len(jsonData), -1
	}

	return len(jsonData), len(cborData)
}
