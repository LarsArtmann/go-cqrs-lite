package asyncapi

import (
	"encoding/json"

	"gopkg.in/yaml.v3"

	"github.com/larsartmann/go-cqrs-lite/catalog/v2"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/internal/caseutil"
	"github.com/larsartmann/go-cqrs-lite/catalog/v2/schema"
)

func SchemaToAny(s *catalog.Schema) any {
	return schema.ToAny(s)
}

func dotSeparated(s string) string {
	return caseutil.DotSeparated(s)
}

func toExamples(raw []json.RawMessage) []Example {
	if len(raw) == 0 {
		return nil
	}

	examples := make([]Example, len(raw))

	for i, r := range raw {
		examples[i] = Example{Payload: r, Summary: ""}
	}

	return examples
}

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
	return json.MarshalIndent((*alias)(d), "", "  ")
}
