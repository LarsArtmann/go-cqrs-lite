package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
)

func TestWriteSSEEvent_Simple(t *testing.T) {
	var buf bytes.Buffer

	err := WriteSSEEvent(&buf, SSEEvent{
		Event: "user.created",
		ID:    NewSSEEventID("evt-123"),
		Data:  `{"name":"Alice"}`,
	})
	if err != nil {
		t.Fatalf("WriteSSEEvent: %v", err)
	}

	got := buf.String()
	want := "event: user.created\ndata: {\"name\":\"Alice\"}\nid: evt-123\n\n"

	if got != want {
		t.Errorf("wire format mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteSSEEvent_MultiLineData(t *testing.T) {
	var buf bytes.Buffer

	err := WriteSSEEvent(&buf, SSEEvent{
		Event: "user.created",
		ID:    NewSSEEventID("evt-456"),
		Data:  "line1\nline2\nline3",
	})
	if err != nil {
		t.Fatalf("WriteSSEEvent: %v", err)
	}

	got := buf.String()

	// Per SSE spec, each line of data must get its own "data:" prefix
	if !strings.Contains(got, "data: line1\n") {
		t.Errorf("missing data: line1, got: %q", got)
	}

	if !strings.Contains(got, "data: line2\n") {
		t.Errorf("missing data: line2, got: %q", got)
	}

	if !strings.Contains(got, "data: line3\n") {
		t.Errorf("missing data: line3, got: %q", got)
	}

	// Must end with blank line (event terminator)
	if !strings.HasSuffix(got, "\n\n") {
		t.Errorf("missing event terminator, got: %q", got)
	}
}

func TestWriteSSEEvent_OptionalFields(t *testing.T) {
	var buf bytes.Buffer

	// No Type, no ID, no Retry — just data
	err := WriteSSEEvent(&buf, SSEEvent{Data: "hello"})
	if err != nil {
		t.Fatalf("WriteSSEEvent: %v", err)
	}

	got := buf.String()
	want := "data: hello\n\n"

	if got != want {
		t.Errorf("minimal event mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestWriteSSEEvent_Retry(t *testing.T) {
	var buf bytes.Buffer

	err := WriteSSEEvent(&buf, SSEEvent{
		Data:  "x",
		Retry: 5000,
	})
	if err != nil {
		t.Fatalf("WriteSSEEvent: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "retry: 5000\n") {
		t.Errorf("missing retry field, got: %q", got)
	}
}

func TestWriteSSEHeartbeat(t *testing.T) {
	var buf bytes.Buffer

	err := WriteSSEHeartbeat(&buf)
	if err != nil {
		t.Fatalf("WriteSSEHeartbeat: %v", err)
	}

	got := buf.String()
	if got != ": heartbeat\n\n" {
		t.Errorf("heartbeat mismatch: got %q", got)
	}
}

func TestParseSSEEventID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		id, err := ParseSSEEventID("evt-123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if id.Get() != "evt-123" {
			t.Errorf("got %q, want evt-123", id)
		}
	})

	t.Run("rejects newline", func(t *testing.T) {
		_, err := ParseSSEEventID("evt\nmalicious")
		if err == nil {
			t.Fatal("expected error for newline in ID")
		}
	})

	t.Run("rejects carriage return", func(t *testing.T) {
		_, err := ParseSSEEventID("evt\rmalicious")
		if err == nil {
			t.Fatal("expected error for carriage return in ID")
		}
	})
}

func TestSSEBroker_MultiClient(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ch1 := broker.AddClient("client-1")
	ch2 := broker.AddClient("client-2")

	if broker.ClientCount() != 2 {
		t.Fatalf("expected 2 clients, got %d", broker.ClientCount())
	}

	evt, err := event.NewEvent("TestEvent", id.NewAggregateID(), "Test", 1, []byte("payload"))
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// Both clients should receive the event
	for i, ch := range []chan event.Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.Type() != "TestEvent" {
				t.Fatalf("client %d: expected TestEvent, got %s", i+1, received.Type())
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("client %d: timeout waiting for event", i+1)
		}
	}
}

func TestSSEBroker_NilBus(t *testing.T) {
	_, err := NewSSEBroker(nil)
	if err == nil {
		t.Fatal("expected error for nil bus")
	}
}

func TestSSEBroker_Close(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}

	ch := broker.AddClient("client-1")
	if broker.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", broker.ClientCount())
	}

	broker.Close()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for channel close")
	}

	// ClientCount should be 0 after close
	if broker.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after close, got %d", broker.ClientCount())
	}
}

func TestSSEHandler_WireFormat(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	rec, stop := startSSE(broker, "test", "")

	// Wait for client registration
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && broker.ClientCount() == 0 {
		time.Sleep(5 * time.Millisecond)
	}

	if broker.ClientCount() == 0 {
		t.Fatal("SSE handler did not register as client within timeout")
	}

	evt, err := event.NewEvent(
		"UserCreated",
		id.NewAggregateID(),
		"User",
		1,
		[]byte(`{"name":"Alice"}`),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	if err := bus.Publish(context.Background(), evt); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	stop()

	body := rec.Body.String()

	// Verify spec-correct wire format
	if !strings.Contains(body, "event: UserCreated\n") {
		t.Errorf("missing event: field in body: %q", body)
	}

	if !strings.Contains(body, `data: {"name":"Alice"}`) {
		t.Errorf("missing data: field in body: %q", body)
	}

	if !strings.Contains(body, "id: ") {
		t.Errorf("missing id: field in body: %q", body)
	}
}

func TestSSEHandler_StatusOK(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	rec, stop := startSSE(broker, "test", "")
	time.Sleep(50 * time.Millisecond)
	stop()

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestSSEHandler_ConcurrentClients(t *testing.T) {
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	const numClients = 10

	var wg sync.WaitGroup

	for i := range numClients {
		wg.Add(1)

		go func(n int) {
			defer wg.Done()

			_, stop := startSSE(broker, strings.Repeat("c", n+1), "")
			time.Sleep(50 * time.Millisecond)
			stop()
		}(i)
	}

	wg.Wait()

	// All clients should have been cleaned up
	if broker.ClientCount() != 0 {
		t.Errorf("expected 0 clients after all disconnected, got %d", broker.ClientCount())
	}
}

func TestSSEHandler_LastEventID_Reconnect(t *testing.T) {
	store, events := newFakeStoreWithEvents(t, "TestEvent", "Test",
		[]byte(`{"seq":1}`), []byte(`{"seq":2}`))

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus, WithReconnectJournal(store, 100))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Reconnect with Last-Event-ID = events[0].ID() → should replay events[1].
	rec, stop := startSSE(broker, "reconnect", events[0].ID().String())
	time.Sleep(100 * time.Millisecond)
	stop()

	body := rec.Body.String()

	if !strings.Contains(body, `{"seq":2}`) {
		t.Errorf("expected replayed evt2 in body, got: %q", body)
	}

	if strings.Contains(body, `{"seq":1}`) {
		t.Errorf("evt1 should NOT be replayed (it was the last-seen ID), got: %q", body)
	}
}

func TestSSEHandler_ReplayDedup_NoDuplicates(t *testing.T) {
	// Verify that events replayed from journal are NOT re-delivered via
	// the live bus. This is the real dedup test: events[1] is in the journal
	// AND published live after the client connects with Last-Event-ID=events[0].
	store, events := newFakeStoreWithEvents(t, "TestEvent", "Test",
		[]byte(`{"seq":0}`), []byte(`{"seq":1}`))

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus, WithReconnectJournal(store, 100))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Connect with Last-Event-ID=events[0] → replays events[1] from journal.
	rec, stop := startSSE(broker, "dedup", events[0].ID().String())
	time.Sleep(100 * time.Millisecond)

	// Publish events[1] again via live bus — should be suppressed by dedup
	// because it was already replayed from the journal.
	if err := bus.Publish(context.Background(), events[1]); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	stop()

	body := rec.Body.String()
	count := strings.Count(body, `{"seq":1}`)

	if count != 1 {
		t.Errorf(
			"evt1 should appear exactly once (replayed, not re-delivered live), got %d times in: %q",
			count,
			body,
		)
	}
}

func TestSSEHandler_UnlimitedReplay(t *testing.T) {
	// Verify that replayLimit <= 0 streams ALL events from the journal
	// in batches, not just the first 1000 (or batch size).
	const totalEvents = 600 // > sseReplayBatchSize (500) to force multiple batches

	payloads := make([][]byte, totalEvents)
	for i := range payloads {
		payloads[i] = fmt.Appendf(nil, `{"seq":%d}`, i)
	}

	store, events := newFakeStoreWithEvents(t, "TestEvent", "Test", payloads...)

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	// Unlimited replay (0 = unlimited streaming).
	broker, err := NewSSEBroker(bus, WithReconnectJournal(store, 0))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Connect with Last-Event-ID = first event → should replay events[1..599].
	rec, stop := startSSE(broker, "unlimited", events[0].ID().String())
	time.Sleep(200 * time.Millisecond)
	stop()

	body := rec.Body.String()

	// Count how many events were delivered by counting "id:" lines.
	// events[0] is the cursor (not replayed), so we expect totalEvents-1.
	idCount := strings.Count(body, "\nid:")

	if idCount != totalEvents-1 {
		t.Errorf("expected %d replayed events (unlimited), got %d", totalEvents-1, idCount)
	}

	// Verify the last event made it through (not truncated at batch boundary).
	lastSeq := fmt.Sprintf(`{"seq":%d}`, totalEvents-1)
	if !strings.Contains(body, lastSeq) {
		t.Errorf("last event %q missing from unlimited replay body (len=%d)", lastSeq, len(body))
	}
}

// delayedJournal was promoted to testutil.DelayedJournal for reuse across
// modules. See testutil/journal.go.

func TestSSEHandler_ReplayTimeout_SendsAdvisoryEvent(t *testing.T) {
	// When replayTimeout fires before the journal read completes, the broker
	// must send an SSEReplayIncompleteEvent advisory and switch to live.
	store, events := newFakeStoreWithEvents(t, "TestEvent", "Test",
		[]byte(`{"seq":0}`), []byte(`{"seq":1}`))

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	// Wrap the store with a 200ms delay so the 10ms timeout fires mid-read.
	slowStore := testutil.NewDelayedJournal(store, 200*time.Millisecond)

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(slowStore, 100),
		WithReplayTimeout(10*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	rec, stop := startSSE(broker, "timeout", events[0].ID().String())
	time.Sleep(200 * time.Millisecond)
	stop()

	body := rec.Body.String()

	if !strings.Contains(body, SSEReplayIncompleteEvent) {
		t.Errorf("expected %q advisory event in body, got: %q", SSEReplayIncompleteEvent, body)
	}
}
