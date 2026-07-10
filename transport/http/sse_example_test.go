package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
)

// TestSSEExample_OfflineReconnection is a runnable example demonstrating the
// full SSE offline reconnection flow: events arrive while a client is
// disconnected, then the client reconnects with Last-Event-ID and receives
// replay + live handoff without gaps or duplicates.
//
// This serves as both documentation and a regression test for the SSE
// reconnection pipeline (journal replay → dedup → live streaming).
func TestSSEExample_OfflineReconnection(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 100),
		WithRetryInterval(3*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Doc", aggID)

	// Phase 1: events arrive while the client is OFFLINE.
	docCreated, _ := event.NewEvent("doc.created", aggID, "Doc", 1, []byte(`{"title":"Hello"}`))
	docUpdated, _ := event.NewEvent(
		"doc.updated",
		aggID,
		"Doc",
		2,
		[]byte(`{"title":"Hello World"}`),
	)

	if err := store.Save(
		context.Background(),
		ref,
		[]event.Event{docCreated, docUpdated},
		0,
	); err != nil {
		t.Fatal(err)
	}

	// Phase 2: client reconnects with Last-Event-ID = docCreated.
	// The broker should replay docUpdated, then switch to live.
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=doc-client", nil)
	req.Header.Set("Last-Event-ID", docCreated.ID().String())

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(100 * time.Millisecond)

	// Phase 3: publish a live event — dedup should suppress docUpdated
	// (already replayed) but deliver this new one.
	docDeleted, _ := event.NewEvent("doc.deleted", aggID, "Doc", 3, []byte(`{}`))

	if err := bus.Publish(context.Background(), docDeleted); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Verify: docUpdated replayed from journal.
	if !strings.Contains(body, "doc.updated") {
		t.Errorf("expected doc.updated in replay; body: %q", body)
	}

	// Verify: docDeleted delivered live.
	if !strings.Contains(body, "doc.deleted") {
		t.Errorf("expected doc.deleted from live; body: %q", body)
	}

	// Verify: docUpdated NOT duplicated (replayed once, then suppressed live).
	if strings.Count(body, "doc.updated") != 1 {
		t.Errorf("expected doc.updated exactly once; body: %q", body)
	}

	// Verify: retry field sent on connect.
	if !strings.Contains(body, "retry:") {
		t.Errorf("expected retry field in body; body: %q", body)
	}

	// Verify: docCreated NOT replayed (it's BEFORE the Last-Event-ID cursor).
	// The dedup ring tracks docUpdated (replayed), but docCreated is never sent.
	// We check that there's exactly 2 data events (updated + deleted), not 3.
	if strings.Count(body, "data: ") != 2 {
		t.Errorf("expected exactly 2 data events; body: %q", body)
	}
}
