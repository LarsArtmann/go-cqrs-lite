package query

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TypedQuery is a query with a typed payload P, closing the type-safety hole
// where [PersistedQuery].Payload is an untyped []byte.
// cqrs-lint:ignore(E007) library code or intentional pattern
type TypedQuery[P any] struct {
	ID         id.RequestID
	Type       Type
	ReceivedAt time.Time
	Payload    P
	Metadata   Metadata
}

// TypedQueryStore adapts an untyped [QueryStore] plus a [codec.Codec] into a
// typed interface over P. It handles encode/decode at the store boundary.
type TypedQueryStore[P any] struct {
	store QueryStore
	codec codec.Codec
}

// NewTypedQueryStore creates a typed adapter over store using c for payload
// serialization. If c is nil, [codec.CBORCodec] is used.
// Pre-envelope data (raw JSON) is auto-detected on read.
func NewTypedQueryStore[P any](store QueryStore, c codec.Codec) *TypedQueryStore[P] {
	if c == nil {
		c = codec.CBORCodec{}
	}

	return &TypedQueryStore[P]{store: store, codec: c}
}

// SaveQuery encodes q.Payload and delegates to the underlying [QueryStore].
func (t *TypedQueryStore[P]) SaveQuery(ctx context.Context, q TypedQuery[P]) error {
	data, err := codec.WrapEncode(q.Payload, t.codec)
	if err != nil {
		return errorfamily.WrapCorruption(err, "query.typed_store.encode",
			"encode typed payload")
	}

	opts := []QueryPersistOption{
		WithQueryMetadata(q.Metadata),
	}

	if q.ReceivedAt.IsZero() {
		opts = append(opts, WithQueryReceivedAt(time.Now()))
	} else {
		opts = append(opts, WithQueryReceivedAt(q.ReceivedAt))
	}

	if q.ID != (id.RequestID{}) {
		opts = append(opts, WithQueryID(q.ID))
	}

	persisted, err := NewPersistedQuery(q.Type, data, opts...)
	if err != nil {
		return err
	}

	err = t.store.SaveQuery(ctx, persisted)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "query.typed_store.save", "save typed query")
	}

	return nil
}

// LoadQueries retrieves all queries after `after`, decoding each payload into P.
func (t *TypedQueryStore[P]) LoadQueries(
	ctx context.Context,
	after time.Time,
) ([]TypedQuery[P], error) {
	queries, err := t.store.LoadQueries(ctx, after)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"query.typed_store.load",
			"load typed queries",
		)
	}

	result := make([]TypedQuery[P], 0, len(queries))

	for _, q := range queries {
		var payload P

		c, inner := codec.UnwrapDecode(q.Payload(), codec.JSONCodec{})

		err := c.Decode(inner, &payload)
		if err != nil {
			return nil, errorfamily.WrapCorruption(err, "query.typed_store.decode",
				fmt.Sprintf("decode typed payload for %s", q.ID()))
		}

		result = append(result, TypedQuery[P]{
			ID:         q.ID(),
			Type:       q.Type(),
			ReceivedAt: q.ReceivedAt(),
			Payload:    payload,
			Metadata:   q.Metadata(),
		})
	}

	return result, nil
}
