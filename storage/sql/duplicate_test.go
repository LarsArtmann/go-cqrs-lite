package sql

import (
	"errors"
	"fmt"
	"testing"
)

// pgFakeError satisfies pgCodeError (Code() string) like pgconn.PgError does.
type pgFakeError struct{ code string }

func (p pgFakeError) Error() string { return "pg error " + p.code }
func (p pgFakeError) Code() string  { return p.code }

// sqliteFakeError satisfies sqliteCodeError (Code() int) like modernc.org/sqlite.
type sqliteFakeError struct{ code int }

func (s sqliteFakeError) Error() string { return "sqlite error" }
func (s sqliteFakeError) Code() int     { return s.code }

func TestIsDuplicateKeyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"pg duplicate", pgFakeError{code: pgDuplicateCode}, true},
		{"pg other", pgFakeError{code: "42P01"}, false},
		{"sqlite duplicate", sqliteFakeError{code: sqliteExtendedCode}, true},
		{"sqlite other", sqliteFakeError{code: 1}, false},
		{"unrelated", errors.New("connection refused"), false},

		// String fallbacks for drivers without typed errors.
		{"sqlite string fallback", errors.New("UNIQUE constraint failed: users.id"), true},
		{
			"pg string fallback",
			errors.New("duplicate key value violates unique constraint \"users_pkey\""), true,
		},

		// The classified error chain must still be traversable.
		{"wrapped pg duplicate", fmt.Errorf("save: %w", pgFakeError{code: pgDuplicateCode}), true},
		{"wrapped sqlite string", fmt.Errorf("insert: %w",
			errors.New("UNIQUE constraint failed: t.x")), true},

		// DuckDB string fallback (DuckDB reports "Constraint Error: UNIQUE constraint violated").
		{"duckdb string fallback", errors.New("Constraint Error: UNIQUE constraint violated: t.x"), true},
		{"duckdb wrapped", fmt.Errorf("insert: %w",
			errors.New("Constraint Error: UNIQUE constraint violated: events.id")), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := IsDuplicateKeyError(tc.err); got != tc.want {
				t.Fatalf("IsDuplicateKeyError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
