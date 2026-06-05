package id

import (
	"crypto/rand"
	"fmt"
	"time"

	cbid "github.com/larsartmann/go-branded-id"
	"github.com/oklog/ulid/v2"
)

// Of is a branded type for strongly-typed identifiers backed by ULID.
// The type parameter T is a phantom type used only for type differentiation.
//
// Of aliases go-branded-id's ID[T, ulid.ULID], inheriting all serialization
// (JSON, SQL, Text, Binary, Gob) and utility methods (IsZero, Equal, Or,
// Reset, Get, Ptr, FromPtr, String, GoString, Format, etc.).
type Of[T any] = cbid.ID[T, ulid.ULID]

func newULID() ulid.ULID {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
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

		return zero, fmt.Errorf("cannot parse empty string as %T: %w", zero, errEmptyString)
	}

	id, err := ulid.Parse(s)
	if err != nil {
		var zero Of[T]

		return zero, fmt.Errorf("cannot parse %q as ULID for %T: %w", s, zero, err)
	}

	return cbid.NewID[T](id), nil
}

// MustParse converts a ULID string to a strongly-typed ID, panicking on error.
//
// WARNING: This function panics if the input is not a valid ULID. Use only in:
//   - Test code where the input is guaranteed valid
//   - Initialization code with hardcoded valid IDs
//   - When you explicitly want a panic on invalid input
//
// For production code, prefer Parse[T]() which returns an error.
func MustParse[T any](s string) Of[T] {
	parsed, err := Parse[T](s)
	if err != nil {
		panic(fmt.Sprintf("id.MustParse: %v (input: %q)", err, s))
	}

	return parsed
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
