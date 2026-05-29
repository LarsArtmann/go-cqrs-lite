package snapshot

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/event"
)

type SnapshotStrategy interface {
	ShouldSnapshot(aggregateType event.AggregateType, version event.Version) bool
}

func EveryNEvents(n int) (SnapshotStrategy, error) {
	if n <= 0 {
		return nil, fmt.Errorf("snapshot interval must be positive, got %d", n)
	}

	return &everyN{interval: n}, nil
}

func MustEveryNEvents(n int) SnapshotStrategy {
	s, err := EveryNEvents(n)
	if err != nil {
		panic("snapshot.MustEveryNEvents: " + err.Error())
	}

	return s
}

type everyN struct{ interval int }

var _ SnapshotStrategy = (*everyN)(nil)

func (s *everyN) ShouldSnapshot(_ event.AggregateType, version event.Version) bool {
	return version.IsPositive() && version.Mod(s.interval) == 0
}
