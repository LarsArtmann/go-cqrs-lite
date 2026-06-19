//go:build integration

package postgres_test

import (
	"context"
	"database/sql"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/stack/postgres/v2"
	"github.com/larsartmann/go-cqrs-lite/storage/v2"
)

// These integration tests verify storage.PostgresBus + PgxListener against
// a real PostgreSQL instance. They exercise:
//   - SELECT pg_notify() (publish side)
//   - LISTEN + WaitForNotification (listener side)
//   - LoadByEventID refetch path
//   - End-to-end cross-bus delivery (the canonical multi-process pattern)
//
// Run with: go test -tags=integration ./...
// Requires: POSTGRES_TEST_DSN env var (set automatically in CI's
// postgres-integration job).

// uniqueChannel returns a valid Postgres identifier unique per test invocation.
// Prevents parallel integration tests from cross-contaminating each other.
func uniqueChannel(t *testing.T) string {
	t.Helper()

	name := strings.ToLower(t.Name())
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, name)

	return "ch_" + name
}

func openTestDB(t *testing.T) (*sql.DB, *storage.SQLBackend) {
	t.Helper()

	dsn := postgresDSN(t)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping pg: %v", err)
	}

	// Reset tables so each test starts clean.
	_, _ = db.ExecContext(ctx,
		`DROP TABLE IF EXISTS events, commands, queries, snapshots, checkpoints, cqrs_kv`)

	backend, err := storage.NewSQLBackend(db)
	if err != nil {
		t.Fatalf("NewSQLBackend: %v", err)
	}

	return db, backend
}

// TestPostgresBus_E2E_RefetchAndDeliver verifies the canonical multi-process
// pattern: bus B publishes, bus A (different listener, same DB) receives via
// LISTEN/NOTIFY, refetches the full event via LoadByEventID, and dispatches
// to its local handler.
func TestPostgresBus_E2E_RefetchAndDeliver(t *testing.T) {
	db, backend := openTestDB(t)
	ctx := context.Background()

	store := backend.EventStore()
	channel := uniqueChannel(t)

	// Bus A: subscriber. Listener A acquires its own dedicated pgx conn.
	listenerA, err := postgres.NewPgxListenerFromDSN(ctx, postgresDSN(t))
	if err != nil {
		t.Fatalf("listener A: %v", err)
	}

	busA, err := storage.NewPostgresBus(db, store, listenerA,
		storage.WithBusChannel(channel))
	if err != nil {
		t.Fatalf("bus A: %v", err)
	}
	t.Cleanup(func() { _ = busA.Close() })

	var got atomic.Value // event.Event
	received := make(chan event.Event, 1)

	err = busA.Subscribe("user.created", func(_ context.Context, e event.Event) error {
		got.Store(e)
		received <- e

		return nil
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// Bus B: publisher on the SAME channel.
	listenerB, err := postgres.NewPgxListenerFromDSN(ctx, postgresDSN(t))
	if err != nil {
		t.Fatalf("listener B: %v", err)
	}

	busB, err := storage.NewPostgresBus(db, store, listenerB,
		storage.WithBusChannel(channel))
	if err != nil {
		t.Fatalf("bus B: %v", err)
	}
	t.Cleanup(func() { _ = busB.Close() })

	// Save the event to the store BEFORE publishing, so the listener's
	// refetch (LoadByEventID) can find it once NOTIFY arrives.
	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("User", aggID)
	evt, err := event.NewEvent("user.created", aggID, "User", event.Version(1),
		[]byte(`{"name":"alice"}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := store.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Publish from bus B. This dispatches locally (no subscribers on B) and
	// sends NOTIFY. Bus A's listener refetches and dispatches to handler A.
	if err := busB.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Wait for cross-bus delivery (deterministic via channel).
	select {
	case receivedEvt := <-received:
		if receivedEvt.ID() != evt.ID() {
			t.Fatalf("received wrong event ID: got %s, want %s",
				receivedEvt.ID(), evt.ID())
		}

		if receivedEvt.Type() != evt.Type() {
			t.Fatalf("received wrong event type: got %s, want %s",
				receivedEvt.Type(), evt.Type())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: handler A was not invoked via LISTEN/NOTIFY within 5s")
	}
}

// TestPostgresBus_BadChannelRejected confirms the listener validates channel
// names against the real Postgres LISTEN syntax (a regression check on
// validateChannelName's SQL-injection defence).
func TestPostgresBus_BadChannelRejected(t *testing.T) {
	db, backend := openTestDB(t)

	listener, err := postgres.NewPgxListenerFromDSN(context.Background(), postgresDSN(t))
	if err != nil {
		t.Fatalf("listener: %v", err)
	}

	// Bus attempts to make the listener LISTEN on an invalid channel.
	_, err = storage.NewPostgresBus(db, backend.EventStore(), listener,
		storage.WithBusChannel("bad channel!"))
	if err == nil {
		t.Fatal("expected error for invalid channel name, got nil")
	}

	_ = listener.Close()
}

// TestPostgresBus_PresetWiring exercises the full stack/postgres preset with
// WithDistributedBus — the integration of every layer added in this commit's
// plan: PgxListener + storage.PostgresBus + preset wiring.
func TestPostgresBus_PresetWiring(t *testing.T) {
	dsn := postgresDSN(t)
	ctx := context.Background()

	listener, err := postgres.NewPgxListenerFromDSN(ctx, dsn)
	if err != nil {
		t.Fatalf("listener: %v", err)
	}

	bundle, err := postgres.New(
		dsn,
		postgres.WithDistributedBus(listener,
			storage.WithBusChannel(uniqueChannel(t))),
	)
	if err != nil {
		t.Fatalf("preset New: %v", err)
	}
	t.Cleanup(func() { _ = bundle.Close() })

	if bundle.Publisher == nil {
		t.Fatal("bundle.Publisher is nil")
	}

	if bundle.Subscriber == nil {
		t.Fatal("bundle.Subscriber is nil")
	}

	if bundle.EventSink == nil {
		t.Fatal("bundle.EventSink is nil")
	}

	// Round-trip: subscribe, save, publish, wait for local delivery.
	received := make(chan event.Event, 1)

	err = bundle.Subscriber.Subscribe("demo.happened",
		func(_ context.Context, e event.Event) error {
			received <- e

			return nil
		})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	aggID := id.NewAggregateID()
	ref := event.NewAggregateRef("Demo", aggID)
	evt, err := event.NewEvent("demo.happened", aggID, "Demo", event.Version(1),
		[]byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if err := bundle.EventSink.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := bundle.Publisher.Publish(ctx, evt); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Local dispatch fires synchronously in Publish. Verify via channel.
	select {
	case receivedEvt := <-received:
		if receivedEvt.ID() != evt.ID() {
			t.Fatalf("received wrong event ID: got %s, want %s",
				receivedEvt.ID(), evt.ID())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: local handler not invoked within 2s")
	}
}
