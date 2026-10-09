package event

import "time"

// SliceFromVersion returns the sub-slice of events starting after the given
// version (exclusive). If version >= len(events), an empty slice is returned.
func SliceFromVersion(events []Event, version Version) []Event {
	if version.Int() >= len(events) {
		return []Event{}
	}

	return events[version.Int():]
}

// SliceToVersion returns the sub-slice of events up to and including maxVersion.
func SliceToVersion(events []Event, maxVersion Version) []Event {
	end := min(maxVersion.Int(), len(events))

	return events[:end]
}

// FilterByTimestamp returns events where OccurredAt <= maxTime.
func FilterByTimestamp(events []Event, maxTime time.Time) []Event {
	filtered := make([]Event, 0, len(events))

	for _, e := range events {
		if !e.OccurredAt().After(maxTime) {
			filtered = append(filtered, e)
		}
	}

	return filtered
}
