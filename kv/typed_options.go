package kv

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
)

// errNilTypedValue is returned by [TypedStore.Set] when val is nil.
var errNilTypedValue = errorfamily.NewRejection(
	"kv.nil_value",
	"kv: TypedStore.Set called with a nil value",
)

// TypedOption configures a [TypedStore]. It follows the codebase convention of
// func(*T) with no error return: invalid options are either ignored or panic
// at construction, matching event.Option and snapshot.Option.
type TypedOption[T any, K fmt.Stringer] func(*TypedStore[T, K])

// WithTypedCodec sets the serialization codec used by [TypedStore.Set] and [TypedStore.Get].
// Defaults to [codec.JSONCodec] when constructing a TypedStore directly.
// When using [stack.ReadModel] or [stack.NewMaterialize], the Bundle's
// [stack.Bundle.DefaultCodec] (CBORCodec by default) is used instead.
//
// Pass [codec.CBORCodec] for smaller payloads or [codec.CBORCompactCodec]
// for maximum compactness (see the codec package docs for the toarray tradeoff).
func WithTypedCodec[T any, K fmt.Stringer](c codec.Codec) TypedOption[T, K] {
	return func(s *TypedStore[T, K]) {
		if c != nil {
			s.codec = c
		}
	}
}

// WithTypedKeyPrefix prepends prefix to every key the [TypedStore] reads or writes.
// Use it to namespace multiple read models that share one [Store]:
//
//	todos := kv.NewTypedStore[Todo, TodoID](backend, kv.WithTypedKeyPrefix[Todo, TodoID]("todos:"))
//
// The prefix is applied before the per-record key derived from K.
func WithTypedKeyPrefix[T any, K fmt.Stringer](prefix string) TypedOption[T, K] {
	return func(s *TypedStore[T, K]) {
		s.prefix = []byte(prefix)
	}
}

// WithTypedKeyFunc overrides how a key of type K is encoded to bytes.
// The default is the key's String() form. Override it when you need a
// different on-disk representation, for example storing the raw 16-byte ULID
// instead of its 26-character base32 string form.
//
// The function must produce a stable, unique byte slice for each distinct K.
// Returned slices are not retained by the store after a call completes.
func WithTypedKeyFunc[T any, K fmt.Stringer](fn func(K) []byte) TypedOption[T, K] {
	return func(s *TypedStore[T, K]) {
		if fn != nil {
			s.keyFunc = fn
		}
	}
}
