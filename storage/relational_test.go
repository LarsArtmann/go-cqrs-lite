package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// discordSchema mirrors DiscordSync's real relational read model: multiple
// related tables, foreign-key-like references, a junction table (member_roles),
// and an append-only history table (message_edits). This is exactly the shape
// SQLViewStore/Materialize CANNOT express — proving the new capability.
func discordSchema() RelationalSchema {
	return RelationalSchema{Tables: []RelationalTable{
		{
			Name: "guilds", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "name", Type: "TEXT"},
				{Name: "created_at", Type: "TEXT"},
				{Name: "updated_at", Type: "TEXT"},
			},
		},
		{
			Name: "channels", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "guild_id", Type: "TEXT", Nullable: true},
				{Name: "name", Type: "TEXT"},
				{Name: "created_at", Type: "TEXT"},
			},
		},
		{
			Name: "users", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "username", Type: "TEXT"},
				{Name: "created_at", Type: "TEXT"},
			},
		},
		{
			Name: "messages", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "channel_id", Type: "TEXT"},
				{Name: "guild_id", Type: "TEXT", Nullable: true},
				{Name: "author_id", Type: "TEXT"},
				{Name: "content", Type: "TEXT"},
				{Name: "created_at", Type: "TEXT"},
				{Name: "edited_at", Type: "TEXT", Nullable: true},
				{Name: "deleted_at", Type: "TEXT", Nullable: true},
			},
		},
		{
			Name: "attachments", PrimaryKey: []string{"id"},
			Columns: []RelationalColumn{
				{Name: "id", Type: "TEXT"},
				{Name: "message_id", Type: "TEXT"},
				{Name: "filename", Type: "TEXT"},
				{Name: "url", Type: "TEXT"},
			},
		},
		{
			// Append-only history table: autoincrement PK, no declared PrimaryKey.
			Name: "message_edits",
			Columns: []RelationalColumn{
				{Name: "id", Type: "INTEGER PRIMARY KEY AUTOINCREMENT", Nullable: true},
				{Name: "message_id", Type: "TEXT"},
				{Name: "before_content", Type: "TEXT"},
				{Name: "after_content", Type: "TEXT"},
				{Name: "edited_at", Type: "TEXT"},
			},
		},
		{
			// Junction table: composite primary key (many-to-many).
			Name:       "member_roles",
			PrimaryKey: []string{"guild_id", "user_id", "role_id"},
			Columns: []RelationalColumn{
				{Name: "guild_id", Type: "TEXT"},
				{Name: "user_id", Type: "TEXT"},
				{Name: "role_id", Type: "TEXT"},
			},
		},
	}}
}

type messageCreatedPayload struct {
	ID          string         `json:"id"`
	ChannelID   string         `json:"channel_id"`
	GuildID     string         `json:"guild_id"`
	AuthorID    string         `json:"author_id"`
	Content     string         `json:"content"`
	CreatedAt   time.Time      `json:"created_at"`
	Attachments []attachmentPL `json:"attachments"`
}

type attachmentPL struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// handleMessagesCreated writes across 4+ tables atomically through the sink —
// the exact pattern DiscordSync hand-rolls with raw *sql.Tx today.
func handleMessagesCreated(ctx context.Context, evt cqrsevent.Event, sink ProjectionSink) error {
	var p messageCreatedPayload
	if err := json.Unmarshal(evt.Payload(), &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	if p.GuildID != "" {
		if err := sink.Ensure(ctx, "guilds", Row{
			"id": p.GuildID, "name": "", "created_at": p.CreatedAt, "updated_at": p.CreatedAt,
		}); err != nil {
			return err
		}
	}

	if err := sink.Ensure(ctx, "channels", Row{
		"id": p.ChannelID, "guild_id": p.GuildID, "name": "", "created_at": p.CreatedAt,
	}); err != nil {
		return err
	}

	if err := sink.Ensure(ctx, "users", Row{
		"id": p.AuthorID, "username": "", "created_at": p.CreatedAt,
	}); err != nil {
		return err
	}

	if err := sink.Upsert(ctx, "messages", Row{
		"id": p.ID, "channel_id": p.ChannelID, "guild_id": p.GuildID,
		"author_id": p.AuthorID, "content": p.Content, "created_at": p.CreatedAt,
	}); err != nil {
		return err
	}

	for _, att := range p.Attachments {
		if err := sink.Ensure(ctx, "attachments", Row{
			"id": att.ID, "message_id": p.ID, "filename": att.Filename, "url": att.URL,
		}); err != nil {
			return err
		}
	}

	return nil
}

func openRelationalDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?_loc=auto&_time_format=sqlite")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	db.SetMaxOpenConns(1)

	t.Cleanup(func() { _ = db.Close() })

	return db
}

func newEvent(t *testing.T, eventType string, payload any) cqrsevent.Event {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	aggID, err := id.ParseAggregateID("msg-1")
	if err != nil {
		t.Fatalf("parse agg id: %v", err)
	}

	evt, err := cqrsevent.NewEvent(
		cqrsevent.Type(eventType),
		aggID,
		"Message",
		cqrsevent.Version(1),
		raw,
	)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}

	return evt
}

func TestRelationalProjection_MultiTableAtomicWrite(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	proj, err := NewRelationalProjection(
		"discord-messages", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated,
		[]cqrsevent.Type{"MESSAGE_CREATED"},
	)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	now := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)

	evt := newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
		ID:        "m1",
		ChannelID: "c1",
		GuildID:   "g1",
		AuthorID:  "u1",
		Content:   "hello",
		CreatedAt: now,
		Attachments: []attachmentPL{
			{ID: "a1", Filename: "f.png", URL: "http://x/f.png"},
			{ID: "a2", Filename: "g.png", URL: "http://x/g.png"},
		},
	})

	if err := proj.Handle(ctx, evt); err != nil {
		t.Fatalf("handle: %v", err)
	}

	// All 5 tables written in one atomic Handle call.
	assertCount(t, db, "guilds", 1)
	assertCount(t, db, "channels", 1)
	assertCount(t, db, "users", 1)
	assertCount(t, db, "messages", 1)
	assertCount(t, db, "attachments", 2)

	// Verify the message content landed.
	var content string
	if err := db.QueryRowContext(ctx, `SELECT content FROM messages WHERE id = ?`, "m1").
		Scan(&content); err != nil {
		t.Fatalf("select message: %v", err)
	}

	if content != "hello" {
		t.Fatalf("content = %q, want %q", content, "hello")
	}

	// Name/EventTypes conform to event.Projection.
	if proj.Name() != "discord-messages" {
		t.Fatalf("name = %q", proj.Name())
	}

	if len(proj.EventTypes()) != 1 || string(proj.EventTypes()[0]) != "MESSAGE_CREATED" {
		t.Fatalf("types = %v", proj.EventTypes())
	}
}

func TestRelationalProjection_AtomicRollbackOnError(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	wantErr := errors.New("boom mid-handler")

	handler := func(ctx context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.Ensure(
			ctx,
			"guilds",
			Row{"id": "g1", "name": "x", "created_at": "t", "updated_at": "t"},
		); err != nil {
			return err
		}

		if err := sink.Ensure(
			ctx,
			"users",
			Row{"id": "u1", "username": "x", "created_at": "t"},
		); err != nil {
			return err
		}

		return wantErr // fail AFTER writes — both must roll back.
	}

	proj, err := NewRelationalProjection(
		"rb",
		discordSchema(),
		db,
		sqlpkg.SQLiteDialect{},
		handler,
		nil,
	)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	err = proj.Handle(ctx, newEvent(t, "X", nil))
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}

	// No partial state: both writes rolled back.
	assertCount(t, db, "guilds", 0)
	assertCount(t, db, "users", 0)
}

func TestRelationalProjection_IdempotentReplay(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	proj, err := NewRelationalProjection("idem", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	evt := newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
		ID: "m1", ChannelID: "c1", GuildID: "g1", AuthorID: "u1",
		Content: "hello", CreatedAt: time.Now(),
	})

	for range 3 { // replay the same event 3x
		if err := proj.Handle(ctx, evt); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	// Idempotent: still exactly one of each (Upsert + Ensure).
	assertCount(t, db, "messages", 1)
	assertCount(t, db, "guilds", 1)
}

func TestRelationalProjection_JunctionTable(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	handler := func(ctx context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		// Composite-PK junction row; Ensure defaults conflict to the 3-col PK.
		return sink.Ensure(ctx, "member_roles", Row{
			"guild_id": "g1", "user_id": "u1", "role_id": "r1",
		})
	}

	proj, err := NewRelationalProjection(
		"roles",
		discordSchema(),
		db,
		sqlpkg.SQLiteDialect{},
		handler,
		nil,
	)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	for range 2 { // idempotent across replays
		if err := proj.Handle(ctx, newEvent(t, "R", nil)); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}

	assertCount(t, db, "member_roles", 1)
}

func TestRelationalProjection_ReadThenWriteHistory(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	// Seed a message.
	seed, err := NewRelationalProjection("seed", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new seed: %v", err)
	}

	if err := seed.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
		ID: "m1", ChannelID: "c1", AuthorID: "u1", Content: "before", CreatedAt: time.Now(),
	})); err != nil {
		t.Fatalf("seed handle: %v", err)
	}

	// Update handler: read current content via QueryOne, append edit history,
	// then update the message — all in one atomic tx.
	update := func(ctx context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		old, err := sink.QueryOne(ctx, "messages", "content", Row{"id": "m1"})
		if err != nil {
			return fmt.Errorf("query old content: %w", err)
		}

		oldStr, _ := old.(string)

		if err := sink.Ensure(ctx, "message_edits", Row{
			"message_id":     "m1",
			"before_content": oldStr,
			"after_content":  "after",
			"edited_at":      "t",
		}); err != nil {
			return err
		}

		return sink.Update(ctx, "messages", Row{"content": "after"}, Row{"id": "m1"})
	}

	proj, err := NewRelationalProjection(
		"edit",
		discordSchema(),
		db,
		sqlpkg.SQLiteDialect{},
		update,
		nil,
	)
	if err != nil {
		t.Fatalf("new edit proj: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "MESSAGE_UPDATED", nil)); err != nil {
		t.Fatalf("edit handle: %v", err)
	}

	// History row captured the "before" value; message now holds "after".
	assertCount(t, db, "message_edits", 1)

	var newContent string
	if err := db.QueryRowContext(ctx, `SELECT content FROM messages WHERE id = ?`, "m1").
		Scan(&newContent); err != nil {
		t.Fatalf("select: %v", err)
	}

	if newContent != "after" {
		t.Fatalf("content = %q, want after", newContent)
	}
}

func TestRelationalStore_CountAndCountMany(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	proj, err := NewRelationalProjection("s", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	_ = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
		ID:        "m1",
		ChannelID: "c1",
		GuildID:   "g1",
		AuthorID:  "u1",
		Content:   "a",
		CreatedAt: time.Now(),
	}))
	_ = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
		ID:        "m2",
		ChannelID: "c1",
		GuildID:   "g1",
		AuthorID:  "u2",
		Content:   "b",
		CreatedAt: time.Now(),
	}))

	store, err := NewRelationalStore(discordSchema(), db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	got, err := store.CountMany(ctx, []string{"messages", "users", "channels"})
	if err != nil {
		t.Fatalf("count many: %v", err)
	}

	if got["messages"] != 2 || got["users"] != 2 || got["channels"] != 1 {
		t.Fatalf("counts = %v", got)
	}

	// Conditional count: messages in channel c1.
	chCount, err := store.Count(ctx, "messages", []kv.Condition{
		{Column: "channel_id", Op: kv.OpEq, Value: "c1"},
	})
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if chCount != 2 {
		t.Fatalf("channel count = %d, want 2", chCount)
	}
}

func TestRelationalStore_CursorPagination(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	proj, err := NewRelationalProjection("p", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	base := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)

	for i := range 5 {
		_ = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
			ID: fmt.Sprintf("m%d", i), ChannelID: "c1", AuthorID: "u1",
			Content: fmt.Sprintf("msg%d", i), CreatedAt: base.Add(time.Duration(i) * time.Minute),
		}))
	}

	store, err := NewRelationalStore(discordSchema(), db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	type row struct{ id, content string }

	collect := func(q RelationalQuery) []row {
		var rows []row

		scan := func(scan func(dest ...any) error) error {
			var r row
			if err := scan(&r.id, &r.content); err != nil {
				return err
			}

			rows = append(rows, r)

			return nil
		}

		_ = store.Query(ctx, "messages", []string{"id", "content"}, q, scan)

		return rows
	}

	// Page 1: newest 2 messages.
	page1 := collect(RelationalQuery{
		OrderBy: "created_at", Desc: true, Limit: 2,
	})

	if len(page1) != 2 || page1[0].id != "m4" {
		t.Fatalf("page1 = %+v", page1)
	}

	// Page 2: created_at < page1 cursor, next 2.
	cursor := base.Add(4 * time.Minute)
	page2 := collect(RelationalQuery{
		Conditions: []kv.Condition{{Column: "created_at", Op: kv.OpLt, Value: cursor}},
		OrderBy:    "created_at", Desc: true, Limit: 2,
	})

	if len(page2) != 2 || page2[0].id != "m3" {
		t.Fatalf("page2 = %+v", page2)
	}
}

func TestRelationalSchema_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schema  RelationalSchema
		wantErr error
	}{
		{"no tables", RelationalSchema{}, errSchemaNoTables},
		{"duplicate table", RelationalSchema{Tables: []RelationalTable{
			{Name: "t", Columns: []RelationalColumn{{Name: "id", Type: "TEXT"}}},
			{Name: "t", Columns: []RelationalColumn{{Name: "id", Type: "TEXT"}}},
		}}, errSchemaDuplicateTable},
		{"pk not in columns", RelationalSchema{Tables: []RelationalTable{
			{
				Name: "t", PrimaryKey: []string{"missing"},
				Columns: []RelationalColumn{{Name: "id", Type: "TEXT"}},
			},
		}}, errSchemaUnknownPKColumn},
		{"column no type", RelationalSchema{Tables: []RelationalTable{
			{Name: "t", Columns: []RelationalColumn{{Name: "id"}}},
		}}, errSchemaColumnNoType},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.schema.Validate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRelationalSchema_DDLIsDialectPortable(t *testing.T) {
	t.Parallel()

	ddl := RelationalTable{
		Name:       "member_roles",
		PrimaryKey: []string{"guild_id", "user_id"},
		Columns: []RelationalColumn{
			{Name: "guild_id", Type: "TEXT"},
			{Name: "user_id", Type: "TEXT", Nullable: true},
		},
	}.DDL()

	// Both SQLite and PostgreSQL accept this exact statement.
	for _, mustContain := range []string{
		"CREATE TABLE IF NOT EXISTS member_roles",
		"guild_id TEXT NOT NULL",
		"user_id TEXT",
		"PRIMARY KEY (guild_id, user_id)",
	} {
		if !strings.Contains(ddl, mustContain) {
			t.Fatalf("DDL missing %q:\n%s", mustContain, ddl)
		}
	}
}

func TestNewRelationalProjection_RejectsBadInputs(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	sch := discordSchema()
	handler := func(context.Context, cqrsevent.Event, ProjectionSink) error { return nil }

	if _, err := NewRelationalProjection(
		"",
		sch,
		db,
		sqlpkg.SQLiteDialect{},
		handler,
		nil,
	); !errors.Is(
		err,
		errRelationalNoName,
	) {
		t.Fatalf("empty name err = %v", err)
	}

	if _, err := NewRelationalProjection(
		"x",
		sch,
		nil,
		sqlpkg.SQLiteDialect{},
		handler,
		nil,
	); !errors.Is(
		err,
		errRelationalNilDB,
	) {
		t.Fatalf("nil db err = %v", err)
	}

	if _, err := NewRelationalProjection(
		"x",
		sch,
		db,
		nil,
		handler,
		nil,
	); !errors.Is(
		err,
		errRelationalNilDialect,
	) {
		t.Fatalf("nil dialect err = %v", err)
	}

	if _, err := NewRelationalProjection(
		"x",
		sch,
		db,
		sqlpkg.SQLiteDialect{},
		nil,
		nil,
	); !errors.Is(
		err,
		errRelationalNilHandler,
	) {
		t.Fatalf("nil handler err = %v", err)
	}
}

func TestRelationalProjection_DeleteWhereAndQueryOneMissing(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	// Seed two messages.
	seed, err := NewRelationalProjection("seed", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new seed: %v", err)
	}

	for _, id := range []string{"keep", "drop"} {
		_ = seed.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
			ID: id, ChannelID: "c1", AuthorID: "u1", Content: id, CreatedAt: time.Now(),
		}))
	}

	// DeleteWhere removes the "drop" message; QueryOne on a missing row errors.
	del := func(ctx context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
		if err := sink.DeleteWhere(ctx, "messages", Row{"id": "drop"}); err != nil {
			return err
		}

		_, qerr := sink.QueryOne(ctx, "messages", "content", Row{"id": "drop"})
		if !errors.Is(qerr, errSinkNoRows) {
			return fmt.Errorf("expected errSinkNoRows after delete, got %w", qerr)
		}

		return nil
	}

	proj, err := NewRelationalProjection(
		"del",
		discordSchema(),
		db,
		sqlpkg.SQLiteDialect{},
		del,
		nil,
	)
	if err != nil {
		t.Fatalf("new del proj: %v", err)
	}

	if err := proj.Handle(ctx, newEvent(t, "DELETE", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	assertCount(t, db, "messages", 1) // only "keep" remains
}

func TestRelationalProjection_WithoutAutoMigrate(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	schema := discordSchema()

	// Pre-create the schema externally, then construct with auto-migrate off.
	if err := schema.Migrate(context.Background(), db); err != nil {
		t.Fatalf("external migrate: %v", err)
	}

	proj, err := NewRelationalProjection("nm", schema, db, sqlpkg.SQLiteDialect{},
		func(ctx context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
			return sink.Upsert(ctx, "messages", Row{
				"id": "m1", "channel_id": "c1", "author_id": "u1",
				"content": "x", "created_at": "t",
			})
		}, nil, WithoutRelationalAutoMigrate())
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	if err := proj.Handle(context.Background(), newEvent(t, "X", nil)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	assertCount(t, db, "messages", 1)
}

func TestRelationalStore_DefaultOrderOnNoPKTable(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	// message_edits has no declared PrimaryKey → default OrderBy is "rowid".
	proj, err := NewRelationalProjection("ed", discordSchema(), db, sqlpkg.SQLiteDialect{},
		func(ctx context.Context, _ cqrsevent.Event, sink ProjectionSink) error {
			return sink.Ensure(ctx, "message_edits", Row{
				"message_id": "m1", "before_content": "a", "after_content": "b", "edited_at": "t",
			})
		}, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	_ = proj.Handle(ctx, newEvent(t, "E", nil))

	store, err := NewRelationalStore(discordSchema(), db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// Query with no OrderBy → falls back to rowid (no SQL error on either backend).
	var after string

	if err := store.Query(ctx, "message_edits", []string{"after_content"}, RelationalQuery{},
		func(scan func(dest ...any) error) error { return scan(&after) }); err != nil {
		t.Fatalf("query message_edits with default order: %v", err)
	}

	if after != "b" {
		t.Fatalf("after_content = %q, want %q", after, "b")
	}
}

// assertCount queries the row count of table directly (test-only helper).
func assertCount(t *testing.T, db *sql.DB, table string, want int64) {
	t.Helper()

	var got int64

	if err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM "+table).
		Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}

	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}
