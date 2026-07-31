package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ─── Shared test types ───

type benchItem struct {
	ID    string
	Name  string
	Price int
}

type findItem struct {
	ID string
}

// ─── Helper: build a store + adapter for item queries ───

func setupItemAdapter(
	t *testing.T,
	decoder projectionadapter.PayloadDecoder,
) (*projectionadapter.Adapter, *metaengine.Store) {
	t.Helper()

	q := metaengine.Query[findItem, benchItem](
		"find-item",
		metaengine.On(benchItem{}, func(e benchItem) (string, benchItem) {
			return e.ID, e
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	return projectionadapter.New("items", store, decoder), store
}

func makeEvent(t *testing.T, eventType string, payload any) event.Event {
	t.Helper()

	streamID := id.NewStreamID()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	evt, err := event.NewEvent(
		event.Type(eventType), streamID, "Item", event.Version(1),
		payloadBytes,
	)
	if err != nil {
		t.Fatalf("event.NewEvent: %v", err)
	}

	return evt
}

// ─── Tests ───

func TestAdapter_DecoderFailure(t *testing.T) {
	t.Parallel()

	decoderErr := errors.New("simulated decode failure")
	decoder := func(string, []byte) (any, error) {
		return nil, decoderErr
	}

	adapter, store := setupItemAdapter(t, decoder)
	defer store.Close()

	evt := makeEvent(t, "benchItem", benchItem{ID: "i1", Name: "Widget"})

	err := adapter.Handle(context.Background(), evt)
	if err == nil {
		t.Fatal("Handle should return error when decoder fails")
	}

	if !errors.Is(err, decoderErr) {
		t.Fatalf("error should wrap decoder error, got: %v", err)
	}
}

func TestAdapter_EventTypes_DerivedFromStore(t *testing.T) {
	t.Parallel()

	adapter, store := setupItemAdapter(t, func(eventType string, payload []byte) (any, error) {
		var e benchItem
		err := json.Unmarshal(payload, &e)

		return e, err
	})
	defer store.Close()

	types := adapter.EventTypes()
	if len(types) != 1 {
		t.Fatalf("EventTypes() returned %d types, want 1", len(types))
	}

	if string(types[0]) != "benchItem" {
		t.Fatalf("EventTypes()[0] = %q, want %q", types[0], "benchItem")
	}
}

func TestAdapter_SuccessfulHandle(t *testing.T) {
	t.Parallel()

	adapter, store := setupItemAdapter(t, func(_ string, payload []byte) (any, error) {
		var e benchItem
		err := json.Unmarshal(payload, &e)

		return e, err
	})
	defer store.Close()

	evt := makeEvent(t, "benchItem", benchItem{ID: "item-1", Name: "Widget", Price: 999})

	if err := adapter.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle should succeed, got: %v", err)
	}

	// Verify the data was stored by querying the store.
	result, err := store.Execute(findItem{ID: "item-1"})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}

	item, ok := result.(benchItem)
	if !ok {
		t.Fatalf("result is %T, want benchItem", result)
	}

	if item.Name != "Widget" || item.Price != 999 {
		t.Fatalf("stored item = %+v, want Name=Widget Price=999", item)
	}
}

// ─── Benchmark ───

func BenchmarkAdapter_Handle(b *testing.B) {
	q := metaengine.Query[findItem, benchItem](
		"find-item",
		metaengine.On(benchItem{}, func(e benchItem) (string, benchItem) {
			return e.ID, e
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		b.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	decoder := func(_ string, payload []byte) (any, error) {
		var e benchItem
		err := json.Unmarshal(payload, &e)

		return e, err
	}

	adapter := projectionadapter.New("items-bench", store, decoder)

	streamID := id.NewStreamID()

	payloadBytes, _ := json.Marshal(benchItem{ID: "bench-1", Name: "Bench", Price: 100})

	evt, err := event.NewEvent(
		event.Type("benchItem"), streamID, "Item", event.Version(1),
		payloadBytes,
	)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ResetTimer()

	for range b.N {
		if err := adapter.Handle(ctx, evt); err != nil {
			b.Fatal(err)
		}
	}
}

// ─── EventDecoder tests ───

// eventWithID mirrors the taskmanager pattern: wraps a payload with the
// stream ID so fold handlers can key Map entries by entity ID.
type eventWithID struct {
	ID      string
	Payload benchItem
}

func TestAdapter_EventDecoder_ReceivesFullEvent(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[findItem, benchItem](
		"find-item-ed",
		metaengine.On(eventWithID{}, func(e eventWithID) (string, benchItem) {
			return e.ID, e.Payload
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	var capturedStreamID string

	var capturedVersion event.Version

	dec := func(evt event.Event) (any, error) {
		capturedStreamID = evt.StreamID().String()
		capturedVersion = evt.Version()

		var p benchItem
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, err
		}

		return eventWithID{ID: capturedStreamID, Payload: p}, nil
	}

	adapter := projectionadapter.New("items-ed", store, nil,
		projectionadapter.WithEventDecoder(dec))

	evt := makeEvent(t, "eventWithID", benchItem{ID: "ed-1", Name: "ED Item", Price: 42})

	if err := adapter.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle should succeed, got: %v", err)
	}

	// EventDecoder must receive the full event (StreamID + Version).
	if capturedStreamID != evt.StreamID().String() {
		t.Fatalf("EventDecoder received StreamID %q, want %q",
			capturedStreamID, evt.StreamID().String())
	}

	if capturedVersion != evt.Version() {
		t.Fatalf("EventDecoder received Version %d, want %d",
			capturedVersion, evt.Version())
	}

	// Verify data stored correctly via the wrapped payload.
	result, err := store.Execute(findItem{ID: evt.StreamID().String()})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}

	item, ok := result.(benchItem)
	if !ok {
		t.Fatalf("result is %T, want benchItem", result)
	}

	if item.Name != "ED Item" {
		t.Fatalf("stored item Name = %q, want %q", item.Name, "ED Item")
	}
}

func TestAdapter_EventDecoder_PrecedenceOverPayloadDecoder(t *testing.T) {
	t.Parallel()

	q := metaengine.Query[findItem, benchItem](
		"find-item-prec",
		metaengine.On(benchItem{}, func(e benchItem) (string, benchItem) {
			return e.ID, e
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		q,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}
	defer store.Close()

	payloadDecoderCalled := false

	// Both decoders set — EventDecoder MUST win.
	payloadDec := func(string, []byte) (any, error) {
		payloadDecoderCalled = true
		return benchItem{}, nil
	}

	eventDec := func(evt event.Event) (any, error) {
		var p benchItem
		if err := json.Unmarshal(evt.Payload(), &p); err != nil {
			return nil, err
		}

		return p, nil
	}

	adapter := projectionadapter.New("items-prec", store, payloadDec,
		projectionadapter.WithEventDecoder(eventDec))

	evt := makeEvent(t, "benchItem", benchItem{ID: "prec-1", Name: "Prec", Price: 1})

	if err := adapter.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle should succeed, got: %v", err)
	}

	if payloadDecoderCalled {
		t.Fatal("PayloadDecoder was called but EventDecoder should take precedence")
	}

	// Verify the EventDecoder's output was used (not the PayloadDecoder's stub).
	result, err := store.Execute(findItem{ID: "prec-1"})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}

	item, ok := result.(benchItem)
	if !ok || item.Name != "Prec" {
		t.Fatalf("EventDecoder output not used; result = %+v", result)
	}
}
