// Package testhelpers provides shared test utilities for go-cqrs-lite modules.
// This package can be imported by any module without creating import cycles.
package testhelpers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/query"
)

// AppendEventsHandler returns a bus handler that appends received events to *events.
func AppendEventsHandler(events *[]event.Event) event.Handler {
	return func(_ context.Context, evt event.Event) error {
		*events = append(*events, evt)

		return nil
	}
}

// TestMetrics is a metrics collector for testing.
type TestMetrics struct {
	mu        sync.Mutex
	Records   []string
	Durations []time.Duration
}

// Observe records a metric observation.
func (m *TestMetrics) Observe(name string, duration time.Duration, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Records = append(m.Records, name)
	m.Durations = append(m.Durations, duration)
}

// NoopCommandHandler returns a handler that does nothing.
func NoopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

// NoopEventHandler returns a handler that does nothing.
func NoopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return nil
	}
}

// NoopQueryHandler returns a handler that does nothing and returns nil.
func NoopQueryHandler() func(context.Context, query.Query) (any, error) {
	return func(_ context.Context, _ query.Query) (any, error) {
		return nil, nil
	}
}

// FailingCommandHandler returns a handler that returns an error.
func FailingCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg) //nolint:err113 // test helper with dynamic message
	}
}

// FailingEventHandler returns a handler that returns an error.
func FailingEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return errors.New(msg) //nolint:err113 // test helper with dynamic message
	}
}

// PanicCommandHandler returns a handler that panics with the given message.
func PanicCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		panic(msg)
	}
}

// PanicEventHandler returns a handler that panics with the given message.
func PanicEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		panic(msg)
	}
}

// CallbackCommandHandler returns a handler that sets *called to true.
func CallbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

// CallbackEventHandler returns a handler that sets *called to true.
func CallbackEventHandler(called *bool) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		*called = true

		return nil
	}
}

// CommandMiddleware creates middleware that tracks call order for command handlers.
func CommandMiddleware(callOrder *[]string, name string) func(h command.Handler) command.Handler {
	return func(h command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, cmd)
		}
	}
}

// AssertCallOrder asserts the call order matches expected.
func AssertCallOrder(t *testing.T, callOrder, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}

// EventMiddleware creates middleware that tracks call order for event handlers.
func EventMiddleware(callOrder *[]string, name string) func(h event.Handler) event.Handler {
	return func(h event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, evt)
		}
	}
}
