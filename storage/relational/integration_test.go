package relational

import (
	"context"
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

// TestIntegration_EndToEnd exercises the full Phase-0 enrichment in one
// pipeline: a declarative schema (Default/FK/Indexes/Uniques) is migrated, a
// sequence of projection handlers exercises every ProjectionSink method
// (Ensure, Upsert, UpsertCols, UpsertExpr, Increment, QueryOne, Update,
// DeleteWhere, Tx escape-hatch), and a RelationalStore verifies the reads
// (Count, CountMany, filtered+paginated Query). This is the single test that
// would break if any new capability regresses — the contract the 21 DiscordSync
// projections will rely on.
func TestIntegration_EndToEnd(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)
	schema := integrationSchema()

	// Migrate once for the whole pipeline. NewRelationalProjection also
	// migrates, but doing it explicitly documents the read-side dependency:
	// RelationalStore assumes the schema already exists.
	if err := schema.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	dialect := sqlpkg.SQLiteDialect{}

	// Phase A — CreateMessage: ensure FK parents, upsert the message, and seed
	// the reaction rollup counter. Proves Ensure + Upsert + Increment compose
	// atomically inside one Handle call.
	createHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Ensure(ctx, "guilds", Row{
			"id": "g1", "name": "Test Guild", "created_at": "2026-01-01T00:00:00Z",
		}); err != nil {
			return err
		}

		if err := sink.Ensure(ctx, "channels", Row{
			"id": "c1", "guild_id": "g1", "name": "general",
		}); err != nil {
			return err
		}

		for i, content := range []string{"first", "second", "third"} {
			msgID := fmt.Sprintf("m%d", i+1)

			if err := sink.Upsert(ctx, "messages", Row{
				"id":         msgID,
				"channel_id": "c1",
				"author_id":  "u-author",
				"content":    content,
				"created_at": "2026-01-01T00:00:00Z",
			}); err != nil {
				return err
			}

			// Seed each message's rollup counter to 0 via Increment(+0), then
			// the reaction phase adds real deltas.
			if err := sink.Increment(ctx, "reaction_counts", Row{
				"message_id": msgID,
			}, "total", 0); err != nil {
				return err
			}
		}

		return nil
	}

	proj, err := NewRelationalProjection("int-create", schema, db, dialect, createHandler, nil)
	if err != nil {
		t.Fatalf("new create projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "CreateMessages", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	store, err := NewRelationalStore(schema, db, dialect)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// Read checkpoint A: 3 messages, indexed channel filter, CountMany.
	t.Run("phaseA_reads", func(t *testing.T) {
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

		// Default value: guilds.member_count should be 0 (Default "0"), not NULL,
		// because Ensure inserted without member_count.
		var mc int
		err = db.QueryRowContext(ctx, "SELECT member_count FROM guilds WHERE id='g1'").Scan(&mc)
		if err != nil {
			t.Fatalf("query member_count: %v", err)
		}
		if mc != 0 {
			t.Fatalf("default member_count should be 0, got %d", mc)
		}
	})

	// Phase B — EditMessage: read-then-write via QueryOne, record the edit
	// history via the Tx() escape hatch, then partial-update content + edited_at
	// with UpsertCols (created_at and author_id must be preserved).
	editHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		// QueryOne: read current content before recording history.
		before, err := sink.QueryOne(ctx, "messages", "content", Row{"id": "m1"})
		if err != nil {
			return err
		}

		// Tx() escape hatch: append to the autoincrement history table with raw
		// SQL (no sink method covers INSERT ... RETURNING / autoincrement rows).
		tx := sink.Tx()
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO message_edits (message_id, before_content, after_content, edited_at) VALUES (?, ?, ?, ?)",
			"m1", before, "first (edited)", "2026-01-02T00:00:00Z"); err != nil {
			return err
		}

		// UpsertCols: partial update — only content + edited_at; created_at and
		// author_id must survive untouched.
		return sink.UpsertCols(ctx, "messages", Row{
			"id":         "m1",
			"channel_id": "c1",
			"author_id":  "SHOULD_NOT_OVERWRITE",
			"content":    "first (edited)",
			"created_at": "2099-01-01T00:00:00Z",
			"edited_at":  "2026-01-02T00:00:00Z",
		}, []string{"content", "edited_at"})
	}

	projEdit, err := NewRelationalProjection("int-edit", schema, db, dialect, editHandler, nil)
	if err != nil {
		t.Fatalf("new edit projection: %v", err)
	}

	if err := projEdit.Handle(ctx, newEvent(t, "EditMessage", nil)); err != nil {
		t.Fatalf("handle edit: %v", err)
	}

	t.Run("phaseB_partial_update_preserves_columns", func(t *testing.T) {
		var content, author, created, edited string
		err = db.QueryRowContext(ctx,
			"SELECT content, author_id, created_at, edited_at FROM messages WHERE id='m1'").
			Scan(&content, &author, &created, &edited)
		if err != nil {
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

		// Tx() escape hatch wrote a history row.
		var edits int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM message_edits WHERE message_id='m1'").Scan(&edits)
		if err != nil {
			t.Fatalf("count edits: %v", err)
		}
		if edits != 1 {
			t.Fatalf("expected 1 edit history row, got %d", edits)
		}
	})

	// Phase C — Reaction rollup: Increment the counter and UpsertExpr to bump
	// edited_at with a COALESCE that ignores empty values.
	reactHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Increment(ctx, "reaction_counts", Row{
			"message_id": "m1",
		}, "total", 5); err != nil {
			return err
		}
		if err := sink.Increment(ctx, "reaction_counts", Row{
			"message_id": "m1",
		}, "total", 3); err != nil {
			return err
		}
		// UpsertExpr with a bound arg: append a marker to content on conflict.
		return sink.UpsertExpr(ctx, "messages", Row{
			"id":         "m1",
			"channel_id": "c1",
			"author_id":  "u-author",
			"content":    "",
			"created_at": "2026-01-01T00:00:00Z",
		}, []SetExpr{{
			Column: "content",
			Expr:   "messages.content || ?",
			Args:   []any{" ❤"},
		}})
	}

	projReact, err := NewRelationalProjection("int-react", schema, db, dialect, reactHandler, nil)
	if err != nil {
		t.Fatalf("new react projection: %v", err)
	}

	if err := projReact.Handle(ctx, newEvent(t, "React", nil)); err != nil {
		t.Fatalf("handle react: %v", err)
	}

	t.Run("phaseC_increment_and_expr", func(t *testing.T) {
		var total int
		err = db.QueryRowContext(ctx, "SELECT total FROM reaction_counts WHERE message_id='m1'").Scan(&total)
		if err != nil {
			t.Fatalf("query total: %v", err)
		}
		if total != 8 {
			t.Fatalf("incremented total should be 5+3=8, got %d", total)
		}
		var content string
		err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id='m1'").Scan(&content)
		if err != nil {
			t.Fatalf("query content: %v", err)
		}
		if content != "first (edited) ❤" {
			t.Fatalf("UpsertExpr bound-arg should append ' ❤', got %q", content)
		}
	})

	// Phase D — Bulk update + delete by predicate: mark a message and delete
	// its reaction rollup, proving Update + DeleteWhere work set-predicate style.
	mutateHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Update(ctx, "messages",
			Row{"content": "deleted"},
			Row{"id": "m3"}); err != nil {
			return err
		}
		return sink.DeleteWhere(ctx, "reaction_counts", Row{"message_id": "m3"})
	}

	projMutate, err := NewRelationalProjection("int-mutate", schema, db, dialect, mutateHandler, nil)
	if err != nil {
		t.Fatalf("new mutate projection: %v", err)
	}

	if err := projMutate.Handle(ctx, newEvent(t, "Mutate", nil)); err != nil {
		t.Fatalf("handle mutate: %v", err)
	}

	t.Run("phaseD_update_and_deletewhere", func(t *testing.T) {
		var content string
		err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id='m3'").Scan(&content)
		if err != nil {
			t.Fatalf("query m3: %v", err)
		}
		if content != "deleted" {
			t.Fatalf("Update should set content to 'deleted', got %q", content)
		}
		var n int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reaction_counts WHERE message_id='m3'").Scan(&n)
		if err != nil {
			t.Fatalf("query deleted rollup: %v", err)
		}
		if n != 0 {
			t.Fatalf("DeleteWhere should have removed m3 rollup, got %d rows", n)
		}
	})

	// Phase E — paginated Query reads via the store: verify ordering + offset.
	t.Run("phaseE_paginated_query", func(t *testing.T) {
		var ids []string
		err = store.Query(ctx, "messages", []string{"id"}, kv.ViewQuery{
			OrderBy: "id",
			Limit:   2,
		}, func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
			return nil
		})
		if err != nil {
			t.Fatalf("query page 1: %v", err)
		}
		if len(ids) != 2 || ids[0] != "m1" || ids[1] != "m2" {
			t.Fatalf("page 1 wrong: %v", ids)
		}

		var ids2 []string
		err = store.Query(ctx, "messages", []string{"id"}, kv.ViewQuery{
			OrderBy: "id",
			Limit:   2,
			Offset:  2,
		}, func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}
			ids2 = append(ids2, id)
			return nil
		})
		if err != nil {
			t.Fatalf("query page 2: %v", err)
		}
		if len(ids2) != 1 || ids2[0] != "m3" {
			t.Fatalf("page 2 wrong: %v", ids2)
		}
	})

	// Phase F — atomic rollback: a handler that writes then returns an error
	// must leave the database untouched (proves the transaction boundary).
	rollbackHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Upsert(ctx, "messages", Row{
			"id":         "m-ghost",
			"channel_id": "c1",
			"author_id":  "u-author",
			"content":    "should-not-exist",
			"created_at": "2026-01-01T00:00:00Z",
		}); err != nil {
			return err
		}
		return errors.New("intentional handler failure")
	}

	projRollback, err := NewRelationalProjection("int-rollback", schema, db, dialect, rollbackHandler, nil)
	if err != nil {
		t.Fatalf("new rollback projection: %v", err)
	}

	handleErr := projRollback.Handle(ctx, newEvent(t, "Rollback", nil))

	t.Run("phaseF_rollback_on_error", func(t *testing.T) {
		if handleErr == nil {
			t.Fatal("handler should have returned an error")
		}
		var n int
		err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages WHERE id='m-ghost'").Scan(&n)
		if err != nil {
			t.Fatalf("query ghost: %v", err)
		}
		if n != 0 {
			t.Fatalf("rollback should have prevented the ghost row, got %d", n)
		}
	})
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

	want := []string{"idx_channels_guild", "idx_messages_channel", "idx_messages_author"}
	for _, name := range want {
		var got string
		err := db.QueryRowContext(ctx,
			"SELECT name FROM sqlite_master WHERE type='index' AND name=?", name).Scan(&got)
		if err != nil {
			t.Fatalf("index %q not created: %v", name, err)
		}
	}
}
