package middleware

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
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

	testhelpers.AssertLen(t, "info logs", logger.Logs, 2)
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

	testhelpers.AssertLen(t, "error logs", logger.Errors, 1)
}

func TestQueryLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := QueryLogging(logger)

	handler := mw(testhelpers.NoopQueryHandler())

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	testhelpers.AssertLen(t, "info logs", logger.Logs, 2)
}

func TestQueryLogging_Error(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
	mw := QueryLogging(logger)

	handler := mw(testhelpers.FailingQueryHandler("boom"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected error")
	}

	testhelpers.AssertLen(t, "error logs", logger.Errors, 1)
}

func TestEventLogging_Success(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
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

	testhelpers.AssertLen(t, "info logs", logger.Logs, 2)
}

func TestEventLogging_Error(t *testing.T) {
	t.Parallel()

	logger := &testLogger{}
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

	testhelpers.AssertLen(t, "error logs", logger.Errors, 1)
}
