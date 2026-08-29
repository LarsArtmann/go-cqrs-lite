package query

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// Type identifies a query type.
//
// It is an alias of record.Type (ADR-0111): one definition shared with
// event and command, so the per-module copies cannot drift.
type Type = record.Type

// ParseType validates and returns a Type. Returns an error if empty.
//
// Deprecated: removed in v5. Use record.ParseType(s, ErrEmptyQueryType).
func ParseType(s string) (Type, error) {
	return record.ParseType(s, ErrEmptyQueryType)
}

// Query represents a read-side query.
// cqrs-lint:ignore(E007) library code or intentional pattern
type Query interface {
	Type() Type
}

// MetadataKey represents a custom metadata key for queries.
// It is query-local so consumers adding custom metadata need not import
// event/ for a domain-neutral string type (ADR-0031).
type MetadataKey string

// Metadata contains tracing and contextual information for queries.
// It is a type alias for [metadata.Metadata] keyed by the query-local
// MetadataKey, so Clone, Merge, and WithCustom are inherited from the
// canonical generic (ADR-0031, WAL unification). The JSON shape is
// unchanged: {"correlationId": ..., "custom": {...}}.
//
// Unlike event.Metadata, query.Metadata does NOT carry event-only concerns
// (Tombstone, Causation). Each module owns its own Metadata so a change to
// the event's shape cannot silently reshape queries.
type Metadata = metadata.Metadata[MetadataKey]

// Option configures query creation.
type Option func(*BasicQuery)

// WithCorrelationID sets the correlation ID for distributed tracing.
func WithCorrelationID(v id.CorrelationID) Option {
	return func(q *BasicQuery) { q.metadata.CorrelationID = v }
}

// WithCausationID sets the causation ID (indicates what triggered this query).
func WithCausationID(v id.CausationID) Option {
	return func(q *BasicQuery) { q.metadata.CausationID = v }
}

// WithUserID sets the user ID who issued the query.
func WithUserID(v id.UserID) Option {
	return func(q *BasicQuery) { q.metadata.UserID = v }
}

// WithActor sets the effective actor (user, bot, system, or service) that
// issued the query. This is the primary audit-trail field for compliance.
func WithActor(v id.ActorID) Option {
	return func(q *BasicQuery) { q.metadata.ActorID = v }
}

// WithRequestID sets the request ID for debugging.
func WithRequestID(v id.RequestID) Option {
	return func(q *BasicQuery) { q.metadata.RequestID = v }
}

// WithCustomMetadata sets a custom metadata key-value pair on the query.
// Multiple calls accumulate. Used by transport adapters to carry wire-level
// metadata (e.g. gRPC payload, correlation context).
func WithCustomMetadata(key, value string) Option {
	return func(q *BasicQuery) {
		q.metadata = q.metadata.WithCustom(MetadataKey(key), value)
	}
}

// BasicQuery provides a default implementation.
// cqrs-lint:ignore(E007) library code or intentional pattern
type BasicQuery struct {
	queryType Type
	metadata  Metadata
}

var _ Query = (*BasicQuery)(nil)

// Type returns the query type.
func (q *BasicQuery) Type() Type { return q.queryType }

// Metadata returns a defensive copy of the query metadata.
func (q *BasicQuery) Metadata() Metadata { return q.metadata.Clone() }

// ApplyOptions applies metadata options to an already-constructed query.
// Intended for pipeline enrichment: transport adapters inject request-scoped
// metadata (actor IDs, correlation IDs) after the domain decoder creates the
// query but before dispatch. Options that set already-populated fields
// will overwrite them.
func (q *BasicQuery) ApplyOptions(opts ...Option) {
	for _, opt := range opts {
		opt(q)
	}
}

// New creates a new query with validation. Error style mirrors
// [command.New]: the sentinel is wrapped in a Rejection with a contextual
// message, so errors.Is(err, ErrEmptyQueryType) keeps matching.
func New(queryType Type, opts ...Option) (*BasicQuery, error) {
	if queryType == "" {
		return nil, errorfamily.WrapRejection(
			ErrEmptyQueryType,
			"query.empty_query_type",
			"query type is required: got empty",
		)
	}

	q := &BasicQuery{
		queryType: queryType,
		metadata:  Metadata{}, //nolint:exhaustruct // zero-value metadata is the correct initial state
	}

	for _, opt := range opts {
		opt(q)
	}

	return q, nil
}

// Middleware wraps query handlers for cross-cutting concerns.
type Middleware func(Handler) Handler

// TypedHandler processes a typed query and returns a typed result.
// Q is the concrete query type, R is the result type.
// Use with RegisterTyped for compile-time type safety at registration,
// eliminating the need for manual type assertions in handlers.
type TypedHandler[Q Query, R any] func(ctx context.Context, q Q) (R, error)
