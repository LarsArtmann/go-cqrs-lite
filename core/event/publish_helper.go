package event

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

// PublishChanges publishes events via the outbox if configured,
// or directly via the publisher otherwise.
func PublishChanges(
	ctx context.Context,
	publisher Publisher,
	outbox Outbox,
	events []Event,
) error {
	if outbox != nil {
		err := outbox.Append(ctx, events)
		if err != nil {
			return WrapInfrastructure(
				err,
				"event.outbox_stage_failed",
				"stage events in outbox",
			)
		}
	} else {
		err := publisher.Publish(ctx, events...)
		if err != nil {
			return WrapInfrastructure(
				err,
				"event.publish_failed",
				"publish events",
			)
		}
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
