package snapshot

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v2"
	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

func ShouldSnapshot(
	strategy SnapshotStrategy,
	sink SnapshotSink,
	c codec.Codec,
	aggType event.AggregateType,
	version event.Version,
) bool {
	return strategy != nil &&
		sink != nil &&
		c != nil &&
		strategy.ShouldSnapshot(aggType, version)
}

func SaveSnapshot(
	ctx context.Context,
	sink SnapshotSink,
	aggType event.AggregateType,
	aggID id.AggregateID,
	version event.Version,
	state []byte,
) error {
	err := sink.Save(ctx, Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       version,
		State:         state,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return event.WrapInfrastructure(
			err,
			"snapshot.save_failed",
			"save snapshot for "+string(aggType)+" "+aggID.String(),
		)
	}

	return nil
}
