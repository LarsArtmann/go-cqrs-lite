package sql

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
)

// fakeSQLiteError is a test double implementing sqliteCodeError (Code() int).
type fakeSQLiteError struct {
	code int
	msg  string
}

func (e *fakeSQLiteError) Error() string { return e.msg }
func (e *fakeSQLiteError) Code() int     { return e.code }

// fakePgError is a test double implementing pgCodeError (Code() string).
type fakePgError struct {
	code string
	msg  string
}

func (e *fakePgError) Error() string { return e.msg }
func (e *fakePgError) Code() string  { return e.code }

func TestInit_StdlibDefaultsRegistered(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errorfamily.Family
	}{
		{"ErrNoRows → Rejection", sql.ErrNoRows, errorfamily.Rejection},
		{"ErrConnDone → Transient", sql.ErrConnDone, errorfamily.Transient},
		{"context.Canceled → Rejection", context.Canceled, errorfamily.Rejection},
		{"context.DeadlineExceeded → Transient", context.DeadlineExceeded, errorfamily.Transient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := errorfamily.Classify(tt.err)
			if got != tt.want {
				t.Errorf("Classify(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestInit_SQLiteClassifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errorfamily.Family
	}{
		{
			"BUSY (5) → Transient",
			&fakeSQLiteError{code: 5, msg: "database is locked"},
			errorfamily.Transient,
		},
		{
			"LOCKED (6) → Transient",
			&fakeSQLiteError{code: 6, msg: "table is locked"},
			errorfamily.Transient,
		},
		{
			"CONSTRAINT (19) → Conflict",
			&fakeSQLiteError{code: 19, msg: "UNIQUE constraint failed"},
			errorfamily.Conflict,
		},
		{
			"unknown code → unclassified",
			&fakeSQLiteError{code: 1, msg: "SQLITE_ERROR"},
			errorfamily.Transient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := classifySQLiteError(tt.err)
			if tt.name == "unknown code → unclassified" {
				if ok {
					t.Errorf("expected unclassified for unknown SQLite code, got %v", got)
				}

				return
			}

			if !ok {
				t.Fatalf("expected classified, got ok=false")
			}

			if got != tt.want {
				t.Errorf("classifySQLiteError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestInit_PostgresClassifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want errorfamily.Family
	}{
		{
			"23505 (unique violation) → Conflict",
			&fakePgError{code: "23505", msg: "duplicate key"},
			errorfamily.Conflict,
		},
		{
			"23P01 (exclusion violation) → Conflict",
			&fakePgError{code: "23P01", msg: "exclusion"},
			errorfamily.Conflict,
		},
		{
			"40001 (serialization failure) → Transient",
			&fakePgError{code: "40001", msg: "could not serialize"},
			errorfamily.Transient,
		},
		{
			"40P01 (deadlock) → Transient",
			&fakePgError{code: "40P01", msg: "deadlock detected"},
			errorfamily.Transient,
		},
		{
			"53000 (insufficient resources) → Transient",
			&fakePgError{code: "53000", msg: "out of memory"},
			errorfamily.Transient,
		},
		{
			"57014 (query canceled) → Transient",
			&fakePgError{code: "57014", msg: "canceling statement"},
			errorfamily.Transient,
		},
		{
			"unknown class → unclassified",
			&fakePgError{code: "42P01", msg: "undefined table"},
			errorfamily.Transient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := classifyPostgresError(tt.err)
			if tt.name == "unknown class → unclassified" {
				if ok {
					t.Errorf("expected unclassified for unknown PG class, got %v", got)
				}

				return
			}

			if !ok {
				t.Fatalf("expected classified, got ok=false")
			}

			if got != tt.want {
				t.Errorf("classifyPostgresError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestInit_ClassifierHandlesNonMatching(t *testing.T) {
	t.Parallel()

	// Plain errors should return (Transient, false) — not classified.
	err := errors.New("plain error")

	fam, ok := classifySQLiteError(err)
	if ok {
		t.Errorf("classifySQLiteError on plain error should return ok=false, got %v, %v", fam, ok)
	}

	fam, ok = classifyPostgresError(err)
	if ok {
		t.Errorf("classifyPostgresError on plain error should return ok=false, got %v, %v", fam, ok)
	}
}
