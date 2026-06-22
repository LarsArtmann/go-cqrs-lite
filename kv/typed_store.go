package kv

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

// TypedStore is a typed read-model store over an untyped [Store].
//
// It serializes values of type T via a [codec.Codec] and addresses them by
// keys of type K, which must implement [fmt.Stringer] (as all branded
// identifiers from the id package do). One [TypedStore] corresponds to one
// read-model type; create a separate TypedStore per read model.
//
// TypedStore is safe for concurrent use when the underlying [Store] is.
// [MemStore] and [pebble.KVAdapter] both are.
//
// This type was moved verbatim from kv.TypedStore (ADR-0032).
type TypedStore[T any, K fmt.Stringer] struct {
	backend Store
	codec   codec.Codec
	prefix  []byte
	keyFunc func(K) []byte
}

// NewTypedStore creates a [TypedStore] over backend, applying the given options.
// The default codec is [codec.JSONCodec]; the default key encoding is the
// key's String() form.
func NewTypedStore[T any, K fmt.Stringer](
	backend Store,
	opts ...TypedOption[T, K],
) *TypedStore[T, K] {
	s := &TypedStore[T, K]{ //nolint:exhaustruct // prefix set via WithKeyPrefix option
		backend: backend,
		codec:   codec.JSONCodec{},
		keyFunc: func(k K) []byte { return []byte(k.String()) },
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Get retrieves the value for id, decoding it into a new T.
// Returns [ErrNotFound] if no value exists.
func (s *TypedStore[T, K]) Get(_ context.Context, id K) (*T, error) {
	data, err := s.backend.Get(s.key(id))
	if err != nil {
		return nil, fmt.Errorf("kv: get key %q: %w", s.key(id), err)
	}

	var val T

	err = s.codec.Decode(data, &val)
	if err != nil {
		return nil, fmt.Errorf("kv: decode key %q: %w", s.key(id), err)
	}

	return &val, nil
}

// Has reports whether a value exists for id without reading it.
func (s *TypedStore[T, K]) Has(_ context.Context, id K) (bool, error) {
	has, err := s.backend.Has(s.key(id))
	if err != nil {
		return false, fmt.Errorf("kv: has key %q: %w", s.key(id), err)
	}

	return has, nil
}

// Set encodes val and stores it under id, replacing any existing value.
func (s *TypedStore[T, K]) Set(_ context.Context, id K, val *T) error {
	if val == nil {
		return fmt.Errorf("%w for key %q", errNilTypedValue, s.key(id))
	}

	data, err := s.codec.Encode(val)
	if err != nil {
		return fmt.Errorf("kv: encode key %q: %w", s.key(id), err)
	}

	err = s.backend.Set(s.key(id), data)
	if err != nil {
		return fmt.Errorf("kv: set key %q: %w", s.key(id), err)
	}

	return nil
}

// Delete removes the value for id. Deleting a missing key is a no-op.
func (s *TypedStore[T, K]) Delete(_ context.Context, id K) error {
	err := s.backend.Delete(s.key(id))
	if err != nil {
		return fmt.Errorf("kv: delete key %q: %w", s.key(id), err)
	}

	return nil
}

// Scan returns all values whose keys start with the store's key prefix
// (if set via [WithTypedKeyPrefix]) concatenated with prefix. Values are returned
// in lexicographic key order as yielded by the [Store] iterator.
//
// Pass an empty prefix to scan every key in this store's namespace.
// The caller owns the returned slice; the store does not retain it.
func (s *TypedStore[T, K]) Scan(_ context.Context, prefix []byte) ([]*T, error) {
	scanPrefix := prefix

	if len(s.prefix) > 0 {
		scanPrefix = append(append([]byte{}, s.prefix...), prefix...)
	}

	iter, err := s.backend.NewIterator(scanPrefix)
	if err != nil {
		return nil, fmt.Errorf("kv: scan iterator: %w", err)
	}

	defer func() { _ = iter.Close() }()

	results := make([]*T, 0)

	for iter.Next() {
		var val T

		err = s.codec.Decode(iter.Value(), &val)
		if err != nil {
			return nil, fmt.Errorf("kv: scan decode key %q: %w", iter.Key(), err)
		}

		results = append(results, &val)
	}

	err = iter.Error()
	if err != nil {
		return nil, fmt.Errorf("kv: scan iteration: %w", err)
	}

	return results, nil
}

// Backend returns the underlying [Store] the typed store reads from and writes to.
// Exposed so callers can access backend-specific functionality (batches,
// iterators over raw keys) when the typed API is insufficient.
func (s *TypedStore[T, K]) Backend() Store { return s.backend }

func (s *TypedStore[T, K]) key(id K) []byte {
	k := s.keyFunc(id)

	if len(s.prefix) > 0 {
		return append(append([]byte{}, s.prefix...), k...)
	}

	return k
}
