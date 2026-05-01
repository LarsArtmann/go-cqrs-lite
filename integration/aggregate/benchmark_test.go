package aggregate_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/aggregate"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func BenchmarkAggregate_RecordEvent(b *testing.B) {
	core := aggregate.MustNewCore(id.NewAggregateID(), "BenchAggregate")
	ctx := context.Background()

	evt, err := event.NewEvent(
		"Benched",
		core.ID(),
		"BenchAggregate",
		1,
		nil,
	)
	if err != nil {
		b.Fatalf("create event: %v", err)
	}

	for b.Loop() {
		core.RecordEvent(ctx, evt)
		core.MarkChangesAsCommitted()
	}
}

func BenchmarkAggregate_LoadFromHistory(b *testing.B) {
	core := aggregate.MustNewCore(id.NewAggregateID(), "BenchAggregate")
	events := make([]event.Event, 100)

	for i := range 100 {
		evt, err := event.NewEvent(
			"Benched",
			core.ID(),
			"BenchAggregate",
			i+1,
			nil,
		)
		if err != nil {
			b.Fatalf("create event: %v", err)
		}

		events[i] = evt
	}

	root := &benchRoot{Core: core}

	b.ResetTimer()

	for b.Loop() {
		err := core.LoadFromHistory(root, events)
		if err != nil {
			b.Fatalf("load from history: %v", err)
		}

		core.MarkChangesAsCommitted()
	}
}

type benchRoot struct {
	*aggregate.Core
}

func (r *benchRoot) Apply(_ event.Event) error { return nil }

func (r *benchRoot) ApplySnapshot(_ []byte) error { return nil }

func (r *benchRoot) LoadEvents(events []event.Event) error {
	return r.LoadFromHistory(r, events)
}

func BenchmarkRepository_Save(b *testing.B) {
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo, _ := aggregate.NewRepository(store, bus)
	ctx := context.Background()

	for b.Loop() {
		o := newOrder(id.NewAggregateID())

		err := o.Place(ctx)
		if err != nil {
			b.Fatalf("place: %v", err)
		}

		err = repo.Save(ctx, o)
		if err != nil {
			b.Fatalf("save: %v", err)
		}
	}
}

func BenchmarkRepository_Load(b *testing.B) {
	store := memory.NewMemoryStore()
	bus := memory.NewMemoryBus()
	repo, _ := aggregate.NewRepository(store, bus)
	ctx := context.Background()

	orderID := id.NewAggregateID()
	o := newOrder(orderID)

	err := o.Place(ctx)
	if err != nil {
		b.Fatalf("place: %v", err)
	}

	err = repo.Save(ctx, o)
	if err != nil {
		b.Fatalf("save: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		loaded := newOrder(orderID)

		err = repo.Load(ctx, loaded)
		if err != nil {
			b.Fatalf("load: %v", err)
		}
	}
}
