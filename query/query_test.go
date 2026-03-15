package query_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/query"
)

func TestNewQuery(t *testing.T) {
	q := query.New("GetUser")

	if q.Type() != "GetUser" {
		t.Errorf("expected type GetUser, got %s", q.Type())
	}
}

func TestBaseQuery_ImplementsInterface(t *testing.T) {
	var _ query.Query = query.New("TestQuery")
}

func TestDispatcher_Register(t *testing.T) {
	dispatcher := query.NewDispatcher()

	handler := func(q query.Query) (any, error) {
		return "result", nil
	}

	err := dispatcher.Register("GetUser", handler)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	dispatcher := query.NewDispatcher()

	_ = dispatcher.Register("GetUser", func(q query.Query) (any, error) {
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
	dispatcher := query.NewDispatcher()

	q := query.New("UnknownQuery")
	_, err := dispatcher.Dispatch(context.Background(), q)
	if err == nil {
		t.Error("expected query not supported error")
	}
}

func TestDispatchTyped(t *testing.T) {
	dispatcher := query.NewDispatcher()

	_ = dispatcher.Register("GetUserName", func(q query.Query) (any, error) {
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
	dispatcher := query.NewDispatcher()

	_ = dispatcher.Register("GetCount", func(q query.Query) (any, error) {
		return 42, nil
	})

	q := query.New("GetCount")
	_, err := query.DispatchTyped[string](context.Background(), dispatcher, q)
	if err == nil {
		t.Error("expected unexpected result type error")
	}
}

func TestDispatcher_Middleware(t *testing.T) {
	dispatcher := query.NewDispatcher()

	var callOrder []string

	dispatcher.Use(
		func(h func(query.Query) (any, error)) func(query.Query) (any, error) {
			return func(q query.Query) (any, error) {
				callOrder = append(callOrder, "middleware1")
				return h(q)
			}
		},
		func(h func(query.Query) (any, error)) func(query.Query) (any, error) {
			return func(q query.Query) (any, error) {
				callOrder = append(callOrder, "middleware2")
				return h(q)
			}
		},
	)

	_ = dispatcher.Register("TestQuery", func(q query.Query) (any, error) {
		callOrder = append(callOrder, "handler")
		return "result", nil
	})

	q := query.New("TestQuery")
	_, _ = dispatcher.Dispatch(context.Background(), q)

	expected := []string{"middleware1", "middleware2", "handler"}
	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)
			break
		}
	}
}

func TestDispatcher_Closed(t *testing.T) {
	dispatcher := query.NewDispatcher()
	_ = dispatcher.Close()

	handler := func(q query.Query) (any, error) { return nil, nil }

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
