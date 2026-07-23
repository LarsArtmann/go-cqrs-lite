package sql_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func openSQLite(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestSchema(t *testing.T) {
	t.Parallel()

	s := sqlpkg.Schema()
	if s == "" {
		t.Error("Schema() returned empty string")
	}

	if !contains(s, "events") {
		t.Error("Schema() should contain 'events' table")
	}
}

func TestSQLiteSchema(t *testing.T) {
	t.Parallel()

	s := sqlpkg.SQLiteSchema()
	if s == "" {
		t.Error("SQLiteSchema() returned empty string")
	}
}

func TestReconstructEvent(t *testing.T) {
	t.Parallel()

	eventID := id.NewEventID()
	aggID := id.NewStreamID()
	now := time.Now().UTC().Truncate(time.Millisecond)

	evt, err := sqlpkg.ReconstructEvent(
		eventID, event.Type("user.created"), id.StreamType("User"), aggID,
		1, 1,
		[]byte(`{"name":"Alice"}`),
		[]byte(`{}`),
		now,
		codec.EncodingJSON,
	)
	if err != nil {
		t.Fatalf("ReconstructEvent: %v", err)
	}

	if evt.Type() != "user.created" {
		t.Errorf("Type = %q, want %q", evt.Type(), "user.created")
	}

	if evt.StreamID() != aggID {
		t.Errorf("StreamID = %v, want %v", evt.StreamID(), aggID)
	}

	if evt.Version() != 1 {
		t.Errorf("Version = %d, want 1", evt.Version())
	}
}

func TestMarshalMetadata_Roundtrip(t *testing.T) {
	t.Parallel()

	m := event.Metadata{
		Source: "test",
		Custom: map[event.MetadataKey]string{"correlation_id": "abc-123"},
	}

	data, err := sqlpkg.MarshalMetadata(m)
	if err != nil {
		t.Fatalf("MarshalMetadata: %v", err)
	}

	opts, err := sqlpkg.UnmarshalEventMetadata(data, "test.event")
	if err != nil {
		t.Fatalf("UnmarshalEventMetadata: %v", err)
	}

	if len(opts) == 0 {
		t.Error("expected at least one option from unmarshaled metadata")
	}
}

func TestOwnedDBHandle_NilDB(t *testing.T) {
	t.Parallel()

	_, err := sqlpkg.NewBorrowedDBHandle(nil, sqlpkg.SQLiteDialect{})
	if err == nil {
		t.Error("expected error for nil DB")
	}
}

func TestOwnedDBHandle_Lifecycle(t *testing.T) {
	db := openSQLite(t)

	cb, err := sqlpkg.NewBorrowedDBHandle(db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewBorrowedDBHandle: %v", err)
	}

	if err := cb.CheckClosed(errors.New("closed")); err != nil {
		t.Error("CheckClosed should return nil when not closed")
	}

	if err := cb.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if err := cb.CheckClosed(errors.New("closed")); err == nil {
		t.Error("CheckClosed should return error after Close")
	}
}

func TestOwnedDBHandle_OwnedDB(t *testing.T) {
	db := openSQLite(t)

	cb, err := sqlpkg.NewOwningDBHandle(db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("NewOwningDBHandle: %v", err)
	}

	if err := cb.Close(); err != nil {
		t.Fatalf("Close with owned DB: %v", err)
	}
}

func TestTracer(t *testing.T) {
	t.Parallel()

	tr := sqlpkg.Tracer()
	if tr == nil {
		t.Error("Tracer() returned nil")
	}
}

func TestStartStreamSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())

	_, span := sqlpkg.StartStreamSpan(ctx, "test.span", ref)
	if span == nil {
		t.Error("StartStreamSpan returned nil span")
	}

	span.End()
}

func TestStartSaveSpan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ref := id.NewStreamRef("User", id.NewStreamID())

	_, span := sqlpkg.StartSaveSpan(ctx, "test.save", ref, event.Version(0), 3)
	if span == nil {
		t.Error("StartSaveSpan returned nil span")
	}

	span.End()
}

func TestCommitTx(t *testing.T) {
	db := openSQLite(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, "CREATE TABLE test (id INTEGER PRIMARY KEY)")

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	_, _ = tx.ExecContext(ctx, "INSERT INTO test (id) VALUES (1)")

	if err := sqlpkg.CommitTx(tx); err != nil {
		t.Fatalf("CommitTx: %v", err)
	}
}

func TestScanSlice_Empty(t *testing.T) {
	db := openSQLite(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, "CREATE TABLE test (val TEXT)")

	rows, err := db.QueryContext(ctx, "SELECT val FROM test")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	results, err := sqlpkg.ScanSlice[string](rows, func(r *sql.Rows) (string, error) {
		var s string
		return s, r.Scan(&s)
	})
	if err != nil {
		t.Fatalf("ScanSlice: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestScanSlice_WithData(t *testing.T) {
	db := openSQLite(t)
	ctx := context.Background()

	_, _ = db.ExecContext(ctx, "CREATE TABLE test (val TEXT)")
	_, _ = db.ExecContext(ctx, "INSERT INTO test (val) VALUES ('a')")
	_, _ = db.ExecContext(ctx, "INSERT INTO test (val) VALUES ('b')")

	rows, err := db.QueryContext(ctx, "SELECT val FROM test ORDER BY val")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	results, err := sqlpkg.ScanSlice[string](rows, func(r *sql.Rows) (string, error) {
		var s string
		return s, r.Scan(&s)
	})
	if err != nil {
		t.Fatalf("ScanSlice: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0] != "a" || results[1] != "b" {
		t.Errorf("results = %v, want [a b]", results)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
