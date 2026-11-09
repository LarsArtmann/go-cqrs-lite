package encryption

import (
	"encoding/base64"
	"encoding/json/v2"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

const (
	// EnvelopeVersionV1 is the initial ciphertext envelope version: a JSON
	// object, base64-encoded as one opaque string. It predates the discovery
	// that snapshot state lives in JSON/JSONB columns on PostgreSQL and
	// MySQL, where a bare base64 string is rejected ("invalid input syntax
	// for type json"). Still readable, never written.
	EnvelopeVersionV1 = "v1"

	// EnvelopeVersionV2 is the current envelope version: the JSON object
	// itself, with no outer base64 wrap. Storable in JSON columns on every
	// SQL dialect, and as opaque bytes on KV engines.
	EnvelopeVersionV2 = "v2"

	// EnvelopeKey is the metadata key for the ciphertext envelope.
	// Opt-in: consumers who want envelope-based metadata can use this key
	// when storing encrypted payloads via their own integration code.
	EnvelopeKey event.MetadataKey = "event.encryption.envelope"
)

// Envelope wraps ciphertext with versioning metadata for forward-compatible
// algorithm changes. Future versions can add fields without breaking consumers
// that only inspect the Version field.
type Envelope struct {
	Version    string     `json:"v"`
	Ciphertext Ciphertext `json:"ct"`
	Algorithm  Algorithm  `json:"alg,omitempty"`
	KeyID      KeyID      `json:"kid,omitempty"`
}

// MarshalEnvelope serializes an Envelope to its versioned wire format.
// Since v2 the output is the JSON object itself — storable in JSON/JSONB
// columns (the v1 outer base64 wrap was rejected by PostgreSQL and MySQL
// snapshot state columns). v1 envelopes remain readable via
// [UnmarshalEnvelope].
func MarshalEnvelope(env Envelope) (string, error) {
	if env.Version == "" {
		env.Version = EnvelopeVersionV2
	}

	data, err := json.Marshal(env, json.Deterministic(true))
	if err != nil {
		return "", errorfamily.WrapInfrastructure(
			err,
			"encryption.marshal_envelope",
			"marshal envelope",
		)
	}

	return string(data), nil
}

// UnmarshalEnvelope deserializes an Envelope from its wire format, reading
// both generations: v2 raw JSON and the v1 base64-wrapped JSON.
func UnmarshalEnvelope(encoded string) (Envelope, error) {
	trimmed := strings.TrimSpace(encoded)

	var data []byte

	if strings.HasPrefix(trimmed, "{") {
		data = []byte(trimmed)
	} else {
		decoded, err := base64.URLEncoding.DecodeString(trimmed)
		if err != nil {
			return Envelope{}, errorfamily.WrapInfrastructure(
				err,
				"encryption.unmarshal_envelope",
				"decode envelope base64",
			)
		}

		data = decoded
	}

	var env Envelope

	if err := json.Unmarshal(data, &env, json.MatchCaseInsensitiveNames(true)); err != nil {
		return Envelope{}, errorfamily.WrapInfrastructure(
			err,
			"encryption.unmarshal_envelope",
			"unmarshal envelope JSON",
		)
	}

	if env.Version == "" {
		return Envelope{}, errorfamily.NewRejection(
			"encryption.missing_envelope_version",
			"envelope has no version field: "+string(data),
		)
	}

	return env, nil
}
