package snapshot

import "github.com/larsartmann/go-cqrs-lite/event/v3"

type SnapshotStrategy interface {
	ShouldSnapshot(aggregateType event.AggregateType, version event.Version) bool
}

func EveryNEvents(n int) (SnapshotStrategy, error) {
	if n <= 0 {
		return nil, ErrInvalidInterval
	}

	return &everyN{interval: n}, nil
}

type everyN struct{ interval int }

var _ SnapshotStrategy = (*everyN)(nil)

func (s *everyN) ShouldSnapshot(_ event.AggregateType, version event.Version) bool {
	return version.IsPositive() && version.Mod(s.interval) == 0
}
