package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
)

var (
	scanTableRe    = regexp.MustCompile(`SCAN TABLE\s+(\S+)`)
	searchIndexRe  = regexp.MustCompile(`USING INDEX\s+(\S+)`)
	searchCoverRe  = regexp.MustCompile(`USING COVERING INDEX\s+(\S+)`)
	autoIndexRe    = regexp.MustCompile(`sqlite_autoindex_`)
	usingIntegerPK = regexp.MustCompile(`USING INTEGER PRIMARY KEY`)
)

// PlanRow represents one row from EXPLAIN QUERY PLAN output.
type PlanRow struct {
	ID     int
	Parent int
	Detail string
}

// Recommendation is a suggested index with context.
type Recommendation struct {
	Index        Index
	Explanation  string
	QueryPattern string
}

// AdvisorOption configures an Advisor.
type AdvisorOption func(*Advisor)

// WithExcludedTables prevents the advisor from analyzing the given tables.
func WithExcludedTables(tables ...string) AdvisorOption {
	return func(a *Advisor) {
		for _, t := range tables {
			a.excluded[t] = true
		}
	}
}

// Advisor analyzes SQLite query plans and recommends indexes.
type Advisor struct {
	db *sql.DB
	mu sync.RWMutex

	// existing indexes cached to avoid repeated schema queries
	existing map[string]bool
	excluded map[string]bool
}

// NewAdvisor creates an index advisor for the given database.
func NewAdvisor(db *sql.DB, opts ...AdvisorOption) *Advisor {
	a := &Advisor{
		db:       db,
		existing: make(map[string]bool),
		excluded: make(map[string]bool),
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// AnalyzeQuery runs EXPLAIN QUERY PLAN on the provided query and returns
// recommendations for any full-table scans detected.
func (a *Advisor) AnalyzeQuery(
	ctx context.Context,
	query string,
	args ...any,
) ([]Recommendation, error) {
	plan, err := a.explain(ctx, query, args...)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "indexing.explain",
			"explain query plan")
	}

	return a.recommendationsFromPlan(plan, query), nil
}

// AnalyzeTable checks common CQRS query patterns against a table and
// returns recommendations.
func (a *Advisor) AnalyzeTable(
	ctx context.Context,
	table string,
) ([]Recommendation, error) {
	patterns := a.tableQueryPatterns(table)

	var all []Recommendation

	for _, pat := range patterns {
		recs, err := a.AnalyzeQuery(ctx, pat.Query, pat.Args...)
		if err != nil {
			return nil, event.WrapInfrastructure(err, "indexing.analyze_table",
				fmt.Sprintf("analyze table %s", table))
		}

		all = append(all, recs...)
	}

	return a.deduplicate(all), nil
}

// MissingIndexes scans all user tables and returns recommendations for
// detected full-table scans on common CQRS patterns.
func (a *Advisor) MissingIndexes(ctx context.Context) ([]Recommendation, error) {
	tables, err := a.userTables(ctx)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "indexing.list_tables",
			"list user tables")
	}

	var all []Recommendation

	for _, tbl := range tables {
		if a.excluded[tbl] {
			continue
		}

		recs, err := a.AnalyzeTable(ctx, tbl)
		if err != nil {
			return nil, err
		}

		all = append(all, recs...)
	}

	return a.deduplicate(all), nil
}

// ExistingIndexes refreshes the cache of existing index names.
func (a *Advisor) ExistingIndexes(ctx context.Context) error {
	rows, err := a.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'index' AND sql IS NOT NULL")
	if err != nil {
		return event.WrapInfrastructure(err, "indexing.list_indexes",
			"list existing indexes")
	}

	defer func() { _ = rows.Close() }()

	newExisting := make(map[string]bool)

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return event.WrapInfrastructure(err, "indexing.scan_index",
				"scan index name")
		}

		newExisting[name] = true
	}

	a.mu.Lock()
	a.existing = newExisting
	a.mu.Unlock()

	return rows.Err() //nolint:wrapcheck // transparent delegation
}

// HasIndex reports whether an index with the given name already exists.
func (a *Advisor) HasIndex(name string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.existing[name]
}

type queryPattern struct {
	Query string
	Args  []any
}

func (a *Advisor) tableQueryPatterns(table string) []queryPattern {
	switch table {
	case "events":
		return []queryPattern{
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
		}
	case "commands":
		return []queryPattern{
			{
				Query: "SELECT * FROM commands WHERE aggregate_type = ? AND aggregate_id = ? ORDER BY received_at ASC",
				Args:  []any{"User", "dummy-id"},
			},
			{
				Query: "SELECT * FROM commands WHERE command_type = ? ORDER BY received_at ASC",
				Args:  []any{"CreateUser"},
			},
		}
	case "snapshots":
		return []queryPattern{
			{
				Query: "SELECT * FROM snapshots WHERE aggregate_type = ? AND aggregate_id = ?",
				Args:  []any{"User", "dummy-id"},
			},
		}
	case "checkpoints":
		return []queryPattern{
			{
				Query: "SELECT * FROM checkpoints WHERE projection_name = ?",
				Args:  []any{"user-projection"},
			},
		}
	}

	return nil
}

func (a *Advisor) explain(
	ctx context.Context,
	query string,
	args ...any,
) ([]PlanRow, error) {
	rows, err := a.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		return nil, err //nolint:wrapcheck // internal helper
	}

	defer func() { _ = rows.Close() }()

	var plan []PlanRow

	for rows.Next() {
		var row PlanRow

		var notUsed int

		if scanErr := rows.Scan(&row.ID, &row.Parent, &notUsed, &row.Detail); scanErr != nil {
			return nil, event.WrapInfrastructure(scanErr, "indexing.scan_plan",
				"scan query plan row")
		}

		plan = append(plan, row)
	}

	if planErr := rows.Err(); planErr != nil {
		return nil, event.WrapInfrastructure(planErr, "indexing.plan_rows",
			"read query plan rows")
	}

	return plan, nil
}

func (a *Advisor) recommendationsFromPlan(plan []PlanRow, query string) []Recommendation {
	var recs []Recommendation

	for _, row := range plan {
		rec := a.recommendationFromDetail(row.Detail, query)
		if rec != nil {
			recs = append(recs, *rec)
		}
	}

	return recs
}

func (a *Advisor) recommendationFromDetail(detail, query string) *Recommendation {
	detail = strings.TrimSpace(detail)

	// If it's using an index or the primary key, no recommendation needed.
	if searchIndexRe.MatchString(detail) ||
		searchCoverRe.MatchString(detail) ||
		usingIntegerPK.MatchString(detail) ||
		autoIndexRe.MatchString(detail) {
		return nil
	}

	// Check for full table scan.
	m := scanTableRe.FindStringSubmatch(detail)
	if m == nil {
		return nil
	}

	table := m[1]

	idx, reason := a.inferIndex(table, query)
	if idx == nil {
		return nil
	}

	return &Recommendation{
		Index:        *idx,
		Explanation:  reason,
		QueryPattern: query,
	}
}

func (a *Advisor) inferIndex(table, query string) (*Index, string) {
	queryUpper := strings.ToUpper(query)

	switch table {
	case "events":
		if strings.Contains(queryUpper, "AGGREGATE_TYPE") &&
			strings.Contains(queryUpper, "AGGREGATE_ID") &&
			strings.Contains(queryUpper, "VERSION") {
			return &Index{
				Name:    "idx_events_agg_ver",
				Table:   "events",
				Columns: []string{"aggregate_type", "aggregate_id", "version"},
				Reason:  "avoid full table scan on aggregate version queries",
			}, "aggregate load with version filter triggers SCAN TABLE"
		}

		if strings.Contains(queryUpper, "EVENT_TYPE") {
			return &Index{
				Name:    "idx_events_type_time",
				Table:   "events",
				Columns: []string{"event_type", "occurred_at"},
				Reason:  "avoid full table scan on event type filter queries",
			}, "event type projection queries trigger SCAN TABLE"
		}

		if strings.Contains(queryUpper, "OCCURRED_AT") && strings.Contains(queryUpper, "ID") {
			return &Index{
				Name:    "idx_events_cursor",
				Table:   "events",
				Columns: []string{"occurred_at", "id"},
				Reason:  "avoid full table scan on cursor pagination",
			}, "ReadFrom cursor pagination triggers SCAN TABLE"
		}
	case "commands":
		if strings.Contains(queryUpper, "AGGREGATE_TYPE") &&
			strings.Contains(queryUpper, "AGGREGATE_ID") {
			return &Index{
				Name:    "idx_commands_agg_time",
				Table:   "commands",
				Columns: []string{"aggregate_type", "aggregate_id", "received_at"},
				Reason:  "avoid full table scan on command aggregate queries",
			}, "command audit by aggregate triggers SCAN TABLE"
		}

		if strings.Contains(queryUpper, "COMMAND_TYPE") {
			return &Index{
				Name:    "idx_commands_type_time",
				Table:   "commands",
				Columns: []string{"command_type", "received_at"},
				Reason:  "avoid full table scan on command type queries",
			}, "command type analytics triggers SCAN TABLE"
		}
	}

	return nil, ""
}

func (a *Advisor) userTables(ctx context.Context) ([]string, error) {
	rows, err := a.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil, event.WrapInfrastructure(err, "indexing.list_tables",
			"list user tables")
	}

	defer func() { _ = rows.Close() }()

	var tables []string

	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, event.WrapInfrastructure(scanErr, "indexing.scan_table_name",
				"scan table name")
		}

		tables = append(tables, name)
	}

	return tables, rows.Err() //nolint:wrapcheck // transparent delegation
}

func (a *Advisor) deduplicate(recs []Recommendation) []Recommendation {
	seen := make(map[string]bool)

	var out []Recommendation

	for _, r := range recs {
		if seen[r.Index.Name] {
			continue
		}

		seen[r.Index.Name] = true
		out = append(out, r)
	}

	return out
}
