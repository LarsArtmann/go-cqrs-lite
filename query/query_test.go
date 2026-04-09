package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query"
)

func TestNewQuery(t *testing.T) {
	t.Parallel()

	q := query.New("GetUser")

	if q.Type() != "GetUser" {
		t.Errorf("expected type GetUser, got %s", q.Type())
	}
}

func TestBaseQuery_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ query.Query = query.New("TestQuery")
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	handler := func(_ query.Query) (any, error) {
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

	_ = dispatcher.Register("GetUser", func(_ query.Query) (any, error) {
		return "user-123", nil
	})

	q := query.New("GetUser")

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

	q := query.New("UnknownQuery")

	_, err := dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected query not supported error")
	}
}

func TestDispatchTyped(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	_ = dispatcher.Register("GetUserName", func(_ query.Query) (any, error) {
		return "John Doe", nil
	})

	q := query.New("GetUserName")

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

	_ = dispatcher.Register("GetCount", func(_ query.Query) (any, error) {
		return 42, nil
	})

	q := query.New("GetCount")

	_, err := query.DispatchTyped[string](context.Background(), dispatcher, q)
	if err == nil {
		t.Error("expected unexpected result type error")
	}
}

func makeTestMiddleware(
	callOrder *[]string,
	name string,
) func(func(query.Query) (any, error)) func(query.Query) (any, error) {
	return func(h func(query.Query) (any, error)) func(query.Query) (any, error) {
		return func(q query.Query) (any, error) {
			*callOrder = append(*callOrder, name)

			return h(q)
		}
	}
}

func assertCallOrder(t *testing.T, callOrder []string, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}

func TestDispatcher_Middleware(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()

	var callOrder []string

	dispatcher.Use(
		makeTestMiddleware(&callOrder, "middleware1"),
		makeTestMiddleware(&callOrder, "middleware2"),
	)

	_ = dispatcher.Register("TestQuery", func(_ query.Query) (any, error) {
		callOrder = append(callOrder, "handler")

		return "result", nil
	})

	q := query.New("TestQuery")
	_, _ = dispatcher.Dispatch(context.Background(), q)

	assertCallOrder(t, callOrder, []string{"middleware1", "middleware2", "handler"})
}

func TestDispatcher_Closed(t *testing.T) {
	t.Parallel()

	dispatcher := query.NewDispatcher()
	_ = dispatcher.Close()

	handler := func(_ query.Query) (any, error) {
		return nil, query.ErrQueryValidation
	}

	err := dispatcher.Register("TestQuery", handler)
	if err == nil {
		t.Error("expected dispatcher closed error on Register")
	}

	q := query.New("TestQuery")

	_, err = dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected dispatcher closed error on Dispatch")
	}
}
