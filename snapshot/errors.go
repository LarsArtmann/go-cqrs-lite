package snapshot

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	ErrSnapshotNotFound    = errorfamily.NewRejection("snapshot.not_found", "snapshot not found")
	ErrSnapshotStoreClosed = errorfamily.NewInfrastructure(
		"snapshot.store_closed",
		"snapshot store is closed",
	)
	ErrInvalidInterval = errorfamily.NewRejection(
		"snapshot.invalid_interval",
		"snapshot interval must be positive",
	)
)
