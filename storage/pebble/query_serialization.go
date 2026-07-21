package pebble

import (
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

	return marshalCBOROrErr(serialized, "pebble.serialize_query", "marshal query")
}

func (s *QueryStore) deserializeQuery(data []byte) (*query.PersistedQuery, error) {
	var serialized serializableQuery

	if err := unmarshalCBOROrJSON(data, &serialized, "pebble.unmarshal_query",
		"failed to unmarshal query"); err != nil {
		return nil, err
	}

	q, err := query.NewPersistedQuery(
		query.Type(serialized.Type),
		serialized.Payload,
		query.WithQueryID(serialized.ID),
		query.WithQueryReceivedAt(time.Unix(0, serialized.ReceivedAt).UTC()),
		query.WithQueryMetadata(serialized.Metadata),
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "pebble.reconstruct_query",
			"failed to reconstruct query from stored fields")
	}

	return q, nil
}
