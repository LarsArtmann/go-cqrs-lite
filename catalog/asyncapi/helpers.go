package asyncapi

import (
	"encoding/json"

	"github.com/go-faster/yaml"
	"github.com/larsartmann/go-cqrs-lite/catalog"
)

// SchemaToAny converts a catalog.Schema to a generic map for JSON serialization.
func SchemaToAny(s *catalog.Schema) any {
	if s == nil {
		return map[string]string{"type": "object"}
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return map[string]string{"type": "object"}
	}

	var result any

	err = json.Unmarshal(raw, &result)
	if err != nil {
		return map[string]string{"type": "object"}
	}

	return result
}

func toDotAddress(s string) string {
	var result []byte

	runes := []rune(s)

	for i, c := range runes {
		switch {
		case c >= 'A' && c <= 'Z':
			if i > 0 {
				prev := runes[i-1]
				prevIsUpper := prev >= 'A' && prev <= 'Z'
				prevIsLower := prev >= 'a' && prev <= 'z'
				nextIsLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'

				if prevIsLower || (prevIsUpper && nextIsLower) {
					result = append(result, '.')
				}
			}

			result = append(result, byte(c+'a'-'A'))
		case c >= '0' && c <= '9':
			if i > 0 {
				prev := runes[i-1]
				isLetter := (prev >= 'a' && prev <= 'z') || (prev >= 'A' && prev <= 'Z')

				if isLetter {
					result = append(result, '.')
				}
			}

			result = append(result, byte(c))
		case c >= 0 && c <= 127:
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
