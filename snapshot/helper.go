package snapshot

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func ShouldSnapshot(
	strategy SnapshotStrategy,
	sink SnapshotSink,
	c codec.Codec,
	aggType id.StreamType,
	version event.Version,
) bool {
	return strategy != nil &&
		sink != nil &&
		c != nil &&
		strategy.ShouldSnapshot(aggType, version)
}

// ShouldSnapshotFor evaluates whether to snapshot a specific aggregate.
// If the strategy implements AggregateAwareStrategy, it delegates to
// ShouldSnapshotFor (passing the full ref); otherwise it falls back to
// ShouldSnapshot. This allows per-aggregate strategies like ReadPressure
// to coexist with simple strategies like EveryNEvents.
func ShouldSnapshotFor(
	strategy SnapshotStrategy,
	sink SnapshotSink,
	c codec.Codec,
	ref id.StreamRef,
	version event.Version,
) bool {
	if strategy == nil || sink == nil || c == nil {
		return false
	}

	if s, ok := strategy.(AggregateAwareStrategy); ok {
		return s.ShouldSnapshotFor(ref, version)
	}

	return strategy.ShouldSnapshot(ref.Type, version)
}

func SaveSnapshot(
	ctx context.Context,
	sink SnapshotSink,
	aggType id.StreamType,
	aggID id.StreamID,
	version event.Version,
	state []byte,
) error {
	err := sink.Save(ctx, Snapshot{
		StreamID:   aggID,
		StreamType: aggType,
		Version:    version,
		State:      state,
		CreatedAt:  time.Now().UTC(),
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
