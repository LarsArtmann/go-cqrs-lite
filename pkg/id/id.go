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
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Of is a branded type for strongly-typed identifiers.
// The type parameter T is a phantom type used only for type differentiation.
type Of[T any] string

// New generates a new random ID using UUID v4.
func New[T any]() Of[T] {
	return Of[T](uuid.New().String())
}

// NewWithPrefix generates a new ID with a human-readable prefix.
func NewWithPrefix[T any](prefix string) Of[T] {
	return Of[T](prefix + "_" + uuid.New().String())
}

// Parse converts a string to a strongly-typed ID.
// Returns an error if the input is empty.
func Parse[T any](s string) (Of[T], error) {
	if s == "" {
		var zero Of[T]
		return zero, fmt.Errorf("cannot parse empty string as %T", zero)
	}
	return Of[T](s), nil
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
	id, err := Parse[T](s)
	if err != nil {
		panic(fmt.Sprintf("id.MustParse: %v (input: %q)", err, s))
	}
	return id
}

// String returns the underlying string value.
func (id Of[T]) String() string {
	return string(id)
}

// IsEmpty returns true if the ID is empty.
func (id Of[T]) IsEmpty() bool {
	return id == ""
}

// IsValid returns true if the ID is not empty.
func (id Of[T]) IsValid() bool {
	return id != ""
}

// MarshalJSON implements json.Marshaler.
func (id Of[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *Of[T]) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("unmarshal ID: %w (input: %q)", err, string(data))
	}
	parsed, err := Parse[T](s)
	if err != nil {
		return fmt.Errorf("parse ID from JSON: %w", err)
	}
	*id = parsed
	return nil
}

// Value implements driver.Valuer for database storage.
func (id Of[T]) Value() (driver.Value, error) {
	return id.String(), nil
}

// Scan implements sql.Scanner for database retrieval.
func (id *Of[T]) Scan(src any) error {
	switch v := src.(type) {
	case string:
		parsed, err := Parse[T](v)
		if err != nil {
			return fmt.Errorf("scan ID from string: %w (input: %q)", err, v)
		}
		*id = parsed
		return nil
	case []byte:
		parsed, err := Parse[T](string(v))
		if err != nil {
			return fmt.Errorf("scan ID from bytes: %w (input: %q)", err, string(v))
		}
		*id = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into %T", src, id)
	}
}
