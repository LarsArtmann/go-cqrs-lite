package view

import (
	"context"
	"testing"
)

func TestSQLViewStore_WithoutAutoMigrate(t *testing.T) {
	t.Parallel()

	db, err := openSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLiteViewStore[testView, testKey](db, testMapper(),
		WithoutViewAutoMigrate())
	if err != nil {
		t.Fatalf("NewSQLiteViewStore without migrate: %v", err)
	}

	err = store.Set(context.Background(), testKey("k1"), &testView{Name: "Alice"})
	if err == nil {
		t.Fatal("Set without table: expected error, got nil")
	}
}

func TestSQLViewStore_ValidationErrors(t *testing.T) {
	t.Parallel()

	db, err := openSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	tests := []struct {
		name    string
		mapper  ViewMapper[testView]
		wantErr string
	}{
		{
			name:    "empty table",
			mapper:  ViewMapper[testView]{},
			wantErr: "Table is required",
		},
		{
			name: "missing ScanRow",
			mapper: ViewMapper[testView]{
				Table: "t",
				Columns: []ViewColumn[testView]{
					{Name: "x", Type: "TEXT", Extract: func(v *testView) any { return v.Name }},
				},
			},
			wantErr: "ScanRow is required",
		},
		{
			name: "no columns",
			mapper: ViewMapper[testView]{
				Table:   "t",
				ScanRow: func(scan func(dest ...any) error) (*testView, error) { return &testView{}, nil },
			},
			wantErr: "at least one Column",
		},
		{
			name: "reserved key column",
			mapper: ViewMapper[testView]{
				Table: "t",
				Columns: []ViewColumn[testView]{
					{Name: "key", Type: "TEXT", Extract: func(v *testView) any { return v.Name }},
				},
				ScanRow: func(scan func(dest ...any) error) (*testView, error) { return &testView{}, nil },
			},
			wantErr: "reserved",
		},
		{
			name: "nil extract",
			mapper: ViewMapper[testView]{
				Table:   "t",
				Columns: []ViewColumn[testView]{{Name: "x", Type: "TEXT"}},
				ScanRow: func(scan func(dest ...any) error) (*testView, error) { return &testView{}, nil },
			},
			wantErr: "Extract is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewSQLiteViewStore[testView, testKey](db, tt.mapper)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !containsStr(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSQLViewStore_DuplicateColumn(t *testing.T) {
	t.Parallel()

	db, err := openSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	mapper := ViewMapper[testView]{
		Table: "dup",
		Columns: []ViewColumn[testView]{
			{Name: "name", Type: "TEXT", Extract: func(v *testView) any { return v.Name }},
			{Name: "name", Type: "TEXT", Extract: func(v *testView) any { return v.Name }},
		},
		ScanRow: func(scan func(dest ...any) error) (*testView, error) { return &testView{}, nil },
	}

	_, err = NewSQLiteViewStore[testView, testKey](db, mapper)
	if err == nil || !containsStr(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got: %v", err)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return len(substr) == 0
}
