package relational

import (
	"context"
	"testing"

	_ "modernc.org/sqlite"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func TestSinkUpsertCols_PartialUpdateOnly(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	insertHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Upsert(ctx, "messages", Row{
			"id":         "m1",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "original",
			"created_at": "2026-01-01T00:00:00Z",
		})
	}

	proj, err := NewRelationalProjection("upsert-cols", schema, db, sqlpkg.SQLiteDialect{},
		insertHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	updateHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertCols(ctx, "messages", Row{
			"id":         "m1",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "DIFFERENT",
			"content":    "edited",
			"created_at": "2099-01-01T00:00:00Z",
		}, []string{"content"})
	}

	proj2, err := NewRelationalProjection("upsert-cols2", schema, db, sqlpkg.SQLiteDialect{},
		updateHandler, nil)
	if err != nil {
		t.Fatalf("new projection2: %v", err)
	}

	if err := proj2.Handle(ctx, newEvent(t, "Update", nil)); err != nil {
		t.Fatalf("handle update: %v", err)
	}

	var content, author, created string
	err = db.QueryRowContext(ctx,
		"SELECT content, author_id, created_at FROM messages WHERE id = 'm1'").Scan(
		&content, &author, &created,
	)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "edited" {
		t.Fatalf("content should be updated to 'edited', got %q", content)
	}

	if author != "u1" {
		t.Fatalf("author_id should be preserved as 'u1', got %q", author)
	}

	if created != "2026-01-01T00:00:00Z" {
		t.Fatalf("created_at should be preserved, got %q", created)
	}
}

func TestSinkUpsertCols_EmptyUpdateColsDoesNothing(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	insertHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Upsert(ctx, "messages", Row{
			"id":         "m2",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "original",
			"created_at": "2026-01-01T00:00:00Z",
		})
	}

	proj, err := NewRelationalProjection("upsert-cols-empty", schema, db, sqlpkg.SQLiteDialect{},
		insertHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	touchHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertCols(ctx, "messages", Row{
			"id":         "m2",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "DIFFERENT",
			"content":    "should-not-apply",
			"created_at": "2099-01-01T00:00:00Z",
		}, nil)
	}

	proj2, err := NewRelationalProjection("upsert-cols-empty2", schema, db, sqlpkg.SQLiteDialect{},
		touchHandler, nil)
	if err != nil {
		t.Fatalf("new projection2: %v", err)
	}

	if err := proj2.Handle(ctx, newEvent(t, "Touch", nil)); err != nil {
		t.Fatalf("handle touch: %v", err)
	}

	var content string
	err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id = 'm2'").Scan(&content)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "original" {
		t.Fatalf(
			"empty updateCols should do nothing, content should be 'original', got %q",
			content,
		)
	}
}

func TestSinkUpsertExpr_COALESCE(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	insertHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Upsert(ctx, "messages", Row{
			"id":         "m3",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "original content",
			"created_at": "2026-01-01T00:00:00Z",
		})
	}

	proj, err := NewRelationalProjection("upsert-expr", schema, db, sqlpkg.SQLiteDialect{},
		insertHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	emptyContentHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertExpr(ctx, "messages", Row{
			"id":         "m3",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "",
			"created_at": "2026-01-01T00:00:00Z",
		}, []SetExpr{{
			Column: "content",
			Expr:   "COALESCE(NULLIF(excluded.content, ''), messages.content)",
		}})
	}

	proj2, err := NewRelationalProjection("upsert-expr2", schema, db, sqlpkg.SQLiteDialect{},
		emptyContentHandler, nil)
	if err != nil {
		t.Fatalf("new projection2: %v", err)
	}

	if err := proj2.Handle(ctx, newEvent(t, "EmptyUpdate", nil)); err != nil {
		t.Fatalf("handle empty update: %v", err)
	}

	var content string
	err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id = 'm3'").Scan(&content)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "original content" {
		t.Fatalf("COALESCE should preserve original content when new is empty, got %q", content)
	}
}

func TestSinkUpsertExpr_NonEmptyContentUpdates(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	insertHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Upsert(ctx, "messages", Row{
			"id":         "m4",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "old content",
			"created_at": "2026-01-01T00:00:00Z",
		})
	}

	proj, err := NewRelationalProjection("upsert-expr3", schema, db, sqlpkg.SQLiteDialect{},
		insertHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	editHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertExpr(ctx, "messages", Row{
			"id":         "m4",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "new content",
			"created_at": "2026-01-01T00:00:00Z",
		}, []SetExpr{{
			Column: "content",
			Expr:   "COALESCE(NULLIF(excluded.content, ''), messages.content)",
		}})
	}

	proj2, err := NewRelationalProjection("upsert-expr4", schema, db, sqlpkg.SQLiteDialect{},
		editHandler, nil)
	if err != nil {
		t.Fatalf("new projection2: %v", err)
	}

	if err := proj2.Handle(ctx, newEvent(t, "Edit", nil)); err != nil {
		t.Fatalf("handle edit: %v", err)
	}

	var content string
	err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id = 'm4'").Scan(&content)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "new content" {
		t.Fatalf("non-empty content should update, got %q", content)
	}
}

func TestSinkUpsertExpr_EmptyExprsDoesNothing(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	insertHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Upsert(ctx, "messages", Row{
			"id":         "m5",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "untouched",
			"created_at": "2026-01-01T00:00:00Z",
		})
	}

	proj, err := NewRelationalProjection("upsert-expr5", schema, db, sqlpkg.SQLiteDialect{},
		insertHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	touchHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertExpr(ctx, "messages", Row{
			"id":         "m5",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "DIFFERENT",
			"content":    "should-not-apply",
			"created_at": "2099-01-01T00:00:00Z",
		}, nil)
	}

	proj2, err := NewRelationalProjection("upsert-expr6", schema, db, sqlpkg.SQLiteDialect{},
		touchHandler, nil)
	if err != nil {
		t.Fatalf("new projection2: %v", err)
	}

	if err := proj2.Handle(ctx, newEvent(t, "Touch", nil)); err != nil {
		t.Fatalf("handle touch: %v", err)
	}

	var content string
	err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id = 'm5'").Scan(&content)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "untouched" {
		t.Fatalf("empty setExprs should do nothing, got %q", content)
	}
}

func TestSinkUpsertExpr_BoundArgs(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	insertHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.Upsert(ctx, "messages", Row{
			"id":         "m6",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "hello",
			"created_at": "2026-01-01T00:00:00Z",
		})
	}

	proj, err := NewRelationalProjection("upsert-expr-args", schema, db, sqlpkg.SQLiteDialect{},
		insertHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	suffixHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertExpr(ctx, "messages", Row{
			"id":         "m6",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "ignored-on-conflict",
			"created_at": "2026-01-01T00:00:00Z",
		}, []SetExpr{{
			Column: "content",
			Expr:   "messages.content || ?",
			Args:   []any{" world"},
		}})
	}

	proj2, err := NewRelationalProjection("upsert-expr-args2", schema, db, sqlpkg.SQLiteDialect{},
		suffixHandler, nil)
	if err != nil {
		t.Fatalf("new projection2: %v", err)
	}

	if err := proj2.Handle(ctx, newEvent(t, "Append", nil)); err != nil {
		t.Fatalf("handle append: %v", err)
	}

	var content string
	err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id = 'm6'").Scan(&content)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "hello world" {
		t.Fatalf("bound-arg expression should append ' world' to 'hello', got %q", content)
	}
}

func TestSinkUpsertCols_FreshInsert(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	// UpsertCols on a row that does not exist yet: the INSERT path runs and
	// writes ALL provided columns, regardless of updateCols (which only
	// governs the ON CONFLICT DO UPDATE branch).
	freshHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertCols(ctx, "messages", Row{
			"id":         "m7",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "fresh-insert",
			"created_at": "2026-01-01T00:00:00Z",
		}, []string{"content"})
	}

	proj, err := NewRelationalProjection("upsert-cols-fresh", schema, db, sqlpkg.SQLiteDialect{},
		freshHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	var content, author, channel string
	err = db.QueryRowContext(ctx,
		"SELECT content, author_id, channel_id FROM messages WHERE id = 'm7'").Scan(
		&content, &author, &channel,
	)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "fresh-insert" {
		t.Fatalf("fresh insert should write content, got %q", content)
	}

	if author != "u1" {
		t.Fatalf("fresh insert should write author_id even though not in updateCols, got %q", author)
	}

	if channel != "c1" {
		t.Fatalf("fresh insert should write channel_id, got %q", channel)
	}
}

func TestSinkUpsertExpr_FreshInsert(t *testing.T) {
	t.Parallel()

	db, ctx := openRelationalCtx(t)

	schema := discordSchema()

	// UpsertExpr on a row that does not exist yet: the INSERT path runs and
	// writes ALL provided columns; SetExprs are only evaluated on conflict.
	freshHandler := func(_ context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		return sink.UpsertExpr(ctx, "messages", Row{
			"id":         "m8",
			"channel_id": "c1",
			"guild_id":   "g1",
			"author_id":  "u1",
			"content":    "fresh-expr-insert",
			"created_at": "2026-01-01T00:00:00Z",
		}, []SetExpr{{
			Column: "content",
			Expr:   "messages.content || ?",
			Args:   []any{" should-not-apply"},
		}})
	}

	proj, err := NewRelationalProjection("upsert-expr-fresh", schema, db, sqlpkg.SQLiteDialect{},
		freshHandler, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "Create", nil)); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	var content string
	err = db.QueryRowContext(ctx, "SELECT content FROM messages WHERE id = 'm8'").Scan(&content)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "fresh-expr-insert" {
		t.Fatalf("fresh insert should write provided content, not apply SetExpr, got %q", content)
	}
}
