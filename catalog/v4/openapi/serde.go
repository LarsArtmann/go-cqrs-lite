package openapi

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/go-faster/yaml"
)

// MarshalYAML serializes the document to YAML format.
func (d *Document) MarshalYAML() ([]byte, error) {
	//nolint:wrapcheck // MarshalYAML returns bytes; caller handles error
	return yaml.Marshal(d)
}

// MarshalJSON serializes the document to JSON format.
// Uses type alias to avoid infinite recursion.
func (d *Document) MarshalJSON() ([]byte, error) {
	type alias Document

	//nolint:wrapcheck // MarshalJSON returns bytes; caller handles error
	return json.Marshal(
		(*alias)(d),
		json.Deterministic(true),
		jsontext.WithIndentPrefix(""),
		jsontext.WithIndent("  "),
	)
}
