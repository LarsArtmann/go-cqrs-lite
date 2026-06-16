package indexing

import (
	"regexp"
	"strings"
)

var advisoryRegexes = []*regexp.Regexp{
	searchIndexRe,
	searchCoverRe,
	usingIntegerPK,
	autoIndexRe,
}

var queryPatternsByTable = map[string][]queryPattern{
	"events": {
		{
			Query: "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ? ORDER BY version ASC",
			Args:  []any{"User", "dummy-id"},
		},
		{
			Query: "SELECT * FROM events WHERE aggregate_type = ? AND aggregate_id = ? AND version > ? ORDER BY version ASC",
			Args:  []any{"User", "dummy-id", 1},
		},
		{
			Query: "SELECT * FROM events WHERE event_type = ? ORDER BY occurred_at ASC",
			Args:  []any{"UserCreated"},
		},
		{
			Query: "SELECT * FROM events ORDER BY occurred_at ASC, id ASC LIMIT ?",
			Args:  []any{100},
		},
	},
	"commands": {
		{
			Query: "SELECT * FROM commands WHERE aggregate_type = ? AND aggregate_id = ? ORDER BY received_at ASC",
			Args:  []any{"User", "dummy-id"},
		},
		{
			Query: "SELECT * FROM commands WHERE command_type = ? ORDER BY received_at ASC",
			Args:  []any{"CreateUser"},
		},
	},
	"snapshots": {
		{
			Query: "SELECT * FROM snapshots WHERE aggregate_type = ? AND aggregate_id = ?",
			Args:  []any{"User", "dummy-id"},
		},
	},
	"checkpoints": {
		{
			Query: "SELECT * FROM checkpoints WHERE projection_name = ?",
			Args:  []any{"user-projection"},
		},
	},
}

type indexRule struct {
	needs    []string
	index    Index
	reason   string
	priority Priority
}

var indexInferenceRules = map[string][]indexRule{
	"events": {
		{
			needs: []string{"AGGREGATE_TYPE", "AGGREGATE_ID", "VERSION"},
			index: Index{
				Name:    "idx_events_agg_ver",
				Table:   "events",
				Columns: []string{"aggregate_type", "aggregate_id", "version"},
				Reason:  "avoid full table scan on aggregate version queries",
			},
			reason:   "aggregate load with version filter triggers SCAN TABLE",
			priority: PriorityCritical,
		},
		{
			needs: []string{"EVENT_TYPE"},
			index: Index{
				Name:    "idx_events_type_time",
				Table:   "events",
				Columns: []string{"event_type", "occurred_at"},
				Reason:  "avoid full table scan on event type filter queries",
			},
			reason:   "event type projection queries trigger SCAN TABLE",
			priority: PriorityRecommended,
		},
		{
			needs: []string{"OCCURRED_AT", "ID"},
			index: Index{
				Name:    "idx_events_cursor",
				Table:   "events",
				Columns: []string{"occurred_at", "id"},
				Reason:  "avoid full table scan on cursor pagination",
			},
			reason:   "ReadFrom cursor pagination triggers SCAN TABLE",
			priority: PriorityCritical,
		},
	},
	"commands": {
		{
			needs: []string{"AGGREGATE_TYPE", "AGGREGATE_ID"},
			index: Index{
				Name:    "idx_commands_agg_time",
				Table:   "commands",
				Columns: []string{"aggregate_type", "aggregate_id", "received_at"},
				Reason:  "avoid full table scan on command aggregate queries",
			},
			reason:   "command audit by aggregate triggers SCAN TABLE",
			priority: PriorityRecommended,
		},
		{
			needs: []string{"COMMAND_TYPE"},
			index: Index{
				Name:    "idx_commands_type_time",
				Table:   "commands",
				Columns: []string{"command_type", "received_at"},
				Reason:  "avoid full table scan on command type queries",
			},
			reason:   "command type analytics triggers SCAN TABLE",
			priority: PriorityOptional,
		},
	},
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if !strings.Contains(s, sub) {
			return false
		}
	}

	return true
}
