package relational

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// integrationSchema is a richer schema than discordSchema: it exercises every
// enrichment added in Phase 0 — column Defaults (member_count, count, total),
// a foreign-key References (channels.guild_id), per-table IndexSpec (channels,
// messages), a composite UniqueSpec (reactions), and a counter rollup table
// (reaction_counts) for Increment. The append-only message_edits table covers
// the autoincrement-PK escape-hatch path.
func integrationSchema() RelationalSchema {
	return RelationalSchema{Tables: []RelationalTable{
		{
			Name: "guilds", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "name", Type: "TEXT"},
				{Name: "member_count", Type: "INTEGER", Default: "0"},
				{Name: "created_at", Type: "TEXT"},
			},
		},
		{
			Name: "channels", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "guild_id", Type: "TEXT", References: "guilds(id)", Nullable: true},
				{Name: "name", Type: "TEXT"},
			},
			Indexes: []IndexSpec{
				{Name: "idx_channels_guild", Columns: []string{"guild_id"}},
			},
		},
		{
			Name: "messages", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "channel_id", Type: "TEXT"},
				{Name: "author_id", Type: "TEXT"},
				{Name: "content", Type: "TEXT"},
				{Name: "created_at", Type: "TEXT"},
				{Name: "edited_at", Type: "TEXT", Nullable: true},
			},
			Indexes: []IndexSpec{
				{Name: "idx_messages_channel", Columns: []string{"channel_id"}},
				{Name: "idx_messages_author", Columns: []string{"author_id"}},
			},
		},
		{
			Name: "reactions", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "message_id", Type: "TEXT"},
				{Name: "emoji", Type: "TEXT"},
				{Name: "count", Type: "INTEGER", Default: "0"},
			},
			Uniques: []UniqueSpec{
				{Name: "uq_reactions_msg_emoji", Columns: []string{"message_id", "emoji"}},
			},
		},
		{
			Name:       "reaction_counts",
			PrimaryKey: []string{"message_id"},
			Columns: []RelationalColumn{
				{Name: "message_id", Type: "TEXT"},
				{Name: "total", Type: "INTEGER", Default: "0"},
			},
		},
		{
			Name: "message_edits",
			Columns: []RelationalColumn{
				{Name: "id", Type: "INTEGER PRIMARY KEY AUTOINCREMENT", Nullable: true},
				{Name: "message_id", Type: "TEXT"},
				{Name: "before_content", Type: "TEXT"},
				{Name: "after_content", Type: "TEXT"},
				{Name: "edited_at", Type: "TEXT"},
			},
		},
	}}
}

// intCreateHandler ensures FK parents, upserts 3 messages, and seeds each
// message's rollup counter. Proves Ensure + Upsert + Increment compose
// atomically inside one Handle call.
func intCreateHandler(ctx context.Context) RelationalHandler {
	return func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		steps := []struct {
			table string
			row   Row
		}{
			{"guilds", Row{"id": "g1", "name": "Test Guild", "created_at": "2026-01-01T00:00:00Z"}},
			{"channels", Row{"id": "c1", "guild_id": "g1", "name": "general"}},
		}
		for _, s := range steps {
			if err := sink.Ensure(ctx, s.table, s.row); err != nil {
				return err
			}
		}

		for i, content := range []string{"first", "second", "third"} {
			msgID := fmt.Sprintf("m%d", i+1)

			if err := sink.Upsert(ctx, "messages", Row{
				"id": msgID, "channel_id": "c1", "author_id": "u-author",
				"content": content, "created_at": "2026-01-01T00:00:00Z",
			}); err != nil {
				return err
			}

			if err := sink.Increment(ctx, "reaction_counts", Row{"message_id": msgID}, "total", 0); err != nil {
				return err
			}
		}

		return nil
	}
}

// intEditHandler reads current content (QueryOne), records an edit-history row
// via the Tx() escape hatch, then partial-updates content + edited_at with
// UpsertCols (created_at and author_id preserved).
func intEditHandler(ctx context.Context) RelationalHandler {
	return func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		before, err := sink.QueryOne(ctx, "messages", "content", Row{"id": "m1"})
		if err != nil {
			return err
		}

		tx := sink.Tx()
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO message_edits (message_id, before_content, after_content, edited_at) VALUES (?, ?, ?, ?)",
			"m1", before, "first (edited)", "2026-01-02T00:00:00Z"); err != nil {
			return err
		}

		return sink.UpsertCols(ctx, "messages", Row{
			"id": "m1", "channel_id": "c1", "author_id": "SHOULD_NOT_OVERWRITE",
			"content": "first (edited)", "created_at": "2099-01-01T00:00:00Z",
			"edited_at": "2026-01-02T00:00:00Z",
		}, []string{"content", "edited_at"})
	}
}

// intReactHandler Increments the rollup counter twice and appends a marker to
// content via UpsertExpr with a bound arg.
func intReactHandler(ctx context.Context) RelationalHandler {
	return func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		for _, delta := range []int64{5, 3} {
			if err := sink.Increment(ctx, "reaction_counts", Row{"message_id": "m1"}, "total", delta); err != nil {
				return err
			}
		}

		return sink.UpsertExpr(ctx, "messages", Row{
			"id": "m1", "channel_id": "c1", "author_id": "u-author",
			"content": "", "created_at": "2026-01-01T00:00:00Z",
		}, []SetExpr{{
			Column: "content",
			Expr:   "messages.content || ?",
			Args:   []any{" ❤"},
		}})
	}
}

// intMutateHandler bulk-updates one message and deletes another's rollup row.
func intMutateHandler(ctx context.Context) RelationalHandler {
	return func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Update(ctx, "messages", Row{"content": "deleted"}, Row{"id": "m3"}); err != nil {
			return err
		}

		return sink.DeleteWhere(ctx, "reaction_counts", Row{"message_id": "m3"})
	}
}

// intRollbackHandler writes a row then returns an error, proving the
// transaction rolls back atomically.
func intRollbackHandler(ctx context.Context) RelationalHandler {
	return func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Upsert(ctx, "messages", Row{
			"id": "m-ghost", "channel_id": "c1", "author_id": "u-author",
			"content": "should-not-exist", "created_at": "2026-01-01T00:00:00Z",
		}); err != nil {
			return err
		}

		return errors.New("intentional handler failure")
	}
}

// intHandle wires a handler into a projection and handles one event. Returns
// the handle error so callers can assert on success or intentional failure.
func intHandle(
	t *testing.T, name string, schema RelationalSchema, db *sql.DB,
	dialect sqlpkg.Dialect, ctx context.Context, handler RelationalHandler,
) error {
	t.Helper()

	proj, err := NewRelationalProjection(name, schema, db, dialect, handler, nil)
	if err != nil {
		t.Fatalf("new %s projection: %v", name, err)
	}

	return proj.Handle(ctx, newEvent(t, name, nil))
}

// TestIntegration_EndToEnd exercises the full Phase-0 enrichment in one
// pipeline: a declarative schema (Default/FK/Indexes/Uniques) is migrated, a
// sequence of projection handlers exercises every ProjectionSink method
// (Ensure, Upsert, UpsertCols, UpsertExpr, Increment, QueryOne, Update,
// DeleteWhere, Tx escape-hatch), and a RelationalStore verifies the reads
// (Count, CountMany, filtered+paginated Query). The single test that would
// break if any new capability regresses — the contract the 21 DiscordSync
// projections will rely on.
func TestIntegration_EndToEnd(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)
	schema := integrationSchema()
	dialect := sqlpkg.SQLiteDialect{}

	if err := schema.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	mustHandle(t, "CreateMessages", schema, db, dialect, ctx, intCreateHandler(ctx))

	store, err := NewRelationalStore(schema, db, dialect)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	t.Run("phaseA_reads", func(t *testing.T) { intAssertPhaseA(t, store, db, ctx) })

	mustHandle(t, "EditMessage", schema, db, dialect, ctx, intEditHandler(ctx))
	t.Run("phaseB_partial_update", func(t *testing.T) { intAssertPhaseB(t, db, ctx) })

	mustHandle(t, "React", schema, db, dialect, ctx, intReactHandler(ctx))
	t.Run("phaseC_increment_expr", func(t *testing.T) { intAssertPhaseC(t, db, ctx) })

	mustHandle(t, "Mutate", schema, db, dialect, ctx, intMutateHandler(ctx))
	t.Run("phaseD_update_delete", func(t *testing.T) { intAssertPhaseD(t, db, ctx) })
	t.Run("phaseE_paginated_query", func(t *testing.T) { intAssertPhaseE(t, store, ctx) })

	rollbackErr := intHandle(t, "Rollback", schema, db, dialect, ctx, intRollbackHandler(ctx))
	t.Run("phaseF_rollback", func(t *testing.T) { intAssertPhaseF(t, db, ctx, rollbackErr) })
}

// mustHandle runs intHandle and fails the test if the handler returns an error.
func mustHandle(
	t *testing.T, name string, schema RelationalSchema, db *sql.DB,
	dialect sqlpkg.Dialect, ctx context.Context, handler RelationalHandler,
) {
	t.Helper()

	if err := intHandle(t, name, schema, db, dialect, ctx, handler); err != nil {
		t.Fatalf("handle %s: %v", name, err)
	}
}

func intAssertPhaseA(t *testing.T, store *RelationalStore, db *sql.DB, ctx context.Context) {
	t.Helper()

	msgCount, err := store.Count(ctx, "messages", nil)
	if err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if msgCount != 3 {
		t.Fatalf("expected 3 messages, got %d", msgCount)
	}

	chanCount, err := store.Count(ctx, "messages", []kv.Condition{
		{Column: "channel_id", Op: kv.OpEq, Value: "c1"},
	})
	if err != nil {
		t.Fatalf("count by channel: %v", err)
	}
	if chanCount != 3 {
		t.Fatalf("expected 3 messages in c1, got %d", chanCount)
	}

	counts, err := store.CountMany(ctx, []string{"messages", "channels", "guilds"})
	if err != nil {
		t.Fatalf("count many: %v", err)
	}
	if counts["messages"] != 3 || counts["channels"] != 1 || counts["guilds"] != 1 {
		t.Fatalf("CountMany mismatch: %+v", counts)
	}

	var mc int
	if err := db.QueryRowContext(ctx, "SELECT member_count FROM guilds WHERE id='g1'").Scan(&mc); err != nil {
		t.Fatalf("query member_count: %v", err)
	}
	if mc != 0 {
		t.Fatalf("default member_count should be 0, got %d", mc)
	}
}

func intAssertPhaseB(t *testing.T, db *sql.DB, ctx context.Context) {
	t.Helper()

	var content, author, created, edited string
	if err := db.QueryRowContext(ctx,
		"SELECT content, author_id, created_at, edited_at FROM messages WHERE id='m1'").
		Scan(&content, &author, &created, &edited); err != nil {
		t.Fatalf("query m1: %v", err)
	}
	if content != "first (edited)" {
		t.Fatalf("content should be updated, got %q", content)
	}
	if author != "u-author" {
		t.Fatalf("author_id should be preserved, got %q", author)
	}
	if created != "2026-01-01T00:00:00Z" {
		t.Fatalf("created_at should be preserved, got %q", created)
	}
	if edited != "2026-01-02T00:00:00Z" {
		t.Fatalf("edited_at should be set, got %q", edited)
	}

	var edits int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM message_edits WHERE message_id='m1'").Scan(&edits); err != nil {
		t.Fatalf("count edits: %v", err)
	}
	if edits != 1 {
		t.Fatalf("expected 1 edit history row, got %d", edits)
	}
}

func intAssertPhaseC(t *testing.T, db *sql.DB, ctx context.Context) {
	t.Helper()

	var total int
	if err := db.QueryRowContext(ctx, "SELECT total FROM reaction_counts WHERE message_id='m1'").Scan(&total); err != nil {
		t.Fatalf("query total: %v", err)
	}
	if total != 8 {
		t.Fatalf("incremented total should be 5+3=8, got %d", total)
	}

	var content string
	if err := db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id='m1'").Scan(&content); err != nil {
		t.Fatalf("query content: %v", err)
	}
	if content != "first (edited) ❤" {
		t.Fatalf("UpsertExpr bound-arg should append ' ❤', got %q", content)
	}
}

func intAssertPhaseD(t *testing.T, db *sql.DB, ctx context.Context) {
	t.Helper()

	var content string
	if err := db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id='m3'").Scan(&content); err != nil {
		t.Fatalf("query m3: %v", err)
	}
	if content != "deleted" {
		t.Fatalf("Update should set content to 'deleted', got %q", content)
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reaction_counts WHERE message_id='m3'").Scan(&n); err != nil {
		t.Fatalf("query deleted rollup: %v", err)
	}
	if n != 0 {
		t.Fatalf("DeleteWhere should have removed m3 rollup, got %d rows", n)
	}
}

func intAssertPhaseE(t *testing.T, store *RelationalStore, ctx context.Context) {
	t.Helper()

	queryIDs := func(offset, limit int) []string {
		t.Helper()

		var ids []string
		if err := store.Query(ctx, "messages", []string{"id"}, kv.ViewQuery{
			OrderBy: "id", Limit: limit, Offset: offset,
		}, func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
			return nil
		}); err != nil {
			t.Fatalf("query offset=%d: %v", offset, err)
		}
		return ids
	}

	page1 := queryIDs(0, 2)
	if len(page1) != 2 || page1[0] != "m1" || page1[1] != "m2" {
		t.Fatalf("page 1 wrong: %v", page1)
	}

	page2 := queryIDs(2, 2)
	if len(page2) != 1 || page2[0] != "m3" {
		t.Fatalf("page 2 wrong: %v", page2)
	}
}

func intAssertPhaseF(t *testing.T, db *sql.DB, ctx context.Context, handleErr error) {
	t.Helper()

	if handleErr == nil {
		t.Fatal("handler should have returned an error")
	}

	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE id='m-ghost'").Scan(&n); err != nil {
		t.Fatalf("query ghost: %v", err)
	}
	if n != 0 {
		t.Fatalf("rollback should have prevented the ghost row, got %d", n)
	}
}

// TestIntegration_CompositeUniqueEnforced proves the schema's UniqueSpec is
// realised as a real database constraint (not just DDL text): inserting a
// duplicate (message_id, emoji) pair must fail.
func TestIntegration_CompositeUniqueEnforced(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)
	schema := integrationSchema()

	if err := schema.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		"INSERT INTO reactions (id, message_id, emoji, count) VALUES ('r1','m1','👍',1)"); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err := db.ExecContext(ctx,
		"INSERT INTO reactions (id, message_id, emoji, count) VALUES ('r2','m1','👍',1)")
	if err == nil {
		t.Fatal("duplicate (message_id, emoji) should violate the unique constraint")
	}
}

// TestIntegration_IndexesCreated proves the IndexSpec declarations become real
// database indexes after Migrate.
func TestIntegration_IndexesCreated(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)
	schema := integrationSchema()

	if err := schema.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	for _, name := range []string{"idx_channels_guild", "idx_messages_channel", "idx_messages_author"} {
		var got string
		err := db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?", name).Scan(&got)
		if err != nil {
			t.Fatalf("index %q not created: %v", name, err)
		}
	}
}
