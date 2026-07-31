package query

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// PersistedQuery is a stored query with full audit metadata.
// It is the query-side equivalent of command.PersistedCommand.
// cqrs-lint:ignore(E007) library code or intentional pattern

type PersistedQuery struct {
	id         id.RequestID
	queryType  Type
	receivedAt time.Time
	payload    []byte
	metadata   Metadata
}

var _ fmt.Stringer = (*PersistedQuery)(nil)

// cqrs-lint:ignore(A001,E005) library code or intentional pattern

func (q *PersistedQuery) ID() id.RequestID      { return q.id }
func (q *PersistedQuery) Type() Type            { return q.queryType }
func (q *PersistedQuery) ReceivedAt() time.Time { return q.receivedAt }

func (q *PersistedQuery) Payload() []byte {
	if q.payload == nil {
		return nil
	}

	return slices.Clone(q.payload)
}

func (q *PersistedQuery) Metadata() Metadata { return q.metadata.Clone() }

func (q *PersistedQuery) String() string {
	return fmt.Sprintf("%s(%s)@%s", q.queryType, q.id, q.receivedAt.Format(time.RFC3339))
}

// QueryPersistOption configures a PersistedQuery.
type QueryPersistOption func(*PersistedQuery)

// WithQueryReceivedAt sets the received-at timestamp.
func WithQueryReceivedAt(t time.Time) QueryPersistOption {
	return func(q *PersistedQuery) { q.receivedAt = t }
}

// WithQueryID sets the query/request ID.
func WithQueryID(requestID id.RequestID) QueryPersistOption {
	return func(q *PersistedQuery) { q.id = requestID }
}

// WithQueryMetadata sets the metadata.
func WithQueryMetadata(m Metadata) QueryPersistOption {
	return func(q *PersistedQuery) { q.metadata = m.Clone() }
}

// NewPersistedQuery creates a query record for persistence.
func NewPersistedQuery(
	queryType Type,
	payload []byte,
	opts ...QueryPersistOption,
) (*PersistedQuery, error) {
	if queryType == "" {
		return nil, errorfamily.WrapRejection(
			ErrEmptyQueryType,
			"query.empty_query_type",
			"query type is required",
		)
	}

	q := &PersistedQuery{
		id:         id.NewRequestID(),
		queryType:  queryType,
		receivedAt: time.Now(),
		payload:    slices.Clone(payload),
		metadata:   Metadata{}, //nolint:exhaustruct // zero-value metadata is the correct initial state
	}

	for _, opt := range opts {
		opt(q)
	}

	return q, nil
}

// QuerySink persists queries for audit and replay.
type QuerySink interface {
	io.Closer
	SaveQuery(ctx context.Context, q *PersistedQuery) error
}

// QuerySource reads persisted queries.
type QuerySource interface {
	io.Closer
	LoadQueries(ctx context.Context, after time.Time) ([]*PersistedQuery, error)
}

// QueryStore combines sink and source for full query persistence.
type QueryStore interface {
	QuerySink
	QuerySource
}

// QueryJournal reads all queries across the system, ordered by ReceivedAt.
// This is the query-side equivalent of event.Journal.
// Use for audit: "who queried what data and when?".
type QueryJournal interface {
	ReadAllQueries(ctx context.Context) ([]*PersistedQuery, error)
}

// SeekableQueryJournal extends QueryJournal with position-based reading.
type SeekableQueryJournal interface {
	QueryJournal
	ReadQueriesFrom(
		ctx context.Context,
		afterRequestID id.RequestID,
		limit int,
	) ([]*PersistedQuery, error)
}
