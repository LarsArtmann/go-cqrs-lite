package storage

import (
	"errors"
	"testing"
	"time"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
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
		got := d.Placeholder(tt.index)
		if got != tt.expected {
			t.Errorf("Placeholder(%d) = %q, want %q", tt.index, got, tt.expected)
		}
	}
}

func TestPostgresDialect_FormatTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}
	now := time.Now()

	result := d.FormatTime(now)
	if result != now {
		t.Errorf("FormatTime should return time.Time directly, got %T", result)
	}
}

func TestPostgresDialect_ScanTimeDest(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}
	dest := d.ScanTimeDest()

	tp, ok := dest.(*time.Time)
	if !ok {
		t.Fatalf("ScanTimeDest() = %T, want *time.Time", dest)
	}

	if !tp.IsZero() {
		t.Error("ScanTimeDest() should return pointer to zero time")
	}
}

func TestPostgresDialect_ParseTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.PostgresDialect{}
	now := time.Now()

	parsed, err := d.ParseTime(&now)
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

	_, err := d.ParseTime("not a time pointer")
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestSQLiteDialect_Placeholder(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}

	for _, index := range []int{1, 2, 10} {
		got := d.Placeholder(index)
		if got != "?" {
			t.Errorf("Placeholder(%d) = %q, want ?", index, got)
		}
	}
}

func TestSQLiteDialect_FormatTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	now := time.Now()

	result := d.FormatTime(now)
	str, ok := result.(string)
	if !ok {
		t.Fatalf("FormatTime should return string, got %T", result)
	}

	parsed, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		t.Fatalf("FormatTime produced unparseable output: %v", err)
	}

	if !parsed.Equal(now) {
		t.Errorf("round-trip mismatch: %v != %v", parsed, now)
	}
}

func TestSQLiteDialect_ScanTimeDest(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	dest := d.ScanTimeDest()

	sp, ok := dest.(*string)
	if !ok {
		t.Fatalf("ScanTimeDest() = %T, want *string", dest)
	}

	if *sp != "" {
		t.Error("ScanTimeDest() should return pointer to empty string")
	}
}

func TestSQLiteDialect_ParseTime(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	ts := "2024-01-15T10:30:00.123456789Z"

	parsed, err := d.ParseTime(&ts)
	if err != nil {
		t.Fatalf("ParseTime: %v", err)
	}

	expected, _ := time.Parse(time.RFC3339Nano, ts)
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

func TestSQLiteDialect_ParseTime_Empty(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	empty := ""

	parsed, err := d.ParseTime(&empty)
	if err != nil {
		t.Fatalf("ParseTime empty: %v", err)
	}

	if !parsed.IsZero() {
		t.Errorf("ParseTime empty should return zero time, got %v", parsed)
	}
}

func TestSQLiteDialect_ParseTime_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	d := sqlpkg.SQLiteDialect{}
	bad := "not-a-date"

	_, err := d.ParseTime(&bad)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}

	if !errors.Is(err, sqlpkg.ErrUnsupportedTimestamp) {
		t.Errorf("error = %v, want sqlpkg.ErrUnsupportedTimestamp", err)
	}
}

func TestPlaceholders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dialect  sqlpkg.Dialect
		count    int
		offset   int
		expected string
	}{
		{"postgres 3 from 0", sqlpkg.PostgresDialect{}, 3, 0, "$1, $2, $3"},
		{"postgres 2 from 5", sqlpkg.PostgresDialect{}, 2, 5, "$6, $7"},
		{"sqlite 3 from 0", sqlpkg.SQLiteDialect{}, 3, 0, "?, ?, ?"},
		{"sqlite 2 from 5", sqlpkg.SQLiteDialect{}, 2, 5, "?, ?"},
	}

	for _, tt := range tests {
		got := sqlpkg.Placeholders(tt.dialect, tt.count, tt.offset)
		if got != tt.expected {
			t.Errorf("%s: sqlpkg.Placeholders(%d, %d) = %q, want %q",
				tt.name, tt.count, tt.offset, got, tt.expected)
		}
	}
}
