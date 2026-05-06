package storage

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
)

func TestTursoConnector_OpenLocalDB(t *testing.T) {
	db, err := OpenTurso(":memory:")
	if err != nil {
		t.Fatalf("OpenTurso: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestTursoConnector_EventStore(t *testing.T) {
	db, err := OpenTurso(":memory:")
	if err != nil {
		t.Fatalf("OpenTurso: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(SQLiteSchema()); err != nil {
		t.Fatalf("exec schema: %v", err)
	}

	store, err := NewSQLiteEventStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteEventStore: %v", err)
	}

	aggType := event.AggregateType("test_agg")
	aggID := id.NewAggregateID()

	evt, err := event.NewEvent(
		"test.created",
		aggID,
		aggType,
		1,
		[]byte(`{"name":"hello"}`),
	)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.Save(context.Background(), aggType, aggID, []event.Event{evt}, 0); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), aggType, aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 event, got %d", len(loaded))
	}

	if loaded[0].Type() != evt.Type() {
		t.Fatalf("expected type %s, got %s", evt.Type(), loaded[0].Type())
	}
}

func TestTursoConnector_SnapshotStore(t *testing.T) {
	db, err := OpenTurso(":memory:")
	if err != nil {
		t.Fatalf("OpenTurso: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(SQLiteSnapshotSchema()); err != nil {
		t.Fatalf("exec snapshot schema: %v", err)
	}

	store, err := NewSQLiteSnapshotStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteSnapshotStore: %v", err)
	}

	aggType := event.AggregateType("test_agg")
	aggID := id.NewAggregateID()
	state := []byte(`{"count":42}`)

	snap := event.Snapshot{
		AggregateID:   aggID,
		AggregateType: aggType,
		Version:       event.Version(5),
		State:         state,
		CreatedAt:     time.Now().Truncate(time.Microsecond),
	}

	if err := store.Save(context.Background(), snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), aggType, aggID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if string(loaded.State) != string(state) {
		t.Fatalf("expected %s, got %s", state, loaded.State)
	}
}

func TestTursoConnector_CheckpointStore(t *testing.T) {
	db, err := OpenTurso(":memory:")
	if err != nil {
		t.Fatalf("OpenTurso: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(SQLiteCheckpointSchema()); err != nil {
		t.Fatalf("exec checkpoint schema: %v", err)
	}

	store, err := NewSQLiteCheckpointStore(db)
	if err != nil {
		t.Fatalf("NewSQLiteCheckpointStore: %v", err)
	}

	projection := "test_projection"
	checkpoint := id.NewEventID()

	if err := store.Save(context.Background(), projection, checkpoint); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(context.Background(), projection)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded != checkpoint {
		t.Fatalf("expected %s, got %s", checkpoint, loaded)
	}
}
