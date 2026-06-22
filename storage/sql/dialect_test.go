package sql_test

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

func TestPostgresDialect_Placeholder(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}

	tests := []struct {
		index    int
		expected string
	}{
		{1, "$1"},
		{2, "$2"},
		{10, "$10"},
	}

	for _, tt := range tests {
		if got := d.Placeholder(tt.index); got != tt.expected {
			t.Errorf("Placeholder(%d) = %q, want %q", tt.index, got, tt.expected)
		}
	}
}

func TestPostgresDialect_FormatTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}
	now := time.Now()

	result := d.FormatTime(now)
	tp, ok := result.(time.Time)
	if !ok {
		t.Fatalf("FormatTime returned %T, want time.Time", result)
	}

	if !tp.Equal(now) {
		t.Errorf("FormatTime = %v, want %v", tp, now)
	}
}

func TestPostgresDialect_ScanTimeDest(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}
	dest := d.ScanTimeDest()

	if _, ok := dest.(*time.Time); !ok {
		t.Fatalf("ScanTimeDest returned %T, want *time.Time", dest)
	}
}

func TestPostgresDialect_ParseTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}
	now := time.Now()

	tp := &now
	parsed, err := d.ParseTime(tp)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}

	if !parsed.Equal(now) {
		t.Errorf("ParseTime = %v, want %v", parsed, now)
	}
}

func TestPostgresDialect_ParseTime_WrongType(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}

	_, err := d.ParseTime("not a *time.Time")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func assertSchemasNonEmpty(t *testing.T, d interface {
	EventSchema() string
	SnapshotSchema() string
	CheckpointSchema() string
},
) {
	t.Helper()

	for _, schema := range []struct {
		name string
		fn   func() string
	}{
		{"EventSchema", d.EventSchema},
		{"SnapshotSchema", d.SnapshotSchema},
		{"CheckpointSchema", d.CheckpointSchema},
	} {
		s := schema.fn()
		if s == "" {
			t.Errorf("%s returned empty string", schema.name)
		}
	}
}

func TestPostgresDialect_Schemas(t *testing.T) {
	t.Parallel()

	assertSchemasNonEmpty(t, sqlpkg.PostgresDialect{})
}

func TestSQLiteDialect_Placeholder(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}

	for _, idx := range []int{0, 1, 5, 100} {
		if got := d.Placeholder(idx); got != "?" {
			t.Errorf("Placeholder(%d) = %q, want %q", idx, got, "?")
		}
	}
}

func TestSQLiteDialect_FormatTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	now := time.Now()

	result := d.FormatTime(now)
	s, ok := result.(string)
	if !ok {
		t.Fatalf("FormatTime returned %T, want string", result)
	}

	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("Parse RFC3339Nano: %v", err)
	}

	if !parsed.Equal(now) {
		t.Errorf("roundtrip: %v != %v", parsed, now)
	}
}

func TestSQLiteDialect_ScanTimeDest(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	dest := d.ScanTimeDest()

	if _, ok := dest.(*string); !ok {
		t.Fatalf("ScanTimeDest returned %T, want *string", dest)
	}
}

func TestSQLiteDialect_ParseTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	ts := "2024-01-15T10:30:00Z"

	parsed, err := d.ParseTime(&ts)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}

	expected, _ := time.Parse(time.RFC3339, ts)
	if !parsed.Equal(expected) {
		t.Errorf("ParseTime = %v, want %v", parsed, expected)
	}
}

func TestSQLiteDialect_ParseTime_WrongType(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}

	_, err := d.ParseTime(42)
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestSQLiteDialect_Schemas(t *testing.T) {
	t.Parallel()

	assertSchemasNonEmpty(t, sqlpkg.SQLiteDialect{})
}

func TestParseSQLiteTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{"RFC3339Nano", "2024-01-15T10:30:00.123456789Z"},
		{"RFC3339", "2024-01-15T10:30:00Z"},
		{"datetime with T", "2024-01-15T10:30:05"},
		{"datetime with space", "2024-01-15 10:30:05"},
		{"datetime with offset", "2024-01-15T10:30:05+02:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			parsed, err := sqlpkg.ParseSQLiteTimestamp(tt.input)
			if err != nil {
				t.Fatalf("ParseSQLiteTimestamp(%q): %v", tt.input, err)
			}

			if parsed.IsZero() {
				t.Errorf("expected non-zero time for %q", tt.input)
			}
		})
	}
}

func TestParseSQLiteTimestamp_Empty(t *testing.T) {
	t.Parallel()

	parsed, err := sqlpkg.ParseSQLiteTimestamp("")
	if err != nil {
		t.Fatalf("ParseSQLiteTimestamp empty: %v", err)
	}

	if !parsed.IsZero() {
		t.Errorf("expected zero time for empty string, got %v", parsed)
	}
}

func TestParseSQLiteTimestamp_Invalid(t *testing.T) {
	t.Parallel()

	_, err := sqlpkg.ParseSQLiteTimestamp("not-a-timestamp")
	if err == nil {
		t.Fatal("expected error for invalid timestamp")
	}
}

func TestNewDBHandle_NilDB(t *testing.T) {
	t.Parallel()

	_, err := sqlpkg.NewDBHandle(nil, sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestNewDBHandle_Valid(t *testing.T) {
	t.Parallel()

	db, err := openTestDB(t)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	handle, err := sqlpkg.NewDBHandle(db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewDBHandle: %v", err)
	}

	if handle.DB != db {
		t.Error("DB not set")
	}
}

func TestDBHandle_Close(t *testing.T) {
	t.Parallel()

	db, _ := openTestDB(t)
	handle, _ := sqlpkg.NewDBHandle(db, sqlpkg.SQLiteDialect{})

	if err := handle.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestPlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dialect  sqlpkg.Dialect
		count    int
		offset   int
		expected string
	}{
		{sqlpkg.SQLiteDialect{}, 3, 0, "?, ?, ?"},
		{sqlpkg.PostgresDialect{}, 3, 0, "$1, $2, $3"},
		{sqlpkg.PostgresDialect{}, 2, 3, "$4, $5"},
		{sqlpkg.SQLiteDialect{}, 1, 0, "?"},
	}

	for _, tt := range tests {
		got := sqlpkg.Placeholders(tt.dialect, tt.count, tt.offset)
		if got != tt.expected {
			t.Errorf("Placeholders(%T, %d, %d) = %q, want %q",
				tt.dialect, tt.count, tt.offset, got, tt.expected)
		}
	}
}

func openTestDB(t *testing.T) (*sql.DB, error) {
	t.Helper()

	return sql.Open("sqlite", ":memory:")
}

func TestPostgresDialect_CommandSchema(t *testing.T) {
	t.Parallel()

	s := sqlpkg.PostgresDialect{}.CommandSchema()
	if s == "" {
		t.Error("CommandSchema returned empty string")
	}
	if !contains(s, "commands") {
		t.Error("CommandSchema should contain 'commands' table")
	}
}

func TestSQLiteDialect_CommandSchema(t *testing.T) {
	t.Parallel()

	s := sqlpkg.SQLiteDialect{}.CommandSchema()
	if s == "" {
		t.Error("CommandSchema returned empty string")
	}
	if !contains(s, "commands") {
		t.Error("CommandSchema should contain 'commands' table")
	}
}
