package query_test

import (
	"context"
	"errors"
	"testing"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func TestNewQueryBus_BroadcastsToSubscribers(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()
	q := makeTestQuery("user.get")

	var got []query.Type
	bus.Subscribe(ro.OnNext(func(qry query.Query) {
		got = append(got, qry.Type())
	}))

	bus.Next(q)
	bus.Complete()

	if len(got) != 1 || got[0] != q.Type() {
		t.Fatalf("expected [%s], got %v", q.Type(), got)
	}
}

func TestFilterQueryType_DropsOtherTypes(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()
	filtered := ro.Pipe1(bus, query.FilterQueryType("user.get"))

	var got []query.Type
	filtered.Subscribe(ro.OnNext(func(qry query.Query) {
		got = append(got, qry.Type())
	}))

	bus.Next(makeTestQuery("user.get"))
	bus.Next(makeTestQuery("user.list"))
	bus.Next(makeTestQuery("user.get"))
	bus.Complete()

	want := []query.Type{"user.get", "user.get"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestFilterQueryTypes_AllowsMatchingTypes(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()
	filtered := ro.Pipe1(bus, query.FilterQueryTypes("user.get", "user.list"))

	var got []query.Type
	filtered.Subscribe(ro.OnNext(func(qry query.Query) {
		got = append(got, qry.Type())
	}))

	bus.Next(makeTestQuery("user.get"))
	bus.Next(makeTestQuery("user.count"))
	bus.Next(makeTestQuery("user.list"))
	bus.Complete()

	want := []query.Type{"user.get", "user.list"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestFilterQueryTypes_EmptyAllowsAll(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()
	filtered := ro.Pipe1(bus, query.FilterQueryTypes())

	var got int
	filtered.Subscribe(ro.OnNext(func(_ query.Query) {
		got++
	}))

	bus.Next(makeTestQuery("user.get"))
	bus.Next(makeTestQuery("user.list"))
	bus.Complete()

	if got != 2 {
		t.Fatalf("expected 2 queries, got %d", got)
	}
}

func TestHandlerToObserver_RoutesQueriesAndForwardsErrors(t *testing.T) {
	t.Parallel()

	bus := query.NewQueryBus()
	wantErr := errors.New("boom")

	handler := func(_ context.Context, q query.Query) (any, error) {
		if q.Type() == "user.fail" {
			return nil, wantErr
		}

		return struct{}{}, nil
	}

	obs := query.HandlerToObserver(handler)
	bus.Subscribe(obs)

	bus.Next(makeTestQuery("user.ok"))
	bus.Next(makeTestQuery("user.fail"))
	bus.Complete()
}

func TestNewReplayQueryBus_ReplaysToLateSubscribers(t *testing.T) {
	t.Parallel()

	bus := query.NewReplayQueryBus(2)
	first := makeTestQuery("user.first")
	second := makeTestQuery("user.second")
	third := makeTestQuery("user.third")

	bus.Next(first)
	bus.Next(second)
	bus.Next(third)

	var got []query.Type
	bus.Subscribe(ro.OnNext(func(qry query.Query) {
		got = append(got, qry.Type())
	}))

	if len(got) != 2 {
		t.Fatalf("expected 2 replayed queries, got %d", len(got))
	}
	if got[0] != second.Type() || got[1] != third.Type() {
		t.Fatalf("expected [%s %s], got %v", second.Type(), third.Type(), got)
	}
}

func TestNewBehaviorQueryBus_ReplaysLatest(t *testing.T) {
	t.Parallel()

	initial := makeTestQuery("user.initial")
	bus := query.NewBehaviorQueryBus(initial)
	latest := makeTestQuery("user.latest")

	bus.Next(latest)

	var got []query.Type
	bus.Subscribe(ro.OnNext(func(qry query.Query) {
		got = append(got, qry.Type())
	}))

	if len(got) != 1 || got[0] != latest.Type() {
		t.Fatalf("expected [%s], got %v", latest.Type(), got)
	}
}

func makeTestQuery(qType query.Type) query.Query {
	q, err := query.New(qType)
	if err != nil {
		panic(err)
	}

	return q
}
