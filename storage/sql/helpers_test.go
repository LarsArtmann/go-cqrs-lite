package sql_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

func setupEventsTable(t *testing.T) *sql.DB {
	t.Helper()

	db := openSQLite(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, sqlpkg.SQLiteSchema())
	if err != nil {
		t.Fatalf("create events table: %v", err)
	}

	return db
}

func setupCheckpointsTable(t *testing.T) *sql.DB {
	t.Helper()

	db := openSQLite(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, sqlpkg.SQLiteDialect{}.CheckpointSchema())
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
		id.AggregateType("User"),
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

func beginTx(t *testing.T, db *sql.DB) *sql.Tx {
	t.Helper()

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}

	return tx
}

func rollbackOnFail(t *testing.T, tx *sql.Tx) {
	t.Helper()

	_ = tx.Rollback()
}

func TestSharedInsertEvents_Success(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewAggregateRef("User", id.NewAggregateID())
	ctx := context.Background()

	tx := beginTx(t, db)
	defer rollbackOnFail(t, tx)

	events := []event.Event{makeTestEvent(t, 1), makeTestEvent(t, 2)}

	err := sqlpkg.SharedInsertEvents(
		ctx, tx, ref, events,
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
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestSharedInsertEvents_ErrorPath(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	tx := beginTx(t, db)

	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	err := sqlpkg.SharedInsertEvents(
		context.Background(), tx, ref, []event.Event{makeTestEvent(t, 1)},
		sqliteInsertQuery(),
		func(t time.Time) any { return t.Format(time.RFC3339Nano) },
	)
	if err == nil {
		t.Fatal("expected error for closed transaction")
	}
}

func TestSharedCheckVersion_Match(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewAggregateRef("User", id.NewAggregateID())
	ctx := context.Background()

	evt := makeTestEvent(t, 1)
	tx := beginTx(t, db)
	defer rollbackOnFail(t, tx)

	_ = sqlpkg.SharedInsertEvents(
		ctx, tx, ref, []event.Event{evt},
		sqliteInsertQuery(),
		func(t time.Time) any { return t.Format(time.RFC3339Nano) },
	)

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(version), 0) FROM %s WHERE aggregate_type = ? AND aggregate_id = ?",
		sqlpkg.TableEvents,
	)

	err := sqlpkg.SharedCheckVersion(ctx, tx, ref, event.Version(1), query)
	if err != nil {
		t.Fatalf("SharedCheckVersion: %v", err)
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

func TestSharedCheckVersion_DBError(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	tx := beginTx(t, db)
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(version), 0) FROM %s WHERE aggregate_type = ? AND aggregate_id = ?",
		sqlpkg.TableEvents,
	)

	err := sqlpkg.SharedCheckVersion(context.Background(), tx, ref, event.Version(0), query)
	if err == nil {
		t.Fatal("expected error for closed transaction")
	}
}

func TestSharedCheckVersion_Mismatch(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	ref := id.NewAggregateRef("User", id.NewAggregateID())

	tx := beginTx(t, db)
	defer rollbackOnFail(t, tx)

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(version), 0) FROM %s WHERE aggregate_type = ? AND aggregate_id = ?",
		sqlpkg.TableEvents,
	)

	err := sqlpkg.SharedCheckVersion(context.Background(), tx, ref, event.Version(5), query)
	if err == nil {
		t.Fatal("expected version conflict error")
	}

	if !errors.Is(err, sqlpkg.ErrConcurrencyConflict) {
		t.Errorf("err = %v, want ErrConcurrencyConflict", err)
	}
}

func TestSharedCheckpointLoad_DBError(t *testing.T) {
	t.Parallel()

	db := setupCheckpointsTable(t)
	_ = db.Close()

	_, err := sqlpkg.SharedCheckpointLoad(
		context.Background(), db, "my-projection", sqlpkg.SQLiteDialect{},
	)
	if err == nil {
		t.Fatal("expected error for closed database")
	}
}

func TestSharedCheckpointSave_DBError(t *testing.T) {
	t.Parallel()

	db := setupCheckpointsTable(t)
	_ = db.Close()

	err := sqlpkg.SharedCheckpointSave(
		context.Background(), db, "my-projection",
		event.Checkpoint{EventID: id.NewEventID(), ProcessedAt: time.Now()},
		sqlpkg.SQLiteDialect{},
	)
	if err == nil {
		t.Fatal("expected error for closed database")
	}
}

func TestDeleteByAggregate_ErrorPath(t *testing.T) {
	t.Parallel()

	db := setupEventsTable(t)
	_ = db.Close()

	ref := id.NewAggregateRef("User", id.NewAggregateID())
	ctx := context.Background()

	err := sqlpkg.DeleteByAggregate(db, ctx, ref, sqlpkg.TableEvents, "?", "?", "events")
	if err == nil {
		t.Fatal("expected error for closed database")
	}
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
	ref := id.NewAggregateRef("User", id.NewAggregateID())
	ctx := context.Background()

	tx := beginTx(t, db)

	_ = sqlpkg.SharedInsertEvents(
		ctx, tx, ref, []event.Event{makeTestEvent(t, 1)},
		sqliteInsertQuery(),
		func(t time.Time) any { return t.Format(time.RFC3339Nano) },
	)
	_ = tx.Commit()

	err := sqlpkg.DeleteByAggregate(db, ctx, ref, sqlpkg.TableEvents, "?", "?", "events")
	if err != nil {
		t.Fatalf("DeleteByAggregate: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&count); err != nil {
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
	ref1 := id.NewAggregateRef("User", id.NewAggregateID())
	ref2 := id.NewAggregateRef("User", id.NewAggregateID())

	for _, ref := range []id.AggregateRef{ref1, ref2} {
		tx := beginTx(t, db)
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
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (other aggregate untouched)", count)
	}
}
