package metaengine

import (
	"slices"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// NormalizeEncoded converts an encoded payload into JSON bytes, detecting the
// codec from the payload itself (via [codec.AutoDetect]). It is the ingress
// bridge for callers that hold only bytes — typically an event's Payload().
//
// Supported encodings are JSON (returned unchanged) and CBOR (transcoded with
// [codec.TranscodeToJSON]). The returned encoding names the detected codec, so
// callers can log or persist the decision. Payloads that match neither codec
// fail with a Corruption-family error: the bytes are deterministic, so retrying
// cannot fix them.
func NormalizeEncoded(payload []byte) ([]byte, codec.Encoding, error) {
	return normalizeEncoded(payload, "")
}

// NormalizeEncodedWithEncoding is [NormalizeEncoded] for callers that already
// carry a codec stamp (for example record.Record.Encoding, or
// event.Event.Encoding()). Known stamps are honored; an empty or [EncodingRaw]
// stamp falls back to detection. A stamp that lies about the bytes surfaces as
// a decode error downstream rather than silently guessing.
func NormalizeEncodedWithEncoding(payload []byte, encoding codec.Encoding) ([]byte, error) {
	normalized, _, err := normalizeEncoded(payload, encoding)

	return normalized, err
}

func normalizeEncoded(payload []byte, stamped codec.Encoding) ([]byte, codec.Encoding, error) {
	if len(payload) == 0 {
		return nil, codec.EncodingRaw, errorfamily.NewCorruption(
			"metaengine.encoded.empty_payload",
			"metaengine: encoded payload is empty",
		)
	}

	encoding := stamped
	if encoding == "" || encoding == codec.EncodingRaw {
		encoding = codec.AutoDetect(payload)
	}

	switch encoding {
	case codec.EncodingJSON:
		return slices.Clone(payload), encoding, nil
	case codec.EncodingCBOR:
		normalized, err := codec.TranscodeToJSON(payload, codec.EncodingCBOR)
		if err != nil {
			return nil, encoding, errorfamily.WrapCorruption(
				err,
				"metaengine.encoded.transcode_cbor",
				"metaengine: transcode CBOR payload to JSON",
			)
		}

		return normalized, encoding, nil
	default:
		return nil, encoding, errorfamily.NewCorruption(
			"metaengine.encoded.unknown_encoding",
			"metaengine: payload is neither JSON nor CBOR",
		)
	}
}

// codecEncodingOf converts a Record's compact stamp into the codec vocabulary.
// The zero stamp (EncodingUnknown) yields "", which normalizeEncoded treats as
// "detect from the bytes".
func codecEncodingOf(encoding record.Encoding) codec.Encoding {
	switch encoding {
	case record.EncodingJSON:
		return codec.EncodingJSON
	case record.EncodingCBOR:
		return codec.EncodingCBOR
	default:
		return ""
	}
}
