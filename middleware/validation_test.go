package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

func TestCommandValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ command.Command) error { return nil }
	mw := CommandValidation(validate)

	called := false
	handler := mw(callbackCommandHandler(&called))

	cmd := &testCommand{streamID: id.NewStreamID()}

	err := handler(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestCommandValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ command.Command) error {
		return errors.New("invalid")
	}
	mw := CommandValidation(validate)

	handler := mw(failingCommandHandler("should not be called"))

	cmd := &testCommand{streamID: id.NewStreamID()}

	err := handler(context.Background(), cmd)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEventValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ event.Event) error { return nil }
	mw := EventValidation(validate)

	called := false
	handler := mw(eventtest.CallbackEventHandler(&called))

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestEventValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ event.Event) error {
		return errors.New("invalid event")
	}
	mw := EventValidation(validate)

	handler := mw(eventtest.FailingEventHandler("should not be called"))

	evt, err := eventtest.NewTestEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = handler(context.Background(), evt)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestQueryValidation_Pass(t *testing.T) {
	t.Parallel()

	validate := func(_ query.Query) error { return nil }
	mw := QueryValidation(validate)

	called := false
	handler := mw(callbackQueryHandler(&called))

	_, err := handler(context.Background(), &testQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

func TestQueryValidation_Fail(t *testing.T) {
	t.Parallel()

	validate := func(_ query.Query) error {
		return errors.New("invalid")
	}
	mw := QueryValidation(validate)

	handler := mw(failingQueryHandler("should not be called"))

	_, err := handler(context.Background(), &testQuery{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestCommandValidation_SentinelError(t *testing.T) {
	t.Parallel()

	validatorErr := errors.New("invalid")
	mw := CommandValidation(func(_ command.Command) error {
		return validatorErr
	})
	handler := mw(NoopCommandHandler())

	err := handler(context.Background(), &testCommand{streamID: id.NewStreamID()})
	if !errors.Is(err, validatorErr) {
		t.Errorf("expected errors.Is(err, validatorErr), got %v", err)
	}
}

func TestEventValidation_SentinelError(t *testing.T) {
	t.Parallel()

	validatorErr := errors.New("invalid")
	mw := EventValidation(func(_ event.Event) error {
		return validatorErr
	})
	handler := mw(eventtest.NoopEventHandler())

	evt, evtErr := eventtest.NewTestEvent()
	if evtErr != nil {
		t.Fatalf("NewTestEvent: %v", evtErr)
	}

	err := handler(context.Background(), evt)
	if !errors.Is(err, validatorErr) {
		t.Errorf("expected errors.Is(err, validatorErr), got %v", err)
	}
}
