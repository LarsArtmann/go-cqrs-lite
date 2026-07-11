package pebble

import (
	"encoding/json/v2"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// serializableQuery is the CBOR (and legacy JSON) storage format for queries.
type serializableQuery struct {
	ID         id.RequestID   `json:"id"`
	Type       string         `json:"type"`
	ReceivedAt int64          `json:"received_at"`
	Payload    []byte         `json:"payload"`
	Metadata   query.Metadata `json:"metadata"`
}

func (s *QueryStore) serializeQuery(q *query.PersistedQuery) ([]byte, error) {
	serialized := serializableQuery{
		ID:         q.ID(),
		Type:       string(q.Type()),
		ReceivedAt: q.ReceivedAt().UnixNano(),
		Payload:    q.Payload(),
		Metadata:   q.Metadata(),
	}

	data, err := marshalCBOR(serialized)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.serialize_query", "marshal query")
	}

	return data, nil
}

func (s *QueryStore) deserializeQuery(data []byte) (*query.PersistedQuery, error) {
	var serialized serializableQuery

	var err error

	if isCBOR(data) {
		err = unmarshalCBOR(data, &serialized)
	} else {
		err = json.Unmarshal(
			data,
			&serialized,
		) //nolint:nolintlint // legacy JSON fallback for backward compat
	}

	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.unmarshal_query",
			"failed to unmarshal query")
	}

	q, err := query.NewPersistedQuery(
		query.Type(serialized.Type),
		serialized.Payload,
		query.WithQueryID(serialized.ID),
		query.WithQueryReceivedAt(time.Unix(0, serialized.ReceivedAt)),
		query.WithQueryMetadata(serialized.Metadata),
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.reconstruct_query",
			"failed to reconstruct query from stored fields")
	}

	return q, nil
}
