package snapshot

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

type SnapshotStrategy interface {
	ShouldSnapshot(aggregateType id.AggregateType, version event.Version) bool
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
