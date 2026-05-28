package stream_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/stream"
)

func TestAggregateProjection_InvalidPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		prefix string
	}{
		{"empty", ""},
		{"uppercase", "MyApp"},
		{"spaces", "my app"},
		{"special chars", "my-app"},
		{"SQL injection", "myapp; DROP TABLE users;--"},
		{"numbers first", "123app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := stream.NewAggregateProjection(nil, tt.prefix)
			if err == nil {
				t.Errorf("expected error for prefix %q", tt.prefix)
			}
		})
	}
}

func TestAggregateProjection_ValidPrefix(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	_, err := stream.NewAggregateProjection(db, "myapp_")
	if err != nil {
		t.Fatalf("valid prefix should succeed: %v", err)
	}
}

func TestAggregateProjection_HandleNormalEvent(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	proj, err := stream.NewAggregateProjection(db, "test_")
	if err != nil {
		t.Fatal(err)
	}

	aggID := id.NewAggregateID()
	evt := newProjEvent(t, "user.created", "User", aggID, 1, nil)

	if err := proj.Handle(context.Background(), evt); err != nil {
		t.Fatalf("handle: %v", err)
	}

	reader, err := stream.NewSQLAggregateReader(db, "test_")
	if err != nil {
		t.Fatal(err)
	}

	page, err := reader.List(context.Background(), stream.ListOptions{
		Type: "User",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}

	if page.Items[0].Version != 1 {
		t.Errorf("version = %d, want 1", page.Items[0].Version)
	}

	if page.Items[0].EventCount != 1 {
		t.Errorf("event_count = %d, want 1", page.Items[0].EventCount)
	}
}

func TestAggregateProjection_HandleTombstone(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	proj, err := stream.NewAggregateProjection(db, "test_")
	if err != nil {
		t.Fatal(err)
	}

	aggID := id.NewAggregateID()

	evt1 := newProjEvent(t, "user.created", "User", aggID, 1, nil)
	if err := proj.Handle(context.Background(), evt1); err != nil {
		t.Fatalf("handle create: %v", err)
	}

	evt2 := newProjEvent(t, "user.deleted", "User", aggID, 2,
		map[string]string{string(event.MetadataKeyTombstone): "true"})
	if err := proj.Handle(context.Background(), evt2); err != nil {
		t.Fatalf("handle delete: %v", err)
	}

	reader, err := stream.NewSQLAggregateReader(db, "test_")
	if err != nil {
		t.Fatal(err)
	}

	page, err := reader.ListWithStatus(context.Background(), stream.ListOptions{
		Type:      "User",
		Tombstone: stream.TombstoneInclude,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}

	if !page.Items[0].Status.IsTombstoned() {
		t.Error("expected tombstoned status")
	}

	if page.Items[0].Ref.Version != 2 {
		t.Errorf("version = %d, want 2", page.Items[0].Ref.Version)
	}

	if page.Items[0].Ref.EventCount != 2 {
		t.Errorf("event_count = %d, want 2", page.Items[0].Ref.EventCount)
	}
}

func TestAggregateProjection_HandleRebirth(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	proj, err := stream.NewAggregateProjection(db, "test_")
	if err != nil {
		t.Fatal(err)
	}

	aggID := id.NewAggregateID()

	evt1 := newProjEvent(t, "user.created", "User", aggID, 1, nil)
	_ = proj.Handle(context.Background(), evt1)

	evt2 := newProjEvent(t, "user.deleted", "User", aggID, 2,
		map[string]string{string(event.MetadataKeyTombstone): "true"})
	_ = proj.Handle(context.Background(), evt2)

	evt3 := newProjEvent(t, "user.reactivated", "User", aggID, 3,
		map[string]string{string(event.MetadataKeyRebirth): "true"})
	_ = proj.Handle(context.Background(), evt3)

	reader, _ := stream.NewSQLAggregateReader(db, "test_")

	page, _ := reader.ListWithStatus(context.Background(), stream.ListOptions{Type: "User"})

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}

	if page.Items[0].Status.IsTombstoned() {
		t.Error("expected active status after rebirth")
	}

	if !page.Items[0].Status.IsActive() {
		t.Error("expected active status after rebirth")
	}
}

func TestAggregateProjection_NormalEventPreservesTombstone(t *testing.T) {
	t.Parallel()

	db := newSQLiteDB(t)
	defer db.Close()

	proj, err := stream.NewAggregateProjection(db, "test_")
	if err != nil {
		t.Fatal(err)
	}

	aggID := id.NewAggregateID()

	evt1 := newProjEvent(t, "user.created", "User", aggID, 1, nil)
	_ = proj.Handle(context.Background(), evt1)

	evt2 := newProjEvent(t, "user.deleted", "User", aggID, 2,
		map[string]string{string(event.MetadataKeyTombstone): "true"})
	_ = proj.Handle(context.Background(), evt2)

	evt3 := newProjEvent(t, "user.updated", "User", aggID, 3, nil)
	_ = proj.Handle(context.Background(), evt3)

	reader, _ := stream.NewSQLAggregateReader(db, "test_")

	page, _ := reader.ListWithStatus(
		context.Background(),
		stream.ListOptions{Type: "User", Tombstone: stream.TombstoneInclude},
	)

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}

	if !page.Items[0].Status.IsTombstoned() {
		t.Error("normal event after tombstone should preserve tombstoned status")
	}
}

func newProjEvent(
	t *testing.T,
	eventType string,
	aggType event.AggregateType,
	aggID id.AggregateID,
	version int,
	custom map[string]string,
) event.Event {
	t.Helper()

	opts := make([]event.Option, 0, len(custom))
	for k, v := range custom {
		opts = append(opts, event.WithCustom(event.MetadataKey(k), v))
	}

	evt, err := event.NewEvent(
		event.Type(eventType),
		aggID,
		aggType,
		event.Version(version),
		[]byte(`{}`),
		opts...,
	)
	if err != nil {
		t.Fatal(err)
	}

	return evt
}

func newSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}
