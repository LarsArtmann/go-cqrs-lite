package indexing

import (
	"context"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"
)

type queryPattern struct {
	Query string
	Args  []any
}

func (a *Advisor) tableQueryPatterns(table string) []queryPattern {
	return queryPatternsByTable[table]
}

func (a *Advisor) explain(
	ctx context.Context,
	query string,
	args ...any,
) ([]PlanRow, error) {
	rows, err := a.db.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		return nil, errorfamily.Wrapf(
			err,
			errorfamily.Infrastructure,
			"turso.explain_query",
			"query=%v",
			query,
		)
	}

	defer func() { _ = rows.Close() }()

	var plan []PlanRow

	for rows.Next() {
		var row PlanRow

		var notUsed int

		if scanErr := rows.Scan(&row.ID, &row.Parent, &notUsed, &row.Detail); scanErr != nil {
			return nil, errorfamily.WrapInfrastructure(scanErr, "indexing.scan_plan_row",
				"scan query plan row")
		}

		plan = append(plan, row)
	}

	if planErr := rows.Err(); planErr != nil {
		return nil, errorfamily.WrapInfrastructure(planErr, "indexing.plan_rows",
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

	for _, re := range advisoryRegexes {
		if re.MatchString(detail) {
			return nil
		}
	}

	// Check for full table scan.
	m := scanTableRe.FindStringSubmatch(detail)
	if m == nil {
		return nil
	}

	table := m[1]

	idx, reason, priority := a.inferIndex(table, query)
	if idx == nil {
		return nil
	}

	return &Recommendation{
		Index:        *idx,
		Explanation:  reason,
		QueryPattern: query,
		Priority:     priority,
		AdvisorVer:   Version("v1"),
	}
}

func (a *Advisor) inferIndex(table, query string) (*Index, string, Priority) {
	queryUpper := strings.ToUpper(query)

	for _, rule := range indexInferenceRules[table] {
		if containsAll(queryUpper, rule.needs...) {
			idx := rule.index

			return &idx, rule.reason, rule.priority
		}
	}

	return nil, "", PriorityOptional
}

func (a *Advisor) userTables(ctx context.Context) ([]string, error) {
	rows, err := a.db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "indexing.list_tables",
			"list user tables")
	}

	defer func() { _ = rows.Close() }()

	var tables []string

	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, errorfamily.WrapInfrastructure(scanErr, "indexing.scan_table_name",
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
