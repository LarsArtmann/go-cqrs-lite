package event

// ShouldSnapshot returns true if a snapshot should be created based on the
// given strategy, snapshot store availability, codec availability, and
// the aggregate's current type and version.
func ShouldSnapshot(
	strategy SnapshotStrategy,
	snapshotStore SnapshotStore,
	codec Codec,
	aggType AggregateType,
	version Version,
) bool {
	return strategy != nil &&
		snapshotStore != nil &&
		codec != nil &&
		strategy.ShouldSnapshot(aggType, version)
}
