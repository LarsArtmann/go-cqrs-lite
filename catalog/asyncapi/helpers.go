package asyncapi

import (
	"encoding/json"

	"github.com/go-faster/yaml"
	"github.com/larsartmann/go-cqrs-lite/catalog"
)

func SchemaToAny(s *catalog.Schema) any {
	if s == nil {
		return map[string]string{"type": "object"}
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return map[string]string{"type": "object"}
	}

	var result any

	_ = json.Unmarshal(raw, &result)

	return result
}

func toDotAddress(s string) string {
	var result []byte

	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '.')
			}

			result = append(result, byte(c+'a'-'A'))
		} else if c >= 0 && c <= 127 {
			result = append(result, byte(c))
		}
	}

	return string(result)
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

func (d *Document) MarshalYAML() ([]byte, error) {
	//nolint:wrapcheck // MarshalYAML returns bytes; caller handles error
	return yaml.Marshal(d)
}

func (d *Document) MarshalJSON() ([]byte, error) {
	type alias Document

	//nolint:wrapcheck // MarshalJSON returns bytes; caller handles error
	return json.MarshalIndent((*alias)(d), "", "  ")
}
