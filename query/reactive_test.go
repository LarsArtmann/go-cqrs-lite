package query_test

import (
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/query"
)

func TestNewQueryBus_PublishSubscribe(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()

	var received query.Query

	bus.Subscribe(ro.OnNext(func(q query.Query) {
		received = q
	}))

	q := query.MustNew("get.user")
	bus.Next(q)
	bus.Complete()

	if received == nil {
		t.Fatal("expected to receive a query")
	}

	if received.Type() != query.Type("get.user") {
		t.Errorf("expected get.user, got %s", received.Type())
	}
}

func TestNewQueryBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()

	bus.Subscribe(ro.OnNext(func(q query.Query) {
		_ = q
	}))

	bus.Subscribe(ro.OnNext(func(q query.Query) {
		_ = q
	}))

	bus.Next(query.MustNew("get.user"))

	count := bus.CountObservers()

	if count != 2 {
		t.Errorf("expected 2 observers, got %d", count)
	}

	bus.Complete()
}

func TestFilterQueryType(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()

	filtered := ro.Pipe1(bus, query.FilterQueryType("get.user"))

	var received []query.Query

	filtered.Subscribe(ro.OnNext(func(q query.Query) {
		received = append(received, q)
	}))

	bus.Next(query.MustNew("get.user"))
	bus.Next(query.MustNew("list.users"))
	bus.Next(query.MustNew("get.user"))
	bus.Complete()

	if len(received) != 2 {
		t.Fatalf("expected 2 filtered queries, got %d", len(received))
	}

	for _, q := range received {
		if q.Type() != query.Type("get.user") {
			t.Errorf("expected get.user, got %s", q.Type())
		}
	}
}
