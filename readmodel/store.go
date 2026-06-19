package readmodel

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

// Store is a typed read-model store over an untyped [Backend].
//
// It serializes values of type T via a [codec.Codec] and addresses them by
// keys of type K, which must implement [fmt.Stringer] (as all branded
// identifiers from the id package do). One [Store] corresponds to one
// read-model type; create a separate Store per read model.
//
// Store is safe for concurrent use when the underlying [Backend] is.
// [kv.MemStore] and [pebble.KVAdapter] both are.
//
// ctx arguments are accepted to match the codebase-wide store convention;
// the current kv.Backend does not yet honour context cancellation, so a
// cancelled context will not interrupt an in-flight Get or Set. This will
// change when the Backend gains context support.
type Store[T any, K fmt.Stringer] struct {
	backend Backend
	codec   codec.Codec
	prefix  []byte
	keyFunc func(K) []byte
}

// New creates a [Store] over backend, applying the given options.
// The default codec is [codec.JSONCodec]; the default key encoding is the
// key's String() form.
func New[T any, K fmt.Stringer](backend Backend, opts ...Option[T, K]) *Store[T, K] {
	s := &Store[T, K]{
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
// Returns [ErrNotFound] (alias for [kv.ErrNotFound]) if no value exists.
func (s *Store[T, K]) Get(_ context.Context, id K) (*T, error) {
	data, err := s.backend.Get(s.key(id))
	if err != nil {
		return nil, err
	}

	var val T

	if err := s.codec.Decode(data, &val); err != nil {
		return nil, fmt.Errorf("readmodel: decode key %q: %w", s.key(id), err)
	}

	return &val, nil
}

// Has reports whether a value exists for id without reading it.
func (s *Store[T, K]) Has(_ context.Context, id K) (bool, error) {
	return s.backend.Has(s.key(id))
}

// Set encodes val and stores it under id, replacing any existing value.
func (s *Store[T, K]) Set(_ context.Context, id K, val *T) error {
	if val == nil {
		return fmt.Errorf("readmodel: Set called with nil value for key %q", s.key(id))
	}

	data, err := s.codec.Encode(val)
	if err != nil {
		return fmt.Errorf("readmodel: encode key %q: %w", s.key(id), err)
	}

	return s.backend.Set(s.key(id), data)
}

// Delete removes the value for id. Deleting a missing key is a no-op.
func (s *Store[T, K]) Delete(_ context.Context, id K) error {
	return s.backend.Delete(s.key(id))
}

// Scan returns all values whose keys start with the store's key prefix
// (if set via [WithKeyPrefix]) concatenated with prefix. Values are returned
// in lexicographic key order as yielded by the [Backend] iterator.
//
// Pass an empty prefix to scan every key in this store's namespace.
// The caller owns the returned slice; the store does not retain it.
func (s *Store[T, K]) Scan(_ context.Context, prefix []byte) ([]*T, error) {
	scanPrefix := prefix

	if len(s.prefix) > 0 {
		scanPrefix = append(append([]byte{}, s.prefix...), prefix...)
	}

	iter, err := s.backend.NewIterator(scanPrefix)
	if err != nil {
		return nil, fmt.Errorf("readmodel: scan iterator: %w", err)
	}

	defer func() { _ = iter.Close() }()

	results := make([]*T, 0)

	for iter.Next() {
		var val T

		if err := s.codec.Decode(iter.Value(), &val); err != nil {
			return nil, fmt.Errorf("readmodel: scan decode key %q: %w", iter.Key(), err)
		}

		results = append(results, &val)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("readmodel: scan iteration: %w", err)
	}

	return results, nil
}

// Backend returns the underlying [Backend] the store reads from and writes to.
// Exposed so callers can access backend-specific functionality (batches,
// iterators over raw keys) when the typed API is insufficient.
func (s *Store[T, K]) Backend() Backend { return s.backend }

func (s *Store[T, K]) key(id K) []byte {
	k := s.keyFunc(id)

	if len(s.prefix) > 0 {
		return append(append([]byte{}, s.prefix...), k...)
	}

	return k
}
