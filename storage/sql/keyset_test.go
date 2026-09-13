package sql_test

import (
	"errors"
	"strings"
	"testing"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	errorfamily "github.com/larsartmann/go-error-family"
)

func TestKeysetPositionQueryChecked_ValidIdentifiers(t *testing.T) {
	t.Parallel()

	query, err := sqlpkg.KeysetPositionQueryChecked(
		sqlpkg.SQLiteDialect{}, "e.id, e.occurred_at", "events", "occurred_at",
	)
	if err != nil {
		t.Fatalf("valid identifiers rejected: %v", err)
	}

	for _, want := range []string{
		"FROM events e",
		"WHERE e.occurred_at >= ?",
		"e.occurred_at > ? OR e.id > ?",
		"ORDER BY e.occurred_at ASC, e.id ASC",
	} {
		if !strings.Contains(query, want) {
			t.Errorf("query missing %q:\n%s", want, query)
		}
	}

	if got := strings.Count(query, "?"); got != 3 {
		t.Errorf("want 3 placeholders, got %d:\n%s", got, query)
	}
}

func TestKeysetPositionQueryChecked_RejectsInjection(t *testing.T) {
	t.Parallel()

	hostile := []struct {
		name   string
		table  string
		column string
	}{
		{"comment_evasion_table", "events--", "occurred_at"},
		{"union_select_table", "events UNION SELECT 1", "occurred_at"},
		{"quoted_column", "events", `occurred_at"DROP`},
		{"semicolon_column", "events", "occurred_at; DROP TABLE events"},
		{"empty_table", "", "occurred_at"},
		{"empty_column", "events", ""},
		{"digit_leading", "1events", "occurred_at"},
		{"dotted", "main.events", "occurred_at"},
		{"newline_payload", "events\nWHERE 1=1", "occurred_at"},
	}

	for _, tc := range hostile {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			query, err := sqlpkg.KeysetPositionQueryChecked(
				sqlpkg.SQLiteDialect{}, "e.id", tc.table, tc.column,
			)
			if err == nil {
				t.Fatalf("expected rejection for table=%q column=%q, got query:\n%s",
					tc.table, tc.column, query)
			}

			if query != "" {
				t.Errorf("error path must not return a partial query, got:\n%s", query)
			}

			var famErr *errorfamily.Error
			if !errors.As(err, &famErr) || famErr.ErrorFamily() != errorfamily.Infrastructure {
				t.Errorf("want Infrastructure-classified error, got %T: %v", err, err)
			}
		})
	}
}

func TestKeysetPositionQueryChecked_MatchesDeprecatedWrapper(t *testing.T) {
	t.Parallel()

	const (
		columns      = "e.id, e.occurred_at"
		table        = "events"
		timestampCol = "occurred_at"
	)

	fromChecked, err := sqlpkg.KeysetPositionQueryChecked(
		sqlpkg.SQLiteDialect{}, columns, table, timestampCol,
	)
	if err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}

	fromWrapper := sqlpkg.KeysetPositionQuery(sqlpkg.SQLiteDialect{}, columns, table, timestampCol)
	if fromChecked != fromWrapper {
		t.Errorf(
			"checked and wrapper disagree:\nchecked:\n%s\nwrapper:\n%s",
			fromChecked,
			fromWrapper,
		)
	}
}
