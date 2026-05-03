package event

import (
	"context"
	"fmt"
	"time"

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
			return fmt.Errorf("stage events in outbox: %w", err)
		}
	} else {
		err := publisher.Publish(ctx, events...)
		if err != nil {
			return fmt.Errorf("publish events: %w", err)
		}
	}

	return nil
}

// SaveSnapshot persists a snapshot with the given pre-encoded state.
// The caller is responsible for encoding the state before calling.
func SaveSnapshot(
	ctx context.Context,
	store SnapshotStore,
	aggType AggregateType,
	aggID id.AggregateID,
	version Version,
	state []byte,
) error {
	err := store.Save(ctx, Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       version,
		State:         state,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("save snapshot for %s %s: %w", aggType, aggID, err)
	}

	return nil
}
