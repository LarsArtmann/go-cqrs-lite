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
	"encoding"
	"fmt"

	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
)

// Of is a branded type for strongly-typed identifiers.
// The type parameter T is a phantom type used only for type differentiation.
type Of[T any] string

// New generates a new random ID using UUID v4.
func New[T any]() Of[T] {
	return Of[T](uuid.New().String())
}

// PrefixString is a string type for human-readable ID prefixes.
type PrefixString string

// NewWithPrefix generates a new ID with a human-readable prefix.
func NewWithPrefix[T any](prefix PrefixString) Of[T] {
	return Of[T](string(prefix) + "_" + uuid.New().String())
}

// Parse converts a string to a strongly-typed ID.
// Returns an error if the input is empty.
func Parse[T any](s string) (Of[T], error) {
	if s == "" {
		var zero Of[T]
		return zero, fmt.Errorf("cannot parse empty string as %T (input: %q)", zero, s)
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

// Equal returns true if this ID equals another.
func (id Of[T]) Equal(other Of[T]) bool {
	return id == other
}

// Compare returns -1, 0, or 1 depending on whether id is less than, equal to,
// or greater than other.
func (id Of[T]) Compare(other Of[T]) int {
	if id < other {
		return -1
	}
	if id > other {
		return 1
	}
	return 0
}

// Or returns the first non-empty ID. If id is empty, returns fallback.
func (id Of[T]) Or(fallback Of[T]) Of[T] {
	if id.IsEmpty() {
		return fallback
	}
	return id
}

// Reset clears the ID to its zero value.
func (id *Of[T]) Reset() {
	*id = ""
}

// GoString returns a Go-syntax representation of the ID for use in %#v formatting.
func (id Of[T]) GoString() string {
	return fmt.Sprintf("id.Of[%T](%q)", id, id.String())
}

// Format implements fmt.Formatter for custom formatting.
func (id Of[T]) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('#') {
			_, _ = fmt.Fprint(f, id.GoString())
			return
		}
		_, _ = fmt.Fprint(f, id.String())
	case 's':
		_, _ = fmt.Fprint(f, id.String())
	case 'q':
		_, _ = fmt.Fprintf(f, "%q", id.String())
	default:
		_, _ = fmt.Fprintf(f, "%%!%c(id.Of=%s)", verb, id.String())
	}
}

// MarshalJSON implements json.Marshaler.
// Zero-value IDs serialize to null.
func (id Of[T]) MarshalJSON() ([]byte, error) {
	if id.IsEmpty() {
		return []byte("null"), nil
	}
	data, err := json.Marshal(id.String())
	if err != nil {
		return nil, fmt.Errorf("marshal ID to JSON: %w", err)
	}
	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler.
// Supports both null and string values.
func (id *Of[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*id = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("unmarshal ID: %w (input: %q)", err, string(data))
	}
	parsed, err := Parse[T](s)
	if err != nil {
		return fmt.Errorf("parse ID from JSON %q: %w", s, err)
	}
	*id = parsed
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (id Of[T]) MarshalBinary() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (id *Of[T]) UnmarshalBinary(data []byte) error {
	parsed, err := Parse[T](string(data))
	if err != nil {
		return fmt.Errorf("unmarshal ID from binary: %w", err)
	}
	*id = parsed
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (id Of[T]) MarshalText() ([]byte, error) {
	return []byte(id.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (id *Of[T]) UnmarshalText(data []byte) error {
	parsed, err := Parse[T](string(data))
	if err != nil {
		return fmt.Errorf("unmarshal ID from text: %w", err)
	}
	*id = parsed
	return nil
}

var (
	_ encoding.BinaryMarshaler   = Of[struct{}]("")
	_ encoding.BinaryUnmarshaler = (*Of[struct{}])(nil)
	_ encoding.TextMarshaler     = Of[struct{}]("")
	_ encoding.TextUnmarshaler   = (*Of[struct{}])(nil)
)

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
			return fmt.Errorf("scan ID from string: %w (input: %q, src: %T)", err, v, src)
		}
		*id = parsed
		return nil
	case []byte:
		parsed, err := Parse[T](string(v))
		if err != nil {
			return fmt.Errorf("scan ID from bytes: %w (input: %q, src: %T)", err, string(v), src)
		}
		*id = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into %T", src, id)
	}
}
