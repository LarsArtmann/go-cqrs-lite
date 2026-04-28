// Package id provides strongly-typed identifiers for domain entities.
//
// Using branded types prevents mixing up IDs of different entities at compile time.
// For example, you cannot accidentally pass a UserID where an AggregateID is expected.
//
// Example usage:
//
//	type userAggregate struct{}
//	type UserAggregate = id.Of[userAggregate]
//
//	userID := id.New[UserAggregate]()
//	aggregateID := id.New[AggregateID]()
//	// userID = aggregateID // Compile error!
package id

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"time"

	cbid "github.com/larsartmann/go-composable-business-types/id"
	"github.com/oklog/ulid/v2"
)

// Sentinel errors for id package.
var (
	errEmptyString     = errors.New("empty string")
	errNilReceiver     = errors.New("nil receiver")
	errUnsupportedType = errors.New("unsupported type")
)

// Of is a branded type for strongly-typed identifiers backed by ULID.
// The type parameter T is a phantom type used only for type differentiation.
type Of[T any] struct {
	wrapped cbid.ID[T, ulid.ULID]
}

func newULID() ulid.ULID {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
}

// New generates a new random ULID-backed ID.
func New[T any]() Of[T] {
	return Of[T]{wrapped: cbid.NewID[T, ulid.ULID](newULID())}
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

	return Of[T]{wrapped: cbid.NewID[T, ulid.ULID](id)}, nil
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

// ULID returns the timestamp encoded in the ID, if it is a valid ULID.
// Returns an error if the ID is not a valid ULID.
func ULID(id Of[struct{}]) (time.Time, error) {
	return ulid.Time(id.wrapped.Get().Time()), nil
}

// MaxULIDsPerMs is exported for testing/benchmarking.
const MaxULIDsPerMs = math.MaxInt

// IsZero returns true if the ID has its zero value.
func (id Of[T]) IsZero() bool { return id.wrapped.IsZero() }

// Equal returns true if this ID equals the other ID.
func (id Of[T]) Equal(other Of[T]) bool { return id.wrapped.Equal(other.wrapped) }

// Compare returns -1 if id < other, 0 if equal, 1 if id > other.
// Uses ULID's Compare method for lexicographic byte comparison.
func (id Of[T]) Compare(other Of[T]) int {
	return id.wrapped.Get().Compare(other.wrapped.Get())
}

// Get returns the underlying ULID value.
func (id Of[T]) Get() ulid.ULID { return id.wrapped.Get() }

// Or returns the ID if not zero, otherwise returns the provided default.
func (id Of[T]) Or(defaultValue Of[T]) Of[T] {
	if id.IsZero() {
		return defaultValue
	}

	return id
}

// Reset sets the ID to its zero value.
func (id *Of[T]) Reset() { id.wrapped.Reset() }

// String returns the canonical ULID string representation.
func (id Of[T]) String() string { return id.wrapped.Get().String() }

// GoString implements fmt.GoStringer for debugging.
func (id Of[T]) GoString() string { return id.String() }

// Compile-time interface assertions for core interfaces.
var (
	_ fmt.Stringer   = Of[struct{}]{wrapped: cbid.ID[struct{}, ulid.ULID]{}}
	_ fmt.GoStringer = Of[struct{}]{wrapped: cbid.ID[struct{}, ulid.ULID]{}}
)
