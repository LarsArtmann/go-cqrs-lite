package sqlite_test

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

type sqlUserKey string

func (k sqlUserKey) String() string { return string(k) }

type sqlUserView struct {
	Name       string `view:"name"`
	Email      string `view:"email"`
	Tombstoned bool   `view:"tombstoned"`
}

func (v *sqlUserView) IsTombstoned() bool { return v.Tombstoned }

func buildViewMessage(evt event.Event, eventType string) *message.Message {
	msg := message.NewMessage(evt.ID().String(), evt.Payload())
	msg.Metadata.Set("event_type", eventType)
	msg.Metadata.Set("event_id", evt.ID().String())
	msg.Metadata.Set("aggregate_id", evt.StreamID().String())
	msg.Metadata.Set("aggregate_type", string(evt.StreamType()))
	msg.Metadata.Set("version", "1")
	msg.Metadata.Set("schema_version", "1")

	md := evt.Metadata()
	if md.Tombstone != nil {
		msg.Metadata.Set("tombstone_status", strconv.Itoa(int(md.Tombstone.Status)))
		if md.Tombstone.Reason != "" {
			msg.Metadata.Set("tombstone_reason", md.Tombstone.Reason)
		}
	}

	return msg
}

// TestIntegration_SQLViewStoreWithMaterialize proves the full end-to-end path:
// SQLite bundle → SQLViewModel → Materialize.HandlerFunc → SQL table → View/List.
func TestIntegration_SQLViewStoreWithMaterialize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "views.db")

	b, err := sqlite.New(dsn)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}

	defer func() { _ = b.Close() }()

	mapper := storage.AutoMapperWithTombstone[sqlUserView]("sql_users", "tombstoned")
	store, err := sqlite.SQLViewModel[sqlUserView, sqlUserKey](b, mapper)
	if err != nil {
		t.Fatalf("SQLViewModel: %v", err)
	}

	mat := stack.Materialize[sqlUserView, sqlUserKey]{
		Store: store,
		KeyFromEvent: func(evt event.Event) (sqlUserKey, error) {
			return sqlUserKey(evt.StreamID().String()), nil
		},
		OnCreate: func(_ context.Context, _ event.Event) (*sqlUserView, error) {
			return &sqlUserView{Name: "Alice", Email: "alice@example.com"}, nil
		},
		OnUpdate: func(_ context.Context, evt event.Event, existing *sqlUserView) (*sqlUserView, error) {
			return &sqlUserView{Name: existing.Name + " Updated", Email: existing.Email}, nil
		},
	}

	ctx := context.Background()

	// Create event → OnCreate → Set in SQL table.
	aggID := id.NewStreamID()
	evt, err := event.NewEvent(event.Type("user.created"), aggID, "User", event.Version(1), nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	handler := mat.HandlerFunc()
	if err := handler(buildViewMessage(evt, "user.created")); err != nil {
		t.Fatalf("Handler create: %v", err)
	}

	// Verify via View (SQL Get).
	got, err := mat.View(ctx, sqlUserKey(aggID.String()))
	if err != nil {
		t.Fatalf("View after create: %v", err)
	}

	if got.Name != "Alice" || got.Email != "alice@example.com" {
		t.Fatalf("View: got %+v, want Alice", got)
	}

	// Update event → OnUpdate → upsert in SQL table.
	evt2, _ := event.NewEvent(event.Type("user.updated"), aggID, "User", event.Version(2), nil)
	if err := handler(buildViewMessage(evt2, "user.updated")); err != nil {
		t.Fatalf("Handler update: %v", err)
	}

	got, err = mat.View(ctx, sqlUserKey(aggID.String()))
	if err != nil {
		t.Fatalf("View after update: %v", err)
	}

	if got.Name != "Alice Updated" {
		t.Fatalf("View after update: name=%s, want 'Alice Updated'", got.Name)
	}

	// List with ExcludeTombstoned → should use TombstoneQuerier fast path.
	results, err := mat.List(ctx, stack.ExcludeTombstoned)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(results) != 1 || results[0].Name != "Alice Updated" {
		t.Fatalf("List: got %d results, first=%s; want 1, Alice Updated",
			len(results), safeSqlName(results))
	}

	// Tombstone the user.
	tombEvt, _ := event.NewEvent(event.Type("user.deleted"), aggID, "User", event.Version(3), nil)
	tombEvt, _ = event.MarkTombstone(tombEvt)

	mat.OnTombstone = func(_ context.Context, _ event.Event, existing *sqlUserView) (*sqlUserView, error) {
		return &sqlUserView{Name: existing.Name, Email: existing.Email, Tombstoned: true}, nil
	}

	if err := handler(buildViewMessage(tombEvt, "user.deleted")); err != nil {
		t.Fatalf("Handler tombstone: %v", err)
	}

	// List with ExcludeTombstoned → 0 results (server-side filtered).
	results, err = mat.List(ctx, stack.ExcludeTombstoned)
	if err != nil {
		t.Fatalf("List after tombstone: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("List after tombstone: got %d, want 0", len(results))
	}

	// List with OnlyTombstoned → 1 result.
	results, err = mat.List(ctx, stack.OnlyTombstoned)
	if err != nil {
		t.Fatalf("List only tombstoned: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("List only tombstoned: got %d, want 1", len(results))
	}

	// Query via SQLViewStore directly (proves real columns work).
	queried, err := store.Query(ctx, kv.ViewQuery{
		Conditions: []kv.Condition{{Column: "name", Op: kv.OpLike, Value: "Alice%"}},
		OrderBy:    "name",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(queried) != 1 {
		t.Fatalf("Query: got %d, want 1", len(queried))
	}

	// Count.
	count, err := store.Count(ctx, kv.ViewQuery{})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}

	if count != 1 {
		t.Fatalf("Count: got %d, want 1", count)
	}
}

func safeSqlName(views []*sqlUserView) string {
	if len(views) == 0 {
		return "(empty)"
	}

	return views[0].Name
}
