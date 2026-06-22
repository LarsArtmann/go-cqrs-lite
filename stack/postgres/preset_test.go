package postgres_test

import (
	"context"
	"os"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v3"
)

// postgresDSN returns the test Postgres DSN from the environment, or skips.
func postgresDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping Postgres integration test")
	}

	return dsn
}

func TestNew_ProducesWorkingBundle(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	if b.EventSink == nil {
		t.Fatal("EventSink not set")
	}

	if b.EventSource == nil {
		t.Fatal("EventSource not set")
	}

	if b.CommandSink == nil {
		t.Fatal("CommandSink not set")
	}

	if b.Publisher == nil {
		t.Fatal("Publisher not set (bus)")
	}

	if b.ReadModels == nil {
		t.Fatal("ReadModels not set")
	}
}

func TestNew_E2E_EventSaveLoadRoundtrip(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	defer func() { _ = b.Close() }()

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Todo", aggID)

	types := []event.Type{"todo.created", "todo.completed"}
	payloads := []any{
		map[string]any{"title": "buy milk"},
		map[string]any{"at": "now"},
	}

	events, err := event.NewEvents(aggID, "Todo", 0, types, payloads)
	if err != nil {
		t.Fatalf("NewEvents: %v", err)
	}

	if err := b.EventSink.Save(ctx, ref, events, 0); err != nil {
		t.Fatalf("EventSink.Save: %v", err)
	}

	loaded, err := b.EventSource.Load(ctx, ref)
	if err != nil {
		t.Fatalf("EventSource.Load: %v", err)
	}

	if len(loaded) != len(events) {
		t.Fatalf("loaded %d events, want %d", len(loaded), len(events))
	}
}

func TestNew_CloseIsIdempotent(t *testing.T) {
	dsn := postgresDSN(t)

	b, err := postgres.New(dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
