package encryption

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

const (
	// EnvelopeVersionV1 is the initial ciphertext envelope version.
	EnvelopeVersionV1 = "v1"

	// EnvelopeKey is the metadata key for the ciphertext envelope.
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

// MarshalEnvelope serializes an Envelope to a base64-encoded JSON string.
func MarshalEnvelope(env Envelope) (string, error) {
	if env.Version == "" {
		env.Version = EnvelopeVersionV1
	}

	data, err := json.Marshal(env)
	if err != nil {
		return "", event.WrapInfrastructure(err, "encryption.marshal_envelope", "marshal envelope")
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

// UnmarshalEnvelope deserializes a base64-encoded JSON string into an Envelope.
func UnmarshalEnvelope(encoded string) (Envelope, error) {
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return Envelope{}, event.WrapInfrastructure(
			err,
			"encryption.unmarshal_envelope",
			"decode envelope base64",
		)
	}

	var env Envelope

	if err := json.Unmarshal(data, &env); err != nil {
		return Envelope{}, event.WrapInfrastructure(
			err,
			"encryption.unmarshal_envelope",
			"unmarshal envelope JSON",
		)
	}

	if env.Version == "" {
		return Envelope{}, event.NewRejection(
			"encryption.missing_envelope_version",
			fmt.Sprintf("envelope has no version field: %s", string(data)),
		)
	}

	return env, nil
}
