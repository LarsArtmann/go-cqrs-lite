package id

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"

	cbid "github.com/larsartmann/go-composable-business-types/id"
	"github.com/oklog/ulid/v2"
)

// MarshalJSON implements json.Marshaler, serializing as a JSON string.
func (id Of[T]) MarshalJSON() ([]byte, error) {
	if id.IsZero() {
		return []byte("null"), nil
	}

	// Manually construct JSON string to avoid json.Marshal double-encoding.
	// ULID uses Crockford Base32 (0-9, A-Z except I/L/O/U) — no JSON escaping needed.
	return []byte(`"` + id.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler, parsing from a JSON string.
func (id *Of[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		id.Reset()

		return nil
	}

	var s string

	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("id: cannot unmarshal %s into ULID-based ID: %w", string(data), err)
	}

	parsed, err := ulid.Parse(s)
	if err != nil {
		return fmt.Errorf("id: cannot unmarshal %q as ULID: %w", s, err)
	}

	*id = Of[T]{wrapped: cbid.NewID[T, ulid.ULID](parsed)}

	return nil
}

// Scan implements sql.Scanner, supporting string and []byte sources.
func (id *Of[T]) Scan(src any) error {
	if id == nil {
		return fmt.Errorf("id: scan: receiver is nil: %w", errNilReceiver)
	}

	if src == nil {
		id.Reset()

		return nil
	}

	var s string

	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("id: cannot scan %T into ULID-based ID: %w", src, errUnsupportedType)
	}

	parsed, err := ulid.Parse(s)
	if err != nil {
		return fmt.Errorf("id: cannot scan %q as ULID: %w", s, err)
	}

	*id = Of[T]{wrapped: cbid.NewID[T, ulid.ULID](parsed)}

	return nil
}

// Value implements driver.Valuer, returning the ULID as a string for SQL storage.
func (id Of[T]) Value() (driver.Value, error) {
	if id.IsZero() {
		return nil, nil //nolint:nilnil // SQL NULL is represented as (nil, nil) per database/sql convention
	}

	return id.String(), nil
}

// MarshalBinary implements encoding.BinaryMarshaler, returning 16-byte big-endian ULID.
func (id Of[T]) MarshalBinary() ([]byte, error) {
	if id.IsZero() {
		return nil, nil //nolint:nilnil // consistent with Value() — SQL NULL convention
	}

	val, err := id.wrapped.Get().MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("id: MarshalBinary: %w", err)
	}

	return val, nil
}

// ulidBinarySize is the size in bytes of a ULID in binary form.
const ulidBinarySize = 16

// UnmarshalBinary implements encoding.BinaryUnmarshaler, parsing 16-byte big-endian ULID.
func (id *Of[T]) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		id.Reset()

		return nil
	}

	if len(data) != ulidBinarySize {
		return fmt.Errorf("id: insufficient data for ULID: %w", ulid.ErrDataSize)
	}

	var inner ulid.ULID

	err := inner.UnmarshalBinary(data)
	if err != nil {
		return fmt.Errorf("id: cannot unmarshal binary as ULID: %w", err)
	}

	*id = Of[T]{wrapped: cbid.NewID[T, ulid.ULID](inner)}

	return nil
}

// MarshalText implements encoding.TextMarshaler, returning the ULID string.
func (id Of[T]) MarshalText() ([]byte, error) {
	if id.IsZero() {
		return nil, nil //nolint:nilnil // consistent with Value() — SQL NULL convention
	}

	return []byte(id.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler, parsing a ULID string.
func (id *Of[T]) UnmarshalText(data []byte) error {
	if len(data) == 0 {
		id.Reset()

		return nil
	}

	parsed, err := ulid.Parse(string(data))
	if err != nil {
		return fmt.Errorf("id: cannot unmarshal text as ULID: %w", err)
	}

	*id = Of[T]{wrapped: cbid.NewID[T, ulid.ULID](parsed)}

	return nil
}

// Compile-time interface assertions for encoding and SQL interfaces.
var (
	_ json.Marshaler             = Of[struct{}]{wrapped: cbid.ID[struct{}, ulid.ULID]{}}
	_ json.Unmarshaler           = (*Of[struct{}])(nil)
	_ sql.Scanner                = (*Of[struct{}])(nil)
	_ driver.Valuer              = Of[struct{}]{wrapped: cbid.ID[struct{}, ulid.ULID]{}}
	_ encoding.BinaryMarshaler   = Of[struct{}]{wrapped: cbid.ID[struct{}, ulid.ULID]{}}
	_ encoding.BinaryUnmarshaler = (*Of[struct{}])(nil)
	_ encoding.TextMarshaler     = Of[struct{}]{wrapped: cbid.ID[struct{}, ulid.ULID]{}}
	_ encoding.TextUnmarshaler   = (*Of[struct{}])(nil)
)
