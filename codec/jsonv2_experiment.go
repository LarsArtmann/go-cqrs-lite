//go:build goexperiment.jsonv2

package codec

import "encoding/json/v2"

// JSONCodecV2 wraps Go's experimental JSON v2 encoder/decoder.
// It implements the Codec interface with the jsonv2 encoding.
//
// This file is only compiled when the goexperiment.jsonv2 build tag is enabled.
// Enable with: go build -tags goexperiment.jsonv2
//
// Note: encoding/json/v2 is still in draft. This experiment may break between
// Go releases. See ADR-0026 for details.
type JSONCodecV2 struct{}

// Encoding returns "json".
func (JSONCodecV2) Encoding() Encoding { return EncodingJSON }

// Encode serializes v using json v2.
func (JSONCodecV2) Encode(v any) ([]byte, error) {
	return json.Marshal(v) //nolint:wrapcheck // codec layer
}

// Decode deserializes data into v using json v2.
func (JSONCodecV2) Decode(data []byte, v any) error {
	return json.Unmarshal(
		data,
		v,
		json.MatchCaseInsensitiveNames(true),
	)
}
