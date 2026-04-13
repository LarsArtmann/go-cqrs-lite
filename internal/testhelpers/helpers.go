package testhelpers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog"
	"github.com/larsartmann/go-cqrs-lite/catalog/eventcatalog"
	"github.com/larsartmann/go-cqrs-lite/command"
	"github.com/larsartmann/go-cqrs-lite/event"
)

// TestLogger is a logger for testing that captures log messages.
type TestLogger struct {
	mu     sync.Mutex
	Logs   []string
	Errors []string
}

func (l *TestLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Logs = append(l.Logs, msg)
}

func (l *TestLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.Errors = append(l.Errors, msg)
}

// TestMetrics is a metrics collector for testing.
type TestMetrics struct {
	mu        sync.Mutex
	Records   []string
	Durations []time.Duration
}

func (m *TestMetrics) Observe(name string, duration time.Duration, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Records = append(m.Records, name)
	m.Durations = append(m.Durations, duration)
}

func AssertCallOrder(t *testing.T, callOrder, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}

func NoopCommandHandler() command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return nil
	}
}

func NoopEventHandler() event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return nil
	}
}

func CallbackCommandHandler(called *bool) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		*called = true

		return nil
	}
}

func PanicCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		panic(msg)
	}
}

func PanicEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		panic(msg)
	}
}

func FailingCommandHandler(msg string) command.Handler {
	return func(_ context.Context, _ command.Command) error {
		return errors.New(msg)
	}
}

func FailingEventHandler(msg string) event.Handler {
	return func(_ context.Context, _ event.Event) error {
		return errors.New(msg)
	}
}

func EventMiddleware(callOrder *[]string, name string) func(h event.Handler) event.Handler {
	return func(h event.Handler) event.Handler {
		return func(ctx context.Context, evt event.Event) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, evt)
		}
	}
}

func CommandMiddleware(callOrder *[]string, name string) func(h command.Handler) command.Handler {
	return func(h command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			*callOrder = append(*callOrder, name)

			return h(ctx, cmd)
		}
	}
}

// BenchmarkEventCatalogExport benchmarks exporting a catalog to EventCatalog format.
func BenchmarkEventCatalogExport(b *testing.B, cat *catalog.Catalog) {
	for b.Loop() {
		exp := eventcatalog.NewExporter(b.TempDir())
		_ = exp.Export(cat)
	}
}
