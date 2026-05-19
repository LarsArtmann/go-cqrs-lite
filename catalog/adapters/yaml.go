package adapters

import (
	"encoding/json"

	"github.com/go-faster/yaml"
)

// JSONToYAML converts JSON bytes to YAML bytes.
func JSONToYAML(jsonBytes []byte) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(jsonBytes, &obj); err != nil {
		return nil, err
	}

	return yaml.Marshal(obj)
}
