package system

import (
	"context"
	"encoding/json/v2"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// AdapterCore is the shared machinery for wrapping a
// [metaengine.StreamLogBackend] as a typed CQRS store adapter
// (EventAdapter, CommandAdapter, QueryAdapter). It owns the backend and
// collection, the serialize/passthrough switch for Memory-vs-persistent
// engines, and the value dispatch used on every read.
//
// Embed it by value in adapter types, set the function fields in the
// adapter's constructor, and promote its ReadAll/ReadFromAfter methods where
// the adapted interface matches.
type AdapterCore[T any] struct {
	Backend    metaengine.StreamLogBackend
	Collection string

	// Noun labels the adapter in error messages ("event", "command", "query").
	Noun string

	// Serialize switches between direct-pointer passthrough (Memory engine)
	// and envelope strings (persistent engines). Toggled by the adapter's
	// With*Serialization option.
	Serialize bool

	// Encode renders one item as its persistent envelope string.
	Encode func(T) string

	// Decode parses one envelope string back into an item.
	Decode func(string) (T, error)

	// IDOf returns the journal identity of one item, used by ReadFromAfter
	// to resolve cursor IDs into journal sequence numbers.
	IDOf func(T) string
}

// ToAny converts items into backend values: direct pointers when Serialize
// is off, envelope strings when it is on.
func (c *AdapterCore[T]) ToAny(items []T) []any {
	result := make([]any, len(items))

	if !c.Serialize {
		for i, item := range items {
			result[i] = item
		}

		return result
	}

	for i, item := range items {
		result[i] = c.Encode(item)
	}

	return result
}

// FromAny converts backend values back into items via decodeValue.
func (c *AdapterCore[T]) FromAny(values []any) ([]T, error) {
	result := make([]T, 0, len(values))

	for _, val := range values {
		item, err := c.decodeValue(val)
		if err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, nil
}

// decodeValue dispatches one backend value: a direct T pointer (Memory
// engine), a serialized envelope string (SQLite/Pebble passthrough), or a
// decoded JSON map (SQLite engines that auto-decode TEXT values on read,
// re-marshaled here before envelope decoding).
func (c *AdapterCore[T]) decodeValue(val any) (T, error) {
	if item, ok := val.(T); ok {
		return item, nil
	}

	if s, ok := val.(string); ok {
		return c.Decode(s)
	}

	if m, ok := val.(map[string]any); ok {
		data, err := json.Marshal(m)
		if err != nil {
			var zero T

			return zero, fmt.Errorf("%s adapter: re-marshal decoded value: %w", c.Noun, err)
		}

		return c.Decode(string(data))
	}

	var zero T

	return zero, fmt.Errorf("%w: %T", ErrUnsupportedValueType, val)
}

// ReadAll returns every item in the journal, ordered by insertion.
func (c *AdapterCore[T]) ReadAll(ctx context.Context) ([]T, error) {
	values, err := c.Backend.JournalReadAll(ctx, c.Collection)
	if err != nil {
		return nil, fmt.Errorf("%s adapter: read all: %w", c.Noun, err)
	}

	return c.FromAny(values)
}

// LoadStream reads one stream by key and decodes its values. Shared body of
// EventAdapter.Load and CommandAdapter.Load.
func (c *AdapterCore[T]) LoadStream(ctx context.Context, streamKey string) ([]T, error) {
	values, err := c.Backend.StreamRead(ctx, c.Collection, streamKey)
	if err != nil {
		return nil, fmt.Errorf("%s adapter: load: %w", c.Noun, err)
	}

	return c.FromAny(values)
}

// ReadFromAfter returns up to limit journal items after the item whose IDOf
// matches afterID. An empty afterID reads from the start of the journal.
// The cursor is resolved by scanning the journal once; adapters that need
// cheaper resolution (EventAdapter's sequence cache) implement their own.
func (c *AdapterCore[T]) ReadFromAfter(
	ctx context.Context,
	afterID string,
	limit int,
) ([]T, error) {
	afterSeq := int64(0)

	if afterID != "" {
		all, err := c.ReadAll(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s adapter: read from: %w", c.Noun, err)
		}

		for i, item := range all {
			if c.IDOf(item) == afterID {
				afterSeq = int64(i + 1)

				break
			}
		}
	}

	values, err := c.Backend.JournalReadFrom(ctx, c.Collection, afterSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("%s adapter: read from: %w", c.Noun, err)
	}

	return c.FromAny(values)
}
