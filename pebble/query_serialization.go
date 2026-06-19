package pebble

import (
	"encoding/json"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// serializableQuery is the CBOR (and legacy JSON) storage format for queries.
type serializableQuery struct {
	ID         id.RequestID   `json:"id"`
	Type       string         `json:"type"`
	ReceivedAt int64          `json:"received_at"` //nolint:tagliatelle // on-disk format uses snake_case
	Payload    []byte         `json:"payload"`
	Metadata   query.Metadata `json:"metadata"`
}

func (s *QueryStore) serializeQuery(q *query.PersistedQuery) ([]byte, error) {
	sq := serializableQuery{
		ID:         q.ID(),
		Type:       string(q.Type()),
		ReceivedAt: q.ReceivedAt().UnixNano(),
		Payload:    q.Payload(),
		Metadata:   q.Metadata(),
	}

	return pebbleEncMode.Marshal(sq) //nolint:wrapcheck // storage serialization, not domain error
}

func (s *QueryStore) deserializeQuery(data []byte) (*query.PersistedQuery, error) {
	var sq serializableQuery

	var err error

	if isCBOR(data) {
		err = pebbleDecMode.Unmarshal(data, &sq)
	} else {
		err = json.Unmarshal(
			data,
			&sq,
		) //nolint:nolintlint // legacy JSON fallback for backward compat
	}

	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.unmarshal_query",
			"failed to unmarshal query")
	}

	q, err := query.NewPersistedQuery(
		query.Type(sq.Type),
		sq.Payload,
		query.WithQueryID(sq.ID),
		query.WithQueryReceivedAt(time.Unix(0, sq.ReceivedAt)),
		query.WithQueryMetadata(sq.Metadata),
	)
	if err != nil {
		return nil, event.WrapCorruption(err, "pebble.reconstruct_query",
			"failed to reconstruct query from stored fields")
	}

	return q, nil
}
