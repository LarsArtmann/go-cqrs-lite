package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestCommandLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := CommandLogging(logger)

	handler := mw(testhelpers.NoopCommandHandler())

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.Logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.Logs))
	}
}

func TestCommandLogging_Error(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	cmdMw := CommandLogging(logger)

	handler := cmdMw(testhelpers.FailingCommandHandler("boom"))

	cmd := &testCommand{aggregateID: id.NewAggregateID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected error")
	}

	if len(logger.Errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.Errors))
	}
}

func TestQueryLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := QueryLogging(logger)

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return "ok", nil
	})

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.Logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.Logs))
	}
}

func TestQueryLogging_Error(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := QueryLogging(logger)

	handler := mw(func(_ context.Context, _ query.Query) (any, error) {
		return nil, errors.New("boom")
	})

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	if len(logger.Errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.Errors))
	}
}

func TestEventLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := EventLogging(logger)

	handler := mw(testhelpers.NoopEventHandler())

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(logger.Logs) != 2 {
		t.Errorf("expected 2 info logs, got %d", len(logger.Logs))
	}
}

func TestEventLogging_Error(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := EventLogging(logger)

	handler := mw(testhelpers.FailingEventHandler("boom"))

	evt, err := event.NewEvent("test.evt", id.NewAggregateID(), "Test", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected error")
	}

	if len(logger.Errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logger.Errors))
	}
}
