package decider

import "github.com/larsartmann/go-cqrs-lite/event/v2"

// ErrNilStore is returned by NewRepository when the event store is nil.
var ErrNilStore = event.NewInfrastructure(
	"decider.nil_store",
	"event store is required",
)

// ErrNilBus is returned by NewRepository when the event publisher is nil.
var ErrNilBus = event.NewInfrastructure(
	"decider.nil_bus",
	"event bus is required",
)

// ErrNilFold is returned by NewRepository when the decider Fold function is nil.
var ErrNilFold = event.NewRejection(
	"decider.nil_fold",
	"fold function is required",
)

// ErrLoadFailed is returned when loading events from the store fails.
var ErrLoadFailed = event.NewTransient(
	"decider.load_failed",
	"failed to load events",
)

// ErrFoldFailed is returned when folding an event onto state fails.
var ErrFoldFailed = event.NewCorruption(
	"decider.fold_failed",
	"failed to fold events",
)

// ErrSaveFailed is returned when saving events to the store fails.
var ErrSaveFailed = event.NewTransient(
	"decider.save_failed",
	"failed to save events",
)

// ErrIncompleteSnapshotConfig is returned when snapshot strategy is set without snapshot store or codec.
var ErrIncompleteSnapshotConfig = event.NewInfrastructure(
	"decider.incomplete_snapshot_config",
	"snapshot strategy requires both snapshot store and codec",
)
