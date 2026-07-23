package turso_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/turso/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

type tursoVMKey string

func (k tursoVMKey) String() string { return string(k) }

type tursoVMView struct {
	Name       string `view:"name"`
	Email      string `view:"email"`
	Tombstoned bool   `view:"tombstoned"`
}

func (v *tursoVMView) IsTombstoned() bool { return v.Tombstoned }

func buildVMMessage(evt event.Event, eventType string) *message.Message {
	msg := message.NewMessage(evt.ID().String(), evt.Payload())
	msg.Metadata.Set("event_type", eventType)
	msg.Metadata.Set("event_id", evt.ID().String())
	msg.Metadata.Set("aggregate_id", evt.StreamID().String())
	msg.Metadata.Set("aggregate_type", string(evt.StreamType()))
	msg.Metadata.Set("version", "1")
	msg.Metadata.Set("schema_version", "1")

	return msg
}

// TestIntegration_TursoSQLViewStoreWithMaterialize proves the full end-to-end
// path: Turso bundle → SQLViewModel → Materialize.HandlerFunc → SQL table →
// View/List.
func TestIntegration_TursoSQLViewStoreWithMaterialize(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dsn := filepath.Join(dir, "views.db")

	b, err := turso.New(dsn)
	if err != nil {
		t.Fatalf("turso.New: %v", err)
	}

	defer func() { _ = b.Close() }()

	mapper := storage.AutoMapperWithTombstone[tursoVMView]("turso_users", "tombstoned")
	store, err := turso.SQLViewModel[tursoVMView, tursoVMKey](b.Bundle, mapper)
	if err != nil {
		t.Fatalf("SQLViewModel: %v", err)
	}

	mat := stack.Materialize[tursoVMView, tursoVMKey]{
		Store: store,
		KeyFromEvent: func(evt event.Event) (tursoVMKey, error) {
			return tursoVMKey(evt.StreamID().String()), nil
		},
		OnCreate: func(_ context.Context, _ event.Event) (*tursoVMView, error) {
			return &tursoVMView{Name: "Alice", Email: "alice@example.com"}, nil
		},
		OnUpdate: func(
			_ context.Context,
			_ event.Event,
			existing *tursoVMView,
		) (*tursoVMView, error) {
			return &tursoVMView{
				Name:  existing.Name + " Updated",
				Email: existing.Email,
			}, nil
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
	if err := handler(buildVMMessage(evt, "user.created")); err != nil {
		t.Fatalf("Handler create: %v", err)
	}

	// Verify via View (SQL Get).
	got, err := mat.View(ctx, tursoVMKey(aggID.String()))
	if err != nil {
		t.Fatalf("View after create: %v", err)
	}

	if got.Name != "Alice" || got.Email != "alice@example.com" {
		t.Fatalf("View: got %+v, want Alice", got)
	}

	// Update event → OnUpdate → upsert in SQL table.
	evt2, _ := event.NewEvent(event.Type("user.updated"), aggID, "User", event.Version(2), nil)
	if err := handler(buildVMMessage(evt2, "user.updated")); err != nil {
		t.Fatalf("Handler update: %v", err)
	}

	got, err = mat.View(ctx, tursoVMKey(aggID.String()))
	if err != nil {
		t.Fatalf("View after update: %v", err)
	}

	if got.Name != "Alice Updated" {
		t.Fatalf("View after update: name=%s, want 'Alice Updated'", got.Name)
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
