package sql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v2/sql"
)

func setupEventsTable(t *testing.T) *sql.DB {
	t.Helper()

	db := openSQLite(t)
	_, err := db.Exec(sqlpkg.SQLiteSchema())
	if err != nil {
		t.Fatalf("create events table: %v", err)
	}

	return db
}

func setupCheckpointsTable(t *testing.T) *sql.DB {
	t.Helper()

	db := openSQLite(t)
	_, err := db.Exec(sqlpkg.SQLiteDialect{}.CheckpointSchema())
	if err != nil {
		t.Fatalf("create checkpoints table: %v", err)
	}

	return db
}

func makeTestEvent(t *testing.T, version int) event.Event {
	t.Helper()

	evt, err := event.NewEvent(
		event.Type("user.created"),
		id.NewAggregateID(),
		event.AggregateType("User"),
		event.Version(version),
		[]byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func sqliteInsertQuery() string {
	return fmt.Sprintf(
		"INSERT INTO %s (id, event_type, aggregate_type, aggregate_id, version, schema_version, payload, payload_encoding, metadata, occurred_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		sqlpkg.TableEvents,
	)
}

func TestSharedInsertEvents_Success(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := event.NewAggregateRef("User", id.NewAggregateID())

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	events := []event.Event{makeTestEvent(t, 1), makeTestEvent(t, 2)}

	err = sqlpkg.SharedInsertEvents(
		context.Background(), tx, ref, events,
		sqliteInsertQuery(),
		func(t time.Time) any { return t.Format(time.RFC3339Nano) },
	)
	if err != nil {
		t.Fatalf("SharedInsertEvents: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestSharedCheckVersion_Match(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := event.NewAggregateRef("User", id.NewAggregateID())

	evt := makeTestEvent(t, 1)
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	_ = sqlpkg.SharedInsertEvents(
		context.Background(), tx, ref, []event.Event{evt},
		sqliteInsertQuery(),
		func(t time.Time) any { return t.Format(time.RFC3339Nano) },
	)

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(version), 0) FROM %s WHERE aggregate_type = ? AND aggregate_id = ?",
		sqlpkg.TableEvents,
	)

	err = sqlpkg.SharedCheckVersion(context.Background(), tx, ref, event.Version(1), query)
	if err != nil {
		t.Fatalf("SharedCheckVersion: %v", err)
	}

	_ = tx.Commit()
}

func TestSharedCheckVersion_Mismatch(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := event.NewAggregateRef("User", id.NewAggregateID())

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	// No events inserted, so MAX(version) = 0
	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(version), 0) FROM %s WHERE aggregate_type = ? AND aggregate_id = ?",
		sqlpkg.TableEvents,
	)

	err = sqlpkg.SharedCheckVersion(context.Background(), tx, ref, event.Version(5), query)
	if err == nil {
		t.Fatal("expected version conflict error")
	}

	if !errors.Is(err, sqlpkg.ErrConcurrencyConflict) {
		t.Errorf("err = %v, want ErrConcurrencyConflict", err)
	}

	_ = tx.Rollback()
}

func TestSharedCheckpointLoad_NoRows(t *testing.T) {
	t.Parallel()

	db := setupCheckpointsTable(t)

	cp, err := sqlpkg.SharedCheckpointLoad(
		context.Background(), db, "my-projection", sqlpkg.SQLiteDialect{},
	)
	if err != nil {
		t.Fatalf("SharedCheckpointLoad: %v", err)
	}
	if cp.EventID != (id.EventID{}) {
		t.Errorf("expected zero EventID for no rows, got %v", cp.EventID)
	}
}

func TestSharedCheckpointSaveAndLoad(t *testing.T) {
	t.Parallel()

	db := setupCheckpointsTable(t)
	ctx := context.Background()
	d := sqlpkg.SQLiteDialect{}

	eid := id.NewEventID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	cp := event.Checkpoint{EventID: eid, ProcessedAt: now}

	err := sqlpkg.SharedCheckpointSave(ctx, db, "my-projection", cp, d)
	if err != nil {
		t.Fatalf("SharedCheckpointSave: %v", err)
	}

	loaded, err := sqlpkg.SharedCheckpointLoad(ctx, db, "my-projection", d)
	if err != nil {
		t.Fatalf("SharedCheckpointLoad: %v", err)
	}

	if loaded.EventID != eid {
		t.Errorf("EventID = %v, want %v", loaded.EventID, eid)
	}
}

func TestDeleteByAggregate(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := event.NewAggregateRef("User", id.NewAggregateID())
	ctx := context.Background()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	_ = sqlpkg.SharedInsertEvents(
		ctx, tx, ref, []event.Event{makeTestEvent(t, 1)},
		sqliteInsertQuery(),
		func(t time.Time) any { return t.Format(time.RFC3339Nano) },
	)
	_ = tx.Commit()

	err = sqlpkg.DeleteByAggregate(db, ctx, ref, sqlpkg.TableEvents, "?", "?", "events")
	if err != nil {
		t.Fatalf("DeleteByAggregate: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 after delete", count)
	}
}

func TestDeleteByAggregate_OtherAggregateUntouched(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ctx := context.Background()
	ref1 := event.NewAggregateRef("User", id.NewAggregateID())
	ref2 := event.NewAggregateRef("User", id.NewAggregateID())

	for _, ref := range []event.AggregateRef{ref1, ref2} {
		tx, _ := db.Begin()
		_ = sqlpkg.SharedInsertEvents(
			ctx, tx, ref, []event.Event{makeTestEvent(t, 1)},
			sqliteInsertQuery(),
			func(t time.Time) any { return t.Format(time.RFC3339Nano) },
		)
		_ = tx.Commit()
	}

	err := sqlpkg.DeleteByAggregate(db, ctx, ref1, sqlpkg.TableEvents, "?", "?", "events")
	if err != nil {
		t.Fatalf("DeleteByAggregate: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (other aggregate untouched)", count)
	}
}
