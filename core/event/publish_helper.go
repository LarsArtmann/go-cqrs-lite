package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// PublishChanges publishes events via the publisher.
func PublishChanges(
	ctx context.Context,
	publisher Publisher,
	events []Event,
) error {
	err := publisher.Publish(ctx, events...)
	if err != nil {
		return WrapInfrastructure(
			err,
			"event.publish_failed",
			"publish events",
		)
	}

	return nil
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
