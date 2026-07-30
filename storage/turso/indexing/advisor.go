package indexing

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

var (
	// scanTableRe matches SQLite EXPLAIN QUERY PLAN scan output.
	// Supports both legacy ("SCAN TABLE events") and modern ("SCAN events")
	// formats — older SQLite includes "TABLE", newer versions omit it.
	scanTableRe    = regexp.MustCompile(`SCAN\s+(?:TABLE\s+)?(\S+)`)
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

// Priority classifies how important a recommendation is.
type Priority int

const (
	// PriorityOptional indicates an index that may help but is not critical.
	PriorityOptional Priority = iota
	// PriorityRecommended indicates an index that should be created for production.
	PriorityRecommended
	// PriorityCritical indicates an index that fixes a severe performance issue
	// (e.g., full table scan on a query that runs frequently).
	PriorityCritical
)

// String returns a human-readable priority name.
func (p Priority) String() string {
	switch p {
	case PriorityCritical:
		return "critical"
	case PriorityRecommended:
		return "recommended"
	default:
		return "optional"
	}
}

// Version tracks which schema or advisor version produced this index/recommendation.
//cqrs-lint:ignore(A008) library code or intentional pattern
type Version string

// Recommendation is a suggested index with context.
type Recommendation struct {
	Index        Index
	Explanation  string
	QueryPattern string
	Priority     Priority
	AdvisorVer   Version
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
	ctx, span := startIndexingSpan(
		ctx,
		SpanAdvisorAnalyzeQuery,
		trace.WithAttributes(attribute.String("db.statement", query)),
	)
	defer endSpan(span, nil)

	plan, err := a.explain(ctx, query, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(err, "indexing.explain",
			"explain query plan")
	}

	recs := a.recommendationsFromPlan(plan, query)
	span.SetAttributes(attribute.Int(AttrRecommendationCount, len(recs)))

	return recs, nil
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
			return nil, errorfamily.WrapInfrastructure(err, "indexing.analyze_table",
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
		return nil, errorfamily.WrapInfrastructure(err, "indexing.list_tables",
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
		return errorfamily.WrapInfrastructure(err, "indexing.list_indexes",
			"list existing indexes")
	}

	defer func() { _ = rows.Close() }()

	newExisting := make(map[string]bool)

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return errorfamily.WrapInfrastructure(err, "indexing.scan_index",
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
