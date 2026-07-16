package http

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// TestSSEHandler_ByteBudget_StopsReplayEarly verifies that WithReplayByteBudget
// cuts replay short when cumulative payload bytes exceed the budget, sending an
// SSEReplayIncompleteEvent advisory.
func TestSSEHandler_ByteBudget_StopsReplayEarly(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Big", aggID)

	// 10 events × ~50 bytes each = ~500 bytes total. Budget of 150 → ~3 events.
	const eventCount = 10
	events := make([]event.Event, 0, eventCount)

	for i := range eventCount {
		evt, err := event.NewEvent("BigEvent", aggID, "Big", 1,
			[]byte(`{"payload":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`))
		if err != nil {
			t.Fatalf("create evt %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(context.Background(), ref, events, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 0), // unlimited → batched streaming
		WithReplayByteBudget(150),      // very tight budget
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=bytebudget", nil)
	req.Header.Set("Last-Event-ID", events[0].ID().String()) // cursor → replay events[1..]

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Should have stopped early — NOT all 9 events replayed.
	got := strings.Count(body, "event: BigEvent")
	if got >= eventCount-1 {
		t.Errorf(
			"byte budget should have cut replay short; got %d events (expected < %d)",
			got,
			eventCount-1,
		)
	}

	if got == 0 {
		t.Errorf("expected at least one event before budget cut, got 0; body: %q", body)
	}

	if !strings.Contains(body, SSEReplayIncompleteEvent) {
		t.Errorf(
			"expected %q advisory when byte budget exceeded; body: %q",
			SSEReplayIncompleteEvent,
			body,
		)
	}
}

// TestSSEHandler_ByteBudget_DisabledSentinel verifies that
// WithReplayByteBudget(SSEReplayBudgetDisabled) allows all events to be
// replayed without any byte-budget cut, even when total payload would
// exceed the default 8MB auto-default.
func TestSSEHandler_ByteBudget_DisabledSentinel(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Big", aggID)

	const eventCount = 10
	events := make([]event.Event, 0, eventCount)

	for i := range eventCount {
		evt, err := event.NewEvent("BigEvent", aggID, "Big", 1,
			[]byte(`{"payload":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`))
		if err != nil {
			t.Fatalf("create evt %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(context.Background(), ref, events, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 0), // unlimited → batched streaming
		WithReplayByteBudget(SSEReplayBudgetDisabled), // explicitly disabled
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=no-budget", nil)
	req.Header.Set("Last-Event-ID", events[0].ID().String())

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// All 9 remaining events should be replayed — no budget cut.
	got := strings.Count(body, "event: BigEvent")
	if got != eventCount-1 {
		t.Errorf("expected %d events (all replayed, budget disabled), got %d; body: %q",
			eventCount-1, got, body)
	}

	if strings.Contains(body, SSEReplayIncompleteEvent) {
		t.Errorf("did not expect %q advisory when budget disabled; body: %q",
			SSEReplayIncompleteEvent, body)
	}
}

// capacity still correctly deduplicates replay→live overlap. Uses a tiny ring
// to confirm the option is wired through.
func TestSSEHandler_DedupRingCapacity_Custom(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)

	evt0, err := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`{"seq":0}`))
	if err != nil {
		t.Fatalf("create evt0: %v", err)
	}

	evt1, err := event.NewEvent("TestEvent", aggID, "Test", 2, []byte(`{"seq":1}`))
	if err != nil {
		t.Fatalf("create evt1: %v", err)
	}

	if err := store.Save(context.Background(), ref, []event.Event{evt0, evt1}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 100),
		WithDedupRingCapacity(2), // tiny ring, still holds both replayed IDs
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	// Replay from evt0 → only evt1 is replayed and added to the ring.
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=tiny-ring", nil)
	req.Header.Set("Last-Event-ID", evt0.ID().String())

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(100 * time.Millisecond)

	// Now publish evt1 live — it should be suppressed by the dedup ring.
	if err := bus.Publish(context.Background(), evt1); err != nil {
		t.Fatalf("publish evt1 live: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	count := strings.Count(body, `{"seq":1}`)

	// evt1 should appear exactly once (from replay, NOT again from live).
	if count != 1 {
		t.Errorf("expected evt1 delivered once (deduped), got %d; body: %q", count, body)
	}
}

// TestSSEHandler_EventFilter verifies that WithEventFilter drops events
// the predicate rejects, and forwards those it accepts.
func TestSSEHandler_EventFilter(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(
		bus,
		WithEventFilter(func(typ event.Type) bool {
			return string(typ) == "user.created"
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=filter", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)

	aggID := id.NewAggregateID()

	accepted, _ := event.NewEvent("user.created", aggID, "User", 1, []byte(`{"n":"alice"}`))
	rejected, _ := event.NewEvent("order.placed", aggID, "Order", 1, []byte(`{"item":"book"}`))

	_ = bus.Publish(context.Background(), accepted)
	_ = bus.Publish(context.Background(), rejected)

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, "user.created") {
		t.Errorf("accepted event should be in body; got: %q", body)
	}

	if strings.Contains(body, "order.placed") {
		t.Errorf("rejected event should NOT be in body; got: %q", body)
	}
}

// TestSSEHandler_RetryField verifies that the SSE retry field is sent on
// connect and reflects WithRetryInterval.
func TestSSEHandler_RetryField(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus, WithRetryInterval(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=retry", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, "retry: 3000") {
		t.Errorf("expected 'retry: 3000' in body; got: %q", body)
	}
}

// TestSSEHandler_ConcurrentClose verifies that Close() during active
// streaming doesn't panic or race (close guard for #15).
func TestSSEHandler_ConcurrentClose(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(bus)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=conc", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)

	// Close while the handler is running — should not panic.
	broker.Close()

	cancel()
	<-done
}

// TestSSEHandler_ByteBudget_LargePayload verifies that the byte budget
// correctly handles events with large payloads (exceeding the budget).
func TestSSEHandler_ByteBudget_LargePayload(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Big", aggID)

	// Create 5 events, each with ~100KB payload → 500KB total.
	// Budget of 250KB → ~2-3 events before cutoff.
	const payloadSize = 100 * 1024 // 100KB
	const eventCount = 5

	events := make([]event.Event, 0, eventCount)
	for i := range eventCount {
		payload := make([]byte, payloadSize)
		for j := range payload {
			payload[j] = byte('A' + (i % 26))
		}

		evt, err := event.NewEvent("BigEvent", aggID, "Big", event.Version(i+1), payload)
		if err != nil {
			t.Fatalf("create event %d: %v", i, err)
		}

		events = append(events, evt)
	}

	if err := store.Save(context.Background(), ref, events, 0); err != nil {
		t.Fatalf("save events: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 0), // unlimited streaming
		WithReplayByteBudget(250*1024), // 250KB budget
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=big-payload", nil)
	req.Header.Set("Last-Event-ID", events[0].ID().String()) // cursor → replay events[1..]

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	// Should have delivered some events but been cut short by the budget.
	delivered := strings.Count(body, "event: BigEvent")
	if delivered == 0 {
		t.Error("expected at least one event delivered before budget cutoff")
	}

	// 4 remaining events × 100KB = 400KB. Budget is 250KB → ~2 events max.
	if delivered >= eventCount-1 {
		t.Errorf(
			"budget should cut replay short; got %d events (expected < %d)",
			delivered,
			eventCount-1,
		)
	}
}

// TestSSEHandler_PayloadTransform_Live verifies that WithPayloadTransform
// is applied to events on the live delivery path — the raw payload does NOT
// appear on the wire, and the transformed payload does.
func TestSSEHandler_PayloadTransform_Live(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(
		bus,
		WithPayloadTransform(func(evt event.Event) []byte {
			return []byte(`{"transformed":true}`)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=xform-live", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)

	aggID := id.NewAggregateID()
	evt, _ := event.NewEvent("user.created", aggID, "User", 1, []byte(`{"raw":"cbor-bytes"}`))
	_ = bus.Publish(context.Background(), evt)

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, `{"transformed":true}`) {
		t.Errorf("transformed payload missing from wire; body: %q", body)
	}

	if strings.Contains(body, `{"raw":"cbor-bytes"}`) {
		t.Errorf("raw payload should NOT appear on wire; body: %q", body)
	}
}

// TestSSEHandler_PayloadTransform_CBOR_ToJSON_BrowserFlow verifies the
// cross-codec translation that documents ADR-0051: typed CBOR-stamped events
// (created via [event.New]) flow through [WithPayloadTransform], where the
// consumer decodes the CBOR body and re-emits it as JSON. Without the
// transform, browsers cannot parse raw CBOR — the transform is the only
// sanctioned way to bridge internal CBOR encoding to browser-friendly JSON.
func TestSSEHandler_PayloadTransform_CBOR_ToJSON_BrowserFlow(t *testing.T) {
	t.Parallel()

	bus := eventtest.NewFakeBus()
	defer bus.Close()

	broker, err := NewSSEBroker(
		bus,
		WithPayloadTransform(func(evt event.Event) []byte {
			type payload struct {
				Name string `json:"name"`
			}

			p, decodeErr := event.DecodePayloadAuto[payload](evt)
			if decodeErr != nil {
				return []byte(`{"decode_error":"` + decodeErr.Error() + `"}`)
			}

			out, marshalErr := json.Marshal(p)
			if marshalErr != nil {
				return []byte(`{"marshal_error":"` + marshalErr.Error() + `"}`)
			}

			return out
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=cbor-json", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(50 * time.Millisecond)

	aggID := id.NewAggregateID()
	typed := struct {
		Name string `cbor:"name"`
	}{Name: "alice-cbor"}

	evt, err := event.New("user.created", aggID, "User", 1, typed)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	if evt.Encoding() != codec.EncodingCBOR {
		t.Fatalf(
			"expected CBOR encoding, got %q — New() didn't pick up DefaultCodec",
			evt.Encoding(),
		)
	}

	if publishErr := bus.Publish(context.Background(), evt); publishErr != nil {
		t.Fatalf("publish: %v", publishErr)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, `{"name":"alice-cbor"}`) {
		t.Errorf("transformed CBOR→JSON payload missing from wire; body: %q", body)
	}

	if strings.Contains(body, `alice-cbor`) && !strings.Contains(body, `"alice-cbor"`) {
		t.Errorf("payload bytes appeared without JSON quoting; body: %q", body)
	}
}

// TestSSEHandler_PayloadTransform_Replay verifies that WithPayloadTransform
// is applied to events on the replay path — the same transform applied to
// live events is also applied to journal-replayed events.
func TestSSEHandler_PayloadTransform_Replay(t *testing.T) {
	t.Parallel()

	store := eventtest.NewFakeStore()
	bus := eventtest.NewFakeBus()
	defer bus.Close()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Test", aggID)

	evt0, _ := event.NewEvent("TestEvent", aggID, "Test", 1, []byte(`{"seq":0}`))
	evt1, _ := event.NewEvent("TestEvent", aggID, "Test", 2, []byte(`{"seq":1}`))

	if err := store.Save(context.Background(), ref, []event.Event{evt0, evt1}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	broker, err := NewSSEBroker(
		bus,
		WithReconnectJournal(store, 100),
		WithPayloadTransform(func(evt event.Event) []byte {
			var data map[string]any
			_ = json.Unmarshal(evt.Payload(), &data)
			data["transformed"] = true

			out, _ := json.Marshal(data)

			return out
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events?client=xform-replay", nil)
	req.Header.Set("Last-Event-ID", evt0.ID().String())

	rec := httptest.NewRecorder()
	done := make(chan struct{})

	go func() {
		defer close(done)
		SSEHandler(broker).ServeHTTP(rec, req)
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()

	if !strings.Contains(body, `"transformed":true`) {
		t.Errorf("replayed event should be transformed; body: %q", body)
	}
}
