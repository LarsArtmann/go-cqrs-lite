package event_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2"

	ro "github.com/samber/ro"
)

func TestMap(t *testing.T) {
	t.Parallel()

	bus := event.NewEventBus()

	mapped := ro.Pipe1(bus, ro.Map(func(e event.Event) event.Event {
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

	tapped := ro.Pipe1(bus, ro.TapOnNext(func(e event.Event) {
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

	scanned := ro.Pipe1(bus, ro.Scan(func(state count, e event.Event) count {
		return count{Total: state.Total + 1}
	}, count{}))

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
