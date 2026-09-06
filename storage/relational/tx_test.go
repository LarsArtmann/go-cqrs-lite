package relational

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

var errIntentionalHandlerFailure = errors.New("intentional handler failure")

func TestSinkTx_ReturnsActiveTransaction(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	var capturedTx *sql.Tx

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		capturedTx = sink.Tx()
		return nil
	}

	proj, err := NewRelationalProjection("tx-test", schema, db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "X", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	if capturedTx == nil {
		t.Fatalf("sink.Tx() returned nil")
	}
}

func TestSinkTx_RawSQLCommitsWithSinkWrites(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Upsert(ctx, "guilds", Row{
			"id":         "g1",
			"name":       "Test Guild",
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-01T00:00:00Z",
		}); err != nil {
			return err
		}

		tx := sink.Tx()
		if tx == nil {
			t.Fatalf("sink.Tx() returned nil")
		}

		if _, err := tx.ExecContext(ctx,
			"UPDATE guilds SET name = 'Updated Guild' WHERE id = 'g1'"); err != nil {
			return err
		}

		return nil
	}

	proj, err := NewRelationalProjection("tx-raw", schema, db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "X", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	var name string
	err = db.QueryRowContext(ctx, "SELECT name FROM guilds WHERE id = 'g1'").Scan(&name)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if name != "Updated Guild" {
		t.Fatalf("expected 'Updated Guild', got %q", name)
	}
}

func TestSinkTx_RawSQLRollsBackOnHandlerError(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	handler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Upsert(ctx, "guilds", Row{
			"id":         "g1",
			"name":       "Original",
			"created_at": "2026-01-01T00:00:00Z",
			"updated_at": "2026-01-01T00:00:00Z",
		}); err != nil {
			return err
		}

		tx := sink.Tx()

		if _, err := tx.ExecContext(ctx,
			"UPDATE guilds SET name = 'Should Rollback' WHERE id = 'g1'"); err != nil {
			return err
		}

		return errIntentionalHandlerFailure
	}

	proj, err := NewRelationalProjection("tx-rollback", schema, db, sqlpkg.SQLiteDialect{},
		handler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if err == nil {
		t.Fatalf("handler should have returned error")
	}

	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM guilds").Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if count != 0 {
		t.Fatalf("raw SQL via Tx() should roll back with sink writes, got count=%d", count)
	}
}
