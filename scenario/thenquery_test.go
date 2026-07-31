package scenario_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/scenario/v4"
)

type thenQueryTestProjection struct {
	counts map[string]int64
}

func (p *thenQueryTestProjection) Name() string { return "test-query-projection" }

func (p *thenQueryTestProjection) Handle(_ context.Context, evt event.Event) error {
	if evt.Type() == "test.increment" {
		p.counts["total"]++
	}

	return nil
}

func (p *thenQueryTestProjection) EventTypes() []event.Type {
	return []event.Type{"test.increment"}
}

var _ projection.Projection = (*thenQueryTestProjection)(nil)

func TestThenQueryResult_SimpleClosure(t *testing.T) {
	t.Parallel()

	proj := &thenQueryTestProjection{counts: make(map[string]int64)}

	evt := mustTestEvent(t, "test.increment")

	s := scenario.GivenProjection(t, proj, evt, evt, evt)
	s.ThenNoError()
	s.ThenQueryResult(
		func() (any, error) {
			return proj.counts["total"], nil
		},
		int64(3),
	)
}

func TestThenQueryResult_MapResult(t *testing.T) {
	t.Parallel()

	proj := &thenQueryTestProjection{counts: map[string]int64{"total": 10}}

	s := scenario.GivenProjection(t, proj)
	s.ThenNoError()
	s.ThenQueryResult(
		func() (any, error) {
			return proj.counts, nil
		},
		map[string]int64{"total": 10},
	)
}

func mustTestEvent(t *testing.T, eventType string) event.Event {
	t.Helper()

	evt, err := event.New(
		event.Type(eventType),
		id.NewStreamID(),
		"Test",
		event.Version(1),
		map[string]any{"key": "value"},
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}
