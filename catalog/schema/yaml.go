package schema

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

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
