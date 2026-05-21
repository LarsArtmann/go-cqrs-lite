package adapters

import (
	"github.com/larsartmann/go-cqrs-lite/catalog/internal/schemautil"
)

// JSONToYAML converts JSON bytes to YAML bytes.
func JSONToYAML(jsonBytes []byte) ([]byte, error) {
	//nolint:wrapcheck // Thin wrapper for backward compatibility
	return schemautil.JSONToYAML(jsonBytes)
}
