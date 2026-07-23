package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	. "github.com/onsi/gomega"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

type traceCmd struct {
	commandID id.CommandID
	streamID  id.StreamID
}

func (c *traceCmd) Type() command.Type    { return "test.command" }
func (c *traceCmd) StreamID() id.StreamID { return c.streamID }
func (c *traceCmd) ID() id.CommandID      { return c.commandID }

func newTraceLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer

	return slog.New(slog.NewTextHandler(&buf, nil)), &buf
}

func newTP() *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()))
}

func TestCommandTraceLogging_WithSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tracerProvider := newTP()
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	logger, buf := newTraceLogger()
	mw := middleware.CommandTraceLogging(logger)

	called := false
	handler := mw(func(_ context.Context, _ command.Command) error {
		called = true

		return nil
	})

	ctx, span := tracerProvider.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	err := handler(ctx, &traceCmd{streamID: id.NewAggregateID()})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())

	output := buf.String()
	g.Expect(output).ToNot(ContainSubstring("trace_id=none"))
	g.Expect(output).To(ContainSubstring("command dispatching"))
	g.Expect(output).To(ContainSubstring("command succeeded"))
}

func TestCommandTraceLogging_NoSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	logger, buf := newTraceLogger()
	mw := middleware.CommandTraceLogging(logger)

	handler := mw(middleware.NoopCommandHandler())

	err := handler(context.Background(), &traceCmd{streamID: id.NewAggregateID()})
	g.Expect(err).ToNot(HaveOccurred())

	output := buf.String()
	g.Expect(output).To(ContainSubstring("trace_id=none"))
}

func TestEventTraceLogging_WithSpan(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tracerProvider := newTP()
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	logger, buf := newTraceLogger()
	mw := middleware.EventTraceLogging(logger)

	called := false
	handler := mw(func(_ context.Context, _ event.Event) error {
		called = true

		return nil
	})

	ctx, span := tracerProvider.Tracer("test").Start(context.Background(), "test")
	defer span.End()

	aggID := id.NewAggregateID()
	evt, err := event.NewEvent("test.event", aggID, "Test", event.Version(1), nil)
	g.Expect(err).ToNot(HaveOccurred())

	err = handler(ctx, evt)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())

	output := buf.String()
	g.Expect(output).ToNot(ContainSubstring("trace_id=none"))
	g.Expect(output).To(ContainSubstring("event handling"))
	g.Expect(output).To(ContainSubstring("event handled"))
}

func TestCommandTraceLogging_Error(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	logger, buf := newTraceLogger()
	mw := middleware.CommandTraceLogging(logger)

	testErr := errors.New("test error")
	handler := mw(func(_ context.Context, _ command.Command) error {
		return testErr
	})

	err := handler(context.Background(), &traceCmd{streamID: id.NewAggregateID()})
	g.Expect(err).To(MatchError(testErr))

	output := buf.String()
	g.Expect(output).To(ContainSubstring("command failed"))
}
