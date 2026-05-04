package query_test

import (
	"context"
	"testing"

	"errors"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/dispatcher"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func TestNew_EmptyType(t *testing.T) {
	t.Parallel()

	_, err := query.New("")
	if err == nil {
		t.Error("expected error for empty query type")
	}
}

func TestMustNew_PanicsOnEmptyType(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty query type")
		}
	}()

	_ = query.MustNew("")
}

func TestNewCatalogCore_Success(t *testing.T) {
	t.Parallel()

	meta := query.CatalogMeta{
		Name:    "GetUser",
		Version: "1.0.0",
		Summary: "Gets a user",
	}

	catalog, err := query.NewCatalogCore("user.get", meta)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if catalog.Type() != query.Type("user.get") {
		t.Errorf("Type() = %q, want %q", catalog.Type(), "user.get")
	}
}

func TestNewCatalogCore_InvalidInput(t *testing.T) {
	t.Parallel()

	_, err := query.NewCatalogCore("", query.CatalogMeta{})
	if err == nil {
		t.Error("expected error for empty query type")
	}
}

func TestMustNewCatalogCore_PanicsOnInvalidInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for invalid input")
		}
	}()

	_ = query.MustNewCatalogCore("", query.CatalogMeta{})
}

func TestCatalogInfo(t *testing.T) {
	t.Parallel()

	meta := query.CatalogMeta{
		Name:    "GetUser",
		Version: "1.0.0",
		Summary: "Gets a user",
	}

	cc := query.MustNewCatalogCore("user.get", meta)
	got := cc.CatalogInfo()

	if got.Name != meta.Name {
		t.Errorf("Name = %q, want %q", got.Name, meta.Name)
	}

	if got.Version != meta.Version {
		t.Errorf("Version = %q, want %q", got.Version, meta.Version)
	}

	if got.Summary != meta.Summary {
		t.Errorf("Summary = %q, want %q", got.Summary, meta.Summary)
	}
}

func TestDispatcher_Dispatch_HandlerError(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	ctx := context.Background()

	handlerErr := errors.New("handler failed")
	_ = d.Register("FailQuery", func(_ context.Context, _ query.Query) (any, error) {
		return nil, handlerErr
	})

	q := query.MustNew("FailQuery")

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

	q := query.MustNew("UnknownQuery")

	_, err := d.Dispatch(context.Background(), q)
	if err == nil {
		t.Fatal("expected error for unregistered query")
	}

	if !errors.Is(err, query.ErrQueryNotSupported) {
		t.Errorf("error should be ErrQueryNotSupported, got: %v", err)
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

	q := query.MustNew("TestQuery")

	_, err := d.Dispatch(context.Background(), q)
	if err == nil {
		t.Fatal("expected error on closed dispatcher")
	}

	if !errors.Is(err, dispatcher.ErrDispatcherClosed) {
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

func TestDispatchTyped_DispatchError(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	q := query.MustNew("UnknownQuery")

	_, err := query.DispatchTyped[string](context.Background(), d, q)
	if err == nil {
		t.Fatal("expected error from DispatchTyped when Dispatch fails")
	}
}

func TestDispatcher_Use(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()
	called := false

	d.Use(func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			called = true

			return next(ctx, q)
		}
	})

	_ = d.Register("TestQuery", func(_ context.Context, _ query.Query) (any, error) {
		return "result", nil
	})

	q := query.MustNew("TestQuery")

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

	err := d.Register("Q", func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("unreachable")
	})
	if err == nil {
		t.Fatal("expected error registering on closed dispatcher")
	}
}

func TestDispatchTyped_TypeMismatch(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	_ = d.Register("IntQuery", func(_ context.Context, _ query.Query) (any, error) {
		return 42, nil
	})

	q := query.MustNew("IntQuery")

	_, err := query.DispatchTyped[string](context.Background(), d, q)
	if err == nil {
		t.Fatal("expected error from DispatchTyped when result type mismatches")
	}
}

func TestDispatchTyped_Success(t *testing.T) {
	t.Parallel()

	d := query.NewDispatcher()

	_ = d.Register("StringQuery", func(_ context.Context, _ query.Query) (any, error) {
		return "hello", nil
	})

	q := query.MustNew("StringQuery")

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

	err := query.RegisterTyped(d, "GetUser", func(_ context.Context, _ query.Query) (*User, error) {
		return &User{Name: "Alice"}, nil
	})
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	q := query.MustNew("GetUser")

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

	err := query.RegisterTyped(d, "FailQuery", func(_ context.Context, _ query.Query) (string, error) {
		return "", handlerErr
	})
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	q := query.MustNew("FailQuery")

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

	handler := func(_ context.Context, _ query.Query) (string, error) { return "", nil }

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

	called := false

	d.Use(func(next query.Handler) query.Handler {
		return func(ctx context.Context, q query.Query) (any, error) {
			called = true

			return next(ctx, q)
		}
	})

	err := query.RegisterTyped(d, "MWQuery", func(_ context.Context, _ query.Query) (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("RegisterTyped() error = %v", err)
	}

	q := query.MustNew("MWQuery")

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
