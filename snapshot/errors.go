package snapshot

import "github.com/larsartmann/go-cqrs-lite/event/v3"

var (
	ErrSnapshotNotFound    = event.NewRejection("snapshot.not_found", "snapshot not found")
	ErrSnapshotStoreClosed = event.NewInfrastructure(
		"snapshot.store_closed",
		"snapshot store is closed",
	)
	ErrInvalidInterval = event.NewRejection(
		"snapshot.invalid_interval",
		"snapshot interval must be positive",
	)
)
