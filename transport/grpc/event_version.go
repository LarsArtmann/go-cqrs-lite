package grpc

import "github.com/larsartmann/go-cqrs-lite/event/v4"

// safeInt64FromVersion converts event.Version (uint64) to int64. Isolated as a
// helper per AGENTS.md convention so any future overflow hardening has one home.
func safeInt64FromVersion(v event.Version) int64 {
	return int64(v)
}

// safeVersionFromInt64 converts int64 to event.Version (uint64). Isolated as a
// helper per AGENTS.md convention so any future overflow hardening has one home.
func safeVersionFromInt64(v int64) event.Version {
	u := uint64(v)

	return event.Version(u)
}
