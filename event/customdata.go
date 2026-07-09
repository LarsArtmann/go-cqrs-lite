package event

import "github.com/larsartmann/go-cqrs-lite/metadata/v3"

// Deprecated: Use metadata.CustomData[K] directly. This alias exists for
// backward compatibility and will be removed in v4.
type CustomData[K ~string] = metadata.CustomData[K]
