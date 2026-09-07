package metaengine

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
)

// MaterializedViewSpec is an OPERATOR-declared acceleration: "keep an
// incrementally-maintained materialized view for this aggregate shape". The
// developer never sees it — declaring matviews is a deployment-time concern
// (metaengine north star: operators decide where data lives), declared in
// system.EngineConfig (or passed straight to a driver factory).
//
// The engine derives the CREATE MATERIALIZED VIEW DDL from the spec, so the
// mapping between spec and view columns is exact by construction — the library
// never guesses what an arbitrary operator-written SELECT means.
//
// Currently implemented by the Turso (libSQL) engine, whose incremental view
// maintenance (IVM) keeps the view consistent with meta_map inside every write
// transaction. Engines that cannot serve a spec fail construction loudly
// rather than silently ignoring it.
type MaterializedViewSpec struct {
	// Collection is the meta_map collection the view accelerates (e.g. "orders").
	Collection string

	// Fn is the accelerated aggregate function: COUNT, SUM, MIN, MAX, or AVG.
	Fn AggregateFn

	// Column is the JSON field name aggregated over (e.g. "amount"). Required
	// for SUM/MIN/MAX/AVG; must be empty for COUNT (COUNT(*) counts rows).
	Column string

	// GroupBy is the optional JSON field name to group by (e.g. "customer").
	// Empty means a single-row scalar view (fastest possible reads). When set,
	// the view has one row per group and also serves the scalar aggregate via
	// exact algebraic derivation (e.g. SUM over group sums).
	GroupBy string
}

// Aggregate functions that a MaterializedViewSpec may declare.
const (
	MatViewCount = AggregateCount
	MatViewSum   = AggregateSum
	MatViewMin   = AggregateMin
	MatViewMax   = AggregateMax
	MatViewAvg   = AggregateAvg
)

// Validate checks the spec's shape and rejects strings that could break out of
// the generated SQL (specs reach DDL text; the strings are operator-supplied
// config, but the check is defense in depth with a clear error).
func (s MaterializedViewSpec) Validate() error {
	for _, bad := range []struct {
		name  string
		value string
	}{
		{"collection", s.Collection},
		{"column", s.Column},
		{"groupBy", s.GroupBy},
	} {
		if err := validateMatViewString(bad.value); err != nil {
			return fmt.Errorf("materialized view %s: %w", bad.name, err)
		}
	}

	if s.Collection == "" {
		return errors.New("materialized view: collection must not be empty")
	}

	switch s.Fn {
	case MatViewCount:
		if s.Column != "" {
			return fmt.Errorf("materialized view %q: COUNT does not take a column (it counts rows)", s.Collection)
		}
	case MatViewSum, MatViewMin, MatViewMax, MatViewAvg:
		if s.Column == "" {
			return fmt.Errorf("materialized view %q: %s requires a column", s.Collection, s.Fn)
		}
	default:
		return fmt.Errorf(
			"materialized view %q: unsupported aggregate fn %q (want COUNT, SUM, MIN, MAX, or AVG)",
			s.Collection,
			s.Fn,
		)
	}

	return nil
}

// validateMatViewString rejects empty-forbidden cases are handled by callers;
// this checks injection-relevant characters.
func validateMatViewString(s string) error {
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '_', r == '.', r == '-', r == ' ', r >= 0x80:
			// allowed
		default:
			return fmt.Errorf("invalid character %q (use letters, digits, '_', '.', '-')", r)
		}
	}

	return nil
}

// ViewName returns the deterministic SQL identifier of the view for this
// spec: readable, sanitized, and suffixed with a stable hash of the full
// shape so distinct specs can never collide and the same spec always maps to
// the same name across restarts (keeping CREATE ... IF NOT EXISTS idempotent).
func (s MaterializedViewSpec) ViewName() string {
	canonical := strings.Join([]string{s.Collection, string(s.Fn), s.Column, s.GroupBy}, "\x00")

	h := fnv.New64a()
	_, _ = h.Write([]byte(canonical))

	name := "cqrs_mv_" + sanitize(strings.ToLower(s.Collection))
	name += "_" + strings.ToLower(string(s.Fn))

	if s.Column != "" {
		name += "_" + sanitize(strings.ToLower(s.Column))
	}

	if s.GroupBy != "" {
		name += "_by_" + sanitize(strings.ToLower(s.GroupBy))
	}

	const maxReadable = 48

	if len(name) > maxReadable {
		name = name[:maxReadable]
	}

	return name + "_x" + fmt.Sprintf("%08x", h.Sum64()&0xFFFFFFFF)
}

// MaterializedViewInfo describes one operator-declared materialized view for
// observability (Doctor output, engine stats): the spec it serves and the
// physical SQL view name.
type MaterializedViewInfo struct {
	Spec MaterializedViewSpec
	Name string
	Rows int64
}

// MaterializedViewsReporter is an optional capability: engines that own
// operator-declared materialized views report them so Store.Doctor and
// GetEngineStats can surface the acceleration without querying the catalog.
type MaterializedViewsReporter interface {
	MaterializedViews() []MaterializedViewInfo
}
