package query

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metadata/v4"
)

// Type identifies a query type.
type Type string

// String returns the query type as a string.
func (t Type) String() string { return string(t) }

// IsZero returns true if the query type is empty.
func (t Type) IsZero() bool { return t == "" }

// ParseType validates and returns a Type. Returns an error if empty.
func ParseType(s string) (Type, error) {
	if s == "" {
		return "", ErrEmptyQueryType
	}

	return Type(s), nil
}

// Query represents a read-side query.
type Query interface {
	Type() Type
}

// MetadataKey represents a custom metadata key for queries.
// It is query-local so consumers adding custom metadata need not import
// event/ for a domain-neutral string type (ADR-0031).
type MetadataKey string

// Metadata contains tracing and contextual information for queries.
// It is a type alias for metadata.CustomData[MetadataKey] so that Clone, Merge,
// and EnsureCustom are inherited directly — no per-module wrapper methods are
// needed. See ADR-0031.
//
// Unlike the old alias of event.Metadata, query.Metadata does NOT carry
// event-only concerns (Tombstone, Causation). Each module owns its own
// Metadata so a change to the event's shape cannot silently reshape queries.
type Metadata = metadata.CustomData[MetadataKey]

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

// WithRequestID sets the request ID for debugging.
func WithRequestID(v id.RequestID) Option {
	return func(q *BasicQuery) { q.metadata.RequestID = v }
}

// BasicQuery provides a default implementation.
type BasicQuery struct {
	queryType Type
	metadata  Metadata
}

var _ Query = (*BasicQuery)(nil)

// Type returns the query type.
func (q *BasicQuery) Type() Type { return q.queryType }

// Metadata returns a defensive copy of the query metadata.
func (q *BasicQuery) Metadata() Metadata { return q.metadata.Clone() }

// New creates a new query with validation.
func New(queryType Type, opts ...Option) (*BasicQuery, error) {
	if queryType == "" {
		return nil, ErrEmptyQueryType
	}

	q := &BasicQuery{
		queryType: queryType,
		metadata:  Metadata{},
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
