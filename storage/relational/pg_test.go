//go:build integration

package relational

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
	"github.com/larsartmann/go-cqrs-lite/testutil/pgtestcontainer/v4"
)

// pgDiscordSchema is the same as discordSchema but with PG-compatible
// autoincrement syntax (SERIAL instead of INTEGER PRIMARY KEY AUTOINCREMENT).
func pgDiscordSchema() RelationalSchema {
	s := discordSchema()

	for i, t := range s.Tables {
		if t.Name == "message_edits" {
			s.Tables[i].Columns[0] = RelationalColumn{
				Name: "id", Type: "SERIAL PRIMARY KEY", Nullable: true,
			}
		}
	}

	return s
}

// TestMain starts a shared Postgres server (external DSN via
// POSTGRES_TEST_DSN, else a testcontainer) and gives every test its own
// database — including under an explicit DSN, so packages sharing one CI
// service container cannot contaminate each other.
func TestMain(m *testing.M) { pgtestcontainer.TestMain(m) }

func openPostgresDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", pgtestcontainer.DSN(t))
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func TestPostgres_RelationalProjection_BasicCRUD(t *testing.T) {
	db := openPostgresDB(t)
	ctx := context.Background()

	// Clean slate.
	_, _ = db.ExecContext(
		ctx,
		`DROP TABLE IF EXISTS messages, channels, users, guilds, attachments, message_edits, member_roles CASCADE`,
	)

	proj, err := NewRelationalProjection("pg-test", pgDiscordSchema(), db, sqlpkg.PostgresDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	_ = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", messageCreatedPayload{
		ID: "m1", ChannelID: "c1", GuildID: "g1", AuthorID: "u1",
		Content: "hello pg", CreatedAt: time.Now(),
	}))

	store, err := NewRelationalStore(pgDiscordSchema(), db, sqlpkg.PostgresDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	var content string

	err = store.Query(
		ctx, "messages", []string{"content"},
		kv.ViewQuery{
			Conditions: []kv.Condition{{Column: "id", Op: kv.OpEq, Value: "m1"}},
		},
		func(scan func(dest ...any) error) error {
			return scan(&content)
		},
	)
	if err != nil {
		t.Fatalf("query: %v", err)
	}

	if content != "hello pg" {
		t.Fatalf("content = %q, want 'hello pg'", content)
	}

	count, err := store.Count(ctx, "messages", nil)
	if err != nil {
		t.Fatalf("count: %v", err)
	}

	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	_ = cqrsevent.Type("")
}
