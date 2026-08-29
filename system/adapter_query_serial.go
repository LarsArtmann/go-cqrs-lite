package system

import (
	"encoding/json/v2"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// serializedQuery is the JSON envelope for persisting queries in SQL-based
// StreamLogBackends. The Memory engine stores pointers directly; SQL engines
// store this envelope as a TEXT value.
type serializedQuery struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	ReceivedAt time.Time `json:"received_at"`
	Payload    []byte    `json:"payload"`
	Metadata   []byte    `json:"metadata"`
}

func (a *QueryAdapter) encodeQuery(q *query.PersistedQuery) string {
	// encodeQuery cannot propagate errors: AdapterCore.Encode is `func(T) string`
	// by design (ADR-0126 core constraint). On a failed metadata marshal the
	// envelope persists a nil Metadata field (decodes to zero-value metadata)
	// instead of partial JSON. Today's fields are all marshal-safe (typed
	// string IDs, map[K]string custom data); the guard keeps that guarantee
	// if richer values ever land.
	metaJSON, metaErr := json.Marshal(q.Metadata(), json.Deterministic(true))
	if metaErr != nil {
		metaJSON = nil
	}

	env := serializedQuery{
		ID:         q.ID().String(),
		Type:       string(q.Type()),
		ReceivedAt: q.ReceivedAt(),
		Payload:    q.Payload(),
		Metadata:   metaJSON,
	}

	data, _ := json.Marshal(env)

	return string(data)
}

func (a *QueryAdapter) decodeQuery(s string) (*query.PersistedQuery, error) {
	var env serializedQuery
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, fmt.Errorf("query adapter: decode envelope: %w", err)
	}

	requestID, err := id.ParseRequestID(env.ID)
	if err != nil {
		return nil, fmt.Errorf("query adapter: parse request ID: %w", err)
	}

	var meta query.Metadata
	if len(env.Metadata) > 0 {
		if err := json.Unmarshal(env.Metadata, &meta); err != nil {
			return nil, fmt.Errorf("query adapter: decode metadata: %w", err)
		}
	}

	q, err := query.NewPersistedQuery(
		query.Type(env.Type), env.Payload,
		query.WithQueryID(requestID),
		query.WithQueryReceivedAt(env.ReceivedAt),
		query.WithQueryMetadata(meta),
	)
	if err != nil {
		return nil, fmt.Errorf("query adapter: reconstruct query: %w", err)
	}

	return q, nil
}
