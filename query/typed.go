package query

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-codec"
	errorfamily "github.com/larsartmann/go-error-family"

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
// Pre-envelope data decodes via the configured codec with a JSON↔CBOR
// cross-retry, so legacy rows written with either standard codec read
// correctly regardless of the codec configured now.
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
		payload, err := decodeEnvelopeOrLegacy[P](q.Payload(), t.codec)
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

// decodeEnvelopeOrLegacy decodes ADR-0044 envelope-stamped data with its
// stamped codec, and non-envelope data with the configured codec. When the
// configured codec fails on non-envelope bytes, one cross-retry with the
// other standard codec rescues legacy rows written before the envelope
// existed (raw JSON under a CBOR-configured store, or vice versa), keeping
// ADR-0050's permanent-readability guarantee.
//
// art-dupl:accept duplicated across dep-isolated blind stores (kv/snapshot/command/query); sharing would add a cross-module dependency
func decodeEnvelopeOrLegacy[P any](data []byte, configured codec.Codec) (P, error) {
	c, inner := codec.UnwrapDecode(data, configured)

	var val P

	err := c.Decode(inner, &val)
	if err == nil {
		return val, nil
	}

	alt, ok := otherStandardCodec(c)
	if !ok {
		return val, err
	}

	var retry P

	if altErr := alt.Decode(inner, &retry); altErr == nil {
		return retry, nil
	}

	return val, err
}

// otherStandardCodec returns the opposite built-in codec, or false for
// envelope-stamped or custom codecs (their data only decodes with themselves).
func otherStandardCodec(c codec.Codec) (codec.Codec, bool) { //nolint:ireturn // built-in cross-retry
	switch c.(type) {
	case codec.CBORCodec:
		return codec.JSONCodec{}, true
	case codec.JSONCodec:
		return codec.CBORCodec{}, true
	default:
		return nil, false
	}
}
