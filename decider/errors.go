package decider

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrNilStore is returned by NewRepository when the event store is nil.
var ErrNilStore = errorfamily.NewInfrastructure(
	"decider.nil_store",
	"event store is required",
)

// ErrNilPublisher is deprecated: NewRepository no longer requires a publisher.
// It is retained for consumers that reference it. A nil publisher enables
// pure event-sourcing mode (persist without publish).
var ErrNilPublisher = errorfamily.NewInfrastructure(
	"decider.nil_publisher",
	"event publisher is required",
)

// ErrNilBus is deprecated: use ErrNilPublisher instead.
var ErrNilBus = ErrNilPublisher

// ErrNilApply is returned by NewRepository when the decider Apply function is nil.
var ErrNilApply = errorfamily.NewRejection(
	"decider.nil_fold",
	"apply function is required",
)

// ErrLoadFailed is returned when loading events from the store fails.
var ErrLoadFailed = errorfamily.NewTransient(
	"decider.load_failed",
	"failed to load events",
)

// ErrApplyFailed is returned when applying an event onto state fails.
var ErrApplyFailed = errorfamily.NewCorruption(
	"decider.fold_failed",
	"failed to apply events",
)

// ErrSaveFailed is returned when saving events to the store fails.
var ErrSaveFailed = errorfamily.NewTransient(
	"decider.save_failed",
	"failed to save events",
)

// ErrIncompleteSnapshotConfig is returned when snapshot strategy is set without snapshot store or codec.
var ErrIncompleteSnapshotConfig = errorfamily.NewInfrastructure(
	"decider.incomplete_snapshot_config",
	"snapshot strategy requires both snapshot store and codec",
)
