package event_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"

	ro "github.com/samber/ro"
)

func newTestEvent(t *testing.T, eventType event.Type) event.Event {
	t.Helper()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent(eventType, aggID, "Test", 1, nil)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	return evt
}

func subscribeAndCollect(src ro.Observable[event.Event]) (*sync.Mutex, *[]event.Event) {
	var mu sync.Mutex
	var received []event.Event

	src.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
	}))

	return &mu, &received
}

func assertEventType(t *testing.T, evt event.Event, want string) {
	t.Helper()
	if evt.Type() != event.Type(want) {
		t.Errorf("expected %s, got %s", want, evt.Type())
	}
}

func TestNewEventBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	eventMu, events := subscribeAndCollect(bus)

	evt := newTestEvent(t, "user.created")
	bus.Next(evt)
	bus.Complete()

	eventMu.Lock()
	defer eventMu.Unlock()

	if len(*events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(*events))
	}

	assertEventType(t, (*events)[0], "user.created")
}

func TestNewEventBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var mu sync.Mutex
	count1, count2 := 0, 0

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		count1++
		mu.Unlock()
	}))

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		mu.Lock()
		count2++
		mu.Unlock()
	}))

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if count1 != 2 {
		t.Errorf("subscriber 1 expected 2 events, got %d", count1)
	}

	if count2 != 2 {
		t.Errorf("subscriber 2 expected 2 events, got %d", count2)
	}
}

func TestNewReplayEventBus(t *testing.T) {
	t.Parallel()

	bus := event.NewReplayEventBus(3)

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.deleted"))

	var received []event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		received = append(received, e)
	}))
	bus.Complete()

	if len(received) != 3 {
		t.Fatalf("expected 3 replayed events, got %d", len(received))
	}

	types := make([]event.Type, len(received))
	for i, e := range received {
		types[i] = e.Type()
	}

	expected := []event.Type{"user.created", "user.changed", "user.deleted"}
	for i, exp := range expected {
		if types[i] != exp {
			t.Errorf("event %d: expected %s, got %s", i, exp, types[i])
		}
	}
}

func TestNewReplayEventBus_LateSubscriber(t *testing.T) {
	t.Parallel()

	bus := event.NewReplayEventBus(2)

	bus.Next(newTestEvent(t, "one"))
	bus.Next(newTestEvent(t, "two"))
	bus.Next(newTestEvent(t, "three"))

	var received []event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		received = append(received, e)
	}))
	bus.Complete()

	if len(received) != 2 {
		t.Fatalf("expected 2 replayed events (buffer=2), got %d", len(received))
	}

	if received[0].Type() != event.Type("two") {
		t.Errorf("expected 'two' first, got %s", received[0].Type())
	}

	if received[1].Type() != event.Type("three") {
		t.Errorf("expected 'three' second, got %s", received[1].Type())
	}
}

func TestNewBehaviorEventBus(t *testing.T) {
	t.Parallel()

	initial := newTestEvent(t, "initial")
	bus := event.NewBehaviorEventBus(initial)

	var received event.Event

	bus.Subscribe(ro.OnNext(func(e event.Event) {
		received = e
	}))

	if received == nil {
		t.Fatal("expected immediate replay of initial value")
	}

	if received.Type() != event.Type("initial") {
		t.Errorf("expected initial, got %s", received.Type())
	}

	bus.Next(newTestEvent(t, "updated"))
	bus.Complete()

	if received.Type() != event.Type("updated") {
		t.Errorf("expected updated, got %s", received.Type())
	}
}

func TestFilterEventType(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	filtered := ro.Pipe1(bus, event.FilterEventType("user.created"))

	mu, received := subscribeAndCollect(filtered)

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.created"))
	bus.Complete()

	mu.Lock()
	defer mu.Unlock()

	if len(*received) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(*received))
	}

	for _, e := range *received {
		if e.Type() != event.Type("user.created") {
			t.Errorf("expected type user.created, got %s", e.Type())
		}
	}
}

func TestFilterEventTypes(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	filtered := ro.Pipe1(bus, event.FilterEventTypes("user.created", "user.deleted"))

	eventMu, events := subscribeAndCollect(filtered)

	bus.Next(newTestEvent(t, "user.created"))
	bus.Next(newTestEvent(t, "user.changed"))
	bus.Next(newTestEvent(t, "user.deleted"))
	bus.Next(newTestEvent(t, "user.loggedIn"))
	bus.Complete()

	eventMu.Lock()
	defer eventMu.Unlock()

	if len(*events) != 2 {
		t.Fatalf("expected 2 filtered events, got %d", len(*events))
	}

	assertEventType(t, (*events)[0], "user.created")
	assertEventType(t, (*events)[1], "user.deleted")
}

func TestHandlerToObserver_Success(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var received []event.Event

	handler := func(ctx context.Context, e event.Event) error {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()

		return nil
	}

	observer := event.HandlerToObserver(handler)

	observer.Next(newTestEvent(t, "user.created"))

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}

	if observer.HasThrown() {
		t.Fatal("expected observer to not have thrown")
	}
}

func TestHandlerToObserver_ErrorTerminatesObserver(t *testing.T) {
	t.Parallel()

	handlerErr := errTestHandler("handler failed")

	handler := func(_ context.Context, _ event.Event) error {
		return handlerErr
	}

	observer := event.HandlerToObserver(handler)

	observer.Next(newTestEvent(t, "user.created"))

	if !observer.HasThrown() {
		t.Fatal("expected observer to have thrown after handler error")
	}

	if !observer.IsClosed() {
		t.Fatal("expected observer to be closed after error")
	}
}

func TestHandlerToObserver_SubsequentEventsDroppedAfterError(t *testing.T) {
	t.Parallel()

	var callCount int

	handler := func(_ context.Context, _ event.Event) error {
		callCount++
		if callCount > 1 {
			return errTestHandler("should not be called")
		}

		return errTestHandler("first error")
	}

	observer := event.HandlerToObserver(handler)

	observer.Next(newTestEvent(t, "first"))
	observer.Next(newTestEvent(t, "second"))
	observer.Next(newTestEvent(t, "third"))

	if callCount != 1 {
		t.Fatalf("expected 1 handler call, got %d (subsequent events should be dropped)", callCount)
	}
}

func TestHandlerToObserver_UsesStreamContext(t *testing.T) {
	t.Parallel()

	key := struct{}{}
	streamCtx := context.WithValue(context.Background(), key, "from-stream")

	var gotCtx context.Context

	handler := func(ctx context.Context, _ event.Event) error {
		gotCtx = ctx
		return nil
	}

	observer := event.HandlerToObserver(handler)

	observer.NextWithContext(streamCtx, newTestEvent(t, "ctx.test"))

	if gotCtx == nil {
		t.Fatal("expected context from stream")
	}

	if gotCtx.Value(key) != "from-stream" {
		t.Error("expected stream context, not event or background context")
	}
}

func TestHandlerToObserver_UsesEventContext(t *testing.T) {
	t.Parallel()

	var gotDeadline time.Time
	var hasDeadline bool

	deadline := time.Now().Add(30 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	handler := func(ctx context.Context, _ event.Event) error {
		gotDeadline, hasDeadline = ctx.Deadline()
		return nil
	}

	observer := event.HandlerToObserver(handler)

	evt, err := event.New(
		"ctx.test",
		id.NewAggregateID(),
		"TestAggregate",
		1,
		[]byte(`{}`),
		event.FromContext(ctx),
	)
	if err != nil {
		t.Fatalf("create event: %v", err)
	}

	observer.NextWithContext(ctx, evt)

	if !hasDeadline {
		t.Fatal("expected deadline from event context")
	}

	if !gotDeadline.Equal(deadline) {
		t.Errorf("expected deadline %v, got %v", deadline, gotDeadline)
	}
}

func TestHandlerToObserverWithContext_OverridesStreamContext(t *testing.T) {
	t.Parallel()

	key := struct{}{}
	overrideCtx := context.WithValue(context.Background(), key, "override")

	var gotCtx context.Context

	handler := func(ctx context.Context, _ event.Event) error {
		gotCtx = ctx
		return nil
	}

	observer := event.HandlerToObserverWithContext(overrideCtx, handler)

	streamCtx := context.WithValue(context.Background(), key, "stream")
	observer.NextWithContext(streamCtx, newTestEvent(t, "ctx.test"))

	if gotCtx == nil {
		t.Fatal("expected context")
	}

	if gotCtx.Value(key) != "override" {
		t.Error("expected override context, not stream context")
	}
}

func TestHandlerToObserverWithContext_ErrorTerminatesObserver(t *testing.T) {
	t.Parallel()

	handlerErr := errTestHandler("ctx handler failed")

	handler := func(_ context.Context, _ event.Event) error {
		return handlerErr
	}

	observer := event.HandlerToObserverWithContext(context.Background(), handler)

	observer.Next(newTestEvent(t, "user.created"))

	if !observer.HasThrown() {
		t.Fatal("expected observer to have thrown after handler error")
	}
}

func TestMap(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	mapped := ro.Pipe1(bus, event.Map(func(e event.Event) event.Event {
		return e
	}))

	var received event.Event

	mapped.Subscribe(ro.OnNext(func(e event.Event) {
		received = e
	}))

	evt := newTestEvent(t, "original")
	bus.Next(evt)
	bus.Complete()

	if received == nil {
		t.Fatal("expected to receive mapped event")
	}
}

func TestTap(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	var sideEffectCount int

	tapped := ro.Pipe1(bus, event.Tap(func(e event.Event) {
		sideEffectCount++
	}))

	var received event.Event

	tapped.Subscribe(ro.OnNext(func(e event.Event) {
		received = e
	}))

	bus.Next(newTestEvent(t, "tapped"))
	bus.Complete()

	if received == nil {
		t.Fatal("expected to receive event after tap")
	}

	if sideEffectCount != 1 {
		t.Fatalf("expected 1 side effect, got %d", sideEffectCount)
	}
}

func TestReplayFilter(t *testing.T) {
	t.Parallel()

	events := []event.Event{
		newTestEvent(t, "before1"),
		newTestEvent(t, "before2"),
		newTestEvent(t, "checkpoint"),
		newTestEvent(t, "after1"),
		newTestEvent(t, "after2"),
	}

	checkpoint := event.Checkpoint{EventID: events[2].ID()}

	obs := ro.FromSlice(events)
	filtered := ro.Pipe1(obs, event.ReplayFilter(nil, checkpoint))

	values, err := ro.Collect(filtered)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 events after checkpoint, got %d", len(values))
	}

	if values[0].Type() != event.Type("after1") {
		t.Errorf("expected after1, got %s", values[0].Type())
	}

	if values[1].Type() != event.Type("after2") {
		t.Errorf("expected after2, got %s", values[1].Type())
	}
}

func TestReplayFilter_WithTypeFilter(t *testing.T) {
	t.Parallel()

	events := []event.Event{
		newTestEvent(t, "before"),
		newTestEvent(t, "target"),
		newTestEvent(t, "other"),
	}

	checkpoint := event.Checkpoint{EventID: events[0].ID()}

	obs := ro.FromSlice(events)
	filtered := ro.Pipe1(obs, event.ReplayFilter([]event.Type{"target"}, checkpoint))

	values, err := ro.Collect(filtered)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}

	if len(values) != 1 {
		t.Fatalf("expected 1 event, got %d", len(values))
	}

	if values[0].Type() != event.Type("target") {
		t.Errorf("expected target, got %s", values[0].Type())
	}
}

func TestReplayFilter_ZeroCheckpoint(t *testing.T) {
	t.Parallel()

	events := []event.Event{
		newTestEvent(t, "one"),
		newTestEvent(t, "two"),
	}

	checkpoint := event.Checkpoint{}

	obs := ro.FromSlice(events)
	filtered := ro.Pipe1(obs, event.ReplayFilter(nil, checkpoint))

	values, err := ro.Collect(filtered)
	if err != nil {
		t.Fatalf("collect failed: %v", err)
	}

	if len(values) != 2 {
		t.Fatalf("expected 2 events (zero checkpoint = all events), got %d", len(values))
	}
}

func TestScanState(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	type count struct{ Total int }

	scanned := ro.Pipe1(bus, event.ScanState(count{}, func(state count, e event.Event) count {
		return count{Total: state.Total + 1}
	}))

	var results []count

	scanned.Subscribe(ro.OnNext(func(c count) {
		results = append(results, c)
	}))

	bus.Next(newTestEvent(t, "a"))
	bus.Next(newTestEvent(t, "b"))
	bus.Next(newTestEvent(t, "c"))
	bus.Complete()

	if len(results) != 3 {
		t.Fatalf("expected 3 scan results, got %d", len(results))
	}

	if results[0].Total != 1 {
		t.Errorf("scan 1: expected 1, got %d", results[0].Total)
	}

	if results[1].Total != 2 {
		t.Errorf("scan 2: expected 2, got %d", results[1].Total)
	}

	if results[2].Total != 3 {
		t.Errorf("scan 3: expected 3, got %d", results[2].Total)
	}
}

type errTestHandler string

func (e errTestHandler) Error() string { return string(e) }
