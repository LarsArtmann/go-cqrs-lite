package event

import (
	"encoding/json/v2"

	errorfamily "github.com/larsartmann/go-error-family"
)

// UnmarshalMetadataJSON parses metadata JSON into event options.
// Returns nil options for empty data. Wraps parse errors as corruption errors
// with the provided error code prefix.
func UnmarshalMetadataJSON(data []byte, errCode, eventType string) ([]Option, error) {
	//art-dupl:accept ADR-0152 v5 copy-forward twin of the v4 train; removed with v4 in T26
	if len(data) == 0 {
		return nil, nil
	}

	var meta Metadata

	err := json.Unmarshal(data, &meta)
	if err != nil {
		return nil, errorfamily.WrapCorruption(
			err,
			errCode,
			"unmarshal metadata for event "+eventType,
		)
	}

	return []Option{WithMetadata(meta)}, nil
}

// MarshalMetadataJSON serializes event metadata to JSON.
// Wraps serialization errors as corruption errors with the provided error code prefix.
func MarshalMetadataJSON(m Metadata, errCode string) ([]byte, error) {
	data, err := json.Marshal(m, json.Deterministic(true))
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, errCode, "marshal metadata")
	}

	return data, nil
}
