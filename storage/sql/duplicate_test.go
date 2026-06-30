package sql

import (
	"errors"
	"fmt"
	"testing"
)

// pgFake satisfies pgCodeError (Code() string) like pgconn.PgError does.
type pgFake struct{ code string }

func (p pgFake) Error() string { return "pg error " + p.code }
func (p pgFake) Code() string  { return p.code }

// sqliteFake satisfies sqliteCodeError (Code() int) like modernc.org/sqlite.
type sqliteFake struct{ code int }

func (s sqliteFake) Error() string { return "sqlite error" }
func (s sqliteFake) Code() int     { return s.code }

func TestIsDuplicateKeyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"pg duplicate", pgFake{code: pgDuplicateCode}, true},
		{"pg other", pgFake{code: "42P01"}, false},
		{"sqlite duplicate", sqliteFake{code: sqliteExtendedCode}, true},
		{"sqlite other", sqliteFake{code: 1}, false},
		{"unrelated", errors.New("connection refused"), false},

		// String fallbacks for drivers without typed errors.
		{"sqlite string fallback", errors.New("UNIQUE constraint failed: users.id"), true},
		{
			"pg string fallback",
			errors.New("duplicate key value violates unique constraint \"users_pkey\""), true,
		},

		// The classified error chain must still be traversable.
		{"wrapped pg duplicate", fmt.Errorf("save: %w", pgFake{code: pgDuplicateCode}), true},
		{"wrapped sqlite string", fmt.Errorf("insert: %w",
			errors.New("UNIQUE constraint failed: t.x")), true},
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
