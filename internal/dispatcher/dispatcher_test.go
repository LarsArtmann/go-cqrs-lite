package dispatcher

import (
	"errors"
	"sync"
	"testing"
)

type testHandler func(string) string

type testMiddleware func(testHandler) testHandler

func TestLifecycleMixin_Close(t *testing.T) {
	t.Parallel()

	m := &LifecycleMixin{}
	if m.IsClosed() {
		t.Error("new mixin should not be closed")
	}

	if err := m.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !m.IsClosed() {
		t.Error("should be closed after Close()")
	}

	if err := m.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestLifecycleMixin_CheckClosed(t *testing.T) {
	t.Parallel()

	m := &LifecycleMixin{}
	closedErr := errors.New("closed")

	if err := m.CheckClosed(closedErr); err != nil {
		t.Errorf("CheckClosed() on open mixin should return nil, got %v", err)
	}

	_ = m.Close()
	if err := m.CheckClosed(closedErr); !errors.Is(err, closedErr) {
		t.Errorf("CheckClosed() on closed mixin should return closedErr, got %v", err)
	}
}

func TestLifecycle_Close(t *testing.T) {
	t.Parallel()

	l := &Lifecycle{}
	if l.IsClosed() {
		t.Error("new lifecycle should not be closed")
	}

	if err := l.Close(); err != nil {
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

	if err := l.CheckClosed(closedErr); err != nil {
		t.Errorf("CheckClosed() on open lifecycle should return nil, got %v", err)
	}

	_ = l.Close()
	if err := l.CheckClosed(closedErr); !errors.Is(err, closedErr) {
		t.Errorf("CheckClosed() on closed lifecycle should return closedErr, got %v", err)
	}
}

func TestMiddlewareChain_Add(t *testing.T) {
	t.Parallel()

	c := &MiddlewareChain[testHandler, testMiddleware]{}

	count := 0
	mw := func(h testHandler) testHandler {
		return func(s string) string {
			count++
			return h(s)
		}
	}

	c.Add(mw, mw)
	if len(c.Middleware()) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(c.Middleware()))
	}
}

func TestMiddlewareChain_Apply(t *testing.T) {
	t.Parallel()

	c := &MiddlewareChain[testHandler, testMiddleware]{}

	var order []string
	c.Add(
		func(h testHandler) testHandler {
			return func(s string) string {
				order = append(order, "mw1")
				return h(s)
			}
		},
		func(h testHandler) testHandler {
			return func(s string) string {
				order = append(order, "mw2")
				return h(s)
			}
		},
	)

	handler := func(s string) string {
		order = append(order, "handler")
		return s
	}

	wrapped := c.Apply(handler, func(m testMiddleware, h testHandler) testHandler {
		return m(h)
	})

	result := wrapped("test")
	if result != "test" {
		t.Errorf("expected test, got %s", result)
	}

	expected := []string{"mw1", "mw2", "handler"}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected order %v, got %v", expected, order)
			break
		}
	}
}

func TestMiddlewareChain_Apply_NoMiddleware(t *testing.T) {
	t.Parallel()

	c := &MiddlewareChain[testHandler, testMiddleware]{}

	handler := func(s string) string { return "result:" + s }
	wrapped := c.Apply(handler, func(m testMiddleware, h testHandler) testHandler {
		return m(h)
	})

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
	if d.Handlers == nil {
		t.Error("Handlers map should be initialized")
	}
}

func TestDispatcher_Use(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	d.Use(func(h testHandler) testHandler {
		return func(s string) string { return h(s) }
	})

	if len(d.Middleware.Middleware()) != 1 {
		t.Error("expected 1 middleware after Use()")
	}
}

func TestDispatcher_Register(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	handler := func(s string) string { return s }
	if err := d.Register("test", handler); err != nil {
		t.Errorf("Register() error = %v", err)
	}
	if d.Handlers["test"] == nil {
		t.Error("handler should be registered")
	}
}

func TestDispatcher_Register_Closed(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()
	_ = d.Close()

	handler := func(s string) string { return s }
	if err := d.Register("test", handler); err == nil {
		t.Error("expected error when registering on closed dispatcher")
	}
}

func TestDispatcher_Dispatch(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	handler := func(s string) string { return "handled:" + s }
	_ = d.Register("test", handler)

	wrap := func(m testMiddleware, h testHandler) testHandler {
		return m(h)
	}

	result, err := d.Dispatch("test", handler, wrap)
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

	handler := func(s string) string { return s }
	wrap := func(m testMiddleware, h testHandler) testHandler {
		return m(h)
	}

	_, err := d.Dispatch("missing", handler, wrap)
	if err == nil {
		t.Error("expected error for missing handler")
	}
}

func TestDispatcher_Dispatch_Closed(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()
	_ = d.Close()

	handler := func(s string) string { return s }
	wrap := func(m testMiddleware, h testHandler) testHandler {
		return m(h)
	}

	_, err := d.Dispatch("test", handler, wrap)
	if err == nil {
		t.Error("expected error when dispatching on closed dispatcher")
	}
}

func TestDispatcher_Dispatch_WithMiddleware(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()

	var order []string
	d.Use(func(h testHandler) testHandler {
		return func(s string) string {
			order = append(order, "mw1")
			return h(s)
		}
	})

	handler := func(s string) string {
		order = append(order, "handler")
		return "result"
	}
	_ = d.Register("test", handler)

	wrap := func(m testMiddleware, h testHandler) testHandler {
		return m(h)
	}

	result, err := d.Dispatch("test", handler, wrap)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	if result("x") != "result" {
		t.Error("unexpected result")
	}

	expected := []string{"mw1", "handler"}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("expected order %v, got %v", expected, order)
			break
		}
	}
}

func TestDispatcher_Close(t *testing.T) {
	t.Parallel()

	d := NewDispatcher[testHandler, testMiddleware]()
	if err := d.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !d.Lifecycle.IsClosed() {
		t.Error("dispatcher should be closed")
	}
}

func TestLifecycleMixin_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &LifecycleMixin{}
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.IsClosed()
			_ = m.CheckClosed(nil)
		}()
	}

	_ = m.Close()
	wg.Wait()
}

func TestMiddlewareChain_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := &MiddlewareChain[testHandler, testMiddleware]{}
	var wg sync.WaitGroup

	mw := func(h testHandler) testHandler { return h }

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.Add(mw)
		}()
		go func() {
			defer wg.Done()
			_ = c.Middleware()
		}()
	}

	wg.Wait()
}
