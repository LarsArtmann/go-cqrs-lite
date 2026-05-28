package query_test

import (
	"context"
	"testing"

	catdispatcher "github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func registerHandler[T any](d *query.Dispatcher, queryType string, result T) {
	err := d.Register(query.Type(queryType), func(_ context.Context, _ query.Query) (any, error) {
		return result, nil
	})
	if err != nil {
		panic("registerHandler: " + err.Error())
	}
}

func registerCallOrderHandler[T any](
	d *query.Dispatcher, queryType string, callOrder *[]string, result T,
) {
	err := d.Register(query.Type(queryType), func(_ context.Context, _ query.Query) (any, error) {
		*callOrder = append(*callOrder, "handler")

		return result, nil
	})
	if err != nil {
		panic("registerCallOrderHandler: " + err.Error())
	}
}

func TestNewQuery(t *testing.T) {
	t.Parallel()

	q := query.MustNew("GetUser")

	if q.Type() != "GetUser" {
		t.Errorf("expected type GetUser, got %s", q.Type())
	}
}

func TestBaseQuery_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ query.Query = query.MustNew("TestQuery")
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

	registerHandler(dispatcher, "GetUser", "user-123")

	q := query.MustNew("GetUser")

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

	q := query.MustNew("UnknownQuery")

	_, err := dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected query not supported error")
	}
}

func TestDispatchTyped(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	registerHandler(dispatcher, "GetUserName", "John Doe")

	q := query.MustNew("GetUserName")

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

	registerHandler(dispatcher, "GetCount", 42)

	q := query.MustNew("GetCount")

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
		testhelpers.QueryMiddleware(&callOrder, "middleware1"),
		testhelpers.QueryMiddleware(&callOrder, "middleware2"),
	)

	registerCallOrderHandler(dispatcher, "TestQuery", &callOrder, "result")

	q := query.MustNew("TestQuery")
	_, _ = dispatcher.Dispatch(context.Background(), q)

	testhelpers.AssertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
}

func TestDispatcher_Closed(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()
	_ = dispatcher.Close()

	handler := func(_ context.Context, _ query.Query) (any, error) {
		return nil, query.ErrQueryNotSupported
	}

	err := dispatcher.Register("TestQuery", handler)
	if err == nil {
		t.Error("expected dispatcher closed error on Register")
	}

	q := query.MustNew("TestQuery")

	_, err = dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected dispatcher closed error on Dispatch")
	}
}

func TestDispatcher_RegisterHandlerMeta(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()
	meta := catdispatcher.HandlerMeta{
		Name:    "GetUser",
		Version: "1.0.0",
		Summary: "Gets a user by ID",
	}

	dispatcher.RegisterHandlerMeta("user.get", meta)

	entries := dispatcher.CatalogEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	got, ok := entries["user.get"]
	if !ok {
		t.Fatal("missing user.get entry")
	}

	if got.Name != "GetUser" {
		t.Errorf("name = %q, want GetUser", got.Name)
	}
}

func TestDispatcher_CatalogEntries_ReturnsCopy(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()
	dispatcher.RegisterHandlerMeta("q.a", catdispatcher.HandlerMeta{Name: "A"})

	entries := dispatcher.CatalogEntries()
	entries["q.b"] = catdispatcher.HandlerMeta{Name: "B"}

	if len(dispatcher.CatalogEntries()) != 1 {
		t.Error("CatalogEntries should return a copy, not a reference")
	}
}
