package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/middleware"
)

type traceCmd struct {
	aggregateID id.AggregateID
}

func (c *traceCmd) Type() command.Type          { return "test.command" }
func (c *traceCmd) AggregateID() id.AggregateID { return c.aggregateID }

func newTraceLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer

	return slog.New(slog.NewTextHandler(&buf, nil)), &buf
}

func newTP() *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
}

func TestCommandTraceLogging_WithSpan(t *testing.T) {
	tp := newTP()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	logger, buf := newTraceLogger()
	mw := middleware.CommandTraceLogging(logger)

	called := false
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true
		return nil
	})

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	err := handler(ctx, &traceCmd{aggregateID: id.NewAggregateID()})
	require.NoError(t, err)
	require.True(t, called)

	output := buf.String()
	require.NotContains(t, output, "trace_id=none")
	require.Contains(t, output, "command dispatching")
	require.Contains(t, output, "command succeeded")
}

func TestCommandTraceLogging_NoSpan(t *testing.T) {
	logger, buf := newTraceLogger()
	mw := middleware.CommandTraceLogging(logger)

	handler := mw(func(_ context.Context, _ command.Command) error {
		return nil
	})

	err := handler(context.Background(), &traceCmd{aggregateID: id.NewAggregateID()})
	require.NoError(t, err)

	output := buf.String()
	require.Contains(t, output, "trace_id=none")
}

func TestEventTraceLogging_WithSpan(t *testing.T) {
	tp := newTP()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	logger, buf := newTraceLogger()
	mw := middleware.EventTraceLogging(logger)

	called := false
	handler := mw(func(_ context.Context, _ event.Event) error {
		called = true
		return nil
	})

	ctx, span := tp.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.event", aggID, "Test", event.Version(1), nil)
	require.NoError(t, err)

	err = handler(ctx, evt)
	require.NoError(t, err)
	require.True(t, called)

	output := buf.String()
	require.NotContains(t, output, "trace_id=none")
	require.Contains(t, output, "event handling")
	require.Contains(t, output, "event handled")
}

func TestCommandTraceLogging_Error(t *testing.T) {
	logger, buf := newTraceLogger()
	mw := middleware.CommandTraceLogging(logger)

	testErr := errors.New("test error")
	handler := mw(func(_ context.Context, _ command.Command) error {
		return testErr
	})

	err := handler(context.Background(), &traceCmd{aggregateID: id.NewAggregateID()})
	require.ErrorIs(t, err, testErr)

	output := buf.String()
	require.Contains(t, output, "command failed")
}
