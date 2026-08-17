package metaengine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// --- vector payload codec (binary float32, legacy JSON fallback) ---
//
// Brute-force vector engines (pebble, bbolt, badger) store each embedding
// under the "vec\x00<col>\x00<id>" key family. The payload is
// self-describing:
//
//	binary (current): 'b' | dim uint32 LE | dim x float32 LE
//	JSON (legacy):    a bare []float32 array, e.g. [1,2,3]
//
// The 1-byte format marker keeps legacy JSON rows readable — a JSON text can
// never start with 'b' — so deployments upgrade in place: new writes are
// binary, old rows keep decoding through DecodeVectorAuto. JSON decode was
// the dominant brute-force scan cost (~17us/vector on pebble vs the ~90ns
// in-RAM ceiling); see
// docs/planning/2026-08-16_VECTOR-SEARCH-AT-SCALE-SPIKE.md §2-§4.

// vectorBinaryMarker tags a binary-encoded vector payload. 'b' can never
// begin a JSON text (JSON values start with '[', '{', '"', '-', a digit, or
// a literal keyword), so the format sniff in DecodeVectorAuto is unambiguous.
const vectorBinaryMarker = 'b'

// vectorBinaryHeaderLen is the fixed prefix of a binary payload: the marker
// byte plus the uint32 dimension count.
const vectorBinaryHeaderLen = 5

// EncodeVectorBinary serializes values as marker + dimension (uint32 LE) +
// little-endian float32s — fixed-width, no text parsing on decode.
func EncodeVectorBinary(values []float32) []byte {
	dim := vectorBinaryDim(values)

	data := make([]byte, vectorBinaryHeaderLen+4*len(values))
	data[0] = vectorBinaryMarker
	binary.LittleEndian.PutUint32(data[1:5], dim)

	for i, v := range values {
		binary.LittleEndian.PutUint32(data[vectorBinaryHeaderLen+4*i:], math.Float32bits(v))
	}

	return data
}

// vectorBinaryDim converts the slice length to the wire-format dimension
// count without triggering gosec G115. Extracted as a helper per AGENTS.md
// convention.
func vectorBinaryDim(values []float32) uint32 {
	return uint32(len(values)) //nolint:gosec // G115: bounded by memory long before uint32
}

// errNotBinaryVector rejects payloads without the binary marker or with a
// truncated header.
var errNotBinaryVector = errors.New("metaengine.DecodeVectorBinary: not a binary vector payload")

// DecodeVectorBinary decodes a binary-encoded vector payload produced by
// EncodeVectorBinary. It errors when the marker is absent or the dimension
// header does not match the payload length (torn or foreign bytes).
func DecodeVectorBinary(data []byte) ([]float32, error) {
	if len(data) < vectorBinaryHeaderLen || data[0] != vectorBinaryMarker {
		return nil, errNotBinaryVector
	}

	dim := binary.LittleEndian.Uint32(data[1:5])
	// uint64 arithmetic: on 32-bit platforms 4*dim would overflow int.
	if want := uint64(vectorBinaryHeaderLen) + 4*uint64(dim); want != uint64(len(data)) {
		return nil, fmt.Errorf(
			"metaengine.DecodeVectorBinary: payload length %d does not match dimension %d",
			len(data), dim,
		)
	}

	values := make([]float32, dim)
	for i := range values {
		values[i] = math.Float32frombits(
			binary.LittleEndian.Uint32(data[vectorBinaryHeaderLen+4*i:]),
		)
	}

	return values, nil
}

// DecodeVectorAuto decodes a stored vector payload in either format: payloads
// carrying the binary marker take the fixed-width path, anything else falls
// back to the legacy JSON array (DecodeVectorJSON). This is the read path for
// brute-force engines, so rows written before the binary format keep working
// across upgrades.
func DecodeVectorAuto(data []byte) ([]float32, error) {
	if len(data) > 0 && data[0] == vectorBinaryMarker {
		return DecodeVectorBinary(data) //nolint:wrapcheck // error already names the function
	}

	return DecodeVectorJSON(data) //nolint:wrapcheck // error already names the function
}
