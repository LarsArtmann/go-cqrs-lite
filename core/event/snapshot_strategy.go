package event

// SnapshotStrategy decides when to create a snapshot after saving events.
type SnapshotStrategy interface {
	// ShouldSnapshot returns true if a snapshot should be created
	// for the given aggregate after it has reached the given version.
	ShouldSnapshot(aggregateType AggregateType, version Version) bool
}

// EveryNEvents creates a SnapshotStrategy that snapshots every N events.
// Returns ErrInvalidSnapshotInterval if n <= 0.
func EveryNEvents(n int) (SnapshotStrategy, error) {
	if n <= 0 {
		return nil, errInvalidSnapshotInterval
	}

	return &everyN{interval: n}, nil
}

// MustEveryNEvents creates a SnapshotStrategy that snapshots every N events.
// Panics if n <= 0. Use only in tests where inputs are guaranteed valid.
func MustEveryNEvents(n int) SnapshotStrategy {
	s, err := EveryNEvents(n)
	if err != nil {
		panic("event.MustEveryNEvents: " + err.Error())
	}

	return s
}

type everyN struct{ interval int }

var _ SnapshotStrategy = (*everyN)(nil)

func (s *everyN) ShouldSnapshot(_ AggregateType, version Version) bool {
	return version.IsPositive() && version.Mod(s.interval) == 0
}
