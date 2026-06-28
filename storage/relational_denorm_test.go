package storage

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// TestRelationalDenormalization_FKQueries demonstrates the denormalization
// pattern that replaces JOINs: projection handlers write FK columns directly
// into the messages table (channel_id, guild_id, author_id). Queries then
// filter on these columns — no JOIN needed.
//
// This is the pattern DiscordSync should use instead of hand-written JOINs.
func TestRelationalDenormalization_FKQueries(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	proj, err := NewRelationalProjection("denorm", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	// Seed: 3 messages across 2 channels, 2 authors, 1 guild.
	msgs := []messageCreatedPayload{
		{
			ID:        "m1",
			ChannelID: "c1",
			GuildID:   "g1",
			AuthorID:  "alice",
			Content:   "hello",
			CreatedAt: time.Now(),
		},
		{
			ID:        "m2",
			ChannelID: "c1",
			GuildID:   "g1",
			AuthorID:  "bob",
			Content:   "hi",
			CreatedAt: time.Now(),
		},
		{
			ID:        "m3",
			ChannelID: "c2",
			GuildID:   "g1",
			AuthorID:  "alice",
			Content:   "other chan",
			CreatedAt: time.Now(),
		},
	}

	for _, m := range msgs {
		_ = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", m))
	}

	store, err := NewRelationalStore(discordSchema(), db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// Query 1: messages in channel c1 (filter on denormalized channel_id).
	var chanMsgs []string

	err = store.Query(
		ctx, "messages", []string{"id"},
		kv.ViewQuery{
			Conditions: []kv.Condition{{Column: "channel_id", Op: kv.OpEq, Value: "c1"}},
			OrderBy:    "id",
		},
		func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}

			chanMsgs = append(chanMsgs, id)

			return nil
		},
	)
	if err != nil {
		t.Fatalf("query by channel_id: %v", err)
	}

	if len(chanMsgs) != 2 {
		t.Fatalf("channel c1: expected 2 messages, got %d: %v", len(chanMsgs), chanMsgs)
	}

	// Query 2: messages by author alice (filter on denormalized author_id).
	var authorMsgs []string

	err = store.Query(
		ctx, "messages", []string{"id"},
		kv.ViewQuery{
			Conditions: []kv.Condition{{Column: "author_id", Op: kv.OpEq, Value: "alice"}},
			OrderBy:    "id",
		},
		func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}

			authorMsgs = append(authorMsgs, id)

			return nil
		},
	)
	if err != nil {
		t.Fatalf("query by author_id: %v", err)
	}

	if len(authorMsgs) != 2 {
		t.Fatalf("author alice: expected 2 messages, got %d: %v", len(authorMsgs), authorMsgs)
	}

	// Query 3: count messages in guild g1 (denormalized guild_id).
	guildCount, err := store.Count(ctx, "messages", []kv.Condition{
		{Column: "guild_id", Op: kv.OpEq, Value: "g1"},
	})
	if err != nil {
		t.Fatalf("count by guild_id: %v", err)
	}

	if guildCount != 3 {
		t.Fatalf("guild g1: expected 3 messages, got %d", guildCount)
	}

	// Query 4: compound condition — alice's messages in c1 only.
	var compoundMsgs []string

	err = store.Query(
		ctx, "messages", []string{"id"},
		kv.ViewQuery{
			Conditions: []kv.Condition{
				{Column: "author_id", Op: kv.OpEq, Value: "alice"},
				{Column: "channel_id", Op: kv.OpEq, Value: "c1"},
			},
		},
		func(scan func(dest ...any) error) error {
			var id string
			if err := scan(&id); err != nil {
				return err
			}

			compoundMsgs = append(compoundMsgs, id)

			return nil
		},
	)
	if err != nil {
		t.Fatalf("compound query: %v", err)
	}

	if len(compoundMsgs) != 1 || compoundMsgs[0] != "m1" {
		t.Fatalf("alice in c1: expected [m1], got %v", compoundMsgs)
	}
}

// TestRelationalDenormalization_AggregateColumns shows that denormalized
// FK columns also work with ORDER BY and pagination — the full SQL toolkit.
func TestRelationalDenormalization_AggregateColumns(t *testing.T) {
	t.Parallel()

	db := openRelationalDB(t)
	ctx := context.Background()

	proj, err := NewRelationalProjection("denorm2", discordSchema(), db, sqlpkg.SQLiteDialect{},
		handleMessagesCreated, nil)
	if err != nil {
		t.Fatalf("new projection: %v", err)
	}

	for i, m := range []messageCreatedPayload{
		{ID: "old", ChannelID: "c1", GuildID: "g1", AuthorID: "a", Content: "first", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{ID: "mid", ChannelID: "c1", GuildID: "g1", AuthorID: "b", Content: "second", CreatedAt: time.Now().Add(-1 * time.Hour)},
		{ID: "new", ChannelID: "c1", GuildID: "g1", AuthorID: "a", Content: "third", CreatedAt: time.Now()},
	} {
		_ = i
		_ = proj.Handle(ctx, newEvent(t, "MESSAGE_CREATED", m))
	}

	store, err := NewRelationalStore(discordSchema(), db, sqlpkg.SQLiteDialect{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	// Paginated query: latest message first, limit 1.
	var latest string

	err = store.Query(
		ctx, "messages", []string{"id"},
		kv.ViewQuery{
			Conditions: []kv.Condition{{Column: "channel_id", Op: kv.OpEq, Value: "c1"}},
			OrderBy:    "created_at",
			Desc:       true,
			Limit:      1,
		},
		func(scan func(dest ...any) error) error {
			return scan(&latest)
		},
	)
	if err != nil {
		t.Fatalf("latest message: %v", err)
	}

	if latest != "new" {
		t.Fatalf("expected 'new', got %q", latest)
	}
}
