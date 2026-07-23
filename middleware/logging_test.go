package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func TestCommandLogging_Success(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	mw := CommandLogging(logger)

	handler := mw(NoopCommandHandler())

	cmd := &testCommand{streamID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := h.InfoCount(); got != 2 {
		t.Errorf("info logs: got %d, want 2", got)
	}
}

func TestCommandLogging_Error(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	cmdMw := CommandLogging(logger)

	handler := cmdMw(failingCommandHandler("boom"))

	cmd := &testCommand{streamID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if got := h.ErrorCount(); got != 1 {
		t.Errorf("error logs: got %d, want 1", got)
	}
}

func TestQueryLogging_Success(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	mw := QueryLogging(logger)

	handler := mw(noopQueryHandler())

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := h.InfoCount(); got != 2 {
		t.Errorf("info logs: got %d, want 2", got)
	}
}

func TestQueryLogging_Error(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	mw := QueryLogging(logger)

	handler := mw(failingQueryHandler("boom"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	if got := h.ErrorCount(); got != 1 {
		t.Errorf("error logs: got %d, want 1", got)
	}
}

func TestEventLogging_Success(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	mw := EventLogging(logger)

	handler := mw(eventtest.NoopEventHandler())

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := h.InfoCount(); got != 2 {
		t.Errorf("info logs: got %d, want 2", got)
	}
}

func TestEventLogging_Error(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	mw := EventLogging(logger)

	handler := mw(eventtest.FailingEventHandler("boom"))

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	if got := h.ErrorCount(); got != 1 {
		t.Errorf("error logs: got %d, want 1", got)
	}
}
