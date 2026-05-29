package event

import "github.com/larsartmann/go-cqrs-lite/codec"

// ShouldSnapshot returns true if a snapshot should be created based on the
// given strategy, snapshot store availability, codec availability, and
// the aggregate's current type and version.
func ShouldSnapshot(
	strategy SnapshotStrategy,
	sink SnapshotSink,
	c codec.Codec,
	aggType AggregateType,
	version Version,
) bool {
	return strategy != nil &&
		sink != nil &&
		c != nil &&
		strategy.ShouldSnapshot(aggType, version)
}
