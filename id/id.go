package id

import (
	"crypto/rand"
	"sync"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/oklog/ulid/v2"
)

// Of is a branded type for strongly-typed identifiers backed by ULID.
// The type parameter T is a phantom type used only for type differentiation.
//
// Of aliases go-branded-id's ID[T, ulid.ULID], inheriting all serialization
// (JSON, SQL, Text, Binary, Gob) and utility methods (IsZero, Equal, Or,
// Reset, Get, Ptr, FromPtr, String, GoString, Format, etc.).
type Of[T any] = cbid.ID[T, ulid.ULID]

// ulidMu guards the monotonic entropy source to ensure thread-safe
// ULID generation with guaranteed ordering within the same millisecond.
//
//nolint:gochecknoglobals // guards the monotonic entropy source
var ulidMu sync.Mutex

// mono is a monotonic entropy source that guarantees ULIDs generated within
// the same millisecond are monotonically ordered — critical for event sourcing
// where event ordering must be deterministic.
//
//nolint:gochecknoglobals // package-level monotonic entropy, guarded by ulidMu
//nolint:gochecknoglobals // thread-safe monotonic entropy source
var mono = ulid.Monotonic(rand.Reader, 0)

func newULID() ulid.ULID {
	ulidMu.Lock()
	defer ulidMu.Unlock()

	return ulid.MustNew(ulid.Timestamp(time.Now()), mono)
}

// New generates a new random ULID-backed ID.
func New[T any]() Of[T] {
	return cbid.NewID[T](newULID())
}

// Parse converts a ULID string to a strongly-typed ID.
// Returns an error if the input is not a valid ULID.
func Parse[T any](s string) (Of[T], error) {
	if s == "" {
		var zero Of[T]

		return zero, errorfamily.Wrapf(
			errEmptyString,
			errorfamily.Rejection,
			"id.parse_empty",
			"cannot parse empty string as %T",
			zero,
		)
	}

	id, err := ulid.Parse(s)
	if err != nil {
		var zero Of[T]

		return zero, errorfamily.Wrapf(
			err,
			errorfamily.Rejection,
			"id.parse_ulid",
			"cannot parse %q as ULID for %T",
			s,
			zero,
		)
	}

	return cbid.NewID[T](id), nil
}

// ULID returns the timestamp encoded in the ID.
func ULID[T any](id Of[T]) time.Time {
	return ulid.Time(id.Get().Time())
}

// CompareIDs compares two branded IDs by their ULID values.
// Use this instead of the built-in Compare method, which does not
// support ULID types.
func CompareIDs[T any](a, b Of[T]) int {
	return a.Get().Compare(b.Get())
}

// FromPtr dereferences a pointer-to-ID, returning the zero value if the pointer is nil.
func FromPtr[T any](p *Of[T]) Of[T] {
	return cbid.FromPtr(p)
}
