package testhelpers

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

func testCmd(t *testing.T) command.Command {
	t.Helper()

	cmd, err := command.New("test.cmd", id.NewAggregateID())
	if err != nil {
		t.Fatalf("command.New: %v", err)
	}

	return cmd
}

func TestAppendEventsHandler(t *testing.T) {
	t.Parallel()

	var collected []event.Event

	handler := AppendEventsHandler(&collected)

	evt, _ := NewTestEvent()

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if len(collected) != 1 {
		t.Fatalf("len(collected) = %d, want 1", len(collected))
	}
}

func TestNoopCommandHandler(t *testing.T) {
	t.Parallel()

	handler := NoopCommandHandler()

	err := handler(context.Background(), testCmd(t))
	if err != nil {
		t.Fatalf("NoopCommandHandler: %v", err)
	}
}

func TestNoopEventHandler(t *testing.T) {
	t.Parallel()

	handler := NoopEventHandler()

	evt, _ := NewTestEvent()

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("NoopEventHandler: %v", err)
	}
}

func TestNoopQueryHandler(t *testing.T) {
	t.Parallel()

	handler := NoopQueryHandler()

	result, err := handler(context.Background(), nil)
	if err != nil {
		t.Fatalf("NoopQueryHandler: %v", err)
	}

	if result != nil {
		t.Error("expected nil result")
	}
}

func TestFailingCommandHandler(t *testing.T) {
	t.Parallel()

	handler := FailingCommandHandler("test fail")

	err := handler(context.Background(), testCmd(t))
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "test fail" {
		t.Errorf("error = %q, want test fail", err.Error())
	}
}

func TestFailingEventHandler(t *testing.T) {
	t.Parallel()

	handler := FailingEventHandler("evt fail")

	evt, _ := NewTestEvent()

	err := handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCallbackCommandHandler(t *testing.T) {
	t.Parallel()

	var called bool

	handler := CallbackCommandHandler(&called)

	err := handler(context.Background(), testCmd(t))
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if !called {
		t.Error("expected called to be true")
	}
}

func TestCallbackEventHandler(t *testing.T) {
	t.Parallel()

	var called bool

	handler := CallbackEventHandler(&called)

	evt, _ := NewTestEvent()

	err := handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if !called {
		t.Error("expected called to be true")
	}
}

func TestCallbackQueryHandler(t *testing.T) {
	t.Parallel()

	var called bool

	handler := CallbackQueryHandler(&called)

	_, err := handler(context.Background(), query.Query(nil))
	if err != nil {
		t.Fatalf("handler: %v", err)
	}

	if !called {
		t.Error("expected called to be true")
	}
}

func TestFailingQueryHandler(t *testing.T) {
	t.Parallel()

	handler := FailingQueryHandler("query fail")

	_, err := handler(context.Background(), query.Query(nil))
	if err == nil {
		t.Fatal("expected error")
	}

	if err.Error() != "query fail" {
		t.Errorf("error = %q, want query fail", err.Error())
	}
}

func TestPanicCommandHandler(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}

		if r != "boom" {
			t.Errorf("panic = %v, want boom", r)
		}
	}()

	handler := PanicCommandHandler("boom")
	_ = handler(context.Background(), testCmd(t))
}

func TestPanicEventHandler(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}

		if r != "evt-boom" {
			t.Errorf("panic = %v, want evt-boom", r)
		}
	}()

	handler := PanicEventHandler("evt-boom")
	evt, _ := NewTestEvent()
	_ = handler(context.Background(), evt)
}

func TestPanicQueryHandler(t *testing.T) {
	t.Parallel()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}

		if r != "q-boom" {
			t.Errorf("panic = %v, want q-boom", r)
		}
	}()

	handler := PanicQueryHandler("q-boom")
	_, _ = handler(context.Background(), nil)
}

func TestCommandMiddleware(t *testing.T) {
	t.Parallel()

	var order []string

	mw := CommandMiddleware(&order, "mw1")

	inner := NoopCommandHandler()

	wrapped := mw(inner)

	err := wrapped(context.Background(), testCmd(t))
	if err != nil {
		t.Fatalf("wrapped: %v", err)
	}

	if len(order) != 1 || order[0] != "mw1" {
		t.Errorf("order = %v, want [mw1]", order)
	}
}

func TestEventMiddleware(t *testing.T) {
	t.Parallel()

	var order []string

	mw := EventMiddleware(&order, "evt-mw")

	inner := NoopEventHandler()

	wrapped := mw(inner)

	evt, _ := NewTestEvent()

	err := wrapped(context.Background(), evt)
	if err != nil {
		t.Fatalf("wrapped: %v", err)
	}

	if len(order) != 1 || order[0] != "evt-mw" {
		t.Errorf("order = %v, want [evt-mw]", order)
	}
}
