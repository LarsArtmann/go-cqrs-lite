package system

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Engine-side deployment config: named storage engines and their
// materialized-view accelerations (operator concerns, D11).

// EngineConfig declares a named storage engine.
type EngineConfig struct {
	Driver  string   `koanf:"driver"`  // "sqlite", "memory", "pebble", "duckdb", "postgres"
	DSN     string   `koanf:"dsn"`     // connection string (empty for memory)
	Pragmas []string `koanf:"pragmas"` // SQLite pragmas (e.g., "wal", "foreign_keys")

	// Priority is the operator's layout-planning objective for this engine
	// (ADR-0124): one of "WriteSpeed", "ReadSpeed", "StorageSpace", or
	// "Balanced" (default). It flows into DriverConfig.Priority and weights
	// the cost model's scoring for queries routed to this engine. Leave
	// empty for the global default.
	Priority metaengine.Priority `koanf:"priority"`

	// MaterializedViews declares operator-owned aggregate accelerations
	// (Turso/libSQL incremental view maintenance). Each entry names a
	// collection and an aggregate shape (fn, column, optional groupBy); the
	// engine derives and creates the view at construction and serves matching
	// unfiltered aggregates from it. This is a deployment-time concern —
	// developers never declare views. Unsupported engines fail construction
	// loudly.
	MaterializedViews []MaterializedViewConfig `koanf:"materialized_views"`
}

// MaterializedViewConfig is the operator-facing YAML shape for one
// materialized view acceleration (see metaengine.MaterializedViewSpec).
type MaterializedViewConfig struct {
	// Collection is the collection whose aggregate this view accelerates
	// (e.g. "orders").
	Collection string `koanf:"collection"`

	// Fn is the aggregate function: "COUNT", "SUM", "MIN", "MAX", or "AVG"
	// (case-insensitive).
	Fn string `koanf:"fn"`

	// Column is the JSON field name aggregated over (e.g. "amount"). Empty
	// (and required to be empty) for COUNT.
	Column string `koanf:"column"`

	// GroupBy is the optional JSON field name to group by (e.g. "customer").
	// Empty creates a single-row scalar view (fastest reads).
	GroupBy string `koanf:"group_by"`
}

// materializedViewSpecs maps the operator config to validated
// metaengine.MaterializedViewSpec values.
func (c EngineConfig) materializedViewSpecs() ([]metaengine.MaterializedViewSpec, error) {
	if len(c.MaterializedViews) == 0 {
		return nil, nil
	}

	specs := make([]metaengine.MaterializedViewSpec, 0, len(c.MaterializedViews))

	for i, mv := range c.MaterializedViews {
		fn, err := parseMatViewFn(mv.Fn)
		if err != nil {
			return nil, fmt.Errorf(
				"system: engine %q materialized_views[%d]: %w",
				labelOr(c.Driver, i+1),
				i,
				err,
			)
		}

		spec := metaengine.MaterializedViewSpec{
			Collection: mv.Collection,
			Fn:         fn,
			Column:     mv.Column,
			GroupBy:    mv.GroupBy,
		}

		if err := spec.Validate(); err != nil {
			return nil, fmt.Errorf(
				"system: engine %q materialized_views[%d]: %w",
				labelOr(c.Driver, i+1),
				i,
				err,
			)
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

// parseMatViewFn resolves the operator-supplied aggregate function name
// (case-insensitive) to a metaengine.AggregateFn.
func parseMatViewFn(s string) (metaengine.AggregateFn, error) {
	switch metaengine.AggregateFn(strings.ToUpper(strings.TrimSpace(s))) {
	case metaengine.MatViewCount:
		return metaengine.MatViewCount, nil
	case metaengine.MatViewSum:
		return metaengine.MatViewSum, nil
	case metaengine.MatViewMin:
		return metaengine.MatViewMin, nil
	case metaengine.MatViewMax:
		return metaengine.MatViewMax, nil
	case metaengine.MatViewAvg:
		return metaengine.MatViewAvg, nil
	case "":
		return "", fmt.Errorf("fn is required (COUNT, SUM, MIN, MAX, or AVG)")
	default:
		return "", fmt.Errorf("unknown fn %q (want COUNT, SUM, MIN, MAX, or AVG)", s)
	}
}

// labelOr returns label when non-empty, else a positional fallback for error
// messages.
func labelOr(label string, pos int) string {
	if label != "" {
		return label
	}

	return fmt.Sprintf("#%d", pos)
}
