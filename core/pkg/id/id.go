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
	"fmt"
	"math"
	"time"

	cbid "github.com/larsartmann/go-composable-business-types/id"
	"github.com/oklog/ulid/v2"
)

// Of is a branded type for strongly-typed identifiers.
// The type parameter T is a phantom type used only for type differentiation.
type Of[T any] = cbid.ID[T, string]

// PrefixString is a string type for human-readable ID prefixes.
type PrefixString string

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// New generates a new random ID using ULID.
func New[T any]() Of[T] {
	return cbid.NewID[T, string](newULID())
}

// NewWithPrefix generates a new ID with a human-readable prefix.
func NewWithPrefix[T any](prefix PrefixString) Of[T] {
	return cbid.NewID[T, string](string(prefix) + "_" + newULID())
}

// Parse converts a string to a strongly-typed ID.
// Returns an error if the input is empty.
func Parse[T any](s string) (Of[T], error) {
	if s == "" {
		var zero Of[T]

		//nolint:err113 // dynamic error required to include type information
		return zero, fmt.Errorf("cannot parse empty string as %T", zero)
	}

	return cbid.NewID[T, string](s), nil
}

// MustParse converts a string to a strongly-typed ID, panicking on error.
//
// WARNING: This function panics if the input is empty. Use only in:
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
// Returns an error if the ID string is not a valid ULID.
func ULID(id Of[struct{}]) (time.Time, error) {
	parsed, err := ulid.Parse(id.String())
	if err != nil {
		return time.Time{}, fmt.Errorf("parse ULID: %w", err)
	}

	return ulid.Time(parsed.Time()), nil
}

// MaxULIDsPerMs is exported for testing/benchmarking.
const MaxULIDsPerMs = math.MaxInt
