package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/codec"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// ShouldSnapshot returns true if a snapshot should be created based on the
// given strategy, snapshot store availability, codec availability, and
// the aggregate's current type and version.
func ShouldSnapshot(
	strategy SnapshotStrategy,
	sink SnapshotSink,
	c codec.Codec,
	aggType AggregateType,
	version Version,
) bool {
	return strategy != nil &&
		sink != nil &&
		c != nil &&
		strategy.ShouldSnapshot(aggType, version)
}

// SaveSnapshot persists a snapshot with the given pre-encoded state.
// The caller is responsible for encoding the state before calling.
func SaveSnapshot(
	ctx context.Context,
	sink SnapshotSink,
	aggType AggregateType,
	aggID id.AggregateID,
	version Version,
	state []byte,
) error {
	err := sink.Save(ctx, Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       version,
		State:         state,
		CreatedAt:     defaultClock().UTC(),
	})
	if err != nil {
		return WrapInfrastructure(
			err,
			"event.snapshot_save_failed",
			"save snapshot for "+string(aggType)+" "+aggID.String(),
		)
	}

	return nil
}
