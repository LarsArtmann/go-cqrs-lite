package main

// ──────────────────────────────────────────────────────────────────────────
// Read Model — the materialised view that projections build from events.
//
// TaskView is a flat, query-optimised projection of the Task aggregate.
// It is NOT the aggregate state — it's the data shape the query side serves
// to HTTP clients. The metaengine Map ADT projection (declared in
// metaengine.go) rebuilds it from the event stream.
// ──────────────────────────────────────────────────────────────────────────

// TaskView is the read-model representation served by the query side.
type TaskView struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    Priority `json:"priority"`
	Status      Status   `json:"status"`
	AssigneeID  string   `json:"assigneeId,omitempty"`
	DueDate     string   `json:"dueDate,omitempty"`
	BlockedBy   []string `json:"blockedBy,omitempty"`
	Tombstoned  bool     `json:"tombstoned,omitempty"`
}

// IsTombstoned enables tombstone-aware filtering.
func (v *TaskView) IsTombstoned() bool { return v.Tombstoned }

func removeString(slice []string, target string) []string {
	result := make([]string, 0, len(slice))

	for _, s := range slice {
		if s != target {
			result = append(result, s)
		}
	}

	return result
}
