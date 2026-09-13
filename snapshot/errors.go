package snapshot

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	ErrInvalidSnapshot error = errorfamily.NewRejection(
		"snapshot.invalid",
		"invalid snapshot",
	)
	ErrSnapshotNotFound error = errorfamily.NewRejection(
		"snapshot.not_found",
		"snapshot not found",
	)
	ErrSnapshotStoreClosed error = errorfamily.NewInfrastructure(
		"snapshot.store_closed",
		"snapshot store is closed",
	)
	ErrInvalidInterval error = errorfamily.NewRejection(
		"snapshot.invalid_interval",
		"snapshot interval must be positive",
	)
	ErrInvalidThreshold error = errorfamily.NewRejection(
		"snapshot.invalid_threshold",
		"read pressure threshold must be positive",
	)
)
