package decider

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func mustEvent(t *testing.T, eventType string, aggID id.StreamID) event.Event {
	t.Helper()
	evt, err := event.New(event.Type(eventType), aggID, "Test", 1, struct{}{})
	if err != nil {
		t.Fatalf("event.New() error: %v", err)
	}

	return evt
}

type counterState struct {
	Count int
}

func foldCounter(s counterState, evt event.Event) (counterState, error) {
	next := s
	switch evt.Type() {
	case "counter.incremented":
		next.Count++
	case "counter.decremented":
		next.Count--
	}

	return next, nil
}

func TestStrictApply_KnownEvent(t *testing.T) {
	apply := StrictApply(foldCounter, []event.Type{
		"counter.incremented",
		"counter.decremented",
	})

	aggID := id.NewAggregateID()
	state, err := apply(counterState{Count: 0}, mustEvent(t, "counter.incremented", aggID))
	if err != nil {
		t.Fatalf("StrictApply() error: %v", err)
	}
	if state.Count != 1 {
		t.Errorf("Count = %d, want 1", state.Count)
	}
}

func TestStrictApply_UnknownEvent(t *testing.T) {
	apply := StrictApply(foldCounter, []event.Type{
		"counter.incremented",
		"counter.decremented",
	})

	aggID := id.NewAggregateID()
	_, err := apply(counterState{Count: 5}, mustEvent(t, "counter.reset", aggID))
	if err == nil {
		t.Fatal("StrictApply() should return error for unknown event type")
	}
	if !errors.Is(err, ErrStrictApplyUnknownType) {
		t.Errorf("StrictApply() error should wrap ErrStrictApplyUnknownType, got: %v", err)
	}
}

func TestStrictApply_PassesThroughErrors(t *testing.T) {
	errorApply := func(s counterState, evt event.Event) (counterState, error) {
		return s, errTestApply
	}

	apply := StrictApply(errorApply, []event.Type{"test.event"})

	aggID := id.NewAggregateID()
	_, err := apply(counterState{}, mustEvent(t, "test.event", aggID))
	if err == nil {
		t.Fatal("StrictApply() should pass through errors from inner apply")
	}
}

var errTestApply = errApplySentinelError("test apply error")

type errApplySentinelError string

func (e errApplySentinelError) Error() string { return string(e) }
