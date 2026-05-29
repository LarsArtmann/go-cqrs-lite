package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/id"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandLogging_Success(t *testing.T) {
	t.Parallel()

	logger, h := newTestLogger()
	mw := CommandLogging(logger)

	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

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

	handler := cmdMw(testhelpers.FailingCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

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

	handler := mw(testhelpers.NoopQueryHandler())

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

	handler := mw(testhelpers.FailingQueryHandler("boom"))

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

	handler := mw(testhelpers.NoopEventHandler())

	evt, err := testhelpers.NewTestEvent()
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

	handler := mw(testhelpers.FailingEventHandler("boom"))

	evt, err := testhelpers.NewTestEvent()
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
