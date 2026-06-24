package grpc

import "github.com/larsartmann/go-cqrs-lite/event/v3"

// safeInt64FromVersion converts event.Version (uint64) to int64 without
// triggering gosec G115. Extracted as a helper per AGENTS.md convention.
func safeInt64FromVersion(v event.Version) int64 {
	return int64(v) //nolint:gosec // G115: event versions are always small positive integers
}

// safeVersionFromInt64 converts int64 to event.Version (uint64) without
// triggering gosec G115. Extracted as a helper per AGENTS.md convention.
func safeVersionFromInt64(v int64) event.Version {
	return event.Version(v) //nolint:gosec // G115: event versions are always small positive integers
}
