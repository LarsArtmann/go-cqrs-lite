package codec

import (
	"encoding/json/v2"
	"fmt"
)

// envelopeMagic is the marker value that identifies envelope-wrapped data.
// Its presence in the Magic field confirms the data is an envelope, not raw.
const envelopeMagic = "cqrs"

// envelope wraps serialized data with its encoding format tag, making blind
// stores self-describing (like events are with evt.Encoding()).
type envelope struct {
	Magic    string   `json:"$"`   // always "cqrs" — distinguishes envelope from raw data
	Encoding Encoding `json:"enc"` // codec encoding: "json" or "cbor"
	Data     []byte   `json:"dat"` // inner serialized data
}

// WrapEncode serializes v with codec c and wraps the result in a JSON envelope.
// The envelope itself is always JSON-encoded so it can be read without knowing
// the inner codec. Callers should use [UnwrapDecode] on the read path to
// extract the codec and inner data, or fall back to raw decode for old data.
func WrapEncode(v any, c Codec) ([]byte, error) {
	inner, err := c.Encode(v)
	if err != nil {
		return nil, fmt.Errorf("codec: encode for envelope: %w", err)
	}

	env := envelope{Magic: envelopeMagic, Encoding: c.Encoding(), Data: inner}

	out, err := json.Marshal(env, json.Deterministic(true))
	if err != nil {
		return nil, fmt.Errorf("codec: marshal envelope: %w", err)
	}

	return out, nil
}

// UnwrapDecode inspects data for an envelope wrapper. If found, it returns the
// stamped codec and inner data bytes. If not found (old unenveloped data), it
// returns the fallback codec and the original data unchanged.
//
// This enables backward-compatible migration: old data decodes via the fallback,
// new data decodes via the stamped codec. Stores can gradually transition
// without a full clear-and-rebuild.
func UnwrapDecode(data []byte, fallback Codec) (Codec, []byte) {
	var env envelope
	if err := json.Unmarshal(data, &env); err == nil &&
		env.Magic == envelopeMagic && len(env.Data) > 0 {
		if c, err := ForEncoding(env.Encoding); err == nil {
			return c, env.Data
		}
	}

	return fallback, data
}
