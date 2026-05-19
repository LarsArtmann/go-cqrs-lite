package adapters

import (
	"encoding/json"
	"fmt"

	"github.com/go-faster/yaml"
)

// JSONToYAML converts JSON bytes to YAML bytes.
func JSONToYAML(jsonBytes []byte) ([]byte, error) {
	var obj any

	err := json.Unmarshal(jsonBytes, &obj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return out, nil
}
