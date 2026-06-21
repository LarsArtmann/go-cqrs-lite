package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2/querytest"
)

func registerHandler[T any](d *query.Dispatcher, queryType string, result T) error {
	return d.Register(query.Type(queryType), func(_ context.Context, _ query.Query) (any, error) {
		return result, nil
	})
}

func registerCallOrderHandler[T any](
	d *query.Dispatcher, queryType string, callOrder *[]string, result T,
) error {
	return d.Register(query.Type(queryType), func(_ context.Context, _ query.Query) (any, error) {
		*callOrder = append(*callOrder, "handler")

		return result, nil
	})
}

func TestNewQuery(t *testing.T) {
	t.Parallel()

	q := querytest.New(t, "GetUser")

	if q.Type() != "GetUser" {
		t.Errorf("expected type GetUser, got %s", q.Type())
	}
}

func TestBaseQuery_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ query.Query = querytest.New(t, "TestQuery")
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	handler := func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	}

	err := dispatcher.Register("GetUser", handler)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	if err := registerHandler(dispatcher, "GetUser", "user-123"); err != nil {
		t.Fatal(err)
	}

	q := querytest.New(t, "GetUser")

	result, err := dispatcher.Dispatch(context.Background(), q)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "user-123" {
		t.Errorf("expected user-123, got %v", result)
	}
}

func TestDispatcher_Dispatch_QueryNotSupported(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	q := querytest.New(t, "UnknownQuery")

	_, err := dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected query not supported error")
	}
}

func TestDispatchTyped(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	if err := registerHandler(dispatcher, "GetUserName", "John Doe"); err != nil {
		t.Fatal(err)
	}

	q := querytest.New(t, "GetUserName")

	result, err := query.DispatchTyped[string](context.Background(), dispatcher, q)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "John Doe" {
		t.Errorf("expected John Doe, got %s", result)
	}
}

func TestDispatchTyped_WrongType(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	if err := registerHandler(dispatcher, "GetCount", 42); err != nil {
		t.Fatal(err)
	}

	q := querytest.New(t, "GetCount")

	_, err := query.DispatchTyped[string](context.Background(), dispatcher, q)
	if err == nil {
		t.Error("expected unexpected result type error")
	}
}

func TestDispatcher_Middleware(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	var callOrder []string

	dispatcher.Use(
		queryMiddleware(&callOrder, "middleware1"),
		queryMiddleware(&callOrder, "middleware2"),
	)

	if err := registerCallOrderHandler(dispatcher, "TestQuery", &callOrder, "result"); err != nil {
		t.Fatal(err)
	}

	q := querytest.New(t, "TestQuery")
	_, _ = dispatcher.Dispatch(context.Background(), q)

	eventtest.AssertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
}

func TestDispatcher_Closed(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()
	_ = dispatcher.Close()

	handler := func(_ context.Context, _ query.Query) (any, error) {
		return nil, query.ErrHandlerNotFound
	}

	err := dispatcher.Register("TestQuery", handler)
	if err == nil {
		t.Error("expected dispatcher closed error on Register")
	}

	q := querytest.New(t, "TestQuery")

	_, err = dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected dispatcher closed error on Dispatch")
	}
}
