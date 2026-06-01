package dispatcher

import (
	"errors"
	"sync"
	"testing"
)

type testHandler func(string) string

type testMiddleware func(testHandler) testHandler

func testWrap(m testMiddleware, h testHandler) testHandler {
	return m(h)
}

func testMW(order *[]string, name string) testMiddleware {
	return func(h testHandler) testHandler {
		return func(s string) string {
			*order = append(*order, name)

			return h(s)
		}
	}
}

func assertCallOrder(t *testing.T, order, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected order %v, got %v", expected, order)

			break
		}
	}
}

func TestLifecycle_Close(t *testing.T) {
	t.Parallel()

	l := &Lifecycle{}
	if l.IsClosed() {
		t.Error("new lifecycle should not be closed")
	}

	err := l.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !l.IsClosed() {
		t.Error("should be closed after Close()")
	}
}

func TestLifecycle_CheckClosed(t *testing.T) {
	t.Parallel()

	l := &Lifecycle{}
	closedErr := errors.New("closed")

	err := l.CheckClosed(closedErr)
	if err != nil {
		t.Errorf("CheckClosed() on open lifecycle should return nil, got %v", err)
	}

	_ = l.Close()

	err = l.CheckClosed(closedErr)
	if !errors.Is(err, closedErr) {
		t.Errorf("CheckClosed() on closed lifecycle should return closedErr, got %v", err)
	}
}

func TestMiddlewareChain_Add(t *testing.T) {
	t.Parallel()

	c := &middlewareChain[testHandler, testMiddleware]{}

	count := 0
	middleware := func(h testHandler) testHandler {
		return func(s string) string {
			count++

			return h(s)
		}
	}

	c.Add(middleware, middleware)

	if len(c.Middleware()) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(c.Middleware()))
	}
}

func TestMiddlewareChain_Apply(t *testing.T) {
	t.Parallel()

	c := &middlewareChain[testHandler, testMiddleware]{}

	var order []string

	c.Add(testMW(&order, "mw1"), testMW(&order, "mw2"))

	handler := func(s string) string {
		order = append(order, "handler")

		return s
	}

	wrapped := c.Apply(handler, testWrap)

	result := wrapped("test")
	if result != "test" {
		t.Errorf("expected test, got %s", result)
	}

	assertCallOrder(t, order, []string{"mw1", "mw2", "handler"})
}

func TestMiddlewareChain_Apply_NoMiddleware(t *testing.T) {
	t.Parallel()

	c := &middlewareChain[testHandler, testMiddleware]{}

	handler := func(s string) string { return "result:" + s }
	wrapped := c.Apply(handler, testWrap)

	if wrapped("x") != "result:x" {
		t.Error("handler should pass through without middleware")
	}
}

func TestNewDispatcher(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()
	if d == nil {
		t.Fatal("NewDispatcher() returned nil")
	}

	if _, ok := d.getHandler("nonexistent"); ok {
		t.Error("should not find handler for unregistered type")
	}
}

func TestDispatcher_Use(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	d.Use(func(h testHandler) testHandler {
		return func(s string) string { return h(s) }
	})

	if len(d.middleware.Middleware()) != 1 {
		t.Error("expected 1 middleware after Use()")
	}
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	handler := func(s string) string { return s }

	err := d.Register("test", handler, testWrap)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if h, ok := d.getHandler("test"); !ok || h == nil {
		t.Error("handler should be registered")
	}
}

func TestDispatcher_Register_Closed(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()
	_ = d.Close()

	handler := func(s string) string { return s }

	err := d.Register("test", handler, testWrap)
	if err == nil {
		t.Error("expected error when registering on closed dispatcher")
	}
}

func TestDispatcher_Register_Duplicate(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	handler := func(s string) string { return s }

	err := d.Register("test", handler, testWrap)
	if err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err = d.Register("test", handler, testWrap)
	if err == nil {
		t.Error("expected error when registering duplicate handler")
	}

	if !errors.Is(err, ErrHandlerAlreadyRegistered) {
		t.Errorf("expected ErrHandlerAlreadyRegistered, got %v", err)
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	handler := func(s string) string { return "handled:" + s }
	_ = d.Register("test", handler, testWrap)

	result, err := d.Dispatch("test")
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if result("x") != "handled:x" {
		t.Errorf("expected handled:x, got %s", result("x"))
	}
}

func TestDispatcher_Dispatch_HandlerNotFound(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	_, err := d.Dispatch("missing")
	if err == nil {
		t.Error("expected error for missing handler")
	}
}

func TestDispatcher_Dispatch_Closed(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()
	_ = d.Close()

	_, err := d.Dispatch("test")
	if err == nil {
		t.Error("expected error when dispatching on closed dispatcher")
	}
}

func TestDispatcher_Dispatch_WithMiddleware(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	var order []string

	d.Use(testMW(&order, "mw1"))

	handler := func(_ string) string {
		order = append(order, "handler")

		return "result"
	}
	_ = d.Register("test", handler, testWrap)

	result, err := d.Dispatch("test")
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if result("x") != "result" {
		t.Error("unexpected result")
	}

	assertCallOrder(t, order, []string{"mw1", "handler"})
}

func TestDispatcher_Close(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	err := d.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !d.Lifecycle.IsClosed() {
		t.Error("dispatcher should be closed")
	}
}

func TestLifecycle_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &Lifecycle{}

	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() {
			_ = m.IsClosed()
			_ = m.CheckClosed(nil)
		})
	}

	_ = m.Close()

	wg.Wait()
}

func TestMiddlewareChain_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := &middlewareChain[testHandler, testMiddleware]{}

	var wg sync.WaitGroup

	middleware := func(h testHandler) testHandler { return h }

	for range 50 {
		wg.Add(2)

		go func() {
			defer wg.Done()

			c.Add(middleware)
		}()
		go func() {
			defer wg.Done()

			_ = c.Middleware()
		}()
	}

	wg.Wait()
}
