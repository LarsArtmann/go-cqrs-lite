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
	metaJSON, _ := json.Marshal(q.Metadata())

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

// ─── encode/decode helpers ───

func (a *QueryAdapter) queriesToAny(queries []*query.PersistedQuery) []any {
	if !a.serialize {
		result := make([]any, len(queries))
		for i, q := range queries {
			result[i] = q
		}

		return result
	}

	result := make([]any, len(queries))
	for i, q := range queries {
		result[i] = a.encodeQuery(q)
	}

	return result
}

func (a *QueryAdapter) anyToQueries(values []any) ([]*query.PersistedQuery, error) {
	result := make([]*query.PersistedQuery, 0, len(values))
	for _, val := range values {
		q, err := a.decodeQueryValue(val)
		if err != nil {
			return nil, err
		}

		result = append(result, q)
	}

	return result, nil
}

func (a *QueryAdapter) decodeQueryValue(val any) (*query.PersistedQuery, error) {
	// Direct pointer (Memory engine).
	if q, ok := val.(*query.PersistedQuery); ok {
		return q, nil
	}

	// Serialized string (SQLite/Pebble engine, raw string passthrough).
	if s, ok := val.(string); ok {
		return a.decodeQuery(s)
	}

	// Decoded JSON map (SQLite engine auto-decodes JSON strings on read).
	// Re-marshal to JSON and decode as a serializedQuery envelope.
	if m, ok := val.(map[string]any); ok {
		data, err := json.Marshal(m)
		if err != nil {
			return nil, fmt.Errorf("query adapter: re-marshal decoded value: %w", err)
		}

		return a.decodeQuery(string(data))
	}

	return nil, fmt.Errorf("%w: %T", ErrUnsupportedValueType, val)
}
