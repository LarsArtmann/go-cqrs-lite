package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v4"
)

func TestGettingStarted_CounterValue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	bundle, err := sqlite.New(dbPath)
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}

	defer func() { _ = bundle.Close() }()

	repo, err := stack.Repository(bundle, decider.Decider[CounterState]{
		Initial: CounterState{},
		Apply:   applyCounter,
	})
	if err != nil {
		t.Fatalf("repo: %v", err)
	}

	mat, err := stack.NewMaterialize[CounterView, id.AggregateID](bundle, nil,
		func(evt event.Event) (id.AggregateID, error) { return evt.AggregateID(), nil })
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}

	mat.OnCreate = func(_ context.Context, evt event.Event) (*CounterView, error) {
		p, _ := event.DecodePayloadAuto[IncrementedPayload](evt)

		return &CounterView{Value: p.Amount}, nil
	}

	mat.OnUpdate = func(_ context.Context, evt event.Event, ex *CounterView) (*CounterView, error) {
		p, _ := event.DecodePayloadAuto[IncrementedPayload](evt)

		return &CounterView{Value: ex.Value + p.Amount}, nil
	}

	counterID := id.NewAggregateID()

	for _, amt := range []int{5, 3, 2} {
		if err := repo.Execute(ctx, counterID, "Counter", increment(counterID, amt)); err != nil {
			t.Fatalf("execute: %v", err)
		}
	}

	catchUp, _ := bundle.CatchUpSubscriber()
	msgs, _ := catchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)

	go func() {
		handler := mat.HandlerFunc()
		for msg := range msgs {
			_ = handler(msg)
			msg.Ack()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	view, err := mat.View(ctx, counterID)
	if err != nil {
		t.Fatalf("view: %v", err)
	}

	if view.Value != 10 {
		t.Errorf("counter value: got %d, want 10", view.Value)
	}
}
