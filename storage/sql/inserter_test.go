package sql

import (
	"errors"
	"testing"
)

func TestInserterInsertSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dialect Dialect
		want    string
	}{
		{
			name:    "sqlite",
			dialect: SQLiteDialect{},
			want:    "INSERT INTO cqrs_commands (id, type) VALUES (?, ?)",
		},
		{
			name:    "postgres",
			dialect: PostgresDialect{},
			want:    "INSERT INTO cqrs_commands (id, type) VALUES ($1, $2)",
		},
		{
			name:    "mysql",
			dialect: MySQLDialect{},
			want:    "INSERT INTO cqrs_commands (id, type) VALUES (?, ?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			in := &Inserter[string]{
				Dialect: tt.dialect,
				Table:   "cqrs_commands",
				Columns: []string{"id", "type"},
			}

			if got := in.insertSQL(); got != tt.want {
				t.Errorf("insertSQL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInserterInsertAllStopsAtFirstFailure(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("boom")
	in := &Inserter[int]{
		Dialect: SQLiteDialect{},
		Table:   "cqrs_commands",
		Columns: []string{"id"},
		RowArgs: func(item int) ([]any, error) {
			return nil, wantErr
		},
		MarshalErrCode: "test.marshal",
		Describe:       func(int) string { return "item" },
		EntityNoun:     "item",
	}

	err := in.InsertAll(t.Context(), nil, []int{1, 2, 3})
	if err == nil {
		t.Fatal("InsertAll: expected error for failing RowArgs, got nil")
	}

	if !errors.Is(err, wantErr) {
		t.Errorf("InsertAll error should wrap RowArgs failure, got: %v", err)
	}

	if got := in.InsertAll(t.Context(), nil, nil); got != nil {
		t.Errorf("InsertAll(nil): %v", got)
	}
}
