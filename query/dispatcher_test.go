package query_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

func mustNewQuery(queryType query.Type) *query.BasicQuery {
	q, err := query.New(queryType)
	if err != nil {
		panic(err)
	}
	return q
}


func registerHandler(d *query.Dispatcher, name query.Type, result any) {
	_ = d.Register(name, func(_ context.Context, _ query.Query) (any, error) {
		return result, nil
	})
}

func useMiddleware(called *bool, d *query.Dispatcher) {
	d.Use(func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			*called = true

			return next(ctx, q)
		}
	})
}

func TestNew_EmptyType(t *testing.T) {
	t.Parallel()

	_, err := query.New("")
	if err == nil {
		t.Error("expected error for empty query type")
	}
}

func TestMustNew_PanicsOnEmptyType(t *testing.T) {
	t.Parallel()

	eventtest.AssertPanics(t, func() { _ = mustNewQuery("") })
}

func TestDispatcher_Dispatch_HandlerError(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	ctx := context.Background()

	handlerErr := errors.New("handler failed")
	_ = d.Register("FailQuery", func(_ context.Context, _ query.Query) (any, error) {
		return nil, handlerErr
	})

	q := mustNewQuery("FailQuery")

	_, err := d.Dispatch(ctx, q)
	if err == nil {
		t.Fatal("expected error from handler")
	}

	if !errors.Is(err, handlerErr) {
		t.Errorf("error should wrap handler error, got: %v", err)
	}
}

func TestDispatcher_Dispatch_QueryNotSupported_ErrorChain(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	q := mustNewQuery("UnknownQuery")

	_, err := d.Dispatch(context.Background(), q)
	if err == nil {
		t.Fatal("expected error for unregistered query")
	}

	if !errors.Is(err, query.ErrHandlerNotFound) {
		t.Errorf("error should be ErrHandlerNotFound, got: %v", err)
	}
}

func TestDispatcher_Register_Duplicate(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	handler := func(_ context.Context, _ query.Query) (any, error) {
		return "", nil
	}

	err := d.Register("GetUser", handler)
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err = d.Register("GetUser", handler)
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestDispatcher_Closed_DispatchErrorChain(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	_ = d.Close()

	q := mustNewQuery("TestQuery")

	_, err := d.Dispatch(context.Background(), q)
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, query.ErrDispatcherClosed) {
		t.Errorf("error should wrap ErrDispatcherClosed, got: %v", err)
	}
}

func TestDispatcher_Closed_RegisterErrorChain(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	_ = d.Close()

	err := d.Register("TestQuery", func(_ context.Context, _ query.Query) (any, error) {
		return "", nil
	})
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, query.ErrDispatcherClosed) {
		t.Errorf("error should wrap ErrDispatcherClosed, got: %v", err)
	}
}

func TestDispatcher_Dispatch_Success(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	registerHandler(d, "TestQuery", "result")

	q := mustNewQuery("TestQuery")

	result, err := d.Dispatch(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

func TestDispatcher_Dispatch_WrappedClosedError(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	registerHandler(d, "TestQuery", "")
	_ = d.Close()

	q := mustNewQuery("TestQuery")

	_, err := d.Dispatch(context.Background(), q)
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, query.ErrDispatcherClosed) {
		t.Errorf("error should wrap ErrDispatcherClosed, got: %v", err)
	}
}

func TestDispatchTyped_DispatchError(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	q := mustNewQuery("UnknownQuery")

	_, err := query.DispatchTyped[string](context.Background(), d, q)
	if err == nil {
		t.Fatal("expected error from DispatchTyped when Dispatch fails")
	}
}

func TestDispatcher_Use(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	var called bool
	useMiddleware(&called, d)

	registerHandler(d, "TestQuery", "result")

	q := mustNewQuery("TestQuery")

	result, err := d.Dispatch(context.Background(), q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected middleware to be called")
	}

	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

func TestDispatcher_Register_ClosedDispatcher(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	_ = d.Close()

	err := d.Register("Q", failingQueryHandler("unreachable"))
	if err == nil {
		t.Fatal("expected error registering on closed dispatcher")
	}
}

func TestDispatchTyped_TypeMismatch(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	registerHandler(d, "IntQuery", 42)

	q := mustNewQuery("IntQuery")

	_, err := query.DispatchTyped[string](context.Background(), d, q)
	if err == nil {
		t.Fatal("expected error from DispatchTyped when result type mismatches")
	}
}

func TestDispatchTyped_Success(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	registerHandler(d, "StringQuery", "hello")

	q := mustNewQuery("StringQuery")

	result, err := query.DispatchTyped[string](context.Background(), d, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}
}

func TestRegisterTyped_Success(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	type User struct{ Name string }

	err := query.RegisterTyped(
		d, "GetUser",
		func(_ context.Context, _ *query.BasicQuery) (*User, error) {
			return &User{Name: "Alice"}, nil
		},
	)
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	q := mustNewQuery("GetUser")

	result, err := query.DispatchTyped[*User](context.Background(), d, q)
	if err != nil {
		t.Fatalf("DispatchTyped() error = %v", err)
	}

	if result.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got %q", result.Name)
	}
}

func TestRegisterTyped_HandlerError(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	handlerErr := errors.New("db down")

	err := query.RegisterTyped(
		d,
		"FailQuery",
		func(_ context.Context, _ *query.BasicQuery) (string, error) {
			return "", handlerErr
		},
	)
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	q := mustNewQuery("FailQuery")

	_, err = query.DispatchTyped[string](context.Background(), d, q)
	if err == nil {
		t.Fatal("expected error from typed handler")
	}

	if !errors.Is(err, handlerErr) {
		t.Errorf("error should wrap handler error, got: %v", err)
	}
}

func TestRegisterTyped_Duplicate(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	handler := func(_ context.Context, _ *query.BasicQuery) (string, error) { return "", nil }

	err := query.RegisterTyped(d, "DupQuery", handler)
	if err != nil {
		t.Fatalf("first RegisterTyped() error = %v", err)
	}

	err = query.RegisterTyped(d, "DupQuery", handler)
	if err == nil {
		t.Fatal("expected error for duplicate typed registration")
	}
}

func TestRegisterTyped_WorksWithMiddleware(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	var called bool
	useMiddleware(&called, d)

	err := query.RegisterTyped(
		d, "MWQuery",
		func(_ context.Context, _ *query.BasicQuery) (int, error) {
			return 42, nil
		},
	)
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	q := mustNewQuery("MWQuery")

	result, err := query.DispatchTyped[int](context.Background(), d, q)
	if err != nil {
		t.Fatalf("DispatchTyped() error = %v", err)
	}

	if !called {
		t.Error("expected middleware to be called")
	}

	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}
