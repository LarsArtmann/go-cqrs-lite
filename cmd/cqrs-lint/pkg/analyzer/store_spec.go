package analyzer

import (
	"fmt"
	"strings"

	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
)

// StoreSpec is the config representation of one or more persistence
// backends: `"store": "sqlite"` (single) or `"store": ["postgres",
// "sqlite"]` (mixed pools, e.g. a distributed journal plus embedded
// projection engines). The first kind is the primary — the single-store
// compatibility view that rules and doctor output render first.
type StoreSpec struct {
	Kinds []StoreKind
}

// Primary returns the first declared kind, or StoreNone when empty.
func (s StoreSpec) Primary() StoreKind {
	if len(s.Kinds) == 0 {
		return StoreNone
	}
	return s.Kinds[0]
}

// String renders the spec for log and doctor lines: "sqlite" or
// "postgres+sqlite".
func (s StoreSpec) String() string {
	if len(s.Kinds) == 0 {
		return string(StoreNone)
	}
	parts := make([]string, len(s.Kinds))
	for i, k := range s.Kinds {
		parts[i] = string(k)
	}
	return strings.Join(parts, "+")
}

// UnmarshalFrom implements the encoding/json/v2 Unmarshaler interface so the
// config accepts a store given as either a string or an array of strings.
func (s *StoreSpec) UnmarshalFrom(dec *jsontext.Decoder) error {
	switch dec.PeekKind() {
	case '"':
		var kind StoreKind
		if err := jsonv2.UnmarshalDecode(dec, &kind); err != nil {
			return fmt.Errorf("store: %w", err)
		}
		s.Kinds = []StoreKind{kind}
		return nil
	case '[':
		var kinds []StoreKind
		if err := jsonv2.UnmarshalDecode(dec, &kinds); err != nil {
			return fmt.Errorf("store: %w", err)
		}
		if len(kinds) == 0 {
			return fmt.Errorf("store: array form must list at least one backend")
		}
		s.Kinds = kinds
		return nil
	default:
		return fmt.Errorf("store: expected a store name string or an array of store names, got %s", dec.PeekKind())
	}
}

// MarshalTo implements the encoding/json/v2 Marshaler interface: a scalar
// for a single backend, an array for mixed pools, so doctor's
// copy-pasteable config suggestions stay minimal.
func (s StoreSpec) MarshalTo(enc *jsontext.Encoder) error {
	var raw []byte
	var err error
	switch len(s.Kinds) {
	case 0:
		raw, err = jsonv2.Marshal(StoreNone)
	case 1:
		raw, err = jsonv2.Marshal(s.Kinds[0])
	default:
		raw, err = jsonv2.Marshal(s.Kinds)
	}
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	return enc.WriteValue(jsontext.Value(raw))
}
