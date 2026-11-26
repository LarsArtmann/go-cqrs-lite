package metaengine_test

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestValidateQueryIdents pins the gate in front of every SQL-engine query
// surface: the operator, the filter / sort / extra column names and the
// FilterIn list form are each rejected before a builder splices them into
// statement text.
func TestValidateQueryIdents(t *testing.T) {
	t.Parallel()

	hostile := []metaengine.FilterSpec{
		{Column: "status", Op: "= 1 OR 1 =", Value: "open"},
		{Column: "x') OR ('1'='1", Op: metaengine.FilterEq, Value: "open"},
		{Column: "status", Op: metaengine.FilterIn, Value: []string{"open"}},
	}

	for _, f := range hostile {
		err := metaengine.ValidateQueryIdents([]metaengine.FilterSpec{f}, nil)
		if err == nil {
			t.Errorf("filter %+v must be rejected", f)
		}
	}

	for _, column := range []string{"n); DROP TABLE x; --", "status'"} {
		cases := []struct {
			name string
			call func() error
		}{
			{"sort column", func() error {
				return metaengine.ValidateQueryIdents(nil, &metaengine.SortSpec{Column: column})
			}},
			{"extra column", func() error {
				return metaengine.ValidateQueryIdents(nil, nil, "ok", column)
			}},
		}

		for _, tc := range cases {
			if err := tc.call(); err == nil {
				t.Errorf("hostile %s %q must be rejected", tc.name, column)
			}
		}

		if err := metaengine.ValidateQueryIdents(nil, nil, "", column[:0]); err != nil {
			t.Errorf("empty column names must be skipped, got %v", err)
		}
	}

	clean := []metaengine.FilterSpec{
		{Column: "status", Op: metaengine.FilterIn, Value: []any{}},
		{Column: "n", Op: metaengine.FilterIn, Value: nil},
		{Column: "n", Op: metaengine.FilterGt, Value: 1.0},
	}

	if err := metaengine.ValidateQueryIdents(clean, &metaengine.SortSpec{Column: "priority"}, "", "n"); err != nil {
		t.Errorf("clean input must pass, got %v", err)
	}

	err := metaengine.ValidateQueryIdents([]metaengine.FilterSpec{{Column: "status", Op: "= 1 =", Value: "x"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid filter operator") {
		t.Errorf("hostile operator error = %v, want the invalid-operator error", err)
	}
}

// TestAppendPlannedFilterEmptyIn pins ruling S041-02 on the shared
// planned-table clause builder: an empty FilterIn list renders the false
// predicate instead of being dropped.
func TestAppendPlannedFilterEmptyIn(t *testing.T) {
	t.Parallel()

	var (
		b       strings.Builder
		args    []any
		started bool
	)

	for _, list := range []any{[]any{}, nil} {
		metaengine.AppendPlannedFilter(&b, &args, metaengine.FilterSpec{
			Column: "n", Op: metaengine.FilterIn, Value: list,
		}, &started, metaengine.QuoteIdent, metaengine.QuestionPlaceholders, ", ")
	}

	if got, want := b.String(), " WHERE 1 = 0 AND 1 = 0"; got != want {
		t.Errorf("empty IN clauses = %q, want %q", got, want)
	}

	if len(args) != 0 {
		t.Errorf("empty IN clause bound %d args, want 0", len(args))
	}
}
