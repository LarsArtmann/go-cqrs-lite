package readmodel

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
)

// Option configures a [Store]. It follows the codebase convention of
// func(*T) with no error return: invalid options are either ignored or panic
// at construction, matching event.Option and snapshot.Option.
type Option[T any, K fmt.Stringer] func(*Store[T, K])

// WithCodec sets the serialization codec used by [Store.Set] and [Store.Get].
// Defaults to [codec.JSONCodec].
//
// Pass [codec.CBORCodec] for smaller payloads or [codec.CBORCompactCodec]
// for maximum compactness (see the codec package docs for the toarray tradeoff).
func WithCodec[T any, K fmt.Stringer](c codec.Codec) Option[T, K] {
	return func(s *Store[T, K]) {
		if c != nil {
			s.codec = c
		}
	}
}

// WithKeyPrefix prepends prefix to every key the [Store] reads or writes.
// Use it to namespace multiple read models that share one [Backend]:
//
//	todos := readmodel.New[Todo, TodoID](backend, readmodel.WithKeyPrefix[Todo, TodoID]("todos:"))
//
// The prefix is applied before the per-record key derived from K.
func WithKeyPrefix[T any, K fmt.Stringer](prefix string) Option[T, K] {
	return func(s *Store[T, K]) {
		s.prefix = []byte(prefix)
	}
}

// WithKeyFunc overrides how a key of type K is encoded to bytes.
// The default is the key's String() form. Override it when you need a
// different on-disk representation, for example storing the raw 16-byte ULID
// instead of its 26-character base32 string form:
//
//	readmodel.WithKeyFunc[Todo, TodoID](func(id TodoID) []byte {
//	    raw, _ := id.Get().MarshalBinary()
//	    return raw
//	})
//
// The function must produce a stable, unique byte slice for each distinct K.
// Returned slices are not retained by the store after a call completes.
func WithKeyFunc[T any, K fmt.Stringer](fn func(K) []byte) Option[T, K] {
	return func(s *Store[T, K]) {
		if fn != nil {
			s.keyFunc = fn
		}
	}
}
