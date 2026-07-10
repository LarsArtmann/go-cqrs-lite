package snapshot

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

func ShouldSnapshot(
	strategy SnapshotStrategy,
	sink SnapshotSink,
	c codec.Codec,
	aggType id.AggregateType,
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
	aggType id.AggregateType,
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
		return errorfamily.WrapInfrastructure(
			err,
			"snapshot.save_failed",
			"save snapshot for "+string(aggType)+" "+aggID.String(),
		)
	}

	return nil
}
