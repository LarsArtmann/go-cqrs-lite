package snapshot

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type SnapshotStrategy interface {
	ShouldSnapshot(aggregateType id.AggregateType, version event.Version) bool
}

// AggregateAwareStrategy is an optional capability that strategies may implement
// when they need the aggregate identity to make per-aggregate decisions.
//
// If a SnapshotStrategy implements this interface, the Repository calls
// ShouldSnapshotFor (passing the full ref) instead of ShouldSnapshot.
// This enables strategies that track per-aggregate state, such as
// ReadPressure.
type AggregateAwareStrategy interface {
	ShouldSnapshotFor(ref id.AggregateRef, version event.Version) bool
}

// ReadTracker is an optional capability that strategies may implement to
// track aggregate read frequency.
//
// If a SnapshotStrategy implements this interface, the Repository calls
// RecordRead on every successful Load. This enables read-pressure-aware
// strategies like ReadPressure to count reads and trigger snapshots when
// hot-read aggregates accumulate replay cost.
type ReadTracker interface {
	RecordRead(ref id.AggregateRef, version event.Version)
}

func EveryNEvents(n int) (SnapshotStrategy, error) {
	if n <= 0 {
		return nil, ErrInvalidInterval
	}

	return &everyN{interval: n}, nil
}

type everyN struct{ interval int }

var _ SnapshotStrategy = (*everyN)(nil)

func (s *everyN) ShouldSnapshot(_ id.AggregateType, version event.Version) bool {
	return version.IsPositive() && version.Mod(s.interval) == 0
}
