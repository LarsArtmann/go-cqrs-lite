package kv

import (
	"context"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
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
// The default codec is [codec.CBORCodec]; the default key encoding is the
// key's String() form. Pre-envelope data (raw JSON) is auto-detected on read.
func NewTypedStore[T any, K fmt.Stringer](
	backend Store,
	opts ...TypedOption[T, K],
) *TypedStore[T, K] {
	s := &TypedStore[T, K]{ //nolint:exhaustruct // prefix set via WithKeyPrefix option
		backend: backend,
		codec:   codec.CBORCodec{},
		keyFunc: func(k K) []byte { return []byte(k.String()) },
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Get retrieves the value for id, decoding it into a new T.
// Returns [ErrNotFound] if no value exists.
func (s *TypedStore[T, K]) Get(ctx context.Context, id K) (*T, error) {
	data, err := s.backend.Get(ctx, s.key(id))
	if err != nil {
		return nil, errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.get", fmt.Sprintf("get key %q", s.key(id)))
	}

	var val T

	c, inner := codec.UnwrapDecode(data, codec.JSONCodec{})

	err = c.Decode(inner, &val)
	if err != nil {
		return nil, errorfamily.WrapCorruption(err, "kv.typed_store.decode",
			fmt.Sprintf("decode key %q", s.key(id)))
	}

	return &val, nil
}

// Has reports whether a value exists for id without reading it.
func (s *TypedStore[T, K]) Has(ctx context.Context, id K) (bool, error) {
	has, err := s.backend.Has(ctx, s.key(id))
	if err != nil {
		return false, errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.has", fmt.Sprintf("has key %q", s.key(id)))
	}

	return has, nil
}

// Set encodes val and stores it under id, replacing any existing value.
func (s *TypedStore[T, K]) Set(ctx context.Context, id K, val *T) error {
	if val == nil {
		return errorfamily.WrapRejection(errNilTypedValue, "kv.typed_store.set_nil",
			fmt.Sprintf("nil value for key %q", s.key(id)))
	}

	data, err := codec.WrapEncode(val, s.codec)
	if err != nil {
		return errorfamily.WrapCorruption(err, "kv.typed_store.encode",
			fmt.Sprintf("encode key %q", s.key(id)))
	}

	err = s.backend.Set(ctx, s.key(id), data)
	if err != nil {
		return errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.set", fmt.Sprintf("set key %q", s.key(id)))
	}

	return nil
}

// Delete removes the value for id. Deleting a missing key is a no-op.
func (s *TypedStore[T, K]) Delete(ctx context.Context, id K) error {
	err := s.backend.Delete(ctx, s.key(id))
	if err != nil {
		return errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.delete", fmt.Sprintf("delete key %q", s.key(id)))
	}

	return nil
}

// Scan returns all values whose keys start with the store's key prefix
// (if set via [WithTypedKeyPrefix]) concatenated with prefix. Values are returned
// in lexicographic key order as yielded by the [Store] iterator.
//
// Pass an empty prefix to scan every key in this store's namespace.
// The caller owns the returned slice; the store does not retain it.
func (s *TypedStore[T, K]) Scan(ctx context.Context, prefix []byte) ([]*T, error) {
	scanPrefix := prefix

	if len(s.prefix) > 0 {
		scanPrefix = append(append([]byte{}, s.prefix...), prefix...)
	}

	iter, err := s.backend.NewIterator(ctx, scanPrefix)
	if err != nil {
		return nil, errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.scan_iter", "create scan iterator")
	}

	defer func() { _ = iter.Close() }()

	results := make([]*T, 0)

	for iter.Next() {
		var val T

		c, inner := codec.UnwrapDecode(iter.Value(), codec.JSONCodec{})

		err = c.Decode(inner, &val)
		if err != nil {
			return nil, errorfamily.WrapCorruption(err, "kv.typed_store.scan_decode",
				fmt.Sprintf("decode key %q", iter.Key()))
		}

		results = append(results, &val)
	}

	err = iter.Error()
	if err != nil {
		return nil, errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.scan_error", "scan iteration")
	}

	return results, nil
}

// Backend returns the underlying [Store] the typed store reads from and writes to.
// Exposed so callers can access backend-specific functionality (batches,
// iterators over raw keys) when the typed API is insufficient.
func (s *TypedStore[T, K]) Backend() Store { return s.backend }

// DeleteAll removes all values in this store's namespace (key prefix). This
// implements [ViewResetter] and is used for projection resets — wiping a read
// model before rebuilding it from the event journal.
//
// The operation iterates all keys and deletes them via a [Batch] when the
// backend supports it (atomic), otherwise one-by-one.
func (s *TypedStore[T, K]) DeleteAll(ctx context.Context) error {
	iter, err := s.backend.NewIterator(ctx, s.prefix)
	if err != nil {
		return errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.delete_all_iter", "create delete-all iterator")
	}

	keys := make([][]byte, 0)

	for iter.Next() {
		keys = append(keys, append([]byte{}, iter.Key()...))
	}

	if err = iter.Error(); err != nil {
		_ = iter.Close()

		return errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.delete_all_error", "delete-all iteration")
	}

	if err = iter.Close(); err != nil {
		return errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.delete_all_close", "close delete-all iterator")
	}

	if len(keys) == 0 {
		return nil
	}

	batch, batchErr := s.backend.Batch(ctx)
	if batchErr != nil {
		return s.deleteAllOneByOne(ctx, keys)
	}

	defer func() { _ = batch.Close() }()

	for _, k := range keys {
		err = batch.Delete(ctx, k)
		if err != nil {
			return errorfamily.Wrap(err, errorfamily.Classify(err),
				"kv.typed_store.delete_all_batch", "batch delete key")
		}
	}

	if err = batch.Commit(ctx); err != nil {
		return errorfamily.Wrap(err, errorfamily.Classify(err),
			"kv.typed_store.delete_all_commit", "commit delete-all batch")
	}

	return nil
}

func (s *TypedStore[T, K]) deleteAllOneByOne(ctx context.Context, keys [][]byte) error {
	for _, k := range keys {
		err := s.backend.Delete(ctx, k)
		if err != nil {
			return errorfamily.Wrap(err, errorfamily.Classify(err),
				"kv.typed_store.delete_one", fmt.Sprintf("delete key %q", k))
		}
	}

	return nil
}

func (s *TypedStore[T, K]) key(id K) []byte {
	k := s.keyFunc(id)

	if len(s.prefix) > 0 {
		return append(append([]byte{}, s.prefix...), k...)
	}

	return k
}
