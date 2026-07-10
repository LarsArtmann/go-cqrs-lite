package event

import "github.com/larsartmann/go-cqrs-lite/metadata/v3"

// Deprecated: Use metadata.CustomData[K] directly. This alias exists for
// backward compatibility and will be removed in v4.
// v4-removal: remove this alias and update all consumers to import metadata/ directly.
type CustomData[K ~string] = metadata.CustomData[K]
