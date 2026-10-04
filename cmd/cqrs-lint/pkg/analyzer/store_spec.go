package analyzer

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strings"

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

// UnmarshalJSON implements encoding/json/v2's Unmarshaler interface so the
// config accepts a store given as either a string or an array of strings.
// Unknown store names are rejected with the valid list — a typo like
// "sqllite" must fail loudly at config load, not silently degrade rules.
func (s *StoreSpec) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) == 0 {
		return errors.New(
			"store: expected a store name string or an array of store names, got empty value",
		)
	}
	switch trimmed[0] {
	case '"':
		var kind StoreKind
		if err := jsonv2.Unmarshal(data, &kind); err != nil {
			return fmt.Errorf("store: %w", err)
		}
		if err := validateStoreKind(kind); err != nil {
			return err
		}
		s.Kinds = []StoreKind{kind}
		return nil
	case '[':
		var kinds []StoreKind
		if err := jsonv2.Unmarshal(data, &kinds); err != nil {
			return fmt.Errorf("store: %w", err)
		}
		if len(kinds) == 0 {
			return errors.New("store: array form must list at least one backend")
		}
		for _, kind := range kinds {
			if err := validateStoreKind(kind); err != nil {
				return err
			}
		}
		s.Kinds = kinds
		return nil
	default:
		return fmt.Errorf(
			"store: expected a store name string or an array of store names, got %q",
			trimmed[0],
		)
	}
}

func validateStoreKind(kind StoreKind) error {
	if slices.Contains(AllStoreKinds(), kind) {
		return nil
	}
	return fmt.Errorf(
		"store: unknown backend %q (valid: %s)",
		kind,
		joinStoreKindNames(AllStoreKinds()),
	)
}

func joinStoreKindNames(kinds []StoreKind) string {
	parts := make([]string, len(kinds))
	for i, k := range kinds {
		parts[i] = string(k)
	}
	return strings.Join(parts, ", ")
}

// MarshalJSON implements encoding/json/v2's Marshaler interface: a scalar
// for a single backend, an array for mixed pools, so doctor's
// copy-pasteable config suggestions stay minimal.
func (s StoreSpec) MarshalJSON() ([]byte, error) {
	switch len(s.Kinds) {
	case 0:
		return jsonv2.Marshal(StoreNone)
	case 1:
		return jsonv2.Marshal(s.Kinds[0])
	default:
		return jsonv2.Marshal(s.Kinds)
	}
}
