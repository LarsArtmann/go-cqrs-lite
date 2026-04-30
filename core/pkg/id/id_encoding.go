//nolint:wrapcheck // pure delegation to wrapped cbid.ID — errors are already well-typed
package id

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"

	cbid "github.com/larsartmann/go-branded-id"
	"github.com/oklog/ulid/v2"
)

func (id Of[T]) MarshalJSON() ([]byte, error)       { return id.wrapped.MarshalJSON() }
func (id *Of[T]) UnmarshalJSON(data []byte) error   { return id.wrapped.UnmarshalJSON(data) }
func (id Of[T]) MarshalBinary() ([]byte, error)     { return id.wrapped.MarshalBinary() }
func (id *Of[T]) UnmarshalBinary(data []byte) error { return id.wrapped.UnmarshalBinary(data) }
func (id Of[T]) MarshalText() ([]byte, error)       { return id.wrapped.MarshalText() }
func (id *Of[T]) UnmarshalText(data []byte) error   { return id.wrapped.UnmarshalText(data) }
func (id *Of[T]) Scan(src any) error                { return id.wrapped.Scan(src) }
func (id Of[T]) Value() (driver.Value, error)       { return id.wrapped.Value() }

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
