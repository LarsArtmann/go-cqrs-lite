package bbolt

import (
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

type serializableQuery struct {
	ID         id.RequestID   `json:"id"`
	Type       string         `json:"type"`
	ReceivedAt int64          `json:"received_at"`
	Payload    []byte         `json:"payload"`
	Metadata   query.Metadata `json:"metadata"`
}

func marshalQuery(q *query.PersistedQuery) ([]byte, error) {
	sq := serializableQuery{
		ID:         q.ID(),
		Type:       string(q.Type()),
		ReceivedAt: q.ReceivedAt().UnixNano(),
		Payload:    q.Payload(),
		Metadata:   q.Metadata(),
	}

	//art-dupl:accept mirror of event/command serialization — type-specific wire structs
	data, err := marshalCBOR(sq)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.serialize_query",
			"marshal query")
	}

	return data, nil
}

func unmarshalQuery(data []byte) (*query.PersistedQuery, error) {
	var sq serializableQuery

	if err := unmarshalCBOROrJSON(data, &sq, "bbolt.unmarshal_query",
		"failed to unmarshal query"); err != nil {
		return nil, err
	}

	q, err := query.NewPersistedQuery(
		query.Type(sq.Type),
		sq.Payload,
		query.WithQueryReceivedAt(time.Unix(0, sq.ReceivedAt).UTC()),
		query.WithQueryID(sq.ID),
		query.WithQueryMetadata(sq.Metadata),
	)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "bbolt.reconstruct_query",
			"reconstruct query")
	}

	return q, nil
}
