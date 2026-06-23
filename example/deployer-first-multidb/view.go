package main

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

type CounterView struct {
	Count int `json:"count"`
}

func configureMaterialize(mat *stack.Materialize[CounterView, id.AggregateID]) {
	mat.OnCreate = func(_ context.Context, evt event.Event) (*CounterView, error) {
		if evt.Type() == "counter.incremented" {
			return &CounterView{Count: 1}, nil
		}

		return &CounterView{}, nil
	}

	mat.OnUpdate = func(_ context.Context, evt event.Event, existing *CounterView) (*CounterView, error) {
		if evt.Type() == "counter.incremented" {
			return &CounterView{Count: existing.Count + 1}, nil
		}

		return existing, nil
	}
}

func waitForView(
	ctx context.Context,
	mat *stack.Materialize[CounterView, id.AggregateID],
	counterID id.AggregateID,
	matches func(*CounterView) bool,
) (*CounterView, bool) {
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if view, err := mat.View(ctx, counterID); err == nil && matches(view) {
			return view, true
		}

		select {
		case <-ctx.Done():
			return nil, false
		case <-time.After(10 * time.Millisecond):
		}
	}

	return nil, false
}
