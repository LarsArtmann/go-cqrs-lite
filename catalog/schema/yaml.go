package schema

import (
	"encoding/json/v2"

	"github.com/go-faster/yaml"
	errorfamily "github.com/larsartmann/go-error-family"
)

func JSONToYAML(jsonBytes []byte) ([]byte, error) {
	var obj any

	err := json.Unmarshal(jsonBytes, &obj)
	if err != nil {
		return nil, errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.yaml.1",
			"failed to unmarshal JSON: %v",
			err,
		)
	}

	out, err := yaml.Marshal(obj)
	if err != nil {
		return nil, errorfamily.Newf(
			errorfamily.Infrastructure,
			"catalog.yaml.2",
			"failed to marshal YAML: %v",
			err,
		)
	}

	return out, nil
}
