//go:build integration

package relational

import (
	"context"
	"database/sql"
	"strings"
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

// PostgreSQL rejects "LIMIT -1"; an Offset without a Limit must return the
// remaining rows instead of failing.
func TestPostgres_RelationalStore_OffsetWithoutLimit(t *testing.T) {
	db := openPostgresDB(t)
	ctx := context.Background()

	schema := pgDiscordSchema()
	if err := schema.Migrate(ctx, db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	store, err := NewRelationalStore(schema, db, sqlpkg.PostgresDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	for _, id := range []string{"g1", "g2", "g3", "g4", "g5"} {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO guilds (id, name, created_at, updated_at) VALUES ($1, $2, now(), now())`,
			id,
			"n"+id,
		); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	var ids []string
	err = store.Query(ctx, "guilds", []string{"id"}, kv.ViewQuery{OrderBy: "id", Offset: 2},
		func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}

			ids = append(ids, id)

			return nil
		})
	if err != nil {
		t.Fatalf("Query offset without limit: %v", err)
	}

	if strings.Join(ids, ",") != "g3,g4,g5" {
		t.Fatalf("offset 2 without limit: got %v, want [g3 g4 g5]", ids)
	}
}
